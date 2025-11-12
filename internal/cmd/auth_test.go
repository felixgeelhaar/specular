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
		"whoami": false,
		"token":  false,
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
	if loginCmd.Flags().Lookup("user") == nil {
		t.Error("flag 'user' not found on auth login command")
	}
	if loginCmd.Flags().Lookup("token") == nil {
		t.Error("flag 'token' not found on auth login command")
	}
	if loginCmd.Flags().Lookup("registry") == nil {
		t.Error("flag 'registry' not found on auth login command")
	}
}

// TestAuthTokenFlags tests that auth token has correct flags
func TestAuthTokenFlags(t *testing.T) {
	// Find token subcommand
	var tokenCmd *cobra.Command
	for _, cmd := range authCmd.Commands() {
		if cmd.Name() == "token" {
			tokenCmd = cmd
			break
		}
	}

	if tokenCmd == nil {
		t.Fatal("token subcommand not found")
	}

	// Check flags
	if tokenCmd.Flags().Lookup("refresh") == nil {
		t.Error("flag 'refresh' not found on auth token command")
	}
	if tokenCmd.Flags().Lookup("show") == nil {
		t.Error("flag 'show' not found on auth token command")
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

// TestAuthWhoamiCommand tests the auth whoami command configuration
func TestAuthWhoamiCommand(t *testing.T) {
	// Find whoami subcommand
	var whoamiCmd *cobra.Command
	for _, cmd := range authCmd.Commands() {
		if cmd.Name() == "whoami" {
			whoamiCmd = cmd
			break
		}
	}

	if whoamiCmd == nil {
		t.Fatal("whoami subcommand not found")
	}

	// Check command configuration
	if whoamiCmd.Use != "whoami" {
		t.Errorf("whoami Use = %q, want %q", whoamiCmd.Use, "whoami")
	}

	if whoamiCmd.Short == "" {
		t.Error("whoami Short description is empty")
	}
}

// TestAuthTokenCommand tests the auth token command configuration
func TestAuthTokenCommand(t *testing.T) {
	// Find token subcommand
	var tokenCmd *cobra.Command
	for _, cmd := range authCmd.Commands() {
		if cmd.Name() == "token" {
			tokenCmd = cmd
			break
		}
	}

	if tokenCmd == nil {
		t.Fatal("token subcommand not found")
	}

	// Check command configuration
	if tokenCmd.Use != "token" {
		t.Errorf("token Use = %q, want %q", tokenCmd.Use, "token")
	}

	if tokenCmd.Short == "" {
		t.Error("token Short description is empty")
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

// TestAuthCredentialsStruct tests the AuthCredentials struct definition
func TestAuthCredentialsStruct(t *testing.T) {
	// Create a sample credentials to verify struct works
	creds := AuthCredentials{
		User:      "alice@example.com",
		Token:     "token_abc123",
		Registry:  "https://registry.example.com",
	}

	// Verify fields are accessible
	if creds.User != "alice@example.com" {
		t.Errorf("User = %q, want %q", creds.User, "alice@example.com")
	}
	if creds.Token != "token_abc123" {
		t.Errorf("Token = %q, want %q", creds.Token, "token_abc123")
	}
	if creds.Registry != "https://registry.example.com" {
		t.Errorf("Registry = %q, want %q", creds.Registry, "https://registry.example.com")
	}
}
