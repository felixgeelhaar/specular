package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestDebugSubcommands tests that all debug subcommands are registered
func TestDebugSubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"status":  false,
		"context": false,
		"doctor":  false,
		"logs":    false,
		"explain": false,
	}

	for _, cmd := range debugCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand '%s' not found in debug command", name)
		}
	}
}

// TestDebugStatusCommand tests the debug status command configuration
func TestDebugStatusCommand(t *testing.T) {
	// Find status subcommand
	var statusCmd *cobra.Command
	for _, cmd := range debugCmd.Commands() {
		if cmd.Name() == "status" {
			statusCmd = cmd
			break
		}
	}

	if statusCmd == nil {
		t.Fatal("status subcommand not found")
	}

	// Check command configuration
	if statusCmd.Use != "status" {
		t.Errorf("status Use = %q, want %q", statusCmd.Use, "status")
	}

	if statusCmd.Short == "" {
		t.Error("status Short description is empty")
	}
}

// TestDebugContextCommand tests the debug context command configuration
func TestDebugContextCommand(t *testing.T) {
	// Find context subcommand
	var contextCmd *cobra.Command
	for _, cmd := range debugCmd.Commands() {
		if cmd.Name() == "context" {
			contextCmd = cmd
			break
		}
	}

	if contextCmd == nil {
		t.Fatal("context subcommand not found")
	}

	// Check command configuration
	if contextCmd.Use != "context" {
		t.Errorf("context Use = %q, want %q", contextCmd.Use, "context")
	}

	if contextCmd.Short == "" {
		t.Error("context Short description is empty")
	}
}

// TestDebugDoctorCommand tests the debug doctor command configuration
func TestDebugDoctorCommand(t *testing.T) {
	// Find doctor subcommand
	var doctorCmd *cobra.Command
	for _, cmd := range debugCmd.Commands() {
		if cmd.Name() == "doctor" {
			doctorCmd = cmd
			break
		}
	}

	if doctorCmd == nil {
		t.Fatal("doctor subcommand not found")
	}

	// Check command configuration
	if doctorCmd.Use != "doctor" {
		t.Errorf("doctor Use = %q, want %q", doctorCmd.Use, "doctor")
	}

	if doctorCmd.Short == "" {
		t.Error("doctor Short description is empty")
	}
}

// TestDebugLogsCommand tests the debug logs command configuration
func TestDebugLogsCommand(t *testing.T) {
	// Find logs subcommand
	var logsCmd *cobra.Command
	for _, cmd := range debugCmd.Commands() {
		if cmd.Name() == "logs" {
			logsCmd = cmd
			break
		}
	}

	if logsCmd == nil {
		t.Fatal("logs subcommand not found")
	}

	// Check command configuration
	if logsCmd.Use != "logs" {
		t.Errorf("logs Use = %q, want %q", logsCmd.Use, "logs")
	}

	if logsCmd.Short == "" {
		t.Error("logs Short description is empty")
	}
}

// TestDebugExplainCommand tests the debug explain command configuration
func TestDebugExplainCommand(t *testing.T) {
	// Find explain subcommand
	var explainCmd *cobra.Command
	for _, cmd := range debugCmd.Commands() {
		if cmd.Name() == "explain" {
			explainCmd = cmd
			break
		}
	}

	if explainCmd == nil {
		t.Fatal("explain subcommand not found")
	}

	// Check command configuration
	if explainCmd.Use != "explain <checkpoint-id>" {
		t.Errorf("explain Use = %q, want %q", explainCmd.Use, "explain <checkpoint-id>")
	}

	if explainCmd.Short == "" {
		t.Error("explain Short description is empty")
	}

	// Check Args is set (requires exactly 1 arg)
	if explainCmd.Args == nil {
		t.Error("explain command should have Args validator")
	}
}

// TestDebugCommand tests the debug command configuration
func TestDebugCommand(t *testing.T) {
	// Check command configuration
	if debugCmd.Use != "debug" {
		t.Errorf("debug Use = %q, want %q", debugCmd.Use, "debug")
	}

	if debugCmd.Short == "" {
		t.Error("debug Short description is empty")
	}

	// Verify it has subcommands
	if len(debugCmd.Commands()) == 0 {
		t.Error("debug command should have subcommands")
	}
}
