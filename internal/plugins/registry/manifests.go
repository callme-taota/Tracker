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
	boolPtr := func(v bool) *bool { return &v }
	sourceIO := &plugin.PipelineIOSpec{
		EmitsItems:         boolPtr(true),
		AcceptsItems:       boolPtr(false),
		AllowOutboundEdges: boolPtr(true),
	}
	transformIO := &plugin.PipelineIOSpec{
		EmitsItems:         boolPtr(true),
		AcceptsItems:       boolPtr(true),
		AllowOutboundEdges: boolPtr(true),
	}
	dispatchIO := &plugin.PipelineIOSpec{
		EmitsItems:         boolPtr(false),
		AcceptsItems:       boolPtr(true),
		AllowOutboundEdges: boolPtr(false),
	}
	operatorIO := &plugin.PipelineIOSpec{
		EmitsItems:         boolPtr(false),
		AcceptsItems:       boolPtr(false),
		AllowOutboundEdges: boolPtr(false),
	}
	emptyConfig := json.RawMessage(`{"type":"object","properties":{},"additionalProperties":true}`)
	feedsSchema := json.RawMessage(`{"type":"object","properties":{"feeds":{"type":"array","items":{"type":"string"},"description":"RSS feed URLs"},"user_agent":{"type":"string","description":"Optional custom User-Agent header"},"time_from":{"type":"string","description":"Optional RFC3339 start time filter"},"time_to":{"type":"string","description":"Optional RFC3339 end time filter"}},"additionalProperties":true}`)
	itemRef := json.RawMessage(`{"$comment":"tracker.item.v1","type":"object"}`)
	return []plugin.Manifest{
		{
			ID: "rss", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "RSS",
			ConfigSchema: feedsSchema, OutputFormats: []string{plugin.FormatTrackerItemV1},
			OutputSchema: itemRef, PipelineIO: sourceIO,
		},
		{
			ID: "news", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "News",
			ConfigSchema:  json.RawMessage(`{"type":"object","properties":{"feeds":{"type":"array","items":{"type":"string"},"description":"Explicit news feeds override presets"},"presets":{"type":"array","items":{"type":"string","enum":["global","cn"]},"description":"Preset feed groups to enable"},"extra_feeds":{"type":"array","items":{"type":"string"},"description":"Additional feeds appended after presets"}},"additionalProperties":true}`),
			OutputFormats: []string{plugin.FormatTrackerItemV1}, OutputSchema: itemRef, PipelineIO: sourceIO,
		},
		{
			ID: "reddit", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "Reddit",
			ConfigSchema:  json.RawMessage(`{"type":"object","properties":{"subreddits":{"type":"array","items":{"type":"string"}},"subreddit":{"type":"string"},"user_agent":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":100}},"additionalProperties":true}`),
			OutputFormats: []string{plugin.FormatTrackerItemV1}, OutputSchema: itemRef,
			PipelineIO: sourceIO,
		},
		{
			ID: "github_trending", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "GitHub Search",
			ConfigSchema:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"sort":{"type":"string","description":"GitHub repository search sort key"},"token":{"type":"string","description":"Optional GitHub API token"},"per_page":{"type":"integer","minimum":1,"maximum":100}},"additionalProperties":true}`),
			OutputFormats: []string{plugin.FormatTrackerItemV1}, OutputSchema: itemRef,
			PipelineIO: sourceIO,
		},
		{
			ID: "x_fetch", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "X (Twitter)",
			ConfigSchema:  json.RawMessage(`{"type":"object","properties":{"bearer_token":{"type":"string"},"query":{"type":"string"},"max_results":{"type":"integer","minimum":10,"maximum":100}},"additionalProperties":true}`),
			OutputFormats: []string{plugin.FormatTrackerItemV1}, OutputSchema: itemRef,
			PipelineIO: sourceIO,
		},
		{
			ID: "telegram_fetch", Version: "1.0", Kind: plugin.TypeSource, DisplayName: "Telegram fetch",
			ConfigSchema:  json.RawMessage(`{"type":"object","properties":{"bot_token":{"type":"string"},"text_contains":{"type":"string"}},"additionalProperties":true}`),
			OutputFormats: []string{plugin.FormatTrackerItemV1}, OutputSchema: itemRef,
			PipelineIO: sourceIO,
		},
		{
			ID: "clean", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "Clean / Format",
			ConfigSchema: emptyConfig,
			InputFormats: []string{plugin.FormatTrackerItemV1, "*"}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef, PipelineIO: transformIO,
		},
		{
			ID: "dedup_basic", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "Dedup (URL/title/hash)",
			ConfigSchema: emptyConfig,
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef, PipelineIO: transformIO,
		},
		{
			ID: "simhash_dedup", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "SimHash dedup",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"hamming_threshold":{"type":"integer","minimum":1,"maximum":10,"description":"Near-duplicate cutoff in Hamming distance"}},"additionalProperties":true}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef, PipelineIO: transformIO,
		},
		{
			ID: "embedding_dedup", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "Embedding dedup",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"api_key":{"type":"string"},"description":"OpenAI-compatible API key"},"model":{"type":"string"},"base_url":{"type":"string","description":"OpenAI-compatible base URL"},"threshold":{"type":"number","minimum":0,"maximum":1}},"additionalProperties":true}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef, PipelineIO: transformIO,
		},
		{
			ID: "cluster_kmeans", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "Cluster (online k-means)",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"k":{"type":"integer","minimum":1,"maximum":32,"description":"Number of online centroids"}},"additionalProperties":true}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef, PipelineIO: transformIO,
		},
		{
			ID: "llm_event_dedup", Version: "1.0", Kind: plugin.TypeProcessor, DisplayName: "LLM event dedup",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"api_key":{"type":"string"},"description":"OpenAI-compatible API key"},"model":{"type":"string"},"base_url":{"type":"string","description":"OpenAI-compatible base URL"}},"additionalProperties":true}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef, PipelineIO: transformIO,
		},
		{
			ID: "openai_summary", Version: "1.0", Kind: plugin.TypeSummary, DisplayName: "OpenAI Summary",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"api_key":{"type":"string","description":"OpenAI-compatible API key"},"model":{"type":"string"},"base_url":{"type":"string","description":"OpenAI-compatible base URL"},"include_mermaid":{"type":"boolean","description":"Append a Mermaid diagram after the summary"}},"additionalProperties":true}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef, PipelineIO: transformIO,
		},
		{
			ID: "keyword_interest", Version: "1.0", Kind: plugin.TypeInterest, DisplayName: "Keyword Interest",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"keywords":{"type":"array","items":{"type":"string"},"description":"Keywords used to score interest"}},"additionalProperties":true}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, OutputFormats: []string{plugin.FormatTrackerItemV1},
			InputSchema: itemRef, OutputSchema: itemRef, PipelineIO: transformIO,
		},
		{
			ID: "telegram", Version: "1.0", Kind: plugin.TypeDispatch, DisplayName: "Telegram",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"bot_token":{"type":"string"},"chat_id":{"type":"string"},"api_base_url":{"type":"string","description":"Advanced: override Telegram Bot API base URL"}},"additionalProperties":true}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, InputSchema: itemRef, PipelineIO: dispatchIO,
		},
		{
			ID: "feishu", Version: "1.0", Kind: plugin.TypeDispatch, DisplayName: "Feishu",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"webhook_url":{"type":"string","description":"Custom bot webhook URL"}},"additionalProperties":true}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, InputSchema: itemRef, PipelineIO: dispatchIO,
		},
		{
			ID: "discord", Version: "1.0", Kind: plugin.TypeDispatch, DisplayName: "Discord",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"webhook_url":{"type":"string"},"username":{"type":"string","description":"Optional webhook username override"}},"additionalProperties":true}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, InputSchema: itemRef, PipelineIO: dispatchIO,
		},
		{
			ID: "notion", Version: "1.0", Kind: plugin.TypeDispatch, DisplayName: "Notion",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"token":{"type":"string"},"database_id":{"type":"string"},"base_url":{"type":"string","description":"Advanced: override Notion API base URL"}},"additionalProperties":true}`),
			InputFormats: []string{plugin.FormatTrackerItemV1}, InputSchema: itemRef, PipelineIO: dispatchIO,
		},
		{
			ID: "llm_operator", Version: "1.0.0", Kind: plugin.TypeOperator, DisplayName: "LLM operator (API)",
			ConfigSchema: json.RawMessage(`{"type":"object","properties":{"llm_profile":{"type":"string","description":"LLM profile key (default, cheap, heavy)"},"max_tool_rounds":{"type":"integer"},"system_prompt_extra":{"type":"string"}}}`),
			PipelineIO:   operatorIO,
		},
	}
}
