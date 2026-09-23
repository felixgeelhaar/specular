package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/evidence"
	"github.com/felixgeelhaar/specular/internal/gate"
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
	board := BuildStatusBoardWithEvidence([]Record{{ID: "auth", Status: StatusCompleted}}, dir, "")
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
	}}, t.TempDir(), "")
	if board.Evidence["demo"].Commit != sha {
		t.Fatalf("evidence commit=%q want %q", board.Evidence["demo"].Commit, sha)
	}
}

func TestNewestGateBySessionAndBoard(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)

	older := mustWriteEvidence(t, root, &evidence.Record{
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-2 * time.Hour),
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Provenance: gate.ProvenanceSection{
				Sessions: []string{"auth", "shared"},
			},
		},
	})
	newer := mustWriteEvidence(t, root, &evidence.Record{
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-10 * time.Minute),
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Provenance: gate.ProvenanceSection{
				Sessions: []string{"auth"},
			},
		},
	})
	_ = mustWriteEvidence(t, root, &evidence.Record{
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-5 * time.Minute),
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Provenance: gate.ProvenanceSection{
				Sessions: []string{"review"},
			},
		},
	})
	soft := mustWriteEvidence(t, root, &evidence.Record{
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-1 * time.Minute),
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Provenance: gate.ProvenanceSection{
				Sessions: []string{"migrate"},
			},
			Approvals: gate.ApprovalsSection{
				Overrules: []gate.ExceptionOverrule{{
					Kind:       "drift",
					ResourceID: "exception-drift",
				}},
			},
		},
	})
	// Older soft-ALLOW must not win over newer clean ALLOW for "auth".
	_ = mustWriteEvidence(t, root, &evidence.Record{
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-90 * time.Minute),
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Provenance: gate.ProvenanceSection{
				Sessions: []string{"auth"},
			},
			Approvals: gate.ApprovalsSection{
				Overrules: []gate.ExceptionOverrule{{
					Kind:       "policy",
					ResourceID: "exception-old",
				}},
			},
		},
	})

	gates := NewestGateBySession(root)
	if gates["auth"].Verdict != "ALLOW" || gates["auth"].EvidenceID != newer.ID || gates["auth"].SoftAllow {
		t.Fatalf("auth=%+v want clean ALLOW/%s (not older %s)", gates["auth"], newer.ID, older.ID)
	}
	if gates["shared"].Verdict != "DENY" || gates["shared"].EvidenceID != older.ID || gates["shared"].SoftAllow {
		t.Fatalf("shared=%+v", gates["shared"])
	}
	if gates["review"].Verdict != "DENY" || gates["review"].SoftAllow {
		t.Fatalf("review=%+v", gates["review"])
	}
	if gates["migrate"].Verdict != "ALLOW" || !gates["migrate"].SoftAllow || gates["migrate"].EvidenceID != soft.ID {
		t.Fatalf("migrate=%+v want soft ALLOW/%s", gates["migrate"], soft.ID)
	}
	if _, ok := gates["missing"]; ok {
		t.Fatal("unexpected missing session")
	}
	if NewestGateBySession("") != nil || NewestGateBySession(t.TempDir()) != nil {
		t.Fatal("expected nil for empty/unpopulated roots")
	}

	store := t.TempDir()
	board := BuildStatusBoardWithEvidence([]Record{
		{ID: "auth", Status: StatusCompleted},
		{ID: "shared", Status: StatusFailed},
		{ID: "migrate", Status: StatusCompleted},
		{ID: "orphan", Status: StatusStopped},
	}, store, root)
	if board.Evidence["auth"].Verdict != "ALLOW" || board.Evidence["auth"].EvidenceID != newer.ID || board.Evidence["auth"].SoftAllow {
		t.Fatalf("board auth=%+v", board.Evidence["auth"])
	}
	if board.Evidence["shared"].Verdict != "DENY" || board.Evidence["shared"].SoftAllow {
		t.Fatalf("board shared=%+v", board.Evidence["shared"])
	}
	if board.Evidence["migrate"].Verdict != "ALLOW" || !board.Evidence["migrate"].SoftAllow || board.Evidence["migrate"].EvidenceID != soft.ID {
		t.Fatalf("board migrate=%+v", board.Evidence["migrate"])
	}
	if board.Evidence["orphan"].Verdict != "" || board.Evidence["orphan"].EvidenceID != "" || board.Evidence["orphan"].SoftAllow {
		t.Fatalf("board orphan=%+v", board.Evidence["orphan"])
	}

	ev := EvidenceFlagsFor(store, root, Record{ID: "migrate"})
	if ev.Verdict != "ALLOW" || !ev.SoftAllow || ev.EvidenceID != soft.ID {
		t.Fatalf("EvidenceFlagsFor=%+v", ev)
	}
}

func mustWriteEvidence(t *testing.T, root string, rec *evidence.Record) *evidence.Record {
	t.Helper()
	// contentID is unexported; Write path via NewFromGate-style: set ID by writing through package API.
	// Use evidence.Write after computing a stable unique ID via Save through List roundtrip.
	// Persist with a unique CreatedAt already set; evidence.Write requires ID.
	payload := *rec
	idRec, err := evidence.NewFromGate(root, rec.Gate)
	if err != nil {
		t.Fatal(err)
	}
	idRec.CreatedAt = payload.CreatedAt
	if err := evidence.Write(root, idRec); err != nil {
		t.Fatal(err)
	}
	return idRec
}
