package interview

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Engine manages the interview flow
type Engine struct {
	session *Session
	presets map[string]Preset
}

// NewEngine creates a new interview engine
func NewEngine(preset string, strict bool) (*Engine, error) {
	presets := GetPresets()
	p, ok := presets[preset]
	if !ok {
		return nil, fmt.Errorf("unknown preset: %s", preset)
	}

	session := &Session{
		ID:        uuid.New().String(),
		Preset:    preset,
		Strict:    strict,
		Questions: p.Questions,
		Answers:   make(map[string]Answer),
		Current:   0,
	}

	return &Engine{
		session: session,
		presets: presets,
	}, nil
}

// Start begins the interview
func (e *Engine) Start() error {
	if e.session.Current != 0 {
		return fmt.Errorf("interview already started")
	}
	return nil
}

// CurrentQuestion returns the current question
func (e *Engine) CurrentQuestion() (*Question, error) {
	if e.session.Current >= len(e.session.Questions) {
		return nil, fmt.Errorf("interview completed")
	}

	q := &e.session.Questions[e.session.Current]

	// Check if question should be skipped
	if e.shouldSkip(q) {
		return e.nextQuestion()
	}

	return q, nil
}

// shouldSkip checks if a question should be skipped based on SkipIf condition
func (e *Engine) shouldSkip(q *Question) bool {
	if q.SkipIf == "" {
		return false
	}

	// Parse SkipIf condition (format: "question-id=value")
	parts := strings.Split(q.SkipIf, "=")
	if len(parts) != 2 {
		return false
	}

	questionID := parts[0]
	expectedValue := parts[1]

	answer, exists := e.session.Answers[questionID]
	if !exists {
		return false
	}

	// Normalize values for comparison
	normalizedAnswer := strings.ToLower(strings.TrimSpace(answer.Value))
	normalizedExpected := strings.ToLower(strings.TrimSpace(expectedValue))

	return normalizedAnswer == normalizedExpected
}

// nextQuestion advances to the next question
func (e *Engine) nextQuestion() (*Question, error) {
	e.session.Current++

	if e.session.Current >= len(e.session.Questions) {
		return nil, nil // Interview complete
	}

	return e.CurrentQuestion()
}

// Answer records an answer and advances to the next question
func (e *Engine) Answer(answer Answer) (*Question, error) {
	if e.session.Current >= len(e.session.Questions) {
		return nil, fmt.Errorf("interview already completed")
	}

	currentQ := &e.session.Questions[e.session.Current]

	// Validate answer
	if err := e.validateAnswer(currentQ, &answer); err != nil {
		if e.session.Strict {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
		// In non-strict mode, log but continue
		fmt.Printf("Warning: %v\n", err)
	}

	// Store answer
	answer.QuestionID = currentQ.ID
	e.session.Answers[currentQ.ID] = answer

	// Move to next question
	return e.nextQuestion()
}

// validateAnswer validates an answer against question constraints
func (e *Engine) validateAnswer(q *Question, a *Answer) error {
	// Check if required question is answered
	if q.Required {
		if a.Value == "" && len(a.Values) == 0 {
			return fmt.Errorf("answer is required for: %s", q.Text)
		}
	}

	// Validate based on question type
	switch q.Type {
	case QuestionTypeYesNo:
		normalized := strings.ToLower(strings.TrimSpace(a.Value))
		if normalized != "yes" && normalized != "no" {
			return fmt.Errorf("answer must be 'yes' or 'no'")
		}

	case QuestionTypeChoice:
		if len(q.Choices) > 0 {
			valid := false
			for _, choice := range q.Choices {
				if strings.EqualFold(choice, a.Value) {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("answer must be one of: %v", q.Choices)
			}
		}

	case QuestionTypeMulti:
		if len(a.Values) == 0 && a.Value != "" {
			// Convert single value to values array
			a.Values = strings.Split(a.Value, "\n")
			a.Value = ""
		}
	}

	return nil
}

// Progress returns the current progress (percentage)
func (e *Engine) Progress() float64 {
	if len(e.session.Questions) == 0 {
		return 100.0
	}
	return float64(e.session.Current) / float64(len(e.session.Questions)) * 100.0
}

// IsComplete returns true if all questions have been answered
func (e *Engine) IsComplete() bool {
	return e.session.Current >= len(e.session.Questions)
}

// GetResult converts the interview answers into a ProductSpec
func (e *Engine) GetResult() (*InterviewResult, error) {
	if !e.IsComplete() {
		return nil, fmt.Errorf("interview not complete")
	}

	startTime := time.Now()

	spec, err := e.generateSpec()
	if err != nil {
		return nil, fmt.Errorf("generate spec: %w", err)
	}

	duration := time.Since(startTime).Milliseconds()

	return &InterviewResult{
		Spec:     spec,
		Answers:  e.session.Answers,
		Duration: duration,
	}, nil
}

// GetSession returns the current session
func (e *Engine) GetSession() *Session {
	return e.session
}

// ListPresets returns all available presets
func ListPresets() []Preset {
	presets := GetPresets()
	result := make([]Preset, 0, len(presets))
	for _, p := range presets {
		result = append(result, p)
	}
	return result
}
