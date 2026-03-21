package model

import "time"

// Item is the unified data structure flowing through the pipeline (AGENT.md).
type Item struct {
	Title          string            `json:"title"`
	Content        string            `json:"content"`
	Summary        string            `json:"summary"`
	URL            string            `json:"url"`
	Timestamp      time.Time         `json:"timestamp"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Source         string            `json:"source"`
	KeyPoints      []string          `json:"key_points,omitempty"`
	InterestScore  float64           `json:"interest_score,omitempty"`
	MatchedTopics  []string          `json:"matched_topics,omitempty"`
	// Extra holds plugin-specific payloads (embeddings, simhash, cluster ids, etc.).
	Extra map[string]interface{} `json:"extra,omitempty"`
	// SchemaVersion hints the shape of Extra for cross-plugin compatibility.
	SchemaVersion string `json:"schema_version,omitempty"`
}

// ItemError is a structured error returned by pipeline stages.
type ItemError struct {
	Code    string `json:"error"`
	Message string `json:"message"`
}

func (e *ItemError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}
