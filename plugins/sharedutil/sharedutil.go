package sharedutil

import (
	"fmt"
	"strconv"
	"strings"

	"Tracker/internal/plugin"
)

// StringWithEnv returns a config string, falling back to an env-backed default.
func StringWithEnv(cfg plugin.Config, key, envValue string) string {
	if v := strings.TrimSpace(plugin.GetString(cfg, key)); v != "" {
		return v
	}
	return strings.TrimSpace(envValue)
}

// Bool returns a best-effort boolean value from config.
func Bool(cfg plugin.Config, key string, def bool) bool {
	if cfg == nil {
		return def
	}
	v, ok := cfg[key]
	if !ok {
		return def
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	case float64:
		return x != 0
	case int:
		return x != 0
	case int64:
		return x != 0
	}
	return def
}

// Int returns a best-effort integer value from config.
func Int(cfg plugin.Config, key string, def int) int {
	if cfg == nil {
		return def
	}
	v, ok := cfg[key]
	if !ok {
		return def
	}
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(x))
		if err == nil {
			return n
		}
	}
	return def
}

// Float returns a best-effort float value from config.
func Float(cfg plugin.Config, key string, def float64) float64 {
	if cfg == nil {
		return def
	}
	v, ok := cfg[key]
	if !ok {
		return def
	}
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err == nil {
			return n
		}
	}
	return def
}

// RequireFields ensures all named config values are present and non-empty.
func RequireFields(values map[string]string) error {
	for key, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is empty", key)
		}
	}
	return nil
}
