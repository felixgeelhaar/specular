package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewEvalTUI(t *testing.T) {
	config := EvalTUIConfig{
		SpecFile:    "spec.yaml",
		TestCases:   10,
		Providers:   []string{"openai", "anthropic"},
		ShowDetails: true,
		Parallel:    true,
	}

	model := NewEvalTUI(config)

	if model == nil {
		t.Fatal("NewEvalTUI returned nil")
	}
	if model.state != StateInitializing {
		t.Errorf("Expected state %v, got %v", StateInitializing, model.state)
	}
	if model.totalTests != 20 { // 10 test cases * 2 providers
		t.Errorf("Expected totalTests 20, got %d", model.totalTests)
	}
}

func TestEvalTUIModel_Init(t *testing.T) {
	model := NewEvalTUI(EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
	})

	cmd := model.Init()
	if cmd == nil {
		t.Error("Init should return a tick command")
	}
}

func TestEvalTUIModel_Update_WindowSize(t *testing.T) {
	model := NewEvalTUI(EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
	})

	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	newModel, _ := model.Update(msg)
	m := newModel.(EvalTUIModel)

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

func TestEvalTUIModel_Update_EvalStart(t *testing.T) {
	model := NewEvalTUI(EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
	})

	msg := EvalStartMsg{TotalTests: 10}
	newModel, _ := model.Update(msg)
	m := newModel.(EvalTUIModel)

	if m.state != StateRunning {
		t.Errorf("Expected state %v, got %v", StateRunning, m.state)
	}
	if m.totalTests != 10 {
		t.Errorf("Expected totalTests 10, got %d", m.totalTests)
	}
}

func TestEvalTUIModel_Update_EvalResult(t *testing.T) {
	model := NewEvalTUI(EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
	})

	result := EvalResult{
		Name:       "Test 1",
		Provider:   "openai",
		Model:      "gpt-4",
		Score:      85.0,
		MaxScore:   100.0,
		Duration:   time.Second,
		TokensUsed: 500,
		Cost:       0.01,
		Passed:     true,
	}

	msg := EvalResultMsg{Result: result}
	newModel, _ := model.Update(msg)
	m := newModel.(EvalTUIModel)

	if len(m.results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(m.results))
	}
	if m.currentIdx != 1 {
		t.Errorf("Expected currentIdx 1, got %d", m.currentIdx)
	}
	if m.passedCount != 1 {
		t.Errorf("Expected passedCount 1, got %d", m.passedCount)
	}
	if m.totalScore != 85.0 {
		t.Errorf("Expected totalScore 85.0, got %f", m.totalScore)
	}
}

func TestEvalTUIModel_Update_EvalComplete(t *testing.T) {
	tests := []struct {
		name      string
		success   bool
		err       error
		wantState CommandState
	}{
		{"success", true, nil, StateCompleted},
		{"failure", false, &testError{msg: "eval failed"}, StateFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewEvalTUI(EvalTUIConfig{
				SpecFile:  "spec.yaml",
				TestCases: 5,
				Providers: []string{"openai"},
			})

			msg := EvalCompleteMsg{Success: tt.success, Error: tt.err}
			newModel, _ := model.Update(msg)
			m := newModel.(EvalTUIModel)

			if m.state != tt.wantState {
				t.Errorf("Expected state %v, got %v", tt.wantState, m.state)
			}
		})
	}
}

func TestEvalTUIModel_Update_KeyPress(t *testing.T) {
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
			model := NewEvalTUI(EvalTUIConfig{
				SpecFile:  "spec.yaml",
				TestCases: 5,
				Providers: []string{"openai"},
			})
			model.ready = true

			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "ctrl+c" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			} else if tt.key == "tab" {
				msg = tea.KeyMsg{Type: tea.KeyTab}
			}

			newModel, cmd := model.Update(msg)
			m := newModel.(EvalTUIModel)

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

func TestEvalTUIModel_View(t *testing.T) {
	model := NewEvalTUI(EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
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
	if !bytes.Contains([]byte(view), []byte("Evaluation")) {
		t.Error("View should contain 'Evaluation'")
	}
}

func TestEvalTUIModel_updateStats(t *testing.T) {
	model := NewEvalTUI(EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
	})

	// Add passing result
	model.updateStats(EvalResult{
		Score:      80.0,
		MaxScore:   100.0,
		TokensUsed: 500,
		Cost:       0.01,
		Duration:   time.Second,
		Passed:     true,
	})

	if model.passedCount != 1 {
		t.Errorf("Expected passedCount 1, got %d", model.passedCount)
	}
	if model.totalScore != 80.0 {
		t.Errorf("Expected totalScore 80.0, got %f", model.totalScore)
	}
	if model.totalTokens != 500 {
		t.Errorf("Expected totalTokens 500, got %d", model.totalTokens)
	}

	// Add failing result
	model.updateStats(EvalResult{
		Score:      40.0,
		MaxScore:   100.0,
		TokensUsed: 300,
		Cost:       0.005,
		Duration:   2 * time.Second,
		Passed:     false,
	})

	if model.failedCount != 1 {
		t.Errorf("Expected failedCount 1, got %d", model.failedCount)
	}
	if model.totalScore != 120.0 {
		t.Errorf("Expected totalScore 120.0, got %f", model.totalScore)
	}
}

func TestEvalTUIModel_getScoreStyle(t *testing.T) {
	model := NewEvalTUI(EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
	})

	// High score
	style := model.getScoreStyle(90.0)
	if style.GetForeground() != model.styles.ScoreHigh.GetForeground() {
		t.Error("Expected ScoreHigh style for 90%")
	}

	// Medium score
	style = model.getScoreStyle(70.0)
	if style.GetForeground() != model.styles.ScoreMid.GetForeground() {
		t.Error("Expected ScoreMid style for 70%")
	}

	// Low score
	style = model.getScoreStyle(50.0)
	if style.GetForeground() != model.styles.ScoreLow.GetForeground() {
		t.Error("Expected ScoreLow style for 50%")
	}
}

func TestNewEvalTUIAdapter(t *testing.T) {
	config := EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
	}

	var buf bytes.Buffer
	adapter := NewEvalTUIAdapter(config, &buf)

	if adapter == nil {
		t.Fatal("NewEvalTUIAdapter returned nil")
	}
	if adapter.model == nil {
		t.Error("Adapter model should not be nil")
	}
}

func TestNewEvalTUIAdapter_NilOutput(t *testing.T) {
	config := EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
	}

	// Should default to os.Stdout
	adapter := NewEvalTUIAdapter(config, nil)
	if adapter == nil {
		t.Fatal("NewEvalTUIAdapter returned nil")
	}
}

func TestEvalTUIAdapter_Methods(t *testing.T) {
	config := EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
	}

	var buf bytes.Buffer
	adapter := NewEvalTUIAdapter(config, &buf)

	// These should not panic when program is nil
	adapter.StartEval(10)
	adapter.AddResult(EvalResult{Name: "Test", Passed: true})
}

func TestEvalTUIRunner(t *testing.T) {
	config := EvalTUIConfig{
		SpecFile:  "spec.yaml",
		TestCases: 5,
		Providers: []string{"openai"},
	}

	var buf bytes.Buffer
	runner := NewEvalTUIRunner(config, &buf)

	if runner == nil {
		t.Fatal("NewEvalTUIRunner returned nil")
	}
	if runner.adapter == nil {
		t.Error("Runner adapter should not be nil")
	}
}

func TestDefaultEvalStyles(t *testing.T) {
	styles := DefaultEvalStyles()

	// Verify styles are set
	if styles.Title.GetBold() != true {
		t.Error("Title should be bold")
	}
	if styles.Pass.GetBold() != true {
		t.Error("Pass should be bold")
	}
	if styles.Fail.GetBold() != true {
		t.Error("Fail should be bold")
	}
}

func TestEvalResult(t *testing.T) {
	result := EvalResult{
		Name:       "Test Case 1",
		Provider:   "openai",
		Model:      "gpt-4",
		Score:      85.5,
		MaxScore:   100.0,
		Duration:   1500 * time.Millisecond,
		TokensUsed: 1000,
		Cost:       0.02,
		Details: map[string]interface{}{
			"accuracy": 0.85,
		},
		Passed: true,
		Error:  nil,
	}

	if result.Name != "Test Case 1" {
		t.Errorf("Expected name 'Test Case 1', got %q", result.Name)
	}
	if result.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got %q", result.Provider)
	}
	if result.Score != 85.5 {
		t.Errorf("Expected score 85.5, got %f", result.Score)
	}
	if !result.Passed {
		t.Error("Expected passed to be true")
	}
}

func TestEvalTUIConfig(t *testing.T) {
	config := EvalTUIConfig{
		SpecFile:    "spec.yaml",
		TestCases:   10,
		Providers:   []string{"openai", "anthropic"},
		ShowDetails: true,
		Parallel:    true,
	}

	if config.SpecFile != "spec.yaml" {
		t.Errorf("Expected SpecFile 'spec.yaml', got %q", config.SpecFile)
	}
	if config.TestCases != 10 {
		t.Errorf("Expected TestCases 10, got %d", config.TestCases)
	}
	if len(config.Providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(config.Providers))
	}
	if !config.ShowDetails {
		t.Error("Expected ShowDetails to be true")
	}
	if !config.Parallel {
		t.Error("Expected Parallel to be true")
	}
}
