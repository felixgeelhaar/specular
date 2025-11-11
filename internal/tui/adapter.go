package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/felixgeelhaar/specular/internal/auto"
	"github.com/felixgeelhaar/specular/internal/plan"
)

// Adapter bridges between the orchestrator and the TUI
type Adapter struct {
	program *tea.Program
	model   *Model
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewAdapter creates a new TUI adapter
func NewAdapter(goal, profile string) *Adapter {
	model := NewModel(goal, profile)

	return &Adapter{
		model: &model,
	}
}

// Start starts the TUI program
func (a *Adapter) Start() error {
	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.program = tea.NewProgram(*a.model)

	// Start the TUI in a goroutine
	go func() {
		if _, err := a.program.Run(); err != nil {
			fmt.Printf("TUI error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the TUI program
func (a *Adapter) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	if a.program != nil {
		a.program.Quit()
	}
}

// SetActionPlan sets the action plan in the TUI
func (a *Adapter) SetActionPlan(plan *auto.ActionPlan) {
	if a.model != nil {
		a.model.SetActionPlan(plan)
	}
}

// SetAutoOutput sets the auto output in the TUI
func (a *Adapter) SetAutoOutput(output *auto.AutoOutput) {
	if a.model != nil {
		a.model.SetAutoOutput(output)
	}
}

// NotifyStepStart notifies the TUI that a step has started
func (a *Adapter) NotifyStepStart(stepIndex int, stepName string) {
	if a.program != nil {
		a.program.Send(StepStartMsg{
			StepIndex: stepIndex,
			StepName:  stepName,
		})
	}
}

// NotifyStepComplete notifies the TUI that a step has completed
func (a *Adapter) NotifyStepComplete(stepIndex int, stepName string, totalCost float64) {
	if a.program != nil {
		a.program.Send(StepCompleteMsg{
			StepIndex: stepIndex,
			StepName:  stepName,
			TotalCost: totalCost,
		})
	}
}

// NotifyStepFail notifies the TUI that a step has failed
func (a *Adapter) NotifyStepFail(stepIndex int, stepName string, err error) {
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
	}
}

// RequestApproval requests user approval for the plan
// Returns true if approved, false if rejected
func (a *Adapter) RequestApproval(execPlan *plan.Plan) (bool, error) {
	if a.program == nil {
		return false, fmt.Errorf("TUI not started")
	}

	// Build plan summary
	summary := fmt.Sprintf("Plan: %d tasks\n\n", len(execPlan.Tasks))
	for i, task := range execPlan.Tasks {
		if i < 10 { // Limit to first 10 tasks
			summary += fmt.Sprintf("%d. %s\n", i+1, string(task.ID))
		}
	}
	if len(execPlan.Tasks) > 10 {
		summary += fmt.Sprintf("... and %d more tasks\n", len(execPlan.Tasks)-10)
	}

	// Send approval request
	responseChan := make(chan bool, 1)

	// Create a custom message handler
	a.program.Send(ApprovalRequestMsg{
		PlanSummary: summary,
	})

	// Wait for response (with timeout)
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()

	select {
	case <-ctx.Done():
		return false, fmt.Errorf("approval timeout")
	case approved := <-responseChan:
		return approved, nil
	}
}

// NotifyComplete notifies the TUI that the workflow has completed
func (a *Adapter) NotifyComplete(success bool, totalCost float64, duration time.Duration) {
	if a.program != nil {
		a.program.Send(WorkflowCompleteMsg{
			Success:   success,
			TotalCost: totalCost,
			Duration:  duration,
		})
	}

	// Wait a bit for the user to see the completion message
	time.Sleep(2 * time.Second)
}
