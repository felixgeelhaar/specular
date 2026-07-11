package auto

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// ApprovalDecision differentiates the three terminal states of the
// approval gate. Previously the gate collapsed "rejected" and
// "cancelled" into a single false value, which made the intervention
// metric blind to user-aborted runs and surprised callers that needed
// to distinguish "user said no" from "user pressed Esc."
type ApprovalDecision int

const (
	// ApprovalRejected indicates the user explicitly declined the plan
	// (said "no"). It is the zero value so an unset decision defaults to
	// "not approved" rather than accidentally proceeding.
	ApprovalRejected ApprovalDecision = iota
	// ApprovalApproved indicates the user accepted the plan and the run
	// may proceed.
	ApprovalApproved
	// ApprovalCancelled indicates the user aborted the gate (e.g. pressed
	// Esc) without making an explicit approve/reject choice.
	ApprovalCancelled
)

// approvalModel is the bubbletea model for the approval gate
type approvalModel struct {
	plan          *plan.Plan
	spec          *spec.ProductSpec
	featureTitles map[string]string
	decision      ApprovalDecision
	showHelp      bool
	quitting      bool
}

// ShowApprovalGate displays the plan and requests user approval. Returns
// an ApprovalDecision so callers can distinguish approve / reject /
// cancel without overloading the boolean.
func ShowApprovalGate(p *plan.Plan, s *spec.ProductSpec) (ApprovalDecision, error) {
	// Build feature title lookup map
	featureTitles := make(map[string]string)
	for _, feature := range s.Features {
		featureTitles[feature.ID.String()] = feature.Title
	}

	model := approvalModel{
		plan:          p,
		spec:          s,
		featureTitles: featureTitles,
		decision:      ApprovalRejected,
	}
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return ApprovalRejected, fmt.Errorf("run approval UI: %w", err)
	}

	return finalModel.(approvalModel).decision, nil
}

func (m approvalModel) Init() tea.Cmd {
	return nil
}

func (m approvalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			// Enter takes the documented default (approve).
			m.decision = ApprovalApproved
			m.quitting = true
			return m, tea.Quit
		case "n", "N":
			m.decision = ApprovalRejected
			m.quitting = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			// Esc cancels the run without rejecting the plan. The
			// caller can decide whether to retry the gate or abort.
			m.decision = ApprovalCancelled
			m.quitting = true
			return m, tea.Quit
		case "q":
			// q remains a hard quit (cancel) for backwards compatibility
			// with anyone who built muscle memory on the old behaviour.
			m.decision = ApprovalCancelled
			m.quitting = true
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		}
	}
	return m, nil
}

func (m approvalModel) View() string {
	if m.quitting {
		switch m.decision {
		case ApprovalApproved:
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("2")).
				Render("[OK] Plan approved! Proceeding with execution...\n")
		case ApprovalCancelled:
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("3")).
				Render("[ABORT] Plan gate cancelled (no decision recorded). Exiting...\n")
		default:
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("1")).
				Render("[X] Plan rejected. Exiting...\n")
		}
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
	s += "\n\n"

	if m.showHelp {
		s += labelStyle.Render(
			"  y / Y / Enter   approve and proceed\n" +
				"  n / N           reject (records explicit user_reject)\n" +
				"  Esc / q         cancel without rejecting (no intervention recorded)\n" +
				"  ?               toggle this help\n",
		)
	} else {
		s += labelStyle.Render("  y approve · n reject · esc cancel · ? help\n")
	}

	return s
}

// countTasksByPriority counts tasks by priority level
func countTasksByPriority(tasks []plan.Task) (p0, p1, p2 int) {
	for _, task := range tasks {
		switch task.Priority {
		case types.PriorityP0:
			p0++
		case types.PriorityP1:
			p1++
		case types.PriorityP2:
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
func getPriorityColor(priority types.Priority) string {
	switch priority {
	case types.PriorityP0:
		return "1" // Red
	case types.PriorityP1:
		return "3" // Yellow
	case types.PriorityP2:
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
