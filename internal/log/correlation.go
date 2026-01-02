package log

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

// correlationIDKey is the context key for correlation IDs
type correlationIDKey struct{}

// requestIDKey is the context key for request IDs (shorter, for logging)
type requestIDKey struct{}

// CorrelationID represents a unique identifier for correlating log entries
// across distributed systems and service boundaries.
type CorrelationID string

// RequestID represents a shorter identifier for request-level tracing
// within a single service instance.
type RequestID string

var (
	// idPool is used to reduce allocations for ID generation
	idPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 16)
		},
	}
)

// NewCorrelationID generates a new unique correlation ID.
// The ID is a 32-character hex string (128 bits of randomness).
func NewCorrelationID() CorrelationID {
	b := idPool.Get().([]byte)
	defer idPool.Put(b)

	if _, err := rand.Read(b); err != nil {
		// Fallback to a less random but still unique ID
		return CorrelationID("fallback-" + hex.EncodeToString(b[:8]))
	}

	return CorrelationID(hex.EncodeToString(b))
}

// NewRequestID generates a new short request ID.
// The ID is a 16-character hex string (64 bits of randomness).
func NewRequestID() RequestID {
	b := idPool.Get().([]byte)
	defer idPool.Put(b)

	if _, err := rand.Read(b[:8]); err != nil {
		return RequestID("fallback")
	}

	return RequestID(hex.EncodeToString(b[:8]))
}

// String returns the string representation of the correlation ID
func (c CorrelationID) String() string {
	return string(c)
}

// String returns the string representation of the request ID
func (r RequestID) String() string {
	return string(r)
}

// IsEmpty returns true if the correlation ID is empty
func (c CorrelationID) IsEmpty() bool {
	return c == ""
}

// IsEmpty returns true if the request ID is empty
func (r RequestID) IsEmpty() bool {
	return r == ""
}

// WithCorrelationID adds a correlation ID to the context.
// If an empty correlation ID is provided, a new one is generated.
func WithCorrelationID(ctx context.Context, id CorrelationID) context.Context {
	if id.IsEmpty() {
		id = NewCorrelationID()
	}
	return context.WithValue(ctx, correlationIDKey{}, id)
}

// WithRequestID adds a request ID to the context.
// If an empty request ID is provided, a new one is generated.
func WithRequestID(ctx context.Context, id RequestID) context.Context {
	if id.IsEmpty() {
		id = NewRequestID()
	}
	return context.WithValue(ctx, requestIDKey{}, id)
}

// CorrelationIDFromContext extracts the correlation ID from the context.
// Returns an empty CorrelationID if none is set.
func CorrelationIDFromContext(ctx context.Context) CorrelationID {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(correlationIDKey{}).(CorrelationID); ok {
		return id
	}
	return ""
}

// RequestIDFromContext extracts the request ID from the context.
// Returns an empty RequestID if none is set.
func RequestIDFromContext(ctx context.Context) RequestID {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDKey{}).(RequestID); ok {
		return id
	}
	return ""
}

// TraceIDFromContext extracts the OpenTelemetry trace ID from the context.
// Returns an empty string if no valid trace is in the context.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	sc := span.SpanContext()
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

// SpanIDFromContext extracts the OpenTelemetry span ID from the context.
// Returns an empty string if no valid span is in the context.
func SpanIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	sc := span.SpanContext()
	if !sc.IsValid() {
		return ""
	}
	return sc.SpanID().String()
}

// ExtractContextIDs extracts all available IDs from the context.
// This is useful for adding all IDs to log entries at once.
// Returns a map of ID name to value, only including non-empty IDs.
func ExtractContextIDs(ctx context.Context) map[string]string {
	ids := make(map[string]string)

	if correlationID := CorrelationIDFromContext(ctx); !correlationID.IsEmpty() {
		ids["correlation_id"] = correlationID.String()
	}

	if requestID := RequestIDFromContext(ctx); !requestID.IsEmpty() {
		ids["request_id"] = requestID.String()
	}

	if traceID := TraceIDFromContext(ctx); traceID != "" {
		ids["trace_id"] = traceID
	}

	if spanID := SpanIDFromContext(ctx); spanID != "" {
		ids["span_id"] = spanID
	}

	return ids
}

// ContextWithIDs adds both correlation and request IDs to the context.
// If either ID is empty, a new one is generated.
func ContextWithIDs(ctx context.Context, correlationID CorrelationID, requestID RequestID) context.Context {
	ctx = WithCorrelationID(ctx, correlationID)
	ctx = WithRequestID(ctx, requestID)
	return ctx
}

// NewContextWithIDs creates a new context with fresh correlation and request IDs.
func NewContextWithIDs(ctx context.Context) context.Context {
	return ContextWithIDs(ctx, NewCorrelationID(), NewRequestID())
}

// PropagateCorrelationID ensures that a correlation ID exists in the context.
// If one already exists, the context is returned unchanged.
// If none exists, a new correlation ID is generated and added.
func PropagateCorrelationID(ctx context.Context) context.Context {
	if CorrelationIDFromContext(ctx).IsEmpty() {
		ctx = WithCorrelationID(ctx, NewCorrelationID())
	}
	return ctx
}
