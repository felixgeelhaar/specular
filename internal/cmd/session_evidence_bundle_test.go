package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/internal/policylibrary"
)

func TestRunSessionEvidenceBundleWithPolicy(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	policyPath := filepath.Join(dir, "soc2.yaml")
	if err := policylibrary.Install("soc2-cc8.1", policyPath, true); err != nil {
		t.Fatal(err)
	}
	// Minimal drift SARIF-ish file for include
	if err := os.WriteFile("drift.sarif", []byte(`{"version":"2.1.0","runs":[{"results":[]}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "evidence.sbundle.tgz")
	if err := runSessionEvidenceBundle(sessionEvidenceBundleOptions{
		Output:   out,
		Policies: []string{policyPath},
		Quiet:    true,
	}); err != nil {
		t.Fatalf("bundle: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("missing bundle: %v", err)
	}
}

func TestRunSessionEvidenceBundleNeedsInput(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	err = runSessionEvidenceBundle(sessionEvidenceBundleOptions{
		Output: filepath.Join(dir, "empty.sbundle.tgz"),
		Quiet:  true,
	})
	if err == nil {
		t.Fatal("expected error with no inputs")
	}
}

func TestRunSessionWaitPostBundleImpliesGate(t *testing.T) {
	dir := t.TempDir()
	specular := filepath.Join(dir, ".specular")
	if err := os.MkdirAll(specular, 0o750); err != nil {
		t.Fatal(err)
	}
	specYAML := []byte(`
product: BundleDemo
goals: [demo]
features:
  - id: feat-1
    title: Demo
    desc: Demo feature
    priority: P0
    success: [ok]
    trace: []
non_functional: {}
acceptance: [ok]
milestones: []
`)
	if err := os.WriteFile(filepath.Join(specular, "spec.yaml"), specYAML, 0o600); err != nil {
		t.Fatal(err)
	}
	lockJSON := []byte(`{"version":"1","features":{"feat-1":{"hash":"abc","openapi_path":"","test_paths":[]}}}`)
	if err := os.WriteFile(filepath.Join(specular, "spec.lock.json"), lockJSON, 0o600); err != nil {
		t.Fatal(err)
	}
	planJSON := []byte(`{"tasks":[{"id":"t1","feature_id":"feat-1","expected_hash":"abc","depends_on":[],"skill":"go-backend","priority":"P0","model_hint":"codegen","estimate":1}]}`)
	if err := os.WriteFile(filepath.Join(specular, "plan.json"), planJSON, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "demo_test.go"), []byte("package demo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	policyPath := filepath.Join(dir, ".specular", "policies", "soc2-cc8.1.yaml")
	if err := policylibrary.Install("soc2-cc8.1", policyPath, true); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	out := filepath.Join(dir, "session-evidence.sbundle.tgz")
	if err := runSessionWaitPost(sessionWaitPostOptions{
		Bundle:    true,
		BundleOut: out,
		Policies:  []string{policyPath},
		Quiet:     true,
	}); err != nil {
		t.Fatalf("post: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("bundle missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "drift.sarif")); err != nil {
		t.Fatalf("gate should write drift.sarif: %v", err)
	}
}

func TestRunSessionEvidenceBundleIncludesProvenance(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	sess := filepath.Join(dir, ".specular", "sessions")
	if err := os.MkdirAll(sess, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sess, "auth.attestation.json"), []byte(`{"version":"1.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sess, "auth.provenance.json"), []byte(`{"schema":"specular.provenance/v1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	policyPath := filepath.Join(dir, "soc2.yaml")
	if err := policylibrary.Install("soc2-cc8.1", policyPath, true); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "with-app.sbundle.tgz")
	if err := runSessionEvidenceBundle(sessionEvidenceBundleOptions{
		Output:   out,
		Policies: []string{policyPath},
		Quiet:    true,
	}); err != nil {
		t.Fatalf("bundle: %v", err)
	}
	// Smoke: builder accepted APP doc path (Validate requires ≥1 include).
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
}
