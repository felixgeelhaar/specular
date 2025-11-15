package log

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/errors"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "default config",
			config: DefaultConfig(),
		},
		{
			name:   "development config",
			config: DevelopmentConfig(),
		},
		{
			name:   "production config",
			config: ProductionConfig(),
		},
		{
			name: "custom config json",
			config: Config{
				Level:     LevelDebug,
				Format:    FormatJSON,
				Output:    OutputStdout(),
				AddSource: true,
			},
		},
		{
			name: "custom config text",
			config: Config{
				Level:     LevelWarn,
				Format:    FormatText,
				Output:    OutputStderr(),
				AddSource: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			if logger == nil {
				t.Fatal("expected logger, got nil")
			}
			if logger.slog == nil {
				t.Fatal("expected slog logger, got nil")
			}
			if logger.config.Level != tt.config.Level {
				t.Errorf("expected level %v, got %v", tt.config.Level, logger.config.Level)
			}
		})
	}
}

func TestDefaultConstructors(t *testing.T) {
	tests := []struct {
		name     string
		newFunc  func() *Logger
		wantFunc func(Config) bool
	}{
		{
			name:    "Default",
			newFunc: Default,
			wantFunc: func(c Config) bool {
				return c.Level == LevelInfo && c.Format == FormatJSON
			},
		},
		{
			name:    "Development",
			newFunc: Development,
			wantFunc: func(c Config) bool {
				return c.Level == LevelDebug && c.Format == FormatText && c.AddSource
			},
		},
		{
			name:    "Production",
			newFunc: Production,
			wantFunc: func(c Config) bool {
				return c.Level == LevelInfo && c.Format == FormatJSON && !c.AddSource
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := tt.newFunc()
			if logger == nil {
				t.Fatal("expected logger, got nil")
			}
			if !tt.wantFunc(logger.config) {
				t.Errorf("unexpected config: %+v", logger.config)
			}
		})
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelWarn,
		Format:    FormatJSON,
		Output:    NewOutput(&buf),
		AddSource: false,
	}
	logger := New(config)

	// Debug and Info should be filtered out
	logger.Debug("debug message")
	logger.Info("info message")

	if buf.Len() > 0 {
		t.Errorf("expected no output for debug/info at warn level, got: %s", buf.String())
	}

	// Warn should be logged
	logger.Warn("warn message")
	if buf.Len() == 0 {
		t.Error("expected output for warn message")
	}

	buf.Reset()

	// Error should be logged
	logger.Error("error message")
	if buf.Len() == 0 {
		t.Error("expected output for error message")
	}
}

func TestJSONFormatOutput(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    NewOutput(&buf),
		AddSource: false,
	}
	logger := New(config)

	logger.Info("test message", "key1", "value1", "key2", 42)

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
	}

	// Verify required fields
	if logEntry["msg"] != "test message" {
		t.Errorf("expected msg 'test message', got %v", logEntry["msg"])
	}
	if logEntry["level"] != "INFO" {
		t.Errorf("expected level 'INFO', got %v", logEntry["level"])
	}
	if logEntry["key1"] != "value1" {
		t.Errorf("expected key1 'value1', got %v", logEntry["key1"])
	}
	if logEntry["key2"] != float64(42) { // JSON numbers are float64
		t.Errorf("expected key2 42, got %v", logEntry["key2"])
	}
	if _, ok := logEntry["time"]; !ok {
		t.Error("expected time field in JSON output")
	}
}

func TestTextFormatOutput(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelInfo,
		Format:    FormatText,
		Output:    NewOutput(&buf),
		AddSource: false,
	}
	logger := New(config)

	logger.Info("test message", "key1", "value1")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("expected output to contain 'key1=value1', got: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("expected output to contain 'INFO', got: %s", output)
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    NewOutput(&buf),
		AddSource: false,
	}
	logger := New(config)

	// Create logger with additional fields
	loggerWithFields := logger.With("request_id", "123", "user_id", "456")

	loggerWithFields.Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if logEntry["request_id"] != "123" {
		t.Errorf("expected request_id '123', got %v", logEntry["request_id"])
	}
	if logEntry["user_id"] != "456" {
		t.Errorf("expected user_id '456', got %v", logEntry["user_id"])
	}
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    NewOutput(&buf),
		AddSource: false,
	}
	logger := New(config)

	loggerWithGroup := logger.WithGroup("request")
	loggerWithGroup.Info("test message", "id", "123")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Grouped attributes should be nested
	if request, ok := logEntry["request"].(map[string]interface{}); ok {
		if request["id"] != "123" {
			t.Errorf("expected request.id '123', got %v", request["id"])
		}
	} else {
		t.Error("expected 'request' group in output")
	}
}

func TestWithError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCode  bool
		wantSuggs bool
	}{
		{
			name:      "nil error",
			err:       nil,
			wantCode:  false,
			wantSuggs: false,
		},
		{
			name:      "regular error",
			err:       errors.New("SPEC-001", "test error"),
			wantCode:  true,
			wantSuggs: false,
		},
		{
			name: "error with suggestions",
			err: errors.New("SPEC-001", "test error").
				WithSuggestion("Try this"),
			wantCode:  true,
			wantSuggs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := Config{
				Level:     LevelInfo,
				Format:    FormatJSON,
				Output:    NewOutput(&buf),
				AddSource: false,
			}
			logger := New(config)

			loggerWithError := logger.WithError(tt.err)
			loggerWithError.Info("test")

			if tt.err == nil {
				// No error fields should be added
				var logEntry map[string]interface{}
				if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if _, ok := logEntry["error"]; ok {
					t.Error("expected no error field for nil error")
				}
				return
			}

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			if tt.wantCode {
				if _, ok := logEntry["error_code"]; !ok {
					t.Error("expected error_code field")
				}
			}

			if tt.wantSuggs {
				if _, ok := logEntry["suggestions"]; !ok {
					t.Error("expected suggestions field")
				}
			}
		})
	}
}

func TestLogError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantCode     string
		wantMessage  string
		wantSuggs    bool
		wantDocs     bool
		wantCause    bool
	}{
		{
			name:        "basic error",
			err:         errors.New("SPEC-001", "spec not found"),
			wantCode:    "SPEC-001",
			wantMessage: "spec not found",
		},
		{
			name: "error with suggestions",
			err: errors.New("SPEC-001", "spec not found").
				WithSuggestion("Run specular spec init"),
			wantCode:    "SPEC-001",
			wantMessage: "spec not found",
			wantSuggs:   true,
		},
		{
			name: "error with docs",
			err: errors.New("SPEC-001", "spec not found").
				WithDocs("https://docs.specular.dev/spec"),
			wantCode:    "SPEC-001",
			wantMessage: "spec not found",
			wantDocs:    true,
		},
		{
			name:        "error with cause",
			err:         errors.Wrap("SPEC-002", "failed to parse spec", errors.New("IO-001", "file not found")),
			wantCode:    "SPEC-002",
			wantMessage: "failed to parse spec",
			wantCause:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := Config{
				Level:     LevelInfo,
				Format:    FormatJSON,
				Output:    NewOutput(&buf),
				AddSource: false,
			}
			logger := New(config)

			logger.LogError(tt.err)

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			if logEntry["error_code"] != tt.wantCode {
				t.Errorf("expected error_code '%s', got %v", tt.wantCode, logEntry["error_code"])
			}

			if logEntry["error_message"] != tt.wantMessage {
				t.Errorf("expected error_message '%s', got %v", tt.wantMessage, logEntry["error_message"])
			}

			if tt.wantSuggs {
				if _, ok := logEntry["suggestions"]; !ok {
					t.Error("expected suggestions field")
				}
			}

			if tt.wantDocs {
				if _, ok := logEntry["docs_url"]; !ok {
					t.Error("expected docs_url field")
				}
			}

			if tt.wantCause {
				if _, ok := logEntry["cause"]; !ok {
					t.Error("expected cause field")
				}
			}
		})
	}
}

func TestLogErrorContext(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    NewOutput(&buf),
		AddSource: false,
	}
	logger := New(config)

	ctx := context.Background()
	err := errors.New("SPEC-001", "test error")

	logger.LogErrorContext(ctx, err)

	var logEntry map[string]interface{}
	if jsonErr := json.Unmarshal(buf.Bytes(), &logEntry); jsonErr != nil {
		t.Fatalf("failed to parse JSON: %v", jsonErr)
	}

	if logEntry["error_code"] != "SPEC-001" {
		t.Errorf("expected error_code 'SPEC-001', got %v", logEntry["error_code"])
	}
}

func TestContextMethods(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    NewOutput(&buf),
		AddSource: false,
	}
	logger := New(config)

	ctx := context.Background()

	tests := []struct {
		name   string
		logFn  func()
		level  string
	}{
		{
			name:  "DebugContext",
			logFn: func() { logger.DebugContext(ctx, "debug msg") },
			level: "DEBUG",
		},
		{
			name:  "InfoContext",
			logFn: func() { logger.InfoContext(ctx, "info msg") },
			level: "INFO",
		},
		{
			name:  "WarnContext",
			logFn: func() { logger.WarnContext(ctx, "warn msg") },
			level: "WARN",
		},
		{
			name:  "ErrorContext",
			logFn: func() { logger.ErrorContext(ctx, "error msg") },
			level: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFn()

			if tt.level == "DEBUG" {
				// Debug is below INFO level, should be filtered
				if buf.Len() > 0 {
					t.Errorf("expected no output for debug at info level, got: %s", buf.String())
				}
				return
			}

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
			}

			if logEntry["level"] != tt.level {
				t.Errorf("expected level '%s', got %v", tt.level, logEntry["level"])
			}
		})
	}
}

func TestEnabled(t *testing.T) {
	logger := New(Config{
		Level:  LevelWarn,
		Format: FormatJSON,
		Output: OutputStdout(),
	})

	ctx := context.Background()

	tests := []struct {
		level Level
		want  bool
	}{
		{LevelDebug, false},
		{LevelInfo, false},
		{LevelWarn, true},
		{LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			got := logger.Enabled(ctx, tt.level)
			if got != tt.want {
				t.Errorf("Enabled(%v) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	config := Config{
		Level:          LevelDebug,
		Format:         FormatJSON,
		Output:         OutputStdout(),
		AddSource:      true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	logger := New(config)
	gotConfig := logger.Config()

	if gotConfig.Level != config.Level {
		t.Errorf("expected level %v, got %v", config.Level, gotConfig.Level)
	}
	if gotConfig.Format != config.Format {
		t.Errorf("expected format %v, got %v", config.Format, gotConfig.Format)
	}
	if gotConfig.AddSource != config.AddSource {
		t.Errorf("expected addSource %v, got %v", config.AddSource, gotConfig.AddSource)
	}
	if gotConfig.ServiceName != config.ServiceName {
		t.Errorf("expected serviceName %s, got %s", config.ServiceName, gotConfig.ServiceName)
	}
	if gotConfig.ServiceVersion != config.ServiceVersion {
		t.Errorf("expected serviceVersion %s, got %s", config.ServiceVersion, gotConfig.ServiceVersion)
	}
}

func TestHandler(t *testing.T) {
	logger := Default()
	if logger.Handler() == nil {
		t.Error("expected non-nil handler")
	}
}

func TestWithContext(t *testing.T) {
	logger := Default()
	ctx := context.Background()

	loggerWithCtx := logger.WithContext(ctx)
	if loggerWithCtx == nil {
		t.Error("expected non-nil logger")
	}
}

func TestLogErrorWithRegularError(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    NewOutput(&buf),
		AddSource: false,
	}
	logger := New(config)

	// Test with regular error (not SpecularError)
	regularErr := errors.New("SPEC-001", "test error")
	logger.LogError(regularErr)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if logEntry["error_code"] != "SPEC-001" {
		t.Errorf("expected error_code 'SPEC-001', got %v", logEntry["error_code"])
	}
}

func TestLogErrorWithNil(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    NewOutput(&buf),
		AddSource: false,
	}
	logger := New(config)

	// Should not panic or log anything
	logger.LogError(nil)

	if buf.Len() > 0 {
		t.Errorf("expected no output for nil error, got: %s", buf.String())
	}
}

func TestLogErrorContextWithNil(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    NewOutput(&buf),
		AddSource: false,
	}
	logger := New(config)

	ctx := context.Background()

	// Should not panic or log anything
	logger.LogErrorContext(ctx, nil)

	if buf.Len() > 0 {
		t.Errorf("expected no output for nil error, got: %s", buf.String())
	}
}

func TestNewWithInvalidFormat(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     LevelInfo,
		Format:    Format(999), // Invalid format
		Output:    NewOutput(&buf),
		AddSource: false,
	}

	logger := New(config)
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}

	// Should default to JSON format
	logger.Info("test")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON (should default to JSON format): %v", err)
	}
}
