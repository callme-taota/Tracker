package runtimeflow

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"Tracker/internal/config"
	"Tracker/internal/core"
	"Tracker/internal/featureflags"
	"Tracker/internal/model"
	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
	"Tracker/internal/storage"
)

type executorContextKey struct{}

// RuntimeMetadata is the shared per-run experiment context injected into API, worker, scheduler, and CLI executions.
type RuntimeMetadata struct {
	Source      string                     `json:"source"`
	RequestPath string                     `json:"request_path,omitempty"`
	SubjectID   string                     `json:"subject_id,omitempty"`
	RequestID   string                     `json:"request_id,omitempty"`
	RemoteAddr  string                     `json:"remote_addr,omitempty"`
	PipelineID  int64                      `json:"pipeline_id,omitempty"`
	JobID       int64                      `json:"job_id,omitempty"`
	Release     featureflags.ReleaseConfig `json:"release"`
	Flags       featureflags.Snapshot      `json:"flags"`
	Attributes  map[string]string          `json:"attributes,omitempty"`
}

// RunInput describes one pipeline execution request.
type RunInput struct {
	Ctx           context.Context
	RequestPath   string
	Source        string
	SubjectID     string
	RequestID     string
	RemoteAddr    string
	PipelineID    int64
	JobID         int64
	Payload       map[string]interface{}
	FlagOverrides map[string]string
	PersistOutput bool
	UseDBDefault  bool
}

// RunResult is the normalized execution response.
type RunResult struct {
	Items      []*model.Item
	Saved      bool
	Source     string
	PipelineID int64
	Flags      featureflags.Snapshot
}

// Executor centralizes engine init, flag evaluation, graph loading, and optional persistence.
type Executor struct {
	Engine       *core.Engine
	DB           *storage.DB
	PipelinePath string
	App          config.App
	Flags        *featureflags.Engine

	mu       sync.Mutex
	initDone bool
}

// NewExecutor builds a runtime executor for all entrypoints.
func NewExecutor(eng *core.Engine, db *storage.DB, app config.App, pipelinePath string) *Executor {
	return &Executor{
		Engine:       eng,
		DB:           db,
		PipelinePath: pipelinePath,
		App:          app,
		Flags:        featureflags.NewEngine(app.Release, app.FeatureFlags),
	}
}

// WithRuntimeMetadata attaches runtime metadata to context.
func WithRuntimeMetadata(ctx context.Context, meta RuntimeMetadata) context.Context {
	return context.WithValue(ctx, executorContextKey{}, meta)
}

// RuntimeMetadataFromContext returns the metadata stored in context.
func RuntimeMetadataFromContext(ctx context.Context) (RuntimeMetadata, bool) {
	meta, ok := ctx.Value(executorContextKey{}).(RuntimeMetadata)
	return meta, ok
}

// ResolveRequestOverrides reads explicit request overrides from header or query.
func ResolveRequestOverrides(r *http.Request) map[string]string {
	if r == nil {
		return nil
	}
	raw := r.Header.Get("X-Tracker-Experiment")
	if raw == "" {
		raw = r.URL.Query().Get("experiment")
	}
	return parseOverrides(raw)
}

// ResolveSubjectID picks a stable subject for bucket assignment.
func ResolveSubjectID(r *http.Request) string {
	if r == nil {
		return ""
	}
	for _, candidate := range []string{
		r.Header.Get("X-Tracker-Subject"),
		r.Header.Get("X-Tracker-User"),
		r.URL.Query().Get("subject"),
	} {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

// RunDefault executes the default pipeline, preferring the DB default when present.
func (e *Executor) RunDefault(input RunInput) (RunResult, error) {
	if e.DB != nil && input.UseDBDefault {
		if row, err := e.DB.GetDefaultPipelineDefinition(); err == nil && row != nil {
			return e.RunPipelineByID(RunInput{
				Ctx:           input.Ctx,
				RequestPath:   input.RequestPath,
				Source:        "db",
				SubjectID:     input.SubjectID,
				RequestID:     input.RequestID,
				RemoteAddr:    input.RemoteAddr,
				PipelineID:    row.ID,
				JobID:         input.JobID,
				Payload:       input.Payload,
				PersistOutput: input.PersistOutput,
			})
		}
	}
	return e.RunPipelineFile(input)
}

// RunPipelineFile executes the configured pipeline YAML file.
func (e *Executor) RunPipelineFile(input RunInput) (RunResult, error) {
	if err := e.EnsureInitialized(); err != nil {
		return RunResult{}, err
	}
	ctx := input.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	pipe, err := pipeline.LoadFromFile(e.PipelinePath)
	if err != nil {
		return RunResult{}, err
	}
	flags := e.snapshot(input, "file")
	if flags.Variant("runtime.executor_v2") == "legacy" {
		items, err := e.Engine.Run(pipe)
		if err != nil {
			return RunResult{}, err
		}
		saved, err := PersistItems(e.DB, items, input.PersistOutput)
		if err != nil {
			return RunResult{}, err
		}
		return RunResult{Items: items, Saved: saved, Source: "file", Flags: flags}, nil
	}
	graph := pipeline.LinearToGraph(pipe)
	return e.runGraph(input, graph, 0, "file", flags)
}

// RunPipelineByID executes one stored graph pipeline.
func (e *Executor) RunPipelineByID(input RunInput) (RunResult, error) {
	if e.DB == nil {
		return RunResult{}, fmt.Errorf("storage not configured")
	}
	if input.PipelineID <= 0 {
		return RunResult{}, fmt.Errorf("invalid pipeline id")
	}
	if err := e.EnsureInitialized(); err != nil {
		return RunResult{}, err
	}
	row, err := e.DB.GetPipelineDefinition(input.PipelineID)
	if err != nil {
		return RunResult{}, err
	}
	if row == nil {
		return RunResult{}, fmt.Errorf("pipeline not found")
	}
	graph, err := pipeline.ParseGraphJSON([]byte(row.GraphJSON))
	if err != nil {
		return RunResult{}, err
	}
	flags := e.snapshot(input, "db")
	if flags.Variant("runtime.executor_v2") == "legacy" {
		items, err := e.Engine.RunGraphWithContext(contextOrBackground(input.Ctx), input.PipelineID, graph)
		if err != nil {
			return RunResult{}, err
		}
		saved, err := PersistItems(e.DB, items, input.PersistOutput)
		if err != nil {
			return RunResult{}, err
		}
		return RunResult{Items: items, Saved: saved, Source: "db", PipelineID: input.PipelineID, Flags: flags}, nil
	}
	return e.runGraph(input, graph, input.PipelineID, "db", flags)
}

func (e *Executor) runGraph(input RunInput, graph *pipeline.PipelineGraph, pipelineID int64, source string, flags featureflags.Snapshot) (RunResult, error) {
	runCtx := contextOrBackground(input.Ctx)
	meta := RuntimeMetadata{
		Source:      input.Source,
		RequestPath: input.RequestPath,
		SubjectID:   input.SubjectID,
		RequestID:   input.RequestID,
		RemoteAddr:  input.RemoteAddr,
		PipelineID:  pipelineID,
		JobID:       input.JobID,
		Release:     e.App.Release,
		Flags:       flags,
		Attributes: map[string]string{
			"source":       source,
			"request_path": input.RequestPath,
			"request_id":   input.RequestID,
			"remote_addr":  input.RemoteAddr,
		},
	}
	runCtx = WithRuntimeMetadata(runCtx, meta)
	payload := make(map[string]interface{}, len(input.Payload)+1)
	for k, v := range input.Payload {
		payload[k] = v
	}
	payload[core.TrackerRuntimeKey] = meta
	runCtx = core.WithRunPayload(runCtx, payload)
	items, err := e.Engine.RunGraphWithContext(runCtx, pipelineID, graph)
	if err != nil {
		return RunResult{}, err
	}
	saved, err := PersistItems(e.DB, items, input.PersistOutput)
	if err != nil {
		return RunResult{}, err
	}
	return RunResult{Items: items, Saved: saved, Source: source, PipelineID: pipelineID, Flags: flags}, nil
}

// EnsureInitialized initializes the shared engine once per process config.
func (e *Executor) EnsureInitialized() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.initDone {
		return nil
	}
	if e.Engine == nil {
		return fmt.Errorf("engine not configured")
	}
	if err := e.Engine.Init(plugin.Config(e.App.GlobalPluginConfig())); err != nil {
		return err
	}
	e.initDone = true
	return nil
}

func (e *Executor) snapshot(input RunInput, source string) featureflags.Snapshot {
	if e.Flags == nil {
		return featureflags.Snapshot{}
	}
	return e.Flags.Snapshot(featureflags.EvalContext{
		SubjectID:   input.SubjectID,
		Source:      source,
		RequestPath: input.RequestPath,
		PipelineID:  input.PipelineID,
		JobID:       input.JobID,
		Overrides:   input.FlagOverrides,
		Attributes: map[string]string{
			"request_id":  input.RequestID,
			"remote_addr": input.RemoteAddr,
		},
	}, false)
}

// SnapshotForRequest exposes the web-visible experiment view for a request.
func (e *Executor) SnapshotForRequest(r *http.Request) featureflags.Snapshot {
	if e.Flags == nil {
		return featureflags.Snapshot{}
	}
	return e.Flags.Snapshot(featureflags.EvalContext{
		SubjectID:   ResolveSubjectID(r),
		Source:      "web",
		RequestPath: requestPath(r),
		Overrides:   ResolveRequestOverrides(r),
		Attributes: map[string]string{
			"request_id":  r.Header.Get("X-Request-Id"),
			"remote_addr": r.RemoteAddr,
		},
	}, true)
}

// PersistItems stores pipeline output rows and summaries when enabled.
func PersistItems(db *storage.DB, items []*model.Item, persist bool) (bool, error) {
	if db == nil || !persist || len(items) == 0 {
		return false, nil
	}
	for _, it := range items {
		var sid *int64
		ts := it.Timestamp
		if ts.IsZero() {
			ts = time.Now()
		}
		itemID, err := db.SaveItem(sid, it.Title, it.URL, it.Content, it.Summary, ts, "")
		if err != nil {
			return false, err
		}
		if itemID > 0 && (it.Summary != "" || len(it.KeyPoints) > 0) {
			kpJSON := storage.KeyPointsToJSON(it.KeyPoints)
			if _, err := db.SaveSummary(itemID, it.Summary, kpJSON); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// InjectNodeRuntimeConfig appends runtime metadata into plugin config without changing legacy keys.
func InjectNodeRuntimeConfig(base plugin.Config, meta RuntimeMetadata) plugin.Config {
	out := make(plugin.Config, len(base)+1)
	for k, v := range base {
		out[k] = v
	}
	out["__tracker_runtime"] = meta
	return out
}

func parseOverrides(raw string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func requestPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.Path
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
