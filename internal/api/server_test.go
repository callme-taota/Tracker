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
	"Tracker/internal/config"
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
	return api.NewServer(eng, nil, config.DefaultApp(), defaultConfigPath(t), nil)
}

func TestTestPluginConfig_UnsupportedPluginReturns501(t *testing.T) {
	srv := testServer(t)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/clean/test", strings.NewReader(`{"config":{}}`))
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

func TestPluginQualityReport_IncludesGovernanceDetails(t *testing.T) {
	t.Setenv("TRACKER_API_KEY", "quality-key")
	srv := testServer(t)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/plugins/quality-report", nil)
	req.Header.Set("X-Tracker-Api-Key", "quality-key")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status got %d body=%s", rr.Code, rr.Body.String())
	}
	var rows []struct {
		Name          string `json:"name"`
		HasSchema     bool   `json:"has_schema"`
		HasTester     bool   `json:"has_tester"`
		NeedsTester   bool   `json:"needs_tester"`
		HasPipelineIO bool   `json:"has_pipeline_io"`
		Score         int    `json:"score"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected quality report rows")
	}
	lookup := make(map[string]struct {
		HasSchema     bool
		HasTester     bool
		NeedsTester   bool
		HasPipelineIO bool
		Score         int
	}, len(rows))
	for _, row := range rows {
		lookup[row.Name] = struct {
			HasSchema     bool
			HasTester     bool
			NeedsTester   bool
			HasPipelineIO bool
			Score         int
		}{
			HasSchema:     row.HasSchema,
			HasTester:     row.HasTester,
			NeedsTester:   row.NeedsTester,
			HasPipelineIO: row.HasPipelineIO,
			Score:         row.Score,
		}
	}
	openAI, ok := lookup["openai_summary"]
	if !ok {
		t.Fatal("missing openai_summary row")
	}
	if !openAI.HasSchema || !openAI.HasTester || !openAI.NeedsTester || !openAI.HasPipelineIO {
		t.Fatalf("unexpected openai_summary governance row: %#v", openAI)
	}
	if openAI.Score < 90 {
		t.Fatalf("unexpected openai_summary score: %#v", openAI)
	}
	clean, ok := lookup["clean"]
	if !ok {
		t.Fatal("missing clean row")
	}
	if clean.NeedsTester {
		t.Fatalf("clean should not require tester: %#v", clean)
	}
}

func TestFeatureFlagSnapshot_ExposesWebFlagsOnly(t *testing.T) {
	t.Setenv("TRACKER_API_KEY", "flag-key")
	srv := testServer(t)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/feature-flags/snapshot", nil)
	req.Header.Set("X-Tracker-Api-Key", "flag-key")
	req.Header.Set("X-Tracker-Subject", "user-123")
	req.Header.Set("X-Tracker-Experiment", "web.runtime_experiments=on")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status got %d body=%s", rr.Code, rr.Body.String())
	}
	var out struct {
		SubjectID string `json:"subject_id"`
		Flags     map[string]struct {
			Variant string `json:"variant"`
		} `json:"flags"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.SubjectID != "user-123" {
		t.Fatalf("subject_id got %q", out.SubjectID)
	}
	if out.Flags["web.runtime_experiments"].Variant != "on" {
		t.Fatalf("unexpected web.runtime_experiments: %#v", out.Flags)
	}
	if _, ok := out.Flags["runtime.executor_v2"]; ok {
		t.Fatalf("runtime-only flag should not be exposed: %#v", out.Flags)
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
	srv := api.NewServer(eng, db, config.DefaultApp(), defaultConfigPath(t), nil)
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
	srv := api.NewServer(eng, db, config.DefaultApp(), defaultConfigPath(t), nil)
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
	srv := api.NewServer(eng, nil, config.DefaultApp(), defaultConfigPath(t), nil)
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
	srv := api.NewServer(eng, db, config.DefaultApp(), defaultConfigPath(t), nil)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/operator/chat", strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status got %d want 401 body=%s", rr.Code, rr.Body.String())
	}
}
