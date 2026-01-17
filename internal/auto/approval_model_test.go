package auto

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

func newApprovalModel() approvalModel {
	featureID, _ := types.NewFeatureID("feature-one")
	plan := &plan.Plan{
		Tasks: []plan.Task{
			{
				ID:        types.TaskID("task-one"),
				FeatureID: featureID,
				Priority:  types.PriorityP1,
				Skill:     "go-backend",
			},
		},
	}
	return approvalModel{
		plan: plan,
		spec: &spec.ProductSpec{
			Features: []spec.Feature{
				{ID: featureID, Title: "Feature One"},
			},
		},
		featureTitles: map[string]string{
			featureID.String(): "Feature One",
		},
	}
}

func TestApprovalModelUpdateHandlesKeys(t *testing.T) {
	model := newApprovalModel()
	if got := model.Init(); got != nil {
		t.Fatalf("Init() returned %T, expected nil", got)
	}

	yesMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updated, _ := model.Update(yesMsg)
	result := updated.(approvalModel)
	if !result.approved || !result.quitting {
		t.Fatalf("expected approved and quitting after 'y', got approved=%v quitting=%v", result.approved, result.quitting)
	}

	noModel := newApprovalModel()
	noMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updated, _ = noModel.Update(noMsg)
	result = updated.(approvalModel)
	if result.approved || !result.quitting {
		t.Fatalf("expected rejected plan after 'n', got approved=%v quitting=%v", result.approved, result.quitting)
	}
}

func TestApprovalModelViewFormats(t *testing.T) {
	model := newApprovalModel()
	view := model.View()
	if !strings.Contains(view, "Generated Execution Plan") {
		t.Fatalf("expected plan summary view, got %q", view)
	}

	approved := model
	approved.quitting = true
	approved.approved = true
	if !strings.Contains(approved.View(), "Plan approved") {
		t.Fatal("expected approved view to mention success")
	}

	rejected := model
	rejected.quitting = true
	rejected.approved = false
	if !strings.Contains(rejected.View(), "Plan rejected") {
		t.Fatal("expected rejected view to mention rejection")
	}
}
