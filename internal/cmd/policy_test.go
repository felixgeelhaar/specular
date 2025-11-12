package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestPolicySubcommands tests that all policy subcommands are registered
func TestPolicySubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"new":   false,
		"apply": false,
	}

	for _, cmd := range policyCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand '%s' not found in policy command", name)
		}
	}
}

// TestPolicyNewFlags tests that policy new has correct flags
func TestPolicyNewFlags(t *testing.T) {
	// Find new subcommand
	var newCmd *cobra.Command
	for _, cmd := range policyCmd.Commands() {
		if cmd.Name() == "new" {
			newCmd = cmd
			break
		}
	}

	if newCmd == nil {
		t.Fatal("new subcommand not found")
	}

	// Check flags
	if newCmd.Flags().Lookup("output") == nil {
		t.Error("flag 'output' not found on policy new command")
	}
	if newCmd.Flags().Lookup("strict") == nil {
		t.Error("flag 'strict' not found on policy new command")
	}
	if newCmd.Flags().Lookup("force") == nil {
		t.Error("flag 'force' not found on policy new command")
	}
}

// TestPolicyApplyFlags tests that policy apply has correct flags
func TestPolicyApplyFlags(t *testing.T) {
	// Find apply subcommand
	var applyCmd *cobra.Command
	for _, cmd := range policyCmd.Commands() {
		if cmd.Name() == "apply" {
			applyCmd = cmd
			break
		}
	}

	if applyCmd == nil {
		t.Fatal("apply subcommand not found")
	}

	// Check flags
	if applyCmd.Flags().Lookup("file") == nil {
		t.Error("flag 'file' not found on policy apply command")
	}
	if applyCmd.Flags().Lookup("target") == nil {
		t.Error("flag 'target' not found on policy apply command")
	}
}

// TestPolicyNewCommand tests the policy new command configuration
func TestPolicyNewCommand(t *testing.T) {
	// Find new subcommand
	var newCmd *cobra.Command
	for _, cmd := range policyCmd.Commands() {
		if cmd.Name() == "new" {
			newCmd = cmd
			break
		}
	}

	if newCmd == nil {
		t.Fatal("new subcommand not found")
	}

	// Check command configuration
	if newCmd.Use != "new" {
		t.Errorf("new Use = %q, want %q", newCmd.Use, "new")
	}

	if newCmd.Short == "" {
		t.Error("new Short description is empty")
	}
}

// TestPolicyApplyCommand tests the policy apply command configuration
func TestPolicyApplyCommand(t *testing.T) {
	// Find apply subcommand
	var applyCmd *cobra.Command
	for _, cmd := range policyCmd.Commands() {
		if cmd.Name() == "apply" {
			applyCmd = cmd
			break
		}
	}

	if applyCmd == nil {
		t.Fatal("apply subcommand not found")
	}

	// Check command configuration
	if applyCmd.Use != "apply" {
		t.Errorf("apply Use = %q, want %q", applyCmd.Use, "apply")
	}

	if applyCmd.Short == "" {
		t.Error("apply Short description is empty")
	}
}

// TestPolicyCommand tests the policy command configuration
func TestPolicyCommand(t *testing.T) {
	// Check command configuration
	if policyCmd.Use != "policy" {
		t.Errorf("policy Use = %q, want %q", policyCmd.Use, "policy")
	}

	if policyCmd.Short == "" {
		t.Error("policy Short description is empty")
	}

	// Verify it has subcommands
	if len(policyCmd.Commands()) == 0 {
		t.Error("policy command should have subcommands")
	}
}
