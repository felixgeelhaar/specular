package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewLogger tests logger creation
func TestNewLogger(t *testing.T) {
	tmpDir := t.TempDir()

	config := Config{
		WorkflowID:  "test-workflow",
		LogDir:      tmpDir,
		MaxFileSize: 1024,
		MaxFiles:    3,
		Enabled:     true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	if logger.workflowID != "test-workflow" {
		t.Errorf("Expected workflow ID 'test-workflow', got '%s'", logger.workflowID)
	}

	if !logger.enabled {
		t.Error("Expected logger to be enabled")
	}

	// Verify log file was created
	logPath := logger.GetLogPath()
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file not created at %s", logPath)
	}
}

// TestNewLoggerDisabled tests disabled logger
func TestNewLoggerDisabled(t *testing.T) {
	config := Config{
		WorkflowID: "test-workflow",
		Enabled:    false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create disabled logger: %v", err)
	}

	if logger.enabled {
		t.Error("Expected logger to be disabled")
	}

	if logger.GetLogPath() != "" {
		t.Error("Disabled logger should not have a log path")
	}
}

// TestLogEvent tests basic event logging
func TestLogEvent(t *testing.T) {
	tmpDir := t.TempDir()

	config := Config{
		WorkflowID:  "test-workflow",
		LogDir:      tmpDir,
		MaxFileSize: 1024 * 1024,
		MaxFiles:    3,
		Enabled:     true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log an event
	event := NewEvent(EventTypeInfo, "test-workflow", "Test message").
		WithData("key", "value")

	if err := logger.Log(event); err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	// Verify event was tracked in memory
	events := logger.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event in memory, got %d", len(events))
	}

	// Verify event was written to file
	logger.Close()

	content, err := os.ReadFile(logger.GetLogPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "Test message") {
		t.Error("Log file should contain the test message")
	}
}

// TestLogWorkflowStart tests workflow start logging
func TestLogWorkflowStart(t *testing.T) {
	tmpDir := t.TempDir()

	config := Config{
		WorkflowID:  "test-workflow",
		LogDir:      tmpDir,
		MaxFileSize: 1024 * 1024,
		MaxFiles:    3,
		Enabled:     true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	if err := logger.LogWorkflowStart("Build API", "default"); err != nil {
		t.Fatalf("Failed to log workflow start: %v", err)
	}

	events := logger.GetEvents()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Type != EventTypeWorkflowStart {
		t.Errorf("Expected EventTypeWorkflowStart, got %v", event.Type)
	}

	if event.Data["goal"] != "Build API" {
		t.Errorf("Expected goal 'Build API', got '%v'", event.Data["goal"])
	}

	if event.Data["profile"] != "default" {
		t.Errorf("Expected profile 'default', got '%v'", event.Data["profile"])
	}
}

// TestLogStepLifecycle tests step start/complete/fail logging
func TestLogStepLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	config := Config{
		WorkflowID:  "test-workflow",
		LogDir:      tmpDir,
		MaxFileSize: 1024 * 1024,
		MaxFiles:    3,
		Enabled:     true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log step start
	if err := logger.LogStepStart("step-1", "Generate Spec"); err != nil {
		t.Fatalf("Failed to log step start: %v", err)
	}

	// Log step complete
	if err := logger.LogStepComplete("step-1", "Generate Spec", 2*time.Second, 0.05); err != nil {
		t.Fatalf("Failed to log step complete: %v", err)
	}

	// Log step fail
	if err := logger.LogStepFail("step-2", "Generate Plan", fmt.Errorf("connection timeout")); err != nil {
		t.Fatalf("Failed to log step fail: %v", err)
	}

	events := logger.GetEvents()
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	// Verify step start
	if events[0].Type != EventTypeStepStart {
		t.Errorf("Expected EventTypeStepStart, got %v", events[0].Type)
	}
	if events[0].StepID != "step-1" {
		t.Errorf("Expected step ID 'step-1', got '%s'", events[0].StepID)
	}

	// Verify step complete
	if events[1].Type != EventTypeStepComplete {
		t.Errorf("Expected EventTypeStepComplete, got %v", events[1].Type)
	}
	if events[1].Duration == nil {
		t.Error("Expected duration to be set")
	}

	// Verify step fail
	if events[2].Type != EventTypeStepFail {
		t.Errorf("Expected EventTypeStepFail, got %v", events[2].Type)
	}
	if events[2].Error == "" {
		t.Error("Expected error to be set")
	}
	if events[2].Level != "error" {
		t.Errorf("Expected level 'error', got '%s'", events[2].Level)
	}
}

// TestLogPolicyCheck tests policy check logging
func TestLogPolicyCheck(t *testing.T) {
	tmpDir := t.TempDir()

	config := Config{
		WorkflowID:  "test-workflow",
		LogDir:      tmpDir,
		MaxFileSize: 1024 * 1024,
		MaxFiles:    3,
		Enabled:     true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	metadata := map[string]interface{}{
		"checker": "max_steps",
		"limit":   5,
		"current": 3,
	}

	// Log allowed policy check
	if err := logger.LogPolicyCheck("step-1", true, "", metadata); err != nil {
		t.Fatalf("Failed to log policy check: %v", err)
	}

	// Log denied policy check
	if err := logger.LogPolicyCheck("step-2", false, "max steps exceeded", metadata); err != nil {
		t.Fatalf("Failed to log policy check: %v", err)
	}

	events := logger.GetEvents()
	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	// Verify allowed check
	if events[0].Data["allowed"] != true {
		t.Error("Expected policy to be allowed")
	}

	// Verify denied check
	if events[1].Data["allowed"] != false {
		t.Error("Expected policy to be denied")
	}
	if events[1].Level != "warning" {
		t.Errorf("Expected level 'warning' for denied policy, got '%s'", events[1].Level)
	}
}

// TestLogApproval tests approval logging
func TestLogApproval(t *testing.T) {
	tmpDir := t.TempDir()

	config := Config{
		WorkflowID:  "test-workflow",
		LogDir:      tmpDir,
		MaxFileSize: 1024 * 1024,
		MaxFiles:    3,
		Enabled:     true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log approval request
	if err := logger.LogApprovalRequest("Plan with 5 tasks"); err != nil {
		t.Fatalf("Failed to log approval request: %v", err)
	}

	// Log approval response (approved)
	if err := logger.LogApprovalResponse(true); err != nil {
		t.Fatalf("Failed to log approval response: %v", err)
	}

	// Log approval response (rejected)
	if err := logger.LogApprovalResponse(false); err != nil {
		t.Fatalf("Failed to log approval response: %v", err)
	}

	events := logger.GetEvents()
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	// Verify request
	if events[0].Type != EventTypeApprovalRequest {
		t.Errorf("Expected EventTypeApprovalRequest, got %v", events[0].Type)
	}

	// Verify approved response
	if events[1].Data["approved"] != true {
		t.Error("Expected approval to be true")
	}

	// Verify rejected response
	if events[2].Data["approved"] != false {
		t.Error("Expected approval to be false")
	}
	if events[2].Level != "warning" {
		t.Errorf("Expected level 'warning' for rejection, got '%s'", events[2].Level)
	}
}

// TestLogRotation tests log file rotation
func TestLogRotation(t *testing.T) {
	tmpDir := t.TempDir()

	config := Config{
		WorkflowID:  "test-workflow",
		LogDir:      tmpDir,
		MaxFileSize: 200, // Small size to trigger rotation
		MaxFiles:    2,
		Enabled:     true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log enough events to trigger rotation
	for i := 0; i < 10; i++ {
		event := NewEvent(EventTypeInfo, "test-workflow", fmt.Sprintf("Message %d", i)).
			WithData("index", i)

		if err := logger.Log(event); err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	// Force sync
	logger.Close()

	// Check for rotated files
	pattern := filepath.Join(tmpDir, "trace_test-workflow_*.json")
	rotatedFiles, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob rotated files: %v", err)
	}

	if len(rotatedFiles) == 0 {
		t.Error("Expected at least one rotated file")
	}
}

// TestDisabledLoggerTracksEvents tests that disabled logger still tracks events in memory
func TestDisabledLoggerTracksEvents(t *testing.T) {
	config := Config{
		WorkflowID: "test-workflow",
		Enabled:    false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log events even though disabled
	for i := 0; i < 5; i++ {
		event := NewEvent(EventTypeInfo, "test-workflow", fmt.Sprintf("Message %d", i))
		if err := logger.Log(event); err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	// Verify events are tracked in memory
	events := logger.GetEvents()
	if len(events) != 5 {
		t.Errorf("Expected 5 events in memory, got %d", len(events))
	}
}

// TestEventSerialization tests event JSON serialization
func TestEventSerialization(t *testing.T) {
	event := NewEvent(EventTypeStepComplete, "test-workflow", "Step completed").
		WithStepID("step-1").
		WithData("cost", 0.05).
		WithDuration(2 * time.Second)

	// Serialize to JSON
	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize event: %v", err)
	}

	// Deserialize from JSON
	parsed, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize event: %v", err)
	}

	// Verify fields
	if parsed.Type != EventTypeStepComplete {
		t.Errorf("Expected type %v, got %v", EventTypeStepComplete, parsed.Type)
	}

	if parsed.StepID != "step-1" {
		t.Errorf("Expected step ID 'step-1', got '%s'", parsed.StepID)
	}

	if parsed.Data["cost"] != 0.05 {
		t.Errorf("Expected cost 0.05, got %v", parsed.Data["cost"])
	}
}

// TestEventBuilder tests the fluent event builder
func TestEventBuilder(t *testing.T) {
	event := NewEventBuilder(EventTypeStepStart, "test-workflow").
		Message("Starting step").
		StepID("step-1").
		Data("name", "Generate Spec").
		Build()

	if event.Type != EventTypeStepStart {
		t.Errorf("Expected EventTypeStepStart, got %v", event.Type)
	}

	if event.Message != "Starting step" {
		t.Errorf("Expected message 'Starting step', got '%s'", event.Message)
	}

	if event.StepID != "step-1" {
		t.Errorf("Expected step ID 'step-1', got '%s'", event.StepID)
	}

	if event.Data["name"] != "Generate Spec" {
		t.Errorf("Expected name 'Generate Spec', got '%v'", event.Data["name"])
	}
}

// TestEventContext tests event context
func TestEventContext(t *testing.T) {
	ctx := &EventContext{
		Goal:           "Build API",
		Profile:        "default",
		CompletedSteps: 2,
		TotalSteps:     4,
		TotalCost:      0.15,
		ElapsedTime:    5 * time.Minute,
	}

	event := NewEvent(EventTypeStepStart, "test-workflow", "Starting step").
		WithContext(ctx)

	if event.Context == nil {
		t.Fatal("Expected context to be set")
	}

	if event.Context.Goal != "Build API" {
		t.Errorf("Expected goal 'Build API', got '%s'", event.Context.Goal)
	}

	if event.Context.CompletedSteps != 2 {
		t.Errorf("Expected 2 completed steps, got %d", event.Context.CompletedSteps)
	}

	// Verify JSON serialization includes context
	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	if !strings.Contains(string(jsonData), "Build API") {
		t.Error("JSON should contain context goal")
	}
}
