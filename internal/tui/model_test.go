package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/felixgeelhaar/specular/internal/auto"
)

// TestNewModel tests model initialization
func TestNewModel(t *testing.T) {
	model := NewModel("Test goal", "default")

	if model.goal != "Test goal" {
		t.Errorf("Expected goal 'Test goal', got '%s'", model.goal)
	}

	if model.profile != "default" {
		t.Errorf("Expected profile 'default', got '%s'", model.profile)
	}

	if model.currentView != ViewMain {
		t.Errorf("Expected ViewMain, got %v", model.currentView)
	}

	if model.verboseMode {
		t.Error("Expected verbose mode to be false by default")
	}

	if model.quitting {
		t.Error("Expected quitting to be false by default")
	}
}

// TestSetActionPlan tests setting the action plan
func TestSetActionPlan(t *testing.T) {
	model := NewModel("Test goal", "default")

	actionPlan := auto.CreateDefaultActionPlan("Test goal", "default")
	model.SetActionPlan(actionPlan)

	if model.actionPlan == nil {
		t.Fatal("Expected action plan to be set")
	}

	if model.totalSteps != len(actionPlan.Steps) {
		t.Errorf("Expected totalSteps %d, got %d", len(actionPlan.Steps), model.totalSteps)
	}
}

// TestStepStartMessage tests step start message handling
func TestStepStartMessage(t *testing.T) {
	model := NewModel("Test goal", "default")

	msg := StepStartMsg{
		StepIndex: 1,
		StepName:  "Generate Specification",
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	if m.currentStep != 1 {
		t.Errorf("Expected currentStep 1, got %d", m.currentStep)
	}

	if m.currentStepName != "Generate Specification" {
		t.Errorf("Expected currentStepName 'Generate Specification', got '%s'", m.currentStepName)
	}
}

// TestStepCompleteMessage tests step complete message handling
func TestStepCompleteMessage(t *testing.T) {
	model := NewModel("Test goal", "default")
	model.totalSteps = 4

	msg := StepCompleteMsg{
		StepIndex: 1,
		StepName:  "Generate Specification",
		TotalCost: 0.05,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	if m.completedSteps != 1 {
		t.Errorf("Expected completedSteps 1, got %d", m.completedSteps)
	}

	if m.totalCost != 0.05 {
		t.Errorf("Expected totalCost 0.05, got %.2f", m.totalCost)
	}
}

// TestStepFailMessage tests step fail message handling
func TestStepFailMessage(t *testing.T) {
	model := NewModel("Test goal", "default")

	msg := StepFailMsg{
		StepIndex: 2,
		StepName:  "Generate Plan",
		Error:     "connection timeout",
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	if m.failedSteps != 1 {
		t.Errorf("Expected failedSteps 1, got %d", m.failedSteps)
	}

	if m.lastError != "connection timeout" {
		t.Errorf("Expected lastError 'connection timeout', got '%s'", m.lastError)
	}
}

// TestApprovalRequestMessage tests approval request handling
func TestApprovalRequestMessage(t *testing.T) {
	model := NewModel("Test goal", "default")

	msg := ApprovalRequestMsg{
		PlanSummary: "Plan with 5 tasks",
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	if !m.approvalPending {
		t.Error("Expected approvalPending to be true")
	}

	if m.currentView != ViewApproval {
		t.Errorf("Expected ViewApproval, got %v", m.currentView)
	}

	if !m.awaitingInput {
		t.Error("Expected awaitingInput to be true")
	}

	if m.approvalPlan != "Plan with 5 tasks" {
		t.Errorf("Expected approvalPlan 'Plan with 5 tasks', got '%s'", m.approvalPlan)
	}
}

// TestWorkflowCompleteMessage tests workflow completion
func TestWorkflowCompleteMessage(t *testing.T) {
	model := NewModel("Test goal", "default")

	msg := WorkflowCompleteMsg{
		Success:   true,
		TotalCost: 1.50,
		Duration:  5 * time.Minute,
	}

	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	if !m.quitting {
		t.Error("Expected quitting to be true")
	}

	// Verify that quit command is returned
	if cmd == nil {
		t.Error("Expected quit command to be returned")
	}
}

// TestKeyPressToggleHelp tests '?' key to toggle help
func TestKeyPressToggleHelp(t *testing.T) {
	model := NewModel("Test goal", "default")
	model.ready = true

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}

	// Toggle help on
	updatedModel, _ := model.Update(keyMsg)
	m := updatedModel.(Model)

	if m.currentView != ViewHelp {
		t.Errorf("Expected ViewHelp, got %v", m.currentView)
	}

	// Toggle help off
	updatedModel, _ = m.Update(keyMsg)
	m = updatedModel.(Model)

	if m.currentView != ViewMain {
		t.Errorf("Expected ViewMain, got %v", m.currentView)
	}
}

// TestKeyPressToggleStepList tests 's' key to toggle step list
func TestKeyPressToggleStepList(t *testing.T) {
	model := NewModel("Test goal", "default")
	model.ready = true

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}

	// Toggle step list on
	updatedModel, _ := model.Update(keyMsg)
	m := updatedModel.(Model)

	if m.currentView != ViewStepList {
		t.Errorf("Expected ViewStepList, got %v", m.currentView)
	}

	// Toggle step list off
	updatedModel, _ = m.Update(keyMsg)
	m = updatedModel.(Model)

	if m.currentView != ViewMain {
		t.Errorf("Expected ViewMain, got %v", m.currentView)
	}
}

// TestKeyPressToggleVerbose tests 'v' key to toggle verbose mode
func TestKeyPressToggleVerbose(t *testing.T) {
	model := NewModel("Test goal", "default")
	model.ready = true

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}

	// Toggle verbose on
	updatedModel, _ := model.Update(keyMsg)
	m := updatedModel.(Model)

	if !m.verboseMode {
		t.Error("Expected verbose mode to be true")
	}

	// Toggle verbose off
	updatedModel, _ = m.Update(keyMsg)
	m = updatedModel.(Model)

	if m.verboseMode {
		t.Error("Expected verbose mode to be false")
	}
}

// TestKeyPressQuit tests 'q' key to quit
func TestKeyPressQuit(t *testing.T) {
	model := NewModel("Test goal", "default")
	model.ready = true

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	updatedModel, cmd := model.Update(keyMsg)
	m := updatedModel.(Model)

	if !m.quitting {
		t.Error("Expected quitting to be true")
	}

	// Verify that quit command is returned
	if cmd == nil {
		t.Error("Expected quit command to be returned")
	}
}

// TestKeyPressApprovalAccept tests 'y' key during approval
func TestKeyPressApprovalAccept(t *testing.T) {
	model := NewModel("Test goal", "default")
	model.ready = true
	model.currentView = ViewApproval
	model.awaitingInput = true
	model.approvalPending = true

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}

	updatedModel, cmd := model.Update(keyMsg)
	m := updatedModel.(Model)

	if m.approvalPending {
		t.Error("Expected approvalPending to be false")
	}

	if m.awaitingInput {
		t.Error("Expected awaitingInput to be false")
	}

	if !m.approvalChoice {
		t.Error("Expected approvalChoice to be true")
	}

	if m.currentView != ViewMain {
		t.Errorf("Expected ViewMain, got %v", m.currentView)
	}

	// Verify that approval response command is returned
	if cmd == nil {
		t.Error("Expected approval response command to be returned")
	}
}

// TestKeyPressApprovalReject tests 'n' key during approval
func TestKeyPressApprovalReject(t *testing.T) {
	model := NewModel("Test goal", "default")
	model.ready = true
	model.currentView = ViewApproval
	model.awaitingInput = true
	model.approvalPending = true

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}

	updatedModel, cmd := model.Update(keyMsg)
	m := updatedModel.(Model)

	if m.approvalPending {
		t.Error("Expected approvalPending to be false")
	}

	if m.awaitingInput {
		t.Error("Expected awaitingInput to be false")
	}

	if m.approvalChoice {
		t.Error("Expected approvalChoice to be false")
	}

	if m.currentView != ViewMain {
		t.Errorf("Expected ViewMain, got %v", m.currentView)
	}

	// Verify that approval response command is returned
	if cmd == nil {
		t.Error("Expected approval response command to be returned")
	}
}

// TestProgressPercentage tests progress calculation
func TestProgressPercentage(t *testing.T) {
	model := NewModel("Test goal", "default")

	// No steps
	if model.progressPercentage() != 0 {
		t.Errorf("Expected 0%%, got %.2f%%", model.progressPercentage())
	}

	// 2 of 4 steps completed
	model.totalSteps = 4
	model.completedSteps = 2

	expected := 50.0
	actual := model.progressPercentage()
	if actual != expected {
		t.Errorf("Expected %.2f%%, got %.2f%%", expected, actual)
	}

	// All steps completed
	model.completedSteps = 4

	expected = 100.0
	actual = model.progressPercentage()
	if actual != expected {
		t.Errorf("Expected %.2f%%, got %.2f%%", expected, actual)
	}
}

// TestViewRendering tests that views render without crashing
func TestViewRendering(t *testing.T) {
	model := NewModel("Test goal", "default")
	model.ready = true

	// Test main view
	model.currentView = ViewMain
	view := model.View()
	if !strings.Contains(view, "Specular Auto Mode") {
		t.Error("Main view should contain title")
	}

	// Test help view
	model.currentView = ViewHelp
	view = model.View()
	if !strings.Contains(view, "Help") {
		t.Error("Help view should contain 'Help'")
	}

	// Test step list view
	model.currentView = ViewStepList
	view = model.View()
	if !strings.Contains(view, "Step List") {
		t.Error("Step list view should contain 'Step List'")
	}

	// Test approval view
	model.currentView = ViewApproval
	model.approvalPlan = "Test plan"
	view = model.View()
	if !strings.Contains(view, "Approval Required") {
		t.Error("Approval view should contain 'Approval Required'")
	}

	// Test completion view
	model.quitting = true
	view = model.View()
	if !strings.Contains(view, "Complete") && !strings.Contains(view, "Failed") {
		t.Error("Completion view should contain status")
	}
}
