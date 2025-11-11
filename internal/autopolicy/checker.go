package autopolicy

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/specular/internal/auto"
)

// PolicyChecker defines the interface for policy enforcement.
// Implementations can check various constraints before step execution.
type PolicyChecker interface {
	// CheckStep validates whether a step is allowed to execute.
	// Returns a PolicyResult indicating whether the step is allowed
	// and any warnings or reasons for denial.
	CheckStep(ctx context.Context, step *auto.ActionStep) (*PolicyResult, error)

	// Name returns the name of this policy checker for logging.
	Name() string
}

// PolicyResult contains the result of a policy check.
type PolicyResult struct {
	// Allowed indicates whether the step is permitted to execute.
	Allowed bool

	// Reason provides explanation when step is not allowed.
	// This should be a clear, actionable message for the user.
	Reason string

	// Warnings contains non-fatal policy warnings.
	// The step may proceed but these should be logged.
	Warnings []string

	// Metadata contains additional policy-specific information.
	Metadata map[string]interface{}
}

// NewAllowedResult creates a PolicyResult that allows execution.
func NewAllowedResult() *PolicyResult {
	return &PolicyResult{
		Allowed:  true,
		Warnings: []string{},
		Metadata: make(map[string]interface{}),
	}
}

// NewDeniedResult creates a PolicyResult that denies execution.
func NewDeniedResult(reason string) *PolicyResult {
	return &PolicyResult{
		Allowed:  false,
		Reason:   reason,
		Warnings: []string{},
		Metadata: make(map[string]interface{}),
	}
}

// AddWarning adds a warning to the policy result.
func (r *PolicyResult) AddWarning(warning string) {
	r.Warnings = append(r.Warnings, warning)
}

// SetMetadata sets a metadata key-value pair.
func (r *PolicyResult) SetMetadata(key string, value interface{}) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	r.Metadata[key] = value
}

// CompositeChecker runs multiple policy checkers in sequence.
// All checkers must pass for the step to be allowed.
type CompositeChecker struct {
	checkers []PolicyChecker
}

// NewCompositeChecker creates a checker that runs multiple policies.
func NewCompositeChecker(checkers ...PolicyChecker) *CompositeChecker {
	return &CompositeChecker{
		checkers: checkers,
	}
}

// CheckStep runs all registered checkers and combines results.
// Returns denied if any checker denies, otherwise allowed with all warnings.
func (c *CompositeChecker) CheckStep(ctx context.Context, step *auto.ActionStep) (*PolicyResult, error) {
	result := NewAllowedResult()

	for _, checker := range c.checkers {
		checkResult, err := checker.CheckStep(ctx, step)
		if err != nil {
			return nil, fmt.Errorf("policy check %s failed: %w", checker.Name(), err)
		}

		// Collect warnings from all checkers
		result.Warnings = append(result.Warnings, checkResult.Warnings...)

		// If any checker denies, deny the entire step
		if !checkResult.Allowed {
			return NewDeniedResult(fmt.Sprintf("[%s] %s", checker.Name(), checkResult.Reason)), nil
		}

		// Merge metadata
		for k, v := range checkResult.Metadata {
			result.SetMetadata(k, v)
		}
	}

	return result, nil
}

// Name returns the composite checker name.
func (c *CompositeChecker) Name() string {
	return "composite"
}

// PolicyContext provides contextual information for policy checks.
type PolicyContext struct {
	// CurrentStep is the step being evaluated.
	CurrentStep *auto.ActionStep

	// Plan is the full action plan for context.
	Plan *auto.ActionPlan

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
func NewPolicyContext(step *auto.ActionStep, plan *auto.ActionPlan, stepIndex int) *PolicyContext {
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
