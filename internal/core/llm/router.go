package llm

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
)

// Profile names a model endpoint (heavy vs cheap, etc.).
type Profile struct {
	Name       string
	BaseURL    string // e.g. https://api.openai.com/v1
	Model      string
	APIKey     string
	Timeout    time.Duration
	MaxRetries int
}

// Router dispatches chat-completions-style requests to configured profiles.
type Router struct {
	profiles   map[string]Profile
	httpClient *http.Client
}

// NewRouterFromEnv builds profiles from environment:
//   - default: OPENAI_API_KEY, OPENAI_BASE_URL (optional), OPENAI_MODEL (optional gpt-4o)
//   - cheap: same key, OPENAI_MODEL_CHEAP (optional gpt-4o-mini)
func NewRouterFromEnv() *Router {
	key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	base := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = "gpt-4o"
	}
	cheapModel := strings.TrimSpace(os.Getenv("OPENAI_MODEL_CHEAP"))
	if cheapModel == "" {
		cheapModel = "gpt-4o-mini"
	}
	r := &Router{
		profiles: make(map[string]Profile),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
	if key != "" {
		r.profiles["default"] = Profile{
			Name: "default", BaseURL: strings.TrimSuffix(base, "/"),
			Model: model, APIKey: key, Timeout: 120 * time.Second, MaxRetries: 2,
		}
		r.profiles["heavy"] = r.profiles["default"]
		r.profiles["cheap"] = Profile{
			Name: "cheap", BaseURL: strings.TrimSuffix(base, "/"),
			Model: cheapModel, APIKey: key, Timeout: 60 * time.Second, MaxRetries: 2,
		}
	}
	return r
}

// RegisterProfile adds or replaces a profile (for tests / programmatic config).
func (r *Router) RegisterProfile(p Profile) {
	if r.profiles == nil {
		r.profiles = make(map[string]Profile)
	}
	if p.Timeout == 0 {
		p.Timeout = 60 * time.Second
	}
	if p.MaxRetries == 0 {
		p.MaxRetries = 1
	}
	r.profiles[p.Name] = p
}

// ProfileNames returns known profile keys.
func (r *Router) ProfileNames() []string {
	var out []string
	for k := range r.profiles {
		out = append(out, k)
	}
	return out
}

// MessageToolCall is an assistant tool invocation (OpenAI chat format).
type MessageToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Message is a chat message for the completions API.
type Message struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	ToolCalls  []MessageToolCall `json:"tool_calls,omitempty"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Chat runs a chat completion using the named profile (falls back to "default").
func (r *Router) Chat(ctx context.Context, profileName string, messages []Message, temperature float64) (string, error) {
	if r == nil || len(r.profiles) == 0 {
		return "", fmt.Errorf("llm: no profiles configured (set OPENAI_API_KEY)")
	}
	p, ok := r.profiles[profileName]
	if !ok {
		p, ok = r.profiles["default"]
	}
	if !ok {
		return "", fmt.Errorf("llm: unknown profile %q", profileName)
	}
	body, err := json.Marshal(chatRequest{Model: p.Model, Messages: messages, Temperature: temperature})
	if err != nil {
		return "", err
	}
	url := p.BaseURL + "/chat/completions"
	var lastErr error
	retries := p.MaxRetries
	if retries < 1 {
		retries = 1
	}
	for attempt := 0; attempt < retries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		if p.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+p.APIKey)
		}
		client := r.httpClient
		if client == nil {
			client = http.DefaultClient
		}
		if p.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, p.Timeout)
			defer cancel()
			req = req.WithContext(ctx)
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		b, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("llm: HTTP %d: %s", resp.StatusCode, string(b))
			continue
		}
		var cr chatResponse
		if err := json.Unmarshal(b, &cr); err != nil {
			lastErr = err
			continue
		}
		if cr.Error != nil {
			lastErr = fmt.Errorf("llm api: %s", cr.Error.Message)
			continue
		}
		if len(cr.Choices) == 0 {
			return "", fmt.Errorf("llm: empty choices")
		}
		return cr.Choices[0].Message.Content, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("llm: request failed")
	}
	return "", lastErr
}

// ToolSpec is an OpenAI-style function tool definition.
type ToolSpec struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

// ToolCall is one assistant tool invocation from the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// ChatChoice is one completion choice (text and/or tool calls).
type ChatChoice struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
}

type chatRequestTools struct {
	Model       string      `json:"model"`
	Messages    []Message   `json:"messages"`
	Temperature float64     `json:"temperature,omitempty"`
	Tools       []ToolSpec  `json:"tools,omitempty"`
	ToolChoice  interface{} `json:"tool_choice,omitempty"`
}

type chatResponseTools struct {
	Choices []struct {
		FinishReason string  `json:"finish_reason"`
		Message      Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ChatWithTools runs chat/completions with function tools (OpenAI-compatible).
func (r *Router) ChatWithTools(ctx context.Context, profileName string, messages []Message, tools []ToolSpec, temperature float64) (*ChatChoice, error) {
	if r == nil || len(r.profiles) == 0 {
		return nil, fmt.Errorf("llm: no profiles configured (set OPENAI_API_KEY)")
	}
	p, ok := r.profiles[profileName]
	if !ok {
		p, ok = r.profiles["default"]
	}
	if !ok {
		return nil, fmt.Errorf("llm: unknown profile %q", profileName)
	}
	body, err := json.Marshal(chatRequestTools{
		Model: p.Model, Messages: messages, Temperature: temperature, Tools: tools, ToolChoice: "auto",
	})
	if err != nil {
		return nil, err
	}
	url := p.BaseURL + "/chat/completions"
	var lastErr error
	retries := p.MaxRetries
	if retries < 1 {
		retries = 1
	}
	for attempt := 0; attempt < retries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if p.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+p.APIKey)
		}
		client := r.httpClient
		if client == nil {
			client = http.DefaultClient
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		b, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("llm: HTTP %d: %s", resp.StatusCode, string(b))
			continue
		}
		var cr chatResponseTools
		if err := json.Unmarshal(b, &cr); err != nil {
			lastErr = err
			continue
		}
		if cr.Error != nil {
			lastErr = fmt.Errorf("llm api: %s", cr.Error.Message)
			continue
		}
		if len(cr.Choices) == 0 {
			return nil, fmt.Errorf("llm: empty choices")
		}
		ch := cr.Choices[0]
		out := &ChatChoice{
			Content:      ch.Message.Content,
			FinishReason: ch.FinishReason,
		}
		for _, tc := range ch.Message.ToolCalls {
			if tc.Type != "" && tc.Type != "function" {
				continue
			}
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID: tc.ID, Name: tc.Function.Name, Arguments: tc.Function.Arguments,
			})
		}
		return out, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("llm: request failed")
	}
	return nil, lastErr
}
