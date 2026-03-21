package pipeline

import (
	"os"

	"Tracker/internal/plugin"
	"gopkg.in/yaml.v3"
)

// FilePipeline is the YAML file shape (plugin_type as string).
type FilePipeline struct {
	Name   string        `yaml:"name"`
	Stages []FileStage   `yaml:"stages"`
}

type FileStage struct {
	Name       string                 `yaml:"name"`
	PluginType string                 `yaml:"plugin_type"`
	PluginID   string                 `yaml:"plugin_id"`
	Config     map[string]interface{}  `yaml:"config"`
}

// LoadFromFile reads a pipeline from a YAML file.
func LoadFromFile(path string) (*Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fp FilePipeline
	if err := yaml.Unmarshal(data, &fp); err != nil {
		return nil, err
	}
	return fileToPipeline(&fp), nil
}

func fileToPipeline(fp *FilePipeline) *Pipeline {
	pipe := &Pipeline{Name: fp.Name}
	for _, fs := range fp.Stages {
		pipe.Stages = append(pipe.Stages, Stage{
			Name:       fs.Name,
			PluginType: plugin.Type(fs.PluginType),
			PluginID:   fs.PluginID,
			Config:     plugin.Config(fs.Config),
		})
	}
	return pipe
}
