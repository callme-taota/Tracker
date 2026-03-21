package pipeline

import "Tracker/internal/plugin"

// Stage defines one step in a pipeline (name, plugin type, plugin id, and config).
type Stage struct {
	Name      string       `yaml:"name"`
	PluginType plugin.Type `yaml:"plugin_type"`
	PluginID  string       `yaml:"plugin_id"`
	Config    plugin.Config `yaml:"config"`
}

// Pipeline defines an ordered list of stages.
type Pipeline struct {
	Name   string  `yaml:"name"`
	Stages []Stage `yaml:"stages"`
}
