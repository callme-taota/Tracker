package scheduler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScheduler_EmptySchedule_NoPanic(t *testing.T) {
	dir := t.TempDir()
	pipePath := filepath.Join(dir, "p.yaml")
	if err := os.WriteFile(pipePath, []byte(`
name: t
stages:
  - name: s
    plugin_type: source
    plugin_id: rss
    config:
      feeds: ["https://www.theverge.com/rss/index.xml"]
`), 0644); err != nil {
		t.Fatal(err)
	}
	runner := &Runner{PipelinePath: pipePath, DB: nil, Env: nil}
	sched := New(runner, "")
	sched.Start()
	sched.Stop()
}

func TestRunner_Run_InvalidPath_ReturnsError(t *testing.T) {
	runner := &Runner{PipelinePath: "/nonexistent/pipeline.yaml", DB: nil}
	_, err := runner.Run()
	if err == nil {
		t.Fatal("expected error for missing pipeline file")
	}
}
