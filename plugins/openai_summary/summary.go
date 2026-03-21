package openai_summary

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// Plugin implements OpenAI-based summarization.
type Plugin struct {
	apiKey string
	model  string
}

// New returns a new openai_summary plugin instance.
func New() plugin.Plugin {
	return &Plugin{model: "gpt-3.5-turbo"}
}

func (p *Plugin) Name() string    { return "openai_summary" }
func (p *Plugin) Version() string { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeSummary }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.apiKey = plugin.GetString(cfg, "api_key")
	if p.apiKey == "" {
		p.apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if m := plugin.GetString(cfg, "model"); m != "" {
		p.model = m
	}
	return nil
}

// Execute generates summary and key points for the item.
func (p *Plugin) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) {
	if in == nil {
		return nil, nil
	}
	apiKey := plugin.GetString(cfg, "api_key")
	if apiKey == "" {
		apiKey = p.apiKey
	}
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		out := *in
		out.Summary = truncate(in.Content, 300)
		return &out, nil
	}
	modelName := plugin.GetString(cfg, "model")
	if modelName == "" {
		modelName = p.model
	}
	summary, keyPoints, err := callOpenAI(apiKey, modelName, in.Title, in.Content)
	if err != nil {
		out := *in
		out.Summary = truncate(in.Content, 300)
		return &out, nil
	}
	out := *in
	out.Summary = summary
	out.KeyPoints = keyPoints
	im := plugin.GetString(cfg, "include_mermaid")
	if im == "true" || im == "1" {
		if m, err := callOpenAIMermaid(apiKey, modelName, summary); err == nil && strings.TrimSpace(m) != "" {
			out.Summary = summary + "\n\n---\n" + m
			if out.Extra == nil {
				out.Extra = make(map[string]interface{})
			}
			out.Extra["mermaid"] = m
		}
	}
	return &out, nil
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
}

func callOpenAI(apiKey, model, title, content string) (summary string, keyPoints []string, err error) {
	prompt := "Summarize the following in 2-3 sentences and list 2-4 key points as bullet lines. Title: " + title + "\n\nContent: " + truncate(content, 4000)
	body, _ := json.Marshal(openAIReq{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	})
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	var r openAIResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", nil, err
	}
	if len(r.Choices) == 0 {
		return "", nil, nil
	}
	text := r.Choices[0].Message.Content
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

func callOpenAIMermaid(apiKey, model, summary string) (string, error) {
	prompt := "Output ONLY a ```mermaid code block (flowchart or mindmap) that visualizes the summary below. Keep under 15 nodes.\n\nSummary:\n" + truncate(summary, 2000)
	body, _ := json.Marshal(openAIReq{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	})
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var r openAIResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	if len(r.Choices) == 0 {
		return "", nil
	}
	return strings.TrimSpace(r.Choices[0].Message.Content), nil
}
