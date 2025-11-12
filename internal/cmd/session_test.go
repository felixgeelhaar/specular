package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestSessionSubcommands tests that all session subcommands are registered
func TestSessionSubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"list": false,
		"show": false,
	}

	for _, cmd := range sessionCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand '%s' not found in session command", name)
		}
	}
}

// TestSessionListCommand tests the session list command configuration
func TestSessionListCommand(t *testing.T) {
	// Find list subcommand
	var listCmd *cobra.Command
	for _, cmd := range sessionCmd.Commands() {
		if cmd.Name() == "list" {
			listCmd = cmd
			break
		}
	}

	if listCmd == nil {
		t.Fatal("list subcommand not found")
	}

	// Check command configuration
	if listCmd.Use != "list" {
		t.Errorf("list Use = %q, want %q", listCmd.Use, "list")
	}

	if listCmd.Short == "" {
		t.Error("list Short description is empty")
	}
}

// TestSessionShowCommand tests the session show command configuration
func TestSessionShowCommand(t *testing.T) {
	// Find show subcommand
	var showCmd *cobra.Command
	for _, cmd := range sessionCmd.Commands() {
		if cmd.Name() == "show" {
			showCmd = cmd
			break
		}
	}

	if showCmd == nil {
		t.Fatal("show subcommand not found")
	}

	// Check command configuration
	if showCmd.Use != "show <session-id>" {
		t.Errorf("show Use = %q, want %q", showCmd.Use, "show <session-id>")
	}

	if showCmd.Short == "" {
		t.Error("show Short description is empty")
	}

	// Check Args is set (requires exactly 1 arg)
	if showCmd.Args == nil {
		t.Error("show command should have Args validator")
	}
}

// TestSessionShowFlags tests that session show has correct flags
func TestSessionShowFlags(t *testing.T) {
	// Find show subcommand
	var showCmd *cobra.Command
	for _, cmd := range sessionCmd.Commands() {
		if cmd.Name() == "show" {
			showCmd = cmd
			break
		}
	}

	if showCmd == nil {
		t.Fatal("show subcommand not found")
	}

	// Check flags
	if showCmd.Flags().Lookup("verbose") == nil {
		t.Error("flag 'verbose' not found on session show command")
	}
	if showCmd.Flags().Lookup("json") == nil {
		t.Error("flag 'json' not found on session show command")
	}
}

// TestSessionCommand tests the session command configuration
func TestSessionCommand(t *testing.T) {
	// Check command configuration
	if sessionCmd.Use != "session" {
		t.Errorf("session Use = %q, want %q", sessionCmd.Use, "session")
	}

	if sessionCmd.Short == "" {
		t.Error("session Short description is empty")
	}

	// Verify it has subcommands
	if len(sessionCmd.Commands()) == 0 {
		t.Error("session command should have subcommands")
	}
}
