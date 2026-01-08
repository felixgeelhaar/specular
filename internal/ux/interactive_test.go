package ux

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewInteractiveConfig(t *testing.T) {
	// Save current env and restore after test
	oldCI := os.Getenv("CI")
	defer os.Setenv("CI", oldCI)

	// Test without CI
	os.Unsetenv("CI")
	cfg := NewInteractiveConfig()
	if cfg.IsCI {
		t.Error("expected IsCI to be false when CI env is not set")
	}
	if cfg.ForcePrompt {
		t.Error("expected ForcePrompt to be false by default")
	}
	if cfg.Quiet {
		t.Error("expected Quiet to be false by default")
	}

	// Test with CI
	os.Setenv("CI", "true")
	cfg = NewInteractiveConfig()
	if !cfg.IsCI {
		t.Error("expected IsCI to be true when CI env is set")
	}
}

func TestDetectCIEnvironment(t *testing.T) {
	// Save and restore all CI env vars
	ciVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"JENKINS_URL",
		"TRAVIS",
		"CIRCLECI",
	}

	savedVars := make(map[string]string)
	for _, v := range ciVars {
		savedVars[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range savedVars {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	// Test no CI env
	if DetectCIEnvironment() {
		t.Error("expected false when no CI env vars are set")
	}

	// Test each CI env var
	testCases := []struct {
		envVar string
		value  string
	}{
		{"CI", "true"},
		{"GITHUB_ACTIONS", "true"},
		{"GITLAB_CI", "true"},
		{"JENKINS_URL", "http://jenkins.example.com"},
		{"TRAVIS", "true"},
		{"CIRCLECI", "true"},
	}

	for _, tc := range testCases {
		t.Run(tc.envVar, func(t *testing.T) {
			os.Setenv(tc.envVar, tc.value)
			if !DetectCIEnvironment() {
				t.Errorf("expected true when %s is set", tc.envVar)
			}
			os.Unsetenv(tc.envVar)
		})
	}
}

func TestShouldPrompt(t *testing.T) {
	tests := []struct {
		name     string
		cfg      InteractiveConfig
		expected bool
	}{
		{
			name: "quiet mode disables prompts",
			cfg: InteractiveConfig{
				IsCI:        false,
				ForcePrompt: true,
				Quiet:       true,
			},
			expected: false,
		},
		{
			name: "force prompt overrides CI",
			cfg: InteractiveConfig{
				IsCI:        true,
				ForcePrompt: true,
				Quiet:       false,
			},
			expected: true,
		},
		{
			name: "CI environment disables prompts",
			cfg: InteractiveConfig{
				IsCI:        true,
				ForcePrompt: false,
				Quiet:       false,
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Note: This test won't fully work in CI because IsInteractive()
			// checks if stdin is a terminal, which it won't be in tests.
			// We're testing the logic flow here.
			result := ShouldPrompt(tc.cfg)

			// Only check expected cases that don't depend on terminal state
			if tc.cfg.Quiet && result {
				t.Error("expected false when quiet is true")
			}
			if tc.cfg.ForcePrompt && !tc.cfg.Quiet && !result {
				t.Error("expected true when force prompt is set and not quiet")
			}
			if tc.cfg.IsCI && !tc.cfg.ForcePrompt && result {
				t.Error("expected false in CI without force prompt")
			}
		})
	}
}

func TestValidateRequiredFlags(t *testing.T) {
	// Create a test command with flags
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("goal", "", "goal description")
	cmd.Flags().String("profile", "default", "profile description")
	cmd.Flags().String("optional", "", "optional flag")

	prompts := []FlagPrompt{
		{Name: "goal", Required: true},
		{Name: "profile", Required: true},
		{Name: "optional", Required: false},
	}

	// Test with missing required flag
	err := validateRequiredFlags(cmd, prompts)
	if err == nil {
		t.Error("expected error for missing required flag 'goal'")
	}

	// Set the required flag
	cmd.Flags().Set("goal", "test goal")
	err = validateRequiredFlags(cmd, prompts)
	if err != nil {
		t.Errorf("unexpected error after setting goal: %v", err)
	}
}

func TestMustHaveValue(t *testing.T) {
	tests := []struct {
		input    string
		hasError bool
	}{
		{"valid", false},
		{"", true},
		{"  ", true},
		{"  valid  ", false},
	}

	for _, tc := range tests {
		err := MustHaveValue(tc.input)
		if tc.hasError && err == nil {
			t.Errorf("expected error for input %q", tc.input)
		}
		if !tc.hasError && err != nil {
			t.Errorf("unexpected error for input %q: %v", tc.input, err)
		}
	}
}

func TestMustBeOneOf(t *testing.T) {
	allowed := []string{"alpha", "beta", "gamma"}
	validator := MustBeOneOf(allowed)

	tests := []struct {
		input    string
		hasError bool
	}{
		{"alpha", false},
		{"beta", false},
		{"gamma", false},
		{"delta", true},
		{"", true},
		{"ALPHA", true}, // case sensitive
	}

	for _, tc := range tests {
		err := validator(tc.input)
		if tc.hasError && err == nil {
			t.Errorf("expected error for input %q", tc.input)
		}
		if !tc.hasError && err != nil {
			t.Errorf("unexpected error for input %q: %v", tc.input, err)
		}
	}
}

func TestPromptType_Constants(t *testing.T) {
	// Verify prompt types are distinct
	types := map[PromptType]string{
		PromptText:        "text",
		PromptSelect:      "select",
		PromptConfirm:     "confirm",
		PromptMultiSelect: "multiselect",
	}

	if len(types) != 4 {
		t.Error("expected 4 distinct prompt types")
	}
}

func TestFlagPrompt_Struct(t *testing.T) {
	// Test creating a FlagPrompt
	prompt := FlagPrompt{
		Name:        "goal",
		Description: "What is your goal?",
		Type:        PromptText,
		Required:    true,
		Default:     "default goal",
		Choices:     nil,
		Validator:   MustHaveValue,
	}

	if prompt.Name != "goal" {
		t.Errorf("expected name 'goal', got %q", prompt.Name)
	}
	if prompt.Type != PromptText {
		t.Errorf("expected type PromptText, got %v", prompt.Type)
	}
	if !prompt.Required {
		t.Error("expected Required to be true")
	}
	if prompt.Validator == nil {
		t.Error("expected Validator to be set")
	}
}

func TestInteractiveConfig_Struct(t *testing.T) {
	cfg := InteractiveConfig{
		IsCI:        true,
		ForcePrompt: false,
		Quiet:       true,
	}

	if !cfg.IsCI {
		t.Error("expected IsCI to be true")
	}
	if cfg.ForcePrompt {
		t.Error("expected ForcePrompt to be false")
	}
	if !cfg.Quiet {
		t.Error("expected Quiet to be true")
	}
}

func TestPromptMissingFlags_NonInteractive(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("goal", "", "goal description")

	prompts := []FlagPrompt{
		{Name: "goal", Required: true},
	}

	// Non-interactive config (CI mode)
	cfg := InteractiveConfig{
		IsCI:        true,
		ForcePrompt: false,
		Quiet:       false,
	}

	// Should return error for missing required flag in non-interactive mode
	err := PromptMissingFlags(cmd, prompts, cfg)
	if err == nil {
		t.Error("expected error for missing required flag in non-interactive mode")
	}

	// Set the flag and try again
	cmd.Flags().Set("goal", "test goal")
	err = PromptMissingFlags(cmd, prompts, cfg)
	if err != nil {
		t.Errorf("unexpected error after setting flag: %v", err)
	}
}

func TestPromptMissingFlags_AlreadySet(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("goal", "", "goal description")
	cmd.Flags().Set("goal", "already set") // Mark as changed

	prompts := []FlagPrompt{
		{Name: "goal", Required: true},
	}

	cfg := InteractiveConfig{
		IsCI:        true,
		ForcePrompt: false,
		Quiet:       false,
	}

	// Should succeed because flag is already set
	err := PromptMissingFlags(cmd, prompts, cfg)
	if err != nil {
		t.Errorf("unexpected error for already-set flag: %v", err)
	}
}

func TestPromptMissingFlags_NonExistentFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	// Note: not adding any flags to the command

	prompts := []FlagPrompt{
		{Name: "nonexistent", Required: true},
	}

	cfg := InteractiveConfig{
		IsCI:        true,
		ForcePrompt: false,
		Quiet:       false,
	}

	// Should not error for non-existent flags (they're just skipped)
	err := PromptMissingFlags(cmd, prompts, cfg)
	if err != nil {
		t.Errorf("unexpected error for non-existent flag: %v", err)
	}
}
