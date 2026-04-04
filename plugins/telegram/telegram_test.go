package telegram

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
		switch r.URL.Path {
		case "/bottoken/getMe":
			_, _ = io.WriteString(w, `{"ok":true,"result":{"id":1}}`)
		case "/bottoken/getChat":
			_, _ = io.WriteString(w, `{"ok":true,"result":{"id":2}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	p := New().(*Plugin)
	p.httpClient = srv.Client()
	err := p.TestConfig(context.Background(), plugin.Config{
		"bot_token":    "token",
		"chat_id":      "42",
		"api_base_url": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPlugin_ExecuteDispatch_MissingConfig(t *testing.T) {
	p := New().(*Plugin)
	err := p.ExecuteDispatch(&model.Item{Title: "hello"}, plugin.Config{})
	if err == nil || !strings.Contains(err.Error(), "bot_token and chat_id") {
		t.Fatalf("unexpected err: %v", err)
	}
}
