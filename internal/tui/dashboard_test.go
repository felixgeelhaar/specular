package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewDashboardModel(t *testing.T) {
	goal := "Test goal"
	profile := "test-profile"

	model := NewDashboardModel(goal, profile)

	// Verify base model properties
	if model.goal != goal {
		t.Errorf("expected goal %s, got %s", goal, model.goal)
	}
	if model.profile != profile {
		t.Errorf("expected profile %s, got %s", profile, model.profile)
	}

	// Verify dashboard-specific properties
	if model.maxLogEntries != 100 {
		t.Errorf("expected maxLogEntries 100, got %d", model.maxLogEntries)
	}
	if model.activePanel != PanelProgress {
		t.Errorf("expected activePanel PanelProgress, got %v", model.activePanel)
	}
	if !model.dashboardMode {
		t.Error("expected dashboardMode to be true")
	}
	if model.budgetLimit != 5.0 {
		t.Errorf("expected default budgetLimit 5.0, got %f", model.budgetLimit)
	}
}

func TestDashboardModel_SetBudgetLimit(t *testing.T) {
	model := NewDashboardModel("goal", "profile")
	model.SetBudgetLimit(10.0)

	if model.budgetLimit != 10.0 {
		t.Errorf("expected budgetLimit 10.0, got %f", model.budgetLimit)
	}
	if model.budgetRemaining != 10.0 {
		t.Errorf("expected budgetRemaining 10.0 (no spent), got %f", model.budgetRemaining)
	}
}

func TestDashboardModel_UpdateBudget(t *testing.T) {
	model := NewDashboardModel("goal", "profile")
	model.SetBudgetLimit(10.0)
	model.UpdateBudget(3.5, "test warning")

	if model.budgetSpent != 3.5 {
		t.Errorf("expected budgetSpent 3.5, got %f", model.budgetSpent)
	}
	if model.budgetRemaining != 6.5 {
		t.Errorf("expected budgetRemaining 6.5, got %f", model.budgetRemaining)
	}
	if model.budgetWarning != "test warning" {
		t.Errorf("expected warning 'test warning', got %s", model.budgetWarning)
	}
}

func TestDashboardModel_AddLogEntry(t *testing.T) {
	model := NewDashboardModel("goal", "profile")

	// Add some entries
	model.AddLogEntry(LogLevelInfo, "Test message 1", "step-1")
	model.AddLogEntry(LogLevelWarn, "Test message 2", "step-2")
	model.AddLogEntry(LogLevelError, "Test message 3", "step-3")

	if len(model.logEntries) != 3 {
		t.Errorf("expected 3 log entries, got %d", len(model.logEntries))
	}

	// Verify first entry
	if model.logEntries[0].Level != LogLevelInfo {
		t.Errorf("expected first entry level INFO, got %s", model.logEntries[0].Level)
	}
	if model.logEntries[0].Message != "Test message 1" {
		t.Errorf("expected first entry message 'Test message 1', got %s", model.logEntries[0].Message)
	}
}

func TestDashboardModel_LogCircularBuffer(t *testing.T) {
	model := NewDashboardModel("goal", "profile")
	model.maxLogEntries = 3 // Set small buffer for testing

	// Add 5 entries to trigger circular buffer behavior
	for i := 1; i <= 5; i++ {
		model.AddLogEntry(LogLevelInfo, "message", "step")
	}

	// Should only have 3 entries (the last 3)
	if len(model.logEntries) != 3 {
		t.Errorf("expected 3 log entries (circular buffer), got %d", len(model.logEntries))
	}
}

func TestDashboardModel_IncrementAPICall(t *testing.T) {
	model := NewDashboardModel("goal", "profile")

	model.IncrementAPICall()
	if model.apiCallCount != 1 {
		t.Errorf("expected apiCallCount 1, got %d", model.apiCallCount)
	}

	model.IncrementAPICall()
	model.IncrementAPICall()
	if model.apiCallCount != 3 {
		t.Errorf("expected apiCallCount 3, got %d", model.apiCallCount)
	}
}

func TestDashboardModel_GetBudgetPercentage(t *testing.T) {
	tests := []struct {
		name     string
		limit    float64
		spent    float64
		expected float64
	}{
		{"zero limit", 0, 0, 0},
		{"no spent", 10.0, 0, 0},
		{"50 percent", 10.0, 5.0, 50.0},
		{"75 percent", 10.0, 7.5, 75.0},
		{"100 percent", 10.0, 10.0, 100.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := NewDashboardModel("goal", "profile")
			model.budgetLimit = tc.limit
			model.budgetSpent = tc.spent

			result := model.GetBudgetPercentage()
			if result != tc.expected {
				t.Errorf("expected %f, got %f", tc.expected, result)
			}
		})
	}
}

func TestDashboardModel_GetETA(t *testing.T) {
	model := NewDashboardModel("goal", "profile")
	model.totalSteps = 10
	model.completedSteps = 5
	model.startTime = time.Now().Add(-5 * time.Minute) // 5 minutes elapsed

	eta := model.GetETA()

	// With 5 steps in 5 minutes, average is 1 minute per step
	// 5 remaining steps = ~5 minutes ETA
	// Allow some variance for timing
	if eta < 4*time.Minute || eta > 6*time.Minute {
		t.Errorf("expected ETA around 5 minutes, got %v", eta)
	}
}

func TestDashboardModel_GetETA_NoSteps(t *testing.T) {
	model := NewDashboardModel("goal", "profile")

	// No completed steps
	model.totalSteps = 10
	model.completedSteps = 0

	eta := model.GetETA()
	if eta != 0 {
		t.Errorf("expected ETA 0 with no completed steps, got %v", eta)
	}

	// No total steps
	model.totalSteps = 0
	eta = model.GetETA()
	if eta != 0 {
		t.Errorf("expected ETA 0 with no total steps, got %v", eta)
	}
}

func TestDashboardModel_GetLogEntries(t *testing.T) {
	model := NewDashboardModel("goal", "profile")

	// Add 10 entries
	for i := 1; i <= 10; i++ {
		model.AddLogEntry(LogLevelInfo, "message", "step")
	}

	// Get 5 entries
	entries := model.GetLogEntries(5)
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}

func TestDashboardModel_UpdateKeyPress(t *testing.T) {
	model := NewDashboardModel("goal", "profile")
	model.ready = true

	tests := []struct {
		name          string
		key           string
		expectedPanel PanelType
	}{
		{"tab cycles panel", "tab", PanelBudget},
		{"l toggles logs", "l", PanelLogs},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewDashboardModel("goal", "profile")
			m.ready = true

			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)}
			if tc.key == "tab" {
				msg = tea.KeyMsg{Type: tea.KeyTab}
			}

			result, _ := m.Update(msg)
			updatedModel := result.(DashboardModel)

			if updatedModel.activePanel != tc.expectedPanel {
				t.Errorf("expected panel %v, got %v", tc.expectedPanel, updatedModel.activePanel)
			}
		})
	}
}

func TestDashboardModel_BudgetUpdateMsg(t *testing.T) {
	model := NewDashboardModel("goal", "profile")
	model.ready = true

	msg := BudgetUpdateMsg{
		Spent:     2.5,
		Remaining: 7.5,
		Limit:     10.0,
		Warning:   "test warning",
	}

	result, _ := model.Update(msg)
	updatedModel := result.(DashboardModel)

	if updatedModel.budgetSpent != 2.5 {
		t.Errorf("expected budgetSpent 2.5, got %f", updatedModel.budgetSpent)
	}
	if updatedModel.budgetRemaining != 7.5 {
		t.Errorf("expected budgetRemaining 7.5, got %f", updatedModel.budgetRemaining)
	}
	if updatedModel.budgetWarning != "test warning" {
		t.Errorf("expected warning 'test warning', got %s", updatedModel.budgetWarning)
	}
}

func TestDashboardModel_APIMetricsMsg(t *testing.T) {
	model := NewDashboardModel("goal", "profile")
	model.ready = true

	msg := APIMetricsMsg{
		CallCount: 42,
	}

	result, _ := model.Update(msg)
	updatedModel := result.(DashboardModel)

	if updatedModel.apiCallCount != 42 {
		t.Errorf("expected apiCallCount 42, got %d", updatedModel.apiCallCount)
	}
}

func TestDashboardModel_LogEntryMsg(t *testing.T) {
	model := NewDashboardModel("goal", "profile")
	model.ready = true

	msg := LogEntryMsg{
		Level:   LogLevelError,
		Message: "Test error",
		StepID:  "step-1",
	}

	result, _ := model.Update(msg)
	updatedModel := result.(DashboardModel)

	if len(updatedModel.logEntries) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(updatedModel.logEntries))
	}
	if updatedModel.logEntries[0].Level != LogLevelError {
		t.Errorf("expected level ERROR, got %s", updatedModel.logEntries[0].Level)
	}
}

func TestDashboardModel_ViewDashboardMode(t *testing.T) {
	model := NewDashboardModel("goal", "profile")
	model.ready = true
	model.dashboardMode = true

	view := model.View()

	// Should contain dashboard elements
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestDashboardModel_ViewNotReady(t *testing.T) {
	model := NewDashboardModel("goal", "profile")
	model.ready = false

	view := model.View()

	if view != "Initializing dashboard..." {
		t.Errorf("expected 'Initializing dashboard...', got %s", view)
	}
}

func TestLogLevel_Constants(t *testing.T) {
	// Verify log level constants
	if LogLevelInfo != "INFO" {
		t.Errorf("expected LogLevelInfo 'INFO', got %s", LogLevelInfo)
	}
	if LogLevelWarn != "WARN" {
		t.Errorf("expected LogLevelWarn 'WARN', got %s", LogLevelWarn)
	}
	if LogLevelError != "ERROR" {
		t.Errorf("expected LogLevelError 'ERROR', got %s", LogLevelError)
	}
	if LogLevelDebug != "DEBUG" {
		t.Errorf("expected LogLevelDebug 'DEBUG', got %s", LogLevelDebug)
	}
}

func TestPanelType_Constants(t *testing.T) {
	// Verify panel type constants
	if PanelProgress != 0 {
		t.Errorf("expected PanelProgress 0, got %d", PanelProgress)
	}
	if PanelBudget != 1 {
		t.Errorf("expected PanelBudget 1, got %d", PanelBudget)
	}
	if PanelLogs != 2 {
		t.Errorf("expected PanelLogs 2, got %d", PanelLogs)
	}
}
