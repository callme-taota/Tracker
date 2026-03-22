package plugin

import "context"

// ChatMessage is one turn in an operator chat (OpenAI-style roles).
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OperatorChatRequest is input to the LLM operator.
type OperatorChatRequest struct {
	Messages      []ChatMessage `json:"messages"`
	LLMProfile    string        `json:"llm_profile,omitempty"`
	MaxToolRounds int           `json:"max_tool_rounds,omitempty"`
}

// ToolTraceEntry records one executed tool for auditing.
type ToolTraceEntry struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}

// OperatorChatResponse is the assistant reply plus optional tool trace.
type OperatorChatResponse struct {
	Reply     string             `json:"reply"`
	ToolTrace []ToolTraceEntry   `json:"tool_trace,omitempty"`
}

// OperatorCapability is implemented by plugins that drive the system via LLM (not pipeline nodes).
type OperatorCapability interface {
	Chat(ctx context.Context, cfg Config, req OperatorChatRequest) (OperatorChatResponse, error)
}
