package tui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EvalResult represents the result of an evaluation
type EvalResult struct {
	Name       string
	Provider   string
	Model      string
	Score      float64
	MaxScore   float64
	Duration   time.Duration
	TokensUsed int
	Cost       float64
	Details    map[string]interface{}
	Passed     bool
	Error      error
}

// EvalTUIConfig configures the eval TUI
type EvalTUIConfig struct {
	SpecFile    string
	TestCases   int
	Providers   []string
	ShowDetails bool
	Parallel    bool
}

// EvalTUIModel is the evaluation TUI model
type EvalTUIModel struct {
	// Configuration
	config EvalTUIConfig

	// State
	state      CommandState
	startTime  time.Time
	results    []EvalResult
	currentIdx int
	totalTests int

	// UI state
	width       int
	height      int
	ready       bool
	quitting    bool
	activeTab   int
	resultScroll int

	// Aggregated stats
	totalScore   float64
	maxScore     float64
	passedCount  int
	failedCount  int
	totalTokens  int
	totalCost    float64
	avgDuration  time.Duration

	// Styles
	styles EvalStyles
}

// EvalStyles contains styles for the eval TUI
type EvalStyles struct {
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Pass      lipgloss.Style
	Fail      lipgloss.Style
	Score     lipgloss.Style
	ScoreHigh lipgloss.Style
	ScoreMid  lipgloss.Style
	ScoreLow  lipgloss.Style
	Muted     lipgloss.Style
	Border    lipgloss.Style
	Tab       lipgloss.Style
	TabActive lipgloss.Style
	Key       lipgloss.Style
	KeyDesc   lipgloss.Style
	Progress  lipgloss.Style
}

// DefaultEvalStyles returns default eval TUI styles
func DefaultEvalStyles() EvalStyles {
	return EvalStyles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1),
		Pass: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46")),
		Fail: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")),
		Score: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")),
		ScoreHigh: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46")),
		ScoreMid: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")),
		ScoreLow: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")),
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
		Progress: lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")),
	}
}

// NewEvalTUI creates a new eval TUI model
func NewEvalTUI(config EvalTUIConfig) *EvalTUIModel {
	return &EvalTUIModel{
		config:     config,
		state:      StateInitializing,
		startTime:  time.Now(),
		results:    make([]EvalResult, 0),
		totalTests: config.TestCases * len(config.Providers),
		styles:     DefaultEvalStyles(),
	}
}

// EvalStartMsg indicates evaluation has started
type EvalStartMsg struct {
	TotalTests int
}

// EvalResultMsg indicates a single eval result
type EvalResultMsg struct {
	Result EvalResult
}

// EvalCompleteMsg indicates evaluation is complete
type EvalCompleteMsg struct {
	Success bool
	Error   error
}

// Init initializes the eval TUI
func (m EvalTUIModel) Init() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

// Update handles messages
func (m EvalTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case TickMsg:
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return TickMsg{}
		})

	case EvalStartMsg:
		m.state = StateRunning
		m.totalTests = msg.TotalTests
		return m, nil

	case EvalResultMsg:
		m.results = append(m.results, msg.Result)
		m.currentIdx++
		m.updateStats(msg.Result)
		return m, nil

	case EvalCompleteMsg:
		if msg.Success {
			m.state = StateCompleted
		} else {
			m.state = StateFailed
		}
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m EvalTUIModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit

	case "tab":
		m.activeTab = (m.activeTab + 1) % 3
		return m, nil

	case "j", "down":
		if m.activeTab == 1 && m.resultScroll < len(m.results)-5 {
			m.resultScroll++
		}
		return m, nil

	case "k", "up":
		if m.activeTab == 1 && m.resultScroll > 0 {
			m.resultScroll--
		}
		return m, nil
	}

	return m, nil
}

// View renders the TUI
func (m EvalTUIModel) View() string {
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
	case 0: // Progress/Summary
		b.WriteString(m.renderSummary())
	case 1: // Results
		b.WriteString(m.renderResults())
	case 2: // Comparison
		b.WriteString(m.renderComparison())
	}

	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

// renderHeader renders the header
func (m EvalTUIModel) renderHeader() string {
	title := m.styles.Title.Render("📊 Evaluation")

	progress := fmt.Sprintf("%d/%d", m.currentIdx, m.totalTests)
	elapsed := formatDuration(time.Since(m.startTime))
	statusText := m.styles.Muted.Render(fmt.Sprintf("(%s • %s)", progress, elapsed))

	return title + "  " + statusText
}

// renderTabs renders the tab bar
func (m EvalTUIModel) renderTabs() string {
	tabs := []string{"Summary", "Results", "Comparison"}
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

// renderSummary renders the summary view
func (m EvalTUIModel) renderSummary() string {
	var b strings.Builder

	// Progress bar
	if m.totalTests > 0 {
		progress := float64(m.currentIdx) / float64(m.totalTests) * 100
		barWidth := 40
		filled := int(progress / 100 * float64(barWidth))
		empty := barWidth - filled

		bar := fmt.Sprintf("[%s%s] %.0f%%",
			strings.Repeat("█", filled),
			strings.Repeat("░", empty),
			progress)
		b.WriteString(m.styles.Progress.Render(bar))
		b.WriteString("\n\n")
	}

	// Stats
	b.WriteString(m.styles.Subtitle.Render("Statistics"))
	b.WriteString("\n")

	// Pass/Fail counts
	passStyle := m.styles.Pass
	failStyle := m.styles.Fail
	b.WriteString(fmt.Sprintf("  Passed: %s  Failed: %s\n",
		passStyle.Render(fmt.Sprintf("%d", m.passedCount)),
		failStyle.Render(fmt.Sprintf("%d", m.failedCount))))

	// Overall score
	if m.maxScore > 0 {
		percentage := m.totalScore / m.maxScore * 100
		scoreStyle := m.getScoreStyle(percentage)
		b.WriteString(fmt.Sprintf("  Score:  %s (%.1f/%.1f)\n",
			scoreStyle.Render(fmt.Sprintf("%.1f%%", percentage)),
			m.totalScore, m.maxScore))
	}

	b.WriteString("\n")

	// Cost and token stats
	b.WriteString(m.styles.Subtitle.Render("Resources"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Tokens: %d\n", m.totalTokens))
	b.WriteString(fmt.Sprintf("  Cost:   $%.4f\n", m.totalCost))
	if len(m.results) > 0 {
		avgDur := time.Duration(int64(m.avgDuration) / int64(len(m.results)))
		b.WriteString(fmt.Sprintf("  Avg Time: %s\n", formatDuration(avgDur)))
	}

	return b.String()
}

// renderResults renders the results list
func (m EvalTUIModel) renderResults() string {
	if len(m.results) == 0 {
		return m.styles.Muted.Render("  No results yet...")
	}

	var b strings.Builder
	maxLines := 10
	if m.height > 0 {
		maxLines = m.height - 10
	}

	start := m.resultScroll
	end := start + maxLines
	if end > len(m.results) {
		end = len(m.results)
	}

	for i := start; i < end; i++ {
		result := m.results[i]
		icon := "✓"
		style := m.styles.Pass
		if !result.Passed {
			icon = "✗"
			style = m.styles.Fail
		}

		line := fmt.Sprintf("  %s %s", icon, result.Name)
		if result.Provider != "" {
			line += m.styles.Muted.Render(fmt.Sprintf(" [%s]", result.Provider))
		}

		// Score
		if result.MaxScore > 0 {
			percentage := result.Score / result.MaxScore * 100
			scoreStyle := m.getScoreStyle(percentage)
			line += fmt.Sprintf(" %s", scoreStyle.Render(fmt.Sprintf("%.0f%%", percentage)))
		}

		// Duration
		line += m.styles.Muted.Render(fmt.Sprintf(" (%s)", formatDuration(result.Duration)))

		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.results) > maxLines {
		scrollInfo := fmt.Sprintf("  [%d-%d of %d]", start+1, end, len(m.results))
		b.WriteString(m.styles.Muted.Render(scrollInfo))
	}

	return b.String()
}

// renderComparison renders the comparison view
func (m EvalTUIModel) renderComparison() string {
	// Group results by provider
	providers := make(map[string][]EvalResult)
	for _, r := range m.results {
		if r.Provider == "" {
			continue
		}
		providers[r.Provider] = append(providers[r.Provider], r)
	}

	if len(providers) == 0 {
		return m.styles.Muted.Render("  No multi-provider results to compare...")
	}

	var b strings.Builder
	b.WriteString(m.styles.Subtitle.Render("Provider Comparison"))
	b.WriteString("\n\n")

	// Calculate stats per provider
	for provider, results := range providers {
		var totalScore, maxScore float64
		var passed, total int
		var totalCost float64
		var totalDuration time.Duration

		for _, r := range results {
			totalScore += r.Score
			maxScore += r.MaxScore
			if r.Passed {
				passed++
			}
			total++
			totalCost += r.Cost
			totalDuration += r.Duration
		}

		percentage := float64(0)
		if maxScore > 0 {
			percentage = totalScore / maxScore * 100
		}
		scoreStyle := m.getScoreStyle(percentage)

		avgDur := time.Duration(0)
		if total > 0 {
			avgDur = totalDuration / time.Duration(total)
		}

		b.WriteString(fmt.Sprintf("  %s\n", m.styles.Subtitle.Render(provider)))
		b.WriteString(fmt.Sprintf("    Score:    %s\n", scoreStyle.Render(fmt.Sprintf("%.1f%%", percentage))))
		b.WriteString(fmt.Sprintf("    Pass Rate: %d/%d\n", passed, total))
		b.WriteString(fmt.Sprintf("    Avg Time:  %s\n", formatDuration(avgDur)))
		b.WriteString(fmt.Sprintf("    Cost:      $%.4f\n", totalCost))
		b.WriteString("\n")
	}

	return b.String()
}

// renderComplete renders the completion view
func (m EvalTUIModel) renderComplete() string {
	var b strings.Builder

	icon := "✓"
	color := m.styles.Pass
	status := "Completed"

	if m.state == StateFailed {
		icon = "✗"
		color = m.styles.Fail
		status = "Failed"
	}

	duration := formatDuration(time.Since(m.startTime))

	b.WriteString(color.Render(fmt.Sprintf("\n%s Evaluation %s\n", icon, status)))
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("Duration: %s\n", duration)))
	b.WriteString(fmt.Sprintf("Results: %d passed, %d failed\n", m.passedCount, m.failedCount))

	if m.maxScore > 0 {
		percentage := m.totalScore / m.maxScore * 100
		b.WriteString(fmt.Sprintf("Overall Score: %.1f%%\n", percentage))
	}

	b.WriteString(fmt.Sprintf("Total Cost: $%.4f\n", m.totalCost))

	return b.String()
}

// renderHelp renders the help line
func (m EvalTUIModel) renderHelp() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"Tab", "switch tabs"},
		{"j/k", "scroll"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		part := m.styles.Key.Render("["+k.key+"]") + " " + m.styles.KeyDesc.Render(k.desc)
		parts = append(parts, part)
	}

	return strings.Join(parts, "  ")
}

// updateStats updates aggregated statistics
func (m *EvalTUIModel) updateStats(result EvalResult) {
	m.totalScore += result.Score
	m.maxScore += result.MaxScore
	m.totalTokens += result.TokensUsed
	m.totalCost += result.Cost
	m.avgDuration += result.Duration

	if result.Passed {
		m.passedCount++
	} else {
		m.failedCount++
	}
}

// getScoreStyle returns the appropriate style for a score percentage
func (m EvalTUIModel) getScoreStyle(percentage float64) lipgloss.Style {
	if percentage >= 80 {
		return m.styles.ScoreHigh
	} else if percentage >= 60 {
		return m.styles.ScoreMid
	}
	return m.styles.ScoreLow
}

// EvalTUIAdapter provides an interface for the eval command to interact with the TUI
type EvalTUIAdapter struct {
	program *tea.Program
	model   *EvalTUIModel
	output  io.Writer
}

// NewEvalTUIAdapter creates a new eval TUI adapter
func NewEvalTUIAdapter(config EvalTUIConfig, output io.Writer) *EvalTUIAdapter {
	if output == nil {
		output = os.Stdout
	}

	model := NewEvalTUI(config)

	return &EvalTUIAdapter{
		model:  model,
		output: output,
	}
}

// Start starts the TUI
func (a *EvalTUIAdapter) Start() error {
	a.program = tea.NewProgram(*a.model, tea.WithOutput(a.output))

	go func() {
		if _, err := a.program.Run(); err != nil {
			_, _ = fmt.Fprintf(a.output, "TUI error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the TUI
func (a *EvalTUIAdapter) Stop() {
	if a.program != nil {
		a.program.Quit()
	}
}

// StartEval signals that evaluation has started
func (a *EvalTUIAdapter) StartEval(totalTests int) {
	if a.program != nil {
		a.program.Send(EvalStartMsg{TotalTests: totalTests})
	}
}

// AddResult adds an evaluation result
func (a *EvalTUIAdapter) AddResult(result EvalResult) {
	if a.program != nil {
		a.program.Send(EvalResultMsg{Result: result})
	}
}

// Complete signals that evaluation is complete
func (a *EvalTUIAdapter) Complete(success bool, err error) {
	if a.program != nil {
		a.program.Send(EvalCompleteMsg{Success: success, Error: err})
	}
	// Wait for user to see results
	time.Sleep(time.Second)
}

// EvalTUIRunner wraps an eval operation with TUI
type EvalTUIRunner struct {
	adapter *EvalTUIAdapter
	config  EvalTUIConfig
}

// NewEvalTUIRunner creates a new eval TUI runner
func NewEvalTUIRunner(config EvalTUIConfig, output io.Writer) *EvalTUIRunner {
	return &EvalTUIRunner{
		adapter: NewEvalTUIAdapter(config, output),
		config:  config,
	}
}

// Run executes the evaluation with TUI wrapper
func (r *EvalTUIRunner) Run(evalFn func(*EvalTUIAdapter) error) error {
	if err := r.adapter.Start(); err != nil {
		return fmt.Errorf("failed to start TUI: %w", err)
	}
	defer r.adapter.Stop()

	// Signal start
	r.adapter.StartEval(r.config.TestCases * len(r.config.Providers))

	// Execute the eval function
	err := evalFn(r.adapter)

	// Signal completion
	r.adapter.Complete(err == nil, err)

	return err
}
