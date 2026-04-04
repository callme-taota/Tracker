package github_trending

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

// Plugin searches GitHub repositories (no single "trending" API — uses search sorted by stars/updated).
type Plugin struct {
	client *http.Client
}

func New() plugin.Plugin {
	return &Plugin{client: &http.Client{Timeout: 30 * time.Second}}
}

func (p *Plugin) Name() string                 { return "github_trending" }
func (p *Plugin) Version() string              { return "1.0" }
func (p *Plugin) Type() plugin.Type            { return plugin.TypeSource }
func (p *Plugin) Init(cfg plugin.Config) error { return nil }

// TestConfig validates that GitHub search is reachable with the provided query/token.
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	req, err := p.newSearchRequest(ctx, cfg, 1)
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
		return fmt.Errorf("github search: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (p *Plugin) ExecuteSource(cfg plugin.Config) ([]*model.Item, error) {
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
		return nil, &model.ItemError{Code: "github_api", Message: string(body)}
	}
	var out struct {
		Items []struct {
			FullName    string    `json:"full_name"`
			HTMLURL     string    `json:"html_url"`
			Description string    `json:"description"`
			Language    string    `json:"language"`
			PushedAt    time.Time `json:"pushed_at"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	var items []*model.Item
	for _, it := range out.Items {
		desc := it.Description
		if desc == "" {
			desc = it.Language
		}
		items = append(items, &model.Item{
			Title: it.FullName, Content: desc, URL: it.HTMLURL, Timestamp: it.PushedAt,
			Source: "github", Metadata: map[string]string{"language": it.Language},
		})
	}
	return items, nil
}

func (p *Plugin) newSearchRequest(ctx context.Context, cfg plugin.Config, forcePerPage int) (*http.Request, error) {
	q := strings.TrimSpace(plugin.GetString(cfg, "query"))
	if q == "" {
		q = "created:>" + time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}
	sort := strings.TrimSpace(plugin.GetString(cfg, "sort"))
	if sort == "" {
		sort = "stars"
	}
	perPage := sharedutil.Int(cfg, "per_page", 20)
	if forcePerPage > 0 {
		perPage = forcePerPage
	}
	if perPage <= 0 {
		perPage = 20
	}
	v := url.Values{}
	v.Set("q", q)
	v.Set("sort", sort)
	v.Set("per_page", fmt.Sprintf("%d", perPage))
	apiURL := "https://api.github.com/search/repositories?" + v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := sharedutil.StringWithEnv(cfg, "token", os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}
