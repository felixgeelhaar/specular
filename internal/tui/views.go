package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/felixgeelhaar/specular/internal/auto"
)

// renderMain renders the main view showing progress and status
func (m Model) renderMain() string {
	var b strings.Builder

	// Title
	title := m.styles.Title.Render("🤖 Specular Auto Mode")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Goal
	goalLabel := m.styles.Muted.Render("Goal: ")
	goalText := m.styles.Subtitle.Render(m.goal)
	b.WriteString(goalLabel + goalText)
	b.WriteString("\n\n")

	// Profile
	if m.profile != "" {
		profileLabel := m.styles.Muted.Render("Profile: ")
		profileText := m.styles.Subtitle.Render(m.profile)
		b.WriteString(profileLabel + profileText)
		b.WriteString("\n\n")
	}

	// Progress section
	progressBox := m.renderProgressBox()
	b.WriteString(progressBox)
	b.WriteString("\n\n")

	// Current step
	if m.currentStepName != "" {
		currentLabel := m.styles.Muted.Render("Current Step: ")
		currentText := m.styles.Status.Render(m.currentStepName)
		b.WriteString(currentLabel + currentText)
		b.WriteString("\n\n")
	}

	// Error display
	if m.lastError != "" {
		errorBox := m.styles.Border.
			BorderForeground(lipgloss.Color("196")). // Red border
			Render(m.styles.Error.Render("❌ Error: ") + m.lastError)
		b.WriteString(errorBox)
		b.WriteString("\n\n")
	}

	// Help text
	b.WriteString(m.renderHelpLine())

	return b.String()
}

// renderProgressBox renders the progress statistics box
func (m Model) renderProgressBox() string {
	var b strings.Builder

	// Status icon and text
	icon := m.statusIcon()
	statusStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.statusColor())

	status := statusStyle.Render(fmt.Sprintf("%s Progress", icon))
	b.WriteString(status)
	b.WriteString("\n\n")

	// Progress bar
	progressBar := m.renderProgressBar()
	b.WriteString(progressBar)
	b.WriteString("\n\n")

	// Statistics
	stats := m.renderStats()
	b.WriteString(stats)

	return m.styles.Border.Render(b.String())
}

// renderProgressBar renders an ASCII progress bar
func (m Model) renderProgressBar() string {
	if m.totalSteps == 0 {
		return m.styles.Muted.Render("No steps yet")
	}

	barWidth := 40
	filled := int(float64(m.completedSteps) / float64(m.totalSteps) * float64(barWidth))

	var bar strings.Builder
	bar.WriteString("[")
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar.WriteString("█")
		} else {
			bar.WriteString("░")
		}
	}
	bar.WriteString("]")

	percentage := m.progressPercentage()
	progressText := fmt.Sprintf(" %d/%d (%.0f%%)", m.completedSteps, m.totalSteps, percentage)

	return m.styles.Status.Render(bar.String()) + m.styles.Muted.Render(progressText)
}

// renderStats renders execution statistics
func (m Model) renderStats() string {
	elapsed := m.elapsed()

	stats := []string{
		fmt.Sprintf("Completed: %s", m.styles.Success.Render(fmt.Sprintf("%d", m.completedSteps))),
		fmt.Sprintf("Pending:   %s", m.styles.Muted.Render(fmt.Sprintf("%d", m.totalSteps-m.completedSteps-m.failedSteps))),
	}

	if m.failedSteps > 0 {
		stats = append(stats, fmt.Sprintf("Failed:    %s", m.styles.Error.Render(fmt.Sprintf("%d", m.failedSteps))))
	}

	stats = append(stats,
		fmt.Sprintf("Cost:      %s", m.styles.Warning.Render(fmt.Sprintf("$%.4f", m.totalCost))),
		fmt.Sprintf("Elapsed:   %s", m.styles.Muted.Render(formatDuration(elapsed))),
	)

	return strings.Join(stats, "\n")
}

// renderStepList renders the step list view
func (m Model) renderStepList() string {
	var b strings.Builder

	// Title
	title := m.styles.Title.Render("📋 Step List")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Check if action plan is available
	if m.actionPlan == nil || len(m.actionPlan.Steps) == 0 {
		b.WriteString(m.styles.Muted.Render("No steps available yet"))
		b.WriteString("\n\n")
		b.WriteString(m.renderHelpLine())
		return b.String()
	}

	// Render each step
	for i, step := range m.actionPlan.Steps {
		stepLine := m.renderStepLine(i, &step)
		b.WriteString(stepLine)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.renderHelpLine())

	return b.String()
}

// renderStepLine renders a single step in the list
func (m Model) renderStepLine(index int, step *auto.ActionStep) string {
	var b strings.Builder

	// Status icon
	var icon string
	var style lipgloss.Style
	switch step.Status {
	case auto.StepStatusCompleted:
		icon = "✓"
		style = m.styles.Success
	case auto.StepStatusInProgress:
		icon = "⟳"
		style = m.styles.Status
	case auto.StepStatusFailed:
		icon = "✗"
		style = m.styles.Error
	default: // Pending
		icon = "○"
		style = m.styles.Muted
	}

	// Highlight current step
	if index == m.currentStep {
		icon = m.styles.Highlighted.Render(fmt.Sprintf(" %s ", icon))
	} else {
		icon = style.Render(icon)
	}

	b.WriteString(icon)
	b.WriteString(" ")

	// Step info
	stepText := fmt.Sprintf("%s - %s", step.ID, step.Description)
	if index == m.currentStep {
		stepText = m.styles.Status.Bold(true).Render(stepText)
	} else {
		stepText = style.Render(stepText)
	}

	b.WriteString(stepText)

	// Step type
	typeText := m.styles.Muted.Render(fmt.Sprintf(" (%s)", step.Type))
	b.WriteString(typeText)

	return b.String()
}

// renderApproval renders the approval view
func (m Model) renderApproval() string {
	var b strings.Builder

	// Title
	title := m.styles.Title.Render("🔐 Approval Required")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Plan summary
	summaryBox := m.styles.Border.Render(m.approvalPlan)
	b.WriteString(summaryBox)
	b.WriteString("\n\n")

	// Prompt
	prompt := m.styles.Status.Bold(true).Render("Approve execution plan?")
	b.WriteString(prompt)
	b.WriteString("\n\n")

	// Options
	yesOption := m.styles.Key.Render("[y/Enter]") + " " + m.styles.KeyDesc.Render("Yes, approve")
	noOption := m.styles.Key.Render("[n/Esc]") + " " + m.styles.KeyDesc.Render("No, reject")
	b.WriteString(yesOption + "  " + noOption)

	return b.String()
}

// renderHelp renders the help view
func (m Model) renderHelp() string {
	var b strings.Builder

	// Title
	title := m.styles.Title.Render("❓ Help")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Hotkeys
	hotkeys := []struct {
		key  string
		desc string
	}{
		{"?", "Toggle help"},
		{"s", "Toggle step list"},
		{"v", "Toggle verbose mode"},
		{"q", "Quit"},
		{"Ctrl+C", "Force quit"},
		{"Esc", "Return to main view"},
	}

	for _, hk := range hotkeys {
		keyText := m.styles.Key.Render(fmt.Sprintf("%-10s", hk.key))
		descText := m.styles.KeyDesc.Render(hk.desc)
		b.WriteString(keyText + " " + descText)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Muted.Render("Press ? or Esc to return to main view"))

	return b.String()
}

// renderComplete renders the completion screen
func (m Model) renderComplete() string {
	var b strings.Builder

	if m.lastError != "" {
		// Failure
		title := m.styles.Error.Render("❌ Workflow Failed")
		b.WriteString(title)
		b.WriteString("\n\n")
		b.WriteString(m.styles.Muted.Render("Error: ") + m.lastError)
	} else {
		// Success
		title := m.styles.Success.Render("✅ Workflow Complete!")
		b.WriteString(title)
		b.WriteString("\n\n")
		stats := []string{
			fmt.Sprintf("Completed: %d/%d steps", m.completedSteps, m.totalSteps),
			fmt.Sprintf("Cost: $%.4f", m.totalCost),
			fmt.Sprintf("Duration: %s", formatDuration(m.elapsed())),
		}
		b.WriteString(strings.Join(stats, "\n"))
	}

	return b.String()
}

// renderHelpLine renders the help line at the bottom
func (m Model) renderHelpLine() string {
	helpItems := []string{
		m.styles.Key.Render("?") + " help",
		m.styles.Key.Render("s") + " steps",
		m.styles.Key.Render("v") + " verbose",
		m.styles.Key.Render("q") + " quit",
	}

	helpLine := strings.Join(helpItems, " • ")
	return m.styles.Help.Render(helpLine)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}
