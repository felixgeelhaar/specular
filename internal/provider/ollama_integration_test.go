package provider_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/provider/testhelpers"

	_ "github.com/felixgeelhaar/specular/internal/provider/native/ollama"
)

func TestIntegration_OllamaProvider(t *testing.T) {
	server := testhelpers.StartFakeOllamaServer(t)
	defer server.Close()

	config := &provider.ProviderConfig{
		Name:    "ollama",
		Type:    provider.ProviderTypeNative,
		Enabled: true,
		Source:  "builtin",
		Version: "1.0.0",
		Config: map[string]interface{}{
			"base_url": server.URL,
			"model":    "llama3.2",
		},
		Models: map[string]string{
			"fast":    "llama3.2",
			"codegen": "llama3.2",
		},
	}

	registry := provider.NewRegistry()
	if err := registry.LoadFromConfig(config); err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	providerClient, err := registry.Get("ollama")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	info := providerClient.GetInfo()
	if info.Name != "ollama" {
		t.Errorf("GetInfo().Name = %s, want ollama", info.Name)
	}
	if info.Type != provider.ProviderTypeNative {
		t.Errorf("GetInfo().Type = %s, want %s", info.Type, provider.ProviderTypeNative)
	}

	caps := providerClient.GetCapabilities()
	if caps == nil {
		t.Fatal("GetCapabilities() returned nil")
	}
	if !caps.SupportsMultiTurn {
		t.Error("Expected SupportsMultiTurn to be true")
	}

	if !providerClient.IsAvailable() {
		t.Error("IsAvailable() = false, want true")
	}

	ctx := context.Background()
	if err := providerClient.Health(ctx); err != nil {
		t.Errorf("Health() error = %v", err)
	}

	req := &provider.GenerateRequest{
		Prompt:      "What is 2 + 2? Answer with just the number.",
		Temperature: 0.1,
		Config: map[string]interface{}{
			"model": "llama3.2",
		},
	}

	resp, err := providerClient.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

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

	req2 := &provider.GenerateRequest{
		Prompt:      "What about 3 + 3?",
		Temperature: 0.1,
		Context: []provider.Message{
			{Role: "user", Content: "What is 2 + 2?"},
			{Role: "assistant", Content: resp.Content},
		},
		Config: map[string]interface{}{
			"model": "llama3.2",
		},
	}

	resp2, err := providerClient.Generate(ctx, req2)
	if err != nil {
		t.Fatalf("Generate() second request error = %v", err)
	}

	if resp2 == nil {
		t.Fatal("Generate() second response is nil")
	}

	if resp2.Content == "" {
		t.Error("Second response content is empty")
	}

	t.Logf("Second response: %s (tokens: %d, latency: %v)", resp2.Content, resp2.TokensUsed, resp2.Latency)

	if err := registry.Remove("ollama"); err != nil {
		t.Errorf("Remove() error = %v", err)
	}

	if _, err := registry.Get("ollama"); err == nil {
		t.Error("Get() after Remove() should return error")
	}
}
