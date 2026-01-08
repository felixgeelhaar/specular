package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/felixgeelhaar/specular/internal/auto"
	"github.com/felixgeelhaar/specular/internal/plan"
)

// DashboardAdapter bridges between the orchestrator and the Dashboard TUI
type DashboardAdapter struct {
	program *tea.Program
	model   *DashboardModel
	ctx     context.Context
	cancel  context.CancelFunc

	// Budget configuration
	budgetLimit float64
}

// NewDashboardAdapter creates a new dashboard TUI adapter
func NewDashboardAdapter(goal, profile string, budgetLimit float64) *DashboardAdapter {
	model := NewDashboardModel(goal, profile)
	model.SetBudgetLimit(budgetLimit)

	return &DashboardAdapter{
		model:       &model,
		budgetLimit: budgetLimit,
	}
}

// Start starts the dashboard TUI program
func (a *DashboardAdapter) Start() error {
	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.program = tea.NewProgram(*a.model, tea.WithAltScreen())

	// Start the TUI in a goroutine
	go func() {
		if _, err := a.program.Run(); err != nil {
			fmt.Printf("Dashboard TUI error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the dashboard TUI program
func (a *DashboardAdapter) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	if a.program != nil {
		a.program.Quit()
	}
}

// SetActionPlan sets the action plan in the dashboard
func (a *DashboardAdapter) SetActionPlan(plan *auto.ActionPlan) {
	if a.model != nil {
		a.model.SetActionPlan(plan)
	}
}

// SetAutoOutput sets the auto output in the dashboard
func (a *DashboardAdapter) SetAutoOutput(output *auto.AutoOutput) {
	if a.model != nil {
		a.model.SetAutoOutput(output)
	}
}

// NotifyStepStart notifies the dashboard that a step has started
func (a *DashboardAdapter) NotifyStepStart(stepIndex int, stepName string) {
	if a.program != nil {
		a.program.Send(StepStartMsg{
			StepIndex: stepIndex,
			StepName:  stepName,
		})

		// Also add a log entry
		a.AddLogEntry(LogLevelInfo, fmt.Sprintf("Starting: %s", stepName), stepName)
	}
}

// NotifyStepComplete notifies the dashboard that a step has completed
func (a *DashboardAdapter) NotifyStepComplete(stepIndex int, stepName string, totalCost float64) {
	if a.program != nil {
		a.program.Send(StepCompleteMsg{
			StepIndex: stepIndex,
			StepName:  stepName,
			TotalCost: totalCost,
		})

		// Update budget
		a.NotifyBudgetUpdate(totalCost)

		// Add log entry
		a.AddLogEntry(LogLevelInfo, fmt.Sprintf("Completed: %s", stepName), stepName)
	}
}

// NotifyStepFail notifies the dashboard that a step has failed
func (a *DashboardAdapter) NotifyStepFail(stepIndex int, stepName string, err error) {
	if a.program != nil {
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}
		a.program.Send(StepFailMsg{
			StepIndex: stepIndex,
			StepName:  stepName,
			Error:     errorMsg,
		})

		// Add error log entry
		a.AddLogEntry(LogLevelError, fmt.Sprintf("Failed: %s - %s", stepName, errorMsg), stepName)
	}
}

// NotifyBudgetUpdate updates the budget display
func (a *DashboardAdapter) NotifyBudgetUpdate(spent float64) {
	if a.program != nil {
		remaining := a.budgetLimit - spent
		warning := ""

		// Calculate percentage and set warnings
		percentage := (spent / a.budgetLimit) * 100
		if percentage >= 90 {
			warning = "Budget critical: >90% spent"
		} else if percentage >= 75 {
			warning = "Budget warning: >75% spent"
		}

		a.program.Send(BudgetUpdateMsg{
			Spent:     spent,
			Remaining: remaining,
			Limit:     a.budgetLimit,
			Warning:   warning,
		})

		// Log budget warnings
		if warning != "" && percentage >= 90 {
			a.AddLogEntry(LogLevelError, warning, "budget")
		} else if warning != "" {
			a.AddLogEntry(LogLevelWarn, warning, "budget")
		}
	}
}

// NotifyAPICall records an API call for metrics
func (a *DashboardAdapter) NotifyAPICall(callCount int) {
	if a.program != nil {
		a.program.Send(APIMetricsMsg{
			CallCount: callCount,
		})
	}
}

// AddLogEntry adds a log entry to the dashboard
func (a *DashboardAdapter) AddLogEntry(level LogLevel, message, stepID string) {
	if a.program != nil {
		a.program.Send(LogEntryMsg{
			Level:   level,
			Message: message,
			StepID:  stepID,
		})
	}
}

// RequestApproval requests user approval for the plan
func (a *DashboardAdapter) RequestApproval(execPlan *plan.Plan) (bool, error) {
	if a.program == nil {
		return false, fmt.Errorf("dashboard TUI not started")
	}

	// Build plan summary
	summary := fmt.Sprintf("Plan: %d tasks\n\n", len(execPlan.Tasks))
	for i, task := range execPlan.Tasks {
		if i < 10 {
			summary += fmt.Sprintf("%d. %s\n", i+1, string(task.ID))
		}
	}
	if len(execPlan.Tasks) > 10 {
		summary += fmt.Sprintf("... and %d more tasks\n", len(execPlan.Tasks)-10)
	}

	// Send approval request
	a.program.Send(ApprovalRequestMsg{
		PlanSummary: summary,
	})

	// Add log entry
	a.AddLogEntry(LogLevelInfo, "Awaiting plan approval...", "approval")

	// Wait for response (with timeout)
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()

	// For now, return a placeholder - actual implementation would use channels
	<-ctx.Done()
	return false, fmt.Errorf("approval timeout")
}

// NotifyComplete notifies the dashboard that the workflow has completed
func (a *DashboardAdapter) NotifyComplete(success bool, totalCost float64, duration time.Duration) {
	if a.program != nil {
		a.program.Send(WorkflowCompleteMsg{
			Success:   success,
			TotalCost: totalCost,
			Duration:  duration,
		})

		// Add completion log entry
		if success {
			a.AddLogEntry(LogLevelInfo, fmt.Sprintf("Workflow completed successfully (cost: $%.4f)", totalCost), "workflow")
		} else {
			a.AddLogEntry(LogLevelError, "Workflow failed", "workflow")
		}
	}

	// Wait for user to see the completion message
	time.Sleep(2 * time.Second)
}

// GetModel returns the underlying dashboard model (for testing)
func (a *DashboardAdapter) GetModel() *DashboardModel {
	return a.model
}
