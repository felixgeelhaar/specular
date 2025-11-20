package log

import (
	"sync"
)

var (
	defaultLogger *Logger
	loggerMu      sync.RWMutex
	loggerOnce    sync.Once
)

// SetDefaultLogger sets the process-wide default logger.
func SetDefaultLogger(logger *Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	defaultLogger = logger
}

// DefaultLogger returns the process-wide default logger.
// If none was configured, it falls back to a basic logger.
// Thread-safe lazy initialization using sync.Once.
func DefaultLogger() *Logger {
	// Use sync.Once to ensure lazy initialization happens exactly once
	loggerOnce.Do(func() {
		loggerMu.Lock()
		defer loggerMu.Unlock()
		if defaultLogger == nil {
			defaultLogger = Default()
		}
	})

	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return defaultLogger
}

// resetDefaultLogger resets the global logger state for testing.
// This function should only be used in tests.
func resetDefaultLogger() {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	defaultLogger = nil
	// Reset sync.Once by creating a new instance
	loggerOnce = sync.Once{}
}
