package telemetry

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestCostBand(t *testing.T) {
	cases := []struct {
		name string
		usd  float64
		want string
	}{
		{"free zero", 0, "free"},
		{"free negative", -0.05, "free"},
		{"sub cent", 0.005, "<0.01"},
		{"single cent", 0.05, "0.01-0.10"},
		{"sub dollar", 0.5, "0.10-1.00"},
		{"single dollar", 5, "1.00-10.00"},
		{"high", 25, ">=10.00"},
		{"exact ten", 10, ">=10.00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CostBand(tc.usd)
			if got != tc.want {
				t.Fatalf("CostBand(%v) = %q, want %q", tc.usd, got, tc.want)
			}
		})
	}
}

func TestRecordActivationStep(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() { _ = mp.Shutdown(context.Background()) }()

	ctx := context.Background()
	RecordActivationStep(ctx, ActivationStepStarted, ActivationStatusOK,
		attribute.String("template", "web-app"),
	)
	RecordActivationStep(ctx, ActivationStepAbandoned, ActivationStatusAbandoned,
		attribute.String("template", "web-app"),
	)

	sum := findSum(t, reader, "specular.activation.step")
	if len(sum.DataPoints) != 2 {
		t.Fatalf("expected 2 data points, got %d", len(sum.DataPoints))
	}
	for _, dp := range sum.DataPoints {
		if dp.Value != 1 {
			t.Errorf("expected value 1, got %d", dp.Value)
		}
		if !hasAttribute(dp.Attributes.ToSlice(), "step") || !hasAttribute(dp.Attributes.ToSlice(), "status") {
			t.Errorf("missing required attributes on %v", dp.Attributes.ToSlice())
		}
	}
}

func TestRecordActivationDuration(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() { _ = mp.Shutdown(context.Background()) }()

	RecordActivationDuration(context.Background(), ActivationMilestoneInitComplete, 12500*time.Millisecond,
		attribute.String("template", "cli-tool"),
	)

	hist := findHistogram(t, reader, "specular.activation.duration")
	if len(hist.DataPoints) != 1 {
		t.Fatalf("expected 1 datapoint, got %d", len(hist.DataPoints))
	}
	dp := hist.DataPoints[0]
	if dp.Sum < 12.0 || dp.Sum > 13.0 {
		t.Errorf("expected sum near 12.5s, got %f", dp.Sum)
	}
	if !hasAttributeValue(dp.Attributes.ToSlice(), "milestone", ActivationMilestoneInitComplete) {
		t.Errorf("missing milestone attribute on %v", dp.Attributes.ToSlice())
	}
}

func TestRecordRoutingDecision(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() { _ = mp.Shutdown(context.Background()) }()

	RecordRoutingDecision(context.Background(),
		"anthropic", "claude-3-5-sonnet", "codegen",
		"Selected claude-3-5-sonnet (anthropic): matched hint: codegen, budget-optimized",
		0.42,
	)

	sum := findSum(t, reader, "specular.ai_trust.routing_decision")
	if len(sum.DataPoints) != 1 {
		t.Fatalf("expected 1 datapoint, got %d", len(sum.DataPoints))
	}
	attrs := sum.DataPoints[0].Attributes.ToSlice()
	for _, key := range []string{"provider", "model", "hint", "reason", "cost_band"} {
		if !hasAttribute(attrs, key) {
			t.Errorf("missing attribute %q on routing decision: %v", key, attrs)
		}
	}
	if !hasAttributeValue(attrs, "cost_band", "0.10-1.00") {
		t.Errorf("expected cost_band 0.10-1.00, got %v", attrs)
	}

	hist := findHistogram(t, reader, "specular.ai_trust.routing_cost_estimate")
	if len(hist.DataPoints) != 1 {
		t.Fatalf("expected 1 cost estimate datapoint, got %d", len(hist.DataPoints))
	}
	if hist.DataPoints[0].Sum < 0.41 || hist.DataPoints[0].Sum > 0.43 {
		t.Errorf("expected cost estimate near 0.42, got %f", hist.DataPoints[0].Sum)
	}
}

func TestRecordRoutingDecisionEmptyHintNormalises(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() { _ = mp.Shutdown(context.Background()) }()

	RecordRoutingDecision(context.Background(), "openai", "gpt-4o", "", "best overall capability", 0)

	sum := findSum(t, reader, "specular.ai_trust.routing_decision")
	if len(sum.DataPoints) != 1 {
		t.Fatalf("expected 1 datapoint, got %d", len(sum.DataPoints))
	}
	attrs := sum.DataPoints[0].Attributes.ToSlice()
	if !hasAttributeValue(attrs, "hint", "none") {
		t.Errorf("expected hint=none for empty hint, got %v", attrs)
	}
	if !hasAttributeValue(attrs, "cost_band", "free") {
		t.Errorf("expected cost_band=free for $0 estimate, got %v", attrs)
	}
}

func TestRecordIntervention(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() { _ = mp.Shutdown(context.Background()) }()

	ctx := context.Background()
	RecordIntervention(ctx, InterventionGatePlanApproval, InterventionDecisionApproved)
	RecordIntervention(ctx, InterventionGatePlanApproval, InterventionDecisionRejected)
	RecordIntervention(ctx, InterventionGateDriftApproval, InterventionDecisionApproved)

	sum := findSum(t, reader, "specular.ai_trust.intervention")
	if len(sum.DataPoints) != 3 {
		t.Fatalf("expected 3 datapoints, got %d", len(sum.DataPoints))
	}
}

func TestRecordRegenerate(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() { _ = mp.Shutdown(context.Background()) }()

	RecordRegenerate(context.Background(), "plan", "validation_failed")
	RecordRegenerate(context.Background(), "plan", "")

	sum := findSum(t, reader, "specular.ai_trust.regenerate")
	if len(sum.DataPoints) != 2 {
		t.Fatalf("expected 2 datapoints, got %d", len(sum.DataPoints))
	}
	for _, dp := range sum.DataPoints {
		attrs := dp.Attributes.ToSlice()
		if !hasAttribute(attrs, "command") || !hasAttribute(attrs, "reason") {
			t.Errorf("missing required attributes on %v", attrs)
		}
	}
}

func TestActivationRecordersAreNoopWhenUninitialised(t *testing.T) {
	resetMetricsForTest()
	defer resetMetricsForTest()

	// All recorders must be safe to call before metrics are initialised so
	// that telemetry never blocks the user-facing CLI flow.
	ctx := context.Background()
	RecordActivationStep(ctx, ActivationStepStarted, ActivationStatusOK)
	RecordActivationDuration(ctx, ActivationMilestoneInitComplete, time.Second)
	RecordRoutingDecision(ctx, "openai", "gpt-4o", "fast", "test", 0.1)
	RecordIntervention(ctx, InterventionGatePlanApproval, InterventionDecisionApproved)
	RecordRegenerate(ctx, "plan", "user_request")
}

func findSum(t *testing.T, reader interface {
	Collect(context.Context, *metricdata.ResourceMetrics) error
}, name string) metricdata.Sum[int64] {
	t.Helper()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("metric %q is not Sum[int64]: %T", name, m.Data)
			}
			return sum
		}
	}
	t.Fatalf("metric %q not found", name)
	return metricdata.Sum[int64]{}
}

func findHistogram(t *testing.T, reader interface {
	Collect(context.Context, *metricdata.ResourceMetrics) error
}, name string) metricdata.Histogram[float64] {
	t.Helper()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			hist, ok := m.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("metric %q is not Histogram[float64]: %T", name, m.Data)
			}
			return hist
		}
	}
	t.Fatalf("metric %q not found", name)
	return metricdata.Histogram[float64]{}
}

func hasAttribute(attrs []attribute.KeyValue, key string) bool {
	for _, kv := range attrs {
		if string(kv.Key) == key {
			return true
		}
	}
	return false
}

func hasAttributeValue(attrs []attribute.KeyValue, key, value string) bool {
	for _, kv := range attrs {
		if string(kv.Key) == key && kv.Value.AsString() == value {
			return true
		}
	}
	return false
}

func resetMetricsForTest() {
	meterMu.Lock()
	defer meterMu.Unlock()
	metrics = nil
}
