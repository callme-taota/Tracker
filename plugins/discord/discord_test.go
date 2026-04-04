package discord

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

func TestPlugin_TestConfig_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"id":"123"}`))
	}))
	defer srv.Close()

	p := New().(*Plugin)
	p.httpClient = srv.Client()
	if err := p.TestConfig(context.Background(), plugin.Config{"webhook_url": srv.URL}); err != nil {
		t.Fatal(err)
	}
}

func TestPlugin_ExecuteDispatch_MissingConfig(t *testing.T) {
	p := New().(*Plugin)
	err := p.ExecuteDispatch(&model.Item{Title: "hello"}, plugin.Config{})
	if err == nil || !strings.Contains(err.Error(), "webhook_url is required") {
		t.Fatalf("unexpected err: %v", err)
	}
}
