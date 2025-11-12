package provider

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProvidersConfig represents the complete providers.yaml configuration
type ProvidersConfig struct {
	Providers []ProviderConfig `yaml:"providers"`
	Strategy  StrategyConfig   `yaml:"strategy,omitempty"`
}

// StrategyConfig represents the provider selection strategy
type StrategyConfig struct {
	Preference  []string          `yaml:"preference,omitempty"`
	Budget      BudgetConfig      `yaml:"budget,omitempty"`
	Performance PerformanceConfig `yaml:"performance,omitempty"`
	Fallback    FallbackConfig    `yaml:"fallback,omitempty"`
}

// BudgetConfig represents budget constraints
type BudgetConfig struct {
	MaxCostPerDay     float64 `yaml:"max_cost_per_day,omitempty"`
	MaxCostPerRequest float64 `yaml:"max_cost_per_request,omitempty"`
}

// PerformanceConfig represents performance requirements
type PerformanceConfig struct {
	MaxLatencyMs int  `yaml:"max_latency_ms,omitempty"`
	PreferCheap  bool `yaml:"prefer_cheap,omitempty"`
}

// FallbackConfig represents fallback behavior
type FallbackConfig struct {
	Enabled       bool   `yaml:"enabled,omitempty"`
	MaxRetries    int    `yaml:"max_retries,omitempty"`
	RetryDelayMs  int    `yaml:"retry_delay_ms,omitempty"`
	FallbackModel string `yaml:"fallback_model,omitempty"`
}

// LoadProvidersConfig loads provider configuration from a YAML file
func LoadProvidersConfig(path string) (*ProvidersConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Expand environment variables in the config
	configStr := os.ExpandEnv(string(data))

	var config ProvidersConfig
	if err := yaml.Unmarshal([]byte(configStr), &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate config
	if err := ValidateProvidersConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// ValidateProvidersConfig validates a providers configuration
func ValidateProvidersConfig(config *ProvidersConfig) error {
	if len(config.Providers) == 0 {
		return fmt.Errorf("no providers configured")
	}

	// Check that at least one provider is enabled
	hasEnabled := false
	for _, p := range config.Providers {
		if p.Enabled {
			hasEnabled = true
			break
		}
	}

	if !hasEnabled {
		return fmt.Errorf("at least one provider must be enabled")
	}

	// Validate individual provider configurations
	for i, p := range config.Providers {
		if err := ValidateProviderConfig(&p); err != nil {
			return fmt.Errorf("provider %d (%s): %w", i, p.Name, err)
		}
	}

	// Validate strategy configuration
	if config.Strategy.Budget.MaxCostPerDay < 0 {
		return fmt.Errorf("budget max_cost_per_day must be non-negative")
	}
	if config.Strategy.Budget.MaxCostPerRequest < 0 {
		return fmt.Errorf("budget max_cost_per_request must be non-negative")
	}
	if config.Strategy.Performance.MaxLatencyMs < 0 {
		return fmt.Errorf("performance max_latency_ms must be non-negative")
	}
	if config.Strategy.Fallback.MaxRetries < 0 {
		return fmt.Errorf("fallback max_retries must be non-negative")
	}
	if config.Strategy.Fallback.RetryDelayMs < 0 {
		return fmt.Errorf("fallback retry_delay_ms must be non-negative")
	}

	return nil
}

// ValidateProviderConfig validates a single provider configuration
func ValidateProviderConfig(config *ProviderConfig) error {
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}

	if config.Type == "" {
		return fmt.Errorf("type is required")
	}

	// Validate provider type
	switch config.Type {
	case ProviderTypeCLI, ProviderTypeAPI, ProviderTypeGRPC, ProviderTypeNative:
		// Valid types
	default:
		return fmt.Errorf("invalid provider type: %s (must be cli, api, grpc, or native)", config.Type)
	}

	// Type-specific validation
	switch config.Type {
	case ProviderTypeCLI, ProviderTypeNative:
		// CLI and native providers must have a path
		if path, ok := config.Config["path"].(string); !ok || path == "" {
			return fmt.Errorf("CLI and native providers require 'path' in config")
		}
	case ProviderTypeAPI:
		// API providers might need API keys (but allow env var expansion)
		// We don't enforce this here as some APIs might not need keys
	}

	return nil
}

// SaveProvidersConfig saves provider configuration to a YAML file
func SaveProvidersConfig(config *ProvidersConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// DefaultProvidersConfig returns a configuration with sensible defaults
func DefaultProvidersConfig() *ProvidersConfig {
	return &ProvidersConfig{
		Providers: []ProviderConfig{
			{
				Name:    "ollama",
				Type:    ProviderTypeCLI,
				Enabled: false, // User must enable and configure
				Source:  "local",
				Version: "1.0.0",
				Config: map[string]interface{}{
					"path": "./providers/ollama/ollama-provider",
				},
				Models: map[string]string{
					"fast":    "llama3.2",
					"codegen": "codellama",
					"cheap":   "llama3.2",
				},
			},
		},
		Strategy: StrategyConfig{
			Preference: []string{"ollama"},
			Budget: BudgetConfig{
				MaxCostPerDay:     20.0,
				MaxCostPerRequest: 1.0,
			},
			Performance: PerformanceConfig{
				MaxLatencyMs: 60000,
				PreferCheap:  true,
			},
			Fallback: FallbackConfig{
				Enabled:       true,
				MaxRetries:    3,
				RetryDelayMs:  1000,
				FallbackModel: "ollama/llama3.2",
			},
		},
	}
}

// LoadRegistryFromConfig loads providers into a registry from configuration
func LoadRegistryFromConfig(configPath string) (*Registry, error) {
	config, err := LoadProvidersConfig(configPath)
	if err != nil {
		return nil, err
	}

	return LoadRegistryFromProvidersConfig(config)
}

// LoadRegistryFromProvidersConfig loads providers into a registry from a ProvidersConfig
func LoadRegistryFromProvidersConfig(config *ProvidersConfig) (*Registry, error) {
	registry := NewRegistry()

	// Load only enabled providers
	for _, providerConfig := range config.Providers {
		if !providerConfig.Enabled {
			continue
		}

		if err := registry.LoadFromConfig(&providerConfig); err != nil {
			// Log error but continue with other providers
			fmt.Fprintf(os.Stderr, "Warning: failed to load provider %s: %v\n", providerConfig.Name, err)
			continue
		}
	}

	// Check if any providers loaded successfully
	if len(registry.List()) == 0 {
		return nil, fmt.Errorf("no providers loaded successfully")
	}

	return registry, nil
}

// LoadRegistryWithAutoDiscovery loads providers with auto-discovery fallback
// If configPath doesn't exist or is empty, auto-discovers available providers
func LoadRegistryWithAutoDiscovery(configPath string) (*Registry, error) {
	// Try to load from config file first
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return LoadRegistryFromConfig(configPath)
		}
	}

	// No config file found - use auto-discovery
	return LoadRegistryFromAutoDiscovery()
}

// LoadRegistryFromAutoDiscovery creates a registry by auto-discovering available providers
func LoadRegistryFromAutoDiscovery() (*Registry, error) {
	// Import detect package to use detection
	// Note: This creates a dependency on internal/detect
	// We'll need to add the import at the top of the file

	registry := NewRegistry()

	// Auto-discover and load available providers
	// For now, we'll use a simple approach: try to load known providers
	// and skip those that aren't available

	knownProviders := []struct {
		name       string
		configPath string
		required   bool
	}{
		{"ollama", "", false},
		{"anthropic", "", false},
		{"openai", "", false},
	}

	loadedCount := 0
	for _, p := range knownProviders {
		config := generateProviderConfig(p.name)
		if config != nil {
			if err := registry.LoadFromConfig(config); err != nil {
				if p.required {
					return nil, fmt.Errorf("failed to load required provider %s: %w", p.name, err)
				}
				// Skip optional providers that fail to load
				continue
			}
			loadedCount++
		}
	}

	if loadedCount == 0 {
		return nil, fmt.Errorf("no providers available - please install at least one AI provider (ollama, anthropic, openai)")
	}

	return registry, nil
}

// generateProviderConfig creates a provider configuration based on auto-detection
func generateProviderConfig(providerName string) *ProviderConfig {
	switch providerName {
	case "ollama":
		// Check if ollama CLI is available
		if path, err := lookupCommand("ollama"); err == nil {
			return &ProviderConfig{
				Name:    "ollama",
				Type:    ProviderTypeCLI,
				Enabled: true,
				Source:  "local",
				Config: map[string]interface{}{
					"path": path,
				},
				Models: map[string]string{
					"fast":         "llama3.3:70b",
					"codegen":      "qwen2.5-coder:7b",
					"cheap":        "llama3.2",
					"long-context": "llama3.3:70b",
				},
			}
		}

	case "anthropic":
		// Check if ANTHROPIC_API_KEY is set
		if IsEnvVarSet("ANTHROPIC_API_KEY") {
			return &ProviderConfig{
				Name:    "anthropic",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Source:  "api",
				Config: map[string]interface{}{
					"api_key":  "${ANTHROPIC_API_KEY}",
					"base_url": "https://api.anthropic.com",
				},
				Models: map[string]string{
					"fast":         "claude-haiku-4-5-20251015",
					"codegen":      "claude-sonnet-4-5-20250929",
					"agentic":      "claude-opus-4-1-20250805",
					"long-context": "claude-sonnet-4-5-20250929",
					"cheap":        "claude-haiku-4-5-20251015",
				},
			}
		}

	case "openai":
		// Check if OPENAI_API_KEY is set
		if IsEnvVarSet("OPENAI_API_KEY") {
			return &ProviderConfig{
				Name:    "openai",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Source:  "api",
				Config: map[string]interface{}{
					"api_key":  "${OPENAI_API_KEY}",
					"base_url": "https://api.openai.com/v1",
				},
				Models: map[string]string{
					"fast":         "gpt-5-mini",
					"codegen":      "gpt-5",
					"cheap":        "gpt-5-nano",
					"long-context": "gpt-5",
					"agentic":      "gpt-5",
				},
			}
		}
	}

	return nil
}

// lookupCommand checks if a command exists in PATH and returns its path
func lookupCommand(name string) (string, error) {
	return exec.LookPath(name)
}

// expandEnvVars expands environment variables in a string
// Supports ${VAR} and $VAR syntax
func expandEnvVars(s string) string {
	return os.ExpandEnv(s)
}

// IsEnvVarSet checks if an environment variable is set and non-empty
func IsEnvVarSet(name string) bool {
	val := strings.TrimSpace(os.Getenv(name))
	return val != ""
}
