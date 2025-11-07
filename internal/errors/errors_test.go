package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(ErrCodeSpecNotFound, "test error message")

	if err.Code != ErrCodeSpecNotFound {
		t.Errorf("expected code %s, got %s", ErrCodeSpecNotFound, err.Code)
	}

	if err.Message != "test error message" {
		t.Errorf("expected message 'test error message', got '%s'", err.Message)
	}

	if err.Cause != nil {
		t.Errorf("expected nil cause, got %v", err.Cause)
	}
}

func TestWrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := Wrap(ErrCodeFileReadFailed, "failed to read file", cause)

	if err.Code != ErrCodeFileReadFailed {
		t.Errorf("expected code %s, got %s", ErrCodeFileReadFailed, err.Code)
	}

	if err.Cause != cause {
		t.Errorf("expected cause to be set")
	}

	// Test unwrapping
	if !errors.Is(err, cause) {
		t.Errorf("Wrap should support errors.Is")
	}
}

func TestErrorFormatting(t *testing.T) {
	tests := []struct {
		name     string
		err      *SpecularError
		wantCode string
		wantMsg  string
	}{
		{
			name:     "simple error",
			err:      New(ErrCodeSpecInvalid, "invalid spec"),
			wantCode: "SPEC-002",
			wantMsg:  "invalid spec",
		},
		{
			name:     "error with cause",
			err:      Wrap(ErrCodeFileReadFailed, "read failed", fmt.Errorf("permission denied")),
			wantCode: "IO-002",
			wantMsg:  "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()

			if !strings.Contains(errStr, tt.wantCode) {
				t.Errorf("error string should contain code %s, got: %s", tt.wantCode, errStr)
			}

			if !strings.Contains(errStr, tt.wantMsg) {
				t.Errorf("error string should contain message '%s', got: %s", tt.wantMsg, errStr)
			}
		})
	}
}

func TestWithSuggestion(t *testing.T) {
	err := New(ErrCodeSpecNotFound, "spec not found").
		WithSuggestion("Check the file path")

	if len(err.Suggestions) != 1 {
		t.Errorf("expected 1 suggestion, got %d", len(err.Suggestions))
	}

	if err.Suggestions[0] != "Check the file path" {
		t.Errorf("unexpected suggestion: %s", err.Suggestions[0])
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "Suggestions:") {
		t.Errorf("error string should contain suggestions section")
	}

	if !strings.Contains(errStr, "Check the file path") {
		t.Errorf("error string should contain suggestion text")
	}
}

func TestWithSuggestions(t *testing.T) {
	err := New(ErrCodePolicyViolation, "policy violated").
		WithSuggestions("Suggestion 1", "Suggestion 2", "Suggestion 3")

	if len(err.Suggestions) != 3 {
		t.Errorf("expected 3 suggestions, got %d", len(err.Suggestions))
	}

	errStr := err.Error()
	for _, suggestion := range err.Suggestions {
		if !strings.Contains(errStr, suggestion) {
			t.Errorf("error string should contain suggestion: %s", suggestion)
		}
	}
}

func TestWithDocs(t *testing.T) {
	docsURL := "https://github.com/felixgeelhaar/specular#docs"
	err := New(ErrCodeSpecInvalid, "invalid spec").
		WithDocs(docsURL)

	if err.DocsURL != docsURL {
		t.Errorf("expected DocsURL %s, got %s", docsURL, err.DocsURL)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "Documentation:") {
		t.Errorf("error string should contain documentation section")
	}

	if !strings.Contains(errStr, docsURL) {
		t.Errorf("error string should contain docs URL")
	}
}

func TestNewSpecNotFoundError(t *testing.T) {
	err := NewSpecNotFoundError("/path/to/spec.yaml")

	if err.Code != ErrCodeSpecNotFound {
		t.Errorf("expected code %s, got %s", ErrCodeSpecNotFound, err.Code)
	}

	if !strings.Contains(err.Message, "/path/to/spec.yaml") {
		t.Errorf("error message should contain file path")
	}

	if len(err.Suggestions) < 2 {
		t.Errorf("expected at least 2 suggestions, got %d", len(err.Suggestions))
	}

	if err.DocsURL == "" {
		t.Errorf("expected docs URL to be set")
	}
}

func TestNewSpecInvalidError(t *testing.T) {
	err := NewSpecInvalidError("missing required field 'product'")

	if err.Code != ErrCodeSpecInvalid {
		t.Errorf("expected code %s, got %s", ErrCodeSpecInvalid, err.Code)
	}

	if !strings.Contains(err.Message, "missing required field") {
		t.Errorf("error message should contain details")
	}

	if len(err.Suggestions) == 0 {
		t.Errorf("expected suggestions to be provided")
	}
}

func TestNewPolicyViolationError(t *testing.T) {
	err := NewPolicyViolationError("Docker image not in allowlist")

	if err.Code != ErrCodePolicyViolation {
		t.Errorf("expected code %s, got %s", ErrCodePolicyViolation, err.Code)
	}

	if !strings.Contains(err.Message, "Docker image") {
		t.Errorf("error message should contain violation details")
	}

	if len(err.Suggestions) == 0 {
		t.Errorf("expected suggestions for policy violations")
	}
}

func TestNewProviderAuthError(t *testing.T) {
	err := NewProviderAuthError("openai")

	if err.Code != ErrCodeProviderAuth {
		t.Errorf("expected code %s, got %s", ErrCodeProviderAuth, err.Code)
	}

	if !strings.Contains(err.Message, "openai") {
		t.Errorf("error message should contain provider name")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "OPENAI_API_KEY") {
		t.Errorf("suggestions should mention API key env variable")
	}

	if len(err.Suggestions) < 3 {
		t.Errorf("expected at least 3 suggestions for auth errors")
	}
}

func TestNewProviderRateLimitError(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		retryAfter string
		wantRetry  bool
	}{
		{
			name:       "with retry after",
			provider:   "anthropic",
			retryAfter: "60s",
			wantRetry:  true,
		},
		{
			name:       "without retry after",
			provider:   "openai",
			retryAfter: "",
			wantRetry:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewProviderRateLimitError(tt.provider, tt.retryAfter)

			if err.Code != ErrCodeProviderRateLimit {
				t.Errorf("expected code %s, got %s", ErrCodeProviderRateLimit, err.Code)
			}

			if !strings.Contains(err.Message, tt.provider) {
				t.Errorf("error message should contain provider name")
			}

			if tt.wantRetry && !strings.Contains(err.Message, tt.retryAfter) {
				t.Errorf("error message should contain retry after time")
			}
		})
	}
}

func TestNewExecDockerNotAvailableError(t *testing.T) {
	err := NewExecDockerNotAvailableError()

	if err.Code != ErrCodeExecDockerNotAvailable {
		t.Errorf("expected code %s, got %s", ErrCodeExecDockerNotAvailable, err.Code)
	}

	if len(err.Suggestions) < 3 {
		t.Errorf("expected at least 3 suggestions for Docker issues")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "docker version") {
		t.Errorf("suggestions should mention docker version command")
	}
}

func TestNewInterviewPresetUnknownError(t *testing.T) {
	err := NewInterviewPresetUnknownError("invalid-preset")

	if err.Code != ErrCodeInterviewPresetUnknown {
		t.Errorf("expected code %s, got %s", ErrCodeInterviewPresetUnknown, err.Code)
	}

	if !strings.Contains(err.Message, "invalid-preset") {
		t.Errorf("error message should contain preset name")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "--list") {
		t.Errorf("suggestions should mention --list flag")
	}
}

func TestNewInterviewAnswerRequiredError(t *testing.T) {
	err := NewInterviewAnswerRequiredError("What is the product name?")

	if err.Code != ErrCodeInterviewAnswerRequired {
		t.Errorf("expected code %s, got %s", ErrCodeInterviewAnswerRequired, err.Code)
	}

	if !strings.Contains(err.Message, "product name") {
		t.Errorf("error message should contain question text")
	}
}

func TestNewInterviewAnswerInvalidError(t *testing.T) {
	err := NewInterviewAnswerInvalidError("Choose your priority", "P0, P1, or P2")

	if err.Code != ErrCodeInterviewAnswerInvalid {
		t.Errorf("expected code %s, got %s", ErrCodeInterviewAnswerInvalid, err.Code)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "P0, P1, or P2") {
		t.Errorf("suggestions should contain expected values")
	}
}

func TestNewPlanDriftError(t *testing.T) {
	err := NewPlanDriftError("feature-001", "abc123", "def456")

	if err.Code != ErrCodePlanDriftDetected {
		t.Errorf("expected code %s, got %s", ErrCodePlanDriftDetected, err.Code)
	}

	if !strings.Contains(err.Message, "feature-001") {
		t.Errorf("error message should contain feature ID")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "abc123") || !strings.Contains(errStr, "def456") {
		t.Errorf("suggestions should contain hash values")
	}
}

func TestNewFileNotFoundError(t *testing.T) {
	err := NewFileNotFoundError("/path/to/file.yaml")

	if err.Code != ErrCodeFileNotFound {
		t.Errorf("expected code %s, got %s", ErrCodeFileNotFound, err.Code)
	}

	if !strings.Contains(err.Message, "/path/to/file.yaml") {
		t.Errorf("error message should contain file path")
	}
}

func TestNewFileUnmarshalError(t *testing.T) {
	cause := fmt.Errorf("invalid YAML syntax at line 5")
	err := NewFileUnmarshalError("/path/to/spec.yaml", "YAML", cause)

	if err.Code != ErrCodeFileUnmarshal {
		t.Errorf("expected code %s, got %s", ErrCodeFileUnmarshal, err.Code)
	}

	if err.Cause != cause {
		t.Errorf("expected cause to be preserved")
	}

	if !strings.Contains(err.Message, "YAML") {
		t.Errorf("error message should contain format")
	}

	if !strings.Contains(err.Message, "/path/to/spec.yaml") {
		t.Errorf("error message should contain file path")
	}
}

func TestErrorChaining(t *testing.T) {
	// Test that errors can be chained with suggestions and docs
	err := New(ErrCodeSpecInvalid, "validation failed").
		WithSuggestion("Check field 'product'").
		WithSuggestion("Check field 'features'").
		WithDocs("https://example.com/docs")

	if len(err.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(err.Suggestions))
	}

	if err.DocsURL == "" {
		t.Errorf("expected docs URL to be set")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "SPEC-002") {
		t.Errorf("error should contain code")
	}

	if !strings.Contains(errStr, "Check field 'product'") {
		t.Errorf("error should contain first suggestion")
	}

	if !strings.Contains(errStr, "Check field 'features'") {
		t.Errorf("error should contain second suggestion")
	}

	if !strings.Contains(errStr, "https://example.com/docs") {
		t.Errorf("error should contain docs URL")
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := Wrap(ErrCodeFileReadFailed, "read failed", cause)

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Errorf("Unwrap should return the cause")
	}

	// Test errors.Is
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is should work with wrapped errors")
	}
}

func TestErrorCodes(t *testing.T) {
	// Test that all error codes follow the expected pattern
	codes := []ErrorCode{
		// Spec codes
		ErrCodeSpecNotFound,
		ErrCodeSpecInvalid,
		ErrCodeSpecUnmarshal,
		ErrCodeSpecMarshal,

		// Policy codes
		ErrCodePolicyNotFound,
		ErrCodePolicyInvalid,
		ErrCodePolicyViolation,

		// Plan codes
		ErrCodePlanNotFound,
		ErrCodePlanInvalid,
		ErrCodePlanDriftDetected,

		// Interview codes
		ErrCodeInterviewPresetUnknown,
		ErrCodeInterviewAlreadyStarted,
		ErrCodeInterviewNotComplete,

		// Provider codes
		ErrCodeProviderNotFound,
		ErrCodeProviderAuth,
		ErrCodeProviderRateLimit,

		// Execution codes
		ErrCodeExecDockerNotAvailable,
		ErrCodeExecImagePullFailed,
		ErrCodeExecContainerFailed,

		// I/O codes
		ErrCodeFileNotFound,
		ErrCodeFileReadFailed,
		ErrCodeFileWriteFailed,
	}

	for _, code := range codes {
		codeStr := string(code)

		// Check format: CATEGORY-NNN
		if !strings.Contains(codeStr, "-") {
			t.Errorf("error code %s should contain hyphen", code)
		}

		parts := strings.Split(codeStr, "-")
		if len(parts) != 2 {
			t.Errorf("error code %s should have format CATEGORY-NNN", code)
		}

		// Check that number part is 3 digits
		if len(parts[1]) != 3 {
			t.Errorf("error code %s should have 3-digit number", code)
		}
	}
}
