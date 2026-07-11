package telemetry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"pgregory.net/rapid"
)

// costBandIndex assigns a deterministic order to bands so a property test
// can assert monotonicity without depending on string comparison.
func costBandIndex(band string) int {
	order := []string{"free", "<0.01", "0.01-0.10", "0.10-1.00", "1.00-10.00", "10-100", ">=100"}
	for i, b := range order {
		if b == band {
			return i
		}
	}
	return -1
}

func TestCostBandMonotonic(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		usd1 := rapid.Float64Range(-1000, 100000).Draw(rt, "usd1")
		usd2 := rapid.Float64Range(-1000, 100000).Draw(rt, "usd2")
		if usd1 > usd2 {
			usd1, usd2 = usd2, usd1
		}
		i1 := costBandIndex(CostBand(usd1))
		i2 := costBandIndex(CostBand(usd2))
		if i1 < 0 || i2 < 0 {
			rt.Fatalf("CostBand returned unknown band for usd1=%v usd2=%v", usd1, usd2)
		}
		if i1 > i2 {
			rt.Fatalf("CostBand not monotonic: usd1=%v -> %s (idx %d) vs usd2=%v -> %s (idx %d)",
				usd1, CostBand(usd1), i1, usd2, CostBand(usd2), i2)
		}
	})
}

func TestCostBandBoundaries(t *testing.T) {
	// Pin exact boundary behaviour so a future refactor cannot silently
	// shift the cutoffs that downstream dashboards depend on.
	cases := []struct {
		usd  float64
		want string
	}{
		{0.01, "0.01-0.10"},
		{0.0099999, "<0.01"},
		{0.10, "0.10-1.00"},
		{0.099999, "0.01-0.10"},
		{1.00, "1.00-10.00"},
		{0.999999, "0.10-1.00"},
		{10.00, "10-100"},
		{9.999999, "1.00-10.00"},
		{99.99, "10-100"},
		{100.00, ">=100"},
		{2500.0, ">=100"},
	}
	for _, tc := range cases {
		if got := CostBand(tc.usd); got != tc.want {
			t.Errorf("CostBand(%v) = %q, want %q", tc.usd, got, tc.want)
		}
	}
}

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
		{"frontier mid", 25, "10-100"},
		{"exact ten", 10, "10-100"},
		{"frontier upper", 250, ">=100"},
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
	// Counter must carry only low-cardinality attributes. Reason is unbounded
	// and lives on the span event, never on the metric.
	for _, key := range []string{"provider", "model", "hint", "cost_band"} {
		if !hasAttribute(attrs, key) {
			t.Errorf("missing attribute %q on routing decision: %v", key, attrs)
		}
	}
	if hasAttribute(attrs, "reason") {
		t.Errorf("reason must not appear on the routing-decision counter (high cardinality): %v", attrs)
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

func TestRecordRoutingDecisionNormalisesArbitraryHint(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() { _ = mp.Shutdown(context.Background()) }()

	// Arbitrary user-supplied hint must collapse onto the bounded enum to
	// avoid blowing up downstream metric cardinality.
	RecordRoutingDecision(context.Background(),
		"openai", "gpt-4o", "completely-made-up-string",
		"reason text",
		0.05,
	)

	sum := findSum(t, reader, "specular.ai_trust.routing_decision")
	if len(sum.DataPoints) != 1 {
		t.Fatalf("expected 1 datapoint, got %d", len(sum.DataPoints))
	}
	if !hasAttributeValue(sum.DataPoints[0].Attributes.ToSlice(), "hint", "other") {
		t.Errorf("expected unknown hint to normalise to 'other', got %v", sum.DataPoints[0].Attributes.ToSlice())
	}
}

func TestRoutingDecisionCardinalityBounded(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() { _ = mp.Shutdown(context.Background()) }()

	// Drive 1k distinct free-form hint strings through the recorder and
	// confirm the resulting metric stays bounded by the hint enum
	// (codegen / fast / cheap / agentic / long-context / other / none plus
	// provider+model+cost_band combinations are bounded). This is the
	// fitness function that should fail loudly if reason or hint ever
	// regresses to free-form on the counter.
	for i := 0; i < 1000; i++ {
		RecordRoutingDecision(context.Background(),
			"anthropic", "claude-3-5-sonnet",
			fmt.Sprintf("user-hint-%d", i),
			fmt.Sprintf("reason-%d", i),
			0.05,
		)
	}

	sum := findSum(t, reader, "specular.ai_trust.routing_decision")
	// All 1000 calls share provider, model, cost_band, and the hint
	// normalises to "other", so we expect exactly one data-point series.
	if len(sum.DataPoints) > 4 {
		t.Errorf("routing decision cardinality unbounded: %d distinct series after 1000 calls", len(sum.DataPoints))
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

	RecordRegenerate(context.Background(), "plan", RegenerateTriggerEvalFailure, "claude-haiku-4-5")
	RecordRegenerate(context.Background(), "plan", RegenerateTriggerUserReject, "")

	sum := findSum(t, reader, "specular.ai_trust.regenerate")
	if len(sum.DataPoints) != 2 {
		t.Fatalf("expected 2 datapoints, got %d", len(sum.DataPoints))
	}
	for _, dp := range sum.DataPoints {
		attrs := dp.Attributes.ToSlice()
		for _, key := range []string{"command", "trigger", "previous_model"} {
			if !hasAttribute(attrs, key) {
				t.Errorf("missing required attribute %q on regenerate: %v", key, attrs)
			}
		}
	}
}

func TestRecordSafetyEvent(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() { _ = mp.Shutdown(context.Background()) }()

	RecordSafetyEvent(context.Background(),
		SafetyCategorySecretLeak, SafetySeverityHigh, SafetyActionBlocked,
		attribute.String("hook", "pre-build-secrets-scan"),
	)

	sum := findSum(t, reader, "specular.ai_trust.safety_event")
	if len(sum.DataPoints) != 1 {
		t.Fatalf("expected 1 datapoint, got %d", len(sum.DataPoints))
	}
	attrs := sum.DataPoints[0].Attributes.ToSlice()
	for _, key := range []string{"category", "severity", "action_taken"} {
		if !hasAttribute(attrs, key) {
			t.Errorf("missing required attribute %q on safety_event: %v", key, attrs)
		}
	}
}

// TestActivationMetricContract pins the attribute schema for every
// activation / AI-trust metric. Downstream dashboards and alert rules
// depend on these key + value sets, so any drift must surface as a test
// failure rather than a silent dashboard outage.
func TestActivationMetricContract(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() { _ = mp.Shutdown(context.Background()) }()

	ctx := context.Background()

	// Drive every recorder once so we can inspect the resulting attribute
	// schema against a written contract.
	RecordActivationStep(ctx, ActivationStepStarted, ActivationStatusOK)
	RecordActivationDuration(ctx, ActivationMilestoneInitComplete, time.Second)
	RecordRoutingDecision(ctx, "anthropic", "claude-3-5-sonnet", "codegen", "reason text", 0.5)
	RecordIntervention(ctx, InterventionGatePlanApproval, InterventionDecisionApproved)
	RecordRegenerate(ctx, "plan", RegenerateTriggerUserReject, "claude-haiku-4-5")
	RecordSafetyEvent(ctx, SafetyCategorySecretLeak, SafetySeverityHigh, SafetyActionBlocked)

	type contract struct {
		requiredKeys  map[string]struct{}
		forbiddenKeys map[string]struct{}
		enumValues    map[string]map[string]struct{} // key -> allowed values; empty map = any value allowed
	}

	contracts := map[string]contract{
		"specular.activation.step": {
			requiredKeys: keyset("step", "status"),
			enumValues: map[string]map[string]struct{}{
				"status": stringSet(ActivationStatusOK, ActivationStatusSkipped, ActivationStatusFailed, ActivationStatusAbandoned),
			},
		},
		"specular.activation.duration": {
			requiredKeys: keyset("milestone"),
			enumValues: map[string]map[string]struct{}{
				"milestone": stringSet(ActivationMilestoneInitComplete, ActivationMilestoneFirstSuccess),
			},
		},
		"specular.ai_trust.routing_decision": {
			requiredKeys:  keyset("provider", "model", "hint", "cost_band"),
			forbiddenKeys: keyset("reason"),
		},
		"specular.ai_trust.intervention": {
			requiredKeys: keyset("gate", "decision"),
			enumValues: map[string]map[string]struct{}{
				"decision": stringSet(InterventionDecisionApproved, InterventionDecisionRejected, InterventionDecisionImplicitReject),
			},
		},
		"specular.ai_trust.regenerate": {
			requiredKeys: keyset("command", "trigger", "previous_model"),
			enumValues: map[string]map[string]struct{}{
				"trigger": stringSet(
					RegenerateTriggerUserReject,
					RegenerateTriggerEvalFailure,
					RegenerateTriggerAgentSelfCorrect,
					RegenerateTriggerDriftRevert,
					RegenerateTriggerPolicyBlock,
					"unspecified",
				),
			},
		},
		"specular.ai_trust.safety_event": {
			requiredKeys: keyset("category", "severity", "action_taken"),
			enumValues: map[string]map[string]struct{}{
				"category": stringSet(
					SafetyCategoryPromptInjection,
					SafetyCategorySecretLeak,
					SafetyCategoryForbiddenToolCall,
					SafetyCategoryScopeViolation,
					SafetyCategoryRefusal,
					SafetyCategoryJailbreakAttempt,
				),
				"severity":     stringSet(SafetySeverityLow, SafetySeverityMedium, SafetySeverityHigh, SafetySeverityCritical),
				"action_taken": stringSet(SafetyActionAllowed, SafetyActionWarned, SafetyActionBlocked, SafetyActionEscalated),
			},
		},
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}

	checked := map[string]bool{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			c, ok := contracts[m.Name]
			if !ok {
				continue
			}
			checked[m.Name] = true
			for _, dp := range datapointAttributes(m) {
				attrs := dp.ToSlice()
				for k := range c.requiredKeys {
					if !hasAttribute(attrs, k) {
						t.Errorf("%s missing required attribute %q (%v)", m.Name, k, attrs)
					}
				}
				for k := range c.forbiddenKeys {
					if hasAttribute(attrs, k) {
						t.Errorf("%s contains forbidden attribute %q (%v)", m.Name, k, attrs)
					}
				}
				for k, allowed := range c.enumValues {
					for _, kv := range attrs {
						if string(kv.Key) != k {
							continue
						}
						if _, ok := allowed[kv.Value.AsString()]; !ok {
							t.Errorf("%s attribute %q value %q outside allowed enum %v",
								m.Name, k, kv.Value.AsString(), keys(allowed))
						}
					}
				}
			}
		}
	}

	for name := range contracts {
		if !checked[name] {
			t.Errorf("contract metric %q was not emitted", name)
		}
	}
}

func datapointAttributes(m metricdata.Metrics) []attribute.Set {
	switch d := m.Data.(type) {
	case metricdata.Sum[int64]:
		out := make([]attribute.Set, 0, len(d.DataPoints))
		for _, dp := range d.DataPoints {
			out = append(out, dp.Attributes)
		}
		return out
	case metricdata.Histogram[float64]:
		out := make([]attribute.Set, 0, len(d.DataPoints))
		for _, dp := range d.DataPoints {
			out = append(out, dp.Attributes)
		}
		return out
	}
	return nil
}

func keyset(items ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, k := range items {
		out[k] = struct{}{}
	}
	return out
}

func stringSet(items ...string) map[string]struct{} {
	return keyset(items...)
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
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
	RecordRegenerate(ctx, "plan", RegenerateTriggerUserReject, "")
	RecordSafetyEvent(ctx, SafetyCategorySecretLeak, SafetySeverityHigh, SafetyActionBlocked)
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
	resetMetricsForRetry()
}
