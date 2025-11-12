// +build integration

package detect_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/felixgeelhaar/specular/internal/detect"
)

// TestDetectProviders tests AI provider detection
func TestDetectProviders(t *testing.T) {
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	// Check that Providers map is not nil
	if ctx.Providers == nil {
		t.Fatal("Providers map should not be nil")
	}

	// All expected providers should have entries
	expectedProviders := []string{"ollama", "claude", "openai", "gemini", "anthropic"}
	for _, name := range expectedProviders {
		if _, exists := ctx.Providers[name]; !exists {
			t.Errorf("Provider %s should exist in Providers map", name)
		}
	}

	t.Logf("Detected %d providers", len(ctx.Providers))
}

// TestDetectOllama tests Ollama detection
func TestDetectOllama(t *testing.T) {
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	info, exists := ctx.Providers["ollama"]
	if !exists {
		t.Fatal("ollama should exist in Providers map")
	}

	// Check basic fields
	if info.Name != "ollama" {
		t.Errorf("Name = %s, want ollama", info.Name)
	}

	if info.Type != "local" {
		t.Errorf("Type = %s, want local", info.Type)
	}

	// If Ollama CLI is available
	if _, err := exec.LookPath("ollama"); err == nil {
		if !info.Available {
			t.Error("Ollama should be detected as available when CLI is present")
		}
		if info.Version == "" {
			t.Error("Ollama version should be populated when available")
		}
		t.Logf("Ollama detected: version %s", info.Version)
	} else {
		if info.Available {
			t.Error("Ollama should not be available when CLI is not present")
		}
		t.Log("Ollama CLI not available")
	}
}

// TestDetectClaude tests Claude CLI detection
func TestDetectClaude(t *testing.T) {
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	info, exists := ctx.Providers["claude"]
	if !exists {
		t.Fatal("claude should exist in Providers map")
	}

	// Check basic fields
	if info.Name != "claude" {
		t.Errorf("Name = %s, want claude", info.Name)
	}

	if info.Type != "cli" {
		t.Errorf("Type = %s, want cli", info.Type)
	}

	if info.EnvVar != "ANTHROPIC_API_KEY" {
		t.Errorf("EnvVar = %s, want ANTHROPIC_API_KEY", info.EnvVar)
	}

	// Check environment variable detection
	hasAPIKey := os.Getenv("ANTHROPIC_API_KEY") != ""
	if info.EnvSet != hasAPIKey {
		t.Errorf("EnvSet = %v, want %v", info.EnvSet, hasAPIKey)
	}

	// If Claude CLI is available
	if _, err := exec.LookPath("claude"); err == nil {
		if !info.Available {
			t.Error("Claude should be detected as available when CLI is present")
		}
		t.Logf("Claude CLI detected: version %s, API key set: %v", info.Version, info.EnvSet)
	} else {
		if info.Available {
			t.Error("Claude should not be available when CLI is not present")
		}
		t.Logf("Claude CLI not available, API key set: %v", info.EnvSet)
	}
}

// TestDetectOpenAI tests OpenAI detection
func TestDetectOpenAI(t *testing.T) {
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	info, exists := ctx.Providers["openai"]
	if !exists {
		t.Fatal("openai should exist in Providers map")
	}

	// Check basic fields
	if info.Name != "openai" {
		t.Errorf("Name = %s, want openai", info.Name)
	}

	if info.EnvVar != "OPENAI_API_KEY" {
		t.Errorf("EnvVar = %s, want OPENAI_API_KEY", info.EnvVar)
	}

	// Check environment variable detection
	hasAPIKey := os.Getenv("OPENAI_API_KEY") != ""
	if info.EnvSet != hasAPIKey {
		t.Errorf("EnvSet = %v, want %v", info.EnvSet, hasAPIKey)
	}

	t.Logf("OpenAI: Available=%v, Type=%s, API key set=%v", info.Available, info.Type, info.EnvSet)
}

// TestDetectGemini tests Gemini detection
func TestDetectGemini(t *testing.T) {
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	info, exists := ctx.Providers["gemini"]
	if !exists {
		t.Fatal("gemini should exist in Providers map")
	}

	// Check basic fields
	if info.Name != "gemini" {
		t.Errorf("Name = %s, want gemini", info.Name)
	}

	if info.EnvVar != "GEMINI_API_KEY" {
		t.Errorf("EnvVar = %s, want GEMINI_API_KEY", info.EnvVar)
	}

	// Check environment variable detection
	hasAPIKey := os.Getenv("GEMINI_API_KEY") != ""
	if info.EnvSet != hasAPIKey {
		t.Errorf("EnvSet = %v, want %v", info.EnvSet, hasAPIKey)
	}

	t.Logf("Gemini: Available=%v, Type=%s, API key set=%v", info.Available, info.Type, info.EnvSet)
}

// TestDetectAnthropic tests Anthropic API detection
func TestDetectAnthropic(t *testing.T) {
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	info, exists := ctx.Providers["anthropic"]
	if !exists {
		t.Fatal("anthropic should exist in Providers map")
	}

	// Check basic fields
	if info.Name != "anthropic" {
		t.Errorf("Name = %s, want anthropic", info.Name)
	}

	if info.Type != "api" {
		t.Errorf("Type = %s, want api", info.Type)
	}

	if info.EnvVar != "ANTHROPIC_API_KEY" {
		t.Errorf("EnvVar = %s, want ANTHROPIC_API_KEY", info.EnvVar)
	}

	// Check environment variable detection
	hasAPIKey := os.Getenv("ANTHROPIC_API_KEY") != ""
	if info.EnvSet != hasAPIKey {
		t.Errorf("EnvSet = %v, want %v", info.EnvSet, hasAPIKey)
	}

	// Anthropic is API-only, so Available should match EnvSet
	if info.Available != info.EnvSet {
		t.Errorf("Available = %v, want %v (should match EnvSet for API-only provider)", info.Available, info.EnvSet)
	}

	t.Logf("Anthropic API: Available=%v, API key set=%v", info.Available, info.EnvSet)
}

// TestProviderFieldConsistency tests that all providers have consistent field patterns
func TestProviderFieldConsistency(t *testing.T) {
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	for name, info := range ctx.Providers {
		t.Run(name, func(t *testing.T) {
			// Name field should match map key
			if info.Name != name {
				t.Errorf("Provider %s: Name field = %s, want %s", name, info.Name, name)
			}

			// Type should be one of: local, cli, api
			validTypes := map[string]bool{"local": true, "cli": true, "api": true}
			if !validTypes[info.Type] {
				t.Errorf("Provider %s: invalid Type = %s", name, info.Type)
			}

			// If EnvVar is set, EnvSet should reflect actual env var state
			if info.EnvVar != "" {
				expected := os.Getenv(info.EnvVar) != ""
				if info.EnvSet != expected {
					t.Errorf("Provider %s: EnvSet = %v, want %v (env var %s)", name, info.EnvSet, expected, info.EnvVar)
				}
			}

			// If Available is true, either CLI should exist OR env var should be set (depending on type)
			if info.Available {
				switch info.Type {
				case "local", "cli":
					if _, err := exec.LookPath(name); err != nil {
						t.Errorf("Provider %s: Available=true but CLI not found", name)
					}
				case "api":
					if !info.EnvSet {
						t.Errorf("Provider %s: Available=true but API key not set", name)
					}
				}
			}

			t.Logf("Provider %s: Type=%s, Available=%v, Version=%s, EnvVar=%s, EnvSet=%v",
				name, info.Type, info.Available, info.Version, info.EnvVar, info.EnvSet)
		})
	}
}
