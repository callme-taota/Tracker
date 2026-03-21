package clean

import (
	"regexp"
	"strings"

	"Tracker/internal/core/format"
	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// Plugin implements content clean (strip HTML, normalize).
type Plugin struct{}

// New returns a new clean plugin instance.
func New() plugin.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return "clean" }
func (p *Plugin) Version() string { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeProcessor }

func (p *Plugin) Init(cfg plugin.Config) error {
	return nil
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)
var spaceRe = regexp.MustCompile(`\s+`)

// Execute strips HTML and normalizes whitespace.
func (p *Plugin) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) {
	if in == nil {
		return nil, nil
	}
	out := *in
	out.Content = stripHTML(in.Content)
	out.Content = spaceRe.ReplaceAllString(strings.TrimSpace(out.Content), " ")
	out.Content = format.NormalizeMarkdown(out.Content)
	out.Content = format.StripInvisible(out.Content)
	if out.Content == "" && in.Summary != "" {
		out.Content = stripHTML(in.Summary)
	}
	return &out, nil
}

func stripHTML(s string) string {
	return htmlTagRe.ReplaceAllString(s, " ")
}
