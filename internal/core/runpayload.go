package core

import "context"

type runPayloadKey struct{}

// TrackerRuntimeKey is the reserved config key injected into every node for experiment / rollout metadata.
const TrackerRuntimeKey = "__tracker_runtime"

// WithRunPayload attaches a shallow map merged into each source node's config during GraphRunner.RunWithContext.
func WithRunPayload(ctx context.Context, m map[string]interface{}) context.Context {
	if m == nil || len(m) == 0 {
		return ctx
	}
	return context.WithValue(ctx, runPayloadKey{}, m)
}

// RunPayload reads the payload from context (may be nil).
func RunPayload(ctx context.Context) map[string]interface{} {
	v, _ := ctx.Value(runPayloadKey{}).(map[string]interface{})
	return v
}
