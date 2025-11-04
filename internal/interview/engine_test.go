package interview

import (
	"testing"
)

func TestNewEngine(t *testing.T) {
	tests := []struct {
		name    string
		preset  string
		strict  bool
		wantErr bool
	}{
		{
			name:    "valid web-app preset",
			preset:  "web-app",
			strict:  false,
			wantErr: false,
		},
		{
			name:    "valid api-service preset",
			preset:  "api-service",
			strict:  true,
			wantErr: false,
		},
		{
			name:    "invalid preset",
			preset:  "nonexistent",
			strict:  false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(tt.preset, tt.strict)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEngine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && engine == nil {
				t.Error("NewEngine() returned nil engine")
			}
			if !tt.wantErr && engine.session.Preset != tt.preset {
				t.Errorf("NewEngine() preset = %v, want %v", engine.session.Preset, tt.preset)
			}
		})
	}
}

func TestEngineQuestionFlow(t *testing.T) {
	engine, err := NewEngine("cli-tool", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	if err := engine.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Get first question
	q, err := engine.CurrentQuestion()
	if err != nil {
		t.Fatalf("CurrentQuestion() failed: %v", err)
	}
	if q == nil {
		t.Fatal("CurrentQuestion() returned nil")
	}
	if q.ID != "product-name" {
		t.Errorf("First question ID = %v, want product-name", q.ID)
	}

	// Answer first question
	answer := Answer{Value: "TestTool"}
	nextQ, err := engine.Answer(answer)
	if err != nil {
		t.Fatalf("Answer() failed: %v", err)
	}
	if nextQ == nil && !engine.IsComplete() {
		t.Error("Expected next question but got nil")
	}

	// Check progress
	if engine.Progress() <= 0 {
		t.Error("Progress should be > 0 after answering first question")
	}
}

func TestEngineValidation(t *testing.T) {
	engine, err := NewEngine("web-app", true) // Strict mode
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	engine.Start()

	tests := []struct {
		name    string
		answer  Answer
		wantErr bool
	}{
		{
			name:    "valid text answer",
			answer:  Answer{Value: "My Product"},
			wantErr: false,
		},
		{
			name:    "empty required answer",
			answer:  Answer{Value: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to first question for each test
			engine.session.Current = 0

			_, err := engine.Answer(tt.answer)
			if (err != nil) != tt.wantErr {
				t.Errorf("Answer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListPresets(t *testing.T) {
	presets := ListPresets()
	if len(presets) == 0 {
		t.Error("ListPresets() returned empty list")
	}

	// Verify expected presets exist
	expectedPresets := map[string]bool{
		"web-app":       false,
		"api-service":   false,
		"cli-tool":      false,
		"microservice":  false,
		"data-pipeline": false,
	}

	for _, preset := range presets {
		if _, exists := expectedPresets[preset.Name]; exists {
			expectedPresets[preset.Name] = true
		}
	}

	for name, found := range expectedPresets {
		if !found {
			t.Errorf("Expected preset %s not found", name)
		}
	}
}

func TestSkipLogic(t *testing.T) {
	engine, err := NewEngine("web-app", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	engine.Start()

	// Navigate to auth-required question
	for i := 0; i < 4; i++ {
		q, _ := engine.CurrentQuestion()
		engine.Answer(Answer{Value: "test"})
		if q.ID == "auth-required" {
			break
		}
	}

	// Answer "no" to auth-required
	engine.Answer(Answer{Value: "no"})

	// Check that auth-type question is skipped
	q, _ := engine.CurrentQuestion()
	if q != nil && q.ID == "auth-type" {
		t.Error("auth-type question should have been skipped")
	}
}
