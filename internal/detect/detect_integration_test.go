//go:build integration

package detect

import (
	"os"
	"os/exec"
	"testing"
)

// TestDetectDocker tests Docker detection with real Docker installation
func TestDetectDocker(t *testing.T) {
	// Check if Docker is available in test environment
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available in test environment")
	}

	runtime := detectDocker()

	// Verify detection worked
	if !runtime.Available {
		t.Error("Docker should be detected as available")
	}

	// Version should be populated if Docker daemon is running
	if runtime.Running && runtime.Version == "" {
		t.Error("Docker version should be detected when daemon is running")
	}

	t.Logf("Docker detected: Available=%v, Running=%v, Version=%s",
		runtime.Available, runtime.Running, runtime.Version)
}

// TestDetectPodman tests Podman detection with real Podman installation
func TestDetectPodman(t *testing.T) {
	// Check if Podman is available in test environment
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("Podman not available in test environment")
	}

	runtime := detectPodman()

	// Verify detection worked
	if !runtime.Available {
		t.Error("Podman should be detected as available")
	}

	// Version should be populated
	if runtime.Version == "" {
		t.Error("Podman version should be detected")
	}

	t.Logf("Podman detected: Available=%v, Running=%v, Version=%s",
		runtime.Available, runtime.Running, runtime.Version)
}

// TestDetectOllama tests Ollama detection with real Ollama installation
func TestDetectOllama(t *testing.T) {
	// Check if Ollama is available in test environment
	if _, err := exec.LookPath("ollama"); err != nil {
		t.Skip("Ollama not available in test environment")
	}

	info := detectOllama()

	// Verify detection worked
	if !info.Available {
		t.Error("Ollama should be detected as available")
	}

	if info.Name != "ollama" {
		t.Errorf("Expected name 'ollama', got '%s'", info.Name)
	}

	if info.Type != "local" {
		t.Errorf("Expected type 'local', got '%s'", info.Type)
	}

	t.Logf("Ollama detected: Available=%v, Version=%s, Type=%s",
		info.Available, info.Version, info.Type)
}

// TestDetectClaude tests Claude CLI detection
func TestDetectClaude(t *testing.T) {
	// Check if Claude CLI is available in test environment
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("Claude CLI not available in test environment")
	}

	info := detectClaude()

	// Verify detection worked
	if !info.Available {
		t.Error("Claude should be detected as available")
	}

	if info.Name != "claude" {
		t.Errorf("Expected name 'claude', got '%s'", info.Name)
	}

	if info.Type != "cli" {
		t.Errorf("Expected type 'cli', got '%s'", info.Type)
	}

	if info.EnvVar != "ANTHROPIC_API_KEY" {
		t.Errorf("Expected EnvVar 'ANTHROPIC_API_KEY', got '%s'", info.EnvVar)
	}

	t.Logf("Claude detected: Available=%v, EnvSet=%v, Version=%s",
		info.Available, info.EnvSet, info.Version)
}

// TestDetectProviderWithCLI tests the generic provider detection function
func TestDetectProviderWithCLI(t *testing.T) {
	tests := []struct {
		name         string
		cliName      string
		providerName string
		envVar       string
		shouldSkip   func() bool
	}{
		{
			name:         "openai",
			cliName:      "openai",
			providerName: "openai",
			envVar:       "OPENAI_API_KEY",
			shouldSkip: func() bool {
				_, err := exec.LookPath("openai")
				return err != nil
			},
		},
		{
			name:         "gemini",
			cliName:      "gemini",
			providerName: "gemini",
			envVar:       "GEMINI_API_KEY",
			shouldSkip: func() bool {
				_, err := exec.LookPath("gemini")
				return err != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSkip() {
				t.Skipf("%s CLI not available in test environment", tt.cliName)
			}

			info := detectProviderWithCLI(tt.providerName, tt.cliName, tt.envVar)

			// Verify detection worked
			if !info.Available {
				t.Errorf("%s should be detected as available", tt.providerName)
			}

			if info.Name != tt.providerName {
				t.Errorf("Expected name '%s', got '%s'", tt.providerName, info.Name)
			}

			if info.EnvVar != tt.envVar {
				t.Errorf("Expected EnvVar '%s', got '%s'", tt.envVar, info.EnvVar)
			}

			t.Logf("%s detected: Available=%v, Type=%s, EnvSet=%v, Version=%s",
				tt.providerName, info.Available, info.Type, info.EnvSet, info.Version)
		})
	}
}

// TestDetectOpenAI tests OpenAI detection
func TestDetectOpenAI(t *testing.T) {
	// This test will work if either:
	// 1. OpenAI CLI is installed
	// 2. OPENAI_API_KEY environment variable is set

	// Save original env var
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	// Test with CLI if available
	if _, err := exec.LookPath("openai"); err == nil {
		info := detectOpenAI()

		if !info.Available {
			t.Error("OpenAI should be detected as available when CLI exists")
		}

		if info.Name != "openai" {
			t.Errorf("Expected name 'openai', got '%s'", info.Name)
		}

		t.Logf("OpenAI detected: Available=%v, Type=%s, EnvSet=%v",
			info.Available, info.Type, info.EnvSet)
	}

	// Test with API key only
	os.Setenv("OPENAI_API_KEY", "test-key")
	info := detectOpenAI()

	if !info.Available {
		t.Error("OpenAI should be detected as available when API key is set")
	}

	if !info.EnvSet {
		t.Error("EnvSet should be true when OPENAI_API_KEY is set")
	}

	if info.Type != "api" && info.Type != "cli" {
		t.Errorf("Expected type 'api' or 'cli', got '%s'", info.Type)
	}

	t.Logf("OpenAI (with API key) detected: Available=%v, Type=%s, EnvSet=%v",
		info.Available, info.Type, info.EnvSet)
}

// TestDetectGemini tests Gemini detection
func TestDetectGemini(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("GEMINI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("GEMINI_API_KEY", originalKey)
		} else {
			os.Unsetenv("GEMINI_API_KEY")
		}
	}()

	// Test with CLI if available
	if _, err := exec.LookPath("gemini"); err == nil {
		info := detectGemini()

		if !info.Available {
			t.Error("Gemini should be detected as available when CLI exists")
		}

		if info.Name != "gemini" {
			t.Errorf("Expected name 'gemini', got '%s'", info.Name)
		}

		t.Logf("Gemini detected: Available=%v, Type=%s, EnvSet=%v",
			info.Available, info.Type, info.EnvSet)
	}

	// Test with API key only
	os.Setenv("GEMINI_API_KEY", "test-key")
	info := detectGemini()

	if !info.Available {
		t.Error("Gemini should be detected as available when API key is set")
	}

	if !info.EnvSet {
		t.Error("EnvSet should be true when GEMINI_API_KEY is set")
	}

	t.Logf("Gemini (with API key) detected: Available=%v, Type=%s, EnvSet=%v",
		info.Available, info.Type, info.EnvSet)
}

// TestDetectAnthropic tests Anthropic API detection
func TestDetectAnthropic(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		} else {
			os.Unsetenv("ANTHROPIC_API_KEY")
		}
	}()

	// Test without API key
	os.Unsetenv("ANTHROPIC_API_KEY")
	info := detectAnthropic()

	if info.Name != "anthropic" {
		t.Errorf("Expected name 'anthropic', got '%s'", info.Name)
	}

	if info.Type != "api" {
		t.Errorf("Expected type 'api', got '%s'", info.Type)
	}

	if info.Available {
		t.Error("Anthropic should not be available without API key")
	}

	if info.EnvSet {
		t.Error("EnvSet should be false when API key is not set")
	}

	// Test with API key
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	info = detectAnthropic()

	if !info.Available {
		t.Error("Anthropic should be available when API key is set")
	}

	if !info.EnvSet {
		t.Error("EnvSet should be true when ANTHROPIC_API_KEY is set")
	}

	t.Logf("Anthropic detected: Available=%v, Type=%s, EnvSet=%v",
		info.Available, info.Type, info.EnvSet)
}

// TestDetectAll tests the main detection orchestration function
func TestDetectAll(t *testing.T) {
	// This test requires at least one of Docker/Podman to be available
	hasDocker := func() bool {
		_, err := exec.LookPath("docker")
		return err == nil
	}
	hasPodman := func() bool {
		_, err := exec.LookPath("podman")
		return err == nil
	}

	if !hasDocker() && !hasPodman() {
		t.Skip("Neither Docker nor Podman available in test environment")
	}

	ctx, err := DetectAll()

	// Verify no error occurred
	if err != nil {
		t.Fatalf("DetectAll should not return error: %v", err)
	}

	// Verify context was populated
	if ctx == nil {
		t.Fatal("DetectAll should return non-nil context")
	}

	// Verify at least one runtime was detected
	if !hasDocker() && !hasPodman() {
		if ctx.Runtime != "" {
			t.Error("Runtime should be empty when no container runtime available")
		}
	} else {
		if ctx.Runtime == "" && (hasDocker() || hasPodman()) {
			t.Error("Runtime should be detected when Docker or Podman is available")
		}
	}

	// Verify providers map exists
	if ctx.Providers == nil {
		t.Error("Providers map should not be nil")
	}

	// Log detection results
	t.Logf("DetectAll results:")
	t.Logf("  Runtime: %s", ctx.Runtime)
	if ctx.Runtime == "docker" {
		t.Logf("  Docker: Available=%v, Running=%v, Version=%s",
			ctx.Docker.Available, ctx.Docker.Running, ctx.Docker.Version)
	} else if ctx.Runtime == "podman" {
		t.Logf("  Podman: Available=%v, Running=%v, Version=%s",
			ctx.Podman.Available, ctx.Podman.Running, ctx.Podman.Version)
	}
	t.Logf("  Providers detected: %d", len(ctx.Providers))
	for name, info := range ctx.Providers {
		if info.Available {
			t.Logf("    - %s: Type=%s, Version=%s, EnvSet=%v",
				name, info.Type, info.Version, info.EnvSet)
		}
	}
	t.Logf("  Languages: %v", ctx.Languages)
	t.Logf("  Frameworks: %v", ctx.Frameworks)
	t.Logf("  Git initialized: %v", ctx.Git.Initialized)
	if ctx.Git.Initialized {
		t.Logf("    Branch: %s, Dirty: %v", ctx.Git.Branch, ctx.Git.Dirty)
	}
	t.Logf("  CI detected: %v", ctx.CI.Detected)
	if ctx.CI.Detected {
		t.Logf("    CI Name: %s", ctx.CI.Name)
	}
}
