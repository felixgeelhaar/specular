package exitcode

import (
	"errors"
	"testing"
)

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"Success", Success, 0},
		{"GeneralError", GeneralError, 1},
		{"UsageError", UsageError, 2},
		{"PolicyViolation", PolicyViolation, 3},
		{"DriftDetected", DriftDetected, 4},
		{"AuthError", AuthError, 5},
		{"NetworkError", NetworkError, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("Exit code %s = %d, want %d", tt.name, tt.code, tt.expected)
			}
		})
	}
}

func TestDetermineExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error returns success",
			err:      nil,
			expected: Success,
		},
		{
			name:     "policy violation error",
			err:      errors.New("policy violation: unauthorized image"),
			expected: PolicyViolation,
		},
		{
			name:     "not allowed by policy",
			err:      errors.New("image not allowed by policy"),
			expected: PolicyViolation,
		},
		{
			name:     "drift detected error",
			err:      errors.New("drift detected in specification"),
			expected: DriftDetected,
		},
		{
			name:     "hash mismatch drift",
			err:      errors.New("hash mismatch detected"),
			expected: DriftDetected,
		},
		{
			name:     "authentication error",
			err:      errors.New("authentication failed: invalid token"),
			expected: AuthError,
		},
		{
			name:     "unauthorized error",
			err:      errors.New("unauthorized access"),
			expected: AuthError,
		},
		{
			name:     "api key error",
			err:      errors.New("invalid api key provided"),
			expected: AuthError,
		},
		{
			name:     "network error",
			err:      errors.New("network error: connection timeout"),
			expected: NetworkError,
		},
		{
			name:     "connection error",
			err:      errors.New("connection refused"),
			expected: NetworkError,
		},
		{
			name:     "timeout error",
			err:      errors.New("request timeout"),
			expected: NetworkError,
		},
		{
			name:     "usage error - invalid flag",
			err:      errors.New("invalid flag: --foo"),
			expected: UsageError,
		},
		{
			name:     "usage error - required flag",
			err:      errors.New("required flag --input not set"),
			expected: UsageError,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: GeneralError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := DetermineExitCode(tt.err)
			if code != tt.expected {
				t.Errorf("DetermineExitCode(%v) = %d, want %d", tt.err, code, tt.expected)
			}
		})
	}
}

func TestGetExitCodeDescription(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{Success, "Success"},
		{GeneralError, "General error"},
		{UsageError, "Usage error (invalid flags or arguments)"},
		{PolicyViolation, "Policy violation"},
		{DriftDetected, "Configuration drift detected"},
		{AuthError, "Authentication error"},
		{NetworkError, "Network error"},
		{99, "Unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GetExitCodeDescription(tt.code)
			if result != tt.expected {
				t.Errorf("GetExitCodeDescription(%d) = %s, want %s", tt.code, result, tt.expected)
			}
		})
	}
}
