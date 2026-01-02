package logexport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SplunkConfig holds Splunk HEC configuration.
type SplunkConfig struct {
	// URL is the Splunk HEC endpoint URL
	URL string `yaml:"url,omitempty"`

	// Token is the HEC token
	Token string `yaml:"token,omitempty"`

	// Index is the Splunk index name
	Index string `yaml:"index,omitempty"`

	// Source is the source value for events
	Source string `yaml:"source,omitempty"`

	// SourceType is the sourcetype value for events
	SourceType string `yaml:"sourcetype,omitempty"`

	// Host is the host value for events
	Host string `yaml:"host,omitempty"`

	// SkipTLSVerify skips TLS certificate verification
	SkipTLSVerify bool `yaml:"skip_tls_verify,omitempty"`

	// Common configuration
	Config `yaml:",inline"`
}

// SplunkExporter implements Exporter for Splunk HEC.
type SplunkExporter struct {
	config     SplunkConfig
	httpClient *http.Client
}

// splunkEvent represents a Splunk HEC event.
type splunkEvent struct {
	Time       float64                `json:"time,omitempty"`
	Host       string                 `json:"host,omitempty"`
	Source     string                 `json:"source,omitempty"`
	SourceType string                 `json:"sourcetype,omitempty"`
	Index      string                 `json:"index,omitempty"`
	Event      map[string]interface{} `json:"event"`
}

// NewSplunkExporter creates a new Splunk HEC exporter.
func NewSplunkExporter(config SplunkConfig) (*SplunkExporter, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("Splunk HEC URL is required")
	}

	if config.Token == "" {
		return nil, fmt.Errorf("Splunk HEC token is required")
	}

	if config.Source == "" {
		config.Source = "specular"
	}

	if config.SourceType == "" {
		config.SourceType = "specular:logs"
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &SplunkExporter{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// Name returns the exporter name.
func (s *SplunkExporter) Name() string {
	return "splunk"
}

// Export sends log entries to Splunk HEC.
func (s *SplunkExporter) Export(ctx context.Context, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Build batch payload
	var buf bytes.Buffer
	for _, entry := range entries {
		event := s.entryToEvent(entry)
		eventBytes, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}
		buf.Write(eventBytes)
	}

	return s.sendRequest(ctx, buf.Bytes())
}

// Close closes the exporter.
func (s *SplunkExporter) Close() error {
	return nil
}

// Healthy checks if Splunk HEC is reachable.
func (s *SplunkExporter) Healthy(ctx context.Context) bool {
	// Send a health check event
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.config.URL+"/services/collector/health/1.0", nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Splunk "+s.config.Token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// entryToEvent converts a LogEntry to a Splunk event.
func (s *SplunkExporter) entryToEvent(entry LogEntry) *splunkEvent {
	eventData := map[string]interface{}{
		"level":   string(entry.Level),
		"message": entry.Message,
	}

	if entry.Logger != "" {
		eventData["logger"] = entry.Logger
	}
	if entry.CorrelationID != "" {
		eventData["correlation_id"] = entry.CorrelationID
	}
	if entry.TraceID != "" {
		eventData["trace_id"] = entry.TraceID
	}
	if entry.SpanID != "" {
		eventData["span_id"] = entry.SpanID
	}
	if entry.Source != "" {
		eventData["source_component"] = entry.Source
	}
	if entry.Service != "" {
		eventData["service"] = entry.Service
	}
	if entry.Environment != "" {
		eventData["environment"] = entry.Environment
	}
	for k, v := range entry.Attributes {
		eventData[k] = v
	}

	host := s.config.Host
	if entry.Host != "" {
		host = entry.Host
	}

	event := &splunkEvent{
		Time:       float64(entry.Timestamp.UnixNano()) / float64(time.Second),
		Host:       host,
		Source:     s.config.Source,
		SourceType: s.config.SourceType,
		Event:      eventData,
	}

	if s.config.Index != "" {
		event.Index = s.config.Index
	}

	return event
}

// sendRequest sends the request to Splunk HEC.
func (s *SplunkExporter) sendRequest(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.URL+"/services/collector/event", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Splunk "+s.config.Token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Splunk HEC error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Check response for errors
	var hecResp struct {
		Text string `json:"text"`
		Code int    `json:"code"`
	}

	if err := json.Unmarshal(respBody, &hecResp); err != nil {
		return nil // Assume success if we can't parse
	}

	if hecResp.Code != 0 {
		return fmt.Errorf("Splunk HEC error (code %d): %s", hecResp.Code, hecResp.Text)
	}

	return nil
}
