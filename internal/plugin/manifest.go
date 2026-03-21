package plugin

import "encoding/json"

// FormatTrackerItemV1 is the default IO format token for article-shaped pipeline data.
const FormatTrackerItemV1 = "tracker.item.v1"

// Manifest describes plugin metadata, JSON Schemas, and format compatibility for the PluginHub.
type Manifest struct {
	ID             string          `json:"id"`
	Version        string          `json:"version"`
	Kind           Type            `json:"kind"`
	DisplayName    string          `json:"display_name,omitempty"`
	ConfigSchema   json.RawMessage `json:"config_schema,omitempty"`
	InputSchema    json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema   json.RawMessage `json:"output_schema,omitempty"`
	InputFormats   []string        `json:"input_formats,omitempty"`
	OutputFormats  []string        `json:"output_formats,omitempty"`
	CompatibleWith []string        `json:"compatible_with,omitempty"` // explicit upstream plugin ids
	ValidatorRef   string          `json:"validator_ref,omitempty"`
}

// ManifestProvider is implemented by plugins that supply a static manifest.
type ManifestProvider interface {
	Manifest() Manifest
}
