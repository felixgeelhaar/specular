package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewCommandTUI(t *testing.T) {
	config := CommandTUIConfig{
		Command:     CommandBuild,
		Title:       "Test Build",
		Description: "Test description",
		Phases:      []string{"Phase 1", "Phase 2", "Phase 3"},
		ShowLogs:    true,
		ShowMetrics: true,
		Interactive: true,
	}

	model := NewCommandTUI(config)

	if model == nil {
		t.Fatal("NewCommandTUI returned nil")
	}
	if model.state != StateInitializing {
		t.Errorf("Expected state %v, got %v", StateInitializing, model.state)
	}
	if len(model.phases) != 3 {
		t.Errorf("Expected 3 phases, got %d", len(model.phases))
	}
	for i, phase := range model.phases {
		if phase.Status != PhasePending {
			t.Errorf("Phase %d: expected status %v, got %v", i, PhasePending, phase.Status)
		}
	}
}

func TestCommandTUIModel_Init(t *testing.T) {
	model := NewCommandTUI(CommandTUIConfig{
		Command: CommandBuild,
		Title:   "Test",
		Phases:  []string{"Phase 1"},
	})

	cmd := model.Init()
	if cmd == nil {
		t.Error("Init should return a tick command")
	}
}

func TestCommandTUIModel_Update_WindowSize(t *testing.T) {
	model := NewCommandTUI(CommandTUIConfig{
		Command: CommandBuild,
		Title:   "Test",
		Phases:  []string{"Phase 1"},
	})

	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	newModel, _ := model.Update(msg)
	m := newModel.(CommandTUIModel)

	if m.width != 100 {
		t.Errorf("Expected width 100, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
	if !m.ready {
		t.Error("Expected ready to be true")
	}
}

func TestCommandTUIModel_Update_PhaseStart(t *testing.T) {
	model := NewCommandTUI(CommandTUIConfig{
		Command: CommandBuild,
		Title:   "Test",
		Phases:  []string{"Phase 1", "Phase 2"},
	})

	msg := PhaseStartMsg{Phase: 0, Name: "Phase 1"}
	newModel, _ := model.Update(msg)
	m := newModel.(CommandTUIModel)

	if m.phases[0].Status != PhaseRunning {
		t.Errorf("Expected phase status %v, got %v", PhaseRunning, m.phases[0].Status)
	}
	if m.currentPhase != 0 {
		t.Errorf("Expected current phase 0, got %d", m.currentPhase)
	}
	if m.state != StateRunning {
		t.Errorf("Expected state %v, got %v", StateRunning, m.state)
	}
}

func TestCommandTUIModel_Update_PhaseComplete(t *testing.T) {
	model := NewCommandTUI(CommandTUIConfig{
		Command: CommandBuild,
		Title:   "Test",
		Phases:  []string{"Phase 1"},
	})

	// Start phase first
	model.Update(PhaseStartMsg{Phase: 0, Name: "Phase 1"})

	details := map[string]interface{}{"files": 5}
	msg := PhaseCompleteMsg{Phase: 0, Name: "Phase 1", Details: details}
	newModel, _ := model.Update(msg)
	m := newModel.(CommandTUIModel)

	if m.phases[0].Status != PhaseCompleted {
		t.Errorf("Expected phase status %v, got %v", PhaseCompleted, m.phases[0].Status)
	}
	if m.phases[0].Details["files"] != 5 {
		t.Error("Phase details not set correctly")
	}
}

func TestCommandTUIModel_Update_PhaseFail(t *testing.T) {
	model := NewCommandTUI(CommandTUIConfig{
		Command: CommandBuild,
		Title:   "Test",
		Phases:  []string{"Phase 1"},
	})

	// Start phase first
	model.Update(PhaseStartMsg{Phase: 0, Name: "Phase 1"})

	testErr := &testError{msg: "test error"}
	msg := PhaseFailMsg{Phase: 0, Name: "Phase 1", Error: testErr}
	newModel, _ := model.Update(msg)
	m := newModel.(CommandTUIModel)

	if m.phases[0].Status != PhaseFailed {
		t.Errorf("Expected phase status %v, got %v", PhaseFailed, m.phases[0].Status)
	}
	if m.lastError != "test error" {
		t.Errorf("Expected lastError 'test error', got %q", m.lastError)
	}
}

func TestCommandTUIModel_Update_CommandDone(t *testing.T) {
	tests := []struct {
		name      string
		success   bool
		err       error
		wantState CommandState
	}{
		{"success", true, nil, StateCompleted},
		{"failure", false, &testError{msg: "failed"}, StateFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewCommandTUI(CommandTUIConfig{
				Command: CommandBuild,
				Title:   "Test",
				Phases:  []string{"Phase 1"},
			})

			msg := CommandDoneMsg{Success: tt.success, Error: tt.err}
			newModel, _ := model.Update(msg)
			m := newModel.(CommandTUIModel)

			if m.state != tt.wantState {
				t.Errorf("Expected state %v, got %v", tt.wantState, m.state)
			}
		})
	}
}

func TestCommandTUIModel_Update_KeyPress(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		wantQuit bool
		wantTab  int
	}{
		{"quit", "q", true, 0},
		{"ctrl+c", "ctrl+c", true, 0},
		{"tab", "tab", false, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewCommandTUI(CommandTUIConfig{
				Command: CommandBuild,
				Title:   "Test",
				Phases:  []string{"Phase 1"},
			})
			model.ready = true

			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "ctrl+c" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			} else if tt.key == "tab" {
				msg = tea.KeyMsg{Type: tea.KeyTab}
			}

			newModel, cmd := model.Update(msg)
			m := newModel.(CommandTUIModel)

			if m.quitting != tt.wantQuit {
				t.Errorf("Expected quitting %v, got %v", tt.wantQuit, m.quitting)
			}
			if !tt.wantQuit && m.activeTab != tt.wantTab {
				t.Errorf("Expected activeTab %d, got %d", tt.wantTab, m.activeTab)
			}
			if tt.wantQuit && cmd == nil {
				t.Error("Expected quit command")
			}
		})
	}
}

func TestCommandTUIModel_View(t *testing.T) {
	model := NewCommandTUI(CommandTUIConfig{
		Command: CommandBuild,
		Title:   "Test Build",
		Phases:  []string{"Phase 1", "Phase 2"},
	})

	// Not ready
	view := model.View()
	if view != "Initializing..." {
		t.Errorf("Expected 'Initializing...' when not ready, got %q", view)
	}

	// Ready
	model.ready = true
	model.width = 80
	model.height = 24
	view = model.View()

	if view == "" {
		t.Error("View should not be empty when ready")
	}
	if !bytes.Contains([]byte(view), []byte("Test Build")) {
		t.Error("View should contain title")
	}
}

func TestCommandTUIAdapter(t *testing.T) {
	var buf bytes.Buffer

	config := CommandTUIConfig{
		Command: CommandBuild,
		Title:   "Test",
		Phases:  []string{"Phase 1"},
	}

	adapter := NewCommandTUIAdapter(config, &buf)
	if adapter == nil {
		t.Fatal("NewCommandTUIAdapter returned nil")
	}
	if adapter.model == nil {
		t.Error("Adapter model should not be nil")
	}
}

func TestDefaultCommandStyles(t *testing.T) {
	styles := DefaultCommandStyles()

	// Just verify styles are not zero values
	if styles.Title.GetBold() != true {
		t.Error("Title should be bold")
	}
	if styles.Error.GetBold() != true {
		t.Error("Error should be bold")
	}
	if styles.Success.GetBold() != true {
		t.Error("Success should be bold")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestCommandPhase(t *testing.T) {
	phase := CommandPhase{
		Name:        "Test Phase",
		Description: "Test description",
		Status:      PhasePending,
		Details:     make(map[string]interface{}),
	}

	if phase.Name != "Test Phase" {
		t.Errorf("Expected name 'Test Phase', got %q", phase.Name)
	}
	if phase.Status != PhasePending {
		t.Errorf("Expected status %v, got %v", PhasePending, phase.Status)
	}
}

func TestOutputItem(t *testing.T) {
	output := OutputItem{
		Name:    "test.yaml",
		Path:    "/path/to/test.yaml",
		Size:    1024,
		Created: time.Now(),
		Metadata: map[string]string{
			"type": "spec",
		},
	}

	if output.Name != "test.yaml" {
		t.Errorf("Expected name 'test.yaml', got %q", output.Name)
	}
	if output.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", output.Size)
	}
}

func TestCommandTUIModel_LogEntry(t *testing.T) {
	model := NewCommandTUI(CommandTUIConfig{
		Command: CommandBuild,
		Title:   "Test",
		Phases:  []string{"Phase 1"},
	})

	msg := LogEntryMsg{Level: LogLevelInfo, Message: "Test log"}
	newModel, _ := model.Update(msg)
	m := newModel.(CommandTUIModel)

	if len(m.logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(m.logs))
	}
	if m.logs[0].Message != "Test log" {
		t.Errorf("Expected message 'Test log', got %q", m.logs[0].Message)
	}
}

func TestCommandTUIModel_MetricUpdate(t *testing.T) {
	model := NewCommandTUI(CommandTUIConfig{
		Command: CommandBuild,
		Title:   "Test",
		Phases:  []string{"Phase 1"},
	})

	msg := MetricUpdateMsg{Key: "files", Value: 10}
	newModel, _ := model.Update(msg)
	m := newModel.(CommandTUIModel)

	if m.metrics["files"] != 10 {
		t.Errorf("Expected metric 'files' = 10, got %v", m.metrics["files"])
	}
}

func TestCommandTUIModel_OutputAdded(t *testing.T) {
	model := NewCommandTUI(CommandTUIConfig{
		Command: CommandBuild,
		Title:   "Test",
		Phases:  []string{"Phase 1"},
	})

	output := OutputItem{
		Name: "test.yaml",
		Path: "/path/to/test.yaml",
		Size: 512,
	}
	msg := OutputAddedMsg{Output: output}
	newModel, _ := model.Update(msg)
	m := newModel.(CommandTUIModel)

	if len(m.outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(m.outputs))
	}
	if m.outputs[0].Name != "test.yaml" {
		t.Errorf("Expected output name 'test.yaml', got %q", m.outputs[0].Name)
	}
}

// testError is a simple error for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
