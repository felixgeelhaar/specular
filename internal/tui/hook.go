package tui

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/specular/internal/hooks"
)

// Hook is a hook implementation that forwards orchestrator events to the TUI
type Hook struct {
	adapter *Adapter
	enabled bool
}

// NewHook creates a new TUI hook
func NewHook(adapter *Adapter) *Hook {
	return &Hook{
		adapter: adapter,
		enabled: true,
	}
}

// Name returns the hook name
func (h *Hook) Name() string {
	return "tui"
}

// EventTypes returns the events this hook handles
func (h *Hook) EventTypes() []hooks.EventType {
	return []hooks.EventType{
		hooks.EventWorkflowStart,
		hooks.EventWorkflowComplete,
		hooks.EventWorkflowFailed,
		hooks.EventPlanCreated,
		hooks.EventPlanApproved,
		hooks.EventPlanRejected,
		hooks.EventStepBefore,
		hooks.EventStepAfter,
		hooks.EventStepFailed,
	}
}

// Execute runs the hook for an event
func (h *Hook) Execute(ctx context.Context, event *hooks.Event) error {
	if !h.enabled || h.adapter == nil {
		return nil
	}

	switch event.Type {
	case hooks.EventStepBefore:
		// Extract step information from event data
		stepIndex, ok := event.Data["step_index"].(int)
		if !ok {
			return fmt.Errorf("missing or invalid step_index in event data")
		}

		stepName, ok := event.Data["step_name"].(string)
		if !ok {
			// Try step_id as fallback
			if stepID, ok := event.Data["step_id"].(string); ok {
				stepName = stepID
			} else {
				return fmt.Errorf("missing step_name and step_id in event data")
			}
		}

		h.adapter.NotifyStepStart(stepIndex, stepName)

	case hooks.EventStepAfter:
		// Extract step information
		stepIndex, ok := event.Data["step_index"].(int)
		if !ok {
			return fmt.Errorf("missing or invalid step_index in event data")
		}

		stepName, ok := event.Data["step_name"].(string)
		if !ok {
			if stepID, ok := event.Data["step_id"].(string); ok {
				stepName = stepID
			} else {
				return fmt.Errorf("missing step_name and step_id in event data")
			}
		}

		// Get total cost if available
		totalCost := 0.0
		if cost, ok := event.Data["total_cost"].(float64); ok {
			totalCost = cost
		}

		h.adapter.NotifyStepComplete(stepIndex, stepName, totalCost)

	case hooks.EventStepFailed:
		// Extract step information
		stepIndex, ok := event.Data["step_index"].(int)
		if !ok {
			return fmt.Errorf("missing or invalid step_index in event data")
		}

		stepName, ok := event.Data["step_name"].(string)
		if !ok {
			if stepID, ok := event.Data["step_id"].(string); ok {
				stepName = stepID
			} else {
				return fmt.Errorf("missing step_name and step_id in event data")
			}
		}

		// Get error if available
		var err error
		if errMsg, ok := event.Data["error"].(string); ok {
			err = fmt.Errorf("%s", errMsg)
		} else if errObj, ok := event.Data["error"].(error); ok {
			err = errObj
		}

		h.adapter.NotifyStepFail(stepIndex, stepName, err)

	case hooks.EventWorkflowComplete, hooks.EventWorkflowFailed:
		// Get success status
		success := event.Type == hooks.EventWorkflowComplete

		// Get total cost and duration
		totalCost := 0.0
		if cost, ok := event.Data["total_cost"].(float64); ok {
			totalCost = cost
		}

		// Get duration - use model's elapsed time as it's the most accurate
		duration := h.adapter.model.elapsed()

		h.adapter.NotifyComplete(success, totalCost, duration)

	case hooks.EventPlanCreated:
		// Plan created - we could set the action plan here if needed
		// For now, this is handled separately in the CLI

	case hooks.EventPlanApproved, hooks.EventPlanRejected:
		// Approval events are handled through the RequestApproval method
		// These events are just for notifications/hooks
	}

	return nil
}

// Enabled returns whether the hook is currently enabled
func (h *Hook) Enabled() bool {
	return h.enabled
}

// Enable enables the hook
func (h *Hook) Enable() {
	h.enabled = true
}

// Disable disables the hook
func (h *Hook) Disable() {
	h.enabled = false
}
