package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestIntegration_OllamaProvider(t *testing.T) {
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

	// Create provider config
	config := &ProviderConfig{
		Name:    "ollama-test",
		Type:    ProviderTypeCLI,
		Enabled: true,
		Source:  "local",
		Version: "1.0.0",
		Config: map[string]interface{}{
			"path":        providerPath,
			"trust_level": "community",
			"capabilities": map[string]interface{}{
				"streaming":        false,
				"tools":            false,
				"multi_turn":       true,
				"max_context_tokens": 8192,
			},
		},
		Models: map[string]string{
			"fast":    "llama3.2",
			"codegen": "llama3.2",
		},
	}

	// Create registry and load provider
	registry := NewRegistry()
	if err := registry.LoadFromConfig(config); err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	// Get provider from registry
	provider, err := registry.Get("ollama-test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Check provider info
	info := provider.GetInfo()
	if info.Name != "ollama-test" {
		t.Errorf("GetInfo().Name = %s, want ollama-test", info.Name)
	}
	if info.Type != ProviderTypeCLI {
		t.Errorf("GetInfo().Type = %s, want %s", info.Type, ProviderTypeCLI)
	}

	// Check provider capabilities
	caps := provider.GetCapabilities()
	if caps == nil {
		t.Fatal("GetCapabilities() returned nil")
	}
	if caps.SupportsMultiTurn != true {
		t.Error("Expected SupportsMultiTurn to be true")
	}

	// Check provider availability
	if !provider.IsAvailable() {
		t.Error("IsAvailable() = false, want true")
	}

	// Test health check
	ctx := context.Background()
	if err := provider.Health(ctx); err != nil {
		t.Errorf("Health() error = %v", err)
	}

	// Test generation
	req := &GenerateRequest{
		Prompt:      "What is 2 + 2? Answer with just the number.",
		Temperature: 0.1,
		Config: map[string]interface{}{
			"model": "llama3.2",
		},
	}

	resp, err := provider.Generate(ctx, req)
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

	if resp.Model != "llama3.2" {
		t.Errorf("Response model = %s, want llama3.2", resp.Model)
	}

	if resp.Provider != "ollama" {
		t.Errorf("Response provider = %s, want ollama", resp.Provider)
	}

	if resp.TokensUsed == 0 {
		t.Error("Response TokensUsed = 0, expected > 0")
	}

	if resp.Latency == 0 {
		t.Error("Response Latency = 0, expected > 0")
	}

	if resp.FinishReason == "" {
		t.Error("Response FinishReason is empty")
	}

	t.Logf("Generated response: %s (tokens: %d, latency: %v)", resp.Content, resp.TokensUsed, resp.Latency)

	// Test multi-turn conversation
	req2 := &GenerateRequest{
		Prompt:      "What about 3 + 3?",
		Temperature: 0.1,
		Context: []Message{
			{Role: "user", Content: "What is 2 + 2?"},
			{Role: "assistant", Content: resp.Content},
		},
		Config: map[string]interface{}{
			"model": "llama3.2",
		},
	}

	resp2, err := provider.Generate(ctx, req2)
	if err != nil {
		t.Fatalf("Generate() second request error = %v", err)
	}

	if resp2 == nil {
		t.Fatal("Generate() second request returned nil response")
	}

	if resp2.Content == "" {
		t.Error("Second response content is empty")
	}

	t.Logf("Second response: %s (tokens: %d, latency: %v)", resp2.Content, resp2.TokensUsed, resp2.Latency)

	// Test provider removal
	if err := registry.Remove("ollama-test"); err != nil {
		t.Errorf("Remove() error = %v", err)
	}

	// Verify provider is removed
	if _, err := registry.Get("ollama-test"); err == nil {
		t.Error("Get() after Remove() should return error")
	}
}

func TestIntegration_ProviderRegistry(t *testing.T) {
	registry := NewRegistry()

	// Test listing empty registry
	names := registry.List()
	if len(names) != 0 {
		t.Errorf("List() on empty registry = %v, want []", names)
	}

	// Create mock provider config (disabled)
	config := &ProviderConfig{
		Name:    "mock-disabled",
		Type:    ProviderTypeCLI,
		Enabled: false, // Disabled
		Config: map[string]interface{}{
			"path": "/fake/path",
		},
	}

	// Load disabled provider (should be skipped)
	err := registry.LoadFromConfig(config)
	if err != nil {
		t.Errorf("LoadFromConfig() with disabled provider error = %v, want nil", err)
	}

	// Verify disabled provider was not registered
	names = registry.List()
	if len(names) != 0 {
		t.Errorf("List() after loading disabled provider = %v, want []", names)
	}

	// Test loading provider with missing name
	config2 := &ProviderConfig{
		Type:    ProviderTypeCLI,
		Enabled: true,
	}
	err = registry.LoadFromConfig(config2)
	if err == nil {
		t.Error("LoadFromConfig() with empty name should return error")
	}

	// Test loading provider with unknown API provider name
	config3 := &ProviderConfig{
		Name:    "unknown-api",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
	}
	err = registry.LoadFromConfig(config3)
	if err == nil {
		t.Error("LoadFromConfig() with unknown API provider should return error")
	}

	// Test CloseAll on empty registry
	if err := registry.CloseAll(); err != nil {
		t.Errorf("CloseAll() on empty registry error = %v", err)
	}
}

func TestExecutableProvider_IsAvailable(t *testing.T) {
	// Test with non-existent executable
	config := &ProviderConfig{
		Name: "fake",
		Type: ProviderTypeCLI,
		Config: map[string]interface{}{
			"path": "/non/existent/path",
		},
	}

	_, err := NewExecutableProvider("/non/existent/path", config)
	if err == nil {
		t.Error("NewExecutableProvider() with non-existent path should return error")
	}
}

func TestExecutableProvider_Stream_NotSupported(t *testing.T) {
	// Create a mock provider with streaming disabled
	config := &ProviderConfig{
		Name: "test",
		Type: ProviderTypeCLI,
		Config: map[string]interface{}{
			"path": "echo", // Use echo as a mock executable
		},
	}

	provider, err := NewExecutableProvider("echo", config)
	if err != nil {
		t.Fatalf("NewExecutableProvider() error = %v", err)
	}

	// Try to stream (should fail)
	ctx := context.Background()
	req := &GenerateRequest{Prompt: "test"}

	_, err = provider.Stream(ctx, req)
	if err == nil {
		t.Error("Stream() should return error when streaming not supported")
	}
}

func TestExecutableProvider_GenerateTimeout(t *testing.T) {
	// Skip this test in short mode
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	// Create a provider that sleeps (to test timeout)
	config := &ProviderConfig{
		Name: "slow",
		Type: ProviderTypeCLI,
		Config: map[string]interface{}{
			"path": "sleep",
			"args": []interface{}{"10"}, // Sleep for 10 seconds
		},
	}

	provider, err := NewExecutableProvider("sleep", config)
	if err != nil {
		t.Fatalf("NewExecutableProvider() error = %v", err)
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &GenerateRequest{Prompt: "test"}

	// This should timeout
	_, err = provider.Generate(ctx, req)
	if err == nil {
		t.Error("Generate() with timeout should return error")
	}
}

func TestIntegration_OpenAIProvider(t *testing.T) {
	// Create mock OpenAI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" {
			resp := openAIResponse{
				ID:      "chatcmpl-123",
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   "gpt-4o-mini",
				Choices: []openAIChoice{
					{
						Index: 0,
						Message: openAIMessage{
							Role:    "assistant",
							Content: "Hello from OpenAI!",
						},
						FinishReason: "stop",
					},
				},
				Usage: openAIUsage{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	// Create provider config
	config := &ProviderConfig{
		Name:    "openai",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Version: "1.0.0",
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"base_url": server.URL,
			"capabilities": map[string]interface{}{
				"streaming":          true,
				"tools":              true,
				"multi_turn":         true,
				"max_context_tokens": 128000,
			},
		},
	}

	// Create registry and load provider
	registry := NewRegistry()
	if err := registry.LoadFromConfig(config); err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	// Get provider from registry
	provider, err := registry.Get("openai")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Check provider info
	info := provider.GetInfo()
	if info.Name != "openai" {
		t.Errorf("GetInfo().Name = %s, want openai", info.Name)
	}
	if info.Type != ProviderTypeAPI {
		t.Errorf("GetInfo().Type = %s, want %s", info.Type, ProviderTypeAPI)
	}

	// Check provider capabilities
	caps := provider.GetCapabilities()
	if caps == nil {
		t.Fatal("GetCapabilities() returned nil")
	}
	if !caps.SupportsStreaming {
		t.Error("Expected SupportsStreaming to be true")
	}
	if !caps.SupportsTools {
		t.Error("Expected SupportsTools to be true")
	}

	// Check provider availability
	if !provider.IsAvailable() {
		t.Error("IsAvailable() = false, want true")
	}

	// Test generation
	ctx := context.Background()
	req := &GenerateRequest{
		Prompt:      "Hello!",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := provider.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Generate() returned nil response")
	}

	if resp.Content != "Hello from OpenAI!" {
		t.Errorf("Response content = %s, want 'Hello from OpenAI!'", resp.Content)
	}

	if resp.Model != "gpt-4o-mini" {
		t.Errorf("Response model = %s, want gpt-4o-mini", resp.Model)
	}

	if resp.Provider != "openai" {
		t.Errorf("Response provider = %s, want openai", resp.Provider)
	}

	if resp.TokensUsed != 15 {
		t.Errorf("Response TokensUsed = %d, want 15", resp.TokensUsed)
	}

	t.Logf("Generated response: %s (tokens: %d, latency: %v)", resp.Content, resp.TokensUsed, resp.Latency)

	// Test provider removal
	if err := registry.Remove("openai"); err != nil {
		t.Errorf("Remove() error = %v", err)
	}

	// Verify provider is removed
	if _, err := registry.Get("openai"); err == nil {
		t.Error("Get() after Remove() should return error")
	}
}

func TestIntegration_AnthropicProvider(t *testing.T) {
	// Create mock Anthropic server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/messages" {
			resp := anthropicResponse{
				ID:    "msg_123",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-sonnet-3.5",
				Content: []anthropicContent{
					{
						Type: "text",
						Text: "Hello from Claude!",
					},
				},
				StopReason: "end_turn",
				Usage: anthropicUsage{
					InputTokens:  10,
					OutputTokens: 5,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	// Create provider config
	config := &ProviderConfig{
		Name:    "anthropic",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Version: "1.0.0",
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"base_url": server.URL,
			"capabilities": map[string]interface{}{
				"streaming":          true,
				"tools":              true,
				"multi_turn":         true,
				"vision":             true,
				"max_context_tokens": 200000,
			},
		},
	}

	// Create registry and load provider
	registry := NewRegistry()
	if err := registry.LoadFromConfig(config); err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	// Get provider from registry
	provider, err := registry.Get("anthropic")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Check provider info
	info := provider.GetInfo()
	if info.Name != "anthropic" {
		t.Errorf("GetInfo().Name = %s, want anthropic", info.Name)
	}
	if info.Type != ProviderTypeAPI {
		t.Errorf("GetInfo().Type = %s, want %s", info.Type, ProviderTypeAPI)
	}

	// Check provider capabilities
	caps := provider.GetCapabilities()
	if caps == nil {
		t.Fatal("GetCapabilities() returned nil")
	}
	if !caps.SupportsStreaming {
		t.Error("Expected SupportsStreaming to be true")
	}
	if !caps.SupportsVision {
		t.Error("Expected SupportsVision to be true")
	}

	// Check provider availability
	if !provider.IsAvailable() {
		t.Error("IsAvailable() = false, want true")
	}

	// Test generation
	ctx := context.Background()
	req := &GenerateRequest{
		Prompt:      "Hello!",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := provider.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Generate() returned nil response")
	}

	if resp.Content != "Hello from Claude!" {
		t.Errorf("Response content = %s, want 'Hello from Claude!'", resp.Content)
	}

	if resp.Model != "claude-sonnet-3.5" {
		t.Errorf("Response model = %s, want claude-sonnet-3.5", resp.Model)
	}

	if resp.Provider != "anthropic" {
		t.Errorf("Response provider = %s, want anthropic", resp.Provider)
	}

	if resp.TokensUsed != 15 {
		t.Errorf("Response TokensUsed = %d, want 15", resp.TokensUsed)
	}

	t.Logf("Generated response: %s (tokens: %d, latency: %v)", resp.Content, resp.TokensUsed, resp.Latency)

	// Test provider removal
	if err := registry.Remove("anthropic"); err != nil {
		t.Errorf("Remove() error = %v", err)
	}

	// Verify provider is removed
	if _, err := registry.Get("anthropic"); err == nil {
		t.Error("Get() after Remove() should return error")
	}
}

func TestIntegration_MultiProviderRegistry(t *testing.T) {
	// Create mock servers for both providers
	openaiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Model: "gpt-4o-mini",
			Choices: []openAIChoice{
				{Message: openAIMessage{Content: "From OpenAI"}},
			},
			Usage: openAIUsage{TotalTokens: 10},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer openaiServer.Close()

	anthropicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := anthropicResponse{
			Model: "claude-sonnet-3.5",
			Content: []anthropicContent{
				{Type: "text", Text: "From Anthropic"},
			},
			Usage: anthropicUsage{InputTokens: 5, OutputTokens: 5},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer anthropicServer.Close()

	// Create registry
	registry := NewRegistry()

	// Load OpenAI provider
	openaiConfig := &ProviderConfig{
		Name:    "openai",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key":  "openai-key",
			"base_url": openaiServer.URL,
		},
	}
	if err := registry.LoadFromConfig(openaiConfig); err != nil {
		t.Fatalf("LoadFromConfig(openai) error = %v", err)
	}

	// Load Anthropic provider
	anthropicConfig := &ProviderConfig{
		Name:    "anthropic",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key":  "anthropic-key",
			"base_url": anthropicServer.URL,
		},
	}
	if err := registry.LoadFromConfig(anthropicConfig); err != nil {
		t.Fatalf("LoadFromConfig(anthropic) error = %v", err)
	}

	// Verify both providers are registered
	names := registry.List()
	if len(names) != 2 {
		t.Errorf("List() = %v, want 2 providers", names)
	}

	// Test OpenAI provider
	openaiProvider, err := registry.Get("openai")
	if err != nil {
		t.Fatalf("Get(openai) error = %v", err)
	}

	ctx := context.Background()
	openaiResp, err := openaiProvider.Generate(ctx, &GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("OpenAI Generate() error = %v", err)
	}
	if openaiResp.Content != "From OpenAI" {
		t.Errorf("OpenAI response = %s, want 'From OpenAI'", openaiResp.Content)
	}

	// Test Anthropic provider
	anthropicProvider, err := registry.Get("anthropic")
	if err != nil {
		t.Fatalf("Get(anthropic) error = %v", err)
	}

	anthropicResp, err := anthropicProvider.Generate(ctx, &GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Anthropic Generate() error = %v", err)
	}
	if anthropicResp.Content != "From Anthropic" {
		t.Errorf("Anthropic response = %s, want 'From Anthropic'", anthropicResp.Content)
	}

	// Test CloseAll
	if err := registry.CloseAll(); err != nil {
		t.Errorf("CloseAll() error = %v", err)
	}

	// Verify all providers are removed
	names = registry.List()
	if len(names) != 0 {
		t.Errorf("List() after CloseAll() = %v, want []", names)
	}
}
