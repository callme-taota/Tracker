package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
	"Tracker/plugins/sharedutil"
)

// Plugin implements Feishu (飞书/Lark) webhook dispatch.
type Plugin struct {
	webhookURL string
}

// New returns a new feishu plugin instance.
func New() plugin.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string      { return "feishu" }
func (p *Plugin) Version() string   { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeDispatch }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.webhookURL = sharedutil.StringWithEnv(cfg, "webhook_url", os.Getenv("FEISHU_WEBHOOK_URL"))
	return nil
}

// TestConfig sends a minimal text message to verify the webhook URL (implements plugin.ConfigTester).
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	url := sharedutil.StringWithEnv(cfg, "webhook_url", p.webhookURL)
	if url == "" {
		return fmt.Errorf("webhook_url is empty (set in config or FEISHU_WEBHOOK_URL)")
	}
	body := map[string]interface{}{
		"msg_type": "text",
		"content":  map[string]string{"text": "Tracker: connection test (safe to ignore)"},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bytes.TrimSpace(respBody)))
	}
	// Custom bot returns JSON; treat non-zero business codes as failure.
	var v struct {
		Code       int    `json:"code"`
		Msg        string `json:"msg"`
		StatusCode int    `json:"StatusCode"`
	}
	if json.Unmarshal(respBody, &v) == nil {
		if v.Code != 0 {
			return fmt.Errorf("feishu: code=%d msg=%s", v.Code, v.Msg)
		}
		if v.StatusCode != 0 && v.StatusCode != 200 {
			return fmt.Errorf("feishu: StatusCode=%d", v.StatusCode)
		}
	}
	return nil
}

// ExecuteDispatch sends the item to Feishu via webhook.
func (p *Plugin) ExecuteDispatch(in *model.Item, cfg plugin.Config) error {
	if in == nil {
		return nil
	}
	url := sharedutil.StringWithEnv(cfg, "webhook_url", p.webhookURL)
	if url == "" {
		return &model.ItemError{Code: "feishu_config", Message: "webhook_url is required"}
	}
	text := formatMessage(in)
	body := map[string]interface{}{
		"msg_type": "text",
		"content":  map[string]string{"text": text},
	}
	raw, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu webhook: %d", resp.StatusCode)
	}
	return nil
}

func formatMessage(in *model.Item) string {
	s := in.Title
	if in.URL != "" {
		s += "\n" + in.URL
	}
	if in.Summary != "" {
		s += "\n\n" + in.Summary
	}
	if len(in.KeyPoints) > 0 {
		s += "\n\n关键点:"
		for _, k := range in.KeyPoints {
			s += "\n• " + k
		}
	}
	return s
}
