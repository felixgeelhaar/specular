package cmd

import (
	"fmt"
	"strings"
)

// ErrorWithSuggestion wraps an error with actionable recovery suggestions
type ErrorWithSuggestion struct {
	Message     string
	Suggestions []string
	err         error
}

func (e *ErrorWithSuggestion) Error() string {
	var b strings.Builder
	b.WriteString(e.Message)

	if len(e.Suggestions) > 0 {
		b.WriteString("\n\nSuggestions:")
		for _, s := range e.Suggestions {
			b.WriteString("\n  • ")
			b.WriteString(s)
		}
	}

	if e.err != nil {
		b.WriteString("\n\nDetails: ")
		b.WriteString(e.err.Error())
	}

	return b.String()
}

func (e *ErrorWithSuggestion) Unwrap() error {
	return e.err
}

// NewErrorWithSuggestions creates an error with recovery suggestions
func NewErrorWithSuggestions(msg string, err error, suggestions ...string) error {
	return &ErrorWithSuggestion{
		Message:     msg,
		Suggestions: suggestions,
		err:         err,
	}
}

// ProfileLoadError creates a helpful error for profile loading failures
func ProfileLoadError(profileName string, err error) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("Failed to load profile %q", profileName),
		err,
		"List available profiles: specular auto --list-profiles",
		"Use a built-in profile: --profile default, --profile ci, or --profile strict",
		"Check your custom profile at ~/.specular/auto.profiles.yaml or ./auto.profiles.yaml",
		"Create a new profile: copy from internal/profiles/builtin/default.yaml",
	)
}

// ProviderLoadError creates a helpful error for provider loading failures
func ProviderLoadError(configPath string, err error) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("Failed to load provider configuration from %q", configPath),
		err,
		"Initialize providers: specular init",
		"Check provider configuration: cat ~/.specular/providers.yaml",
		"Verify API keys are set in environment variables",
		"Run health check: specular provider health",
	)
}

// RouterError creates a helpful error for routing failures
func RouterError(err error) error {
	return NewErrorWithSuggestions(
		"Failed to create router",
		err,
		"Check that at least one provider is configured",
		"Verify provider configuration: specular provider list",
		"Test provider connectivity: specular provider health",
		"Review routing configuration in your profile",
	)
}

// CheckpointNotFoundError creates a helpful error when checkpoint is not found
func CheckpointNotFoundError(checkpointID string) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("Checkpoint %q not found", checkpointID),
		nil,
		"List available checkpoints: specular checkpoint list",
		"View checkpoint details: specular checkpoint show <id>",
		"Start a new session instead of resuming",
	)
}

// FileNotFoundError creates a helpful error for missing files
func FileNotFoundError(filePath string, suggestions ...string) error {
	defaultSuggestions := []string{
		fmt.Sprintf("Check if file exists: ls -l %s", filePath),
		fmt.Sprintf("Verify the path is correct"),
	}

	allSuggestions := append(defaultSuggestions, suggestions...)

	return NewErrorWithSuggestions(
		fmt.Sprintf("File not found: %s", filePath),
		nil,
		allSuggestions...
	)
}

// ValidationError creates a helpful error for validation failures
func ValidationError(field string, value interface{}, validValues string) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("Invalid value for %s: %v", field, value),
		nil,
		fmt.Sprintf("Valid values: %s", validValues),
		"Run with --help to see all available options",
	)
}

// JSONOutputError creates a helpful error when JSON output is not available
func JSONOutputError() error {
	return NewErrorWithSuggestions(
		"JSON output not available",
		nil,
		"Ensure the command completed successfully",
		"Check that --json flag is set early in the command",
		"Review logs for any errors: cat ~/.specular/logs/latest.log",
	)
}

// PolicyNotFoundError creates a helpful error when policy file is missing
func PolicyNotFoundError(policyPath string) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("Policy file not found: %s", policyPath),
		nil,
		"Initialize policy: specular init",
		"Create policy manually at: .specular/policy.yaml",
		"Use a different policy file: --policy <path>",
		"Skip policy enforcement: use --no-policy flag (not recommended for production)",
	)
}
