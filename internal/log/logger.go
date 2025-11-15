package log

import (
	"context"
	"log/slog"

	"github.com/felixgeelhaar/specular/internal/errors"
)

// Logger provides structured logging with slog
type Logger struct {
	slog   *slog.Logger
	config Config
}

// New creates a new Logger with the given configuration
func New(config Config) *Logger {
	opts := &slog.HandlerOptions{
		Level:     config.Level.ToSlogLevel(),
		AddSource: config.AddSource,
	}

	var handler slog.Handler
	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(config.Output.Writer(), opts)
	case FormatText:
		handler = slog.NewTextHandler(config.Output.Writer(), opts)
	default:
		handler = slog.NewJSONHandler(config.Output.Writer(), opts)
	}

	return &Logger{
		slog:   slog.New(handler),
		config: config,
	}
}

// Default creates a logger with default configuration
func Default() *Logger {
	return New(DefaultConfig())
}

// Development creates a logger with development configuration
func Development() *Logger {
	return New(DevelopmentConfig())
}

// Production creates a logger with production configuration
func Production() *Logger {
	return New(ProductionConfig())
}

// With returns a new Logger with the given attributes added to all log entries
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		slog:   l.slog.With(args...),
		config: l.config,
	}
}

// WithGroup returns a new Logger with a group name that prefixes all attributes
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		slog:   l.slog.WithGroup(name),
		config: l.config,
	}
}

// WithError adds error details to the logger
// If the error is a SpecularError, it adds error_code and suggestions
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}

	if specErr, ok := err.(*errors.SpecularError); ok {
		args := []any{
			"error", specErr.Message,
			"error_code", string(specErr.Code),
		}

		if len(specErr.Suggestions) > 0 {
			args = append(args, "suggestions", specErr.Suggestions)
		}

		if specErr.Cause != nil {
			args = append(args, "cause", specErr.Cause.Error())
		}

		return l.With(args...)
	}

	return l.With("error", err.Error())
}

// WithContext returns a new Logger with context values added
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract common context values for correlation
	// This can be extended to extract trace IDs, user IDs, etc.
	return l
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

// DebugContext logs a debug message with context
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.slog.DebugContext(ctx, msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

// InfoContext logs an info message with context
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.slog.InfoContext(ctx, msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

// WarnContext logs a warning message with context
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.slog.WarnContext(ctx, msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

// ErrorContext logs an error message with context
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.slog.ErrorContext(ctx, msg, args...)
}

// LogError logs a SpecularError with full details
func (l *Logger) LogError(err error) {
	if err == nil {
		return
	}

	if specErr, ok := err.(*errors.SpecularError); ok {
		args := []any{
			"error_code", string(specErr.Code),
			"error_message", specErr.Message,
		}

		if len(specErr.Suggestions) > 0 {
			args = append(args, "suggestions", specErr.Suggestions)
		}

		if specErr.DocsURL != "" {
			args = append(args, "docs_url", specErr.DocsURL)
		}

		if specErr.Cause != nil {
			args = append(args, "cause", specErr.Cause.Error())
		}

		l.Error("operation failed", args...)
	} else {
		l.Error("operation failed", "error", err.Error())
	}
}

// LogErrorContext logs a SpecularError with full details and context
func (l *Logger) LogErrorContext(ctx context.Context, err error) {
	if err == nil {
		return
	}

	if specErr, ok := err.(*errors.SpecularError); ok {
		args := []any{
			"error_code", string(specErr.Code),
			"error_message", specErr.Message,
		}

		if len(specErr.Suggestions) > 0 {
			args = append(args, "suggestions", specErr.Suggestions)
		}

		if specErr.DocsURL != "" {
			args = append(args, "docs_url", specErr.DocsURL)
		}

		if specErr.Cause != nil {
			args = append(args, "cause", specErr.Cause.Error())
		}

		l.ErrorContext(ctx, "operation failed", args...)
	} else {
		l.ErrorContext(ctx, "operation failed", "error", err.Error())
	}
}

// Enabled returns whether the logger is enabled for the given level
func (l *Logger) Enabled(ctx context.Context, level Level) bool {
	return l.slog.Enabled(ctx, level.ToSlogLevel())
}

// Handler returns the underlying slog.Handler
func (l *Logger) Handler() slog.Handler {
	return l.slog.Handler()
}

// Config returns the logger configuration
func (l *Logger) Config() Config {
	return l.config
}
