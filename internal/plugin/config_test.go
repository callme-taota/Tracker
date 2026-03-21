package plugin

import (
	"testing"
)

func TestGetString(t *testing.T) {
	cfg := Config{"a": "x", "b": 1}
	if v := GetString(cfg, "a"); v != "x" {
		t.Errorf("GetString(a) want x got %q", v)
	}
	if v := GetString(cfg, "b"); v != "" {
		t.Errorf("GetString(b) want empty got %q", v)
	}
	if v := GetString(nil, "a"); v != "" {
		t.Errorf("GetString(nil) want empty got %q", v)
	}
}

func TestGetStringSlice(t *testing.T) {
	cfg := Config{"list": []interface{}{"a", "b"}}
	v := GetStringSlice(cfg, "list")
	if len(v) != 2 || v[0] != "a" || v[1] != "b" {
		t.Errorf("GetStringSlice want [a b] got %v", v)
	}
	cfg["single"] = "only"
	v2 := GetStringSlice(cfg, "single")
	if len(v2) != 1 || v2[0] != "only" {
		t.Errorf("GetStringSlice(single) want [only] got %v", v2)
	}
}
