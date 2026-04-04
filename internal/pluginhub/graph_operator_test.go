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

func TestValidatePipelineGraph_RejectsDispatchOutbound(t *testing.T) {
	h := New()
	h.RegisterManifest(plugin.Manifest{
		ID: "rss", Version: "1", Kind: plugin.TypeSource,
		OutputFormats: []string{plugin.FormatTrackerItemV1},
	})
	h.RegisterManifest(plugin.Manifest{
		ID: "telegram", Version: "1", Kind: plugin.TypeDispatch,
		InputFormats: []string{plugin.FormatTrackerItemV1},
	})
	h.RegisterManifest(plugin.Manifest{
		ID: "clean", Version: "1", Kind: plugin.TypeProcessor,
		InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
	})
	g := &pipeline.PipelineGraph{
		Name: "bad",
		Nodes: []pipeline.GraphNode{
			{ID: "s", Type: plugin.TypeSource, PluginID: "rss"},
			{ID: "d", Type: plugin.TypeDispatch, PluginID: "telegram"},
			{ID: "p", Type: plugin.TypeProcessor, PluginID: "clean"},
		},
		Edges: []pipeline.GraphEdge{
			{ID: "e1", Source: "s", Target: "d"},
			{ID: "e2", Source: "d", Target: "p"},
		},
	}
	err := h.ValidatePipelineGraph(g)
	if err == nil {
		t.Fatal("expected error for dispatch -> processor")
	}
	if !strings.Contains(err.Error(), "outbound") && !strings.Contains(err.Error(), "emit") {
		t.Fatalf("got %v", err)
	}
}

func TestValidatePipelineGraph_RejectsIncomingToSource(t *testing.T) {
	h := New()
	h.RegisterManifest(plugin.Manifest{
		ID: "rss", Version: "1", Kind: plugin.TypeSource,
		OutputFormats: []string{plugin.FormatTrackerItemV1},
	})
	h.RegisterManifest(plugin.Manifest{
		ID: "clean", Version: "1", Kind: plugin.TypeProcessor,
		InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
	})
	g := &pipeline.PipelineGraph{
		Name: "bad",
		Nodes: []pipeline.GraphNode{
			{ID: "s", Type: plugin.TypeSource, PluginID: "rss"},
			{ID: "p", Type: plugin.TypeProcessor, PluginID: "clean"},
		},
		Edges: []pipeline.GraphEdge{
			{ID: "e1", Source: "p", Target: "s"},
		},
	}
	err := h.ValidatePipelineGraph(g)
	if err == nil {
		t.Fatal("expected error for edge into source")
	}
}

func TestValidatePipelineGraph_AllowNoIncoming(t *testing.T) {
	h := New()
	h.RegisterManifest(plugin.Manifest{
		ID: "rss", Version: "1", Kind: plugin.TypeSource,
		OutputFormats: []string{plugin.FormatTrackerItemV1},
	})
	h.RegisterManifest(plugin.Manifest{
		ID: "side", Version: "1", Kind: plugin.TypeProcessor,
		InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
		PipelineIO: &plugin.PipelineIOSpec{AllowNoIncoming: true},
	})
	// Bootstrap processor: no incoming edges, allowed by manifest.pipeline_io.allow_no_incoming
	g := &pipeline.PipelineGraph{
		Name: "bootstrap",
		Nodes: []pipeline.GraphNode{
			{ID: "s", Type: plugin.TypeSource, PluginID: "rss"},
			{ID: "x", Type: plugin.TypeProcessor, PluginID: "side"},
		},
		Edges: []pipeline.GraphEdge{},
	}
	if err := h.ValidatePipelineGraph(g); err != nil {
		t.Fatal(err)
	}
}

