package core

import (
	"context"
	"fmt"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
	"Tracker/internal/pluginhub"
)

func mergeSourceConfig(base plugin.Config, payload map[string]interface{}) plugin.Config {
	if len(payload) == 0 {
		return base
	}
	out := make(plugin.Config)
	for k, v := range base {
		out[k] = v
	}
	for k, v := range payload {
		out[k] = v
	}
	return out
}

// GraphRunner executes a PipelineGraph (DAG) in topological order.
type GraphRunner struct {
	agent *AgentRunner
	hub   *pluginhub.Hub
	core  *CoreServices
}

// NewGraphRunner creates a GraphRunner. hub may be nil to skip lifecycle events.
// core is injected into pluginhub.RuntimeContext for plugins (LLM, storage, etc.).
func NewGraphRunner(agent *AgentRunner, hub *pluginhub.Hub, core *CoreServices) *GraphRunner {
	return &GraphRunner{agent: agent, hub: hub, core: core}
}

// Run executes the graph with a background context and pipelineID 0.
func (gr *GraphRunner) Run(g *pipeline.PipelineGraph) ([]*model.Item, error) {
	return gr.RunWithContext(context.Background(), 0, g)
}

// RunWithContext executes the graph, emitting PluginHub lifecycle events when hub is set.
func (gr *GraphRunner) RunWithContext(ctx context.Context, pipelineID int64, g *pipeline.PipelineGraph) ([]*model.Item, error) {
	if g == nil || len(g.Nodes) == 0 {
		return nil, nil
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	order, err := g.TopologicalOrder()
	if err != nil {
		return nil, err
	}
	nodes := g.NodeByID()
	outMap := make(map[string][]*model.Item)
	var lastNonDispatch []*model.Item

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	payload := RunPayload(ctx)
	rt := &pluginhub.RuntimeContext{Ctx: ctx, RunID: runID, PipelineID: pipelineID, Core: gr.core, Payload: payload}

	if gr.hub != nil {
		gr.hub.Emit(rt, pluginhub.Event{Type: pluginhub.PipelineStart, RunID: runID, PipelineID: pipelineID})
	}
	defer func() {
		if gr.hub != nil {
			gr.hub.Emit(rt, pluginhub.Event{Type: pluginhub.PipelineEnd, RunID: runID, PipelineID: pipelineID})
		}
	}()

	for _, nid := range order {
		n := nodes[nid]
		preds := g.SuccessPredecessors(nid)
		rt.NodeID = nid
		rt.PluginID = n.PluginID

		if gr.hub != nil {
			gr.hub.Emit(rt, pluginhub.Event{Type: pluginhub.NodeStart, RunID: runID, PipelineID: pipelineID, NodeID: nid, PluginID: n.PluginID})
		}

		switch n.Type {
		case plugin.TypeSource:
			srcCfg := mergeSourceConfig(n.Config, payload)
			items, err := gr.agent.RunSource(n.PluginID, srcCfg)
			if err != nil {
				if gr.hub != nil {
					gr.hub.Emit(rt, pluginhub.Event{Type: pluginhub.ErrorEvent, RunID: runID, PipelineID: pipelineID, NodeID: nid, PluginID: n.PluginID, Err: err})
				}
				return nil, err
			}
			if items == nil {
				items = []*model.Item{}
			}
			outMap[nid] = items
			lastNonDispatch = items
			if gr.hub != nil {
				gr.hub.Emit(rt, pluginhub.Event{Type: pluginhub.NodeEnd, RunID: runID, PipelineID: pipelineID, NodeID: nid, PluginID: n.PluginID, ItemsOut: len(items)})
			}

		case plugin.TypeProcessor, plugin.TypeSummary, plugin.TypeInterest:
			var inputs []*model.Item
			for _, pid := range preds {
				inputs = append(inputs, outMap[pid]...)
			}
			var next []*model.Item
			for _, it := range inputs {
				o, err := gr.agent.RunProcess(n.PluginID, it, n.Config)
				if err != nil {
					if gr.hub != nil {
						gr.hub.Emit(rt, pluginhub.Event{Type: pluginhub.ErrorEvent, RunID: runID, PipelineID: pipelineID, NodeID: nid, PluginID: n.PluginID, Err: err})
					}
					continue
				}
				if o != nil {
					next = append(next, o)
				}
			}
			outMap[nid] = next
			lastNonDispatch = next
			if gr.hub != nil {
				gr.hub.Emit(rt, pluginhub.Event{Type: pluginhub.NodeEnd, RunID: runID, PipelineID: pipelineID, NodeID: nid, PluginID: n.PluginID, ItemsIn: len(inputs), ItemsOut: len(next)})
			}

		case plugin.TypeDispatch:
			var inputs []*model.Item
			for _, pid := range preds {
				inputs = append(inputs, outMap[pid]...)
			}
			for _, it := range inputs {
				if err := gr.agent.RunDispatch(n.PluginID, it, n.Config); err != nil {
					if gr.hub != nil {
						gr.hub.Emit(rt, pluginhub.Event{Type: pluginhub.ErrorEvent, RunID: runID, PipelineID: pipelineID, NodeID: nid, PluginID: n.PluginID, Err: err})
					}
				}
			}
			outMap[nid] = nil
			if gr.hub != nil {
				gr.hub.Emit(rt, pluginhub.Event{Type: pluginhub.NodeEnd, RunID: runID, PipelineID: pipelineID, NodeID: nid, PluginID: n.PluginID, ItemsIn: len(inputs)})
			}

		default:
			err := &model.ItemError{Code: "unknown_node_type", Message: string(n.Type)}
			if gr.hub != nil {
				gr.hub.Emit(rt, pluginhub.Event{Type: pluginhub.ErrorEvent, RunID: runID, PipelineID: pipelineID, NodeID: nid, PluginID: n.PluginID, Err: err})
			}
			return nil, err
		}
	}

	return lastNonDispatch, nil
}
