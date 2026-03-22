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
		ID: "llm_operator", Version: "1", Kind: plugin.TypeOperator,
	})
	g := &pipeline.PipelineGraph{
		Name: "bad",
		Nodes: []pipeline.GraphNode{
			{ID: "n1", Type: plugin.TypeOperator, PluginID: "llm_operator"},
		},
	}
	err := h.ValidatePipelineGraph(g)
	if err == nil {
		t.Fatal("expected error for operator in graph")
	}
	if !strings.Contains(err.Error(), "operator") {
		t.Fatalf("got %v", err)
	}
}
