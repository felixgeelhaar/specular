package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// BenchmarkCounterInc benchmarks counter increments
// Target: < 100 ns/op (from ADR 0009)
func BenchmarkCounterInc(b *testing.B) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.CommandExecutions.WithLabelValues("test", "true").Inc()
	}
}

// BenchmarkCounterAdd benchmarks counter additions
func BenchmarkCounterAdd(b *testing.B) {
	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "benchmark_counter_add",
		Help: "Benchmark counter add",
	})
	reg.MustRegister(counter)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		counter.Add(1.0)
	}
}

// BenchmarkGaugeSet benchmarks gauge updates
func BenchmarkGaugeSet(b *testing.B) {
	reg := prometheus.NewRegistry()
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "benchmark_gauge",
		Help: "Benchmark gauge",
	})
	reg.MustRegister(gauge)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gauge.Set(float64(i))
	}
}

// BenchmarkHistogramObserve benchmarks histogram observations
func BenchmarkHistogramObserve(b *testing.B) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.CommandDuration.WithLabelValues("test").Observe(0.123)
	}
}

// BenchmarkCounterVecWithLabels benchmarks counter with labels
func BenchmarkCounterVecWithLabels(b *testing.B) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.ProviderCalls.WithLabelValues("test-provider", "test-model", "true").Inc()
	}
}

// BenchmarkHistogramVecWithLabels benchmarks histogram with labels
func BenchmarkHistogramVecWithLabels(b *testing.B) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.ProviderLatency.WithLabelValues("test-provider", "test-model").Observe(0.5)
	}
}

// BenchmarkMetricsParallel benchmarks concurrent metric updates
func BenchmarkMetricsParallel(b *testing.B) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.CommandExecutions.WithLabelValues("test", "true").Inc()
		}
	})
}

// BenchmarkMetricsInit benchmarks metrics initialization with fresh registry
func BenchmarkMetricsInit(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = NewRegistry()
	}
}

// BenchmarkErrorMetrics benchmarks error counter
func BenchmarkErrorMetrics(b *testing.B) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.Errors.WithLabelValues("validation_error", "cmd").Inc()
	}
}
