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

func TestProgress_NoQuestions(t *testing.T) {
	// Create engine with empty session (no questions)
	engine := &Engine{
		session: &Session{
			Questions: []Question{},
			Answers:   make(map[string]Answer),
			Current:   0,
		},
	}

	progress := engine.Progress()
	if progress != 100.0 {
		t.Errorf("Progress() with no questions = %.1f, want 100.0", progress)
	}
}

func TestStart_AlreadyStarted(t *testing.T) {
	engine, err := NewEngine("cli-tool", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Start successfully first time
	if err := engine.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Advance to simulate progress
	engine.session.Current = 1

	// Try to start again should fail
	err = engine.Start()
	if err == nil {
		t.Error("Start() expected error when already started, got nil")
	}
	if !contains(err.Error(), "already started") {
		t.Errorf("Start() error = %v, want error containing 'already started'", err)
	}
}

func TestNextQuestion_InterviewComplete(t *testing.T) {
	engine, err := NewEngine("cli-tool", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	engine.Start()

	// Set current to last question index
	engine.session.Current = len(engine.session.Questions) - 1

	// Call nextQuestion to complete the interview
	q, err := engine.nextQuestion()
	if err != nil {
		t.Fatalf("nextQuestion() unexpected error: %v", err)
	}

	// Should return nil when interview is complete
	if q != nil {
		t.Errorf("nextQuestion() at end of interview returned %v, want nil", q)
	}
}

func TestAnswer_InterviewCompleted(t *testing.T) {
	engine, err := NewEngine("cli-tool", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	engine.Start()

	// Set current to beyond last question to simulate completed interview
	engine.session.Current = len(engine.session.Questions)

	// Try to answer should fail
	_, err = engine.Answer(Answer{Value: "test"})
	if err == nil {
		t.Error("Answer() expected error when interview completed, got nil")
	}
	if !contains(err.Error(), "already completed") {
		t.Errorf("Answer() error = %v, want error containing 'already completed'", err)
	}
}

func TestAnswer_NonStrictMode(t *testing.T) {
	engine, err := NewEngine("cli-tool", false) // Non-strict mode
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	engine.Start()

	// Answer with invalid value (should warn but continue in non-strict mode)
	q, err := engine.Answer(Answer{Value: ""}) // Empty answer for required question

	// Should not return error in non-strict mode
	if err != nil {
		t.Errorf("Answer() in non-strict mode unexpected error: %v", err)
	}

	// Should have moved to next question
	if q == nil && !engine.IsComplete() {
		t.Error("Expected next question in non-strict mode")
	}
}

func TestCurrentQuestion_InterviewCompleted(t *testing.T) {
	engine, err := NewEngine("cli-tool", false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	engine.Start()

	// Set current to beyond last question
	engine.session.Current = len(engine.session.Questions)

	// CurrentQuestion should return error
	q, err := engine.CurrentQuestion()
	if err == nil {
		t.Error("CurrentQuestion() expected error when interview completed, got nil")
	}
	if !contains(err.Error(), "completed") {
		t.Errorf("CurrentQuestion() error = %v, want error containing 'completed'", err)
	}
	if q != nil {
		t.Errorf("CurrentQuestion() returned question %v, want nil", q)
	}
}

func TestValidateAnswer(t *testing.T) {
	tests := []struct {
		name      string
		question  Question
		answer    Answer
		wantErr   bool
		errMsg    string
	}{
		{
			name: "required question with empty answer",
			question: Question{
				ID:       "req-1",
				Text:     "Required question",
				Type:     QuestionTypeText,
				Required: true,
			},
			answer:  Answer{Value: ""},
			wantErr: true,
			errMsg:  "answer is required",
		},
		{
			name: "optional question with empty answer",
			question: Question{
				ID:       "opt-1",
				Text:     "Optional question",
				Type:     QuestionTypeText,
				Required: false,
			},
			answer:  Answer{Value: ""},
			wantErr: false,
		},
		{
			name: "yes/no question with valid yes",
			question: Question{
				ID:   "yn-1",
				Text: "Yes/No question",
				Type: QuestionTypeYesNo,
			},
			answer:  Answer{Value: "yes"},
			wantErr: false,
		},
		{
			name: "yes/no question with valid no",
			question: Question{
				ID:   "yn-2",
				Text: "Yes/No question",
				Type: QuestionTypeYesNo,
			},
			answer:  Answer{Value: "NO"},
			wantErr: false,
		},
		{
			name: "yes/no question with invalid answer",
			question: Question{
				ID:   "yn-3",
				Text: "Yes/No question",
				Type: QuestionTypeYesNo,
			},
			answer:  Answer{Value: "maybe"},
			wantErr: true,
			errMsg:  "must be 'yes' or 'no'",
		},
		{
			name: "choice question with valid choice",
			question: Question{
				ID:      "ch-1",
				Text:    "Choice question",
				Type:    QuestionTypeChoice,
				Choices: []string{"Option A", "Option B", "Option C"},
			},
			answer:  Answer{Value: "Option B"},
			wantErr: false,
		},
		{
			name: "choice question with case insensitive match",
			question: Question{
				ID:      "ch-2",
				Text:    "Choice question",
				Type:    QuestionTypeChoice,
				Choices: []string{"Option A", "Option B"},
			},
			answer:  Answer{Value: "option a"},
			wantErr: false,
		},
		{
			name: "choice question with invalid choice",
			question: Question{
				ID:      "ch-3",
				Text:    "Choice question",
				Type:    QuestionTypeChoice,
				Choices: []string{"Option A", "Option B"},
			},
			answer:  Answer{Value: "Option X"},
			wantErr: true,
			errMsg:  "must be one of",
		},
		{
			name: "multi question converts single value",
			question: Question{
				ID:   "multi-1",
				Text: "Multi question",
				Type: QuestionTypeMulti,
			},
			answer:  Answer{Value: "item1\nitem2\nitem3"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{}
			err := engine.validateAnswer(&tt.question, &tt.answer)

			if tt.wantErr {
				if err == nil {
					t.Error("validateAnswer() expected error, got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateAnswer() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateAnswer() unexpected error = %v", err)
				}
			}
		})
	}
}
