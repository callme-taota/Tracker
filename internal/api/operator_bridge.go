package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"Tracker/internal/core"
	"Tracker/internal/core/llm"
	"Tracker/internal/operator"
	"Tracker/internal/pipeline"
	"Tracker/internal/storage"
)

// ServerBridge implements operator.Bridge using the core engine and SQLite DB.
type ServerBridge struct {
	Eng *core.Engine
	DB  *storage.DB
}

// NewServerBridge returns nil if db is nil (operator disabled for storage-less mode).
func NewServerBridge(eng *core.Engine, db *storage.DB) *ServerBridge {
	if eng == nil || db == nil {
		return nil
	}
	return &ServerBridge{Eng: eng, DB: db}
}

// LLMRouter returns the shared LLM client from core services.
func (b *ServerBridge) LLMRouter() *llm.Router {
	if b.Eng == nil || b.Eng.Core == nil {
		return nil
	}
	return b.Eng.Core.LLM
}

func (b *ServerBridge) ListPipelines(ctx context.Context) ([]storage.PipelineSummary, error) {
	_ = ctx
	if b.DB == nil {
		return nil, fmt.Errorf("storage not configured")
	}
	return b.DB.ListPipelineDefinitions()
}

func (b *ServerBridge) GetPipeline(ctx context.Context, id int64) (*storage.PipelineDefinition, error) {
	_ = ctx
	if b.DB == nil {
		return nil, fmt.Errorf("storage not configured")
	}
	return b.DB.GetPipelineDefinition(id)
}

func (b *ServerBridge) RunPipeline(ctx context.Context, id int64) (int, error) {
	if b.DB == nil {
		return 0, fmt.Errorf("storage not configured")
	}
	row, err := b.DB.GetPipelineDefinition(id)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, fmt.Errorf("pipeline %d not found", id)
	}
	g, err := pipeline.ParseGraphJSON([]byte(row.GraphJSON))
	if err != nil {
		return 0, err
	}
	items, err := b.Eng.RunGraphWithContext(ctx, id, g)
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (b *ServerBridge) EnqueuePipelineJob(ctx context.Context, id int64) (int64, error) {
	_ = ctx
	if b.DB == nil {
		return 0, fmt.Errorf("storage not configured")
	}
	row, err := b.DB.GetPipelineDefinition(id)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, fmt.Errorf("pipeline %d not found", id)
	}
	return b.DB.EnqueueJob(id, "run", nil, 3)
}

func (b *ServerBridge) GetJob(ctx context.Context, id int64) (*storage.Job, error) {
	_ = ctx
	if b.DB == nil {
		return nil, fmt.Errorf("storage not configured")
	}
	return b.DB.GetJob(id)
}

func (b *ServerBridge) ListPlugins(ctx context.Context) ([]operator.PluginListEntry, error) {
	_ = ctx
	all := b.Eng.PM.ListAll()
	out := make([]operator.PluginListEntry, 0, len(all))
	for _, p := range all {
		rt := "builtin"
		if b.Eng.PluginHost != nil && b.Eng.PluginHost.IsRemote(p.Name()) {
			rt = "remote"
		}
		out = append(out, operator.PluginListEntry{
			Name: p.Name(), Version: p.Version(), Type: string(p.Type()), Runtime: rt,
		})
	}
	return out, nil
}

func (b *ServerBridge) GetPluginManifestJSON(ctx context.Context, id string) (json.RawMessage, error) {
	_ = ctx
	m, ok := b.Eng.Hub.Manifest(id)
	if !ok {
		return nil, fmt.Errorf("unknown plugin %q", id)
	}
	return json.Marshal(m)
}

func (b *ServerBridge) CorePing(ctx context.Context) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out := map[string]interface{}{}
	var profiles []string
	if b.Eng.Core != nil && b.Eng.Core.LLM != nil {
		profiles = b.Eng.Core.LLM.ProfileNames()
	}
	out["llm_profiles"] = profiles
	sp := make(map[string]string)
	if b.Eng.Core != nil && b.Eng.Core.Storage != nil {
		for k, err := range b.Eng.Core.Storage.Ping(ctx) {
			if err != nil {
				sp[k] = err.Error()
			} else {
				sp[k] = "ok"
			}
		}
	}
	out["storage_ping"] = sp
	return out, nil
}

var _ operator.Bridge = (*ServerBridge)(nil)
