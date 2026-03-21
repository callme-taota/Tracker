package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"Tracker/internal/plugin"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.yaml")
	content := `
name: mypipe
stages:
  - name: rss
    plugin_type: source
    plugin_id: rss
    config:
      feeds:
        - https://example.com/feed.xml
  - name: clean
    plugin_type: processor
    plugin_id: clean
    config: {}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	pipe, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Name != "mypipe" {
		t.Errorf("Name want mypipe got %q", pipe.Name)
	}
	if len(pipe.Stages) != 2 {
		t.Fatalf("Stages want 2 got %d", len(pipe.Stages))
	}
	if pipe.Stages[0].PluginType != plugin.TypeSource || pipe.Stages[0].PluginID != "rss" {
		t.Errorf("Stage 0 want source/rss got %s/%s", pipe.Stages[0].PluginType, pipe.Stages[0].PluginID)
	}
	if pipe.Stages[1].PluginType != plugin.TypeProcessor || pipe.Stages[1].PluginID != "clean" {
		t.Errorf("Stage 1 want processor/clean got %s/%s", pipe.Stages[1].PluginType, pipe.Stages[1].PluginID)
	}
}

func TestLoadFromFile_NotExist(t *testing.T) {
	_, err := LoadFromFile(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
