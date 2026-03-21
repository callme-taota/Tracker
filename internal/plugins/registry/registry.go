// Package registry is the single place that registers all built-in plugins.
// Core engine must not import individual plugins/* packages — only this package does.
package registry

import (
	"Tracker/internal/plugin"
	"Tracker/internal/pluginhub"
	"Tracker/plugins/clean"
	"Tracker/plugins/cluster_kmeans"
	"Tracker/plugins/dedup_basic"
	"Tracker/plugins/discord"
	"Tracker/plugins/embedding_dedup"
	"Tracker/plugins/feishu"
	"Tracker/plugins/github_trending"
	"Tracker/plugins/keyword_interest"
	"Tracker/plugins/llm_event_dedup"
	"Tracker/plugins/news"
	"Tracker/plugins/notion"
	"Tracker/plugins/openai_summary"
	"Tracker/plugins/reddit"
	"Tracker/plugins/rss"
	"Tracker/plugins/simhash_dedup"
	"Tracker/plugins/telegram"
	"Tracker/plugins/telegram_fetch"
	"Tracker/plugins/x_fetch"
)

// PluginRegistrar is satisfied by *core.PluginManager (avoids import cycle).
type PluginRegistrar interface {
	Register(p plugin.Plugin)
}

// BuiltinPlugins returns all built-in plugin instances (for tests / introspection).
func BuiltinPlugins() []plugin.Plugin {
	return []plugin.Plugin{
		rss.New(),
		news.New(),
		reddit.New(),
		github_trending.New(),
		x_fetch.New(),
		telegram_fetch.New(),
		clean.New(),
		dedup_basic.New(),
		simhash_dedup.New(),
		embedding_dedup.New(),
		cluster_kmeans.New(),
		llm_event_dedup.New(),
		openai_summary.New(),
		keyword_interest.New(),
		telegram.New(),
		feishu.New(),
		discord.New(),
		notion.New(),
	}
}

// RegisterAll registers every built-in plugin on the manager.
func RegisterAll(reg PluginRegistrar) {
	RegisterAllWithHub(reg, nil)
}

// RegisterAllWithHub registers plugins and, if hub is non-nil, builtin manifests and validators.
func RegisterAllWithHub(reg PluginRegistrar, hub *pluginhub.Hub) {
	for _, p := range BuiltinPlugins() {
		reg.Register(p)
	}
	RegisterBuiltinManifests(hub)
}
