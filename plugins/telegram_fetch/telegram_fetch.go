package telegram_fetch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// Plugin pulls recent updates via Bot API getUpdates (bot must be member of groups/channels to see messages).
type Plugin struct {
	client *http.Client
}

func New() plugin.Plugin {
	return &Plugin{client: &http.Client{Timeout: 45 * time.Second}}
}

func (p *Plugin) Name() string             { return "telegram_fetch" }
func (p *Plugin) Version() string          { return "1.0" }
func (p *Plugin) Type() plugin.Type        { return plugin.TypeSource }
func (p *Plugin) Init(cfg plugin.Config) error { return nil }

func (p *Plugin) ExecuteSource(cfg plugin.Config) ([]*model.Item, error) {
	token := plugin.GetString(cfg, "bot_token")
	if token == "" {
		token = os.Getenv("TELEGRAM_BOT_TOKEN")
	}
	if token == "" {
		return nil, &model.ItemError{Code: "telegram_token", Message: "bot_token required"}
	}
	filter := strings.ToLower(strings.TrimSpace(plugin.GetString(cfg, "text_contains")))
	limit := 50
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=0&limit=%d", token, limit)
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var raw struct {
		OK     bool `json:"ok"`
		Result []struct {
			Message struct {
				Text string `json:"text"`
				Chat struct {
					ID    int64  `json:"id"`
					Title string `json:"title"`
				} `json:"chat"`
				Date int64 `json:"date"`
			} `json:"message"`
			ChannelPost struct {
				Text string `json:"text"`
				Chat struct {
					ID    int64  `json:"id"`
					Title string `json:"title"`
				} `json:"chat"`
				Date int64 `json:"date"`
			} `json:"channel_post"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	var items []*model.Item
	for _, u := range raw.Result {
		text := u.Message.Text
		title := u.Message.Chat.Title
		ts := time.Unix(u.Message.Date, 0).UTC()
		if text == "" {
			text = u.ChannelPost.Text
			title = u.ChannelPost.Chat.Title
			ts = time.Unix(u.ChannelPost.Date, 0).UTC()
		}
		if text == "" {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(text), filter) {
			continue
		}
		items = append(items, &model.Item{
			Title: firstLine(title, text), Content: text, URL: "",
			Timestamp: ts, Source: "telegram", Metadata: map[string]string{"chat_title": title},
		})
	}
	return items, nil
}

func firstLine(title, text string) string {
	if strings.TrimSpace(title) != "" {
		return title
	}
	line := strings.TrimSpace(strings.Split(text, "\n")[0])
	if len(line) > 160 {
		return line[:160] + "…"
	}
	return line
}
