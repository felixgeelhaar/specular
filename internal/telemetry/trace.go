package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartCommandSpan creates a span for a CLI command execution.
// It automatically records command name, arguments, and flags as attributes.
// Also records command invocation metric.
//
// Usage:
//
//	ctx, span := telemetry.StartCommandSpan(ctx, "auto")
//	defer span.End()
//
//	span.SetAttributes(
//	    attribute.String("goal", goal),
//	    attribute.String("profile", profile),
//	)
func StartCommandSpan(ctx context.Context, cmdName string) (context.Context, trace.Span) {
	tracer := GetTracerProvider().Tracer("commands")
	ctx, span := tracer.Start(ctx, "command."+cmdName)

	span.SetAttributes(
		attribute.String("command", cmdName),
		attribute.String("component", "cli"),
	)

	// Record command invocation metric
	RecordCommandInvocation(ctx, cmdName, "started")

	return ctx, span
}

// StartProviderSpan creates a span for a provider API call.
// It automatically records provider name, operation, and model as attributes.
// Also records provider call metric.
//
// Usage:
//
//	ctx, span := telemetry.StartProviderSpan(ctx, "anthropic", "generate")
//	defer span.End()
//
//	span.SetAttributes(
//	    attribute.String("model", "claude-sonnet-3.5"),
//	    attribute.Int("max_tokens", 4096),
//	)
func StartProviderSpan(ctx context.Context, providerName, operation string) (context.Context, trace.Span) {
	tracer := GetTracerProvider().Tracer("providers")
	ctx, span := tracer.Start(ctx, "provider."+operation)

	span.SetAttributes(
		attribute.String("provider", providerName),
		attribute.String("operation", operation),
		attribute.String("component", "provider"),
	)

	// Record provider call metric
	RecordProviderCall(ctx, providerName, operation, "started")

	return ctx, span
}

// StartSubprocessSpan creates a span for subprocess operations (auto mode steps).
// It automatically records step name and sequence information.
//
// Usage:
//
//	ctx, span := telemetry.StartSubprocessSpan(ctx, "spec_generation")
//	defer span.End()
//
//	span.SetAttributes(
//	    attribute.Int("step_number", 1),
//	    attribute.Int("total_steps", 5),
//	)
func StartSubprocessSpan(ctx context.Context, stepName string) (context.Context, trace.Span) {
	tracer := GetTracerProvider().Tracer("auto")
	ctx, span := tracer.Start(ctx, "auto."+stepName)

	span.SetAttributes(
		attribute.String("step", stepName),
		attribute.String("component", "auto"),
	)

	return ctx, span
}

// RecordSuccess marks a span as successful with optional result attributes.
//
// Usage:
//
//	telemetry.RecordSuccess(span,
//	    attribute.Int("tokens_used", 1234),
//	    attribute.String("model", "claude-sonnet-3.5"),
//	)
func RecordSuccess(span trace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
	span.SetStatus(codes.Ok, "")
}

// RecordError records an error in a span and sets error status.
// This should be called when an operation fails.
//
// Usage:
//
//	if err != nil {
//	    telemetry.RecordError(span, err)
//	    return err
//	}
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.SetAttributes(
		attribute.Bool("error", true),
	)
}

// RecordCommandSuccess records command success in both traces and metrics.
// Call this when a command completes successfully.
//
// Usage:
//
//	telemetry.RecordCommandSuccess(ctx, span, "auto", duration,
//	    attribute.Int("tasks_count", 5),
//	)
func RecordCommandSuccess(ctx context.Context, span trace.Span, cmdName string, duration time.Duration, attrs ...attribute.KeyValue) {
	// Record in trace
	RecordSuccess(span, attrs...)

	// Record metrics
	RecordCommandInvocation(ctx, cmdName, "success", attrs...)
	RecordCommandDuration(ctx, cmdName, duration, attrs...)
}

// RecordCommandFailure records command failure in both traces and metrics.
// Call this when a command fails.
//
// Usage:
//
//	if err != nil {
//	    telemetry.RecordCommandFailure(ctx, span, "auto", err)
//	    return err
//	}
func RecordCommandFailure(ctx context.Context, span trace.Span, cmdName string, err error) {
	if err == nil {
		return
	}

	// Record in trace
	RecordError(span, err)

	// Record metrics
	errorType := "unknown"
	if err != nil {
		errorType = "execution_error"
	}
	RecordCommandInvocation(ctx, cmdName, "failed")
	RecordCommandError(ctx, cmdName, errorType)
}

// RecordProviderSuccess records provider call success in both traces and metrics.
// Call this when a provider API call completes successfully.
//
// Usage:
//
//	telemetry.RecordProviderSuccess(ctx, span, "anthropic", "generate", duration,
//	    attribute.String("model", "claude-3-sonnet"),
//	    attribute.Int("tokens", 1234),
//	)
func RecordProviderSuccess(ctx context.Context, span trace.Span, provider string, operation string, duration time.Duration, attrs ...attribute.KeyValue) {
	// Record in trace
	RecordSuccess(span, attrs...)

	// Record metrics
	RecordProviderCall(ctx, provider, operation, "success", attrs...)
	RecordProviderLatency(ctx, provider, operation, duration, attrs...)
}

// RecordProviderFailure records provider call failure in both traces and metrics.
// Call this when a provider API call fails.
//
// Usage:
//
//	if err != nil {
//	    telemetry.RecordProviderFailure(ctx, span, "anthropic", "generate", err)
//	    return err
//	}
func RecordProviderFailure(ctx context.Context, span trace.Span, provider string, operation string, err error) {
	if err == nil {
		return
	}

	// Record in trace
	RecordError(span, err)

	// Record metrics
	errorType := "api_error"
	if err != nil {
		errorType = "api_error"
	}
	RecordProviderCall(ctx, provider, operation, "failed")
	RecordProviderError(ctx, provider, operation, errorType)
}

// RecordDuration records the duration of an operation as a span attribute.
// This is useful when you want to manually track timing.
//
// Usage:
//
//	start := time.Now()
//	// ... operation ...
//	telemetry.RecordDuration(span, "api_call_duration", time.Since(start))
func RecordDuration(span trace.Span, name string, duration time.Duration) {
	span.SetAttributes(
		attribute.Int64(name+"_ms", duration.Milliseconds()),
	)
}

// RecordMetrics records common metrics as span attributes.
// This is useful for operations that produce measurable results.
//
// Usage:
//
//	telemetry.RecordMetrics(span,
//	    "lines_of_code", 1234,
//	    "files_modified", 5,
//	)
func RecordMetrics(span trace.Span, metrics map[string]int64) {
	for key, value := range metrics {
		span.SetAttributes(
			attribute.Int64(key, value),
		)
	}
}

// TraceFunction wraps a function call with automatic span creation and error handling.
// This is useful for tracing simple functions without boilerplate.
//
// Usage:
//
//	result, err := telemetry.TraceFunction(ctx, "process_spec", func(ctx context.Context) (interface{}, error) {
//	    return processSpec(ctx, spec)
//	})
func TraceFunction(ctx context.Context, name string, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	tracer := GetTracerProvider().Tracer("general")
	ctx, span := tracer.Start(ctx, name)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		RecordError(span, err)
		return nil, err
	}

	RecordSuccess(span)
	return result, nil
}
