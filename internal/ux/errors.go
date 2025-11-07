package ux

import (
	"fmt"
	"strings"
)

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

	// File not found errors
	if strings.Contains(errMsg, "no such file or directory") {
		if strings.Contains(errMsg, "spec.yaml") {
			return NewErrorWithSuggestion(err,
				"Create a spec by running 'specular interview' or 'specular spec generate --in PRD.md'")
		}
		if strings.Contains(errMsg, "spec.lock.json") {
			return NewErrorWithSuggestion(err,
				"Generate a SpecLock by running 'specular spec lock'")
		}
		if strings.Contains(errMsg, "plan.json") {
			return NewErrorWithSuggestion(err,
				"Generate a plan by running 'specular plan'")
		}
		if strings.Contains(errMsg, "policy.yaml") {
			return NewErrorWithSuggestion(err,
				"Use default policy or copy example: cp .specular/examples/policy.yaml .specular/policy.yaml")
		}
		if strings.Contains(errMsg, "providers.yaml") {
			return NewErrorWithSuggestion(err,
				"Configure providers by running 'specular init' or check .specular/examples/providers.yaml")
		}
	}

	// Docker errors
	if strings.Contains(errMsg, "docker") && strings.Contains(errMsg, "daemon") {
		return NewErrorWithSuggestion(err,
			"Start Docker Desktop or Docker daemon, then try again")
	}

	if strings.Contains(errMsg, "Cannot connect to the Docker daemon") {
		return NewErrorWithSuggestion(err,
			"Docker is not running. Start Docker and run 'docker ps' to verify")
	}

	// Permission errors
	if strings.Contains(errMsg, "permission denied") {
		if strings.Contains(errMsg, "/var/run/docker.sock") {
			return NewErrorWithSuggestion(err,
				"Add your user to the docker group: sudo usermod -aG docker $USER (then logout/login)")
		}
		return NewErrorWithSuggestion(err,
			"Check file permissions and ensure you have access to the required files/directories")
	}

	// Provider errors
	if strings.Contains(errMsg, "no providers available") {
		return NewErrorWithSuggestion(err,
			"Configure at least one AI provider by running 'specular init' and selecting your providers")
	}

	if strings.Contains(errMsg, "provider") && (strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "not configured")) {
		return NewErrorWithSuggestion(err,
			"Check your provider configuration in .specular/router.yaml or run 'specular init' to configure providers")
	}

	// Policy violations
	if strings.Contains(errMsg, "policy violation") || strings.Contains(errMsg, "docker_only") {
		return NewErrorWithSuggestion(err,
			"Policy requires Docker-only execution. Ensure Docker is running and tasks use allowed images")
	}

	// Validation errors
	if strings.Contains(errMsg, "validation failed") {
		return NewErrorWithSuggestion(err,
			"Fix the validation errors above, then run 'specular spec validate' to verify")
	}

	// Drift detected
	if strings.Contains(errMsg, "drift detected") {
		return NewErrorWithSuggestion(err,
			"Review drift with 'specular build drift' and update spec or code to align")
	}

	// Network errors
	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no route to host") {
		return NewErrorWithSuggestion(err,
			"Check your network connection and firewall settings")
	}

	// API key errors
	if strings.Contains(errMsg, "API key") || strings.Contains(errMsg, "authentication") {
		return NewErrorWithSuggestion(err,
			"Set your API key environment variable (e.g., OPENAI_API_KEY, ANTHROPIC_API_KEY)")
	}

	// Generic suggestion based on error type
	if strings.Contains(errMsg, "failed to") {
		return NewErrorWithSuggestion(err,
			fmt.Sprintf("Next steps: %s", SuggestNextSteps()))
	}

	return err
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
