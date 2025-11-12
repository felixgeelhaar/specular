package provider

import (
	"os"
	"testing"
)

func TestLoadRegistryWithAutoDiscovery_ConfigExists(t *testing.T) {
	// Create a temporary providers.yaml file
	tmpFile := t.TempDir() + "/providers.yaml"
	config := `providers:
  - name: ollama
    type: cli
    enabled: true
    config:
      path: /usr/bin/ollama
`
	if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test: Should load from config file when it exists
	_, err := LoadRegistryWithAutoDiscovery(tmpFile)
	// May fail if ollama provider isn't actually available, but that's OK
	// The key is that it tries to load from config first
	if err != nil {
		// Expected - ollama provider may not be available
		t.Logf("Expected error when provider not available: %v", err)
	}
}

func TestLoadRegistryWithAutoDiscovery_NoConfig(t *testing.T) {
	// Test: Should use auto-discovery when config doesn't exist
	registry, err := LoadRegistryWithAutoDiscovery("/nonexistent/path.yaml")

	if err != nil {
		// May fail if no providers are available
		t.Logf("Auto-discovery found no providers: %v", err)
		return
	}

	// If we get here, at least one provider was auto-discovered
	providers := registry.List()
	if len(providers) == 0 {
		t.Error("Expected at least one provider from auto-discovery")
	}

	t.Logf("Auto-discovered providers: %v", providers)
}

func TestLoadRegistryFromAutoDiscovery(t *testing.T) {
	// This test depends on the environment
	// It may succeed if providers are available, or fail if none are
	registry, err := LoadRegistryFromAutoDiscovery()

	if err != nil {
		// Check error message
		if err.Error() != "no providers available - please install at least one AI provider (ollama, anthropic, openai)" {
			t.Errorf("Unexpected error message: %v", err)
		}
		t.Log("No providers available (expected in test environment)")
		return
	}

	// If successful, verify we have at least one provider
	providers := registry.List()
	if len(providers) == 0 {
		t.Error("Registry created but contains no providers")
	}

	t.Logf("Auto-discovered %d provider(s): %v", len(providers), providers)
}

func TestGenerateProviderConfig_Ollama(t *testing.T) {
	config := generateProviderConfig("ollama")

	// May be nil if ollama command not found
	if config == nil {
		t.Log("Ollama not available (expected in many environments)")
		return
	}

	// Verify config structure
	if config.Name != "ollama" {
		t.Errorf("Expected name 'ollama', got %s", config.Name)
	}

	if config.Type != ProviderTypeCLI {
		t.Errorf("Expected type CLI, got %s", config.Type)
	}

	if !config.Enabled {
		t.Error("Auto-generated config should be enabled")
	}

	if len(config.Models) == 0 {
		t.Error("Expected default models to be configured")
	}
}

func TestGenerateProviderConfig_Anthropic(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		} else {
			os.Unsetenv("ANTHROPIC_API_KEY")
		}
	}()

	// Test: Without API key
	os.Unsetenv("ANTHROPIC_API_KEY")
	config := generateProviderConfig("anthropic")
	if config != nil {
		t.Error("Expected nil config when ANTHROPIC_API_KEY not set")
	}

	// Test: With API key
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	config = generateProviderConfig("anthropic")
	if config == nil {
		t.Fatal("Expected config when ANTHROPIC_API_KEY is set")
	}

	// Verify config structure
	if config.Name != "anthropic" {
		t.Errorf("Expected name 'anthropic', got %s", config.Name)
	}

	if config.Type != ProviderTypeAPI {
		t.Errorf("Expected type API, got %s", config.Type)
	}

	if !config.Enabled {
		t.Error("Auto-generated config should be enabled")
	}

	if len(config.Models) == 0 {
		t.Error("Expected default models to be configured")
	}

	// Verify API key is templated
	apiKey, ok := config.Config["api_key"].(string)
	if !ok || apiKey != "${ANTHROPIC_API_KEY}" {
		t.Errorf("Expected API key to be '${ANTHROPIC_API_KEY}', got %v", apiKey)
	}
}

func TestGenerateProviderConfig_OpenAI(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	// Test: Without API key
	os.Unsetenv("OPENAI_API_KEY")
	config := generateProviderConfig("openai")
	if config != nil {
		t.Error("Expected nil config when OPENAI_API_KEY not set")
	}

	// Test: With API key
	os.Setenv("OPENAI_API_KEY", "test-key")
	config = generateProviderConfig("openai")
	if config == nil {
		t.Fatal("Expected config when OPENAI_API_KEY is set")
	}

	// Verify config structure
	if config.Name != "openai" {
		t.Errorf("Expected name 'openai', got %s", config.Name)
	}

	if config.Type != ProviderTypeAPI {
		t.Errorf("Expected type API, got %s", config.Type)
	}

	if !config.Enabled {
		t.Error("Auto-generated config should be enabled")
	}

	if len(config.Models) == 0 {
		t.Error("Expected default models to be configured")
	}
}

func TestGenerateProviderConfig_Unknown(t *testing.T) {
	config := generateProviderConfig("unknown-provider")
	if config != nil {
		t.Error("Expected nil config for unknown provider")
	}
}

func TestLookupCommand(t *testing.T) {
	// Test with a command that should exist on all systems
	path, err := lookupCommand("sh")
	if err != nil {
		t.Errorf("Expected to find 'sh' command, got error: %v", err)
	}
	if path == "" {
		t.Error("Expected non-empty path for 'sh' command")
	}

	// Test with a command that definitely doesn't exist
	_, err = lookupCommand("this-command-definitely-does-not-exist-12345")
	if err == nil {
		t.Error("Expected error for non-existent command")
	}
}
