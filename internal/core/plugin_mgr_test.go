package core

import (
	"testing"

	"Tracker/internal/plugin"
)

type mockPlugin struct {
	name, version string
	typ          plugin.Type
}

func (m mockPlugin) Name() string    { return m.name }
func (m mockPlugin) Version() string { return m.version }
func (m mockPlugin) Type() plugin.Type { return m.typ }
func (m mockPlugin) Init(cfg plugin.Config) error { return nil }

func TestPluginManager_RegisterGetByType(t *testing.T) {
	pm := NewPluginManager()
	pm.Register(mockPlugin{name: "rss", version: "1", typ: plugin.TypeSource})
	pm.Register(mockPlugin{name: "clean", version: "1", typ: plugin.TypeProcessor})
	list := pm.GetByType(plugin.TypeSource)
	if len(list) != 1 {
		t.Fatalf("GetByType(source) want 1 got %d", len(list))
	}
	if list[0].Name() != "rss" {
		t.Errorf("want rss got %s", list[0].Name())
	}
}

func TestPluginManager_GetByName(t *testing.T) {
	pm := NewPluginManager()
	pm.Register(mockPlugin{name: "rss", version: "1", typ: plugin.TypeSource})
	p := pm.GetByName("rss")
	if p == nil {
		t.Fatal("GetByName(rss) want non-nil")
	}
	if p.Name() != "rss" {
		t.Errorf("want rss got %s", p.Name())
	}
	if pm.GetByName("missing") != nil {
		t.Error("GetByName(missing) want nil")
	}
}
