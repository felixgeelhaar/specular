package cmd

import (
	"testing"
)

// TestProviderCommand tests the provider command configuration
func TestProviderCommand(t *testing.T) {
	if providerCmd == nil {
		t.Fatal("providerCmd is nil")
	}

	if providerCmd.Use != "provider" {
		t.Errorf("providerCmd.Use = %q, want %q", providerCmd.Use, "provider")
	}

	if providerCmd.Short == "" {
		t.Error("providerCmd.Short is empty")
	}
}

// TestProviderSubcommands tests that all provider subcommands are registered
func TestProviderSubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"list":   false,
		"doctor": false,
		"init":   false,
		"add":    false,
		"remove": false,
	}

	for _, cmd := range providerCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand '%s' not found in provider command", name)
		}
	}
}

// TestProviderListCommand tests the provider list subcommand
func TestProviderListCommand(t *testing.T) {
	found := false
	for _, cmd := range providerCmd.Commands() {
		if cmd.Name() == "list" {
			found = true
			if cmd.Short == "" {
				t.Error("list subcommand has empty Short description")
			}
			break
		}
	}

	if !found {
		t.Error("list subcommand not found")
	}
}

// TestProviderDoctorCommand tests the provider doctor subcommand
func TestProviderDoctorCommand(t *testing.T) {
	found := false
	for _, cmd := range providerCmd.Commands() {
		if cmd.Name() == "doctor" {
			found = true
			if cmd.Short == "" {
				t.Error("doctor subcommand has empty Short description")
			}
			break
		}
	}

	if !found {
		t.Error("doctor subcommand not found")
	}
}

// TestProviderInitCommand tests the provider init subcommand
func TestProviderInitCommand(t *testing.T) {
	found := false
	for _, cmd := range providerCmd.Commands() {
		if cmd.Name() == "init" {
			found = true
			if cmd.Short == "" {
				t.Error("init subcommand has empty Short description")
			}
			break
		}
	}

	if !found {
		t.Error("init subcommand not found")
	}
}

// TestProviderAddCommand tests the provider add subcommand
func TestProviderAddCommand(t *testing.T) {
	found := false
	for _, cmd := range providerCmd.Commands() {
		if cmd.Name() == "add" {
			found = true
			if cmd.Short == "" {
				t.Error("add subcommand has empty Short description")
			}
			break
		}
	}

	if !found {
		t.Error("add subcommand not found")
	}
}

// TestProviderRemoveCommand tests the provider remove subcommand
func TestProviderRemoveCommand(t *testing.T) {
	found := false
	for _, cmd := range providerCmd.Commands() {
		if cmd.Name() == "remove" {
			found = true
			if cmd.Short == "" {
				t.Error("remove subcommand has empty Short description")
			}
			break
		}
	}

	if !found {
		t.Error("remove subcommand not found")
	}
}

// TestProviderListFlags tests flags on provider list command
func TestProviderListFlags(t *testing.T) {
	var listCmd interface{}
	for _, cmd := range providerCmd.Commands() {
		if cmd.Name() == "list" {
			listCmd = cmd
			break
		}
	}

	if listCmd == nil {
		t.Fatal("list subcommand not found")
	}
}
