package embedding_dedup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
	"Tracker/plugins/sharedutil"
)

// Plugin calls OpenAI-compatible embeddings and drops items too similar to prior vectors (cosine >= threshold).
type Plugin struct {
	mu         sync.Mutex
	apiKey     string
	model      string
	baseURL    string
	threshold  float64
	vectors    [][]float64
	httpClient *http.Client
}

func New() plugin.Plugin {
	return &Plugin{
		model: "text-embedding-3-small", threshold: 0.92,
		baseURL: "https://api.openai.com/v1", httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *Plugin) Name() string      { return "embedding_dedup" }
func (p *Plugin) Version() string   { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeProcessor }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.apiKey = sharedutil.StringWithEnv(cfg, "api_key", os.Getenv("OPENAI_API_KEY"))
	if m := strings.TrimSpace(plugin.GetString(cfg, "model")); m != "" {
		p.model = m
	}
	if b := strings.TrimSpace(plugin.GetString(cfg, "base_url")); b != "" {
		p.baseURL = strings.TrimRight(b, "/")
	}
	if threshold := sharedutil.Float(cfg, "threshold", p.threshold); threshold > 0 && threshold <= 1 {
		p.threshold = threshold
	}
	return nil
}

// TestConfig validates the OpenAI-compatible embedding endpoint credentials.
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	key, baseURL := p.resolveConfig(cfg)
	if err := sharedutil.RequireFields(map[string]string{"api_key": key, "base_url": baseURL}); err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]interface{}{
		"model": p.resolveModel(cfg),
		"input": "tracker connectivity test",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("embedding api: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (p *Plugin) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) {
	if in == nil {
		return nil, nil
	}
	key, _ := p.resolveConfig(cfg)
	if key == "" {
		out := *in
		return &out, nil
	}
	text := in.Title + "\n" + in.URL + "\n" + truncate(in.Content, 6000)
	vec, err := p.embed(key, p.resolveBaseURL(cfg), p.resolveModel(cfg), text)
	if err != nil || len(vec) == 0 {
		out := *in
		return &out, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	threshold := p.resolveThreshold(cfg)
	for _, prev := range p.vectors {
		if cosine(prev, vec) >= threshold {
			return nil, nil
		}
	}
	cp := make([]float64, len(vec))
	copy(cp, vec)
	p.vectors = append(p.vectors, cp)
	out := *in
	if out.Extra == nil {
		out.Extra = make(map[string]interface{})
	}
	out.Extra["embedding_dim"] = len(vec)
	return &out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func (p *Plugin) resolveConfig(cfg plugin.Config) (apiKey, baseURL string) {
	apiKey = sharedutil.StringWithEnv(cfg, "api_key", p.apiKey)
	baseURL = p.resolveBaseURL(cfg)
	return apiKey, baseURL
}

func (p *Plugin) resolveBaseURL(cfg plugin.Config) string {
	if b := strings.TrimSpace(plugin.GetString(cfg, "base_url")); b != "" {
		return strings.TrimRight(b, "/")
	}
	return strings.TrimRight(p.baseURL, "/")
}

func (p *Plugin) resolveModel(cfg plugin.Config) string {
	if m := strings.TrimSpace(plugin.GetString(cfg, "model")); m != "" {
		return m
	}
	return p.model
}

func (p *Plugin) resolveThreshold(cfg plugin.Config) float64 {
	threshold := sharedutil.Float(cfg, "threshold", p.threshold)
	if threshold <= 0 || threshold > 1 {
		return p.threshold
	}
	return threshold
}

func (p *Plugin) embed(apiKey, baseURL, model, input string) ([]float64, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"model": model, "input": input,
	})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var er struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &er); err != nil {
		return nil, err
	}
	if len(er.Data) == 0 {
		return nil, nil
	}
	return er.Data[0].Embedding, nil
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
