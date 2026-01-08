package auto

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/specular/internal/router"
)

// CostEstimate represents a comprehensive cost estimation for a workflow
type CostEstimate struct {
	// Component costs
	SpecGeneration float64 `json:"spec_generation"`
	PlanGeneration float64 `json:"plan_generation"`
	TaskExecution  float64 `json:"task_execution"`
	Total          float64 `json:"total"`

	// Context
	ModelUsed       string `json:"model_used"`
	GoalLength      int    `json:"goal_length"`
	EstFeatureCount int    `json:"est_feature_count"`
	EstTaskCount    int    `json:"est_task_count"`

	// Budget context (if available)
	BudgetLimit     float64 `json:"budget_limit"`
	BudgetRemaining float64 `json:"budget_remaining"`
	BudgetUsage     float64 `json:"budget_usage_percent"`
}

// EstimateWorkflowCost estimates the total cost for an auto workflow
// This provides a conservative upper-bound estimate before execution
func EstimateWorkflowCost(goal string, r *router.Router) *CostEstimate {
	estimate := &CostEstimate{
		GoalLength: len(goal),
		ModelUsed:  "claude-haiku", // Default estimation model
	}

	// Get cost per million tokens using defaults
	// Claude Haiku: ~$0.25/1M input, ~$1.25/1M output tokens (average ~$0.75/1M)
	// Auto mode uses cheap models by default
	costPerMToken := 0.75 // Default to Haiku pricing
	estimate.ModelUsed = "claude-haiku"

	if r != nil {
		// Get budget info from router
		budget := r.GetBudget()
		if budget != nil {
			estimate.BudgetLimit = budget.LimitUSD
			estimate.BudgetRemaining = budget.RemainingUSD
		}
	}

	// Estimate feature count from goal length
	// Heuristic: complex goals (longer) tend to have more features
	estimate.EstFeatureCount = estimateFeatureCount(goal)

	// Estimate task count from feature count
	// Heuristic: ~2-3 tasks per feature (spec, code, test)
	estimate.EstTaskCount = estimate.EstFeatureCount * 2

	// Calculate component costs
	estimate.SpecGeneration = EstimateSpecGenerationCost(len(goal), costPerMToken)
	estimate.PlanGeneration = EstimatePlanGenerationCost(estimate.EstFeatureCount, costPerMToken)
	estimate.TaskExecution = EstimateTaskExecutionCost(estimate.EstTaskCount, costPerMToken)

	// Calculate total
	estimate.Total = estimate.SpecGeneration + estimate.PlanGeneration + estimate.TaskExecution

	// Calculate budget usage percentage
	if estimate.BudgetLimit > 0 {
		estimate.BudgetUsage = (estimate.Total / estimate.BudgetLimit) * 100
	}

	return estimate
}

// estimateFeatureCount estimates the number of features based on goal complexity
func estimateFeatureCount(goal string) int {
	// Heuristic based on goal analysis
	featureCount := 1 // Minimum 1 feature

	// Check for explicit feature indicators
	words := strings.Fields(strings.ToLower(goal))

	// Keywords that often indicate multiple features
	featureKeywords := []string{
		"and", "with", "plus", "also", "including",
		"authentication", "api", "database", "ui", "frontend",
		"backend", "crud", "endpoints", "dashboard", "admin",
	}

	for _, word := range words {
		for _, keyword := range featureKeywords {
			if word == keyword {
				featureCount++
				break
			}
		}
	}

	// Cap at reasonable maximum for estimation
	if featureCount > 10 {
		featureCount = 10
	}

	return featureCount
}

// FormatCostEstimate returns a formatted string representation of the cost estimate
func FormatCostEstimate(est *CostEstimate) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  Cost Estimation\n")
	b.WriteString("  ===============\n")
	b.WriteString("\n")

	// Component breakdown
	b.WriteString(fmt.Sprintf("  Spec Generation:    $%.4f  (%s)\n", est.SpecGeneration, est.ModelUsed))
	b.WriteString(fmt.Sprintf("  Plan Generation:    $%.4f  (%d features x ~$%.4f)\n",
		est.PlanGeneration, est.EstFeatureCount, est.PlanGeneration/float64(max(est.EstFeatureCount, 1))))
	b.WriteString(fmt.Sprintf("  Task Execution:     $%.4f  (est. %d AI tasks)\n",
		est.TaskExecution, int(float64(est.EstTaskCount)*0.2)))
	b.WriteString("  ─────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Estimated Total:    $%.4f\n", est.Total))
	b.WriteString("\n")

	// Budget context
	if est.BudgetLimit > 0 {
		budgetWarning := ""
		if est.BudgetUsage >= 90 {
			budgetWarning = "  (HIGH)"
		} else if est.BudgetUsage >= 75 {
			budgetWarning = "  (warning)"
		}
		b.WriteString(fmt.Sprintf("  Budget:  $%.2f remaining (%.1f%% usage)%s\n",
			est.BudgetRemaining, est.BudgetUsage, budgetWarning))
	}

	b.WriteString("\n")
	b.WriteString("  Run without --estimate-cost to execute.\n")

	return b.String()
}

// max returns the maximum of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
