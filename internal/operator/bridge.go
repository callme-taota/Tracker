package operator

import (
	"context"
	"encoding/json"

	"Tracker/internal/core/llm"
	"Tracker/internal/storage"
)

// Bridge exposes a narrow, auditable surface for the LLM operator plugin (no arbitrary SQL/shell).
type Bridge interface {
	LLMRouter() *llm.Router

	ListPipelines(ctx context.Context) ([]storage.PipelineSummary, error)
	GetPipeline(ctx context.Context, id int64) (*storage.PipelineDefinition, error)
	RunPipeline(ctx context.Context, id int64) (itemsProcessed int, err error)
	EnqueuePipelineJob(ctx context.Context, id int64) (jobID int64, err error)
	GetJob(ctx context.Context, id int64) (*storage.Job, error)
	ListPlugins(ctx context.Context) ([]PluginListEntry, error)
	GetPluginManifestJSON(ctx context.Context, id string) (json.RawMessage, error)
	CorePing(ctx context.Context) (map[string]interface{}, error)
}

// PluginListEntry is one row for list_plugins tool output.
type PluginListEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
	Runtime string `json:"runtime"` // "builtin" or "remote"
}
