package core

import (
	"context"
	"fmt"
	"log"
	"os"

	"Tracker/internal/model"
	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
	"Tracker/internal/pluginhub"
	"Tracker/internal/pluginruntime"
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
	PluginHost  *pluginruntime.Host // external subprocess plugins (always set; may have zero entries)
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
	extPath := os.Getenv("TRACKER_EXTERNAL_PLUGINS_FILE")
	ph := pluginruntime.NewHost(pm, hub, extPath, registry.BuiltinPluginIDs())
	if err := ph.Load(context.Background()); err != nil {
		log.Printf("external plugins: load: %v", err)
	}
	return &Engine{
		PM:          pm,
		Hub:         hub,
		Core:        core,
		Agent:       agent,
		Runner:      runner,
		graphRunner: gr,
		PluginHost:  ph,
	}
}

// Init initializes all registered plugins with global config (e.g. API keys from env).
func (e *Engine) Init(global plugin.Config) error {
	if err := e.PM.InitAll(global); err != nil {
		return err
	}
	if e.PluginHost != nil {
		if err := e.PluginHost.InitRemote(context.Background(), global); err != nil {
			return err
		}
	}
	return nil
}

// Close releases subprocess plugins and core services.
func (e *Engine) Close() error {
	if e.PluginHost != nil {
		e.PluginHost.Shutdown(context.Background())
	}
	if e.Core != nil {
		return e.Core.Close()
	}
	return nil
}

// ReloadExternalPlugins stops and restarts external plugins from the YAML file, then re-runs InitRemote.
func (e *Engine) ReloadExternalPlugins(ctx context.Context) error {
	if e.PluginHost == nil {
		return fmt.Errorf("plugin host not available")
	}
	if err := e.PluginHost.Reload(ctx); err != nil {
		return err
	}
	return e.PluginHost.InitRemote(ctx, e.PluginHost.LastGlobalConfig())
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
