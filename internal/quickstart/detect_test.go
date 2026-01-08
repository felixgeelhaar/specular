package quickstart

import (
	"os"
	"testing"
)

func TestDetectBestProvider_WithAnthropicKey(t *testing.T) {
	// Save current env
	original := os.Getenv("ANTHROPIC_API_KEY")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	// Set test key
	os.Setenv("ANTHROPIC_API_KEY", "test-key-123")

	provider, err := detectBestProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name != "anthropic" {
		t.Errorf("expected anthropic, got %s", provider.Name)
	}
	if provider.Type != "api" {
		t.Errorf("expected api type, got %s", provider.Type)
	}
	if !provider.Ready {
		t.Error("expected provider to be ready")
	}
}

func TestDetectBestProvider_WithOpenAIKey(t *testing.T) {
	// Save and clear Anthropic key
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	defer os.Setenv("ANTHROPIC_API_KEY", anthropicKey)
	os.Unsetenv("ANTHROPIC_API_KEY")

	// Save current OpenAI env
	original := os.Getenv("OPENAI_API_KEY")
	defer os.Setenv("OPENAI_API_KEY", original)

	// Set test key
	os.Setenv("OPENAI_API_KEY", "test-key-123")

	provider, err := detectBestProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name != "openai" {
		t.Errorf("expected openai, got %s", provider.Name)
	}
}

func TestDetectBestProvider_Priority(t *testing.T) {
	// Save all keys
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	defer func() {
		os.Setenv("ANTHROPIC_API_KEY", anthropicKey)
		os.Setenv("OPENAI_API_KEY", openaiKey)
		os.Setenv("GEMINI_API_KEY", geminiKey)
	}()

	// Set all keys - Anthropic should win due to priority
	os.Setenv("ANTHROPIC_API_KEY", "anthropic-key")
	os.Setenv("OPENAI_API_KEY", "openai-key")
	os.Setenv("GEMINI_API_KEY", "gemini-key")

	provider, err := detectBestProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name != "anthropic" {
		t.Errorf("expected anthropic (highest priority), got %s", provider.Name)
	}
}

func TestDetectProviderByName_Anthropic(t *testing.T) {
	// Save current env
	original := os.Getenv("ANTHROPIC_API_KEY")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	// Set test key
	os.Setenv("ANTHROPIC_API_KEY", "test-key-123")

	provider, err := DetectProviderByName("anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name != "anthropic" {
		t.Errorf("expected anthropic, got %s", provider.Name)
	}
}

func TestDetectProviderByName_MissingKey(t *testing.T) {
	// Save and clear env
	original := os.Getenv("ANTHROPIC_API_KEY")
	defer os.Setenv("ANTHROPIC_API_KEY", original)
	os.Unsetenv("ANTHROPIC_API_KEY")

	_, err := DetectProviderByName("anthropic")
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

func TestDetectProviderByName_Unknown(t *testing.T) {
	_, err := DetectProviderByName("unknown-provider")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestNoProviderError(t *testing.T) {
	err := &NoProviderError{Suggestion: "test suggestion"}
	expected := "no AI provider found: test suggestion"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestDetectContainerRuntime(t *testing.T) {
	// This test just verifies the function doesn't panic
	// Actual availability depends on system
	status := detectContainerRuntime()

	// Status should always have a warning if not available
	if !status.Available && status.Warning == "" {
		t.Error("expected warning when Docker not available")
	}
}
