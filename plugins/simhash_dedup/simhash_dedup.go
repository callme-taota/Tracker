package simhash_dedup

import (
	"hash/fnv"
	"regexp"
	"strings"
	"sync"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// Plugin drops near-duplicate content using 64-bit simhash + Hamming threshold.
type Plugin struct {
	mu       sync.Mutex
	hashes   []uint64
	threshold int
}

func New() plugin.Plugin {
	return &Plugin{threshold: 3}
}

func (p *Plugin) Name() string             { return "simhash_dedup" }
func (p *Plugin) Version() string          { return "1.0" }
func (p *Plugin) Type() plugin.Type        { return plugin.TypeProcessor }
func (p *Plugin) Init(cfg plugin.Config) error {
	if plugin.GetString(cfg, "hamming_threshold") == "4" {
		p.threshold = 4
	}
	return nil
}

func (p *Plugin) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) {
	if in == nil {
		return nil, nil
	}
	h := simhash64(in.Title + "\n" + in.Content)
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, prev := range p.hashes {
		if hamming(prev, h) <= uint32(p.threshold) {
			return nil, nil
		}
	}
	p.hashes = append(p.hashes, h)
	out := *in
	if out.Extra == nil {
		out.Extra = make(map[string]interface{})
	}
	out.Extra["simhash"] = h
	return &out, nil
}

var wordRe = regexp.MustCompile(`[a-zA-Z0-9\x{4e00}-\x{9fff}]+`)

func tokenize(s string) []string {
	s = strings.ToLower(s)
	return wordRe.FindAllString(s, -1)
}

func fnv64(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

func simhash64(s string) uint64 {
	words := tokenize(s)
	if len(words) == 0 {
		return 0
	}
	var v [64]int
	for _, w := range words {
		x := fnv64(w)
		for b := 0; b < 64; b++ {
			if (x>>b)&1 == 1 {
				v[b]++
			} else {
				v[b]--
			}
		}
	}
	var out uint64
	for b := 0; b < 64; b++ {
		if v[b] >= 0 {
			out |= 1 << b
		}
	}
	return out
}

func hamming(a, b uint64) uint32 {
	x := a ^ b
	var n uint32
	for x != 0 {
		n++
		x &= x - 1
	}
	return n
}
