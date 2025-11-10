package plan

import (
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/domain"
)

func TestTask_Validate(t *testing.T) {
	validTask := Task{
		ID:           domain.TaskID("task-001"),
		FeatureID:    domain.FeatureID("user-auth"),
		ExpectedHash: "abc123hash",
		DependsOn:    []domain.TaskID{domain.TaskID("task-000")},
		Skill:        "go-backend",
		Priority:     domain.Priority("P0"),
		ModelHint:    "codegen",
		Estimate:     5,
	}

	tests := []struct {
		name    string
		task    Task
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid task",
			task:    validTask,
			wantErr: false,
		},
		{
			name: "invalid task ID - empty",
			task: Task{
				ID:           "",
				FeatureID:    domain.FeatureID("user-auth"),
				ExpectedHash: "hash",
				Skill:        "go-backend",
				Priority:     domain.Priority("P0"),
				ModelHint:    "codegen",
				Estimate:     5,
			},
			wantErr: true,
			errMsg:  "invalid task ID",
		},
		{
			name: "invalid task ID - uppercase",
			task: Task{
				ID:           "Task-001",
				FeatureID:    domain.FeatureID("user-auth"),
				ExpectedHash: "hash",
				Skill:        "go-backend",
				Priority:     domain.Priority("P0"),
				ModelHint:    "codegen",
				Estimate:     5,
			},
			wantErr: true,
			errMsg:  "invalid task ID",
		},
		{
			name: "invalid feature ID - empty",
			task: Task{
				ID:           domain.TaskID("task-001"),
				FeatureID:    "",
				ExpectedHash: "hash",
				Skill:        "go-backend",
				Priority:     domain.Priority("P0"),
				ModelHint:    "codegen",
				Estimate:     5,
			},
			wantErr: true,
			errMsg:  "invalid feature ID",
		},
		{
			name: "invalid feature ID - uppercase",
			task: Task{
				ID:           domain.TaskID("task-001"),
				FeatureID:    domain.FeatureID("User-Auth"),
				ExpectedHash: "hash",
				Skill:        "go-backend",
				Priority:     domain.Priority("P0"),
				ModelHint:    "codegen",
				Estimate:     5,
			},
			wantErr: true,
			errMsg:  "invalid feature ID",
		},
		{
			name: "empty expected hash",
			task: Task{
				ID:           domain.TaskID("task-001"),
				FeatureID:    domain.FeatureID("user-auth"),
				ExpectedHash: "",
				Skill:        "go-backend",
				Priority:     domain.Priority("P0"),
				ModelHint:    "codegen",
				Estimate:     5,
			},
			wantErr: true,
			errMsg:  "expected hash cannot be empty",
		},
		{
			name: "invalid dependency task ID",
			task: Task{
				ID:           domain.TaskID("task-001"),
				FeatureID:    domain.FeatureID("user-auth"),
				ExpectedHash: "hash",
				DependsOn:    []domain.TaskID{domain.TaskID("Task-000")},
				Skill:        "go-backend",
				Priority:     domain.Priority("P0"),
				ModelHint:    "codegen",
				Estimate:     5,
			},
			wantErr: true,
			errMsg:  "dependency at index 0 has invalid task ID",
		},
		{
			name: "empty skill",
			task: Task{
				ID:           domain.TaskID("task-001"),
				FeatureID:    domain.FeatureID("user-auth"),
				ExpectedHash: "hash",
				Skill:        "",
				Priority:     domain.Priority("P0"),
				ModelHint:    "codegen",
				Estimate:     5,
			},
			wantErr: true,
			errMsg:  "skill cannot be empty",
		},
		{
			name: "invalid priority",
			task: Task{
				ID:           domain.TaskID("task-001"),
				FeatureID:    domain.FeatureID("user-auth"),
				ExpectedHash: "hash",
				Skill:        "go-backend",
				Priority:     domain.Priority("P3"),
				ModelHint:    "codegen",
				Estimate:     5,
			},
			wantErr: true,
			errMsg:  "invalid priority",
		},
		{
			name: "empty model hint",
			task: Task{
				ID:           domain.TaskID("task-001"),
				FeatureID:    domain.FeatureID("user-auth"),
				ExpectedHash: "hash",
				Skill:        "go-backend",
				Priority:     domain.Priority("P0"),
				ModelHint:    "",
				Estimate:     5,
			},
			wantErr: true,
			errMsg:  "model hint cannot be empty",
		},
		{
			name: "zero estimate",
			task: Task{
				ID:           domain.TaskID("task-001"),
				FeatureID:    domain.FeatureID("user-auth"),
				ExpectedHash: "hash",
				Skill:        "go-backend",
				Priority:     domain.Priority("P0"),
				ModelHint:    "codegen",
				Estimate:     0,
			},
			wantErr: true,
			errMsg:  "estimate must be positive",
		},
		{
			name: "negative estimate",
			task: Task{
				ID:           domain.TaskID("task-001"),
				FeatureID:    domain.FeatureID("user-auth"),
				ExpectedHash: "hash",
				Skill:        "go-backend",
				Priority:     domain.Priority("P0"),
				ModelHint:    "codegen",
				Estimate:     -5,
			},
			wantErr: true,
			errMsg:  "estimate must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Task.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Task.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestPlan_Validate(t *testing.T) {
	validPlan := Plan{
		Tasks: []Task{
			{
				ID:           domain.TaskID("task-001"),
				FeatureID:    domain.FeatureID("feature-1"),
				ExpectedHash: "hash1",
				Skill:        "go-backend",
				Priority:     domain.Priority("P0"),
				ModelHint:    "codegen",
				Estimate:     5,
			},
			{
				ID:           domain.TaskID("task-002"),
				FeatureID:    domain.FeatureID("feature-1"),
				ExpectedHash: "hash2",
				DependsOn:    []domain.TaskID{domain.TaskID("task-001")},
				Skill:        "ui-react",
				Priority:     domain.Priority("P1"),
				ModelHint:    "agentic",
				Estimate:     3,
			},
		},
	}

	tests := []struct {
		name    string
		plan    Plan
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid plan",
			plan:    validPlan,
			wantErr: false,
		},
		{
			name: "empty plan",
			plan: Plan{
				Tasks: []Task{},
			},
			wantErr: true,
			errMsg:  "at least one task",
		},
		{
			name: "invalid task in plan",
			plan: Plan{
				Tasks: []Task{
					{
						ID:           "",
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash",
						Skill:        "go-backend",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     5,
					},
				},
			},
			wantErr: true,
			errMsg:  "task at index 0",
		},
		{
			name: "duplicate task IDs",
			plan: Plan{
				Tasks: []Task{
					{
						ID:           domain.TaskID("task-001"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash1",
						Skill:        "go-backend",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     5,
					},
					{
						ID:           domain.TaskID("task-001"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash2",
						Skill:        "ui-react",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     3,
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate task ID",
		},
		{
			name: "dependency references non-existent task",
			plan: Plan{
				Tasks: []Task{
					{
						ID:           domain.TaskID("task-001"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash",
						DependsOn:    []domain.TaskID{domain.TaskID("task-999")},
						Skill:        "go-backend",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     5,
					},
				},
			},
			wantErr: true,
			errMsg:  "dependency \"task-999\" that does not exist",
		},
		{
			name: "circular dependency - self reference",
			plan: Plan{
				Tasks: []Task{
					{
						ID:           domain.TaskID("task-001"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash",
						DependsOn:    []domain.TaskID{domain.TaskID("task-001")},
						Skill:        "go-backend",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     5,
					},
				},
			},
			wantErr: true,
			errMsg:  "circular dependency detected",
		},
		{
			name: "circular dependency - two tasks",
			plan: Plan{
				Tasks: []Task{
					{
						ID:           domain.TaskID("task-001"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash1",
						DependsOn:    []domain.TaskID{domain.TaskID("task-002")},
						Skill:        "go-backend",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     5,
					},
					{
						ID:           domain.TaskID("task-002"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash2",
						DependsOn:    []domain.TaskID{domain.TaskID("task-001")},
						Skill:        "ui-react",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     3,
					},
				},
			},
			wantErr: true,
			errMsg:  "circular dependency detected",
		},
		{
			name: "circular dependency - three tasks",
			plan: Plan{
				Tasks: []Task{
					{
						ID:           domain.TaskID("task-001"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash1",
						DependsOn:    []domain.TaskID{domain.TaskID("task-003")},
						Skill:        "go-backend",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     5,
					},
					{
						ID:           domain.TaskID("task-002"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash2",
						DependsOn:    []domain.TaskID{domain.TaskID("task-001")},
						Skill:        "ui-react",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     3,
					},
					{
						ID:           domain.TaskID("task-003"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash3",
						DependsOn:    []domain.TaskID{domain.TaskID("task-002")},
						Skill:        "infra",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     2,
					},
				},
			},
			wantErr: true,
			errMsg:  "circular dependency detected",
		},
		{
			name: "valid complex DAG",
			plan: Plan{
				Tasks: []Task{
					{
						ID:           domain.TaskID("task-001"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash1",
						Skill:        "go-backend",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     5,
					},
					{
						ID:           domain.TaskID("task-002"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash2",
						DependsOn:    []domain.TaskID{domain.TaskID("task-001")},
						Skill:        "ui-react",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     3,
					},
					{
						ID:           domain.TaskID("task-003"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash3",
						DependsOn:    []domain.TaskID{domain.TaskID("task-001")},
						Skill:        "infra",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     2,
					},
					{
						ID:           domain.TaskID("task-004"),
						FeatureID:    domain.FeatureID("feature-1"),
						ExpectedHash: "hash4",
						DependsOn:    []domain.TaskID{domain.TaskID("task-002"), domain.TaskID("task-003")},
						Skill:        "testing",
						Priority:     domain.Priority("P0"),
						ModelHint:    "codegen",
						Estimate:     4,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Plan.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Plan.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}
