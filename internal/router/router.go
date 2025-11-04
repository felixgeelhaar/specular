package router

import (
	"fmt"
	"strings"
)

// Router manages model selection and routing
type Router struct {
	config  *RouterConfig
	budget  *Budget
	models  []Model
	usage   []Usage
}

// NewRouter creates a new router with configuration
func NewRouter(config *RouterConfig) (*Router, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	return &Router{
		config: config,
		budget: &Budget{
			LimitUSD:     config.BudgetUSD,
			SpentUSD:     0,
			RemainingUSD: config.BudgetUSD,
			UsageCount:   0,
		},
		models: GetAvailableModels(),
		usage:  []Usage{},
	}, nil
}

// SelectModel chooses the best model for a routing request
func (r *Router) SelectModel(req RoutingRequest) (*RoutingResult, error) {
	// Check budget
	if r.budget.RemainingUSD <= 0 {
		return nil, fmt.Errorf("budget exhausted (spent: $%.2f / limit: $%.2f)", r.budget.SpentUSD, r.budget.LimitUSD)
	}

	// Get candidate models based on hint
	candidates := r.getCandidateModels(req)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable models found for request")
	}

	// Score and rank candidates
	scored := r.scoreModels(candidates, req)
	if len(scored) == 0 {
		return nil, fmt.Errorf("no models passed scoring criteria")
	}

	// Select best model
	best := scored[0]

	// Estimate cost
	estimatedTokens := r.estimateTokens(req)
	estimatedCost := (float64(estimatedTokens) / 1000000.0) * best.CostPerMToken

	// Check if estimated cost exceeds budget
	if estimatedCost > r.budget.RemainingUSD {
		// Try to find a cheaper model
		cheaper := r.findCheaperModel(candidates, estimatedCost)
		if cheaper != nil {
			best = cheaper
			estimatedCost = (float64(estimatedTokens) / 1000000.0) * best.CostPerMToken
		} else {
			return nil, fmt.Errorf("estimated cost ($%.2f) exceeds remaining budget ($%.2f)", estimatedCost, r.budget.RemainingUSD)
		}
	}

	reason := r.buildSelectionReason(best, req)

	return &RoutingResult{
		Model:           best,
		Reason:          reason,
		EstimatedCost:   estimatedCost,
		EstimatedTokens: estimatedTokens,
	}, nil
}

// getCandidateModels filters models based on routing request
func (r *Router) getCandidateModels(req RoutingRequest) []Model {
	var candidates []Model

	// Map hint to model type
	var preferredType ModelType
	switch strings.ToLower(req.ModelHint) {
	case "codegen", "code":
		preferredType = ModelTypeCodegen
	case "long-context", "longcontext":
		preferredType = ModelTypeLongContext
	case "agentic", "agent":
		preferredType = ModelTypeAgentic
	case "fast", "quick":
		preferredType = ModelTypeFast
	case "cheap", "budget":
		preferredType = ModelTypeCheap
	}

	// Filter by type if specified
	if preferredType != "" {
		for _, m := range r.models {
			if m.Available && m.Type == preferredType {
				candidates = append(candidates, m)
			}
		}
	}

	// If no candidates or no hint, use all available models
	if len(candidates) == 0 {
		for _, m := range r.models {
			if m.Available {
				candidates = append(candidates, m)
			}
		}
	}

	// Filter by context window requirement
	if req.ContextSize > 0 {
		filtered := []Model{}
		for _, m := range candidates {
			if m.ContextWindow >= req.ContextSize {
				filtered = append(filtered, m)
			}
		}
		if len(filtered) > 0 {
			candidates = filtered
		}
	}

	// Filter by latency requirement
	if r.config.MaxLatencyMs > 0 {
		filtered := []Model{}
		for _, m := range candidates {
			if m.MaxLatencyMs <= r.config.MaxLatencyMs {
				filtered = append(filtered, m)
			}
		}
		if len(filtered) > 0 {
			candidates = filtered
		}
	}

	return candidates
}

// scoreModels ranks candidate models based on request requirements
func (r *Router) scoreModels(candidates []Model, req RoutingRequest) []*Model {
	type scoredModel struct {
		model *Model
		score float64
	}

	var scored []scoredModel

	for i := range candidates {
		m := &candidates[i]
		score := 0.0

		// Base score from capability
		score += m.CapabilityScore

		// Boost for P0 tasks - use best models
		if req.Priority == "P0" {
			score += 20
		}

		// Complexity adjustment
		if req.Complexity >= 7 {
			// High complexity - prefer capable models
			score += m.CapabilityScore * 0.3
		} else {
			// Low complexity - cost matters more
			if r.config.PreferCheap {
				// Inverse cost score (cheaper is better)
				maxCost := 10.0 // Reference max cost
				costScore := (maxCost - m.CostPerMToken) / maxCost * 30
				score += costScore
			}
		}

		// Penalize high latency models if latency matters
		if r.config.MaxLatencyMs > 0 && m.MaxLatencyMs > r.config.MaxLatencyMs/2 {
			score -= 10
		}

		scored = append(scored, scoredModel{model: m, score: score})
	}

	// Sort by score (descending)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Extract models
	result := make([]*Model, len(scored))
	for i, sm := range scored {
		result[i] = sm.model
	}

	return result
}

// estimateTokens estimates token usage for a request
func (r *Router) estimateTokens(req RoutingRequest) int {
	// Base estimation
	baseTokens := 1000

	// Add tokens based on complexity
	baseTokens += req.Complexity * 500

	// Add context size
	baseTokens += req.ContextSize

	// Response size estimate (output tokens)
	responseTokens := baseTokens / 2

	return baseTokens + responseTokens
}

// findCheaperModel finds a cheaper alternative that fits budget
func (r *Router) findCheaperModel(candidates []Model, maxCost float64) *Model {
	var cheapest *Model
	minCost := maxCost

	for i := range candidates {
		m := &candidates[i]
		cost := (float64(r.estimateTokens(RoutingRequest{})) / 1000000.0) * m.CostPerMToken
		if cost < minCost {
			minCost = cost
			cheapest = m
		}
	}

	return cheapest
}

// buildSelectionReason creates a human-readable explanation
func (r *Router) buildSelectionReason(model *Model, req RoutingRequest) string {
	reasons := []string{}

	if req.ModelHint != "" {
		reasons = append(reasons, fmt.Sprintf("matched hint: %s", req.ModelHint))
	}

	if req.Priority == "P0" {
		reasons = append(reasons, "high priority task")
	}

	if req.Complexity >= 7 {
		reasons = append(reasons, "high complexity requires capable model")
	}

	if r.config.PreferCheap && model.CostPerMToken < 1.0 {
		reasons = append(reasons, "budget-optimized selection")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "best overall capability")
	}

	return fmt.Sprintf("Selected %s (%s): %s", model.ID, model.Provider, strings.Join(reasons, ", "))
}

// RecordUsage records model usage and updates budget
func (r *Router) RecordUsage(usage Usage) error {
	// Update budget
	r.budget.SpentUSD += usage.CostUSD
	r.budget.RemainingUSD = r.budget.LimitUSD - r.budget.SpentUSD
	r.budget.UsageCount++

	// Store usage
	r.usage = append(r.usage, usage)

	return nil
}

// GetBudget returns the current budget status
func (r *Router) GetBudget() *Budget {
	return r.budget
}

// GetUsageStats returns usage statistics
func (r *Router) GetUsageStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["total_requests"] = len(r.usage)
	stats["budget_spent"] = r.budget.SpentUSD
	stats["budget_remaining"] = r.budget.RemainingUSD

	// Model usage counts
	modelCounts := make(map[string]int)
	for _, u := range r.usage {
		modelCounts[u.Model]++
	}
	stats["model_usage"] = modelCounts

	// Provider usage
	providerCounts := make(map[Provider]int)
	for _, u := range r.usage {
		providerCounts[u.Provider]++
	}
	stats["provider_usage"] = providerCounts

	return stats
}
