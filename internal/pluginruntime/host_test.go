package pluginruntime

import (
	"context"
	"sync"
	"testing"

	"Tracker/internal/plugin"
	"Tracker/internal/pluginhub"
)

type recordingPM struct {
	mu           sync.Mutex
	registered   []string
	unregistered []string
}

func (r *recordingPM) Register(p plugin.Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registered = append(r.registered, p.Name())
}

func (r *recordingPM) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unregistered = append(r.unregistered, name)
}

func TestHost_Load_EmptyPathNoOps(t *testing.T) {
	pm := &recordingPM{}
	hub := pluginhub.New()
	h := NewHost(pm, hub, "", nil)
	if err := h.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(pm.registered) != 0 {
		t.Fatalf("registered %v", pm.registered)
	}
}

func TestHost_IsRemote_Unknown(t *testing.T) {
	h := NewHost(&recordingPM{}, pluginhub.New(), "", nil)
	if h.IsRemote("any") {
		t.Fatal("expected false")
	}
}

func TestHost_Health_UnknownPlugin(t *testing.T) {
	h := NewHost(&recordingPM{}, pluginhub.New(), "", nil)
	_, _, err := h.Health(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartPlugin_MissingID(t *testing.T) {
	h := NewHost(&recordingPM{}, pluginhub.New(), "", nil)
	err := h.startPlugin(context.Background(), PluginSpec{Command: []string{"true"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartPlugin_MissingCommand(t *testing.T) {
	h := NewHost(&recordingPM{}, pluginhub.New(), "", nil)
	err := h.startPlugin(context.Background(), PluginSpec{ID: "ext1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartPlugin_BuiltinConflict(t *testing.T) {
	h := NewHost(&recordingPM{}, pluginhub.New(), "", map[string]bool{"rss": true})
	err := h.startPlugin(context.Background(), PluginSpec{ID: "rss", Command: []string{"/bin/true"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHost_Shutdown_Empty(t *testing.T) {
	pm := &recordingPM{}
	h := NewHost(pm, pluginhub.New(), "", nil)
	h.Shutdown(context.Background())
	if len(pm.unregistered) != 0 {
		t.Fatalf("unexpected unregister %v", pm.unregistered)
	}
}

func TestHost_InitRemote_NoPlugins(t *testing.T) {
	h := NewHost(&recordingPM{}, pluginhub.New(), "", nil)
	if err := h.InitRemote(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if got := h.LastGlobalConfig(); got != nil {
		t.Fatalf("lastInit %v", got)
	}
}
