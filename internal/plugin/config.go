package plugin

import "fmt"

// GetString returns config value as string, or empty if missing/wrong type.
func GetString(cfg Config, key string) string {
	if cfg == nil {
		return ""
	}
	v, ok := cfg[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// GetStringSlice returns config value as []string (from []interface{} or comma-separated string).
func GetStringSlice(cfg Config, key string) []string {
	if cfg == nil {
		return nil
	}
	v, ok := cfg[key]
	if !ok {
		return nil
	}
	if s, ok := v.(string); ok {
		if s == "" {
			return nil
		}
		return []string{s}
	}
	if list, ok := v.([]interface{}); ok {
		var out []string
		for _, x := range list {
			out = append(out, fmt.Sprint(x))
		}
		return out
	}
	if list, ok := v.([]string); ok {
		return list
	}
	return nil
}
