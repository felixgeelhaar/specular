package health

import (
	"testing"
	"time"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusHealthy, "healthy"},
		{StatusDegraded, "degraded"},
		{StatusUnhealthy, "unhealthy"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("Status.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewResult(t *testing.T) {
	result := NewResult(StatusHealthy, "test message")

	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusHealthy)
	}

	if result.Message != "test message" {
		t.Errorf("Message = %q, want %q", result.Message, "test message")
	}

	if result.Details == nil {
		t.Error("Details should be initialized, got nil")
	}

	if len(result.Details) != 0 {
		t.Errorf("Details should be empty, got %d items", len(result.Details))
	}
}

func TestWithDetail(t *testing.T) {
	result := NewResult(StatusHealthy, "test")

	// Test chaining
	returned := result.WithDetail("key1", "value1")
	if returned != result {
		t.Error("WithDetail should return same result for chaining")
	}

	// Test single detail
	result.WithDetail("foo", "bar")
	if val, ok := result.Details["foo"].(string); !ok || val != "bar" {
		t.Errorf("Details[foo] = %v, want %q", result.Details["foo"], "bar")
	}

	// Test multiple details
	result.WithDetail("count", 42).WithDetail("enabled", true)

	if val, ok := result.Details["count"].(int); !ok || val != 42 {
		t.Errorf("Details[count] = %v, want 42", result.Details["count"])
	}

	if val, ok := result.Details["enabled"].(bool); !ok || !val {
		t.Errorf("Details[enabled] = %v, want true", result.Details["enabled"])
	}
}

func TestWithLatency(t *testing.T) {
	result := NewResult(StatusHealthy, "test")
	latency := 123 * time.Millisecond

	// Test chaining
	returned := result.WithLatency(latency)
	if returned != result {
		t.Error("WithLatency should return same result for chaining")
	}

	if result.Latency != latency {
		t.Errorf("Latency = %v, want %v", result.Latency, latency)
	}
}

func TestHealthy(t *testing.T) {
	result := Healthy("all good")

	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusHealthy)
	}

	if result.Message != "all good" {
		t.Errorf("Message = %q, want %q", result.Message, "all good")
	}

	if result.Details == nil {
		t.Error("Details should be initialized")
	}
}

func TestDegraded(t *testing.T) {
	result := Degraded("partially working")

	if result.Status != StatusDegraded {
		t.Errorf("Status = %v, want %v", result.Status, StatusDegraded)
	}

	if result.Message != "partially working" {
		t.Errorf("Message = %q, want %q", result.Message, "partially working")
	}

	if result.Details == nil {
		t.Error("Details should be initialized")
	}
}

func TestUnhealthy(t *testing.T) {
	result := Unhealthy("broken")

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusUnhealthy)
	}

	if result.Message != "broken" {
		t.Errorf("Message = %q, want %q", result.Message, "broken")
	}

	if result.Details == nil {
		t.Error("Details should be initialized")
	}
}

func TestFluentAPI(t *testing.T) {
	// Test chaining all methods together
	result := Healthy("test").
		WithDetail("version", "1.0").
		WithDetail("uptime", 3600).
		WithLatency(50 * time.Millisecond)

	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusHealthy)
	}

	if result.Message != "test" {
		t.Errorf("Message = %q, want %q", result.Message, "test")
	}

	if result.Latency != 50*time.Millisecond {
		t.Errorf("Latency = %v, want %v", result.Latency, 50*time.Millisecond)
	}

	if val, ok := result.Details["version"].(string); !ok || val != "1.0" {
		t.Errorf("Details[version] = %v, want %q", result.Details["version"], "1.0")
	}

	if val, ok := result.Details["uptime"].(int); !ok || val != 3600 {
		t.Errorf("Details[uptime] = %v, want 3600", result.Details["uptime"])
	}
}
