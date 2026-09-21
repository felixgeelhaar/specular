package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		ID: "ev_test",
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Reason:  "hash mismatch",
			Drift: gate.DriftSection{
				Status: gate.StatusFail,
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
		"VERDICT: DENY",
		"Why this decision?",
		"Drift failed",
		"HASH_MISMATCH",
		"at src/a.go",
		"→ DENY",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
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
