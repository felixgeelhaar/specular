package quickstart

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/felixgeelhaar/specular/internal/provider"
)

// VerificationResult holds the results of provider verification
type VerificationResult struct {
	Success  bool
	Provider string
	Response string
	Latency  time.Duration
	Error    string
}

// VerifyProvider tests that the configured provider is working correctly
func VerifyProvider(selection *ProviderSelection) (*VerificationResult, error) {
	result := &VerificationResult{
		Provider: selection.Name,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create the appropriate provider client
	client, err := createProviderClient(selection)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create provider client: %v", err)
		return result, nil
	}
	defer func() { _ = client.Close() }()

	// First try a health check
	if err := client.Health(ctx); err != nil {
		result.Error = fmt.Sprintf("provider health check failed: %v", err)
		return result, nil
	}

	// Run a simple test prompt
	start := time.Now()
	req := &provider.GenerateRequest{
		Prompt:    "Say 'Specular is ready!' in exactly three words.",
		MaxTokens: 50,
	}

	resp, err := client.Generate(ctx, req)
	if err != nil {
		result.Error = fmt.Sprintf("test generation failed: %v", err)
		return result, nil
	}

	result.Success = true
	result.Response = resp.Content
	result.Latency = time.Since(start)

	return result, nil
}

// createProviderClient creates the appropriate provider client for verification
func createProviderClient(selection *ProviderSelection) (provider.ProviderClient, error) {
	config := &provider.ProviderConfig{
		Name:    selection.Name,
		Enabled: true,
		Config:  make(map[string]interface{}),
	}

	switch selection.Name {
	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		config.Type = provider.ProviderTypeAPI
		config.Config["api_key"] = apiKey
		config.Config["model"] = "claude-sonnet-4-20250514"
		return provider.NewAnthropicProvider(config)

	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY not set")
		}
		config.Type = provider.ProviderTypeAPI
		config.Config["api_key"] = apiKey
		config.Config["model"] = "gpt-4o-mini"
		return provider.NewOpenAIProvider(config)

	case "gemini":
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY not set")
		}
		config.Type = provider.ProviderTypeAPI
		config.Config["api_key"] = apiKey
		config.Config["model"] = "gemini-2.0-flash"
		return provider.NewGeminiProvider(config)

	case "ollama":
		config.Type = provider.ProviderTypeAPI
		config.Config["endpoint"] = "http://localhost:11434"
		config.Config["model"] = "llama3.2"
		// Ollama uses a different provider pattern - use executable provider
		// For now, skip full verification for Ollama
		return nil, fmt.Errorf("ollama verification not yet implemented - provider detected successfully")

	default:
		return nil, fmt.Errorf("unsupported provider for verification: %s", selection.Name)
	}
}

// QuickVerify performs a minimal check without full generation
// This is faster and uses no API tokens for paid providers
func QuickVerify(selection *ProviderSelection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := createProviderClient(selection)
	if err != nil {
		// Special case for ollama - detection is sufficient
		if selection.Name == "ollama" {
			return nil
		}
		return fmt.Errorf("failed to create provider: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Just check if provider is available and healthy
	if !client.IsAvailable() {
		return fmt.Errorf("provider %s is not available", selection.Name)
	}

	if err := client.Health(ctx); err != nil {
		return fmt.Errorf("provider health check failed: %w", err)
	}

	return nil
}
