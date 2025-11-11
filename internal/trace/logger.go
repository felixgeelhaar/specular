package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger handles trace event logging to disk
type Logger struct {
	// workflowID identifies this workflow
	workflowID string

	// logDir is the directory where logs are stored
	logDir string

	// logFile is the current log file
	logFile *os.File

	// mu protects concurrent writes
	mu sync.Mutex

	// maxFileSize is the maximum size before rotation (in bytes)
	maxFileSize int64

	// maxFiles is the maximum number of rotated files to keep
	maxFiles int

	// enabled indicates if logging is enabled
	enabled bool

	// events buffer for in-memory tracking
	events []*Event
}

// Config contains logger configuration
type Config struct {
	// WorkflowID identifies the workflow
	WorkflowID string

	// LogDir is the directory for log files (default: ~/.specular/logs)
	LogDir string

	// MaxFileSize is the max size before rotation (default: 10MB)
	MaxFileSize int64

	// MaxFiles is the max number of rotated files (default: 5)
	MaxFiles int

	// Enabled controls whether logging is active
	Enabled bool
}

// DefaultConfig returns default logger configuration
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir() // Best-effort, falls back to current directory
	logDir := filepath.Join(homeDir, ".specular", "logs")

	return Config{
		WorkflowID:  generateWorkflowID(),
		LogDir:      logDir,
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		MaxFiles:    5,
		Enabled:     false, // Disabled by default
	}
}

// NewLogger creates a new trace logger
func NewLogger(config Config) (*Logger, error) {
	if !config.Enabled {
		return &Logger{
			workflowID: config.WorkflowID,
			enabled:    false,
			events:     []*Event{},
		}, nil
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logPath := filepath.Join(config.LogDir, fmt.Sprintf("trace_%s.json", config.WorkflowID))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := &Logger{
		workflowID:  config.WorkflowID,
		logDir:      config.LogDir,
		logFile:     logFile,
		maxFileSize: config.MaxFileSize,
		maxFiles:    config.MaxFiles,
		enabled:     true,
		events:      []*Event{},
	}

	// Write initial metadata
	metadata := map[string]interface{}{
		"workflow_id": config.WorkflowID,
		"started_at":  time.Now(),
		"version":     "specular/v1",
	}
	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	fmt.Fprintf(logFile, "%s\n", metadataJSON)

	return logger, nil
}

// Log logs a trace event
func (l *Logger) Log(event *Event) error {
	if !l.enabled {
		// Still track events in memory even if logging is disabled
		l.mu.Lock()
		l.events = append(l.events, event)
		l.mu.Unlock()
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Track event in memory
	l.events = append(l.events, event)

	// Check if rotation is needed
	if err := l.checkRotation(); err != nil {
		return fmt.Errorf("log rotation failed: %w", err)
	}

	// Write event to file
	eventJSON, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	if _, err := fmt.Fprintf(l.logFile, "%s\n", eventJSON); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	// Sync to disk periodically
	if len(l.events)%10 == 0 {
		if err := l.logFile.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to sync trace log: %v\n", err)
		}
	}

	return nil
}

// LogWorkflowStart logs a workflow start event
func (l *Logger) LogWorkflowStart(goal, profile string) error {
	event := NewEvent(EventTypeWorkflowStart, l.workflowID, "Workflow started").
		WithData("goal", goal).
		WithData("profile", profile)

	return l.Log(event)
}

// LogWorkflowComplete logs a workflow completion event
func (l *Logger) LogWorkflowComplete(success bool, duration time.Duration, totalCost float64) error {
	event := NewEvent(EventTypeWorkflowComplete, l.workflowID, "Workflow completed").
		WithData("success", success).
		WithData("total_cost", totalCost).
		WithDuration(duration)

	return l.Log(event)
}

// LogStepStart logs a step start event
func (l *Logger) LogStepStart(stepID, stepName string) error {
	event := NewEvent(EventTypeStepStart, l.workflowID, fmt.Sprintf("Step started: %s", stepName)).
		WithStepID(stepID).
		WithData("step_name", stepName)

	return l.Log(event)
}

// LogStepComplete logs a step completion event
func (l *Logger) LogStepComplete(stepID, stepName string, duration time.Duration, cost float64) error {
	event := NewEvent(EventTypeStepComplete, l.workflowID, fmt.Sprintf("Step completed: %s", stepName)).
		WithStepID(stepID).
		WithData("step_name", stepName).
		WithData("cost", cost).
		WithDuration(duration)

	return l.Log(event)
}

// LogStepFail logs a step failure event
func (l *Logger) LogStepFail(stepID, stepName string, err error) error {
	event := NewEvent(EventTypeStepFail, l.workflowID, fmt.Sprintf("Step failed: %s", stepName)).
		WithStepID(stepID).
		WithData("step_name", stepName).
		WithError(err)

	return l.Log(event)
}

// LogPolicyCheck logs a policy check event
func (l *Logger) LogPolicyCheck(stepID string, allowed bool, reason string, metadata map[string]interface{}) error {
	event := NewEvent(EventTypePolicyCheck, l.workflowID, "Policy check").
		WithStepID(stepID).
		WithData("allowed", allowed).
		WithData("reason", reason)

	for k, v := range metadata {
		event.WithData(k, v)
	}

	if !allowed {
		event.Level = "warning"
	}

	return l.Log(event)
}

// LogApprovalRequest logs an approval request event
func (l *Logger) LogApprovalRequest(planSummary string) error {
	event := NewEvent(EventTypeApprovalRequest, l.workflowID, "Approval requested").
		WithData("plan_summary", planSummary)

	return l.Log(event)
}

// LogApprovalResponse logs an approval response event
func (l *Logger) LogApprovalResponse(approved bool) error {
	event := NewEvent(EventTypeApprovalResponse, l.workflowID, "Approval response").
		WithData("approved", approved)

	if !approved {
		event.Level = "warning"
	}

	return l.Log(event)
}

// LogError logs an error event
func (l *Logger) LogError(message string, err error) error {
	event := NewEvent(EventTypeError, l.workflowID, message).
		WithError(err)

	return l.Log(event)
}

// LogWarning logs a warning event
func (l *Logger) LogWarning(message string) error {
	event := NewEvent(EventTypeWarning, l.workflowID, message)
	event.Level = "warning"

	return l.Log(event)
}

// LogInfo logs an informational event
func (l *Logger) LogInfo(message string) error {
	event := NewEvent(EventTypeInfo, l.workflowID, message)

	return l.Log(event)
}

// checkRotation checks if log file needs rotation
func (l *Logger) checkRotation() error {
	if l.logFile == nil {
		return nil
	}

	info, err := l.logFile.Stat()
	if err != nil {
		return err
	}

	if info.Size() < l.maxFileSize {
		return nil // No rotation needed
	}

	return l.rotate()
}

// rotate rotates the log file
func (l *Logger) rotate() error {
	// Close current file
	if err := l.logFile.Close(); err != nil {
		return err
	}

	// Rename current file with timestamp
	currentPath := filepath.Join(l.logDir, fmt.Sprintf("trace_%s.json", l.workflowID))
	timestamp := time.Now().Format("20060102_150405")
	rotatedPath := filepath.Join(l.logDir, fmt.Sprintf("trace_%s_%s.json", l.workflowID, timestamp))

	if err := os.Rename(currentPath, rotatedPath); err != nil {
		return err
	}

	// Clean up old rotated files
	if err := l.cleanupOldFiles(); err != nil {
		// Log but don't fail on cleanup errors
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup old log files: %v\n", err)
	}

	// Open new file
	logFile, err := os.OpenFile(currentPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	l.logFile = logFile
	return nil
}

// cleanupOldFiles removes old rotated log files
func (l *Logger) cleanupOldFiles() error {
	pattern := filepath.Join(l.logDir, fmt.Sprintf("trace_%s_*.json", l.workflowID))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// Keep only the most recent maxFiles
	if len(files) <= l.maxFiles {
		return nil
	}

	// Remove oldest files
	for i := 0; i < len(files)-l.maxFiles; i++ {
		if err := os.Remove(files[i]); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the logger and syncs any buffered data
func (l *Logger) Close() error {
	if !l.enabled || l.logFile == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.logFile.Sync(); err != nil {
		return err
	}

	return l.logFile.Close()
}

// GetLogPath returns the path to the current log file
func (l *Logger) GetLogPath() string {
	if !l.enabled {
		return ""
	}
	return filepath.Join(l.logDir, fmt.Sprintf("trace_%s.json", l.workflowID))
}

// GetWorkflowID returns the workflow ID
func (l *Logger) GetWorkflowID() string {
	return l.workflowID
}

// GetEvents returns all logged events (from memory)
func (l *Logger) GetEvents() []*Event {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Return a copy
	events := make([]*Event, len(l.events))
	copy(events, l.events)
	return events
}

// generateWorkflowID generates a unique workflow ID
func generateWorkflowID() string {
	return fmt.Sprintf("auto-%d", time.Now().Unix())
}
