package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSessionSubcommands(t *testing.T) {
	required := map[string]bool{
		"start":     false,
		"list":      false,
		"show":      false,
		"stop":      false,
		"logs":      false,
		"fork":      false,
		"harnesses": false,
		"status":    false,
		"open":      false,
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

func TestSessionStatusAndOpenFlags(t *testing.T) {
	var statusCmd, openCmd *cobra.Command
	for _, cmd := range sessionCmd.Commands() {
		switch cmd.Name() {
		case "status":
			statusCmd = cmd
		case "open":
			openCmd = cmd
		}
	}
	if statusCmd == nil {
		t.Fatal("status subcommand not found")
	}
	if openCmd == nil {
		t.Fatal("open subcommand not found")
	}
	for _, name := range []string{"watch", "interval", "json"} {
		if statusCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session status", name)
		}
	}
	if openCmd.Flags().Lookup("shell") == nil {
		t.Error("flag \"shell\" not found on session open")
	}
}

func TestSessionCommand(t *testing.T) {
	if sessionCmd.Use != "session" {
		t.Errorf("session Use = %q, want session", sessionCmd.Use)
	}
	if len(sessionCmd.Commands()) < 7 {
		t.Errorf("expected at least 7 subcommands, got %d", len(sessionCmd.Commands()))
	}
}
