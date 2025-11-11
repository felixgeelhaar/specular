package explain

import (
	"strings"
	"testing"
	"time"
)

func TestFormatText(t *testing.T) {
	explanation := createTestExplanation()
	formatter := NewFormatter(true)

	output := formatter.FormatText(explanation)

	// Verify key sections are present
	if !strings.Contains(output, "Routing Explanation") {
		t.Error("Output missing header")
	}
	if !strings.Contains(output, "Workflow ID: test-workflow") {
		t.Error("Output missing workflow ID")
	}
	if !strings.Contains(output, "Routing Strategy") {
		t.Error("Output missing strategy section")
	}
	if !strings.Contains(output, "Step-by-Step Routing Decisions") {
		t.Error("Output missing steps section")
	}
	if !strings.Contains(output, "Summary") {
		t.Error("Output missing summary section")
	}

	// Verify step details
	if !strings.Contains(output, "openai/gpt-4") {
		t.Error("Output missing provider/model")
	}
	if !strings.Contains(output, "$0.5000") {
		t.Error("Output missing cost")
	}
}

func TestFormatJSON(t *testing.T) {
	explanation := createTestExplanation()
	formatter := NewFormatter(false)

	output, err := formatter.FormatJSON(explanation)
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}

	// Verify it's valid JSON
	if !strings.HasPrefix(output, "{") {
		t.Error("Output is not JSON")
	}
	if !strings.Contains(output, "\"workflowId\":") {
		t.Error("JSON missing workflowId field")
	}
	if !strings.Contains(output, "\"strategy\":") {
		t.Error("JSON missing strategy field")
	}
	if !strings.Contains(output, "\"steps\":") {
		t.Error("JSON missing steps field")
	}
}

func TestFormatCompact(t *testing.T) {
	explanation := createTestExplanation()
	formatter := NewFormatter(false)

	output := formatter.FormatCompact(explanation)

	// Verify compact format includes key info
	if !strings.Contains(output, "test-workflow") {
		t.Error("Compact output missing workflow ID")
	}
	if !strings.Contains(output, "Test goal") {
		t.Error("Compact output missing goal")
	}
	if !strings.Contains(output, "$1.0000") {
		t.Error("Compact output missing total cost")
	}
	if !strings.Contains(output, "Steps: 2") {
		t.Error("Compact output missing step count")
	}

	// Verify it's actually compact (fewer lines)
	lines := strings.Split(output, "\n")
	if len(lines) > 10 {
		t.Errorf("Compact output too long: %d lines", len(lines))
	}
}

func TestFormatMarkdown(t *testing.T) {
	explanation := createTestExplanation()
	formatter := NewFormatter(false)

	output := formatter.FormatMarkdown(explanation)

	// Verify markdown formatting
	if !strings.Contains(output, "# Routing Explanation") {
		t.Error("Markdown missing main header")
	}
	if !strings.Contains(output, "## Routing Strategy") {
		t.Error("Markdown missing strategy header")
	}
	if !strings.Contains(output, "## Step-by-Step Routing Decisions") {
		t.Error("Markdown missing steps header")
	}
	if !strings.Contains(output, "## Summary") {
		t.Error("Markdown missing summary header")
	}

	// Verify markdown table for provider breakdown
	if !strings.Contains(output, "| Provider | Requests | Cost | Models |") {
		t.Error("Markdown missing provider breakdown table")
	}

	// Verify bullet points
	if strings.Count(output, "- **") < 5 {
		t.Error("Markdown missing bullet points")
	}
}

func TestFormatterWithNoProviders(t *testing.T) {
	explanation := &RoutingExplanation{
		WorkflowID:  "empty-workflow",
		Goal:        "Empty test",
		Profile:     "test",
		CompletedAt: time.Now(),
		Strategy: RoutingStrategy{
			BudgetLimit: 10.0,
		},
		Steps: []StepRouting{},
		Summary: RoutingSummary{
			TotalCost:         0,
			StepsExecuted:     0,
			ProviderBreakdown: map[string]ProviderUsage{},
		},
	}

	formatter := NewFormatter(false)

	// Should not panic with empty data
	_ = formatter.FormatText(explanation)
	_, err := formatter.FormatJSON(explanation)
	if err != nil {
		t.Errorf("FormatJSON failed with empty data: %v", err)
	}
	_ = formatter.FormatCompact(explanation)
	_ = formatter.FormatMarkdown(explanation)
}

func TestFormatterWithMultipleProviders(t *testing.T) {
	explanation := createTestExplanation()

	// Add steps from different providers
	explanation.Steps = append(explanation.Steps, StepRouting{
		StepID:           "step-3",
		StepType:         "spec:update",
		SelectedProvider: "anthropic",
		SelectedModel:    "claude-3",
		Cost:             0.25,
		Duration:         "500ms",
		Reason:           "Anthropic preferred for spec generation",
		Candidates:       []string{"openai/gpt-4", "anthropic/claude-3"},
	})

	explanation.Summary.ProviderBreakdown["anthropic"] = ProviderUsage{
		Provider: "anthropic",
		Requests: 1,
		Cost:     0.25,
		Models:   []string{"claude-3"},
	}
	explanation.Summary.TotalCost += 0.25
	explanation.Summary.StepsExecuted++

	formatter := NewFormatter(false)
	output := formatter.FormatText(explanation)

	// Verify multiple providers are shown
	if !strings.Contains(output, "openai") {
		t.Error("Output missing openai provider")
	}
	if !strings.Contains(output, "anthropic") {
		t.Error("Output missing anthropic provider")
	}
	if !strings.Contains(output, "Steps Executed:     3") {
		t.Error("Output shows wrong step count")
	}
}

// createTestExplanation creates a sample explanation for testing
func createTestExplanation() *RoutingExplanation {
	return &RoutingExplanation{
		WorkflowID:  "test-workflow",
		Goal:        "Test goal",
		Profile:     "default",
		CompletedAt: time.Now(),
		Strategy: RoutingStrategy{
			BudgetLimit:         10.0,
			PreferCheap:         true,
			MaxLatency:          60000,
			FallbackEnabled:     true,
			ProviderPreferences: []string{"openai", "anthropic"},
		},
		Steps: []StepRouting{
			{
				StepID:           "step-1",
				StepType:         "spec:update",
				SelectedProvider: "openai",
				SelectedModel:    "gpt-4",
				Cost:             0.50,
				Duration:         "1.2s",
				Reason:           "Selected based on routing strategy",
				Candidates:       []string{"openai/gpt-4", "anthropic/claude-3"},
				Signals: map[string]string{
					"complexity": "high",
					"context":    "large",
				},
			},
			{
				StepID:           "step-2",
				StepType:         "plan:gen",
				SelectedProvider: "openai",
				SelectedModel:    "gpt-3.5-turbo",
				Cost:             0.50,
				Duration:         "800ms",
				Reason:           "Cheaper model selected due to PreferCheap strategy",
				Candidates:       []string{"openai/gpt-4", "openai/gpt-3.5-turbo"},
			},
		},
		Summary: RoutingSummary{
			TotalCost:         1.0,
			StepsExecuted:     2,
			AvgLatency:        "1.0s",
			BudgetUtilization: 10.0,
			ProviderBreakdown: map[string]ProviderUsage{
				"openai": {
					Provider: "openai",
					Requests: 2,
					Cost:     1.0,
					Models:   []string{"gpt-4", "gpt-3.5-turbo"},
				},
			},
		},
	}
}
