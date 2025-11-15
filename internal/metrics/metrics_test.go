package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	if m == nil {
		t.Fatal("expected metrics, got nil")
	}

	// Verify all metrics are initialized
	tests := []struct {
		name   string
		metric interface{}
	}{
		{"CommandExecutions", m.CommandExecutions},
		{"CommandDuration", m.CommandDuration},
		{"CommandErrors", m.CommandErrors},
		{"ProviderCalls", m.ProviderCalls},
		{"ProviderLatency", m.ProviderLatency},
		{"ProviderErrors", m.ProviderErrors},
		{"ProviderCost", m.ProviderCost},
		{"SpecGenerations", m.SpecGenerations},
		{"SpecDuration", m.SpecDuration},
		{"SpecErrors", m.SpecErrors},
		{"PlanGenerations", m.PlanGenerations},
		{"PlanDuration", m.PlanDuration},
		{"PlanFeatureCount", m.PlanFeatureCount},
		{"PlanTaskCount", m.PlanTaskCount},
		{"PlanErrors", m.PlanErrors},
		{"TaskExecutions", m.TaskExecutions},
		{"TaskDuration", m.TaskDuration},
		{"TaskErrors", m.TaskErrors},
		{"ImagePulls", m.ImagePulls},
		{"ImagePullDuration", m.ImagePullDuration},
		{"ImagePullErrors", m.ImagePullErrors},
		{"CacheHits", m.CacheHits},
		{"CacheMisses", m.CacheMisses},
		{"PolicyChecks", m.PolicyChecks},
		{"PolicyViolations", m.PolicyViolations},
		{"PolicyDuration", m.PolicyDuration},
		{"DriftDetections", m.DriftDetections},
		{"DriftFound", m.DriftFound},
		{"AutoWorkflows", m.AutoWorkflows},
		{"AutoSteps", m.AutoSteps},
		{"AutoStepDuration", m.AutoStepDuration},
		{"AutoApprovals", m.AutoApprovals},
		{"AutoApprovalLatency", m.AutoApprovalLatency},
		{"InterviewSessions", m.InterviewSessions},
		{"InterviewQuestions", m.InterviewQuestions},
		{"InterviewDuration", m.InterviewDuration},
		{"Errors", m.Errors},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metric == nil {
				t.Errorf("%s metric is nil", tt.name)
			}
		})
	}
}

func TestCommandMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record successful command
	m.CommandExecutions.WithLabelValues("spec", "true").Inc()
	m.CommandDuration.WithLabelValues("spec").Observe(1.5)

	// Record failed command
	m.CommandExecutions.WithLabelValues("plan", "false").Inc()
	m.CommandErrors.WithLabelValues("plan", "SPEC-001").Inc()

	// Verify metrics
	if got := testutil.ToFloat64(m.CommandExecutions.WithLabelValues("spec", "true")); got != 1 {
		t.Errorf("CommandExecutions spec/true = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.CommandExecutions.WithLabelValues("plan", "false")); got != 1 {
		t.Errorf("CommandExecutions plan/false = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.CommandErrors.WithLabelValues("plan", "SPEC-001")); got != 1 {
		t.Errorf("CommandErrors = %v, want 1", got)
	}
}

func TestProviderMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record provider call
	m.ProviderCalls.WithLabelValues("claude", "sonnet", "true").Inc()
	m.ProviderLatency.WithLabelValues("claude", "sonnet").Observe(2.5)
	m.ProviderCost.WithLabelValues("claude", "sonnet", "input").Add(1000)
	m.ProviderCost.WithLabelValues("claude", "sonnet", "output").Add(500)

	// Record provider error
	m.ProviderErrors.WithLabelValues("claude", "sonnet", "rate_limit").Inc()

	// Verify metrics
	if got := testutil.ToFloat64(m.ProviderCalls.WithLabelValues("claude", "sonnet", "true")); got != 1 {
		t.Errorf("ProviderCalls = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.ProviderCost.WithLabelValues("claude", "sonnet", "input")); got != 1000 {
		t.Errorf("ProviderCost input = %v, want 1000", got)
	}

	if got := testutil.ToFloat64(m.ProviderCost.WithLabelValues("claude", "sonnet", "output")); got != 500 {
		t.Errorf("ProviderCost output = %v, want 500", got)
	}

	if got := testutil.ToFloat64(m.ProviderErrors.WithLabelValues("claude", "sonnet", "rate_limit")); got != 1 {
		t.Errorf("ProviderErrors = %v, want 1", got)
	}
}

func TestSpecAndPlanMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record spec generation
	m.SpecGenerations.WithLabelValues("true").Inc()
	m.SpecDuration.WithLabelValues().Observe(5.0)

	// Record plan generation
	m.PlanGenerations.WithLabelValues("true").Inc()
	m.PlanDuration.WithLabelValues().Observe(10.0)
	m.PlanFeatureCount.WithLabelValues().Observe(5)
	m.PlanTaskCount.WithLabelValues().Observe(15)

	// Verify metrics
	if got := testutil.ToFloat64(m.SpecGenerations.WithLabelValues("true")); got != 1 {
		t.Errorf("SpecGenerations = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.PlanGenerations.WithLabelValues("true")); got != 1 {
		t.Errorf("PlanGenerations = %v, want 1", got)
	}
}

func TestDockerMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	image := "specular/nodejs:20"

	// Record cache hit
	m.CacheHits.WithLabelValues(image).Inc()

	// Record image pull
	m.ImagePulls.WithLabelValues(image, "true").Inc()
	m.ImagePullDuration.WithLabelValues(image).Observe(30.0)

	// Record cache miss
	m.CacheMisses.WithLabelValues(image).Inc()

	// Verify metrics
	if got := testutil.ToFloat64(m.CacheHits.WithLabelValues(image)); got != 1 {
		t.Errorf("CacheHits = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.CacheMisses.WithLabelValues(image)); got != 1 {
		t.Errorf("CacheMisses = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.ImagePulls.WithLabelValues(image, "true")); got != 1 {
		t.Errorf("ImagePulls = %v, want 1", got)
	}
}

func TestPolicyMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record policy check
	m.PolicyChecks.WithLabelValues("docker_image", "pass").Inc()
	m.PolicyDuration.WithLabelValues("docker_image").Observe(0.1)

	// Record policy violation
	m.PolicyViolations.WithLabelValues("tool_allowlist", "warning").Inc()

	// Verify metrics
	if got := testutil.ToFloat64(m.PolicyChecks.WithLabelValues("docker_image", "pass")); got != 1 {
		t.Errorf("PolicyChecks = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.PolicyViolations.WithLabelValues("tool_allowlist", "warning")); got != 1 {
		t.Errorf("PolicyViolations = %v, want 1", got)
	}
}

func TestDriftMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record drift detection
	m.DriftDetections.WithLabelValues("plan_spec").Inc()
	m.DriftFound.WithLabelValues("plan_spec").Inc()

	// Verify metrics
	if got := testutil.ToFloat64(m.DriftDetections.WithLabelValues("plan_spec")); got != 1 {
		t.Errorf("DriftDetections = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.DriftFound.WithLabelValues("plan_spec")); got != 1 {
		t.Errorf("DriftFound = %v, want 1", got)
	}
}

func TestAutoMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record autonomous workflow
	m.AutoWorkflows.WithLabelValues("true").Inc()

	// Record step
	m.AutoSteps.WithLabelValues("generate_spec", "true").Inc()
	m.AutoStepDuration.WithLabelValues("generate_spec").Observe(5.0)

	// Record approval
	m.AutoApprovals.WithLabelValues("true").Inc()
	m.AutoApprovalLatency.WithLabelValues().Observe(10.0)

	// Verify metrics
	if got := testutil.ToFloat64(m.AutoWorkflows.WithLabelValues("true")); got != 1 {
		t.Errorf("AutoWorkflows = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.AutoSteps.WithLabelValues("generate_spec", "true")); got != 1 {
		t.Errorf("AutoSteps = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.AutoApprovals.WithLabelValues("true")); got != 1 {
		t.Errorf("AutoApprovals = %v, want 1", got)
	}
}

func TestInterviewMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record interview session
	m.InterviewSessions.WithLabelValues("feature", "true").Inc()
	m.InterviewQuestions.WithLabelValues("feature").Add(5)
	m.InterviewDuration.WithLabelValues("feature").Observe(120.0)

	// Verify metrics
	if got := testutil.ToFloat64(m.InterviewSessions.WithLabelValues("feature", "true")); got != 1 {
		t.Errorf("InterviewSessions = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.InterviewQuestions.WithLabelValues("feature")); got != 5 {
		t.Errorf("InterviewQuestions = %v, want 5", got)
	}
}

func TestErrorMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record errors by code
	m.Errors.WithLabelValues("SPEC-001", "spec_generator").Inc()
	m.Errors.WithLabelValues("DOM-002", "domain").Inc()
	m.Errors.WithLabelValues("EXEC-001", "docker").Inc()

	// Verify metrics
	if got := testutil.ToFloat64(m.Errors.WithLabelValues("SPEC-001", "spec_generator")); got != 1 {
		t.Errorf("Errors SPEC-001 = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.Errors.WithLabelValues("DOM-002", "domain")); got != 1 {
		t.Errorf("Errors DOM-002 = %v, want 1", got)
	}

	if got := testutil.ToFloat64(m.Errors.WithLabelValues("EXEC-001", "docker")); got != 1 {
		t.Errorf("Errors EXEC-001 = %v, want 1", got)
	}
}

func TestMetricsExport(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record some metrics
	m.CommandExecutions.WithLabelValues("spec", "true").Inc()
	m.ProviderCalls.WithLabelValues("claude", "sonnet", "true").Inc()

	// Create HTTP handler
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	// Make request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// Verify metrics are present
	if !strings.Contains(body, "specular_command_executions_total") {
		t.Error("metrics output does not contain command_executions_total")
	}

	if !strings.Contains(body, "specular_provider_calls_total") {
		t.Error("metrics output does not contain provider_calls_total")
	}

	// Verify labels
	if !strings.Contains(body, `command="spec"`) {
		t.Error("metrics output does not contain command label")
	}

	if !strings.Contains(body, `provider="claude"`) {
		t.Error("metrics output does not contain provider label")
	}
}

func TestHistogramBuckets(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record various durations
	m.CommandDuration.WithLabelValues("spec").Observe(0.5)
	m.ProviderLatency.WithLabelValues("claude", "sonnet").Observe(2.5)
	m.ImagePullDuration.WithLabelValues("specular/nodejs:20").Observe(30.0)

	// Make request to get metrics
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify histogram buckets exist
	if !strings.Contains(body, "_bucket{") {
		t.Error("metrics output does not contain histogram buckets")
	}

	if !strings.Contains(body, "le=") {
		t.Error("metrics output does not contain bucket labels")
	}
}
