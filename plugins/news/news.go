package news

import (
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
	"github.com/mmcdole/gofeed"
)

// Preset RSS feeds for mainstream news (global + CN). Config can override via "feeds" or add via "extra_feeds".
var presetFeeds = map[string][]string{
	"global": {
		"https://www.theverge.com/rss/index.xml",
		"https://www.wired.com/feed/rss",
		"https://techcrunch.com/feed/",
		"https://feeds.bbci.co.uk/news/rss.xml",
		"https://rss.nytimes.com/services/xml/rss/nyt/World.xml",
		"https://rss.nytimes.com/services/xml/rss/nyt/Technology.xml",
		"https://feeds.feedburner.com/ArsTechnica",
		"https://www.reutersagency.com/feed/?best-topics=tech",
	},
	"cn": {
		"https://www.36kr.com/feed",
		"https://www.ithome.com/rss/",
		"https://rss.sina.com.cn/news/marquee/ddt.xml",
	},
}

// Plugin implements a news source with preset mainstream RSS feeds. Same capability as rss but with built-in presets.
type Plugin struct {
	feeds []string
}

// New returns a new news plugin instance.
func New() plugin.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return "news" }
func (p *Plugin) Version() string { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeSource }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.feeds = buildFeeds(cfg)
	return nil
}

func buildFeeds(cfg plugin.Config) []string {
	// Explicit feeds override
	if f := plugin.GetStringSlice(cfg, "feeds"); len(f) > 0 {
		return f
	}
	var out []string
	presets := plugin.GetStringSlice(cfg, "presets")
	if len(presets) == 0 {
		presets = []string{"global", "cn"}
	}
	seen := make(map[string]bool)
	for _, name := range presets {
		for _, url := range presetFeeds[name] {
			if !seen[url] {
				seen[url] = true
				out = append(out, url)
			}
		}
	}
	for _, url := range plugin.GetStringSlice(cfg, "extra_feeds") {
		if !seen[url] {
			seen[url] = true
			out = append(out, url)
		}
	}
	return out
}

// ExecuteSource fetches items from preset or configured news RSS feeds.
func (p *Plugin) ExecuteSource(cfg plugin.Config) ([]*model.Item, error) {
	feeds := plugin.GetStringSlice(cfg, "feeds")
	if len(feeds) == 0 {
		feeds = p.feeds
	}
	if len(feeds) == 0 {
		feeds = buildFeeds(cfg)
	}
	if len(feeds) == 0 {
		return nil, &model.ItemError{Code: "no_feeds", Message: "no news feeds configured (set presets: [global, cn] or feeds)"}
	}
	fp := gofeed.NewParser()
	var items []*model.Item
	for _, url := range feeds {
		feed, err := fp.ParseURL(url)
		if err != nil {
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
	return items, nil
}
