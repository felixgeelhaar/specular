package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/felixgeelhaar/specular/internal/auto"
)

// ViewType represents the current view being displayed
type ViewType int

// View type constants
const (
	// ViewMain is the main workflow view
	ViewMain ViewType = iota
	// ViewStepList shows the list of steps
	ViewStepList
	// ViewApproval is the approval prompt view
	ViewApproval
	// ViewHelp is the help screen
	ViewHelp
)

// Model represents the TUI application state
type Model struct {
	// Workflow state
	goal       string
	profile    string
	actionPlan *auto.ActionPlan
	autoOutput *auto.AutoOutput

	// Execution state
	currentStep     int
	totalSteps      int
	completedSteps  int
	failedSteps     int
	currentStepName string
	totalCost       float64
	startTime       time.Time

	// UI state
	currentView   ViewType
	verboseMode   bool
	width         int
	height        int
	ready         bool
	quitting      bool
	awaitingInput bool // True when waiting for user input (e.g., approval)

	// Approval state
	approvalPending bool
	approvalPlan    string // Plan details for approval
	approvalChoice  bool   // True = approve, False = reject

	// Error state
	lastError string

	// Styles
	styles Styles
}

// Styles contains lipgloss styles for the TUI
type Styles struct {
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Status      lipgloss.Style
	Error       lipgloss.Style
	Success     lipgloss.Style
	Warning     lipgloss.Style
	Muted       lipgloss.Style
	Border      lipgloss.Style
	Highlighted lipgloss.Style
	Help        lipgloss.Style
	Key         lipgloss.Style
	KeyDesc     lipgloss.Style
}

// NewModel creates a new TUI model
func NewModel(goal, profile string) Model {
	return Model{
		goal:        goal,
		profile:     profile,
		currentView: ViewMain,
		startTime:   time.Now(),
		styles:      DefaultStyles(),
	}
}

// DefaultStyles returns the default lipgloss styles
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")). // Purple
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")). // Gray
			MarginBottom(1),
		Status: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")), // Cyan
		Error: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")), // Red
		Success: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46")), // Green
		Warning: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")), // Yellow
		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")), // Gray
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")). // Purple
			Padding(1, 2),
		Highlighted: lipgloss.NewStyle().
			Background(lipgloss.Color("63")).  // Purple
			Foreground(lipgloss.Color("230")). // Light yellow
			Bold(true).
			Padding(0, 1),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")). // Gray
			MarginTop(1),
		Key: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")), // Purple
		KeyDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")), // Gray
	}
}

// Init initializes the TUI model (required by Bubble Tea)
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state (required by Bubble Tea)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case StepStartMsg:
		m.currentStep = msg.StepIndex
		m.currentStepName = msg.StepName
		return m, nil

	case StepCompleteMsg:
		m.completedSteps++
		m.totalCost = msg.TotalCost
		return m, nil

	case StepFailMsg:
		m.failedSteps++
		m.lastError = msg.Error
		return m, nil

	case ApprovalRequestMsg:
		m.approvalPending = true
		m.approvalPlan = msg.PlanSummary
		m.currentView = ViewApproval
		m.awaitingInput = true
		return m, nil

	case WorkflowCompleteMsg:
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// View renders the TUI (required by Bubble Tea)
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.quitting {
		return m.renderComplete()
	}

	switch m.currentView {
	case ViewMain:
		return m.renderMain()
	case ViewStepList:
		return m.renderStepList()
	case ViewApproval:
		return m.renderApproval()
	case ViewHelp:
		return m.renderHelp()
	default:
		return "Unknown view"
	}
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Ctrl+C always quits
	if msg.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	// Handle approval input
	if m.awaitingInput && m.currentView == ViewApproval {
		switch msg.String() {
		case "y", "enter":
			m.approvalChoice = true
			m.awaitingInput = false
			m.approvalPending = false
			m.currentView = ViewMain
			return m, func() tea.Msg {
				return ApprovalResponseMsg{Approved: true}
			}
		case "n", "esc":
			m.approvalChoice = false
			m.awaitingInput = false
			m.approvalPending = false
			m.currentView = ViewMain
			return m, func() tea.Msg {
				return ApprovalResponseMsg{Approved: false}
			}
		}
		return m, nil
	}

	// Normal key handling
	switch msg.String() {
	case "q":
		if !m.awaitingInput {
			m.quitting = true
			return m, tea.Quit
		}

	case "?":
		if m.currentView == ViewHelp {
			m.currentView = ViewMain
		} else {
			m.currentView = ViewHelp
		}

	case "v":
		m.verboseMode = !m.verboseMode

	case "s":
		if m.currentView == ViewStepList {
			m.currentView = ViewMain
		} else {
			m.currentView = ViewStepList
		}

	case "esc":
		if !m.awaitingInput {
			m.currentView = ViewMain
		}
	}

	return m, nil
}

// SetActionPlan sets the action plan for the model
func (m *Model) SetActionPlan(plan *auto.ActionPlan) {
	m.actionPlan = plan
	if plan != nil {
		m.totalSteps = len(plan.Steps)
	}
}

// SetAutoOutput sets the auto output for the model
func (m *Model) SetAutoOutput(output *auto.AutoOutput) {
	m.autoOutput = output
}

// Custom messages for workflow events

// StepStartMsg indicates a step has started
type StepStartMsg struct {
	StepIndex int
	StepName  string
}

// StepCompleteMsg indicates a step has completed
type StepCompleteMsg struct {
	StepIndex int
	StepName  string
	TotalCost float64
}

// StepFailMsg indicates a step has failed
type StepFailMsg struct {
	StepIndex int
	StepName  string
	Error     string
}

// ApprovalRequestMsg requests user approval
type ApprovalRequestMsg struct {
	PlanSummary string
}

// ApprovalResponseMsg contains the user's approval decision
type ApprovalResponseMsg struct {
	Approved bool
}

// WorkflowCompleteMsg indicates the workflow has finished
type WorkflowCompleteMsg struct {
	Success   bool
	TotalCost float64
	Duration  time.Duration
}

// Helper functions

func (m Model) elapsed() time.Duration {
	return time.Since(m.startTime)
}

func (m Model) progressPercentage() float64 {
	if m.totalSteps == 0 {
		return 0
	}
	return float64(m.completedSteps) / float64(m.totalSteps) * 100
}

func (m Model) statusIcon() string {
	if m.lastError != "" {
		return "✗"
	}
	if m.completedSteps == m.totalSteps && m.totalSteps > 0 {
		return "✓"
	}
	return "⟳"
}

func (m Model) statusColor() lipgloss.TerminalColor {
	if m.lastError != "" {
		return lipgloss.Color("196") // Red
	}
	if m.completedSteps == m.totalSteps && m.totalSteps > 0 {
		return lipgloss.Color("46") // Green
	}
	return lipgloss.Color("86") // Cyan
}
