package feishu

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"Tracker/internal/plugin"
)

func TestPlugin_TestConfig_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		if len(b) == 0 {
			t.Fatal("empty body")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok"}`))
	}))
	defer srv.Close()

	p := &Plugin{}
	err := p.TestConfig(context.Background(), plugin.Config{"webhook_url": srv.URL})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPlugin_TestConfig_emptyURL(t *testing.T) {
	p := &Plugin{}
	err := p.TestConfig(context.Background(), plugin.Config{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPlugin_TestConfig_httpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	defer srv.Close()

	p := &Plugin{}
	err := p.TestConfig(context.Background(), plugin.Config{"webhook_url": srv.URL})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPlugin_TestConfig_feishuCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":19001,"msg":"invalid"}`))
	}))
	defer srv.Close()

	p := &Plugin{}
	err := p.TestConfig(context.Background(), plugin.Config{"webhook_url": srv.URL})
	if err == nil {
		t.Fatal("expected error")
	}
}
