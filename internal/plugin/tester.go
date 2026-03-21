package plugin

import "context"

// ConfigTester is optionally implemented by plugins to validate connectivity / config from the API or UI.
type ConfigTester interface {
	TestConfig(ctx context.Context, cfg Config) error
}
