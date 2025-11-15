package log

import (
	"testing"
)

func TestSetDefaultLogger(t *testing.T) {
	// Save original default logger to restore later
	originalLogger := defaultLogger
	defer func() {
		defaultLogger = originalLogger
	}()

	// Create a custom logger
	customLogger := Development()

	// Set it as default
	SetDefaultLogger(customLogger)

	// Verify it was set
	if defaultLogger != customLogger {
		t.Error("SetDefaultLogger did not set the default logger")
	}

	// Verify we can retrieve it
	retrieved := DefaultLogger()
	if retrieved != customLogger {
		t.Error("DefaultLogger did not return the custom logger")
	}
}

func TestDefaultLogger(t *testing.T) {
	// Save original default logger to restore later
	originalLogger := defaultLogger
	defer func() {
		defaultLogger = originalLogger
	}()

	t.Run("returns existing logger", func(t *testing.T) {
		// Set a custom logger
		customLogger := Production()
		SetDefaultLogger(customLogger)

		// DefaultLogger should return the set logger
		logger := DefaultLogger()
		if logger != customLogger {
			t.Error("DefaultLogger did not return the existing logger")
		}
	})

	t.Run("creates new logger when nil", func(t *testing.T) {
		// Reset default logger to nil
		defaultLogger = nil

		// DefaultLogger should create and return a new logger
		logger := DefaultLogger()
		if logger == nil {
			t.Fatal("DefaultLogger returned nil when no default was set")
		}

		// Verify it's now set as the default
		if defaultLogger != logger {
			t.Error("DefaultLogger did not set itself as the default")
		}

		// Subsequent calls should return the same logger
		logger2 := DefaultLogger()
		if logger2 != logger {
			t.Error("DefaultLogger did not return the same logger on second call")
		}
	})
}

func TestDefaultLoggerConcurrency(t *testing.T) {
	// Save original default logger to restore later
	originalLogger := defaultLogger
	defer func() {
		defaultLogger = originalLogger
	}()

	// Reset to nil
	defaultLogger = nil

	// Call DefaultLogger concurrently
	const goroutines = 100
	loggers := make([]*Logger, goroutines)
	done := make(chan bool)

	for i := 0; i < goroutines; i++ {
		go func(index int) {
			loggers[index] = DefaultLogger()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// All loggers should be the same instance
	firstLogger := loggers[0]
	for i := 1; i < goroutines; i++ {
		if loggers[i] != firstLogger {
			t.Errorf("Logger at index %d is different from the first logger", i)
		}
	}
}
