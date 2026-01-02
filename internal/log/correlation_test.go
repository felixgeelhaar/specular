package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestNewCorrelationID(t *testing.T) {
	id1 := NewCorrelationID()
	id2 := NewCorrelationID()

	// Should generate non-empty IDs
	assert.False(t, id1.IsEmpty())
	assert.False(t, id2.IsEmpty())

	// Should be 32 characters (128 bits hex encoded)
	assert.Len(t, id1.String(), 32)
	assert.Len(t, id2.String(), 32)

	// Should be unique
	assert.NotEqual(t, id1, id2)
}

func TestNewRequestID(t *testing.T) {
	id1 := NewRequestID()
	id2 := NewRequestID()

	// Should generate non-empty IDs
	assert.False(t, id1.IsEmpty())
	assert.False(t, id2.IsEmpty())

	// Should be 16 characters (64 bits hex encoded)
	assert.Len(t, id1.String(), 16)
	assert.Len(t, id2.String(), 16)

	// Should be unique
	assert.NotEqual(t, id1, id2)
}

func TestCorrelationID_String(t *testing.T) {
	id := CorrelationID("test-correlation-id")
	assert.Equal(t, "test-correlation-id", id.String())
}

func TestRequestID_String(t *testing.T) {
	id := RequestID("test-request-id")
	assert.Equal(t, "test-request-id", id.String())
}

func TestCorrelationID_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		id       CorrelationID
		expected bool
	}{
		{"empty string", CorrelationID(""), true},
		{"non-empty string", CorrelationID("abc123"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.id.IsEmpty())
		})
	}
}

func TestRequestID_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		id       RequestID
		expected bool
	}{
		{"empty string", RequestID(""), true},
		{"non-empty string", RequestID("abc123"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.id.IsEmpty())
		})
	}
}

func TestWithCorrelationID(t *testing.T) {
	ctx := context.Background()

	// Test with provided ID
	id := CorrelationID("my-correlation-id")
	ctx = WithCorrelationID(ctx, id)

	retrieved := CorrelationIDFromContext(ctx)
	assert.Equal(t, id, retrieved)
}

func TestWithCorrelationID_GeneratesWhenEmpty(t *testing.T) {
	ctx := context.Background()

	// Test with empty ID - should generate one
	ctx = WithCorrelationID(ctx, "")

	retrieved := CorrelationIDFromContext(ctx)
	assert.False(t, retrieved.IsEmpty())
	assert.Len(t, retrieved.String(), 32)
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()

	id := RequestID("my-request-id")
	ctx = WithRequestID(ctx, id)

	retrieved := RequestIDFromContext(ctx)
	assert.Equal(t, id, retrieved)
}

func TestWithRequestID_GeneratesWhenEmpty(t *testing.T) {
	ctx := context.Background()

	ctx = WithRequestID(ctx, "")

	retrieved := RequestIDFromContext(ctx)
	assert.False(t, retrieved.IsEmpty())
	assert.Len(t, retrieved.String(), 16)
}

func TestCorrelationIDFromContext_NilContext(t *testing.T) {
	id := CorrelationIDFromContext(nil)
	assert.True(t, id.IsEmpty())
}

func TestCorrelationIDFromContext_NoValue(t *testing.T) {
	ctx := context.Background()
	id := CorrelationIDFromContext(ctx)
	assert.True(t, id.IsEmpty())
}

func TestRequestIDFromContext_NilContext(t *testing.T) {
	id := RequestIDFromContext(nil)
	assert.True(t, id.IsEmpty())
}

func TestRequestIDFromContext_NoValue(t *testing.T) {
	ctx := context.Background()
	id := RequestIDFromContext(ctx)
	assert.True(t, id.IsEmpty())
}

func TestTraceIDFromContext(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		id := TraceIDFromContext(nil)
		assert.Empty(t, id)
	})

	t.Run("context without span", func(t *testing.T) {
		ctx := context.Background()
		id := TraceIDFromContext(ctx)
		assert.Empty(t, id)
	})

	t.Run("context with valid span", func(t *testing.T) {
		// Create a tracer provider
		tp := sdktrace.NewTracerProvider()
		defer func() { _ = tp.Shutdown(context.Background()) }()

		otel.SetTracerProvider(tp)
		tracer := tp.Tracer("test")

		ctx, span := tracer.Start(context.Background(), "test-span")
		defer span.End()

		id := TraceIDFromContext(ctx)
		assert.NotEmpty(t, id)
		assert.Len(t, id, 32) // Trace IDs are 16 bytes = 32 hex chars
	})
}

func TestSpanIDFromContext(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		id := SpanIDFromContext(nil)
		assert.Empty(t, id)
	})

	t.Run("context without span", func(t *testing.T) {
		ctx := context.Background()
		id := SpanIDFromContext(ctx)
		assert.Empty(t, id)
	})

	t.Run("context with valid span", func(t *testing.T) {
		tp := sdktrace.NewTracerProvider()
		defer func() { _ = tp.Shutdown(context.Background()) }()

		otel.SetTracerProvider(tp)
		tracer := tp.Tracer("test")

		ctx, span := tracer.Start(context.Background(), "test-span")
		defer span.End()

		id := SpanIDFromContext(ctx)
		assert.NotEmpty(t, id)
		assert.Len(t, id, 16) // Span IDs are 8 bytes = 16 hex chars
	})
}

func TestExtractContextIDs(t *testing.T) {
	t.Run("empty context", func(t *testing.T) {
		ctx := context.Background()
		ids := ExtractContextIDs(ctx)
		assert.Empty(t, ids)
	})

	t.Run("with correlation ID only", func(t *testing.T) {
		ctx := WithCorrelationID(context.Background(), "corr-123")
		ids := ExtractContextIDs(ctx)

		assert.Len(t, ids, 1)
		assert.Equal(t, "corr-123", ids["correlation_id"])
	})

	t.Run("with request ID only", func(t *testing.T) {
		ctx := WithRequestID(context.Background(), "req-456")
		ids := ExtractContextIDs(ctx)

		assert.Len(t, ids, 1)
		assert.Equal(t, "req-456", ids["request_id"])
	})

	t.Run("with both correlation and request IDs", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithCorrelationID(ctx, "corr-123")
		ctx = WithRequestID(ctx, "req-456")

		ids := ExtractContextIDs(ctx)

		assert.Len(t, ids, 2)
		assert.Equal(t, "corr-123", ids["correlation_id"])
		assert.Equal(t, "req-456", ids["request_id"])
	})

	t.Run("with trace context", func(t *testing.T) {
		tp := sdktrace.NewTracerProvider()
		defer func() { _ = tp.Shutdown(context.Background()) }()

		otel.SetTracerProvider(tp)
		tracer := tp.Tracer("test")

		ctx, span := tracer.Start(context.Background(), "test-span")
		defer span.End()

		ctx = WithCorrelationID(ctx, "corr-123")

		ids := ExtractContextIDs(ctx)

		assert.Equal(t, "corr-123", ids["correlation_id"])
		assert.NotEmpty(t, ids["trace_id"])
		assert.NotEmpty(t, ids["span_id"])
	})
}

func TestContextWithIDs(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithIDs(ctx, "corr-123", "req-456")

	assert.Equal(t, CorrelationID("corr-123"), CorrelationIDFromContext(ctx))
	assert.Equal(t, RequestID("req-456"), RequestIDFromContext(ctx))
}

func TestContextWithIDs_GeneratesWhenEmpty(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithIDs(ctx, "", "")

	correlationID := CorrelationIDFromContext(ctx)
	requestID := RequestIDFromContext(ctx)

	assert.False(t, correlationID.IsEmpty())
	assert.False(t, requestID.IsEmpty())
}

func TestNewContextWithIDs(t *testing.T) {
	ctx := NewContextWithIDs(context.Background())

	correlationID := CorrelationIDFromContext(ctx)
	requestID := RequestIDFromContext(ctx)

	assert.False(t, correlationID.IsEmpty())
	assert.False(t, requestID.IsEmpty())
	assert.Len(t, correlationID.String(), 32)
	assert.Len(t, requestID.String(), 16)
}

func TestPropagateCorrelationID(t *testing.T) {
	t.Run("generates new ID when none exists", func(t *testing.T) {
		ctx := context.Background()
		ctx = PropagateCorrelationID(ctx)

		id := CorrelationIDFromContext(ctx)
		assert.False(t, id.IsEmpty())
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		ctx := WithCorrelationID(context.Background(), "existing-id")
		ctx = PropagateCorrelationID(ctx)

		id := CorrelationIDFromContext(ctx)
		assert.Equal(t, CorrelationID("existing-id"), id)
	})
}

func TestLoggerWithContext(t *testing.T) {
	logger := Default()

	t.Run("with nil context", func(t *testing.T) {
		newLogger := logger.WithContext(nil)
		require.NotNil(t, newLogger)
	})

	t.Run("with empty context", func(t *testing.T) {
		newLogger := logger.WithContext(context.Background())
		require.NotNil(t, newLogger)
	})

	t.Run("with correlation ID", func(t *testing.T) {
		ctx := WithCorrelationID(context.Background(), "test-corr-id")
		newLogger := logger.WithContext(ctx)
		require.NotNil(t, newLogger)
		// The logger should have the correlation ID as an attribute
		// We can't easily verify this without inspecting internal state
	})
}

// Benchmark tests
func BenchmarkNewCorrelationID(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewCorrelationID()
	}
}

func BenchmarkNewRequestID(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewRequestID()
	}
}

func BenchmarkExtractContextIDs_Empty(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ExtractContextIDs(ctx)
	}
}

func BenchmarkExtractContextIDs_WithIDs(b *testing.B) {
	ctx := ContextWithIDs(context.Background(), "corr-123", "req-456")
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ExtractContextIDs(ctx)
	}
}

func BenchmarkExtractContextIDs_WithTrace(b *testing.B) {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	ctx = ContextWithIDs(ctx, "corr-123", "req-456")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ExtractContextIDs(ctx)
	}
}

// Test that trace.SpanFromContext returns a non-nil noop span for empty context
func TestTraceSpanFromContext_NoopBehavior(t *testing.T) {
	ctx := context.Background()
	span := trace.SpanFromContext(ctx)

	// SpanFromContext should return a non-nil noop span
	require.NotNil(t, span)

	// The span context should not be valid (it's a noop)
	assert.False(t, span.SpanContext().IsValid())
}
