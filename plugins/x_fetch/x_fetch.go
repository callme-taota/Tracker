package x_fetch

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// Plugin uses X API v2 recent search (requires bearer token — set X_BEARER_TOKEN or stage config bearer_token).
type Plugin struct {
	client *http.Client
}

func New() plugin.Plugin {
	return &Plugin{client: &http.Client{Timeout: 30 * time.Second}}
}

func (p *Plugin) Name() string             { return "x_fetch" }
func (p *Plugin) Version() string          { return "1.0" }
func (p *Plugin) Type() plugin.Type        { return plugin.TypeSource }
func (p *Plugin) Init(cfg plugin.Config) error { return nil }

func (p *Plugin) ExecuteSource(cfg plugin.Config) ([]*model.Item, error) {
	bearer := plugin.GetString(cfg, "bearer_token")
	if bearer == "" {
		bearer = os.Getenv("X_BEARER_TOKEN")
	}
	if bearer == "" {
		return nil, &model.ItemError{Code: "x_auth", Message: "bearer_token or X_BEARER_TOKEN required (X API v2)"}
	}
	query := plugin.GetString(cfg, "query")
	if query == "" {
		query = "news -is:retweet lang:en"
	}
	max := plugin.GetString(cfg, "max_results")
	if max == "" {
		max = "10"
	}
	u := "https://api.twitter.com/2/tweets/search/recent?query=" + url.QueryEscape(query) + "&max_results=" + max + "&tweet.fields=created_at,author_id"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
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

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
