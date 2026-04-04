package plugin

// PipelineIOSpec declares how a plugin participates in DAG edges and ordering.
// Omitted fields are inferred from kind and format lists for backward compatibility.
type PipelineIOSpec struct {
	// EmitsItems: produces item stream for downstream (nil = infer).
	EmitsItems *bool `json:"emits_items,omitempty"`
	// AcceptsItems: consumes item stream from upstream (nil = infer).
	AcceptsItems *bool `json:"accepts_items,omitempty"`
	// AllowOutboundEdges: may emit edges to other nodes (nil = same as InferredEmitsItems()).
	AllowOutboundEdges *bool `json:"allow_outbound_edges,omitempty"`
	// AllowNoIncoming: non-source node may have zero incoming success edges (bootstrap / self-scheduled).
	AllowNoIncoming bool `json:"allow_no_incoming,omitempty"`
}

// AllowNoIncomingEdge reports whether this plugin may appear without incoming edges (non-source).
func (m Manifest) AllowNoIncomingEdge() bool {
	return m.PipelineIO != nil && m.PipelineIO.AllowNoIncoming
}

// InferredEmitsItems is whether the plugin outputs items usable by downstream nodes.
func (m Manifest) InferredEmitsItems() bool {
	if m.PipelineIO != nil && m.PipelineIO.EmitsItems != nil {
		return *m.PipelineIO.EmitsItems
	}
	switch m.Kind {
	case TypeSource:
		return true
	case TypeProcessor, TypeSummary, TypeInterest:
		return true
	case TypeDispatch:
		return false
	default:
		return len(m.OutputFormats) > 0
	}
}

// InferredAcceptsItems is whether the plugin expects upstream item flow on edges.
func (m Manifest) InferredAcceptsItems() bool {
	if m.PipelineIO != nil && m.PipelineIO.AcceptsItems != nil {
		return *m.PipelineIO.AcceptsItems
	}
	return m.Kind != TypeSource
}

// InferredAllowOutboundEdges is whether edges from this node to others are allowed.
func (m Manifest) InferredAllowOutboundEdges() bool {
	if m.PipelineIO != nil && m.PipelineIO.AllowOutboundEdges != nil {
		return *m.PipelineIO.AllowOutboundEdges
	}
	return m.InferredEmitsItems()
}
