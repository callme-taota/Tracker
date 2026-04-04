package cluster_kmeans

import (
	"math"
	"sync"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
	"Tracker/plugins/sharedutil"
)

// Plugin assigns an online k-means style cluster id using 3-D features (title len, content len, hour).
type Plugin struct {
	mu        sync.Mutex
	k         int
	centroids [][]float64
	counts    []int
}

func New() plugin.Plugin {
	return &Plugin{k: 3}
}

func (p *Plugin) Name() string      { return "cluster_kmeans" }
func (p *Plugin) Version() string   { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeProcessor }
func (p *Plugin) Init(cfg plugin.Config) error {
	if k := sharedutil.Int(cfg, "k", p.k); k > 0 {
		p.k = k
	}
	return nil
}

func feat(it *model.Item) []float64 {
	h := 0.0
	if !it.Timestamp.IsZero() {
		h = float64(it.Timestamp.Hour())
	}
	return []float64{float64(len(it.Title)), float64(len(it.Content)), h}
}

func dist(a, b []float64) float64 {
	var s float64
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return math.Sqrt(s)
}

func (p *Plugin) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) {
	if in == nil {
		return nil, nil
	}
	f := feat(in)
	k := sharedutil.Int(cfg, "k", p.k)
	if k <= 0 {
		k = p.k
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if k != p.k && k > 0 {
		p.k = k
	}
	if len(p.centroids) < p.k {
		c := make([]float64, len(f))
		copy(c, f)
		p.centroids = append(p.centroids, c)
		p.counts = append(p.counts, 1)
		out := *in
		if out.Extra == nil {
			out.Extra = make(map[string]interface{})
		}
		out.Extra["cluster_id"] = len(p.centroids) - 1
		return &out, nil
	}
	best := 0
	bestd := math.MaxFloat64
	for i, c := range p.centroids {
		d := dist(f, c)
		if d < bestd {
			bestd = d
			best = i
		}
	}
	n := p.counts[best]
	for j := range p.centroids[best] {
		p.centroids[best][j] = (p.centroids[best][j]*float64(n) + f[j]) / float64(n+1)
	}
	p.counts[best]++
	out := *in
	if out.Extra == nil {
		out.Extra = make(map[string]interface{})
	}
	out.Extra["cluster_id"] = best
	return &out, nil
}
