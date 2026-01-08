package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/profiles"
	"github.com/felixgeelhaar/specular/internal/validate"
)

// TestAutoSubcommands tests that all auto subcommands are registered
func TestAutoSubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"resume":  false,
		"history": false,
		"explain": false,
	}

	for _, cmd := range autoCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand '%s' not found in auto command", name)
		}
	}
}

// TestCommandAliases tests that main commands have their aliases configured
func TestCommandAliases(t *testing.T) {
	tests := []struct {
		cmd     *cobra.Command
		name    string
		aliases []string
	}{
		{autoCmd, "auto", []string{"a", "run"}},
		{specCmd, "spec", []string{"s"}},
		{planCmd, "plan", []string{"p"}},
		{buildCmd, "build", []string{"b"}},
		{initCmd, "init", []string{"i", "new"}},
		{configCmd, "config", []string{"c", "cfg"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.cmd.Aliases) != len(tc.aliases) {
				t.Errorf("%s: expected %d aliases, got %d", tc.name, len(tc.aliases), len(tc.cmd.Aliases))
				return
			}

			for i, expected := range tc.aliases {
				if tc.cmd.Aliases[i] != expected {
					t.Errorf("%s: alias[%d] = %q, want %q", tc.name, i, tc.cmd.Aliases[i], expected)
				}
			}
		})
	}
}

// TestAutoResumeFlags tests that auto resume command has correct configuration
func TestAutoResumeFlags(t *testing.T) {
	// Find resume subcommand
	var resumeCmd *cobra.Command
	for _, cmd := range autoCmd.Commands() {
		if cmd.Name() == "resume" {
			resumeCmd = cmd
			break
		}
	}

	if resumeCmd == nil {
		t.Fatal("resume subcommand not found")
	}

	// Check command configuration
	if resumeCmd.Use != "resume [session-id]" {
		t.Errorf("resume Use = %q, want %q", resumeCmd.Use, "resume [session-id]")
	}

	if resumeCmd.Short == "" {
		t.Error("resume Short description is empty")
	}
}

// TestAutoHistoryFlags tests that auto history command has correct configuration
func TestAutoHistoryFlags(t *testing.T) {
	// Find history subcommand
	var historyCmd *cobra.Command
	for _, cmd := range autoCmd.Commands() {
		if cmd.Name() == "history" {
			historyCmd = cmd
			break
		}
	}

	if historyCmd == nil {
		t.Fatal("history subcommand not found")
	}

	// Check command configuration
	if historyCmd.Use != "history" {
		t.Errorf("history Use = %q, want %q", historyCmd.Use, "history")
	}

	if historyCmd.Short == "" {
		t.Error("history Short description is empty")
	}
}

// TestAutoExplainFlags tests that auto explain command has correct configuration
func TestAutoExplainFlags(t *testing.T) {
	// Find explain subcommand
	var explainCmd *cobra.Command
	for _, cmd := range autoCmd.Commands() {
		if cmd.Name() == "explain" {
			explainCmd = cmd
			break
		}
	}

	if explainCmd == nil {
		t.Fatal("explain subcommand not found")
	}

	// Check command configuration
	if explainCmd.Use != "explain <session-id> [step]" {
		t.Errorf("explain Use = %q, want %q", explainCmd.Use, "explain <session-id> [step]")
	}

	if explainCmd.Short == "" {
		t.Error("explain Short description is empty")
	}
}

// TestAutoBackwardCompatibilityFlags tests that old auto flags still exist
func TestAutoBackwardCompatibilityFlags(t *testing.T) {
	// Test that the root auto command still has important flags for backward compatibility
	if autoCmd.Flags().Lookup("resume") == nil {
		t.Error("backward compatibility flag 'resume' not found on auto command")
	}
	if autoCmd.Flags().Lookup("profile") == nil {
		t.Error("backward compatibility flag 'profile' not found on auto command")
	}
	if autoCmd.Flags().Lookup("dry-run") == nil {
		t.Error("backward compatibility flag 'dry-run' not found on auto command")
	}
	if autoCmd.Flags().Lookup("verbose") == nil {
		t.Error("backward compatibility flag 'verbose' not found on auto command")
	}
}

// TestAutoCostEstimateFlag tests that the estimate-cost flag exists
func TestAutoCostEstimateFlag(t *testing.T) {
	flag := autoCmd.Flags().Lookup("estimate-cost")
	if flag == nil {
		t.Fatal("estimate-cost flag not found on auto command")
	}

	if flag.DefValue != "false" {
		t.Errorf("estimate-cost default value = %q, want %q", flag.DefValue, "false")
	}
}

// TestAutoTUIFlag tests that the TUI flag exists
func TestAutoTUIFlag(t *testing.T) {
	flag := autoCmd.Flags().Lookup("tui")
	if flag == nil {
		t.Fatal("tui flag not found on auto command")
	}

	if flag.DefValue != "false" {
		t.Errorf("tui default value = %q, want %q", flag.DefValue, "false")
	}
}

// TestAutoNoApprovalFlag tests that the no-approval flag exists
func TestAutoNoApprovalFlag(t *testing.T) {
	flag := autoCmd.Flags().Lookup("no-approval")
	if flag == nil {
		t.Fatal("no-approval flag not found on auto command")
	}
}

// TestAutoSavePatchesFlag tests that the save-patches flag exists
func TestAutoSavePatchesFlag(t *testing.T) {
	flag := autoCmd.Flags().Lookup("save-patches")
	if flag == nil {
		t.Fatal("save-patches flag not found on auto command")
	}
}

// TestAutoAllFlagsExist tests comprehensive flag coverage
func TestAutoAllFlagsExist(t *testing.T) {
	expectedFlags := []string{
		"profile",
		"dry-run",
		"verbose",
		"resume",
		"estimate-cost",
		"tui",
		"no-approval",
		"save-patches",
		"attest",
		"max-cost",
		"max-retries",
		"output",
	}

	for _, flagName := range expectedFlags {
		if autoCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("expected flag %q not found on auto command", flagName)
		}
	}
}

// TestAutoArgsValidation tests command argument validation
func TestAutoArgsValidation(t *testing.T) {
	// Note: The autoCmd.Args function checks flags on the passed command,
	// but also checks tui.ShouldPrompt() which may allow empty args in
	// interactive mode. We test basic validations here.

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "valid_goal",
			args:      []string{"Build a REST API"},
			wantError: false,
		},
		{
			name:      "multiple_words_goal",
			args:      []string{"Build", "a", "REST", "API"},
			wantError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use the actual autoCmd for validation
			err := autoCmd.Args(autoCmd, tc.args)

			if tc.wantError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestAutoListProfiles tests the --list-profiles functionality
func TestAutoListProfiles(t *testing.T) {
	// This tests that listAvailableProfiles can be called
	// and returns profiles from the registry
	loader := profiles.NewLoader()

	// Built-in profiles should always exist
	defaultProfile, err := loader.Load("default")
	if err != nil {
		t.Fatalf("failed to load default profile: %v", err)
	}
	if defaultProfile.Name != "default" {
		t.Errorf("expected profile name 'default', got %q", defaultProfile.Name)
	}

	ciProfile, err := loader.Load("ci")
	if err != nil {
		t.Fatalf("failed to load ci profile: %v", err)
	}
	if ciProfile.Name != "ci" {
		t.Errorf("expected profile name 'ci', got %q", ciProfile.Name)
	}

	strictProfile, err := loader.Load("strict")
	if err != nil {
		t.Fatalf("failed to load strict profile: %v", err)
	}
	if strictProfile.Name != "strict" {
		t.Errorf("expected profile name 'strict', got %q", strictProfile.Name)
	}
}

// TestAutoProfileLoading tests profile loading and validation
func TestAutoProfileLoading(t *testing.T) {
	loader := profiles.NewLoader()

	tests := []struct {
		name      string
		profile   string
		wantError bool
	}{
		{"default_profile", "default", false},
		{"ci_profile", "ci", false},
		{"strict_profile", "strict", false},
		{"nonexistent_profile", "nonexistent-profile-xyz", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loader.Load(tc.profile)
			if tc.wantError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestAutoGoalValidation tests goal validation with the validate package
func TestAutoGoalValidation(t *testing.T) {
	tests := []struct {
		name      string
		goal      string
		wantError bool
	}{
		{"valid_goal", "Build a REST API for user management", false},
		{"empty_goal", "", true},
		{"whitespace_goal", "   ", true},
		{"too_short_goal", "Hi", true},                  // Less than 3 characters - invalid
		{"minimum_valid_goal", "Fix", false},            // Exactly 3 characters - valid
		{"long_goal", strings.Repeat("x", 10001), true}, // Too long
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validate.Goal(tc.goal)
			if tc.wantError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestAutoOutputDirectory tests output directory handling
func TestAutoOutputDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name   string
		output string
	}{
		{"default_output", ".specular"},
		{"custom_output", filepath.Join(tmpDir, "custom-output")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Just verify the path is valid
			if tc.output == "" {
				t.Error("output path should not be empty")
			}

			// Test that directory can be created
			if tc.output != ".specular" {
				err := os.MkdirAll(tc.output, 0755)
				if err != nil {
					t.Errorf("failed to create output directory: %v", err)
				}
			}
		})
	}
}

// TestAutoDryRunMode tests that dry-run flag is respected
func TestAutoDryRunMode(t *testing.T) {
	flag := autoCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("dry-run flag not found")
	}

	// Default should be false
	if flag.DefValue != "false" {
		t.Errorf("dry-run default = %q, want %q", flag.DefValue, "false")
	}
}

// TestAutoMaxCostValidation tests max-cost flag validation
func TestAutoMaxCostValidation(t *testing.T) {
	flag := autoCmd.Flags().Lookup("max-cost")
	if flag == nil {
		t.Fatal("max-cost flag not found")
	}

	// Flag should accept float values
	// Default is 0 (unlimited)
	if flag.DefValue != "0" {
		t.Errorf("max-cost default = %q, want %q", flag.DefValue, "0")
	}
}

// TestAutoTimeoutValidation tests timeout flag
func TestAutoTimeoutValidation(t *testing.T) {
	flag := autoCmd.Flags().Lookup("timeout")
	if flag == nil {
		t.Fatal("timeout flag not found")
	}

	// Default timeout should be reasonable (e.g., 30 minutes)
	if flag.DefValue == "" {
		t.Error("timeout default should not be empty")
	}
}

// TestAutoVerboseMode tests verbose flag
func TestAutoVerboseMode(t *testing.T) {
	flag := autoCmd.Flags().Lookup("verbose")
	if flag == nil {
		t.Fatal("verbose flag not found")
	}

	// Default should be false
	if flag.DefValue != "false" {
		t.Errorf("verbose default = %q, want %q", flag.DefValue, "false")
	}
}

// TestAutoJSONOutput tests JSON output flag
func TestAutoJSONOutput(t *testing.T) {
	flag := autoCmd.Flags().Lookup("json")
	if flag == nil {
		t.Fatal("json flag not found")
	}

	// Default should be false
	if flag.DefValue != "false" {
		t.Errorf("json default = %q, want %q", flag.DefValue, "false")
	}
}

// TestAutoScopeFlag tests scope filtering flag
func TestAutoScopeFlag(t *testing.T) {
	flag := autoCmd.Flags().Lookup("scope")
	if flag == nil {
		t.Fatal("scope flag not found")
	}

	// Should be a string slice
	if flag.Value.Type() != "stringSlice" {
		t.Errorf("scope type = %q, want %q", flag.Value.Type(), "stringSlice")
	}
}

// TestAutoIncludeDependenciesFlag tests include-dependencies flag
func TestAutoIncludeDependenciesFlag(t *testing.T) {
	flag := autoCmd.Flags().Lookup("include-dependencies")
	if flag == nil {
		t.Fatal("include-dependencies flag not found")
	}

	// Default should be true
	if flag.DefValue != "true" {
		t.Errorf("include-dependencies default = %q, want %q", flag.DefValue, "true")
	}
}

// TestAutoTraceFlag tests trace flag for debugging
func TestAutoTraceFlag(t *testing.T) {
	flag := autoCmd.Flags().Lookup("trace")
	if flag == nil {
		t.Fatal("trace flag not found")
	}

	// Default should be false
	if flag.DefValue != "false" {
		t.Errorf("trace default = %q, want %q", flag.DefValue, "false")
	}
}

// TestAutoMaxStepsFlag tests max-steps flag
func TestAutoMaxStepsFlag(t *testing.T) {
	flag := autoCmd.Flags().Lookup("max-steps")
	if flag == nil {
		t.Fatal("max-steps flag not found")
	}

	// Default should be 0 (unlimited)
	if flag.DefValue != "0" {
		t.Errorf("max-steps default = %q, want %q", flag.DefValue, "0")
	}
}

// TestAutoMaxCostPerTaskFlag tests max-cost-per-task flag
func TestAutoMaxCostPerTaskFlag(t *testing.T) {
	flag := autoCmd.Flags().Lookup("max-cost-per-task")
	if flag == nil {
		t.Fatal("max-cost-per-task flag not found")
	}

	// Default should be 0 (unlimited)
	if flag.DefValue != "0" {
		t.Errorf("max-cost-per-task default = %q, want %q", flag.DefValue, "0")
	}
}

// TestAutoCommandHelp tests that help text is comprehensive
func TestAutoCommandHelp(t *testing.T) {
	// Verify Long description contains important information
	long := autoCmd.Long

	expectedPhrases := []string{
		"specification",
		"plan",
		"approval",
		"profile",
		"Exit Codes",
		"Scope Filtering",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(long, phrase) {
			t.Errorf("Long description missing phrase: %q", phrase)
		}
	}
}

// TestAutoExitCodes tests that exit codes are documented
func TestAutoExitCodes(t *testing.T) {
	long := autoCmd.Long

	// Verify exit codes are documented
	exitCodes := []string{
		"0  Success",
		"1  General error",
		"2  Usage error",
		"3  Policy violation",
		"4  Drift detected",
		"5  Auth error",
		"6  Network error",
	}

	for _, code := range exitCodes {
		if !strings.Contains(long, code) {
			t.Errorf("Long description missing exit code: %q", code)
		}
	}
}
