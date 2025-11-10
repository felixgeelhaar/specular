package auto

import (
	"fmt"

	"github.com/felixgeelhaar/specular/internal/router"
)

// BudgetThreshold defines warning thresholds for budget usage
type BudgetThreshold struct {
	Percentage float64
	Message    string
}

var defaultThresholds = []BudgetThreshold{
	{50.0, "⚠️  Budget Warning: 50% of budget used"},
	{75.0, "⚠️  Budget Warning: 75% of budget used"},
	{90.0, "⚠️  Budget Warning: 90% of budget used - approaching limit!"},
}

// CheckBudget verifies sufficient budget is available for estimated cost
func CheckBudget(budget *router.Budget, estimatedCost float64, operation string) error {
	if budget == nil {
		return nil // No budget enforcement if router not available
	}

	if estimatedCost > budget.RemainingUSD {
		return fmt.Errorf(
			"insufficient budget for %s: estimated cost $%.4f exceeds remaining budget $%.2f (limit: $%.2f)",
			operation,
			estimatedCost,
			budget.RemainingUSD,
			budget.LimitUSD,
		)
	}

	return nil
}

// CheckBudgetWithWarning checks budget and returns warning if approaching limits
func CheckBudgetWithWarning(budget *router.Budget, estimatedCost float64, operation string) (warning string, err error) {
	if budget == nil {
		return "", nil
	}

	// Check if we have enough budget
	if err := CheckBudget(budget, estimatedCost, operation); err != nil {
		return "", err
	}

	// Check if we'll cross a warning threshold after this operation
	usagePercent := (budget.SpentUSD / budget.LimitUSD) * 100
	afterUsagePercent := ((budget.SpentUSD + estimatedCost) / budget.LimitUSD) * 100

	for _, threshold := range defaultThresholds {
		// If we're about to cross this threshold
		if usagePercent < threshold.Percentage && afterUsagePercent >= threshold.Percentage {
			warning = fmt.Sprintf("%s (will be at %.1f%% after %s)", threshold.Message, afterUsagePercent, operation)
			break
		}
		// If we're already past this threshold but haven't warned yet
		if usagePercent >= threshold.Percentage {
			warning = fmt.Sprintf("%s (currently at %.1f%%)", threshold.Message, usagePercent)
			break
		}
	}

	return warning, nil
}

// GetBudgetStatus returns a formatted string showing current budget status
func GetBudgetStatus(budget *router.Budget) string {
	if budget == nil {
		return "Budget: Not available"
	}

	usagePercent := (budget.SpentUSD / budget.LimitUSD) * 100
	return fmt.Sprintf(
		"Budget: $%.4f spent / $%.2f limit (%.1f%% used, $%.2f remaining)",
		budget.SpentUSD,
		budget.LimitUSD,
		usagePercent,
		budget.RemainingUSD,
	)
}

// EstimateSpecGenerationCost estimates the cost of generating a spec from a goal
// This is a heuristic based on typical token usage
func EstimateSpecGenerationCost(goalLength int, costPerMToken float64) float64 {
	// Estimate: system prompt (~500 tokens) + goal + response (~1500 tokens)
	// Rule of thumb: 1 char ≈ 0.25 tokens
	estimatedTokens := 500 + (goalLength / 4) + 1500
	return (float64(estimatedTokens) / 1000000.0) * costPerMToken
}

// EstimatePlanGenerationCost estimates the cost of generating a plan from a spec
func EstimatePlanGenerationCost(featureCount int, costPerMToken float64) float64 {
	// Estimate: system prompt + spec context + response
	// More features = more context and more tasks
	estimatedTokens := 1000 + (featureCount * 500) + (featureCount * 300)
	return (float64(estimatedTokens) / 1000000.0) * costPerMToken
}

// EstimateTaskExecutionCost estimates the cost of executing tasks
// Note: This is for AI-powered tasks only, not Docker execution
func EstimateTaskExecutionCost(taskCount int, costPerMToken float64) float64 {
	// Most tasks don't use AI during execution (they use Docker)
	// But some might use AI for code generation
	// Conservative estimate: assume 20% of tasks might use AI
	aiTaskCount := float64(taskCount) * 0.2
	if aiTaskCount < 1 {
		aiTaskCount = 0 // If less than 1, assume no AI usage
	}
	estimatedTokens := int(aiTaskCount) * 2000 // 2000 tokens per AI task
	return (float64(estimatedTokens) / 1000000.0) * costPerMToken
}

// CheckPerTaskBudget verifies a single task doesn't exceed per-task limit
func CheckPerTaskBudget(estimatedCost float64, maxCostPerTask float64, taskID string) error {
	if estimatedCost > maxCostPerTask {
		return fmt.Errorf(
			"task %s estimated cost $%.4f exceeds per-task limit $%.2f",
			taskID,
			estimatedCost,
			maxCostPerTask,
		)
	}
	return nil
}
