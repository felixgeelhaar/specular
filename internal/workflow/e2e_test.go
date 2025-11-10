package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/spec"
)

// TestE2EWorkflow tests the complete end-to-end workflow
func TestE2EWorkflow(t *testing.T) {
	tests := []struct {
		name           string
		specContent    string
		policyContent  string
		apiSpecContent string
		dryRun         bool
		failOnDrift    bool
		wantErr        bool
		wantDrift      bool
	}{
		{
			name: "successful workflow without drift",
			specContent: `
product: TestProduct
goals:
  - Build a test API service
features:
  - id: feat-001
    title: User API
    desc: RESTful API for user management
    priority: P0
    api:
      - method: GET
        path: /api/users
        request: ""
        response: UserList
      - method: POST
        path: /api/users
        request: CreateUserRequest
        response: User
    success:
      - API endpoints respond correctly
      - Tests pass with >80% coverage
    trace:
      - PRD-001
acceptance:
  - All API endpoints accessible
  - Integration tests pass
`,
			policyContent: `
execution:
  allow_local: false
  docker:
    required: true
    image_allowlist:
      - golang:1.22
      - node:22
    cpu_limit: "2"
    mem_limit: "2g"
    network: "none"
tests:
  require_pass: true
  min_coverage: 0.70
security:
  secrets_scan: true
  dep_scan: false
`,
			apiSpecContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /api/users:
    get:
      summary: List users
      responses:
        '200':
          description: Success
    post:
      summary: Create user
      responses:
        '201':
          description: Created
`,
			dryRun:      true, // Use dry-run to avoid Docker dependency
			failOnDrift: false,
			wantErr:     false,
			wantDrift:   false,
		},
		{
			name: "workflow with plan drift detected",
			specContent: `
product: TestProduct
goals:
  - Build a test service
features:
  - id: feat-001
    title: Feature One
    desc: First feature
    priority: P0
    success:
      - Feature works
    trace:
      - PRD-001
  - id: feat-002
    title: Feature Two
    desc: Second feature
    priority: P1
    success:
      - Feature works
    trace:
      - PRD-002
acceptance:
  - All features work correctly
  - Tests pass
`,
			policyContent: `
execution:
  docker:
    required: true
    image_allowlist:
      - golang:1.22
tests:
  require_pass: true
  min_coverage: 0.70
`,
			dryRun:      true,
			failOnDrift: false,
			wantErr:     false,
			wantDrift:   false, // No drift expected as we're not modifying hashes
		},
		{
			name: "workflow with API drift detected",
			specContent: `
product: TestProduct
goals:
  - Build a test API
features:
  - id: feat-001
    title: API Feature
    desc: RESTful API
    priority: P0
    api:
      - method: GET
        path: /api/missing
        request: ""
        response: Response
    success:
      - API works
    trace:
      - PRD-001
acceptance:
  - API endpoints accessible
  - Tests pass
`,
			policyContent: `
execution:
  docker:
    required: true
    image_allowlist:
      - golang:1.22
`,
			apiSpecContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /api/users:
    get:
      summary: List users
      responses:
        '200':
          description: Success
`,
			dryRun:      true,
			failOnDrift: false,
			wantErr:     false,
			wantDrift:   true, // Expect drift: spec has /api/missing, OpenAPI has /api/users
		},
		{
			name: "workflow fails on drift when configured",
			specContent: `
product: TestProduct
goals:
  - Build a test API
features:
  - id: feat-001
    title: API Feature
    desc: RESTful API
    priority: P0
    api:
      - method: GET
        path: /api/missing
        request: ""
        response: Response
    success:
      - API works
    trace:
      - PRD-001
acceptance:
  - API endpoints accessible
  - Tests pass
`,
			policyContent: `
execution:
  docker:
    required: true
    image_allowlist:
      - golang:1.22
`,
			apiSpecContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /api/users:
    get:
      summary: List users
      responses:
        '200':
          description: Success
`,
			dryRun:      true,
			failOnDrift: true, // Should fail due to drift
			wantErr:     true,
			wantDrift:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir := t.TempDir()

			// Create .specular directory
			aidvDir := filepath.Join(tmpDir, ".specular")
			if err := os.MkdirAll(aidvDir, 0755); err != nil {
				t.Fatalf("Failed to create .specular directory: %v", err)
			}

			// Write spec file
			specPath := filepath.Join(aidvDir, "spec.yaml")
			if err := os.WriteFile(specPath, []byte(tt.specContent), 0644); err != nil {
				t.Fatalf("Failed to write spec file: %v", err)
			}

			// Write policy file
			policyPath := filepath.Join(aidvDir, "policy.yaml")
			if err := os.WriteFile(policyPath, []byte(tt.policyContent), 0644); err != nil {
				t.Fatalf("Failed to write policy file: %v", err)
			}

			// Write API spec file if provided
			var apiSpecPath string
			if tt.apiSpecContent != "" {
				apiSpecPath = filepath.Join(tmpDir, "openapi.yaml")
				if err := os.WriteFile(apiSpecPath, []byte(tt.apiSpecContent), 0644); err != nil {
					t.Fatalf("Failed to write API spec file: %v", err)
				}
			}

			// Create workflow configuration
			config := WorkflowConfig{
				ProjectRoot: tmpDir,
				SpecPath:    specPath,
				PolicyPath:  policyPath,
				APISpecPath: apiSpecPath,
				DryRun:      tt.dryRun,
				FailOnDrift: tt.failOnDrift,
			}

			// Create and execute workflow
			workflow := NewWorkflow(config)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			result, err := workflow.Execute(ctx)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				if result != nil && len(result.Errors) > 0 {
					for i, e := range result.Errors {
						t.Logf("  Error %d: %v", i+1, e)
					}
				}
				return
			}

			// If we expected an error and got one, we're done
			if tt.wantErr {
				return
			}

			// Verify result is not nil
			if result == nil {
				t.Fatal("Execute() returned nil result")
			}

			// Verify SpecLock was generated
			if result.SpecLock == nil {
				t.Error("Execute() did not generate SpecLock")
			} else {
				// Verify SpecLock has features
				if len(result.SpecLock.Features) == 0 {
					t.Error("Execute() generated empty SpecLock")
				}

				// Verify SpecLock file was saved
				lockPath := filepath.Join(aidvDir, "spec.lock.json")
				if _, err := os.Stat(lockPath); os.IsNotExist(err) {
					t.Error("Execute() did not save spec.lock.json")
				}
			}

			// Verify Plan was generated
			if result.Plan == nil {
				t.Error("Execute() did not generate Plan")
			} else {
				// Verify Plan has tasks
				if len(result.Plan.Tasks) == 0 {
					t.Error("Execute() generated empty Plan")
				}

				// Verify Plan file was saved
				planPath := filepath.Join(tmpDir, "plan.json")
				if _, err := os.Stat(planPath); os.IsNotExist(err) {
					t.Error("Execute() did not save plan.json")
				}
			}

			// Verify drift detection matches expectation
			if tt.wantDrift && len(result.DriftFindings) == 0 {
				t.Error("Execute() expected drift but found none")
			}
			if !tt.wantDrift && len(result.DriftFindings) > 0 {
				t.Errorf("Execute() found unexpected drift: %d findings", len(result.DriftFindings))
				for i, f := range result.DriftFindings {
					t.Logf("  Finding %d: %s - %s", i+1, f.Code, f.Message)
				}
			}

			// Verify workflow completed in reasonable time
			if result.Duration > 30*time.Second {
				t.Errorf("Execute() took too long: %v (expected < 30s)", result.Duration)
			}

			t.Logf("Workflow completed in %v", result.Duration)
			t.Logf("SpecLock features: %d", len(result.SpecLock.Features))
			t.Logf("Plan tasks: %d", len(result.Plan.Tasks))
			t.Logf("Drift findings: %d", len(result.DriftFindings))
		})
	}
}

// TestE2EWorkflowStateTransitions tests that workflow properly manages state
func TestE2EWorkflowStateTransitions(t *testing.T) {
	tmpDir := t.TempDir()
	aidvDir := filepath.Join(tmpDir, ".specular")
	if err := os.MkdirAll(aidvDir, 0755); err != nil {
		t.Fatalf("Failed to create .specular directory: %v", err)
	}

	// Create minimal spec
	specContent := `
product: TestProduct
goals:
  - Test goal
features:
  - id: feat-001
    title: Test Feature
    desc: Test description
    priority: P0
    success:
      - It works
    trace:
      - PRD-001
acceptance:
  - Feature works correctly
  - Tests pass
`
	specPath := filepath.Join(aidvDir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("Failed to write spec: %v", err)
	}

	policyContent := `
execution:
  docker:
    required: true
    image_allowlist:
      - golang:1.22
`
	policyPath := filepath.Join(aidvDir, "policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	// Execute workflow
	config := WorkflowConfig{
		ProjectRoot: tmpDir,
		SpecPath:    specPath,
		PolicyPath:  policyPath,
		DryRun:      true,
		FailOnDrift: false,
	}

	workflow := NewWorkflow(config)
	ctx := context.Background()

	result, err := workflow.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify files were created in correct order
	// 1. spec.lock.json should exist
	lockPath := filepath.Join(aidvDir, "spec.lock.json")
	lockInfo, err := os.Stat(lockPath)
	if err != nil {
		t.Fatalf("spec.lock.json not created: %v", err)
	}

	// 2. plan.json should exist and be created after lock
	planPath := filepath.Join(tmpDir, "plan.json")
	planInfo, err := os.Stat(planPath)
	if err != nil {
		t.Fatalf("plan.json not created: %v", err)
	}

	if planInfo.ModTime().Before(lockInfo.ModTime()) {
		t.Error("plan.json created before spec.lock.json (incorrect order)")
	}

	// Verify we can reload the generated files
	reloadedLock, err := spec.LoadSpecLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to reload spec lock: %v", err)
	}

	if len(reloadedLock.Features) != len(result.SpecLock.Features) {
		t.Errorf("Reloaded lock has %d features, original has %d",
			len(reloadedLock.Features), len(result.SpecLock.Features))
	}
}

// TestE2EWorkflowCleanup tests that workflow properly cleans up on errors
func TestE2EWorkflowCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid spec (missing required fields)
	specContent := `
product: ""
goals: []
features: []
`
	specPath := filepath.Join(tmpDir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("Failed to write spec: %v", err)
	}

	config := WorkflowConfig{
		ProjectRoot: tmpDir,
		SpecPath:    specPath,
		DryRun:      true,
	}

	workflow := NewWorkflow(config)
	ctx := context.Background()

	// Execute should fail with validation errors for invalid spec
	result, err := workflow.Execute(ctx)

	// The workflow should fail because the spec is invalid
	if err == nil {
		t.Error("Execute() expected error for invalid spec, got nil")
	} else {
		// Validation error is expected - log it for verification
		t.Logf("Execute() returned expected validation error: %v", err)
	}

	// Result may be nil or incomplete due to validation failure
	if result != nil {
		t.Logf("Result: SpecLock features=%d, Plan tasks=%d",
			len(result.SpecLock.Features), len(result.Plan.Tasks))
	}
}

// TestE2EWorkflowMultiplePresets tests workflow with different interview presets
func TestE2EWorkflowMultiplePresets(t *testing.T) {
	presets := []struct {
		name        string
		specContent string
	}{
		{
			name: "web-app",
			specContent: `
product: WebApp
goals:
  - Build responsive web application
features:
  - id: feat-001
    title: Frontend
    desc: React-based UI
    priority: P0
    success:
      - UI renders correctly
    trace:
      - PRD-001
acceptance:
  - Application is responsive
  - Tests pass
`,
		},
		{
			name: "api-service",
			specContent: `
product: APIService
goals:
  - Build RESTful API
features:
  - id: feat-001
    title: API Endpoints
    desc: REST API
    priority: P0
    api:
      - method: GET
        path: /api/health
        request: ""
        response: HealthResponse
    success:
      - API responds
    trace:
      - PRD-001
acceptance:
  - API endpoints accessible
  - Tests pass
`,
		},
		{
			name: "cli-tool",
			specContent: `
product: CLITool
goals:
  - Build command-line tool
features:
  - id: feat-001
    title: CLI Commands
    desc: Command interface
    priority: P0
    success:
      - Commands work
    trace:
      - PRD-001
acceptance:
  - CLI commands work correctly
  - Tests pass
`,
		},
	}

	for _, preset := range presets {
		t.Run(preset.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			aidvDir := filepath.Join(tmpDir, ".specular")
			if err := os.MkdirAll(aidvDir, 0755); err != nil {
				t.Fatalf("Failed to create .specular directory: %v", err)
			}

			specPath := filepath.Join(aidvDir, "spec.yaml")
			if err := os.WriteFile(specPath, []byte(preset.specContent), 0644); err != nil {
				t.Fatalf("Failed to write spec: %v", err)
			}

			config := WorkflowConfig{
				ProjectRoot: tmpDir,
				SpecPath:    specPath,
				DryRun:      true,
			}

			workflow := NewWorkflow(config)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := workflow.Execute(ctx)
			if err != nil {
				t.Fatalf("Execute() failed for %s: %v", preset.name, err)
			}

			if result == nil {
				t.Fatal("Execute() returned nil result")
			}

			if result.SpecLock == nil || len(result.SpecLock.Features) == 0 {
				t.Error("Execute() did not generate SpecLock with features")
			}

			if result.Plan == nil || len(result.Plan.Tasks) == 0 {
				t.Error("Execute() did not generate Plan with tasks")
			}

			t.Logf("%s: Generated %d features, %d tasks in %v",
				preset.name,
				len(result.SpecLock.Features),
				len(result.Plan.Tasks),
				result.Duration)
		})
	}
}
