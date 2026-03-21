package core

import (
	"Tracker/internal/model"
	"Tracker/internal/pipeline"
)

// PipelineRunner runs a linear pipeline by converting it to a graph and delegating to GraphRunner.
type PipelineRunner struct {
	graph *GraphRunner
}

// NewPipelineRunner creates a PipelineRunner backed by the same GraphRunner as Engine.RunGraph.
func NewPipelineRunner(agent *AgentRunner, gr *GraphRunner) *PipelineRunner {
	if gr == nil {
		gr = NewGraphRunner(agent, nil, nil)
	}
	return &PipelineRunner{graph: gr}
}

// Run converts linear stages to a chain graph and executes via GraphRunner.
func (pr *PipelineRunner) Run(pipe *pipeline.Pipeline) ([]*model.Item, error) {
	if pipe == nil || len(pipe.Stages) == 0 {
		return nil, nil
	}
	g := pipeline.LinearToGraph(pipe)
	return pr.graph.Run(g)
}
