package discord

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

const maxContentLen = 2000

// Plugin implements Discord webhook dispatch.
type Plugin struct {
	webhookURL string
	httpClient *http.Client
}

// New returns a new discord plugin instance.
func New() plugin.Plugin {
	return &Plugin{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (p *Plugin) Name() string      { return "discord" }
func (p *Plugin) Version() string   { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeDispatch }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.webhookURL = sharedutil.StringWithEnv(cfg, "webhook_url", os.Getenv("DISCORD_WEBHOOK_URL"))
	return nil
}

// TestConfig performs a read-only webhook probe.
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	url := sharedutil.StringWithEnv(cfg, "webhook_url", p.webhookURL)
	if err := sharedutil.RequireFields(map[string]string{"webhook_url": url}); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// ExecuteDispatch sends the item to Discord via webhook.
func (p *Plugin) ExecuteDispatch(in *model.Item, cfg plugin.Config) error {
	if in == nil {
		return nil
	}
	url := sharedutil.StringWithEnv(cfg, "webhook_url", p.webhookURL)
	if url == "" {
		return &model.ItemError{Code: "discord_config", Message: "webhook_url is required"}
	}
	content := formatMessage(in)
	if len(content) > maxContentLen {
		content = content[:maxContentLen-3] + "..."
	}
	body := map[string]string{"content": content}
	if username := strings.TrimSpace(plugin.GetString(cfg, "username")); username != "" {
		body["username"] = username
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func formatMessage(in *model.Item) string {
	s := "**" + in.Title + "**"
	if in.URL != "" {
		s += "\n" + in.URL
	}
	if in.Summary != "" {
		s += "\n\n" + in.Summary
	}
	if len(in.KeyPoints) > 0 {
		s += "\n\n*Key points:*"
		for _, k := range in.KeyPoints {
			s += "\n• " + k
		}
	}
	return s
}
