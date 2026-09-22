package cmd

import (
	"strings"
	"testing"
)

func TestChangeExplainCommandRegistered(t *testing.T) {
	// Not parallel: rootCmd.Find mutates shared cobra command state.
	cmd, _, err := rootCmd.Find([]string{"explain"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil || cmd.Name() != "explain" {
		t.Fatalf("explain not registered: %v", cmd)
	}
	if !strings.Contains(cmd.Short, "ALLOW") && !strings.Contains(cmd.Short, "DENY") {
		t.Fatalf("short=%q", cmd.Short)
	}
}

func TestEvidenceCommandsRegistered(t *testing.T) {
	// Not parallel: rootCmd.Find mutates shared cobra command state.
	for _, path := range [][]string{{"evidence"}, {"evidence", "list"}, {"evidence", "show"}} {
		cmd, _, err := rootCmd.Find(path)
		if err != nil {
			t.Fatalf("%v: %v", path, err)
		}
		if cmd == nil {
			t.Fatalf("missing %v", path)
		}
	}
}

func TestEvidenceListFilterFlags(t *testing.T) {
	// Not parallel: rootCmd.Find mutates shared cobra command state.
	cmd, _, err := rootCmd.Find([]string{"evidence", "list"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"verdict", "since", "path", "risk", "limit", "json"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s", name)
		}
	}
}
