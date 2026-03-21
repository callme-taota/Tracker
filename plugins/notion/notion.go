package notion

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// Plugin creates a Notion page in a database (API v2022-06-28).
type Plugin struct {
	token  string
	client *http.Client
}

func New() plugin.Plugin {
	return &Plugin{client: &http.Client{Timeout: 45 * time.Second}}
}

func (p *Plugin) Name() string             { return "notion" }
func (p *Plugin) Version() string          { return "1.0" }
func (p *Plugin) Type() plugin.Type        { return plugin.TypeDispatch }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.token = plugin.GetString(cfg, "token")
	if p.token == "" {
		p.token = os.Getenv("NOTION_TOKEN")
	}
	return nil
}

func (p *Plugin) ExecuteDispatch(in *model.Item, cfg plugin.Config) error {
	if in == nil {
		return nil
	}
	token := plugin.GetString(cfg, "token")
	if token == "" {
		token = p.token
	}
	db := plugin.GetString(cfg, "database_id")
	if token == "" || db == "" {
		return nil
	}
	body := map[string]interface{}{
		"parent": map[string]string{"database_id": db},
		"properties": map[string]interface{}{
			"Name": map[string]interface{}{
				"title": []map[string]interface{}{
					{"text": map[string]string{"content": in.Title}},
				},
			},
		},
		"children": []interface{}{
			map[string]interface{}{
				"object": "block",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]string{"content": in.URL + "\n\n" + truncate(in.Content, 1800)}},
					},
				},
			},
		},
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, "https://api.notion.com/v1/pages", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &model.ItemError{Code: "notion_api", Message: string(b)}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
