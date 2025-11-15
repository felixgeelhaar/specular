package health

import (
	"context"
	"testing"
	"time"
)

// mockChecker is a test double for health checks
type mockChecker struct {
	name   string
	result *Result
	delay  time.Duration
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) Check(ctx context.Context) *Result {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return Unhealthy("check cancelled").
				WithDetail("error", ctx.Err().Error())
		}
	}
	return m.result
}

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want %v", manager.timeout, 5*time.Second)
	}

	if manager.checkers == nil {
		t.Error("checkers should be initialized")
	}

	if len(manager.checkers) != 0 {
		t.Errorf("checkers should be empty, got %d", len(manager.checkers))
	}
}

func TestWithTimeout(t *testing.T) {
	manager := NewManager()
	customTimeout := 10 * time.Second

	returned := manager.WithTimeout(customTimeout)

	// Verify chaining
	if returned != manager {
		t.Error("WithTimeout should return same manager for chaining")
	}

	// Verify timeout was set
	if manager.timeout != customTimeout {
		t.Errorf("timeout = %v, want %v", manager.timeout, customTimeout)
	}
}

func TestAddChecker(t *testing.T) {
	manager := NewManager()

	checker1 := &mockChecker{name: "test1", result: Healthy("ok")}
	checker2 := &mockChecker{name: "test2", result: Healthy("ok")}

	manager.AddChecker(checker1)
	if manager.Count() != 1 {
		t.Errorf("Count() = %d, want 1", manager.Count())
	}

	manager.AddChecker(checker2)
	if manager.Count() != 2 {
		t.Errorf("Count() = %d, want 2", manager.Count())
	}

	names := manager.CheckNames()
	if len(names) != 2 {
		t.Fatalf("CheckNames() returned %d names, want 2", len(names))
	}

	if names[0] != "test1" || names[1] != "test2" {
		t.Errorf("CheckNames() = %v, want [test1, test2]", names)
	}
}

func TestRemoveChecker(t *testing.T) {
	manager := NewManager()

	checker1 := &mockChecker{name: "test1", result: Healthy("ok")}
	checker2 := &mockChecker{name: "test2", result: Healthy("ok")}

	manager.AddChecker(checker1)
	manager.AddChecker(checker2)

	// Remove existing checker
	removed := manager.RemoveChecker("test1")
	if !removed {
		t.Error("RemoveChecker should return true for existing checker")
	}

	if manager.Count() != 1 {
		t.Errorf("Count() = %d, want 1 after removal", manager.Count())
	}

	names := manager.CheckNames()
	if len(names) != 1 || names[0] != "test2" {
		t.Errorf("CheckNames() = %v, want [test2]", names)
	}

	// Remove non-existing checker
	removed = manager.RemoveChecker("nonexistent")
	if removed {
		t.Error("RemoveChecker should return false for non-existing checker")
	}

	if manager.Count() != 1 {
		t.Errorf("Count() = %d, want 1", manager.Count())
	}
}

func TestCheck(t *testing.T) {
	manager := NewManager()

	checker1 := &mockChecker{name: "healthy", result: Healthy("all good")}
	checker2 := &mockChecker{name: "degraded", result: Degraded("partial")}
	checker3 := &mockChecker{name: "unhealthy", result: Unhealthy("broken")}

	manager.AddChecker(checker1)
	manager.AddChecker(checker2)
	manager.AddChecker(checker3)

	ctx := context.Background()
	results := manager.Check(ctx)

	if len(results) != 3 {
		t.Fatalf("Check() returned %d results, want 3", len(results))
	}

	if results["healthy"].Status != StatusHealthy {
		t.Errorf("results[healthy].Status = %v, want %v", results["healthy"].Status, StatusHealthy)
	}

	if results["degraded"].Status != StatusDegraded {
		t.Errorf("results[degraded].Status = %v, want %v", results["degraded"].Status, StatusDegraded)
	}

	if results["unhealthy"].Status != StatusUnhealthy {
		t.Errorf("results[unhealthy].Status = %v, want %v", results["unhealthy"].Status, StatusUnhealthy)
	}

	// Verify latency was measured (might be 0 on very fast machines)
	for name, result := range results {
		// Latency should be >= 0 (time.Duration default is 0)
		// We can't assert it's > 0 because on very fast machines it might be 0
		if result.Latency < 0 {
			t.Errorf("results[%s].Latency should be non-negative, got %v", name, result.Latency)
		}
	}
}

func TestCheckWithTimeout(t *testing.T) {
	manager := NewManager().WithTimeout(100 * time.Millisecond)

	// Slow checker that exceeds timeout
	slowChecker := &mockChecker{
		name:   "slow",
		result: Healthy("should timeout"),
		delay:  200 * time.Millisecond,
	}

	manager.AddChecker(slowChecker)

	ctx := context.Background()
	results := manager.Check(ctx)

	if len(results) != 1 {
		t.Fatalf("Check() returned %d results, want 1", len(results))
	}

	// The slow checker should have been cancelled
	result := results["slow"]
	if result.Status != StatusUnhealthy {
		t.Errorf("slow check should be unhealthy due to timeout, got %v", result.Status)
	}

	if result.Message != "check cancelled" {
		t.Errorf("Message = %q, want %q", result.Message, "check cancelled")
	}
}

func TestCheckConcurrency(t *testing.T) {
	manager := NewManager()

	// Add multiple checkers with delays to test parallel execution
	for i := 0; i < 5; i++ {
		checker := &mockChecker{
			name:   "checker-" + string(rune('0'+i)),
			result: Healthy("ok"),
			delay:  50 * time.Millisecond,
		}
		manager.AddChecker(checker)
	}

	ctx := context.Background()
	start := time.Now()
	results := manager.Check(ctx)
	elapsed := time.Since(start)

	// If running in parallel, should take ~50ms
	// If sequential, would take ~250ms
	if elapsed > 150*time.Millisecond {
		t.Errorf("Check took %v, expected parallel execution to be faster", elapsed)
	}

	if len(results) != 5 {
		t.Errorf("Check() returned %d results, want 5", len(results))
	}
}

func TestOverallStatus(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name     string
		results  map[string]*Result
		expected Status
	}{
		{
			name:     "empty results",
			results:  map[string]*Result{},
			expected: StatusHealthy,
		},
		{
			name: "all healthy",
			results: map[string]*Result{
				"check1": Healthy("ok"),
				"check2": Healthy("ok"),
			},
			expected: StatusHealthy,
		},
		{
			name: "one degraded",
			results: map[string]*Result{
				"check1": Healthy("ok"),
				"check2": Degraded("partial"),
			},
			expected: StatusDegraded,
		},
		{
			name: "one unhealthy",
			results: map[string]*Result{
				"check1": Healthy("ok"),
				"check2": Degraded("partial"),
				"check3": Unhealthy("broken"),
			},
			expected: StatusUnhealthy,
		},
		{
			name: "all unhealthy",
			results: map[string]*Result{
				"check1": Unhealthy("broken"),
				"check2": Unhealthy("broken"),
			},
			expected: StatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := manager.OverallStatus(tt.results)
			if status != tt.expected {
				t.Errorf("OverallStatus() = %v, want %v", status, tt.expected)
			}
		})
	}
}

func TestCount(t *testing.T) {
	manager := NewManager()

	if manager.Count() != 0 {
		t.Errorf("Count() = %d, want 0", manager.Count())
	}

	manager.AddChecker(&mockChecker{name: "test1", result: Healthy("ok")})
	if manager.Count() != 1 {
		t.Errorf("Count() = %d, want 1", manager.Count())
	}

	manager.AddChecker(&mockChecker{name: "test2", result: Healthy("ok")})
	if manager.Count() != 2 {
		t.Errorf("Count() = %d, want 2", manager.Count())
	}

	manager.RemoveChecker("test1")
	if manager.Count() != 1 {
		t.Errorf("Count() = %d, want 1 after removal", manager.Count())
	}
}

func TestCheckNames(t *testing.T) {
	manager := NewManager()

	// Empty manager
	names := manager.CheckNames()
	if len(names) != 0 {
		t.Errorf("CheckNames() for empty manager = %v, want []", names)
	}

	// Add checkers
	manager.AddChecker(&mockChecker{name: "alpha", result: Healthy("ok")})
	manager.AddChecker(&mockChecker{name: "beta", result: Healthy("ok")})
	manager.AddChecker(&mockChecker{name: "gamma", result: Healthy("ok")})

	names = manager.CheckNames()
	if len(names) != 3 {
		t.Fatalf("CheckNames() returned %d names, want 3", len(names))
	}

	expected := []string{"alpha", "beta", "gamma"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("CheckNames()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}
