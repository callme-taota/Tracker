package pluginruntime

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FileConfig is the shape of external_plugins.yaml.
type FileConfig struct {
	Plugins []PluginSpec `yaml:"plugins"`
}

// PluginSpec defines one subprocess plugin.
type PluginSpec struct {
	ID      string            `yaml:"id"`
	Command []string          `yaml:"command"`
	Workdir string            `yaml:"workdir"`
	Env     map[string]string `yaml:"env"`
}

// LoadConfigFile reads YAML plugin definitions from path. Empty path returns empty config.
func LoadConfigFile(path string) (FileConfig, error) {
	var fc FileConfig
	if path == "" {
		return fc, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fc, nil
		}
		return fc, err
	}
	if err := yaml.Unmarshal(b, &fc); err != nil {
		return fc, fmt.Errorf("pluginruntime: parse %s: %w", path, err)
	}
	return fc, nil
}
