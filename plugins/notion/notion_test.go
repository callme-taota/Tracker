package notion

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
		if r.URL.Path != "/v1/databases/db123" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"db123"}`))
	}))
	defer srv.Close()

	p := New().(*Plugin)
	p.client = srv.Client()
	if err := p.TestConfig(context.Background(), plugin.Config{
		"token":       "secret",
		"database_id": "db123",
		"base_url":    srv.URL + "/v1",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPlugin_ExecuteDispatch_MissingConfig(t *testing.T) {
	p := New().(*Plugin)
	err := p.ExecuteDispatch(&model.Item{Title: "hello"}, plugin.Config{})
	if err == nil || !strings.Contains(err.Error(), "token and database_id") {
		t.Fatalf("unexpected err: %v", err)
	}
}
