package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewScriptHook(t *testing.T) {
	config := &HookConfig{
		Name:    "test-script",
		Type:    "script",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config: map[string]interface{}{
			"script": "/path/to/script.sh",
			"args":   []interface{}{"arg1", "arg2"},
			"shell":  "/bin/sh",
		},
	}

	hook, err := NewScriptHook(config)
	if err != nil {
		t.Errorf("NewScriptHook failed: %v", err)
	}

	if hook.Name() != "test-script" {
		t.Errorf("Name mismatch: got %s, want test-script", hook.Name())
	}

	if !hook.Enabled() {
		t.Error("Hook should be enabled")
	}

	scriptHook := hook.(*ScriptHook)
	if scriptHook.scriptPath != "/path/to/script.sh" {
		t.Errorf("Script path mismatch: got %s", scriptHook.scriptPath)
	}

	if len(scriptHook.args) != 2 {
		t.Errorf("Args length mismatch: got %d, want 2", len(scriptHook.args))
	}

	if scriptHook.shell != "/bin/sh" {
		t.Errorf("Shell mismatch: got %s, want /bin/sh", scriptHook.shell)
	}
}

func TestNewScriptHookMissingScript(t *testing.T) {
	config := &HookConfig{
		Name:    "test-script",
		Type:    "script",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config:  map[string]interface{}{},
	}

	_, err := NewScriptHook(config)
	if err == nil {
		t.Error("Expected error for missing script path")
	}
}

func TestScriptHookExecute(t *testing.T) {
	// Create a temporary script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")

	script := `#!/bin/sh
echo "Event: $HOOK_EVENT_TYPE"
echo "Workflow: $HOOK_WORKFLOW_ID"
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	config := &HookConfig{
		Name:    "test-script",
		Type:    "script",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config: map[string]interface{}{
			"script": scriptPath,
		},
	}

	hook, err := NewScriptHook(config)
	if err != nil {
		t.Fatalf("NewScriptHook failed: %v", err)
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", map[string]interface{}{
		"testKey": "testValue",
	})

	err = hook.Execute(context.Background(), event)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
}

func TestScriptHookExecuteFailure(t *testing.T) {
	// Create a script that fails
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fail.sh")

	script := `#!/bin/sh
echo "This script fails" >&2
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	config := &HookConfig{
		Name:    "failing-script",
		Type:    "script",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config: map[string]interface{}{
			"script": scriptPath,
		},
	}

	hook, err := NewScriptHook(config)
	if err != nil {
		t.Fatalf("NewScriptHook failed: %v", err)
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)

	err = hook.Execute(context.Background(), event)
	if err == nil {
		t.Error("Expected error from failing script")
	}

	if !strings.Contains(err.Error(), "script failed") {
		t.Errorf("Error message should mention script failure, got: %s", err.Error())
	}
}

func TestNewWebhookHook(t *testing.T) {
	config := &HookConfig{
		Name:    "test-webhook",
		Type:    "webhook",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config: map[string]interface{}{
			"url": "https://example.com/webhook",
			"headers": map[string]interface{}{
				"Authorization": "Bearer token",
				"X-Custom":      "value",
			},
		},
		Timeout: 10 * time.Second,
	}

	hook, err := NewWebhookHook(config)
	if err != nil {
		t.Errorf("NewWebhookHook failed: %v", err)
	}

	if hook.Name() != "test-webhook" {
		t.Errorf("Name mismatch: got %s, want test-webhook", hook.Name())
	}

	webhookHook := hook.(*WebhookHook)
	if webhookHook.url != "https://example.com/webhook" {
		t.Errorf("URL mismatch: got %s", webhookHook.url)
	}

	if len(webhookHook.headers) != 2 {
		t.Errorf("Headers length mismatch: got %d, want 2", len(webhookHook.headers))
	}

	if webhookHook.headers["Authorization"] != "Bearer token" {
		t.Error("Authorization header not set correctly")
	}
}

func TestNewWebhookHookMissingURL(t *testing.T) {
	config := &HookConfig{
		Name:    "test-webhook",
		Type:    "webhook",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config:  map[string]interface{}{},
	}

	_, err := NewWebhookHook(config)
	if err == nil {
		t.Error("Expected error for missing URL")
	}
}

func TestWebhookHookExecute(t *testing.T) {
	// Create test server
	receivedEvent := &Event{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}

		// Read and decode event
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, receivedEvent)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HookConfig{
		Name:    "test-webhook",
		Type:    "webhook",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config: map[string]interface{}{
			"url": server.URL,
		},
		Timeout: 5 * time.Second,
	}

	hook, err := NewWebhookHook(config)
	if err != nil {
		t.Fatalf("NewWebhookHook failed: %v", err)
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", map[string]interface{}{
		"testKey": "testValue",
	})

	err = hook.Execute(context.Background(), event)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	// Verify event was received
	if receivedEvent.Type != EventWorkflowStart {
		t.Errorf("Event type mismatch: got %s, want %s", receivedEvent.Type, EventWorkflowStart)
	}

	if receivedEvent.WorkflowID != "test-workflow" {
		t.Errorf("Workflow ID mismatch: got %s, want test-workflow", receivedEvent.WorkflowID)
	}
}

func TestWebhookHookExecuteFailure(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &HookConfig{
		Name:    "test-webhook",
		Type:    "webhook",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config: map[string]interface{}{
			"url": server.URL,
		},
		Timeout: 5 * time.Second,
	}

	hook, err := NewWebhookHook(config)
	if err != nil {
		t.Fatalf("NewWebhookHook failed: %v", err)
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)

	err = hook.Execute(context.Background(), event)
	if err == nil {
		t.Error("Expected error from webhook")
	}

	if !strings.Contains(err.Error(), "webhook returned status 500") {
		t.Errorf("Error message should mention status code, got: %s", err.Error())
	}
}

func TestNewSlackHook(t *testing.T) {
	config := &HookConfig{
		Name:    "test-slack",
		Type:    "slack",
		Events:  []EventType{EventWorkflowStart, EventWorkflowComplete},
		Enabled: true,
		Config: map[string]interface{}{
			"webhookUrl": "https://hooks.slack.com/services/xxx/yyy/zzz",
			"channel":    "#alerts",
			"username":   "CustomBot",
			"iconEmoji":  ":rocket:",
		},
		Timeout: 10 * time.Second,
	}

	hook, err := NewSlackHook(config)
	if err != nil {
		t.Errorf("NewSlackHook failed: %v", err)
	}

	if hook.Name() != "test-slack" {
		t.Errorf("Name mismatch: got %s, want test-slack", hook.Name())
	}

	slackHook := hook.(*SlackHook)
	if slackHook.channel != "#alerts" {
		t.Errorf("Channel mismatch: got %s, want #alerts", slackHook.channel)
	}

	if slackHook.username != "CustomBot" {
		t.Errorf("Username mismatch: got %s, want CustomBot", slackHook.username)
	}

	if slackHook.iconEmoji != ":rocket:" {
		t.Errorf("Icon emoji mismatch: got %s, want :rocket:", slackHook.iconEmoji)
	}
}

func TestNewSlackHookMissingWebhookURL(t *testing.T) {
	config := &HookConfig{
		Name:    "test-slack",
		Type:    "slack",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config:  map[string]interface{}{},
	}

	_, err := NewSlackHook(config)
	if err == nil {
		t.Error("Expected error for missing webhook URL")
	}
}

func TestSlackHookExecute(t *testing.T) {
	// Create test server
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HookConfig{
		Name:    "test-slack",
		Type:    "slack",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config: map[string]interface{}{
			"webhookUrl": server.URL,
			"channel":    "#test",
		},
		Timeout: 5 * time.Second,
	}

	hook, err := NewSlackHook(config)
	if err != nil {
		t.Fatalf("NewSlackHook failed: %v", err)
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)

	err = hook.Execute(context.Background(), event)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	// Verify payload
	if receivedPayload["channel"] != "#test" {
		t.Errorf("Channel mismatch in payload: got %v", receivedPayload["channel"])
	}

	text, ok := receivedPayload["text"].(string)
	if !ok {
		t.Error("Text field missing or not a string")
	}

	if !strings.Contains(text, "Workflow started") {
		t.Errorf("Text should contain 'Workflow started', got: %s", text)
	}
}

func TestSlackHookFormatMessage(t *testing.T) {
	hook := &SlackHook{
		name:       "test-slack",
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    true,
		webhookURL: "https://example.com",
	}

	tests := []struct {
		eventType      EventType
		data           map[string]interface{}
		expectedPrefix string
	}{
		{
			eventType:      EventWorkflowStart,
			data:           nil,
			expectedPrefix: "🚀 Workflow started:",
		},
		{
			eventType: EventWorkflowComplete,
			data: map[string]interface{}{
				"duration": "5m30s",
				"cost":     1.23,
			},
			expectedPrefix: "✅ Workflow completed:",
		},
		{
			eventType: EventWorkflowFailed,
			data: map[string]interface{}{
				"error": "something went wrong",
			},
			expectedPrefix: "❌ Workflow failed:",
		},
		{
			eventType: EventPlanCreated,
			data: map[string]interface{}{
				"steps": 10,
			},
			expectedPrefix: "📋 Plan created:",
		},
		{
			eventType: EventStepBefore,
			data: map[string]interface{}{
				"stepId":   "step-1",
				"stepType": "execute",
			},
			expectedPrefix: "▶️ Starting step:",
		},
		{
			eventType: EventStepAfter,
			data: map[string]interface{}{
				"stepId": "step-1",
			},
			expectedPrefix: "✅ Completed step:",
		},
		{
			eventType: EventStepFailed,
			data: map[string]interface{}{
				"stepId": "step-1",
				"error":  "execution failed",
			},
			expectedPrefix: "❌ Step failed:",
		},
		{
			eventType: EventPolicyViolation,
			data: map[string]interface{}{
				"policy": "security-policy",
				"reason": "unauthorized action",
			},
			expectedPrefix: "🚫 Policy violation:",
		},
		{
			eventType:      EventDriftDetected,
			data:           nil,
			expectedPrefix: "⚠️ Drift detected",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			event := NewEvent(tt.eventType, "test-workflow", tt.data)
			message := hook.formatMessage(event)

			if !strings.HasPrefix(message, tt.expectedPrefix) {
				t.Errorf("Message should start with '%s', got: %s", tt.expectedPrefix, message)
			}
		})
	}
}

func TestRegisterBuiltinHooks(t *testing.T) {
	registry := NewRegistry()
	RegisterBuiltinHooks(registry)

	// Verify factories are registered
	tests := []struct {
		hookType string
		config   *HookConfig
	}{
		{
			hookType: "script",
			config: &HookConfig{
				Name:    "test-script",
				Type:    "script",
				Events:  []EventType{EventWorkflowStart},
				Enabled: true,
				Config: map[string]interface{}{
					"script": "/tmp/test.sh",
				},
			},
		},
		{
			hookType: "webhook",
			config: &HookConfig{
				Name:    "test-webhook",
				Type:    "webhook",
				Events:  []EventType{EventWorkflowStart},
				Enabled: true,
				Config: map[string]interface{}{
					"url": "https://example.com",
				},
			},
		},
		{
			hookType: "slack",
			config: &HookConfig{
				Name:    "test-slack",
				Type:    "slack",
				Events:  []EventType{EventWorkflowStart},
				Enabled: true,
				Config: map[string]interface{}{
					"webhookUrl": "https://hooks.slack.com/xxx",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.hookType, func(t *testing.T) {
			err := registry.RegisterFromConfig(tt.config)
			if err != nil {
				t.Errorf("Failed to register %s hook: %v", tt.hookType, err)
			}
		})
	}
}

func TestSlackHookExecuteFailure(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	config := &HookConfig{
		Name:    "test-slack",
		Type:    "slack",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
		Config: map[string]interface{}{
			"webhookUrl": server.URL,
		},
		Timeout: 5 * time.Second,
	}

	hook, err := NewSlackHook(config)
	if err != nil {
		t.Fatalf("NewSlackHook failed: %v", err)
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)

	err = hook.Execute(context.Background(), event)
	if err == nil {
		t.Error("Expected error from Slack hook")
	}

	expectedError := fmt.Sprintf("Slack returned status %d", http.StatusBadRequest)
	if err.Error() != expectedError {
		t.Errorf("Error message mismatch: got %s, want %s", err.Error(), expectedError)
	}
}
