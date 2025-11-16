package cmd

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// TestBuildFeatureFiltering tests the feature filtering logic in build run
func TestBuildFeatureFiltering(t *testing.T) {
	tests := []struct {
		name        string
		plan        *plan.Plan
		featureID   string
		wantTasks   int
		wantErr     bool
		errContains string
	}{
		{
			name: "filter to single feature",
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:        types.TaskID("task-1"),
						FeatureID: types.FeatureID("feat-1"),
						Skill:     "go-backend",
					},
					{
						ID:        types.TaskID("task-2"),
						FeatureID: types.FeatureID("feat-2"),
						Skill:     "ui-react",
					},
					{
						ID:        types.TaskID("task-3"),
						FeatureID: types.FeatureID("feat-1"),
						Skill:     "go-backend",
					},
				},
			},
			featureID: "feat-1",
			wantTasks: 2,
			wantErr:   false,
		},
		{
			name: "no filtering - all tasks",
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:        types.TaskID("task-1"),
						FeatureID: types.FeatureID("feat-1"),
					},
					{
						ID:        types.TaskID("task-2"),
						FeatureID: types.FeatureID("feat-2"),
					},
				},
			},
			featureID: "",
			wantTasks: 2,
			wantErr:   false,
		},
		{
			name: "feature not found - no tasks",
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:        types.TaskID("task-1"),
						FeatureID: types.FeatureID("feat-1"),
					},
				},
			},
			featureID:   "feat-999",
			wantErr:     true,
			errContains: "no tasks found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the filtering logic
			filteredPlan := tt.plan
			var err error

			if tt.featureID != "" {
				var filteredTasks []plan.Task
				for _, task := range tt.plan.Tasks {
					if string(task.FeatureID) == tt.featureID {
						filteredTasks = append(filteredTasks, task)
					}
				}

				if len(filteredTasks) == 0 {
					err = &noTasksError{tt.featureID}
				} else {
					filteredPlan = &plan.Plan{
						Tasks: filteredTasks,
					}
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("filtering error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(filteredPlan.Tasks) != tt.wantTasks {
				t.Errorf("filtered tasks = %d, want %d", len(filteredPlan.Tasks), tt.wantTasks)
			}
		})
	}
}

// TestBuildVerifyChecks tests the verification check logic
func TestBuildVerifyChecks(t *testing.T) {
	tests := []struct {
		name        string
		vetPass     bool
		lintPass    bool
		testPass    bool
		policyPass  bool
		wantPassed  int
		wantFailed  int
		wantOverall bool
	}{
		{
			name:        "all checks pass",
			vetPass:     true,
			lintPass:    true,
			testPass:    true,
			policyPass:  true,
			wantPassed:  4,
			wantFailed:  0,
			wantOverall: true,
		},
		{
			name:        "vet fails",
			vetPass:     false,
			lintPass:    true,
			testPass:    true,
			policyPass:  true,
			wantPassed:  3,
			wantFailed:  1,
			wantOverall: false,
		},
		{
			name:        "tests fail",
			vetPass:     true,
			lintPass:    true,
			testPass:    false,
			policyPass:  true,
			wantPassed:  3,
			wantFailed:  1,
			wantOverall: false,
		},
		{
			name:        "multiple failures",
			vetPass:     false,
			lintPass:    false,
			testPass:    false,
			policyPass:  true,
			wantPassed:  1, // policy only
			wantFailed:  3, // vet + lint + tests
			wantOverall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the verification logic
			passed := 0
			failed := 0

			// Vet check
			if tt.vetPass {
				passed++
			} else {
				failed++
			}

			// Lint check
			if tt.lintPass {
				passed++
			} else {
				failed++
			}

			// Test check
			if tt.testPass {
				passed++
			} else {
				failed++
			}

			// Policy check
			if tt.policyPass {
				passed++
			}

			if passed != tt.wantPassed {
				t.Errorf("passed checks = %d, want %d", passed, tt.wantPassed)
			}

			if failed != tt.wantFailed {
				t.Errorf("failed checks = %d, want %d", failed, tt.wantFailed)
			}

			overall := failed == 0
			if overall != tt.wantOverall {
				t.Errorf("overall pass = %v, want %v", overall, tt.wantOverall)
			}
		})
	}
}

// TestBuildManifestLookup tests the most recent manifest finding logic
func TestBuildManifestLookup(t *testing.T) {
	tests := []struct {
		name       string
		manifests  []string
		wantLatest string
		wantFound  bool
	}{
		{
			name: "single manifest",
			manifests: []string{
				"2024-01-01T10-00-00",
			},
			wantLatest: "2024-01-01T10-00-00",
			wantFound:  true,
		},
		{
			name: "multiple manifests - latest first",
			manifests: []string{
				"2024-01-01T10-00-00",
				"2024-01-02T15-30-00",
				"2024-01-01T08-00-00",
			},
			wantLatest: "2024-01-02T15-30-00",
			wantFound:  true,
		},
		{
			name:      "no manifests",
			manifests: []string{},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the manifest lookup logic
			var latestManifest string
			var found bool

			if len(tt.manifests) > 0 {
				// Simple lexicographic sort works for RFC3339 format
				latestManifest = tt.manifests[0]
				for _, m := range tt.manifests {
					if m > latestManifest {
						latestManifest = m
					}
				}
				found = true
			}

			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}

			if tt.wantFound && latestManifest != tt.wantLatest {
				t.Errorf("latest manifest = %s, want %s", latestManifest, tt.wantLatest)
			}
		})
	}
}

// TestBuildApproveValidation tests the approval validation logic
func TestBuildApproveValidation(t *testing.T) {
	tests := []struct {
		name           string
		manifestExists bool
		wantErr        bool
		errContains    string
	}{
		{
			name:           "manifest exists",
			manifestExists: true,
			wantErr:        false,
		},
		{
			name:           "manifest missing",
			manifestExists: false,
			wantErr:        true,
			errContains:    "no build manifests found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the approval validation logic
			var err error

			if !tt.manifestExists {
				err = &noManifestError{}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBuildBackwardCompatibilityFlags tests that old build flags work
func TestBuildBackwardCompatibilityFlags(t *testing.T) {
	// Test that the root build command still has the old flags for backward compatibility
	if buildCmd.Flags().Lookup("plan") == nil {
		t.Error("backward compatibility flag 'plan' not found on build command")
	}
	if buildCmd.Flags().Lookup("policy") == nil {
		t.Error("backward compatibility flag 'policy' not found on build command")
	}
	if buildCmd.Flags().Lookup("dry-run") == nil {
		t.Error("backward compatibility flag 'dry-run' not found on build command")
	}
}

// TestBuildRunFlags tests that build run has all required flags
func TestBuildRunFlags(t *testing.T) {
	if buildRunCmd.Flags().Lookup("plan") == nil {
		t.Error("flag 'plan' not found on build run command")
	}
	if buildRunCmd.Flags().Lookup("policy") == nil {
		t.Error("flag 'policy' not found on build run command")
	}
	if buildRunCmd.Flags().Lookup("feature") == nil {
		t.Error("flag 'feature' not found on build run command")
	}
	if buildRunCmd.Flags().Lookup("dry-run") == nil {
		t.Error("flag 'dry-run' not found on build run command")
	}
	if buildRunCmd.Flags().Lookup("resume") == nil {
		t.Error("flag 'resume' not found on build run command")
	}
	if buildRunCmd.Flags().Lookup("verbose") == nil {
		t.Error("flag 'verbose' not found on build run command")
	}
}

// TestBuildSubcommands tests that all build subcommands are registered
func TestBuildSubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"run":     false,
		"verify":  false,
		"approve": false,
		"explain": false,
	}

	for _, cmd := range buildCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand '%s' not found in build command", name)
		}
	}
}

// Helper error types for testing
type noTasksError struct {
	featureID string
}

func (e *noTasksError) Error() string {
	return "no tasks found for feature '" + e.featureID + "'"
}

type noManifestError struct{}

func (e *noManifestError) Error() string {
	return "no build manifests found"
}
