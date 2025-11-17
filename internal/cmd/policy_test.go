package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestPolicySubcommands tests that all policy subcommands are registered
func TestPolicySubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"init":     false,
		"validate": false,
		"approve":  false,
		"list":     false,
		"diff":     false,
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

// TestPolicyInitFlags tests that policy init has correct flags
func TestPolicyInitFlags(t *testing.T) {
	// Find init subcommand
	var initCmd *cobra.Command
	for _, cmd := range policyCmd.Commands() {
		if cmd.Name() == "init" {
			initCmd = cmd
			break
		}
	}

	if initCmd == nil {
		t.Fatal("init subcommand not found")
	}

	// Check flags
	if initCmd.Flags().Lookup("template") == nil {
		t.Error("flag 'template' not found on policy init command")
	}
}

// TestPolicyValidateFlags tests that policy validate has correct flags
func TestPolicyValidateFlags(t *testing.T) {
	// Find validate subcommand
	var validateCmd *cobra.Command
	for _, cmd := range policyCmd.Commands() {
		if cmd.Name() == "validate" {
			validateCmd = cmd
			break
		}
	}

	if validateCmd == nil {
		t.Fatal("validate subcommand not found")
	}

	// Check flags
	if validateCmd.Flags().Lookup("strict") == nil {
		t.Error("flag 'strict' not found on policy validate command")
	}
	if validateCmd.Flags().Lookup("json") == nil {
		t.Error("flag 'json' not found on policy validate command")
	}
}

// TestPolicyInitCommand tests the policy init command configuration
func TestPolicyInitCommand(t *testing.T) {
	// Find init subcommand
	var initCmd *cobra.Command
	for _, cmd := range policyCmd.Commands() {
		if cmd.Name() == "init" {
			initCmd = cmd
			break
		}
	}

	if initCmd == nil {
		t.Fatal("init subcommand not found")
	}

	// Check command configuration
	if initCmd.Use != "init" {
		t.Errorf("init Use = %q, want %q", initCmd.Use, "init")
	}

	if initCmd.Short == "" {
		t.Error("init Short description is empty")
	}
}

// TestPolicyValidateCommand tests the policy validate command configuration
func TestPolicyValidateCommand(t *testing.T) {
	// Find validate subcommand
	var validateCmd *cobra.Command
	for _, cmd := range policyCmd.Commands() {
		if cmd.Name() == "validate" {
			validateCmd = cmd
			break
		}
	}

	if validateCmd == nil {
		t.Fatal("validate subcommand not found")
	}

	// Check command configuration
	if validateCmd.Use != "validate" {
		t.Errorf("validate Use = %q, want %q", validateCmd.Use, "validate")
	}

	if validateCmd.Short == "" {
		t.Error("validate Short description is empty")
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
