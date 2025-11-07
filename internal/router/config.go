package router

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads router configuration from a YAML file
func LoadConfig(path string) (*RouterConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config RouterConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate config
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *RouterConfig {
	return &RouterConfig{
		Providers: []ProviderConfig{
			{
				Name:    ProviderAnthropic,
				APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
				Enabled: os.Getenv("ANTHROPIC_API_KEY") != "",
				Models: map[string]string{
					"agentic":      "claude-sonnet-4",
					"codegen":      "claude-sonnet-3.5",
					"fast":         "claude-haiku-3.5",
					"long-context": "claude-sonnet-4",
				},
			},
			{
				Name:    ProviderOpenAI,
				APIKey:  os.Getenv("OPENAI_API_KEY"),
				Enabled: os.Getenv("OPENAI_API_KEY") != "",
				Models: map[string]string{
					"codegen":      "gpt-4o",
					"long-context": "gpt-4-turbo",
					"cheap":        "gpt-4o-mini",
					"fast":         "gpt-3.5-turbo",
				},
			},
		},
		BudgetUSD:         20.0,
		MaxLatencyMs:      60000,
		PreferCheap:       false,
		FallbackModel:     "claude-haiku-3.5",
		EnableFallback:    true,               // Enable fallback by default
		MaxRetries:        3,                  // Retry up to 3 times
		RetryBackoffMs:    1000,               // Start with 1 second backoff
		RetryMaxBackoffMs: 30000,              // Max 30 second backoff
		EnableContextValidation: true,         // Validate context fits in model window
		AutoTruncate:      false,              // Error out by default (safer)
		TruncationStrategy: "oldest",          // Remove oldest context messages first
	}
}

// ValidateConfig validates a router configuration
func ValidateConfig(config *RouterConfig) error {
	if config.BudgetUSD < 0 {
		return fmt.Errorf("budget must be non-negative")
	}

	if config.MaxLatencyMs < 0 {
		return fmt.Errorf("max latency must be non-negative")
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

	// Validate provider configurations
	for _, p := range config.Providers {
		if p.Enabled && p.APIKey == "" {
			return fmt.Errorf("provider %s is enabled but API key is missing", p.Name)
		}
	}

	return nil
}

// SaveConfig saves router configuration to a YAML file
func SaveConfig(config *RouterConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
