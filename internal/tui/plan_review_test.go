package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

func TestRunPlanReview_EmptyPlan(t *testing.T) {
	emptyPlan := &plan.Plan{
		Tasks: []plan.Task{},
	}

	result, err := RunPlanReview(emptyPlan)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Approved {
		t.Error("Expected empty plan to be auto-approved")
	}

	if result.Reason != "" {
		t.Errorf("Expected empty reason, got: %s", result.Reason)
	}
}

func TestPlanReviewModel_Init(t *testing.T) {
	testPlan := createTestPlan()
	model := planReviewModel{
		plan:     testPlan,
		cursor:   0,
		viewMode: "list",
	}

	cmd := model.Init()
	if cmd != nil {
		t.Error("Expected Init to return nil cmd")
	}
}

func TestPlanReviewModel_Navigation(t *testing.T) {
	testPlan := createTestPlan()
	model := planReviewModel{
		plan:     testPlan,
		cursor:   0,
		viewMode: "list",
	}

	// Test down navigation
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m := updatedModel.(planReviewModel)
	if m.cursor != 1 {
		t.Errorf("Expected cursor at 1, got %d", m.cursor)
	}

	// Test up navigation
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updatedModel.(planReviewModel)
	if m.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", m.cursor)
	}

	// Test bounds - can't go below 0
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updatedModel.(planReviewModel)
	if m.cursor != 0 {
		t.Errorf("Expected cursor to stay at 0, got %d", m.cursor)
	}

	// Test bounds - can't exceed task count
	model.cursor = len(testPlan.Tasks) - 1
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updatedModel.(planReviewModel)
	if m.cursor != len(testPlan.Tasks)-1 {
		t.Errorf("Expected cursor to stay at max, got %d", m.cursor)
	}
}

func TestPlanReviewModel_ViewModes(t *testing.T) {
	testPlan := createTestPlan()
	model := planReviewModel{
		plan:     testPlan,
		cursor:   0,
		viewMode: "list",
	}

	if model.viewMode != "list" {
		t.Errorf("Expected initial view mode to be 'list', got %s", model.viewMode)
	}

	// Enter detail view
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := updatedModel.(planReviewModel)
	if m.viewMode != "detail" {
		t.Errorf("Expected view mode to be 'detail', got %s", m.viewMode)
	}

	// Return to list view
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updatedModel.(planReviewModel)
	if m.viewMode != "list" {
		t.Errorf("Expected view mode to be 'list', got %s", m.viewMode)
	}
}

func TestPlanReviewModel_ApproveReject(t *testing.T) {
	testPlan := createTestPlan()

	t.Run("approve", func(t *testing.T) {
		model := planReviewModel{
			plan:     testPlan,
			cursor:   0,
			viewMode: "list",
		}

		updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		m := updatedModel.(planReviewModel)

		if m.approved == nil || !*m.approved {
			t.Error("Expected plan to be approved")
		}

		if m.result == nil {
			t.Fatal("Expected result to be set")
		}

		if !m.result.Approved {
			t.Error("Expected result.Approved to be true")
		}

		if m.result.Reason != "" {
			t.Errorf("Expected empty reason, got: %s", m.result.Reason)
		}

		// Should return quit command
		if cmd == nil {
			t.Error("Expected quit command")
		}
	})

	t.Run("reject", func(t *testing.T) {
		model := planReviewModel{
			plan:     testPlan,
			cursor:   0,
			viewMode: "list",
		}

		// Press 'r' to reject
		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		m := updatedModel.(planReviewModel)

		if !m.editingReason {
			t.Error("Expected to be editing rejection reason")
		}

		// Type reason
		updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		m = updatedModel.(planReviewModel)
		updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		m = updatedModel.(planReviewModel)
		updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		m = updatedModel.(planReviewModel)
		updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		m = updatedModel.(planReviewModel)

		if m.rejectionInput != "test" {
			t.Errorf("Expected rejection input 'test', got: %s", m.rejectionInput)
		}

		// Press enter to confirm
		updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updatedModel.(planReviewModel)

		if m.editingReason {
			t.Error("Expected to stop editing reason")
		}
	})
}

func TestPlanReviewModel_RejectionReasonBackspace(t *testing.T) {
	testPlan := createTestPlan()
	model := planReviewModel{
		plan:           testPlan,
		cursor:         0,
		viewMode:       "list",
		editingReason:  true,
		rejectionInput: "test",
	}

	// Backspace
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m := updatedModel.(planReviewModel)

	if m.rejectionInput != "tes" {
		t.Errorf("Expected 'tes', got: %s", m.rejectionInput)
	}
}

func TestPlanReviewModel_CancelRejection(t *testing.T) {
	testPlan := createTestPlan()
	rejected := false
	model := planReviewModel{
		plan:           testPlan,
		cursor:         0,
		viewMode:       "list",
		editingReason:  true,
		rejectionInput: "test reason",
		approved:       &rejected,
	}

	// Press escape to cancel
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m := updatedModel.(planReviewModel)

	if m.editingReason {
		t.Error("Expected to stop editing reason")
	}

	if m.rejectionInput != "" {
		t.Errorf("Expected empty rejection input, got: %s", m.rejectionInput)
	}

	if m.approved != nil {
		t.Error("Expected approved to be nil after cancel")
	}
}

func TestPlanReviewModel_QuitWithoutDecision(t *testing.T) {
	testPlan := createTestPlan()
	model := planReviewModel{
		plan:     testPlan,
		cursor:   0,
		viewMode: "list",
	}

	// Press 'q' to quit
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := updatedModel.(planReviewModel)

	if m.result == nil {
		t.Fatal("Expected result to be set")
	}

	if m.result.Approved {
		t.Error("Expected plan to be rejected when quitting without decision")
	}

	if m.result.Reason != "Review cancelled" {
		t.Errorf("Expected reason 'Review cancelled', got: %s", m.result.Reason)
	}

	// Should return quit command
	if cmd == nil {
		t.Error("Expected quit command")
	}
}

func TestPlanReviewModel_View(t *testing.T) {
	testPlan := createTestPlan()

	t.Run("list view", func(t *testing.T) {
		model := planReviewModel{
			plan:     testPlan,
			cursor:   0,
			viewMode: "list",
		}

		view := model.View()
		if view == "" {
			t.Error("Expected non-empty view")
		}

		// Should contain title
		if !contains(view, "Plan Review") {
			t.Error("Expected view to contain 'Plan Review'")
		}

		// Should contain task count
		if !contains(view, "Total Tasks:") {
			t.Error("Expected view to contain 'Total Tasks:'")
		}
	})

	t.Run("detail view", func(t *testing.T) {
		model := planReviewModel{
			plan:         testPlan,
			cursor:       0,
			selectedTask: 0,
			viewMode:     "detail",
		}

		view := model.View()
		if view == "" {
			t.Error("Expected non-empty view")
		}

		// Should contain task details
		if !contains(view, "ID") {
			t.Error("Expected view to contain 'ID'")
		}

		if !contains(view, "Feature ID") {
			t.Error("Expected view to contain 'Feature ID'")
		}

		if !contains(view, "Skill") {
			t.Error("Expected view to contain 'Skill'")
		}
	})

	t.Run("approved result", func(t *testing.T) {
		model := planReviewModel{
			plan:     testPlan,
			cursor:   0,
			viewMode: "list",
			result: &PlanReviewResult{
				Approved: true,
				Reason:   "",
			},
		}

		view := model.View()
		if !contains(view, "Approved") {
			t.Error("Expected view to contain 'Approved'")
		}
	})

	t.Run("rejected result", func(t *testing.T) {
		model := planReviewModel{
			plan:     testPlan,
			cursor:   0,
			viewMode: "list",
			result: &PlanReviewResult{
				Approved: false,
				Reason:   "Test rejection",
			},
		}

		view := model.View()
		if !contains(view, "Rejected") {
			t.Error("Expected view to contain 'Rejected'")
		}

		if !contains(view, "Test rejection") {
			t.Error("Expected view to contain rejection reason")
		}
	})
}

// Helper functions

func createTestPlan() *plan.Plan {
	return &plan.Plan{
		Tasks: []plan.Task{
			{
				ID:           "task-1",
				FeatureID:    "feature-1",
				ExpectedHash: "hash1",
				DependsOn:    []types.TaskID{},
				Skill:        "go-backend",
				Priority:     "P0",
				ModelHint:    "codegen",
				Estimate:     5,
			},
			{
				ID:           "task-2",
				FeatureID:    "feature-2",
				ExpectedHash: "hash2",
				DependsOn:    []types.TaskID{"task-1"},
				Skill:        "ui-react",
				Priority:     "P1",
				ModelHint:    "agentic",
				Estimate:     3,
			},
			{
				ID:           "task-3",
				FeatureID:    "feature-3",
				ExpectedHash: "hash3",
				DependsOn:    []types.TaskID{"task-1", "task-2"},
				Skill:        "infra",
				Priority:     "P2",
				ModelHint:    "long-context",
				Estimate:     2,
			},
		},
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
