package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// PanelType represents the active panel in the dashboard
type PanelType int

const (
	// PanelProgress is the main progress panel
	PanelProgress PanelType = iota
	// PanelBudget shows budget details
	PanelBudget
	// PanelLogs shows log entries
	PanelLogs
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelDebug LogLevel = "DEBUG"
)

// LogEntry represents a single log entry in the dashboard
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	StepID    string
}

// DashboardModel extends Model with dashboard-specific state
type DashboardModel struct {
	Model // Embed the base model

	// Budget tracking
	budgetLimit     float64
	budgetSpent     float64
	budgetRemaining float64
	budgetWarning   string

	// API metrics
	apiCallCount int
	apiCallRate  float64 // calls per minute
	lastAPICall  time.Time

	// Log entries (circular buffer)
	logEntries    []LogEntry
	maxLogEntries int
	logScrollPos  int

	// Layout state
	activePanel   PanelType
	showBudgetBar bool

	// Dashboard view mode
	dashboardMode bool
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(goal, profile string) DashboardModel {
	return DashboardModel{
		Model:         NewModel(goal, profile),
		maxLogEntries: 100,
		logEntries:    make([]LogEntry, 0, 100),
		activePanel:   PanelProgress,
		showBudgetBar: true,
		dashboardMode: true,
		budgetLimit:   5.0, // Default $5 limit
	}
}

// SetBudgetLimit sets the budget limit for tracking
func (m *DashboardModel) SetBudgetLimit(limit float64) {
	m.budgetLimit = limit
	m.budgetRemaining = limit - m.budgetSpent
}

// UpdateBudget updates the budget tracking
func (m *DashboardModel) UpdateBudget(spent float64, warning string) {
	m.budgetSpent = spent
	m.budgetRemaining = m.budgetLimit - spent
	m.budgetWarning = warning
}

// AddLogEntry adds a new log entry to the dashboard
func (m *DashboardModel) AddLogEntry(level LogLevel, message, stepID string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		StepID:    stepID,
	}

	// Circular buffer behavior
	if len(m.logEntries) >= m.maxLogEntries {
		m.logEntries = append(m.logEntries[1:], entry)
	} else {
		m.logEntries = append(m.logEntries, entry)
	}

	// Auto-scroll to bottom
	if m.logScrollPos > 0 {
		m.logScrollPos = 0
	}
}

// IncrementAPICall records an API call
func (m *DashboardModel) IncrementAPICall() {
	m.apiCallCount++
	m.lastAPICall = time.Now()
	// Calculate rate based on elapsed time
	elapsed := time.Since(m.startTime).Minutes()
	if elapsed > 0 {
		m.apiCallRate = float64(m.apiCallCount) / elapsed
	}
}

// Custom messages for dashboard updates

// BudgetUpdateMsg updates budget display
type BudgetUpdateMsg struct {
	Spent     float64
	Remaining float64
	Limit     float64
	Warning   string
}

// APIMetricsMsg updates API metrics
type APIMetricsMsg struct {
	CallCount int
	LatencyMs int64
}

// LogEntryMsg adds a log entry
type LogEntryMsg struct {
	Level   LogLevel
	Message string
	StepID  string
}

// RefreshTickMsg triggers periodic refresh
type RefreshTickMsg struct{}

// Init initializes the dashboard model
func (m DashboardModel) Init() tea.Cmd {
	// Start a ticker for periodic refresh
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return RefreshTickMsg{}
	})
}

// Update handles messages for the dashboard
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleDashboardKeyPress(msg)

	case BudgetUpdateMsg:
		m.budgetSpent = msg.Spent
		m.budgetRemaining = msg.Remaining
		m.budgetLimit = msg.Limit
		m.budgetWarning = msg.Warning
		return m, nil

	case APIMetricsMsg:
		m.apiCallCount = msg.CallCount
		return m, nil

	case LogEntryMsg:
		m.AddLogEntry(msg.Level, msg.Message, msg.StepID)
		return m, nil

	case RefreshTickMsg:
		// Update API rate
		elapsed := time.Since(m.startTime).Minutes()
		if elapsed > 0 {
			m.apiCallRate = float64(m.apiCallCount) / elapsed
		}
		// Continue ticking
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return RefreshTickMsg{}
		})

	default:
		// Delegate to base model for other messages
		baseModel, cmd := m.Model.Update(msg)
		m.Model = baseModel.(Model)
		return m, cmd
	}
}

// handleDashboardKeyPress handles dashboard-specific keyboard input
func (m DashboardModel) handleDashboardKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		// Cycle through panels
		m.activePanel = (m.activePanel + 1) % 3
		return m, nil

	case "b":
		// Toggle budget bar
		m.showBudgetBar = !m.showBudgetBar
		return m, nil

	case "l":
		// Toggle to logs panel
		if m.activePanel == PanelLogs {
			m.activePanel = PanelProgress
		} else {
			m.activePanel = PanelLogs
		}
		return m, nil

	case "j", "down":
		// Scroll logs down
		if m.activePanel == PanelLogs && len(m.logEntries) > 0 {
			maxScroll := len(m.logEntries) - 10 // Show 10 lines
			if maxScroll > 0 && m.logScrollPos < maxScroll {
				m.logScrollPos++
			}
		}
		return m, nil

	case "k", "up":
		// Scroll logs up
		if m.activePanel == PanelLogs && m.logScrollPos > 0 {
			m.logScrollPos--
		}
		return m, nil

	default:
		// Delegate to base model
		baseModel, cmd := m.Model.handleKeyPress(msg)
		m.Model = baseModel.(Model)
		return m, cmd
	}
}

// View renders the dashboard
func (m DashboardModel) View() string {
	if !m.ready {
		return "Initializing dashboard..."
	}

	if m.quitting {
		return m.renderComplete()
	}

	// Dashboard mode always uses dashboard view
	if m.dashboardMode {
		return m.renderDashboard()
	}

	// Fall back to base model views
	return m.Model.View()
}

// GetLogEntries returns visible log entries for rendering
func (m *DashboardModel) GetLogEntries(maxLines int) []LogEntry {
	if len(m.logEntries) == 0 {
		return nil
	}

	start := m.logScrollPos
	end := start + maxLines
	if end > len(m.logEntries) {
		end = len(m.logEntries)
	}

	// Return entries in reverse order (newest first)
	result := make([]LogEntry, 0, end-start)
	for i := end - 1; i >= start; i-- {
		result = append(result, m.logEntries[i])
	}
	return result
}

// GetBudgetPercentage returns budget usage as a percentage
func (m *DashboardModel) GetBudgetPercentage() float64 {
	if m.budgetLimit <= 0 {
		return 0
	}
	return (m.budgetSpent / m.budgetLimit) * 100
}

// GetETA estimates remaining time based on current progress
func (m *DashboardModel) GetETA() time.Duration {
	if m.completedSteps == 0 || m.totalSteps == 0 {
		return 0
	}

	elapsed := time.Since(m.startTime)
	avgPerStep := elapsed / time.Duration(m.completedSteps)
	remaining := m.totalSteps - m.completedSteps
	return avgPerStep * time.Duration(remaining)
}
