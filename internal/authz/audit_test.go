package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// TestDefaultAuditLogger_LogDecision tests basic logging functionality.
func TestDefaultAuditLogger_LogDecision(t *testing.T) {
	var buf bytes.Buffer
	logger := NewDefaultAuditLogger(AuditLoggerConfig{
		Writer:          &buf,
		LogAllDecisions: true,
	})

	entry := &AuditEntry{
		Timestamp:      time.Now(),
		Allowed:        true,
		Reason:         "access granted by policy",
		UserID:         "user-1",
		Email:          "user@example.com",
		OrganizationID: "org-1",
		Role:           "admin",
		Action:         "plan:approve",
		ResourceType:   "plan",
		ResourceID:     "plan-123",
		PolicyIDs:      []string{"policy-1"},
		Duration:       5 * time.Millisecond,
	}

	err := logger.LogDecision(context.Background(), entry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Close and wait for async processing
	logger.Close()

	// Verify JSON was written
	if buf.Len() == 0 {
		t.Fatal("expected audit log output, got empty buffer")
	}

	// Parse JSON
	var logged AuditEntry
	if err := json.Unmarshal(buf.Bytes(), &logged); err != nil {
		t.Fatalf("failed to parse audit log JSON: %v", err)
	}

	// Verify fields
	if logged.UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", logged.UserID)
	}
	if logged.Action != "plan:approve" {
		t.Errorf("expected Action plan:approve, got %s", logged.Action)
	}
	if !logged.Allowed {
		t.Error("expected Allowed true")
	}
}

// TestDefaultAuditLogger_LogOnlyDenials tests filtering to log only denials.
func TestDefaultAuditLogger_LogOnlyDenials(t *testing.T) {
	var buf bytes.Buffer
	logger := NewDefaultAuditLogger(AuditLoggerConfig{
		Writer:          &buf,
		LogAllDecisions: false, // Only log denials
	})

	// Log an allowed decision
	allowEntry := &AuditEntry{
		Timestamp: time.Now(),
		Allowed:   true,
		Reason:    "access granted",
		UserID:    "user-1",
		Action:    "plan:read",
	}
	err := logger.LogDecision(context.Background(), allowEntry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Log a denied decision
	denyEntry := &AuditEntry{
		Timestamp: time.Now(),
		Allowed:   false,
		Reason:    "access denied",
		UserID:    "user-2",
		Action:    "plan:delete",
	}
	err = logger.LogDecision(context.Background(), denyEntry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	logger.Close()

	// Should only have one entry (the denial)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(lines))
	}

	var logged AuditEntry
	if err := json.Unmarshal([]byte(lines[0]), &logged); err != nil {
		t.Fatalf("failed to parse audit log JSON: %v", err)
	}

	if logged.Allowed {
		t.Error("expected only denied entry to be logged")
	}
	if logged.UserID != "user-2" {
		t.Errorf("expected UserID user-2, got %s", logged.UserID)
	}
}

// TestDefaultAuditLogger_IncludeEnvironment tests environment filtering.
func TestDefaultAuditLogger_IncludeEnvironment(t *testing.T) {
	tests := []struct {
		name               string
		includeEnvironment bool
		expectEnv          bool
	}{
		{
			name:               "include environment",
			includeEnvironment: true,
			expectEnv:          true,
		},
		{
			name:               "exclude environment",
			includeEnvironment: false,
			expectEnv:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewDefaultAuditLogger(AuditLoggerConfig{
				Writer:             &buf,
				LogAllDecisions:    true,
				IncludeEnvironment: tt.includeEnvironment,
			})

			entry := &AuditEntry{
				Timestamp: time.Now(),
				Allowed:   true,
				Reason:    "test",
				UserID:    "user-1",
				Environment: map[string]interface{}{
					"client_ip": "192.168.1.1",
					"method":    "GET",
				},
			}

			err := logger.LogDecision(context.Background(), entry)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			logger.Close()

			var logged AuditEntry
			if err := json.Unmarshal(buf.Bytes(), &logged); err != nil {
				t.Fatalf("failed to parse audit log JSON: %v", err)
			}

			hasEnv := len(logged.Environment) > 0
			if hasEnv != tt.expectEnv {
				t.Errorf("expected environment=%v, got %v", tt.expectEnv, hasEnv)
			}
		})
	}
}

// TestInMemoryAuditLogger tests the in-memory logger.
func TestInMemoryAuditLogger(t *testing.T) {
	logger := NewInMemoryAuditLogger()

	entries := []*AuditEntry{
		{
			Timestamp: time.Now(),
			Allowed:   true,
			UserID:    "user-1",
			Action:    "plan:read",
		},
		{
			Timestamp: time.Now(),
			Allowed:   false,
			UserID:    "user-2",
			Action:    "plan:delete",
		},
	}

	for _, entry := range entries {
		err := logger.LogDecision(context.Background(), entry)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	}

	// Retrieve entries
	logged := logger.GetEntries()
	if len(logged) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(logged))
	}

	// Verify first entry
	if logged[0].UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", logged[0].UserID)
	}

	// Verify second entry
	if logged[1].UserID != "user-2" {
		t.Errorf("expected UserID user-2, got %s", logged[1].UserID)
	}

	// Test Clear
	logger.Clear()
	if len(logger.GetEntries()) != 0 {
		t.Error("expected entries to be cleared")
	}
}

// TestInMemoryAuditLogger_CopyProtection tests that entries are copied to prevent mutations.
func TestInMemoryAuditLogger_CopyProtection(t *testing.T) {
	logger := NewInMemoryAuditLogger()

	entry := &AuditEntry{
		Timestamp: time.Now(),
		Allowed:   true,
		UserID:    "user-1",
		Environment: map[string]interface{}{
			"key": "value",
		},
		PolicyIDs: []string{"policy-1"},
	}

	err := logger.LogDecision(context.Background(), entry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Mutate original entry
	entry.UserID = "modified"
	entry.Environment["key"] = "modified"
	entry.PolicyIDs[0] = "modified"

	// Verify logged entry is not affected
	logged := logger.GetEntries()
	if logged[0].UserID == "modified" {
		t.Error("logged entry should be protected from mutations")
	}
	if logged[0].Environment["key"] == "modified" {
		t.Error("logged entry environment should be protected from mutations")
	}
	if logged[0].PolicyIDs[0] == "modified" {
		t.Error("logged entry policy IDs should be protected from mutations")
	}
}

// TestNoOpAuditLogger tests the no-op logger.
func TestNoOpAuditLogger(t *testing.T) {
	logger := NewNoOpAuditLogger()

	entry := &AuditEntry{
		Timestamp: time.Now(),
		Allowed:   true,
		UserID:    "user-1",
	}

	err := logger.LogDecision(context.Background(), entry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = logger.Close()
	if err != nil {
		t.Fatalf("expected no error on close, got %v", err)
	}
}

// TestNewAuditEntry tests the helper function.
func TestNewAuditEntry(t *testing.T) {
	session := &auth.Session{
		UserID:           "user-1",
		Email:            "user@example.com",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
		Environment: map[string]interface{}{
			"client_ip": "192.168.1.1",
		},
	}

	decision := &Decision{
		Allowed:   true,
		Reason:    "access granted by policy",
		PolicyIDs: []string{"policy-1", "policy-2"},
		Timestamp: time.Now(),
	}

	duration := 5 * time.Millisecond

	entry := NewAuditEntry(req, decision, duration)

	// Verify all fields are set correctly
	if entry.Allowed != decision.Allowed {
		t.Errorf("expected Allowed %v, got %v", decision.Allowed, entry.Allowed)
	}
	if entry.Reason != decision.Reason {
		t.Errorf("expected Reason %s, got %s", decision.Reason, entry.Reason)
	}
	if entry.UserID != session.UserID {
		t.Errorf("expected UserID %s, got %s", session.UserID, entry.UserID)
	}
	if entry.Email != session.Email {
		t.Errorf("expected Email %s, got %s", session.Email, entry.Email)
	}
	if entry.OrganizationID != session.OrganizationID {
		t.Errorf("expected OrganizationID %s, got %s", session.OrganizationID, entry.OrganizationID)
	}
	if entry.Role != session.OrganizationRole {
		t.Errorf("expected Role %s, got %s", session.OrganizationRole, entry.Role)
	}
	if entry.Action != req.Action {
		t.Errorf("expected Action %s, got %s", req.Action, entry.Action)
	}
	if entry.ResourceType != req.Resource.Type {
		t.Errorf("expected ResourceType %s, got %s", req.Resource.Type, entry.ResourceType)
	}
	if entry.ResourceID != req.Resource.ID {
		t.Errorf("expected ResourceID %s, got %s", req.Resource.ID, entry.ResourceID)
	}
	if len(entry.PolicyIDs) != len(decision.PolicyIDs) {
		t.Errorf("expected %d PolicyIDs, got %d", len(decision.PolicyIDs), len(entry.PolicyIDs))
	}
	if entry.Duration != duration {
		t.Errorf("expected Duration %v, got %v", duration, entry.Duration)
	}
	if entry.Environment["client_ip"] != "192.168.1.1" {
		t.Error("expected environment attributes to be copied")
	}
}

// TestEngine_WithAuditLogger tests engine integration with audit logging.
func TestEngine_WithAuditLogger(t *testing.T) {
	// Setup
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	auditLogger := NewInMemoryAuditLogger()

	policy := &Policy{
		ID:             "test-policy",
		OrganizationID: "org-1",
		Name:           "Test Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "admin", Scope: "organization"},
		},
		Actions:   []string{"plan:approve"},
		Resources: []string{"*"},
		Enabled:   true,
	}
	store.CreatePolicy(context.Background(), policy)

	// Create engine with audit logger
	engine := NewEngine(store, resolver)
	engine = WithAuditLogger(engine, auditLogger)

	// Make authorization request
	session := &auth.Session{
		UserID:           "user-1",
		Email:            "user@example.com",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
	}

	decision, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !decision.Allowed {
		t.Error("expected access to be allowed")
	}

	// Verify audit log
	entries := auditLogger.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", entry.UserID)
	}
	if entry.Action != "plan:approve" {
		t.Errorf("expected Action plan:approve, got %s", entry.Action)
	}
	if !entry.Allowed {
		t.Error("expected audit entry to show allowed")
	}
	if entry.Duration <= 0 {
		t.Error("expected positive duration in audit entry")
	}
}

// TestEngine_WithoutAuditLogger tests engine works without audit logger.
func TestEngine_WithoutAuditLogger(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)

	policy := &Policy{
		ID:             "test-policy",
		OrganizationID: "org-1",
		Name:           "Test Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "admin", Scope: "organization"},
		},
		Actions:   []string{"plan:approve"},
		Resources: []string{"*"},
		Enabled:   true,
	}
	store.CreatePolicy(context.Background(), policy)

	// Create engine without audit logger
	engine := NewEngine(store, resolver)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
	}

	// Should work fine without audit logger
	decision, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !decision.Allowed {
		t.Error("expected access to be allowed")
	}
}

// TestDefaultAuditLogger_GracefulShutdown tests that buffered entries are flushed on close.
func TestDefaultAuditLogger_GracefulShutdown(t *testing.T) {
	var buf bytes.Buffer
	logger := NewDefaultAuditLogger(AuditLoggerConfig{
		Writer:          &buf,
		LogAllDecisions: true,
		BufferSize:      10,
	})

	// Log multiple entries
	for i := 0; i < 5; i++ {
		entry := &AuditEntry{
			Timestamp: time.Now(),
			Allowed:   true,
			UserID:    "user-1",
			Action:    "test",
		}
		logger.LogDecision(context.Background(), entry)
	}

	// Close should flush all buffered entries
	logger.Close()

	// Verify all entries were written
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 log entries, got %d", len(lines))
	}
}
