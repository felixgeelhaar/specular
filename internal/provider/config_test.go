package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProvidersConfig(t *testing.T) {
	// Test loading the example config
	config, err := LoadProvidersConfig("../../.specular/providers.yaml.example")
	if err != nil {
		t.Fatalf("Failed to load providers.yaml.example: %v", err)
	}

	// Verify providers were loaded
	if len(config.Providers) == 0 {
		t.Error("No providers loaded from example config")
	}

	// Check for expected providers
	expectedProviders := map[string]bool{
		"ollama":     false,
		"openai":     false,
		"anthropic":  false,
		"claude-cli": false,
	}

	for _, p := range config.Providers {
		if _, exists := expectedProviders[p.Name]; exists {
			expectedProviders[p.Name] = true
		}
	}

	for name, found := range expectedProviders {
		if !found {
			t.Errorf("Expected provider %s not found in config", name)
		}
	}

	// Verify strategy config
	if config.Strategy.Budget.MaxCostPerDay == 0 {
		t.Error("Strategy budget max_cost_per_day not set")
	}
	if config.Strategy.Performance.MaxLatencyMs == 0 {
		t.Error("Strategy performance max_latency_ms not set")
	}
}

func TestLoadProvidersConfig_Errors(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
		errContains string
	}{
		{
			name:        "invalid yaml syntax",
			configYAML:  "invalid: [yaml: syntax",
			wantErr:     true,
			errContains: "unmarshal config",
		},
		{
			name: "validation failure - no providers",
			configYAML: `
strategy:
  budget:
    max_cost_per_day: 20.0
`,
			wantErr:     true,
			errContains: "invalid config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "providers.yaml")

			err := os.WriteFile(configPath, []byte(tt.configYAML), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			_, err = LoadProvidersConfig(configPath)

			if tt.wantErr {
				if err == nil {
					t.Error("LoadProvidersConfig() expected error, got nil")
				} else if !contains(err.Error(), tt.errContains) {
					t.Errorf("LoadProvidersConfig() error = %v, want error containing %q", err, tt.errContains)
				}
			} else if err != nil {
				t.Errorf("LoadProvidersConfig() unexpected error = %v", err)
			}
		})
	}
}

func TestValidateProvidersConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProvidersConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ProvidersConfig{
				Providers: []ProviderConfig{
					{
						Name:    "ollama",
						Type:    ProviderTypeCLI,
						Enabled: true,
						Source:  "local",
						Config: map[string]interface{}{
							"path": "/path/to/provider",
						},
					},
				},
				Strategy: StrategyConfig{
					Budget: BudgetConfig{
						MaxCostPerDay:     20.0,
						MaxCostPerRequest: 1.0,
					},
					Performance: PerformanceConfig{
						MaxLatencyMs: 60000,
					},
					Fallback: FallbackConfig{
						MaxRetries:   3,
						RetryDelayMs: 1000,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "no providers",
			config:  &ProvidersConfig{},
			wantErr: true,
		},
		{
			name: "no enabled providers",
			config: &ProvidersConfig{
				Providers: []ProviderConfig{
					{
						Name:    "ollama",
						Type:    ProviderTypeCLI,
						Enabled: false,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "negative budget max_cost_per_day",
			config: &ProvidersConfig{
				Providers: []ProviderConfig{
					{
						Name:    "ollama",
						Type:    ProviderTypeCLI,
						Enabled: true,
						Config: map[string]interface{}{
							"path": "/path/to/provider",
						},
					},
				},
				Strategy: StrategyConfig{
					Budget: BudgetConfig{
						MaxCostPerDay: -1.0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "negative budget max_cost_per_request",
			config: &ProvidersConfig{
				Providers: []ProviderConfig{
					{
						Name:    "ollama",
						Type:    ProviderTypeCLI,
						Enabled: true,
						Config: map[string]interface{}{
							"path": "/path/to/provider",
						},
					},
				},
				Strategy: StrategyConfig{
					Budget: BudgetConfig{
						MaxCostPerRequest: -0.5,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "negative max_latency_ms",
			config: &ProvidersConfig{
				Providers: []ProviderConfig{
					{
						Name:    "ollama",
						Type:    ProviderTypeCLI,
						Enabled: true,
						Config: map[string]interface{}{
							"path": "/path/to/provider",
						},
					},
				},
				Strategy: StrategyConfig{
					Performance: PerformanceConfig{
						MaxLatencyMs: -1000,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "negative max_retries",
			config: &ProvidersConfig{
				Providers: []ProviderConfig{
					{
						Name:    "ollama",
						Type:    ProviderTypeCLI,
						Enabled: true,
						Config: map[string]interface{}{
							"path": "/path/to/provider",
						},
					},
				},
				Strategy: StrategyConfig{
					Fallback: FallbackConfig{
						MaxRetries: -1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "negative retry_delay_ms",
			config: &ProvidersConfig{
				Providers: []ProviderConfig{
					{
						Name:    "ollama",
						Type:    ProviderTypeCLI,
						Enabled: true,
						Config: map[string]interface{}{
							"path": "/path/to/provider",
						},
					},
				},
				Strategy: StrategyConfig{
					Fallback: FallbackConfig{
						RetryDelayMs: -500,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProvidersConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProvidersConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateProviderConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name: "valid CLI provider",
			config: &ProviderConfig{
				Name:   "ollama",
				Type:   ProviderTypeCLI,
				Source: "local",
				Config: map[string]interface{}{
					"path": "/path/to/provider",
				},
			},
			wantErr: false,
		},
		{
			name: "CLI provider missing path",
			config: &ProviderConfig{
				Name:   "ollama",
				Type:   ProviderTypeCLI,
				Source: "local",
				Config: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			config: &ProviderConfig{
				Type: ProviderTypeCLI,
			},
			wantErr: true,
		},
		{
			name: "missing type",
			config: &ProviderConfig{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "invalid provider type",
			config: &ProviderConfig{
				Name: "test",
				Type: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProviderConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveAndLoadProvidersConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "providers.yaml")

	// Create a test config
	config := &ProvidersConfig{
		Providers: []ProviderConfig{
			{
				Name:    "ollama",
				Type:    ProviderTypeCLI,
				Enabled: true,
				Source:  "local",
				Version: "1.0.0",
				Config: map[string]interface{}{
					"path": "/path/to/ollama-provider",
				},
				Models: map[string]string{
					"fast": "llama3.2",
				},
			},
		},
		Strategy: StrategyConfig{
			Budget: BudgetConfig{
				MaxCostPerDay:     20.0,
				MaxCostPerRequest: 1.0,
			},
			Performance: PerformanceConfig{
				MaxLatencyMs: 60000,
				PreferCheap:  true,
			},
		},
	}

	// Save the config
	if err := SaveProvidersConfig(config, configPath); err != nil {
		t.Fatalf("SaveProvidersConfig() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load the config back
	loaded, err := LoadProvidersConfig(configPath)
	if err != nil {
		t.Fatalf("LoadProvidersConfig() error = %v", err)
	}

	// Verify the loaded config matches
	if len(loaded.Providers) != len(config.Providers) {
		t.Errorf("Loaded %d providers, expected %d", len(loaded.Providers), len(config.Providers))
	}

	if loaded.Providers[0].Name != config.Providers[0].Name {
		t.Errorf("Provider name = %s, want %s", loaded.Providers[0].Name, config.Providers[0].Name)
	}

	if loaded.Strategy.Budget.MaxCostPerDay != config.Strategy.Budget.MaxCostPerDay {
		t.Errorf("Budget max_cost_per_day = %.2f, want %.2f",
			loaded.Strategy.Budget.MaxCostPerDay,
			config.Strategy.Budget.MaxCostPerDay)
	}
}

func TestSaveProvidersConfig_WriteError(t *testing.T) {
	config := &ProvidersConfig{
		Providers: []ProviderConfig{
			{
				Name:    "ollama",
				Type:    ProviderTypeCLI,
				Enabled: true,
				Source:  "local",
				Config: map[string]interface{}{
					"path": "/path/to/provider",
				},
			},
		},
		Strategy: StrategyConfig{
			Budget: BudgetConfig{
				MaxCostPerDay: 20.0,
			},
		},
	}

	// Try to write to a directory that doesn't exist
	err := SaveProvidersConfig(config, "/nonexistent/directory/providers.yaml")
	if err == nil {
		t.Error("SaveProvidersConfig() expected error for invalid path, got nil")
	}
	if !contains(err.Error(), "write config file") {
		t.Errorf("SaveProvidersConfig() error = %v, want error containing 'write config file'", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDefaultProvidersConfig(t *testing.T) {
	config := DefaultProvidersConfig()

	if len(config.Providers) == 0 {
		t.Error("Default config has no providers")
	}

	if config.Strategy.Budget.MaxCostPerDay == 0 {
		t.Error("Default config has no budget configured")
	}

	if config.Strategy.Performance.MaxLatencyMs == 0 {
		t.Error("Default config has no latency constraint")
	}
}

func TestExpandEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		envVars map[string]string
		want    string
	}{
		{
			name:  "${VAR} syntax",
			input: "Hello ${NAME}!",
			envVars: map[string]string{
				"NAME": "World",
			},
			want: "Hello World!",
		},
		{
			name:  "$VAR syntax",
			input: "Hello $NAME!",
			envVars: map[string]string{
				"NAME": "World",
			},
			want: "Hello World!",
		},
		{
			name:  "Multiple variables",
			input: "api_key: ${API_KEY}, url: ${BASE_URL}",
			envVars: map[string]string{
				"API_KEY":  "secret123",
				"BASE_URL": "https://api.example.com",
			},
			want: "api_key: secret123, url: https://api.example.com",
		},
		{
			name:    "Undefined variable",
			input:   "Value: ${UNDEFINED}",
			envVars: map[string]string{},
			want:    "Value: ",
		},
		{
			name:    "No variables",
			input:   "plain text",
			envVars: map[string]string{},
			want:    "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsEnvVarSet(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		value   string
		set     bool
		want    bool
	}{
		{
			name:    "set with value",
			varName: "TEST_VAR_SET",
			value:   "some value",
			set:     true,
			want:    true,
		},
		{
			name:    "set but empty",
			varName: "TEST_VAR_EMPTY",
			value:   "",
			set:     true,
			want:    false,
		},
		{
			name:    "set with whitespace only",
			varName: "TEST_VAR_WHITESPACE",
			value:   "   ",
			set:     true,
			want:    false,
		},
		{
			name:    "not set",
			varName: "TEST_VAR_UNSET",
			set:     false,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing value
			os.Unsetenv(tt.varName)
			defer os.Unsetenv(tt.varName)

			if tt.set {
				os.Setenv(tt.varName, tt.value)
			}

			got := IsEnvVarSet(tt.varName)
			if got != tt.want {
				t.Errorf("IsEnvVarSet(%q) = %v, want %v", tt.varName, got, tt.want)
			}
		})
	}
}

func TestLoadRegistryFromProvidersConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProvidersConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "all providers disabled",
			config: &ProvidersConfig{
				Providers: []ProviderConfig{
					{
						Name:    "ollama",
						Type:    ProviderTypeCLI,
						Enabled: false,
						Source:  "local",
						Config: map[string]interface{}{
							"path": "/some/path",
						},
					},
					{
						Name:    "openai",
						Type:    ProviderTypeAPI,
						Enabled: false,
					},
				},
			},
			wantErr: true,
			errMsg:  "no providers loaded successfully",
		},
		{
			name: "all providers fail to load - invalid config",
			config: &ProvidersConfig{
				Providers: []ProviderConfig{
					{
						Name:    "invalid-provider",
						Type:    ProviderTypeCLI,
						Enabled: true,
						Source:  "local",
						Config:  map[string]interface{}{
							// Missing required 'path' field
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "no providers loaded successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, err := LoadRegistryFromProvidersConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("LoadRegistryFromProvidersConfig() expected error, got nil")
				} else if !contains(err.Error(), tt.errMsg) {
					t.Errorf("LoadRegistryFromProvidersConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
				if registry != nil {
					t.Error("LoadRegistryFromProvidersConfig() expected nil registry on error")
				}
			} else {
				if err != nil {
					t.Errorf("LoadRegistryFromProvidersConfig() unexpected error = %v", err)
				}
				if registry == nil {
					t.Error("LoadRegistryFromProvidersConfig() returned nil registry")
				}
			}
		})
	}
}

func TestLoadRegistryFromConfig(t *testing.T) {
	// Skip if ollama provider doesn't exist
	if _, err := os.Stat("../../providers/ollama/ollama-provider"); os.IsNotExist(err) {
		t.Skip("ollama-provider not built, skipping test")
	}

	// Create a temporary config file with ollama enabled
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "providers.yaml")

	providerPath, err := filepath.Abs("../../providers/ollama/ollama-provider")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	config := &ProvidersConfig{
		Providers: []ProviderConfig{
			{
				Name:    "ollama",
				Type:    ProviderTypeCLI,
				Enabled: true,
				Source:  "local",
				Version: "1.0.0",
				Config: map[string]interface{}{
					"path": providerPath,
				},
			},
		},
	}

	if err := SaveProvidersConfig(config, configPath); err != nil {
		t.Fatalf("SaveProvidersConfig() error = %v", err)
	}

	// Load registry from config
	registry, err := LoadRegistryFromConfig(configPath)
	if err != nil {
		t.Fatalf("LoadRegistryFromConfig() error = %v", err)
	}

	// Verify ollama provider was loaded
	providers := registry.List()
	if len(providers) == 0 {
		t.Error("No providers loaded into registry")
	}

	found := false
	for _, name := range providers {
		if name == "ollama" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Ollama provider not loaded into registry")
	}

	// Try to get the provider
	prov, err := registry.Get("ollama")
	if err != nil {
		t.Errorf("Failed to get ollama provider: %v", err)
	}

	if prov == nil {
		t.Error("Got nil provider")
	}
}

func TestLoadRegistryFromConfig_Error(t *testing.T) {
	// Try to load from non-existent config file
	_, err := LoadRegistryFromConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadRegistryFromConfig() expected error for non-existent file, got nil")
	}
}
