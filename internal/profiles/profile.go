package profiles

import (
	"fmt"
	"time"
)

// Profile represents an autonomous mode profile with environment-specific configurations.
// Profiles define approval rules, safety limits, routing preferences, and execution policies.
type Profile struct {
	// Name is the profile identifier (e.g., "default", "ci", "strict")
	Name string `yaml:"name" json:"name"`

	// Description provides human-readable profile information
	Description string `yaml:"description" json:"description"`

	// Approvals configures approval gates and interactive behavior
	Approvals ApprovalConfig `yaml:"approvals" json:"approvals"`

	// Safety defines execution limits and constraints
	Safety SafetyConfig `yaml:"safety" json:"safety"`

	// Routing configures agent selection and LLM preferences
	Routing RoutingConfig `yaml:"routing" json:"routing"`

	// Policies configures policy checks and enforcement
	Policies PolicyConfig `yaml:"policies" json:"policies"`

	// Execution configures execution behavior (logging, checkpoints, etc.)
	Execution ExecutionConfig `yaml:"execution" json:"execution"`

	// Hooks defines lifecycle hooks for notifications and integrations
	Hooks HooksConfig `yaml:"hooks,omitempty" json:"hooks,omitempty"`
}

// ApprovalConfig defines approval gate behavior.
type ApprovalConfig struct {
	// Mode determines approval strategy: "all", "critical_only", "none"
	Mode ApprovalMode `yaml:"mode" json:"mode"`

	// Interactive enables interactive approval prompts
	Interactive bool `yaml:"interactive" json:"interactive"`

	// AutoApprove lists step types that don't require approval
	AutoApprove []string `yaml:"auto_approve,omitempty" json:"auto_approve,omitempty"`

	// RequireApproval lists step types that always require approval
	RequireApproval []string `yaml:"require_approval,omitempty" json:"require_approval,omitempty"`
}

// ApprovalMode defines approval strategies.
type ApprovalMode string

const (
	// ApprovalModeAll requires approval for all steps
	ApprovalModeAll ApprovalMode = "all"

	// ApprovalModeCriticalOnly requires approval only for critical steps
	ApprovalModeCriticalOnly ApprovalMode = "critical_only"

	// ApprovalModeNone requires no approvals (auto-approve all)
	ApprovalModeNone ApprovalMode = "none"
)

// SafetyConfig defines safety limits and constraints.
type SafetyConfig struct {
	// MaxSteps limits the maximum number of workflow steps
	MaxSteps int `yaml:"max_steps" json:"max_steps"`

	// Timeout limits total workflow execution time
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// MaxCostUSD limits total workflow cost in USD
	MaxCostUSD float64 `yaml:"max_cost_usd" json:"max_cost_usd"`

	// MaxCostPerTask limits cost per individual task in USD
	MaxCostPerTask float64 `yaml:"max_cost_per_task" json:"max_cost_per_task"`

	// MaxRetries limits retry attempts per task
	MaxRetries int `yaml:"max_retries" json:"max_retries"`

	// RequirePolicy enforces policy checks before execution
	RequirePolicy bool `yaml:"require_policy" json:"require_policy"`

	// AllowedStepTypes whitelists step types (empty = allow all)
	AllowedStepTypes []string `yaml:"allowed_step_types,omitempty" json:"allowed_step_types,omitempty"`

	// BlockedStepTypes blacklists step types
	BlockedStepTypes []string `yaml:"blocked_step_types,omitempty" json:"blocked_step_types,omitempty"`
}

// RoutingConfig defines agent selection and model preferences.
type RoutingConfig struct {
	// PreferredAgent is the primary agent for execution
	PreferredAgent string `yaml:"preferred_agent" json:"preferred_agent"`

	// FallbackAgent is used if preferred agent is unavailable
	FallbackAgent string `yaml:"fallback_agent" json:"fallback_agent"`

	// Temperature controls LLM randomness (0.0-1.0)
	Temperature float64 `yaml:"temperature" json:"temperature"`

	// ModelPreferences maps step types to model names
	ModelPreferences map[string]string `yaml:"model_preferences,omitempty" json:"model_preferences,omitempty"`
}

// PolicyConfig defines policy check behavior.
type PolicyConfig struct {
	// Enabled enables policy checks
	Enabled bool `yaml:"enabled" json:"enabled"`

	// PolicyFiles lists policy files to load (Rego, etc.)
	PolicyFiles []string `yaml:"policy_files,omitempty" json:"policy_files,omitempty"`

	// Enforcement determines policy enforcement level: "strict", "warn", "none"
	Enforcement PolicyEnforcement `yaml:"enforcement" json:"enforcement"`
}

// PolicyEnforcement defines policy enforcement levels.
type PolicyEnforcement string

const (
	// PolicyEnforcementStrict aborts on policy violations
	PolicyEnforcementStrict PolicyEnforcement = "strict"

	// PolicyEnforcementWarn warns on policy violations but continues
	PolicyEnforcementWarn PolicyEnforcement = "warn"

	// PolicyEnforcementNone disables policy enforcement
	PolicyEnforcementNone PolicyEnforcement = "none"
)

// ExecutionConfig defines execution behavior.
type ExecutionConfig struct {
	// TraceLogging enables comprehensive trace logging
	TraceLogging bool `yaml:"trace_logging" json:"trace_logging"`

	// SavePatches generates patch files for rollback
	SavePatches bool `yaml:"save_patches" json:"save_patches"`

	// CheckpointFrequency determines checkpoint creation frequency (steps)
	CheckpointFrequency int `yaml:"checkpoint_frequency" json:"checkpoint_frequency"`

	// JSONOutput enables JSON output format
	JSONOutput bool `yaml:"json_output" json:"json_output"`

	// EnableTUI enables terminal UI (if available)
	EnableTUI bool `yaml:"enable_tui" json:"enable_tui"`
}

// HooksConfig defines lifecycle hooks.
type HooksConfig struct {
	// OnPlanCreated hooks execute after plan generation
	OnPlanCreated []Hook `yaml:"on_plan_created,omitempty" json:"on_plan_created,omitempty"`

	// OnStepBefore hooks execute before each step
	OnStepBefore []Hook `yaml:"on_step_before,omitempty" json:"on_step_before,omitempty"`

	// OnStepAfter hooks execute after each step
	OnStepAfter []Hook `yaml:"on_step_after,omitempty" json:"on_step_after,omitempty"`

	// OnApprovalRequested hooks execute when approval needed
	OnApprovalRequested []Hook `yaml:"on_approval_requested,omitempty" json:"on_approval_requested,omitempty"`

	// OnComplete hooks execute on successful completion
	OnComplete []Hook `yaml:"on_complete,omitempty" json:"on_complete,omitempty"`

	// OnError hooks execute on errors
	OnError []Hook `yaml:"on_error,omitempty" json:"on_error,omitempty"`
}

// Hook represents a single lifecycle hook.
type Hook struct {
	// Type identifies the hook type (webhook, slack, email, etc.)
	Type string `yaml:"type" json:"type"`

	// Config contains hook-specific configuration
	Config map[string]interface{} `yaml:",inline" json:"config"`
}

// ProfileCollection represents a collection of profiles from a YAML file.
type ProfileCollection struct {
	// Schema version for forward compatibility
	Schema string `yaml:"schema" json:"schema"`

	// Profiles maps profile names to Profile objects
	Profiles map[string]Profile `yaml:"profiles" json:"profiles"`
}

// Validate validates the profile configuration.
func (p *Profile) Validate() error {
	if err := p.Approvals.Validate(); err != nil {
		return fmt.Errorf("approvals: %w", err)
	}

	if err := p.Safety.Validate(); err != nil {
		return fmt.Errorf("safety: %w", err)
	}

	if err := p.Routing.Validate(); err != nil {
		return fmt.Errorf("routing: %w", err)
	}

	if err := p.Policies.Validate(); err != nil {
		return fmt.Errorf("policies: %w", err)
	}

	if err := p.Execution.Validate(); err != nil {
		return fmt.Errorf("execution: %w", err)
	}

	return nil
}

// Validate validates approval configuration.
func (a *ApprovalConfig) Validate() error {
	switch a.Mode {
	case ApprovalModeAll, ApprovalModeCriticalOnly, ApprovalModeNone:
		// Valid modes
	default:
		return fmt.Errorf("invalid approval mode: %q (must be all, critical_only, or none)", a.Mode)
	}

	// Validate step types
	validStepTypes := map[string]bool{
		"spec:update": true,
		"spec:lock":   true,
		"plan:gen":    true,
		"build:run":   true,
	}

	for _, stepType := range a.AutoApprove {
		if !validStepTypes[stepType] {
			return fmt.Errorf("invalid step type in auto_approve: %q", stepType)
		}
	}

	for _, stepType := range a.RequireApproval {
		if !validStepTypes[stepType] {
			return fmt.Errorf("invalid step type in require_approval: %q", stepType)
		}
	}

	return nil
}

// Validate validates safety configuration.
func (s *SafetyConfig) Validate() error {
	if s.MaxSteps <= 0 || s.MaxSteps > 100 {
		return fmt.Errorf("max_steps must be between 1 and 100, got %d", s.MaxSteps)
	}

	if s.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %s", s.Timeout)
	}

	if s.MaxCostUSD <= 0 {
		return fmt.Errorf("max_cost_usd must be positive, got %.2f", s.MaxCostUSD)
	}

	if s.MaxCostPerTask <= 0 {
		return fmt.Errorf("max_cost_per_task must be positive, got %.2f", s.MaxCostPerTask)
	}

	if s.MaxCostPerTask > s.MaxCostUSD {
		return fmt.Errorf("max_cost_per_task (%.2f) cannot exceed max_cost_usd (%.2f)", s.MaxCostPerTask, s.MaxCostUSD)
	}

	if s.MaxRetries < 0 || s.MaxRetries > 10 {
		return fmt.Errorf("max_retries must be between 0 and 10, got %d", s.MaxRetries)
	}

	// Validate step types
	validStepTypes := map[string]bool{
		"spec:update": true,
		"spec:lock":   true,
		"plan:gen":    true,
		"build:run":   true,
	}

	for _, stepType := range s.AllowedStepTypes {
		if !validStepTypes[stepType] {
			return fmt.Errorf("invalid step type in allowed_step_types: %q", stepType)
		}
	}

	for _, stepType := range s.BlockedStepTypes {
		if !validStepTypes[stepType] {
			return fmt.Errorf("invalid step type in blocked_step_types: %q", stepType)
		}
	}

	return nil
}

// Validate validates routing configuration.
func (r *RoutingConfig) Validate() error {
	if r.PreferredAgent == "" {
		return fmt.Errorf("preferred_agent is required")
	}

	if r.FallbackAgent == "" {
		return fmt.Errorf("fallback_agent is required")
	}

	if r.Temperature < 0.0 || r.Temperature > 1.0 {
		return fmt.Errorf("temperature must be between 0.0 and 1.0, got %.2f", r.Temperature)
	}

	return nil
}

// Validate validates policy configuration.
func (p *PolicyConfig) Validate() error {
	switch p.Enforcement {
	case PolicyEnforcementStrict, PolicyEnforcementWarn, PolicyEnforcementNone:
		// Valid enforcement levels
	default:
		return fmt.Errorf("invalid policy enforcement: %q (must be strict, warn, or none)", p.Enforcement)
	}

	return nil
}

// Validate validates execution configuration.
func (e *ExecutionConfig) Validate() error {
	if e.CheckpointFrequency <= 0 {
		return fmt.Errorf("checkpoint_frequency must be positive, got %d", e.CheckpointFrequency)
	}

	return nil
}

// Merge merges another profile into this one, with the other profile taking precedence.
// Returns a new Profile with merged values.
func (p *Profile) Merge(other *Profile) *Profile {
	merged := &Profile{
		Name:        other.Name,
		Description: other.Description,
	}

	// Merge Approvals
	merged.Approvals = p.Approvals
	if other.Approvals.Mode != "" {
		merged.Approvals.Mode = other.Approvals.Mode
	}
	merged.Approvals.Interactive = other.Approvals.Interactive
	if len(other.Approvals.AutoApprove) > 0 {
		merged.Approvals.AutoApprove = other.Approvals.AutoApprove
	}
	if len(other.Approvals.RequireApproval) > 0 {
		merged.Approvals.RequireApproval = other.Approvals.RequireApproval
	}

	// Merge Safety
	merged.Safety = p.Safety
	if other.Safety.MaxSteps > 0 {
		merged.Safety.MaxSteps = other.Safety.MaxSteps
	}
	if other.Safety.Timeout > 0 {
		merged.Safety.Timeout = other.Safety.Timeout
	}
	if other.Safety.MaxCostUSD > 0 {
		merged.Safety.MaxCostUSD = other.Safety.MaxCostUSD
	}
	if other.Safety.MaxCostPerTask > 0 {
		merged.Safety.MaxCostPerTask = other.Safety.MaxCostPerTask
	}
	if other.Safety.MaxRetries >= 0 {
		merged.Safety.MaxRetries = other.Safety.MaxRetries
	}
	merged.Safety.RequirePolicy = other.Safety.RequirePolicy
	if len(other.Safety.AllowedStepTypes) > 0 {
		merged.Safety.AllowedStepTypes = other.Safety.AllowedStepTypes
	}
	if len(other.Safety.BlockedStepTypes) > 0 {
		merged.Safety.BlockedStepTypes = other.Safety.BlockedStepTypes
	}

	// Merge Routing
	merged.Routing = p.Routing
	if other.Routing.PreferredAgent != "" {
		merged.Routing.PreferredAgent = other.Routing.PreferredAgent
	}
	if other.Routing.FallbackAgent != "" {
		merged.Routing.FallbackAgent = other.Routing.FallbackAgent
	}
	if other.Routing.Temperature > 0 {
		merged.Routing.Temperature = other.Routing.Temperature
	}
	if len(other.Routing.ModelPreferences) > 0 {
		merged.Routing.ModelPreferences = make(map[string]string)
		for k, v := range p.Routing.ModelPreferences {
			merged.Routing.ModelPreferences[k] = v
		}
		for k, v := range other.Routing.ModelPreferences {
			merged.Routing.ModelPreferences[k] = v
		}
	}

	// Merge Policies
	merged.Policies = p.Policies
	merged.Policies.Enabled = other.Policies.Enabled
	if len(other.Policies.PolicyFiles) > 0 {
		merged.Policies.PolicyFiles = other.Policies.PolicyFiles
	}
	if other.Policies.Enforcement != "" {
		merged.Policies.Enforcement = other.Policies.Enforcement
	}

	// Merge Execution
	merged.Execution = p.Execution
	merged.Execution.TraceLogging = other.Execution.TraceLogging
	merged.Execution.SavePatches = other.Execution.SavePatches
	if other.Execution.CheckpointFrequency > 0 {
		merged.Execution.CheckpointFrequency = other.Execution.CheckpointFrequency
	}
	merged.Execution.JSONOutput = other.Execution.JSONOutput
	merged.Execution.EnableTUI = other.Execution.EnableTUI

	// Merge Hooks
	if len(other.Hooks.OnPlanCreated) > 0 {
		merged.Hooks.OnPlanCreated = other.Hooks.OnPlanCreated
	} else {
		merged.Hooks.OnPlanCreated = p.Hooks.OnPlanCreated
	}
	if len(other.Hooks.OnStepBefore) > 0 {
		merged.Hooks.OnStepBefore = other.Hooks.OnStepBefore
	} else {
		merged.Hooks.OnStepBefore = p.Hooks.OnStepBefore
	}
	if len(other.Hooks.OnStepAfter) > 0 {
		merged.Hooks.OnStepAfter = other.Hooks.OnStepAfter
	} else {
		merged.Hooks.OnStepAfter = p.Hooks.OnStepAfter
	}
	if len(other.Hooks.OnApprovalRequested) > 0 {
		merged.Hooks.OnApprovalRequested = other.Hooks.OnApprovalRequested
	} else {
		merged.Hooks.OnApprovalRequested = p.Hooks.OnApprovalRequested
	}
	if len(other.Hooks.OnComplete) > 0 {
		merged.Hooks.OnComplete = other.Hooks.OnComplete
	} else {
		merged.Hooks.OnComplete = p.Hooks.OnComplete
	}
	if len(other.Hooks.OnError) > 0 {
		merged.Hooks.OnError = other.Hooks.OnError
	} else {
		merged.Hooks.OnError = p.Hooks.OnError
	}

	return merged
}

// ShouldRequireApproval determines if a step type requires approval based on the profile.
func (p *Profile) ShouldRequireApproval(stepType string) bool {
	// Check approval mode
	switch p.Approvals.Mode {
	case ApprovalModeAll:
		return true
	case ApprovalModeNone:
		return false
	case ApprovalModeCriticalOnly:
		// Check explicit require_approval list
		for _, st := range p.Approvals.RequireApproval {
			if st == stepType {
				return true
			}
		}

		// Check auto_approve list
		for _, st := range p.Approvals.AutoApprove {
			if st == stepType {
				return false
			}
		}

		// Default critical steps
		criticalSteps := map[string]bool{
			"spec:lock": true,
			"build:run": true,
		}
		return criticalSteps[stepType]
	}

	return false
}

// IsStepTypeAllowed checks if a step type is allowed by the profile.
func (p *Profile) IsStepTypeAllowed(stepType string) bool {
	// Check blocked list first
	for _, st := range p.Safety.BlockedStepTypes {
		if st == stepType {
			return false
		}
	}

	// If allowed list is empty, allow all (except blocked)
	if len(p.Safety.AllowedStepTypes) == 0 {
		return true
	}

	// Check allowed list
	for _, st := range p.Safety.AllowedStepTypes {
		if st == stepType {
			return true
		}
	}

	return false
}
