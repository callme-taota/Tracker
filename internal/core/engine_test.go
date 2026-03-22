package core

import (
	"os"
	"testing"

	"Tracker/internal/config"
	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
)

// TestEngine_InitAndRunLoadsPipeline verifies that with default config we can load pipeline and init engine.
func TestEngine_InitAndRunLoadsPipeline(t *testing.T) {
	app, pipelinePath, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if pipelinePath == "" {
		t.Fatal("pipelinePath should be set")
	}
	if _, err := os.Stat(pipelinePath); err != nil {
		t.Skipf("pipeline file %s not found (run from project root)", pipelinePath)
	}
	pipe, err := pipeline.LoadFromFile(pipelinePath)
	if err != nil {
		t.Fatal(err)
	}
	eng := New()
	global := plugin.Config{
		"api_key":   os.Getenv("OPENAI_API_KEY"),
		"bot_token": os.Getenv("TELEGRAM_BOT_TOKEN"),
		"chat_id":   os.Getenv("TELEGRAM_CHAT_ID"),
	}
	for k, v := range app.Env {
		global[k] = v
	}
	if err := eng.Init(global); err != nil {
		t.Fatal(err)
	}
	// Don't actually Run (would hit network). Just ensure pipeline has expected structure.
	if len(pipe.Stages) == 0 {
		t.Fatal("pipeline has no stages")
	}
	if pipe.Stages[0].PluginType != plugin.TypeSource {
		t.Errorf("first stage should be source got %s", pipe.Stages[0].PluginType)
	}
}

// TestEngine_NewRegistersPlugins verifies all built-in plugins are registered.
func TestEngine_NewRegistersPlugins(t *testing.T) {
	eng := New()
	all := eng.PM.ListAll()
	want := map[string]bool{
		"rss": true, "news": true, "reddit": true, "github_trending": true, "x_fetch": true, "telegram_fetch": true,
		"clean": true, "dedup_basic": true, "simhash_dedup": true, "embedding_dedup": true,
		"cluster_kmeans": true, "llm_event_dedup": true,
		"openai_summary": true, "keyword_interest": true, "llm_operator": true,
		"telegram": true, "feishu": true, "discord": true, "notion": true,
	}
	for _, p := range all {
		delete(want, p.Name())
	}
	if len(want) > 0 {
		t.Errorf("missing plugins: %v", want)
	}
}
