package health

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/specular/internal/provider"
)

// ProviderChecker checks the health of AI provider clients.
// It verifies that providers are available and can respond to health checks.
type ProviderChecker struct {
	providers []provider.ProviderClient
}

// NewProviderChecker creates a new provider health checker.
// The checker will test all provided providers in parallel.
func NewProviderChecker(providers ...provider.ProviderClient) *ProviderChecker {
	return &ProviderChecker{
		providers: providers,
	}
}

// Name returns the name of this health check.
func (c *ProviderChecker) Name() string {
	return "ai-providers"
}

// Check verifies that AI providers are available and healthy.
// Returns:
//   - Healthy if all providers are available and responding
//   - Degraded if some providers are unavailable
//   - Unhealthy if all providers are unavailable or if there are no providers
//
// The check calls IsAvailable() and Health() on each provider.
func (c *ProviderChecker) Check(ctx context.Context) *Result {
	if len(c.providers) == 0 {
		return Unhealthy("no AI providers configured").
			WithDetail("provider_count", 0).
			WithDetail("suggestion", "Configure at least one AI provider")
	}

	availableCount := 0
	healthyCount := 0
	providerDetails := make(map[string]interface{})

	for _, p := range c.providers {
		info := p.GetInfo()
		providerName := info.Name

		// Check availability first
		available := p.IsAvailable()
		if !available {
			providerDetails[providerName] = map[string]interface{}{
				"available": false,
				"healthy":   false,
				"type":      string(info.Type),
			}
			continue
		}
		availableCount++

		// Check health
		err := p.Health(ctx)
		if err != nil {
			providerDetails[providerName] = map[string]interface{}{
				"available": true,
				"healthy":   false,
				"error":     err.Error(),
				"type":      string(info.Type),
			}
			continue
		}
		healthyCount++

		providerDetails[providerName] = map[string]interface{}{
			"available": true,
			"healthy":   true,
			"type":      string(info.Type),
			"version":   info.Version,
		}
	}

	totalProviders := len(c.providers)
	result := &Result{
		Details: make(map[string]interface{}),
	}
	result.Details["total_providers"] = totalProviders
	result.Details["available_providers"] = availableCount
	result.Details["healthy_providers"] = healthyCount
	result.Details["providers"] = providerDetails

	// Determine status
	if healthyCount == 0 {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("no healthy providers (0/%d)", totalProviders)
		return result
	}

	if healthyCount < totalProviders {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("some providers unhealthy (%d/%d)", healthyCount, totalProviders)
		return result
	}

	result.Status = StatusHealthy
	result.Message = fmt.Sprintf("all providers healthy (%d/%d)", healthyCount, totalProviders)
	return result
}
