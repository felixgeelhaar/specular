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
		// Additional policy violation tests
		{
			name:     "budget exceeded",
			err:      errors.New("budget exceeded: $10.00 > $5.00"),
			expected: PolicyViolation,
		},
		{
			name:     "cost limit",
			err:      errors.New("cost limit reached"),
			expected: PolicyViolation,
		},
		{
			name:     "step type blocked",
			err:      errors.New("step type blocked by profile"),
			expected: PolicyViolation,
		},
		{
			name:     "operation blocked",
			err:      errors.New("operation blocked by security policy"),
			expected: PolicyViolation,
		},
		{
			name:     "restricted operation",
			err:      errors.New("operation is restricted"),
			expected: PolicyViolation,
		},
		// Additional drift detection tests
		{
			name:     "spec changed",
			err:      errors.New("spec changed since lock"),
			expected: DriftDetected,
		},
		{
			name:     "verification failed",
			err:      errors.New("verification failed for spec"),
			expected: DriftDetected,
		},
		{
			name:     "checksum mismatch",
			err:      errors.New("checksum mismatch detected"),
			expected: DriftDetected,
		},
		// Additional auth errors
		{
			name:     "forbidden",
			err:      errors.New("forbidden: insufficient permissions"),
			expected: AuthError,
		},
		{
			name:     "permission denied",
			err:      errors.New("permission denied"),
			expected: AuthError,
		},
		{
			name:     "expired token",
			err:      errors.New("expired token, please re-authenticate"),
			expected: AuthError,
		},
		{
			name:     "401 http error",
			err:      errors.New("HTTP 401 Unauthorized"),
			expected: AuthError,
		},
		{
			name:     "403 http error",
			err:      errors.New("HTTP 403 Forbidden"),
			expected: AuthError,
		},
		// Additional network errors
		{
			name:     "dns error",
			err:      errors.New("DNS lookup failed"),
			expected: NetworkError,
		},
		{
			name:     "unreachable host",
			err:      errors.New("host unreachable"),
			expected: NetworkError,
		},
		{
			name:     "service unavailable",
			err:      errors.New("service unavailable"),
			expected: NetworkError,
		},
		{
			name:     "no route to host",
			err:      errors.New("no route to host"),
			expected: NetworkError,
		},
		{
			name:     "502 error",
			err:      errors.New("HTTP 502 Bad Gateway"),
			expected: NetworkError,
		},
		{
			name:     "503 error",
			err:      errors.New("HTTP 503 Service Unavailable"),
			expected: NetworkError,
		},
		{
			name:     "504 error",
			err:      errors.New("HTTP 504 Gateway Timeout"),
			expected: NetworkError,
		},
		// Additional usage errors
		{
			name:     "unknown command",
			err:      errors.New("unknown command: foo"),
			expected: UsageError,
		},
		{
			name:     "missing argument",
			err:      errors.New("missing argument for flag"),
			expected: UsageError,
		},
		{
			name:     "invalid argument",
			err:      errors.New("invalid argument: xyz"),
			expected: UsageError,
		},
		{
			name:     "unknown flag",
			err:      errors.New("unknown flag: --bar"),
			expected: UsageError,
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

func TestDetermineExitCode_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "uppercase POLICY",
			err:      errors.New("POLICY violation occurred"),
			expected: PolicyViolation,
		},
		{
			name:     "mixed case Network",
			err:      errors.New("NeTwOrK error"),
			expected: NetworkError,
		},
		{
			name:     "uppercase UNAUTHORIZED",
			err:      errors.New("UNAUTHORIZED access"),
			expected: AuthError,
		},
		{
			name:     "uppercase DRIFT",
			err:      errors.New("DRIFT DETECTED"),
			expected: DriftDetected,
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
