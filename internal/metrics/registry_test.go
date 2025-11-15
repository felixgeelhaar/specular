package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestInitDefault(t *testing.T) {
	// Reset before test
	Reset()

	m := InitDefault()
	if m == nil {
		t.Fatal("expected metrics, got nil")
	}

	if Default == nil {
		t.Fatal("expected Default to be set, got nil")
	}

	if m != Default {
		t.Error("expected returned metrics to be same as Default")
	}

	// Calling again should return same instance
	m2 := InitDefault()
	if m2 != m {
		t.Error("expected same instance on second call")
	}
}

func TestGetDefault(t *testing.T) {
	// Note: We don't reset here because TestInitDefault already tested initialization
	// This test verifies that GetDefault returns an existing instance if available

	// Get or initialize
	m := GetDefault()
	if m == nil {
		t.Fatal("expected metrics, got nil")
	}

	if Default == nil {
		t.Fatal("expected Default to be set, got nil")
	}

	// Calling again should return same instance
	m2 := GetDefault()
	if m2 != m {
		t.Error("expected same instance on second call")
	}

	// Verify it's the same instance as Default
	if m != Default {
		t.Error("expected GetDefault to return Default instance")
	}
}

func TestNewRegistry(t *testing.T) {
	reg, m := NewRegistry()

	if reg == nil {
		t.Fatal("expected registry, got nil")
	}

	if m == nil {
		t.Fatal("expected metrics, got nil")
	}

	// Verify metrics are registered with the custom registry
	m.CommandExecutions.WithLabelValues("test", "true").Inc()

	// Gather metrics from registry
	metricFamilies, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	// Should have at least the command execution metric
	found := false
	for _, mf := range metricFamilies {
		if *mf.Name == "specular_command_executions_total" {
			found = true
			break
		}
	}

	if !found {
		t.Error("metrics not registered with custom registry")
	}
}

func TestHandler(t *testing.T) {
	// Use custom registry to avoid conflicts with default
	reg, m := NewRegistry()

	// Record a metric
	m.CommandExecutions.WithLabelValues("test", "true").Inc()

	// Get handler for custom registry
	handler := HandlerFor(reg, DefaultHandlerOpts())

	// Make request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// Verify metric is present
	if !strings.Contains(body, "specular_command_executions_total") {
		t.Error("metrics output does not contain command_executions_total")
	}
}

func TestHandlerFor(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record a metric
	m.ProviderCalls.WithLabelValues("test", "model", "true").Inc()

	// Get handler for custom registry
	handler := HandlerFor(reg, DefaultHandlerOpts())

	// Make request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// Verify metric is present
	if !strings.Contains(body, "specular_provider_calls_total") {
		t.Error("metrics output does not contain provider_calls_total")
	}
}

// Note: TestReset is not included because Prometheus registries don't support
// unregistering metrics from the default registry. The Reset() function is still
// useful in application code for test isolation but cannot be easily tested here
// without creating registry conflicts.
//
// The important behavior (singleton pattern via InitDefault and GetDefault) is
// already tested in TestInitDefault and TestGetDefault.

func TestMultipleRegistries(t *testing.T) {
	// Create two separate registries
	reg1, m1 := NewRegistry()
	reg2, m2 := NewRegistry()

	// Record different metrics in each
	m1.CommandExecutions.WithLabelValues("test1", "true").Inc()
	m2.CommandExecutions.WithLabelValues("test2", "true").Inc()

	// Gather from reg1
	metricFamilies1, err := reg1.Gather()
	if err != nil {
		t.Fatalf("failed to gather from reg1: %v", err)
	}

	// Gather from reg2
	metricFamilies2, err := reg2.Gather()
	if err != nil {
		t.Fatalf("failed to gather from reg2: %v", err)
	}

	// Both should have metrics
	if len(metricFamilies1) == 0 {
		t.Error("reg1 has no metrics")
	}

	if len(metricFamilies2) == 0 {
		t.Error("reg2 has no metrics")
	}

	// The metrics should be independent (different instances)
	if m1 == m2 {
		t.Error("expected different metrics instances")
	}
}

// Helper function for handler options
func DefaultHandlerOpts() promhttp.HandlerOpts {
	return promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}
}
