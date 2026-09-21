package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/gate"
)

func TestWriteLoadLatestRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	res := &gate.Result{
		Verdict: gate.Allow,
		Reason:  "drift skipped; policy skipped; provenance unattested",
		Change:  gate.ChangeSection{Branch: "main", Dirty: false},
		Drift:   gate.DriftSection{Status: gate.StatusSkipped, Note: "brownfield"},
		Policy:  gate.PolicySection{Status: gate.StatusSkipped},
		Provenance: gate.ProvenanceSection{
			Status: gate.StatusSkipped,
			Note:   "unattested",
		},
	}
	rec, err := NewFromGate(root, res)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rec.ID, "ev_") {
		t.Fatalf("id=%s", rec.ID)
	}
	if err := Write(root, rec); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLatest(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != rec.ID || got.Gate.Verdict != gate.Allow {
		t.Fatalf("%+v", got)
	}
	path := filepath.Join(Dir(root), rec.ID+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestFormatExplainDenyWithFindings(t *testing.T) {
	t.Parallel()
	rec := &Record{
		ID:        "ev_test",
		CreatedAt: mustTime(t, "2026-09-21T09:41:00Z"),
		Root:      "/repos/payments-api",
		Branch:    "feature/auth",
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Reason:  "hash mismatch",
			Change:  gate.ChangeSection{Dirty: true, Files: 2, Branch: "feature/auth"},
			Drift: gate.DriftSection{
				Status: gate.StatusFail,
				Errors: 1,
				Note:   "hash mismatch",
				Findings: []gate.FindingDetail{{
					Category: "code",
					Code:     "HASH_MISMATCH",
					Severity: "error",
					Message:  "file hash mismatch",
					Location: "src/a.go",
				}},
			},
			Policy:     gate.PolicySection{Status: gate.StatusSkipped},
			Provenance: gate.ProvenanceSection{Status: gate.StatusSkipped},
		},
	}
	text := FormatExplain(rec)
	for _, want := range []string{
		"AI CHANGE RECORD",
		"Decision     DENY",
		"Reason       hash mismatch",
		"Timestamp    2026-09-21T09:41:00Z",
		"Evidence     ev_test",
		"Repository   payments-api",
		"Branch       feature/auth",
		"Files        2 uncommitted",
		"Status       UNATTESTED",
		"✗ FAIL",
		"HASH_MISMATCH",
		"at src/a.go",
		"Why",
		"Drift failed",
		"→ DENY",
		".specular/evidence/ev_test.json",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "SPECULAR EXPLAIN") {
		t.Fatalf("legacy title still present:\n%s", text)
	}
}

func TestFormatExplainAllowAttested(t *testing.T) {
	t.Parallel()
	rec := &Record{
		ID: "ev_ok",
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Reason:  "drift pass; policy pass",
			Change:  gate.ChangeSection{Dirty: false, Files: 0, Branch: "main", Root: "/tmp/demo"},
			Drift:   gate.DriftSection{Status: gate.StatusPass},
			Policy: gate.PolicySection{
				Status:  gate.StatusPass,
				Passed:  3,
				Failed:  0,
				Skipped: 1,
			},
			Provenance: gate.ProvenanceSection{
				Status:    gate.StatusPass,
				Attested:  true,
				Harnesses: []string{"Claude Code"},
				Sessions:  []string{"sp_92d1"},
			},
		},
	}
	text := FormatExplain(rec)
	for _, want := range []string{
		"AI CHANGE RECORD",
		"Decision     ALLOW",
		"Status       ATTESTED",
		"Harness      Claude Code",
		"Session      sp_92d1",
		"✓ PASS",
		"Summary      None detected",
		"Checks       passed=3 failed=0 skipped=1",
		"→ ALLOW",
		"Files        clean working tree",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
}

func TestFormatExplainNil(t *testing.T) {
	t.Parallel()
	if got := FormatExplain(nil); got != "No evidence record.\n" {
		t.Fatalf("got %q", got)
	}
}

func mustTime(t *testing.T, rfc3339 string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func TestListIDsOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	r1, err := NewFromGate(root, &gate.Result{
		Verdict: gate.Allow,
		Reason:  "a",
		Drift:   gate.DriftSection{Status: gate.StatusSkipped},
		Policy:  gate.PolicySection{Status: gate.StatusSkipped},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(root, r1); err != nil {
		t.Fatal(err)
	}
	// Distinct gate payload → distinct id
	r2, err := NewFromGate(root, &gate.Result{
		Verdict: gate.Deny,
		Reason:  "b",
		Drift:   gate.DriftSection{Status: gate.StatusFail, Note: "x"},
		Policy:  gate.PolicySection{Status: gate.StatusSkipped},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(root, r2); err != nil {
		t.Fatal(err)
	}
	ids, err := ListIDs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids=%v", ids)
	}
	latest, err := LoadLatest(root)
	if err != nil {
		t.Fatal(err)
	}
	if latest.ID != r2.ID {
		t.Fatalf("latest=%s want %s", latest.ID, r2.ID)
	}
}
