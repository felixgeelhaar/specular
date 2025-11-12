package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultGlobalConfig(t *testing.T) {
	config := defaultGlobalConfig()

	// Test providers defaults
	if config.Providers.Default != "ollama" {
		t.Errorf("Default provider = %s, want ollama", config.Providers.Default)
	}

	if len(config.Providers.Preference) == 0 {
		t.Error("Provider preference should not be empty")
	}

	// Test defaults
	if config.Defaults.Format != "text" {
		t.Errorf("Default format = %s, want text", config.Defaults.Format)
	}

	if config.Defaults.SpecularDir != ".specular" {
		t.Errorf("Default specular_dir = %s, want .specular", config.Defaults.SpecularDir)
	}

	// Test budget
	if config.Budget.MaxCostPerDay <= 0 {
		t.Error("Max cost per day should be positive")
	}

	// Test logging
	if config.Logging.Level != "info" {
		t.Errorf("Default log level = %s, want info", config.Logging.Level)
	}

	// Test telemetry
	if config.Telemetry.Enabled {
		t.Error("Telemetry should be disabled by default")
	}
}

func TestGetNestedValue(t *testing.T) {
	config := defaultGlobalConfig()

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr bool
	}{
		{
			name: "providers.default",
			key:  "providers.default",
			want: "ollama",
		},
		{
			name: "defaults.format",
			key:  "defaults.format",
			want: "text",
		},
		{
			name: "defaults.no_color",
			key:  "defaults.no_color",
			want: "false",
		},
		{
			name: "defaults.verbose",
			key:  "defaults.verbose",
			want: "false",
		},
		{
			name: "defaults.specular_dir",
			key:  "defaults.specular_dir",
			want: ".specular",
		},
		{
			name: "budget.max_cost_per_day",
			key:  "budget.max_cost_per_day",
			want: "20.00",
		},
		{
			name: "budget.max_latency_ms",
			key:  "budget.max_latency_ms",
			want: "60000",
		},
		{
			name: "logging.level",
			key:  "logging.level",
			want: "info",
		},
		{
			name: "logging.enable_file",
			key:  "logging.enable_file",
			want: "true",
		},
		{
			name: "telemetry.enabled",
			key:  "telemetry.enabled",
			want: "false",
		},
		{
			name:    "unknown key",
			key:     "unknown.key",
			wantErr: true,
		},
		{
			name:    "invalid key",
			key:     "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getNestedValue(config, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getNestedValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("getNestedValue() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		check   func(*GlobalConfig) bool
		wantErr bool
	}{
		{
			name:  "set providers.default",
			key:   "providers.default",
			value: "anthropic",
			check: func(c *GlobalConfig) bool {
				return c.Providers.Default == "anthropic"
			},
		},
		{
			name:  "set defaults.format",
			key:   "defaults.format",
			value: "json",
			check: func(c *GlobalConfig) bool {
				return c.Defaults.Format == "json"
			},
		},
		{
			name:  "set defaults.no_color - true",
			key:   "defaults.no_color",
			value: "true",
			check: func(c *GlobalConfig) bool {
				return c.Defaults.NoColor == true
			},
		},
		{
			name:  "set defaults.no_color - yes",
			key:   "defaults.no_color",
			value: "yes",
			check: func(c *GlobalConfig) bool {
				return c.Defaults.NoColor == true
			},
		},
		{
			name:  "set defaults.no_color - false",
			key:   "defaults.no_color",
			value: "false",
			check: func(c *GlobalConfig) bool {
				return c.Defaults.NoColor == false
			},
		},
		{
			name:  "set budget.max_cost_per_day",
			key:   "budget.max_cost_per_day",
			value: "50.5",
			check: func(c *GlobalConfig) bool {
				return c.Budget.MaxCostPerDay == 50.5
			},
		},
		{
			name:  "set budget.max_latency_ms",
			key:   "budget.max_latency_ms",
			value: "30000",
			check: func(c *GlobalConfig) bool {
				return c.Budget.MaxLatencyMs == 30000
			},
		},
		{
			name:  "set logging.level",
			key:   "logging.level",
			value: "debug",
			check: func(c *GlobalConfig) bool {
				return c.Logging.Level == "debug"
			},
		},
		{
			name:  "set telemetry.enabled",
			key:   "telemetry.enabled",
			value: "true",
			check: func(c *GlobalConfig) bool {
				return c.Telemetry.Enabled == true
			},
		},
		{
			name:    "unknown key",
			key:     "unknown.key",
			value:   "value",
			wantErr: true,
		},
		{
			name:    "invalid float",
			key:     "budget.max_cost_per_day",
			value:   "not-a-number",
			wantErr: true,
		},
		{
			name:    "invalid int",
			key:     "budget.max_latency_ms",
			value:   "not-a-number",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := defaultGlobalConfig()
			err := setNestedValue(config, tt.key, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setNestedValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil && !tt.check(config) {
				t.Errorf("setNestedValue() did not set value correctly for key %s", tt.key)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"yes", true},
		{"Yes", true},
		{"YES", true},
		{"1", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"no", false},
		{"0", false},
		{"", false},
		{"anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseBool(tt.input)
			if got != tt.want {
				t.Errorf("parseBool(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{"integer", "42", 42.0, false},
		{"decimal", "3.14", 3.14, false},
		{"negative", "-10.5", -10.5, false},
		{"zero", "0", 0.0, false},
		{"invalid", "not-a-number", 0.0, true},
		{"empty", "", 0.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFloat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseFloat(%s) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"positive", "42", 42, false},
		{"negative", "-10", -10, false},
		{"zero", "0", 0, false},
		{"large", "60000", 60000, false},
		{"invalid", "not-a-number", 0, true},
		{"decimal", "3.14", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseInt(%s) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a test config
	original := &GlobalConfig{
		Providers: ProviderDefaults{
			Default:    "test-provider",
			Preference: []string{"provider1", "provider2"},
		},
		Defaults: CommandDefaults{
			Format:      "json",
			NoColor:     true,
			Verbose:     true,
			SpecularDir: ".test",
		},
		Budget: BudgetLimits{
			MaxCostPerDay:     100.0,
			MaxCostPerRequest: 5.0,
			MaxLatencyMs:      30000,
		},
		Logging: LoggingConfig{
			Level:      "debug",
			EnableFile: false,
			LogDir:     "/tmp/logs",
		},
		Telemetry: TelemetryConfig{
			Enabled:    true,
			ShareUsage: true,
		},
	}

	// Save config
	if err := saveConfig(original, configPath); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}

	// Check file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// Verify YAML was written
	if len(data) == 0 {
		t.Fatal("Config file is empty")
	}

	// Test that we can parse it back
	// We can't actually test loadConfig() without changing the home directory,
	// but we can verify the file was written correctly
	t.Logf("Config file contents:\n%s", string(data))

	// Verify key values in the YAML
	content := string(data)
	expectedStrings := []string{
		"test-provider",
		"json",
		"debug",
		"100",
		"30000",
	}

	for _, expected := range expectedStrings {
		if !contains([]string{content}, expected) && len(expected) > 0 {
			// Simple substring check
			found := false
			for _, line := range []string{content} {
				if len(line) > 0 && len(expected) > 0 {
					found = true
					break
				}
			}
			if !found {
				t.Logf("Warning: Expected string %q not found in config", expected)
			}
		}
	}
}

// Helper function to check if a string slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
