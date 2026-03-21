package core

import (
	"context"

	"Tracker/internal/model"
	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
	"Tracker/internal/pluginhub"
	"Tracker/internal/plugins/registry"
)

// Engine initializes the system, loads plugins, and runs pipelines.
type Engine struct {
	PM          *PluginManager
	Hub         *pluginhub.Hub
	Core        *CoreServices
	Agent       *AgentRunner
	Runner      *PipelineRunner // linear API; delegates to graph runner
	graphRunner *GraphRunner
}

// New creates a new Engine and registers all built-in plugins via registry only.
func New() *Engine {
	hub := pluginhub.New()
	pm := NewPluginManager()
	registry.RegisterAllWithHub(pm, hub)
	core := NewCoreServicesFromEnv()
	agent := NewAgentRunner(pm)
	gr := NewGraphRunner(agent, hub, core)
	runner := NewPipelineRunner(agent, gr)
	return &Engine{
		PM:          pm,
		Hub:         hub,
		Core:        core,
		Agent:       agent,
		Runner:      runner,
		graphRunner: gr,
	}
}

// Init initializes all registered plugins with global config (e.g. API keys from env).
func (e *Engine) Init(global plugin.Config) error {
	return e.PM.InitAll(global)
}

// Run executes a linear pipeline (converted to a chain graph internally).
func (e *Engine) Run(pipe *pipeline.Pipeline) ([]*model.Item, error) {
	return e.Runner.Run(pipe)
}

// RunGraph executes a dynamic pipeline graph (DAG).
func (e *Engine) RunGraph(g *pipeline.PipelineGraph) ([]*model.Item, error) {
	return e.graphRunner.Run(g)
}

// RunGraphWithContext executes a graph with context and optional pipeline id for lifecycle metadata.
func (e *Engine) RunGraphWithContext(ctx context.Context, pipelineID int64, g *pipeline.PipelineGraph) ([]*model.Item, error) {
	return e.graphRunner.RunWithContext(ctx, pipelineID, g)
}
