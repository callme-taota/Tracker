package pluginruntime

import (
	"context"
	"encoding/json"
	"fmt"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// remoteShim adapts a subprocess plugin to in-process plugin.Plugin + capabilities.
type remoteShim struct {
	id      string
	version string
	kind    plugin.Type
	client  *Client
}

func newRemoteShim(id, version string, kind plugin.Type, c *Client) *remoteShim {
	return &remoteShim{id: id, version: version, kind: kind, client: c}
}

func (s *remoteShim) Name() string { return s.id }

func (s *remoteShim) Version() string { return s.version }

func (s *remoteShim) Type() plugin.Type { return s.kind }

func (s *remoteShim) Init(_ plugin.Config) error { return nil }

func (s *remoteShim) ExecuteSource(cfg plugin.Config) ([]*model.Item, error) {
	raw, err := s.client.ExecuteSource(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	var items []*model.Item
	if len(raw) == 0 {
		return []*model.Item{}, nil
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("pluginruntime: decode items: %w", err)
	}
	if items == nil {
		items = []*model.Item{}
	}
	return items, nil
}

func (s *remoteShim) Execute(in *model.Item, cfg plugin.Config) (*model.Item, error) {
	if in == nil {
		return nil, fmt.Errorf("pluginruntime: nil item")
	}
	itemJSON, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	raw, err := s.client.ExecuteProcess(context.Background(), itemJSON, cfg)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var out model.Item
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("pluginruntime: decode item: %w", err)
	}
	return &out, nil
}

func (s *remoteShim) ExecuteDispatch(in *model.Item, cfg plugin.Config) error {
	if in == nil {
		return fmt.Errorf("pluginruntime: nil item")
	}
	itemJSON, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return s.client.ExecuteDispatch(context.Background(), itemJSON, cfg)
}

func (s *remoteShim) TestConfig(ctx context.Context, cfg plugin.Config) error {
	return s.client.TestConfig(ctx, cfg)
}
