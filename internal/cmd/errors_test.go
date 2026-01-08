package cmd

import (
	"errors"
	"strings"
	"testing"
)

// TestErrorWithSuggestion_Error tests the Error() method
func TestErrorWithSuggestion_Error(t *testing.T) {
	tests := []struct {
		name        string
		error       *ErrorWithSuggestion
		wantMessage string
		wantParts   []string
	}{
		{
			name: "message_only",
			error: &ErrorWithSuggestion{
				Message: "Test error",
			},
			wantMessage: "Test error",
			wantParts:   []string{"Test error"},
		},
		{
			name: "with_suggestions",
			error: &ErrorWithSuggestion{
				Message:     "Something went wrong",
				Suggestions: []string{"Try this", "Or try that"},
			},
			wantParts: []string{
				"Something went wrong",
				"Suggestions:",
				"Try this",
				"Or try that",
			},
		},
		{
			name: "with_underlying_error",
			error: &ErrorWithSuggestion{
				Message: "Failed operation",
				err:     errors.New("underlying cause"),
			},
			wantParts: []string{
				"Failed operation",
				"Details:",
				"underlying cause",
			},
		},
		{
			name: "full_error",
			error: &ErrorWithSuggestion{
				Message:     "Complete error",
				Suggestions: []string{"Suggestion 1"},
				err:         errors.New("wrapped error"),
			},
			wantParts: []string{
				"Complete error",
				"Suggestions:",
				"Suggestion 1",
				"Details:",
				"wrapped error",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.error.Error()

			for _, part := range tc.wantParts {
				if !strings.Contains(result, part) {
					t.Errorf("Error() = %q, missing expected part %q", result, part)
				}
			}
		})
	}
}

// TestErrorWithSuggestion_Unwrap tests the Unwrap() method
func TestErrorWithSuggestion_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")

	tests := []struct {
		name    string
		error   *ErrorWithSuggestion
		wantErr error
	}{
		{
			name: "with_underlying",
			error: &ErrorWithSuggestion{
				Message: "Test",
				err:     underlyingErr,
			},
			wantErr: underlyingErr,
		},
		{
			name: "without_underlying",
			error: &ErrorWithSuggestion{
				Message: "Test",
			},
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.error.Unwrap()
			if result != tc.wantErr {
				t.Errorf("Unwrap() = %v, want %v", result, tc.wantErr)
			}
		})
	}
}

// TestNewErrorWithSuggestions tests the factory function
func TestNewErrorWithSuggestions(t *testing.T) {
	underlyingErr := errors.New("cause")
	suggestions := []string{"Fix A", "Fix B"}

	err := NewErrorWithSuggestions("Something failed", underlyingErr, suggestions...)

	// Type assertion to check the underlying structure
	errWithSuggestions, ok := err.(*ErrorWithSuggestion)
	if !ok {
		t.Fatal("NewErrorWithSuggestions should return *ErrorWithSuggestion")
	}

	if errWithSuggestions.Message != "Something failed" {
		t.Errorf("Message = %q, want %q", errWithSuggestions.Message, "Something failed")
	}

	if len(errWithSuggestions.Suggestions) != 2 {
		t.Errorf("Suggestions count = %d, want 2", len(errWithSuggestions.Suggestions))
	}

	// Check unwrap works with errors.Is/As
	if !errors.Is(err, underlyingErr) {
		t.Error("errors.Is should match underlying error")
	}
}

// TestProfileLoadError tests profile loading error helper
func TestProfileLoadError(t *testing.T) {
	underlyingErr := errors.New("file not found")
	err := ProfileLoadError("my-profile", underlyingErr)

	errStr := err.Error()

	// Check message contains profile name
	if !strings.Contains(errStr, "my-profile") {
		t.Error("error should contain profile name")
	}

	// Check has suggestions
	if !strings.Contains(errStr, "Suggestions:") {
		t.Error("error should contain suggestions")
	}

	// Check specific suggestions
	expectedSuggestions := []string{
		"--list-profiles",
		"--profile default",
		"auto.profiles.yaml",
	}

	for _, sug := range expectedSuggestions {
		if !strings.Contains(errStr, sug) {
			t.Errorf("error should contain suggestion %q", sug)
		}
	}

	// Check underlying error is wrapped
	if !errors.Is(err, underlyingErr) {
		t.Error("underlying error should be wrapped")
	}
}

// TestProviderLoadError tests provider loading error helper
func TestProviderLoadError(t *testing.T) {
	underlyingErr := errors.New("permission denied")
	err := ProviderLoadError("/path/to/config", underlyingErr)

	errStr := err.Error()

	// Check message contains path
	if !strings.Contains(errStr, "/path/to/config") {
		t.Error("error should contain config path")
	}

	// Check specific suggestions
	expectedSuggestions := []string{
		"specular init",
		"providers.yaml",
		"API keys",
		"provider health",
	}

	for _, sug := range expectedSuggestions {
		if !strings.Contains(errStr, sug) {
			t.Errorf("error should contain suggestion %q", sug)
		}
	}
}

// TestRouterError tests router error helper
func TestRouterError(t *testing.T) {
	underlyingErr := errors.New("no providers available")
	err := RouterError(underlyingErr)

	errStr := err.Error()

	// Check message
	if !strings.Contains(errStr, "router") || !strings.Contains(errStr, "Failed") {
		t.Error("error should mention router creation failure")
	}

	// Check specific suggestions
	expectedSuggestions := []string{
		"provider is configured",
		"provider list",
		"provider health",
	}

	for _, sug := range expectedSuggestions {
		if !strings.Contains(errStr, sug) {
			t.Errorf("error should contain suggestion %q", sug)
		}
	}
}

// TestCheckpointNotFoundError tests checkpoint not found error helper
func TestCheckpointNotFoundError(t *testing.T) {
	err := CheckpointNotFoundError("checkpoint-123")

	errStr := err.Error()

	// Check message contains checkpoint ID
	if !strings.Contains(errStr, "checkpoint-123") {
		t.Error("error should contain checkpoint ID")
	}

	// Check specific suggestions
	expectedSuggestions := []string{
		"checkpoint list",
		"checkpoint show",
		"new session",
	}

	for _, sug := range expectedSuggestions {
		if !strings.Contains(errStr, sug) {
			t.Errorf("error should contain suggestion %q", sug)
		}
	}
}

// TestFileNotFoundError tests file not found error helper
func TestFileNotFoundError(t *testing.T) {
	t.Run("default_suggestions", func(t *testing.T) {
		err := FileNotFoundError("/path/to/file.yaml")
		errStr := err.Error()

		if !strings.Contains(errStr, "/path/to/file.yaml") {
			t.Error("error should contain file path")
		}

		if !strings.Contains(errStr, "ls -l") {
			t.Error("error should contain ls command suggestion")
		}
	})

	t.Run("with_custom_suggestions", func(t *testing.T) {
		err := FileNotFoundError("/path/to/spec.yaml", "Create spec: specular spec init", "Use template: --template rest-api")
		errStr := err.Error()

		if !strings.Contains(errStr, "specular spec init") {
			t.Error("error should contain custom suggestion")
		}

		if !strings.Contains(errStr, "--template rest-api") {
			t.Error("error should contain second custom suggestion")
		}
	})
}

// TestValidationError tests validation error helper
func TestValidationError(t *testing.T) {
	tests := []struct {
		field       string
		value       interface{}
		validValues string
	}{
		{"profile", "invalid-profile", "default, ci, strict"},
		{"timeout", -1, "positive integer"},
		{"output", "xyz", "json, yaml, table"},
	}

	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			err := ValidationError(tc.field, tc.value, tc.validValues)
			errStr := err.Error()

			if !strings.Contains(errStr, tc.field) {
				t.Errorf("error should contain field name %q", tc.field)
			}

			if !strings.Contains(errStr, tc.validValues) {
				t.Errorf("error should contain valid values %q", tc.validValues)
			}

			if !strings.Contains(errStr, "--help") {
				t.Error("error should suggest --help")
			}
		})
	}
}

// TestJSONOutputError tests JSON output error helper
func TestJSONOutputError(t *testing.T) {
	err := JSONOutputError()
	errStr := err.Error()

	if !strings.Contains(errStr, "JSON output") {
		t.Error("error should mention JSON output")
	}

	expectedSuggestions := []string{
		"--json",
		"logs",
	}

	for _, sug := range expectedSuggestions {
		if !strings.Contains(errStr, sug) {
			t.Errorf("error should contain suggestion %q", sug)
		}
	}
}

// TestPolicyNotFoundError tests policy not found error helper
func TestPolicyNotFoundError(t *testing.T) {
	err := PolicyNotFoundError("/custom/policy.yaml")
	errStr := err.Error()

	if !strings.Contains(errStr, "/custom/policy.yaml") {
		t.Error("error should contain policy path")
	}

	expectedSuggestions := []string{
		"specular init",
		"policy.yaml",
		"--policy",
		"--no-policy",
	}

	for _, sug := range expectedSuggestions {
		if !strings.Contains(errStr, sug) {
			t.Errorf("error should contain suggestion %q", sug)
		}
	}
}

// TestErrorWithSuggestion_BulletPoints tests that suggestions use bullet points
func TestErrorWithSuggestion_BulletPoints(t *testing.T) {
	err := &ErrorWithSuggestion{
		Message:     "Test error",
		Suggestions: []string{"First", "Second", "Third"},
	}

	errStr := err.Error()

	// Count bullet points
	bulletCount := strings.Count(errStr, "•")
	if bulletCount != 3 {
		t.Errorf("expected 3 bullet points, got %d", bulletCount)
	}
}
