package session

import (
	"testing"
)

func TestResolveLaunchClaude(t *testing.T) {
	t.Parallel()
	rec := &Record{
		ID:           "s1",
		Goal:         "fix the bug",
		Harness:      "claude-code",
		WorktreePath: "/repo/.specular/worktrees/s1",
	}
	plan, err := ResolveLaunch(StartOptions{}, rec)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Binary != "claude" {
		t.Fatalf("binary=%s", plan.Binary)
	}
	if plan.Kind != "native" {
		t.Fatalf("kind=%s", plan.Kind)
	}
	if plan.WorkDir != rec.WorktreePath {
		t.Fatalf("workdir=%s", plan.WorkDir)
	}
	joined := joinArgs(plan.Args)
	if !containsAll(plan.Args, "--print", "--dangerously-skip-permissions", "fix the bug") {
		t.Fatalf("args=%v", plan.Args)
	}
	_ = joined
}

func TestResolveLaunchCodexGemini(t *testing.T) {
	t.Parallel()
	codex, err := ResolveLaunch(StartOptions{}, &Record{ID: "c", Goal: "g", Harness: "codex", WorktreePath: "/w"})
	if err != nil {
		t.Fatal(err)
	}
	if codex.Binary != "codex" || codex.Args[0] != "exec" {
		t.Fatalf("%+v", codex)
	}

	gem, err := ResolveLaunch(StartOptions{}, &Record{ID: "g", Goal: "goal", Harness: "gemini", WorktreePath: "/w"})
	if err != nil {
		t.Fatal(err)
	}
	if gem.Binary != "gemini" || gem.Args[0] != "--prompt" {
		t.Fatalf("%+v", gem)
	}
}

func TestResolveLaunchUnknown(t *testing.T) {
	t.Parallel()
	_, err := ResolveLaunch(StartOptions{}, &Record{ID: "x", Goal: "g", Harness: "devin"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeAndNative(t *testing.T) {
	t.Parallel()
	if normalizeHarness("") != "specular-auto" {
		t.Fatal(normalizeHarness(""))
	}
	if !IsNativeHarness("Claude-Code") {
		t.Fatal("expected native")
	}
	if IsNativeHarness("specular-auto") {
		t.Fatal("expected not native")
	}
}

func joinArgs(args []string) string {
	out := ""
	for _, a := range args {
		out += a + " "
	}
	return out
}

func containsAll(args []string, want ...string) bool {
	set := map[string]bool{}
	for _, a := range args {
		set[a] = true
	}
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}
