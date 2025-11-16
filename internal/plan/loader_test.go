package plan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

func TestLoadPlan(t *testing.T) {
	tests := []struct {
		name        string
		planContent string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *Plan)
	}{
		{
			name: "valid complete plan",
			planContent: `{
  "tasks": [
    {
      "id": "task-001",
      "feature_id": "feat-001",
      "expected_hash": "abc123",
      "depends_on": [],
      "skill": "go-backend",
      "model_hint": "codegen",
      "priority": "P0",
      "estimate": 7
    },
    {
      "id": "task-002",
      "feature_id": "feat-002",
      "expected_hash": "def456",
      "depends_on": ["task-001"],
      "skill": "ui-react",
      "model_hint": "codegen",
      "priority": "P1",
      "estimate": 5
    }
  ]
}`,
			wantErr: false,
			validate: func(t *testing.T, p *Plan) {
				if len(p.Tasks) != 2 {
					t.Errorf("Tasks length = %d, want 2", len(p.Tasks))
				}
				if p.Tasks[0].ID != "task-001" {
					t.Errorf("Task[0].ID = %v, want task-001", p.Tasks[0].ID)
				}
				if p.Tasks[0].Skill != "go-backend" {
					t.Errorf("Task[0].Skill = %v, want go-backend", p.Tasks[0].Skill)
				}
				if len(p.Tasks[1].DependsOn) != 1 {
					t.Errorf("Task[1].DependsOn length = %d, want 1", len(p.Tasks[1].DependsOn))
				}
			},
		},
		{
			name: "minimal plan with empty tasks",
			planContent: `{
  "tasks": []
}`,
			wantErr:     true,
			errContains: "plan must have at least one task",
		},
		{
			name: "single task plan",
			planContent: `{
  "tasks": [
    {
      "id": "task-001",
      "feature_id": "feat-001",
      "expected_hash": "hash1",
      "depends_on": [],
      "skill": "database",
      "model_hint": "agentic",
      "priority": "P0",
      "estimate": 8
    }
  ]
}`,
			wantErr: false,
			validate: func(t *testing.T, p *Plan) {
				if len(p.Tasks) != 1 {
					t.Errorf("Tasks length = %d, want 1", len(p.Tasks))
				}
				if p.Tasks[0].Estimate != 8 {
					t.Errorf("Task[0].Estimate = %d, want 8", p.Tasks[0].Estimate)
				}
			},
		},
		{
			name:        "invalid json",
			planContent: `{invalid json`,
			wantErr:     true,
			errContains: "unmarshal plan",
		},
		{
			name:        "empty file",
			planContent: "",
			wantErr:     true,
			errContains: "unmarshal plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			planFile := filepath.Join(tmpDir, "plan.json")

			err := os.WriteFile(planFile, []byte(tt.planContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test plan file: %v", err)
			}

			plan, err := LoadPlan(planFile)

			if tt.wantErr {
				if err == nil {
					t.Error("LoadPlan() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("LoadPlan() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadPlan() unexpected error = %v", err)
			}

			if plan == nil {
				t.Fatal("LoadPlan() returned nil plan")
			}

			if tt.validate != nil {
				tt.validate(t, plan)
			}
		})
	}
}

func TestLoadPlan_FileNotFound(t *testing.T) {
	_, err := LoadPlan("/nonexistent/path/plan.json")
	if err == nil {
		t.Error("LoadPlan() expected error for nonexistent file, got nil")
	}
	if !contains(err.Error(), "read plan file") {
		t.Errorf("LoadPlan() error = %v, want error containing 'read plan file'", err)
	}
}

func TestSavePlan(t *testing.T) {
	tests := []struct {
		name    string
		plan    *Plan
		wantErr bool
	}{
		{
			name: "complete plan",
			plan: &Plan{
				Tasks: []Task{
					{
						ID:           types.TaskID("task-001"),
						FeatureID:    types.FeatureID("feat-001"),
						ExpectedHash: "abc123",
						DependsOn:    []types.TaskID{},
						Skill:        "go-backend",
						ModelHint:    "codegen",
						Priority:     types.Priority("P0"),
						Estimate:     7,
					},
					{
						ID:           types.TaskID("task-002"),
						FeatureID:    types.FeatureID("feat-002"),
						ExpectedHash: "def456",
						DependsOn:    []types.TaskID{types.TaskID("task-001")},
						Skill:        "ui-react",
						ModelHint:    "codegen",
						Priority:     types.Priority("P1"),
						Estimate:     5,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "minimal plan",
			plan: &Plan{
				Tasks: []Task{},
			},
			wantErr: true, // SavePlan validates during LoadPlan roundtrip
		},
		{
			name: "single task",
			plan: &Plan{
				Tasks: []Task{
					{
						ID:           types.TaskID("task-only"),
						FeatureID:    types.FeatureID("feat-only"),
						ExpectedHash: "hash1",
						DependsOn:    []types.TaskID{},
						Skill:        "testing",
						ModelHint:    "fast",
						Priority:     types.Priority("P2"),
						Estimate:     3,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			planFile := filepath.Join(tmpDir, "plan.json")

			err := SavePlan(tt.plan, planFile)
			if err != nil {
				t.Fatalf("SavePlan() unexpected error = %v", err)
			}

			// Verify file was created
			if _, err := os.Stat(planFile); os.IsNotExist(err) {
				t.Error("SavePlan() did not create file")
			}

			// Verify file can be loaded back - validation happens here
			loaded, err := LoadPlan(planFile)

			if tt.wantErr {
				if err == nil {
					t.Error("LoadPlan() after SavePlan() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadPlan() after SavePlan() failed: %v", err)
			}

			if len(loaded.Tasks) != len(tt.plan.Tasks) {
				t.Errorf("Loaded Tasks length = %d, want %d", len(loaded.Tasks), len(tt.plan.Tasks))
			}
		})
	}
}

func TestPlanRoundTrip(t *testing.T) {
	// Create a plan with various task configurations
	plan := &Plan{
		Tasks: []Task{
			{
				ID:           types.TaskID("task-001"),
				FeatureID:    types.FeatureID("feat-001"),
				ExpectedHash: "hash001",
				DependsOn:    []types.TaskID{},
				Skill:        "go-backend",
				ModelHint:    "codegen",
				Priority:     types.Priority("P0"),
				Estimate:     8,
			},
			{
				ID:           types.TaskID("task-002"),
				FeatureID:    types.FeatureID("feat-002"),
				ExpectedHash: "hash002",
				DependsOn:    []types.TaskID{types.TaskID("task-001")},
				Skill:        "ui-react",
				ModelHint:    "agentic",
				Priority:     types.Priority("P1"),
				Estimate:     6,
			},
			{
				ID:           types.TaskID("task-003"),
				FeatureID:    types.FeatureID("feat-003"),
				ExpectedHash: "hash003",
				DependsOn:    []types.TaskID{types.TaskID("task-001"), types.TaskID("task-002")},
				Skill:        "infra",
				ModelHint:    "fast",
				Priority:     types.Priority("P2"),
				Estimate:     4,
			},
		},
	}

	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.json")

	// Save plan
	err := SavePlan(plan, planFile)
	if err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Load plan
	loaded, err := LoadPlan(planFile)
	if err != nil {
		t.Fatalf("LoadPlan() error = %v", err)
	}

	// Verify round-trip
	if len(loaded.Tasks) != len(plan.Tasks) {
		t.Errorf("Round-trip Tasks length = %d, want %d", len(loaded.Tasks), len(plan.Tasks))
	}

	for i := range plan.Tasks {
		if loaded.Tasks[i].ID != plan.Tasks[i].ID {
			t.Errorf("Round-trip Task[%d].ID = %v, want %v", i, loaded.Tasks[i].ID, plan.Tasks[i].ID)
		}
		if loaded.Tasks[i].Skill != plan.Tasks[i].Skill {
			t.Errorf("Round-trip Task[%d].Skill = %v, want %v", i, loaded.Tasks[i].Skill, plan.Tasks[i].Skill)
		}
		if loaded.Tasks[i].Estimate != plan.Tasks[i].Estimate {
			t.Errorf("Round-trip Task[%d].Estimate = %d, want %d", i, loaded.Tasks[i].Estimate, plan.Tasks[i].Estimate)
		}
		if len(loaded.Tasks[i].DependsOn) != len(plan.Tasks[i].DependsOn) {
			t.Errorf("Round-trip Task[%d].DependsOn length = %d, want %d",
				i, len(loaded.Tasks[i].DependsOn), len(plan.Tasks[i].DependsOn))
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
