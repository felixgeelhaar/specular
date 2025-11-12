package cmd

import (
	"testing"

	"github.com/spf13/cobra"
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
