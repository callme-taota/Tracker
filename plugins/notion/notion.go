package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
	"Tracker/plugins/sharedutil"
)

// Plugin creates a Notion page in a database (API v2022-06-28).
type Plugin struct {
	token   string
	baseURL string
	client  *http.Client
}

func New() plugin.Plugin {
	return &Plugin{
		baseURL: "https://api.notion.com/v1",
		client:  &http.Client{Timeout: 45 * time.Second},
	}
}

func (p *Plugin) Name() string      { return "notion" }
func (p *Plugin) Version() string   { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeDispatch }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.token = sharedutil.StringWithEnv(cfg, "token", os.Getenv("NOTION_TOKEN"))
	if baseURL := strings.TrimSpace(plugin.GetString(cfg, "base_url")); baseURL != "" {
		p.baseURL = strings.TrimRight(baseURL, "/")
	}
	return nil
}

// TestConfig validates token and database access without writing a page.
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	token, dbID, baseURL := p.resolveConfig(cfg)
	if err := sharedutil.RequireFields(map[string]string{"token": token, "database_id": dbID}); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/databases/"+dbID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", "2022-06-28")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notion database: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (p *Plugin) ExecuteDispatch(in *model.Item, cfg plugin.Config) error {
	if in == nil {
		return nil
	}
	token, db, baseURL := p.resolveConfig(cfg)
	if token == "" || db == "" {
		return &model.ItemError{Code: "notion_config", Message: "token and database_id are required"}
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
	req, err := http.NewRequest(http.MethodPost, baseURL+"/pages", bytes.NewReader(raw))
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

func (p *Plugin) resolveConfig(cfg plugin.Config) (token, databaseID, baseURL string) {
	token = sharedutil.StringWithEnv(cfg, "token", p.token)
	databaseID = strings.TrimSpace(plugin.GetString(cfg, "database_id"))
	baseURL = p.baseURL
	if v := strings.TrimSpace(plugin.GetString(cfg, "base_url")); v != "" {
		baseURL = strings.TrimRight(v, "/")
	}
	if baseURL == "" {
		baseURL = "https://api.notion.com/v1"
	}
	return token, databaseID, baseURL
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
