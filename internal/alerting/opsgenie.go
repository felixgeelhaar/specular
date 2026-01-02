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
	opsgenieUSURL = "https://api.opsgenie.com/v2/alerts"
	opsgenieEUURL = "https://api.eu.opsgenie.com/v2/alerts"
)

// OpsgenieConfig holds Opsgenie-specific configuration.
type OpsgenieConfig struct {
	// APIKey is the Opsgenie API key
	APIKey string

	// Region is the Opsgenie region ("us" or "eu")
	Region string

	// TeamID is the optional Opsgenie team ID
	TeamID string

	// Common configuration
	Config
}

// OpsgenieManager implements AlertManager for Opsgenie.
type OpsgenieManager struct {
	config     OpsgenieConfig
	httpClient *http.Client
	baseURL    string
}

// opsgenieCreateAlert represents the Opsgenie create alert request.
type opsgenieCreateAlert struct {
	Message     string              `json:"message"`
	Description string              `json:"description,omitempty"`
	Priority    string              `json:"priority,omitempty"`
	Alias       string              `json:"alias,omitempty"`
	Source      string              `json:"source,omitempty"`
	Entity      string              `json:"entity,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Details     map[string]string   `json:"details,omitempty"`
	Responders  []opsgenieResponder `json:"responders,omitempty"`
}

// opsgenieCloseAlert represents the Opsgenie close alert request.
type opsgenieCloseAlert struct {
	Source string `json:"source,omitempty"`
	Note   string `json:"note,omitempty"`
}

// opsgenieResponder represents an Opsgenie responder.
type opsgenieResponder struct {
	Type string `json:"type"` // "team", "user", "schedule"
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// opsgenieResponse represents the Opsgenie API response.
type opsgenieResponse struct {
	Result    string `json:"result"`
	RequestID string `json:"requestId"`
	Message   string `json:"message,omitempty"`
}

// NewOpsgenieManager creates a new Opsgenie alert manager.
func NewOpsgenieManager(config OpsgenieConfig) (*OpsgenieManager, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("opsgenie API key is required")
	}

	if config.Region == "" {
		config.Region = "us"
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	var baseURL string
	switch config.Region {
	case "eu":
		baseURL = opsgenieEUURL
	default:
		baseURL = opsgenieUSURL
	}

	return &OpsgenieManager{
		config:  config,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// Name returns the name of the alert manager.
func (m *OpsgenieManager) Name() string {
	return "opsgenie"
}

// Send sends an alert to Opsgenie.
func (m *OpsgenieManager) Send(ctx context.Context, alert *Alert) error {
	payload := m.buildCreatePayload(alert)
	return m.sendRequest(ctx, http.MethodPost, m.baseURL, payload)
}

// Resolve resolves an alert in Opsgenie.
func (m *OpsgenieManager) Resolve(ctx context.Context, dedupeKey string) error {
	url := fmt.Sprintf("%s/%s/close?identifierType=alias", m.baseURL, dedupeKey)
	payload := &opsgenieCloseAlert{
		Source: "specular",
		Note:   "Resolved by Specular",
	}
	return m.sendRequest(ctx, http.MethodPost, url, payload)
}

// Test sends a test alert to verify connectivity.
func (m *OpsgenieManager) Test(ctx context.Context) error {
	alert := NewAlert(
		"Specular Test Alert",
		"This is a test alert from Specular to verify Opsgenie connectivity.",
		SeverityInfo,
	).WithDedupeKey("specular-test-alert").
		WithLabel("type", "test")

	if err := m.Send(ctx, alert); err != nil {
		return err
	}

	// Immediately resolve the test alert
	return m.Resolve(ctx, "specular-test-alert")
}

// buildCreatePayload builds the Opsgenie create alert payload.
func (m *OpsgenieManager) buildCreatePayload(alert *Alert) *opsgenieCreateAlert {
	alias := alert.DedupeKey
	if alias == "" {
		alias = alert.ID
	}

	// Convert labels to tags and details
	var tags []string
	details := make(map[string]string)
	for k, v := range alert.Labels {
		tags = append(tags, fmt.Sprintf("%s:%s", k, v))
		details[k] = v
	}
	details["alert_id"] = alert.ID

	payload := &opsgenieCreateAlert{
		Message:     alert.Title,
		Description: alert.Description,
		Priority:    alert.Severity.ToOpsgeniePriority(),
		Alias:       alias,
		Source:      alert.Source,
		Entity:      "specular-cli",
		Tags:        tags,
		Details:     details,
	}

	// Add team responder if configured
	if m.config.TeamID != "" {
		payload.Responders = []opsgenieResponder{
			{Type: "team", ID: m.config.TeamID},
		}
	}

	return payload
}

// sendRequest sends an HTTP request to the Opsgenie API.
func (m *OpsgenieManager) sendRequest(ctx context.Context, method, url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Opsgenie payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Opsgenie request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("GenieKey %s", m.config.APIKey))

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Opsgenie request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Opsgenie response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("opsgenie API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var ogResp opsgenieResponse
	if unmarshalErr := json.Unmarshal(respBody, &ogResp); unmarshalErr != nil {
		// Some endpoints may not return JSON
		return nil
	}

	if ogResp.Message != "" && ogResp.Result == "" {
		return fmt.Errorf("opsgenie error: %s", ogResp.Message)
	}

	return nil
}
