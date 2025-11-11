package auto

import (
	"fmt"
	"time"
)

// ActionPlan represents a structured workflow plan with typed steps.
// This is a higher-level abstraction than the detailed task plan.
type ActionPlan struct {
	// Schema version for forward compatibility
	Schema string `json:"schema" yaml:"schema"`

	// Goal is the natural language goal provided by the user
	Goal string `json:"goal" yaml:"goal"`

	// Steps are the workflow steps to execute
	Steps []ActionStep `json:"steps" yaml:"steps"`

	// Metadata contains plan generation metadata
	Metadata PlanMetadata `json:"metadata" yaml:"metadata"`
}

// ActionStep represents a single step in the workflow.
type ActionStep struct {
	// ID is the unique step identifier
	ID string `json:"id" yaml:"id"`

	// Type is the step type (spec:update, spec:lock, plan:gen, build:run)
	Type StepType `json:"type" yaml:"type"`

	// Description provides human-readable step information
	Description string `json:"description" yaml:"description"`

	// RequiresApproval indicates if this step needs user approval
	RequiresApproval bool `json:"requiresApproval" yaml:"requiresApproval"`

	// Reason explains why this step is needed (optional)
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`

	// Signals provide routing hints for agent selection
	Signals map[string]string `json:"signals,omitempty" yaml:"signals,omitempty"`

	// Dependencies lists step IDs that must complete before this step
	Dependencies []string `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`

	// Status tracks execution status (pending, in_progress, completed, failed)
	Status StepStatus `json:"status,omitempty" yaml:"status,omitempty"`

	// StartedAt is the step start timestamp
	StartedAt *time.Time `json:"startedAt,omitempty" yaml:"startedAt,omitempty"`

	// CompletedAt is the step completion timestamp
	CompletedAt *time.Time `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`

	// Error contains error message if step failed
	Error string `json:"error,omitempty" yaml:"error,omitempty"`
}

// StepType defines the type of workflow step.
type StepType string

const (
	// StepTypeSpecUpdate updates the product specification
	StepTypeSpecUpdate StepType = "spec:update"

	// StepTypeSpecLock locks the specification with cryptographic hash
	StepTypeSpecLock StepType = "spec:lock"

	// StepTypePlanGen generates the execution plan
	StepTypePlanGen StepType = "plan:gen"

	// StepTypeBuildRun executes build and implementation tasks
	StepTypeBuildRun StepType = "build:run"
)

// StepStatus tracks the execution status of a step.
type StepStatus string

const (
	// StepStatusPending indicates the step hasn't started
	StepStatusPending StepStatus = "pending"

	// StepStatusInProgress indicates the step is currently executing
	StepStatusInProgress StepStatus = "in_progress"

	// StepStatusCompleted indicates the step completed successfully
	StepStatusCompleted StepStatus = "completed"

	// StepStatusFailed indicates the step failed
	StepStatusFailed StepStatus = "failed"

	// StepStatusSkipped indicates the step was skipped
	StepStatusSkipped StepStatus = "skipped"
)

// PlanMetadata contains metadata about plan generation.
type PlanMetadata struct {
	// CreatedAt is the plan creation timestamp
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`

	// Version is the plan format version
	Version string `json:"version" yaml:"version"`

	// Profile is the profile used for plan generation
	Profile string `json:"profile,omitempty" yaml:"profile,omitempty"`

	// EstimatedDuration is the estimated total execution time
	EstimatedDuration time.Duration `json:"estimatedDuration,omitempty" yaml:"estimatedDuration,omitempty"`

	// EstimatedCost is the estimated total cost in USD
	EstimatedCost float64 `json:"estimatedCost,omitempty" yaml:"estimatedCost,omitempty"`
}

// NewActionPlan creates a new action plan with the given goal.
func NewActionPlan(goal string, profile string) *ActionPlan {
	return &ActionPlan{
		Schema: "specular.auto.plan/v1",
		Goal:   goal,
		Steps:  []ActionStep{},
		Metadata: PlanMetadata{
			CreatedAt: time.Now(),
			Version:   "1.0.0",
			Profile:   profile,
		},
	}
}

// AddStep adds a new step to the action plan.
func (p *ActionPlan) AddStep(step ActionStep) {
	// Auto-generate ID if not provided
	if step.ID == "" {
		step.ID = fmt.Sprintf("step-%d", len(p.Steps)+1)
	}

	// Initialize status if not set
	if step.Status == "" {
		step.Status = StepStatusPending
	}

	p.Steps = append(p.Steps, step)
}

// GetStep returns the step with the given ID.
func (p *ActionPlan) GetStep(id string) (*ActionStep, error) {
	for i := range p.Steps {
		if p.Steps[i].ID == id {
			return &p.Steps[i], nil
		}
	}
	return nil, fmt.Errorf("step %q not found", id)
}

// UpdateStepStatus updates the status of a step.
func (p *ActionPlan) UpdateStepStatus(id string, status StepStatus) error {
	step, err := p.GetStep(id)
	if err != nil {
		return err
	}

	step.Status = status

	// Update timestamps based on status
	now := time.Now()
	switch status {
	case StepStatusInProgress:
		if step.StartedAt == nil {
			step.StartedAt = &now
		}
	case StepStatusCompleted, StepStatusFailed, StepStatusSkipped:
		if step.CompletedAt == nil {
			step.CompletedAt = &now
		}
	}

	return nil
}

// GetPendingSteps returns all steps with pending status.
func (p *ActionPlan) GetPendingSteps() []ActionStep {
	var pending []ActionStep
	for _, step := range p.Steps {
		if step.Status == StepStatusPending {
			pending = append(pending, step)
		}
	}
	return pending
}

// GetCompletedSteps returns all steps with completed status.
func (p *ActionPlan) GetCompletedSteps() []ActionStep {
	var completed []ActionStep
	for _, step := range p.Steps {
		if step.Status == StepStatusCompleted {
			completed = append(completed, step)
		}
	}
	return completed
}

// GetFailedSteps returns all steps with failed status.
func (p *ActionPlan) GetFailedSteps() []ActionStep {
	var failed []ActionStep
	for _, step := range p.Steps {
		if step.Status == StepStatusFailed {
			failed = append(failed, step)
		}
	}
	return failed
}

// IsComplete returns true if all steps are completed or skipped.
func (p *ActionPlan) IsComplete() bool {
	for _, step := range p.Steps {
		if step.Status != StepStatusCompleted && step.Status != StepStatusSkipped {
			return false
		}
	}
	return true
}

// HasFailedSteps returns true if any step has failed.
func (p *ActionPlan) HasFailedSteps() bool {
	for _, step := range p.Steps {
		if step.Status == StepStatusFailed {
			return true
		}
	}
	return false
}

// GetNextStep returns the next step that can be executed based on dependencies.
func (p *ActionPlan) GetNextStep() (*ActionStep, error) {
	// Build map of completed steps
	completedMap := make(map[string]bool)
	for _, step := range p.Steps {
		if step.Status == StepStatusCompleted || step.Status == StepStatusSkipped {
			completedMap[step.ID] = true
		}
	}

	// Find first pending step with all dependencies completed
	for i := range p.Steps {
		step := &p.Steps[i]

		// Skip if not pending
		if step.Status != StepStatusPending {
			continue
		}

		// Check if all dependencies are completed
		allDepsMet := true
		for _, depID := range step.Dependencies {
			if !completedMap[depID] {
				allDepsMet = false
				break
			}
		}

		if allDepsMet {
			return step, nil
		}
	}

	return nil, fmt.Errorf("no executable steps found")
}

// Validate validates the action plan structure.
func (p *ActionPlan) Validate() error {
	if p.Schema == "" {
		return fmt.Errorf("schema is required")
	}
	if p.Goal == "" {
		return fmt.Errorf("goal is required")
	}
	if len(p.Steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}

	// Validate step types
	validTypes := map[StepType]bool{
		StepTypeSpecUpdate: true,
		StepTypeSpecLock:   true,
		StepTypePlanGen:    true,
		StepTypeBuildRun:   true,
	}

	// Validate each step
	stepIDs := make(map[string]bool)
	for i, step := range p.Steps {
		if step.ID == "" {
			return fmt.Errorf("step %d: ID is required", i)
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("step %d: duplicate ID %q", i, step.ID)
		}
		stepIDs[step.ID] = true

		if !validTypes[step.Type] {
			return fmt.Errorf("step %d (%s): invalid type %q", i, step.ID, step.Type)
		}
		if step.Description == "" {
			return fmt.Errorf("step %d (%s): description is required", i, step.ID)
		}

		// Validate dependencies refer to existing steps
		for _, depID := range step.Dependencies {
			if !stepIDs[depID] {
				return fmt.Errorf("step %d (%s): dependency %q not found", i, step.ID, depID)
			}
		}
	}

	// Check for circular dependencies
	if err := p.checkCircularDependencies(); err != nil {
		return fmt.Errorf("circular dependency detected: %w", err)
	}

	return nil
}

// checkCircularDependencies checks for circular dependencies in the plan.
func (p *ActionPlan) checkCircularDependencies() error {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, step := range p.Steps {
		graph[step.ID] = step.Dependencies
	}

	// DFS to detect cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(stepID string) bool {
		visited[stepID] = true
		recStack[stepID] = true

		for _, depID := range graph[stepID] {
			if !visited[depID] {
				if hasCycle(depID) {
					return true
				}
			} else if recStack[depID] {
				return true
			}
		}

		recStack[stepID] = false
		return false
	}

	for stepID := range graph {
		if !visited[stepID] {
			if hasCycle(stepID) {
				return fmt.Errorf("circular dependency involving step %q", stepID)
			}
		}
	}

	return nil
}

// CreateDefaultActionPlan creates the default action plan for autonomous mode.
func CreateDefaultActionPlan(goal string, profile string) *ActionPlan {
	plan := NewActionPlan(goal, profile)

	// Step 1: Update specification
	plan.AddStep(ActionStep{
		ID:               "step-1",
		Type:             StepTypeSpecUpdate,
		Description:      "Generate product specification from goal",
		RequiresApproval: false,
		Signals: map[string]string{
			"model_hint": "long-context",
			"skill":      "spec-generation",
		},
	})

	// Step 2: Lock specification
	plan.AddStep(ActionStep{
		ID:               "step-2",
		Type:             StepTypeSpecLock,
		Description:      "Lock specification with cryptographic hash",
		RequiresApproval: true,
		Dependencies:     []string{"step-1"},
		Signals: map[string]string{
			"critical": "true",
		},
	})

	// Step 3: Generate plan
	plan.AddStep(ActionStep{
		ID:               "step-3",
		Type:             StepTypePlanGen,
		Description:      "Generate execution plan from specification",
		RequiresApproval: false,
		Dependencies:     []string{"step-2"},
		Signals: map[string]string{
			"model_hint": "agentic",
		},
	})

	// Step 4: Execute build
	plan.AddStep(ActionStep{
		ID:               "step-4",
		Type:             StepTypeBuildRun,
		Description:      "Execute implementation tasks",
		RequiresApproval: true,
		Dependencies:     []string{"step-3"},
		Signals: map[string]string{
			"model_hint": "codegen",
			"critical":   "true",
		},
	})

	return plan
}
