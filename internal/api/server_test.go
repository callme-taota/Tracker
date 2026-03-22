package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Tracker/internal/api"
	"Tracker/internal/core"
	"Tracker/internal/plugin"
	"Tracker/internal/storage"
)

// moduleRoot returns the directory containing go.mod (tests run with cwd = package dir).
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from test cwd")
		}
		dir = parent
	}
}

func defaultConfigPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(moduleRoot(t), "configs", "default.yaml")
}

func testServer(t *testing.T) *api.Server {
	t.Helper()
	eng := core.New()
	if err := eng.Init(plugin.Config{}); err != nil {
		t.Fatal(err)
	}
	return api.NewServer(eng, nil, defaultConfigPath(t), nil)
}

func TestTestPluginConfig_UnsupportedPluginReturns501(t *testing.T) {
	srv := testServer(t)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/rss/test", strings.NewReader(`{"config":{}}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status got %d want %d body=%s", rr.Code, http.StatusNotImplemented, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if supported, _ := out["supported"].(bool); supported {
		t.Fatalf("supported want false got %#v", out)
	}
}

func TestTestPluginConfig_UnknownPlugin404(t *testing.T) {
	srv := testServer(t)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/no_such_plugin_xyz/test", strings.NewReader(`{"config":{}}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status got %d want 404", rr.Code)
	}
}

func TestTestPluginConfig_FeishuOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"msg":"ok"}`)
	}))
	defer ts.Close()

	srv := testServer(t)
	h := srv.Handler()
	body := fmt.Sprintf(`{"config":{"webhook_url":%q}}`, ts.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/feishu/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status got %d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if ok, _ := out["ok"].(bool); !ok {
		t.Fatalf("ok want true got %#v", out)
	}
	if sup, _ := out["supported"].(bool); !sup {
		t.Fatalf("supported want true got %#v", out)
	}
}

func TestStats_StorageDisabled(t *testing.T) {
	srv := testServer(t)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if se, ok := out["storage_enabled"].(bool); !ok || se {
		t.Fatalf("storage_enabled want false got %#v", out)
	}
}

func TestStats_StorageEnabled(t *testing.T) {
	eng := core.New()
	if err := eng.Init(plugin.Config{}); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(filepath.Join(t.TempDir(), "api_stats.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	srv := api.NewServer(eng, db, defaultConfigPath(t), nil)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if se, ok := out["storage_enabled"].(bool); !ok || !se {
		t.Fatalf("storage_enabled want true got %#v", out)
	}
}

func TestOperatorChat_NoEnvKey503(t *testing.T) {
	t.Cleanup(func() { _ = os.Unsetenv("TRACKER_OPERATOR_API_KEY") })
	_ = os.Unsetenv("TRACKER_OPERATOR_API_KEY")
	eng := core.New()
	if err := eng.Init(plugin.Config{}); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(filepath.Join(t.TempDir(), "op_nokey.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	srv := api.NewServer(eng, db, defaultConfigPath(t), nil)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/operator/chat", strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status got %d want 503 body=%s", rr.Code, rr.Body.String())
	}
}

func TestReloadExternalPlugins_NoAdminKey503(t *testing.T) {
	t.Cleanup(func() { _ = os.Unsetenv("TRACKER_PLUGIN_ADMIN_KEY") })
	_ = os.Unsetenv("TRACKER_PLUGIN_ADMIN_KEY")
	eng := core.New()
	defer eng.Close()
	if err := eng.Init(plugin.Config{}); err != nil {
		t.Fatal(err)
	}
	srv := api.NewServer(eng, nil, defaultConfigPath(t), nil)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/external/reload", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status got %d want 503 body=%s", rr.Code, rr.Body.String())
	}
}

func TestOperatorChat_Unauthorized401(t *testing.T) {
	t.Setenv("TRACKER_OPERATOR_API_KEY", "correct-key")
	t.Cleanup(func() { _ = os.Unsetenv("TRACKER_OPERATOR_API_KEY") })
	eng := core.New()
	if err := eng.Init(plugin.Config{}); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(filepath.Join(t.TempDir(), "op_auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	srv := api.NewServer(eng, db, defaultConfigPath(t), nil)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/operator/chat", strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status got %d want 401 body=%s", rr.Code, rr.Body.String())
	}
}
