package auto

import (
	"fmt"
	"testing"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
)

func TestParsePattern(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedType    PatternType
		expectedPattern string
	}{
		{
			name:            "feature ID",
			input:           "feature:feat-1",
			expectedType:    PatternTypeFeatureID,
			expectedPattern: "feat-1",
		},
		{
			name:            "feature title pattern",
			input:           "feature:user*",
			expectedType:    PatternTypeFeatureTitle,
			expectedPattern: "user*",
		},
		{
			name:            "path glob",
			input:           "src/components/**",
			expectedType:    PatternTypePath,
			expectedPattern: "src/components/**",
		},
		{
			name:            "tag pattern",
			input:           "@auth",
			expectedType:    PatternTypeTag,
			expectedPattern: "auth",
		},
		{
			name:            "simple path",
			input:           "api/users",
			expectedType:    PatternTypePath,
			expectedPattern: "api/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, err := parsePattern(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if pattern.Type != tt.expectedType {
				t.Errorf("expected type %q, got %q", tt.expectedType, pattern.Type)
			}

			if pattern.Pattern != tt.expectedPattern {
				t.Errorf("expected pattern %q, got %q", tt.expectedPattern, pattern.Pattern)
			}
		})
	}
}

func TestNewScope(t *testing.T) {
	t.Run("creates scope with multiple patterns", func(t *testing.T) {
		patterns := []string{"feature:feat-1", "src/**", "@auth"}
		scope, err := NewScope(patterns, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(scope.Patterns) != 3 {
			t.Errorf("expected 3 patterns, got %d", len(scope.Patterns))
		}

		if !scope.IncludeDependencies {
			t.Error("expected IncludeDependencies to be true")
		}
	})

	t.Run("creates empty scope", func(t *testing.T) {
		scope, err := NewScope([]string{}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(scope.Patterns) != 0 {
			t.Errorf("expected 0 patterns, got %d", len(scope.Patterns))
		}
	})
}

func TestMatchesFeature(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		feature  spec.Feature
		expected bool
	}{
		{
			name:     "matches feature ID",
			patterns: []string{"feature:feat-1"},
			feature: spec.Feature{
				ID:    "feat-1",
				Title: "User Authentication",
			},
			expected: true,
		},
		{
			name:     "does not match feature ID",
			patterns: []string{"feature:feat-2"},
			feature: spec.Feature{
				ID:    "feat-1",
				Title: "User Authentication",
			},
			expected: false,
		},
		{
			name:     "matches feature title pattern",
			patterns: []string{"feature:User*"},
			feature: spec.Feature{
				ID:    "feat-1",
				Title: "User Authentication",
			},
			expected: true,
		},
		{
			name:     "matches API path pattern",
			patterns: []string{"/api/users*"},
			feature: spec.Feature{
				ID:    "feat-1",
				Title: "User Management",
				API: []spec.API{
					{Method: "GET", Path: "/api/users"},
					{Method: "POST", Path: "/api/users"},
				},
			},
			expected: true,
		},
		{
			name:     "does not match API path pattern",
			patterns: []string{"/api/posts*"},
			feature: spec.Feature{
				ID:    "feat-1",
				Title: "User Management",
				API: []spec.API{
					{Method: "GET", Path: "/api/users"},
				},
			},
			expected: false,
		},
		{
			name:     "matches with multiple patterns (OR logic)",
			patterns: []string{"feature:feat-2", "/api/users*"},
			feature: spec.Feature{
				ID:    "feat-1",
				Title: "User Management",
				API: []spec.API{
					{Method: "GET", Path: "/api/users"},
				},
			},
			expected: true,
		},
		{
			name:     "empty scope matches all",
			patterns: []string{},
			feature: spec.Feature{
				ID:    "feat-1",
				Title: "Any Feature",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, err := NewScope(tt.patterns, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result := scope.MatchesFeature(tt.feature)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMatchesTask(t *testing.T) {
	productSpec := &spec.ProductSpec{
		Features: []spec.Feature{
			{
				ID:    "feat-1",
				Title: "User Authentication",
				API: []spec.API{
					{Method: "POST", Path: "/api/auth/login"},
				},
			},
			{
				ID:    "feat-2",
				Title: "User Profile",
				API: []spec.API{
					{Method: "GET", Path: "/api/users/profile"},
				},
			},
		},
	}

	tests := []struct {
		name     string
		patterns []string
		task     plan.Task
		expected bool
	}{
		{
			name:     "matches task by feature ID",
			patterns: []string{"feature:feat-1"},
			task: plan.Task{
				ID:        "task-1",
				FeatureID: "feat-1",
			},
			expected: true,
		},
		{
			name:     "does not match task by feature ID",
			patterns: []string{"feature:feat-2"},
			task: plan.Task{
				ID:        "task-1",
				FeatureID: "feat-1",
			},
			expected: false,
		},
		{
			name:     "matches task by API path",
			patterns: []string{"/api/auth/*"},
			task: plan.Task{
				ID:        "task-1",
				FeatureID: "feat-1",
			},
			expected: true,
		},
		{
			name:     "empty scope matches all tasks",
			patterns: []string{},
			task: plan.Task{
				ID:        "task-1",
				FeatureID: "feat-1",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, err := NewScope(tt.patterns, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result := scope.MatchesTask(tt.task, productSpec)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFilterPlan(t *testing.T) {
	productSpec := &spec.ProductSpec{
		Features: []spec.Feature{
			{
				ID:    "feat-1",
				Title: "User Authentication",
			},
			{
				ID:    "feat-2",
				Title: "User Profile",
			},
			{
				ID:    "feat-3",
				Title: "User Settings",
			},
		},
	}

	t.Run("filters plan without dependencies", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", FeatureID: "feat-1"},
				{ID: "task-2", FeatureID: "feat-2"},
				{ID: "task-3", FeatureID: "feat-3"},
			},
		}

		scope, _ := NewScope([]string{"feature:feat-1"}, false)
		filtered := scope.FilterPlan(execPlan, productSpec)

		if len(filtered.Tasks) != 1 {
			t.Errorf("expected 1 task, got %d", len(filtered.Tasks))
		}

		if filtered.Tasks[0].ID != "task-1" {
			t.Errorf("expected task-1, got %s", filtered.Tasks[0].ID)
		}
	})

	t.Run("filters plan with dependencies", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", FeatureID: "feat-1", DependsOn: []types.TaskID{}},
				{ID: "task-2", FeatureID: "feat-2", DependsOn: []types.TaskID{"task-1"}},
				{ID: "task-3", FeatureID: "feat-3", DependsOn: []types.TaskID{"task-2"}},
			},
		}

		// Filter for feat-2, but include dependencies
		scope, _ := NewScope([]string{"feature:feat-2"}, true)
		filtered := scope.FilterPlan(execPlan, productSpec)

		// Should include task-2 (matched) and task-1 (dependency)
		if len(filtered.Tasks) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(filtered.Tasks))
		}

		taskIDs := make(map[types.TaskID]bool)
		for _, task := range filtered.Tasks {
			taskIDs[task.ID] = true
		}

		if !taskIDs["task-1"] {
			t.Error("expected task-1 to be included (dependency)")
		}
		if !taskIDs["task-2"] {
			t.Error("expected task-2 to be included (matched)")
		}
		if taskIDs["task-3"] {
			t.Error("did not expect task-3 to be included")
		}
	})

	t.Run("filters plan with multiple dependencies", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", FeatureID: "feat-1", DependsOn: []types.TaskID{}},
				{ID: "task-2", FeatureID: "feat-1", DependsOn: []types.TaskID{}},
				{ID: "task-3", FeatureID: "feat-2", DependsOn: []types.TaskID{"task-1", "task-2"}},
				{ID: "task-4", FeatureID: "feat-3", DependsOn: []types.TaskID{}},
			},
		}

		// Filter for feat-2 with dependencies
		scope, _ := NewScope([]string{"feature:feat-2"}, true)
		filtered := scope.FilterPlan(execPlan, productSpec)

		// Should include task-3 (matched) and task-1, task-2 (dependencies)
		if len(filtered.Tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(filtered.Tasks))
		}

		taskIDs := make(map[types.TaskID]bool)
		for _, task := range filtered.Tasks {
			taskIDs[task.ID] = true
		}

		if !taskIDs["task-1"] {
			t.Error("expected task-1 to be included (dependency)")
		}
		if !taskIDs["task-2"] {
			t.Error("expected task-2 to be included (dependency)")
		}
		if !taskIDs["task-3"] {
			t.Error("expected task-3 to be included (matched)")
		}
		if taskIDs["task-4"] {
			t.Error("did not expect task-4 to be included")
		}
	})

	t.Run("empty scope returns all tasks", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", FeatureID: "feat-1"},
				{ID: "task-2", FeatureID: "feat-2"},
				{ID: "task-3", FeatureID: "feat-3"},
			},
		}

		scope, _ := NewScope([]string{}, false)
		filtered := scope.FilterPlan(execPlan, productSpec)

		if len(filtered.Tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(filtered.Tasks))
		}
	})

	t.Run("filters with multiple patterns (OR logic)", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", FeatureID: "feat-1"},
				{ID: "task-2", FeatureID: "feat-2"},
				{ID: "task-3", FeatureID: "feat-3"},
			},
		}

		scope, _ := NewScope([]string{"feature:feat-1", "feature:feat-3"}, false)
		filtered := scope.FilterPlan(execPlan, productSpec)

		if len(filtered.Tasks) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(filtered.Tasks))
		}

		taskIDs := make(map[types.TaskID]bool)
		for _, task := range filtered.Tasks {
			taskIDs[task.ID] = true
		}

		if !taskIDs["task-1"] || !taskIDs["task-3"] {
			t.Error("expected task-1 and task-3 to be included")
		}
	})
}

func TestExpandDependencies(t *testing.T) {
	t.Run("expands single level dependencies", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", DependsOn: []types.TaskID{}},
				{ID: "task-2", DependsOn: []types.TaskID{"task-1"}},
			},
		}

		scope := &Scope{}
		matched := map[types.TaskID]bool{"task-2": true}
		expanded := scope.expandDependencies(execPlan, matched)

		if !expanded["task-1"] {
			t.Error("expected task-1 to be included as dependency")
		}
		if !expanded["task-2"] {
			t.Error("expected task-2 to remain included")
		}
	})

	t.Run("expands multi-level dependencies", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", DependsOn: []types.TaskID{}},
				{ID: "task-2", DependsOn: []types.TaskID{"task-1"}},
				{ID: "task-3", DependsOn: []types.TaskID{"task-2"}},
			},
		}

		scope := &Scope{}
		matched := map[types.TaskID]bool{"task-3": true}
		expanded := scope.expandDependencies(execPlan, matched)

		if len(expanded) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(expanded))
		}

		if !expanded["task-1"] || !expanded["task-2"] || !expanded["task-3"] {
			t.Error("expected all tasks to be included in dependency chain")
		}
	})

	t.Run("handles diamond dependencies", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", DependsOn: []types.TaskID{}},
				{ID: "task-2", DependsOn: []types.TaskID{"task-1"}},
				{ID: "task-3", DependsOn: []types.TaskID{"task-1"}},
				{ID: "task-4", DependsOn: []types.TaskID{"task-2", "task-3"}},
			},
		}

		scope := &Scope{}
		matched := map[types.TaskID]bool{"task-4": true}
		expanded := scope.expandDependencies(execPlan, matched)

		if len(expanded) != 4 {
			t.Errorf("expected 4 tasks, got %d", len(expanded))
		}

		for i := 1; i <= 4; i++ {
			taskID := types.TaskID(fmt.Sprintf("task-%d", i))
			if !expanded[taskID] {
				t.Errorf("expected %s to be included", taskID)
			}
		}
	})
}

func TestScopeSummary(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []string
		includeDeps bool
		expected    string
	}{
		{
			name:        "empty scope",
			patterns:    []string{},
			includeDeps: false,
			expected:    "all features",
		},
		{
			name:        "single feature pattern",
			patterns:    []string{"feature:feat-1"},
			includeDeps: false,
			expected:    "feature:feat-1",
		},
		{
			name:        "multiple patterns",
			patterns:    []string{"feature:feat-1", "/api/users*"},
			includeDeps: false,
			expected:    "feature:feat-1, path:/api/users*",
		},
		{
			name:        "with dependencies",
			patterns:    []string{"feature:feat-1"},
			includeDeps: true,
			expected:    "feature:feat-1 (with dependencies)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, _ := NewScope(tt.patterns, tt.includeDeps)
			summary := scope.Summary()

			if summary != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, summary)
			}
		})
	}
}

func TestEstimateImpact(t *testing.T) {
	productSpec := &spec.ProductSpec{
		Features: []spec.Feature{
			{ID: "feat-1", Title: "Feature 1"},
			{ID: "feat-2", Title: "Feature 2"},
			{ID: "feat-3", Title: "Feature 3"},
		},
	}

	t.Run("estimates impact without dependencies", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", FeatureID: "feat-1"},
				{ID: "task-2", FeatureID: "feat-2"},
				{ID: "task-3", FeatureID: "feat-3"},
			},
		}

		scope, _ := NewScope([]string{"feature:feat-1"}, false)
		matched, total := scope.EstimateImpact(execPlan, productSpec)

		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if matched != 1 {
			t.Errorf("expected matched 1, got %d", matched)
		}
	})

	t.Run("estimates impact with dependencies", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", FeatureID: "feat-1", DependsOn: []types.TaskID{}},
				{ID: "task-2", FeatureID: "feat-2", DependsOn: []types.TaskID{"task-1"}},
				{ID: "task-3", FeatureID: "feat-3", DependsOn: []types.TaskID{}},
			},
		}

		scope, _ := NewScope([]string{"feature:feat-2"}, true)
		matched, total := scope.EstimateImpact(execPlan, productSpec)

		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		// Should match task-2 and its dependency task-1
		if matched != 2 {
			t.Errorf("expected matched 2, got %d", matched)
		}
	})

	t.Run("empty scope matches all", func(t *testing.T) {
		execPlan := &plan.Plan{
			Tasks: []plan.Task{
				{ID: "task-1", FeatureID: "feat-1"},
				{ID: "task-2", FeatureID: "feat-2"},
				{ID: "task-3", FeatureID: "feat-3"},
			},
		}

		scope, _ := NewScope([]string{}, false)
		matched, total := scope.EstimateImpact(execPlan, productSpec)

		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if matched != 3 {
			t.Errorf("expected matched 3, got %d", matched)
		}
	})
}
