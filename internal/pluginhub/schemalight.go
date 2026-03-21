package pluginhub

import (
	"encoding/json"
	"fmt"

	"Tracker/internal/plugin"
)

// jsonSchemaLight is a minimal subset used for config validation without a full JSON Schema engine.
type jsonSchemaLight struct {
	Type        string              `json:"type"`
	Required    []string            `json:"required"`
	Properties  map[string]any      `json:"properties"`
	MinItems    *int                `json:"minItems"`
	Description string              `json:"description"`
}

// ValidateConfigJSONSchema validates cfg against a small JSON-schema-like document:
// - if schema empty, skip
// - if type is object, ensure required keys exist in cfg
// - if properties.feeds or similar array fields have minItems in schema as extension, check slice length
func ValidateConfigJSONSchema(schemaJSON json.RawMessage, cfg plugin.Config) error {
	if len(schemaJSON) == 0 {
		return nil
	}
	var s jsonSchemaLight
	if err := json.Unmarshal(schemaJSON, &s); err != nil {
		return fmt.Errorf("config_schema: invalid JSON: %w", err)
	}
	if s.Type != "" && s.Type != "object" {
		// MVP: only object configs supported
		return nil
	}
	for _, key := range s.Required {
		if _, ok := cfg[key]; !ok {
			return fmt.Errorf("config_schema: missing required key %q", key)
		}
		v := cfg[key]
		if key == "feeds" {
			if arr, ok := v.([]any); ok && s.MinItems != nil && len(arr) < *s.MinItems {
				return fmt.Errorf("config_schema: %q must have at least %d items", key, *s.MinItems)
			}
			if arr, ok := v.([]string); ok && s.MinItems != nil && len(arr) < *s.MinItems {
				return fmt.Errorf("config_schema: %q must have at least %d items", key, *s.MinItems)
			}
		}
	}
	return nil
}
