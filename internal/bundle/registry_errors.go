package bundle

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// RegistryErrorType categorizes different types of registry errors
type RegistryErrorType string

const (
	ErrTypeAuthentication RegistryErrorType = "AUTHENTICATION"
	ErrTypeNotFound       RegistryErrorType = "NOT_FOUND"
	ErrTypeNetwork        RegistryErrorType = "NETWORK"
	ErrTypePermission     RegistryErrorType = "PERMISSION"
	ErrTypeInvalidRef     RegistryErrorType = "INVALID_REFERENCE"
	ErrTypeInvalidBundle  RegistryErrorType = "INVALID_BUNDLE"
	ErrTypeUnknown        RegistryErrorType = "UNKNOWN"
)

// RegistryError wraps registry errors with context and suggestions
type RegistryError struct {
	Type       RegistryErrorType
	Message    string
	Suggestion string
	Cause      error
	Reference  string
}

// Error implements the error interface
func (e *RegistryError) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.Type, e.Message)
	if e.Suggestion != "" {
		msg += fmt.Sprintf("\n\nSuggestion: %s", e.Suggestion)
	}
	if e.Cause != nil {
		msg += fmt.Sprintf("\n\nCause: %v", e.Cause)
	}
	return msg
}

// Unwrap returns the underlying cause
func (e *RegistryError) Unwrap() error {
	return e.Cause
}

// ClassifyRegistryError analyzes an error and wraps it with helpful context
func ClassifyRegistryError(err error, ref string, operation string) error {
	if err == nil {
		return nil
	}

	// Check for transport.Error (most common go-containerregistry error)
	var transportErr *transport.Error
	if errors.As(err, &transportErr) {
		return classifyTransportError(transportErr, ref, operation)
	}

	// Check for name.ErrBadName (invalid reference)
	var nameErr *name.ErrBadName
	if errors.As(err, &nameErr) {
		return &RegistryError{
			Type:    ErrTypeInvalidRef,
			Message: fmt.Sprintf("Invalid registry reference: %s", ref),
			Suggestion: `Registry references must follow the format:
  - registry.com/org/repo:tag
  - ghcr.io/org/repo:tag
  - docker.io/username/repo:tag

Examples:
  - ghcr.io/myorg/bundle:v1.0.0
  - docker.io/username/bundle:latest
  - registry.company.com/team/bundle:v1.0.0`,
			Cause:     err,
			Reference: ref,
		}
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return classifyNetworkError(netErr, ref, operation)
	}

	// Check for specific error messages in the error string
	errMsg := err.Error()

	// Authentication errors
	if strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "authentication required") {
		return &RegistryError{
			Type:    ErrTypeAuthentication,
			Message: fmt.Sprintf("Authentication failed for %s", ref),
			Suggestion: `Please authenticate to the registry:

1. Docker Hub:
   docker login docker.io

2. GitHub Container Registry:
   echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

3. Google Container Registry:
   gcloud auth configure-docker

4. Private Registry:
   docker login registry.company.com

Credentials are read from ~/.docker/config.json`,
			Cause:     err,
			Reference: ref,
		}
	}

	// Permission/forbidden errors
	if strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "denied") {
		return &RegistryError{
			Type:    ErrTypePermission,
			Message: fmt.Sprintf("Permission denied for %s", ref),
			Suggestion: `You don't have permission to access this repository.

Possible causes:
1. Repository is private and you're not authenticated
2. Your credentials don't have the required permissions
3. The repository doesn't exist

Try:
- Verify the repository exists and you have access
- Check your authentication credentials
- For GitHub: Ensure your token has 'read:packages' or 'write:packages' scope`,
			Cause:     err,
			Reference: ref,
		}
	}

	// Not found errors
	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "manifest unknown") {
		return &RegistryError{
			Type:    ErrTypeNotFound,
			Message: fmt.Sprintf("Bundle not found: %s", ref),
			Suggestion: `The bundle does not exist in the registry.

Check:
- The repository name is correct
- The tag/version exists
- You're authenticated if it's a private repository

To list available tags:
  docker images <repository>
  crane ls <repository>`,
			Cause:     err,
			Reference: ref,
		}
	}

	// Invalid artifact/bundle errors
	if strings.Contains(errMsg, "invalid") && strings.Contains(errMsg, "artifact") {
		return &RegistryError{
			Type:    ErrTypeInvalidBundle,
			Message: fmt.Sprintf("Invalid bundle artifact: %s", ref),
			Suggestion: `The artifact exists but is not a valid Specular bundle.

A valid bundle must:
- Have artifact type: application/vnd.specular.bundle.v1
- Have layer media type: application/vnd.specular.bundle.layer.v1.tar+gzip
- Contain exactly one layer (the bundle tarball)

This reference may point to a regular container image instead of a bundle.`,
			Cause:     err,
			Reference: ref,
		}
	}

	// Default to unknown error type
	return &RegistryError{
		Type:       ErrTypeUnknown,
		Message:    fmt.Sprintf("Registry operation failed: %s", operation),
		Suggestion: "Check your network connection and registry credentials",
		Cause:      err,
		Reference:  ref,
	}
}

// classifyTransportError handles transport.Error from go-containerregistry
func classifyTransportError(err *transport.Error, ref string, operation string) error {
	switch err.StatusCode {
	case 401:
		return &RegistryError{
			Type:    ErrTypeAuthentication,
			Message: fmt.Sprintf("Authentication required for %s", ref),
			Suggestion: `Please log in to the registry:

For Docker Hub:
  docker login docker.io

For GitHub Container Registry:
  echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

For private registries:
  docker login <registry-url>`,
			Cause:     err,
			Reference: ref,
		}

	case 403:
		return &RegistryError{
			Type:    ErrTypePermission,
			Message: fmt.Sprintf("Access forbidden: %s", ref),
			Suggestion: `You don't have permission to perform this operation.

For push operations:
- Verify you have write access to the repository
- Check that your authentication token has the correct scopes
- GitHub requires 'write:packages' scope for pushing

For pull operations:
- Verify the repository exists and you have read access
- For private repositories, ensure you're authenticated`,
			Cause:     err,
			Reference: ref,
		}

	case 404:
		return &RegistryError{
			Type:    ErrTypeNotFound,
			Message: fmt.Sprintf("Repository or tag not found: %s", ref),
			Suggestion: `The repository or tag doesn't exist.

For pull operations:
- Verify the repository name and tag are correct
- Check if the repository is private and requires authentication

For push operations:
- The repository may not exist yet (it will be created automatically)
- Verify you have permission to create repositories in this namespace`,
			Cause:     err,
			Reference: ref,
		}

	case 429:
		return &RegistryError{
			Type:    ErrTypeNetwork,
			Message: "Rate limit exceeded",
			Suggestion: `The registry rate limit has been exceeded.

Solutions:
- Wait a few minutes and try again
- Authenticate to increase rate limits (Docker Hub anonymous: 100/6h, authenticated: 200/6h)
- Use a different registry or mirror
- Contact your registry administrator for enterprise rate limits`,
			Cause:     err,
			Reference: ref,
		}

	case 500, 502, 503, 504:
		return &RegistryError{
			Type:    ErrTypeNetwork,
			Message: "Registry server error",
			Suggestion: `The registry is experiencing issues.

Try:
- Wait a few minutes and retry
- Check the registry status page
- Use a different registry mirror if available
- Contact your registry administrator`,
			Cause:     err,
			Reference: ref,
		}

	default:
		return &RegistryError{
			Type:       ErrTypeUnknown,
			Message:    fmt.Sprintf("Registry HTTP error %d", err.StatusCode),
			Suggestion: "Check registry documentation for this error code",
			Cause:      err,
			Reference:  ref,
		}
	}
}

// classifyNetworkError handles network connectivity errors
func classifyNetworkError(err net.Error, ref string, operation string) error {
	if err.Timeout() {
		return &RegistryError{
			Type:    ErrTypeNetwork,
			Message: fmt.Sprintf("Connection timeout to registry for %s", ref),
			Suggestion: `The connection to the registry timed out.

Check:
- Your internet connection
- The registry URL is correct
- Corporate firewall/proxy settings
- DNS resolution is working

For insecure/local registries, use --insecure flag`,
			Cause:     err,
			Reference: ref,
		}
	}

	return &RegistryError{
		Type:    ErrTypeNetwork,
		Message: fmt.Sprintf("Network error accessing %s", ref),
		Suggestion: `Unable to connect to the registry.

Check:
- Your internet connection
- The registry URL is correct
- DNS resolution (try: ping registry-host)
- Corporate firewall/proxy settings
- For local registries: ensure the registry is running`,
		Cause:     err,
		Reference: ref,
	}
}

// WrapRegistryError wraps an error with registry context if it's not already a RegistryError
func WrapRegistryError(err error, ref string, operation string) error {
	if err == nil {
		return nil
	}

	// Don't double-wrap
	var regErr *RegistryError
	if errors.As(err, &regErr) {
		return err
	}

	return ClassifyRegistryError(err, ref, operation)
}
