package health

import (
	"context"
	"testing"
)

func TestNewVaultChecker(t *testing.T) {
	// Test with nil client
	checker := NewVaultChecker(nil)

	if checker == nil {
		t.Fatal("NewVaultChecker returned nil")
	}

	if checker.client != nil {
		t.Error("Expected client to be nil")
	}
}

func TestVaultCheckerName(t *testing.T) {
	checker := NewVaultChecker(nil)

	name := checker.Name()
	if name != "vault-server" {
		t.Errorf("Name() = %q, want %q", name, "vault-server")
	}
}

func TestVaultCheckerCheckWithNilClient(t *testing.T) {
	checker := NewVaultChecker(nil)
	ctx := context.Background()

	result := checker.Check(ctx)

	if result == nil {
		t.Fatal("Check() returned nil")
	}

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusUnhealthy)
	}

	if result.Message != "Vault client not configured" {
		t.Errorf("Message = %q, want %q", result.Message, "Vault client not configured")
	}

	if suggestion, ok := result.Details["suggestion"]; !ok {
		t.Error("Unhealthy result should include suggestion")
	} else if suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}

func TestNewVaultSignerChecker(t *testing.T) {
	// Test with nil signer
	checker := NewVaultSignerChecker(nil)

	if checker == nil {
		t.Fatal("NewVaultSignerChecker returned nil")
	}

	if checker.signer != nil {
		t.Error("Expected signer to be nil")
	}
}

func TestVaultSignerCheckerName(t *testing.T) {
	checker := NewVaultSignerChecker(nil)

	name := checker.Name()
	if name != "vault-signer" {
		t.Errorf("Name() = %q, want %q", name, "vault-signer")
	}
}

func TestVaultSignerCheckerCheckWithNilSigner(t *testing.T) {
	checker := NewVaultSignerChecker(nil)
	ctx := context.Background()

	result := checker.Check(ctx)

	if result == nil {
		t.Fatal("Check() returned nil")
	}

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusUnhealthy)
	}

	if result.Message != "Vault signer not configured" {
		t.Errorf("Message = %q, want %q", result.Message, "Vault signer not configured")
	}

	if suggestion, ok := result.Details["suggestion"]; !ok {
		t.Error("Unhealthy result should include suggestion")
	} else if suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}

func TestVaultCheckersImplementInterface(t *testing.T) {
	// Verify both checkers implement the Checker interface
	var _ Checker = (*VaultChecker)(nil)
	var _ Checker = (*VaultSignerChecker)(nil)
}
