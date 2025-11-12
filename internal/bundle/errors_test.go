package bundle

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestBundleError_Error tests the Error method of BundleError
func TestBundleError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *BundleError
		contains []string
	}{
		{
			name: "full error with all fields",
			err: &BundleError{
				Operation:  "build",
				Message:    "failed to create bundle",
				Suggestion: "check the manifest file",
				Details:    "manifest.yaml not found",
				Cause:      fmt.Errorf("file not found"),
			},
			contains: []string{
				"bundle build failed",
				"failed to create bundle",
				"Suggestion:",
				"check the manifest file",
				"Details:",
				"manifest.yaml not found",
				"Cause:",
				"file not found",
			},
		},
		{
			name: "minimal error with only message",
			err: &BundleError{
				Message: "something went wrong",
			},
			contains: []string{
				"something went wrong",
			},
		},
		{
			name: "error with operation but no suggestion",
			err: &BundleError{
				Operation: "verify",
				Message:   "verification failed",
			},
			contains: []string{
				"bundle verify failed",
				"verification failed",
			},
		},
		{
			name: "error with details but no cause",
			err: &BundleError{
				Operation: "apply",
				Message:   "failed to apply bundle",
				Details:   "permission denied",
			},
			contains: []string{
				"bundle apply failed",
				"failed to apply bundle",
				"Details:",
				"permission denied",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()

			for _, substr := range tt.contains {
				if !strings.Contains(errMsg, substr) {
					t.Errorf("Error() message missing expected substring %q\nGot: %s", substr, errMsg)
				}
			}
		})
	}
}

// TestBundleError_Unwrap tests the Unwrap method
func TestBundleError_Unwrap(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		causeErr := fmt.Errorf("root cause error")
		bundleErr := &BundleError{
			Message: "bundle error",
			Cause:   causeErr,
		}

		unwrapped := bundleErr.Unwrap()
		if unwrapped != causeErr {
			t.Errorf("Unwrap() = %v, want %v", unwrapped, causeErr)
		}

		// Test errors.Is works correctly
		if !errors.Is(bundleErr, causeErr) {
			t.Error("errors.Is() should return true for wrapped error")
		}
	})

	t.Run("without cause", func(t *testing.T) {
		bundleErr := &BundleError{
			Message: "bundle error",
			Cause:   nil,
		}

		unwrapped := bundleErr.Unwrap()
		if unwrapped != nil {
			t.Errorf("Unwrap() = %v, want nil", unwrapped)
		}
	})
}

// TestErrInvalidManifest tests the ErrInvalidManifest constructor
func TestErrInvalidManifest(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		reason := "missing required field 'schema'"
		causeErr := fmt.Errorf("yaml parse error")

		err := ErrInvalidManifest(reason, causeErr)

		if err.Operation != "validate" {
			t.Errorf("Operation = %s, want validate", err.Operation)
		}

		if !strings.Contains(err.Message, reason) {
			t.Errorf("Message should contain reason %q, got %s", reason, err.Message)
		}

		if err.Suggestion == "" {
			t.Error("Suggestion should not be empty")
		}

		if !strings.Contains(err.Suggestion, "manifest.yaml") {
			t.Errorf("Suggestion should mention manifest.yaml, got %s", err.Suggestion)
		}

		if err.Details != reason {
			t.Errorf("Details = %s, want %s", err.Details, reason)
		}

		if err.Cause != causeErr {
			t.Errorf("Cause = %v, want %v", err.Cause, causeErr)
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		reason := "invalid version format"
		err := ErrInvalidManifest(reason, nil)

		if err.Cause != nil {
			t.Errorf("Cause = %v, want nil", err.Cause)
		}

		if !strings.Contains(err.Message, reason) {
			t.Errorf("Message should contain reason %q, got %s", reason, err.Message)
		}
	})
}

// TestErrChecksumMismatch tests the ErrChecksumMismatch constructor
func TestErrChecksumMismatch(t *testing.T) {
	file := "spec.yaml"
	expected := "abc123"
	actual := "def456"

	err := ErrChecksumMismatch(file, expected, actual)

	if err.Operation != "verify" {
		t.Errorf("Operation = %s, want verify", err.Operation)
	}

	if !strings.Contains(err.Message, file) {
		t.Errorf("Message should contain file %q, got %s", file, err.Message)
	}

	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty")
	}

	if !strings.Contains(err.Suggestion, "modified") {
		t.Errorf("Suggestion should mention modification, got %s", err.Suggestion)
	}

	if !strings.Contains(err.Details, expected) {
		t.Errorf("Details should contain expected checksum %q, got %s", expected, err.Details)
	}

	if !strings.Contains(err.Details, actual) {
		t.Errorf("Details should contain actual checksum %q, got %s", actual, err.Details)
	}
}

// TestErrMissingApproval tests the ErrMissingApproval constructor
func TestErrMissingApproval(t *testing.T) {
	role := "security"

	err := ErrMissingApproval(role)

	if err.Operation != "verify" {
		t.Errorf("Operation = %s, want verify", err.Operation)
	}

	if !strings.Contains(err.Message, role) {
		t.Errorf("Message should contain role %q, got %s", role, err.Message)
	}

	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty")
	}

	if !strings.Contains(err.Suggestion, "specular bundle approve") {
		t.Errorf("Suggestion should mention approval command, got %s", err.Suggestion)
	}

	if !strings.Contains(err.Suggestion, role) {
		t.Errorf("Suggestion should contain role %q, got %s", role, err.Suggestion)
	}

	if !strings.Contains(err.Details, role) {
		t.Errorf("Details should contain role %q, got %s", role, err.Details)
	}
}

// customError is a simple error type for testing error chaining
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

// TestBundleError_ErrorChaining tests error chaining with errors.Is and errors.As
func TestBundleError_ErrorChaining(t *testing.T) {
	// Create a custom error instance
	customErr := &customError{msg: "custom error"}
	bundleErr := &BundleError{
		Message: "bundle error",
		Cause:   customErr,
	}

	// Test errors.Is
	if !errors.Is(bundleErr, customErr) {
		t.Error("errors.Is() should work with wrapped custom errors")
	}

	// Test errors.As
	var targetErr *customError
	if !errors.As(bundleErr, &targetErr) {
		t.Error("errors.As() should work with wrapped custom errors")
	}

	if targetErr.msg != "custom error" {
		t.Errorf("errors.As() returned wrong error: got %q, want %q", targetErr.msg, "custom error")
	}
}
