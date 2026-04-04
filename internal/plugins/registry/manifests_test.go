package registry

import "testing"

func TestBuiltinManifestsCoverRegisteredPlugins(t *testing.T) {
	manifests := builtinManifests()
	if len(manifests) == 0 {
		t.Fatal("expected builtin manifests")
	}
	manifestByID := make(map[string]bool, len(manifests))
	for _, manifest := range manifests {
		manifestByID[manifest.ID] = true
		if len(manifest.ConfigSchema) == 0 {
			t.Fatalf("manifest %s missing config schema", manifest.ID)
		}
		if manifest.PipelineIO == nil {
			t.Fatalf("manifest %s missing pipeline io", manifest.ID)
		}
	}
	for _, builtin := range BuiltinPlugins() {
		if !manifestByID[builtin.Name()] {
			t.Fatalf("builtin plugin %s missing manifest", builtin.Name())
		}
	}
}
