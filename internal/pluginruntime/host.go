package pluginruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"Tracker/internal/plugin"
	"Tracker/internal/pluginhub"
)

// PluginManagerSubset is satisfied by *core.PluginManager (avoids import cycle).
type PluginManagerSubset interface {
	Register(p plugin.Plugin)
	Unregister(name string)
}

// Host manages external subprocess plugins (hot reload / unload).
type Host struct {
	pm      PluginManagerSubset
	hub     *pluginhub.Hub
	builtin map[string]bool
	path    string

	mu       sync.Mutex
	entries  map[string]*pluginEntry
	extra    []PluginSpec
	lastInit plugin.Config
}

type pluginEntry struct {
	spec   PluginSpec
	cmd    *exec.Cmd
	client *Client
}

type stdoutAddr struct {
	ListenAddr string `json:"listen_addr"`
}

// NewHost builds a host. configPath is YAML file path (may be empty).
// builtin must list built-in plugin ids (e.g. from registry.BuiltinPluginIDs()); remote ids must not collide.
// If builtin is nil, treated as empty (no collision guard — callers should pass the real set from registry).
func NewHost(pm PluginManagerSubset, hub *pluginhub.Hub, configPath string, builtin map[string]bool) *Host {
	if builtin == nil {
		builtin = map[string]bool{}
	}
	return &Host{
		pm:      pm,
		hub:     hub,
		builtin: builtin,
		path:    configPath,
		entries: make(map[string]*pluginEntry),
	}
}

// LastGlobalConfig returns the last InitRemote config (may be nil).
func (h *Host) LastGlobalConfig() plugin.Config {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastInit
}

// IsRemote reports whether id is loaded as an external plugin.
func (h *Host) IsRemote(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.entries[id]
	return ok
}

// RemoteIDs lists external plugin ids.
func (h *Host) RemoteIDs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []string
	for id := range h.entries {
		out = append(out, id)
	}
	return out
}

// Health calls the remote health op.
func (h *Host) Health(ctx context.Context, id string) (status string, details string, err error) {
	h.mu.Lock()
	e := h.entries[id]
	h.mu.Unlock()
	if e == nil {
		return "", "", fmt.Errorf("pluginruntime: unknown remote plugin %q", id)
	}
	hp, err := e.client.Health(ctx)
	if err != nil {
		return "", "", err
	}
	return hp.Status, hp.Details, nil
}

// Load starts all plugins from the config file. Idempotent only after Shutdown.
func (h *Host) Load(ctx context.Context) error {
	fc, err := LoadConfigFile(h.path)
	if err != nil {
		return err
	}
	for _, spec := range fc.Plugins {
		if err := h.startPlugin(ctx, spec); err != nil {
			log.Printf("pluginruntime: external plugin %q: %v", spec.ID, err)
		}
	}
	h.mu.Lock()
	extra := append([]PluginSpec(nil), h.extra...)
	h.mu.Unlock()
	for _, spec := range extra {
		if err := h.startPlugin(ctx, spec); err != nil {
			log.Printf("pluginruntime: workspace plugin %q: %v", spec.ID, err)
		}
	}
	return nil
}

func (h *Host) startPlugin(ctx context.Context, spec PluginSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("missing id")
	}
	if len(spec.Command) == 0 {
		return fmt.Errorf("missing command")
	}
	if h.builtin[spec.ID] {
		return fmt.Errorf("id %q conflicts with built-in plugin", spec.ID)
	}
	h.mu.Lock()
	if _, exists := h.entries[spec.ID]; exists {
		h.mu.Unlock()
		return fmt.Errorf("plugin %q already loaded", spec.ID)
	}
	h.mu.Unlock()

	_ = ctx
	cmd := exec.Command(spec.Command[0], spec.Command[1:]...)
	if spec.Workdir != "" {
		cmd.Dir = spec.Workdir
	}
	cmd.Env = append(os.Environ(), "TRACKER_PLUGIN_PROTOCOL=1")
	for k, v := range spec.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	line, err := bufio.NewReader(stdout).ReadBytes('\n')
	if err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("read listen_addr: %w", err)
	}
	var sa stdoutAddr
	if err := json.Unmarshal(line, &sa); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("parse listen_addr line: %w", err)
	}
	if sa.ListenAddr == "" {
		_ = cmd.Process.Kill()
		return fmt.Errorf("empty listen_addr")
	}

	cli, err := Dial(strings.TrimSpace(sa.ListenAddr))
	if err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("dial plugin: %w", err)
	}

	hs, err := cli.Handshake(ctx)
	if err != nil {
		_ = cli.Close()
		_ = cmd.Process.Kill()
		return fmt.Errorf("handshake: %w", err)
	}
	if hs.PluginID != spec.ID {
		_ = cli.Close()
		_ = cmd.Process.Kill()
		return fmt.Errorf("handshake plugin_id %q != config id %q", hs.PluginID, spec.ID)
	}
	var man plugin.Manifest
	if err := json.Unmarshal(hs.Manifest, &man); err != nil {
		_ = cli.Close()
		_ = cmd.Process.Kill()
		return fmt.Errorf("manifest json: %w", err)
	}
	if man.ID == "" {
		man.ID = hs.PluginID
	}
	if man.ID != spec.ID {
		_ = cli.Close()
		_ = cmd.Process.Kill()
		return fmt.Errorf("manifest id %q != config id %q", man.ID, spec.ID)
	}

	shim := newRemoteShim(hs.PluginID, hs.Version, man.Kind, cli)

	h.mu.Lock()
	h.entries[spec.ID] = &pluginEntry{spec: spec, cmd: cmd, client: cli}
	h.mu.Unlock()

	h.hub.RegisterManifest(man)
	h.pm.Register(shim)
	return nil
}

// InitRemote calls the remote Init RPC with global config (after built-ins InitAll).
func (h *Host) InitRemote(ctx context.Context, cfg plugin.Config) error {
	h.mu.Lock()
	h.lastInit = cfg
	entries := make([]*pluginEntry, 0, len(h.entries))
	for _, e := range h.entries {
		entries = append(entries, e)
	}
	h.mu.Unlock()

	for _, e := range entries {
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := e.client.Init(cctx, cfg)
		cancel()
		if err != nil {
			return fmt.Errorf("remote plugin %q: %w", e.spec.ID, err)
		}
	}
	return nil
}

// Shutdown stops all remote plugins and unregisters them.
func (h *Host) Shutdown(_ context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, e := range h.entries {
		if e.client != nil {
			_ = e.client.Close()
		}
		if e.cmd != nil && e.cmd.Process != nil {
			_ = e.cmd.Process.Kill()
			_, _ = e.cmd.Process.Wait()
		}
		h.pm.Unregister(id)
		h.hub.RemoveManifest(id)
	}
	h.entries = make(map[string]*pluginEntry)
}

// Reload stops and reloads external plugins from file.
func (h *Host) Reload(ctx context.Context) error {
	h.Shutdown(ctx)
	return h.Load(ctx)
}

// SetExtraSpecs replaces in-memory managed plugin specs and reloads all remote plugins.
func (h *Host) SetExtraSpecs(ctx context.Context, specs []PluginSpec) error {
	h.mu.Lock()
	h.extra = append([]PluginSpec(nil), specs...)
	h.mu.Unlock()
	return h.Reload(ctx)
}
