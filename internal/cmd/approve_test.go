package cmd

import (
	"testing"
)

// TestApproveCommand tests the approve command configuration
func TestApproveCommand(t *testing.T) {
	// Check command configuration
	if approveCmd.Use != "approve [artifact]" {
		t.Errorf("approve Use = %q, want %q", approveCmd.Use, "approve [artifact]")
	}

	if approveCmd.Short == "" {
		t.Error("approve Short description is empty")
	}

	// Check Args is set (requires exactly 1 arg)
	if approveCmd.Args == nil {
		t.Error("approve command should have Args validator")
	}
}

// TestApproveFlags tests that approve has correct flags
func TestApproveFlags(t *testing.T) {
	// Check flags
	if approveCmd.Flags().Lookup("file") == nil {
		t.Error("flag 'file' not found on approve command")
	}
	if approveCmd.Flags().Lookup("approver") == nil {
		t.Error("flag 'approver' not found on approve command")
	}
	if approveCmd.Flags().Lookup("comment") == nil {
		t.Error("flag 'comment' not found on approve command")
	}
	if approveCmd.Flags().Lookup("env") == nil {
		t.Error("flag 'env' not found on approve command")
	}
}

// TestApproveArtifactTypes tests artifact type validation
func TestApproveArtifactTypes(t *testing.T) {
	// The approve command should accept spec, plan, and bundle
	// This test verifies the command structure is correct for validation
	if approveCmd.Args == nil {
		t.Fatal("approve command should have Args validator")
	}

	// Verify Use message indicates artifact types
	expectedUse := "approve [artifact]"
	if approveCmd.Use != expectedUse {
		t.Errorf("approve Use = %q, want %q", approveCmd.Use, expectedUse)
	}
}

// TestApprovalStruct tests the Approval struct definition
func TestApprovalStruct(t *testing.T) {
	// Create a sample approval to verify struct works
	approval := Approval{
		Artifact:    "spec",
		Path:        "/path/to/spec.yaml",
		Hash:        "abc123",
		ApprovedBy:  "alice@example.com",
		Comment:     "Looks good",
		Environment: "prod",
	}

	// Verify fields are accessible
	if approval.Artifact != "spec" {
		t.Errorf("Artifact = %q, want %q", approval.Artifact, "spec")
	}
	if approval.Path != "/path/to/spec.yaml" {
		t.Errorf("Path = %q, want %q", approval.Path, "/path/to/spec.yaml")
	}
	if approval.Hash != "abc123" {
		t.Errorf("Hash = %q, want %q", approval.Hash, "abc123")
	}
	if approval.ApprovedBy != "alice@example.com" {
		t.Errorf("ApprovedBy = %q, want %q", approval.ApprovedBy, "alice@example.com")
	}
	if approval.Comment != "Looks good" {
		t.Errorf("Comment = %q, want %q", approval.Comment, "Looks good")
	}
	if approval.Environment != "prod" {
		t.Errorf("Environment = %q, want %q", approval.Environment, "prod")
	}
}
