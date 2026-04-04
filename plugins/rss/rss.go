package rss

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"

	"github.com/mmcdole/gofeed"
)

// Plugin implements RSS source plugin.
type Plugin struct {
	feeds     []string
	userAgent string
}

// New returns a new RSS plugin instance.
func New() plugin.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string      { return "rss" }
func (p *Plugin) Version() string   { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeSource }

func (p *Plugin) Init(cfg plugin.Config) error {
	// feeds can be set in Init (global) or in ExecuteSource (stage config)
	p.feeds = plugin.GetStringSlice(cfg, "feeds")
	p.userAgent = plugin.GetString(cfg, "user_agent")
	return nil
}

// TestConfig validates that at least one configured RSS feed is reachable.
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	feeds := plugin.GetStringSlice(cfg, "feeds")
	if len(feeds) == 0 {
		feeds = p.feeds
	}
	if len(feeds) == 0 {
		return fmt.Errorf("no RSS feeds configured")
	}
	parser := gofeed.NewParser()
	ua := plugin.GetString(cfg, "user_agent")
	if ua == "" {
		ua = p.userAgent
	}
	if ua == "" {
		ua = "Tracker-RSS/1.0"
	}
	parser.UserAgent = ua
	for _, url := range feeds {
		feed, err := parser.ParseURLWithContext(url, ctx)
		if err == nil && feed != nil {
			return nil
		}
	}
	return fmt.Errorf("failed to load all configured RSS feeds")
}

// ExecuteSource fetches items from RSS feeds. cfg["feeds"] overrides Init feeds.
func (p *Plugin) ExecuteSource(cfg plugin.Config) ([]*model.Item, error) {
	feeds := plugin.GetStringSlice(cfg, "feeds")
	if len(feeds) == 0 {
		feeds = p.feeds
	}
	if len(feeds) == 0 {
		return nil, &model.ItemError{Code: "no_feeds", Message: "no RSS feeds configured"}
	}
	fp := gofeed.NewParser()
	ua := plugin.GetString(cfg, "user_agent")
	if ua == "" {
		ua = p.userAgent
	}
	if ua == "" {
		ua = "Tracker-RSS/1.0"
	}
	fp.UserAgent = ua
	var items []*model.Item
	failures := 0
	for _, url := range feeds {
		feed, err := fp.ParseURL(url)
		if err != nil {
			failures++
			continue
		}
		for _, i := range feed.Items {
			var t time.Time
			if i.PublishedParsed != nil {
				t = *i.PublishedParsed
			} else if i.UpdatedParsed != nil {
				t = *i.UpdatedParsed
			}
			content := i.Content
			if content == "" {
				content = i.Description
			}
			items = append(items, &model.Item{
				Title:     i.Title,
				Content:   content,
				URL:       i.Link,
				Timestamp: t,
				Source:    feed.Title,
				Metadata:  map[string]string{"feed_url": url},
			})
		}
	}
	if len(items) == 0 && failures == len(feeds) {
		return nil, &model.ItemError{Code: "rss_fetch_failed", Message: "failed to load all configured RSS feeds"}
	}
	return filterByTimeWindow(items, cfg), nil
}

func filterByTimeWindow(items []*model.Item, cfg plugin.Config) []*model.Item {
	fromS := strings.TrimSpace(plugin.GetString(cfg, "time_from"))
	toS := strings.TrimSpace(plugin.GetString(cfg, "time_to"))
	if fromS == "" && toS == "" {
		return items
	}
	var fromT, toT time.Time
	var err error
	if fromS != "" {
		fromT, err = time.Parse(time.RFC3339, fromS)
		if err != nil {
			return items
		}
	}
	if toS != "" {
		toT, err = time.Parse(time.RFC3339, toS)
		if err != nil {
			return items
		}
	}
	var out []*model.Item
	for _, it := range items {
		ts := it.Timestamp
		if ts.IsZero() {
			out = append(out, it)
			continue
		}
		if !fromT.IsZero() && ts.Before(fromT) {
			continue
		}
		if !toT.IsZero() && ts.After(toT) {
			continue
		}
		out = append(out, it)
	}
	return out
}
