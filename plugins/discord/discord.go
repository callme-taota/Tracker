package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

const maxContentLen = 2000

// Plugin implements Discord webhook dispatch.
type Plugin struct {
	webhookURL string
}

// New returns a new discord plugin instance.
func New() plugin.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return "discord" }
func (p *Plugin) Version() string { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeDispatch }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.webhookURL = plugin.GetString(cfg, "webhook_url")
	if p.webhookURL == "" {
		p.webhookURL = os.Getenv("DISCORD_WEBHOOK_URL")
	}
	return nil
}

// ExecuteDispatch sends the item to Discord via webhook.
func (p *Plugin) ExecuteDispatch(in *model.Item, cfg plugin.Config) error {
	if in == nil {
		return nil
	}
	url := plugin.GetString(cfg, "webhook_url")
	if url == "" {
		url = p.webhookURL
	}
	if url == "" {
		url = os.Getenv("DISCORD_WEBHOOK_URL")
	}
	if url == "" {
		return nil
	}
	content := formatMessage(in)
	if len(content) > maxContentLen {
		content = content[:maxContentLen-3] + "..."
	}
	body := map[string]string{"content": content}
	raw, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook: %d", resp.StatusCode)
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
