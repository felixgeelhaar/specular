package autopolicy

import (
	"context"
	"time"

	"github.com/felixgeelhaar/specular/internal/auto"
	"github.com/felixgeelhaar/specular/internal/profiles"
)

// NewCheckerFromProfile creates a composite policy checker from a profile.
// This integrates the profile system with per-step policy enforcement.
func NewCheckerFromProfile(profile *profiles.Profile) PolicyChecker {
	var checkers []PolicyChecker

	// Add cost limit checker
	if profile.Safety.MaxCostUSD > 0 || profile.Safety.MaxCostPerTask > 0 {
		checkers = append(checkers,
			NewCostLimitChecker(profile.Safety.MaxCostUSD, profile.Safety.MaxCostPerTask),
		)
	}

	// Add timeout checker
	if profile.Safety.Timeout > 0 {
		// Per-step timeout can be estimated as total/max_steps
		perStepTimeout := profile.Safety.Timeout
		if profile.Safety.MaxSteps > 0 {
			perStepTimeout = profile.Safety.Timeout / time.Duration(profile.Safety.MaxSteps)
		}
		checkers = append(checkers,
			NewTimeoutChecker(profile.Safety.Timeout, perStepTimeout),
		)
	}

	// Add step type checker
	if len(profile.Safety.AllowedStepTypes) > 0 || len(profile.Safety.BlockedStepTypes) > 0 {
		// Convert string slices to StepType slices
		allowed := make([]auto.StepType, len(profile.Safety.AllowedStepTypes))
		for i, t := range profile.Safety.AllowedStepTypes {
			allowed[i] = auto.StepType(t)
		}

		blocked := make([]auto.StepType, len(profile.Safety.BlockedStepTypes))
		for i, t := range profile.Safety.BlockedStepTypes {
			blocked[i] = auto.StepType(t)
		}

		checkers = append(checkers,
			NewStepTypeChecker(allowed, blocked),
		)
	}

	// Add max steps checker
	if profile.Safety.MaxSteps > 0 {
		checkers = append(checkers,
			NewMaxStepsChecker(profile.Safety.MaxSteps),
		)
	}

	// Add max retries checker
	if profile.Safety.MaxRetries > 0 {
		checkers = append(checkers,
			NewMaxRetriesChecker(profile.Safety.MaxRetries),
		)
	}

	// If no checkers configured, return a permissive checker
	if len(checkers) == 0 {
		return &PermissiveChecker{}
	}

	return NewCompositeChecker(checkers...)
}

// PermissiveChecker allows all steps (used when no policies are configured).
type PermissiveChecker struct{}

// CheckStep always allows execution.
func (p *PermissiveChecker) CheckStep(ctx context.Context, step *auto.ActionStep) (*PolicyResult, error) {
	return NewAllowedResult(), nil
}

// Name returns the checker name.
func (p *PermissiveChecker) Name() string {
	return "permissive"
}
