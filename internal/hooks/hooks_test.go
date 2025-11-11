package hooks

import (
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	data := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	event := NewEvent(EventWorkflowStart, "workflow-123", data)

	if event.Type != EventWorkflowStart {
		t.Errorf("Event type mismatch: got %s, want %s", event.Type, EventWorkflowStart)
	}
	if event.WorkflowID != "workflow-123" {
		t.Errorf("Workflow ID mismatch: got %s, want workflow-123", event.WorkflowID)
	}
	if len(event.Data) != 2 {
		t.Errorf("Data length mismatch: got %d, want 2", len(event.Data))
	}
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestEventGetters(t *testing.T) {
	data := map[string]interface{}{
		"stringKey": "test",
		"intKey":    42,
		"floatKey":  3.14,
		"boolKey":   true,
	}

	event := NewEvent(EventStepBefore, "workflow-1", data)

	// Test GetString
	if val := event.GetString("stringKey"); val != "test" {
		t.Errorf("GetString failed: got %s, want test", val)
	}
	if val := event.GetString("missing"); val != "" {
		t.Errorf("GetString for missing key should return empty string, got %s", val)
	}

	// Test GetInt
	if val := event.GetInt("intKey"); val != 42 {
		t.Errorf("GetInt failed: got %d, want 42", val)
	}
	if val := event.GetInt("missing"); val != 0 {
		t.Errorf("GetInt for missing key should return 0, got %d", val)
	}

	// Test GetFloat
	if val := event.GetFloat("floatKey"); val != 3.14 {
		t.Errorf("GetFloat failed: got %f, want 3.14", val)
	}
	if val := event.GetFloat("missing"); val != 0.0 {
		t.Errorf("GetFloat for missing key should return 0.0, got %f", val)
	}

	// Test GetBool
	if val := event.GetBool("boolKey"); !val {
		t.Error("GetBool failed: got false, want true")
	}
	if val := event.GetBool("missing"); val {
		t.Error("GetBool for missing key should return false, got true")
	}
}

func TestIsValidFailureMode(t *testing.T) {
	tests := []struct {
		mode  string
		valid bool
	}{
		{"ignore", true},
		{"warn", true},
		{"fail", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			result := IsValidFailureMode(tt.mode)
			if result != tt.valid {
				t.Errorf("IsValidFailureMode(%s) = %v, want %v", tt.mode, result, tt.valid)
			}
		})
	}
}

func TestHookConfig(t *testing.T) {
	config := &HookConfig{
		Name:    "test-hook",
		Type:    "webhook",
		Events:  []EventType{EventWorkflowStart, EventWorkflowComplete},
		Enabled: true,
		Config: map[string]interface{}{
			"url": "https://example.com/webhook",
		},
		Timeout:     30 * time.Second,
		FailureMode: "warn",
	}

	if config.Name != "test-hook" {
		t.Errorf("Name mismatch: got %s, want test-hook", config.Name)
	}
	if len(config.Events) != 2 {
		t.Errorf("Events length mismatch: got %d, want 2", len(config.Events))
	}
	if !IsValidFailureMode(config.FailureMode) {
		t.Errorf("Invalid failure mode: %s", config.FailureMode)
	}
}

func TestExecutionResult(t *testing.T) {
	result := ExecutionResult{
		HookName:  "test-hook",
		EventType: EventWorkflowStart,
		Success:   true,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
	}

	if result.HookName != "test-hook" {
		t.Errorf("HookName mismatch: got %s, want test-hook", result.HookName)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Error != "" {
		t.Errorf("Error should be empty for successful result, got %s", result.Error)
	}
}

func TestExecutionResultWithError(t *testing.T) {
	result := ExecutionResult{
		HookName:  "test-hook",
		EventType: EventStepFailed,
		Success:   false,
		Error:     "hook execution failed",
		Duration:  50 * time.Millisecond,
		Timestamp: time.Now(),
	}

	if result.Success {
		t.Error("Success should be false")
	}
	if result.Error == "" {
		t.Error("Error should not be empty for failed result")
	}
}
