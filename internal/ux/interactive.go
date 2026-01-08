// Package ux provides user experience utilities for the Specular CLI.
//
// The interactive module handles prompting users for missing required flags
// in interactive terminal sessions while respecting CI environments.
package ux

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/felixgeelhaar/specular/internal/tui"
	"github.com/spf13/cobra"
)

// PromptType defines the type of interactive prompt to display
type PromptType int

const (
	// PromptText displays a text input field
	PromptText PromptType = iota
	// PromptSelect displays a single-choice selection
	PromptSelect
	// PromptConfirm displays a yes/no confirmation
	PromptConfirm
	// PromptMultiSelect displays a multi-choice selection
	PromptMultiSelect
)

// InteractiveConfig controls when and how interactive prompts are shown
type InteractiveConfig struct {
	// IsCI indicates the session is running in a CI environment
	IsCI bool
	// ForcePrompt forces prompts even in non-interactive environments
	ForcePrompt bool
	// Quiet suppresses all interactive prompts
	Quiet bool
}

// FlagPrompt defines an interactive prompt for a missing flag
type FlagPrompt struct {
	// Name is the flag name (e.g., "goal", "profile")
	Name string
	// Description is shown as the prompt title
	Description string
	// Type determines the prompt UI
	Type PromptType
	// Required indicates if the flag must have a value
	Required bool
	// Default is the default value if user provides none
	Default interface{}
	// Choices are the available options for Select/MultiSelect
	Choices []string
	// Validator validates user input (optional)
	Validator func(string) error
}

// NewInteractiveConfig creates an InteractiveConfig by detecting the environment
func NewInteractiveConfig() InteractiveConfig {
	return InteractiveConfig{
		IsCI:        DetectCIEnvironment(),
		ForcePrompt: false,
		Quiet:       false,
	}
}

// ShouldPrompt determines if interactive prompts should be shown
func ShouldPrompt(cfg InteractiveConfig) bool {
	// Never prompt in quiet mode
	if cfg.Quiet {
		return false
	}

	// Force prompt overrides CI detection
	if cfg.ForcePrompt {
		return true
	}

	// Don't prompt in CI environments
	if cfg.IsCI {
		return false
	}

	// Check if terminal is interactive
	return tui.IsInteractive()
}

// DetectCIEnvironment checks if running in a CI/CD environment
func DetectCIEnvironment() bool {
	// Common CI environment variables
	ciEnvVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"JENKINS_URL",
		"TRAVIS",
		"CIRCLECI",
		"BUILDKITE",
		"DRONE",
		"AZURE_PIPELINES",
		"TEAMCITY_VERSION",
		"CODEBUILD_BUILD_ID",
		"BITBUCKET_COMMIT",
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// PromptMissingFlags prompts the user for missing required flags
// and sets their values on the command.
func PromptMissingFlags(cmd *cobra.Command, prompts []FlagPrompt, cfg InteractiveConfig) error {
	if !ShouldPrompt(cfg) {
		// In non-interactive mode, just check if required flags are set
		return validateRequiredFlags(cmd, prompts)
	}

	for _, prompt := range prompts {
		// Check if flag already has a value
		flag := cmd.Flags().Lookup(prompt.Name)
		if flag == nil {
			continue // Flag doesn't exist on this command
		}

		// Skip if flag was explicitly set
		if flag.Changed {
			continue
		}

		// Skip non-required flags that have defaults
		if !prompt.Required && prompt.Default != nil {
			continue
		}

		// Prompt for the value
		value, err := promptForValue(prompt)
		if err != nil {
			return fmt.Errorf("failed to prompt for %s: %w", prompt.Name, err)
		}

		// Validate if validator is provided
		if prompt.Validator != nil {
			if err := prompt.Validator(value); err != nil {
				return fmt.Errorf("invalid value for %s: %w", prompt.Name, err)
			}
		}

		// Set the flag value
		if err := cmd.Flags().Set(prompt.Name, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", prompt.Name, err)
		}
	}

	return nil
}

// promptForValue displays the appropriate prompt and returns the user's input
func promptForValue(p FlagPrompt) (string, error) {
	switch p.Type {
	case PromptText:
		return promptText(p)
	case PromptSelect:
		return promptSelect(p)
	case PromptConfirm:
		return promptConfirm(p)
	case PromptMultiSelect:
		return promptMultiSelect(p)
	default:
		return promptText(p)
	}
}

func promptText(p FlagPrompt) (string, error) {
	defaultStr := ""
	if p.Default != nil {
		defaultStr = fmt.Sprintf("%v", p.Default)
	}

	prompt := tui.Prompt{
		Message:     p.Description,
		Default:     defaultStr,
		Placeholder: p.Description,
		Required:    p.Required,
	}

	return tui.PromptForString(prompt)
}

func promptSelect(p FlagPrompt) (string, error) {
	if len(p.Choices) == 0 {
		// Fall back to text prompt if no choices
		return promptText(p)
	}

	return tui.PromptForSelect(p.Description, p.Choices)
}

func promptConfirm(p FlagPrompt) (string, error) {
	defaultBool := false
	if p.Default != nil {
		if b, ok := p.Default.(bool); ok {
			defaultBool = b
		}
	}

	confirmed, err := tui.PromptForConfirmation(p.Description, defaultBool)
	if err != nil {
		return "", err
	}

	return strconv.FormatBool(confirmed), nil
}

func promptMultiSelect(p FlagPrompt) (string, error) {
	if len(p.Choices) == 0 {
		return promptText(p)
	}

	selected, err := tui.PromptForMultiSelect(p.Description, p.Choices)
	if err != nil {
		return "", err
	}

	return strings.Join(selected, ","), nil
}

// validateRequiredFlags checks that all required flags have values
func validateRequiredFlags(cmd *cobra.Command, prompts []FlagPrompt) error {
	var missing []string

	for _, prompt := range prompts {
		if !prompt.Required {
			continue
		}

		flag := cmd.Flags().Lookup(prompt.Name)
		if flag == nil {
			continue
		}

		// Check if flag has a value (either changed or has default)
		if !flag.Changed && flag.Value.String() == "" {
			missing = append(missing, prompt.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required flags: %s (use --help for usage)", strings.Join(missing, ", "))
	}

	return nil
}

// MustHaveValue is a validator that ensures the value is non-empty
func MustHaveValue(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("value cannot be empty")
	}
	return nil
}

// MustBeOneOf creates a validator that ensures the value is one of the allowed values
func MustBeOneOf(allowed []string) func(string) error {
	return func(s string) error {
		for _, a := range allowed {
			if s == a {
				return nil
			}
		}
		return fmt.Errorf("must be one of: %s", strings.Join(allowed, ", "))
	}
}
