// Package alerting provides alert routing and notification delivery for
// incident management integrations like PagerDuty, Opsgenie, and Slack.
package alerting

import (
	"context"
	"fmt"
	"time"
)

// Severity represents the alert severity level.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
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

// ToPagerDutySeverity converts to PagerDuty severity format.
func (s Severity) ToPagerDutySeverity() string {
	switch s {
	case SeverityCritical:
		return "critical"
	case SeverityHigh:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return "info"
	}
}

// ToOpsgeniePriority converts to Opsgenie priority format.
func (s Severity) ToOpsgeniePriority() string {
	switch s {
	case SeverityCritical:
		return "P1"
	case SeverityHigh:
		return "P2"
	case SeverityWarning:
		return "P3"
	default:
		return "P4"
	}
}

// Alert represents an alert to be sent to notification systems.
type Alert struct {
	// ID is a unique identifier for this alert
	ID string `json:"id"`

	// Title is the alert title/summary
	Title string `json:"title"`

	// Description provides detailed information about the alert
	Description string `json:"description"`

	// Severity is the alert severity level
	Severity Severity `json:"severity"`

	// DedupeKey is used to deduplicate alerts (e.g., for auto-resolution)
	DedupeKey string `json:"dedupe_key"`

	// Source identifies where the alert originated
	Source string `json:"source"`

	// Labels are additional metadata for the alert
	Labels map[string]string `json:"labels,omitempty"`

	// Timestamp is when the alert was created
	Timestamp time.Time `json:"timestamp"`

	// Links are related URLs
	Links []AlertLink `json:"links,omitempty"`
}

// AlertLink represents a related URL for an alert.
type AlertLink struct {
	Text string `json:"text"`
	Href string `json:"href"`
}

// AlertEvent represents an alert lifecycle event.
type AlertEvent string

const (
	AlertEventTrigger  AlertEvent = "trigger"
	AlertEventResolve  AlertEvent = "resolve"
	AlertEventAck      AlertEvent = "acknowledge"
	AlertEventUpdate   AlertEvent = "update"
)

// AlertManager defines the interface for sending and managing alerts.
type AlertManager interface {
	// Send sends an alert to the notification system.
	Send(ctx context.Context, alert *Alert) error

	// Resolve resolves/closes an alert by its dedupe key.
	Resolve(ctx context.Context, dedupeKey string) error

	// Test sends a test alert to verify connectivity.
	Test(ctx context.Context) error

	// Name returns the name of the alert manager (e.g., "pagerduty", "slack")
	Name() string
}

// Config holds common configuration for alert managers.
type Config struct {
	// Enabled controls whether the alert manager is active
	Enabled bool

	// DefaultSeverity is used when an alert doesn't specify severity
	DefaultSeverity Severity

	// Timeout is the HTTP request timeout
	Timeout time.Duration

	// RetryCount is the number of retries on failure
	RetryCount int

	// RetryDelay is the delay between retries
	RetryDelay time.Duration
}

// DefaultConfig returns the default alert manager configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:         false,
		DefaultSeverity: SeverityWarning,
		Timeout:         30 * time.Second,
		RetryCount:      3,
		RetryDelay:      1 * time.Second,
	}
}

// Router routes alerts to multiple alert managers.
type Router struct {
	managers []AlertManager
	config   Config
}

// RouterOption configures the Router.
type RouterOption func(*Router)

// WithManager adds an alert manager to the router.
func WithManager(m AlertManager) RouterOption {
	return func(r *Router) {
		r.managers = append(r.managers, m)
	}
}

// WithConfig sets the router configuration.
func WithConfig(c Config) RouterOption {
	return func(r *Router) {
		r.config = c
	}
}

// NewRouter creates a new alert router.
func NewRouter(opts ...RouterOption) *Router {
	r := &Router{
		managers: make([]AlertManager, 0),
		config:   DefaultConfig(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Send sends an alert to all configured managers.
func (r *Router) Send(ctx context.Context, alert *Alert) error {
	if len(r.managers) == 0 {
		return nil
	}

	// Set default severity if not specified
	if !alert.Severity.IsValid() {
		alert.Severity = r.config.DefaultSeverity
	}

	// Set timestamp if not specified
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	var errs []error
	for _, m := range r.managers {
		if err := m.Send(ctx, alert); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", m.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to send alert: %v", errs)
	}

	return nil
}

// Resolve resolves an alert across all configured managers.
func (r *Router) Resolve(ctx context.Context, dedupeKey string) error {
	if len(r.managers) == 0 {
		return nil
	}

	var errs []error
	for _, m := range r.managers {
		if err := m.Resolve(ctx, dedupeKey); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", m.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to resolve alert: %v", errs)
	}

	return nil
}

// Test tests connectivity to all configured managers.
func (r *Router) Test(ctx context.Context) error {
	if len(r.managers) == 0 {
		return nil
	}

	var errs []error
	for _, m := range r.managers {
		if err := m.Test(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", m.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("connectivity test failed: %v", errs)
	}

	return nil
}

// Managers returns the list of configured alert managers.
func (r *Router) Managers() []AlertManager {
	return r.managers
}

// NewAlert creates a new alert with the given parameters.
func NewAlert(title, description string, severity Severity) *Alert {
	return &Alert{
		ID:          generateID(),
		Title:       title,
		Description: description,
		Severity:    severity,
		Source:      "specular",
		Timestamp:   time.Now(),
		Labels:      make(map[string]string),
	}
}

// WithDedupeKey sets the dedupe key for the alert.
func (a *Alert) WithDedupeKey(key string) *Alert {
	a.DedupeKey = key
	return a
}

// WithLabel adds a label to the alert.
func (a *Alert) WithLabel(key, value string) *Alert {
	if a.Labels == nil {
		a.Labels = make(map[string]string)
	}
	a.Labels[key] = value
	return a
}

// WithLink adds a link to the alert.
func (a *Alert) WithLink(text, href string) *Alert {
	a.Links = append(a.Links, AlertLink{Text: text, Href: href})
	return a
}

// generateID generates a unique alert ID.
func generateID() string {
	return fmt.Sprintf("alert-%d", time.Now().UnixNano())
}
