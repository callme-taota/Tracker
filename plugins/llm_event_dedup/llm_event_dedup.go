package llm_event_dedup

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
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

func (p *Plugin) Name() string             { return "llm_event_dedup" }
func (p *Plugin) Version() string          { return "1.0" }
func (p *Plugin) Type() plugin.Type        { return plugin.TypeProcessor }

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
	key := plugin.GetString(cfg, "api_key")
	if key == "" {
		key = p.apiKey
	}
	if key == "" {
		cp := *in
		p.last = &cp
		return in, nil
	}
	dup, err := p.askDuplicate(key, p.last, in)
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

func (p *Plugin) askDuplicate(apiKey string, a, b *model.Item) (bool, error) {
	prompt := "Are these two news items likely the SAME event? Reply only YES or NO.\nA: " + a.Title + " " + a.URL + "\nB: " + b.Title + " " + b.URL
	body, _ := json.Marshal(map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.0,
	})
	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
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
