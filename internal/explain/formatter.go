package explain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Formatter formats routing explanations for display
type Formatter struct {
	colorEnabled bool
}

// NewFormatter creates a new formatter
func NewFormatter(colorEnabled bool) *Formatter {
	return &Formatter{
		colorEnabled: colorEnabled,
	}
}

// FormatText formats an explanation as human-readable text
func (f *Formatter) FormatText(explanation *RoutingExplanation) string {
	var b strings.Builder

	// Header
	b.WriteString("🔍 Routing Explanation\n")
	b.WriteString("=" + strings.Repeat("=", 70) + "\n\n")

	// Workflow info
	b.WriteString(fmt.Sprintf("Workflow ID: %s\n", explanation.WorkflowID))
	b.WriteString(fmt.Sprintf("Goal:        %s\n", explanation.Goal))
	b.WriteString(fmt.Sprintf("Profile:     %s\n", explanation.Profile))
	b.WriteString(fmt.Sprintf("Completed:   %s\n\n", explanation.CompletedAt.Format("2006-01-02 15:04:05")))

	// Strategy
	b.WriteString("📋 Routing Strategy\n")
	b.WriteString("-" + strings.Repeat("-", 70) + "\n")
	b.WriteString(fmt.Sprintf("  Budget Limit:      $%.2f\n", explanation.Strategy.BudgetLimit))
	b.WriteString(fmt.Sprintf("  Prefer Cheap:      %v\n", explanation.Strategy.PreferCheap))
	b.WriteString(fmt.Sprintf("  Max Latency:       %dms\n", explanation.Strategy.MaxLatency))
	b.WriteString(fmt.Sprintf("  Fallback Enabled:  %v\n", explanation.Strategy.FallbackEnabled))
	if len(explanation.Strategy.ProviderPreferences) > 0 {
		b.WriteString(fmt.Sprintf("  Provider Order:    %s\n", strings.Join(explanation.Strategy.ProviderPreferences, " → ")))
	}
	b.WriteString("\n")

	// Steps
	b.WriteString("📍 Step-by-Step Routing Decisions\n")
	b.WriteString("-" + strings.Repeat("-", 70) + "\n\n")

	for i, step := range explanation.Steps {
		b.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, step.StepID, step.StepType))
		b.WriteString(fmt.Sprintf("   Selected: %s/%s\n", step.SelectedProvider, step.SelectedModel))
		b.WriteString(fmt.Sprintf("   Cost:     $%.4f\n", step.Cost))
		b.WriteString(fmt.Sprintf("   Duration: %s\n", step.Duration))
		b.WriteString(fmt.Sprintf("   Reason:   %s\n", step.Reason))

		if len(step.Candidates) > 0 {
			b.WriteString(fmt.Sprintf("   Candidates: %s\n", strings.Join(step.Candidates, ", ")))
		}

		if len(step.Signals) > 0 {
			b.WriteString("   Signals:\n")
			for key, value := range step.Signals {
				b.WriteString(fmt.Sprintf("     - %s: %s\n", key, value))
			}
		}

		b.WriteString("\n")
	}

	// Summary
	b.WriteString("📊 Summary\n")
	b.WriteString("-" + strings.Repeat("-", 70) + "\n")
	b.WriteString(fmt.Sprintf("  Total Cost:         $%.4f\n", explanation.Summary.TotalCost))
	b.WriteString(fmt.Sprintf("  Steps Executed:     %d\n", explanation.Summary.StepsExecuted))
	b.WriteString(fmt.Sprintf("  Avg Latency:        %s\n", explanation.Summary.AvgLatency))
	if explanation.Strategy.BudgetLimit > 0 {
		b.WriteString(fmt.Sprintf("  Budget Utilization: %.1f%%\n", explanation.Summary.BudgetUtilization))
	}
	b.WriteString("\n")

	// Provider breakdown
	if len(explanation.Summary.ProviderBreakdown) > 0 {
		b.WriteString("  Provider Breakdown:\n")
		for provider, usage := range explanation.Summary.ProviderBreakdown {
			b.WriteString(fmt.Sprintf("    %s:\n", provider))
			b.WriteString(fmt.Sprintf("      Requests: %d\n", usage.Requests))
			b.WriteString(fmt.Sprintf("      Cost:     $%.4f\n", usage.Cost))
			b.WriteString(fmt.Sprintf("      Models:   %s\n", strings.Join(usage.Models, ", ")))
		}
	}

	return b.String()
}

// FormatJSON formats an explanation as JSON
func (f *Formatter) FormatJSON(explanation *RoutingExplanation) (string, error) {
	data, err := json.MarshalIndent(explanation, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// FormatCompact formats an explanation in a compact summary format
func (f *Formatter) FormatCompact(explanation *RoutingExplanation) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Workflow %s (%s)\n", explanation.WorkflowID, explanation.Profile))
	b.WriteString(fmt.Sprintf("Goal: %s\n", explanation.Goal))
	b.WriteString(fmt.Sprintf("Cost: $%.4f | Steps: %d | Budget: %.1f%%\n\n",
		explanation.Summary.TotalCost,
		explanation.Summary.StepsExecuted,
		explanation.Summary.BudgetUtilization))

	b.WriteString("Step Routing:\n")
	for i, step := range explanation.Steps {
		b.WriteString(fmt.Sprintf("  %d. %s → %s/%s ($%.4f)\n",
			i+1, step.StepType, step.SelectedProvider, step.SelectedModel, step.Cost))
	}

	return b.String()
}

// FormatMarkdown formats an explanation as Markdown
func (f *Formatter) FormatMarkdown(explanation *RoutingExplanation) string {
	var b strings.Builder

	// Header
	b.WriteString("# Routing Explanation\n\n")
	b.WriteString(fmt.Sprintf("**Workflow ID:** %s  \n", explanation.WorkflowID))
	b.WriteString(fmt.Sprintf("**Goal:** %s  \n", explanation.Goal))
	b.WriteString(fmt.Sprintf("**Profile:** %s  \n", explanation.Profile))
	b.WriteString(fmt.Sprintf("**Completed:** %s  \n\n", explanation.CompletedAt.Format("2006-01-02 15:04:05")))

	// Strategy
	b.WriteString("## Routing Strategy\n\n")
	b.WriteString(fmt.Sprintf("- **Budget Limit:** $%.2f\n", explanation.Strategy.BudgetLimit))
	b.WriteString(fmt.Sprintf("- **Prefer Cheap:** %v\n", explanation.Strategy.PreferCheap))
	b.WriteString(fmt.Sprintf("- **Max Latency:** %dms\n", explanation.Strategy.MaxLatency))
	b.WriteString(fmt.Sprintf("- **Fallback Enabled:** %v\n", explanation.Strategy.FallbackEnabled))
	if len(explanation.Strategy.ProviderPreferences) > 0 {
		b.WriteString(fmt.Sprintf("- **Provider Order:** %s\n", strings.Join(explanation.Strategy.ProviderPreferences, " → ")))
	}
	b.WriteString("\n")

	// Steps
	b.WriteString("## Step-by-Step Routing Decisions\n\n")
	for i, step := range explanation.Steps {
		b.WriteString(fmt.Sprintf("### %d. %s (%s)\n\n", i+1, step.StepID, step.StepType))
		b.WriteString(fmt.Sprintf("- **Selected:** %s/%s\n", step.SelectedProvider, step.SelectedModel))
		b.WriteString(fmt.Sprintf("- **Cost:** $%.4f\n", step.Cost))
		b.WriteString(fmt.Sprintf("- **Duration:** %s\n", step.Duration))
		b.WriteString(fmt.Sprintf("- **Reason:** %s\n", step.Reason))

		if len(step.Candidates) > 0 {
			b.WriteString(fmt.Sprintf("- **Candidates:** %s\n", strings.Join(step.Candidates, ", ")))
		}

		if len(step.Signals) > 0 {
			b.WriteString("- **Signals:**\n")
			for key, value := range step.Signals {
				b.WriteString(fmt.Sprintf("  - %s: %s\n", key, value))
			}
		}

		b.WriteString("\n")
	}

	// Summary
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("- **Total Cost:** $%.4f\n", explanation.Summary.TotalCost))
	b.WriteString(fmt.Sprintf("- **Steps Executed:** %d\n", explanation.Summary.StepsExecuted))
	b.WriteString(fmt.Sprintf("- **Avg Latency:** %s\n", explanation.Summary.AvgLatency))
	if explanation.Strategy.BudgetLimit > 0 {
		b.WriteString(fmt.Sprintf("- **Budget Utilization:** %.1f%%\n", explanation.Summary.BudgetUtilization))
	}
	b.WriteString("\n")

	// Provider breakdown
	if len(explanation.Summary.ProviderBreakdown) > 0 {
		b.WriteString("### Provider Breakdown\n\n")
		b.WriteString("| Provider | Requests | Cost | Models |\n")
		b.WriteString("|----------|----------|------|--------|\n")
		for provider, usage := range explanation.Summary.ProviderBreakdown {
			b.WriteString(fmt.Sprintf("| %s | %d | $%.4f | %s |\n",
				provider, usage.Requests, usage.Cost, strings.Join(usage.Models, ", ")))
		}
	}

	return b.String()
}
