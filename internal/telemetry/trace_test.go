package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTestTracer creates a test tracer with in-memory exporter
func setupTestTracer(t *testing.T) (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	cfg := DefaultConfig()
	res, err := createResource(cfg)
	if err != nil {
		t.Fatalf("createResource failed: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global provider
	providerMu.Lock()
	globalProvider = tp
	providerMu.Unlock()

	return tp, exporter
}

func TestStartCommandSpan(t *testing.T) {
	tp, exporter := setupTestTracer(t)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	cmdName := "test-command"

	// Start span
	spanCtx, span := StartCommandSpan(ctx, cmdName)
	if span == nil {
		t.Fatal("expected span, got nil")
	}

	// Verify span context is propagated
	if spanCtx == ctx {
		t.Error("expected new context with span, got same context")
	}

	// End span
	span.End()

	// Verify span was recorded
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	recordedSpan := spans[0]
	expectedName := "command." + cmdName

	if recordedSpan.Name != expectedName {
		t.Errorf("span name = %q, want %q", recordedSpan.Name, expectedName)
	}

	// Verify attributes
	attrs := recordedSpan.Attributes
	hasCommand := false
	hasComponent := false

	for _, attr := range attrs {
		if attr.Key == "command" && attr.Value.AsString() == cmdName {
			hasCommand = true
		}
		if attr.Key == "component" && attr.Value.AsString() == "cli" {
			hasComponent = true
		}
	}

	if !hasCommand {
		t.Error("missing 'command' attribute")
	}
	if !hasComponent {
		t.Error("missing 'component' attribute")
	}
}

func TestStartProviderSpan(t *testing.T) {
	tp, exporter := setupTestTracer(t)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	providerName := "test-provider"
	operation := "generate"

	// Start span
	spanCtx, span := StartProviderSpan(ctx, providerName, operation)
	if span == nil {
		t.Fatal("expected span, got nil")
	}

	// Verify span context is propagated
	if spanCtx == ctx {
		t.Error("expected new context with span, got same context")
	}

	// End span
	span.End()

	// Verify span was recorded
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	recordedSpan := spans[0]
	expectedName := "provider." + operation

	if recordedSpan.Name != expectedName {
		t.Errorf("span name = %q, want %q", recordedSpan.Name, expectedName)
	}

	// Verify attributes
	attrs := recordedSpan.Attributes
	hasProvider := false
	hasOperation := false
	hasComponent := false

	for _, attr := range attrs {
		if attr.Key == "provider" && attr.Value.AsString() == providerName {
			hasProvider = true
		}
		if attr.Key == "operation" && attr.Value.AsString() == operation {
			hasOperation = true
		}
		if attr.Key == "component" && attr.Value.AsString() == "provider" {
			hasComponent = true
		}
	}

	if !hasProvider {
		t.Error("missing 'provider' attribute")
	}
	if !hasOperation {
		t.Error("missing 'operation' attribute")
	}
	if !hasComponent {
		t.Error("missing 'component' attribute")
	}
}

func TestStartSubprocessSpan(t *testing.T) {
	tp, exporter := setupTestTracer(t)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	stepName := "spec_generation"

	// Start span
	spanCtx, span := StartSubprocessSpan(ctx, stepName)
	if span == nil {
		t.Fatal("expected span, got nil")
	}

	// Verify span context is propagated
	if spanCtx == ctx {
		t.Error("expected new context with span, got same context")
	}

	// End span
	span.End()

	// Verify span was recorded
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	recordedSpan := spans[0]
	expectedName := "auto." + stepName

	if recordedSpan.Name != expectedName {
		t.Errorf("span name = %q, want %q", recordedSpan.Name, expectedName)
	}

	// Verify attributes
	attrs := recordedSpan.Attributes
	hasStep := false
	hasComponent := false

	for _, attr := range attrs {
		if attr.Key == "step" && attr.Value.AsString() == stepName {
			hasStep = true
		}
		if attr.Key == "component" && attr.Value.AsString() == "auto" {
			hasComponent = true
		}
	}

	if !hasStep {
		t.Error("missing 'step' attribute")
	}
	if !hasComponent {
		t.Error("missing 'component' attribute")
	}
}

func TestRecordSuccess(t *testing.T) {
	tp, exporter := setupTestTracer(t)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	_, span := StartCommandSpan(ctx, "test")

	// Record success with attributes
	RecordSuccess(span,
		attribute.Int("tokens_used", 1234),
		attribute.String("model", "test-model"),
	)

	span.End()

	// Verify span status
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	recordedSpan := spans[0]

	// Verify status is OK
	if recordedSpan.Status.Code != codes.Ok {
		t.Errorf("status code = %v, want %v", recordedSpan.Status.Code, codes.Ok)
	}

	// Verify attributes were added
	hasTokens := false
	hasModel := false

	for _, attr := range recordedSpan.Attributes {
		if attr.Key == "tokens_used" && attr.Value.AsInt64() == 1234 {
			hasTokens = true
		}
		if attr.Key == "model" && attr.Value.AsString() == "test-model" {
			hasModel = true
		}
	}

	if !hasTokens {
		t.Error("missing 'tokens_used' attribute")
	}
	if !hasModel {
		t.Error("missing 'model' attribute")
	}
}

func TestRecordError(t *testing.T) {
	tp, exporter := setupTestTracer(t)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	_, span := StartCommandSpan(ctx, "test")

	// Record error
	testErr := errors.New("test error")
	RecordError(span, testErr)

	span.End()

	// Verify span status
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	recordedSpan := spans[0]

	// Verify status is Error
	if recordedSpan.Status.Code != codes.Error {
		t.Errorf("status code = %v, want %v", recordedSpan.Status.Code, codes.Error)
	}

	// Verify error message
	if recordedSpan.Status.Description != testErr.Error() {
		t.Errorf("status description = %q, want %q", recordedSpan.Status.Description, testErr.Error())
	}

	// Verify error attribute
	hasErrorAttr := false
	for _, attr := range recordedSpan.Attributes {
		if attr.Key == "error" && attr.Value.AsBool() {
			hasErrorAttr = true
		}
	}

	if !hasErrorAttr {
		t.Error("missing 'error' attribute")
	}

	// Verify error event was recorded
	if len(recordedSpan.Events) == 0 {
		t.Error("expected error event, got none")
	}
}

func TestRecordErrorWithNil(t *testing.T) {
	tp, exporter := setupTestTracer(t)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	_, span := StartCommandSpan(ctx, "test")

	// Record nil error (should be no-op)
	RecordError(span, nil)

	span.End()

	// Verify span status is still unset (not error)
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	recordedSpan := spans[0]

	// Status should be Unset, not Error
	if recordedSpan.Status.Code == codes.Error {
		t.Error("status should not be Error when error is nil")
	}
}

func TestRecordDuration(t *testing.T) {
	tp, exporter := setupTestTracer(t)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	_, span := StartCommandSpan(ctx, "test")

	// Record duration
	duration := 1500 * time.Millisecond
	RecordDuration(span, "api_call_duration", duration)

	span.End()

	// Verify duration attribute
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	recordedSpan := spans[0]

	hasDuration := false
	for _, attr := range recordedSpan.Attributes {
		if attr.Key == "api_call_duration_ms" && attr.Value.AsInt64() == 1500 {
			hasDuration = true
		}
	}

	if !hasDuration {
		t.Error("missing 'api_call_duration_ms' attribute with correct value")
	}
}

func TestRecordMetrics(t *testing.T) {
	tp, exporter := setupTestTracer(t)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	_, span := StartCommandSpan(ctx, "test")

	// Record metrics
	metrics := map[string]int64{
		"lines_of_code":  1234,
		"files_modified": 5,
		"tests_added":    12,
	}
	RecordMetrics(span, metrics)

	span.End()

	// Verify all metrics were recorded as attributes
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	recordedSpan := spans[0]

	expectedMetrics := map[string]int64{
		"lines_of_code":  1234,
		"files_modified": 5,
		"tests_added":    12,
	}

	for key, expectedValue := range expectedMetrics {
		found := false
		for _, attr := range recordedSpan.Attributes {
			if string(attr.Key) == key && attr.Value.AsInt64() == expectedValue {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing or incorrect metric %q with value %d", key, expectedValue)
		}
	}
}

func TestTraceFunction(t *testing.T) {
	tp, exporter := setupTestTracer(t)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		exporter.Reset()

		result, err := TraceFunction(ctx, "test_function", func(ctx context.Context) (interface{}, error) {
			return "success result", nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result != "success result" {
			t.Errorf("result = %v, want %q", result, "success result")
		}

		// Verify span was recorded
		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}

		recordedSpan := spans[0]

		if recordedSpan.Name != "test_function" {
			t.Errorf("span name = %q, want %q", recordedSpan.Name, "test_function")
		}

		if recordedSpan.Status.Code != codes.Ok {
			t.Errorf("status code = %v, want %v", recordedSpan.Status.Code, codes.Ok)
		}
	})

	t.Run("error", func(t *testing.T) {
		exporter.Reset()

		testErr := errors.New("test error")

		result, err := TraceFunction(ctx, "test_function_error", func(ctx context.Context) (interface{}, error) {
			return nil, testErr
		})

		if err != testErr {
			t.Errorf("error = %v, want %v", err, testErr)
		}

		if result != nil {
			t.Errorf("result = %v, want nil", result)
		}

		// Verify span was recorded with error
		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}

		recordedSpan := spans[0]

		if recordedSpan.Status.Code != codes.Error {
			t.Errorf("status code = %v, want %v", recordedSpan.Status.Code, codes.Error)
		}
	})
}

func TestSpanContextPropagation(t *testing.T) {
	tp, exporter := setupTestTracer(t)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	ctx := context.Background()

	// Create parent span
	parentCtx, parentSpan := StartCommandSpan(ctx, "parent")

	// Create child span using parent context
	_, childSpan := StartProviderSpan(parentCtx, "test-provider", "generate")

	// End child first
	childSpan.End()
	// Then parent
	parentSpan.End()

	// Verify both spans were recorded
	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	// Verify parent-child relationship
	childSpanData := spans[0] // Child ends first
	parentSpanData := spans[1]

	if childSpanData.Parent.SpanID() != parentSpanData.SpanContext.SpanID() {
		t.Error("child span should have parent span as parent")
	}
}
