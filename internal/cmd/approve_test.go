package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/approval"
)

// TestApprovalRecord tests the ApprovalRecord struct definition
func TestApprovalRecord(t *testing.T) {
	// Create a sample approval to verify struct works
	now := time.Now()
	rec := ApprovalRecord{
		Version:      "1.0",
		Type:         "bundle",
		ResourceID:   "bundle-abc123",
		ResourceHash: "sha256:def456",
		ApprovedBy:   "alice@example.com",
		ApprovedAt:   now,
		Message:      "Approved for production",
		Metadata: map[string]string{
			"environment": "prod",
			"reviewer":    "bob",
		},
	}

	// Verify fields are accessible
	if rec.Version != "1.0" {
		t.Errorf("Version = %q, want %q", rec.Version, "1.0")
	}
	if rec.Type != "bundle" {
		t.Errorf("Type = %q, want %q", rec.Type, "bundle")
	}
	if rec.ResourceID != "bundle-abc123" {
		t.Errorf("ResourceID = %q, want %q", rec.ResourceID, "bundle-abc123")
	}
	if rec.ResourceHash != "sha256:def456" {
		t.Errorf("ResourceHash = %q, want %q", rec.ResourceHash, "sha256:def456")
	}
	if rec.ApprovedBy != "alice@example.com" {
		t.Errorf("ApprovedBy = %q, want %q", rec.ApprovedBy, "alice@example.com")
	}
	if !rec.ApprovedAt.Equal(now) {
		t.Errorf("ApprovedAt = %v, want %v", rec.ApprovedAt, now)
	}
	if rec.Message != "Approved for production" {
		t.Errorf("Message = %q, want %q", rec.Message, "Approved for production")
	}
	if len(rec.Metadata) != 2 {
		t.Errorf("Metadata length = %d, want %d", len(rec.Metadata), 2)
	}
	if rec.Metadata["environment"] != "prod" {
		t.Errorf("Metadata[environment] = %q, want %q", rec.Metadata["environment"], "prod")
	}
}

func TestFormatApprovalExplainException(t *testing.T) {
	t.Parallel()
	exp := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	text := formatApprovalExplain(approval.Record{
		Type:       approval.TypeException,
		ResourceID: "exception-EX-192",
		ApprovedBy: "alice",
		ApprovedAt: time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC),
		Message:    "temporary",
		Reason:     "emergency hotfix",
		Scope:      "internal/auth/**",
		Policy:     "SEC-17",
		Requester:  "bob",
		ExpiresAt:  &exp,
		EvidenceID: "ev_soft_allow",
		Path:       ".specular/approvals/exception-20260921-100000.yaml",
	})
	for _, want := range []string{
		"APPROVAL / EXCEPTION RECORD",
		"Type         exception",
		"Resource     exception-EX-192",
		"Reason       emergency hotfix",
		"Scope        internal/auth/**",
		"Policy       SEC-17",
		"Status       OPEN",
		"Evidence     ev_soft_allow",
		"Evidence     specular evidence show ev_soft_allow",
		"             specular explain ev_soft_allow",
		"specular approvals list",
		"Open         specular approvals list --status open",
		"Gate         specular gate",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
}

func TestFormatApprovalExplainNoEvidenceRefs(t *testing.T) {
	t.Parallel()
	text := formatApprovalExplain(approval.Record{
		Type:       approval.TypeBundle,
		ResourceID: "bundle-abc",
		ApprovedBy: "alice",
		ApprovedAt: time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC),
		Message:    "ok",
	})
	if strings.Contains(text, "evidence show") || strings.Contains(text, "list --status open") {
		t.Fatalf("unexpected evidence/open refs:\n%s", text)
	}
	if !strings.Contains(text, "Gate         specular gate") {
		t.Fatalf("missing gate ref:\n%s", text)
	}
}
