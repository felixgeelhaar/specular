package errors

import (
	"fmt"
	"strings"
)

// ErrorCode represents a unique error identifier
type ErrorCode string

// Error categories
const (
	// Spec errors (SPEC-001 to SPEC-099)
	ErrCodeSpecNotFound     ErrorCode = "SPEC-001"
	ErrCodeSpecInvalid      ErrorCode = "SPEC-002"
	ErrCodeSpecUnmarshal    ErrorCode = "SPEC-003"
	ErrCodeSpecMarshal      ErrorCode = "SPEC-004"
	ErrCodeSpecLockNotFound ErrorCode = "SPEC-005"
	ErrCodeSpecLockInvalid  ErrorCode = "SPEC-006"
	ErrCodeSpecHashMismatch ErrorCode = "SPEC-007"

	// Policy errors (POLICY-001 to POLICY-099)
	ErrCodePolicyNotFound      ErrorCode = "POLICY-001"
	ErrCodePolicyInvalid       ErrorCode = "POLICY-002"
	ErrCodePolicyViolation     ErrorCode = "POLICY-003"
	ErrCodePolicyToolMissing   ErrorCode = "POLICY-004"
	ErrCodePolicyImageDenied   ErrorCode = "POLICY-005"
	ErrCodePolicyNetworkDenied ErrorCode = "POLICY-006"

	// Plan errors (PLAN-001 to PLAN-099)
	ErrCodePlanNotFound      ErrorCode = "PLAN-001"
	ErrCodePlanInvalid       ErrorCode = "PLAN-002"
	ErrCodePlanDriftDetected ErrorCode = "PLAN-003"
	ErrCodePlanTaskMissing   ErrorCode = "PLAN-004"
	ErrCodePlanCyclicDep     ErrorCode = "PLAN-005"

	// Interview errors (INTERVIEW-001 to INTERVIEW-099)
	ErrCodeInterviewPresetUnknown    ErrorCode = "INTERVIEW-001"
	ErrCodeInterviewAlreadyStarted   ErrorCode = "INTERVIEW-002"
	ErrCodeInterviewNotComplete      ErrorCode = "INTERVIEW-003"
	ErrCodeInterviewValidationFailed ErrorCode = "INTERVIEW-004"
	ErrCodeInterviewAnswerRequired   ErrorCode = "INTERVIEW-005"
	ErrCodeInterviewAnswerInvalid    ErrorCode = "INTERVIEW-006"

	// Provider errors (PROVIDER-001 to PROVIDER-099)
	ErrCodeProviderNotFound      ErrorCode = "PROVIDER-001"
	ErrCodeProviderConfig        ErrorCode = "PROVIDER-002"
	ErrCodeProviderAuth          ErrorCode = "PROVIDER-003"
	ErrCodeProviderAPI           ErrorCode = "PROVIDER-004"
	ErrCodeProviderRateLimit     ErrorCode = "PROVIDER-005"
	ErrCodeProviderTimeout       ErrorCode = "PROVIDER-006"
	ErrCodeProviderModelNotFound ErrorCode = "PROVIDER-007"

	// Execution errors (EXEC-001 to EXEC-099)
	ErrCodeExecDockerNotAvailable ErrorCode = "EXEC-001"
	ErrCodeExecImagePullFailed    ErrorCode = "EXEC-002"
	ErrCodeExecContainerFailed    ErrorCode = "EXEC-003"
	ErrCodeExecTimeout            ErrorCode = "EXEC-004"
	ErrCodeExecResourceLimit      ErrorCode = "EXEC-005"

	// Drift errors (DRIFT-001 to DRIFT-099)
	ErrCodeDriftPlanSpec     ErrorCode = "DRIFT-001"
	ErrCodeDriftCodeContract ErrorCode = "DRIFT-002"
	ErrCodeDriftInfraPolicy  ErrorCode = "DRIFT-003"

	// File I/O errors (IO-001 to IO-099)
	ErrCodeFileNotFound    ErrorCode = "IO-001"
	ErrCodeFileReadFailed  ErrorCode = "IO-002"
	ErrCodeFileWriteFailed ErrorCode = "IO-003"
	ErrCodeDirectoryFailed ErrorCode = "IO-004"
	ErrCodeFileUnmarshal   ErrorCode = "IO-005"
	ErrCodeFileMarshal     ErrorCode = "IO-006"
)

// SpecularError represents an enhanced error with code, suggestions, and documentation
type SpecularError struct {
	Code        ErrorCode
	Message     string
	Suggestions []string
	DocsURL     string
	Cause       error
}

// Error implements the error interface
func (e *SpecularError) Error() string {
	var b strings.Builder

	// Error code and message
	b.WriteString(fmt.Sprintf("[%s] %s", e.Code, e.Message))

	// Add cause if present
	if e.Cause != nil {
		b.WriteString(fmt.Sprintf(": %v", e.Cause))
	}

	// Add suggestions
	if len(e.Suggestions) > 0 {
		b.WriteString("\n\nSuggestions:")
		for _, suggestion := range e.Suggestions {
			b.WriteString(fmt.Sprintf("\n  • %s", suggestion))
		}
	}

	// Add documentation link
	if e.DocsURL != "" {
		b.WriteString(fmt.Sprintf("\n\nDocumentation: %s", e.DocsURL))
	}

	return b.String()
}

// Unwrap implements error unwrapping for errors.Is and errors.As
func (e *SpecularError) Unwrap() error {
	return e.Cause
}

// New creates a new SpecularError
func New(code ErrorCode, message string) *SpecularError {
	return &SpecularError{
		Code:    code,
		Message: message,
	}
}

// Wrap creates a new SpecularError wrapping an existing error
func Wrap(code ErrorCode, message string, cause error) *SpecularError {
	return &SpecularError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WithSuggestion adds a suggestion to the error
func (e *SpecularError) WithSuggestion(suggestion string) *SpecularError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithSuggestions adds multiple suggestions to the error
func (e *SpecularError) WithSuggestions(suggestions ...string) *SpecularError {
	e.Suggestions = append(e.Suggestions, suggestions...)
	return e
}

// WithDocs adds a documentation URL to the error
func (e *SpecularError) WithDocs(url string) *SpecularError {
	e.DocsURL = url
	return e
}

// Common error constructors for frequently used errors

// NewSpecNotFoundError creates a spec file not found error
func NewSpecNotFoundError(path string) *SpecularError {
	return New(ErrCodeSpecNotFound, fmt.Sprintf("specification file not found: %s", path)).
		WithSuggestion("Run 'specular interview' to create a new spec").
		WithSuggestion("Check if the file path is correct").
		WithDocs("https://github.com/felixgeelhaar/specular#specification-management")
}

// NewSpecInvalidError creates a spec validation error
func NewSpecInvalidError(details string) *SpecularError {
	return New(ErrCodeSpecInvalid, fmt.Sprintf("invalid specification: %s", details)).
		WithSuggestion("Run 'specular spec validate --in <file>' to see validation errors").
		WithSuggestion("Check the spec schema requirements").
		WithDocs("https://github.com/felixgeelhaar/specular#specification-schema")
}

// NewPolicyViolationError creates a policy violation error
func NewPolicyViolationError(violation string) *SpecularError {
	return New(ErrCodePolicyViolation, fmt.Sprintf("policy violation: %s", violation)).
		WithSuggestion("Review your policy configuration in .specular/policy.yaml").
		WithSuggestion("Contact your administrator if you need policy exceptions").
		WithDocs("https://github.com/felixgeelhaar/specular#policy-enforcement")
}

// NewProviderAuthError creates a provider authentication error
func NewProviderAuthError(provider string) *SpecularError {
	return New(ErrCodeProviderAuth, fmt.Sprintf("authentication failed for provider: %s", provider)).
		WithSuggestion(fmt.Sprintf("Set the %s_API_KEY environment variable", strings.ToUpper(provider))).
		WithSuggestion("Check if your API key is valid and not expired").
		WithSuggestion("Run 'specular provider health <provider>' to verify connectivity").
		WithDocs("https://github.com/felixgeelhaar/specular#provider-configuration")
}

// NewProviderRateLimitError creates a rate limit error
func NewProviderRateLimitError(provider string, retryAfter string) *SpecularError {
	msg := fmt.Sprintf("rate limit exceeded for provider: %s", provider)
	if retryAfter != "" {
		msg += fmt.Sprintf(" (retry after: %s)", retryAfter)
	}

	return New(ErrCodeProviderRateLimit, msg).
		WithSuggestion("Wait before retrying the request").
		WithSuggestion("Consider upgrading your provider plan for higher limits").
		WithSuggestion("Use a different provider if available").
		WithDocs("https://github.com/felixgeelhaar/specular#rate-limiting")
}

// NewExecDockerNotAvailableError creates a Docker not available error
func NewExecDockerNotAvailableError() *SpecularError {
	return New(ErrCodeExecDockerNotAvailable, "Docker is not available").
		WithSuggestion("Install Docker Desktop or Docker Engine").
		WithSuggestion("Make sure Docker daemon is running").
		WithSuggestion("Run 'docker version' to verify Docker installation").
		WithDocs("https://docs.docker.com/get-docker/")
}

// NewInterviewPresetUnknownError creates an unknown preset error
func NewInterviewPresetUnknownError(preset string) *SpecularError {
	return New(ErrCodeInterviewPresetUnknown, fmt.Sprintf("unknown interview preset: %s", preset)).
		WithSuggestion("Run 'specular interview --list' to see available presets").
		WithSuggestion("Use one of: web-app, api-service, cli-tool, microservice, data-pipeline").
		WithDocs("https://github.com/felixgeelhaar/specular#interview-presets")
}

// NewInterviewAnswerRequiredError creates a required answer error
func NewInterviewAnswerRequiredError(question string) *SpecularError {
	return New(ErrCodeInterviewAnswerRequired, fmt.Sprintf("answer is required for: %s", question)).
		WithSuggestion("Provide a non-empty answer").
		WithSuggestion("Use --strict=false to allow skipping optional questions")
}

// NewInterviewAnswerInvalidError creates an invalid answer error
func NewInterviewAnswerInvalidError(question string, expected string) *SpecularError {
	return New(ErrCodeInterviewAnswerInvalid, fmt.Sprintf("invalid answer for: %s", question)).
		WithSuggestion(fmt.Sprintf("Expected: %s", expected)).
		WithSuggestion("Check the question type and provide a valid answer")
}

// NewPlanDriftError creates a plan drift detection error
func NewPlanDriftError(featureID string, expectedHash string, actualHash string) *SpecularError {
	return New(ErrCodePlanDriftDetected, fmt.Sprintf("plan drift detected for feature: %s", featureID)).
		WithSuggestion("Regenerate the plan with 'specular plan' to sync with spec").
		WithSuggestion("Review changes in the specification file").
		WithSuggestion(fmt.Sprintf("Expected hash: %s, got: %s", expectedHash, actualHash)).
		WithDocs("https://github.com/felixgeelhaar/specular#drift-detection")
}

// NewFileNotFoundError creates a file not found error
func NewFileNotFoundError(path string) *SpecularError {
	return New(ErrCodeFileNotFound, fmt.Sprintf("file not found: %s", path)).
		WithSuggestion("Check if the file path is correct").
		WithSuggestion("Verify the file exists and you have read permissions")
}

// NewFileUnmarshalError creates an unmarshal error
func NewFileUnmarshalError(path string, format string, cause error) *SpecularError {
	return Wrap(ErrCodeFileUnmarshal, fmt.Sprintf("failed to parse %s file: %s", format, path), cause).
		WithSuggestion("Check the file syntax and format").
		WithSuggestion(fmt.Sprintf("Ensure the file is valid %s", format))
}
