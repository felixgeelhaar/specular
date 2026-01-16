package ux

import (
	"fmt"
	"regexp"
	"strings"
)

// ErrorCategory categorizes errors for targeted suggestions and formatting
type ErrorCategory string

const (
	// CategoryYAMLSyntax indicates YAML parsing errors
	CategoryYAMLSyntax ErrorCategory = "yaml_syntax"
	// CategorySchemaViolation indicates configuration schema errors
	CategorySchemaViolation ErrorCategory = "schema"
	// CategoryMissingFile indicates required file not found
	CategoryMissingFile ErrorCategory = "missing_file"
	// CategoryNetwork indicates network connectivity issues
	CategoryNetwork ErrorCategory = "network"
	// CategoryAuth indicates authentication/authorization failures
	CategoryAuth ErrorCategory = "auth"
	// CategoryProvider indicates AI provider issues
	CategoryProvider ErrorCategory = "provider"
	// CategoryPolicy indicates policy enforcement issues
	CategoryPolicy ErrorCategory = "policy"
	// CategoryDocker indicates container/Docker issues
	CategoryDocker ErrorCategory = "docker"
	// CategoryPermission indicates file/directory permission issues
	CategoryPermission ErrorCategory = "permission"
	// CategoryValidation indicates spec/config validation errors
	CategoryValidation ErrorCategory = "validation"
	// CategoryUnknown is the default for unrecognized errors
	CategoryUnknown ErrorCategory = "unknown"
)

// ANSI color codes for terminal output
const (
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
)

// EnhancedError provides rich error information with category,
// suggestions, and colored output support.
type EnhancedError struct {
	// Category of the error for targeted handling
	Category ErrorCategory
	// Message is the primary error message
	Message string
	// Suggestions are recovery commands or actions
	Suggestions []string
	// Code is an optional error code (e.g., "E001")
	Code string
	// NoColor disables colored output
	NoColor bool
	// underlying is the original error
	underlying error
}

// Error implements the error interface
func (e *EnhancedError) Error() string {
	return e.Message
}

// Unwrap returns the underlying error
func (e *EnhancedError) Unwrap() error {
	return e.underlying
}

// ColoredOutput returns the error with ANSI color formatting
func (e *EnhancedError) ColoredOutput() string {
	if e.NoColor {
		return e.PlainOutput()
	}

	var sb strings.Builder

	// Error code and category
	if e.Code != "" {
		sb.WriteString(colorRed + colorBold)
		sb.WriteString(fmt.Sprintf("[%s] ", e.Code))
		sb.WriteString(colorReset)
	}

	// Error message
	sb.WriteString(colorRed)
	sb.WriteString(e.Message)
	sb.WriteString(colorReset)

	// Category hint
	if e.Category != CategoryUnknown {
		sb.WriteString(colorCyan)
		sb.WriteString(fmt.Sprintf(" (%s)", e.Category))
		sb.WriteString(colorReset)
	}

	// Suggestions
	if len(e.Suggestions) > 0 {
		sb.WriteString("\n\n")
		sb.WriteString(colorYellow + colorBold)
		sb.WriteString("💡 Suggestions:")
		sb.WriteString(colorReset)
		for _, s := range e.Suggestions {
			sb.WriteString("\n")
			sb.WriteString(colorGreen)
			sb.WriteString("   → ")
			sb.WriteString(s)
			sb.WriteString(colorReset)
		}
	}

	return sb.String()
}

// PlainOutput returns the error without ANSI color codes
func (e *EnhancedError) PlainOutput() string {
	var sb strings.Builder

	if e.Code != "" {
		sb.WriteString(fmt.Sprintf("[%s] ", e.Code))
	}

	sb.WriteString(e.Message)

	if e.Category != CategoryUnknown {
		sb.WriteString(fmt.Sprintf(" (%s)", e.Category))
	}

	if len(e.Suggestions) > 0 {
		sb.WriteString("\n\nSuggestions:")
		for _, s := range e.Suggestions {
			sb.WriteString("\n   → ")
			sb.WriteString(s)
		}
	}

	return sb.String()
}

// NewEnhancedError creates an EnhancedError from an existing error
func NewEnhancedError(err error) *EnhancedError {
	if err == nil {
		return nil
	}

	enhanced := &EnhancedError{
		Message:    err.Error(),
		Category:   CategoryUnknown,
		underlying: err,
	}

	// Detect category and add suggestions
	enhanced.detectCategory()
	enhanced.addSuggestions()
	enhanced.assignCode()

	return enhanced
}

// detectCategory analyzes the error message to determine category
func (e *EnhancedError) detectCategory() {
	msg := strings.ToLower(e.Message)

	switch {
	// Check file-related errors first (before yaml: which could match file paths)
	case strings.Contains(msg, "no such file") || strings.Contains(msg, "file not found"):
		e.Category = CategoryMissingFile
	case strings.Contains(msg, "permission denied"):
		e.Category = CategoryPermission
	// YAML syntax errors (actual parsing errors, not file paths)
	case strings.Contains(msg, "yaml: ") || strings.Contains(msg, "yaml syntax"):
		e.Category = CategoryYAMLSyntax
	case strings.Contains(msg, "schema") || strings.Contains(msg, "invalid field"):
		e.Category = CategorySchemaViolation
	// Network and auth errors
	case strings.Contains(msg, "connection refused") || strings.Contains(msg, "no route"):
		e.Category = CategoryNetwork
	case strings.Contains(msg, "401") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "api key"):
		e.Category = CategoryAuth
	// Provider-specific "not found" (checked after general file not found)
	case strings.Contains(msg, "provider") && (strings.Contains(msg, "not found") || strings.Contains(msg, "not available")):
		e.Category = CategoryProvider
	// General "not found" (less specific, checked after provider)
	case strings.Contains(msg, "not found"):
		e.Category = CategoryMissingFile
	case strings.Contains(msg, "policy") || strings.Contains(msg, "violation"):
		e.Category = CategoryPolicy
	case strings.Contains(msg, "docker") || strings.Contains(msg, "container"):
		e.Category = CategoryDocker
	case strings.Contains(msg, "validation") || strings.Contains(msg, "drift"):
		e.Category = CategoryValidation
	}
}

// addSuggestions adds category-specific recovery suggestions
func (e *EnhancedError) addSuggestions() {
	msg := e.Message

	switch e.Category {
	case CategoryYAMLSyntax:
		// Extract line number if available
		lineMatch := regexp.MustCompile(`line (\d+)`).FindStringSubmatch(msg)
		if len(lineMatch) > 1 {
			e.Suggestions = append(e.Suggestions, fmt.Sprintf("Check YAML syntax at line %s", lineMatch[1]))
		}
		e.Suggestions = append(e.Suggestions, "Run: specular config validate")
		e.Suggestions = append(e.Suggestions, "Use a YAML linter to check your configuration")

	case CategorySchemaViolation:
		e.Suggestions = append(e.Suggestions, "Check documentation for valid field names and types")
		e.Suggestions = append(e.Suggestions, "Run: specular config validate --verbose")

	case CategoryMissingFile:
		if strings.Contains(msg, "spec.yaml") {
			e.Suggestions = append(e.Suggestions, "Run: specular spec new")
			e.Suggestions = append(e.Suggestions, "Run: specular spec new --from PRD.md")
		} else if strings.Contains(msg, "providers.yaml") || strings.Contains(msg, "routing.yaml") {
			e.Suggestions = append(e.Suggestions, "Run: specular init")
		}

	case CategoryAuth:
		e.Suggestions = append(e.Suggestions, "Check your API key is set correctly")
		e.Suggestions = append(e.Suggestions, "Run: specular auth login")
		e.Suggestions = append(e.Suggestions, "Verify: echo $OPENAI_API_KEY or $ANTHROPIC_API_KEY")

	case CategoryProvider:
		e.Suggestions = append(e.Suggestions, "Run: specular provider list")
		e.Suggestions = append(e.Suggestions, "Run: specular provider doctor")
		e.Suggestions = append(e.Suggestions, "Check .specular/routing.yaml configuration")

	case CategoryNetwork:
		e.Suggestions = append(e.Suggestions, "Check your internet connection")
		e.Suggestions = append(e.Suggestions, "Verify firewall/proxy settings")
		e.Suggestions = append(e.Suggestions, "Try: specular provider doctor")

	case CategoryDocker:
		e.Suggestions = append(e.Suggestions, "Ensure Docker Desktop/daemon is running")
		e.Suggestions = append(e.Suggestions, "Run: docker info (to verify)")

	case CategoryPolicy:
		e.Suggestions = append(e.Suggestions, "Review policy constraints in .specular/policy.yaml")
		e.Suggestions = append(e.Suggestions, "Consider increasing budget with --max-cost")

	case CategoryPermission:
		e.Suggestions = append(e.Suggestions, "Check file/directory permissions")
		e.Suggestions = append(e.Suggestions, "For Docker socket: sudo usermod -aG docker $USER")

	case CategoryValidation:
		e.Suggestions = append(e.Suggestions, "Run: specular spec validate")
		e.Suggestions = append(e.Suggestions, "Run: specular drift check")
	}
}

// assignCode assigns an error code based on category
func (e *EnhancedError) assignCode() {
	codes := map[ErrorCategory]string{
		CategoryYAMLSyntax:      "E001",
		CategorySchemaViolation: "E002",
		CategoryMissingFile:     "E003",
		CategoryNetwork:         "E004",
		CategoryAuth:            "E005",
		CategoryProvider:        "E006",
		CategoryPolicy:          "E007",
		CategoryDocker:          "E008",
		CategoryPermission:      "E009",
		CategoryValidation:      "E010",
	}

	if code, ok := codes[e.Category]; ok {
		e.Code = code
	}
}

// ErrorWithSuggestion wraps an error with helpful recovery suggestions
type ErrorWithSuggestion struct {
	Err        error
	Suggestion string
}

// Error implements the error interface
func (e *ErrorWithSuggestion) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%v\n\n💡 Suggestion: %s", e.Err, e.Suggestion)
	}
	return e.Err.Error()
}

// Unwrap provides access to the underlying error
func (e *ErrorWithSuggestion) Unwrap() error {
	return e.Err
}

// NewErrorWithSuggestion creates a new error with a suggestion
func NewErrorWithSuggestion(err error, suggestion string) error {
	if err == nil {
		return nil
	}
	return &ErrorWithSuggestion{
		Err:        err,
		Suggestion: suggestion,
	}
}

// EnhanceError analyzes an error and adds contextual suggestions
func EnhanceError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Try each error category
	if enhanced := enhanceFileNotFoundError(err, errMsg); enhanced != nil {
		return enhanced
	}
	if enhanced := enhanceDockerError(err, errMsg); enhanced != nil {
		return enhanced
	}
	if enhanced := enhancePermissionError(err, errMsg); enhanced != nil {
		return enhanced
	}
	if enhanced := enhanceProviderError(err, errMsg); enhanced != nil {
		return enhanced
	}
	if enhanced := enhancePolicyError(err, errMsg); enhanced != nil {
		return enhanced
	}
	if enhanced := enhanceValidationError(err, errMsg); enhanced != nil {
		return enhanced
	}
	if enhanced := enhanceNetworkError(err, errMsg); enhanced != nil {
		return enhanced
	}
	if enhanced := enhanceAPIKeyError(err, errMsg); enhanced != nil {
		return enhanced
	}
	if enhanced := enhanceGenericError(err, errMsg); enhanced != nil {
		return enhanced
	}

	return err
}

// enhanceFileNotFoundError adds suggestions for file not found errors
func enhanceFileNotFoundError(err error, errMsg string) error {
	if !strings.Contains(errMsg, "no such file or directory") {
		return nil
	}

	if strings.Contains(errMsg, "spec.yaml") {
		return NewErrorWithSuggestion(err,
			"Create a spec by running 'specular spec new' or 'specular spec new --from PRD.md'")
	}
	if strings.Contains(errMsg, "spec.lock.json") {
		return NewErrorWithSuggestion(err,
			"Generate a SpecLock by running 'specular spec lock'")
	}
	if strings.Contains(errMsg, "plan.json") {
		return NewErrorWithSuggestion(err,
			"Generate a plan by running 'specular plan create'")
	}
	if strings.Contains(errMsg, "policy.yaml") {
		return NewErrorWithSuggestion(err,
			"Use default policy or copy example: cp .specular/examples/policy.yaml .specular/policy.yaml")
	}
	if strings.Contains(errMsg, "providers.yaml") {
		return NewErrorWithSuggestion(err,
			"Configure providers by running 'specular init' or check .specular/providers.yaml.example")
	}

	return nil
}

// enhanceDockerError adds suggestions for Docker-related errors
func enhanceDockerError(err error, errMsg string) error {
	if strings.Contains(errMsg, "docker") && strings.Contains(errMsg, "daemon") {
		return NewErrorWithSuggestion(err,
			"Start Docker Desktop or Docker daemon, then try again")
	}
	if strings.Contains(errMsg, "Cannot connect to the Docker daemon") {
		return NewErrorWithSuggestion(err,
			"Docker is not running. Start Docker and run 'docker ps' to verify")
	}
	return nil
}

// enhancePermissionError adds suggestions for permission errors
func enhancePermissionError(err error, errMsg string) error {
	if !strings.Contains(errMsg, "permission denied") {
		return nil
	}

	if strings.Contains(errMsg, "/var/run/docker.sock") {
		return NewErrorWithSuggestion(err,
			"Add your user to the docker group: sudo usermod -aG docker $USER (then logout/login)")
	}
	return NewErrorWithSuggestion(err,
		"Check file permissions and ensure you have access to the required files/directories")
}

// enhanceProviderError adds suggestions for provider configuration errors
func enhanceProviderError(err error, errMsg string) error {
	if strings.Contains(errMsg, "no providers available") {
		return NewErrorWithSuggestion(err,
			"Configure at least one AI provider by running 'specular init' and selecting your providers")
	}
	if strings.Contains(errMsg, "provider") && (strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "not configured")) {
		return NewErrorWithSuggestion(err,
			"Check your provider configuration in .specular/routing.yaml or run 'specular init' to configure providers")
	}
	return nil
}

// enhancePolicyError adds suggestions for policy violation errors
func enhancePolicyError(err error, errMsg string) error {
	if strings.Contains(errMsg, "policy violation") || strings.Contains(errMsg, "docker_only") {
		return NewErrorWithSuggestion(err,
			"Policy requires Docker-only execution. Ensure Docker is running and tasks use allowed images")
	}
	return nil
}

// enhanceValidationError adds suggestions for validation errors
func enhanceValidationError(err error, errMsg string) error {
	if strings.Contains(errMsg, "validation failed") {
		return NewErrorWithSuggestion(err,
			"Fix the validation errors above, then run 'specular spec validate' to verify")
	}
	if strings.Contains(errMsg, "drift detected") {
		return NewErrorWithSuggestion(err,
			"Review drift with 'specular drift check' and update spec or code to align")
	}
	return nil
}

// enhanceNetworkError adds suggestions for network errors
func enhanceNetworkError(err error, errMsg string) error {
	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no route to host") {
		return NewErrorWithSuggestion(err,
			"Check your network connection and firewall settings")
	}
	return nil
}

// enhanceAPIKeyError adds suggestions for API key errors
func enhanceAPIKeyError(err error, errMsg string) error {
	if strings.Contains(errMsg, "API key") || strings.Contains(errMsg, "authentication") {
		return NewErrorWithSuggestion(err,
			"Set your API key environment variable (e.g., OPENAI_API_KEY, ANTHROPIC_API_KEY)")
	}
	return nil
}

// enhanceGenericError adds generic suggestions for unmatched errors
func enhanceGenericError(err error, errMsg string) error {
	if strings.Contains(errMsg, "failed to") {
		return NewErrorWithSuggestion(err,
			fmt.Sprintf("Next steps: %s", SuggestNextSteps()))
	}
	return nil
}

// FormatError provides consistent error formatting with context
func FormatError(err error, context string) error {
	if err == nil {
		return nil
	}

	enhanced := EnhanceError(err)
	if context != "" {
		return fmt.Errorf("%s: %w", context, enhanced)
	}
	return enhanced
}
