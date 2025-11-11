package autopolicy

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/specular/internal/auto"
)

// CostLimitChecker enforces cost budget constraints.
type CostLimitChecker struct {
	// MaxTotalCost is the maximum total cost allowed for the workflow.
	MaxTotalCost float64

	// MaxPerStepCost is the maximum cost allowed per individual step.
	MaxPerStepCost float64

	// EstimatedCostPerStep estimates the cost for different step types.
	EstimatedCostPerStep map[auto.StepType]float64
}

// NewCostLimitChecker creates a checker with cost limits.
func NewCostLimitChecker(maxTotal, maxPerStep float64) *CostLimitChecker {
	return &CostLimitChecker{
		MaxTotalCost:   maxTotal,
		MaxPerStepCost: maxPerStep,
		EstimatedCostPerStep: map[auto.StepType]float64{
			auto.StepTypeSpecUpdate: 0.50, // Spec generation ~$0.50
			auto.StepTypeSpecLock:   0.01, // Locking is cheap
			auto.StepTypePlanGen:    0.30, // Plan generation ~$0.30
			auto.StepTypeBuildRun:   1.00, // Build varies, conservative estimate
		},
	}
}

// CheckStep validates cost constraints.
func (c *CostLimitChecker) CheckStep(ctx context.Context, step *auto.ActionStep) (*PolicyResult, error) {
	// Extract context if available
	policyCtx, ok := ctx.Value("policy_context").(*PolicyContext)
	if !ok {
		// Without context, we can only do basic validation
		return c.checkBasicCost(step)
	}

	return c.checkCostWithContext(step, policyCtx)
}

func (c *CostLimitChecker) checkBasicCost(step *auto.ActionStep) (*PolicyResult, error) {
	estimatedCost := c.EstimatedCostPerStep[step.Type]
	if estimatedCost == 0 {
		estimatedCost = 0.50 // Default estimate
	}

	// Check per-step limit
	if c.MaxPerStepCost > 0 && estimatedCost > c.MaxPerStepCost {
		return NewDeniedResult(fmt.Sprintf(
			"estimated step cost $%.2f exceeds per-step limit $%.2f",
			estimatedCost, c.MaxPerStepCost,
		)), nil
	}

	result := NewAllowedResult()
	result.SetMetadata("estimated_cost", estimatedCost)
	return result, nil
}

func (c *CostLimitChecker) checkCostWithContext(step *auto.ActionStep, policyCtx *PolicyContext) (*PolicyResult, error) {
	estimatedCost := c.EstimatedCostPerStep[step.Type]
	if estimatedCost == 0 {
		estimatedCost = 0.50
	}

	// Check per-step limit
	if c.MaxPerStepCost > 0 && estimatedCost > c.MaxPerStepCost {
		return NewDeniedResult(fmt.Sprintf(
			"estimated step cost $%.2f exceeds per-step limit $%.2f",
			estimatedCost, c.MaxPerStepCost,
		)), nil
	}

	// Check total cost limit
	projectedTotal := policyCtx.TotalCostSoFar + estimatedCost
	if c.MaxTotalCost > 0 && projectedTotal > c.MaxTotalCost {
		return NewDeniedResult(fmt.Sprintf(
			"projected total cost $%.2f exceeds budget $%.2f (spent: $%.2f, step: $%.2f)",
			projectedTotal, c.MaxTotalCost, policyCtx.TotalCostSoFar, estimatedCost,
		)), nil
	}

	// Warning if approaching limit
	result := NewAllowedResult()
	if c.MaxTotalCost > 0 {
		remaining := c.MaxTotalCost - projectedTotal
		threshold := c.MaxTotalCost * 0.2 // Warn at 80% usage
		if remaining < threshold {
			result.AddWarning(fmt.Sprintf(
				"Approaching cost limit: $%.2f remaining of $%.2f budget",
				remaining, c.MaxTotalCost,
			))
		}
	}

	result.SetMetadata("estimated_cost", estimatedCost)
	result.SetMetadata("total_cost_so_far", policyCtx.TotalCostSoFar)
	result.SetMetadata("projected_total", projectedTotal)
	return result, nil
}

// Name returns the checker name.
func (c *CostLimitChecker) Name() string {
	return "cost_limit"
}

// TimeoutChecker enforces workflow timeout constraints.
type TimeoutChecker struct {
	// MaxDuration is the maximum allowed workflow duration.
	MaxDuration time.Duration

	// MaxStepDuration is the maximum allowed duration per step.
	MaxStepDuration time.Duration
}

// NewTimeoutChecker creates a checker with timeout limits.
func NewTimeoutChecker(maxDuration, maxStepDuration time.Duration) *TimeoutChecker {
	return &TimeoutChecker{
		MaxDuration:     maxDuration,
		MaxStepDuration: maxStepDuration,
	}
}

// CheckStep validates timeout constraints.
func (t *TimeoutChecker) CheckStep(ctx context.Context, step *auto.ActionStep) (*PolicyResult, error) {
	// Check context deadline
	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) < t.MaxStepDuration {
			return NewDeniedResult(fmt.Sprintf(
				"insufficient time remaining: %s < %s required",
				time.Until(deadline), t.MaxStepDuration,
			)), nil
		}
	}

	// Extract context if available
	policyCtx, ok := ctx.Value("policy_context").(*PolicyContext)
	if !ok {
		return NewAllowedResult(), nil
	}

	// Check total elapsed time
	elapsed := policyCtx.ElapsedTime()
	if t.MaxDuration > 0 && elapsed > t.MaxDuration {
		return NewDeniedResult(fmt.Sprintf(
			"workflow timeout exceeded: %s > %s limit",
			elapsed, t.MaxDuration,
		)), nil
	}

	// Warning if approaching timeout
	result := NewAllowedResult()
	if t.MaxDuration > 0 {
		remaining := t.MaxDuration - elapsed
		threshold := t.MaxDuration / 5 // Warn at 80% usage
		if remaining < threshold {
			result.AddWarning(fmt.Sprintf(
				"Approaching timeout: %s remaining of %s limit",
				remaining, t.MaxDuration,
			))
		}
	}

	result.SetMetadata("elapsed_time", elapsed.String())
	return result, nil
}

// Name returns the checker name.
func (t *TimeoutChecker) Name() string {
	return "timeout"
}

// StepTypeChecker enforces allowed/blocked step types.
type StepTypeChecker struct {
	// AllowedTypes lists the only step types permitted (whitelist).
	// Empty list means all types are allowed unless blocked.
	AllowedTypes []auto.StepType

	// BlockedTypes lists step types that are forbidden (blacklist).
	BlockedTypes []auto.StepType
}

// NewStepTypeChecker creates a checker with type constraints.
func NewStepTypeChecker(allowed, blocked []auto.StepType) *StepTypeChecker {
	return &StepTypeChecker{
		AllowedTypes: allowed,
		BlockedTypes: blocked,
	}
}

// CheckStep validates step type constraints.
func (s *StepTypeChecker) CheckStep(ctx context.Context, step *auto.ActionStep) (*PolicyResult, error) {
	// Check blacklist first
	for _, blocked := range s.BlockedTypes {
		if step.Type == blocked {
			return NewDeniedResult(fmt.Sprintf(
				"step type %q is blocked by policy",
				step.Type,
			)), nil
		}
	}

	// Check whitelist if configured
	if len(s.AllowedTypes) > 0 {
		allowed := false
		for _, allowedType := range s.AllowedTypes {
			if step.Type == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return NewDeniedResult(fmt.Sprintf(
				"step type %q is not in allowed list",
				step.Type,
			)), nil
		}
	}

	return NewAllowedResult(), nil
}

// Name returns the checker name.
func (s *StepTypeChecker) Name() string {
	return "step_type"
}

// MaxStepsChecker enforces maximum step count.
type MaxStepsChecker struct {
	// MaxSteps is the maximum number of steps allowed in total.
	MaxSteps int
}

// NewMaxStepsChecker creates a checker with step count limit.
func NewMaxStepsChecker(maxSteps int) *MaxStepsChecker {
	return &MaxStepsChecker{
		MaxSteps: maxSteps,
	}
}

// CheckStep validates step count constraints.
func (m *MaxStepsChecker) CheckStep(ctx context.Context, step *auto.ActionStep) (*PolicyResult, error) {
	policyCtx, ok := ctx.Value("policy_context").(*PolicyContext)
	if !ok {
		return NewAllowedResult(), nil
	}

	totalSteps := policyCtx.CompletedSteps + 1 // +1 for current step
	if m.MaxSteps > 0 && totalSteps > m.MaxSteps {
		return NewDeniedResult(fmt.Sprintf(
			"maximum step count exceeded: %d > %d limit",
			totalSteps, m.MaxSteps,
		)), nil
	}

	// Warning if approaching limit
	result := NewAllowedResult()
	if m.MaxSteps > 0 {
		remaining := m.MaxSteps - totalSteps
		if remaining <= 2 {
			result.AddWarning(fmt.Sprintf(
				"Approaching step limit: %d steps remaining of %d maximum",
				remaining, m.MaxSteps,
			))
		}
	}

	result.SetMetadata("completed_steps", policyCtx.CompletedSteps)
	result.SetMetadata("total_steps", totalSteps)
	return result, nil
}

// Name returns the checker name.
func (m *MaxStepsChecker) Name() string {
	return "max_steps"
}

// MaxRetriesChecker enforces maximum retry count for failed steps.
type MaxRetriesChecker struct {
	// MaxRetries is the maximum number of retries allowed.
	MaxRetries int

	// CurrentRetries tracks retry count per step ID.
	CurrentRetries map[string]int
}

// NewMaxRetriesChecker creates a checker with retry limit.
func NewMaxRetriesChecker(maxRetries int) *MaxRetriesChecker {
	return &MaxRetriesChecker{
		MaxRetries:     maxRetries,
		CurrentRetries: make(map[string]int),
	}
}

// CheckStep validates retry constraints.
func (m *MaxRetriesChecker) CheckStep(ctx context.Context, step *auto.ActionStep) (*PolicyResult, error) {
	retries := m.CurrentRetries[step.ID]
	if m.MaxRetries > 0 && retries >= m.MaxRetries {
		return NewDeniedResult(fmt.Sprintf(
			"maximum retry count exceeded for step %s: %d >= %d limit",
			step.ID, retries, m.MaxRetries,
		)), nil
	}

	result := NewAllowedResult()
	result.SetMetadata("retry_count", retries)
	return result, nil
}

// RecordRetry records a retry attempt for a step.
func (m *MaxRetriesChecker) RecordRetry(stepID string) {
	m.CurrentRetries[stepID]++
}

// Name returns the checker name.
func (m *MaxRetriesChecker) Name() string {
	return "max_retries"
}
