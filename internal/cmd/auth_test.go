package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestAuthSubcommands tests that all auth subcommands are registered
func TestAuthSubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"login":  false,
		"logout": false,
	}

	for _, cmd := range authCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand '%s' not found in auth command", name)
		}
	}
}

// TestAuthLoginFlags tests that auth login has correct flags
func TestAuthLoginFlags(t *testing.T) {
	// Find login subcommand
	var loginCmd *cobra.Command
	for _, cmd := range authCmd.Commands() {
		if cmd.Name() == "login" {
			loginCmd = cmd
			break
		}
	}

	if loginCmd == nil {
		t.Fatal("login subcommand not found")
	}

	// Check flags
	if loginCmd.Flags().Lookup("email") == nil {
		t.Error("flag 'email' not found on auth login command")
	}
	if loginCmd.Flags().Lookup("password") == nil {
		t.Error("flag 'password' not found on auth login command")
	}
}

// TestAuthLoginCommand tests the auth login command configuration
func TestAuthLoginCommand(t *testing.T) {
	// Find login subcommand
	var loginCmd *cobra.Command
	for _, cmd := range authCmd.Commands() {
		if cmd.Name() == "login" {
			loginCmd = cmd
			break
		}
	}

	if loginCmd == nil {
		t.Fatal("login subcommand not found")
	}

	// Check command configuration
	if loginCmd.Use != "login" {
		t.Errorf("login Use = %q, want %q", loginCmd.Use, "login")
	}

	if loginCmd.Short == "" {
		t.Error("login Short description is empty")
	}
}

// TestAuthLogoutCommand tests the auth logout command configuration
func TestAuthLogoutCommand(t *testing.T) {
	// Find logout subcommand
	var logoutCmd *cobra.Command
	for _, cmd := range authCmd.Commands() {
		if cmd.Name() == "logout" {
			logoutCmd = cmd
			break
		}
	}

	if logoutCmd == nil {
		t.Fatal("logout subcommand not found")
	}

	// Check command configuration
	if logoutCmd.Use != "logout" {
		t.Errorf("logout Use = %q, want %q", logoutCmd.Use, "logout")
	}

	if logoutCmd.Short == "" {
		t.Error("logout Short description is empty")
	}
}

// TestAuthCommand tests the auth command configuration
func TestAuthCommand(t *testing.T) {
	// Check command configuration
	if authCmd.Use != "auth" {
		t.Errorf("auth Use = %q, want %q", authCmd.Use, "auth")
	}

	if authCmd.Short == "" {
		t.Error("auth Short description is empty")
	}

	// Verify it has subcommands
	if len(authCmd.Commands()) == 0 {
		t.Error("auth command should have subcommands")
	}
}
