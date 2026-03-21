package pipeline

import (
	"encoding/json"
	"fmt"

	"Tracker/internal/plugin"
)

// Position is UI layout (React Flow); persisted with the graph.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GraphNode is one node in the pipeline DAG.
type GraphNode struct {
	ID       string         `json:"id"`
	Type     plugin.Type    `json:"plugin_type"`
	PluginID string         `json:"plugin_id"`
	Config   plugin.Config  `json:"config,omitempty"`
	Position Position       `json:"position,omitempty"`
}

// GraphEdge connects two nodes (optional handles for multi-port future).
// OnCondition: empty or "on_success" participates in main DAG ordering; "on_failure" is reserved for conditional routing (documented; execution order uses success edges only).
type GraphEdge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
	OnCondition  string `json:"on_condition,omitempty"`
}

// PipelineGraph is a DAG of plugin nodes; dynamic pipeline definition.
type PipelineGraph struct {
	Name  string      `json:"name"`
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// LinearToGraph converts legacy linear Pipeline to a left-to-right chain graph.
func LinearToGraph(p *Pipeline) *PipelineGraph {
	if p == nil || len(p.Stages) == 0 {
		return &PipelineGraph{Name: "empty"}
	}
	g := &PipelineGraph{Name: p.Name}
	for i, st := range p.Stages {
		id := fmt.Sprintf("n%d", i)
		g.Nodes = append(g.Nodes, GraphNode{
			ID:       id,
			Type:     st.PluginType,
			PluginID: st.PluginID,
			Config:   st.Config,
			Position: Position{X: float64(i) * 280, Y: 0},
		})
		if i > 0 {
			g.Edges = append(g.Edges, GraphEdge{
				ID:     fmt.Sprintf("e%d", i),
				Source: fmt.Sprintf("n%d", i-1),
				Target: id,
			})
		}
	}
	return g
}

// ParseGraphJSON unmarshals a pipeline graph from JSON bytes.
func ParseGraphJSON(data []byte) (*PipelineGraph, error) {
	var g PipelineGraph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// ToJSON serializes the graph.
func (g *PipelineGraph) ToJSON() ([]byte, error) {
	return json.Marshal(g)
}

// NodeByID returns a map id -> node.
func (g *PipelineGraph) NodeByID() map[string]GraphNode {
	m := make(map[string]GraphNode, len(g.Nodes))
	for _, n := range g.Nodes {
		m[n.ID] = n
	}
	return m
}

// Validate checks basic DAG rules.
func (g *PipelineGraph) Validate() error {
	if len(g.Nodes) == 0 {
		return fmt.Errorf("pipeline graph: no nodes")
	}
	ids := g.NodeByID()
	for _, e := range g.Edges {
		if _, ok := ids[e.Source]; !ok {
			return fmt.Errorf("pipeline graph: edge source %q unknown", e.Source)
		}
		if _, ok := ids[e.Target]; !ok {
			return fmt.Errorf("pipeline graph: edge target %q unknown", e.Target)
		}
	}
	if _, err := g.topologicalOrderEdges(successEdges(g.Edges)); err != nil {
		return err
	}
	var hasSource bool
	for _, n := range g.Nodes {
		if n.Type == plugin.TypeSource {
			hasSource = true
			break
		}
	}
	if !hasSource {
		return fmt.Errorf("pipeline graph: at least one source node required")
	}
	for _, n := range g.Nodes {
		if n.Type == plugin.TypeSource {
			continue
		}
		if len(g.SuccessPredecessors(n.ID)) == 0 {
			return fmt.Errorf("pipeline graph: non-source node %q must have an incoming success edge (on_success or default)", n.ID)
		}
	}
	return nil
}

func successEdges(edges []GraphEdge) []GraphEdge {
	var out []GraphEdge
	for _, e := range edges {
		if e.OnCondition == "" || e.OnCondition == "on_success" {
			out = append(out, e)
		}
	}
	return out
}

// TopologicalOrder returns node ids in execution order (Kahn) using success-path edges only.
func (g *PipelineGraph) TopologicalOrder() ([]string, error) {
	return g.topologicalOrderEdges(successEdges(g.Edges))
}

func (g *PipelineGraph) topologicalOrderEdges(edges []GraphEdge) ([]string, error) {
	ids := g.NodeByID()
	inDeg := make(map[string]int, len(g.Nodes))
	for id := range ids {
		inDeg[id] = 0
	}
	adj := make(map[string][]string)
	for _, e := range edges {
		inDeg[e.Target]++
		adj[e.Source] = append(adj[e.Source], e.Target)
	}
	var q []string
	for id := range ids {
		if inDeg[id] == 0 {
			q = append(q, id)
		}
	}
	var order []string
	for len(q) > 0 {
		id := q[0]
		q = q[1:]
		order = append(order, id)
		for _, to := range adj[id] {
			inDeg[to]--
			if inDeg[to] == 0 {
				q = append(q, to)
			}
		}
	}
	if len(order) != len(ids) {
		return nil, fmt.Errorf("pipeline graph: cycle or disconnected nodes")
	}
	return order, nil
}

// Predecessors returns node ids that have an edge -> id.
func (g *PipelineGraph) Predecessors(id string) []string {
	var p []string
	for _, e := range g.Edges {
		if e.Target == id {
			p = append(p, e.Source)
		}
	}
	return p
}

// SuccessPredecessors returns predecessors linked via success-path edges only.
func (g *PipelineGraph) SuccessPredecessors(id string) []string {
	var p []string
	for _, e := range successEdges(g.Edges) {
		if e.Target == id {
			p = append(p, e.Source)
		}
	}
	return p
}
