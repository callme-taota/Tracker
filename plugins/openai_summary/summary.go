package openai_summary

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
	"Tracker/plugins/sharedutil"
)

// Plugin implements OpenAI-based summarization.
type Plugin struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// New returns a new openai_summary plugin instance.
func New() plugin.Plugin {
	return &Plugin{
		model:      "gpt-4o-mini",
		baseURL:    "https://api.openai.com/v1",
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (p *Plugin) Name() string      { return "openai_summary" }
func (p *Plugin) Version() string   { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeSummary }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.apiKey = sharedutil.StringWithEnv(cfg, "api_key", os.Getenv("OPENAI_API_KEY"))
	if m := strings.TrimSpace(plugin.GetString(cfg, "model")); m != "" {
		p.model = m
	}
	if baseURL := strings.TrimSpace(plugin.GetString(cfg, "base_url")); baseURL != "" {
		p.baseURL = strings.TrimRight(baseURL, "/")
	}
	return nil
}

// TestConfig verifies that the configured OpenAI-compatible endpoint accepts the API key.
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	apiKey := sharedutil.StringWithEnv(cfg, "api_key", p.apiKey)
	baseURL := p.resolveBaseURL(cfg)
	if err := sharedutil.RequireFields(map[string]string{"api_key": apiKey, "base_url": baseURL}); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("openai models: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// Execute generates summary and key points for the item.
func (p *Plugin) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) {
	if in == nil {
		return nil, nil
	}
	apiKey := sharedutil.StringWithEnv(cfg, "api_key", p.apiKey)
	if apiKey == "" {
		out := *in
		out.Summary = truncate(in.Content, 300)
		return &out, nil
	}
	modelName := strings.TrimSpace(plugin.GetString(cfg, "model"))
	if modelName == "" {
		modelName = p.model
	}
	baseURL := p.resolveBaseURL(cfg)
	summary, keyPoints, err := p.callOpenAI(apiKey, baseURL, modelName, in.Title, in.Content)
	if err != nil {
		out := *in
		out.Summary = truncate(in.Content, 300)
		return &out, nil
	}
	out := *in
	out.Summary = summary
	out.KeyPoints = keyPoints
	if sharedutil.Bool(cfg, "include_mermaid", false) {
		if m, err := p.callOpenAIMermaid(apiKey, baseURL, modelName, summary); err == nil && strings.TrimSpace(m) != "" {
			out.Summary = summary + "\n\n---\n" + m
			if out.Extra == nil {
				out.Extra = make(map[string]interface{})
			}
			out.Extra["mermaid"] = m
		}
	}
	return &out, nil
}

func (p *Plugin) resolveBaseURL(cfg plugin.Config) string {
	if v := strings.TrimSpace(plugin.GetString(cfg, "base_url")); v != "" {
		return strings.TrimRight(v, "/")
	}
	if p.baseURL != "" {
		return p.baseURL
	}
	return "https://api.openai.com/v1"
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

type openAIReq struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *Plugin) callOpenAI(apiKey, baseURL, model, title, content string) (summary string, keyPoints []string, err error) {
	prompt := "Summarize the following in 2-3 sentences and list 2-4 key points as bullet lines. Title: " + title + "\n\nContent: " + truncate(content, 4000)
	text, err := p.chatCompletion(apiKey, baseURL, openAIReq{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", nil, err
	}
	parts := strings.Split(text, "\n")
	for _, line := range parts {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			keyPoints = append(keyPoints, strings.TrimLeft(line, "-* "))
		} else if summary == "" {
			summary = line
		} else {
			summary += " " + line
		}
	}
	if summary == "" && len(parts) > 0 {
		summary = text
	}
	return summary, keyPoints, nil
}

func (p *Plugin) callOpenAIMermaid(apiKey, baseURL, model, summary string) (string, error) {
	prompt := "Output ONLY a ```mermaid code block (flowchart or mindmap) that visualizes the summary below. Keep under 15 nodes.\n\nSummary:\n" + truncate(summary, 2000)
	return p.chatCompletion(apiKey, baseURL, openAIReq{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	})
}

func (p *Plugin) chatCompletion(apiKey, baseURL string, reqBody openAIReq) (string, error) {
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai chat: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var r openAIResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", err
	}
	if r.Error != nil && strings.TrimSpace(r.Error.Message) != "" {
		return "", fmt.Errorf("openai chat: %s", strings.TrimSpace(r.Error.Message))
	}
	if len(r.Choices) == 0 {
		return "", nil
	}
	return strings.TrimSpace(r.Choices[0].Message.Content), nil
}
