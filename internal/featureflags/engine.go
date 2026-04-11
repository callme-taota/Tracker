package featureflags

import (
	"fmt"
	"hash/fnv"
	"os"
	"sort"
	"strings"
	"time"
)

// ReleaseConfig identifies the currently deployed release channel and ring.
type ReleaseConfig struct {
	Channel  string `yaml:"channel" json:"channel"`
	Ring     string `yaml:"ring" json:"ring"`
	Version  string `yaml:"version" json:"version"`
	Instance string `yaml:"instance" json:"instance"`
}

// VariantConfig declares one available variant for a feature flag.
type VariantConfig struct {
	Name     string   `yaml:"name" json:"name"`
	Weight   int      `yaml:"weight" json:"weight"`
	Subjects []string `yaml:"subjects,omitempty" json:"subjects,omitempty"`
	Channels []string `yaml:"channels,omitempty" json:"channels,omitempty"`
	Rings    []string `yaml:"rings,omitempty" json:"rings,omitempty"`
}

// FlagConfig is a runtime-configurable feature flag / experiment definition.
type FlagConfig struct {
	Key            string          `yaml:"key" json:"key"`
	Description    string          `yaml:"description,omitempty" json:"description,omitempty"`
	DefaultVariant string          `yaml:"default_variant" json:"default_variant"`
	ExposeToWeb    bool            `yaml:"expose_to_web,omitempty" json:"expose_to_web,omitempty"`
	Deprecated     bool            `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Variants       []VariantConfig `yaml:"variants,omitempty" json:"variants,omitempty"`
}

// EvalContext is the stable input used for experiment evaluation.
type EvalContext struct {
	SubjectID   string
	Source      string
	RequestPath string
	PipelineID  int64
	JobID       int64
	Overrides   map[string]string
	Attributes  map[string]string
}

// ResolvedFlag is one evaluated flag result.
type ResolvedFlag struct {
	Key         string `json:"key"`
	Variant     string `json:"variant"`
	Description string `json:"description,omitempty"`
	Reason      string `json:"reason,omitempty"`
	ExposeToWeb bool   `json:"expose_to_web,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
}

// Snapshot is the evaluated experiment view for one request/run.
type Snapshot struct {
	Release     ReleaseConfig           `json:"release"`
	SubjectID   string                  `json:"subject_id"`
	GeneratedAt string                  `json:"generated_at"`
	Flags       map[string]ResolvedFlag `json:"flags"`
}

// Variant returns the resolved variant for key.
func (s Snapshot) Variant(key string) string {
	if s.Flags == nil {
		return ""
	}
	return s.Flags[key].Variant
}

// Engine evaluates feature flags against release metadata and stable subjects.
type Engine struct {
	release      ReleaseConfig
	definitions  map[string]FlagConfig
	forceVariant map[string]string
}

// DefaultDefinitions returns the built-in default flags used to guard new logic.
func DefaultDefinitions() []FlagConfig {
	return []FlagConfig{
		{
			Key:            "runtime.executor_v2",
			Description:    "Unified runtime execution flow across API, worker, scheduler, and CLI.",
			DefaultVariant: "legacy",
			Variants: []VariantConfig{
				{Name: "legacy", Weight: 100},
				{Name: "unified", Weight: 0},
			},
		},
		{
			Key:            "web.runtime_experiments",
			Description:    "Expose runtime experiment snapshot in the web shell.",
			DefaultVariant: "off",
			ExposeToWeb:    true,
			Variants: []VariantConfig{
				{Name: "off", Weight: 100},
				{Name: "on", Weight: 0},
			},
		},
		{
			Key:            "runtime.legacy_formats_compat",
			Description:    "Temporary compatibility guard for legacy format matching helpers.",
			DefaultVariant: "on",
			Variants: []VariantConfig{
				{Name: "on", Weight: 100},
				{Name: "off", Weight: 0},
			},
			Deprecated: true,
		},
	}
}

// NewEngine builds a feature flag engine from release and flag definitions.
func NewEngine(release ReleaseConfig, defs []FlagConfig) *Engine {
	if strings.TrimSpace(release.Channel) == "" {
		release.Channel = "stable"
	}
	if strings.TrimSpace(release.Ring) == "" {
		release.Ring = "global"
	}
	if strings.TrimSpace(release.Instance) == "" {
		release.Instance = strings.TrimSpace(os.Getenv("TRACKER_INSTANCE_ID"))
	}
	m := make(map[string]FlagConfig, len(defs))
	for _, def := range defs {
		if strings.TrimSpace(def.Key) == "" {
			continue
		}
		def.Key = strings.TrimSpace(def.Key)
		if strings.TrimSpace(def.DefaultVariant) == "" {
			def.DefaultVariant = "control"
		}
		if len(def.Variants) == 0 {
			def.Variants = []VariantConfig{{Name: def.DefaultVariant, Weight: 100}}
		}
		m[def.Key] = def
	}
	return &Engine{
		release:      release,
		definitions:  m,
		forceVariant: parseOverrideString(os.Getenv("TRACKER_FLAG_OVERRIDES")),
	}
}

// Definitions returns the known flag definitions in sorted order.
func (e *Engine) Definitions() []FlagConfig {
	out := make([]FlagConfig, 0, len(e.definitions))
	for _, def := range e.definitions {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Snapshot evaluates all flags for one runtime context.
func (e *Engine) Snapshot(ctx EvalContext, exposeOnly bool) Snapshot {
	flags := make(map[string]ResolvedFlag)
	for _, def := range e.Definitions() {
		if exposeOnly && !def.ExposeToWeb {
			continue
		}
		res := e.evaluateOne(ctx, def)
		if exposeOnly && !res.ExposeToWeb {
			continue
		}
		flags[def.Key] = res
	}
	return Snapshot{
		Release:     e.release,
		SubjectID:   normalizedSubject(ctx, e.release),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Flags:       flags,
	}
}

// Evaluate returns the resolved flag for one key.
func (e *Engine) Evaluate(ctx EvalContext, key string) ResolvedFlag {
	def, ok := e.definitions[key]
	if !ok {
		return ResolvedFlag{Key: key, Variant: "", Reason: "undefined"}
	}
	return e.evaluateOne(ctx, def)
}

func (e *Engine) evaluateOne(ctx EvalContext, def FlagConfig) ResolvedFlag {
	if variant, ok := ctx.Overrides[def.Key]; ok && strings.TrimSpace(variant) != "" {
		return resolved(def, strings.TrimSpace(variant), "request_override")
	}
	if variant, ok := e.forceVariant[def.Key]; ok && strings.TrimSpace(variant) != "" {
		return resolved(def, strings.TrimSpace(variant), "env_override")
	}
	for _, variant := range def.Variants {
		if containsFold(variant.Subjects, ctx.SubjectID) {
			return resolved(def, variant.Name, "subject_override")
		}
	}
	matches := make([]VariantConfig, 0, len(def.Variants))
	total := 0
	for _, variant := range def.Variants {
		if len(variant.Channels) > 0 && !containsFold(variant.Channels, e.release.Channel) {
			continue
		}
		if len(variant.Rings) > 0 && !containsFold(variant.Rings, e.release.Ring) {
			continue
		}
		if variant.Weight <= 0 {
			continue
		}
		matches = append(matches, variant)
		total += variant.Weight
	}
	if total == 0 {
		return resolved(def, def.DefaultVariant, "default")
	}
	subject := normalizedSubject(ctx, e.release)
	bucket := stableBucket(def.Key, subject) % total
	offset := 0
	for _, variant := range matches {
		offset += variant.Weight
		if bucket < offset {
			return resolved(def, variant.Name, fmt.Sprintf("weighted:%s", subject))
		}
	}
	return resolved(def, def.DefaultVariant, "default")
}

func resolved(def FlagConfig, variant, reason string) ResolvedFlag {
	return ResolvedFlag{
		Key:         def.Key,
		Variant:     variant,
		Description: def.Description,
		Reason:      reason,
		ExposeToWeb: def.ExposeToWeb,
		Deprecated:  def.Deprecated,
	}
}

func normalizedSubject(ctx EvalContext, release ReleaseConfig) string {
	for _, candidate := range []string{
		strings.TrimSpace(ctx.SubjectID),
		strings.TrimSpace(ctx.Attributes["user_id"]),
		strings.TrimSpace(ctx.Attributes["tenant_id"]),
		strings.TrimSpace(ctx.Attributes["request_id"]),
		strings.TrimSpace(ctx.Attributes["remote_addr"]),
	} {
		if candidate != "" {
			return candidate
		}
	}
	if ctx.PipelineID > 0 {
		return fmt.Sprintf("pipeline:%d", ctx.PipelineID)
	}
	if ctx.JobID > 0 {
		return fmt.Sprintf("job:%d", ctx.JobID)
	}
	if strings.TrimSpace(release.Instance) != "" {
		return "instance:" + strings.TrimSpace(release.Instance)
	}
	return "anonymous"
}

func stableBucket(flagKey, subject string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(flagKey))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(subject))
	return int(h.Sum32() % 10000)
}

func parseOverrideString(raw string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func containsFold(list []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), want) {
			return true
		}
	}
	return false
}
