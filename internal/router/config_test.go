package router

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *RouterConfig)
	}{
		{
			name: "valid complete config",
			configYAML: `
providers:
  - name: anthropic
    api_key: test-key-anthropic
    enabled: true
    models:
      agentic: claude-sonnet-4
      codegen: claude-sonnet-3.5
      fast: claude-haiku-3.5
  - name: openai
    api_key: test-key-openai
    enabled: true
    models:
      codegen: gpt-4o
      cheap: gpt-4o-mini
budget_usd: 50.0
max_latency_ms: 30000
prefer_cheap: true
fallback_model: claude-haiku-3.5
`,
			wantErr: false,
			validate: func(t *testing.T, c *RouterConfig) {
				if len(c.Providers) != 2 {
					t.Errorf("Providers length = %d, want 2", len(c.Providers))
				}
				if c.BudgetUSD != 50.0 {
					t.Errorf("BudgetUSD = %v, want 50.0", c.BudgetUSD)
				}
				if c.MaxLatencyMs != 30000 {
					t.Errorf("MaxLatencyMs = %v, want 30000", c.MaxLatencyMs)
				}
				if !c.PreferCheap {
					t.Error("PreferCheap should be true")
				}
				if c.FallbackModel != "claude-haiku-3.5" {
					t.Errorf("FallbackModel = %v, want claude-haiku-3.5", c.FallbackModel)
				}

				// Validate first provider
				if c.Providers[0].Name != ProviderAnthropic {
					t.Errorf("Provider[0].Name = %v, want %v", c.Providers[0].Name, ProviderAnthropic)
				}
				if !c.Providers[0].Enabled {
					t.Error("Provider[0] should be enabled")
				}
				if len(c.Providers[0].Models) != 3 {
					t.Errorf("Provider[0].Models length = %d, want 3", len(c.Providers[0].Models))
				}
			},
		},
		{
			name: "minimal config",
			configYAML: `
providers:
  - name: anthropic
    api_key: test-key
    enabled: true
budget_usd: 10.0
max_latency_ms: 60000
`,
			wantErr: false,
			validate: func(t *testing.T, c *RouterConfig) {
				if c.BudgetUSD != 10.0 {
					t.Errorf("BudgetUSD = %v, want 10.0", c.BudgetUSD)
				}
				if len(c.Providers) != 1 {
					t.Errorf("Providers length = %d, want 1", len(c.Providers))
				}
			},
		},
		{
			name: "invalid yaml",
			configYAML: `invalid: [yaml: syntax`,
			wantErr:     true,
			errContains: "unmarshal config",
		},
		{
			name: "negative budget",
			configYAML: `
providers:
  - name: anthropic
    api_key: test-key
    enabled: true
budget_usd: -5.0
max_latency_ms: 60000
`,
			wantErr:     true,
			errContains: "budget must be non-negative",
		},
		{
			name: "no enabled providers",
			configYAML: `
providers:
  - name: anthropic
    api_key: test-key
    enabled: false
budget_usd: 10.0
max_latency_ms: 60000
`,
			wantErr:     true,
			errContains: "at least one provider must be enabled",
		},
		{
			name: "enabled provider without api key",
			configYAML: `
providers:
  - name: anthropic
    api_key: ""
    enabled: true
budget_usd: 10.0
max_latency_ms: 60000
`,
			wantErr:     true,
			errContains: "API key is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "router.yaml")

			err := os.WriteFile(configFile, []byte(tt.configYAML), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}

			config, err := LoadConfig(configFile)

			if tt.wantErr {
				if err == nil {
					t.Error("LoadConfig() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("LoadConfig() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadConfig() unexpected error = %v", err)
			}

			if config == nil {
				t.Fatal("LoadConfig() returned nil config")
			}

			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/router.yaml")
	if err == nil {
		t.Error("LoadConfig() expected error for nonexistent file, got nil")
	}
	if !contains(err.Error(), "read config file") {
		t.Errorf("LoadConfig() error = %v, want error containing 'read config file'", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	// Save current env vars
	oldAnthropic := os.Getenv("ANTHROPIC_API_KEY")
	oldOpenAI := os.Getenv("OPENAI_API_KEY")

	// Set test env vars
	os.Setenv("ANTHROPIC_API_KEY", "test-anthropic")
	os.Setenv("OPENAI_API_KEY", "test-openai")
	defer func() {
		// Restore original env vars
		if oldAnthropic != "" {
			os.Setenv("ANTHROPIC_API_KEY", oldAnthropic)
		} else {
			os.Unsetenv("ANTHROPIC_API_KEY")
		}
		if oldOpenAI != "" {
			os.Setenv("OPENAI_API_KEY", oldOpenAI)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Validate structure
	if len(config.Providers) != 2 {
		t.Errorf("Providers length = %d, want 2", len(config.Providers))
	}

	// Validate default values
	if config.BudgetUSD != 20.0 {
		t.Errorf("BudgetUSD = %v, want 20.0", config.BudgetUSD)
	}
	if config.MaxLatencyMs != 60000 {
		t.Errorf("MaxLatencyMs = %v, want 60000", config.MaxLatencyMs)
	}
	if config.PreferCheap {
		t.Error("PreferCheap should be false by default")
	}
	if config.FallbackModel != "claude-haiku-3.5" {
		t.Errorf("FallbackModel = %v, want claude-haiku-3.5", config.FallbackModel)
	}

	// Validate anthropic provider
	anthropic := config.Providers[0]
	if anthropic.Name != ProviderAnthropic {
		t.Errorf("Provider[0].Name = %v, want %v", anthropic.Name, ProviderAnthropic)
	}
	if anthropic.APIKey != "test-anthropic" {
		t.Errorf("Provider[0].APIKey = %v, want test-anthropic", anthropic.APIKey)
	}
	if !anthropic.Enabled {
		t.Error("Anthropic provider should be enabled when API key is set")
	}
	if len(anthropic.Models) != 4 {
		t.Errorf("Anthropic models length = %d, want 4", len(anthropic.Models))
	}

	// Validate openai provider
	openai := config.Providers[1]
	if openai.Name != ProviderOpenAI {
		t.Errorf("Provider[1].Name = %v, want %v", openai.Name, ProviderOpenAI)
	}
	if openai.APIKey != "test-openai" {
		t.Errorf("Provider[1].APIKey = %v, want test-openai", openai.APIKey)
	}
	if !openai.Enabled {
		t.Error("OpenAI provider should be enabled when API key is set")
	}
	if len(openai.Models) != 4 {
		t.Errorf("OpenAI models length = %d, want 4", len(openai.Models))
	}
}

func TestDefaultConfig_NoAPIKeys(t *testing.T) {
	// Clear env vars
	oldAnthropic := os.Getenv("ANTHROPIC_API_KEY")
	oldOpenAI := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if oldAnthropic != "" {
			os.Setenv("ANTHROPIC_API_KEY", oldAnthropic)
		}
		if oldOpenAI != "" {
			os.Setenv("OPENAI_API_KEY", oldOpenAI)
		}
	}()

	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Both providers should be disabled when no API keys
	for i, p := range config.Providers {
		if p.Enabled {
			t.Errorf("Provider[%d] should be disabled when no API key", i)
		}
		if p.APIKey != "" {
			t.Errorf("Provider[%d].APIKey = %v, want empty", i, p.APIKey)
		}
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *RouterConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &RouterConfig{
				Providers: []ProviderConfig{
					{
						Name:    ProviderAnthropic,
						APIKey:  "test-key",
						Enabled: true,
					},
				},
				BudgetUSD:    10.0,
				MaxLatencyMs: 60000,
			},
			wantErr: false,
		},
		{
			name: "negative budget",
			config: &RouterConfig{
				Providers: []ProviderConfig{
					{
						Name:    ProviderAnthropic,
						APIKey:  "test-key",
						Enabled: true,
					},
				},
				BudgetUSD:    -5.0,
				MaxLatencyMs: 60000,
			},
			wantErr:     true,
			errContains: "budget must be non-negative",
		},
		{
			name: "negative latency",
			config: &RouterConfig{
				Providers: []ProviderConfig{
					{
						Name:    ProviderAnthropic,
						APIKey:  "test-key",
						Enabled: true,
					},
				},
				BudgetUSD:    10.0,
				MaxLatencyMs: -1000,
			},
			wantErr:     true,
			errContains: "max latency must be non-negative",
		},
		{
			name: "no enabled providers",
			config: &RouterConfig{
				Providers: []ProviderConfig{
					{
						Name:    ProviderAnthropic,
						APIKey:  "test-key",
						Enabled: false,
					},
					{
						Name:    ProviderOpenAI,
						APIKey:  "test-key2",
						Enabled: false,
					},
				},
				BudgetUSD:    10.0,
				MaxLatencyMs: 60000,
			},
			wantErr:     true,
			errContains: "at least one provider must be enabled",
		},
		{
			name: "enabled provider without api key",
			config: &RouterConfig{
				Providers: []ProviderConfig{
					{
						Name:    ProviderAnthropic,
						APIKey:  "",
						Enabled: true,
					},
				},
				BudgetUSD:    10.0,
				MaxLatencyMs: 60000,
			},
			wantErr:     true,
			errContains: "API key is missing",
		},
		{
			name: "disabled provider without api key is ok",
			config: &RouterConfig{
				Providers: []ProviderConfig{
					{
						Name:    ProviderAnthropic,
						APIKey:  "test-key",
						Enabled: true,
					},
					{
						Name:    ProviderOpenAI,
						APIKey:  "",
						Enabled: false,
					},
				},
				BudgetUSD:    10.0,
				MaxLatencyMs: 60000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("ValidateConfig() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateConfig() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateConfig() unexpected error = %v", err)
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *RouterConfig
	}{
		{
			name: "complete config",
			config: &RouterConfig{
				Providers: []ProviderConfig{
					{
						Name:    ProviderAnthropic,
						APIKey:  "test-key",
						Enabled: true,
						Models: map[string]string{
							"agentic": "claude-sonnet-4",
							"fast":    "claude-haiku-3.5",
						},
					},
				},
				BudgetUSD:     25.0,
				MaxLatencyMs:  45000,
				PreferCheap:   true,
				FallbackModel: "claude-haiku-3.5",
			},
		},
		{
			name: "minimal config",
			config: &RouterConfig{
				Providers: []ProviderConfig{
					{
						Name:    ProviderOpenAI,
						APIKey:  "test-key-2",
						Enabled: true,
					},
				},
				BudgetUSD:    5.0,
				MaxLatencyMs: 30000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "router.yaml")

			err := SaveConfig(tt.config, configFile)
			if err != nil {
				t.Fatalf("SaveConfig() error = %v", err)
			}

			// Verify file exists
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				t.Error("SaveConfig() did not create file")
			}

			// Verify can load back
			loaded, err := LoadConfig(configFile)
			if err != nil {
				t.Fatalf("LoadConfig() after SaveConfig() failed: %v", err)
			}

			// Verify content
			if loaded.BudgetUSD != tt.config.BudgetUSD {
				t.Errorf("Loaded BudgetUSD = %v, want %v", loaded.BudgetUSD, tt.config.BudgetUSD)
			}
			if loaded.MaxLatencyMs != tt.config.MaxLatencyMs {
				t.Errorf("Loaded MaxLatencyMs = %v, want %v", loaded.MaxLatencyMs, tt.config.MaxLatencyMs)
			}
			if len(loaded.Providers) != len(tt.config.Providers) {
				t.Errorf("Loaded Providers length = %d, want %d", len(loaded.Providers), len(tt.config.Providers))
			}
		})
	}
}

func TestSaveConfig_WriteError(t *testing.T) {
	config := &RouterConfig{
		Providers: []ProviderConfig{
			{
				Name:    ProviderAnthropic,
				APIKey:  "test-key",
				Enabled: true,
			},
		},
		BudgetUSD:    10.0,
		MaxLatencyMs: 60000,
	}

	// Try to write to a directory that doesn't exist
	err := SaveConfig(config, "/nonexistent/directory/router.yaml")
	if err == nil {
		t.Error("SaveConfig() expected error for invalid path, got nil")
	}
	if !contains(err.Error(), "write config file") {
		t.Errorf("SaveConfig() error = %v, want error containing 'write config file'", err)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	// Create a config
	original := &RouterConfig{
		Providers: []ProviderConfig{
			{
				Name:    ProviderAnthropic,
				APIKey:  "test-anthropic-key",
				Enabled: true,
				Models: map[string]string{
					"agentic": "claude-sonnet-4",
					"codegen": "claude-sonnet-3.5",
				},
			},
			{
				Name:    ProviderOpenAI,
				APIKey:  "test-openai-key",
				Enabled: false,
				Models: map[string]string{
					"codegen": "gpt-4o",
				},
			},
		},
		BudgetUSD:     15.0,
		MaxLatencyMs:  50000,
		PreferCheap:   true,
		FallbackModel: "claude-haiku-3.5",
	}

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "router.yaml")

	// Save
	err := SaveConfig(original, configFile)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Load
	loaded, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify round-trip
	if loaded.BudgetUSD != original.BudgetUSD {
		t.Errorf("Round-trip BudgetUSD = %v, want %v", loaded.BudgetUSD, original.BudgetUSD)
	}
	if loaded.MaxLatencyMs != original.MaxLatencyMs {
		t.Errorf("Round-trip MaxLatencyMs = %v, want %v", loaded.MaxLatencyMs, original.MaxLatencyMs)
	}
	if loaded.PreferCheap != original.PreferCheap {
		t.Errorf("Round-trip PreferCheap = %v, want %v", loaded.PreferCheap, original.PreferCheap)
	}
	if loaded.FallbackModel != original.FallbackModel {
		t.Errorf("Round-trip FallbackModel = %v, want %v", loaded.FallbackModel, original.FallbackModel)
	}
	if len(loaded.Providers) != len(original.Providers) {
		t.Errorf("Round-trip Providers length = %d, want %d", len(loaded.Providers), len(original.Providers))
	}

	// Check first provider details
	if loaded.Providers[0].Name != original.Providers[0].Name {
		t.Errorf("Round-trip Provider[0].Name = %v, want %v", loaded.Providers[0].Name, original.Providers[0].Name)
	}
	if loaded.Providers[0].Enabled != original.Providers[0].Enabled {
		t.Errorf("Round-trip Provider[0].Enabled = %v, want %v", loaded.Providers[0].Enabled, original.Providers[0].Enabled)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
