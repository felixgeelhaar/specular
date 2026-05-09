package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Activation funnel step identifiers. Values are kept short and stable so they
// stay safe to pin in dashboards and alerts.
const (
	ActivationStepStarted        = "started"
	ActivationStepDetected       = "context_detected"
	ActivationStepConfigWritten  = "config_written"
	ActivationStepProvidersReady = "providers_configured"
	ActivationStepCompleted      = "completed"
	ActivationStepAbandoned      = "abandoned"
	ActivationStepFirstSuccess   = "first_command_success"
)

// Activation step status values.
const (
	ActivationStatusOK        = "ok"
	ActivationStatusSkipped   = "skipped"
	ActivationStatusFailed    = "failed"
	ActivationStatusAbandoned = "abandoned"
)

// Activation duration milestones.
const (
	ActivationMilestoneInitComplete = "init_complete"
	ActivationMilestoneFirstSuccess = "first_success"
)

// Intervention gate types capture the surface where a human approves or rejects
// AI-driven output.
const (
	InterventionGatePlanApproval   = "plan_approval"
	InterventionGateDriftApproval  = "drift_approval"
	InterventionGateBundleApproval = "bundle_approval"
	InterventionGatePolicyApproval = "policy_approval"
	InterventionGateOther          = "other"
)

// Intervention decisions.
const (
	InterventionDecisionApproved = "approved"
	InterventionDecisionRejected = "rejected"
)

// RecordActivationStep records a step in the activation funnel. The combination
// of step + status is what lets dashboards measure setup drop-off (e.g. count of
// step="config_written",status="ok" divided by step="started",status="ok").
func RecordActivationStep(ctx context.Context, step, status string, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.ActivationStepCounter == nil {
		return
	}

	base := []attribute.KeyValue{
		attribute.String("step", step),
		attribute.String("status", status),
	}
	base = append(base, attrs...)

	m.ActivationStepCounter.Add(ctx, 1, metric.WithAttributes(base...))
}

// RecordActivationDuration records elapsed time from the activation start to a
// milestone (init_complete, first_success). Time-to-first-success is computed
// from milestone="first_success" samples.
func RecordActivationDuration(ctx context.Context, milestone string, duration time.Duration, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.ActivationDuration == nil {
		return
	}

	base := []attribute.KeyValue{
		attribute.String("milestone", milestone),
	}
	base = append(base, attrs...)

	m.ActivationDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(base...))
}

// RecordRoutingDecision records a router model selection together with the
// human-readable reason that drove it. The reason and cost_band attributes are
// the explainability signals used to audit AI trust over time.
func RecordRoutingDecision(ctx context.Context, providerName, model, hint, reason string, estimatedCostUSD float64, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.RoutingDecisionCounter == nil {
		return
	}

	if hint == "" {
		hint = "none"
	}

	base := []attribute.KeyValue{
		attribute.String("provider", providerName),
		attribute.String("model", model),
		attribute.String("hint", hint),
		attribute.String("reason", reason),
		attribute.String("cost_band", CostBand(estimatedCostUSD)),
	}
	base = append(base, attrs...)

	m.RoutingDecisionCounter.Add(ctx, 1, metric.WithAttributes(base...))

	if m.RoutingCostEstimate != nil {
		m.RoutingCostEstimate.Record(ctx, estimatedCostUSD, metric.WithAttributes(
			attribute.String("provider", providerName),
			attribute.String("model", model),
		))
	}
}

// RecordIntervention records a human-in-the-loop intervention event so the
// intervention rate can be tracked per gate type.
func RecordIntervention(ctx context.Context, gate, decision string, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.InterventionCounter == nil {
		return
	}

	base := []attribute.KeyValue{
		attribute.String("gate", gate),
		attribute.String("decision", decision),
	}
	base = append(base, attrs...)

	m.InterventionCounter.Add(ctx, 1, metric.WithAttributes(base...))
}

// RecordRegenerate records a user-initiated regeneration of AI output. The
// reason field captures why the output was rejected (e.g. "user_request",
// "validation_failed").
func RecordRegenerate(ctx context.Context, command, reason string, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.RegenerateCounter == nil {
		return
	}

	if reason == "" {
		reason = "unspecified"
	}

	base := []attribute.KeyValue{
		attribute.String("command", command),
		attribute.String("reason", reason),
	}
	base = append(base, attrs...)

	m.RegenerateCounter.Add(ctx, 1, metric.WithAttributes(base...))
}

// CostBand bins estimated cost into a small fixed set of labels. Histogram
// recording captures exact values; the band keeps counter cardinality low for
// dashboards filtering by price tier.
func CostBand(usd float64) string {
	switch {
	case usd <= 0:
		return "free"
	case usd < 0.01:
		return "<0.01"
	case usd < 0.10:
		return "0.01-0.10"
	case usd < 1.00:
		return "0.10-1.00"
	case usd < 10.00:
		return "1.00-10.00"
	default:
		return ">=10.00"
	}
}
