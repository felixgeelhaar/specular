package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Panel rendering functions for the dashboard

// renderProgressPanel renders the progress panel
func (m *DashboardModel) renderProgressPanel(width int) string {
	var b strings.Builder

	// Progress bar
	progress := float64(m.completedSteps) / float64(max(m.totalSteps, 1))
	barWidth := width - 20
	filled := int(progress * float64(barWidth))
	empty := barWidth - filled

	bar := fmt.Sprintf("[%s%s] %d%%",
		strings.Repeat("█", filled),
		strings.Repeat("░", empty),
		int(progress*100))

	b.WriteString(m.styles.Status.Render(bar))
	b.WriteString("\n\n")

	// Current step
	stepText := fmt.Sprintf("Step %d/%d: %s",
		m.currentStep+1, m.totalSteps, m.currentStepName)
	b.WriteString(stepText)
	b.WriteString("\n")

	// Stats line
	statsLine := fmt.Sprintf("Completed: %d  Failed: %d  Pending: %d",
		m.completedSteps, m.failedSteps,
		m.totalSteps-m.completedSteps-m.failedSteps)
	b.WriteString(m.styles.Muted.Render(statsLine))
	b.WriteString("\n\n")

	// Timing
	elapsed := time.Since(m.startTime)
	eta := m.GetETA()

	timingLine := fmt.Sprintf("Elapsed: %s", formatDuration(elapsed))
	if eta > 0 {
		timingLine += fmt.Sprintf("   ETA: ~%s", formatDuration(eta))
	}
	b.WriteString(m.styles.Muted.Render(timingLine))

	// Error display
	if m.lastError != "" {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Error.Render("❌ " + truncateText(m.lastError, width-5)))
	}

	return b.String()
}

// renderBudgetPanel renders the budget panel
func (m *DashboardModel) renderBudgetPanel(width int) string {
	var b strings.Builder

	// Budget header
	b.WriteString(m.styles.Subtitle.Render("BUDGET"))
	b.WriteString("\n\n")

	// Limit, Spent, Remaining
	b.WriteString(fmt.Sprintf("Limit:     $%.2f\n", m.budgetLimit))

	// Color code spent amount
	spentStyle := m.styles.Muted
	percentage := m.GetBudgetPercentage()
	if percentage >= 90 {
		spentStyle = m.styles.Error
	} else if percentage >= 75 {
		spentStyle = m.styles.Warning
	}
	b.WriteString(spentStyle.Render(fmt.Sprintf("Spent:     $%.2f\n", m.budgetSpent)))
	b.WriteString(fmt.Sprintf("Remaining: $%.2f\n", m.budgetRemaining))
	b.WriteString("\n")

	// Budget bar
	barWidth := width - 4
	filled := int(percentage / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	barStyle := m.styles.Success
	if percentage >= 90 {
		barStyle = m.styles.Error
	} else if percentage >= 75 {
		barStyle = m.styles.Warning
	} else if percentage >= 50 {
		barStyle = m.styles.Status
	}

	bar := barStyle.Render(strings.Repeat("█", filled)) +
		m.styles.Muted.Render(strings.Repeat("░", empty))
	b.WriteString(fmt.Sprintf("[%s] %.0f%%\n", bar, percentage))
	b.WriteString("\n")

	// API metrics
	b.WriteString(m.styles.Subtitle.Render("API Metrics"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Calls: %d", m.apiCallCount))
	if m.apiCallRate > 0 {
		b.WriteString(fmt.Sprintf(" (%.1f/min)", m.apiCallRate))
	}

	// Warning if set
	if m.budgetWarning != "" {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Warning.Render("⚠️  " + m.budgetWarning))
	}

	return b.String()
}

// renderLogsPanel renders the logs panel
func (m *DashboardModel) renderLogsPanel(width, height int) string {
	var b strings.Builder

	// Logs header
	headerStyle := m.styles.Subtitle
	if m.activePanel == PanelLogs {
		headerStyle = m.styles.Status
	}
	b.WriteString(headerStyle.Render("LOGS"))
	b.WriteString("\n")

	if len(m.logEntries) == 0 {
		b.WriteString(m.styles.Muted.Render("No log entries yet..."))
		return b.String()
	}

	// Calculate visible lines
	maxLines := height - 3 // Account for header and padding
	if maxLines < 3 {
		maxLines = 3
	}

	// Get log entries to display
	startIdx := len(m.logEntries) - maxLines - m.logScrollPos
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + maxLines
	if endIdx > len(m.logEntries) {
		endIdx = len(m.logEntries)
	}

	for i := startIdx; i < endIdx; i++ {
		entry := m.logEntries[i]
		line := m.formatLogEntry(entry, width-2)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator
	if m.logScrollPos > 0 || startIdx > 0 {
		scrollInfo := fmt.Sprintf("↑ %d more", startIdx)
		b.WriteString(m.styles.Muted.Render(scrollInfo))
	}

	return b.String()
}

// formatLogEntry formats a single log entry for display
func (m *DashboardModel) formatLogEntry(entry LogEntry, maxWidth int) string {
	// Format: HH:MM:SS [LEVEL] message
	timestamp := entry.Timestamp.Format("15:04:05")

	var levelStyle lipgloss.Style
	switch entry.Level {
	case LogLevelError:
		levelStyle = m.styles.Error
	case LogLevelWarn:
		levelStyle = m.styles.Warning
	case LogLevelInfo:
		levelStyle = m.styles.Status
	default:
		levelStyle = m.styles.Muted
	}

	levelStr := levelStyle.Render(fmt.Sprintf("[%s]", entry.Level))

	// Calculate available width for message
	prefixLen := len(timestamp) + 1 + len(string(entry.Level)) + 3 // "HH:MM:SS [LEVEL] "
	messageWidth := maxWidth - prefixLen
	if messageWidth < 10 {
		messageWidth = 10
	}

	message := truncateText(entry.Message, messageWidth)

	return fmt.Sprintf("%s %s %s", timestamp, levelStr, message)
}

// renderDashboard renders the complete dashboard layout
func (m *DashboardModel) renderDashboard() string {
	// Calculate dimensions
	width := m.width
	if width == 0 {
		width = 80 // Default width
	}
	height := m.height
	if height == 0 {
		height = 24 // Default height
	}

	var b strings.Builder

	// Header
	header := m.renderDashboardHeader(width)
	b.WriteString(header)
	b.WriteString("\n")

	// Calculate panel dimensions
	progressWidth := (width * 55) / 100
	budgetWidth := width - progressWidth - 3 // -3 for separator

	// Top row: Progress + Budget
	progressPanel := m.styles.Border.Width(progressWidth - 4).Render(
		m.renderProgressPanel(progressWidth - 6))
	budgetPanel := m.styles.Border.Width(budgetWidth - 4).Render(
		m.renderBudgetPanel(budgetWidth - 6))

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, progressPanel, " ", budgetPanel))
	b.WriteString("\n")

	// Bottom row: Logs
	logsHeight := height - 15 // Reserve space for header and top panels
	if logsHeight < 5 {
		logsHeight = 5
	}
	logsPanel := m.styles.Border.Width(width - 4).Height(logsHeight).Render(
		m.renderLogsPanel(width-6, logsHeight))
	b.WriteString(logsPanel)
	b.WriteString("\n")

	// Help line
	b.WriteString(m.renderDashboardHelp())

	return b.String()
}

// renderDashboardHeader renders the dashboard header
func (m *DashboardModel) renderDashboardHeader(width int) string {
	title := m.styles.Title.Render("🤖 SPECULAR Auto Mode")

	goalText := truncateText(m.goal, width-40)
	goal := m.styles.Muted.Render(goalText)

	profile := m.styles.Highlighted.Render(m.profile)

	// Right-align profile
	padding := width - lipgloss.Width(title) - lipgloss.Width(goal) - lipgloss.Width(profile) - 4
	if padding < 1 {
		padding = 1
	}

	return title + "  " + goal + strings.Repeat(" ", padding) + profile
}

// renderDashboardHelp renders the help line for dashboard
func (m *DashboardModel) renderDashboardHelp() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"Tab", "panels"},
		{"j/k", "scroll"},
		{"s", "steps"},
		{"b", "budget"},
		{"?", "help"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		part := m.styles.Key.Render("["+k.key+"]") + " " + m.styles.KeyDesc.Render(k.desc)
		parts = append(parts, part)
	}

	return strings.Join(parts, "  ")
}

// Helper functions

// truncateText truncates text to maxLen with ellipsis
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
