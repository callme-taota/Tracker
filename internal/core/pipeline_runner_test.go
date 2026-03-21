package core

import (
	"testing"

	"Tracker/internal/model"
	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
)

// mockSource returns fixed items.
type mockSource struct{ items []*model.Item }

func (m *mockSource) Name() string    { return "mock_src" }
func (m *mockSource) Version() string { return "1" }
func (m *mockSource) Type() plugin.Type { return plugin.TypeSource }
func (m *mockSource) Init(cfg plugin.Config) error { return nil }
func (m *mockSource) ExecuteSource(cfg plugin.Config) ([]*model.Item, error) { return m.items, nil }

// mockProcess passes through.
type mockProcess struct{}

func (m *mockProcess) Name() string    { return "mock_proc" }
func (m *mockProcess) Version() string { return "1" }
func (m *mockProcess) Type() plugin.Type { return plugin.TypeProcessor }
func (m *mockProcess) Init(cfg plugin.Config) error { return nil }
func (m *mockProcess) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) { return in, nil }

func TestPipelineRunner_Run_SourceOnly(t *testing.T) {
	pm := NewPluginManager()
	pm.Register(&mockSource{items: []*model.Item{
		{Title: "a", URL: "u1"},
		{Title: "b", URL: "u2"},
	}})
	agent := NewAgentRunner(pm)
	runner := NewPipelineRunner(agent, NewGraphRunner(agent, nil, nil))
	pipe := &pipeline.Pipeline{
		Name: "test",
		Stages: []pipeline.Stage{
			{Name: "src", PluginType: plugin.TypeSource, PluginID: "mock_src", Config: nil},
		},
	}
	items, err := runner.Run(pipe)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items got %d", len(items))
	}
	if items[0].Title != "a" || items[1].Title != "b" {
		t.Errorf("items want a,b got %s,%s", items[0].Title, items[1].Title)
	}
}

func TestPipelineRunner_Run_SourceAndProcess(t *testing.T) {
	pm := NewPluginManager()
	pm.Register(&mockSource{items: []*model.Item{{Title: "x", URL: "u"}}})
	pm.Register(&mockProcess{})
	agent := NewAgentRunner(pm)
	runner := NewPipelineRunner(agent, NewGraphRunner(agent, nil, nil))
	pipe := &pipeline.Pipeline{
		Name: "test",
		Stages: []pipeline.Stage{
			{Name: "src", PluginType: plugin.TypeSource, PluginID: "mock_src", Config: nil},
			{Name: "proc", PluginType: plugin.TypeProcessor, PluginID: "mock_proc", Config: nil},
		},
	}
	items, err := runner.Run(pipe)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "x" {
		t.Fatalf("want 1 item x got %d %s", len(items), firstTitle(items))
	}
}

func firstTitle(items []*model.Item) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].Title
}
