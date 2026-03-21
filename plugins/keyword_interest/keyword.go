package keyword_interest

import (
	"strings"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// Plugin implements keyword-based interest scoring.
type Plugin struct {
	keywords []string
}

// New returns a new keyword interest plugin instance.
func New() plugin.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return "keyword_interest" }
func (p *Plugin) Version() string { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeInterest }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.keywords = plugin.GetStringSlice(cfg, "keywords")
	return nil
}

// Execute scores the item by keyword matches and sets MatchedTopics.
func (p *Plugin) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) {
	if in == nil {
		return nil, nil
	}
	keywords := plugin.GetStringSlice(cfg, "keywords")
	if len(keywords) == 0 {
		keywords = p.keywords
	}
	out := *in
	text := strings.ToLower(in.Title + " " + in.Content + " " + in.Summary)
	var matched []string
	for _, k := range keywords {
		if strings.Contains(text, strings.ToLower(k)) {
			matched = append(matched, k)
		}
	}
	out.MatchedTopics = matched
	if len(matched) > 0 {
		out.InterestScore = float64(len(matched)) / float64(len(keywords)+1)
	}
	return &out, nil
}
