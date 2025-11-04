package interview

import "github.com/felixgeelhaar/ai-dev/internal/spec"

// QuestionType defines the type of question
type QuestionType string

const (
	QuestionTypeText     QuestionType = "text"
	QuestionTypeMulti    QuestionType = "multi"
	QuestionTypeYesNo    QuestionType = "yesno"
	QuestionTypeChoice   QuestionType = "choice"
	QuestionTypePriority QuestionType = "priority"
)

// Question represents a single question in the interview
type Question struct {
	ID          string       `json:"id"`
	Type        QuestionType `json:"type"`
	Text        string       `json:"text"`
	Description string       `json:"description,omitempty"`
	Required    bool         `json:"required"`
	Choices     []string     `json:"choices,omitempty"`
	Validation  string       `json:"validation,omitempty"`
	SkipIf      string       `json:"skip_if,omitempty"` // Condition to skip this question
}

// Answer represents an answer to a question
type Answer struct {
	QuestionID string   `json:"question_id"`
	Value      string   `json:"value,omitempty"`
	Values     []string `json:"values,omitempty"` // For multi-select questions
}

// Session represents an interview session
type Session struct {
	ID        string            `json:"id"`
	Preset    string            `json:"preset"`
	Strict    bool              `json:"strict"`
	Questions []Question        `json:"questions"`
	Answers   map[string]Answer `json:"answers"`
	Current   int               `json:"current"` // Index of current question
}

// InterviewResult contains the generated spec and metadata
type InterviewResult struct {
	Spec     *spec.ProductSpec `json:"spec"`
	Answers  map[string]Answer `json:"answers"`
	Duration int64             `json:"duration_ms"`
}

// Preset defines a preset interview template
type Preset struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Questions   []Question `json:"questions"`
}
