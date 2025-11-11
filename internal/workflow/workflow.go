package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felixgeelhaar/specular/internal/drift"
	"github.com/felixgeelhaar/specular/internal/eval"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// WorkflowConfig contains configuration for the E2E workflow
type WorkflowConfig struct {
	// ProjectRoot is the root directory for the workflow
	ProjectRoot string
	// SpecPath is the path to the spec.yaml file
	SpecPath string
	// PolicyPath is the path to the policy.yaml file
	PolicyPath string
	// APISpecPath is the optional path to OpenAPI spec for drift detection
	APISpecPath string
	// DryRun if true, skips actual execution
	DryRun bool
	// FailOnDrift if true, returns error when drift is detected
	FailOnDrift bool
}

// WorkflowResult contains the results of the E2E workflow
type WorkflowResult struct {
	// SpecLock is the generated spec lock
	SpecLock *spec.SpecLock
	// Plan is the generated execution plan
	Plan *plan.Plan
	// EvalResult is the evaluation gate result
	EvalResult *eval.GateReport
	// DriftFindings contains any detected drift
	DriftFindings []drift.Finding
	// Duration is the total workflow duration
	Duration time.Duration
	// Errors contains any errors encountered
	Errors []error
}

// Workflow orchestrates the complete E2E workflow
type Workflow struct {
	config WorkflowConfig
}

// NewWorkflow creates a new workflow orchestrator
func NewWorkflow(config WorkflowConfig) *Workflow {
	return &Workflow{
		config: config,
	}
}

// Execute runs the complete E2E workflow:
// 1. Load and validate spec
// 2. Generate SpecLock
// 3. Generate execution plan
// 4. Run evaluation gate (if policy provided)
// 5. Detect drift
func (w *Workflow) Execute(ctx context.Context) (*WorkflowResult, error) {
	start := time.Now()
	result := &WorkflowResult{
		Errors: []error{},
	}

	// Step 1: Load and validate spec
	productSpec, err := w.loadSpec()
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}

	// Step 2: Generate SpecLock
	specLock, err := w.generateSpecLock(productSpec)
	if err != nil {
		return nil, fmt.Errorf("generate spec lock: %w", err)
	}
	result.SpecLock = specLock

	// Step 3: Generate execution plan
	executionPlan, err := w.generatePlan(ctx, productSpec, specLock)
	if err != nil {
		return nil, fmt.Errorf("generate plan: %w", err)
	}
	result.Plan = executionPlan

	// Step 4: Run evaluation gate (if policy provided and not dry-run)
	if w.config.PolicyPath != "" && !w.config.DryRun {
		evalResult, err := w.runEvalGate(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("run eval gate: %w", err))
		}
		result.EvalResult = evalResult
	}

	// Step 5: Detect drift
	driftFindings, err := w.detectDrift(executionPlan, specLock, productSpec.Features)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("detect drift: %w", err))
	}
	result.DriftFindings = driftFindings

	// Check if we should fail on drift
	if w.config.FailOnDrift && len(driftFindings) > 0 {
		result.Errors = append(result.Errors, fmt.Errorf("drift detected: %d findings", len(driftFindings)))
	}

	result.Duration = time.Since(start)

	// Return error if any step failed
	if len(result.Errors) > 0 {
		return result, fmt.Errorf("workflow completed with %d errors", len(result.Errors))
	}

	return result, nil
}

// loadSpec loads and validates the product spec
func (w *Workflow) loadSpec() (*spec.ProductSpec, error) {
	specPath := w.config.SpecPath
	if !filepath.IsAbs(specPath) {
		specPath = filepath.Join(w.config.ProjectRoot, specPath)
	}

	productSpec, err := spec.LoadSpec(specPath)
	if err != nil {
		return nil, fmt.Errorf("load spec from %s: %w", specPath, err)
	}

	return productSpec, nil
}

// generateSpecLock generates a SpecLock from the product spec
func (w *Workflow) generateSpecLock(productSpec *spec.ProductSpec) (*spec.SpecLock, error) {
	specLock, err := spec.GenerateSpecLock(*productSpec, "1.0.0")
	if err != nil {
		return nil, fmt.Errorf("generate spec lock: %w", err)
	}

	// Save spec lock to .specular/spec.lock.json
	lockPath := filepath.Join(w.config.ProjectRoot, ".specular", "spec.lock.json")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0750); err != nil {
		return nil, fmt.Errorf("create .specular directory: %w", err)
	}

	if err := spec.SaveSpecLock(specLock, lockPath); err != nil {
		return nil, fmt.Errorf("save spec lock: %w", err)
	}

	return specLock, nil
}

// generatePlan generates an execution plan from spec and lock
func (w *Workflow) generatePlan(ctx context.Context, productSpec *spec.ProductSpec, specLock *spec.SpecLock) (*plan.Plan, error) {
	opts := plan.GenerateOptions{
		SpecLock:           specLock,
		EstimateComplexity: true,
	}

	executionPlan, err := plan.Generate(ctx, productSpec, opts)
	if err != nil {
		return nil, fmt.Errorf("generate plan: %w", err)
	}

	// Save plan to plan.json
	planPath := filepath.Join(w.config.ProjectRoot, "plan.json")
	if err := plan.SavePlan(executionPlan, planPath); err != nil {
		return nil, fmt.Errorf("save plan: %w", err)
	}

	return executionPlan, nil
}

// runEvalGate runs the evaluation gate
func (w *Workflow) runEvalGate(ctx context.Context) (*eval.GateReport, error) {
	// Load policy for eval gate
	policyConfig, err := w.loadPolicy()
	if err != nil {
		return nil, fmt.Errorf("load policy: %w", err)
	}

	// Run evaluation
	opts := eval.GateOptions{
		Policy:      policyConfig,
		ProjectRoot: w.config.ProjectRoot,
		Verbose:     false,
	}

	evalResult, err := eval.RunEvalGate(opts)
	if err != nil {
		return nil, fmt.Errorf("run eval gate: %w", err)
	}

	return evalResult, nil
}

// detectDrift detects plan and code drift
func (w *Workflow) detectDrift(executionPlan *plan.Plan, specLock *spec.SpecLock, features []spec.Feature) ([]drift.Finding, error) {
	var allFindings []drift.Finding

	// Detect plan drift
	planFindings := drift.DetectPlanDrift(specLock, executionPlan)
	allFindings = append(allFindings, planFindings...)

	// Detect code drift (if API spec provided)
	if w.config.APISpecPath != "" {
		apiSpecPath := w.config.APISpecPath
		if !filepath.IsAbs(apiSpecPath) {
			apiSpecPath = filepath.Join(w.config.ProjectRoot, apiSpecPath)
		}

		codeFindings := drift.ValidateAPISpec(apiSpecPath, w.config.ProjectRoot, features)
		allFindings = append(allFindings, codeFindings...)
	}

	return allFindings, nil
}

// loadPolicy loads the policy configuration
func (w *Workflow) loadPolicy() (*policy.Policy, error) {
	policyPath := w.config.PolicyPath
	if policyPath == "" {
		// Use default policy
		return policy.DefaultPolicy(), nil
	}

	if !filepath.IsAbs(policyPath) {
		policyPath = filepath.Join(w.config.ProjectRoot, policyPath)
	}

	policyConfig, err := policy.LoadPolicy(policyPath)
	if err != nil {
		return nil, fmt.Errorf("load policy from %s: %w", policyPath, err)
	}

	return policyConfig, nil
}
