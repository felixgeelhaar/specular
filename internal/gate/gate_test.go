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
	text := FormatText(res)
	if !strings.Contains(text, "VERDICT: ALLOW") {
		t.Fatalf("board missing ALLOW:\n%s", text)
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
