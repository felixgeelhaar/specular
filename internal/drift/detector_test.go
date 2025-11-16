package drift

import (
	"testing"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
)

func TestDetectPlanDrift(t *testing.T) {
	tests := []struct {
		name         string
		lock         *spec.SpecLock
		plan         *plan.Plan
		wantFindings int
		wantErrors   int
		wantWarnings int
	}{
		{
			name: "no drift",
			lock: &spec.SpecLock{
				Version: "1.0",
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {
						Hash:        "abc123",
						OpenAPIPath: ".specular/openapi/feat-001.yaml",
						TestPaths:   []string{".specular/tests/feat-001_test.go"},
					},
				},
			},
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:           types.TaskID("task-001"),
						FeatureID:    types.FeatureID("feat-001"),
						ExpectedHash: "abc123",
						DependsOn:    []types.TaskID{},
						Skill:        "go-backend",
						Priority:     types.Priority("P0"),
					},
				},
			},
			wantFindings: 0,
			wantErrors:   0,
			wantWarnings: 0,
		},
		{
			name: "hash mismatch",
			lock: &spec.SpecLock{
				Version: "1.0",
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {
						Hash:        "abc123",
						OpenAPIPath: ".specular/openapi/feat-001.yaml",
						TestPaths:   []string{".specular/tests/feat-001_test.go"},
					},
				},
			},
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:           types.TaskID("task-001"),
						FeatureID:    types.FeatureID("feat-001"),
						ExpectedHash: "xyz789", // Mismatch!
						DependsOn:    []types.TaskID{},
						Skill:        "go-backend",
						Priority:     types.Priority("P0"),
					},
				},
			},
			wantFindings: 1,
			wantErrors:   1,
			wantWarnings: 0,
		},
		{
			name: "unknown feature",
			lock: &spec.SpecLock{
				Version:  "1.0",
				Features: map[types.FeatureID]spec.LockedFeature{},
			},
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:           types.TaskID("task-001"),
						FeatureID:    types.FeatureID("feat-999"), // Unknown!
						ExpectedHash: "abc123",
						DependsOn:    []types.TaskID{},
						Skill:        "go-backend",
						Priority:     types.Priority("P0"),
					},
				},
			},
			wantFindings: 1,
			wantErrors:   1,
			wantWarnings: 0,
		},
		{
			name: "missing task",
			lock: &spec.SpecLock{
				Version: "1.0",
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {
						Hash:        "abc123",
						OpenAPIPath: ".specular/openapi/feat-001.yaml",
						TestPaths:   []string{".specular/tests/feat-001_test.go"},
					},
					types.FeatureID("feat-002"): {
						Hash:        "def456",
						OpenAPIPath: ".specular/openapi/feat-002.yaml",
						TestPaths:   []string{".specular/tests/feat-002_test.go"},
					},
				},
			},
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:           types.TaskID("task-001"),
						FeatureID:    types.FeatureID("feat-001"),
						ExpectedHash: "abc123",
						DependsOn:    []types.TaskID{},
						Skill:        "go-backend",
						Priority:     types.Priority("P0"),
					},
					// feat-002 is missing!
				},
			},
			wantFindings: 1,
			wantErrors:   0,
			wantWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := DetectPlanDrift(tt.lock, tt.plan)

			if len(findings) != tt.wantFindings {
				t.Errorf("DetectPlanDrift() found %d findings, want %d", len(findings), tt.wantFindings)
			}

			errors := 0
			warnings := 0
			for _, f := range findings {
				switch f.Severity {
				case "error":
					errors++
				case "warning":
					warnings++
				}
			}

			if errors != tt.wantErrors {
				t.Errorf("DetectPlanDrift() found %d errors, want %d", errors, tt.wantErrors)
			}

			if warnings != tt.wantWarnings {
				t.Errorf("DetectPlanDrift() found %d warnings, want %d", warnings, tt.wantWarnings)
			}
		})
	}
}

func TestGenerateReport(t *testing.T) {
	planDrift := []Finding{
		{Code: "HASH_MISMATCH", FeatureID: types.FeatureID("feat-001"), Severity: "error"},
	}
	codeDrift := []Finding{
		{Code: "API_MISMATCH", FeatureID: types.FeatureID("feat-002"), Severity: "warning"},
	}
	infraDrift := []Finding{
		{Code: "POLICY_VIOLATION", Severity: "info"},
	}

	report := GenerateReport(planDrift, codeDrift, infraDrift)

	if report.Summary.TotalFindings != 3 {
		t.Errorf("GenerateReport() total = %d, want 3", report.Summary.TotalFindings)
	}

	if report.Summary.Errors != 1 {
		t.Errorf("GenerateReport() errors = %d, want 1", report.Summary.Errors)
	}

	if report.Summary.Warnings != 1 {
		t.Errorf("GenerateReport() warnings = %d, want 1", report.Summary.Warnings)
	}

	if report.Summary.Info != 1 {
		t.Errorf("GenerateReport() info = %d, want 1", report.Summary.Info)
	}

	if !report.HasErrors() {
		t.Error("HasErrors() = false, want true")
	}

	if !report.HasWarnings() {
		t.Error("HasWarnings() = false, want true")
	}

	if report.IsClean() {
		t.Error("IsClean() = true, want false")
	}
}
