package core

import (
	"Tracker/internal/core/llm"
	"Tracker/internal/core/storage"
)

// CoreServices bundles shared capabilities injected into pluginhub.RuntimeContext.Core for plugins.
type CoreServices struct {
	LLM     *llm.Router
	Storage *storage.Router
}

// NewCoreServicesFromEnv builds LLM + multi-backend storage from environment variables.
func NewCoreServicesFromEnv() *CoreServices {
	return &CoreServices{
		LLM:     llm.NewRouterFromEnv(),
		Storage: storage.NewRouterFromEnv(),
	}
}

// Close releases storage pools.
func (c *CoreServices) Close() error {
	if c == nil || c.Storage == nil {
		return nil
	}
	return c.Storage.Close()
}
