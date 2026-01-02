package logexport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// LokiConfig holds Grafana Loki configuration.
type LokiConfig struct {
	// URL is the Loki push endpoint URL
	URL string `yaml:"url,omitempty"`

	// TenantID is the X-Scope-OrgID header value for multi-tenant Loki
	TenantID string `yaml:"tenant_id,omitempty"`

	// Username for basic authentication
	Username string `yaml:"username,omitempty"`

	// Password for basic authentication
	Password string `yaml:"password,omitempty"`

	// Labels are static labels to add to all log streams
	Labels map[string]string `yaml:"labels,omitempty"`

	// Common configuration
	Config `yaml:",inline"`
}

// LokiExporter implements Exporter for Grafana Loki.
type LokiExporter struct {
	config     LokiConfig
	httpClient *http.Client
}

// lokiPushRequest represents the Loki push API request format.
type lokiPushRequest struct {
	Streams []lokiStream `json:"streams"`
}

// lokiStream represents a Loki log stream.
type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// NewLokiExporter creates a new Loki exporter.
func NewLokiExporter(config LokiConfig) (*LokiExporter, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("Loki URL is required")
	}

	if config.Labels == nil {
		config.Labels = make(map[string]string)
	}

	// Add default labels if not present
	if _, ok := config.Labels["app"]; !ok {
		config.Labels["app"] = "specular"
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &LokiExporter{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// Name returns the exporter name.
func (l *LokiExporter) Name() string {
	return "loki"
}

// Export sends log entries to Loki.
func (l *LokiExporter) Export(ctx context.Context, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Group entries by labels (stream)
	streams := l.groupByLabels(entries)

	payload := lokiPushRequest{
		Streams: streams,
	}

	return l.sendRequest(ctx, payload)
}

// Close closes the exporter.
func (l *LokiExporter) Close() error {
	return nil
}

// Healthy checks if Loki is reachable.
func (l *LokiExporter) Healthy(ctx context.Context) bool {
	// Use Loki's ready endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.config.URL+"/ready", nil)
	if err != nil {
		return false
	}

	l.setAuth(req)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// groupByLabels groups log entries into streams by their labels.
func (l *LokiExporter) groupByLabels(entries []LogEntry) []lokiStream {
	// For simplicity, group by level, service, and environment
	streamMap := make(map[string]*lokiStream)

	for _, entry := range entries {
		labels := l.buildLabels(entry)
		key := l.labelsToKey(labels)

		stream, exists := streamMap[key]
		if !exists {
			stream = &lokiStream{
				Stream: labels,
				Values: make([][]string, 0),
			}
			streamMap[key] = stream
		}

		// Format: [timestamp_ns, log_line]
		logLine := l.formatLogLine(entry)
		stream.Values = append(stream.Values, []string{
			strconv.FormatInt(entry.Timestamp.UnixNano(), 10),
			logLine,
		})
	}

	// Convert map to slice
	streams := make([]lokiStream, 0, len(streamMap))
	for _, stream := range streamMap {
		streams = append(streams, *stream)
	}

	return streams
}

// buildLabels builds labels for a log entry.
func (l *LokiExporter) buildLabels(entry LogEntry) map[string]string {
	labels := make(map[string]string)

	// Copy static labels
	for k, v := range l.config.Labels {
		labels[k] = v
	}

	// Add dynamic labels from entry
	labels["level"] = string(entry.Level)

	if entry.Service != "" {
		labels["service"] = entry.Service
	}
	if entry.Environment != "" {
		labels["env"] = entry.Environment
	}
	if entry.Logger != "" {
		labels["logger"] = entry.Logger
	}

	return labels
}

// labelsToKey creates a unique key from labels.
func (l *LokiExporter) labelsToKey(labels map[string]string) string {
	// Sort keys for consistent ordering
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var key string
	for _, k := range keys {
		key += k + "=" + labels[k] + ","
	}
	return key
}

// formatLogLine formats a log entry as a JSON log line.
func (l *LokiExporter) formatLogLine(entry LogEntry) string {
	logData := map[string]interface{}{
		"msg": entry.Message,
	}

	if entry.CorrelationID != "" {
		logData["correlation_id"] = entry.CorrelationID
	}
	if entry.TraceID != "" {
		logData["trace_id"] = entry.TraceID
	}
	if entry.SpanID != "" {
		logData["span_id"] = entry.SpanID
	}
	if entry.Source != "" {
		logData["source"] = entry.Source
	}
	if entry.Host != "" {
		logData["host"] = entry.Host
	}

	// Add attributes
	for k, v := range entry.Attributes {
		logData[k] = v
	}

	jsonBytes, err := json.Marshal(logData)
	if err != nil {
		return entry.Message
	}

	return string(jsonBytes)
}

// setAuth sets authentication headers on the request.
func (l *LokiExporter) setAuth(req *http.Request) {
	if l.config.Username != "" && l.config.Password != "" {
		req.SetBasicAuth(l.config.Username, l.config.Password)
	}
	if l.config.TenantID != "" {
		req.Header.Set("X-Scope-OrgID", l.config.TenantID)
	}
}

// sendRequest sends the push request to Loki.
func (l *LokiExporter) sendRequest(ctx context.Context, payload lokiPushRequest) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.config.URL+"/loki/api/v1/push", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	l.setAuth(req)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Loki error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
