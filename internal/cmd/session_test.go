package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSessionSubcommands(t *testing.T) {
	required := map[string]bool{
		"start": false,
		"list":  false,
		"show":  false,
		"stop":  false,
	}

	for _, cmd := range sessionCmd.Commands() {
		if _, exists := required[cmd.Name()]; exists {
			required[cmd.Name()] = true
		}
	}

	for name, found := range required {
		if !found {
			t.Errorf("subcommand %q not found on session command", name)
		}
	}
}

func TestSessionStartFlags(t *testing.T) {
	var startCmd *cobra.Command
	for _, cmd := range sessionCmd.Commands() {
		if cmd.Name() == "start" {
			startCmd = cmd
			break
		}
	}
	if startCmd == nil {
		t.Fatal("start subcommand not found")
	}
	for _, name := range []string{"name", "harness", "profile", "no-worktree", "foreground", "json"} {
		if startCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session start", name)
		}
	}
}

func TestSessionShowFlags(t *testing.T) {
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
	if showCmd.Flags().Lookup("verbose") == nil {
		t.Error("flag 'verbose' not found on session show")
	}
	if showCmd.Flags().Lookup("json") == nil {
		t.Error("flag 'json' not found on session show")
	}
}

func TestSessionCommand(t *testing.T) {
	if sessionCmd.Use != "session" {
		t.Errorf("session Use = %q, want session", sessionCmd.Use)
	}
	if sessionCmd.Short == "" {
		t.Error("session Short description is empty")
	}
	if len(sessionCmd.Commands()) < 4 {
		t.Errorf("expected at least 4 subcommands, got %d", len(sessionCmd.Commands()))
	}
}
