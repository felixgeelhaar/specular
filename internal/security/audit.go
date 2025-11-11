package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	// Workflow events
	AuditWorkflowStart    AuditEventType = "workflow.start"
	AuditWorkflowComplete AuditEventType = "workflow.complete"
	AuditWorkflowFailed   AuditEventType = "workflow.failed"

	// Credential events
	AuditCredentialCreated  AuditEventType = "credential.created"  //#nosec G101 -- Event type name, not a credential
	AuditCredentialAccessed AuditEventType = "credential.accessed" //#nosec G101 -- Event type name, not a credential
	AuditCredentialUpdated  AuditEventType = "credential.updated"  //#nosec G101 -- Event type name, not a credential
	AuditCredentialDeleted  AuditEventType = "credential.deleted"  //#nosec G101 -- Event type name, not a credential
	AuditCredentialRotated  AuditEventType = "credential.rotated"  //#nosec G101 -- Event type name, not a credential

	// Policy events
	AuditPolicyViolation AuditEventType = "policy.violation"
	AuditPolicyEnforced  AuditEventType = "policy.enforced"

	// Secret scanning events
	AuditSecretDetected AuditEventType = "secret.detected"
	AuditSecretBlocked  AuditEventType = "secret.blocked"

	// Access events
	AuditAccessGranted AuditEventType = "access.granted"
	AuditAccessDenied  AuditEventType = "access.denied"
)

// AuditSeverity represents the severity level of an audit event
type AuditSeverity string

const (
	SeverityInfo     AuditSeverity = "info"
	SeverityWarning  AuditSeverity = "warning"
	SeverityError    AuditSeverity = "error"
	SeverityCritical AuditSeverity = "critical"
)

// AuditEvent represents a security audit event
type AuditEvent struct {
	// ID is a unique identifier for the event
	ID string `json:"id"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Type is the event type
	Type AuditEventType `json:"type"`

	// Severity is the event severity
	Severity AuditSeverity `json:"severity"`

	// Actor is who performed the action (user, system, etc.)
	Actor string `json:"actor"`

	// Resource is what was acted upon
	Resource string `json:"resource"`

	// Action is what was done
	Action string `json:"action"`

	// Result indicates success or failure
	Result string `json:"result"`

	// Details provides additional context
	Details map[string]interface{} `json:"details,omitempty"`

	// IPAddress is the source IP (if applicable)
	IPAddress string `json:"ipAddress,omitempty"`

	// UserAgent is the user agent (if applicable)
	UserAgent string `json:"userAgent,omitempty"`
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	mu sync.Mutex

	// logPath is the directory where audit logs are stored
	logPath string

	// currentFile is the current log file
	currentFile *os.File

	// currentDate is the date of the current log file
	currentDate string

	// enableConsole enables console logging
	enableConsole bool
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logPath string, enableConsole bool) (*AuditLogger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logger := &AuditLogger{
		logPath:       logPath,
		enableConsole: enableConsole,
	}

	// Open initial log file
	if err := logger.rotateIfNeeded(); err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return logger, nil
}

// Log logs an audit event
func (l *AuditLogger) Log(event *AuditEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Generate ID if not set
	if event.ID == "" {
		event.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Rotate log file if needed
	if err := l.rotateIfNeeded(); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write to file
	if _, err := l.currentFile.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	// Flush immediately for audit logs
	if err := l.currentFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Log to console if enabled
	if l.enableConsole {
		fmt.Printf("[AUDIT] %s | %s | %s | %s | %s\n",
			event.Timestamp.Format("2006-01-02 15:04:05"),
			event.Severity,
			event.Type,
			event.Actor,
			event.Action,
		)
	}

	return nil
}

// LogWorkflowStart logs a workflow start event
func (l *AuditLogger) LogWorkflowStart(workflowID, goal, profile, actor string) error {
	return l.Log(&AuditEvent{
		Type:     AuditWorkflowStart,
		Severity: SeverityInfo,
		Actor:    actor,
		Resource: workflowID,
		Action:   "start_workflow",
		Result:   "success",
		Details: map[string]interface{}{
			"goal":    goal,
			"profile": profile,
		},
	})
}

// LogWorkflowComplete logs a workflow completion event
func (l *AuditLogger) LogWorkflowComplete(workflowID, actor string, duration time.Duration, cost float64) error {
	return l.Log(&AuditEvent{
		Type:     AuditWorkflowComplete,
		Severity: SeverityInfo,
		Actor:    actor,
		Resource: workflowID,
		Action:   "complete_workflow",
		Result:   "success",
		Details: map[string]interface{}{
			"duration": duration.String(),
			"cost":     cost,
		},
	})
}

// LogWorkflowFailed logs a workflow failure event
func (l *AuditLogger) LogWorkflowFailed(workflowID, actor, reason string) error {
	return l.Log(&AuditEvent{
		Type:     AuditWorkflowFailed,
		Severity: SeverityError,
		Actor:    actor,
		Resource: workflowID,
		Action:   "workflow_failed",
		Result:   "failure",
		Details: map[string]interface{}{
			"reason": reason,
		},
	})
}

// LogCredentialAccess logs a credential access event
func (l *AuditLogger) LogCredentialAccess(credentialName, actor string, success bool) error {
	result := "success"
	severity := SeverityInfo
	if !success {
		result = "failure"
		severity = SeverityWarning
	}

	return l.Log(&AuditEvent{
		Type:     AuditCredentialAccessed,
		Severity: severity,
		Actor:    actor,
		Resource: credentialName,
		Action:   "access_credential",
		Result:   result,
	})
}

// LogPolicyViolation logs a policy violation event
func (l *AuditLogger) LogPolicyViolation(policy, resource, actor, reason string) error {
	return l.Log(&AuditEvent{
		Type:     AuditPolicyViolation,
		Severity: SeverityWarning,
		Actor:    actor,
		Resource: resource,
		Action:   "policy_violation",
		Result:   "blocked",
		Details: map[string]interface{}{
			"policy": policy,
			"reason": reason,
		},
	})
}

// LogSecretDetected logs a secret detection event
func (l *AuditLogger) LogSecretDetected(secretType, location, actor string, blocked bool) error {
	severity := SeverityWarning
	result := "detected"
	if blocked {
		severity = SeverityCritical
		result = "blocked"
	}

	return l.Log(&AuditEvent{
		Type:     AuditSecretDetected,
		Severity: severity,
		Actor:    actor,
		Resource: location,
		Action:   "secret_detected",
		Result:   result,
		Details: map[string]interface{}{
			"secretType": secretType,
			"blocked":    blocked,
		},
	})
}

// Query queries audit logs based on filters
func (l *AuditLogger) Query(filter AuditFilter) ([]*AuditEvent, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	events := []*AuditEvent{}

	// Get list of log files to search
	files, err := l.getLogFiles(filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get log files: %w", err)
	}

	// Read and parse each file
	for _, file := range files {
		fileEvents, err := l.readLogFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read log file %s: %w", file, err)
		}

		// Apply filters
		for _, event := range fileEvents {
			if filter.Matches(event) {
				events = append(events, event)
			}
		}

		// Stop if we've reached the limit
		if filter.Limit > 0 && len(events) >= filter.Limit {
			break
		}
	}

	// Trim to limit
	if filter.Limit > 0 && len(events) > filter.Limit {
		events = events[:filter.Limit]
	}

	return events, nil
}

// AuditFilter defines filters for querying audit logs
type AuditFilter struct {
	StartDate time.Time
	EndDate   time.Time
	EventType AuditEventType
	Actor     string
	Resource  string
	Severity  AuditSeverity
	Limit     int
}

// Matches checks if an event matches the filter
func (f *AuditFilter) Matches(event *AuditEvent) bool {
	if !f.StartDate.IsZero() && event.Timestamp.Before(f.StartDate) {
		return false
	}

	if !f.EndDate.IsZero() && event.Timestamp.After(f.EndDate) {
		return false
	}

	if f.EventType != "" && event.Type != f.EventType {
		return false
	}

	if f.Actor != "" && event.Actor != f.Actor {
		return false
	}

	if f.Resource != "" && event.Resource != f.Resource {
		return false
	}

	if f.Severity != "" && event.Severity != f.Severity {
		return false
	}

	return true
}

// rotateIfNeeded rotates the log file if the date has changed
func (l *AuditLogger) rotateIfNeeded() error {
	currentDate := time.Now().Format("2006-01-02")

	if l.currentDate == currentDate && l.currentFile != nil {
		return nil
	}

	// Close existing file
	if l.currentFile != nil {
		if err := l.currentFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close audit log file: %v\n", err)
		}
	}

	// Open new file
	filename := filepath.Join(l.logPath, fmt.Sprintf("audit-%s.log", currentDate))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	l.currentFile = file
	l.currentDate = currentDate

	return nil
}

// getLogFiles returns log files between start and end dates
func (l *AuditLogger) getLogFiles(start, end time.Time) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(l.logPath, "audit-*.log"))
	if err != nil {
		return nil, err
	}

	if start.IsZero() && end.IsZero() {
		return files, nil
	}

	filtered := []string{}
	for _, file := range files {
		// Extract date from filename
		basename := filepath.Base(file)
		dateStr := basename[6 : len(basename)-4] // Extract "2006-01-02" from "audit-2006-01-02.log"
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Check if file is in date range
		if !start.IsZero() && fileDate.Before(start) {
			continue
		}
		if !end.IsZero() && fileDate.After(end) {
			continue
		}

		filtered = append(filtered, file)
	}

	return filtered, nil
}

// readLogFile reads and parses a log file
func (l *AuditLogger) readLogFile(filename string) ([]*AuditEvent, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := []byte{}
	events := []*AuditEvent{}

	for _, b := range data {
		if b == '\n' {
			if len(lines) > 0 {
				var event AuditEvent
				if err := json.Unmarshal(lines, &event); err == nil {
					events = append(events, &event)
				}
				lines = []byte{}
			}
		} else {
			lines = append(lines, b)
		}
	}

	return events, nil
}

// Close closes the audit logger
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentFile != nil {
		return l.currentFile.Close()
	}

	return nil
}
