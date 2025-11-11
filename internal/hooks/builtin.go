package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

// ScriptHook executes a shell script
type ScriptHook struct {
	name       string
	eventTypes []EventType
	enabled    bool
	scriptPath string
	args       []string
	shell      string
}

// NewScriptHook creates a new script hook
func NewScriptHook(config *HookConfig) (Hook, error) {
	scriptPath, ok := config.Config["script"].(string)
	if !ok || scriptPath == "" {
		return nil, fmt.Errorf("script path required")
	}

	hook := &ScriptHook{
		name:       config.Name,
		eventTypes: config.Events,
		enabled:    config.Enabled,
		scriptPath: scriptPath,
		shell:      "/bin/bash",
	}

	// Optional args
	if argsInterface, ok := config.Config["args"]; ok {
		if argsList, ok := argsInterface.([]interface{}); ok {
			for _, arg := range argsList {
				if argStr, ok := arg.(string); ok {
					hook.args = append(hook.args, argStr)
				}
			}
		}
	}

	// Optional shell
	if shell, ok := config.Config["shell"].(string); ok && shell != "" {
		hook.shell = shell
	}

	return hook, nil
}

func (h *ScriptHook) Name() string            { return h.name }
func (h *ScriptHook) EventTypes() []EventType { return h.eventTypes }
func (h *ScriptHook) Enabled() bool           { return h.enabled }

func (h *ScriptHook) Execute(ctx context.Context, event *Event) error {
	// Prepare environment variables from event data
	env := []string{}
	env = append(env, fmt.Sprintf("HOOK_EVENT_TYPE=%s", event.Type))
	env = append(env, fmt.Sprintf("HOOK_WORKFLOW_ID=%s", event.WorkflowID))

	// Add event data as env vars
	for key, value := range event.Data {
		if str, ok := value.(string); ok {
			env = append(env, fmt.Sprintf("HOOK_%s=%s", strings.ToUpper(key), str))
		}
	}

	// Create command
	args := append([]string{h.scriptPath}, h.args...)
	cmd := exec.CommandContext(ctx, h.shell, args...)
	cmd.Env = env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script failed: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// WebhookHook sends HTTP POST requests
type WebhookHook struct {
	name       string
	eventTypes []EventType
	enabled    bool
	url        string
	headers    map[string]string
	client     *http.Client
}

// NewWebhookHook creates a new webhook hook
func NewWebhookHook(config *HookConfig) (Hook, error) {
	url, ok := config.Config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("webhook URL required")
	}

	hook := &WebhookHook{
		name:       config.Name,
		eventTypes: config.Events,
		enabled:    config.Enabled,
		url:        url,
		headers:    make(map[string]string),
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}

	// Optional headers
	if headersInterface, ok := config.Config["headers"]; ok {
		if headersMap, ok := headersInterface.(map[string]interface{}); ok {
			for key, value := range headersMap {
				if valStr, ok := value.(string); ok {
					hook.headers[key] = valStr
				}
			}
		}
	}

	return hook, nil
}

func (h *WebhookHook) Name() string            { return h.name }
func (h *WebhookHook) EventTypes() []EventType { return h.eventTypes }
func (h *WebhookHook) Enabled() bool           { return h.enabled }

func (h *WebhookHook) Execute(ctx context.Context, event *Event) error {
	// Marshal event to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", h.url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range h.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// SlackHook sends notifications to Slack
type SlackHook struct {
	name       string
	eventTypes []EventType
	enabled    bool
	webhookURL string
	channel    string
	username   string
	iconEmoji  string
	client     *http.Client
}

// NewSlackHook creates a new Slack hook
func NewSlackHook(config *HookConfig) (Hook, error) {
	webhookURL, ok := config.Config["webhookUrl"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("Slack webhook URL required")
	}

	hook := &SlackHook{
		name:       config.Name,
		eventTypes: config.Events,
		enabled:    config.Enabled,
		webhookURL: webhookURL,
		username:   "Specular",
		iconEmoji:  ":robot_face:",
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}

	// Optional channel
	if channel, ok := config.Config["channel"].(string); ok && channel != "" {
		hook.channel = channel
	}

	// Optional username
	if username, ok := config.Config["username"].(string); ok && username != "" {
		hook.username = username
	}

	// Optional icon
	if icon, ok := config.Config["iconEmoji"].(string); ok && icon != "" {
		hook.iconEmoji = icon
	}

	return hook, nil
}

func (h *SlackHook) Name() string            { return h.name }
func (h *SlackHook) EventTypes() []EventType { return h.eventTypes }
func (h *SlackHook) Enabled() bool           { return h.enabled }

func (h *SlackHook) Execute(ctx context.Context, event *Event) error {
	// Format message based on event type
	message := h.formatMessage(event)

	// Create Slack payload
	payload := map[string]interface{}{
		"text":       message,
		"username":   h.username,
		"icon_emoji": h.iconEmoji,
	}

	if h.channel != "" {
		payload["channel"] = h.channel
	}

	// Marshal to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	// Send request
	req, err := http.NewRequestWithContext(ctx, "POST", h.webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("Slack request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack returned status %d", resp.StatusCode)
	}

	return nil
}

func (h *SlackHook) formatMessage(event *Event) string {
	switch event.Type {
	case EventWorkflowStart:
		return fmt.Sprintf("🚀 Workflow started: %s", event.WorkflowID)

	case EventWorkflowComplete:
		duration := event.GetString("duration")
		cost := event.GetFloat("cost")
		return fmt.Sprintf("✅ Workflow completed: %s\nDuration: %s | Cost: $%.2f", event.WorkflowID, duration, cost)

	case EventWorkflowFailed:
		errorMsg := event.GetString("error")
		return fmt.Sprintf("❌ Workflow failed: %s\nError: %s", event.WorkflowID, errorMsg)

	case EventPlanCreated:
		steps := event.GetInt("steps")
		return fmt.Sprintf("📋 Plan created: %d steps", steps)

	case EventStepBefore:
		stepID := event.GetString("stepId")
		stepType := event.GetString("stepType")
		return fmt.Sprintf("▶️ Starting step: %s (%s)", stepID, stepType)

	case EventStepAfter:
		stepID := event.GetString("stepId")
		return fmt.Sprintf("✅ Completed step: %s", stepID)

	case EventStepFailed:
		stepID := event.GetString("stepId")
		errorMsg := event.GetString("error")
		return fmt.Sprintf("❌ Step failed: %s\nError: %s", stepID, errorMsg)

	case EventPolicyViolation:
		policy := event.GetString("policy")
		reason := event.GetString("reason")
		return fmt.Sprintf("🚫 Policy violation: %s\nReason: %s", policy, reason)

	case EventDriftDetected:
		return fmt.Sprintf("⚠️ Drift detected in workflow: %s", event.WorkflowID)

	default:
		return fmt.Sprintf("Event: %s for workflow %s", event.Type, event.WorkflowID)
	}
}

// RegisterBuiltinHooks registers all built-in hook factories
func RegisterBuiltinHooks(registry *Registry) {
	registry.RegisterFactory("script", NewScriptHook)
	registry.RegisterFactory("webhook", NewWebhookHook)
	registry.RegisterFactory("slack", NewSlackHook)
}
