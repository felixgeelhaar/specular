package gate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateBrownfieldAllow(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Allow {
		t.Fatalf("verdict=%s reason=%s", res.Verdict, res.Reason)
	}
	if res.Drift.Status != StatusSkipped {
		t.Fatalf("drift=%s", res.Drift.Status)
	}
	if res.Policy.Status != StatusSkipped {
		t.Fatalf("policy=%s", res.Policy.Status)
	}
	if res.Provenance.Attested {
		t.Fatal("expected unattested")
	}
	if !containsFactor(res.Risk.Factors, FactorUnattested) {
		t.Fatalf("expected unattested risk factor, got %v", res.Risk.Factors)
	}
	text := FormatText(res)
	if !strings.Contains(text, "VERDICT: ALLOW") {
		t.Fatalf("board missing ALLOW:\n%s", text)
	}
	if !strings.Contains(text, "Risk") {
		t.Fatalf("board missing Risk:\n%s", text)
	}
}

func TestEvaluateStrictSpecDeniesMissing(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	res, err := Evaluate(Options{ProjectRoot: root, StrictSpec: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Deny {
		t.Fatalf("verdict=%s", res.Verdict)
	}
	if res.Drift.Status != StatusFail {
		t.Fatalf("drift=%s note=%s", res.Drift.Status, res.Drift.Note)
	}
}

func TestEvaluateProvenanceFromAttestation(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	att := `{"provenance":{"harness":"claude-code"}}`
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Provenance.Attested || res.Provenance.Status != StatusPass {
		t.Fatalf("%+v", res.Provenance)
	}
	if len(res.Provenance.Harnesses) != 1 || res.Provenance.Harnesses[0] != "claude-code" {
		t.Fatalf("harnesses=%v", res.Provenance.Harnesses)
	}
}

func TestSortFindingsStableOrder(t *testing.T) {
	t.Parallel()
	in := []FindingDetail{
		{Severity: "warning", Category: "plan", Code: "B", FeatureID: "z"},
		{Severity: "error", Category: "code", Code: "A", FeatureID: "a", Location: "b"},
		{Severity: "error", Category: "code", Code: "A", FeatureID: "a", Location: "a"},
		{Severity: "info", Category: "infra", Code: "C"},
	}
	sortFindings(in)
	want := []string{"error|code|A|a|a", "error|code|A|a|b", "warning|plan|B|z|", "info|infra|C||"}
	for i, f := range in {
		got := strings.Join([]string{f.Severity, f.Category, f.Code, f.FeatureID, f.Location}, "|")
		if got != want[i] {
			t.Fatalf("i=%d got %s want %s", i, got, want[i])
		}
	}
}

func TestProvenanceSessionsSorted(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"zeta", "alpha", "mid"} {
		att := `{"provenance":{"harness":"claude-code"}}`
		if err := os.WriteFile(filepath.Join(dir, id+".attestation.json"), []byte(att), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "mid", "zeta"}
	if len(res.Provenance.Sessions) != 3 {
		t.Fatalf("sessions=%v", res.Provenance.Sessions)
	}
	for i := range want {
		if res.Provenance.Sessions[i] != want[i] {
			t.Fatalf("sessions=%v want %v", res.Provenance.Sessions, want)
		}
	}
}

func TestDecidePolicyFail(t *testing.T) {
	t.Parallel()
	res := &Result{
		Drift:  DriftSection{Status: StatusPass},
		Policy: PolicySection{Status: StatusFail, Note: "policy verification failed"},
	}
	v, reason := decide(res)
	if v != Deny || !strings.Contains(reason, "policy") {
		t.Fatalf("%s %s", v, reason)
	}
}

func TestFormatTextIncludesDriftFindings(t *testing.T) {
	t.Parallel()
	res := &Result{
		Verdict: Deny,
		Reason:  "file hash mismatch (src/auth.go)",
		Change:  ChangeSection{Dirty: true, Files: 2, Branch: "feat/x"},
		Provenance: ProvenanceSection{
			Status: StatusSkipped,
			Note:   "unattested",
		},
		Drift: DriftSection{
			Status: StatusFail,
			Errors: 1,
			Findings: []FindingDetail{{
				Category:  "code",
				Code:      "HASH_MISMATCH",
				FeatureID: "AUTH-1",
				Message:   "file hash mismatch for src/auth.go",
				Severity:  "error",
				Location:  "src/auth.go",
			}},
			Note: "file hash mismatch for src/auth.go (src/auth.go)",
		},
		Policy: PolicySection{Status: StatusSkipped, Note: "no policy"},
	}
	text := FormatText(res)
	for _, want := range []string{
		"HASH_MISMATCH",
		"feature=AUTH-1",
		"file hash mismatch for src/auth.go",
		"at src/auth.go",
		"VERDICT: DENY",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("board missing %q:\n%s", want, text)
		}
	}
}

func TestExplainDriftFailurePrefersFirstError(t *testing.T) {
	t.Parallel()
	got := explainDriftFailure([]FindingDetail{
		{Severity: "warning", Message: "warn", Code: "W"},
		{Severity: "error", Message: "plan feature missing", Code: "UNKNOWN_FEATURE", Location: "plan.yaml"},
	}, 1)
	if !strings.Contains(got, "plan feature missing") || !strings.Contains(got, "plan.yaml") {
		t.Fatalf("got %q", got)
	}
}

func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@example.com")
	run(t, dir, "git", "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "README.md")
	run(t, dir, "git", "commit", "-m", "init")
	return dir
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
