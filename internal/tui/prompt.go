package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
)

// Prompt represents a simple interactive prompt configuration
type Prompt struct {
	Message     string
	Default     string
	Placeholder string
	Required    bool
}

// PromptForString displays an interactive prompt and returns the user's input
func PromptForString(p Prompt) (string, error) {
	var value string

	input := huh.NewInput().
		Title(p.Message).
		Placeholder(p.Placeholder).
		Value(&value)

	if p.Default != "" {
		input = input.Value(&value)
		value = p.Default
	}

	form := huh.NewForm(huh.NewGroup(input))

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("prompt failed: %w", err)
	}

	if p.Required && value == "" {
		return "", fmt.Errorf("value is required")
	}

	return value, nil
}

// PromptForConfirmation displays a yes/no confirmation prompt
func PromptForConfirmation(message string, defaultValue bool) (bool, error) {
	var confirmed bool = defaultValue

	confirm := huh.NewConfirm().
		Title(message).
		Value(&confirmed)

	form := huh.NewForm(huh.NewGroup(confirm))

	if err := form.Run(); err != nil {
		return false, fmt.Errorf("prompt failed: %w", err)
	}

	return confirmed, nil
}

// PromptForSelect displays a selection prompt with multiple options
func PromptForSelect(message string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options provided")
	}

	// Convert options to huh options
	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt, opt)
	}

	var selected string
	selectField := huh.NewSelect[string]().
		Title(message).
		Options(huhOptions...).
		Value(&selected)

	form := huh.NewForm(huh.NewGroup(selectField))

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("prompt failed: %w", err)
	}

	return selected, nil
}

// PromptForMultiSelect displays a multi-selection prompt
func PromptForMultiSelect(message string, options []string) ([]string, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("no options provided")
	}

	// Convert options to huh options
	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt, opt)
	}

	var selected []string
	multiSelect := huh.NewMultiSelect[string]().
		Title(message).
		Options(huhOptions...).
		Value(&selected)

	form := huh.NewForm(huh.NewGroup(multiSelect))

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("prompt failed: %w", err)
	}

	return selected, nil
}

// IsInteractive returns true if stdin is a terminal (not piped)
func IsInteractive() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ShouldPrompt returns true if prompts should be shown based on environment
// Prompts are disabled in CI environments or when stdin is not a terminal
func ShouldPrompt() bool {
	// Check common CI environment variables
	ciEnvVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"JENKINS_URL",
		"TRAVIS",
		"CIRCLECI",
		"BUILDKITE",
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return false
		}
	}

	return IsInteractive()
}
