package log

import (
	"io"
	"testing"
)

// BenchmarkLoggerInfo benchmarks Info level logging
// Target: < 580 ns/op (from ADR 0009)
func BenchmarkLoggerInfo(b *testing.B) {
	logger := New(Config{
		Level:       LevelInfo,
		Format:      FormatJSON,
		Output:      NewOutput(io.Discard),
		AddSource:   false,
		ServiceName: "benchmark",
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			"key1", "value1",
			"key2", 42,
			"key3", true,
		)
	}
}

// BenchmarkLoggerInfoWithSource benchmarks Info level logging with source location
func BenchmarkLoggerInfoWithSource(b *testing.B) {
	logger := New(Config{
		Level:       LevelInfo,
		Format:      FormatJSON,
		Output:      NewOutput(io.Discard),
		AddSource:   true,
		ServiceName: "benchmark",
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message with source",
			"key1", "value1",
			"key2", 42,
		)
	}
}

// BenchmarkLoggerDebug benchmarks Debug level logging when disabled
func BenchmarkLoggerDebugDisabled(b *testing.B) {
	logger := New(Config{
		Level:       LevelInfo, // Debug disabled
		Format:      FormatJSON,
		Output:      NewOutput(io.Discard),
		AddSource:   false,
		ServiceName: "benchmark",
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Debug("benchmark debug message",
			"key1", "value1",
			"key2", 42,
		)
	}
}

// BenchmarkLoggerError benchmarks Error level logging
func BenchmarkLoggerError(b *testing.B) {
	logger := New(Config{
		Level:       LevelError,
		Format:      FormatJSON,
		Output:      NewOutput(io.Discard),
		AddSource:   false,
		ServiceName: "benchmark",
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Error("benchmark error message",
			"key1", "value1",
			"error", "test error",
		)
	}
}

// BenchmarkLoggerFormatText benchmarks text format logging
func BenchmarkLoggerFormatText(b *testing.B) {
	logger := New(Config{
		Level:       LevelInfo,
		Format:      FormatText,
		Output:      NewOutput(io.Discard),
		AddSource:   false,
		ServiceName: "benchmark",
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			"key1", "value1",
			"key2", 42,
		)
	}
}

// BenchmarkLoggerParallel benchmarks concurrent logging
func BenchmarkLoggerParallel(b *testing.B) {
	logger := New(Config{
		Level:       LevelInfo,
		Format:      FormatJSON,
		Output:      NewOutput(io.Discard),
		AddSource:   false,
		ServiceName: "benchmark",
	})

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("parallel benchmark message",
				"key1", "value1",
				"key2", 42,
			)
		}
	})
}

// BenchmarkLoggerWithManyFields benchmarks logging with many fields
func BenchmarkLoggerWithManyFields(b *testing.B) {
	logger := New(Config{
		Level:       LevelInfo,
		Format:      FormatJSON,
		Output:      NewOutput(io.Discard),
		AddSource:   false,
		ServiceName: "benchmark",
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message with many fields",
			"key1", "value1",
			"key2", 42,
			"key3", true,
			"key4", "value4",
			"key5", 123.456,
			"key6", "value6",
			"key7", []string{"a", "b", "c"},
			"key8", "value8",
		)
	}
}
