package x_fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
	"Tracker/plugins/sharedutil"
)

// Plugin uses X API v2 recent search (requires bearer token — set X_BEARER_TOKEN or stage config bearer_token).
type Plugin struct {
	client *http.Client
}

func New() plugin.Plugin {
	return &Plugin{client: &http.Client{Timeout: 30 * time.Second}}
}

func (p *Plugin) Name() string                 { return "x_fetch" }
func (p *Plugin) Version() string              { return "1.0" }
func (p *Plugin) Type() plugin.Type            { return plugin.TypeSource }
func (p *Plugin) Init(cfg plugin.Config) error { return nil }

// TestConfig validates bearer token access against the X recent search endpoint.
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	req, err := p.newSearchRequest(ctx, cfg, 10)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("x recent search: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (p *Plugin) ExecuteSource(cfg plugin.Config) ([]*model.Item, error) {
	bearer := sharedutil.StringWithEnv(cfg, "bearer_token", os.Getenv("X_BEARER_TOKEN"))
	if bearer == "" {
		return nil, &model.ItemError{Code: "x_auth", Message: "bearer_token or X_BEARER_TOKEN required (X API v2)"}
	}
	req, err := p.newSearchRequest(context.Background(), cfg, 0)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &model.ItemError{Code: "x_api", Message: string(body)}
	}
	var tw struct {
		Data []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &tw); err != nil {
		return nil, err
	}
	var items []*model.Item
	for _, t := range tw.Data {
		link := "https://twitter.com/i/web/status/" + t.ID
		items = append(items, &model.Item{
			Title: truncate(t.Text, 120), Content: t.Text, URL: link, Timestamp: time.Now().UTC(),
			Source: "x", Metadata: map[string]string{"tweet_id": t.ID},
		})
	}
	return items, nil
}

func (p *Plugin) newSearchRequest(ctx context.Context, cfg plugin.Config, defaultMax int) (*http.Request, error) {
	bearer := sharedutil.StringWithEnv(cfg, "bearer_token", os.Getenv("X_BEARER_TOKEN"))
	if bearer == "" {
		return nil, fmt.Errorf("bearer_token is empty")
	}
	query := strings.TrimSpace(plugin.GetString(cfg, "query"))
	if query == "" {
		query = "news -is:retweet lang:en"
	}
	max := sharedutil.Int(cfg, "max_results", defaultMax)
	if max <= 0 {
		max = 10
	}
	u := "https://api.twitter.com/2/tweets/search/recent?query=" + url.QueryEscape(query) + "&max_results=" + fmt.Sprintf("%d", max) + "&tweet.fields=created_at,author_id"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	return req, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
