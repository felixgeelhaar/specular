package alerting

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WebhookConfig holds webhook-specific configuration.
type WebhookConfig struct {
	// URL is the webhook endpoint URL
	URL string

	// Method is the HTTP method (defaults to POST)
	Method string

	// Headers are custom headers to include in requests
	Headers map[string]string

	// Secret is used for HMAC signature generation (optional)
	Secret string

	// SignatureHeader is the header name for the signature (defaults to X-Signature-256)
	SignatureHeader string

	// PayloadTemplate is a custom JSON template (optional)
	// If not set, the default Alert JSON is used
	PayloadTemplate string

	// Common configuration
	Config
}

// WebhookManager implements AlertManager for generic webhooks.
type WebhookManager struct {
	config     WebhookConfig
	httpClient *http.Client
}

// webhookPayload represents the default webhook payload format.
type webhookPayload struct {
	Event     string        `json:"event"`
	Alert     *webhookAlert `json:"alert,omitempty"`
	DedupeKey string        `json:"dedupe_key,omitempty"`
	Timestamp string        `json:"timestamp"`
	Source    string        `json:"source"`
}

// webhookAlert represents the alert data in webhook format.
type webhookAlert struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	DedupeKey   string            `json:"dedupe_key"`
	Source      string            `json:"source"`
	Labels      map[string]string `json:"labels,omitempty"`
	Links       []AlertLink       `json:"links,omitempty"`
	Timestamp   string            `json:"timestamp"`
}

// NewWebhookManager creates a new webhook alert manager.
func NewWebhookManager(config WebhookConfig) (*WebhookManager, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	if config.Method == "" {
		config.Method = http.MethodPost
	}

	if config.SignatureHeader == "" {
		config.SignatureHeader = "X-Signature-256"
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &WebhookManager{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// Name returns the name of the alert manager.
func (m *WebhookManager) Name() string {
	return "webhook"
}

// Send sends an alert via webhook.
func (m *WebhookManager) Send(ctx context.Context, alert *Alert) error {
	payload := m.buildTriggerPayload(alert)
	return m.sendRequest(ctx, payload)
}

// Resolve sends a resolve event via webhook.
func (m *WebhookManager) Resolve(ctx context.Context, dedupeKey string) error {
	payload := &webhookPayload{
		Event:     "resolve",
		DedupeKey: dedupeKey,
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    "specular",
	}
	return m.sendRequest(ctx, payload)
}

// Test sends a test event to verify connectivity.
func (m *WebhookManager) Test(ctx context.Context) error {
	alert := NewAlert(
		"Specular Test Alert",
		"This is a test alert from Specular to verify webhook connectivity.",
		SeverityInfo,
	).WithDedupeKey("specular-test-alert").
		WithLabel("type", "test")

	return m.Send(ctx, alert)
}

// buildTriggerPayload builds the webhook trigger payload.
func (m *WebhookManager) buildTriggerPayload(alert *Alert) *webhookPayload {
	return &webhookPayload{
		Event: "trigger",
		Alert: &webhookAlert{
			ID:          alert.ID,
			Title:       alert.Title,
			Description: alert.Description,
			Severity:    string(alert.Severity),
			DedupeKey:   alert.DedupeKey,
			Source:      alert.Source,
			Labels:      alert.Labels,
			Links:       alert.Links,
			Timestamp:   alert.Timestamp.Format(time.RFC3339),
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    "specular",
	}
}

// sendRequest sends an HTTP request to the webhook endpoint.
func (m *WebhookManager) sendRequest(ctx context.Context, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, m.config.Method, m.config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Specular/1.0")

	// Add custom headers
	for k, v := range m.config.Headers {
		req.Header.Set(k, v)
	}

	// Add HMAC signature if secret is configured
	if m.config.Secret != "" {
		signature := m.computeSignature(body)
		req.Header.Set(m.config.SignatureHeader, "sha256="+signature)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read webhook response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// computeSignature computes the HMAC-SHA256 signature for the payload.
func (m *WebhookManager) computeSignature(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(m.config.Secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies an incoming webhook signature.
// This is useful for webhook receivers to validate requests.
func VerifySignature(payload []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
