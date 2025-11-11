package explain

import (
	"fmt"
	"time"
)

// RoutingExplanation contains the complete routing explanation for a workflow
type RoutingExplanation struct {
	// Workflow metadata
	WorkflowID  string    `json:"workflowId"`
	Goal        string    `json:"goal"`
	Profile     string    `json:"profile"`
	CompletedAt time.Time `json:"completedAt"`

	// Overall routing strategy
	Strategy RoutingStrategy `json:"strategy"`

	// Per-step routing decisions
	Steps []StepRouting `json:"steps"`

	// Summary statistics
	Summary RoutingSummary `json:"summary"`
}

// RoutingStrategy describes the overall routing configuration
type RoutingStrategy struct {
	// BudgetLimit is the maximum cost allowed
	BudgetLimit float64 `json:"budgetLimit"`

	// PreferCheap indicates if cheaper models are preferred
	PreferCheap bool `json:"preferCheap"`

	// MaxLatency is the maximum acceptable latency in milliseconds
	MaxLatency int `json:"maxLatency"`

	// FallbackEnabled indicates if fallback routing is enabled
	FallbackEnabled bool `json:"fallbackEnabled"`

	// ProviderPreferences lists providers in order of preference
	ProviderPreferences []string `json:"providerPreferences"`
}

// StepRouting explains routing decisions for a single step
type StepRouting struct {
	// Step identification
	StepID   string `json:"stepId"`
	StepType string `json:"stepType"`

	// Routing decision
	SelectedProvider string  `json:"selectedProvider"`
	SelectedModel    string  `json:"selectedModel"`
	Cost             float64 `json:"cost"`
	Duration         string  `json:"duration"`

	// Decision rationale
	Reason     string   `json:"reason"`
	Candidates []string `json:"candidates"` // Other providers/models considered

	// Signals that influenced the decision
	Signals map[string]string `json:"signals,omitempty"`
}

// RoutingSummary provides aggregate statistics
type RoutingSummary struct {
	// Total cost across all steps
	TotalCost float64 `json:"totalCost"`

	// Number of steps executed
	StepsExecuted int `json:"stepsExecuted"`

	// Provider usage breakdown
	ProviderBreakdown map[string]ProviderUsage `json:"providerBreakdown"`

	// Average latency per step
	AvgLatency string `json:"avgLatency"`

	// Budget utilization percentage
	BudgetUtilization float64 `json:"budgetUtilization"`
}

// ProviderUsage tracks usage statistics for a provider
type ProviderUsage struct {
	Provider string   `json:"provider"`
	Requests int      `json:"requests"`
	Cost     float64  `json:"cost"`
	Models   []string `json:"models"`
}

// Explainer analyzes and explains routing decisions
type Explainer struct {
	checkpointDir string
}

// NewExplainer creates a new routing explainer
func NewExplainer(checkpointDir string) *Explainer {
	return &Explainer{
		checkpointDir: checkpointDir,
	}
}

// Explain generates a routing explanation for a workflow
func (e *Explainer) Explain(checkpointID string) (*RoutingExplanation, error) {
	// Load checkpoint data
	checkpoint, err := e.loadCheckpoint(checkpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Analyze routing decisions
	explanation := &RoutingExplanation{
		WorkflowID:  checkpointID,
		Goal:        checkpoint.Goal,
		Profile:     checkpoint.Profile,
		CompletedAt: checkpoint.CompletedAt,
		Strategy:    e.analyzeStrategy(checkpoint),
		Steps:       e.analyzeSteps(checkpoint),
	}

	// Calculate summary
	explanation.Summary = e.calculateSummary(explanation.Steps, explanation.Strategy)

	return explanation, nil
}

// loadCheckpoint loads checkpoint data from disk
func (e *Explainer) loadCheckpoint(checkpointID string) (*CheckpointData, error) {
	// This is a placeholder - actual implementation would load from
	// ~/.specular/checkpoints/<checkpointID>/
	return nil, fmt.Errorf("checkpoint loading not yet implemented")
}

// analyzeStrategy extracts the routing strategy from checkpoint
func (e *Explainer) analyzeStrategy(checkpoint *CheckpointData) RoutingStrategy {
	return RoutingStrategy{
		BudgetLimit:         checkpoint.BudgetLimit,
		PreferCheap:         checkpoint.PreferCheap,
		MaxLatency:          checkpoint.MaxLatency,
		FallbackEnabled:     checkpoint.FallbackEnabled,
		ProviderPreferences: checkpoint.ProviderPreferences,
	}
}

// analyzeSteps analyzes routing decisions for each step
func (e *Explainer) analyzeSteps(checkpoint *CheckpointData) []StepRouting {
	steps := make([]StepRouting, 0, len(checkpoint.Steps))

	for _, step := range checkpoint.Steps {
		routing := StepRouting{
			StepID:           step.ID,
			StepType:         step.Type,
			SelectedProvider: step.Provider,
			SelectedModel:    step.Model,
			Cost:             step.Cost,
			Duration:         step.Duration.String(),
			Reason:           e.explainSelection(step),
			Candidates:       step.Candidates,
			Signals:          step.Signals,
		}
		steps = append(steps, routing)
	}

	return steps
}

// explainSelection generates a human-readable explanation for a routing decision
func (e *Explainer) explainSelection(step *StepData) string {
	// Generate explanation based on signals and selection criteria
	if step.SelectionReason != "" {
		return step.SelectionReason
	}

	// Default explanation
	return fmt.Sprintf("Selected %s/%s based on routing strategy", step.Provider, step.Model)
}

// calculateSummary generates summary statistics
func (e *Explainer) calculateSummary(steps []StepRouting, strategy RoutingStrategy) RoutingSummary {
	summary := RoutingSummary{
		ProviderBreakdown: make(map[string]ProviderUsage),
	}

	totalDuration := time.Duration(0)

	for _, step := range steps {
		summary.TotalCost += step.Cost
		summary.StepsExecuted++

		// Parse duration
		if duration, err := time.ParseDuration(step.Duration); err == nil {
			totalDuration += duration
		}

		// Track provider usage
		provider := step.SelectedProvider
		usage, exists := summary.ProviderBreakdown[provider]
		if !exists {
			usage = ProviderUsage{
				Provider: provider,
				Models:   []string{},
			}
		}
		usage.Requests++
		usage.Cost += step.Cost

		// Add model if not already tracked
		modelFound := false
		for _, m := range usage.Models {
			if m == step.SelectedModel {
				modelFound = true
				break
			}
		}
		if !modelFound {
			usage.Models = append(usage.Models, step.SelectedModel)
		}

		summary.ProviderBreakdown[provider] = usage
	}

	// Calculate averages
	if summary.StepsExecuted > 0 {
		avgDuration := totalDuration / time.Duration(summary.StepsExecuted)
		summary.AvgLatency = avgDuration.String()
	}

	// Calculate budget utilization
	if strategy.BudgetLimit > 0 {
		summary.BudgetUtilization = (summary.TotalCost / strategy.BudgetLimit) * 100
	}

	return summary
}

// CheckpointData represents loaded checkpoint information
type CheckpointData struct {
	Goal                string
	Profile             string
	CompletedAt         time.Time
	BudgetLimit         float64
	PreferCheap         bool
	MaxLatency          int
	FallbackEnabled     bool
	ProviderPreferences []string
	Steps               []*StepData
}

// StepData represents a step's execution data
type StepData struct {
	ID              string
	Type            string
	Provider        string
	Model           string
	Cost            float64
	Duration        time.Duration
	Candidates      []string
	Signals         map[string]string
	SelectionReason string
}
