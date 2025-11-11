package auto

import (
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/router"
)

func TestCheckBudget_NilBudget(t *testing.T) {
	err := CheckBudget(nil, 1.0, "test operation")
	if err != nil {
		t.Errorf("CheckBudget with nil budget should return nil, got %v", err)
	}
}

func TestCheckBudget_SufficientBudget(t *testing.T) {
	budget := &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     5.0,
		RemainingUSD: 5.0,
	}

	err := CheckBudget(budget, 2.0, "test operation")
	if err != nil {
		t.Errorf("CheckBudget with sufficient budget should return nil, got %v", err)
	}
}

func TestCheckBudget_InsufficientBudget(t *testing.T) {
	budget := &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     9.0,
		RemainingUSD: 1.0,
	}

	err := CheckBudget(budget, 2.0, "test operation")
	if err == nil {
		t.Error("CheckBudget with insufficient budget should return error")
	}

	expectedMsg := "insufficient budget for test operation"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Error message should contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestCheckBudget_ExactBudget(t *testing.T) {
	budget := &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     8.0,
		RemainingUSD: 2.0,
	}

	err := CheckBudget(budget, 2.0, "test operation")
	if err != nil {
		t.Errorf("CheckBudget with exact budget should return nil, got %v", err)
	}
}

func TestCheckBudgetWithWarning_NilBudget(t *testing.T) {
	warning, err := CheckBudgetWithWarning(nil, 1.0, "test operation")
	if err != nil {
		t.Errorf("CheckBudgetWithWarning with nil budget should return nil error, got %v", err)
	}
	if warning != "" {
		t.Errorf("CheckBudgetWithWarning with nil budget should return empty warning, got %s", warning)
	}
}

func TestCheckBudgetWithWarning_InsufficientBudget(t *testing.T) {
	budget := &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     9.0,
		RemainingUSD: 1.0,
	}

	warning, err := CheckBudgetWithWarning(budget, 2.0, "test operation")
	if err == nil {
		t.Error("CheckBudgetWithWarning with insufficient budget should return error")
	}
	if warning != "" {
		t.Errorf("Warning should be empty when error occurs, got %s", warning)
	}
}

func TestCheckBudgetWithWarning_50PercentThreshold(t *testing.T) {
	budget := &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     4.0, // 40% used
		RemainingUSD: 6.0,
	}

	// This operation will push us to 50%
	warning, err := CheckBudgetWithWarning(budget, 1.0, "test operation")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if warning == "" {
		t.Error("Should return warning when crossing 50% threshold")
	}
	if !strings.Contains(warning, "50%") {
		t.Errorf("Warning should mention 50%%, got: %s", warning)
	}
}

func TestCheckBudgetWithWarning_75PercentThreshold(t *testing.T) {
	budget := &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     7.0, // 70% used, past 50% threshold
		RemainingUSD: 3.0,
	}

	// This operation will push us to 75%
	warning, err := CheckBudgetWithWarning(budget, 0.5, "test operation")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if warning == "" {
		t.Error("Should return warning when past threshold")
	}
	// When already past 50%, it warns about 50%
	if !strings.Contains(warning, "Budget Warning") {
		t.Errorf("Warning should contain budget warning, got: %s", warning)
	}
}

func TestCheckBudgetWithWarning_90PercentThreshold(t *testing.T) {
	budget := &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     8.5, // 85% used, past 50% and 75% thresholds
		RemainingUSD: 1.5,
	}

	// This operation will push us to 90%
	warning, err := CheckBudgetWithWarning(budget, 0.5, "test operation")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if warning == "" {
		t.Error("Should return warning when past threshold")
	}
	// When already past lower thresholds, it warns about the first one
	if !strings.Contains(warning, "Budget Warning") {
		t.Errorf("Warning should contain budget warning, got: %s", warning)
	}
}

func TestCheckBudgetWithWarning_AlreadyPastThreshold(t *testing.T) {
	budget := &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     6.0, // 60% used, already past 50%
		RemainingUSD: 4.0,
	}

	// Small operation that doesn't cross any new thresholds
	warning, err := CheckBudgetWithWarning(budget, 0.5, "test operation")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// Should still warn about current usage
	if warning == "" {
		t.Error("Should return warning when already past threshold")
	}
}

func TestCheckBudgetWithWarning_NoWarning(t *testing.T) {
	budget := &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     2.0, // 20% used
		RemainingUSD: 8.0,
	}

	// Small operation that doesn't cross any thresholds
	warning, err := CheckBudgetWithWarning(budget, 0.5, "test operation")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if warning != "" {
		t.Errorf("Should not return warning when far from thresholds, got: %s", warning)
	}
}

func TestGetBudgetStatus_NilBudget(t *testing.T) {
	status := GetBudgetStatus(nil)
	expected := "Budget: Not available"
	if status != expected {
		t.Errorf("GetBudgetStatus(nil) = %s, want %s", status, expected)
	}
}

func TestGetBudgetStatus_ValidBudget(t *testing.T) {
	budget := &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     5.0,
		RemainingUSD: 5.0,
	}

	status := GetBudgetStatus(budget)
	if !strings.Contains(status, "5.0000") { // spent
		t.Errorf("Status should contain spent amount, got: %s", status)
	}
	if !strings.Contains(status, "10.00") { // limit
		t.Errorf("Status should contain limit amount, got: %s", status)
	}
	if !strings.Contains(status, "50.0%") { // usage percent
		t.Errorf("Status should contain usage percentage, got: %s", status)
	}
	if !strings.Contains(status, "5.00") { // remaining
		t.Errorf("Status should contain remaining amount, got: %s", status)
	}
}

func TestEstimateSpecGenerationCost(t *testing.T) {
	tests := []struct {
		name         string
		goalLength   int
		costPerMTok  float64
		wantPositive bool
	}{
		{"Short goal", 50, 0.01, true},
		{"Medium goal", 500, 0.01, true},
		{"Long goal", 2000, 0.01, true},
		{"Zero cost per token", 500, 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := EstimateSpecGenerationCost(tt.goalLength, tt.costPerMTok)
			if tt.wantPositive && cost <= 0 {
				t.Errorf("EstimateSpecGenerationCost() = %v, want positive value", cost)
			}
			if !tt.wantPositive && cost != 0 {
				t.Errorf("EstimateSpecGenerationCost() = %v, want 0", cost)
			}
		})
	}
}

func TestEstimateSpecGenerationCost_Scaling(t *testing.T) {
	costShort := EstimateSpecGenerationCost(100, 0.01)
	costLong := EstimateSpecGenerationCost(1000, 0.01)

	// Longer goal should cost more
	if costLong <= costShort {
		t.Errorf("Longer goal should cost more: short=%v, long=%v", costShort, costLong)
	}
}

func TestEstimatePlanGenerationCost(t *testing.T) {
	tests := []struct {
		name         string
		featureCount int
		costPerMTok  float64
		wantPositive bool
	}{
		{"Few features", 2, 0.01, true},
		{"Many features", 10, 0.01, true},
		{"Zero features", 0, 0.01, true}, // Still has base cost
		{"Zero cost per token", 5, 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := EstimatePlanGenerationCost(tt.featureCount, tt.costPerMTok)
			if tt.wantPositive && cost <= 0 {
				t.Errorf("EstimatePlanGenerationCost() = %v, want positive value", cost)
			}
			if !tt.wantPositive && cost != 0 {
				t.Errorf("EstimatePlanGenerationCost() = %v, want 0", cost)
			}
		})
	}
}

func TestEstimatePlanGenerationCost_Scaling(t *testing.T) {
	costFew := EstimatePlanGenerationCost(2, 0.01)
	costMany := EstimatePlanGenerationCost(10, 0.01)

	// More features should cost more
	if costMany <= costFew {
		t.Errorf("More features should cost more: few=%v, many=%v", costFew, costMany)
	}
}

func TestEstimateTaskExecutionCost(t *testing.T) {
	tests := []struct {
		name        string
		taskCount   int
		costPerMTok float64
		wantZero    bool
	}{
		{"Few tasks", 3, 0.01, true},    // < 5 tasks, conservative estimate is 0
		{"Many tasks", 10, 0.01, false}, // >= 5 tasks, should have cost
		{"Zero tasks", 0, 0.01, true},
		{"Zero cost per token", 10, 0.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := EstimateTaskExecutionCost(tt.taskCount, tt.costPerMTok)
			if tt.wantZero && cost != 0 {
				t.Errorf("EstimateTaskExecutionCost() = %v, want 0", cost)
			}
			if !tt.wantZero && cost <= 0 {
				t.Errorf("EstimateTaskExecutionCost() = %v, want positive value", cost)
			}
		})
	}
}

func TestEstimateTaskExecutionCost_Conservative(t *testing.T) {
	// Should be conservative - assume only 20% of tasks use AI
	cost10 := EstimateTaskExecutionCost(10, 0.01)
	cost50 := EstimateTaskExecutionCost(50, 0.01)

	// Should scale, but conservatively (not linearly)
	ratio := cost50 / cost10
	if ratio < 2 || ratio > 10 {
		t.Errorf("Cost scaling seems off: 10 tasks=%v, 50 tasks=%v, ratio=%v", cost10, cost50, ratio)
	}
}

func TestCheckPerTaskBudget_WithinLimit(t *testing.T) {
	err := CheckPerTaskBudget(1.0, 2.0, "task-001")
	if err != nil {
		t.Errorf("CheckPerTaskBudget within limit should return nil, got %v", err)
	}
}

func TestCheckPerTaskBudget_ExceedsLimit(t *testing.T) {
	err := CheckPerTaskBudget(3.0, 2.0, "task-001")
	if err == nil {
		t.Error("CheckPerTaskBudget exceeding limit should return error")
	}
	if !strings.Contains(err.Error(), "task-001") {
		t.Errorf("Error should mention task ID, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "3.0") {
		t.Errorf("Error should mention estimated cost, got: %s", err.Error())
	}
}

func TestCheckPerTaskBudget_ExactLimit(t *testing.T) {
	err := CheckPerTaskBudget(2.0, 2.0, "task-001")
	if err != nil {
		t.Errorf("CheckPerTaskBudget at exact limit should return nil, got %v", err)
	}
}

func TestBudgetThresholds_Ordering(t *testing.T) {
	// Verify thresholds are in ascending order
	for i := 1; i < len(defaultThresholds); i++ {
		if defaultThresholds[i].Percentage <= defaultThresholds[i-1].Percentage {
			t.Errorf("Thresholds should be in ascending order: %v <= %v",
				defaultThresholds[i].Percentage, defaultThresholds[i-1].Percentage)
		}
	}
}

func TestBudgetThresholds_NonEmpty(t *testing.T) {
	if len(defaultThresholds) == 0 {
		t.Error("defaultThresholds should not be empty")
	}

	for i, threshold := range defaultThresholds {
		if threshold.Percentage <= 0 || threshold.Percentage > 100 {
			t.Errorf("Threshold %d has invalid percentage: %v", i, threshold.Percentage)
		}
		if threshold.Message == "" {
			t.Errorf("Threshold %d has empty message", i)
		}
	}
}
