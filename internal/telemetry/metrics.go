// Package telemetry provides production-grade metrics collection using
// OpenTelemetry with OTLP HTTP export and Prometheus-compatible metric types.
package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	// globalMeterProvider holds the current meter provider
	globalMeterProvider metric.MeterProvider
	// globalMetricsShutdown holds the shutdown function for metrics
	globalMetricsShutdown func(context.Context) error
	// meterMu protects access to global meter provider state
	meterMu sync.RWMutex
	// metrics holds all registered metrics
	metrics *Metrics
	// metricsOnce ensures metrics are initialized only once
	metricsOnce sync.Once
)

// Metrics holds all registered OpenTelemetry metrics
type Metrics struct {
	// Command metrics
	CommandCounter       metric.Int64Counter
	CommandDuration      metric.Float64Histogram
	CommandErrorCounter  metric.Int64Counter

	// Provider metrics
	ProviderCallCounter  metric.Int64Counter
	ProviderLatency      metric.Float64Histogram
	ProviderErrorCounter metric.Int64Counter
	ProviderTokenCounter metric.Int64Counter
}

// InitMetricsProvider initializes the OpenTelemetry metrics provider
// Returns a shutdown function and any initialization error
func InitMetricsProvider(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	meterMu.Lock()
	defer meterMu.Unlock()

	// If metrics are disabled, use noop provider
	if !cfg.Enabled {
		globalMeterProvider = otel.GetMeterProvider() // Uses global noop by default
		globalMetricsShutdown = func(context.Context) error { return nil }
		return globalMetricsShutdown, nil
	}

	// Create resource with service information (reuse from tracing)
	res, err := createResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource for metrics: %w", err)
	}

	// If endpoint is configured, set up OTLP metrics exporter
	if cfg.Endpoint != "" {
		exporter, err := otlpmetrichttp.New(
			ctx,
			otlpmetrichttp.WithEndpoint(cfg.Endpoint),
			otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP metrics exporter: %w", err)
		}

		// Create meter provider with periodic reader
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(
				sdkmetric.NewPeriodicReader(
					exporter,
					sdkmetric.WithInterval(10*time.Second), // Export every 10 seconds
				),
			),
		)

		globalMeterProvider = mp
		otel.SetMeterProvider(mp)

		// Set up shutdown function
		globalMetricsShutdown = func(shutdownCtx context.Context) error {
			return mp.Shutdown(shutdownCtx)
		}
	} else {
		// No endpoint configured, use noop
		globalMeterProvider = otel.GetMeterProvider()
		globalMetricsShutdown = func(context.Context) error { return nil }
	}

	// Initialize metrics
	if err := initMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	return globalMetricsShutdown, nil
}

// initMetrics initializes all metric instruments
func initMetrics() error {
	var initErr error
	metricsOnce.Do(func() {
		meter := globalMeterProvider.Meter("github.com/felixgeelhaar/specular")

		m := &Metrics{}

		// Command metrics
		m.CommandCounter, initErr = meter.Int64Counter(
			"specular.command.invocations",
			metric.WithDescription("Total number of command invocations"),
			metric.WithUnit("{invocation}"),
		)
		if initErr != nil {
			return
		}

		m.CommandDuration, initErr = meter.Float64Histogram(
			"specular.command.duration",
			metric.WithDescription("Command execution duration in seconds"),
			metric.WithUnit("s"),
		)
		if initErr != nil {
			return
		}

		m.CommandErrorCounter, initErr = meter.Int64Counter(
			"specular.command.errors",
			metric.WithDescription("Total number of command errors"),
			metric.WithUnit("{error}"),
		)
		if initErr != nil {
			return
		}

		// Provider metrics
		m.ProviderCallCounter, initErr = meter.Int64Counter(
			"specular.provider.calls",
			metric.WithDescription("Total number of provider API calls"),
			metric.WithUnit("{call}"),
		)
		if initErr != nil {
			return
		}

		m.ProviderLatency, initErr = meter.Float64Histogram(
			"specular.provider.latency",
			metric.WithDescription("Provider API call latency in seconds"),
			metric.WithUnit("s"),
		)
		if initErr != nil {
			return
		}

		m.ProviderErrorCounter, initErr = meter.Int64Counter(
			"specular.provider.errors",
			metric.WithDescription("Total number of provider API errors"),
			metric.WithUnit("{error}"),
		)
		if initErr != nil {
			return
		}

		m.ProviderTokenCounter, initErr = meter.Int64Counter(
			"specular.provider.tokens",
			metric.WithDescription("Total number of tokens used"),
			metric.WithUnit("{token}"),
		)
		if initErr != nil {
			return
		}

		metrics = m
	})

	return initErr
}

// GetMetrics returns the initialized metrics instance
// Initializes with noop metrics if not already initialized
func GetMetrics() *Metrics {
	meterMu.RLock()
	defer meterMu.RUnlock()

	if metrics != nil {
		return metrics
	}

	// Return empty metrics if not initialized (noop behavior)
	return &Metrics{}
}

// RecordCommandInvocation records a command invocation
func RecordCommandInvocation(ctx context.Context, commandName string, status string, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.CommandCounter == nil {
		return
	}

	baseAttrs := []attribute.KeyValue{
		attribute.String("command", commandName),
		attribute.String("status", status),
	}
	baseAttrs = append(baseAttrs, attrs...)

	m.CommandCounter.Add(ctx, 1, metric.WithAttributes(baseAttrs...))
}

// RecordCommandDuration records command execution duration
func RecordCommandDuration(ctx context.Context, commandName string, duration time.Duration, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.CommandDuration == nil {
		return
	}

	baseAttrs := []attribute.KeyValue{
		attribute.String("command", commandName),
	}
	baseAttrs = append(baseAttrs, attrs...)

	m.CommandDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(baseAttrs...))
}

// RecordCommandError records a command error
func RecordCommandError(ctx context.Context, commandName string, errorType string, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.CommandErrorCounter == nil {
		return
	}

	baseAttrs := []attribute.KeyValue{
		attribute.String("command", commandName),
		attribute.String("error_type", errorType),
	}
	baseAttrs = append(baseAttrs, attrs...)

	m.CommandErrorCounter.Add(ctx, 1, metric.WithAttributes(baseAttrs...))
}

// RecordProviderCall records a provider API call
func RecordProviderCall(ctx context.Context, provider string, operation string, status string, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.ProviderCallCounter == nil {
		return
	}

	baseAttrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("operation", operation),
		attribute.String("status", status),
	}
	baseAttrs = append(baseAttrs, attrs...)

	m.ProviderCallCounter.Add(ctx, 1, metric.WithAttributes(baseAttrs...))
}

// RecordProviderLatency records provider API call latency
func RecordProviderLatency(ctx context.Context, provider string, operation string, duration time.Duration, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.ProviderLatency == nil {
		return
	}

	baseAttrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("operation", operation),
	}
	baseAttrs = append(baseAttrs, attrs...)

	m.ProviderLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(baseAttrs...))
}

// RecordProviderError records a provider API error
func RecordProviderError(ctx context.Context, provider string, operation string, errorType string, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.ProviderErrorCounter == nil {
		return
	}

	baseAttrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("operation", operation),
		attribute.String("error_type", errorType),
	}
	baseAttrs = append(baseAttrs, attrs...)

	m.ProviderErrorCounter.Add(ctx, 1, metric.WithAttributes(baseAttrs...))
}

// RecordProviderTokens records token usage
func RecordProviderTokens(ctx context.Context, provider string, model string, tokenType string, count int, attrs ...attribute.KeyValue) {
	m := GetMetrics()
	if m.ProviderTokenCounter == nil {
		return
	}

	baseAttrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("token_type", tokenType), // "input", "output", "total"
	}
	baseAttrs = append(baseAttrs, attrs...)

	m.ProviderTokenCounter.Add(ctx, int64(count), metric.WithAttributes(baseAttrs...))
}

// ShutdownMetrics gracefully shuts down the metrics provider
func ShutdownMetrics(ctx context.Context) error {
	meterMu.RLock()
	shutdown := globalMetricsShutdown
	meterMu.RUnlock()

	if shutdown != nil {
		return shutdown(ctx)
	}
	return nil
}

// ForceFlushMetrics forces all pending metrics to be exported
func ForceFlushMetrics(ctx context.Context) error {
	meterMu.RLock()
	provider := globalMeterProvider
	meterMu.RUnlock()

	if mp, ok := provider.(*sdkmetric.MeterProvider); ok {
		return mp.ForceFlush(ctx)
	}
	return nil
}
