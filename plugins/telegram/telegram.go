package telegram

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

// Plugin implements Telegram dispatch.
type Plugin struct {
	botToken   string
	chatID     string
	apiBaseURL string
	httpClient *http.Client
}

// New returns a new telegram plugin instance.
func New() plugin.Plugin {
	return &Plugin{
		apiBaseURL: "https://api.telegram.org",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *Plugin) Name() string      { return "telegram" }
func (p *Plugin) Version() string   { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeDispatch }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.botToken = sharedutil.StringWithEnv(cfg, "bot_token", os.Getenv("TELEGRAM_BOT_TOKEN"))
	p.chatID = sharedutil.StringWithEnv(cfg, "chat_id", os.Getenv("TELEGRAM_CHAT_ID"))
	if baseURL := strings.TrimSpace(plugin.GetString(cfg, "api_base_url")); baseURL != "" {
		p.apiBaseURL = strings.TrimRight(baseURL, "/")
	}
	return nil
}

// TestConfig validates Telegram credentials without sending messages.
func (p *Plugin) TestConfig(ctx context.Context, cfg plugin.Config) error {
	token, chatID, baseURL := p.resolveConfig(cfg)
	if err := sharedutil.RequireFields(map[string]string{"bot_token": token, "chat_id": chatID}); err != nil {
		return err
	}
	if _, err := p.callBotAPI(ctx, token, baseURL, "getMe", nil); err != nil {
		return err
	}
	_, err := p.callBotAPI(ctx, token, baseURL, "getChat", map[string]string{"chat_id": chatID})
	return err
}

// ExecuteDispatch sends the item to Telegram.
func (p *Plugin) ExecuteDispatch(in *model.Item, cfg plugin.Config) error {
	if in == nil {
		return nil
	}
	token, chatID, baseURL := p.resolveConfig(cfg)
	if token == "" || chatID == "" {
		return &model.ItemError{Code: "telegram_config", Message: "bot_token and chat_id are required"}
	}
	text := formatMessage(in)
	return sendTelegram(p.httpClient, token, chatID, baseURL, text)
}

func (p *Plugin) resolveConfig(cfg plugin.Config) (token, chatID, baseURL string) {
	token = sharedutil.StringWithEnv(cfg, "bot_token", p.botToken)
	chatID = sharedutil.StringWithEnv(cfg, "chat_id", p.chatID)
	baseURL = p.apiBaseURL
	if v := strings.TrimSpace(plugin.GetString(cfg, "api_base_url")); v != "" {
		baseURL = strings.TrimRight(v, "/")
	}
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	return token, chatID, baseURL
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
		s += "\n\nKey points:"
		for _, k := range in.KeyPoints {
			s += "\n• " + k
		}
	}
	return s
}

func sendTelegram(client *http.Client, token, chatID, baseURL, text string) error {
	payload := map[string]string{
		"chat_id": chatID,
		"text":    text,
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/bot%s/sendMessage", strings.TrimRight(baseURL, "/"), token)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram api: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func (p *Plugin) callBotAPI(ctx context.Context, token, baseURL, method string, body map[string]string) (map[string]any, error) {
	var rawBody []byte
	if body != nil {
		rawBody, _ = json.Marshal(body)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/bot%s/%s", strings.TrimRight(baseURL, "/"), token, method), bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram %s: HTTP %d: %s", method, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed struct {
		OK          bool           `json:"ok"`
		Description string         `json:"description"`
		Result      map[string]any `json:"result"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		if parsed.Description == "" {
			parsed.Description = "telegram api returned ok=false"
		}
		return nil, fmt.Errorf("telegram %s: %s", method, parsed.Description)
	}
	return parsed.Result, nil
}
