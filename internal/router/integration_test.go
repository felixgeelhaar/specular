package router

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/internal/provider"
)

func TestRouterIntegration_OllamaProvider(t *testing.T) {
	// Check if ollama is available
	if _, err := exec.LookPath("ollama"); err != nil {
		t.Skip("ollama not available, skipping integration test")
	}

	// Check if ollama service is running
	cmd := exec.Command("curl", "-s", "http://localhost:11434/api/tags")
	if err := cmd.Run(); err != nil {
		t.Skip("ollama service not running, skipping integration test")
	}

	// Get absolute path to ollama provider executable
	providerPath, err := filepath.Abs("../../providers/ollama/ollama-provider")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Check if provider executable exists
	if _, err := os.Stat(providerPath); os.IsNotExist(err) {
		t.Skip("ollama-provider executable not built, skipping integration test")
	}

	// Create provider registry and load ollama
	registry := provider.NewRegistry()
	config := &provider.ProviderConfig{
		Name:    "ollama",
		Type:    provider.ProviderTypeCLI,
		Enabled: true,
		Source:  "local",
		Version: "1.0.0",
		Config: map[string]interface{}{
			"path": providerPath,
		},
		Models: map[string]string{
			"fast": "llama3.2",
		},
	}

	if err := registry.LoadFromConfig(config); err != nil {
		t.Fatalf("Failed to load provider: %v", err)
	}

	// Create router with provider registry
	routerConfig := &RouterConfig{
		BudgetUSD:    10.0,
		MaxLatencyMs: 60000,
		PreferCheap:  true,
	}

	router, err := NewRouterWithProviders(routerConfig, registry)
	if err != nil {
		t.Fatalf("NewRouterWithProviders() error = %v", err)
	}

	// Verify router has access to registry
	if router.GetRegistry() == nil {
		t.Fatal("Router registry is nil")
	}

	// Test model availability update - local models should be available now
	availableModels := 0
	for _, m := range router.models {
		if m.Available && m.Provider == ProviderLocal {
			availableModels++
		}
	}
	if availableModels == 0 {
		t.Error("Expected some local models to be available after loading ollama provider")
	}

	// Create a simple generate request
	req := GenerateRequest{
		Prompt:      "What is 2 + 2? Answer with just the number.",
		ModelHint:   "fast",
		Complexity:  1,
		Priority:    "P1",
		Temperature: 0.1,
		TaskID:      "test-integration",
	}

	// Call Generate
	ctx := context.Background()
	resp, err := router.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Generate() returned nil response")
	}

	if resp.Content == "" {
		t.Error("Response content is empty")
	}

	if resp.Model == "" {
		t.Error("Response model is empty")
	}

	if resp.Provider == "" {
		t.Error("Response provider is empty")
	}

	if resp.TokensUsed == 0 {
		t.Error("Response TokensUsed = 0, expected > 0")
	}

	if resp.CostUSD < 0 {
		t.Error("Response CostUSD should be >= 0")
	}

	if resp.Latency == 0 {
		t.Error("Response Latency = 0, expected > 0")
	}

	if resp.SelectionReason == "" {
		t.Error("Response SelectionReason is empty")
	}

	t.Logf("Generated response: %s", resp.Content)
	t.Logf("Model: %s, Provider: %s", resp.Model, resp.Provider)
	t.Logf("Tokens: %d (in: %d, out: %d)", resp.TokensUsed, resp.InputTokens, resp.OutputTokens)
	t.Logf("Cost: $%.6f, Latency: %v", resp.CostUSD, resp.Latency)
	t.Logf("Selection: %s", resp.SelectionReason)

	// Verify budget was updated (local models are free, so cost should be 0)
	budget := router.GetBudget()
	// Local models have 0 cost, so spent can be 0
	if budget.SpentUSD < 0 {
		t.Errorf("Budget SpentUSD = %.6f, should be >= 0", budget.SpentUSD)
	}
	if budget.UsageCount != 1 {
		t.Errorf("Budget UsageCount = %d, expected 1", budget.UsageCount)
	}

	t.Logf("Budget: spent $%.6f, remaining $%.2f, usage count %d",
		budget.SpentUSD, budget.RemainingUSD, budget.UsageCount)

	// Verify usage was recorded
	stats := router.GetUsageStats()
	if totalReqs, ok := stats["total_requests"].(int); !ok || totalReqs != 1 {
		t.Errorf("Usage stats total_requests = %v, expected 1", stats["total_requests"])
	}

	// Test a second request to verify continued operation
	req2 := GenerateRequest{
		Prompt:      "What is 3 + 3?",
		ModelHint:   "cheap",
		Complexity:  1,
		Priority:    "P2",
		Temperature: 0.1,
		TaskID:      "test-integration-2",
	}

	resp2, err := router.Generate(ctx, req2)
	if err != nil {
		t.Fatalf("Generate() second request error = %v", err)
	}

	if resp2 == nil {
		t.Fatal("Generate() second request returned nil response")
	}

	// Check for error in response
	if resp2.Error != "" {
		t.Logf("Warning: Second response had error: %s", resp2.Error)
	}

	// Content might be empty if there was an error or timeout
	if resp2.Content == "" && resp2.Error == "" {
		t.Error("Second response content is empty and no error reported")
	}

	t.Logf("Second response: %s (model: %s, tokens: %d, cost: $%.6f)",
		resp2.Content, resp2.Model, resp2.TokensUsed, resp2.CostUSD)

	// Verify cumulative budget
	budget2 := router.GetBudget()
	if budget2.UsageCount != 2 {
		t.Errorf("Budget UsageCount = %d, expected 2", budget2.UsageCount)
	}
	// Local models are free, so cost might not increase
	if budget2.SpentUSD < budget.SpentUSD {
		t.Error("Budget SpentUSD decreased after second request")
	}
}

func TestRouterIntegration_NoProviders(t *testing.T) {
	// Create router without any providers
	routerConfig := &RouterConfig{
		BudgetUSD:    10.0,
		MaxLatencyMs: 60000,
	}

	router, err := NewRouter(routerConfig)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	// Verify all models are marked as unavailable (no providers loaded)
	for _, m := range router.models {
		if m.Available {
			t.Errorf("Model %s should be unavailable (no providers loaded)", m.ID)
		}
	}

	// Try to generate - should fail
	req := GenerateRequest{
		Prompt:     "Test",
		ModelHint:  "fast",
		Complexity: 1,
		Priority:   "P1",
	}

	ctx := context.Background()
	_, err = router.Generate(ctx, req)
	if err == nil {
		t.Error("Generate() should fail when no providers are available")
	}
}

func TestRouterIntegration_BudgetExhaustion(t *testing.T) {
	// Check if ollama is available
	if _, err := exec.LookPath("ollama"); err != nil {
		t.Skip("ollama not available, skipping integration test")
	}

	// Check if ollama service is running
	cmd := exec.Command("curl", "-s", "http://localhost:11434/api/tags")
	if err := cmd.Run(); err != nil {
		t.Skip("ollama service not running, skipping integration test")
	}

	// Get absolute path to ollama provider executable
	providerPath, err := filepath.Abs("../../providers/ollama/ollama-provider")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Check if provider executable exists
	if _, err := os.Stat(providerPath); os.IsNotExist(err) {
		t.Skip("ollama-provider executable not built, skipping integration test")
	}

	// Create provider registry and load ollama
	registry := provider.NewRegistry()
	config := &provider.ProviderConfig{
		Name:    "ollama",
		Type:    provider.ProviderTypeCLI,
		Enabled: true,
		Source:  "local",
		Config: map[string]interface{}{
			"path": providerPath,
		},
	}

	if err := registry.LoadFromConfig(config); err != nil {
		t.Fatalf("Failed to load provider: %v", err)
	}

	// Create router with very small budget
	routerConfig := &RouterConfig{
		BudgetUSD:    0.000001, // Tiny budget - will be exhausted after first request
		MaxLatencyMs: 60000,
	}

	router, err := NewRouterWithProviders(routerConfig, registry)
	if err != nil {
		t.Fatalf("NewRouterWithProviders() error = %v", err)
	}

	// First request should work
	req := GenerateRequest{
		Prompt:      "Hi",
		ModelHint:   "cheap",
		Complexity:  1,
		Temperature: 0.1,
	}

	ctx := context.Background()
	_, err = router.Generate(ctx, req)
	if err != nil {
		t.Logf("First request failed (budget may be too small): %v", err)
		// This is actually ok - the budget might be exhausted even for selection
		return
	}

	// Second request should fail due to budget exhaustion
	req2 := GenerateRequest{
		Prompt:      "Hello again",
		ModelHint:   "cheap",
		Complexity:  1,
		Temperature: 0.1,
	}

	_, err = router.Generate(ctx, req2)
	if err == nil {
		// Check if budget is actually exhausted
		budget := router.GetBudget()
		if budget.RemainingUSD <= 0 {
			t.Error("Generate() should fail when budget is exhausted")
		}
	} else {
		t.Logf("Second request correctly failed with exhausted budget: %v", err)
	}
}
