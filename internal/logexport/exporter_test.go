package logexport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLogEntry(t *testing.T) {
	t.Run("builder pattern", func(t *testing.T) {
		entry := NewLogEntry(LogLevelInfo, "test message").
			WithCorrelationID("corr-123").
			WithTraceID("trace-456").
			WithSpanID("span-789").
			WithSource("test-source").
			WithService("test-service").
			WithEnvironment("test").
			WithAttribute("key", "value")

		if entry.Level != LogLevelInfo {
			t.Errorf("Level = %v, want %v", entry.Level, LogLevelInfo)
		}
		if entry.Message != "test message" {
			t.Errorf("Message = %v, want %v", entry.Message, "test message")
		}
		if entry.CorrelationID != "corr-123" {
			t.Errorf("CorrelationID = %v, want corr-123", entry.CorrelationID)
		}
		if entry.TraceID != "trace-456" {
			t.Errorf("TraceID = %v, want trace-456", entry.TraceID)
		}
		if entry.SpanID != "span-789" {
			t.Errorf("SpanID = %v, want span-789", entry.SpanID)
		}
		if entry.Source != "test-source" {
			t.Errorf("Source = %v, want test-source", entry.Source)
		}
		if entry.Service != "test-service" {
			t.Errorf("Service = %v, want test-service", entry.Service)
		}
		if entry.Environment != "test" {
			t.Errorf("Environment = %v, want test", entry.Environment)
		}
		if entry.Attributes["key"] != "value" {
			t.Errorf("Attributes[key] = %v, want value", entry.Attributes["key"])
		}
		if entry.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Enabled {
		t.Error("Enabled should be false")
	}
	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", cfg.BatchSize)
	}
	if cfg.FlushInterval != 5*time.Second {
		t.Errorf("FlushInterval = %v, want 5s", cfg.FlushInterval)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.RetryDelay != time.Second {
		t.Errorf("RetryDelay = %v, want 1s", cfg.RetryDelay)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
}

type mockExporter struct {
	name        string
	exportFunc  func(ctx context.Context, entries []LogEntry) error
	closeFunc   func() error
	healthyFunc func(ctx context.Context) bool
	exportCount atomic.Int32
}

func (m *mockExporter) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func (m *mockExporter) Export(ctx context.Context, entries []LogEntry) error {
	m.exportCount.Add(1)
	if m.exportFunc != nil {
		return m.exportFunc(ctx, entries)
	}
	return nil
}

func (m *mockExporter) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockExporter) Healthy(ctx context.Context) bool {
	if m.healthyFunc != nil {
		return m.healthyFunc(ctx)
	}
	return true
}

func TestBufferedExporter(t *testing.T) {
	t.Run("buffers entries", func(t *testing.T) {
		var mu sync.Mutex
		var exportedEntries []LogEntry

		mock := &mockExporter{
			exportFunc: func(ctx context.Context, entries []LogEntry) error {
				mu.Lock()
				exportedEntries = append(exportedEntries, entries...)
				mu.Unlock()
				return nil
			},
		}

		config := Config{
			BatchSize:     5,
			FlushInterval: 100 * time.Millisecond,
			Timeout:       5 * time.Second,
		}

		buffered := NewBufferedExporter(mock, config)
		defer buffered.Close()

		// Add entries below batch size
		ctx := context.Background()
		entries := []LogEntry{
			*NewLogEntry(LogLevelInfo, "msg1"),
			*NewLogEntry(LogLevelInfo, "msg2"),
		}

		if err := buffered.Export(ctx, entries); err != nil {
			t.Errorf("Export() error = %v", err)
		}

		// Wait for flush interval
		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		count := len(exportedEntries)
		mu.Unlock()

		if count != 2 {
			t.Errorf("exported %d entries, want 2", count)
		}
	})

	t.Run("flushes on batch size", func(t *testing.T) {
		var exported atomic.Int32

		mock := &mockExporter{
			exportFunc: func(ctx context.Context, entries []LogEntry) error {
				exported.Add(int32(len(entries)))
				return nil
			},
		}

		config := Config{
			BatchSize:     3,
			FlushInterval: time.Hour, // Long interval to not trigger
			Timeout:       5 * time.Second,
		}

		buffered := NewBufferedExporter(mock, config)
		defer buffered.Close()

		ctx := context.Background()
		for i := 0; i < 5; i++ {
			entries := []LogEntry{*NewLogEntry(LogLevelInfo, "msg")}
			buffered.Export(ctx, entries)
		}

		// Give time for flush to complete
		time.Sleep(50 * time.Millisecond)

		if count := exported.Load(); count < 3 {
			t.Errorf("exported %d entries, want at least 3", count)
		}
	})

	t.Run("name delegates to underlying exporter", func(t *testing.T) {
		mock := &mockExporter{name: "test-exporter"}
		buffered := NewBufferedExporter(mock, DefaultConfig())
		defer buffered.Close()

		if buffered.Name() != "test-exporter" {
			t.Errorf("Name() = %s, want test-exporter", buffered.Name())
		}
	})
}

func TestMultiExporter(t *testing.T) {
	t.Run("exports to all", func(t *testing.T) {
		var count1, count2 atomic.Int32

		mock1 := &mockExporter{
			name: "mock1",
			exportFunc: func(ctx context.Context, entries []LogEntry) error {
				count1.Add(int32(len(entries)))
				return nil
			},
		}
		mock2 := &mockExporter{
			name: "mock2",
			exportFunc: func(ctx context.Context, entries []LogEntry) error {
				count2.Add(int32(len(entries)))
				return nil
			},
		}

		multi := NewMultiExporter(mock1, mock2)

		entries := []LogEntry{
			*NewLogEntry(LogLevelInfo, "msg1"),
			*NewLogEntry(LogLevelInfo, "msg2"),
		}

		if err := multi.Export(context.Background(), entries); err != nil {
			t.Errorf("Export() error = %v", err)
		}

		if count1.Load() != 2 {
			t.Errorf("mock1 received %d entries, want 2", count1.Load())
		}
		if count2.Load() != 2 {
			t.Errorf("mock2 received %d entries, want 2", count2.Load())
		}
	})

	t.Run("healthy requires all", func(t *testing.T) {
		mock1 := &mockExporter{healthyFunc: func(ctx context.Context) bool { return true }}
		mock2 := &mockExporter{healthyFunc: func(ctx context.Context) bool { return false }}

		multi := NewMultiExporter(mock1, mock2)

		if multi.Healthy(context.Background()) {
			t.Error("Healthy() should return false when one exporter is unhealthy")
		}
	})

	t.Run("name returns multi", func(t *testing.T) {
		multi := NewMultiExporter()
		if multi.Name() != "multi" {
			t.Errorf("Name() = %s, want multi", multi.Name())
		}
	})
}

func TestELKExporter(t *testing.T) {
	t.Run("requires URL", func(t *testing.T) {
		_, err := NewELKExporter(ELKConfig{})
		if err == nil {
			t.Error("expected error for missing URL")
		}
	})

	t.Run("export to elasticsearch", func(t *testing.T) {
		var receivedBody []byte

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/_bulk" {
				t.Errorf("Path = %s, want /_bulk", r.URL.Path)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/x-ndjson" {
				t.Errorf("Content-Type = %s, want application/x-ndjson", ct)
			}

			body := make([]byte, r.ContentLength)
			r.Body.Read(body)
			receivedBody = body

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"errors":false,"items":[]}`))
		}))
		defer server.Close()

		exporter, err := NewELKExporter(ELKConfig{
			URL:   server.URL,
			Index: "test-logs",
		})
		if err != nil {
			t.Fatalf("NewELKExporter() error = %v", err)
		}

		entries := []LogEntry{
			*NewLogEntry(LogLevelInfo, "test message").WithService("test-svc"),
		}

		if err := exporter.Export(context.Background(), entries); err != nil {
			t.Errorf("Export() error = %v", err)
		}

		if len(receivedBody) == 0 {
			t.Error("no body received")
		}
	})

	t.Run("name", func(t *testing.T) {
		exporter, _ := NewELKExporter(ELKConfig{URL: "http://localhost:9200"})
		if exporter.Name() != "elk" {
			t.Errorf("Name() = %s, want elk", exporter.Name())
		}
	})

	t.Run("index date patterns", func(t *testing.T) {
		exporter, _ := NewELKExporter(ELKConfig{
			URL:   "http://localhost:9200",
			Index: "logs-%Y.%m.%d",
		})

		ts := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		index := exporter.resolveIndex(ts)

		if index != "logs-2024.06.15" {
			t.Errorf("resolveIndex() = %s, want logs-2024.06.15", index)
		}
	})
}

func TestSplunkExporter(t *testing.T) {
	t.Run("requires URL", func(t *testing.T) {
		_, err := NewSplunkExporter(SplunkConfig{Token: "test"})
		if err == nil {
			t.Error("expected error for missing URL")
		}
	})

	t.Run("requires token", func(t *testing.T) {
		_, err := NewSplunkExporter(SplunkConfig{URL: "http://localhost"})
		if err == nil {
			t.Error("expected error for missing token")
		}
	})

	t.Run("export to splunk", func(t *testing.T) {
		var receivedEvents []splunkEvent

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/services/collector/event" {
				t.Errorf("Path = %s, want /services/collector/event", r.URL.Path)
			}
			if auth := r.Header.Get("Authorization"); auth != "Splunk test-token" {
				t.Errorf("Authorization = %s, want Splunk test-token", auth)
			}

			var event splunkEvent
			json.NewDecoder(r.Body).Decode(&event)
			receivedEvents = append(receivedEvents, event)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"text":"Success","code":0}`))
		}))
		defer server.Close()

		exporter, err := NewSplunkExporter(SplunkConfig{
			URL:   server.URL,
			Token: "test-token",
			Index: "main",
		})
		if err != nil {
			t.Fatalf("NewSplunkExporter() error = %v", err)
		}

		entries := []LogEntry{
			*NewLogEntry(LogLevelError, "error message"),
		}

		if err := exporter.Export(context.Background(), entries); err != nil {
			t.Errorf("Export() error = %v", err)
		}

		if len(receivedEvents) != 1 {
			t.Errorf("received %d events, want 1", len(receivedEvents))
		}
	})

	t.Run("name", func(t *testing.T) {
		exporter, _ := NewSplunkExporter(SplunkConfig{URL: "http://localhost", Token: "test"})
		if exporter.Name() != "splunk" {
			t.Errorf("Name() = %s, want splunk", exporter.Name())
		}
	})
}

func TestLokiExporter(t *testing.T) {
	t.Run("requires URL", func(t *testing.T) {
		_, err := NewLokiExporter(LokiConfig{})
		if err == nil {
			t.Error("expected error for missing URL")
		}
	})

	t.Run("export to loki", func(t *testing.T) {
		var receivedPayload lokiPushRequest

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/loki/api/v1/push" {
				t.Errorf("Path = %s, want /loki/api/v1/push", r.URL.Path)
			}

			json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		exporter, err := NewLokiExporter(LokiConfig{
			URL: server.URL,
			Labels: map[string]string{
				"job": "specular",
			},
		})
		if err != nil {
			t.Fatalf("NewLokiExporter() error = %v", err)
		}

		entries := []LogEntry{
			*NewLogEntry(LogLevelInfo, "info message").WithService("api"),
			*NewLogEntry(LogLevelError, "error message").WithService("api"),
		}

		if err := exporter.Export(context.Background(), entries); err != nil {
			t.Errorf("Export() error = %v", err)
		}

		if len(receivedPayload.Streams) == 0 {
			t.Error("expected at least one stream")
		}
	})

	t.Run("groups by labels", func(t *testing.T) {
		exporter, _ := NewLokiExporter(LokiConfig{URL: "http://localhost"})

		entries := []LogEntry{
			*NewLogEntry(LogLevelInfo, "info1").WithService("svc1"),
			*NewLogEntry(LogLevelError, "error1").WithService("svc1"),
			*NewLogEntry(LogLevelInfo, "info2").WithService("svc2"),
		}

		streams := exporter.groupByLabels(entries)

		// Should have streams grouped by level and service
		if len(streams) < 2 {
			t.Errorf("expected at least 2 streams, got %d", len(streams))
		}
	})

	t.Run("name", func(t *testing.T) {
		exporter, _ := NewLokiExporter(LokiConfig{URL: "http://localhost"})
		if exporter.Name() != "loki" {
			t.Errorf("Name() = %s, want loki", exporter.Name())
		}
	})

	t.Run("sets tenant ID header", func(t *testing.T) {
		var receivedTenantID string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedTenantID = r.Header.Get("X-Scope-OrgID")
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		exporter, _ := NewLokiExporter(LokiConfig{
			URL:      server.URL,
			TenantID: "tenant-123",
		})

		entries := []LogEntry{*NewLogEntry(LogLevelInfo, "msg")}
		exporter.Export(context.Background(), entries)

		if receivedTenantID != "tenant-123" {
			t.Errorf("X-Scope-OrgID = %s, want tenant-123", receivedTenantID)
		}
	})
}
