package plan

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

func TestGenerate(t *testing.T) {
	testSpec := &spec.ProductSpec{
		Product: "Test Product",
		Features: []spec.Feature{
			{
				ID:       types.FeatureID("feat-001"),
				Title:    "User Authentication API",
				Desc:     "JWT-based authentication",
				Priority: types.Priority("P0"),
				API: []spec.API{
					{Method: "POST", Path: "/api/login"},
				},
				Success: []string{"Users can login"},
				Trace:   []string{"PRD-001"},
			},
			{
				ID:       types.FeatureID("feat-002"),
				Title:    "User Profile UI",
				Desc:     "React component for user profile",
				Priority: types.Priority("P1"),
				Success:  []string{"Profile displays correctly"},
				Trace:    []string{"PRD-002"},
			},
			{
				ID:       types.FeatureID("feat-003"),
				Title:    "Docker Deployment",
				Desc:     "Containerize the application",
				Priority: types.Priority("P2"),
				Success:  []string{"App runs in Docker"},
				Trace:    []string{"PRD-003"},
			},
		},
	}

	testLock := &spec.SpecLock{
		Version: "1.0",
		Features: map[types.FeatureID]spec.LockedFeature{
			types.FeatureID("feat-001"): {Hash: "hash001"},
			types.FeatureID("feat-002"): {Hash: "hash002"},
			types.FeatureID("feat-003"): {Hash: "hash003"},
		},
	}

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: true,
	}

	plan, err := Generate(context.Background(), testSpec, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(plan.Tasks) != 3 {
		t.Errorf("Generate() created %d tasks, want 3", len(plan.Tasks))
	}

	// Check task IDs
	if plan.Tasks[0].ID != "task-001" {
		t.Errorf("Task 0 ID = %s, want task-001", plan.Tasks[0].ID)
	}

	// Check hashes
	if plan.Tasks[0].ExpectedHash != "hash001" {
		t.Errorf("Task 0 hash = %s, want hash001", plan.Tasks[0].ExpectedHash)
	}

	// Check priorities
	if plan.Tasks[0].Priority != "P0" {
		t.Errorf("Task 0 priority = %s, want P0", plan.Tasks[0].Priority)
	}

	// Check skills
	if plan.Tasks[0].Skill != "go-backend" {
		t.Errorf("Task 0 skill = %s, want go-backend", plan.Tasks[0].Skill)
	}

	if plan.Tasks[1].Skill != "ui-react" {
		t.Errorf("Task 1 skill = %s, want ui-react", plan.Tasks[1].Skill)
	}

	if plan.Tasks[2].Skill != "infra" {
		t.Errorf("Task 2 skill = %s, want infra", plan.Tasks[2].Skill)
	}

	// Check dependencies (P0 has none, P1 depends on P0, P2 depends on P0 and P1)
	if len(plan.Tasks[0].DependsOn) != 0 {
		t.Errorf("Task 0 dependencies = %d, want 0", len(plan.Tasks[0].DependsOn))
	}

	if len(plan.Tasks[1].DependsOn) != 1 {
		t.Errorf("Task 1 dependencies = %d, want 1", len(plan.Tasks[1].DependsOn))
	}

	if len(plan.Tasks[2].DependsOn) != 2 {
		t.Errorf("Task 2 dependencies = %d, want 2", len(plan.Tasks[2].DependsOn))
	}
}

func TestDetermineSkill(t *testing.T) {
	tests := []struct {
		name    string
		feature spec.Feature
		want    string
	}{
		{
			name: "API endpoint",
			feature: spec.Feature{
				API: []spec.API{{Path: "/api/users"}},
			},
			want: "go-backend",
		},
		{
			name: "UI component",
			feature: spec.Feature{
				Title: "User Interface Dashboard",
			},
			want: "ui-react",
		},
		{
			name: "Infrastructure",
			feature: spec.Feature{
				Desc: "Docker deployment configuration",
			},
			want: "infra",
		},
		{
			name: "Database",
			feature: spec.Feature{
				Title: "Database Schema Migration",
			},
			want: "database",
		},
		{
			name: "Testing",
			feature: spec.Feature{
				Desc: "Test validation framework",
			},
			want: "testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineSkill(tt.feature)
			if got != tt.want {
				t.Errorf("determineSkill() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDetermineModelHint(t *testing.T) {
	tests := []struct {
		name    string
		feature spec.Feature
		want    string
	}{
		{
			name: "many APIs",
			feature: spec.Feature{
				API: make([]spec.API, 6),
			},
			want: "long-context",
		},
		{
			name: "many success criteria",
			feature: spec.Feature{
				Success: make([]string, 6),
			},
			want: "agentic",
		},
		{
			name: "simple feature",
			feature: spec.Feature{
				API:     make([]spec.API, 2),
				Success: make([]string, 2),
			},
			want: "codegen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineModelHint(tt.feature)
			if got != tt.want {
				t.Errorf("determineModelHint() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPlanGeneratorWrappers(t *testing.T) {
	task := Task{
		ID:        types.TaskID("task-001"),
		FeatureID: types.FeatureID("feat-001"),
	}

	features := []spec.Feature{
		{
			ID:       types.FeatureID("feat-001"),
			Priority: types.Priority("P0"),
			Success:  []string{"done"},
			API: []spec.API{
				{Path: "/api/test", Method: "GET"},
			},
		},
	}

	_ = determineDependencies(features[0], features, 0)
	if skill := determineSkill(features[0]); skill == "" {
		t.Fatal("determineSkill returned empty string")
	}
	if hint := determineModelHint(features[0]); hint == "" {
		t.Fatal("determineModelHint returned empty string")
	}
	if complexity := estimateComplexity(features[0]); complexity <= 0 {
		t.Fatalf("expected complexity > 0, got %d", complexity)
	}

	tasks := []Task{
		task,
		{
			ID:        types.TaskID("task-002"),
			DependsOn: []types.TaskID{task.ID},
		},
	}

	if err := validateDependencies(tasks); err != nil {
		t.Fatalf("validateDependencies returned error: %v", err)
	}
}

func TestValidateDependencies(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []Task
		wantErr bool
	}{
		{
			name: "valid dependencies",
			tasks: []Task{
				{ID: types.TaskID("task-001"), DependsOn: []types.TaskID{}},
				{ID: types.TaskID("task-002"), DependsOn: []types.TaskID{types.TaskID("task-001")}},
			},
			wantErr: false,
		},
		{
			name: "non-existent dependency",
			tasks: []Task{
				{ID: types.TaskID("task-001"), DependsOn: []types.TaskID{types.TaskID("task-999")}},
			},
			wantErr: true,
		},
		{
			name: "forward dependency",
			tasks: []Task{
				{ID: types.TaskID("task-001"), DependsOn: []types.TaskID{types.TaskID("task-002")}},
				{ID: types.TaskID("task-002"), DependsOn: []types.TaskID{}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDependencies(tt.tasks)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
