// Package slo provides Service Level Objective (SLO) definitions and tracking
// for monitoring application reliability and performance.
package slo

import (
	"fmt"
	"time"
)

// SLO represents a Service Level Objective definition.
// An SLO defines a target for a specific SLI (Service Level Indicator)
// over a given time window.
type SLO struct {
	// Name is the unique identifier for this SLO
	Name string `yaml:"name" json:"name"`

	// Description provides human-readable context about this SLO
	Description string `yaml:"description" json:"description"`

	// Target is the SLO target as a ratio (0.0-1.0)
	// For example, 0.999 means 99.9% availability
	Target float64 `yaml:"target" json:"target"`

	// Window is the time window over which the SLO is evaluated
	Window Duration `yaml:"window" json:"window"`

	// SLI defines the Service Level Indicator for this SLO
	SLI SLISpec `yaml:"sli" json:"sli"`

	// AlertPolicy defines when to alert on SLO violations
	AlertPolicy *AlertPolicy `yaml:"alert_policy,omitempty" json:"alert_policy,omitempty"`

	// Labels are additional metadata for this SLO
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// SLISpec defines the Service Level Indicator specification.
type SLISpec struct {
	// Type is the type of SLI (availability, latency, error_rate, throughput)
	Type SLIType `yaml:"type" json:"type"`

	// Metric is the Prometheus metric name to use for this SLI
	Metric string `yaml:"metric" json:"metric"`

	// Threshold is used for latency-based SLIs (e.g., 500ms)
	// Requests faster than this threshold are considered "good"
	Threshold time.Duration `yaml:"threshold,omitempty" json:"threshold,omitempty"`

	// GoodQuery is the PromQL query for "good" events (optional, for custom SLIs)
	GoodQuery string `yaml:"good_query,omitempty" json:"good_query,omitempty"`

	// TotalQuery is the PromQL query for total events (optional, for custom SLIs)
	TotalQuery string `yaml:"total_query,omitempty" json:"total_query,omitempty"`
}

// SLIType represents the type of Service Level Indicator.
type SLIType string

const (
	// SLITypeAvailability measures the ratio of successful requests to total requests
	SLITypeAvailability SLIType = "availability"

	// SLITypeLatency measures the ratio of requests within a latency threshold
	SLITypeLatency SLIType = "latency"

	// SLITypeErrorRate measures the inverse of error rate (1 - errors/total)
	SLITypeErrorRate SLIType = "error_rate"

	// SLITypeThroughput measures the ratio of processed requests to expected throughput
	SLITypeThroughput SLIType = "throughput"

	// SLITypeCustom allows for custom PromQL queries
	SLITypeCustom SLIType = "custom"
)

// IsValid returns true if the SLI type is valid.
func (t SLIType) IsValid() bool {
	switch t {
	case SLITypeAvailability, SLITypeLatency, SLITypeErrorRate, SLITypeThroughput, SLITypeCustom:
		return true
	default:
		return false
	}
}

// String returns the string representation of the SLI type.
func (t SLIType) String() string {
	return string(t)
}

// AlertPolicy defines alerting behavior for SLO violations.
type AlertPolicy struct {
	// BurnRateThreshold is the burn rate that triggers an alert
	// A burn rate of 1.0 means consuming error budget at exactly the expected rate
	// A burn rate of 14.4 means consuming the monthly error budget in 2 days
	BurnRateThreshold float64 `yaml:"burn_rate_threshold" json:"burn_rate_threshold"`

	// ShortWindow is the short alerting window (e.g., 5m)
	ShortWindow Duration `yaml:"short_window" json:"short_window"`

	// LongWindow is the long alerting window (e.g., 1h)
	LongWindow Duration `yaml:"long_window" json:"long_window"`

	// Severity is the alert severity (critical, high, warning, info)
	Severity Severity `yaml:"severity" json:"severity"`

	// NotificationChannels are the channels to notify on alert
	NotificationChannels []string `yaml:"notification_channels,omitempty" json:"notification_channels,omitempty"`
}

// Severity represents the alert severity level.
type Severity string

const (
	// SeverityCritical represents critical severity requiring immediate attention.
	SeverityCritical Severity = "critical"
	// SeverityHigh represents high severity requiring prompt attention.
	SeverityHigh Severity = "high"
	// SeverityWarning represents warning severity for potential issues.
	SeverityWarning Severity = "warning"
	// SeverityInfo represents informational severity for normal events.
	SeverityInfo Severity = "info"
)

// IsValid returns true if the severity is valid.
func (s Severity) IsValid() bool {
	switch s {
	case SeverityCritical, SeverityHigh, SeverityWarning, SeverityInfo:
		return true
	default:
		return false
	}
}

// Duration is a wrapper around time.Duration that supports YAML unmarshaling
// from human-readable strings like "30d", "1h", "5m".
type Duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler for Duration.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	duration, err := ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(duration)
	return nil
}

// MarshalYAML implements yaml.Marshaler for Duration.
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// UnmarshalJSON implements json.Unmarshaler for Duration.
func (d *Duration) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	duration, err := ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(duration)
	return nil
}

// MarshalJSON implements json.Marshaler for Duration.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

// Duration returns the time.Duration value.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// String returns a human-readable string representation.
func (d Duration) String() string {
	dur := time.Duration(d)

	// Handle days
	if dur >= 24*time.Hour {
		days := dur / (24 * time.Hour)
		return fmt.Sprintf("%dd", days)
	}

	return dur.String()
}

// ParseDuration parses a duration string with support for days (d).
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	// Check for day suffix
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err != nil {
			return 0, fmt.Errorf("invalid duration format: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	return time.ParseDuration(s)
}

// Validate validates the SLO configuration.
func (s *SLO) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("SLO name is required")
	}

	if s.Target <= 0 || s.Target > 1 {
		return fmt.Errorf("SLO target must be between 0 and 1 (exclusive of 0), got %f", s.Target)
	}

	if s.Window.Duration() <= 0 {
		return fmt.Errorf("SLO window must be positive")
	}

	if !s.SLI.Type.IsValid() {
		return fmt.Errorf("invalid SLI type: %s", s.SLI.Type)
	}

	if s.SLI.Type == SLITypeLatency && s.SLI.Threshold <= 0 {
		return fmt.Errorf("latency SLI requires a positive threshold")
	}

	if s.SLI.Type == SLITypeCustom {
		if s.SLI.GoodQuery == "" || s.SLI.TotalQuery == "" {
			return fmt.Errorf("custom SLI requires both good_query and total_query")
		}
	}

	if s.AlertPolicy != nil {
		if err := s.AlertPolicy.Validate(); err != nil {
			return fmt.Errorf("invalid alert policy: %w", err)
		}
	}

	return nil
}

// Validate validates the alert policy configuration.
func (p *AlertPolicy) Validate() error {
	if p.BurnRateThreshold <= 0 {
		return fmt.Errorf("burn rate threshold must be positive")
	}

	if p.ShortWindow.Duration() <= 0 {
		return fmt.Errorf("short window must be positive")
	}

	if p.LongWindow.Duration() <= 0 {
		return fmt.Errorf("long window must be positive")
	}

	if p.ShortWindow.Duration() >= p.LongWindow.Duration() {
		return fmt.Errorf("short window must be shorter than long window")
	}

	if !p.Severity.IsValid() {
		return fmt.Errorf("invalid severity: %s", p.Severity)
	}

	return nil
}

// ErrorBudget calculates the error budget for this SLO.
// The error budget is the allowed failure rate: 1 - target.
func (s *SLO) ErrorBudget() float64 {
	return 1 - s.Target
}

// ErrorBudgetMinutes calculates the error budget in minutes for this SLO.
func (s *SLO) ErrorBudgetMinutes() float64 {
	windowMinutes := s.Window.Duration().Minutes()
	return windowMinutes * s.ErrorBudget()
}

// DefaultSLOs returns a set of default SLOs for Specular.
func DefaultSLOs() []*SLO {
	return []*SLO{
		{
			Name:        "command-success-rate",
			Description: "Specular commands complete without errors",
			Target:      0.995, // 99.5%
			Window:      Duration(30 * 24 * time.Hour),
			SLI: SLISpec{
				Type:   SLITypeAvailability,
				Metric: "specular_command_executions_total",
			},
			AlertPolicy: &AlertPolicy{
				BurnRateThreshold: 14.4, // 2-day budget consumption
				ShortWindow:       Duration(5 * time.Minute),
				LongWindow:        Duration(1 * time.Hour),
				Severity:          SeverityHigh,
			},
			Labels: map[string]string{
				"team":    "platform",
				"service": "specular-cli",
			},
		},
		{
			Name:        "provider-latency-p95",
			Description: "Provider API calls complete within 30 seconds",
			Target:      0.95, // 95%
			Window:      Duration(30 * 24 * time.Hour),
			SLI: SLISpec{
				Type:      SLITypeLatency,
				Metric:    "specular_provider_latency_seconds",
				Threshold: 30 * time.Second,
			},
			AlertPolicy: &AlertPolicy{
				BurnRateThreshold: 6.0, // 5-day budget consumption
				ShortWindow:       Duration(5 * time.Minute),
				LongWindow:        Duration(1 * time.Hour),
				Severity:          SeverityWarning,
			},
			Labels: map[string]string{
				"team":    "platform",
				"service": "specular-cli",
			},
		},
		{
			Name:        "auto-mode-success",
			Description: "Autonomous workflows complete successfully",
			Target:      0.99, // 99%
			Window:      Duration(30 * 24 * time.Hour),
			SLI: SLISpec{
				Type:   SLITypeAvailability,
				Metric: "specular_auto_workflows_total",
			},
			AlertPolicy: &AlertPolicy{
				BurnRateThreshold: 14.4,
				ShortWindow:       Duration(5 * time.Minute),
				LongWindow:        Duration(1 * time.Hour),
				Severity:          SeverityHigh,
			},
			Labels: map[string]string{
				"team":    "platform",
				"service": "specular-cli",
			},
		},
		{
			Name:        "plan-generation-latency",
			Description: "Plans are generated within 60 seconds",
			Target:      0.95, // 95%
			Window:      Duration(30 * 24 * time.Hour),
			SLI: SLISpec{
				Type:      SLITypeLatency,
				Metric:    "specular_plan_duration_seconds",
				Threshold: 60 * time.Second,
			},
			AlertPolicy: &AlertPolicy{
				BurnRateThreshold: 6.0,
				ShortWindow:       Duration(5 * time.Minute),
				LongWindow:        Duration(1 * time.Hour),
				Severity:          SeverityWarning,
			},
			Labels: map[string]string{
				"team":    "platform",
				"service": "specular-cli",
			},
		},
	}
}
