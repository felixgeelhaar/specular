package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/internal/exitcode"
)

func TestRunSessionProductGateClean(t *testing.T) {
	dir := t.TempDir()
	specular := filepath.Join(dir, ".specular")
	if err := os.MkdirAll(specular, 0o750); err != nil {
		t.Fatal(err)
	}
	specYAML := []byte(`
product: GateDemo
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
	// P0 features require a matching *test* file (title/id heuristic).
	if err := os.WriteFile(filepath.Join(dir, "demo_test.go"), []byte("package demo\n"), 0o600); err != nil {
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

	report := filepath.Join(dir, "out.sarif")
	if err := runSessionProductGate(sessionProductGateOptions{
		ProjectRoot: dir,
		ReportFile:  report,
		Quiet:       true,
	}); err != nil {
		t.Fatalf("gate: %v", err)
	}
	if _, err := os.Stat(report); err != nil {
		t.Fatalf("sarif missing: %v", err)
	}
	// Product gate persists Change Evidence Graph records.
	evDir := filepath.Join(dir, ".specular", "evidence")
	entries, listErr := os.ReadDir(evDir)
	if listErr != nil || len(entries) == 0 {
		t.Fatalf("expected evidence record under %s: %v", evDir, listErr)
	}
}

func TestRunSessionProductGateFailsOnDrift(t *testing.T) {
	dir := t.TempDir()
	specular := filepath.Join(dir, ".specular")
	if err := os.MkdirAll(specular, 0o750); err != nil {
		t.Fatal(err)
	}
	specYAML := []byte(`
product: GateDrift
goals: [demo]
features:
  - id: feat-1
    title: Untested
    desc: P0 without tests
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

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	err = runSessionProductGate(sessionProductGateOptions{
		ProjectRoot: dir,
		ReportFile:  filepath.Join(dir, "drift.sarif"),
		Quiet:       true,
	})
	if err == nil {
		t.Fatal("expected drift error")
	}
	if code := exitcode.DetermineExitCode(err); code != exitcode.DriftDetected {
		t.Fatalf("code=%d want %d for %v", code, exitcode.DriftDetected, err)
	}
}

func TestRunSessionProductGateDefaultsProjectRoot(t *testing.T) {
	dir := t.TempDir()
	specular := filepath.Join(dir, ".specular")
	if err := os.MkdirAll(specular, 0o750); err != nil {
		t.Fatal(err)
	}
	specYAML := []byte(`
product: GateDemo
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

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	// Empty ProjectRoot must resolve to cwd (CLI wait --gate path).
	if err := runSessionProductGate(sessionProductGateOptions{
		ReportFile: filepath.Join(dir, "cwd.sarif"),
		Quiet:      true,
	}); err != nil {
		t.Fatalf("gate with default project root: %v", err)
	}
}

func TestRunSessionProductGateBrownfieldAllows(t *testing.T) {
	// Product gate soft-skips missing specs (unlike the old drift-only path).
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	if err := runSessionProductGate(sessionProductGateOptions{Quiet: true}); err != nil {
		t.Fatalf("brownfield should ALLOW: %v", err)
	}
}

func TestRunSessionProductGateStrictSpecMissing(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	err = runSessionProductGate(sessionProductGateOptions{Quiet: true, StrictSpec: true})
	if err == nil {
		t.Fatal("expected strict-spec DENY/error without specs")
	}
}

func TestProductGateDenyMapsToExitCode(t *testing.T) {
	err := fmt.Errorf("drift detection failed: code drift")
	if code := exitcode.DetermineExitCode(err); code != exitcode.DriftDetected {
		t.Fatalf("code=%d want %d for %v", code, exitcode.DriftDetected, err)
	}
}
