package bundle

import (
	"fmt"
	"strings"
)

// BundleError represents a user-friendly bundle operation error with actionable guidance.
type BundleError struct {
	// Operation is the operation that failed (e.g., "build", "verify", "apply")
	Operation string

	// Message is the user-friendly error message
	Message string

	// Suggestion provides actionable guidance to fix the error
	Suggestion string

	// Details contains technical details for debugging
	Details string

	// Cause is the underlying error
	Cause error
}

// Error implements the error interface.
func (e *BundleError) Error() string {
	var parts []string

	// Operation and message
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("bundle %s failed: %s", e.Operation, e.Message))
	} else {
		parts = append(parts, e.Message)
	}

	// Suggestion
	if e.Suggestion != "" {
		parts = append(parts, fmt.Sprintf("\nSuggestion: %s", e.Suggestion))
	}

	// Details (for debugging)
	if e.Details != "" {
		parts = append(parts, fmt.Sprintf("\nDetails: %s", e.Details))
	}

	// Underlying cause
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("\nCause: %v", e.Cause))
	}

	return strings.Join(parts, "")
}

// Unwrap returns the underlying error for errors.Is and errors.As support.
func (e *BundleError) Unwrap() error {
	return e.Cause
}

// Common error constructors with actionable suggestions

// ErrInvalidManifest creates an error for invalid manifest with suggestions.
func ErrInvalidManifest(reason string, err error) *BundleError {
	return &BundleError{
		Operation:  "validate",
		Message:    fmt.Sprintf("invalid manifest: %s", reason),
		Suggestion: "Check the manifest.yaml file structure. Ensure all required fields are present (schema, id, version, created).",
		Details:    reason,
		Cause:      err,
	}
}

// ErrChecksumMismatch creates an error for checksum failures with suggestions.
func ErrChecksumMismatch(file, expected, actual string) *BundleError {
	return &BundleError{
		Operation:  "verify",
		Message:    fmt.Sprintf("checksum mismatch for file: %s", file),
		Suggestion: "The file has been modified after the bundle was created. If this is expected, rebuild the bundle. Otherwise, this may indicate tampering.",
		Details:    fmt.Sprintf("expected: %s, got: %s", expected, actual),
	}
}

// ErrMissingApproval creates an error for missing approvals with suggestions.
func ErrMissingApproval(role string) *BundleError {
	return &BundleError{
		Operation:  "verify",
		Message:    fmt.Sprintf("missing required approval for role: %s", role),
		Suggestion: fmt.Sprintf("Obtain approval from a user with the '%s' role using: specular bundle approve <bundle> --role %s --user <email>", role, role),
		Details:    fmt.Sprintf("role: %s", role),
	}
}
