package auto

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/specular/internal/domain"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// approvalModel is the bubbletea model for the approval gate
type approvalModel struct {
	plan         *plan.Plan
	spec         *spec.ProductSpec
	featureTitles map[string]string
	approved     bool
	quitting     bool
}

// ShowApprovalGate displays the plan and requests user approval
func ShowApprovalGate(p *plan.Plan, s *spec.ProductSpec) (bool, error) {
	// Build feature title lookup map
	featureTitles := make(map[string]string)
	for _, feature := range s.Features {
		featureTitles[feature.ID.String()] = feature.Title
	}

	model := approvalModel{
		plan:          p,
		spec:          s,
		featureTitles: featureTitles,
	}
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return false, fmt.Errorf("run approval UI: %w", err)
	}

	return finalModel.(approvalModel).approved, nil
}

func (m approvalModel) Init() tea.Cmd {
	return nil
}

func (m approvalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.approved = true
			m.quitting = true
			return m, tea.Quit
		case "n", "N", "q", "ctrl+c":
			m.approved = false
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m approvalModel) View() string {
	if m.quitting {
		if m.approved {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("2")).
				Render("✅ Plan approved! Proceeding with execution...\n")
		}
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Render("❌ Plan rejected. Exiting...\n")
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	var s string

	s += titleStyle.Render("📋 Generated Execution Plan") + "\n\n"

	// Plan summary
	s += fmt.Sprintf("Total Tasks: %s\n", headerStyle.Render(fmt.Sprintf("%d", len(m.plan.Tasks))))

	// Estimate duration (rough: 5 minutes per task)
	estimatedMinutes := len(m.plan.Tasks) * 5
	s += fmt.Sprintf("Estimated Duration: %s\n\n", headerStyle.Render(fmt.Sprintf("~%d minutes", estimatedMinutes)))

	// Show task breakdown by priority
	p0Count, p1Count, p2Count := countTasksByPriority(m.plan.Tasks)

	s += labelStyle.Render("Priority Breakdown:") + "\n"
	s += fmt.Sprintf("  P0 (Critical):     %s\n", renderCount(p0Count))
	s += fmt.Sprintf("  P1 (Important):    %s\n", renderCount(p1Count))
	s += fmt.Sprintf("  P2 (Nice-to-have): %s\n\n", renderCount(p2Count))

	// Show task breakdown by skill
	skillCounts := countTasksBySkill(m.plan.Tasks)
	if len(skillCounts) > 0 {
		s += labelStyle.Render("Skills Required:") + "\n"
		for skill, count := range skillCounts {
			s += fmt.Sprintf("  %-15s %s\n", skill+":", renderCount(count))
		}
		s += "\n"
	}

	// Show first 5 tasks
	s += labelStyle.Render("Task Preview (first 5):") + "\n"
	for i, task := range m.plan.Tasks {
		if i >= 5 {
			break
		}
		priorityColor := getPriorityColor(task.Priority)
		priorityStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(priorityColor))

		// Lookup feature title
		featureTitle := m.featureTitles[task.FeatureID.String()]
		if featureTitle == "" {
			featureTitle = task.FeatureID.String()
		}

		s += fmt.Sprintf("  %d. [%s] %s\n",
			i+1,
			priorityStyle.Render(string(task.Priority)),
			featureTitle)
	}

	if len(m.plan.Tasks) > 5 {
		s += fmt.Sprintf("  ... and %d more tasks\n", len(m.plan.Tasks)-5)
	}

	s += "\n"
	s += titleStyle.Render("Approve and execute?") + " "
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("(y)") + " / "
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("(n)")
	s += ": "

	return s
}

// countTasksByPriority counts tasks by priority level
func countTasksByPriority(tasks []plan.Task) (p0, p1, p2 int) {
	for _, task := range tasks {
		switch task.Priority {
		case domain.PriorityP0:
			p0++
		case domain.PriorityP1:
			p1++
		case domain.PriorityP2:
			p2++
		}
	}
	return
}

// countTasksBySkill counts tasks by skill category
func countTasksBySkill(tasks []plan.Task) map[string]int {
	counts := make(map[string]int)
	for _, task := range tasks {
		if task.Skill != "" {
			counts[task.Skill]++
		}
	}
	return counts
}

// getPriorityColor returns the ANSI color code for a priority level
func getPriorityColor(priority domain.Priority) string {
	switch priority {
	case domain.PriorityP0:
		return "1" // Red
	case domain.PriorityP1:
		return "3" // Yellow
	case domain.PriorityP2:
		return "2" // Green
	default:
		return "8" // Gray
	}
}

// renderCount returns a formatted count with color
func renderCount(count int) string {
	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	if count == 0 {
		countStyle = countStyle.Foreground(lipgloss.Color("8"))
	}

	return countStyle.Render(fmt.Sprintf("%d tasks", count))
}
