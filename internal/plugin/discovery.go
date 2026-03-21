package plugin

// BuiltinPluginIDs returns the list of built-in plugin names (for CLI "plugin list" and install).
// MVP does not scan directories; all plugins are registered by the engine.
func BuiltinPluginIDs() []string {
	return []string{
		"rss",
		"news",
		"reddit",
		"github_trending",
		"x_fetch",
		"telegram_fetch",
		"clean",
		"dedup_basic",
		"simhash_dedup",
		"embedding_dedup",
		"cluster_kmeans",
		"llm_event_dedup",
		"openai_summary",
		"keyword_interest",
		"telegram",
		"feishu",
		"discord",
		"notion",
	}
}
