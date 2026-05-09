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
	if result.decision != ApprovalApproved || !result.quitting {
		t.Fatalf("expected approved+quitting after 'y', got decision=%v quitting=%v", result.decision, result.quitting)
	}

	noModel := newApprovalModel()
	noMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updated, _ = noModel.Update(noMsg)
	result = updated.(approvalModel)
	if result.decision != ApprovalRejected || !result.quitting {
		t.Fatalf("expected rejected after 'n', got decision=%v quitting=%v", result.decision, result.quitting)
	}

	// Esc must cancel without recording rejection — distinct from N.
	escModel := newApprovalModel()
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ = escModel.Update(escMsg)
	result = updated.(approvalModel)
	if result.decision != ApprovalCancelled || !result.quitting {
		t.Fatalf("expected cancelled after Esc, got decision=%v quitting=%v", result.decision, result.quitting)
	}

	// Enter must take the documented default (approve).
	enterModel := newApprovalModel()
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ = enterModel.Update(enterMsg)
	result = updated.(approvalModel)
	if result.decision != ApprovalApproved || !result.quitting {
		t.Fatalf("expected approved after Enter, got decision=%v quitting=%v", result.decision, result.quitting)
	}

	// ? must toggle help without quitting.
	helpModel := newApprovalModel()
	helpMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	updated, _ = helpModel.Update(helpMsg)
	result = updated.(approvalModel)
	if result.quitting {
		t.Fatal("? must not quit the gate")
	}
	if !result.showHelp {
		t.Fatal("? must toggle showHelp on first press")
	}
}

func TestApprovalModelViewFormats(t *testing.T) {
	model := newApprovalModel()
	view := model.View()
	if !strings.Contains(view, "Generated Execution Plan") {
		t.Fatalf("expected plan summary view, got %q", view)
	}
	if !strings.Contains(view, "y approve") || !strings.Contains(view, "esc cancel") {
		t.Fatalf("expected default footer hint with y/esc keys, got %q", view)
	}

	approved := model
	approved.quitting = true
	approved.decision = ApprovalApproved
	if !strings.Contains(approved.View(), "Plan approved") {
		t.Fatal("expected approved view to mention success")
	}

	rejected := model
	rejected.quitting = true
	rejected.decision = ApprovalRejected
	if !strings.Contains(rejected.View(), "Plan rejected") {
		t.Fatal("expected rejected view to mention rejection")
	}

	cancelled := model
	cancelled.quitting = true
	cancelled.decision = ApprovalCancelled
	if !strings.Contains(cancelled.View(), "cancelled") {
		t.Fatal("expected cancelled view to mention cancellation")
	}
}
