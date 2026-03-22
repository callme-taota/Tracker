package pluginhub

import (
	"context"
	"fmt"
	"sync"

	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
)

// RuntimeContext carries per-run metadata for lifecycle hooks and plugins (Core services added in later phases).
type RuntimeContext struct {
	Ctx        context.Context
	RunID      string
	PipelineID int64
	NodeID     string
	PluginID   string
	JobID      string
	// Core will be *core.CoreServices or similar in phase 2; keep opaque until then.
	Core interface{}
	// Payload is optional per-run parameters (e.g. time window for sources).
	Payload map[string]interface{}
}

// EventType is emitted by the hub during pipeline execution.
type EventType int

const (
	PipelineStart EventType = iota
	NodeStart
	NodeEnd
	PipelineEnd
	ErrorEvent
)

// Event is a lifecycle notification.
type Event struct {
	Type       EventType
	RunID      string
	PipelineID int64
	NodeID     string
	PluginID   string
	Err        error
	ItemsIn    int
	ItemsOut   int
}

// Subscriber receives lifecycle events after a run starts.
type Subscriber func(rt *RuntimeContext, ev Event)

// ValidatorFunc validates plugin stage config (e.g. RSS URL reachability). Registered by name (ValidatorRef).
type ValidatorFunc func(cfg plugin.Config) error

// Hub holds plugin manifests, optional config validators, and lifecycle subscribers.
type Hub struct {
	mu          sync.RWMutex
	manifests   map[string]plugin.Manifest
	validators  map[string]ValidatorFunc
	subscribers []Subscriber
}

// New creates an empty hub.
func New() *Hub {
	return &Hub{
		manifests:  make(map[string]plugin.Manifest),
		validators: make(map[string]ValidatorFunc),
	}
}

// RegisterManifest registers or replaces manifest for plugin id (must equal plugin.Name()).
func (h *Hub) RegisterManifest(m plugin.Manifest) {
	if m.ID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.manifests[m.ID] = m
}

// RemoveManifest deletes a manifest by plugin id.
func (h *Hub) RemoveManifest(pluginID string) {
	if pluginID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.manifests, pluginID)
}

// Manifest returns manifest for plugin id, or ok=false.
func (h *Hub) Manifest(pluginID string) (plugin.Manifest, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m, ok := h.manifests[pluginID]
	return m, ok
}

// RegisterValidator registers a named config validator (Manifest.ValidatorRef).
func (h *Hub) RegisterValidator(name string, fn ValidatorFunc) {
	if name == "" || fn == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.validators[name] = fn
}

// ValidateConfig runs ValidatorRef against cfg if set.
func (h *Hub) ValidateConfig(m plugin.Manifest, cfg plugin.Config) error {
	if m.ValidatorRef == "" {
		return nil
	}
	h.mu.RLock()
	fn := h.validators[m.ValidatorRef]
	h.mu.RUnlock()
	if fn == nil {
		return fmt.Errorf("pluginhub: unknown validator %q", m.ValidatorRef)
	}
	return fn(cfg)
}

// Subscribe adds a lifecycle subscriber (order preserved).
func (h *Hub) Subscribe(fn Subscriber) {
	if fn == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subscribers = append(h.subscribers, fn)
}

// Emit invokes all subscribers (panics in subscriber are not recovered — keep handlers light).
func (h *Hub) Emit(rt *RuntimeContext, ev Event) {
	h.mu.RLock()
	subs := append([]Subscriber(nil), h.subscribers...)
	h.mu.RUnlock()
	for _, fn := range subs {
		fn(rt, ev)
	}
}

// ValidatePipelineGraph checks DAG structure, per-node config schema (lightweight), format compatibility on edges, and optional validators.
func (h *Hub) ValidatePipelineGraph(g *pipeline.PipelineGraph) error {
	if err := g.Validate(); err != nil {
		return err
	}
	nodes := g.NodeByID()
	for _, n := range g.Nodes {
		m, ok := h.Manifest(n.PluginID)
		if !ok {
			return fmt.Errorf("pluginhub: no manifest for plugin %q (node %s)", n.PluginID, n.ID)
		}
		if m.Kind == plugin.TypeOperator {
			return fmt.Errorf("pluginhub: plugin %q (kind operator) cannot be used in pipeline graphs (node %s)", n.PluginID, n.ID)
		}
		if err := ValidateConfigJSONSchema(m.ConfigSchema, n.Config); err != nil {
			return fmt.Errorf("node %s (%s): %w", n.ID, n.PluginID, err)
		}
		if err := h.ValidateConfig(m, n.Config); err != nil {
			return fmt.Errorf("node %s (%s): %w", n.ID, n.PluginID, err)
		}
	}
	for _, e := range g.Edges {
		src, ok1 := nodes[e.Source]
		tgt, ok2 := nodes[e.Target]
		if !ok1 || !ok2 {
			continue
		}
		ms, okS := h.Manifest(src.PluginID)
		mt, okT := h.Manifest(tgt.PluginID)
		if !okS || !okT {
			continue
		}
		if !FormatsCompatible(ms.OutputFormats, mt.InputFormats) {
			return fmt.Errorf("pluginhub: edge %s -> %s: output formats %v incompatible with input formats %v",
				e.Source, e.Target, ms.OutputFormats, mt.InputFormats)
		}
	}
	return nil
}
