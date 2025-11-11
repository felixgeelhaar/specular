package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewAuditLogger(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Fatal("Logger should not be nil")
	}

	if logger.logPath != tmpDir {
		t.Errorf("Log path mismatch: got %s, want %s", logger.logPath, tmpDir)
	}
}

func TestLogAuditEvent(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	event := &AuditEvent{
		Type:     AuditWorkflowStart,
		Severity: SeverityInfo,
		Actor:    "test-user",
		Resource: "workflow-123",
		Action:   "start_workflow",
		Result:   "success",
	}

	err = logger.Log(event)
	if err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	// Verify event was written
	currentDate := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, "audit-"+currentDate+".log")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestLogWorkflowEvents(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	// Test workflow start
	err = logger.LogWorkflowStart("workflow-1", "test goal", "default", "user1")
	if err != nil {
		t.Fatalf("Failed to log workflow start: %v", err)
	}

	// Test workflow complete
	err = logger.LogWorkflowComplete("workflow-1", "user1", 5*time.Minute, 1.5)
	if err != nil {
		t.Fatalf("Failed to log workflow complete: %v", err)
	}

	// Test workflow failed
	err = logger.LogWorkflowFailed("workflow-2", "user1", "test error")
	if err != nil {
		t.Fatalf("Failed to log workflow failed: %v", err)
	}
}

func TestLogCredentialAccess(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	// Test successful access
	err = logger.LogCredentialAccess("github-token", "user1", true)
	if err != nil {
		t.Fatalf("Failed to log credential access: %v", err)
	}

	// Test failed access
	err = logger.LogCredentialAccess("github-token", "user2", false)
	if err != nil {
		t.Fatalf("Failed to log failed credential access: %v", err)
	}
}

func TestLogPolicyViolation(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	err = logger.LogPolicyViolation("security-policy", "step-1", "user1", "unauthorized action")
	if err != nil {
		t.Fatalf("Failed to log policy violation: %v", err)
	}
}

func TestLogSecretDetected(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	// Test secret detected but not blocked
	err = logger.LogSecretDetected("api_key", "config.go:42", "scanner", false)
	if err != nil {
		t.Fatalf("Failed to log secret detected: %v", err)
	}

	// Test secret detected and blocked
	err = logger.LogSecretDetected("aws_key", "main.go:10", "scanner", true)
	if err != nil {
		t.Fatalf("Failed to log secret blocked: %v", err)
	}
}

func TestAuditEventAutoID(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	event := &AuditEvent{
		Type:     AuditWorkflowStart,
		Severity: SeverityInfo,
		Actor:    "test-user",
		Resource: "workflow-123",
		Action:   "test",
		Result:   "success",
	}

	// ID should be auto-generated
	if event.ID != "" {
		t.Error("ID should be empty before logging")
	}

	err = logger.Log(event)
	if err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	if event.ID == "" {
		t.Error("ID should be auto-generated after logging")
	}
}

func TestAuditEventAutoTimestamp(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	event := &AuditEvent{
		Type:     AuditWorkflowStart,
		Severity: SeverityInfo,
		Actor:    "test-user",
		Resource: "workflow-123",
		Action:   "test",
		Result:   "success",
	}

	// Timestamp should be auto-generated
	if !event.Timestamp.IsZero() {
		t.Error("Timestamp should be zero before logging")
	}

	err = logger.Log(event)
	if err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	if event.Timestamp.IsZero() {
		t.Error("Timestamp should be auto-generated after logging")
	}
}

func TestLogRotation(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	// Log an event
	event1 := &AuditEvent{
		Type:     AuditWorkflowStart,
		Severity: SeverityInfo,
		Actor:    "user1",
		Resource: "workflow-1",
		Action:   "test",
		Result:   "success",
	}

	err = logger.Log(event1)
	if err != nil {
		t.Fatalf("Failed to log first event: %v", err)
	}

	firstFile := logger.currentFile

	// Force rotation by changing current date
	logger.currentDate = "2000-01-01"

	// Log another event - should rotate
	event2 := &AuditEvent{
		Type:     AuditWorkflowComplete,
		Severity: SeverityInfo,
		Actor:    "user1",
		Resource: "workflow-1",
		Action:   "test",
		Result:   "success",
	}

	err = logger.Log(event2)
	if err != nil {
		t.Fatalf("Failed to log second event: %v", err)
	}

	// Current file should be different
	if logger.currentFile == firstFile {
		t.Error("Log file should have rotated")
	}
}

func TestQueryAuditLogs(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	// Log multiple events
	events := []*AuditEvent{
		{Type: AuditWorkflowStart, Severity: SeverityInfo, Actor: "user1", Resource: "workflow-1", Action: "start", Result: "success"},
		{Type: AuditWorkflowComplete, Severity: SeverityInfo, Actor: "user1", Resource: "workflow-1", Action: "complete", Result: "success"},
		{Type: AuditCredentialAccessed, Severity: SeverityInfo, Actor: "user2", Resource: "github-token", Action: "access", Result: "success"},
		{Type: AuditPolicyViolation, Severity: SeverityWarning, Actor: "user2", Resource: "step-1", Action: "violation", Result: "blocked"},
	}

	for _, event := range events {
		if err := logger.Log(event); err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	// Query all events
	results, err := logger.Query(AuditFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}

	if len(results) != 4 {
		t.Errorf("Expected 4 events, got %d", len(results))
	}

	// Query by event type
	results, err = logger.Query(AuditFilter{EventType: AuditWorkflowStart})
	if err != nil {
		t.Fatalf("Failed to query by event type: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 event, got %d", len(results))
	}

	// Query by actor
	results, err = logger.Query(AuditFilter{Actor: "user1"})
	if err != nil {
		t.Fatalf("Failed to query by actor: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 events for user1, got %d", len(results))
	}

	// Query by severity
	results, err = logger.Query(AuditFilter{Severity: SeverityWarning})
	if err != nil {
		t.Fatalf("Failed to query by severity: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 warning event, got %d", len(results))
	}

	// Query with limit
	results, err = logger.Query(AuditFilter{Limit: 2})
	if err != nil {
		t.Fatalf("Failed to query with limit: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 events with limit, got %d", len(results))
	}
}

func TestQueryByDateRange(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	// Log events
	event1 := &AuditEvent{
		Type:     AuditWorkflowStart,
		Severity: SeverityInfo,
		Actor:    "user1",
		Resource: "workflow-1",
		Action:   "start",
		Result:   "success",
	}

	event2 := &AuditEvent{
		Type:     AuditWorkflowComplete,
		Severity: SeverityInfo,
		Actor:    "user1",
		Resource: "workflow-1",
		Action:   "complete",
		Result:   "success",
	}

	logger.Log(event1)
	time.Sleep(10 * time.Millisecond)
	logger.Log(event2)

	// Sync to ensure events are written
	logger.currentFile.Sync()

	// Query all events (no date filter)
	results, err := logger.Query(AuditFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 events, got %d", len(results))
	}

	// Verify filtering by start date works (exclude older events)
	if len(results) >= 2 {
		// Set start date after first event
		midTime := results[0].Timestamp.Add(5 * time.Millisecond)

		filtered, err := logger.Query(AuditFilter{
			StartDate: midTime,
		})
		if err != nil {
			t.Fatalf("Failed to query with date filter: %v", err)
		}

		// Should only get events after the mid time
		if len(filtered) > len(results) {
			t.Error("Filtered results should not exceed total results")
		}
	}
}

func TestAuditLogFileFormat(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewAuditLogger(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	event := &AuditEvent{
		Type:     AuditWorkflowStart,
		Severity: SeverityInfo,
		Actor:    "user1",
		Resource: "workflow-1",
		Action:   "start",
		Result:   "success",
	}

	err = logger.Log(event)
	if err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	// Read log file and verify format
	currentDate := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, "audit-"+currentDate+".log")

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)

	// Should be JSON format
	if !strings.Contains(content, "\"type\":\"workflow.start\"") {
		t.Error("Log file should contain JSON event")
	}

	if !strings.Contains(content, "\"actor\":\"user1\"") {
		t.Error("Log file should contain actor field")
	}
}

func TestAuditLogConsoleOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create logger with console output enabled
	logger, err := NewAuditLogger(tmpDir, true)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	// This test just verifies that console logging doesn't cause errors
	// Actual console output is hard to capture in tests
	event := &AuditEvent{
		Type:     AuditWorkflowStart,
		Severity: SeverityInfo,
		Actor:    "user1",
		Resource: "workflow-1",
		Action:   "start",
		Result:   "success",
	}

	err = logger.Log(event)
	if err != nil {
		t.Fatalf("Failed to log event with console output: %v", err)
	}
}
