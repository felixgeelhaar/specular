// Package tui provides terminal user interface components for the Specular CLI.
package tui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommandType identifies the type of command being executed
type CommandType string

const (
	CommandBuild CommandType = "build"
	CommandPlan  CommandType = "plan"
	CommandEval  CommandType = "eval"
	CommandAuto  CommandType = "auto"
)

// CommandState represents the current state of command execution
type CommandState string

const (
	StateInitializing CommandState = "initializing"
	StateRunning      CommandState = "running"
	StateWaiting      CommandState = "waiting"
	StateCompleted    CommandState = "completed"
	StateFailed       CommandState = "failed"
	StateCancelled    CommandState = "cancelled"
)

// CommandPhase represents a phase in the command execution
type CommandPhase struct {
	Name        string
	Description string
	Status      PhaseStatus
	StartTime   time.Time
	EndTime     time.Time
	Error       error
	Details     map[string]interface{}
}

// PhaseStatus indicates the status of a phase
type PhaseStatus string

const (
	PhasePending   PhaseStatus = "pending"
	PhaseRunning   PhaseStatus = "running"
	PhaseCompleted PhaseStatus = "completed"
	PhaseFailed    PhaseStatus = "failed"
	PhaseSkipped   PhaseStatus = "skipped"
)

// CommandTUIConfig configures the command TUI
type CommandTUIConfig struct {
	Command     CommandType
	Title       string
	Description string
	Phases      []string
	ShowLogs    bool
	ShowMetrics bool
	Interactive bool
}

// CommandTUIModel is the generic command TUI model
type CommandTUIModel struct {
	// Configuration
	config CommandTUIConfig

	// Command state
	state       CommandState
	phases      []CommandPhase
	currentPhase int
	startTime   time.Time

	// UI state
	width       int
	height      int
	ready       bool
	quitting    bool
	activeTab   int
	logScroll   int

	// Data
	logs        []LogEntry
	maxLogs     int
	metrics     map[string]interface{}
	outputs     []OutputItem
	lastError   string

	// Styles
	styles CommandStyles
}

// CommandStyles contains styles for the command TUI
type CommandStyles struct {
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Phase       lipgloss.Style
	PhaseActive lipgloss.Style
	PhaseDone   lipgloss.Style
	PhaseFailed lipgloss.Style
	Status      lipgloss.Style
	Error       lipgloss.Style
	Success     lipgloss.Style
	Warning     lipgloss.Style
	Muted       lipgloss.Style
	Border      lipgloss.Style
	Tab         lipgloss.Style
	TabActive   lipgloss.Style
	Key         lipgloss.Style
	KeyDesc     lipgloss.Style
}

// OutputItem represents an output file or result
type OutputItem struct {
	Name     string
	Path     string
	Size     int64
	Created  time.Time
	Metadata map[string]string
}

// DefaultCommandStyles returns the default command TUI styles
func DefaultCommandStyles() CommandStyles {
	return CommandStyles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1),
		Phase: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		PhaseActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")),
		PhaseDone: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")),
		PhaseFailed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
		Status: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")),
		Error: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")),
		Success: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46")),
		Warning: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")),
		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2),
		Tab: lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("241")),
		TabActive: lipgloss.NewStyle().
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("63")).
			Background(lipgloss.Color("236")),
		Key: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")),
		KeyDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
	}
}

// NewCommandTUI creates a new command TUI model
func NewCommandTUI(config CommandTUIConfig) *CommandTUIModel {
	phases := make([]CommandPhase, len(config.Phases))
	for i, name := range config.Phases {
		phases[i] = CommandPhase{
			Name:   name,
			Status: PhasePending,
		}
	}

	return &CommandTUIModel{
		config:    config,
		state:     StateInitializing,
		phases:    phases,
		startTime: time.Now(),
		maxLogs:   100,
		logs:      make([]LogEntry, 0, 100),
		metrics:   make(map[string]interface{}),
		outputs:   make([]OutputItem, 0),
		styles:    DefaultCommandStyles(),
	}
}

// Init initializes the command TUI
func (m CommandTUIModel) Init() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

// TickMsg is sent periodically to update the UI
type TickMsg struct{}

// PhaseStartMsg indicates a phase has started
type PhaseStartMsg struct {
	Phase int
	Name  string
}

// PhaseCompleteMsg indicates a phase has completed
type PhaseCompleteMsg struct {
	Phase   int
	Name    string
	Details map[string]interface{}
}

// PhaseFailMsg indicates a phase has failed
type PhaseFailMsg struct {
	Phase int
	Name  string
	Error error
}

// OutputAddedMsg indicates an output was added
type OutputAddedMsg struct {
	Output OutputItem
}

// MetricUpdateMsg updates a metric
type MetricUpdateMsg struct {
	Key   string
	Value interface{}
}

// CommandDoneMsg indicates the command has finished
type CommandDoneMsg struct {
	Success bool
	Error   error
}

// Update handles messages
func (m CommandTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case TickMsg:
		// Continue ticking
		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return TickMsg{}
		})

	case PhaseStartMsg:
		if msg.Phase < len(m.phases) {
			m.phases[msg.Phase].Status = PhaseRunning
			m.phases[msg.Phase].StartTime = time.Now()
			m.currentPhase = msg.Phase
			m.state = StateRunning
		}
		return m, nil

	case PhaseCompleteMsg:
		if msg.Phase < len(m.phases) {
			m.phases[msg.Phase].Status = PhaseCompleted
			m.phases[msg.Phase].EndTime = time.Now()
			m.phases[msg.Phase].Details = msg.Details
		}
		return m, nil

	case PhaseFailMsg:
		if msg.Phase < len(m.phases) {
			m.phases[msg.Phase].Status = PhaseFailed
			m.phases[msg.Phase].EndTime = time.Now()
			m.phases[msg.Phase].Error = msg.Error
			m.lastError = msg.Error.Error()
		}
		return m, nil

	case LogEntryMsg:
		m.addLog(msg.Level, msg.Message)
		return m, nil

	case OutputAddedMsg:
		m.outputs = append(m.outputs, msg.Output)
		return m, nil

	case MetricUpdateMsg:
		m.metrics[msg.Key] = msg.Value
		return m, nil

	case CommandDoneMsg:
		if msg.Success {
			m.state = StateCompleted
		} else {
			m.state = StateFailed
			if msg.Error != nil {
				m.lastError = msg.Error.Error()
			}
		}
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m CommandTUIModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit

	case "tab":
		m.activeTab = (m.activeTab + 1) % 3
		return m, nil

	case "shift+tab":
		m.activeTab = (m.activeTab + 2) % 3
		return m, nil

	case "j", "down":
		if m.activeTab == 1 && m.logScroll < len(m.logs)-10 {
			m.logScroll++
		}
		return m, nil

	case "k", "up":
		if m.activeTab == 1 && m.logScroll > 0 {
			m.logScroll--
		}
		return m, nil

	case "g":
		m.logScroll = 0
		return m, nil

	case "G":
		if len(m.logs) > 10 {
			m.logScroll = len(m.logs) - 10
		}
		return m, nil
	}

	return m, nil
}

// View renders the TUI
func (m CommandTUIModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.quitting {
		return m.renderComplete()
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Content based on active tab
	switch m.activeTab {
	case 0: // Progress
		b.WriteString(m.renderProgress())
	case 1: // Logs
		b.WriteString(m.renderLogs())
	case 2: // Outputs
		b.WriteString(m.renderOutputs())
	}

	b.WriteString("\n")

	// Help line
	b.WriteString(m.renderHelp())

	return b.String()
}

// renderHeader renders the header
func (m CommandTUIModel) renderHeader() string {
	icon := m.getCommandIcon()
	title := m.styles.Title.Render(fmt.Sprintf("%s %s", icon, m.config.Title))

	// Status indicator
	status := m.getStatusText()
	elapsed := formatDuration(time.Since(m.startTime))

	statusText := m.styles.Muted.Render(fmt.Sprintf("(%s • %s)", status, elapsed))

	return title + "  " + statusText
}

// renderTabs renders the tab bar
func (m CommandTUIModel) renderTabs() string {
	tabs := []string{"Progress", "Logs", "Outputs"}
	var parts []string

	for i, tab := range tabs {
		if i == m.activeTab {
			parts = append(parts, m.styles.TabActive.Render(tab))
		} else {
			parts = append(parts, m.styles.Tab.Render(tab))
		}
	}

	return strings.Join(parts, " │ ")
}

// renderProgress renders the progress view
func (m CommandTUIModel) renderProgress() string {
	var b strings.Builder

	// Phase list
	for i, phase := range m.phases {
		icon := m.getPhaseIcon(phase.Status)
		style := m.getPhaseStyle(phase.Status)

		line := fmt.Sprintf("  %s %s", icon, phase.Name)

		if phase.Status == PhaseRunning {
			elapsed := formatDuration(time.Since(phase.StartTime))
			line += m.styles.Muted.Render(fmt.Sprintf(" (%s)", elapsed))
		} else if phase.Status == PhaseCompleted && !phase.EndTime.IsZero() {
			duration := formatDuration(phase.EndTime.Sub(phase.StartTime))
			line += m.styles.Muted.Render(fmt.Sprintf(" (%s)", duration))
		} else if phase.Status == PhaseFailed && phase.Error != nil {
			line += m.styles.Error.Render(fmt.Sprintf(" - %s", truncateText(phase.Error.Error(), 40)))
		}

		b.WriteString(style.Render(line))
		b.WriteString("\n")

		// Show details for completed phases
		if phase.Status == PhaseCompleted && len(phase.Details) > 0 && i == m.currentPhase {
			for k, v := range phase.Details {
				b.WriteString(m.styles.Muted.Render(fmt.Sprintf("      %s: %v\n", k, v)))
			}
		}
	}

	// Progress bar
	if len(m.phases) > 0 {
		b.WriteString("\n")
		b.WriteString(m.renderProgressBar())
	}

	// Metrics
	if len(m.metrics) > 0 {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Subtitle.Render("Metrics"))
		b.WriteString("\n")
		for k, v := range m.metrics {
			b.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
	}

	// Error display
	if m.lastError != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.Error.Render("❌ Error: " + m.lastError))
	}

	return b.String()
}

// renderProgressBar renders the progress bar
func (m CommandTUIModel) renderProgressBar() string {
	completed := 0
	for _, p := range m.phases {
		if p.Status == PhaseCompleted {
			completed++
		}
	}

	total := len(m.phases)
	if total == 0 {
		return ""
	}

	percentage := float64(completed) / float64(total) * 100
	barWidth := 40
	filled := int(percentage / 100 * float64(barWidth))
	empty := barWidth - filled

	bar := fmt.Sprintf("[%s%s] %.0f%%",
		strings.Repeat("█", filled),
		strings.Repeat("░", empty),
		percentage)

	return m.styles.Status.Render(bar)
}

// renderLogs renders the logs view
func (m CommandTUIModel) renderLogs() string {
	if len(m.logs) == 0 {
		return m.styles.Muted.Render("  No logs yet...")
	}

	var b strings.Builder
	maxLines := 15
	if m.height > 0 {
		maxLines = m.height - 10
	}

	start := m.logScroll
	end := start + maxLines
	if end > len(m.logs) {
		end = len(m.logs)
	}

	for i := start; i < end; i++ {
		entry := m.logs[i]
		line := m.formatLogEntry(entry)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.logs) > maxLines {
		scrollInfo := fmt.Sprintf("  [%d-%d of %d]", start+1, end, len(m.logs))
		b.WriteString(m.styles.Muted.Render(scrollInfo))
	}

	return b.String()
}

// renderOutputs renders the outputs view
func (m CommandTUIModel) renderOutputs() string {
	if len(m.outputs) == 0 {
		return m.styles.Muted.Render("  No outputs yet...")
	}

	var b strings.Builder

	for _, output := range m.outputs {
		icon := "📄"
		if strings.HasSuffix(output.Path, ".yaml") || strings.HasSuffix(output.Path, ".yml") {
			icon = "📝"
		} else if strings.HasSuffix(output.Path, ".json") {
			icon = "📋"
		}

		b.WriteString(fmt.Sprintf("  %s %s\n", icon, output.Name))
		b.WriteString(m.styles.Muted.Render(fmt.Sprintf("     Path: %s\n", output.Path)))
		if output.Size > 0 {
			b.WriteString(m.styles.Muted.Render(fmt.Sprintf("     Size: %s\n", formatBytes(output.Size))))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderComplete renders the completion screen
func (m CommandTUIModel) renderComplete() string {
	var b strings.Builder

	icon := "✓"
	color := m.styles.Success
	status := "Completed"

	if m.state == StateFailed {
		icon = "✗"
		color = m.styles.Error
		status = "Failed"
	} else if m.state == StateCancelled {
		icon = "⊘"
		color = m.styles.Warning
		status = "Cancelled"
	}

	duration := formatDuration(time.Since(m.startTime))

	b.WriteString(color.Render(fmt.Sprintf("\n%s %s %s\n", icon, m.config.Title, status)))
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("Duration: %s\n", duration)))

	if len(m.outputs) > 0 {
		b.WriteString(m.styles.Muted.Render(fmt.Sprintf("Outputs: %d files\n", len(m.outputs))))
	}

	if m.lastError != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.Error.Render("Error: " + m.lastError))
		b.WriteString("\n")
	}

	return b.String()
}

// renderHelp renders the help line
func (m CommandTUIModel) renderHelp() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"Tab", "switch tabs"},
		{"j/k", "scroll"},
		{"g/G", "top/bottom"},
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

func (m CommandTUIModel) getCommandIcon() string {
	switch m.config.Command {
	case CommandBuild:
		return "🔨"
	case CommandPlan:
		return "📋"
	case CommandEval:
		return "📊"
	case CommandAuto:
		return "🤖"
	default:
		return "⚙️"
	}
}

func (m CommandTUIModel) getStatusText() string {
	switch m.state {
	case StateInitializing:
		return "initializing"
	case StateRunning:
		return "running"
	case StateWaiting:
		return "waiting"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StateCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

func (m CommandTUIModel) getPhaseIcon(status PhaseStatus) string {
	switch status {
	case PhasePending:
		return "○"
	case PhaseRunning:
		return "◐"
	case PhaseCompleted:
		return "●"
	case PhaseFailed:
		return "✗"
	case PhaseSkipped:
		return "⊘"
	default:
		return "?"
	}
}

func (m CommandTUIModel) getPhaseStyle(status PhaseStatus) lipgloss.Style {
	switch status {
	case PhaseRunning:
		return m.styles.PhaseActive
	case PhaseCompleted:
		return m.styles.PhaseDone
	case PhaseFailed:
		return m.styles.PhaseFailed
	default:
		return m.styles.Phase
	}
}

func (m *CommandTUIModel) addLog(level LogLevel, message string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	if len(m.logs) >= m.maxLogs {
		m.logs = append(m.logs[1:], entry)
	} else {
		m.logs = append(m.logs, entry)
	}
}

func (m CommandTUIModel) formatLogEntry(entry LogEntry) string {
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
	return fmt.Sprintf("  %s %s %s", timestamp, levelStr, entry.Message)
}

// formatBytes formats bytes as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// CommandTUIAdapter provides an interface for commands to interact with the TUI
type CommandTUIAdapter struct {
	program *tea.Program
	model   *CommandTUIModel
	ctx     context.Context
	cancel  context.CancelFunc
	output  io.Writer
}

// NewCommandTUIAdapter creates a new command TUI adapter
func NewCommandTUIAdapter(config CommandTUIConfig, output io.Writer) *CommandTUIAdapter {
	model := NewCommandTUI(config)

	return &CommandTUIAdapter{
		model:  model,
		output: output,
	}
}

// Start starts the TUI
func (a *CommandTUIAdapter) Start() error {
	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.program = tea.NewProgram(*a.model, tea.WithOutput(a.output))

	go func() {
		if _, err := a.program.Run(); err != nil {
			_, _ = fmt.Fprintf(a.output, "TUI error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the TUI
func (a *CommandTUIAdapter) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	if a.program != nil {
		a.program.Quit()
	}
}

// StartPhase notifies that a phase has started
func (a *CommandTUIAdapter) StartPhase(phase int, name string) {
	if a.program != nil {
		a.program.Send(PhaseStartMsg{Phase: phase, Name: name})
	}
}

// CompletePhase notifies that a phase has completed
func (a *CommandTUIAdapter) CompletePhase(phase int, name string, details map[string]interface{}) {
	if a.program != nil {
		a.program.Send(PhaseCompleteMsg{Phase: phase, Name: name, Details: details})
	}
}

// FailPhase notifies that a phase has failed
func (a *CommandTUIAdapter) FailPhase(phase int, name string, err error) {
	if a.program != nil {
		a.program.Send(PhaseFailMsg{Phase: phase, Name: name, Error: err})
	}
}

// Log adds a log entry
func (a *CommandTUIAdapter) Log(level LogLevel, message string) {
	if a.program != nil {
		a.program.Send(LogEntryMsg{Level: level, Message: message})
	}
}

// AddOutput adds an output item
func (a *CommandTUIAdapter) AddOutput(output OutputItem) {
	if a.program != nil {
		a.program.Send(OutputAddedMsg{Output: output})
	}
}

// UpdateMetric updates a metric
func (a *CommandTUIAdapter) UpdateMetric(key string, value interface{}) {
	if a.program != nil {
		a.program.Send(MetricUpdateMsg{Key: key, Value: value})
	}
}

// Done signals that the command has finished
func (a *CommandTUIAdapter) Done(success bool, err error) {
	if a.program != nil {
		a.program.Send(CommandDoneMsg{Success: success, Error: err})
	}
	// Wait for user to see the result
	time.Sleep(time.Second)
}
