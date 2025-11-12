package bundle

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestRegistryError_Error tests the Error method
func TestRegistryError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *RegistryError
		contains []string
	}{
		{
			name: "complete error with all fields",
			err: &RegistryError{
				Type:       ErrTypeAuthentication,
				Message:    "Authentication failed",
				Suggestion: "Check your credentials",
				Cause:      fmt.Errorf("401 Unauthorized"),
				Reference:  "ghcr.io/org/bundle:v1.0.0",
			},
			contains: []string{
				"[AUTHENTICATION]",
				"Authentication failed",
				"Suggestion:",
				"Check your credentials",
				"Cause:",
				"401 Unauthorized",
			},
		},
		{
			name: "error without suggestion",
			err: &RegistryError{
				Type:      ErrTypeNotFound,
				Message:   "Bundle not found",
				Cause:     fmt.Errorf("404 Not Found"),
				Reference: "docker.io/user/bundle:latest",
			},
			contains: []string{
				"[NOT_FOUND]",
				"Bundle not found",
				"Cause:",
				"404 Not Found",
			},
		},
		{
			name: "error without cause",
			err: &RegistryError{
				Type:       ErrTypeInvalidRef,
				Message:    "Invalid reference format",
				Suggestion: "Use format: registry/org/repo:tag",
				Reference:  "invalid-ref",
			},
			contains: []string{
				"[INVALID_REFERENCE]",
				"Invalid reference format",
				"Suggestion:",
				"Use format: registry/org/repo:tag",
			},
		},
		{
			name: "minimal error",
			err: &RegistryError{
				Type:    ErrTypeUnknown,
				Message: "Unknown error occurred",
			},
			contains: []string{
				"[UNKNOWN]",
				"Unknown error occurred",
			},
		},
		{
			name: "network error",
			err: &RegistryError{
				Type:       ErrTypeNetwork,
				Message:    "Connection timeout",
				Suggestion: "Check network connectivity",
				Cause:      fmt.Errorf("dial tcp: i/o timeout"),
				Reference:  "registry.example.com/bundle:v1",
			},
			contains: []string{
				"[NETWORK]",
				"Connection timeout",
				"Suggestion:",
				"Check network connectivity",
				"Cause:",
				"dial tcp: i/o timeout",
			},
		},
		{
			name: "permission error",
			err: &RegistryError{
				Type:       ErrTypePermission,
				Message:    "Insufficient permissions",
				Suggestion: "Verify your access rights",
				Cause:      fmt.Errorf("403 Forbidden"),
				Reference:  "private.registry.com/org/bundle",
			},
			contains: []string{
				"[PERMISSION]",
				"Insufficient permissions",
				"Suggestion:",
				"Verify your access rights",
				"Cause:",
				"403 Forbidden",
			},
		},
		{
			name: "invalid bundle error",
			err: &RegistryError{
				Type:       ErrTypeInvalidBundle,
				Message:    "Not a valid Specular bundle",
				Suggestion: "Ensure the artifact was created with 'specular bundle build'",
				Reference:  "ghcr.io/org/not-a-bundle:latest",
			},
			contains: []string{
				"[INVALID_BUNDLE]",
				"Not a valid Specular bundle",
				"Suggestion:",
				"'specular bundle build'",
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

// TestRegistryError_Error_MessageFormat tests the error message format
func TestRegistryError_Error_MessageFormat(t *testing.T) {
	err := &RegistryError{
		Type:       ErrTypeAuthentication,
		Message:    "test message",
		Suggestion: "test suggestion",
		Cause:      fmt.Errorf("test cause"),
	}

	msg := err.Error()

	// Check that suggestion and cause are separated by double newlines
	if !strings.Contains(msg, "\n\nSuggestion:") {
		t.Error("Suggestion should be separated by double newline")
	}

	if !strings.Contains(msg, "\n\nCause:") {
		t.Error("Cause should be separated by double newline")
	}

	// Check that type appears in brackets
	if !strings.HasPrefix(msg, "[AUTHENTICATION]") {
		t.Errorf("Error should start with type in brackets, got: %s", msg)
	}
}

// TestRegistryError_Unwrap tests the Unwrap method
func TestRegistryError_Unwrap(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		causeErr := fmt.Errorf("underlying error")
		registryErr := &RegistryError{
			Type:    ErrTypeNetwork,
			Message: "Network error",
			Cause:   causeErr,
		}

		unwrapped := registryErr.Unwrap()
		if unwrapped != causeErr {
			t.Errorf("Unwrap() = %v, want %v", unwrapped, causeErr)
		}

		// Test errors.Is works correctly
		if !errors.Is(registryErr, causeErr) {
			t.Error("errors.Is() should return true for wrapped error")
		}
	})

	t.Run("without cause", func(t *testing.T) {
		registryErr := &RegistryError{
			Type:    ErrTypeNotFound,
			Message: "Not found",
			Cause:   nil,
		}

		unwrapped := registryErr.Unwrap()
		if unwrapped != nil {
			t.Errorf("Unwrap() = %v, want nil", unwrapped)
		}
	})

	t.Run("error chaining with errors.As", func(t *testing.T) {
		// Create a custom error instance
		customError := fmt.Errorf("custom error: %d", 42)
		registryErr := &RegistryError{
			Type:    ErrTypeUnknown,
			Message: "Wrapped custom error",
			Cause:   customError,
		}

		// Test errors.Is works correctly
		if !errors.Is(registryErr, customError) {
			t.Error("errors.Is() should work with wrapped errors")
		}
	})
}

// TestRegistryError_AllTypes tests all error types
func TestRegistryError_AllTypes(t *testing.T) {
	errorTypes := []struct {
		typ      RegistryErrorType
		typeName string
	}{
		{ErrTypeAuthentication, "AUTHENTICATION"},
		{ErrTypeNotFound, "NOT_FOUND"},
		{ErrTypeNetwork, "NETWORK"},
		{ErrTypePermission, "PERMISSION"},
		{ErrTypeInvalidRef, "INVALID_REFERENCE"},
		{ErrTypeInvalidBundle, "INVALID_BUNDLE"},
		{ErrTypeUnknown, "UNKNOWN"},
	}

	for _, tt := range errorTypes {
		t.Run(string(tt.typ), func(t *testing.T) {
			err := &RegistryError{
				Type:    tt.typ,
				Message: "test message",
			}

			msg := err.Error()
			if !strings.Contains(msg, "["+tt.typeName+"]") {
				t.Errorf("Error message should contain [%s], got: %s", tt.typeName, msg)
			}
		})
	}
}

// TestRegistryError_EmptyFields tests behavior with empty fields
func TestRegistryError_EmptyFields(t *testing.T) {
	t.Run("empty message", func(t *testing.T) {
		err := &RegistryError{
			Type:    ErrTypeNetwork,
			Message: "",
		}

		msg := err.Error()
		if !strings.Contains(msg, "[NETWORK]") {
			t.Errorf("Error should contain type even with empty message: %s", msg)
		}
	})

	t.Run("empty suggestion", func(t *testing.T) {
		err := &RegistryError{
			Type:       ErrTypeAuthentication,
			Message:    "auth failed",
			Suggestion: "",
			Cause:      fmt.Errorf("401"),
		}

		msg := err.Error()
		// Should not contain "Suggestion:" header when suggestion is empty
		if strings.Contains(msg, "Suggestion:") {
			t.Error("Error should not contain 'Suggestion:' when suggestion is empty")
		}
		// But should still contain cause
		if !strings.Contains(msg, "Cause:") {
			t.Error("Error should contain 'Cause:' when cause is present")
		}
	})

	t.Run("empty reference", func(t *testing.T) {
		err := &RegistryError{
			Type:      ErrTypeNotFound,
			Message:   "not found",
			Reference: "",
		}

		// Should not panic with empty reference
		msg := err.Error()
		if msg == "" {
			t.Error("Error() should not return empty string")
		}
	})
}
