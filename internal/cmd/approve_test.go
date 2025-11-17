package cmd

import (
	"testing"
	"time"
)

// TestApprovalRecord tests the ApprovalRecord struct definition
func TestApprovalRecord(t *testing.T) {
	// Create a sample approval to verify struct works
	now := time.Now()
	approval := ApprovalRecord{
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
	if approval.Version != "1.0" {
		t.Errorf("Version = %q, want %q", approval.Version, "1.0")
	}
	if approval.Type != "bundle" {
		t.Errorf("Type = %q, want %q", approval.Type, "bundle")
	}
	if approval.ResourceID != "bundle-abc123" {
		t.Errorf("ResourceID = %q, want %q", approval.ResourceID, "bundle-abc123")
	}
	if approval.ResourceHash != "sha256:def456" {
		t.Errorf("ResourceHash = %q, want %q", approval.ResourceHash, "sha256:def456")
	}
	if approval.ApprovedBy != "alice@example.com" {
		t.Errorf("ApprovedBy = %q, want %q", approval.ApprovedBy, "alice@example.com")
	}
	if !approval.ApprovedAt.Equal(now) {
		t.Errorf("ApprovedAt = %v, want %v", approval.ApprovedAt, now)
	}
	if approval.Message != "Approved for production" {
		t.Errorf("Message = %q, want %q", approval.Message, "Approved for production")
	}
	if len(approval.Metadata) != 2 {
		t.Errorf("Metadata length = %d, want %d", len(approval.Metadata), 2)
	}
	if approval.Metadata["environment"] != "prod" {
		t.Errorf("Metadata[environment] = %q, want %q", approval.Metadata["environment"], "prod")
	}
}
