package pluginruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"Tracker/internal/plugin"
)

// Client talks to one plugin process over a single TCP connection (framed JSON).
type Client struct {
	conn net.Conn
	mu   sync.Mutex
}

// Dial connects to a plugin listener address (host side).
func Dial(addr string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

// Close releases the connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) call(_ context.Context, op string, reqPayload interface{}, respPayload interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var raw json.RawMessage
	if reqPayload != nil {
		var err error
		raw, err = json.Marshal(reqPayload)
		if err != nil {
			return err
		}
	}
	if err := writeFrame(c.conn, wireReq{Op: op, Payload: raw}); err != nil {
		return err
	}
	body, err := readFrame(c.conn)
	if err != nil {
		return err
	}
	var wr wireResp
	if err := json.Unmarshal(body, &wr); err != nil {
		return fmt.Errorf("pluginruntime: bad response frame: %w", err)
	}
	if !wr.OK {
		code := wr.ErrCode
		if code == "" {
			code = "plugin_error"
		}
		msg := wr.ErrMsg
		if msg == "" {
			msg = "plugin returned error"
		}
		return &pluginOpError{Code: code, Message: msg}
	}
	if respPayload != nil && len(wr.Payload) > 0 {
		if err := json.Unmarshal(wr.Payload, respPayload); err != nil {
			return fmt.Errorf("pluginruntime: decode payload: %w", err)
		}
	}
	return nil
}

type pluginOpError struct {
	Code    string
	Message string
}

func (e *pluginOpError) Error() string {
	return e.Code + ": " + e.Message
}

type handshakePayload struct {
	PluginID string          `json:"plugin_id"`
	Version  string          `json:"version"`
	Manifest json.RawMessage `json:"manifest"`
}

// Handshake returns plugin identity and manifest JSON.
func (c *Client) Handshake(ctx context.Context) (handshakePayload, error) {
	var out handshakePayload
	err := c.call(ctx, OpHandshake, nil, &out)
	return out, err
}

type healthPayload struct {
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

// Health probes liveness.
func (c *Client) Health(ctx context.Context) (healthPayload, error) {
	var out healthPayload
	err := c.call(ctx, OpHealth, nil, &out)
	return out, err
}

// Ready probes readiness.
func (c *Client) Ready(ctx context.Context) (healthPayload, error) {
	var out healthPayload
	err := c.call(ctx, OpReady, nil, &out)
	return out, err
}

// Init sends global plugin config (once per process).
func (c *Client) Init(ctx context.Context, global plugin.Config) error {
	return c.call(ctx, OpInit, map[string]any{"config": global}, map[string]any{})
}

type execSourceOut struct {
	Items json.RawMessage `json:"items"`
}

// ExecuteSource runs a source stage.
func (c *Client) ExecuteSource(ctx context.Context, cfg plugin.Config) (json.RawMessage, error) {
	var out execSourceOut
	err := c.call(ctx, OpExecuteSource, map[string]any{"config": cfg}, &out)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

type execProcessOut struct {
	Item json.RawMessage `json:"item"`
}

// ExecuteProcess runs process/summary/interest.
func (c *Client) ExecuteProcess(ctx context.Context, itemJSON json.RawMessage, cfg plugin.Config) (json.RawMessage, error) {
	var out execProcessOut
	err := c.call(ctx, OpExecuteProcess, map[string]any{"item": json.RawMessage(itemJSON), "config": cfg}, &out)
	if err != nil {
		return nil, err
	}
	return out.Item, nil
}

// ExecuteDispatch runs dispatch.
func (c *Client) ExecuteDispatch(ctx context.Context, itemJSON json.RawMessage, cfg plugin.Config) error {
	return c.call(ctx, OpExecuteDispatch, map[string]any{"item": json.RawMessage(itemJSON), "config": cfg}, map[string]any{})
}

// TestConfig runs connectivity test.
func (c *Client) TestConfig(ctx context.Context, cfg plugin.Config) error {
	return c.call(ctx, OpTestConfig, map[string]any{"config": cfg}, nil)
}
