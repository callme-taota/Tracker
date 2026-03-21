package plugin

import "Tracker/internal/model"

// SourceCapability is implemented by source plugins. It fetches items from an external source.
// cfg is the stage-specific config from the pipeline (e.g. feed URLs).
type SourceCapability interface {
	ExecuteSource(cfg Config) ([]*model.Item, error)
}

// ProcessCapability is implemented by processor, summary, and interest plugins.
// It receives one item and stage config, returns a (possibly modified) item.
type ProcessCapability interface {
	Execute(in *model.Item, cfg Config) (*model.Item, error)
}

// DispatchCapability is implemented by dispatch plugins. It consumes the item (e.g. send to Telegram).
// cfg is the stage-specific config (e.g. bot_token, chat_id).
type DispatchCapability interface {
	ExecuteDispatch(in *model.Item, cfg Config) error
}
