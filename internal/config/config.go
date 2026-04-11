package config

import (
	"os"

	"Tracker/internal/featureflags"
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
	// Release defines deployment metadata for rollout / AB evaluation.
	Release featureflags.ReleaseConfig `yaml:"release"`
	// FeatureFlags holds codepath and rollout definitions.
	FeatureFlags []featureflags.FlagConfig `yaml:"feature_flags"`
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
		Release: featureflags.ReleaseConfig{
			Channel: "stable",
			Ring:    "global",
		},
		FeatureFlags: featureflags.DefaultDefinitions(),
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
			if err := yaml.Unmarshal(data, &f); err == nil && hasAppConfig(f.App) {
				app = f.App
				app.FeatureFlags = mergeFeatureFlags(featureflags.DefaultDefinitions(), app.FeatureFlags)
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
	if v := os.Getenv("TRACKER_RELEASE_CHANNEL"); v != "" {
		app.Release.Channel = v
	}
	if v := os.Getenv("TRACKER_RELEASE_RING"); v != "" {
		app.Release.Ring = v
	}
	if v := os.Getenv("TRACKER_RELEASE_VERSION"); v != "" {
		app.Release.Version = v
	}
	if v := os.Getenv("TRACKER_RELEASE_INSTANCE"); v != "" {
		app.Release.Instance = v
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

// GlobalPluginConfig returns the process-level plugin config for Engine.Init / InitRemote.
func (a App) GlobalPluginConfig() map[string]interface{} {
	global := map[string]interface{}{
		"api_key":                    os.Getenv("OPENAI_API_KEY"),
		"bot_token":                  os.Getenv("TELEGRAM_BOT_TOKEN"),
		"chat_id":                    os.Getenv("TELEGRAM_CHAT_ID"),
		"__tracker_release_channel":  a.Release.Channel,
		"__tracker_release_ring":     a.Release.Ring,
		"__tracker_release_version":  a.Release.Version,
		"__tracker_release_instance": a.Release.Instance,
	}
	for k, v := range a.Env {
		global[k] = v
	}
	return global
}

func hasAppConfig(app App) bool {
	return app.DBPath != "" || app.Schedule != "" || app.PipelinePath != "" || app.ServePort != 0 || len(app.Env) > 0 || len(app.FeatureFlags) > 0 || app.Release.Channel != "" || app.Release.Ring != "" || app.Release.Version != "" || app.Release.Instance != ""
}

func mergeFeatureFlags(base []featureflags.FlagConfig, overrides []featureflags.FlagConfig) []featureflags.FlagConfig {
	if len(overrides) == 0 {
		return base
	}
	merged := make(map[string]featureflags.FlagConfig, len(base)+len(overrides))
	for _, flag := range base {
		merged[flag.Key] = flag
	}
	for _, flag := range overrides {
		if flag.Key == "" {
			continue
		}
		merged[flag.Key] = flag
	}
	out := make([]featureflags.FlagConfig, 0, len(merged))
	for _, flag := range merged {
		out = append(out, flag)
	}
	return out
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
