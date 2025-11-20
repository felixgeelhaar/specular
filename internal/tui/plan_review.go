package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/felixgeelhaar/specular/internal/plan"
)

// PlanReviewResult holds the result of a plan review session
type PlanReviewResult struct {
	Approved bool
	Reason   string
}

// planReviewModel is the BubbleTea model for plan review
type planReviewModel struct {
	plan           *plan.Plan
	cursor         int
	selectedTask   int
	viewMode       string // "list" or "detail"
	approved       *bool  // nil = not decided, true/false = approved/rejected
	rejectionInput string
	editingReason  bool
	result         *PlanReviewResult
	width          int
	height         int
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginLeft(2).
			MarginTop(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			MarginLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Bold(true).
				PaddingLeft(2)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	detailKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Bold(true)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(2).
			MarginTop(1)

	approveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	rejectStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)
)

// Init initializes the model
func (m planReviewModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m planReviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// If editing rejection reason
		if m.editingReason {
			switch msg.String() {
			case "enter":
				m.editingReason = false
				return m, nil
			case "esc":
				m.editingReason = false
				m.rejectionInput = ""
				m.approved = nil
				return m, nil
			case "backspace":
				if len(m.rejectionInput) > 0 {
					m.rejectionInput = m.rejectionInput[:len(m.rejectionInput)-1]
				}
				return m, nil
			default:
				// Add character to input
				if len(msg.String()) == 1 {
					m.rejectionInput += msg.String()
				}
				return m, nil
			}
		}

		// Regular navigation
		switch msg.String() {
		case "ctrl+c", "q":
			// Set result and quit
			if m.approved == nil {
				// Default to rejected if not decided
				approved := false
				m.approved = &approved
				m.result = &PlanReviewResult{
					Approved: false,
					Reason:   "Review cancelled",
				}
			}
			return m, tea.Quit

		case "up", "k":
			if m.viewMode == "list" && m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			if m.viewMode == "list" && m.cursor < len(m.plan.Tasks)-1 {
				m.cursor++
			}
			return m, nil

		case "enter", "right", "l":
			if m.viewMode == "list" {
				m.selectedTask = m.cursor
				m.viewMode = "detail"
			}
			return m, nil

		case "left", "h", "esc":
			if m.viewMode == "detail" {
				m.viewMode = "list"
			}
			return m, nil

		case "a", "A":
			// Approve plan
			approved := true
			m.approved = &approved
			m.result = &PlanReviewResult{
				Approved: true,
				Reason:   "",
			}
			return m, tea.Quit

		case "r", "R":
			// Reject plan - prompt for reason
			rejected := false
			m.approved = &rejected
			m.editingReason = true
			return m, nil
		}
	}

	return m, nil
}

// View renders the current state
func (m planReviewModel) View() string {
	if m.result != nil {
		// Show final result
		if m.result.Approved {
			return approveStyle.Render("\n✓ Plan Approved\n\n")
		}
		reason := m.result.Reason
		if reason == "" {
			reason = "No reason provided"
		}
		return rejectStyle.Render(fmt.Sprintf("\n✗ Plan Rejected\n  Reason: %s\n\n", reason))
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("📋 Plan Review"))
	b.WriteString("\n\n")

	// Overview
	b.WriteString(headerStyle.Render(fmt.Sprintf("Total Tasks: %d", len(m.plan.Tasks))))
	b.WriteString("\n\n")

	if m.viewMode == "list" {
		// List view
		for i, task := range m.plan.Tasks {
			style := itemStyle
			cursor := "  "
			if i == m.cursor {
				style = selectedItemStyle
				cursor = "→ "
			}

			line := fmt.Sprintf("%s[%d] %s | %s | %s",
				cursor,
				i+1,
				task.ID,
				task.Skill,
				task.Priority,
			)
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	} else {
		// Detail view
		task := m.plan.Tasks[m.selectedTask]
		b.WriteString(headerStyle.Render(fmt.Sprintf("Task %d of %d", m.selectedTask+1, len(m.plan.Tasks))))
		b.WriteString("\n\n")

		details := []struct {
			key   string
			value string
		}{
			{"ID", string(task.ID)},
			{"Feature ID", string(task.FeatureID)},
			{"Skill", task.Skill},
			{"Priority", string(task.Priority)},
			{"Model Hint", task.ModelHint},
			{"Estimate", fmt.Sprintf("%d", task.Estimate)},
			{"Expected Hash", task.ExpectedHash},
			{"Dependencies", fmt.Sprintf("%d tasks", len(task.DependsOn))},
		}

		for _, detail := range details {
			b.WriteString("  ")
			b.WriteString(detailKeyStyle.Render(fmt.Sprintf("%-15s:", detail.key)))
			b.WriteString(" ")
			b.WriteString(detailValueStyle.Render(detail.value))
			b.WriteString("\n")
		}

		if len(task.DependsOn) > 0 {
			b.WriteString("\n  ")
			b.WriteString(detailKeyStyle.Render("Depends On:"))
			b.WriteString("\n")
			for _, dep := range task.DependsOn {
				b.WriteString(fmt.Sprintf("    • %s\n", dep))
			}
		}
	}

	b.WriteString("\n")

	// Editing rejection reason
	if m.editingReason {
		b.WriteString(rejectStyle.Render("✗ Rejection Reason:"))
		b.WriteString("\n  ")
		b.WriteString(m.rejectionInput)
		b.WriteString("_")
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("enter: submit | esc: cancel"))
	} else {
		// Help text
		if m.viewMode == "list" {
			b.WriteString(helpStyle.Render("↑/↓: navigate | enter: view details | a: approve | r: reject | q: quit"))
		} else {
			b.WriteString(helpStyle.Render("h/esc: back to list | a: approve | r: reject | q: quit"))
		}
	}

	return b.String()
}

// RunPlanReview launches an interactive TUI for reviewing an execution plan
func RunPlanReview(p *plan.Plan) (*PlanReviewResult, error) {
	if len(p.Tasks) == 0 {
		// Auto-approve empty plans
		return &PlanReviewResult{
			Approved: true,
			Reason:   "",
		}, nil
	}

	model := planReviewModel{
		plan:     p,
		cursor:   0,
		viewMode: "list",
	}

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("running plan review UI: %w", err)
	}

	// Extract result from final model
	m, ok := finalModel.(planReviewModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type: %T", finalModel)
	}

	if m.result != nil {
		return m.result, nil
	}

	// Fallback - should not happen
	return &PlanReviewResult{
		Approved: false,
		Reason:   "Unknown error",
	}, nil
}
