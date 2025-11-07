package router

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/provider"
)

// Router manages model selection and routing
type Router struct {
	config           *RouterConfig
	budget           *Budget
	models           []Model
	usage            []Usage
	registry         *provider.Registry
	contextValidator *ContextValidator
	contextTruncator *ContextTruncator
}

// NewRouter creates a new router with configuration
func NewRouter(config *RouterConfig) (*Router, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	r := &Router{
		config: config,
		budget: &Budget{
			LimitUSD:     config.BudgetUSD,
			SpentUSD:     0,
			RemainingUSD: config.BudgetUSD,
			UsageCount:   0,
		},
		models:   GetAvailableModels(),
		usage:    []Usage{},
		registry: provider.NewRegistry(),
	}

	// Initialize context management if enabled
	if config.EnableContextValidation {
		r.contextValidator = NewContextValidator()

		// Create truncator with configured strategy
		strategy := TruncationStrategy(config.TruncationStrategy)
		if strategy == "" {
			strategy = TruncateOldest // Default to oldest
		}
		r.contextTruncator = NewContextTruncator(strategy)
	}

	// Update model availability based on loaded providers
	r.updateModelAvailability()

	return r, nil
}

// NewRouterWithProviders creates a router with pre-loaded providers
func NewRouterWithProviders(config *RouterConfig, registry *provider.Registry) (*Router, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if registry == nil {
		registry = provider.NewRegistry()
	}

	r := &Router{
		config: config,
		budget: &Budget{
			LimitUSD:     config.BudgetUSD,
			SpentUSD:     0,
			RemainingUSD: config.BudgetUSD,
			UsageCount:   0,
		},
		models:   GetAvailableModels(),
		usage:    []Usage{},
		registry: registry,
	}

	// Initialize context management if enabled
	if config.EnableContextValidation {
		r.contextValidator = NewContextValidator()

		// Create truncator with configured strategy
		strategy := TruncationStrategy(config.TruncationStrategy)
		if strategy == "" {
			strategy = TruncateOldest // Default to oldest
		}
		r.contextTruncator = NewContextTruncator(strategy)
	}

	// Update model availability based on provider availability
	r.updateModelAvailability()

	return r, nil
}

// updateModelAvailability checks which models are actually available based on loaded providers
func (r *Router) updateModelAvailability() {
	providerNames := r.registry.List()
	providerMap := make(map[string]bool)
	for _, name := range providerNames {
		providerMap[name] = true
	}

	// Mark models as unavailable if their provider isn't loaded
	for i := range r.models {
		// Check if provider is loaded (map provider name to registry name)
		providerLoaded := false
		switch r.models[i].Provider {
		case ProviderAnthropic:
			providerLoaded = providerMap["anthropic"]
		case ProviderOpenAI:
			providerLoaded = providerMap["openai"]
		case ProviderLocal:
			providerLoaded = providerMap["ollama"] || providerMap["local"]
		}
		r.models[i].Available = providerLoaded
	}
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

// Generate sends a prompt to the selected AI provider and returns a response
func (r *Router) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	// Select the best model for this request
	routing := RoutingRequest{
		ModelHint:   req.ModelHint,
		Complexity:  req.Complexity,
		Priority:    req.Priority,
		ContextSize: req.ContextSize,
	}

	result, err := r.SelectModel(routing)
	if err != nil {
		return nil, fmt.Errorf("model selection failed: %w", err)
	}

	// Validate context window if enabled
	if r.config.EnableContextValidation && r.contextValidator != nil {
		validationErr := r.contextValidator.ValidateRequest(&req, result.Model)
		if validationErr != nil {
			// Try auto-truncation if enabled
			if r.config.AutoTruncate && r.contextTruncator != nil {
				truncatedReq, truncated, truncErr := r.contextTruncator.TruncateRequest(&req, result.Model)
				if truncErr != nil {
					return nil, fmt.Errorf("context validation failed and truncation failed: %w", truncErr)
				}
				if truncated {
					// Use truncated request
					req = *truncatedReq
				}
			} else {
				// Return validation error if auto-truncate is disabled
				return nil, fmt.Errorf("context validation failed: %w", validationErr)
			}
		}
	}

	// Try primary provider with retries
	provResp, err := r.generateWithRetry(ctx, req, result)
	if err != nil {
		// If fallback is enabled, try alternative providers
		if r.config.EnableFallback {
			return r.generateWithFallback(ctx, req, result, startTime)
		}
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	// Calculate actual cost
	actualCost := (float64(provResp.TokensUsed) / 1000000.0) * result.Model.CostPerMToken

	// Record usage
	usage := Usage{
		Model:     result.Model.ID,
		Provider:  result.Model.Provider,
		Tokens:    provResp.TokensUsed,
		CostUSD:   actualCost,
		LatencyMs: int(time.Since(startTime).Milliseconds()),
		Timestamp: time.Now(),
		TaskID:    req.TaskID,
		Success:   provResp.Error == "",
	}
	r.RecordUsage(usage)

	// Build response
	return &GenerateResponse{
		Content:         provResp.Content,
		Model:           result.Model.ID,
		Provider:        result.Model.Provider,
		TokensUsed:      provResp.TokensUsed,
		InputTokens:     provResp.InputTokens,
		OutputTokens:    provResp.OutputTokens,
		CostUSD:         actualCost,
		Latency:         provResp.Latency,
		FinishReason:    provResp.FinishReason,
		SelectionReason: result.Reason,
		ToolCalls:       provResp.ToolCalls,
		Error:           provResp.Error,
	}, nil
}

// Stream sends a prompt and returns a streaming response with retry and fallback
func (r *Router) Stream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error) {
	startTime := time.Now()

	// Select the best model for this request
	routing := RoutingRequest{
		ModelHint:   req.ModelHint,
		Complexity:  req.Complexity,
		Priority:    req.Priority,
		ContextSize: req.ContextSize,
	}

	result, err := r.SelectModel(routing)
	if err != nil {
		return nil, fmt.Errorf("model selection failed: %w", err)
	}

	// Validate context window if enabled
	if r.config.EnableContextValidation && r.contextValidator != nil {
		validationErr := r.contextValidator.ValidateRequest(&req, result.Model)
		if validationErr != nil {
			// Try auto-truncation if enabled
			if r.config.AutoTruncate && r.contextTruncator != nil {
				truncatedReq, truncated, truncErr := r.contextTruncator.TruncateRequest(&req, result.Model)
				if truncErr != nil {
					return nil, fmt.Errorf("context validation failed and truncation failed: %w", truncErr)
				}
				if truncated {
					// Use truncated request
					req = *truncatedReq
				}
			} else {
				// Return validation error if auto-truncate is disabled
				return nil, fmt.Errorf("context validation failed: %w", validationErr)
			}
		}
	}

	// Try primary provider with retries
	provStream, streamResult, err := r.streamWithRetry(ctx, req, result)
	if err != nil {
		// If fallback is enabled, try alternative providers
		if r.config.EnableFallback {
			return r.streamWithFallback(ctx, req, result, startTime)
		}
		return nil, fmt.Errorf("streaming failed: %w", err)
	}

	// Create output channel
	outChan := make(chan StreamChunk, 10)

	// Forward stream chunks with usage tracking
	go func() {
		defer close(outChan)
		var totalTokens int

		for chunk := range provStream {
			outChan <- StreamChunk{
				Content: chunk.Content,
				Delta:   chunk.Delta,
				Done:    chunk.Done,
				Error:   chunk.Error,
			}

			if chunk.Done {
				totalTokens = chunk.TokensUsed
			}
		}

		// Record usage after stream completes
		if totalTokens > 0 {
			actualCost := (float64(totalTokens) / 1000000.0) * streamResult.Model.CostPerMToken
			usage := Usage{
				Model:     streamResult.Model.ID,
				Provider:  streamResult.Model.Provider,
				Tokens:    totalTokens,
				CostUSD:   actualCost,
				LatencyMs: int(time.Since(startTime).Milliseconds()),
				Timestamp: time.Now(),
				TaskID:    req.TaskID,
				Success:   true,
			}
			r.RecordUsage(usage)
		}
	}()

	return outChan, nil
}

// getProviderName maps router Provider to registry provider name
func (r *Router) getProviderName(p Provider) string {
	switch p {
	case ProviderAnthropic:
		return "anthropic"
	case ProviderOpenAI:
		return "openai"
	case ProviderLocal:
		// Try ollama first, then local
		if _, err := r.registry.Get("ollama"); err == nil {
			return "ollama"
		}
		return "local"
	default:
		return ""
	}
}

// GetRegistry returns the provider registry
func (r *Router) GetRegistry() *provider.Registry {
	return r.registry
}

// SetModelsAvailable is a test helper that marks all models as available
// This is useful for testing model selection logic without needing actual providers
func (r *Router) SetModelsAvailable(available bool) {
	for i := range r.models {
		r.models[i].Available = available
	}
}

// generateWithRetry attempts generation with exponential backoff retry logic
func (r *Router) generateWithRetry(ctx context.Context, req GenerateRequest, result *RoutingResult) (*provider.GenerateResponse, error) {
	// Get provider name from model
	providerName := r.getProviderName(result.Model.Provider)
	if providerName == "" {
		return nil, fmt.Errorf("no provider available for model %s", result.Model.ID)
	}

	// Get provider from registry
	prov, err := r.registry.Get(providerName)
	if err != nil {
		return nil, fmt.Errorf("provider %s not available: %w", providerName, err)
	}

	// Build provider request
	provReq := &provider.GenerateRequest{
		Prompt:       req.Prompt,
		SystemPrompt: req.SystemPrompt,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		TopP:         req.TopP,
		Tools:        req.Tools,
		Context:      req.Context,
		Config: map[string]interface{}{
			"model": result.Model.Name,
		},
		Metadata: map[string]string{
			"task_id":  req.TaskID,
			"hint":     req.ModelHint,
			"priority": req.Priority,
		},
	}

	// Retry logic with exponential backoff
	maxRetries := r.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 0 // No retries
	}

	var lastErr error
	backoff := time.Duration(r.config.RetryBackoffMs) * time.Millisecond
	maxBackoff := time.Duration(r.config.RetryMaxBackoffMs) * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Call provider
		provResp, err := prov.Generate(ctx, provReq)
		if err == nil && provResp.Error == "" {
			return provResp, nil
		}

		lastErr = err
		if err == nil && provResp.Error != "" {
			lastErr = fmt.Errorf("provider returned error: %s", provResp.Error)
		}

		// Don't retry on last attempt
		if attempt == maxRetries {
			break
		}

		// Check if error is retryable (network errors, timeouts, rate limits)
		if !r.isRetryableError(lastErr) {
			return nil, lastErr
		}

		// Wait with exponential backoff before retry
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Double the backoff for next retry, up to max
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	return nil, fmt.Errorf("all retry attempts failed (tried %d times): %w", maxRetries+1, lastErr)
}

// generateWithFallback attempts generation with fallback providers
func (r *Router) generateWithFallback(ctx context.Context, req GenerateRequest, primaryResult *RoutingResult, startTime time.Time) (*GenerateResponse, error) {
	// Get all available models sorted by score
	routing := RoutingRequest{
		ModelHint:   req.ModelHint,
		Complexity:  req.Complexity,
		Priority:    req.Priority,
		ContextSize: req.ContextSize,
	}

	candidates := r.getCandidateModels(routing)
	scored := r.scoreModels(candidates, routing)

	// Try each candidate in order (skip the primary that already failed)
	for i := range scored {
		model := scored[i]
		if model.ID == primaryResult.Model.ID {
			continue // Skip primary model that already failed
		}

		// Create result for this fallback model
		fallbackResult := &RoutingResult{
			Model:           model,
			Reason:          fmt.Sprintf("Fallback after primary failure: %s", primaryResult.Model.ID),
			EstimatedCost:   (float64(r.estimateTokens(routing)) / 1000000.0) * model.CostPerMToken,
			EstimatedTokens: r.estimateTokens(routing),
		}

		// Try this fallback model with retries
		provResp, err := r.generateWithRetry(ctx, req, fallbackResult)
		if err == nil && provResp.Error == "" {
			// Success with fallback!
			actualCost := (float64(provResp.TokensUsed) / 1000000.0) * model.CostPerMToken

			// Record usage
			usage := Usage{
				Model:     model.ID,
				Provider:  model.Provider,
				Tokens:    provResp.TokensUsed,
				CostUSD:   actualCost,
				LatencyMs: int(time.Since(startTime).Milliseconds()),
				Timestamp: time.Now(),
				TaskID:    req.TaskID,
				Success:   true,
			}
			r.RecordUsage(usage)

			return &GenerateResponse{
				Content:         provResp.Content,
				Model:           model.ID,
				Provider:        model.Provider,
				TokensUsed:      provResp.TokensUsed,
				InputTokens:     provResp.InputTokens,
				OutputTokens:    provResp.OutputTokens,
				CostUSD:         actualCost,
				Latency:         provResp.Latency,
				FinishReason:    provResp.FinishReason,
				SelectionReason: fmt.Sprintf("Fallback: %s (primary %s failed)", model.ID, primaryResult.Model.ID),
				ToolCalls:       provResp.ToolCalls,
				Error:           provResp.Error,
			}, nil
		}
	}

	return nil, fmt.Errorf("all fallback providers failed")
}

// isRetryableError checks if an error is transient and worth retrying
func (r *Router) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Network and timeout errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "network") {
		return true
	}

	// Rate limiting and service unavailability
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "service unavailable") ||
		strings.Contains(errStr, "too many requests") {
		return true
	}

	// Context errors are not retryable
	if strings.Contains(errStr, "context") {
		return false
	}

	// Authentication and authorization errors are not retryable
	if strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "invalid api key") {
		return false
	}

	// Default: don't retry unknown errors
	return false
}

// streamWithRetry attempts streaming with exponential backoff retry logic
func (r *Router) streamWithRetry(ctx context.Context, req GenerateRequest, result *RoutingResult) (<-chan provider.StreamChunk, *RoutingResult, error) {
	// Get provider from registry
	providerName := r.getProviderName(result.Model.Provider)
	prov, err := r.registry.Get(providerName)
	if err != nil {
		return nil, nil, fmt.Errorf("provider %s not available: %w", providerName, err)
	}

	// Check if provider supports streaming
	caps := prov.GetCapabilities()
	if !caps.SupportsStreaming {
		return nil, nil, fmt.Errorf("provider %s does not support streaming", providerName)
	}

	// Build provider request
	provReq := &provider.GenerateRequest{
		Prompt:       req.Prompt,
		SystemPrompt: req.SystemPrompt,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		TopP:         req.TopP,
		Tools:        req.Tools,
		Context:      req.Context,
		Config: map[string]interface{}{
			"model": result.Model.Name,
		},
		Metadata: map[string]string{
			"task_id":  req.TaskID,
			"hint":     req.ModelHint,
			"priority": req.Priority,
		},
	}

	// Retry logic with exponential backoff
	maxRetries := r.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 0 // No retries
	}

	var lastErr error
	backoff := time.Duration(r.config.RetryBackoffMs) * time.Millisecond
	maxBackoff := time.Duration(r.config.RetryMaxBackoffMs) * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Call provider stream
		provStream, err := prov.Stream(ctx, provReq)
		if err == nil {
			return provStream, result, nil
		}

		lastErr = err

		// Don't retry on last attempt
		if attempt == maxRetries {
			break
		}

		// Check if error is retryable
		if !r.isRetryableError(lastErr) {
			return nil, nil, lastErr
		}

		// Wait with exponential backoff before retry
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(backoff):
			// Double the backoff for next retry, up to max
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	return nil, nil, fmt.Errorf("all streaming retry attempts failed (tried %d times): %w", maxRetries+1, lastErr)
}

// streamWithFallback tries alternative providers when primary streaming fails
func (r *Router) streamWithFallback(ctx context.Context, req GenerateRequest, primaryResult *RoutingResult, startTime time.Time) (<-chan StreamChunk, error) {
	// Get all available models sorted by score
	routing := RoutingRequest{
		ModelHint:   req.ModelHint,
		Complexity:  req.Complexity,
		Priority:    req.Priority,
		ContextSize: req.ContextSize,
	}

	candidates := r.getCandidateModels(routing)
	scored := r.scoreModels(candidates, routing)

	// Try each candidate in order (skip the primary that already failed)
	for i := range scored {
		model := scored[i]
		if model.ID == primaryResult.Model.ID {
			continue // Skip primary model that already failed
		}

		// Create result for this fallback model
		fallbackResult := &RoutingResult{
			Model:           model,
			Reason:          fmt.Sprintf("Fallback after primary streaming failure: %s", primaryResult.Model.ID),
			EstimatedCost:   (float64(r.estimateTokens(routing)) / 1000000.0) * model.CostPerMToken,
			EstimatedTokens: r.estimateTokens(routing),
		}

		// Try this fallback model with retries
		provStream, streamResult, err := r.streamWithRetry(ctx, req, fallbackResult)
		if err == nil {
			// Success with fallback!
			// Create output channel
			outChan := make(chan StreamChunk, 10)

			// Forward stream chunks with usage tracking
			go func() {
				defer close(outChan)
				var totalTokens int

				for chunk := range provStream {
					outChan <- StreamChunk{
						Content: chunk.Content,
						Delta:   chunk.Delta,
						Done:    chunk.Done,
						Error:   chunk.Error,
					}

					if chunk.Done {
						totalTokens = chunk.TokensUsed
					}
				}

				// Record usage after stream completes
				if totalTokens > 0 {
					actualCost := (float64(totalTokens) / 1000000.0) * model.CostPerMToken
					usage := Usage{
						Model:     model.ID,
						Provider:  model.Provider,
						Tokens:    totalTokens,
						CostUSD:   actualCost,
						LatencyMs: int(time.Since(startTime).Milliseconds()),
						Timestamp: time.Now(),
						TaskID:    req.TaskID,
						Success:   true,
					}
					r.RecordUsage(usage)
				}
			}()

			return outChan, nil
		}

		// Continue to next fallback if this one failed
		_ = streamResult // Avoid unused variable warning
	}

	return nil, fmt.Errorf("all fallback providers failed for streaming")
}
