package explain

import (
	"testing"
	"time"
)

func TestNewExplainer(t *testing.T) {
	explainer := NewExplainer("/test/dir")
	if explainer == nil {
		t.Fatal("NewExplainer returned nil")
	}
	if explainer.checkpointDir != "/test/dir" {
		t.Errorf("checkpointDir mismatch: got %s, want /test/dir", explainer.checkpointDir)
	}
}

func TestAnalyzeStrategy(t *testing.T) {
	explainer := NewExplainer("/test")
	checkpoint := &CheckpointData{
		BudgetLimit:         100.0,
		PreferCheap:         true,
		MaxLatency:          30000,
		FallbackEnabled:     true,
		ProviderPreferences: []string{"openai", "anthropic", "ollama"},
	}

	strategy := explainer.analyzeStrategy(checkpoint)

	if strategy.BudgetLimit != 100.0 {
		t.Errorf("BudgetLimit mismatch: got %.2f, want 100.00", strategy.BudgetLimit)
	}
	if !strategy.PreferCheap {
		t.Error("PreferCheap should be true")
	}
	if strategy.MaxLatency != 30000 {
		t.Errorf("MaxLatency mismatch: got %d, want 30000", strategy.MaxLatency)
	}
	if !strategy.FallbackEnabled {
		t.Error("FallbackEnabled should be true")
	}
	if len(strategy.ProviderPreferences) != 3 {
		t.Errorf("ProviderPreferences length mismatch: got %d, want 3", len(strategy.ProviderPreferences))
	}
}

func TestAnalyzeSteps(t *testing.T) {
	explainer := NewExplainer("/test")
	checkpoint := &CheckpointData{
		Steps: []*StepData{
			{
				ID:              "step-1",
				Type:            "spec:update",
				Provider:        "openai",
				Model:           "gpt-4",
				Cost:            1.5,
				Duration:        2 * time.Second,
				Candidates:      []string{"openai/gpt-4", "anthropic/claude-3"},
				Signals:         map[string]string{"complexity": "high"},
				SelectionReason: "High complexity task",
			},
			{
				ID:       "step-2",
				Type:     "plan:gen",
				Provider: "openai",
				Model:    "gpt-3.5-turbo",
				Cost:     0.5,
				Duration: 1 * time.Second,
			},
		},
	}

	steps := explainer.analyzeSteps(checkpoint)

	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(steps))
	}

	// Verify first step
	step1 := steps[0]
	if step1.StepID != "step-1" {
		t.Errorf("Step1 ID mismatch: got %s, want step-1", step1.StepID)
	}
	if step1.SelectedProvider != "openai" {
		t.Errorf("Step1 provider mismatch: got %s, want openai", step1.SelectedProvider)
	}
	if step1.SelectedModel != "gpt-4" {
		t.Errorf("Step1 model mismatch: got %s, want gpt-4", step1.SelectedModel)
	}
	if step1.Cost != 1.5 {
		t.Errorf("Step1 cost mismatch: got %.2f, want 1.50", step1.Cost)
	}
	if step1.Reason != "High complexity task" {
		t.Errorf("Step1 reason mismatch: got %s", step1.Reason)
	}
	if len(step1.Candidates) != 2 {
		t.Errorf("Step1 candidates length mismatch: got %d, want 2", len(step1.Candidates))
	}

	// Verify second step
	step2 := steps[1]
	if step2.StepID != "step-2" {
		t.Errorf("Step2 ID mismatch: got %s, want step-2", step2.StepID)
	}
	if step2.Cost != 0.5 {
		t.Errorf("Step2 cost mismatch: got %.2f, want 0.50", step2.Cost)
	}
}

func TestCalculateSummary(t *testing.T) {
	explainer := NewExplainer("/test")

	steps := []StepRouting{
		{
			StepID:           "step-1",
			SelectedProvider: "openai",
			SelectedModel:    "gpt-4",
			Cost:             1.0,
			Duration:         "2s",
		},
		{
			StepID:           "step-2",
			SelectedProvider: "openai",
			SelectedModel:    "gpt-3.5-turbo",
			Cost:             0.5,
			Duration:         "1s",
		},
		{
			StepID:           "step-3",
			SelectedProvider: "anthropic",
			SelectedModel:    "claude-3",
			Cost:             0.75,
			Duration:         "1.5s",
		},
	}

	strategy := RoutingStrategy{
		BudgetLimit: 10.0,
	}

	summary := explainer.calculateSummary(steps, strategy)

	// Verify total cost
	if summary.TotalCost != 2.25 {
		t.Errorf("TotalCost mismatch: got %.2f, want 2.25", summary.TotalCost)
	}

	// Verify steps executed
	if summary.StepsExecuted != 3 {
		t.Errorf("StepsExecuted mismatch: got %d, want 3", summary.StepsExecuted)
	}

	// Verify budget utilization
	expected := 22.5 // (2.25 / 10.0) * 100
	if summary.BudgetUtilization != expected {
		t.Errorf("BudgetUtilization mismatch: got %.2f, want %.2f", summary.BudgetUtilization, expected)
	}

	// Verify provider breakdown
	if len(summary.ProviderBreakdown) != 2 {
		t.Errorf("Expected 2 providers in breakdown, got %d", len(summary.ProviderBreakdown))
	}

	openai, ok := summary.ProviderBreakdown["openai"]
	if !ok {
		t.Error("OpenAI not in provider breakdown")
	} else {
		if openai.Requests != 2 {
			t.Errorf("OpenAI requests mismatch: got %d, want 2", openai.Requests)
		}
		if openai.Cost != 1.5 {
			t.Errorf("OpenAI cost mismatch: got %.2f, want 1.50", openai.Cost)
		}
		if len(openai.Models) != 2 {
			t.Errorf("OpenAI models count mismatch: got %d, want 2", len(openai.Models))
		}
	}

	anthropic, ok := summary.ProviderBreakdown["anthropic"]
	if !ok {
		t.Error("Anthropic not in provider breakdown")
	} else {
		if anthropic.Requests != 1 {
			t.Errorf("Anthropic requests mismatch: got %d, want 1", anthropic.Requests)
		}
		if anthropic.Cost != 0.75 {
			t.Errorf("Anthropic cost mismatch: got %.2f, want 0.75", anthropic.Cost)
		}
	}
}

func TestCalculateSummaryNoBudget(t *testing.T) {
	explainer := NewExplainer("/test")

	steps := []StepRouting{
		{
			StepID:           "step-1",
			SelectedProvider: "openai",
			SelectedModel:    "gpt-4",
			Cost:             1.0,
			Duration:         "1s",
		},
	}

	strategy := RoutingStrategy{
		BudgetLimit: 0, // No budget set
	}

	summary := explainer.calculateSummary(steps, strategy)

	// With no budget, utilization should be 0
	if summary.BudgetUtilization != 0 {
		t.Errorf("BudgetUtilization should be 0 with no budget, got %.2f", summary.BudgetUtilization)
	}
}

func TestExplainSelection(t *testing.T) {
	explainer := NewExplainer("/test")

	// Test with explicit reason
	step1 := &StepData{
		Provider:        "openai",
		Model:           "gpt-4",
		SelectionReason: "Custom reason",
	}
	reason1 := explainer.explainSelection(step1)
	if reason1 != "Custom reason" {
		t.Errorf("Expected custom reason, got: %s", reason1)
	}

	// Test with default reason
	step2 := &StepData{
		Provider: "anthropic",
		Model:    "claude-3",
	}
	reason2 := explainer.explainSelection(step2)
	if reason2 != "Selected anthropic/claude-3 based on routing strategy" {
		t.Errorf("Unexpected default reason: %s", reason2)
	}
}

func TestCalculateSummaryEmptySteps(t *testing.T) {
	explainer := NewExplainer("/test")

	steps := []StepRouting{}
	strategy := RoutingStrategy{BudgetLimit: 10.0}

	summary := explainer.calculateSummary(steps, strategy)

	if summary.TotalCost != 0 {
		t.Errorf("TotalCost should be 0 for empty steps, got %.2f", summary.TotalCost)
	}
	if summary.StepsExecuted != 0 {
		t.Errorf("StepsExecuted should be 0 for empty steps, got %d", summary.StepsExecuted)
	}
	if len(summary.ProviderBreakdown) != 0 {
		t.Errorf("ProviderBreakdown should be empty, got %d providers", len(summary.ProviderBreakdown))
	}
}
