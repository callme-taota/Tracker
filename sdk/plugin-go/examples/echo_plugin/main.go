// Command echo_plugin is a minimal external Tracker plugin (processor: uppercases item title).
// Build: go build -o tracker-plugin-echo ./sdk/plugin-go/examples/echo_plugin
// Configure external_plugins.yaml with id echo_ext and command pointing to this binary.
package main

import (
	"encoding/json"
	"log"
	"strings"

	srv "Tracker/sdk/plugin-go/server"
)

func main() {
	manifest := json.RawMessage(`{
		"id": "echo_ext",
		"version": "1.0",
		"kind": "processor",
		"display_name": "Echo (external)",
		"input_formats": ["tracker.item.v1"],
		"output_formats": ["tracker.item.v1"]
	}`)
	h := func(op string, payload json.RawMessage) (interface{}, error) {
		switch op {
		case "handshake":
			var m interface{}
			_ = json.Unmarshal(manifest, &m)
			return map[string]interface{}{
				"plugin_id": "echo_ext",
				"version":   "1.0",
				"manifest":  m,
			}, nil
		case "health", "ready":
			return map[string]string{"status": "ok"}, nil
		case "init", "test_config":
			return map[string]interface{}{}, nil
		case "execute_source":
			return map[string]interface{}{"items": []interface{}{}}, nil
		case "execute_process":
			var in struct {
				Item   json.RawMessage       `json:"item"`
				Config map[string]interface{} `json:"config"`
			}
			if err := json.Unmarshal(payload, &in); err != nil {
				return nil, err
			}
			var it map[string]interface{}
			if err := json.Unmarshal(in.Item, &it); err != nil {
				return nil, err
			}
			if t, ok := it["title"].(string); ok {
				it["title"] = strings.ToUpper(t)
			}
			out, err := json.Marshal(it)
			if err != nil {
				return nil, err
			}
			return map[string]json.RawMessage{"item": out}, nil
		case "execute_dispatch":
			return map[string]interface{}{}, nil
		default:
			return nil, nil
		}
	}
	if err := srv.Run(h); err != nil {
		log.Fatal(err)
	}
}
