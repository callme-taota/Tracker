package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// Plugin implements Telegram dispatch.
type Plugin struct {
	botToken string
	chatID   string
}

// New returns a new telegram plugin instance.
func New() plugin.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return "telegram" }
func (p *Plugin) Version() string { return "1.0" }
func (p *Plugin) Type() plugin.Type { return plugin.TypeDispatch }

func (p *Plugin) Init(cfg plugin.Config) error {
	p.botToken = plugin.GetString(cfg, "bot_token")
	if p.botToken == "" {
		p.botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}
	p.chatID = plugin.GetString(cfg, "chat_id")
	if p.chatID == "" {
		p.chatID = os.Getenv("TELEGRAM_CHAT_ID")
	}
	return nil
}

// ExecuteDispatch sends the item to Telegram.
func (p *Plugin) ExecuteDispatch(in *model.Item, cfg plugin.Config) error {
	if in == nil {
		return nil
	}
	token := plugin.GetString(cfg, "bot_token")
	if token == "" {
		token = p.botToken
	}
	if token == "" {
		token = os.Getenv("TELEGRAM_BOT_TOKEN")
	}
	chatID := plugin.GetString(cfg, "chat_id")
	if chatID == "" {
		chatID = p.chatID
	}
	if chatID == "" {
		chatID = os.Getenv("TELEGRAM_CHAT_ID")
	}
	if token == "" || chatID == "" {
		return nil
	}
	text := formatMessage(in)
	return sendTelegram(token, chatID, text)
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

func sendTelegram(token, chatID, text string) error {
	payload := map[string]string{
		"chat_id": chatID,
		"text":    text,
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api: %d", resp.StatusCode)
	}
	return nil
}
