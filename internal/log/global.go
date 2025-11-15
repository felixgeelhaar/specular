package log

import (
	"sync"
)

var (
	defaultLogger *Logger
	loggerMu      sync.RWMutex
)

// SetDefaultLogger sets the process-wide default logger.
func SetDefaultLogger(logger *Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	defaultLogger = logger
}

// DefaultLogger returns the process-wide default logger.
// If none was configured, it falls back to a basic logger.
func DefaultLogger() *Logger {
	loggerMu.RLock()
	if defaultLogger != nil {
		defer loggerMu.RUnlock()
		return defaultLogger
	}
	loggerMu.RUnlock()

	// Initialize lazily with standard defaults.
	logger := Default()
	SetDefaultLogger(logger)
	return logger
}
