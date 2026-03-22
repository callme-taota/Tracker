package llm_operator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"Tracker/internal/core/llm"
	"Tracker/internal/operator"
	"Tracker/internal/plugin"
)

var bridge operator.Bridge

// SetBridge wires the operator after the HTTP server / engine exist (avoids import cycles).
func SetBridge(b operator.Bridge) {
	bridge = b
}

// Plugin is an operator-only built-in: conversation + tools over Bridge + LLM.
type Plugin struct {
	profile   string
	maxRounds int
	extraSys  string
}

// New creates the llm_operator plugin singleton state (Init may override fields).
func New() *Plugin {
	return &Plugin{profile: "cheap", maxRounds: 8}
}

func (p *Plugin) Name() string    { return "llm_operator" }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Type() plugin.Type {
	return plugin.TypeOperator
}

func (p *Plugin) Init(cfg plugin.Config) error {
	if s, ok := cfg["llm_profile"].(string); ok && strings.TrimSpace(s) != "" {
		p.profile = strings.TrimSpace(s)
	}
	switch v := cfg["max_tool_rounds"].(type) {
	case float64:
		if v > 0 {
			p.maxRounds = int(v)
		}
	case int:
		if v > 0 {
			p.maxRounds = v
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			p.maxRounds = n
		}
	}
	if s, ok := cfg["system_prompt_extra"].(string); ok {
		p.extraSys = s
	}
	return nil
}

func emptyObj() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
}

func idIntParams() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"id":{"type":"integer","description":"numeric id"}},"required":["id"]}`)
}

func idStrParams() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"plugin id"}},"required":["id"]}`)
}

func operatorTools() []llm.ToolSpec {
	mk := func(name, desc string, params json.RawMessage) llm.ToolSpec {
		var t llm.ToolSpec
		t.Type = "function"
		t.Function.Name = name
		t.Function.Description = desc
		t.Function.Parameters = params
		return t
	}
	return []llm.ToolSpec{
		mk("list_pipelines", "List pipeline definitions (id, name, is_default).", emptyObj()),
		mk("get_pipeline", "Load one pipeline row including graph_json.", idIntParams()),
		mk("run_pipeline", "Run a pipeline synchronously; returns how many items were produced at the final stage.", idIntParams()),
		mk("enqueue_pipeline_job", "Enqueue an async pipeline run; returns job_id.", idIntParams()),
		mk("get_job", "Get async job record by id.", idIntParams()),
		mk("list_plugins", "List registered plugins (builtin vs remote).", emptyObj()),
		mk("get_plugin_manifest", "Get manifest JSON for a plugin id.", idStrParams()),
		mk("core_ping", "Report LLM profile names and storage ping map.", emptyObj()),
	}
}

func parseIntID(args string) (int64, error) {
	var in map[string]interface{}
	if err := json.Unmarshal([]byte(args), &in); err != nil {
		return 0, err
	}
	raw, ok := in["id"]
	if !ok {
		return 0, fmt.Errorf("missing id")
	}
	switch v := raw.(type) {
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	default:
		return 0, fmt.Errorf("invalid id type")
	}
}

func parseStringID(args string) (string, error) {
	var in struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(args), &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.ID) == "" {
		return "", fmt.Errorf("missing id")
	}
	return strings.TrimSpace(in.ID), nil
}

func runTool(ctx context.Context, name, args string) (string, error) {
	if bridge == nil {
		return "", fmt.Errorf("bridge not configured")
	}
	switch name {
	case "list_pipelines":
		pl, err := bridge.ListPipelines(ctx)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(pl)
		return string(b), nil
	case "get_pipeline":
		id, err := parseIntID(args)
		if err != nil {
			return "", err
		}
		def, err := bridge.GetPipeline(ctx, id)
		if err != nil {
			return "", err
		}
		if def == nil {
			return "", fmt.Errorf("pipeline not found")
		}
		b, _ := json.Marshal(def)
		return string(b), nil
	case "run_pipeline":
		id, err := parseIntID(args)
		if err != nil {
			return "", err
		}
		n, err := bridge.RunPipeline(ctx, id)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(map[string]int{"items_processed": n})
		return string(b), nil
	case "enqueue_pipeline_job":
		id, err := parseIntID(args)
		if err != nil {
			return "", err
		}
		jid, err := bridge.EnqueuePipelineJob(ctx, id)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(map[string]int64{"job_id": jid})
		return string(b), nil
	case "get_job":
		id, err := parseIntID(args)
		if err != nil {
			return "", err
		}
		j, err := bridge.GetJob(ctx, id)
		if err != nil {
			return "", err
		}
		if j == nil {
			return "", fmt.Errorf("job not found")
		}
		b, _ := json.Marshal(j)
		return string(b), nil
	case "list_plugins":
		pl, err := bridge.ListPlugins(ctx)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(pl)
		return string(b), nil
	case "get_plugin_manifest":
		pid, err := parseStringID(args)
		if err != nil {
			return "", err
		}
		raw, err := bridge.GetPluginManifestJSON(ctx, pid)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	case "core_ping":
		m, err := bridge.CorePing(ctx)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(m)
		return string(b), nil
	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}
}

// Chat implements plugin.OperatorCapability.
func (p *Plugin) Chat(ctx context.Context, cfg plugin.Config, req plugin.OperatorChatRequest) (plugin.OperatorChatResponse, error) {
	if bridge == nil {
		return plugin.OperatorChatResponse{}, fmt.Errorf("llm_operator: bridge not configured")
	}
	rtr := bridge.LLMRouter()
	if rtr == nil {
		return plugin.OperatorChatResponse{}, fmt.Errorf("llm_operator: no LLM router (set OPENAI_API_KEY)")
	}
	profile := p.profile
	if req.LLMProfile != "" {
		profile = req.LLMProfile
	}
	if s, ok := cfg["llm_profile"].(string); ok && strings.TrimSpace(s) != "" && req.LLMProfile == "" {
		profile = strings.TrimSpace(s)
	}
	maxR := p.maxRounds
	if req.MaxToolRounds > 0 {
		maxR = req.MaxToolRounds
	}

	sys := `You are Tracker's control assistant. You can inspect and run pipelines, list plugins, and check core health.
Use the provided tools only. Prefer enqueue_pipeline_job for long runs unless the user asks for immediate synchronous results.
Respond concisely in the same language as the user when possible.`
	if strings.TrimSpace(p.extraSys) != "" {
		sys += "\n\n" + strings.TrimSpace(p.extraSys)
	}

	msgs := []llm.Message{{Role: "system", Content: sys}}
	for _, m := range req.Messages {
		if strings.TrimSpace(m.Role) == "" {
			continue
		}
		msgs = append(msgs, llm.Message{Role: m.Role, Content: m.Content})
	}

	tools := operatorTools()
	var trace []plugin.ToolTraceEntry

	for round := 0; round < maxR; round++ {
		choice, err := rtr.ChatWithTools(ctx, profile, msgs, tools, 0.2)
		if err != nil {
			return plugin.OperatorChatResponse{}, err
		}
		if len(choice.ToolCalls) == 0 {
			return plugin.OperatorChatResponse{Reply: strings.TrimSpace(choice.Content), ToolTrace: trace}, nil
		}

		asst := llm.Message{Role: "assistant", Content: choice.Content}
		for _, tc := range choice.ToolCalls {
			mtc := llm.MessageToolCall{ID: tc.ID, Type: "function"}
			mtc.Function.Name = tc.Name
			mtc.Function.Arguments = tc.Arguments
			asst.ToolCalls = append(asst.ToolCalls, mtc)
		}
		msgs = append(msgs, asst)

		for _, tc := range choice.ToolCalls {
			res, err := runTool(ctx, tc.Name, tc.Arguments)
			ent := plugin.ToolTraceEntry{Name: tc.Name, Arguments: tc.Arguments, Result: res}
			if err != nil {
				ent.Error = err.Error()
				res = fmt.Sprintf(`{"error":%q}`, err.Error())
			}
			trace = append(trace, ent)
			msgs = append(msgs, llm.Message{
				Role: "tool", ToolCallID: tc.ID, Content: res,
			})
		}
	}

	return plugin.OperatorChatResponse{
		Reply:     "Stopped after maximum tool rounds; last model reply may be incomplete.",
		ToolTrace: trace,
	}, nil
}

// Ensure Plugin implements interfaces at compile time.
var _ plugin.Plugin = (*Plugin)(nil)
var _ plugin.OperatorCapability = (*Plugin)(nil)
