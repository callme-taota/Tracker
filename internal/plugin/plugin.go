package plugin

// Config is a generic configuration map for plugin Init.
type Config map[string]interface{}

// Plugin is the interface all plugins must implement.
// Capabilities are exposed via type assertion in the runner (SourceCapability, ProcessCapability, DispatchCapability).
type Plugin interface {
	Name() string
	Version() string
	Type() Type
	Init(cfg Config) error
}
