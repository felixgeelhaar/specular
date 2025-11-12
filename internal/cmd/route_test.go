package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestRouteSubcommands tests that all route subcommands are registered
func TestRouteSubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"list":     false,
		"override": false,
		"explain":  false,
	}

	for _, cmd := range routeCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand '%s' not found in route command", name)
		}
	}
}

// TestRouteListFlags tests that route list has correct flags
func TestRouteListFlags(t *testing.T) {
	// Find list subcommand
	var listCmd *cobra.Command
	for _, cmd := range routeCmd.Commands() {
		if cmd.Name() == "list" {
			listCmd = cmd
			break
		}
	}

	if listCmd == nil {
		t.Fatal("list subcommand not found")
	}

	// Check flags
	if listCmd.Flags().Lookup("available") == nil {
		t.Error("flag 'available' not found on route list command")
	}
	if listCmd.Flags().Lookup("provider") == nil {
		t.Error("flag 'provider' not found on route list command")
	}
}

// TestRouteOverrideArgs tests that route override requires exactly one argument
func TestRouteOverrideArgs(t *testing.T) {
	// Find override subcommand
	var overrideCmd *cobra.Command
	for _, cmd := range routeCmd.Commands() {
		if cmd.Name() == "override" {
			overrideCmd = cmd
			break
		}
	}

	if overrideCmd == nil {
		t.Fatal("override subcommand not found")
	}

	// Check Args is set (requires exactly 1 arg)
	if overrideCmd.Args == nil {
		t.Error("override command should have Args validator")
	}
}

// TestRouteExplainArgs tests that route explain requires exactly one argument
func TestRouteExplainArgs(t *testing.T) {
	// Find explain subcommand
	var explainCmd *cobra.Command
	for _, cmd := range routeCmd.Commands() {
		if cmd.Name() == "explain" {
			explainCmd = cmd
			break
		}
	}

	if explainCmd == nil {
		t.Fatal("explain subcommand not found")
	}

	// Check Args is set (requires exactly 1 arg)
	if explainCmd.Args == nil {
		t.Error("explain command should have Args validator")
	}
}

// TestRouteListCommand tests the route list command configuration
func TestRouteListCommand(t *testing.T) {
	// Find list subcommand
	var listCmd *cobra.Command
	for _, cmd := range routeCmd.Commands() {
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

// TestRouteOverrideCommand tests the route override command configuration
func TestRouteOverrideCommand(t *testing.T) {
	// Find override subcommand
	var overrideCmd *cobra.Command
	for _, cmd := range routeCmd.Commands() {
		if cmd.Name() == "override" {
			overrideCmd = cmd
			break
		}
	}

	if overrideCmd == nil {
		t.Fatal("override subcommand not found")
	}

	// Check command configuration
	if overrideCmd.Use != "override <provider>" {
		t.Errorf("override Use = %q, want %q", overrideCmd.Use, "override <provider>")
	}

	if overrideCmd.Short == "" {
		t.Error("override Short description is empty")
	}
}

// TestRouteExplainCommand tests the route explain command configuration
func TestRouteExplainCommand(t *testing.T) {
	// Find explain subcommand
	var explainCmd *cobra.Command
	for _, cmd := range routeCmd.Commands() {
		if cmd.Name() == "explain" {
			explainCmd = cmd
			break
		}
	}

	if explainCmd == nil {
		t.Fatal("explain subcommand not found")
	}

	// Check command configuration
	if explainCmd.Use != "explain <task-type>" {
		t.Errorf("explain Use = %q, want %q", explainCmd.Use, "explain <task-type>")
	}

	if explainCmd.Short == "" {
		t.Error("explain Short description is empty")
	}
}
