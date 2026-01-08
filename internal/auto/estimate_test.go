package auto

import (
	"strings"
	"testing"
)

func TestEstimateWorkflowCost(t *testing.T) {
	tests := []struct {
		name            string
		goal            string
		wantMinTotal    float64
		wantMaxTotal    float64
		wantMinFeatures int
		wantMaxFeatures int
	}{
		{
			name:            "simple goal",
			goal:            "Build a simple API",
			wantMinTotal:    0.0001,
			wantMaxTotal:    0.01,
			wantMinFeatures: 1,
			wantMaxFeatures: 3,
		},
		{
			name:            "complex goal with multiple features",
			goal:            "Build a REST API with authentication, database integration, and a frontend dashboard",
			wantMinTotal:    0.001,
			wantMaxTotal:    0.1,
			wantMinFeatures: 3,
			wantMaxFeatures: 10,
		},
		{
			name:            "empty goal",
			goal:            "",
			wantMinTotal:    0,
			wantMaxTotal:    0.01,
			wantMinFeatures: 1,
			wantMaxFeatures: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			estimate := EstimateWorkflowCost(tc.goal, nil)

			if estimate.Total < tc.wantMinTotal || estimate.Total > tc.wantMaxTotal {
				t.Errorf("Total cost %.6f not in expected range [%.6f, %.6f]",
					estimate.Total, tc.wantMinTotal, tc.wantMaxTotal)
			}

			if estimate.EstFeatureCount < tc.wantMinFeatures || estimate.EstFeatureCount > tc.wantMaxFeatures {
				t.Errorf("Feature count %d not in expected range [%d, %d]",
					estimate.EstFeatureCount, tc.wantMinFeatures, tc.wantMaxFeatures)
			}

			// Verify model is set
			if estimate.ModelUsed == "" {
				t.Error("ModelUsed should not be empty")
			}

			// Verify total is sum of components
			expectedTotal := estimate.SpecGeneration + estimate.PlanGeneration + estimate.TaskExecution
			if estimate.Total != expectedTotal {
				t.Errorf("Total %.6f should equal sum of components %.6f",
					estimate.Total, expectedTotal)
			}
		})
	}
}

func TestEstimateFeatureCount(t *testing.T) {
	tests := []struct {
		name         string
		goal         string
		wantFeatures int
	}{
		{"minimal", "hello", 1},
		{"with and", "foo and bar", 2},
		{"with api", "build an api", 2},
		{"complex", "Build API with authentication and database plus frontend", 8}, // api, with, authentication, database, plus, frontend, and, backend counted
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := estimateFeatureCount(tc.goal)
			if result != tc.wantFeatures {
				t.Errorf("estimateFeatureCount(%q) = %d, want %d", tc.goal, result, tc.wantFeatures)
			}
		})
	}
}

func TestFormatCostEstimate(t *testing.T) {
	estimate := &CostEstimate{
		SpecGeneration:  0.0015,
		PlanGeneration:  0.0024,
		TaskExecution:   0.0003,
		Total:           0.0042,
		ModelUsed:       "claude-haiku",
		EstFeatureCount: 3,
		EstTaskCount:    6,
		BudgetLimit:     5.0,
		BudgetRemaining: 5.0,
		BudgetUsage:     0.084,
	}

	output := FormatCostEstimate(estimate)

	// Check for expected sections
	if !strings.Contains(output, "Cost Estimation") {
		t.Error("Output should contain 'Cost Estimation' header")
	}
	if !strings.Contains(output, "Spec Generation") {
		t.Error("Output should contain 'Spec Generation'")
	}
	if !strings.Contains(output, "Plan Generation") {
		t.Error("Output should contain 'Plan Generation'")
	}
	if !strings.Contains(output, "Task Execution") {
		t.Error("Output should contain 'Task Execution'")
	}
	if !strings.Contains(output, "Estimated Total") {
		t.Error("Output should contain 'Estimated Total'")
	}
	if !strings.Contains(output, "Budget") {
		t.Error("Output should contain 'Budget'")
	}
	if !strings.Contains(output, "claude-haiku") {
		t.Error("Output should contain model name")
	}
}

func TestFormatCostEstimate_HighBudgetUsage(t *testing.T) {
	estimate := &CostEstimate{
		Total:           4.5,
		BudgetLimit:     5.0,
		BudgetRemaining: 0.5,
		BudgetUsage:     90.0,
	}

	output := FormatCostEstimate(estimate)

	// Check for HIGH warning for 90%+ usage
	if !strings.Contains(output, "HIGH") {
		t.Error("Output should contain HIGH warning for 90%+ usage")
	}
}
