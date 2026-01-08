// Package quickstart provides simplified onboarding for Specular CLI.
package quickstart

import (
	"fmt"
	"net/http"
	"time"

	"github.com/felixgeelhaar/specular/internal/detect"
)

// ProviderSelection represents a selected AI provider for quickstart
type ProviderSelection struct {
	Name    string // Provider name (anthropic, openai, ollama, etc.)
	Type    string // Provider type (api, local, cli)
	Ready   bool   // Is the provider ready to use
	EnvVar  string // Environment variable used (if API provider)
	Version string // Version string (if available)
	Reason  string // Why this provider was selected
}

// DockerStatus represents container runtime status
type DockerStatus struct {
	Available bool
	Runtime   string // "docker", "podman", or ""
	Version   string
	Warning   string
}

// DetectionResult holds the complete environment detection results
type DetectionResult struct {
	Docker   DockerStatus
	Provider *ProviderSelection
	Errors   []string
}

// NoProviderError indicates no AI provider was found
type NoProviderError struct {
	Suggestion string
}

func (e *NoProviderError) Error() string {
	return fmt.Sprintf("no AI provider found: %s", e.Suggestion)
}

// DetectEnvironment performs environment detection for quickstart
func DetectEnvironment() (*DetectionResult, error) {
	result := &DetectionResult{
		Errors: []string{},
	}

	// Detect container runtime
	result.Docker = detectContainerRuntime()

	// Detect AI provider (with priority)
	provider, err := detectBestProvider()
	if err != nil {
		return result, err
	}
	result.Provider = provider

	return result, nil
}

// detectContainerRuntime checks for Docker or Podman
func detectContainerRuntime() DockerStatus {
	ctx, err := detect.DetectAll()
	if err != nil {
		return DockerStatus{
			Available: false,
			Warning:   "Failed to detect container runtime",
		}
	}

	if ctx.Docker.Available && ctx.Docker.Running {
		return DockerStatus{
			Available: true,
			Runtime:   "docker",
			Version:   ctx.Docker.Version,
		}
	}

	if ctx.Podman.Available && ctx.Podman.Running {
		return DockerStatus{
			Available: true,
			Runtime:   "podman",
			Version:   ctx.Podman.Version,
		}
	}

	// Check if Docker is installed but not running
	if ctx.Docker.Available && !ctx.Docker.Running {
		return DockerStatus{
			Available: false,
			Runtime:   "docker",
			Version:   ctx.Docker.Version,
			Warning:   "Docker is installed but not running. Start Docker Desktop or the Docker daemon.",
		}
	}

	return DockerStatus{
		Available: false,
		Warning:   "No container runtime detected. Docker is recommended for sandboxed code execution.",
	}
}

// detectBestProvider finds the best available AI provider using priority order:
// 1. ANTHROPIC_API_KEY env var (best quality)
// 2. OPENAI_API_KEY env var
// 3. GEMINI_API_KEY env var
// 4. Ollama running locally (free, private)
// 5. Claude CLI installed
// 6. Error with suggestion
func detectBestProvider() (*ProviderSelection, error) {
	ctx, err := detect.DetectAll()
	if err != nil {
		return nil, fmt.Errorf("failed to detect providers: %w", err)
	}

	// Priority 1: Anthropic API key
	if provider, ok := ctx.Providers["anthropic"]; ok && provider.EnvSet {
		return &ProviderSelection{
			Name:   "anthropic",
			Type:   "api",
			Ready:  true,
			EnvVar: "ANTHROPIC_API_KEY",
			Reason: "ANTHROPIC_API_KEY environment variable detected",
		}, nil
	}

	// Priority 2: OpenAI API key
	if provider, ok := ctx.Providers["openai"]; ok && provider.EnvSet {
		return &ProviderSelection{
			Name:   "openai",
			Type:   "api",
			Ready:  true,
			EnvVar: "OPENAI_API_KEY",
			Reason: "OPENAI_API_KEY environment variable detected",
		}, nil
	}

	// Priority 3: Gemini API key
	if provider, ok := ctx.Providers["gemini"]; ok && provider.EnvSet {
		return &ProviderSelection{
			Name:   "gemini",
			Type:   "api",
			Ready:  true,
			EnvVar: "GEMINI_API_KEY",
			Reason: "GEMINI_API_KEY environment variable detected",
		}, nil
	}

	// Priority 4: Ollama running locally
	if provider, ok := ctx.Providers["ollama"]; ok && provider.Available {
		// Verify Ollama is actually running by pinging the API
		if isOllamaRunning() {
			return &ProviderSelection{
				Name:    "ollama",
				Type:    "local",
				Ready:   true,
				Version: provider.Version,
				Reason:  "Ollama detected running on localhost:11434",
			}, nil
		}
	}

	// Priority 5: Claude CLI installed
	if provider, ok := ctx.Providers["claude"]; ok && provider.Available {
		return &ProviderSelection{
			Name:    "claude",
			Type:    "cli",
			Ready:   true,
			Version: provider.Version,
			Reason:  "Claude CLI detected",
		}, nil
	}

	// No provider found
	return nil, &NoProviderError{
		Suggestion: "Set ANTHROPIC_API_KEY, OPENAI_API_KEY, or install Ollama (https://ollama.com)",
	}
}

// isOllamaRunning checks if Ollama is actively running
func isOllamaRunning() bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// DetectProviderByName attempts to detect a specific provider by name
func DetectProviderByName(name string) (*ProviderSelection, error) {
	ctx, err := detect.DetectAll()
	if err != nil {
		return nil, fmt.Errorf("failed to detect providers: %w", err)
	}

	provider, ok := ctx.Providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	switch name {
	case "anthropic":
		if provider.EnvSet {
			return &ProviderSelection{
				Name:   "anthropic",
				Type:   "api",
				Ready:  true,
				EnvVar: "ANTHROPIC_API_KEY",
				Reason: "ANTHROPIC_API_KEY is set",
			}, nil
		}
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")

	case "openai":
		if provider.EnvSet {
			return &ProviderSelection{
				Name:   "openai",
				Type:   "api",
				Ready:  true,
				EnvVar: "OPENAI_API_KEY",
				Reason: "OPENAI_API_KEY is set",
			}, nil
		}
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")

	case "gemini":
		if provider.EnvSet {
			return &ProviderSelection{
				Name:   "gemini",
				Type:   "api",
				Ready:  true,
				EnvVar: "GEMINI_API_KEY",
				Reason: "GEMINI_API_KEY is set",
			}, nil
		}
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")

	case "ollama":
		if provider.Available && isOllamaRunning() {
			return &ProviderSelection{
				Name:    "ollama",
				Type:    "local",
				Ready:   true,
				Version: provider.Version,
				Reason:  "Ollama is running",
			}, nil
		}
		if provider.Available {
			return nil, fmt.Errorf("Ollama is installed but not running. Start with: ollama serve")
		}
		return nil, fmt.Errorf("Ollama is not installed. Install from: https://ollama.com")

	case "claude":
		if provider.Available {
			return &ProviderSelection{
				Name:    "claude",
				Type:    "cli",
				Ready:   true,
				Version: provider.Version,
				Reason:  "Claude CLI is installed",
			}, nil
		}
		return nil, fmt.Errorf("Claude CLI not found in PATH")

	default:
		return nil, fmt.Errorf("provider %s not supported for quickstart", name)
	}
}
