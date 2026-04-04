package openai_summary

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

func TestPlugin_TestConfig_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("auth %q", got)
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"gpt-4o-mini"}]}`)
	}))
	defer srv.Close()

	p := New().(*Plugin)
	p.httpClient = srv.Client()
	if err := p.TestConfig(context.Background(), plugin.Config{"api_key": "test-key", "base_url": srv.URL}); err != nil {
		t.Fatal(err)
	}
}

func TestPlugin_Execute_IncludeMermaid(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if call == 1 {
			_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"Short summary\n- Point A\n- Point B"}}]}`)
			return
		}
		_, _ = io.WriteString(w, "{\"choices\":[{\"message\":{\"content\":\"```mermaid\\ngraph TD\\nA-->B\\n```\"}}]}")
	}))
	defer srv.Close()

	p := New().(*Plugin)
	p.httpClient = srv.Client()
	out, err := p.Execute(&model.Item{Title: "Title", Content: strings.Repeat("body ", 20)}, plugin.Config{
		"api_key":         "test-key",
		"base_url":        srv.URL,
		"include_mermaid": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || !strings.Contains(out.Summary, "mermaid") {
		t.Fatalf("expected summary with mermaid, got %#v", out)
	}
	if len(out.KeyPoints) != 2 {
		t.Fatalf("unexpected key points: %#v", out.KeyPoints)
	}
}
