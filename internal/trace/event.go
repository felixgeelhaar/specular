package trace

import (
	"encoding/json"
	"time"
)

// EventType represents the type of trace event
type EventType string

const (
	// EventTypeWorkflowStart indicates workflow execution started
	EventTypeWorkflowStart EventType = "workflow_start"

	// EventTypeWorkflowComplete indicates workflow completed
	EventTypeWorkflowComplete EventType = "workflow_complete"

	// EventTypeStepStart indicates a step started
	EventTypeStepStart EventType = "step_start"

	// EventTypeStepComplete indicates a step completed
	EventTypeStepComplete EventType = "step_complete"

	// EventTypeStepFail indicates a step failed
	EventTypeStepFail EventType = "step_fail"

	// EventTypePolicyCheck indicates a policy was checked
	EventTypePolicyCheck EventType = "policy_check"

	// EventTypeApprovalRequest indicates approval was requested
	EventTypeApprovalRequest EventType = "approval_request"

	// EventTypeApprovalResponse indicates approval response received
	EventTypeApprovalResponse EventType = "approval_response"

	// EventTypeBudgetCheck indicates budget was checked
	EventTypeBudgetCheck EventType = "budget_check"

	// EventTypeError indicates an error occurred
	EventTypeError EventType = "error"

	// EventTypeWarning indicates a warning occurred
	EventTypeWarning EventType = "warning"

	// EventTypeInfo indicates informational event
	EventTypeInfo EventType = "info"
)

// Event represents a single trace event
type Event struct {
	// ID is a unique identifier for this event
	ID string `json:"id"`

	// Type is the event type
	Type EventType `json:"type"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// WorkflowID identifies the workflow this event belongs to
	WorkflowID string `json:"workflow_id"`

	// StepID identifies the step (if applicable)
	StepID string `json:"step_id,omitempty"`

	// Message is a human-readable description
	Message string `json:"message"`

	// Level indicates severity (info, warning, error)
	Level string `json:"level"`

	// Data contains event-specific structured data
	Data map[string]interface{} `json:"data,omitempty"`

	// Duration tracks how long an operation took (for start/complete pairs)
	Duration *time.Duration `json:"duration,omitempty"`

	// Error contains error details if applicable
	Error string `json:"error,omitempty"`

	// Context contains additional contextual information
	Context *EventContext `json:"context,omitempty"`
}

// EventContext provides additional context for events
type EventContext struct {
	// Goal is the user's goal
	Goal string `json:"goal,omitempty"`

	// Profile is the profile being used
	Profile string `json:"profile,omitempty"`

	// CompletedSteps is the number of completed steps
	CompletedSteps int `json:"completed_steps,omitempty"`

	// TotalSteps is the total number of steps
	TotalSteps int `json:"total_steps,omitempty"`

	// TotalCost is the accumulated cost so far
	TotalCost float64 `json:"total_cost,omitempty"`

	// ElapsedTime is the time elapsed since workflow start
	ElapsedTime time.Duration `json:"elapsed_time,omitempty"`
}

// ToJSON converts the event to JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}

// FromJSON parses an event from JSON
func FromJSON(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// NewEvent creates a new trace event with common fields populated
func NewEvent(eventType EventType, workflowID string, message string) *Event {
	return &Event{
		ID:         generateEventID(),
		Type:       eventType,
		Timestamp:  time.Now(),
		WorkflowID: workflowID,
		Message:    message,
		Level:      inferLevel(eventType),
		Data:       make(map[string]interface{}),
	}
}

// WithStepID sets the step ID
func (e *Event) WithStepID(stepID string) *Event {
	e.StepID = stepID
	return e
}

// WithData adds data to the event
func (e *Event) WithData(key string, value interface{}) *Event {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	e.Data[key] = value
	return e
}

// WithError sets the error field
func (e *Event) WithError(err error) *Event {
	if err != nil {
		e.Error = err.Error()
		e.Level = "error"
	}
	return e
}

// WithDuration sets the duration
func (e *Event) WithDuration(duration time.Duration) *Event {
	e.Duration = &duration
	return e
}

// WithContext sets the context
func (e *Event) WithContext(ctx *EventContext) *Event {
	e.Context = ctx
	return e
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return time.Now().Format("20060102150405.000000")
}

// inferLevel infers the log level from event type
func inferLevel(eventType EventType) string {
	switch eventType {
	case EventTypeError, EventTypeStepFail:
		return "error"
	case EventTypeWarning:
		return "warning"
	default:
		return "info"
	}
}

// EventBuilder provides a fluent interface for building events
type EventBuilder struct {
	event *Event
}

// NewEventBuilder creates a new event builder
func NewEventBuilder(eventType EventType, workflowID string) *EventBuilder {
	return &EventBuilder{
		event: NewEvent(eventType, workflowID, ""),
	}
}

// Message sets the message
func (b *EventBuilder) Message(msg string) *EventBuilder {
	b.event.Message = msg
	return b
}

// StepID sets the step ID
func (b *EventBuilder) StepID(stepID string) *EventBuilder {
	b.event.StepID = stepID
	return b
}

// Data adds data
func (b *EventBuilder) Data(key string, value interface{}) *EventBuilder {
	if b.event.Data == nil {
		b.event.Data = make(map[string]interface{})
	}
	b.event.Data[key] = value
	return b
}

// Error sets the error
func (b *EventBuilder) Error(err error) *EventBuilder {
	if err != nil {
		b.event.Error = err.Error()
		b.event.Level = "error"
	}
	return b
}

// Duration sets the duration
func (b *EventBuilder) Duration(duration time.Duration) *EventBuilder {
	b.event.Duration = &duration
	return b
}

// Context sets the context
func (b *EventBuilder) Context(ctx *EventContext) *EventBuilder {
	b.event.Context = ctx
	return b
}

// Build returns the constructed event
func (b *EventBuilder) Build() *Event {
	return b.event
}
