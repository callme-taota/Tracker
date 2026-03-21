package dedup_basic

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"sync"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// Plugin drops items already seen by URL/title/content fingerprint (in-memory per process).
type Plugin struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func New() plugin.Plugin {
	return &Plugin{seen: make(map[string]struct{})}
}

func (p *Plugin) Name() string             { return "dedup_basic" }
func (p *Plugin) Version() string          { return "1.0" }
func (p *Plugin) Type() plugin.Type        { return plugin.TypeProcessor }
func (p *Plugin) Init(cfg plugin.Config) error { return nil }

func (p *Plugin) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) {
	if in == nil {
		return nil, nil
	}
	key := fingerprint(in)
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.seen[key]; ok {
		return nil, nil
	}
	p.seen[key] = struct{}{}
	out := *in
	if out.Extra == nil {
		out.Extra = make(map[string]interface{})
	}
	out.Extra["dedup_key"] = key
	return &out, nil
}

func fingerprint(in *model.Item) string {
	u := strings.TrimSpace(strings.ToLower(in.URL))
	if parsed, err := url.Parse(u); err == nil {
		parsed.Fragment = ""
		u = parsed.String()
	}
	title := strings.TrimSpace(strings.ToLower(in.Title))
	body := strings.TrimSpace(in.Content)
	if len(body) > 4096 {
		body = body[:4096]
	}
	h := sha256.Sum256([]byte(u + "\n" + title + "\n" + body))
	return hex.EncodeToString(h[:])
}
