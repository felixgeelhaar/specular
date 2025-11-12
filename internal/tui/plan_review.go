package tui

import (
	"github.com/felixgeelhaar/specular/internal/plan"
)

// PlanReviewResult holds the result of a plan review session
type PlanReviewResult struct {
	Approved bool
	Reason   string
}

// RunPlanReview launches an interactive TUI for reviewing an execution plan
// This is a stub implementation that will be enhanced in future iterations
func RunPlanReview(p *plan.Plan) (*PlanReviewResult, error) {
	// TODO: Implement full TUI for plan review
	// For now, return auto-approved result
	return &PlanReviewResult{
		Approved: true,
		Reason:   "",
	}, nil
}
