package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// App holds application-wide config (overridable by env and flags).
type App struct {
	// DBPath is the SQLite database path. Env: TRACKER_DB_PATH
	DBPath string `yaml:"db_path"`
	// PipelinePath is the path to pipeline YAML. Env: TRACKER_PIPELINE_PATH
	PipelinePath string `yaml:"pipeline_path"`
	// Schedule is a cron expression for auto-running the pipeline (e.g. "0 */2 * * *" = every 2h). Empty = disabled. Env: TRACKER_SCHEDULE
	Schedule string `yaml:"schedule"`
	// ServePort is the HTTP port for web UI. Env: TRACKER_PORT
	ServePort int `yaml:"serve_port"`
	// Env can hold key-value overrides for pipeline (e.g. api_key). Keys can be UPPER_SNAKE and will be set as env for pipeline run.
	Env map[string]string `yaml:"env"`
}

// File is the root structure of tracker config file (tracker.yaml or config.yaml).
type File struct {
	App      App    `yaml:"app"`
	Pipeline string `yaml:"pipeline"` // optional: inline pipeline file path or leave empty to use app.pipeline_path
}

// DefaultApp returns default app config.
func DefaultApp() App {
	return App{
		DBPath:       "tracker.db",
		PipelinePath: "",
		Schedule:     "",
		ServePort:    8080,
		Env:          nil,
	}
}

// Load loads config from path. If path is empty, tries TRACKER_CONFIG, then tracker.yaml, then config.yaml.
// Returns app config (with env overrides applied) and the effective pipeline config path.
func Load(path string) (App, string, error) {
	app := DefaultApp()
	if path == "" {
		path = os.Getenv("TRACKER_CONFIG")
	}
	if path == "" {
		for _, p := range []string{"tracker.yaml", "config.yaml"} {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			var f File
			if err := yaml.Unmarshal(data, &f); err == nil && (f.App.DBPath != "" || f.App.Schedule != "" || f.App.PipelinePath != "" || f.App.ServePort != 0 || len(f.App.Env) > 0) {
				app = f.App
				if app.PipelinePath == "" && f.Pipeline != "" {
					app.PipelinePath = f.Pipeline
				}
				if app.PipelinePath == "" {
					app.PipelinePath = path
				}
			} else {
				app.PipelinePath = path
			}
		}
	}
	// Env overrides
	if v := os.Getenv("TRACKER_DB_PATH"); v != "" {
		app.DBPath = v
	}
	if v := os.Getenv("TRACKER_PIPELINE_PATH"); v != "" {
		app.PipelinePath = v
	}
	if v := os.Getenv("TRACKER_SCHEDULE"); v != "" {
		app.Schedule = v
	}
	if v := os.Getenv("TRACKER_PORT"); v != "" {
		if p := atoi(v); p > 0 {
			app.ServePort = p
		}
	}
	if app.PipelinePath == "" {
		if _, err := os.Stat("config.yaml"); err == nil {
			app.PipelinePath = "config.yaml"
		} else {
			app.PipelinePath = "configs/default.yaml"
		}
	}
	return app, app.PipelinePath, nil
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
