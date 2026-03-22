package core

import (
	"context"
	"os"
	"testing"

	"Tracker/internal/plugin"
)

func TestEngine_PluginHostAlwaysPresent(t *testing.T) {
	_ = os.Unsetenv("TRACKER_EXTERNAL_PLUGINS_FILE")
	eng := New()
	defer eng.Close()
	if eng.PluginHost == nil {
		t.Fatal("PluginHost should always be non-nil for reload/shutdown")
	}
}

func TestEngine_ReloadExternalPlugins_EmptyConfig(t *testing.T) {
	_ = os.Unsetenv("TRACKER_EXTERNAL_PLUGINS_FILE")
	eng := New()
	defer eng.Close()
	if err := eng.Init(plugin.Config{}); err != nil {
		t.Fatal(err)
	}
	if err := eng.ReloadExternalPlugins(context.Background()); err != nil {
		t.Fatal(err)
	}
}
