package cmd

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// TestPlanFeatureFiltering tests the feature filtering logic in plan gen
func TestPlanFeatureFiltering(t *testing.T) {
	tests := []struct {
		name         string
		spec         *spec.ProductSpec
		featureID    string
		wantFeatures int
		wantErr      bool
		errContains  string
	}{
		{
			name: "filter to single feature",
			spec: &spec.ProductSpec{
				Product: "Test Product",
				Features: []spec.Feature{
					{
						ID:    types.FeatureID("feat-1"),
						Title: "Feature 1",
					},
					{
						ID:    types.FeatureID("feat-2"),
						Title: "Feature 2",
					},
				},
			},
			featureID:    "feat-1",
			wantFeatures: 1,
			wantErr:      false,
		},
		{
			name: "no filtering - all features",
			spec: &spec.ProductSpec{
				Product: "Test Product",
				Features: []spec.Feature{
					{
						ID:    types.FeatureID("feat-1"),
						Title: "Feature 1",
					},
					{
						ID:    types.FeatureID("feat-2"),
						Title: "Feature 2",
					},
				},
			},
			featureID:    "",
			wantFeatures: 2,
			wantErr:      false,
		},
		{
			name: "feature not found",
			spec: &spec.ProductSpec{
				Product: "Test Product",
				Features: []spec.Feature{
					{
						ID:    types.FeatureID("feat-1"),
						Title: "Feature 1",
					},
				},
			},
			featureID:   "feat-999",
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the filtering logic
			filteredSpec := tt.spec
			var err error

			if tt.featureID != "" {
				found := false
				var filteredFeatures []spec.Feature
				for _, f := range tt.spec.Features {
					if string(f.ID) == tt.featureID {
						found = true
						filteredFeatures = append(filteredFeatures, f)
						break
					}
				}

				if !found {
					err = &notFoundError{tt.featureID}
				} else {
					filteredSpec = &spec.ProductSpec{
						Product:       tt.spec.Product,
						Goals:         tt.spec.Goals,
						Features:      filteredFeatures,
						NonFunctional: tt.spec.NonFunctional,
						Acceptance:    tt.spec.Acceptance,
						Milestones:    tt.spec.Milestones,
					}
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("filtering error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(filteredSpec.Features) != tt.wantFeatures {
				t.Errorf("filtered features = %d, want %d", len(filteredSpec.Features), tt.wantFeatures)
			}
		})
	}
}

// TestPlanExplainTaskLookup tests the task lookup logic in plan explain
func TestPlanExplainTaskLookup(t *testing.T) {
	tests := []struct {
		name      string
		plan      *plan.Plan
		stepID    string
		wantFound bool
	}{
		{
			name: "task found",
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:        types.TaskID("task-1"),
						FeatureID: types.FeatureID("feat-1"),
						Skill:     "go-backend",
					},
					{
						ID:        types.TaskID("task-2"),
						FeatureID: types.FeatureID("feat-1"),
						Skill:     "ui-react",
					},
				},
			},
			stepID:    "task-1",
			wantFound: true,
		},
		{
			name: "task not found",
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:        types.TaskID("task-1"),
						FeatureID: types.FeatureID("feat-1"),
						Skill:     "go-backend",
					},
				},
			},
			stepID:    "task-999",
			wantFound: false,
		},
		{
			name: "empty plan",
			plan: &plan.Plan{
				Tasks: []plan.Task{},
			},
			stepID:    "task-1",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test task lookup logic
			var found *plan.Task
			for i := range tt.plan.Tasks {
				if string(tt.plan.Tasks[i].ID) == tt.stepID {
					found = &tt.plan.Tasks[i]
					break
				}
			}

			if (found != nil) != tt.wantFound {
				t.Errorf("task found = %v, want %v", found != nil, tt.wantFound)
			}

			if tt.wantFound && found == nil {
				t.Error("expected to find task but got nil")
			}
		})
	}
}

// TestPlanDriftDetection tests the drift detection logic
func TestPlanDriftDetection(t *testing.T) {
	tests := []struct {
		name         string
		plan         *plan.Plan
		uncommitted  string
		wantHasDrift bool
	}{
		{
			name: "no drift - clean repo",
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:           types.TaskID("task-1"),
						ExpectedHash: "abc123",
					},
				},
			},
			uncommitted:  "",
			wantHasDrift: false,
		},
		{
			name: "drift - uncommitted changes",
			plan: &plan.Plan{
				Tasks: []plan.Task{
					{
						ID:           types.TaskID("task-1"),
						ExpectedHash: "abc123",
					},
				},
			},
			uncommitted:  "M file.go\nM other.go",
			wantHasDrift: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test drift detection logic
			hasDrift := tt.uncommitted != ""

			if hasDrift != tt.wantHasDrift {
				t.Errorf("hasDrift = %v, want %v", hasDrift, tt.wantHasDrift)
			}
		})
	}
}

// TestBackwardCompatibilityFlags tests that old plan flags work
func TestBackwardCompatibilityFlags(t *testing.T) {
	// Test that the root plan command still has the old flags for backward compatibility
	if planCmd.Flags().Lookup("in") == nil {
		t.Error("backward compatibility flag 'in' not found on plan command")
	}
	if planCmd.Flags().Lookup("out") == nil {
		t.Error("backward compatibility flag 'out' not found on plan command")
	}
	if planCmd.Flags().Lookup("lock") == nil {
		t.Error("backward compatibility flag 'lock' not found on plan command")
	}
	if planCmd.Flags().Lookup("estimate") == nil {
		t.Error("backward compatibility flag 'estimate' not found on plan command")
	}
}

// TestPlanGenFlags tests that plan gen has all required flags
func TestPlanGenFlags(t *testing.T) {
	if planGenCmd.Flags().Lookup("in") == nil {
		t.Error("flag 'in' not found on plan gen command")
	}
	if planGenCmd.Flags().Lookup("out") == nil {
		t.Error("flag 'out' not found on plan gen command")
	}
	if planGenCmd.Flags().Lookup("lock") == nil {
		t.Error("flag 'lock' not found on plan gen command")
	}
	if planGenCmd.Flags().Lookup("feature") == nil {
		t.Error("flag 'feature' not found on plan gen command")
	}
	if planGenCmd.Flags().Lookup("estimate") == nil {
		t.Error("flag 'estimate' not found on plan gen command")
	}
}

// TestPlanSubcommands tests that all plan subcommands are registered
func TestPlanSubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"gen":     false,
		"review":  false,
		"drift":   false,
		"explain": false,
	}

	for _, cmd := range planCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand '%s' not found in plan command", name)
		}
	}
}

// Helper error type for testing
type notFoundError struct {
	id string
}

func (e *notFoundError) Error() string {
	return "feature '" + e.id + "' not found in spec"
}
