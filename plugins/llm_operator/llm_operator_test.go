package llm_operator

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"Tracker/internal/core/llm"
	"Tracker/internal/operator"
	"Tracker/internal/plugin"
	"Tracker/internal/storage"
)

type stubBridge struct {
	llm *llm.Router
}

func (s *stubBridge) LLMRouter() *llm.Router { return s.llm }

func (s *stubBridge) ListPipelines(ctx context.Context) ([]storage.PipelineSummary, error) {
	return []storage.PipelineSummary{{ID: 1, Name: "p1"}}, nil
}

func (s *stubBridge) GetPipeline(ctx context.Context, id int64) (*storage.PipelineDefinition, error) {
	return nil, nil
}

func (s *stubBridge) RunPipeline(ctx context.Context, id int64) (int, error) { return 0, nil }

func (s *stubBridge) EnqueuePipelineJob(ctx context.Context, id int64) (int64, error) { return 0, nil }

func (s *stubBridge) GetJob(ctx context.Context, id int64) (*storage.Job, error) { return nil, nil }

func (s *stubBridge) ListPlugins(ctx context.Context) ([]operator.PluginListEntry, error) {
	return nil, nil
}

func (s *stubBridge) GetPluginManifestJSON(ctx context.Context, id string) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

func (s *stubBridge) CorePing(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{"ok": true}, nil
}

var _ operator.Bridge = (*stubBridge)(nil)

func TestChatToolRoundThenAnswer(t *testing.T) {
	// First completion: model requests list_pipelines; second: plain text answer.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)
		msgs, _ := req["messages"].([]interface{})
		hasTool := false
		for _, m := range msgs {
			mm, _ := m.(map[string]interface{})
			if mm["role"] == "tool" {
				hasTool = true
				break
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if !hasTool {
			_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"list_pipelines","arguments":"{}"}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"Found pipeline p1."}}]}`))
	}))
	defer srv.Close()

	rtr := &llm.Router{}
	rtr.RegisterProfile(llm.Profile{
		Name: "cheap", BaseURL: srv.URL + "/v1", Model: "x", APIKey: "k",
		Timeout: 10, MaxRetries: 1,
	})
	SetBridge(&stubBridge{llm: rtr})
	defer SetBridge(nil)

	p := New()
	resp, err := p.Chat(context.Background(), nil, plugin.OperatorChatRequest{
		Messages:      []plugin.ChatMessage{{Role: "user", Content: "What pipelines exist?"}},
		LLMProfile:    "cheap",
		MaxToolRounds: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reply != "Found pipeline p1." {
		t.Fatalf("reply %q trace=%+v", resp.Reply, resp.ToolTrace)
	}
	if len(resp.ToolTrace) != 1 || resp.ToolTrace[0].Name != "list_pipelines" {
		t.Fatalf("trace %+v", resp.ToolTrace)
	}
}
