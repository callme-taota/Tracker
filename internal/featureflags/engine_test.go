package featureflags

import "testing"

func TestEngineEvaluate_RequestOverrideWins(t *testing.T) {
	engine := NewEngine(ReleaseConfig{Channel: "stable", Ring: "global"}, []FlagConfig{
		{
			Key:            "runtime.executor_v2",
			DefaultVariant: "legacy",
			Variants: []VariantConfig{
				{Name: "legacy", Weight: 100},
				{Name: "unified", Weight: 0},
			},
		},
	})
	res := engine.Evaluate(EvalContext{
		SubjectID: "user-1",
		Overrides: map[string]string{"runtime.executor_v2": "unified"},
	}, "runtime.executor_v2")
	if res.Variant != "unified" {
		t.Fatalf("want unified got %#v", res)
	}
	if res.Reason != "request_override" {
		t.Fatalf("want request_override got %#v", res)
	}
}

func TestEngineSnapshot_ExposeOnlyWebFlags(t *testing.T) {
	engine := NewEngine(ReleaseConfig{Channel: "stable", Ring: "global"}, DefaultDefinitions())
	snapshot := engine.Snapshot(EvalContext{SubjectID: "user-1"}, true)
	if _, ok := snapshot.Flags["web.runtime_experiments"]; !ok {
		t.Fatalf("expected web.runtime_experiments in %#v", snapshot.Flags)
	}
	if _, ok := snapshot.Flags["runtime.executor_v2"]; ok {
		t.Fatalf("did not expect runtime-only flag in %#v", snapshot.Flags)
	}
}
