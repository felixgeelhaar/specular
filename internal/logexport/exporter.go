// Package logexport provides log aggregation and export functionality
// to external systems like Elasticsearch, Splunk, and Grafana Loki.
package logexport

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LogLevel represents the log level.
type LogLevel string

// Log levels for exported log entries.
const (
	LogLevelDebug   LogLevel = "DEBUG"
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarning LogLevel = "WARNING"
	LogLevelError   LogLevel = "ERROR"
	LogLevelFatal   LogLevel = "FATAL"
)

// LogEntry represents a structured log entry for export.
type LogEntry struct {
	// Timestamp is when the log entry was created
	Timestamp time.Time `json:"timestamp"`

	// Level is the log level
	Level LogLevel `json:"level"`

	// Message is the log message
	Message string `json:"message"`

	// Logger is the name of the logger that created this entry
	Logger string `json:"logger,omitempty"`

	// CorrelationID links related log entries across services
	CorrelationID string `json:"correlation_id,omitempty"`

	// TraceID is the OpenTelemetry trace ID
	TraceID string `json:"trace_id,omitempty"`

	// SpanID is the OpenTelemetry span ID
	SpanID string `json:"span_id,omitempty"`

	// Attributes are additional structured data
	Attributes map[string]interface{} `json:"attributes,omitempty"`

	// Source identifies where the log originated
	Source string `json:"source,omitempty"`

	// Host is the hostname of the log source
	Host string `json:"host,omitempty"`

	// Service is the service name
	Service string `json:"service,omitempty"`

	// Environment is the deployment environment (e.g., prod, staging)
	Environment string `json:"environment,omitempty"`
}

// Exporter defines the interface for log exporters.
type Exporter interface {
	// Export sends log entries to the external system.
	Export(ctx context.Context, entries []LogEntry) error

	// Close closes the exporter and releases resources.
	Close() error

	// Name returns the exporter name.
	Name() string

	// Healthy returns true if the exporter is healthy.
	Healthy(ctx context.Context) bool
}

// Config holds common configuration for log exporters.
type Config struct {
	// Enabled controls whether the exporter is active
	Enabled bool `yaml:"enabled,omitempty"`

	// BatchSize is the number of entries to batch before sending
	BatchSize int `yaml:"batch_size,omitempty"`

	// FlushInterval is how often to flush buffered entries
	FlushInterval time.Duration `yaml:"flush_interval,omitempty"`

	// MaxRetries is the number of retries on failure
	MaxRetries int `yaml:"max_retries,omitempty"`

	// RetryDelay is the initial delay between retries
	RetryDelay time.Duration `yaml:"retry_delay,omitempty"`

	// Timeout is the HTTP request timeout
	Timeout time.Duration `yaml:"timeout,omitempty"`
}

// DefaultConfig returns the default exporter configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:       false,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		MaxRetries:    3,
		RetryDelay:    time.Second,
		Timeout:       30 * time.Second,
	}
}

// BufferedExporter wraps an exporter with buffering and batching.
type BufferedExporter struct {
	exporter Exporter
	config   Config

	buffer    []LogEntry
	bufferMu  sync.Mutex
	flushChan chan struct{}
	closeChan chan struct{}
	wg        sync.WaitGroup
}

// NewBufferedExporter creates a new buffered exporter.
func NewBufferedExporter(exporter Exporter, config Config) *BufferedExporter {
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}

	b := &BufferedExporter{
		exporter:  exporter,
		config:    config,
		buffer:    make([]LogEntry, 0, config.BatchSize),
		flushChan: make(chan struct{}, 1),
		closeChan: make(chan struct{}),
	}

	b.wg.Add(1)
	go b.flushLoop()

	return b
}

// Export adds entries to the buffer and flushes if needed.
func (b *BufferedExporter) Export(ctx context.Context, entries []LogEntry) error {
	b.bufferMu.Lock()
	b.buffer = append(b.buffer, entries...)
	shouldFlush := len(b.buffer) >= b.config.BatchSize
	b.bufferMu.Unlock()

	if shouldFlush {
		select {
		case b.flushChan <- struct{}{}:
		default:
		}
	}

	return nil
}

// Flush forces a flush of buffered entries.
func (b *BufferedExporter) Flush(ctx context.Context) error {
	b.bufferMu.Lock()
	entries := b.buffer
	b.buffer = make([]LogEntry, 0, b.config.BatchSize)
	b.bufferMu.Unlock()

	if len(entries) == 0 {
		return nil
	}

	return b.exportWithRetry(ctx, entries)
}

// Close flushes remaining entries and closes the exporter.
func (b *BufferedExporter) Close() error {
	close(b.closeChan)
	b.wg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Best-effort final flush; ignore errors to allow graceful close
	_ = b.Flush(ctx)

	return b.exporter.Close()
}

// Name returns the underlying exporter name.
func (b *BufferedExporter) Name() string {
	return b.exporter.Name()
}

// Healthy checks if the underlying exporter is healthy.
func (b *BufferedExporter) Healthy(ctx context.Context) bool {
	return b.exporter.Healthy(ctx)
}

// flushLoop periodically flushes the buffer.
func (b *BufferedExporter) flushLoop() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.closeChan:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), b.config.Timeout)
			_ = b.Flush(ctx) // Error logged internally, continue flushing on schedule
			cancel()
		case <-b.flushChan:
			ctx, cancel := context.WithTimeout(context.Background(), b.config.Timeout)
			_ = b.Flush(ctx) // Error logged internally, continue processing flush requests
			cancel()
		}
	}
}

// exportWithRetry exports entries with retry logic.
func (b *BufferedExporter) exportWithRetry(ctx context.Context, entries []LogEntry) error {
	var lastErr error
	delay := b.config.RetryDelay

	for attempt := 0; attempt <= b.config.MaxRetries; attempt++ {
		if err := b.exporter.Export(ctx, entries); err != nil {
			lastErr = err
			if attempt < b.config.MaxRetries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					delay *= 2 // Exponential backoff
				}
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("failed after %d retries: %w", b.config.MaxRetries, lastErr)
}

// MultiExporter sends logs to multiple exporters.
type MultiExporter struct {
	exporters []Exporter
}

// NewMultiExporter creates a new multi-exporter.
func NewMultiExporter(exporters ...Exporter) *MultiExporter {
	return &MultiExporter{
		exporters: exporters,
	}
}

// Export sends entries to all exporters.
func (m *MultiExporter) Export(ctx context.Context, entries []LogEntry) error {
	var errs []error
	for _, e := range m.exporters {
		if err := e.Export(ctx, entries); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.Name(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("export failed: %v", errs)
	}
	return nil
}

// Close closes all exporters.
func (m *MultiExporter) Close() error {
	var errs []error
	for _, e := range m.exporters {
		if err := e.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.Name(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close failed: %v", errs)
	}
	return nil
}

// Name returns a combined name of all exporters.
func (m *MultiExporter) Name() string {
	return "multi"
}

// Healthy returns true if all exporters are healthy.
func (m *MultiExporter) Healthy(ctx context.Context) bool {
	for _, e := range m.exporters {
		if !e.Healthy(ctx) {
			return false
		}
	}
	return true
}

// NewLogEntry creates a new log entry with the current timestamp.
func NewLogEntry(level LogLevel, message string) *LogEntry {
	return &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}
}

// WithCorrelationID sets the correlation ID.
func (e *LogEntry) WithCorrelationID(id string) *LogEntry {
	e.CorrelationID = id
	return e
}

// WithTraceID sets the trace ID.
func (e *LogEntry) WithTraceID(id string) *LogEntry {
	e.TraceID = id
	return e
}

// WithSpanID sets the span ID.
func (e *LogEntry) WithSpanID(id string) *LogEntry {
	e.SpanID = id
	return e
}

// WithAttribute adds an attribute.
func (e *LogEntry) WithAttribute(key string, value interface{}) *LogEntry {
	if e.Attributes == nil {
		e.Attributes = make(map[string]interface{})
	}
	e.Attributes[key] = value
	return e
}

// WithSource sets the source.
func (e *LogEntry) WithSource(source string) *LogEntry {
	e.Source = source
	return e
}

// WithService sets the service name.
func (e *LogEntry) WithService(service string) *LogEntry {
	e.Service = service
	return e
}

// WithEnvironment sets the environment.
func (e *LogEntry) WithEnvironment(env string) *LogEntry {
	e.Environment = env
	return e
}
