package auto

import (
	"context"
	"time"
)

// PolicyChecker defines the interface for policy enforcement.
// This interface allows the orchestrator to check policies without
// depending on the autopolicy package (avoiding import cycles).
type PolicyChecker interface {
	// CheckStep validates whether a step is allowed to execute.
	// The context may contain a PolicyContext value with execution state.
	CheckStep(ctx context.Context, step *ActionStep) (*PolicyResult, error)

	// Name returns the name of this policy checker for logging.
	Name() string
}

// PolicyResult contains the result of a policy check.
type PolicyResult struct {
	// Allowed indicates whether the step is permitted to execute.
	Allowed bool

	// Reason provides explanation when step is not allowed.
	Reason string

	// Warnings contains non-fatal policy warnings.
	Warnings []string

	// Metadata contains additional policy-specific information.
	Metadata map[string]interface{}
}

// PolicyContext provides contextual information for policy checks.
// This is passed via context.Context to avoid import cycles.
type PolicyContext struct {
	// CurrentStep is the step being evaluated.
	CurrentStep *ActionStep

	// Plan is the full action plan for context.
	Plan *ActionPlan

	// StepIndex is the index of the current step in the plan.
	StepIndex int

	// TotalCostSoFar is the accumulated cost from previous steps.
	TotalCostSoFar float64

	// ExecutionStartTime is when the workflow started.
	ExecutionStartTime time.Time

	// CompletedSteps is the number of steps completed so far.
	CompletedSteps int

	// FailedSteps is the number of steps that failed.
	FailedSteps int
}

// NewPolicyContext creates a policy context for step evaluation.
func NewPolicyContext(step *ActionStep, plan *ActionPlan, stepIndex int) *PolicyContext {
	return &PolicyContext{
		CurrentStep:        step,
		Plan:               plan,
		StepIndex:          stepIndex,
		TotalCostSoFar:     0,
		ExecutionStartTime: time.Now(),
		CompletedSteps:     0,
		FailedSteps:        0,
	}
}

// ElapsedTime returns the time elapsed since execution started.
func (c *PolicyContext) ElapsedTime() time.Duration {
	return time.Since(c.ExecutionStartTime)
}

// RemainingSteps returns the number of pending steps.
func (c *PolicyContext) RemainingSteps() int {
	if c.Plan == nil {
		return 0
	}
	return len(c.Plan.Steps) - c.CompletedSteps - c.FailedSteps - 1 // -1 for current step
}
