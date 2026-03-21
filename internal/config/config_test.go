package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultApp(t *testing.T) {
	app := DefaultApp()
	if app.DBPath != "tracker.db" {
		t.Errorf("DBPath want tracker.db got %q", app.DBPath)
	}
	if app.ServePort != 8080 {
		t.Errorf("ServePort want 8080 got %d", app.ServePort)
	}
}

func TestLoad_EmptyPath_ReturnsNonEmptyPipelinePath(t *testing.T) {
	app, pipelinePath, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if pipelinePath == "" {
		t.Error("pipelinePath should be set (config.yaml or configs/default.yaml)")
	}
	if app.DBPath != "tracker.db" {
		t.Errorf("DBPath want tracker.db got %q", app.DBPath)
	}
}

func TestLoad_WithPipelineYAMLPath_TreatsAsPipelineOnly(t *testing.T) {
	dir := t.TempDir()
	pipePath := filepath.Join(dir, "pipe.yaml")
	if err := os.WriteFile(pipePath, []byte(`
name: test
stages:
  - name: s1
    plugin_type: source
    plugin_id: rss
    config: {}
`), 0644); err != nil {
		t.Fatal(err)
	}
	app, pipelinePath, err := Load(pipePath)
	if err != nil {
		t.Fatal(err)
	}
	if pipelinePath != pipePath {
		t.Errorf("pipelinePath want %q got %q", pipePath, pipelinePath)
	}
	if app.DBPath != "tracker.db" {
		t.Errorf("DBPath should remain default got %q", app.DBPath)
	}
}

func TestLoad_WithTrackerYAML_LoadsAppAndPipelinePath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "tracker.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
app:
  db_path: my.db
  pipeline_path: pipes/main.yaml
  schedule: "0 * * * *"
  serve_port: 9000
`), 0644); err != nil {
		t.Fatal(err)
	}
	app, pipelinePath, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if app.DBPath != "my.db" {
		t.Errorf("DBPath want my.db got %q", app.DBPath)
	}
	if app.PipelinePath != "pipes/main.yaml" {
		t.Errorf("PipelinePath want pipes/main.yaml got %q", app.PipelinePath)
	}
	if app.Schedule != "0 * * * *" {
		t.Errorf("Schedule want 0 * * * * got %q", app.Schedule)
	}
	if app.ServePort != 9000 {
		t.Errorf("ServePort want 9000 got %d", app.ServePort)
	}
	if pipelinePath != "pipes/main.yaml" {
		t.Errorf("pipelinePath want pipes/main.yaml got %q", pipelinePath)
	}
}

func TestAtoi(t *testing.T) {
	if n := atoi("8080"); n != 8080 {
		t.Errorf("atoi(8080) want 8080 got %d", n)
	}
	if n := atoi(""); n != 0 {
		t.Errorf("atoi() want 0 got %d", n)
	}
}
