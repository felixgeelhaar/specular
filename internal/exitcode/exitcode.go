package exitcode

import (
	"os"
	"strings"
)

// Exit codes for consistent error handling across the CLI
const (
	// Success indicates successful execution
	Success = 0

	// GeneralError indicates a general error condition
	GeneralError = 1

	// UsageError indicates invalid command usage (bad flags, missing args, etc.)
	UsageError = 2

	// PolicyViolation indicates a policy enforcement failure
	PolicyViolation = 3

	// DriftDetected indicates configuration or state drift was found
	DriftDetected = 4

	// AuthError indicates an authentication or authorization failure
	AuthError = 5

	// NetworkError indicates a network connectivity issue
	NetworkError = 6

	// Interrupted indicates the operation was cancelled by user (Ctrl+C)
	Interrupted = 130
)

// Exit terminates the program with the given exit code
func Exit(code int) {
	os.Exit(code)
}

// ExitWithError exits with an appropriate code based on error type
func ExitWithError(err error) {
	if err == nil {
		Exit(Success)
		return
	}

	code := DetermineExitCode(err)
	Exit(code)
}

// DetermineExitCode analyzes an error and returns the appropriate exit code
func DetermineExitCode(err error) int {
	if err == nil {
		return Success
	}

	errMsg := strings.ToLower(err.Error())

	// Check each error category
	if code, matched := checkPolicyViolation(errMsg); matched {
		return code
	}
	if code, matched := checkDriftDetection(errMsg); matched {
		return code
	}
	if code, matched := checkAuthError(errMsg); matched {
		return code
	}
	if code, matched := checkNetworkError(errMsg); matched {
		return code
	}
	if code, matched := checkUsageError(errMsg); matched {
		return code
	}

	// Default to general error
	return GeneralError
}

// checkPolicyViolation checks if the error is a policy violation
func checkPolicyViolation(errMsg string) (int, bool) {
	keywords := []string{
		"policy violation",
		"policy check failed",
		"not allowed by policy",
		"not allowed",
		"restricted",
		"budget exceeded",
		"cost limit",
		"step type blocked",
		"operation blocked",
	}

	for _, keyword := range keywords {
		if strings.Contains(errMsg, keyword) {
			return PolicyViolation, true
		}
	}
	return 0, false
}

// checkDriftDetection checks if the error is drift detection
func checkDriftDetection(errMsg string) (int, bool) {
	keywords := []string{
		"drift detected",
		"drift",
		"hash mismatch",
		"spec changed",
		"verification failed",
		"checksum mismatch",
	}

	for _, keyword := range keywords {
		if strings.Contains(errMsg, keyword) {
			return DriftDetected, true
		}
	}
	return 0, false
}

// checkAuthError checks if the error is an authentication error
func checkAuthError(errMsg string) (int, bool) {
	keywords := []string{
		"authentication",
		"unauthorized",
		"forbidden",
		"permission denied",
		"api key",
		"invalid api key",
		"invalid token",
		"expired token",
		"access denied",
		"401",
		"403",
	}

	for _, keyword := range keywords {
		if strings.Contains(errMsg, keyword) {
			return AuthError, true
		}
	}
	return 0, false
}

// checkNetworkError checks if the error is a network error
func checkNetworkError(errMsg string) (int, bool) {
	keywords := []string{
		"network",
		"connection",
		"timeout",
		"unreachable",
		"dns",
		"service unavailable",
		"connection refused",
		"no route to host",
		"502",
		"503",
		"504",
	}

	for _, keyword := range keywords {
		if strings.Contains(errMsg, keyword) {
			return NetworkError, true
		}
	}
	return 0, false
}

// checkUsageError checks if the error is a usage error
func checkUsageError(errMsg string) (int, bool) {
	keywords := []string{
		"invalid flag",
		"unknown command",
		"required flag",
		"missing argument",
		"invalid argument",
		"unknown flag",
		"usage:",
	}

	for _, keyword := range keywords {
		if strings.Contains(errMsg, keyword) {
			return UsageError, true
		}
	}
	return 0, false
}

// GetExitCodeDescription returns a human-readable description of an exit code
func GetExitCodeDescription(code int) string {
	switch code {
	case Success:
		return "Success"
	case GeneralError:
		return "General error"
	case UsageError:
		return "Usage error (invalid flags or arguments)"
	case PolicyViolation:
		return "Policy violation"
	case DriftDetected:
		return "Configuration drift detected"
	case AuthError:
		return "Authentication error"
	case NetworkError:
		return "Network error"
	default:
		return "Unknown error"
	}
}
