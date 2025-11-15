// Package health provides health check functionality for monitoring system dependencies.
//
// The health package follows the standard health check pattern with:
//   - Checker interface for pluggable health checks
//   - Result type with status, message, and details
//   - Status enum (Healthy, Degraded, Unhealthy)
//   - Built-in checkers for common dependencies
//
// Example usage:
//
//	manager := health.NewManager()
//	manager.AddChecker(health.NewDockerChecker())
//	manager.AddChecker(health.NewProviderChecker(providers))
//
//	results := manager.Check(ctx)
//	for name, result := range results {
//	    log.Info("Health check", "name", name, "status", result.Status)
//	}
package health

import (
	"context"
	"time"
)

// Checker defines the interface for health checks.
// Each checker should verify a specific system dependency or capability.
type Checker interface {
	// Name returns the unique name of this health check.
	// Should be lowercase with hyphens (e.g., "docker-daemon", "git-binary").
	Name() string

	// Check performs the health check and returns the result.
	// It should respect the context deadline and return quickly.
	// Typical timeout is 5 seconds per check.
	Check(ctx context.Context) *Result
}

// Status represents the health check status.
type Status string

const (
	// StatusHealthy indicates the checked component is fully operational.
	StatusHealthy Status = "healthy"

	// StatusDegraded indicates the component is partially working.
	// The application can continue but with reduced functionality.
	StatusDegraded Status = "degraded"

	// StatusUnhealthy indicates the component is not working.
	// The application may not function correctly.
	StatusUnhealthy Status = "unhealthy"
)

// String returns the string representation of the status.
func (s Status) String() string {
	return string(s)
}

// Result represents the result of a health check.
type Result struct {
	// Status is the health status (healthy, degraded, unhealthy).
	Status Status

	// Message is a human-readable description of the status.
	Message string

	// Details contains additional structured information about the check.
	// This can include version numbers, response times, error details, etc.
	Details map[string]interface{}

	// Latency is how long the health check took to complete.
	Latency time.Duration
}

// NewResult creates a new health check result with the given status and message.
func NewResult(status Status, message string) *Result {
	return &Result{
		Status:  status,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// WithDetail adds a detail to the result and returns the result for chaining.
func (r *Result) WithDetail(key string, value interface{}) *Result {
	r.Details[key] = value
	return r
}

// WithLatency sets the latency and returns the result for chaining.
func (r *Result) WithLatency(latency time.Duration) *Result {
	r.Latency = latency
	return r
}

// Healthy creates a healthy result with the given message.
func Healthy(message string) *Result {
	return NewResult(StatusHealthy, message)
}

// Degraded creates a degraded result with the given message.
func Degraded(message string) *Result {
	return NewResult(StatusDegraded, message)
}

// Unhealthy creates an unhealthy result with the given message.
func Unhealthy(message string) *Result {
	return NewResult(StatusUnhealthy, message)
}
