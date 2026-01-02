package logexport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ELKConfig holds Elasticsearch-specific configuration.
type ELKConfig struct {
	// URL is the Elasticsearch endpoint URL
	URL string `yaml:"url,omitempty"`

	// Index is the index name pattern (supports date patterns like logs-%Y.%m.%d)
	Index string `yaml:"index,omitempty"`

	// Username for basic authentication
	Username string `yaml:"username,omitempty"`

	// Password for basic authentication
	Password string `yaml:"password,omitempty"`

	// APIKey for API key authentication
	APIKey string `yaml:"api_key,omitempty"`

	// CloudID for Elastic Cloud
	CloudID string `yaml:"cloud_id,omitempty"`

	// Common configuration
	Config `yaml:",inline"`
}

// ELKExporter implements Exporter for Elasticsearch/ELK.
type ELKExporter struct {
	config     ELKConfig
	httpClient *http.Client
}

// elkDocument represents a document in Elasticsearch format.
type elkDocument struct {
	Timestamp     string                 `json:"@timestamp"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	Logger        string                 `json:"logger,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	SpanID        string                 `json:"span_id,omitempty"`
	Source        string                 `json:"source,omitempty"`
	Host          string                 `json:"host,omitempty"`
	Service       string                 `json:"service,omitempty"`
	Environment   string                 `json:"environment,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
}

// NewELKExporter creates a new Elasticsearch exporter.
func NewELKExporter(config ELKConfig) (*ELKExporter, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("elasticsearch URL is required")
	}

	if config.Index == "" {
		config.Index = "specular-logs"
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &ELKExporter{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// Name returns the exporter name.
func (e *ELKExporter) Name() string {
	return "elk"
}

// Export sends log entries to Elasticsearch.
func (e *ELKExporter) Export(ctx context.Context, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Use bulk API for efficiency
	bulk, err := e.buildBulkRequest(entries)
	if err != nil {
		return fmt.Errorf("failed to build bulk request: %w", err)
	}

	return e.sendBulkRequest(ctx, bulk)
}

// Close closes the exporter.
func (e *ELKExporter) Close() error {
	return nil
}

// Healthy checks if Elasticsearch is reachable.
func (e *ELKExporter) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.config.URL, nil)
	if err != nil {
		return false
	}

	e.setAuth(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// buildBulkRequest builds the Elasticsearch bulk request body.
func (e *ELKExporter) buildBulkRequest(entries []LogEntry) ([]byte, error) {
	var buf bytes.Buffer

	for _, entry := range entries {
		// Action line
		index := e.resolveIndex(entry.Timestamp)
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index,
			},
		}
		actionBytes, err := json.Marshal(action)
		if err != nil {
			return nil, err
		}
		buf.Write(actionBytes)
		buf.WriteByte('\n')

		// Document line
		doc := e.entryToDocument(entry)
		docBytes, err := json.Marshal(doc)
		if err != nil {
			return nil, err
		}
		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

// sendBulkRequest sends the bulk request to Elasticsearch.
func (e *ELKExporter) sendBulkRequest(ctx context.Context, body []byte) error {
	url := strings.TrimSuffix(e.config.URL, "/") + "/_bulk"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")
	e.setAuth(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("elasticsearch error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Check for errors in the bulk response
	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []struct {
			Index struct {
				Error struct {
					Type   string `json:"type"`
					Reason string `json:"reason"`
				} `json:"error,omitempty"`
			} `json:"index"`
		} `json:"items"`
	}

	if unmarshalErr := json.Unmarshal(respBody, &bulkResp); unmarshalErr != nil {
		return nil // Assume success if we can't parse
	}

	if bulkResp.Errors {
		var errMsgs []string
		for _, item := range bulkResp.Items {
			if item.Index.Error.Type != "" {
				errMsgs = append(errMsgs, item.Index.Error.Reason)
			}
		}
		if len(errMsgs) > 0 {
			return fmt.Errorf("bulk errors: %s", strings.Join(errMsgs, "; "))
		}
	}

	return nil
}

// setAuth sets authentication headers on the request.
func (e *ELKExporter) setAuth(req *http.Request) {
	if e.config.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+e.config.APIKey)
	} else if e.config.Username != "" && e.config.Password != "" {
		req.SetBasicAuth(e.config.Username, e.config.Password)
	}
}

// resolveIndex resolves the index name with date patterns.
func (e *ELKExporter) resolveIndex(timestamp time.Time) string {
	index := e.config.Index

	// Replace date patterns
	index = strings.ReplaceAll(index, "%Y", timestamp.Format("2006"))
	index = strings.ReplaceAll(index, "%m", timestamp.Format("01"))
	index = strings.ReplaceAll(index, "%d", timestamp.Format("02"))

	return index
}

// entryToDocument converts a LogEntry to an Elasticsearch document.
func (e *ELKExporter) entryToDocument(entry LogEntry) elkDocument {
	return elkDocument{
		Timestamp:     entry.Timestamp.Format(time.RFC3339Nano),
		Level:         string(entry.Level),
		Message:       entry.Message,
		Logger:        entry.Logger,
		CorrelationID: entry.CorrelationID,
		TraceID:       entry.TraceID,
		SpanID:        entry.SpanID,
		Source:        entry.Source,
		Host:          entry.Host,
		Service:       entry.Service,
		Environment:   entry.Environment,
		Attributes:    entry.Attributes,
	}
}
