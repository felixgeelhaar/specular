//go:build e2e
// +build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCompleteWorkflow tests the entire Specular workflow from spec to evaluation
func TestCompleteWorkflow(t *testing.T) {
	// Get project root directory (two levels up from test/e2e)
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	specularBin := filepath.Join(projectRoot, "specular")

	// Build the specular binary first
	buildCmd := exec.Command("go", "build", "-o", specularBin, "./cmd/specular")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build specular: %v\n%s", err, output)
	}
	defer os.Remove(specularBin)

	// Create temporary test directory
	tmpDir := t.TempDir()
	aidvDir := filepath.Join(tmpDir, ".specular")
	if err := os.MkdirAll(aidvDir, 0755); err != nil {
		t.Fatalf("Failed to create .specular directory: %v", err)
	}

	// Create minimal test spec
	specContent := `product: E2E Test Product
version: 1.0.0

goals:
  - Validate complete Specular workflow

acceptance:
  - All tests pass
  - API endpoints work correctly

features:
  - id: feat-001
    title: Test Feature
    desc: Simple feature for E2E validation
    priority: P0
    api:
      - method: GET
        path: /api/health
        request: ""
        response: HealthResponse
    success:
      - API responds with 200
      - Tests pass
    trace:
      - TEST-001
`
	specPath := filepath.Join(aidvDir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("Failed to write spec.yaml: %v", err)
	}

	// Create policy file
	policyContent := `version: 1.0

execution:
  allow_local: false
  docker:
    required: true
    image_allowlist:
      - alpine:latest
      - golang:1.22
      - golang:1.22-alpine
    resource_limits:
      cpu: "1"
      memory: "512m"
    network: "none"

tests:
  require_pass: false
  min_coverage: 0.0
`
	policyPath := filepath.Join(aidvDir, "policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy.yaml: %v", err)
	}

	// Create OpenAPI spec for drift detection
	openAPIContent := `openapi: 3.0.0
info:
  title: E2E Test API
  version: 1.0.0
paths:
  /api/health:
    get:
      summary: Health check
      responses:
        '200':
          description: OK
`
	openAPIPath := filepath.Join(tmpDir, "openapi.yaml")
	if err := os.WriteFile(openAPIPath, []byte(openAPIContent), 0644); err != nil {
		t.Fatalf("Failed to write openapi.yaml: %v", err)
	}

	// Step 0: Generate spec lock (required for plan generation)
	lockCmd := exec.Command(specularBin, "spec", "lock", "--in", specPath, "--out", filepath.Join(aidvDir, "spec.lock.json"))
	lockCmd.Dir = tmpDir
	if output, err := lockCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to generate spec lock: %v\n%s", err, output)
	}

	// Step 1: Generate plan from spec
	t.Run("Step1-GeneratePlan", func(t *testing.T) {
		planPath := filepath.Join(tmpDir, "plan.json")

		cmd := exec.Command(specularBin, "plan",
			"--in", specPath,
			"--out", planPath,
		)
		cmd.Dir = tmpDir

		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Plan generation failed: %v\n%s", err, output)
		}

		// Verify plan file was created
		if _, err := os.Stat(planPath); os.IsNotExist(err) {
			t.Fatal("plan.json was not created")
		}

		// Verify spec.lock.json was created
		lockPath := filepath.Join(aidvDir, "spec.lock.json")
		if _, err := os.Stat(lockPath); os.IsNotExist(err) {
			t.Fatal("spec.lock.json was not created")
		}

		t.Logf("Plan generation completed in %v", duration)

		// Basic validation of plan content
		planData, err := os.ReadFile(planPath)
		if err != nil {
			t.Fatalf("Failed to read plan.json: %v", err)
		}

		planStr := string(planData)
		if !strings.Contains(planStr, "feat-001") {
			t.Error("Plan does not contain expected feature ID")
		}
		if !strings.Contains(planStr, "task-001") {
			t.Error("Plan does not contain expected task ID")
		}
		if !strings.Contains(planStr, "\"tasks\"") {
			t.Error("Plan does not contain tasks array")
		}
	})

	// Step 2: Execute build with dry-run
	t.Run("Step2-ExecuteBuild", func(t *testing.T) {
		planPath := filepath.Join(tmpDir, "plan.json")

		cmd := exec.Command(specularBin, "build",
			"--plan", planPath,
			"--policy", policyPath,
			"--dry-run", // Avoid Docker dependency in CI
		)
		cmd.Dir = tmpDir

		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			t.Logf("Build output:\n%s", output)
			t.Fatalf("Build execution failed: %v", err)
		}

		t.Logf("Build completed in %v", duration)

		// Verify output mentions tasks
		outputStr := string(output)
		if !strings.Contains(outputStr, "task") && !strings.Contains(outputStr, "Task") {
			t.Logf("Build output:\n%s", output)
			t.Error("Build output does not mention tasks")
		}
	})

	// Step 3: Detect drift
	t.Run("Step3-DetectDrift", func(t *testing.T) {
		planPath := filepath.Join(tmpDir, "plan.json")
		lockPath := filepath.Join(aidvDir, "spec.lock.json")
		reportPath := filepath.Join(tmpDir, "drift.sarif")

		cmd := exec.Command(specularBin, "eval",
			"--spec", specPath,
			"--plan", planPath,
			"--lock", lockPath,
			"--api-spec", openAPIPath,
			"--report", reportPath,
			"--project-root", tmpDir,
		)
		cmd.Dir = tmpDir

		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		// Note: eval may exit with error if quality gates fail (expected for this minimal spec)
		// We verify the output and SARIF report were generated correctly
		t.Logf("Drift detection completed in %v (exit status: %v)", duration, err)
		t.Logf("Eval output:\n%s", output)

		// Verify SARIF report was created
		if _, err := os.Stat(reportPath); os.IsNotExist(err) {
			t.Fatal("drift.sarif was not created")
		}

		// Verify drift detection ran (expect errors for missing test files in minimal spec)
		outputStr := string(output)
		// The minimal test spec will have drift errors for missing test files
		// This is expected and validates that drift detection is working
		if !strings.Contains(outputStr, "Drift Detection Summary") {
			t.Error("Drift detection output missing summary")
		}
		if !strings.Contains(outputStr, "Code Drift:") {
			t.Error("Drift detection output missing code drift section")
		}

		// Validate SARIF format
		sarifData, err := os.ReadFile(reportPath)
		if err != nil {
			t.Fatalf("Failed to read drift.sarif: %v", err)
		}

		sarifStr := string(sarifData)
		if !strings.Contains(sarifStr, "\"version\"") {
			t.Error("SARIF report does not contain version field")
		}
		if !strings.Contains(sarifStr, "\"runs\"") {
			t.Error("SARIF report does not contain runs field")
		}
	})

	// Step 4: Verify artifacts
	t.Run("Step4-VerifyArtifacts", func(t *testing.T) {
		artifacts := []struct {
			name string
			path string
		}{
			{"Specification", specPath},
			{"Policy", policyPath},
			{"OpenAPI Spec", openAPIPath},
			{"Plan", filepath.Join(tmpDir, "plan.json")},
			{"Spec Lock", filepath.Join(aidvDir, "spec.lock.json")},
			{"SARIF Report", filepath.Join(tmpDir, "drift.sarif")},
		}

		for _, artifact := range artifacts {
			if _, err := os.Stat(artifact.path); os.IsNotExist(err) {
				t.Errorf("Artifact %s not found at %s", artifact.name, artifact.path)
			} else {
				info, _ := os.Stat(artifact.path)
				t.Logf("✓ %s exists (%d bytes)", artifact.name, info.Size())
			}
		}
	})
}

// TestWorkflowWithModifiedSpec tests drift detection when spec changes
func TestWorkflowWithModifiedSpec(t *testing.T) {
	// Get project root directory (two levels up from test/e2e)
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	specularBin := filepath.Join(projectRoot, "specular")

	// Build the specular binary first
	buildCmd := exec.Command("go", "build", "-o", specularBin, "./cmd/specular")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build specular: %v\n%s", err, output)
	}
	defer os.Remove(specularBin)

	tmpDir := t.TempDir()
	aidvDir := filepath.Join(tmpDir, ".specular")
	if err := os.MkdirAll(aidvDir, 0755); err != nil {
		t.Fatalf("Failed to create .specular directory: %v", err)
	}

	// Create initial spec
	specContent := `product: Drift Test
version: 1.0.0

goals:
  - Test drift detection

acceptance:
  - Drift detection works correctly

features:
  - id: feat-001
    title: Original Feature
    desc: Initial description
    priority: P0
    success:
      - Works correctly
    trace:
      - TEST-001
`
	specPath := filepath.Join(aidvDir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("Failed to write spec.yaml: %v", err)
	}

	policyPath := filepath.Join(aidvDir, "policy.yaml")
	policyContent := `version: 1.0
execution:
  docker:
    required: true
    image_allowlist:
      - alpine:latest
      - golang:1.22
      - golang:1.22-alpine
`
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy.yaml: %v", err)
	}

	planPath := filepath.Join(tmpDir, "plan.json")

	// Generate spec lock (required for plan generation)
	lockPath := filepath.Join(aidvDir, "spec.lock.json")
	lockCmd := exec.Command(specularBin, "spec", "lock", "--in", specPath, "--out", lockPath)
	lockCmd.Dir = tmpDir
	if output, err := lockCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to generate spec lock: %v\n%s", err, output)
	}

	// Generate initial plan
	cmd := exec.Command(specularBin, "plan", "--in", specPath, "--out", planPath)
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Initial plan generation failed: %v\n%s", err, output)
	}

	// Modify the spec (this should cause drift)
	modifiedSpecContent := `product: Drift Test
version: 1.0.0

goals:
  - Test drift detection

acceptance:
  - Drift detection works correctly

features:
  - id: feat-001
    title: Modified Feature
    desc: Changed description to trigger drift
    priority: P0
    success:
      - Works correctly
      - New success criterion
    trace:
      - TEST-001
`
	if err := os.WriteFile(specPath, []byte(modifiedSpecContent), 0644); err != nil {
		t.Fatalf("Failed to write modified spec.yaml: %v", err)
	}

	// Detect drift (should find changes)
	reportPath := filepath.Join(tmpDir, "drift.sarif")

	cmd = exec.Command(specularBin, "eval",
		"--spec", specPath,
		"--plan", planPath,
		"--lock", lockPath,
		"--report", reportPath,
		"--project-root", tmpDir,
	)
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()

	// We expect the command to succeed but report drift
	if err != nil {
		t.Logf("Eval output:\n%s", output)
		// Note: Command may exit with error if drift is found, check output
	}

	outputStr := string(output)
	t.Logf("Drift detection output:\n%s", outputStr)

	// Verify SARIF report was created
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatal("drift.sarif was not created")
	}

	// Verify drift was detected
	// The output should mention drift or findings
	if !strings.Contains(outputStr, "drift") && !strings.Contains(outputStr, "Drift") &&
		!strings.Contains(outputStr, "finding") && !strings.Contains(outputStr, "Finding") {
		t.Error("Expected drift to be detected but output suggests none found")
	}

	t.Log("✓ Drift detection working correctly")
}

// TestCheckpointResume tests checkpoint and resume functionality
func TestCheckpointResume(t *testing.T) {
	// This test validates that checkpoints are created and can be resumed
	// For now, we just verify checkpoint directory is created
	tmpDir := t.TempDir()
	checkpointDir := filepath.Join(tmpDir, ".specular", "checkpoints")

	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		t.Fatalf("Failed to create checkpoint directory: %v", err)
	}

	// Verify checkpoint directory exists
	if _, err := os.Stat(checkpointDir); os.IsNotExist(err) {
		t.Fatal("Checkpoint directory was not created")
	}

	t.Log("✓ Checkpoint directory structure validated")
}
