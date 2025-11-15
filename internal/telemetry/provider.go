package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	// globalProvider holds the current tracer provider
	globalProvider trace.TracerProvider
	// globalShutdown holds the shutdown function for the provider
	globalShutdown func(context.Context) error
	// providerMu protects access to global provider state
	providerMu sync.RWMutex
)

// circuitBreaker implements a simple circuit breaker pattern for export failures
type circuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failureCount     int
	lastFailureTime  time.Time
	state            string // "closed", "open", "half-open"
	mu               sync.RWMutex
}

func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{
		failureThreshold: 5,
		resetTimeout:     30 * time.Second,
		state:            "closed",
	}
}

func (cb *circuitBreaker) allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == "closed" {
		return true
	}

	if cb.state == "open" {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			return true // transition to half-open
		}
		return false
	}

	// half-open state allows one request through
	return true
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.state = "closed"
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.failureThreshold {
		cb.state = "open"
	} else if cb.state == "half-open" {
		cb.state = "open"
	}
}

// retryableExporter wraps an exporter with retry logic and circuit breaker
type retryableExporter struct {
	exporter       sdktrace.SpanExporter
	circuitBreaker *circuitBreaker
}

func newRetryableExporter(exporter sdktrace.SpanExporter) *retryableExporter {
	return &retryableExporter{
		exporter:       exporter,
		circuitBreaker: newCircuitBreaker(),
	}
}

func (re *retryableExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if !re.circuitBreaker.allow() {
		return fmt.Errorf("circuit breaker open: too many export failures")
	}

	// Exponential backoff configuration
	const (
		initialInterval = 100 * time.Millisecond
		maxInterval     = 2 * time.Second
		maxElapsedTime  = 10 * time.Second
		multiplier      = 1.5
		maxRetries      = 5
	)

	start := time.Now()
	interval := initialInterval

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check if we've exceeded max elapsed time
		if time.Since(start) > maxElapsedTime {
			break
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			re.circuitBreaker.recordFailure()
			return ctx.Err()
		default:
		}

		// Try to export
		err := re.exporter.ExportSpans(ctx, spans)
		if err == nil {
			re.circuitBreaker.recordSuccess()
			return nil
		}

		lastErr = err

		// Don't sleep on last attempt
		if attempt < maxRetries-1 {
			// Wait with exponential backoff
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				re.circuitBreaker.recordFailure()
				return ctx.Err()
			}

			// Increase interval for next retry
			interval = time.Duration(float64(interval) * multiplier)
			if interval > maxInterval {
				interval = maxInterval
			}
		}
	}

	re.circuitBreaker.recordFailure()
	return fmt.Errorf("export failed after %d attempts: %w", maxRetries, lastErr)
}

func (re *retryableExporter) Shutdown(ctx context.Context) error {
	return re.exporter.Shutdown(ctx)
}

// createResource creates an OTLP resource with service information
func createResource(cfg Config) (*resource.Resource, error) {
	return resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
		resource.WithProcessRuntimeDescription(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithTelemetrySDK(),
	)
}

// InitProvider initializes the OpenTelemetry tracer provider
// Returns a shutdown function and any initialization error
func InitProvider(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	providerMu.Lock()
	defer providerMu.Unlock()

	// If tracing is disabled, use noop provider
	if !cfg.Enabled {
		globalProvider = noop.NewTracerProvider()
		globalShutdown = func(context.Context) error { return nil }
		otel.SetTracerProvider(globalProvider)
		return globalShutdown, nil
	}

	// Create resource with service information
	res, err := createResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider options
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}

	// Configure sampler based on sample rate
	if cfg.SampleRate < 1.0 {
		opts = append(opts, sdktrace.WithSampler(
			sdktrace.TraceIDRatioBased(cfg.SampleRate),
		))
	} else {
		opts = append(opts, sdktrace.WithSampler(
			sdktrace.AlwaysSample(),
		))
	}

	// If endpoint is configured, set up OTLP exporter
	if cfg.Endpoint != "" {
		exporter, err := otlptracehttp.New(
			ctx,
			otlptracehttp.WithEndpoint(cfg.Endpoint),
			otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}

		// Wrap exporter with retry logic and circuit breaker
		retryExporter := newRetryableExporter(exporter)

		// Use batch span processor for better performance
		opts = append(opts, sdktrace.WithBatcher(
			retryExporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		))
	}

	// Create and set the global tracer provider
	tp := sdktrace.NewTracerProvider(opts...)
	globalProvider = tp
	otel.SetTracerProvider(tp)

	// Set up shutdown function
	globalShutdown = func(shutdownCtx context.Context) error {
		return tp.Shutdown(shutdownCtx)
	}

	// Start runtime instrumentation if enabled
	if cfg.Enabled {
		if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
			// Non-fatal: log but continue
			fmt.Printf("Warning: failed to start runtime instrumentation: %v\n", err)
		}
	}

	return globalShutdown, nil
}

// Shutdown gracefully shuts down the tracer provider
func Shutdown(ctx context.Context) error {
	providerMu.RLock()
	shutdown := globalShutdown
	providerMu.RUnlock()

	if shutdown != nil {
		return shutdown(ctx)
	}
	return nil
}

// ForceFlush forces all pending spans to be exported
func ForceFlush(ctx context.Context) error {
	providerMu.RLock()
	provider := globalProvider
	providerMu.RUnlock()

	if tp, ok := provider.(*sdktrace.TracerProvider); ok {
		return tp.ForceFlush(ctx)
	}
	return nil
}

// GetTracerProvider returns the current global tracer provider
func GetTracerProvider() trace.TracerProvider {
	providerMu.RLock()
	defer providerMu.RUnlock()

	if globalProvider != nil {
		return globalProvider
	}
	return noop.NewTracerProvider()
}
