package core

import (
	"Tracker/internal/plugin"
	"sync"
)

// PluginManager maintains plugin registration and lifecycle. No business logic.
type PluginManager struct {
	mu      sync.RWMutex
	byType  map[plugin.Type][]plugin.Plugin
	byName  map[string]plugin.Plugin
}

// NewPluginManager creates a new plugin manager.
func NewPluginManager() *PluginManager {
	return &PluginManager{
		byType: make(map[plugin.Type][]plugin.Plugin),
		byName: make(map[string]plugin.Plugin),
	}
}

// Register registers a plugin. Idempotent by name (last wins).
func (m *PluginManager) Register(p plugin.Plugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := p.Type()
	m.byName[p.Name()] = p
	// avoid duplicate in byType
	for i, existing := range m.byType[t] {
		if existing.Name() == p.Name() {
			m.byType[t][i] = p
			return
		}
	}
	m.byType[t] = append(m.byType[t], p)
}

// GetByType returns all plugins of the given type.
func (m *PluginManager) GetByType(t plugin.Type) []plugin.Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := m.byType[t]
	out := make([]plugin.Plugin, len(list))
	copy(out, list)
	return out
}

// GetByName returns a plugin by name, or nil.
func (m *PluginManager) GetByName(name string) plugin.Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.byName[name]
}

// ListAll returns all registered plugins (each name once).
func (m *PluginManager) ListAll() []plugin.Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	seen := make(map[string]bool)
	var out []plugin.Plugin
	for _, p := range m.byName {
		if !seen[p.Name()] {
			seen[p.Name()] = true
			out = append(out, p)
		}
	}
	return out
}

// InitAll initializes all registered plugins with the given global config.
// Stage-specific config is passed when running the pipeline; here we pass nil or shared config.
func (m *PluginManager) InitAll(global plugin.Config) error {
	m.mu.RLock()
	list := m.ListAll()
	m.mu.RUnlock()
	for _, p := range list {
		if err := p.Init(global); err != nil {
			return err
		}
	}
	return nil
}
