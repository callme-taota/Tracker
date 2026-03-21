package registry

import (
	"encoding/json"

	"Tracker/internal/plugin"
	"Tracker/internal/pluginhub"
)

// RegisterBuiltinManifests registers JSON-schema-light manifests and format tokens for all built-in plugins.
func RegisterBuiltinManifests(h *pluginhub.Hub) {
	if h == nil {
		return
	}
	for _, m := range builtinManifests() {
		h.RegisterManifest(m)
	}
}

func builtinManifests() []plugin.Manifest {
	feedsSchema := json.RawMessage(`{"type":"object","properties":{"feeds":{"type":"array","description":"RSS feed URLs (optional if set globally)"}}}`)
	itemRef := json.RawMessage(`{"$comment":"tracker.item.v1","type":"object"}`)
	return []plugin.Manifest{
		{
			ID: "rss", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "RSS",
			ConfigSchema: feedsSchema, OutputFormats: []string{plugin.FormatTrackerItemV1},
			OutputSchema: itemRef,
		},
		{
			ID: "news", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "News",
			OutputFormats: []string{plugin.FormatTrackerItemV1}, OutputSchema: itemRef,
		},
		{
			ID: "reddit", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "Reddit",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"subreddits":{"type":"array"},"subreddit":{"type":"string"},"user_agent":{"type":"string"},"limit":{"type":"string"}}}`),
			OutputFormats: []string{plugin.FormatTrackerItemV1}, OutputSchema: itemRef,
		},
		{
			ID: "github_trending", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "GitHub Search",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"sort":{"type":"string"},"token":{"type":"string"}}}`),
			OutputFormats: []string{plugin.FormatTrackerItemV1}, OutputSchema: itemRef,
		},
		{
			ID: "x_fetch", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "X (Twitter)",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"bearer_token":{"type":"string"},"query":{"type":"string"}}}`),
			OutputFormats: []string{plugin.FormatTrackerItemV1}, OutputSchema: itemRef,
		},
		{
			ID: "telegram_fetch", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "Telegram fetch",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"bot_token":{"type":"string"},"text_contains":{"type":"string"}}}`),
			OutputFormats: []string{plugin.FormatTrackerItemV1}, OutputSchema: itemRef,
		},
		{
			ID: "clean", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "Clean / Format",
			InputFormats: []string{plugin.FormatTrackerItemV1, "*"}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef,
		},
		{
			ID: "dedup_basic", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "Dedup (URL/title/hash)",
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef,
		},
		{
			ID: "simhash_dedup", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "SimHash dedup",
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef,
		},
		{
			ID: "embedding_dedup", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "Embedding dedup",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"api_key":{"type":"string"},"model":{"type":"string"}}}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef,
		},
		{
			ID: "cluster_kmeans", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "Cluster (online k-means)",
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef,
		},
		{
			ID: "llm_event_dedup", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "LLM event dedup",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"api_key":{"type":"string"},"model":{"type":"string"}}}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef,
		},
		{
			ID: "openai_summary", Version: "1.0", Kind: plugin.TypeSummary, DisplayName: "OpenAI Summary",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"include_mermaid":{"type":"string"}}}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef,
		},
		{
			ID: "keyword_interest", Version: "1.0", Kind: plugin.TypeInterest, DisplayName: "Keyword Interest",
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef,
		},
		{
			ID: "telegram", Version: "1.0", Kind: plugin.TypeDispatch, DisplayName: "Telegram",
			InputFormats: []string{plugin.FormatTrackerItemV1}, InputSchema: itemRef,
		},
		{
			ID: "feishu", Version: "1.0", Kind: plugin.TypeDispatch, DisplayName: "Feishu",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"webhook_url":{"type":"string","description":"Custom bot webhook URL"}}}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, InputSchema: itemRef,
		},
		{
			ID: "discord", Version: "1.0", Kind: plugin.TypeDispatch, DisplayName: "Discord",
			InputFormats: []string{plugin.FormatTrackerItemV1}, InputSchema: itemRef,
		},
		{
			ID: "notion", Version: "1.0", Kind: plugin.TypeDispatch, DisplayName: "Notion",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"token":{"type":"string"},"database_id":{"type":"string"}}}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, InputSchema: itemRef,
		},
	}
}
