package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadDefaultPipeline verifies configs/default.yaml (or config.yaml) can be loaded when present.
func TestLoadDefaultPipeline(t *testing.T) {
	for _, name := range []string{"configs/default.yaml", "config.yaml"} {
		if _, err := os.Stat(name); err != nil {
			continue
		}
		pipe, err := LoadFromFile(name)
		if err != nil {
			t.Fatalf("LoadFromFile(%s): %v", name, err)
		}
		if pipe.Name == "" {
			t.Errorf("LoadFromFile(%s): empty name", name)
		}
		if len(pipe.Stages) == 0 {
			t.Errorf("LoadFromFile(%s): no stages", name)
		}
		return
	}
	t.Skip("no configs/default.yaml or config.yaml in cwd")
}

// TestLoadFromFile_RelativePath loads from project configs when run from project root.
func TestLoadFromFile_RelativePath(t *testing.T) {
	path := "configs/default.yaml"
	if _, err := os.Stat(path); err != nil {
		t.Skip("configs/default.yaml not found (run from project root)")
	}
	abs, _ := filepath.Abs(path)
	pipe, err := LoadFromFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if len(pipe.Stages) < 2 {
		t.Errorf("expected at least 2 stages got %d", len(pipe.Stages))
	}
}
