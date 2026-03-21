package core

import (
	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

// AgentRunner runs a single plugin capability (source, process, or dispatch) with stage config.
type AgentRunner struct {
	pm *PluginManager
}

// NewAgentRunner creates an AgentRunner that uses the given plugin manager.
func NewAgentRunner(pm *PluginManager) *AgentRunner {
	return &AgentRunner{pm: pm}
}

// RunSource runs a source plugin and returns items. Plugin is identified by pluginID (name).
func (r *AgentRunner) RunSource(pluginID string, cfg plugin.Config) ([]*model.Item, error) {
	p := r.pm.GetByName(pluginID)
	if p == nil {
		return nil, &model.ItemError{Code: "plugin_not_found", Message: "plugin " + pluginID + " not found"}
	}
	if p.Type() != plugin.TypeSource {
		return nil, &model.ItemError{Code: "invalid_plugin_type", Message: "plugin " + pluginID + " is not a source"}
	}
	src, ok := p.(plugin.SourceCapability)
	if !ok {
		return nil, &model.ItemError{Code: "missing_capability", Message: "plugin " + pluginID + " does not implement SourceCapability"}
	}
	return src.ExecuteSource(cfg)
}

// RunProcess runs a processor/summary/interest plugin on one item.
func (r *AgentRunner) RunProcess(pluginID string, in *model.Item, cfg plugin.Config) (*model.Item, error) {
	p := r.pm.GetByName(pluginID)
	if p == nil {
		return nil, &model.ItemError{Code: "plugin_not_found", Message: "plugin " + pluginID + " not found"}
	}
	proc, ok := p.(plugin.ProcessCapability)
	if !ok {
		return nil, &model.ItemError{Code: "missing_capability", Message: "plugin " + pluginID + " does not implement ProcessCapability"}
	}
	return proc.Execute(in, cfg)
}

// RunDispatch runs a dispatch plugin (consumes the item).
func (r *AgentRunner) RunDispatch(pluginID string, in *model.Item, cfg plugin.Config) error {
	p := r.pm.GetByName(pluginID)
	if p == nil {
		return &model.ItemError{Code: "plugin_not_found", Message: "plugin " + pluginID + " not found"}
	}
	disp, ok := p.(plugin.DispatchCapability)
	if !ok {
		return &model.ItemError{Code: "missing_capability", Message: "plugin " + pluginID + " does not implement DispatchCapability"}
	}
	return disp.ExecuteDispatch(in, cfg)
}
