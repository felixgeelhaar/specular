package cmd

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/evidence"
	"github.com/felixgeelhaar/specular/internal/gate"
)

func TestChangeExplainFileFlagRegistered(t *testing.T) {
	// Not parallel: rootCmd.Find mutates shared cobra command state.
	cmd, _, err := rootCmd.Find([]string{"explain"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"file", "control", "commit", "session", "policy-file"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s", name)
		}
	}
}

func TestLooksLikeCommitPrefix(t *testing.T) {
	t.Parallel()
	if !looksLikeCommitPrefix("abc123") || !looksLikeCommitPrefix("ABCDEF01") {
		t.Fatal("expected hex prefixes")
	}
	if looksLikeCommitPrefix("ev_abc123") || looksLikeCommitPrefix("ab") || looksLikeCommitPrefix("not-hex!") {
		t.Fatal("rejected invalid")
	}
}

func TestLoadEvidenceByCommitPrefix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 22, 19, 0, 0, 0, time.UTC)
	_ = mustWriteExplainRecord(t, root, &evidence.Record{
		ID:        "ev_old_sha",
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-time.Hour),
		Commit:    "abcdef0123456789deadbeef0000000000000001",
		Gate:      &gate.Result{Verdict: gate.Deny, Reason: "old"},
	})
	newer := mustWriteExplainRecord(t, root, &evidence.Record{
		ID:        "ev_new_sha",
		Schema:    evidence.Schema,
		CreatedAt: now,
		Commit:    "abcdef0123456789deadbeef0000000000000002",
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Reason:  "new",
			Change:  gate.ChangeSection{Commit: "abcdef0123456789deadbeef0000000000000002"},
		},
	})
	rec, err := loadEvidenceByFilter(root, evidence.ListFilter{
		CommitPrefix: "abcdef",
		Limit:        1,
	}, "commit", "abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != newer.ID {
		t.Fatalf("got %s want %s", rec.ID, newer.ID)
	}
}

func TestLoadEvidenceByControlNewestWins(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	_ = mustWriteExplainRecord(t, root, &evidence.Record{
		ID:        "ev_old_sec",
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-time.Hour),
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Policy:  gate.PolicySection{Status: gate.StatusFail, FailedChecks: []string{"SEC-17"}},
		},
	})
	newer := mustWriteExplainRecord(t, root, &evidence.Record{
		ID:        "ev_new_sec",
		Schema:    evidence.Schema,
		CreatedAt: now,
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Policy:  gate.PolicySection{Status: gate.StatusFail, FailedChecks: []string{"SEC-17", "Coverage"}},
		},
	})
	rec, err := loadEvidenceByFilter(root, evidence.ListFilter{
		ControlContains: "SEC-17",
		Limit:           1,
	}, "control", "SEC-17")
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != newer.ID {
		t.Fatalf("got %s want %s", rec.ID, newer.ID)
	}
}

func TestLoadEvidenceBySession(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 22, 20, 0, 0, 0, time.UTC)
	_ = mustWriteExplainRecord(t, root, &evidence.Record{
		ID:        "ev_old_auth",
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-time.Hour),
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Provenance: gate.ProvenanceSection{
				Attested: true,
				Sessions: []string{"auth"},
			},
		},
	})
	newer := mustWriteExplainRecord(t, root, &evidence.Record{
		ID:        "ev_new_auth",
		Schema:    evidence.Schema,
		CreatedAt: now,
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Provenance: gate.ProvenanceSection{
				Attested: true,
				Sessions: []string{"auth", "review"},
			},
		},
	})
	rec, err := loadEvidenceByFilter(root, evidence.ListFilter{
		Session: "auth",
		Limit:   1,
	}, "session", "auth")
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != newer.ID {
		t.Fatalf("got %s want %s", rec.ID, newer.ID)
	}
}

func TestLoadEvidenceByFileNewestWins(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)

	older := mustWriteExplainRecord(t, root, &evidence.Record{
		ID:        "ev_older_auth",
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-2 * time.Hour),
		Root:      "/repos/demo",
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Reason:  "older",
			Drift: gate.DriftSection{
				Status: gate.StatusFail,
				Findings: []gate.FindingDetail{{
					Code: "HASH_MISMATCH", Severity: "error",
					Path: "internal/auth/token.go", Location: "internal/auth/token.go:1",
				}},
			},
		},
	})
	newer := mustWriteExplainRecord(t, root, &evidence.Record{
		ID:        "ev_newer_auth",
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-10 * time.Minute),
		Root:      "/repos/demo",
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Reason:  "newer-auth",
			Drift: gate.DriftSection{
				Status: gate.StatusFail,
				Findings: []gate.FindingDetail{{
					Code: "HASH_MISMATCH", Severity: "error",
					Path: "internal/auth/session.go", Location: "internal/auth/session.go:1",
				}},
			},
		},
	})
	_ = older

	rec, err := loadEvidenceByFile(root, "internal/auth")
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != newer.ID {
		t.Fatalf("got %s want %s", rec.ID, newer.ID)
	}
}

func TestLoadEvidenceByFileNoMatch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := loadEvidenceByFile(root, "no/such/path.go")
	if err == nil || !strings.Contains(err.Error(), "no evidence matches --file") {
		t.Fatalf("err=%v", err)
	}
}

func TestExplainGraphFiltersExclusive(t *testing.T) {
	t.Parallel()
	err := explainGraphFiltersExclusive([]explainNamedFilter{
		{name: "file", value: "x.go"},
		{name: "control", value: "SEC-17"},
	}, false, nil)
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("err=%v", err)
	}
	err = explainGraphFiltersExclusive([]explainNamedFilter{
		{name: "session", value: "auth"},
	}, true, nil)
	if err == nil || !strings.Contains(err.Error(), "--fresh") {
		t.Fatalf("err=%v", err)
	}
	err = explainGraphFiltersExclusive([]explainNamedFilter{
		{name: "commit", value: "abcdef"},
	}, false, []string{"ev_abc"})
	if err == nil || !strings.Contains(err.Error(), "evidence-id") {
		t.Fatalf("err=%v", err)
	}
	if err := explainGraphFiltersExclusive(nil, false, nil); err != nil {
		t.Fatal(err)
	}
}

func TestExplainTrustFilterMerge(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	_ = mustWriteExplainRecord(t, root, &evidence.Record{
		ID:        "ev_allow_soft",
		Schema:    evidence.Schema,
		CreatedAt: now.Add(-time.Hour),
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Risk:    gate.RiskSection{Level: "MEDIUM"},
			Provenance: gate.ProvenanceSection{
				Attested: true,
				Sessions: []string{"auth"},
			},
			Approvals: gate.ApprovalsSection{
				Overrules: []gate.ExceptionOverrule{{Kind: "drift"}},
			},
		},
	})
	deny := mustWriteExplainRecord(t, root, &evidence.Record{
		ID:        "ev_deny_high",
		Schema:    evidence.Schema,
		CreatedAt: now,
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Risk:    gate.RiskSection{Level: "HIGH"},
			Provenance: gate.ProvenanceSection{
				Attested: false,
				Sessions: []string{"auth"},
			},
		},
	})

	yes := true
	no := false
	rec, err := loadEvidenceByFilter(root, withTrust(evidence.ListFilter{
		Session: "auth",
		Limit:   1,
	}, evidence.ListFilter{Verdict: gate.Deny, SoftAllow: &no}), "session", "auth")
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != deny.ID {
		t.Fatalf("got %s want %s", rec.ID, deny.ID)
	}

	rec, err = loadEvidenceByFilter(root, evidence.ListFilter{
		Verdict:   gate.Allow,
		SoftAllow: &yes,
		Limit:     1,
	}, "trust", "")
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != "ev_allow_soft" {
		t.Fatalf("got %s", rec.ID)
	}

	_, err = loadEvidenceByFilter(root, evidence.ListFilter{
		Verdict:   gate.Deny,
		RiskLevel: "CRITICAL",
		Limit:     1,
	}, "trust", "")
	if err == nil || !strings.Contains(err.Error(), "trust filters") {
		t.Fatalf("err=%v", err)
	}

	base := withTrust(evidence.ListFilter{Session: "x"}, evidence.ListFilter{
		Verdict: gate.Allow, RiskLevel: "LOW", Harness: "codex", SoftAllow: &yes,
	})
	if base.Session != "x" || base.Verdict != gate.Allow || base.RiskLevel != "LOW" || base.Harness != "codex" || base.SoftAllow == nil || !*base.SoftAllow {
		t.Fatalf("%+v", base)
	}
}

func mustWriteExplainRecord(t *testing.T, root string, rec *evidence.Record) *evidence.Record {
	t.Helper()
	if rec.ID == "" {
		rec.ID = fmt.Sprintf("ev_test_%d", time.Now().UnixNano())
	}
	if err := evidence.Write(root, rec); err != nil {
		t.Fatal(err)
	}
	return rec
}
