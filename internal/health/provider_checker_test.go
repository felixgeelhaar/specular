package health

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/specular/internal/provider"
)

// mockProvider implements provider.ProviderClient for testing
type mockProvider struct {
	name      string
	available bool
	healthy   bool
	healthErr error
}

func (m *mockProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProvider) Stream(ctx context.Context, req *provider.GenerateRequest) (<-chan provider.StreamChunk, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProvider) GetCapabilities() *provider.ProviderCapabilities {
	return &provider.ProviderCapabilities{}
}

func (m *mockProvider) GetInfo() *provider.ProviderInfo {
	return &provider.ProviderInfo{
		Name:    m.name,
		Version: "1.0.0",
		Type:    provider.ProviderTypeAPI,
	}
}

func (m *mockProvider) IsAvailable() bool {
	return m.available
}

func (m *mockProvider) Health(ctx context.Context) error {
	if m.healthErr != nil {
		return m.healthErr
	}
	if !m.healthy {
		return errors.New("unhealthy")
	}
	return nil
}

func (m *mockProvider) Close() error {
	return nil
}

func TestNewProviderChecker(t *testing.T) {
	provider1 := &mockProvider{name: "test1", available: true, healthy: true}
	provider2 := &mockProvider{name: "test2", available: true, healthy: true}

	checker := NewProviderChecker(provider1, provider2)

	if checker == nil {
		t.Fatal("NewProviderChecker returned nil")
	}

	if len(checker.providers) != 2 {
		t.Errorf("providers count = %d, want 2", len(checker.providers))
	}
}

func TestProviderCheckerName(t *testing.T) {
	checker := NewProviderChecker()

	name := checker.Name()
	if name != "ai-providers" {
		t.Errorf("Name() = %q, want %q", name, "ai-providers")
	}
}

func TestProviderCheckerCheckNoProviders(t *testing.T) {
	checker := NewProviderChecker()
	ctx := context.Background()

	result := checker.Check(ctx)

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want %v for no providers", result.Status, StatusUnhealthy)
	}

	if result.Message != "no AI providers configured" {
		t.Errorf("Message = %q, want %q", result.Message, "no AI providers configured")
	}

	if count, ok := result.Details["provider_count"].(int); !ok || count != 0 {
		t.Errorf("Details[provider_count] = %v, want 0", result.Details["provider_count"])
	}

	if _, ok := result.Details["suggestion"]; !ok {
		t.Error("Unhealthy result should include suggestion")
	}
}

func TestProviderCheckerCheckAllHealthy(t *testing.T) {
	provider1 := &mockProvider{name: "claude", available: true, healthy: true}
	provider2 := &mockProvider{name: "openai", available: true, healthy: true}

	checker := NewProviderChecker(provider1, provider2)
	ctx := context.Background()

	result := checker.Check(ctx)

	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusHealthy)
	}

	if result.Message != "all providers healthy (2/2)" {
		t.Errorf("Message = %q, want %q", result.Message, "all providers healthy (2/2)")
	}

	if count, ok := result.Details["total_providers"].(int); !ok || count != 2 {
		t.Errorf("Details[total_providers] = %v, want 2", result.Details["total_providers"])
	}

	if count, ok := result.Details["healthy_providers"].(int); !ok || count != 2 {
		t.Errorf("Details[healthy_providers] = %v, want 2", result.Details["healthy_providers"])
	}
}

func TestProviderCheckerCheckSomeDegraded(t *testing.T) {
	provider1 := &mockProvider{name: "claude", available: true, healthy: true}
	provider2 := &mockProvider{name: "openai", available: true, healthy: false}

	checker := NewProviderChecker(provider1, provider2)
	ctx := context.Background()

	result := checker.Check(ctx)

	if result.Status != StatusDegraded {
		t.Errorf("Status = %v, want %v", result.Status, StatusDegraded)
	}

	if result.Message != "some providers unhealthy (1/2)" {
		t.Errorf("Message = %q, want %q", result.Message, "some providers unhealthy (1/2)")
	}

	if count, ok := result.Details["healthy_providers"].(int); !ok || count != 1 {
		t.Errorf("Details[healthy_providers] = %v, want 1", result.Details["healthy_providers"])
	}
}

func TestProviderCheckerCheckAllUnhealthy(t *testing.T) {
	provider1 := &mockProvider{name: "claude", available: true, healthy: false}
	provider2 := &mockProvider{name: "openai", available: false}

	checker := NewProviderChecker(provider1, provider2)
	ctx := context.Background()

	result := checker.Check(ctx)

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusUnhealthy)
	}

	if result.Message != "no healthy providers (0/2)" {
		t.Errorf("Message = %q, want %q", result.Message, "no healthy providers (0/2)")
	}

	if count, ok := result.Details["healthy_providers"].(int); !ok || count != 0 {
		t.Errorf("Details[healthy_providers] = %v, want 0", result.Details["healthy_providers"])
	}
}

func TestProviderCheckerCheckUnavailable(t *testing.T) {
	provider1 := &mockProvider{name: "claude", available: false}
	provider2 := &mockProvider{name: "openai", available: false}

	checker := NewProviderChecker(provider1, provider2)
	ctx := context.Background()

	result := checker.Check(ctx)

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusUnhealthy)
	}

	if count, ok := result.Details["available_providers"].(int); !ok || count != 0 {
		t.Errorf("Details[available_providers] = %v, want 0", result.Details["available_providers"])
	}

	// Check provider details
	providers, ok := result.Details["providers"].(map[string]interface{})
	if !ok {
		t.Fatal("Details[providers] should be a map")
	}

	for name, details := range providers {
		providerMap, ok := details.(map[string]interface{})
		if !ok {
			t.Errorf("Provider %s details should be a map", name)
			continue
		}

		if available, ok := providerMap["available"].(bool); !ok || available {
			t.Errorf("Provider %s should be unavailable", name)
		}
	}
}

func TestProviderCheckerCheckHealthError(t *testing.T) {
	provider1 := &mockProvider{
		name:      "claude",
		available: true,
		healthy:   true,
		healthErr: errors.New("connection timeout"),
	}

	checker := NewProviderChecker(provider1)
	ctx := context.Background()

	result := checker.Check(ctx)

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusUnhealthy)
	}

	providers, ok := result.Details["providers"].(map[string]interface{})
	if !ok {
		t.Fatal("Details[providers] should be a map")
	}

	claudeDetails, ok := providers["claude"].(map[string]interface{})
	if !ok {
		t.Fatal("Provider claude details should be a map")
	}

	if healthy, ok := claudeDetails["healthy"].(bool); !ok || healthy {
		t.Error("Provider should be unhealthy when health check fails")
	}

	if errMsg, ok := claudeDetails["error"].(string); !ok || errMsg == "" {
		t.Error("Provider details should include error message")
	}
}

func TestProviderCheckerCheckMixedStatus(t *testing.T) {
	// 3 providers: 1 healthy, 1 unhealthy, 1 unavailable
	provider1 := &mockProvider{name: "claude", available: true, healthy: true}
	provider2 := &mockProvider{name: "openai", available: true, healthy: false}
	provider3 := &mockProvider{name: "gemini", available: false}

	checker := NewProviderChecker(provider1, provider2, provider3)
	ctx := context.Background()

	result := checker.Check(ctx)

	// Should be degraded (some but not all healthy)
	if result.Status != StatusDegraded {
		t.Errorf("Status = %v, want %v", result.Status, StatusDegraded)
	}

	if count, ok := result.Details["total_providers"].(int); !ok || count != 3 {
		t.Errorf("Details[total_providers] = %v, want 3", result.Details["total_providers"])
	}

	if count, ok := result.Details["available_providers"].(int); !ok || count != 2 {
		t.Errorf("Details[available_providers] = %v, want 2", result.Details["available_providers"])
	}

	if count, ok := result.Details["healthy_providers"].(int); !ok || count != 1 {
		t.Errorf("Details[healthy_providers] = %v, want 1", result.Details["healthy_providers"])
	}
}
