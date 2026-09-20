package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSessionSubcommands(t *testing.T) {
	required := map[string]bool{
		"start":     false,
		"batch":     false,
		"list":      false,
		"show":      false,
		"stop":      false,
		"logs":      false,
		"fork":      false,
		"harnesses": false,
		"status":    false,
		"open":      false,
		"wait":      false,
		"restart":   false,
		"rm":        false,
		"prune":     false,
		"diff":      false,
		"exec":      false,
		"commit":    false,
		"sync":      false,
		"attest":    false,
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
	for _, name := range []string{"name", "harness", "profile", "no-worktree", "foreground", "json", "manifest"} {
		if startCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session start", name)
		}
	}
}

func TestSessionLifecycleFlags(t *testing.T) {
	found := map[string]*cobra.Command{}
	for _, cmd := range sessionCmd.Commands() {
		found[cmd.Name()] = cmd
	}
	for _, name := range []string{"status", "open", "wait", "restart", "rm", "prune", "diff", "batch", "exec", "commit", "sync", "attest"} {
		if found[name] == nil {
			t.Fatalf("%s subcommand not found", name)
		}
	}
	for _, name := range []string{"watch", "interval", "json"} {
		if found["status"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session status", name)
		}
	}
	for _, name := range []string{"shell", "editor"} {
		if found["open"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session open", name)
		}
	}
	for _, name := range []string{"timeout", "interval", "any", "json"} {
		if found["wait"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session wait", name)
		}
	}
	if found["wait"].Flags().Lookup("attest") == nil {
		t.Errorf("flag %q not found on session wait", "attest")
	}
	if found["wait"].Flags().Lookup("gate") == nil {
		t.Errorf("flag %q not found on session wait", "gate")
	}
	for _, name := range []string{"bundle", "bundle-out", "policy"} {
		if found["wait"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session wait", name)
		}
	}
	for _, name := range []string{"harness", "goal", "profile", "force", "foreground", "json"} {
		if found["restart"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session restart", name)
		}
	}
	if found["restart"].Flags().Lookup("governed") == nil {
		t.Errorf("flag %q not found on session restart", "governed")
	}
	if found["start"].Flags().Lookup("governed") == nil {
		t.Errorf("flag %q not found on session start", "governed")
	}
	if found["batch"].Flags().Lookup("governed") == nil {
		t.Errorf("flag %q not found on session batch", "governed")
	}
	for _, name := range []string{"force", "keep-worktree", "delete-branch", "json"} {
		if found["rm"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session rm", name)
		}
	}
	for _, name := range []string{"older-than", "keep-worktree", "delete-branch", "json"} {
		if found["prune"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session prune", name)
		}
	}
	for _, name := range []string{"base", "against", "stat", "name-only", "patch", "json"} {
		if found["diff"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session diff", name)
		}
	}
	for _, name := range []string{"harness", "profile", "no-worktree", "json"} {
		if found["batch"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session batch", name)
		}
	}
	for _, name := range []string{"log", "json"} {
		if found["exec"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session exec", name)
		}
	}
	for _, name := range []string{"message", "all", "allow-empty", "force", "json"} {
		if found["commit"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session commit", name)
		}
	}
	for _, name := range []string{"onto", "merge", "autostash", "force", "json"} {
		if found["sync"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session sync", name)
		}
	}
	for _, name := range []string{"output", "force", "json"} {
		if found["attest"].Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found on session attest", name)
		}
	}
}

func TestSessionCommand(t *testing.T) {
	if sessionCmd.Use != "session" {
		t.Errorf("session Use = %q, want session", sessionCmd.Use)
	}
	if len(sessionCmd.Commands()) < 10 {
		t.Errorf("expected at least 10 subcommands, got %d", len(sessionCmd.Commands()))
	}
}
