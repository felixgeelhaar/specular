package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/felixgeelhaar/specular/internal/interview"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// InterviewModel represents the TUI state for the interview
type InterviewModel struct {
	engine    *interview.Engine
	form      *huh.Form
	quitting  bool
	completed bool
	result    *interview.InterviewResult
	err       error
	width     int
	height    int
}

// keyMap defines the keyboard shortcuts
type keyMap struct {
	Quit key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}

// NewInterviewModel creates a new interview TUI model
func NewInterviewModel(engine *interview.Engine) (*InterviewModel, error) {
	if err := engine.Start(); err != nil {
		return nil, fmt.Errorf("start interview: %w", err)
	}

	m := &InterviewModel{
		engine: engine,
	}

	// Create form for first question
	if err := m.createQuestionForm(); err != nil {
		return nil, fmt.Errorf("create initial form: %w", err)
	}

	return m, nil
}

// createQuestionForm creates a huh form for the current question
func (m *InterviewModel) createQuestionForm() error {
	q, err := m.engine.CurrentQuestion()
	if err != nil {
		if m.engine.IsComplete() {
			// Interview complete, generate result
			m.completed = true
			result, genErr := m.engine.GetResult()
			if genErr != nil {
				m.err = fmt.Errorf("generate spec: %w", genErr)
				return m.err
			}
			m.result = result
			return nil
		}
		m.err = err
		return err
	}

	if q == nil {
		m.completed = true
		result, genErr := m.engine.GetResult()
		if genErr != nil {
			m.err = fmt.Errorf("generate spec: %w", genErr)
			return m.err
		}
		m.result = result
		return nil
	}

	// Get previous answer if exists
	session := m.engine.GetSession()
	prevAnswer, hasPrev := session.Answers[q.ID]

	// Create form field based on question type
	var field huh.Field

	switch q.Type {
	case interview.QuestionTypeText:
		// Single-line text input
		defaultVal := ""
		if hasPrev {
			defaultVal = prevAnswer.Value
		}

		field = huh.NewInput().
			Key(q.ID).
			Title(q.Text).
			Description(q.Description).
			Value(&defaultVal).
			Validate(func(s string) error {
				if q.Required && strings.TrimSpace(s) == "" {
					return fmt.Errorf("this field is required")
				}
				return nil
			})

	case interview.QuestionTypeMulti:
		// Multi-line text input
		defaultVal := ""
		if hasPrev && len(prevAnswer.Values) > 0 {
			defaultVal = strings.Join(prevAnswer.Values, "\n")
		}

		field = huh.NewText().
			Key(q.ID).
			Title(q.Text).
			Description(q.Description).
			Value(&defaultVal).
			Validate(func(s string) error {
				if q.Required && strings.TrimSpace(s) == "" {
					return fmt.Errorf("this field is required")
				}
				return nil
			})

	case interview.QuestionTypeYesNo:
		// Yes/No confirmation
		defaultVal := false
		if hasPrev {
			defaultVal = strings.ToLower(prevAnswer.Value) == "yes"
		}

		field = huh.NewConfirm().
			Key(q.ID).
			Title(q.Text).
			Description(q.Description).
			Value(&defaultVal).
			Affirmative("Yes").
			Negative("No")

	case interview.QuestionTypeChoice, interview.QuestionTypePriority:
		// Single choice from list
		var options []huh.Option[string]
		for _, choice := range q.Choices {
			options = append(options, huh.NewOption(choice, choice))
		}

		defaultVal := ""
		if hasPrev {
			defaultVal = prevAnswer.Value
		}

		field = huh.NewSelect[string]().
			Key(q.ID).
			Title(q.Text).
			Description(q.Description).
			Options(options...).
			Value(&defaultVal).
			Validate(func(s string) error {
				if q.Required && s == "" {
					return fmt.Errorf("please select an option")
				}
				return nil
			})

	default:
		return fmt.Errorf("unsupported question type: %s", q.Type)
	}

	// Create form with single question
	m.form = huh.NewForm(
		huh.NewGroup(field).
			Title(m.formatProgress()).
			Description(m.formatHelp()),
	)

	return nil
}

// formatProgress returns a formatted progress string
func (m *InterviewModel) formatProgress() string {
	session := m.engine.GetSession()
	totalQuestions := len(session.Questions)
	currentQuestion := session.Current + 1
	progress := int(m.engine.Progress())

	return fmt.Sprintf("Question %d/%d (%d%%)", currentQuestion, totalQuestions, progress)
}

// formatHelp returns help text
func (m *InterviewModel) formatHelp() string {
	return "Use arrow keys to navigate • Enter to submit • Ctrl+C to quit"
}

// Init initializes the model
func (m *InterviewModel) Init() tea.Cmd {
	if m.form != nil {
		return m.form.Init()
	}
	return nil
}

// Update handles messages and updates the model
func (m *InterviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	// If completed or error, handle quit
	if m.completed || m.err != nil {
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}
		return m, nil
	}

	// Update form
	if m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f

			// Check if form is complete
			if m.form.State == huh.StateCompleted {
				// Submit answer
				if err := m.submitAnswer(); err != nil {
					m.err = err
					return m, nil
				}

				// Create next question form
				if err := m.createQuestionForm(); err != nil {
					// Error already set in createQuestionForm if not completed
					if !m.completed {
						return m, nil
					}
				}

				// If completed, return to trigger final view
				if m.completed {
					return m, nil
				}

				// Initialize new form
				return m, m.form.Init()
			}
		}
		return m, cmd
	}

	return m, nil
}

// submitAnswer extracts the answer from the form and submits it to the engine
func (m *InterviewModel) submitAnswer() error {
	q, err := m.engine.CurrentQuestion()
	if err != nil {
		return fmt.Errorf("get current question: %w", err)
	}

	var answer interview.Answer

	switch q.Type {
	case interview.QuestionTypeText:
		val := m.form.GetString(q.ID)
		answer.Value = val

	case interview.QuestionTypeMulti:
		val := m.form.GetString(q.ID)
		// Split by newlines for multi-line input
		lines := strings.Split(val, "\n")
		var nonEmpty []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				nonEmpty = append(nonEmpty, trimmed)
			}
		}
		answer.Values = nonEmpty

	case interview.QuestionTypeYesNo:
		val := m.form.GetBool(q.ID)
		if val {
			answer.Value = "yes"
		} else {
			answer.Value = "no"
		}

	case interview.QuestionTypeChoice, interview.QuestionTypePriority:
		val := m.form.GetString(q.ID)
		answer.Value = val
	}

	// Submit to engine
	_, err = m.engine.Answer(answer)
	if err != nil {
		return fmt.Errorf("submit answer: %w", err)
	}

	return nil
}

// View renders the UI
func (m *InterviewModel) View() string {
	if m.quitting {
		return "Interview cancelled.\n"
	}

	if m.err != nil {
		return m.renderError()
	}

	if m.completed && m.result != nil {
		return m.renderCompletion()
	}

	if m.form != nil {
		return m.form.View()
	}

	return "Loading...\n"
}

// renderError renders the error view
func (m *InterviewModel) renderError() string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(errorStyle.Render("Error: ") + m.err.Error())
	b.WriteString("\n\n")
	b.WriteString("Press any key to exit.\n")

	return b.String()
}

// renderCompletion renders the completion summary
func (m *InterviewModel) renderCompletion() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")).
		Bold(true)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("✓ Interview Complete!"))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Product: "))
	b.WriteString(valueStyle.Render(m.result.Spec.Product))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Features: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", len(m.result.Spec.Features))))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Generation time: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%dms", m.result.Duration)))
	b.WriteString("\n\n")

	b.WriteString("Specification generated successfully!\n")
	b.WriteString("Press any key to exit.\n")

	return b.String()
}

// RunInterview starts the TUI interview
func RunInterview(engine *interview.Engine) (*interview.InterviewResult, error) {
	model, err := NewInterviewModel(engine)
	if err != nil {
		return nil, fmt.Errorf("create interview model: %w", err)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("run TUI: %w", err)
	}

	m, ok := finalModel.(*InterviewModel)
	if !ok {
		return nil, fmt.Errorf("invalid final model type")
	}

	if m.err != nil {
		return nil, m.err
	}

	if !m.completed || m.result == nil {
		return nil, fmt.Errorf("interview not completed")
	}

	return m.result, nil
}

// SaveResult saves the interview result to a spec file
func SaveResult(result *interview.InterviewResult, outputPath string) error {
	if err := spec.SaveSpec(result.Spec, outputPath); err != nil {
		return fmt.Errorf("save spec: %w", err)
	}
	return nil
}
