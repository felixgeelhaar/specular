package health

import (
	"context"
	"sync/atomic"
	"time"
)

// ProbeManager extends Manager with Kubernetes-style probe support.
// It tracks initialization and shutdown state for liveness, readiness, and startup probes.
type ProbeManager struct {
	*Manager

	// Application state tracking
	startTime   time.Time
	initialized atomic.Bool
	inShutdown  atomic.Bool
	version     string
}

// NewProbeManager creates a new health check manager with probe support.
func NewProbeManager(version string) *ProbeManager {
	return &ProbeManager{
		Manager:   NewManager(),
		startTime: time.Now(),
		version:   version,
	}
}

// MarkInitialized marks the application as fully initialized.
// This allows the startup probe to pass.
func (pm *ProbeManager) MarkInitialized() {
	pm.initialized.Store(true)
}

// MarkShutdown marks the application as shutting down.
// This causes readiness probes to fail, removing the pod from service endpoints.
func (pm *ProbeManager) MarkShutdown() {
	pm.inShutdown.Store(true)
}

// IsInitialized returns whether the application is fully initialized.
func (pm *ProbeManager) IsInitialized() bool {
	return pm.initialized.Load()
}

// IsShuttingDown returns whether the application is shutting down.
func (pm *ProbeManager) IsShuttingDown() bool {
	return pm.inShutdown.Load()
}

// Uptime returns how long the application has been running.
func (pm *ProbeManager) Uptime() time.Duration {
	return time.Since(pm.startTime)
}

// Version returns the application version.
func (pm *ProbeManager) Version() string {
	return pm.version
}

// ProbeResult represents a Kubernetes probe check result.
type ProbeResult struct {
	Status    Status                 `json:"status"`
	Version   string                 `json:"version,omitempty"`
	Uptime    string                 `json:"uptime,omitempty"`
	Checks    map[string]*Result     `json:"checks,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// CheckLiveness performs a liveness probe check.
//
// Liveness probes determine if the application is alive and responsive.
// If this check fails, Kubernetes will restart the container.
//
// Returns:
//   - StatusHealthy if the application is running normally
//   - StatusDegraded if the application is shutting down (but still alive)
//
// This check does NOT run dependency checks - it only verifies the process is responsive.
func (pm *ProbeManager) CheckLiveness(ctx context.Context) *ProbeResult {
	status := StatusHealthy
	if pm.IsShuttingDown() {
		status = StatusDegraded
	}

	return &ProbeResult{
		Status:    status,
		Version:   pm.version,
		Uptime:    pm.Uptime().Round(time.Second).String(),
		Checks:    make(map[string]*Result),
		Timestamp: time.Now(),
	}
}

// CheckReadiness performs a readiness probe check.
//
// Readiness probes determine if the application is ready to accept traffic.
// If this check fails, Kubernetes removes the pod from service endpoints.
//
// Returns:
//   - StatusUnhealthy immediately if shutting down
//   - Otherwise, aggregates all registered health checks
//
// This check DOES run dependency checks to verify the application can serve requests.
func (pm *ProbeManager) CheckReadiness(ctx context.Context) *ProbeResult {
	// If shutting down, immediately return not ready
	if pm.IsShuttingDown() {
		return &ProbeResult{
			Status:    StatusUnhealthy,
			Version:   pm.version,
			Uptime:    pm.Uptime().Round(time.Second).String(),
			Checks:    make(map[string]*Result),
			Timestamp: time.Now(),
		}
	}

	// Run all registered health checks
	checks := pm.Manager.Check(ctx)
	overallStatus := pm.Manager.OverallStatus(checks)

	return &ProbeResult{
		Status:    overallStatus,
		Version:   pm.version,
		Uptime:    pm.Uptime().Round(time.Second).String(),
		Checks:    checks,
		Timestamp: time.Now(),
	}
}

// CheckStartup performs a startup probe check.
//
// Startup probes determine if the application has finished initialization.
// Kubernetes waits for this probe to pass before checking liveness/readiness.
// This allows slow-starting applications more time to initialize.
//
// Returns:
//   - StatusHealthy if the application is initialized
//   - StatusUnhealthy if the application is still starting up
//
// This check does NOT run dependency checks - it only verifies initialization is complete.
func (pm *ProbeManager) CheckStartup(ctx context.Context) *ProbeResult {
	status := StatusUnhealthy
	if pm.IsInitialized() {
		status = StatusHealthy
	}

	return &ProbeResult{
		Status:    status,
		Version:   pm.version,
		Uptime:    pm.Uptime().Round(time.Second).String(),
		Checks:    make(map[string]*Result),
		Timestamp: time.Now(),
	}
}
