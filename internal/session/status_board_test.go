package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildStatusBoard(t *testing.T) {
	t.Parallel()
	list := []Record{
		{ID: "a", Status: StatusWorking},
		{ID: "b", Status: StatusQueued},
		{ID: "c", Status: StatusCompleted},
		{ID: "d", Status: StatusFailed},
		{ID: "e", Status: StatusStopped},
		{ID: "f", Status: StatusWaiting},
		{ID: "g", Status: "mystery"},
	}
	board := BuildStatusBoard(list)
	if board.Summary.Total != 7 {
		t.Fatalf("total=%d", board.Summary.Total)
	}
	if board.Summary.Working != 2 { // working + waiting
		t.Fatalf("working=%d", board.Summary.Working)
	}
	if board.Summary.Queued != 1 || board.Summary.Completed != 1 || board.Summary.Failed != 1 || board.Summary.Stopped != 1 {
		t.Fatalf("%+v", board.Summary)
	}
	if board.Summary.Other != 1 {
		t.Fatalf("other=%d", board.Summary.Other)
	}
	if len(board.Sessions) != 7 {
		t.Fatalf("sessions=%d", len(board.Sessions))
	}
	// Defensive copy: mutating input shouldn't change board snapshot length contract.
	list[0].Status = StatusFailed
	if board.Sessions[0].Status != StatusWorking {
		t.Fatal("expected board to own a copy of session records")
	}
}

func TestEvidenceFlagsAndBoard(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "auth.provenance.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	flags := EvidenceFlags(dir, "auth")
	if !flags.Attested || !flags.App {
		t.Fatalf("%+v", flags)
	}
	if EvidenceFlags(dir, "missing").Attested {
		t.Fatal("expected missing")
	}
	board := BuildStatusBoardWithEvidence([]Record{{ID: "auth", Status: StatusCompleted}}, dir)
	ev := board.Evidence["auth"]
	if !ev.Attested || !ev.App {
		t.Fatalf("%+v", board.Evidence)
	}
	if YesDash(true) != "yes" || YesDash(false) != "-" {
		t.Fatal(YesDash(true), YesDash(false))
	}
	if DashOr("") != "-" || DashOr("abc") != "abc" {
		t.Fatal(DashOr(""), DashOr("abc"))
	}
}

func TestWorktreeHEADShort(t *testing.T) {
	t.Parallel()
	if WorktreeHEADShort("") != "" || WorktreeHEADShort("/no/such/path") != "" {
		t.Fatal("expected empty for missing worktree")
	}
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	run("init")
	run("config", "user.email", "t@example.com")
	run("config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "init")
	sha := WorktreeHEADShort(repo)
	if sha == "" || len(sha) < 4 {
		t.Fatalf("sha=%q", sha)
	}
	board := BuildStatusBoardWithEvidence([]Record{{
		ID: "demo", Status: StatusCompleted, WorktreePath: repo,
	}}, t.TempDir())
	if board.Evidence["demo"].Commit != sha {
		t.Fatalf("evidence commit=%q want %q", board.Evidence["demo"].Commit, sha)
	}
}
