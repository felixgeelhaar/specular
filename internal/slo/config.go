package slo

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the SLO package configuration.
type Config struct {
	// Enabled controls whether SLO tracking is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`

	// ConfigPath is the path to an external SLO definitions file
	ConfigPath string `yaml:"config_path,omitempty" json:"config_path,omitempty"`

	// DefaultWindow is the default SLO window if not specified
	DefaultWindow Duration `yaml:"default_window,omitempty" json:"default_window,omitempty"`

	// CacheTTL is how long to cache SLO status calculations
	CacheTTL Duration `yaml:"cache_ttl,omitempty" json:"cache_ttl,omitempty"`

	// SLOs are inline SLO definitions (alternative to ConfigPath)
	SLOs []*SLO `yaml:"slos,omitempty" json:"slos,omitempty"`
}

// SLOFile represents an external SLO configuration file.
type SLOFile struct {
	// Version is the configuration file version
	Version string `yaml:"version" json:"version"`

	// SLOs are the SLO definitions
	SLOs []*SLO `yaml:"slos" json:"slos"`
}

// DefaultConfig returns the default SLO configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:       false,
		DefaultWindow: Duration(30 * 24 * 60 * 60 * 1e9), // 30 days in nanoseconds
		CacheTTL:      Duration(60 * 1e9),                // 1 minute in nanoseconds
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.ConfigPath != "" && len(c.SLOs) > 0 {
		return fmt.Errorf("cannot specify both config_path and inline slos")
	}

	for _, slo := range c.SLOs {
		if err := slo.Validate(); err != nil {
			return fmt.Errorf("invalid SLO %s: %w", slo.Name, err)
		}
	}

	return nil
}

// LoadSLOs loads SLOs from either the config path or inline definitions.
func (c *Config) LoadSLOs() ([]*SLO, error) {
	if c.ConfigPath != "" {
		return LoadSLOsFromFile(c.ConfigPath)
	}

	if len(c.SLOs) > 0 {
		return c.SLOs, nil
	}

	// Return default SLOs if none specified
	return DefaultSLOs(), nil
}

// LoadSLOsFromFile loads SLO definitions from a YAML file.
func LoadSLOsFromFile(path string) ([]*SLO, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read SLO file: %w", err)
	}

	var file SLOFile
	if unmarshalErr := yaml.Unmarshal(data, &file); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse SLO file: %w", unmarshalErr)
	}

	// Validate all SLOs
	for _, slo := range file.SLOs {
		if validateErr := slo.Validate(); validateErr != nil {
			return nil, fmt.Errorf("invalid SLO %s: %w", slo.Name, validateErr)
		}
	}

	return file.SLOs, nil
}

// SaveSLOsToFile saves SLO definitions to a YAML file.
func SaveSLOsToFile(path string, slos []*SLO) error {
	file := SLOFile{
		Version: "1.0",
		SLOs:    slos,
	}

	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("failed to marshal SLO file: %w", err)
	}

	if writeErr := os.WriteFile(path, data, 0600); writeErr != nil {
		return fmt.Errorf("failed to write SLO file: %w", writeErr)
	}

	return nil
}

// ExampleSLOFile returns an example SLO configuration file content.
func ExampleSLOFile() string {
	return `# Specular SLO Configuration
# See https://specular.dev/docs/slo for more information

version: "1.0"

slos:
  - name: command-success-rate
    description: Specular commands complete without errors
    target: 0.995  # 99.5%
    window: 30d
    sli:
      type: availability
      metric: specular_command_executions_total
    alert_policy:
      burn_rate_threshold: 14.4
      short_window: 5m
      long_window: 1h
      severity: high
    labels:
      team: platform
      service: specular-cli

  - name: provider-latency-p95
    description: Provider API calls complete within 30 seconds
    target: 0.95  # 95%
    window: 30d
    sli:
      type: latency
      metric: specular_provider_latency_seconds
      threshold: 30s
    alert_policy:
      burn_rate_threshold: 6.0
      short_window: 5m
      long_window: 1h
      severity: warning
    labels:
      team: platform
      service: specular-cli

  - name: auto-mode-success
    description: Autonomous workflows complete successfully
    target: 0.99  # 99%
    window: 30d
    sli:
      type: availability
      metric: specular_auto_workflows_total
    alert_policy:
      burn_rate_threshold: 14.4
      short_window: 5m
      long_window: 1h
      severity: high
    labels:
      team: platform
      service: specular-cli
`
}
