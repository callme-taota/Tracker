package sse

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Write sets standard SSE headers. Call once per response.
func WriteHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

// Event writes one SSE event (event + data lines). flusher optional.
func Event(w http.ResponseWriter, event, data string) error {
	if f, ok := w.(http.Flusher); ok {
		defer f.Flush()
	}
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, escapeData(data))
	return err
}

func escapeData(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", "")
}

// WriteChunk writes arbitrary text as a single data line (newlines stripped).
func WriteChunk(w http.ResponseWriter, chunk string) error {
	flat := ""
	for _, c := range chunk {
		if c == '\n' || c == '\r' {
			flat += " "
			continue
		}
		flat += string(c)
	}
	_, err := io.WriteString(w, "data: "+flat+"\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return err
}
