package pluginruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFile_EmptyPath(t *testing.T) {
	fc, err := LoadConfigFile("")
	if err != nil {
		t.Fatal(err)
	}
	if len(fc.Plugins) != 0 {
		t.Fatalf("got %d plugins", len(fc.Plugins))
	}
}

func TestLoadConfigFile_MissingFile(t *testing.T) {
	fc, err := LoadConfigFile(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fc.Plugins) != 0 {
		t.Fatal("expected empty")
	}
}

func TestLoadConfigFile_ValidYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "ext.yaml")
	content := `plugins:
  - id: a
    command: ["/bin/echo"]
    workdir: /tmp
    env:
      FOO: bar
`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	fc, err := LoadConfigFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(fc.Plugins) != 1 {
		t.Fatalf("plugins %d", len(fc.Plugins))
	}
	if fc.Plugins[0].ID != "a" || len(fc.Plugins[0].Command) != 1 {
		t.Fatalf("%+v", fc.Plugins[0])
	}
	if fc.Plugins[0].Env["FOO"] != "bar" {
		t.Fatalf("env %+v", fc.Plugins[0].Env)
	}
}
