package pluginhub

import (
	"strings"
	"testing"

	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
)

func TestValidatePipelineGraph_RejectsOperatorKind(t *testing.T) {
	h := New()
	h.RegisterManifest(plugin.Manifest{
		ID: "rss", Version: "1", Kind: plugin.TypeSource,
	})
	h.RegisterManifest(plugin.Manifest{
		ID: "llm_operator", Version: "1", Kind: plugin.TypeOperator,
	})
	g := &pipeline.PipelineGraph{
		Name: "bad",
		Nodes: []pipeline.GraphNode{
			{ID: "src", Type: plugin.TypeSource, PluginID: "rss"},
			{ID: "n1", Type: plugin.TypeOperator, PluginID: "llm_operator"},
		},
		Edges: []pipeline.GraphEdge{{ID: "e1", Source: "src", Target: "n1"}},
	}
	err := h.ValidatePipelineGraph(g)
	if err == nil {
		t.Fatal("expected error for operator in graph")
	}
	if !strings.Contains(err.Error(), "operator") {
		t.Fatalf("got %v", err)
	}
}
