package auto

import (
	"time"

	"github.com/felixgeelhaar/specular/internal/drift"
	"github.com/felixgeelhaar/specular/internal/eval"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// Config defines auto mode settings
type Config struct {
	// User's goal in natural language
	Goal string `yaml:"goal"`

	// Approval settings
	RequireApproval bool `yaml:"require_approval"`

	// Budget constraints
	MaxCostUSD     float64 `yaml:"max_cost_usd"`
	MaxCostPerTask float64 `yaml:"max_cost_per_task"`

	// Retry settings
	MaxRetries int           `yaml:"max_retries"`
	RetryDelay time.Duration `yaml:"retry_delay"`

	// Timeout settings
	TimeoutMinutes int           `yaml:"timeout_minutes"`
	TaskTimeout    time.Duration `yaml:"task_timeout"`

	// Policy enforcement
	PolicyPath string `yaml:"policy_path"`

	// Behavior flags
	FallbackToManual bool `yaml:"fallback_to_manual"`
	Verbose          bool `yaml:"verbose"`
	DryRun           bool `yaml:"dry_run"`

	// Resume settings
	ResumeFrom string `yaml:"resume_from"` // Checkpoint operation ID to resume from
}

// Result contains the outcome of auto mode execution
type Result struct {
	Success       bool
	Spec          *spec.ProductSpec
	SpecLock      *spec.SpecLock
	Plan          *plan.Plan
	EvalResult    *eval.GateReport
	DriftFindings []drift.Finding
	TotalCost     float64
	Duration      time.Duration
	TasksExecuted int
	TasksFailed   int
	Errors        []error
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		RequireApproval:  true,
		MaxCostUSD:       5.0,
		MaxCostPerTask:   1.0,
		MaxRetries:       3,
		RetryDelay:       time.Second * 2,
		TimeoutMinutes:   30,
		TaskTimeout:      time.Minute * 5,
		PolicyPath:       ".specular/policy.yaml",
		FallbackToManual: true,
		Verbose:          false,
		DryRun:           false,
	}
}
