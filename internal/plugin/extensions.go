package plugin

import (
	"context"
	"encoding/json"
	"fmt"
)

// ProviderExtension provides AI provider functionality through plugins
type ProviderExtension struct {
	manager *Manager
	plugin  *Plugin
}

// NewProviderExtension creates a provider extension from a plugin
func NewProviderExtension(manager *Manager, pluginName string) (*ProviderExtension, error) {
	plugin, ok := manager.Get(pluginName)
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	if plugin.Manifest.Type != PluginTypeProvider {
		return nil, fmt.Errorf("plugin %s is not a provider (type: %s)", pluginName, plugin.Manifest.Type)
	}

	return &ProviderExtension{
		manager: manager,
		plugin:  plugin,
	}, nil
}

// ProviderGenerateRequest is sent to provider plugins
type ProviderGenerateRequest struct {
	Action  string                 `json:"action"`
	Prompt  string                 `json:"prompt"`
	Model   string                 `json:"model,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// ProviderGenerateResponse is returned from provider plugins
type ProviderGenerateResponse struct {
	Content string         `json:"content"`
	Model   string         `json:"model"`
	Usage   *ProviderUsage `json:"usage,omitempty"`
}

// ProviderUsage contains token usage information
type ProviderUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Generate sends a generation request to the provider plugin
func (p *ProviderExtension) Generate(ctx context.Context, prompt string, model string, options map[string]interface{}) (*ProviderGenerateResponse, error) {
	request := ProviderGenerateRequest{
		Action:  "generate",
		Prompt:  prompt,
		Model:   model,
		Options: options,
		Config:  p.plugin.Config,
	}

	resp, err := p.manager.executePlugin(ctx, p.plugin, request)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("provider error: %s", resp.Error)
	}

	// Extract response
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}

	var genResp ProviderGenerateResponse
	if err := json.Unmarshal(resultData, &genResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &genResp, nil
}

// GetModels returns available models from the provider plugin
func (p *ProviderExtension) GetModels(ctx context.Context) ([]string, error) {
	request := PluginRequest{
		Action: "list_models",
		Config: p.plugin.Config,
	}

	resp, err := p.manager.executePlugin(ctx, p.plugin, request)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("provider error: %s", resp.Error)
	}

	// Extract models list
	models, ok := resp.Result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid models response")
	}

	result := make([]string, len(models))
	for i, m := range models {
		if s, ok := m.(string); ok {
			result[i] = s
		}
	}

	return result, nil
}

// ValidatorExtension provides validation functionality through plugins
type ValidatorExtension struct {
	manager *Manager
	plugin  *Plugin
}

// NewValidatorExtension creates a validator extension from a plugin
func NewValidatorExtension(manager *Manager, pluginName string) (*ValidatorExtension, error) {
	plugin, ok := manager.Get(pluginName)
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	if plugin.Manifest.Type != PluginTypeValidator {
		return nil, fmt.Errorf("plugin %s is not a validator (type: %s)", pluginName, plugin.Manifest.Type)
	}

	return &ValidatorExtension{
		manager: manager,
		plugin:  plugin,
	}, nil
}

// Validate sends content to the validator plugin for validation
func (v *ValidatorExtension) Validate(ctx context.Context, content string, rules map[string]interface{}) (*ValidatorResponse, error) {
	request := ValidatorRequest{
		Action:  "validate",
		Content: content,
		Rules:   rules,
		Config:  v.plugin.Config,
	}

	resp, err := v.manager.executePlugin(ctx, v.plugin, request)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("validator error: %s", resp.Error)
	}

	// Extract response
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}

	var valResp ValidatorResponse
	if err := json.Unmarshal(resultData, &valResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &valResp, nil
}

// NotifierExtension provides notification functionality through plugins
type NotifierExtension struct {
	manager *Manager
	plugin  *Plugin
}

// NewNotifierExtension creates a notifier extension from a plugin
func NewNotifierExtension(manager *Manager, pluginName string) (*NotifierExtension, error) {
	plugin, ok := manager.Get(pluginName)
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	if plugin.Manifest.Type != PluginTypeNotifier {
		return nil, fmt.Errorf("plugin %s is not a notifier (type: %s)", pluginName, plugin.Manifest.Type)
	}

	return &NotifierExtension{
		manager: manager,
		plugin:  plugin,
	}, nil
}

// Notify sends a notification through the plugin
func (n *NotifierExtension) Notify(ctx context.Context, event string, data map[string]interface{}) error {
	request := NotifierRequest{
		Action: "notify",
		Event:  event,
		Data:   data,
		Config: n.plugin.Config,
	}

	resp, err := n.manager.executePlugin(ctx, n.plugin, request)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("notifier error: %s", resp.Error)
	}

	return nil
}

// FormatterExtension provides output formatting through plugins
type FormatterExtension struct {
	manager *Manager
	plugin  *Plugin
}

// NewFormatterExtension creates a formatter extension from a plugin
func NewFormatterExtension(manager *Manager, pluginName string) (*FormatterExtension, error) {
	plugin, ok := manager.Get(pluginName)
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	if plugin.Manifest.Type != PluginTypeFormatter {
		return nil, fmt.Errorf("plugin %s is not a formatter (type: %s)", pluginName, plugin.Manifest.Type)
	}

	return &FormatterExtension{
		manager: manager,
		plugin:  plugin,
	}, nil
}

// Format sends data to the formatter plugin for formatting
func (f *FormatterExtension) Format(ctx context.Context, data interface{}, format string) (string, error) {
	request := FormatterRequest{
		Action: "format",
		Data:   data,
		Format: format,
		Config: f.plugin.Config,
	}

	resp, err := f.manager.executePlugin(ctx, f.plugin, request)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf("formatter error: %s", resp.Error)
	}

	// Extract response
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}

	var fmtResp FormatterResponse
	if err := json.Unmarshal(resultData, &fmtResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	return fmtResp.Output, nil
}

// HookExtension provides event hook functionality through plugins
type HookExtension struct {
	manager *Manager
	plugin  *Plugin
}

// NewHookExtension creates a hook extension from a plugin
func NewHookExtension(manager *Manager, pluginName string) (*HookExtension, error) {
	plugin, ok := manager.Get(pluginName)
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	if plugin.Manifest.Type != PluginTypeHook {
		return nil, fmt.Errorf("plugin %s is not a hook (type: %s)", pluginName, plugin.Manifest.Type)
	}

	return &HookExtension{
		manager: manager,
		plugin:  plugin,
	}, nil
}

// HookRequest is sent to hook plugins
type HookRequest struct {
	Action string                 `json:"action"`
	Event  string                 `json:"event"`
	Phase  string                 `json:"phase"` // "before" or "after"
	Data   map[string]interface{} `json:"data"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// HookResponse is returned from hook plugins
type HookResponse struct {
	Continue bool                   `json:"continue"` // Whether to continue execution
	Modified map[string]interface{} `json:"modified,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// Execute runs the hook plugin for an event
func (h *HookExtension) Execute(ctx context.Context, event string, phase string, data map[string]interface{}) (*HookResponse, error) {
	request := HookRequest{
		Action: "hook",
		Event:  event,
		Phase:  phase,
		Data:   data,
		Config: h.plugin.Config,
	}

	resp, err := h.manager.executePlugin(ctx, h.plugin, request)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("hook error: %s", resp.Error)
	}

	// Extract response
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}

	var hookResp HookResponse
	if err := json.Unmarshal(resultData, &hookResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &hookResp, nil
}

// GetExtensionsByType returns all extensions of a specific type
func (m *Manager) GetExtensionsByType(pluginType PluginType) []*Plugin {
	return m.ListByType(pluginType)
}

// ExecuteHooks runs all hook plugins for an event
func (m *Manager) ExecuteHooks(ctx context.Context, event string, phase string, data map[string]interface{}) (map[string]interface{}, error) {
	hooks := m.ListByType(PluginTypeHook)
	result := data

	for _, plugin := range hooks {
		if plugin.State != PluginStateEnabled {
			continue
		}

		ext, err := NewHookExtension(m, plugin.Manifest.Name)
		if err != nil {
			continue
		}

		resp, err := ext.Execute(ctx, event, phase, result)
		if err != nil {
			return nil, fmt.Errorf("hook %s failed: %w", plugin.Manifest.Name, err)
		}

		if !resp.Continue {
			return nil, fmt.Errorf("hook %s stopped execution", plugin.Manifest.Name)
		}

		if resp.Modified != nil {
			result = resp.Modified
		}
	}

	return result, nil
}

// NotifyAll sends a notification to all notifier plugins
func (m *Manager) NotifyAll(ctx context.Context, event string, data map[string]interface{}) []error {
	notifiers := m.ListByType(PluginTypeNotifier)
	var errors []error

	for _, plugin := range notifiers {
		if plugin.State != PluginStateEnabled {
			continue
		}

		ext, err := NewNotifierExtension(m, plugin.Manifest.Name)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if err := ext.Notify(ctx, event, data); err != nil {
			errors = append(errors, fmt.Errorf("notifier %s: %w", plugin.Manifest.Name, err))
		}
	}

	return errors
}
