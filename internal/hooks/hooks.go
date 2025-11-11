package hooks

import (
	"context"
	"time"
)

// EventType represents the type of lifecycle event
type EventType string

const (
	// Workflow-level events
	EventWorkflowStart    EventType = "on_workflow_start"
	EventWorkflowComplete EventType = "on_workflow_complete"
	EventWorkflowFailed   EventType = "on_workflow_failed"

	// Plan-level events
	EventPlanCreated  EventType = "on_plan_created"
	EventPlanApproved EventType = "on_plan_approved"
	EventPlanRejected EventType = "on_plan_rejected"

	// Step-level events
	EventStepBefore EventType = "on_step_before"
	EventStepAfter  EventType = "on_step_after"
	EventStepFailed EventType = "on_step_failed"

	// Policy events
	EventPolicyViolation EventType = "on_policy_violation"

	// Drift events
	EventDriftDetected EventType = "on_drift_detected"
)

// Event represents a lifecycle event that can trigger hooks
type Event struct {
	// Type is the event type
	Type EventType `json:"type"`

	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// WorkflowID identifies the workflow
	WorkflowID string `json:"workflowId"`

	// Data contains event-specific data
	Data map[string]interface{} `json:"data"`
}

// Hook is the interface that all hooks must implement
type Hook interface {
	// Name returns the hook name
	Name() string

	// EventTypes returns the events this hook handles
	EventTypes() []EventType

	// Execute runs the hook for an event
	Execute(ctx context.Context, event *Event) error

	// Enabled returns whether the hook is currently enabled
	Enabled() bool
}

// HookConfig represents hook configuration
type HookConfig struct {
	// Name of the hook
	Name string `yaml:"name" json:"name"`

	// Type of hook (script, webhook, slack, etc.)
	Type string `yaml:"type" json:"type"`

	// Events this hook should trigger on
	Events []EventType `yaml:"events" json:"events"`

	// Enabled indicates if this hook is active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Config contains hook-specific configuration
	Config map[string]interface{} `yaml:"config" json:"config"`

	// Timeout for hook execution
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// FailureMode determines what happens if hook fails
	// "ignore" - log and continue
	// "warn" - log warning and continue
	// "fail" - fail the workflow
	FailureMode string `yaml:"failureMode" json:"failureMode"`
}

// ExecutionResult contains the result of hook execution
type ExecutionResult struct {
	// HookName is the name of the hook that executed
	HookName string `json:"hookName"`

	// EventType is the event that triggered the hook
	EventType EventType `json:"eventType"`

	// Success indicates if the hook executed successfully
	Success bool `json:"success"`

	// Error message if hook failed
	Error string `json:"error,omitempty"`

	// Duration of hook execution
	Duration time.Duration `json:"duration"`

	// Output from the hook (if any)
	Output string `json:"output,omitempty"`

	// Timestamp when hook executed
	Timestamp time.Time `json:"timestamp"`
}

// HookFactory creates hooks from configuration
type HookFactory func(config *HookConfig) (Hook, error)

// DefaultTimeout is the default hook execution timeout
const DefaultTimeout = 30 * time.Second

// ValidFailureModes defines valid failure modes
var ValidFailureModes = []string{"ignore", "warn", "fail"}

// IsValidFailureMode checks if a failure mode is valid
func IsValidFailureMode(mode string) bool {
	for _, valid := range ValidFailureModes {
		if mode == valid {
			return true
		}
	}
	return false
}

// NewEvent creates a new event
func NewEvent(eventType EventType, workflowID string, data map[string]interface{}) *Event {
	return &Event{
		Type:       eventType,
		Timestamp:  time.Now(),
		WorkflowID: workflowID,
		Data:       data,
	}
}

// GetString gets a string value from event data
func (e *Event) GetString(key string) string {
	if val, ok := e.Data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt gets an int value from event data
func (e *Event) GetInt(key string) int {
	if val, ok := e.Data[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
	}
	return 0
}

// GetFloat gets a float64 value from event data
func (e *Event) GetFloat(key string) float64 {
	if val, ok := e.Data[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return 0.0
}

// GetBool gets a bool value from event data
func (e *Event) GetBool(key string) bool {
	if val, ok := e.Data[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}
