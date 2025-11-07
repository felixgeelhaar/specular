package ux

import (
	"errors"
	"strings"
	"testing"
)

func TestNewErrorWithSuggestion(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		suggestion string
		wantNil    bool
	}{
		{
			name:       "nil error returns nil",
			err:        nil,
			suggestion: "some suggestion",
			wantNil:    true,
		},
		{
			name:       "error with suggestion",
			err:        errors.New("something failed"),
			suggestion: "try this fix",
			wantNil:    false,
		},
		{
			name:       "error without suggestion",
			err:        errors.New("something failed"),
			suggestion: "",
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewErrorWithSuggestion(tt.err, tt.suggestion)
			if tt.wantNil {
				if result != nil {
					t.Errorf("NewErrorWithSuggestion() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("NewErrorWithSuggestion() returned nil, want error")
			}

			errMsg := result.Error()
			if !strings.Contains(errMsg, tt.err.Error()) {
				t.Errorf("Error message %q does not contain original error %q", errMsg, tt.err.Error())
			}

			if tt.suggestion != "" && !strings.Contains(errMsg, tt.suggestion) {
				t.Errorf("Error message %q does not contain suggestion %q", errMsg, tt.suggestion)
			}
		})
	}
}

func TestErrorWithSuggestion_Error(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		suggestion string
		wantMsg    string
	}{
		{
			name:       "with suggestion",
			err:        errors.New("test error"),
			suggestion: "do this",
			wantMsg:    "test error\n\n💡 Suggestion: do this",
		},
		{
			name:       "without suggestion",
			err:        errors.New("test error"),
			suggestion: "",
			wantMsg:    "test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ErrorWithSuggestion{
				Err:        tt.err,
				Suggestion: tt.suggestion,
			}

			if e.Error() != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", e.Error(), tt.wantMsg)
			}
		})
	}
}

func TestErrorWithSuggestion_Unwrap(t *testing.T) {
	origErr := errors.New("original error")
	e := &ErrorWithSuggestion{
		Err:        origErr,
		Suggestion: "some suggestion",
	}

	unwrapped := e.Unwrap()
	if unwrapped != origErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, origErr)
	}
}

func TestEnhanceError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantNil        bool
		wantSuggestion string
	}{
		{
			name:    "nil error returns nil",
			err:     nil,
			wantNil: true,
		},
		{
			name:           "spec.yaml not found",
			err:            errors.New("open spec.yaml: no such file or directory"),
			wantSuggestion: "specular interview",
		},
		{
			name:           "spec.lock.json not found",
			err:            errors.New("open spec.lock.json: no such file or directory"),
			wantSuggestion: "specular spec lock",
		},
		{
			name:           "plan.json not found",
			err:            errors.New("open plan.json: no such file or directory"),
			wantSuggestion: "specular plan",
		},
		{
			name:           "policy.yaml not found",
			err:            errors.New("open policy.yaml: no such file or directory"),
			wantSuggestion: "policy.yaml",
		},
		{
			name:           "providers.yaml not found",
			err:            errors.New("open providers.yaml: no such file or directory"),
			wantSuggestion: "specular init",
		},
		{
			name:           "docker daemon error",
			err:            errors.New("error connecting to docker daemon"),
			wantSuggestion: "Docker Desktop",
		},
		{
			name:           "docker connection error",
			err:            errors.New("Cannot connect to the Docker daemon at unix:///var/run/docker.sock"),
			wantSuggestion: "Start Docker",
		},
		{
			name:           "docker socket permission denied",
			err:            errors.New("permission denied while trying to connect to /var/run/docker.sock"),
			wantSuggestion: "docker group",
		},
		{
			name:           "generic permission denied",
			err:            errors.New("permission denied: access forbidden"),
			wantSuggestion: "file permissions",
		},
		{
			name:           "no providers available",
			err:            errors.New("no providers available for routing"),
			wantSuggestion: "Configure at least one AI provider",
		},
		{
			name:           "provider not found",
			err:            errors.New("provider openai not found"),
			wantSuggestion: "provider configuration",
		},
		{
			name:           "provider not configured",
			err:            errors.New("provider anthropic not configured"),
			wantSuggestion: "router.yaml",
		},
		{
			name:           "policy violation",
			err:            errors.New("policy violation: unauthorized image"),
			wantSuggestion: "Docker-only execution",
		},
		{
			name:           "docker_only policy",
			err:            errors.New("docker_only policy requires containerized execution"),
			wantSuggestion: "Docker is running",
		},
		{
			name:           "validation failed",
			err:            errors.New("validation failed: invalid spec format"),
			wantSuggestion: "specular spec validate",
		},
		{
			name:           "drift detected",
			err:            errors.New("drift detected in code implementation"),
			wantSuggestion: "specular build drift",
		},
		{
			name:           "connection refused",
			err:            errors.New("connection refused: dial tcp"),
			wantSuggestion: "network connection",
		},
		{
			name:           "no route to host",
			err:            errors.New("no route to host"),
			wantSuggestion: "firewall settings",
		},
		{
			name:           "API key error",
			err:            errors.New("invalid API key provided"),
			wantSuggestion: "API key environment variable",
		},
		{
			name:           "authentication error",
			err:            errors.New("authentication failed: invalid credentials"),
			wantSuggestion: "ANTHROPIC_API_KEY",
		},
		{
			name:           "generic failed to error",
			err:            errors.New("failed to execute command"),
			wantSuggestion: "Next steps",
		},
		{
			name:           "unrecognized error unchanged",
			err:            errors.New("some random error"),
			wantSuggestion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnhanceError(tt.err)

			if tt.wantNil {
				if result != nil {
					t.Errorf("EnhanceError() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("EnhanceError() returned nil, want error")
			}

			errMsg := result.Error()

			// Original error should be preserved
			if !strings.Contains(errMsg, tt.err.Error()) {
				t.Errorf("Enhanced error %q does not contain original error %q", errMsg, tt.err.Error())
			}

			// Check for expected suggestion
			if tt.wantSuggestion != "" {
				if !strings.Contains(errMsg, tt.wantSuggestion) {
					t.Errorf("Enhanced error %q does not contain expected suggestion %q", errMsg, tt.wantSuggestion)
				}
			}
		})
	}
}

func TestFormatError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		context     string
		wantNil     bool
		wantContext bool
	}{
		{
			name:    "nil error returns nil",
			err:     nil,
			context: "some context",
			wantNil: true,
		},
		{
			name:        "error with context",
			err:         errors.New("something failed"),
			context:     "while processing file",
			wantContext: true,
		},
		{
			name:        "error without context",
			err:         errors.New("something failed"),
			context:     "",
			wantContext: false,
		},
		{
			name:        "enhances and adds context",
			err:         errors.New("open spec.yaml: no such file or directory"),
			context:     "loading specification",
			wantContext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatError(tt.err, tt.context)

			if tt.wantNil {
				if result != nil {
					t.Errorf("FormatError() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("FormatError() returned nil, want error")
			}

			errMsg := result.Error()

			if tt.wantContext && tt.context != "" {
				if !strings.Contains(errMsg, tt.context) {
					t.Errorf("Formatted error %q does not contain context %q", errMsg, tt.context)
				}
			}

			// Should still contain original error message
			if !strings.Contains(errMsg, tt.err.Error()) {
				t.Errorf("Formatted error %q does not contain original error %q", errMsg, tt.err.Error())
			}
		})
	}
}

func TestEnhanceError_PreservesErrorChain(t *testing.T) {
	// Create a wrapped error chain
	baseErr := errors.New("base error")
	wrappedErr := NewErrorWithSuggestion(baseErr, "first suggestion")

	// Enhance it again
	enhanced := EnhanceError(wrappedErr)

	// Should be able to unwrap to get original
	if enhanced == nil {
		t.Fatal("EnhanceError() returned nil")
	}

	// EnhanceError returns the original error if it doesn't match any patterns
	// So for an unrecognized ErrorWithSuggestion, it should return it unchanged
	if enhanced.Error() != wrappedErr.Error() {
		t.Errorf("EnhanceError() changed error message: got %q, want %q", enhanced.Error(), wrappedErr.Error())
	}
}
