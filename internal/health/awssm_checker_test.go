package health

import (
	"context"
	"testing"
)

func TestNewAWSSecretsManagerChecker(t *testing.T) {
	// Test with nil client
	checker := NewAWSSecretsManagerChecker(nil)

	if checker == nil {
		t.Fatal("NewAWSSecretsManagerChecker returned nil")
	}

	if checker.client != nil {
		t.Error("Expected client to be nil")
	}
}

func TestAWSSecretsManagerCheckerName(t *testing.T) {
	checker := NewAWSSecretsManagerChecker(nil)

	name := checker.Name()
	if name != "aws-secrets-manager" {
		t.Errorf("Name() = %q, want %q", name, "aws-secrets-manager")
	}
}

func TestAWSSecretsManagerCheckerCheckWithNilClient(t *testing.T) {
	checker := NewAWSSecretsManagerChecker(nil)
	ctx := context.Background()

	result := checker.Check(ctx)

	if result == nil {
		t.Fatal("Check() returned nil")
	}

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusUnhealthy)
	}

	if result.Message != "AWS Secrets Manager client not configured" {
		t.Errorf("Message = %q, want %q", result.Message, "AWS Secrets Manager client not configured")
	}

	if suggestion, ok := result.Details["suggestion"]; !ok {
		t.Error("Unhealthy result should include suggestion")
	} else if suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}

func TestNewAWSSecretsManagerSignerChecker(t *testing.T) {
	// Test with nil signer
	checker := NewAWSSecretsManagerSignerChecker(nil)

	if checker == nil {
		t.Fatal("NewAWSSecretsManagerSignerChecker returned nil")
	}

	if checker.signer != nil {
		t.Error("Expected signer to be nil")
	}
}

func TestAWSSecretsManagerSignerCheckerName(t *testing.T) {
	checker := NewAWSSecretsManagerSignerChecker(nil)

	name := checker.Name()
	if name != "aws-secrets-manager-signer" {
		t.Errorf("Name() = %q, want %q", name, "aws-secrets-manager-signer")
	}
}

func TestAWSSecretsManagerSignerCheckerCheckWithNilSigner(t *testing.T) {
	checker := NewAWSSecretsManagerSignerChecker(nil)
	ctx := context.Background()

	result := checker.Check(ctx)

	if result == nil {
		t.Fatal("Check() returned nil")
	}

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want %v", result.Status, StatusUnhealthy)
	}

	if result.Message != "AWS Secrets Manager signer not configured" {
		t.Errorf("Message = %q, want %q", result.Message, "AWS Secrets Manager signer not configured")
	}

	if suggestion, ok := result.Details["suggestion"]; !ok {
		t.Error("Unhealthy result should include suggestion")
	} else if suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}

func TestAWSSecretsManagerCheckersImplementInterface(t *testing.T) {
	// Verify both checkers implement the Checker interface
	var _ Checker = (*AWSSecretsManagerChecker)(nil)
	var _ Checker = (*AWSSecretsManagerSignerChecker)(nil)
}
