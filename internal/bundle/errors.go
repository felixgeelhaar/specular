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

// ErrFileNotFound creates an error for missing files with suggestions.
func ErrFileNotFound(path string, err error) *BundleError {
	return &BundleError{
		Message:    fmt.Sprintf("file not found: %s", path),
		Suggestion: "Verify the file path is correct and the file exists. Use absolute paths or check your current directory.",
		Cause:      err,
	}
}

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

// ErrInvalidSignature creates an error for signature verification failures with suggestions.
func ErrInvalidSignature(context, details string, err error) *BundleError {
	suggestions := []string{
		"Verify the signing key matches the public key used for verification",
		"Check that the bundle hasn't been modified after signing",
		"Ensure the signature file is valid and not corrupted",
	}

	return &BundleError{
		Operation:  "verify",
		Message:    fmt.Sprintf("signature verification failed: %s", context),
		Suggestion: strings.Join(suggestions, "\n  • "),
		Details:    details,
		Cause:      err,
	}
}

// ErrAttestationFailed creates an error for attestation failures with suggestions.
func ErrAttestationFailed(reason string, err error) *BundleError {
	return &BundleError{
		Operation:  "verify",
		Message:    fmt.Sprintf("attestation verification failed: %s", reason),
		Suggestion: "Check that:\n  • The attestation was signed with a valid key\n  • The bundle hasn't been modified since attestation\n  • Rekor transparency log is accessible (if using Sigstore)",
		Details:    reason,
		Cause:      err,
	}
}

// ErrRegistryAuth creates an error for registry authentication failures with suggestions.
func ErrRegistryAuth(registry string, err error) *BundleError {
	suggestions := []string{
		fmt.Sprintf("Authenticate to the registry: docker login %s", registry),
		"Verify your credentials are correct and not expired",
		"Check if you have permission to access this registry",
		"For GitHub Container Registry (ghcr.io), use a personal access token with 'read:packages' scope",
	}

	return &BundleError{
		Operation:  "registry",
		Message:    fmt.Sprintf("authentication failed for registry: %s", registry),
		Suggestion: strings.Join(suggestions, "\n  • "),
		Cause:      err,
	}
}

// ErrRegistryNotFound creates an error for missing registry images with suggestions.
func ErrRegistryNotFound(ref string, err error) *BundleError {
	return &BundleError{
		Operation:  "registry",
		Message:    fmt.Sprintf("bundle not found in registry: %s", ref),
		Suggestion: fmt.Sprintf("Verify the bundle reference is correct. You can push a bundle using: specular bundle push <bundle> %s", ref),
		Cause:      err,
	}
}

// ErrBundleCorrupted creates an error for corrupted bundles with suggestions.
func ErrBundleCorrupted(reason string, err error) *BundleError {
	return &BundleError{
		Operation:  "extract",
		Message:    "bundle file is corrupted or invalid",
		Suggestion: "Download the bundle again or rebuild it from source. The bundle may have been damaged during transfer.",
		Details:    reason,
		Cause:      err,
	}
}

// ErrKeyNotFound creates an error for missing signing keys with suggestions.
func ErrKeyNotFound(keyType, path string) *BundleError {
	var suggestion string
	switch keyType {
	case "ssh":
		suggestion = fmt.Sprintf("Generate an SSH key pair:\n  ssh-keygen -t ed25519 -f %s\nOr specify a different key path with --key-path", path)
	case "gpg":
		suggestion = "List available GPG keys:\n  gpg --list-secret-keys\nOr generate a new key:\n  gpg --gen-key"
	default:
		suggestion = fmt.Sprintf("Verify the key exists at: %s", path)
	}

	return &BundleError{
		Operation:  "sign",
		Message:    fmt.Sprintf("%s key not found: %s", keyType, path),
		Suggestion: suggestion,
	}
}

// ErrInvalidBundleFormat creates an error for invalid bundle formats with suggestions.
func ErrInvalidBundleFormat(expected, got string) *BundleError {
	return &BundleError{
		Operation:  "verify",
		Message:    fmt.Sprintf("invalid bundle format: expected %s, got %s", expected, got),
		Suggestion: "Ensure you're using a valid .sbundle.tgz file. Bundles must be created with: specular bundle build",
		Details:    fmt.Sprintf("expected: %s, got: %s", expected, got),
	}
}

// ErrPolicyViolation creates an error for policy violations with suggestions.
func ErrPolicyViolation(policy, violation string) *BundleError {
	return &BundleError{
		Operation:  "verify",
		Message:    fmt.Sprintf("policy violation: %s", violation),
		Suggestion: fmt.Sprintf("Review the policy requirements in: %s\nEither fix the violation or update the policy if the requirements have changed.", policy),
		Details:    violation,
	}
}

// ErrInsufficientPermissions creates an error for permission issues with suggestions.
func ErrInsufficientPermissions(operation, resource string, err error) *BundleError {
	suggestions := []string{
		fmt.Sprintf("Check file permissions: ls -la %s", resource),
		"Ensure you have read/write access to the directory",
		"Try running with appropriate permissions",
	}

	return &BundleError{
		Operation:  operation,
		Message:    fmt.Sprintf("insufficient permissions for: %s", resource),
		Suggestion: strings.Join(suggestions, "\n  • "),
		Cause:      err,
	}
}

// ErrNetworkFailure creates an error for network issues with suggestions.
func ErrNetworkFailure(operation, endpoint string, err error) *BundleError {
	suggestions := []string{
		"Check your internet connection",
		fmt.Sprintf("Verify the endpoint is accessible: %s", endpoint),
		"Check if you're behind a proxy that may be blocking the connection",
		"Try again later if the service is temporarily unavailable",
	}

	return &BundleError{
		Operation:  operation,
		Message:    fmt.Sprintf("network error connecting to: %s", endpoint),
		Suggestion: strings.Join(suggestions, "\n  • "),
		Cause:      err,
	}
}

// ErrInvalidConfiguration creates an error for configuration issues with suggestions.
func ErrInvalidConfiguration(field, reason string) *BundleError {
	return &BundleError{
		Operation:  "configure",
		Message:    fmt.Sprintf("invalid configuration for '%s': %s", field, reason),
		Suggestion: fmt.Sprintf("Check the configuration file and ensure '%s' is set correctly. See documentation for valid values.", field),
		Details:    reason,
	}
}

// ErrDependencyMissing creates an error for missing dependencies with suggestions.
func ErrDependencyMissing(dependency, installCmd string) *BundleError {
	return &BundleError{
		Message:    fmt.Sprintf("required dependency not found: %s", dependency),
		Suggestion: fmt.Sprintf("Install the dependency:\n  %s", installCmd),
	}
}

// ErrOperationTimeout creates an error for timeout issues with suggestions.
func ErrOperationTimeout(operation string, duration string) *BundleError {
	suggestions := []string{
		fmt.Sprintf("The %s operation exceeded the timeout of %s", operation, duration),
		"Try increasing the timeout if working with large bundles",
		"Check network connectivity if downloading from a registry",
		"Verify the system isn't under heavy load",
	}

	return &BundleError{
		Operation:  operation,
		Message:    "operation timed out",
		Suggestion: strings.Join(suggestions, "\n  • "),
	}
}
