package health

import (
	"context"
	"testing"
	"time"
)

func TestNewProbeManager(t *testing.T) {
	version := "1.0.0"
	pm := NewProbeManager(version)

	if pm.Version() != version {
		t.Errorf("expected version %s, got %s", version, pm.Version())
	}

	if pm.IsInitialized() {
		t.Error("probe manager should not be initialized by default")
	}

	if pm.IsShuttingDown() {
		t.Error("probe manager should not be shutting down by default")
	}

	if pm.Manager == nil {
		t.Error("underlying manager should be initialized")
	}
}

func TestProbeManagerMarkInitialized(t *testing.T) {
	pm := NewProbeManager("1.0.0")

	if pm.IsInitialized() {
		t.Error("should not be initialized initially")
	}

	pm.MarkInitialized()

	if !pm.IsInitialized() {
		t.Error("should be initialized after MarkInitialized")
	}
}

func TestProbeManagerMarkShutdown(t *testing.T) {
	pm := NewProbeManager("1.0.0")

	if pm.IsShuttingDown() {
		t.Error("should not be shutting down initially")
	}

	pm.MarkShutdown()

	if !pm.IsShuttingDown() {
		t.Error("should be shutting down after MarkShutdown")
	}
}

func TestProbeManagerUptime(t *testing.T) {
	pm := NewProbeManager("1.0.0")

	// Wait a bit to ensure uptime is non-zero
	time.Sleep(10 * time.Millisecond)

	uptime := pm.Uptime()
	if uptime == 0 {
		t.Error("uptime should be non-zero")
	}

	if uptime < 10*time.Millisecond {
		t.Errorf("uptime should be at least 10ms, got %v", uptime)
	}
}

func TestCheckLiveness(t *testing.T) {
	tests := []struct {
		name           string
		inShutdown     bool
		expectedStatus Status
	}{
		{
			name:           "normal operation",
			inShutdown:     false,
			expectedStatus: StatusHealthy,
		},
		{
			name:           "during shutdown",
			inShutdown:     true,
			expectedStatus: StatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewProbeManager("1.0.0")
			if tt.inShutdown {
				pm.MarkShutdown()
			}

			ctx := context.Background()
			result := pm.CheckLiveness(ctx)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			if result.Version != "1.0.0" {
				t.Errorf("expected version 1.0.0, got %s", result.Version)
			}

			if result.Uptime == "" {
				t.Error("uptime should be set")
			}

			if result.Checks == nil {
				t.Error("checks should not be nil")
			}

			if result.Timestamp.IsZero() {
				t.Error("timestamp should be set")
			}
		})
	}
}

func TestCheckReadiness(t *testing.T) {
	tests := []struct {
		name           string
		inShutdown     bool
		checkers       []Checker
		expectedStatus Status
		expectChecks   bool
	}{
		{
			name:           "in shutdown",
			inShutdown:     true,
			checkers:       nil,
			expectedStatus: StatusUnhealthy,
			expectChecks:   false,
		},
		{
			name:       "all checks passing",
			inShutdown: false,
			checkers: []Checker{
				&mockChecker{name: "database", result: Healthy("ok")},
				&mockChecker{name: "cache", result: Healthy("ok")},
			},
			expectedStatus: StatusHealthy,
			expectChecks:   true,
		},
		{
			name:       "one check degraded",
			inShutdown: false,
			checkers: []Checker{
				&mockChecker{name: "database", result: Healthy("ok")},
				&mockChecker{name: "cache", result: Degraded("slow")},
			},
			expectedStatus: StatusDegraded,
			expectChecks:   true,
		},
		{
			name:       "one check unhealthy",
			inShutdown: false,
			checkers: []Checker{
				&mockChecker{name: "database", result: Healthy("ok")},
				&mockChecker{name: "cache", result: Unhealthy("down")},
			},
			expectedStatus: StatusUnhealthy,
			expectChecks:   true,
		},
		{
			name:           "no checkers registered",
			inShutdown:     false,
			checkers:       nil,
			expectedStatus: StatusHealthy,
			expectChecks:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewProbeManager("1.0.0")

			if tt.inShutdown {
				pm.MarkShutdown()
			}

			for _, checker := range tt.checkers {
				pm.AddChecker(checker)
			}

			ctx := context.Background()
			result := pm.CheckReadiness(ctx)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			if tt.expectChecks && len(result.Checks) == 0 {
				t.Error("expected checks to be present")
			}

			if !tt.expectChecks && len(result.Checks) > 0 {
				t.Error("expected no checks")
			}
		})
	}
}

func TestCheckStartup(t *testing.T) {
	tests := []struct {
		name           string
		initialized    bool
		expectedStatus Status
	}{
		{
			name:           "not initialized",
			initialized:    false,
			expectedStatus: StatusUnhealthy,
		},
		{
			name:           "initialized",
			initialized:    true,
			expectedStatus: StatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewProbeManager("1.0.0")
			if tt.initialized {
				pm.MarkInitialized()
			}

			ctx := context.Background()
			result := pm.CheckStartup(ctx)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			if result.Version != "1.0.0" {
				t.Errorf("expected version 1.0.0, got %s", result.Version)
			}

			if result.Checks == nil {
				t.Error("checks should not be nil")
			}
		})
	}
}

func TestProbeResultFields(t *testing.T) {
	pm := NewProbeManager("2.0.0")
	pm.MarkInitialized()

	// Add a checker
	pm.AddChecker(&mockChecker{name: "test", result: Healthy("ok")})

	ctx := context.Background()

	// Test liveness result
	liveness := pm.CheckLiveness(ctx)
	if liveness.Version != "2.0.0" {
		t.Errorf("liveness version: expected 2.0.0, got %s", liveness.Version)
	}
	if liveness.Timestamp.IsZero() {
		t.Error("liveness timestamp should be set")
	}
	if liveness.Uptime == "" {
		t.Error("liveness uptime should be set")
	}

	// Test readiness result
	readiness := pm.CheckReadiness(ctx)
	if readiness.Version != "2.0.0" {
		t.Errorf("readiness version: expected 2.0.0, got %s", readiness.Version)
	}
	if len(readiness.Checks) == 0 {
		t.Error("readiness should have checks")
	}

	// Test startup result
	startup := pm.CheckStartup(ctx)
	if startup.Status != StatusHealthy {
		t.Errorf("startup status: expected healthy (initialized), got %s", startup.Status)
	}
}

func TestConcurrentProbeAccess(t *testing.T) {
	pm := NewProbeManager("1.0.0")

	// Add some checkers
	for i := 0; i < 5; i++ {
		pm.AddChecker(&mockChecker{
			name:   string(rune('a' + i)),
			result: Healthy("ok"),
		})
	}

	done := make(chan bool)
	ctx := context.Background()

	// Multiple goroutines checking liveness
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				pm.CheckLiveness(ctx)
			}
			done <- true
		}()
	}

	// Multiple goroutines checking readiness
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				pm.CheckReadiness(ctx)
			}
			done <- true
		}()
	}

	// Multiple goroutines checking startup
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				pm.CheckStartup(ctx)
			}
			done <- true
		}()
	}

	// Goroutines toggling state
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				pm.MarkInitialized()
				pm.MarkShutdown()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 35; i++ {
		<-done
	}

	// If we get here without panic, concurrent access is safe
}

func TestProbeWorkflow(t *testing.T) {
	// Simulate a complete application lifecycle
	pm := NewProbeManager("1.0.0")

	ctx := context.Background()

	// 1. Application starting up
	startup := pm.CheckStartup(ctx)
	if startup.Status != StatusUnhealthy {
		t.Error("startup probe should fail before initialization")
	}

	liveness := pm.CheckLiveness(ctx)
	if liveness.Status != StatusHealthy {
		t.Error("liveness probe should pass during startup")
	}

	readiness := pm.CheckReadiness(ctx)
	if readiness.Status != StatusHealthy {
		// No checkers registered, so should be healthy
		t.Error("readiness probe should pass (no dependencies yet)")
	}

	// 2. Application initialized
	pm.MarkInitialized()

	startup = pm.CheckStartup(ctx)
	if startup.Status != StatusHealthy {
		t.Error("startup probe should pass after initialization")
	}

	// 3. Add dependency checkers
	pm.AddChecker(&mockChecker{name: "database", result: Healthy("ok")})

	readiness = pm.CheckReadiness(ctx)
	if readiness.Status != StatusHealthy {
		t.Error("readiness probe should pass with healthy dependencies")
	}

	// 4. Application shutting down
	pm.MarkShutdown()

	liveness = pm.CheckLiveness(ctx)
	if liveness.Status != StatusDegraded {
		t.Error("liveness probe should be degraded during shutdown")
	}

	readiness = pm.CheckReadiness(ctx)
	if readiness.Status != StatusUnhealthy {
		t.Error("readiness probe should fail during shutdown")
	}

	// Startup should still pass (already initialized)
	startup = pm.CheckStartup(ctx)
	if startup.Status != StatusHealthy {
		t.Error("startup probe should still pass during shutdown")
	}
}
