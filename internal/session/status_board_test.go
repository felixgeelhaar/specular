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
			Risk:    gate.RiskSection{Level: "HIGH"},
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
			Risk:    gate.RiskSection{Level: "LOW"},
			Provenance: gate.ProvenanceSection{
				Sessions:     []string{"auth"},
				ProtocolDocs: 1,
				ProtocolOK:   1,
			},
		},
	})
	_ = mustWriteEvidence(t, root, &evidence.Record{
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-5 * time.Minute),
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Provenance: gate.ProvenanceSection{
				Sessions:     []string{"review"},
				ProtocolDocs: 1,
				ProtocolOK:   0, // unbound / invalid
			},
		},
	})
	soft := mustWriteEvidence(t, root, &evidence.Record{
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-1 * time.Minute),
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Risk:    gate.RiskSection{Level: "MEDIUM"},
			Provenance: gate.ProvenanceSection{
				Sessions:     []string{"migrate"},
				ProtocolDocs: 2,
				ProtocolOK:   2,
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
			Risk:    gate.RiskSection{Level: "CRITICAL"},
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
	if gates["auth"].Verdict != "ALLOW" || gates["auth"].EvidenceID != newer.ID || gates["auth"].SoftAllow || gates["auth"].Risk != "LOW" || !gates["auth"].Protocol {
		t.Fatalf("auth=%+v want clean ALLOW/LOW/proto/%s (not older %s)", gates["auth"], newer.ID, older.ID)
	}
	if gates["shared"].Verdict != "DENY" || gates["shared"].EvidenceID != older.ID || gates["shared"].SoftAllow || gates["shared"].Risk != "HIGH" || gates["shared"].Protocol {
		t.Fatalf("shared=%+v", gates["shared"])
	}
	if gates["review"].Verdict != "DENY" || gates["review"].SoftAllow || gates["review"].Risk != "NONE" || gates["review"].Protocol {
		t.Fatalf("review=%+v", gates["review"])
	}
	if gates["migrate"].Verdict != "ALLOW" || !gates["migrate"].SoftAllow || gates["migrate"].EvidenceID != soft.ID || gates["migrate"].Risk != "MEDIUM" || !gates["migrate"].Protocol {
		t.Fatalf("migrate=%+v want soft ALLOW/MEDIUM/proto/%s", gates["migrate"], soft.ID)
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
	if board.Evidence["auth"].Verdict != "ALLOW" || board.Evidence["auth"].EvidenceID != newer.ID || board.Evidence["auth"].SoftAllow || board.Evidence["auth"].Risk != "LOW" || !board.Evidence["auth"].Protocol {
		t.Fatalf("board auth=%+v", board.Evidence["auth"])
	}
	if board.Evidence["shared"].Verdict != "DENY" || board.Evidence["shared"].SoftAllow || board.Evidence["shared"].Risk != "HIGH" || board.Evidence["shared"].Protocol {
		t.Fatalf("board shared=%+v", board.Evidence["shared"])
	}
	if board.Evidence["migrate"].Verdict != "ALLOW" || !board.Evidence["migrate"].SoftAllow || board.Evidence["migrate"].EvidenceID != soft.ID || board.Evidence["migrate"].Risk != "MEDIUM" || !board.Evidence["migrate"].Protocol {
		t.Fatalf("board migrate=%+v", board.Evidence["migrate"])
	}
	if board.Evidence["orphan"].Verdict != "" || board.Evidence["orphan"].EvidenceID != "" || board.Evidence["orphan"].SoftAllow || board.Evidence["orphan"].Risk != "" || board.Evidence["orphan"].Protocol {
		t.Fatalf("board orphan=%+v", board.Evidence["orphan"])
	}

	ev := EvidenceFlagsFor(store, root, Record{ID: "migrate"})
	if ev.Verdict != "ALLOW" || !ev.SoftAllow || ev.EvidenceID != soft.ID || ev.Risk != "MEDIUM" || !ev.Protocol {
		t.Fatalf("EvidenceFlagsFor=%+v", ev)
	}
}

func TestGateDetailsForNextSteps(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)

	deny := mustWriteEvidence(t, root, &evidence.Record{
		Schema:    evidence.Schema,
		CreatedAt: now,
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Risk:    gate.RiskSection{Level: "HIGH"},
			Provenance: gate.ProvenanceSection{
				Status:   gate.StatusFail,
				Attested: false,
				Sessions: []string{"auth"},
				Note:     "attested provenance required",
			},
			Drift: gate.DriftSection{Status: gate.StatusFail},
		},
	})
	allow := mustWriteEvidence(t, root, &evidence.Record{
		Schema:    evidence.Schema,
		CreatedAt: now.Add(time.Hour),
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Risk:    gate.RiskSection{Level: "LOW"},
			Provenance: gate.ProvenanceSection{
				Status:   gate.StatusPass,
				Attested: true,
				Sessions: []string{"clean"},
			},
		},
	})

	store := t.TempDir()
	denyDetails := GateDetailsFor(store, root, Record{ID: "auth"})
	if denyDetails.Verdict != "DENY" || denyDetails.EvidenceID != deny.ID || denyDetails.Risk != "HIGH" {
		t.Fatalf("deny details=%+v", denyDetails)
	}
	if len(denyDetails.NextSteps) == 0 {
		t.Fatal("expected DENY next steps")
	}
	if !denyDetails.HasSurface() {
		t.Fatal("expected HasSurface")
	}

	allowDetails := GateDetailsFor(store, root, Record{ID: "clean"})
	if allowDetails.Verdict != "ALLOW" || allowDetails.EvidenceID != allow.ID || len(allowDetails.NextSteps) != 0 {
		t.Fatalf("allow details=%+v", allowDetails)
	}

	empty := GateDetailsFor(store, root, Record{ID: "missing"})
	if empty.HasSurface() || empty.Verdict != "" || len(empty.NextSteps) != 0 {
		t.Fatalf("empty=%+v", empty)
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
