package auto

import (
	"encoding/json"
	"time"
)

// AutoOutput represents the complete JSON output for autonomous mode execution.
// This format is designed for CI/CD integration and machine parsing.
type AutoOutput struct {
	// Schema version for output format compatibility
	Schema string `json:"schema"`

	// Goal describes the user's original objective
	Goal string `json:"goal"`

	// Status indicates the overall execution outcome: completed, failed, partial
	Status string `json:"status"`

	// Steps contains results for each executed step
	Steps []StepResult `json:"steps"`

	// Artifacts lists generated files and outputs
	Artifacts []ArtifactInfo `json:"artifacts"`

	// Metrics contains execution statistics
	Metrics ExecutionMetrics `json:"metrics"`

	// Audit provides provenance and compliance information
	Audit AuditTrail `json:"audit"`
}

// StepResult captures the execution result of a single step.
type StepResult struct {
	// ID uniquely identifies the step
	ID string `json:"id"`

	// Type categorizes the step (spec:update, spec:lock, plan:gen, build:run)
	Type string `json:"type"`

	// Status indicates step outcome: pending, in_progress, completed, failed, skipped
	Status string `json:"status"`

	// StartedAt records when step execution began
	StartedAt time.Time `json:"startedAt"`

	// CompletedAt records when step execution finished
	CompletedAt time.Time `json:"completedAt"`

	// Duration is the time taken to execute the step
	Duration time.Duration `json:"duration"`

	// Error contains error message if step failed
	Error string `json:"error,omitempty"`

	// CostUSD is the estimated cost for this step
	CostUSD float64 `json:"costUSD,omitempty"`

	// Warnings contains non-fatal issues encountered
	Warnings []string `json:"warnings,omitempty"`

	// Metadata contains step-specific additional information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ArtifactInfo describes a generated artifact.
type ArtifactInfo struct {
	// Path is the file path relative to project root
	Path string `json:"path"`

	// Type categorizes the artifact (spec, lock, plan, code, test, etc.)
	Type string `json:"type"`

	// Size in bytes
	Size int64 `json:"size"`

	// Hash provides content verification (SHA256)
	Hash string `json:"hash"`

	// CreatedAt records when artifact was created
	CreatedAt time.Time `json:"createdAt"`
}

// ExecutionMetrics captures statistical information about the execution.
type ExecutionMetrics struct {
	// TotalDuration is the complete workflow execution time
	TotalDuration time.Duration `json:"totalDuration"`

	// TotalCost is the sum of all step costs in USD
	TotalCost float64 `json:"totalCost"`

	// StepsExecuted is the count of steps that ran
	StepsExecuted int `json:"stepsExecuted"`

	// StepsFailed is the count of steps that failed
	StepsFailed int `json:"stepsFailed"`

	// StepsSkipped is the count of steps that were skipped
	StepsSkipped int `json:"stepsSkipped"`

	// PolicyViolations is the count of policy check failures
	PolicyViolations int `json:"policyViolations"`

	// TokensUsed tracks total token consumption
	TokensUsed int `json:"tokensUsed,omitempty"`

	// RetriesPerformed tracks total retry attempts
	RetriesPerformed int `json:"retriesPerformed,omitempty"`
}

// AuditTrail provides provenance and compliance information.
type AuditTrail struct {
	// CheckpointID identifies the execution checkpoint
	CheckpointID string `json:"checkpointId"`

	// Profile indicates which profile was used
	Profile string `json:"profile"`

	// StartedAt records workflow start time
	StartedAt time.Time `json:"startedAt"`

	// CompletedAt records workflow completion time
	CompletedAt time.Time `json:"completedAt"`

	// User identifies who initiated the workflow
	User string `json:"user,omitempty"`

	// Hostname identifies where execution occurred
	Hostname string `json:"hostname,omitempty"`

	// Approvals tracks approval events during execution
	Approvals []ApprovalEvent `json:"approvals"`

	// Policies tracks policy check events
	Policies []PolicyEvent `json:"policies"`

	// Version tracks the Specular version used
	Version string `json:"version,omitempty"`
}

// ApprovalEvent records a user approval interaction.
type ApprovalEvent struct {
	// StepID identifies which step required approval
	StepID string `json:"stepId"`

	// Timestamp records when approval occurred
	Timestamp time.Time `json:"timestamp"`

	// Approved indicates whether the step was approved
	Approved bool `json:"approved"`

	// Reason explains why approval was required
	Reason string `json:"reason,omitempty"`

	// User identifies who approved
	User string `json:"user,omitempty"`
}

// PolicyEvent records a policy check result.
type PolicyEvent struct {
	// StepID identifies which step was checked
	StepID string `json:"stepId"`

	// Timestamp records when check occurred
	Timestamp time.Time `json:"timestamp"`

	// CheckerName identifies which policy checker ran
	CheckerName string `json:"checkerName"`

	// Allowed indicates whether the policy check passed
	Allowed bool `json:"allowed"`

	// Reason explains policy decision
	Reason string `json:"reason,omitempty"`

	// Warnings contains non-blocking policy warnings
	Warnings []string `json:"warnings,omitempty"`

	// Metadata contains policy-specific additional information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewAutoOutput creates an initialized AutoOutput structure.
func NewAutoOutput(goal, profile string) *AutoOutput {
	return &AutoOutput{
		Schema:    "specular.auto.output/v1",
		Goal:      goal,
		Status:    "in_progress",
		Steps:     []StepResult{},
		Artifacts: []ArtifactInfo{},
		Metrics: ExecutionMetrics{
			StepsExecuted:    0,
			StepsFailed:      0,
			StepsSkipped:     0,
			PolicyViolations: 0,
		},
		Audit: AuditTrail{
			Profile:   profile,
			StartedAt: time.Now(),
			Approvals: []ApprovalEvent{},
			Policies:  []PolicyEvent{},
		},
	}
}

// ToJSON serializes AutoOutput to JSON bytes.
func (o *AutoOutput) ToJSON() ([]byte, error) {
	return json.MarshalIndent(o, "", "  ")
}

// FromJSON deserializes AutoOutput from JSON bytes.
func FromJSON(data []byte) (*AutoOutput, error) {
	var output AutoOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

// AddStepResult adds a step result to the output.
func (o *AutoOutput) AddStepResult(step StepResult) {
	o.Steps = append(o.Steps, step)

	// Update metrics
	o.Metrics.StepsExecuted++
	if step.Status == "failed" {
		o.Metrics.StepsFailed++
	}
	if step.Status == "skipped" {
		o.Metrics.StepsSkipped++
	}
	o.Metrics.TotalCost += step.CostUSD
}

// AddArtifact adds an artifact to the output.
func (o *AutoOutput) AddArtifact(artifact ArtifactInfo) {
	o.Artifacts = append(o.Artifacts, artifact)
}

// AddApproval adds an approval event to the audit trail.
func (o *AutoOutput) AddApproval(event ApprovalEvent) {
	o.Audit.Approvals = append(o.Audit.Approvals, event)
}

// AddPolicy adds a policy event to the audit trail.
func (o *AutoOutput) AddPolicy(event PolicyEvent) {
	o.Audit.Policies = append(o.Audit.Policies, event)
	if !event.Allowed {
		o.Metrics.PolicyViolations++
	}
}

// SetCompleted marks the execution as completed successfully.
func (o *AutoOutput) SetCompleted() {
	o.Status = "completed"
	o.Audit.CompletedAt = time.Now()
	o.Metrics.TotalDuration = o.Audit.CompletedAt.Sub(o.Audit.StartedAt)
}

// SetFailed marks the execution as failed.
func (o *AutoOutput) SetFailed() {
	o.Status = "failed"
	o.Audit.CompletedAt = time.Now()
	o.Metrics.TotalDuration = o.Audit.CompletedAt.Sub(o.Audit.StartedAt)
}

// SetPartial marks the execution as partially completed.
func (o *AutoOutput) SetPartial() {
	o.Status = "partial"
	o.Audit.CompletedAt = time.Now()
	o.Metrics.TotalDuration = o.Audit.CompletedAt.Sub(o.Audit.StartedAt)
}

// SetCheckpointID sets the checkpoint identifier.
func (o *AutoOutput) SetCheckpointID(id string) {
	o.Audit.CheckpointID = id
}

// SetUser sets the user who initiated the workflow.
func (o *AutoOutput) SetUser(user string) {
	o.Audit.User = user
}

// SetHostname sets the hostname where execution occurred.
func (o *AutoOutput) SetHostname(hostname string) {
	o.Audit.Hostname = hostname
}

// SetVersion sets the Specular version.
func (o *AutoOutput) SetVersion(version string) {
	o.Audit.Version = version
}
