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

// SlackConfig holds Slack-specific configuration.
type SlackConfig struct {
	// WebhookURL is the Slack incoming webhook URL
	WebhookURL string

	// Channel is the default channel to send alerts (optional, webhook default is used)
	Channel string

	// Username is the bot username for alerts
	Username string

	// IconEmoji is the emoji icon for alert messages
	IconEmoji string

	// Common configuration
	Config
}

// SlackManager implements AlertManager for Slack.
type SlackManager struct {
	config     SlackConfig
	httpClient *http.Client
}

// slackMessage represents a Slack webhook message.
type slackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

// slackAttachment represents a Slack message attachment.
type slackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title,omitempty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fields     []slackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
}

// slackField represents a field in a Slack attachment.
type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackManager creates a new Slack alert manager.
func NewSlackManager(config SlackConfig) (*SlackManager, error) {
	if config.WebhookURL == "" {
		return nil, fmt.Errorf("slack webhook URL is required")
	}

	if config.Username == "" {
		config.Username = "Specular"
	}

	if config.IconEmoji == "" {
		config.IconEmoji = ":robot_face:"
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &SlackManager{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// Name returns the name of the alert manager.
func (m *SlackManager) Name() string {
	return "slack"
}

// Send sends an alert to Slack.
func (m *SlackManager) Send(ctx context.Context, alert *Alert) error {
	msg := m.buildMessage(alert, AlertEventTrigger)
	return m.sendRequest(ctx, msg)
}

// Resolve sends a resolution message to Slack.
func (m *SlackManager) Resolve(ctx context.Context, dedupeKey string) error {
	msg := &slackMessage{
		Channel:   m.config.Channel,
		Username:  m.config.Username,
		IconEmoji: m.config.IconEmoji,
		Attachments: []slackAttachment{
			{
				Color:     severityToSlackColor(SeverityInfo),
				Title:     "Alert Resolved",
				Text:      fmt.Sprintf("Alert `%s` has been resolved.", dedupeKey),
				Footer:    "Specular",
				Timestamp: time.Now().Unix(),
			},
		},
	}
	return m.sendRequest(ctx, msg)
}

// Test sends a test message to verify connectivity.
func (m *SlackManager) Test(ctx context.Context) error {
	alert := NewAlert(
		"Specular Test Alert",
		"This is a test message from Specular to verify Slack connectivity.",
		SeverityInfo,
	).WithDedupeKey("specular-test-alert").
		WithLabel("type", "test")

	return m.Send(ctx, alert)
}

// buildMessage builds a Slack message from an alert.
func (m *SlackManager) buildMessage(alert *Alert, event AlertEvent) *slackMessage {
	color := severityToSlackColor(alert.Severity)

	var eventIcon string
	switch event {
	case AlertEventTrigger:
		eventIcon = getAlertIcon(alert.Severity)
	case AlertEventResolve:
		eventIcon = ":white_check_mark:"
		color = "#36a64f" // Green
	default:
		eventIcon = ":bell:"
	}

	// Build fields from labels
	var fields []slackField
	for k, v := range alert.Labels {
		fields = append(fields, slackField{
			Title: k,
			Value: v,
			Short: true,
		})
	}

	// Add severity and source fields
	fields = append(fields,
		slackField{
			Title: "Severity",
			Value: string(alert.Severity),
			Short: true,
		},
		slackField{
			Title: "Source",
			Value: alert.Source,
			Short: true,
		},
	)

	attachment := slackAttachment{
		Color:     color,
		Title:     fmt.Sprintf("%s %s", eventIcon, alert.Title),
		Text:      alert.Description,
		Fields:    fields,
		Footer:    "Specular",
		Timestamp: alert.Timestamp.Unix(),
	}

	// Add first link as title link if available
	if len(alert.Links) > 0 {
		attachment.TitleLink = alert.Links[0].Href
	}

	return &slackMessage{
		Channel:     m.config.Channel,
		Username:    m.config.Username,
		IconEmoji:   m.config.IconEmoji,
		Attachments: []slackAttachment{attachment},
	}
}

// sendRequest sends an HTTP request to the Slack webhook.
func (m *SlackManager) sendRequest(ctx context.Context, msg *slackMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Slack response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Slack returns "ok" on success for most webhook configurations.
	// Some Slack configurations might return different success responses,
	// so we only validate the HTTP status code above.

	return nil
}

// severityToSlackColor converts severity to Slack attachment color.
func severityToSlackColor(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "#dc3545" // Red
	case SeverityHigh:
		return "#fd7e14" // Orange
	case SeverityWarning:
		return "#ffc107" // Yellow
	default:
		return "#17a2b8" // Blue (info)
	}
}

// getAlertIcon returns an emoji icon based on severity.
func getAlertIcon(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return ":rotating_light:"
	case SeverityHigh:
		return ":warning:"
	case SeverityWarning:
		return ":large_yellow_circle:"
	default:
		return ":information_source:"
	}
}
