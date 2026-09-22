package evidence

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/gate"
)

func TestListFilters(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)

	allowOld := mustWriteRecord(t, root, &Record{
		Schema:    Schema,
		CreatedAt: now.Add(-48 * time.Hour),
		Root:      "/repos/payments-api",
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Reason:  "old-allow",
			Change:  gate.ChangeSection{Root: "/repos/payments-api"},
			Drift:   gate.DriftSection{Status: gate.StatusPass},
			Policy:  gate.PolicySection{Status: gate.StatusSkipped},
		},
	})
	denyMid := mustWriteRecord(t, root, &Record{
		Schema:    Schema,
		CreatedAt: now.Add(-2 * time.Hour),
		Root:      "/repos/payments-api",
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Reason:  "mid-deny",
			Change:  gate.ChangeSection{Root: "/repos/payments-api"},
			Drift: gate.DriftSection{
				Status: gate.StatusFail,
				Findings: []gate.FindingDetail{{
					Code:     "HASH_MISMATCH",
					Severity: "error",
					Message:  "mismatch",
					Path:     "internal/auth/token.go",
					Location: "internal/auth/token.go:12",
				}},
			},
			Policy: gate.PolicySection{Status: gate.StatusSkipped},
			Risk:   gate.RiskSection{Level: "HIGH", Factors: []string{"authentication / authorization paths modified"}},
			Provenance: gate.ProvenanceSection{
				Status:    gate.StatusPass,
				Attested:  true,
				Sessions:  []string{"auth", "auth-alt"},
				Harnesses: []string{"claude-code"},
			},
		},
	})
	allowNew := mustWriteRecord(t, root, &Record{
		Schema:    Schema,
		CreatedAt: now.Add(-30 * time.Minute),
		Root:      "/repos/other",
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Reason:  "new-allow",
			Change:  gate.ChangeSection{Root: "/repos/other"},
			Drift:   gate.DriftSection{Status: gate.StatusPass},
			Policy:  gate.PolicySection{Status: gate.StatusSkipped},
			Risk:    gate.RiskSection{Level: "LOW"},
			Provenance: gate.ProvenanceSection{
				Status:    gate.StatusPass,
				Attested:  true,
				Sessions:  []string{"fleet-review"},
				Harnesses: []string{"cursor"},
			},
		},
	})
	softAllow := mustWriteRecord(t, root, &Record{
		Schema:    Schema,
		CreatedAt: now.Add(-10 * time.Minute),
		Root:      "/repos/payments-api",
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Reason:  "drift fail (exception soft-ALLOW)",
			Change:  gate.ChangeSection{Root: "/repos/payments-api"},
			Drift:   gate.DriftSection{Status: gate.StatusFail},
			Policy:  gate.PolicySection{Status: gate.StatusSkipped},
			Risk:    gate.RiskSection{Level: "MEDIUM"},
			Provenance: gate.ProvenanceSection{
				Status:   gate.StatusFail,
				Attested: true,
				Sessions: []string{"migrate"},
				Note:     "APP protocol enforce: attested sessions missing .provenance.json",
			},
			Approvals: gate.ApprovalsSection{
				Overrules: []gate.ExceptionOverrule{{
					Kind:       "provenance",
					ResourceID: "exception-app-protocol",
					ApprovedBy: "platform",
				}},
			},
		},
	})

	t.Run("newest_first", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{softAllow.ID, allowNew.ID, denyMid.ID, allowOld.ID}
		got := idsOf(recs)
		if len(got) != len(want) {
			t.Fatalf("got %v want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("got %v want %v", got, want)
			}
		}
	})

	t.Run("verdict_deny", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{Verdict: gate.Deny})
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != denyMid.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
	})

	t.Run("since", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{Since: now.Add(-3 * time.Hour)})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{softAllow.ID, allowNew.ID, denyMid.ID}
		got := idsOf(recs)
		if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
			t.Fatalf("got %v want %v", got, want)
		}
	})

	t.Run("path_finding", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{PathContains: "internal/auth"})
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != denyMid.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
	})

	t.Run("path_root", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{PathContains: "payments-api"})
		if err != nil {
			t.Fatal(err)
		}
		got := idsOf(recs)
		if len(got) != 3 {
			t.Fatalf("got %v", got)
		}
		// newest first among matches
		if got[0] != softAllow.ID || got[1] != denyMid.ID || got[2] != allowOld.ID {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("limit", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != softAllow.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
	})

	t.Run("combined", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{
			Verdict:      gate.Allow,
			Since:        now.Add(-72 * time.Hour),
			PathContains: "payments",
			Limit:        5,
		})
		if err != nil {
			t.Fatal(err)
		}
		got := idsOf(recs)
		if len(got) != 2 || got[0] != softAllow.ID || got[1] != allowOld.ID {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("invalid_verdict", func(t *testing.T) {
		t.Parallel()
		_, err := List(root, ListFilter{Verdict: gate.Verdict("MAYBE")})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("risk_high", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{RiskLevel: "HIGH"})
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != denyMid.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
	})

	t.Run("risk_none_includes_empty", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{RiskLevel: "NONE"})
		if err != nil {
			t.Fatal(err)
		}
		// allowOld has no Risk.Level → treated as NONE
		if len(recs) != 1 || recs[0].ID != allowOld.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
	})

	t.Run("invalid_risk", func(t *testing.T) {
		t.Parallel()
		_, err := List(root, ListFilter{RiskLevel: "EXTREME"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("session_exact", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{Session: "auth"})
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != denyMid.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
		// Must not substring-match auth-alt alone when querying auth-alt's sibling.
		miss, err := List(root, ListFilter{Session: "auth-missing"})
		if err != nil {
			t.Fatal(err)
		}
		if len(miss) != 0 {
			t.Fatalf("got %v", idsOf(miss))
		}
	})

	t.Run("harness_substring", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{Harness: "Claude"})
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != denyMid.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
		cursor, err := List(root, ListFilter{Harness: "cursor"})
		if err != nil {
			t.Fatal(err)
		}
		if len(cursor) != 1 || cursor[0].ID != allowNew.ID {
			t.Fatalf("got %v", idsOf(cursor))
		}
	})

	t.Run("session_and_harness", func(t *testing.T) {
		t.Parallel()
		recs, err := List(root, ListFilter{Session: "auth", Harness: "claude"})
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != denyMid.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
		miss, err := List(root, ListFilter{Session: "auth", Harness: "cursor"})
		if err != nil {
			t.Fatal(err)
		}
		if len(miss) != 0 {
			t.Fatalf("got %v", idsOf(miss))
		}
	})

	t.Run("soft_allow_true", func(t *testing.T) {
		t.Parallel()
		yes := true
		recs, err := List(root, ListFilter{SoftAllow: &yes})
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != softAllow.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
	})

	t.Run("soft_allow_false", func(t *testing.T) {
		t.Parallel()
		no := false
		recs, err := List(root, ListFilter{SoftAllow: &no})
		if err != nil {
			t.Fatal(err)
		}
		got := idsOf(recs)
		if len(got) != 3 {
			t.Fatalf("got %v", got)
		}
		for _, id := range got {
			if id == softAllow.ID {
				t.Fatalf("soft-allow record leaked: %v", got)
			}
		}
	})

	t.Run("attested_true", func(t *testing.T) {
		t.Parallel()
		yes := true
		recs, err := List(root, ListFilter{Attested: &yes})
		if err != nil {
			t.Fatal(err)
		}
		got := idsOf(recs)
		if len(got) != 3 || got[0] != softAllow.ID || got[1] != allowNew.ID || got[2] != denyMid.ID {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("attested_false", func(t *testing.T) {
		t.Parallel()
		no := false
		recs, err := List(root, ListFilter{Attested: &no})
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != allowOld.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
	})

	t.Run("soft_allow_and_attested", func(t *testing.T) {
		t.Parallel()
		yes := true
		recs, err := List(root, ListFilter{SoftAllow: &yes, Attested: &yes, Verdict: gate.Allow})
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != softAllow.ID {
			t.Fatalf("got %v", idsOf(recs))
		}
	})
}

func TestParseSince(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)

	got, err := ParseSince("24h", now)
	if err != nil {
		t.Fatal(err)
	}
	want := now.Add(-24 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}

	got, err = ParseSince("2026-09-01T00:00:00Z", now)
	if err != nil {
		t.Fatal(err)
	}
	want = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}

	if _, err := ParseSince("not-a-time", now); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ParseSince("-1h", now); err == nil {
		t.Fatal("expected negative duration error")
	}
	empty, err := ParseSince("", now)
	if err != nil || !empty.IsZero() {
		t.Fatalf("empty: %v %v", empty, err)
	}
}

func TestListIDsUsesCreatedAtOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	older := mustWriteRecord(t, root, &Record{
		Schema:    Schema,
		CreatedAt: now.Add(-time.Hour),
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Reason:  "older",
			Drift:   gate.DriftSection{Status: gate.StatusSkipped},
			Policy:  gate.PolicySection{Status: gate.StatusSkipped},
		},
	})
	newer := mustWriteRecord(t, root, &Record{
		Schema:    Schema,
		CreatedAt: now,
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Reason:  "newer",
			Drift:   gate.DriftSection{Status: gate.StatusFail},
			Policy:  gate.PolicySection{Status: gate.StatusSkipped},
		},
	})
	ids, err := ListIDs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != newer.ID || ids[1] != older.ID {
		t.Fatalf("ids=%v", ids)
	}
}

func mustWriteRecord(t *testing.T, root string, rec *Record) *Record {
	t.Helper()
	id, err := contentID(rec)
	if err != nil {
		t.Fatal(err)
	}
	rec.ID = id
	if err := Write(root, rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

func idsOf(recs []*Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.ID
	}
	return out
}
