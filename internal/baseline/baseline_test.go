package baseline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/gate"
)

func TestCaptureWriteLoad(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	res := &gate.Result{
		Verdict: gate.Allow,
		Reason:  "drift skipped; policy skipped",
		Drift:   gate.DriftSection{Status: gate.StatusSkipped, Note: "brownfield"},
		Policy:  gate.PolicySection{Status: gate.StatusSkipped},
	}
	doc, err := Capture(root, res, DefaultPosture("advisory"), "initial brownfield", "ev_test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(doc.ID, "bl_") {
		t.Fatalf("id=%s", doc.ID)
	}
	if err := Write(root, doc); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != doc.ID || got.Snapshot.Verdict != "ALLOW" {
		t.Fatalf("%+v", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".specular", DriftPointerName)); err != nil {
		t.Fatal(err)
	}
	text := FormatText(got)
	for _, want := range []string{"SPECULAR BASELINE", "not a claim", "provenance"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
}

func TestCompareDetectsNewDrift(t *testing.T) {
	t.Parallel()
	base, err := Capture(".", &gate.Result{
		Verdict: gate.Allow,
		Drift:   gate.DriftSection{Status: gate.StatusSkipped},
		Policy:  gate.PolicySection{Status: gate.StatusSkipped},
	}, DefaultPosture("advisory"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	cur := &gate.Result{
		Verdict: gate.Deny,
		Drift: gate.DriftSection{
			Status: gate.StatusFail,
			Findings: []gate.FindingDetail{{
				Code:     "HASH_MISMATCH",
				Severity: "error",
			}},
		},
		Policy: gate.PolicySection{Status: gate.StatusSkipped},
	}
	d := Compare(base, cur)
	if !d.Changed || !strings.Contains(d.Summary, "change") {
		t.Fatalf("%+v", d)
	}
	if !strings.Contains(FormatDiff(d), "drift newly failing") {
		t.Fatal(FormatDiff(d))
	}
}

func TestCompareClean(t *testing.T) {
	t.Parallel()
	res := &gate.Result{
		Verdict: gate.Allow,
		Drift:   gate.DriftSection{Status: gate.StatusSkipped},
		Policy:  gate.PolicySection{Status: gate.StatusSkipped},
	}
	base, err := Capture(".", res, DefaultPosture("advisory"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	d := Compare(base, res)
	if d.Changed {
		t.Fatalf("%+v", d)
	}
}
