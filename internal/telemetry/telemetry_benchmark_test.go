package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// BenchmarkSpanCreation benchmarks span creation and end
// Target: < 350 ns/op (from ADR 0009)
func BenchmarkSpanCreation(b *testing.B) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Enabled = true

	// Use in-memory exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	res, _ := createResource(cfg)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	tracer := tp.Tracer("benchmark")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "benchmark-span")
		span.End()
	}

	// Cleanup
	_ = tp.Shutdown(ctx)
}

// BenchmarkSpanWithAttributes benchmarks span with attributes
func BenchmarkSpanWithAttributes(b *testing.B) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Enabled = true

	exporter := tracetest.NewInMemoryExporter()
	res, _ := createResource(cfg)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	tracer := tp.Tracer("benchmark")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "benchmark-span-with-attrs")
		span.SetAttributes(
			attribute.String("key1", "value1"),
			attribute.Int("key2", 42),
			attribute.Bool("key3", true),
		)
		span.End()
	}

	_ = tp.Shutdown(ctx)
}

// BenchmarkNestedSpans benchmarks nested span creation
func BenchmarkNestedSpans(b *testing.B) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Enabled = true

	exporter := tracetest.NewInMemoryExporter()
	res, _ := createResource(cfg)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	tracer := tp.Tracer("benchmark")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, parentSpan := tracer.Start(ctx, "parent-span")
		parentCtx := trace.ContextWithSpan(ctx, parentSpan)

		_, childSpan := tracer.Start(parentCtx, "child-span")
		childSpan.End()

		parentSpan.End()
	}

	_ = tp.Shutdown(ctx)
}

// BenchmarkSpanWithSampling benchmarks span creation with sampling
func BenchmarkSpanWithSampling(b *testing.B) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.SampleRate = 0.5 // 50% sampling

	exporter := tracetest.NewInMemoryExporter()
	res, _ := createResource(cfg)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	)

	tracer := tp.Tracer("benchmark")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "sampled-span")
		span.End()
	}

	_ = tp.Shutdown(ctx)
}

// BenchmarkBatchProcessor benchmarks batch span processor
func BenchmarkBatchProcessor(b *testing.B) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Enabled = true

	exporter := tracetest.NewInMemoryExporter()
	res, _ := createResource(cfg)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(
			exporter,
			sdktrace.WithBatchTimeout(5000),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	tracer := tp.Tracer("benchmark")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "batched-span")
		span.End()
	}

	_ = tp.Shutdown(ctx)
}

// BenchmarkNoopProvider benchmarks noop provider overhead
func BenchmarkNoopProvider(b *testing.B) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Enabled = false

	shutdown, _ := InitProvider(ctx, cfg)
	defer shutdown(ctx)

	tracer := GetTracerProvider().Tracer("benchmark")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "noop-span")
		span.End()
	}
}

// BenchmarkProviderConcurrent benchmarks concurrent span creation
func BenchmarkProviderConcurrent(b *testing.B) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Enabled = true

	exporter := tracetest.NewInMemoryExporter()
	res, _ := createResource(cfg)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	tracer := tp.Tracer("benchmark")

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, span := tracer.Start(ctx, "concurrent-span")
			span.End()
		}
	})

	_ = tp.Shutdown(ctx)
}

// BenchmarkGetTracerProvider benchmarks GetTracerProvider calls
func BenchmarkGetTracerProvider(b *testing.B) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Enabled = true

	shutdown, _ := InitProvider(ctx, cfg)
	defer shutdown(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = GetTracerProvider()
	}
}
