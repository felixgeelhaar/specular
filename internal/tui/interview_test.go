package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/felixgeelhaar/specular/internal/interview"
	"github.com/felixgeelhaar/specular/internal/spec"
)

func TestNewInterviewModel(t *testing.T) {
	tests := []struct {
		name    string
		preset  string
		strict  bool
		wantErr bool
	}{
		{
			name:    "valid api-service preset",
			preset:  "api-service",
			strict:  false,
			wantErr: false,
		},
		{
			name:    "valid web-app preset",
			preset:  "web-app",
			strict:  false,
			wantErr: false,
		},
		{
			name:    "strict mode enabled",
			preset:  "cli-tool",
			strict:  true,
			wantErr: false,
		},
		{
			name:    "invalid preset",
			preset:  "invalid-preset",
			strict:  false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := interview.NewEngine(tt.preset, tt.strict)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewEngine() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewEngine() unexpected error: %v", err)
			}

			model, err := NewInterviewModel(engine)
			if err != nil {
				t.Fatalf("NewInterviewModel() unexpected error: %v", err)
			}

			if model.engine != engine {
				t.Errorf("NewInterviewModel() engine not set correctly")
			}

			if model.form == nil {
				t.Errorf("NewInterviewModel() form not created")
			}

			if model.completed {
				t.Errorf("NewInterviewModel() should not be completed initially")
			}
		})
	}
}

func TestInterviewModelInit(t *testing.T) {
	engine, err := interview.NewEngine("api-service", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	model, err := NewInterviewModel(engine)
	if err != nil {
		t.Fatalf("NewInterviewModel() failed: %v", err)
	}

	cmd := model.Init()
	if cmd == nil {
		t.Errorf("Init() should return a command")
	}
}

func TestInterviewModelView(t *testing.T) {
	tests := []struct {
		name     string
		preset   string
		wantText string
	}{
		{
			name:     "api-service shows form",
			preset:   "api-service",
			wantText: "", // Form view contains dynamic content
		},
		{
			name:     "web-app shows form",
			preset:   "web-app",
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := interview.NewEngine(tt.preset, false)
			if err != nil {
				t.Fatalf("NewEngine() failed: %v", err)
			}

			model, err := NewInterviewModel(engine)
			if err != nil {
				t.Fatalf("NewInterviewModel() failed: %v", err)
			}

			view := model.View()
			if view == "" {
				t.Errorf("View() returned empty string")
			}

			// Check that view is not an error
			if strings.Contains(view, "Error:") {
				t.Errorf("View() contains error: %s", view)
			}
		})
	}
}

func TestInterviewModelUpdate(t *testing.T) {
	engine, err := interview.NewEngine("api-service", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	model, err := NewInterviewModel(engine)
	if err != nil {
		t.Fatalf("NewInterviewModel() failed: %v", err)
	}

	// Test window size message
	t.Run("window size update", func(t *testing.T) {
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		updatedModel, _ := model.Update(msg)

		m, ok := updatedModel.(*InterviewModel)
		if !ok {
			t.Fatalf("Update() returned wrong type")
		}

		if m.width != 80 {
			t.Errorf("Update() width not set, got %d, want 80", m.width)
		}

		if m.height != 24 {
			t.Errorf("Update() height not set, got %d, want 24", m.height)
		}
	})

	// Test quit key
	t.Run("quit on ctrl+c", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		updatedModel, cmd := model.Update(msg)

		m, ok := updatedModel.(*InterviewModel)
		if !ok {
			t.Fatalf("Update() returned wrong type")
		}

		if !m.quitting {
			t.Errorf("Update() should set quitting=true on Ctrl+C")
		}

		if cmd == nil {
			t.Errorf("Update() should return tea.Quit command")
		}
	})
}

func TestFormatProgress(t *testing.T) {
	engine, err := interview.NewEngine("api-service", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	model, err := NewInterviewModel(engine)
	if err != nil {
		t.Fatalf("NewInterviewModel() failed: %v", err)
	}

	progress := model.formatProgress()
	if progress == "" {
		t.Errorf("formatProgress() returned empty string")
	}

	// Should contain question number and percentage
	if !strings.Contains(progress, "Question") {
		t.Errorf("formatProgress() should contain 'Question', got: %s", progress)
	}

	if !strings.Contains(progress, "%") {
		t.Errorf("formatProgress() should contain '%%', got: %s", progress)
	}
}

func TestFormatHelp(t *testing.T) {
	engine, err := interview.NewEngine("api-service", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	model, err := NewInterviewModel(engine)
	if err != nil {
		t.Fatalf("NewInterviewModel() failed: %v", err)
	}

	help := model.formatHelp()
	if help == "" {
		t.Errorf("formatHelp() returned empty string")
	}

	// Should mention navigation
	if !strings.Contains(help, "arrow keys") && !strings.Contains(help, "Enter") {
		t.Errorf("formatHelp() should mention navigation keys, got: %s", help)
	}
}

func TestRenderCompletion(t *testing.T) {
	engine, err := interview.NewEngine("api-service", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	model, err := NewInterviewModel(engine)
	if err != nil {
		t.Fatalf("NewInterviewModel() failed: %v", err)
	}

	// Manually set completion state for testing
	model.completed = true
	model.result = &interview.InterviewResult{
		Spec: &spec.ProductSpec{
			Product:  "Test Product",
			Features: []spec.Feature{{ID: "f1"}},
		},
		Duration: 100,
	}

	view := model.renderCompletion()
	if view == "" {
		t.Errorf("renderCompletion() returned empty string")
	}

	// Should contain success indicator
	if !strings.Contains(view, "Complete") {
		t.Errorf("renderCompletion() should contain 'Complete', got: %s", view)
	}

	// Should show product name
	if !strings.Contains(view, "Test Product") {
		t.Errorf("renderCompletion() should contain product name, got: %s", view)
	}
}

func TestRenderError(t *testing.T) {
	engine, err := interview.NewEngine("api-service", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	model, err := NewInterviewModel(engine)
	if err != nil {
		t.Fatalf("NewInterviewModel() failed: %v", err)
	}

	// Set error state
	model.err = errors.New("test error")

	view := model.renderError()
	if view == "" {
		t.Errorf("renderError() returned empty string")
	}

	// Should contain error text
	if !strings.Contains(view, "Error") {
		t.Errorf("renderError() should contain 'Error', got: %s", view)
	}
}

func TestViewStates(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*InterviewModel)
		wantText  string
	}{
		{
			name: "quitting state",
			setupFunc: func(m *InterviewModel) {
				m.quitting = true
			},
			wantText: "cancelled",
		},
		{
			name: "error state",
			setupFunc: func(m *InterviewModel) {
				m.err = errors.New("test error")
			},
			wantText: "Error",
		},
		{
			name: "completed state",
			setupFunc: func(m *InterviewModel) {
				m.completed = true
				m.result = &interview.InterviewResult{
					Spec: &spec.ProductSpec{
						Product: "Test",
					},
					Duration: 50,
				}
			},
			wantText: "Complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := interview.NewEngine("api-service", false)
			if err != nil {
				t.Fatalf("NewEngine() failed: %v", err)
			}

			model, err := NewInterviewModel(engine)
			if err != nil {
				t.Fatalf("NewInterviewModel() failed: %v", err)
			}

			tt.setupFunc(model)
			view := model.View()

			if !strings.Contains(view, tt.wantText) {
				t.Errorf("View() in %s state should contain '%s', got: %s", tt.name, tt.wantText, view)
			}
		})
	}
}
