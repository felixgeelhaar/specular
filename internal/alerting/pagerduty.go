package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	pagerdutyDefaultURL = "https://events.pagerduty.com/v2/enqueue"
)

// PagerDutyConfig holds PagerDuty-specific configuration.
type PagerDutyConfig struct {
	// RoutingKey is the PagerDuty Events API v2 routing key
	RoutingKey string

	// ServiceID is the optional PagerDuty service ID
	ServiceID string

	// URL is the PagerDuty Events API URL (defaults to v2 events API)
	URL string

	// Common configuration
	Config
}

// PagerDutyManager implements AlertManager for PagerDuty.
type PagerDutyManager struct {
	config     PagerDutyConfig
	httpClient *http.Client
}

// pagerdutyPayload represents the PagerDuty Events API v2 payload.
type pagerdutyPayload struct {
	RoutingKey  string                `json:"routing_key"`
	EventAction string                `json:"event_action"` // trigger, acknowledge, resolve
	DedupKey    string                `json:"dedup_key,omitempty"`
	Payload     *pagerdutyAlertDetail `json:"payload,omitempty"`
	Links       []pagerdutyLink       `json:"links,omitempty"`
}

// pagerdutyAlertDetail represents the alert details in PagerDuty format.
type pagerdutyAlertDetail struct {
	Summary       string                 `json:"summary"`
	Severity      string                 `json:"severity"`
	Source        string                 `json:"source"`
	Timestamp     string                 `json:"timestamp,omitempty"`
	Component     string                 `json:"component,omitempty"`
	Group         string                 `json:"group,omitempty"`
	Class         string                 `json:"class,omitempty"`
	CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}

// pagerdutyLink represents a link in PagerDuty format.
type pagerdutyLink struct {
	Href string `json:"href"`
	Text string `json:"text,omitempty"`
}

// pagerdutyResponse represents the PagerDuty API response.
type pagerdutyResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	DedupKey string `json:"dedup_key"`
}

// NewPagerDutyManager creates a new PagerDuty alert manager.
func NewPagerDutyManager(config PagerDutyConfig) (*PagerDutyManager, error) {
	if config.RoutingKey == "" {
		return nil, fmt.Errorf("PagerDuty routing key is required")
	}

	if config.URL == "" {
		config.URL = pagerdutyDefaultURL
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &PagerDutyManager{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// Name returns the name of the alert manager.
func (m *PagerDutyManager) Name() string {
	return "pagerduty"
}

// Send sends an alert to PagerDuty.
func (m *PagerDutyManager) Send(ctx context.Context, alert *Alert) error {
	payload := m.buildTriggerPayload(alert)
	return m.sendRequest(ctx, payload)
}

// Resolve resolves an alert in PagerDuty.
func (m *PagerDutyManager) Resolve(ctx context.Context, dedupeKey string) error {
	payload := &pagerdutyPayload{
		RoutingKey:  m.config.RoutingKey,
		EventAction: "resolve",
		DedupKey:    dedupeKey,
	}
	return m.sendRequest(ctx, payload)
}

// Test sends a test alert to verify connectivity.
func (m *PagerDutyManager) Test(ctx context.Context) error {
	alert := NewAlert(
		"Specular Test Alert",
		"This is a test alert from Specular to verify PagerDuty connectivity.",
		SeverityInfo,
	).WithDedupeKey("specular-test-alert").
		WithLabel("type", "test")

	if err := m.Send(ctx, alert); err != nil {
		return err
	}

	// Immediately resolve the test alert
	return m.Resolve(ctx, "specular-test-alert")
}

// buildTriggerPayload builds the PagerDuty trigger event payload.
func (m *PagerDutyManager) buildTriggerPayload(alert *Alert) *pagerdutyPayload {
	customDetails := make(map[string]interface{})
	for k, v := range alert.Labels {
		customDetails[k] = v
	}
	customDetails["alert_id"] = alert.ID
	customDetails["description"] = alert.Description

	var links []pagerdutyLink
	for _, l := range alert.Links {
		links = append(links, pagerdutyLink{
			Href: l.Href,
			Text: l.Text,
		})
	}

	dedupeKey := alert.DedupeKey
	if dedupeKey == "" {
		dedupeKey = alert.ID
	}

	return &pagerdutyPayload{
		RoutingKey:  m.config.RoutingKey,
		EventAction: "trigger",
		DedupKey:    dedupeKey,
		Payload: &pagerdutyAlertDetail{
			Summary:       alert.Title,
			Severity:      alert.Severity.ToPagerDutySeverity(),
			Source:        alert.Source,
			Timestamp:     alert.Timestamp.Format(time.RFC3339),
			Component:     "specular-cli",
			Class:         "slo-violation",
			CustomDetails: customDetails,
		},
		Links: links,
	}
}

// sendRequest sends an HTTP request to the PagerDuty API.
func (m *PagerDutyManager) sendRequest(ctx context.Context, payload *pagerdutyPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal PagerDuty payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create PagerDuty request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send PagerDuty request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read PagerDuty response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("PagerDuty API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var pdResp pagerdutyResponse
	if unmarshalErr := json.Unmarshal(respBody, &pdResp); unmarshalErr != nil {
		return fmt.Errorf("failed to parse PagerDuty response: %w", unmarshalErr)
	}

	if pdResp.Status != "success" {
		return fmt.Errorf("PagerDuty error: %s", pdResp.Message)
	}

	return nil
}
