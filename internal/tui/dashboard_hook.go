package tui

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/specular/internal/hooks"
)

// DashboardHook is a hook implementation that forwards orchestrator events to the Dashboard TUI
type DashboardHook struct {
	adapter  *DashboardAdapter
	enabled  bool
	apiCalls int // Track total API calls
}

// NewDashboardHook creates a new dashboard TUI hook
func NewDashboardHook(adapter *DashboardAdapter) *DashboardHook {
	return &DashboardHook{
		adapter: adapter,
		enabled: true,
	}
}

// Name returns the hook name
func (h *DashboardHook) Name() string {
	return "dashboard"
}

// EventTypes returns the events this hook handles
func (h *DashboardHook) EventTypes() []hooks.EventType {
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
		hooks.EventPolicyViolation,
		hooks.EventDriftDetected,
	}
}

// Execute runs the hook for an event
func (h *DashboardHook) Execute(ctx context.Context, event *hooks.Event) error {
	if !h.enabled || h.adapter == nil {
		return nil
	}

	switch event.Type {
	case hooks.EventWorkflowStart:
		// Log workflow start
		goal := event.GetString("goal")
		if goal == "" {
			goal = "Starting workflow..."
		}
		h.adapter.AddLogEntry(LogLevelInfo, fmt.Sprintf("Workflow started: %s", truncateText(goal, 60)), "workflow")

	case hooks.EventStepBefore:
		// Extract step information from event data
		stepIndex := event.GetInt("step_index")
		stepName := event.GetString("step_name")
		if stepName == "" {
			stepName = event.GetString("step_id")
		}
		if stepName == "" {
			return fmt.Errorf("missing step_name and step_id in event data")
		}

		// Increment API call counter (each step typically involves API calls)
		h.apiCalls++
		h.adapter.NotifyAPICall(h.apiCalls)

		h.adapter.NotifyStepStart(stepIndex, stepName)

	case hooks.EventStepAfter:
		// Extract step information
		stepIndex := event.GetInt("step_index")
		stepName := event.GetString("step_name")
		if stepName == "" {
			stepName = event.GetString("step_id")
		}
		if stepName == "" {
			return fmt.Errorf("missing step_name and step_id in event data")
		}

		// Get total cost if available
		totalCost := event.GetFloat("total_cost")

		// Get step cost for logging
		stepCost := event.GetFloat("step_cost")
		if stepCost > 0 {
			h.adapter.AddLogEntry(LogLevelDebug, fmt.Sprintf("Step cost: $%.4f", stepCost), stepName)
		}

		// Increment API call counter for completion
		h.apiCalls++
		h.adapter.NotifyAPICall(h.apiCalls)

		h.adapter.NotifyStepComplete(stepIndex, stepName, totalCost)

	case hooks.EventStepFailed:
		// Extract step information
		stepIndex := event.GetInt("step_index")
		stepName := event.GetString("step_name")
		if stepName == "" {
			stepName = event.GetString("step_id")
		}
		if stepName == "" {
			return fmt.Errorf("missing step_name and step_id in event data")
		}

		// Get error if available
		var err error
		if errMsg := event.GetString("error"); errMsg != "" {
			err = fmt.Errorf("%s", errMsg)
		}

		// Get retry info
		attempt := event.GetInt("attempt")
		maxRetries := event.GetInt("max_retries")
		if attempt > 0 && maxRetries > 0 {
			h.adapter.AddLogEntry(LogLevelWarn, fmt.Sprintf("Retry %d/%d for: %s", attempt, maxRetries, stepName), stepName)
		}

		h.adapter.NotifyStepFail(stepIndex, stepName, err)

	case hooks.EventWorkflowComplete:
		// Get success status and metrics
		totalCost := event.GetFloat("total_cost")

		// Get duration from model's elapsed time
		duration := h.adapter.model.elapsed()

		h.adapter.NotifyComplete(true, totalCost, duration)

	case hooks.EventWorkflowFailed:
		// Get error details
		errMsg := event.GetString("error")
		totalCost := event.GetFloat("total_cost")

		h.adapter.AddLogEntry(LogLevelError, fmt.Sprintf("Workflow failed: %s", errMsg), "workflow")

		duration := h.adapter.model.elapsed()
		h.adapter.NotifyComplete(false, totalCost, duration)

	case hooks.EventPlanCreated:
		// Log plan creation
		taskCount := event.GetInt("task_count")
		h.adapter.AddLogEntry(LogLevelInfo, fmt.Sprintf("Plan created with %d tasks", taskCount), "plan")

	case hooks.EventPlanApproved:
		h.adapter.AddLogEntry(LogLevelInfo, "Plan approved, starting execution", "plan")

	case hooks.EventPlanRejected:
		reason := event.GetString("reason")
		h.adapter.AddLogEntry(LogLevelWarn, fmt.Sprintf("Plan rejected: %s", reason), "plan")

	case hooks.EventPolicyViolation:
		// Log policy violations
		policy := event.GetString("policy")
		reason := event.GetString("reason")
		h.adapter.AddLogEntry(LogLevelError, fmt.Sprintf("Policy violation [%s]: %s", policy, reason), "policy")

	case hooks.EventDriftDetected:
		// Log drift detection
		driftType := event.GetString("drift_type")
		description := event.GetString("description")
		h.adapter.AddLogEntry(LogLevelWarn, fmt.Sprintf("Drift detected [%s]: %s", driftType, description), "drift")
	}

	return nil
}

// Enabled returns whether the hook is currently enabled
func (h *DashboardHook) Enabled() bool {
	return h.enabled
}

// Enable enables the hook
func (h *DashboardHook) Enable() {
	h.enabled = true
}

// Disable disables the hook
func (h *DashboardHook) Disable() {
	h.enabled = false
}

// GetAPICallCount returns the current API call count
func (h *DashboardHook) GetAPICallCount() int {
	return h.apiCalls
}
