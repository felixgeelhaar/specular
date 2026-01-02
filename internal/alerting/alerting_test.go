package alerting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestSeverity(t *testing.T) {
	tests := []struct {
		severity Severity
		valid    bool
		pd       string
		og       string
	}{
		{SeverityCritical, true, "critical", "P1"},
		{SeverityHigh, true, "error", "P2"},
		{SeverityWarning, true, "warning", "P3"},
		{SeverityInfo, true, "info", "P4"},
		{Severity("invalid"), false, "info", "P4"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			if got := tt.severity.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
			if got := tt.severity.ToPagerDutySeverity(); got != tt.pd {
				t.Errorf("ToPagerDutySeverity() = %v, want %v", got, tt.pd)
			}
			if got := tt.severity.ToOpsgeniePriority(); got != tt.og {
				t.Errorf("ToOpsgeniePriority() = %v, want %v", got, tt.og)
			}
		})
	}
}

func TestNewAlert(t *testing.T) {
	alert := NewAlert("Test Title", "Test Description", SeverityHigh)

	if alert.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", alert.Title, "Test Title")
	}
	if alert.Description != "Test Description" {
		t.Errorf("Description = %q, want %q", alert.Description, "Test Description")
	}
	if alert.Severity != SeverityHigh {
		t.Errorf("Severity = %v, want %v", alert.Severity, SeverityHigh)
	}
	if alert.ID == "" {
		t.Error("ID should not be empty")
	}
	if alert.Source != "specular" {
		t.Errorf("Source = %q, want %q", alert.Source, "specular")
	}
	if alert.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestAlertBuilder(t *testing.T) {
	alert := NewAlert("Test", "Desc", SeverityInfo).
		WithDedupeKey("test-key").
		WithLabel("env", "prod").
		WithLabel("service", "api").
		WithLink("Dashboard", "https://example.com/dashboard")

	if alert.DedupeKey != "test-key" {
		t.Errorf("DedupeKey = %q, want %q", alert.DedupeKey, "test-key")
	}
	if alert.Labels["env"] != "prod" {
		t.Errorf("Labels[env] = %q, want %q", alert.Labels["env"], "prod")
	}
	if alert.Labels["service"] != "api" {
		t.Errorf("Labels[service] = %q, want %q", alert.Labels["service"], "api")
	}
	if len(alert.Links) != 1 {
		t.Fatalf("len(Links) = %d, want 1", len(alert.Links))
	}
	if alert.Links[0].Text != "Dashboard" {
		t.Errorf("Links[0].Text = %q, want %q", alert.Links[0].Text, "Dashboard")
	}
	if alert.Links[0].Href != "https://example.com/dashboard" {
		t.Errorf("Links[0].Href = %q, want %q", alert.Links[0].Href, "https://example.com/dashboard")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Enabled {
		t.Error("Enabled should be false by default")
	}
	if cfg.DefaultSeverity != SeverityWarning {
		t.Errorf("DefaultSeverity = %v, want %v", cfg.DefaultSeverity, SeverityWarning)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 30*time.Second)
	}
	if cfg.RetryCount != 3 {
		t.Errorf("RetryCount = %d, want 3", cfg.RetryCount)
	}
	if cfg.RetryDelay != time.Second {
		t.Errorf("RetryDelay = %v, want %v", cfg.RetryDelay, time.Second)
	}
}

func TestRouter(t *testing.T) {
	t.Run("empty router returns nil", func(t *testing.T) {
		router := NewRouter()
		ctx := context.Background()
		alert := NewAlert("Test", "Desc", SeverityInfo)

		if err := router.Send(ctx, alert); err != nil {
			t.Errorf("Send() error = %v", err)
		}
		if err := router.Resolve(ctx, "key"); err != nil {
			t.Errorf("Resolve() error = %v", err)
		}
		if err := router.Test(ctx); err != nil {
			t.Errorf("Test() error = %v", err)
		}
	})

	t.Run("with managers", func(t *testing.T) {
		var sendCalled, resolveCalled, testCalled int
		var mu sync.Mutex

		mock := &mockAlertManager{
			sendFunc: func(ctx context.Context, alert *Alert) error {
				mu.Lock()
				sendCalled++
				mu.Unlock()
				return nil
			},
			resolveFunc: func(ctx context.Context, key string) error {
				mu.Lock()
				resolveCalled++
				mu.Unlock()
				return nil
			},
			testFunc: func(ctx context.Context) error {
				mu.Lock()
				testCalled++
				mu.Unlock()
				return nil
			},
		}

		router := NewRouter(WithManager(mock))
		ctx := context.Background()
		alert := NewAlert("Test", "Desc", SeverityInfo)

		if err := router.Send(ctx, alert); err != nil {
			t.Errorf("Send() error = %v", err)
		}
		if sendCalled != 1 {
			t.Errorf("sendCalled = %d, want 1", sendCalled)
		}

		if err := router.Resolve(ctx, "key"); err != nil {
			t.Errorf("Resolve() error = %v", err)
		}
		if resolveCalled != 1 {
			t.Errorf("resolveCalled = %d, want 1", resolveCalled)
		}

		if err := router.Test(ctx); err != nil {
			t.Errorf("Test() error = %v", err)
		}
		if testCalled != 1 {
			t.Errorf("testCalled = %d, want 1", testCalled)
		}
	})

	t.Run("sets default severity", func(t *testing.T) {
		var capturedAlert *Alert
		mock := &mockAlertManager{
			sendFunc: func(ctx context.Context, alert *Alert) error {
				capturedAlert = alert
				return nil
			},
		}

		router := NewRouter(WithManager(mock))
		alert := &Alert{Title: "Test", Description: "Desc"}

		_ = router.Send(context.Background(), alert)

		if capturedAlert.Severity != SeverityWarning {
			t.Errorf("Severity = %v, want %v", capturedAlert.Severity, SeverityWarning)
		}
	})

	t.Run("managers list", func(t *testing.T) {
		mock1 := &mockAlertManager{name: "mock1"}
		mock2 := &mockAlertManager{name: "mock2"}

		router := NewRouter(WithManager(mock1), WithManager(mock2))
		managers := router.Managers()

		if len(managers) != 2 {
			t.Fatalf("len(Managers()) = %d, want 2", len(managers))
		}
	})
}

// mockAlertManager is a test helper
type mockAlertManager struct {
	name        string
	sendFunc    func(ctx context.Context, alert *Alert) error
	resolveFunc func(ctx context.Context, key string) error
	testFunc    func(ctx context.Context) error
}

func (m *mockAlertManager) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func (m *mockAlertManager) Send(ctx context.Context, alert *Alert) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, alert)
	}
	return nil
}

func (m *mockAlertManager) Resolve(ctx context.Context, key string) error {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, key)
	}
	return nil
}

func (m *mockAlertManager) Test(ctx context.Context) error {
	if m.testFunc != nil {
		return m.testFunc(ctx)
	}
	return nil
}

func TestPagerDutyManager(t *testing.T) {
	t.Run("requires routing key", func(t *testing.T) {
		_, err := NewPagerDutyManager(PagerDutyConfig{})
		if err == nil {
			t.Error("expected error for missing routing key")
		}
	})

	t.Run("send alert", func(t *testing.T) {
		var receivedPayload pagerdutyPayload

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Method = %s, want POST", r.Method)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %s, want application/json", ct)
			}

			if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
				t.Fatalf("failed to decode payload: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(pagerdutyResponse{
				Status:   "success",
				Message:  "Event processed",
				DedupKey: receivedPayload.DedupKey,
			})
		}))
		defer server.Close()

		mgr, err := NewPagerDutyManager(PagerDutyConfig{
			RoutingKey: "test-routing-key",
			URL:        server.URL,
		})
		if err != nil {
			t.Fatalf("NewPagerDutyManager() error = %v", err)
		}

		alert := NewAlert("Test Alert", "Test Description", SeverityCritical).
			WithDedupeKey("test-dedup").
			WithLabel("env", "test")

		if err := mgr.Send(context.Background(), alert); err != nil {
			t.Errorf("Send() error = %v", err)
		}

		if receivedPayload.EventAction != "trigger" {
			t.Errorf("EventAction = %s, want trigger", receivedPayload.EventAction)
		}
		if receivedPayload.RoutingKey != "test-routing-key" {
			t.Errorf("RoutingKey = %s, want test-routing-key", receivedPayload.RoutingKey)
		}
		if receivedPayload.Payload.Severity != "critical" {
			t.Errorf("Severity = %s, want critical", receivedPayload.Payload.Severity)
		}
	})

	t.Run("resolve alert", func(t *testing.T) {
		var receivedPayload pagerdutyPayload

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(pagerdutyResponse{Status: "success"})
		}))
		defer server.Close()

		mgr, _ := NewPagerDutyManager(PagerDutyConfig{
			RoutingKey: "test-key",
			URL:        server.URL,
		})

		if err := mgr.Resolve(context.Background(), "dedup-key"); err != nil {
			t.Errorf("Resolve() error = %v", err)
		}

		if receivedPayload.EventAction != "resolve" {
			t.Errorf("EventAction = %s, want resolve", receivedPayload.EventAction)
		}
		if receivedPayload.DedupKey != "dedup-key" {
			t.Errorf("DedupKey = %s, want dedup-key", receivedPayload.DedupKey)
		}
	})

	t.Run("name", func(t *testing.T) {
		mgr, _ := NewPagerDutyManager(PagerDutyConfig{RoutingKey: "key"})
		if mgr.Name() != "pagerduty" {
			t.Errorf("Name() = %s, want pagerduty", mgr.Name())
		}
	})
}

func TestSlackManager(t *testing.T) {
	t.Run("requires webhook URL", func(t *testing.T) {
		_, err := NewSlackManager(SlackConfig{})
		if err == nil {
			t.Error("expected error for missing webhook URL")
		}
	})

	t.Run("send alert", func(t *testing.T) {
		var receivedMsg slackMessage

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Method = %s, want POST", r.Method)
			}

			json.NewDecoder(r.Body).Decode(&receivedMsg)
			w.Write([]byte("ok"))
		}))
		defer server.Close()

		mgr, err := NewSlackManager(SlackConfig{
			WebhookURL: server.URL,
			Channel:    "#alerts",
		})
		if err != nil {
			t.Fatalf("NewSlackManager() error = %v", err)
		}

		alert := NewAlert("Test Alert", "Test Description", SeverityCritical)
		if err := mgr.Send(context.Background(), alert); err != nil {
			t.Errorf("Send() error = %v", err)
		}

		if receivedMsg.Channel != "#alerts" {
			t.Errorf("Channel = %s, want #alerts", receivedMsg.Channel)
		}
		if receivedMsg.Username != "Specular" {
			t.Errorf("Username = %s, want Specular", receivedMsg.Username)
		}
		if len(receivedMsg.Attachments) != 1 {
			t.Fatalf("len(Attachments) = %d, want 1", len(receivedMsg.Attachments))
		}
	})

	t.Run("name", func(t *testing.T) {
		mgr, _ := NewSlackManager(SlackConfig{WebhookURL: "http://example.com"})
		if mgr.Name() != "slack" {
			t.Errorf("Name() = %s, want slack", mgr.Name())
		}
	})
}

func TestOpsgenieManager(t *testing.T) {
	t.Run("requires API key", func(t *testing.T) {
		_, err := NewOpsgenieManager(OpsgenieConfig{})
		if err == nil {
			t.Error("expected error for missing API key")
		}
	})

	t.Run("defaults to US region", func(t *testing.T) {
		mgr, _ := NewOpsgenieManager(OpsgenieConfig{APIKey: "test"})
		if mgr.baseURL != opsgenieUSURL {
			t.Errorf("baseURL = %s, want %s", mgr.baseURL, opsgenieUSURL)
		}
	})

	t.Run("EU region", func(t *testing.T) {
		mgr, _ := NewOpsgenieManager(OpsgenieConfig{APIKey: "test", Region: "eu"})
		if mgr.baseURL != opsgenieEUURL {
			t.Errorf("baseURL = %s, want %s", mgr.baseURL, opsgenieEUURL)
		}
	})

	t.Run("send alert", func(t *testing.T) {
		var receivedPayload opsgenieCreateAlert

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if auth := r.Header.Get("Authorization"); auth != "GenieKey test-api-key" {
				t.Errorf("Authorization = %s, want GenieKey test-api-key", auth)
			}

			json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(opsgenieResponse{
				Result:    "created",
				RequestID: "req-123",
			})
		}))
		defer server.Close()

		// Create manager with custom URL by directly setting baseURL
		mgr := &OpsgenieManager{
			config: OpsgenieConfig{
				APIKey: "test-api-key",
				TeamID: "team-123",
			},
			baseURL:    server.URL,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		}

		alert := NewAlert("Test Alert", "Test Description", SeverityCritical).
			WithDedupeKey("test-dedup")

		if err := mgr.Send(context.Background(), alert); err != nil {
			t.Errorf("Send() error = %v", err)
		}

		if receivedPayload.Priority != "P1" {
			t.Errorf("Priority = %s, want P1", receivedPayload.Priority)
		}
		if receivedPayload.Alias != "test-dedup" {
			t.Errorf("Alias = %s, want test-dedup", receivedPayload.Alias)
		}
		if len(receivedPayload.Responders) != 1 {
			t.Fatalf("len(Responders) = %d, want 1", len(receivedPayload.Responders))
		}
		if receivedPayload.Responders[0].ID != "team-123" {
			t.Errorf("Responders[0].ID = %s, want team-123", receivedPayload.Responders[0].ID)
		}
	})

	t.Run("name", func(t *testing.T) {
		mgr, _ := NewOpsgenieManager(OpsgenieConfig{APIKey: "key"})
		if mgr.Name() != "opsgenie" {
			t.Errorf("Name() = %s, want opsgenie", mgr.Name())
		}
	})
}

func TestWebhookManager(t *testing.T) {
	t.Run("requires URL", func(t *testing.T) {
		_, err := NewWebhookManager(WebhookConfig{})
		if err == nil {
			t.Error("expected error for missing URL")
		}
	})

	t.Run("send alert", func(t *testing.T) {
		var receivedPayload webhookPayload

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Method = %s, want POST", r.Method)
			}
			if ua := r.Header.Get("User-Agent"); ua != "Specular/1.0" {
				t.Errorf("User-Agent = %s, want Specular/1.0", ua)
			}
			if custom := r.Header.Get("X-Custom-Header"); custom != "custom-value" {
				t.Errorf("X-Custom-Header = %s, want custom-value", custom)
			}

			json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		mgr, _ := NewWebhookManager(WebhookConfig{
			URL: server.URL,
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
			},
		})

		alert := NewAlert("Test Alert", "Test Description", SeverityWarning)
		if err := mgr.Send(context.Background(), alert); err != nil {
			t.Errorf("Send() error = %v", err)
		}

		if receivedPayload.Event != "trigger" {
			t.Errorf("Event = %s, want trigger", receivedPayload.Event)
		}
		if receivedPayload.Alert == nil {
			t.Fatal("Alert should not be nil")
		}
		if receivedPayload.Alert.Severity != "warning" {
			t.Errorf("Alert.Severity = %s, want warning", receivedPayload.Alert.Severity)
		}
	})

	t.Run("resolve alert", func(t *testing.T) {
		var receivedPayload webhookPayload

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		mgr, _ := NewWebhookManager(WebhookConfig{URL: server.URL})
		if err := mgr.Resolve(context.Background(), "test-key"); err != nil {
			t.Errorf("Resolve() error = %v", err)
		}

		if receivedPayload.Event != "resolve" {
			t.Errorf("Event = %s, want resolve", receivedPayload.Event)
		}
		if receivedPayload.DedupeKey != "test-key" {
			t.Errorf("DedupeKey = %s, want test-key", receivedPayload.DedupeKey)
		}
	})

	t.Run("with signature", func(t *testing.T) {
		secret := "test-secret"
		var receivedSignature string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedSignature = r.Header.Get("X-Signature-256")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		mgr, _ := NewWebhookManager(WebhookConfig{
			URL:    server.URL,
			Secret: secret,
		})

		alert := NewAlert("Test", "Desc", SeverityInfo)
		if err := mgr.Send(context.Background(), alert); err != nil {
			t.Errorf("Send() error = %v", err)
		}

		if receivedSignature == "" {
			t.Error("signature header should not be empty")
		}
		if len(receivedSignature) < 10 {
			t.Errorf("signature too short: %s", receivedSignature)
		}
	})

	t.Run("name", func(t *testing.T) {
		mgr, _ := NewWebhookManager(WebhookConfig{URL: "http://example.com"})
		if mgr.Name() != "webhook" {
			t.Errorf("Name() = %s, want webhook", mgr.Name())
		}
	})
}

func TestVerifySignature(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"event":"test"}`)

	// Compute expected signature
	mgr := &WebhookManager{config: WebhookConfig{Secret: secret}}
	expectedSig := "sha256=" + mgr.computeSignature(payload)

	if !VerifySignature(payload, expectedSig, secret) {
		t.Error("VerifySignature() should return true for valid signature")
	}

	if VerifySignature(payload, "sha256=invalid", secret) {
		t.Error("VerifySignature() should return false for invalid signature")
	}

	if VerifySignature(payload, expectedSig, "wrong-secret") {
		t.Error("VerifySignature() should return false for wrong secret")
	}
}

func TestSlackSeverityColors(t *testing.T) {
	tests := []struct {
		severity Severity
		color    string
	}{
		{SeverityCritical, "#dc3545"},
		{SeverityHigh, "#fd7e14"},
		{SeverityWarning, "#ffc107"},
		{SeverityInfo, "#17a2b8"},
		{Severity("unknown"), "#17a2b8"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			if got := severityToSlackColor(tt.severity); got != tt.color {
				t.Errorf("severityToSlackColor(%s) = %s, want %s", tt.severity, got, tt.color)
			}
		})
	}
}

func TestSlackAlertIcons(t *testing.T) {
	tests := []struct {
		severity Severity
		icon     string
	}{
		{SeverityCritical, ":rotating_light:"},
		{SeverityHigh, ":warning:"},
		{SeverityWarning, ":large_yellow_circle:"},
		{SeverityInfo, ":information_source:"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			if got := getAlertIcon(tt.severity); got != tt.icon {
				t.Errorf("getAlertIcon(%s) = %s, want %s", tt.severity, got, tt.icon)
			}
		})
	}
}
