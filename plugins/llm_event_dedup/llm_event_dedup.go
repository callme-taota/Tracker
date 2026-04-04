package llm_event_dedup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
	"Tracker/plugins/sharedutil"
)

// Plugin asks a small chat model if the current item duplicates the previous event.
type Plugin struct {
	mu      sync.Mutex
	last    *model.Item
	client  *http.Client
	apiKey  string
	model   string
	baseURL string
}

func New() plugin.Plugin {
	return &Plugin{
		client:  &http.Client{Timeout: 45 * time.Second},
		model:   "gpt-4o-mini",
		baseURL: "https://api.openai.com/v1",
	}
}

func (p *Plugin) Name() string      { return "llm_event_dedup" }
func (p *Plugin) Version() string   { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeProcessor }

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

// TestConfig validates the configured OpenAI-compatible chat endpoint.
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	apiKey, baseURL := p.resolveConfig(cfg)
	if err := sharedutil.RequireFields(map[string]string{"api_key": apiKey, "base_url": baseURL}); err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]interface{}{
		"model": p.resolveModel(cfg),
		"messages": []map[string]string{
			{"role": "user", "content": "Reply with YES"},
		},
		"temperature": 0.0,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("llm_event_dedup chat: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (p *Plugin) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) {
	if in == nil {
		return nil, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.last == nil {
		cp := *in
		p.last = &cp
		return in, nil
	}
	key, _ := p.resolveConfig(cfg)
	if key == "" {
		cp := *in
		p.last = &cp
		return in, nil
	}
	dup, err := p.askDuplicate(key, p.resolveBaseURL(cfg), p.resolveModel(cfg), p.last, in)
	if err != nil {
		cp := *in
		p.last = &cp
		return in, nil
	}
	if dup {
		return nil, nil
	}
	cp := *in
	p.last = &cp
	return in, nil
}

func (p *Plugin) resolveConfig(cfg plugin.Config) (apiKey, baseURL string) {
	apiKey = sharedutil.StringWithEnv(cfg, "api_key", p.apiKey)
	baseURL = p.resolveBaseURL(cfg)
	return apiKey, baseURL
}

func (p *Plugin) resolveBaseURL(cfg plugin.Config) string {
	if v := strings.TrimSpace(plugin.GetString(cfg, "base_url")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return strings.TrimRight(p.baseURL, "/")
}

func (p *Plugin) resolveModel(cfg plugin.Config) string {
	if m := strings.TrimSpace(plugin.GetString(cfg, "model")); m != "" {
		return m
	}
	return p.model
}

func (p *Plugin) askDuplicate(apiKey, baseURL, modelName string, a, b *model.Item) (bool, error) {
	prompt := "Are these two news items likely the SAME event? Reply only YES or NO.\nA: " + a.Title + " " + a.URL + "\nB: " + b.Title + " " + b.URL
	body, _ := json.Marshal(map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.0,
	})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	bb, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	var cr struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(bb, &cr); err != nil {
		return false, err
	}
	if len(cr.Choices) == 0 {
		return false, nil
	}
	ans := strings.ToUpper(strings.TrimSpace(cr.Choices[0].Message.Content))
	return strings.HasPrefix(ans, "Y"), nil
}
