package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
	"Tracker/plugins/sharedutil"
)

// Plugin fetches posts from subreddits via Reddit's public JSON API (respect rate limits; User-Agent required).
type Plugin struct {
	client *http.Client
}

func New() plugin.Plugin {
	return &Plugin{client: &http.Client{Timeout: 30 * time.Second}}
}

func (p *Plugin) Name() string                 { return "reddit" }
func (p *Plugin) Version() string              { return "1.0" }
func (p *Plugin) Type() plugin.Type            { return plugin.TypeSource }
func (p *Plugin) Init(cfg plugin.Config) error { return nil }

// TestConfig validates subreddit access using a small public JSON request.
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	req, err := p.newRequest(ctx, cfg, 1)
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
		return fmt.Errorf("reddit api: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (p *Plugin) ExecuteSource(cfg plugin.Config) ([]*model.Item, error) {
	subs := configuredSubs(cfg)
	if len(subs) == 0 {
		return nil, &model.ItemError{Code: "reddit_config", Message: "subreddits or subreddit required"}
	}
	var all []*model.Item
	for _, sub := range subs {
		req, err := p.newSubredditRequest(context.Background(), cfg, sub, 0)
		if err != nil {
			continue
		}
		resp, err := p.client.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			continue
		}
		var listing struct {
			Data struct {
				Children []struct {
					Data struct {
						Title     string  `json:"title"`
						URL       string  `json:"url"`
						Selftext  string  `json:"selftext"`
						Permalink string  `json:"permalink"`
						Created   float64 `json:"created_utc"`
					} `json:"data"`
				} `json:"children"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &listing); err != nil {
			continue
		}
		for _, ch := range listing.Data.Children {
			d := ch.Data
			link := d.URL
			if !strings.HasPrefix(link, "http") {
				link = "https://reddit.com" + d.Permalink
			}
			content := d.Selftext
			if content == "" {
				content = d.Title
			}
			ts := time.Unix(int64(d.Created), 0).UTC()
			all = append(all, &model.Item{
				Title: d.Title, Content: content, URL: link, Timestamp: ts,
				Source:   "reddit/r/" + sub,
				Metadata: map[string]string{"subreddit": sub},
			})
		}
	}
	return all, nil
}

func configuredSubs(cfg plugin.Config) []string {
	subs := plugin.GetStringSlice(cfg, "subreddits")
	if len(subs) == 0 {
		if s := plugin.GetString(cfg, "subreddit"); s != "" {
			subs = []string{s}
		}
	}
	return subs
}

func (p *Plugin) newRequest(ctx context.Context, cfg plugin.Config, forceLimit int) (*http.Request, error) {
	subs := configuredSubs(cfg)
	if len(subs) == 0 {
		return nil, fmt.Errorf("subreddits or subreddit required")
	}
	return p.newSubredditRequest(ctx, cfg, subs[0], forceLimit)
}

func (p *Plugin) newSubredditRequest(ctx context.Context, cfg plugin.Config, sub string, forceLimit int) (*http.Request, error) {
	limit := forceLimit
	if limit <= 0 {
		limit = sharedutil.Int(cfg, "limit", 15)
	}
	if limit <= 0 {
		limit = 15
	}
	ua := strings.TrimSpace(plugin.GetString(cfg, "user_agent"))
	if ua == "" {
		ua = "Tracker/1.0 (information bot)"
	}
	sub = strings.TrimPrefix(strings.TrimSpace(sub), "r/")
	url := fmt.Sprintf("https://www.reddit.com/r/%s/new.json?limit=%d", sub, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	return req, nil
}
