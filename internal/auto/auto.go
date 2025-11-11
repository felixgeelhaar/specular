package auto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	"github.com/felixgeelhaar/specular/internal/hooks"
	"github.com/felixgeelhaar/specular/internal/patch"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/internal/trace"
	"gopkg.in/yaml.v3"
)

// Orchestrator manages the autonomous workflow
type Orchestrator struct {
	router         *router.Router
	config         Config
	parser         *GoalParser
	actionPlan     *ActionPlan
	policyChecker  PolicyChecker        // Optional policy checker for step validation
	tracer         *trace.Logger        // Optional trace logger for detailed execution tracking
	patchGenerator *patch.DiffGenerator // Optional patch generator for rollback support
	patchWriter    *patch.Writer        // Optional patch writer for saving patches
	hookRegistry   *hooks.Registry      // Optional hook registry for lifecycle notifications
}

// NewOrchestrator creates a new orchestrator with the given router and config
func NewOrchestrator(r *router.Router, config Config) *Orchestrator {
	return &Orchestrator{
		router: r,
		config: config,
		parser: NewGoalParser(r),
	}
}

// SetPolicyChecker sets the policy checker for step validation.
// This must be called before Execute if policy checks are desired.
func (o *Orchestrator) SetPolicyChecker(checker PolicyChecker) {
	o.policyChecker = checker
}

// SetTracer sets the trace logger for detailed execution tracking.
// This must be called before Execute if tracing is desired.
func (o *Orchestrator) SetTracer(tracer *trace.Logger) {
	o.tracer = tracer
}

// SetPatchGenerator sets the patch generator and writer for rollback support.
// This must be called before Execute if patch generation is desired.
func (o *Orchestrator) SetPatchGenerator(workingDir, patchDir string) {
	o.patchGenerator = patch.NewDiffGenerator(workingDir)
	o.patchWriter = patch.NewWriter(patchDir)
}

// SetHookRegistry sets the hook registry for lifecycle notifications.
// This must be called before Execute if hooks are desired.
func (o *Orchestrator) SetHookRegistry(registry *hooks.Registry) {
	o.hookRegistry = registry
}

// triggerHook safely triggers a hook event if the registry is configured
func (o *Orchestrator) triggerHook(ctx context.Context, eventType hooks.EventType, workflowID string, data map[string]interface{}) {
	if o.hookRegistry == nil {
		return
	}

	event := hooks.NewEvent(eventType, workflowID, data)
	_ = o.hookRegistry.Trigger(ctx, event)

	// Hook execution errors are logged internally by the hooks package
	// We don't block workflow execution on hook failures
}

// Execute runs the complete autonomous workflow
func (o *Orchestrator) Execute(ctx context.Context) (*Result, error) {
	start := time.Now()
	result := &Result{
		Success: false,
		Errors:  []error{},
	}

	// Track workflow ID for hooks
	var workflowID string

	// Defer workflow failed hook if execution doesn't complete successfully
	defer func() {
		if !result.Success && workflowID != "" {
			o.triggerHook(ctx, hooks.EventWorkflowFailed, workflowID, map[string]interface{}{
				"duration": time.Since(start).String(),
				"error":    fmt.Sprintf("%v", result.Errors),
			})
		}
	}()

	// Check if resuming from checkpoint
	if o.config.ResumeFrom != "" {
		return o.executeResume(ctx, start)
	}

	// Create action plan for workflow tracking
	o.actionPlan = CreateDefaultActionPlan(o.config.Goal, o.config.Profile)
	result.ActionPlan = o.actionPlan

	// Create JSON output if enabled
	var autoOutput *AutoOutput
	if o.config.JSONOutput {
		autoOutput = NewAutoOutput(o.config.Goal, o.config.Profile)
		result.AutoOutput = autoOutput
	}

	// Initialize policy context for tracking execution state
	executionStart := time.Now()
	completedSteps := 0
	totalCost := 0.0

	// Log workflow start
	if o.tracer != nil {
		o.tracer.LogWorkflowStart(o.config.Goal, o.config.Profile) //#nosec G104 -- Logging errors not critical
	}

	// Trigger workflow start hook
	workflowID = "unknown"
	if autoOutput != nil {
		workflowID = autoOutput.Audit.CheckpointID
	}
	o.triggerHook(ctx, hooks.EventWorkflowStart, workflowID, map[string]interface{}{
		"goal":    o.config.Goal,
		"profile": o.config.Profile,
	})

	fmt.Printf("📋 Created action plan with %d steps\n\n", len(o.actionPlan.Steps))

	// Pre-flight: Check budget for spec generation
	if o.router != nil {
		budget := o.router.GetBudget()
		estimatedCost := EstimateSpecGenerationCost(len(o.config.Goal), 0.01) // $0.01 per MTok typical
		warning, err := CheckBudgetWithWarning(budget, estimatedCost, "spec generation")
		if err != nil {
			return nil, fmt.Errorf("budget check failed: %w", err)
		}
		if warning != "" {
			fmt.Printf("%s\n\n", warning)
		}
	}

	// Step 1: Parse goal into spec
	step1, _ := o.actionPlan.GetStep("step-1")

	// Check policy before executing step
	allowed, policyEvent, err := o.checkPolicy(ctx, o.policyChecker, step1, 0, completedSteps, totalCost, executionStart)
	if err != nil {
		return nil, fmt.Errorf("step-1 policy check: %w", err)
	}
	if autoOutput != nil && policyEvent != nil {
		autoOutput.AddPolicy(*policyEvent)
	}
	if !allowed {
		fmt.Printf("🚫 Step 1 blocked by policy: %s\n", policyEvent.Reason)
		if autoOutput != nil {
			autoOutput.SetPartial()
		}
		result.Success = false
		result.Duration = time.Since(start)
		return result, fmt.Errorf("step-1 blocked by policy: %s", policyEvent.Reason)
	}

	step1Start := time.Now()
	if err := o.actionPlan.UpdateStepStatus("step-1", StepStatusInProgress); err != nil {
		return nil, fmt.Errorf("update step status: %w", err)
	}
	if o.tracer != nil {
		o.tracer.LogStepStart("step-1", "Generate specification") //#nosec G104 -- Logging errors not critical
	}

	// Capture snapshot before step
	step1Snapshot, err := o.captureSnapshot()
	if err != nil {
		fmt.Printf("⚠️  Failed to capture snapshot: %v\n", err)
	}

	fmt.Println("🤖 Generating specification from goal...")
	productSpec, err := o.parser.ParseGoal(ctx, o.config.Goal)
	if err != nil {
		step, _ := o.actionPlan.GetStep("step-1")
		step.Error = err.Error()
		_ = o.actionPlan.UpdateStepStatus("step-1", StepStatusFailed) //#nosec G104 -- Status update errors handled at workflow level
		if o.tracer != nil {
			o.tracer.LogStepFail("step-1", "Generate specification", err) //#nosec G104 -- Logging errors not critical
		}
		if autoOutput != nil {
			autoOutput.AddStepResult(StepResult{
				ID:          "step-1",
				Type:        "spec:update",
				Status:      "failed",
				StartedAt:   step1Start,
				CompletedAt: time.Now(),
				Duration:    time.Since(step1Start),
				Error:       err.Error(),
			})
		}
		return nil, fmt.Errorf("parse goal: %w", err)
	}
	result.Spec = productSpec
	if err := o.actionPlan.UpdateStepStatus("step-1", StepStatusCompleted); err != nil {
		return nil, fmt.Errorf("update step status: %w", err)
	}
	step1Cost := EstimateSpecGenerationCost(len(o.config.Goal), 0.01)
	if o.tracer != nil {
		o.tracer.LogStepComplete("step-1", "Generate specification", time.Since(step1Start), step1Cost) //#nosec G104 -- Logging errors not critical
	}
	if autoOutput != nil {
		autoOutput.AddStepResult(StepResult{
			ID:          "step-1",
			Type:        "spec:update",
			Status:      "completed",
			StartedAt:   step1Start,
			CompletedAt: time.Now(),
			Duration:    time.Since(step1Start),
			CostUSD:     step1Cost,
		})
	}
	completedSteps++
	totalCost += step1Cost
	fmt.Printf("✅ Generated spec: %s\n", productSpec.Product)
	fmt.Printf("   Features: %d\n\n", len(productSpec.Features))

	// Generate and save patch for step 1
	if err := o.generateAndSavePatch("step-1", "spec:update", "Generate specification", step1Snapshot); err != nil {
		fmt.Printf("⚠️  Patch generation warning: %v\n", err)
	}

	// Step 2: Generate spec lock
	step2, _ := o.actionPlan.GetStep("step-2")

	// Check policy before executing step
	allowed, policyEvent, err = o.checkPolicy(ctx, o.policyChecker, step2, 1, completedSteps, totalCost, executionStart)
	if err != nil {
		return nil, fmt.Errorf("step-2 policy check: %w", err)
	}
	if autoOutput != nil && policyEvent != nil {
		autoOutput.AddPolicy(*policyEvent)
	}
	if !allowed {
		fmt.Printf("🚫 Step 2 blocked by policy: %s\n", policyEvent.Reason)
		if autoOutput != nil {
			autoOutput.SetPartial()
		}
		result.Success = false
		result.Duration = time.Since(start)
		return result, fmt.Errorf("step-2 blocked by policy: %s", policyEvent.Reason)
	}

	step2Start := time.Now()
	if err := o.actionPlan.UpdateStepStatus("step-2", StepStatusInProgress); err != nil {
		return nil, fmt.Errorf("update step status: %w", err)
	}

	// Capture snapshot before step
	step2Snapshot, err := o.captureSnapshot()
	if err != nil {
		fmt.Printf("⚠️  Failed to capture snapshot: %v\n", err)
	}

	fmt.Println("🔒 Locking specification...")
	specLock, err := o.generateSpecLock(productSpec)
	if err != nil {
		step, _ := o.actionPlan.GetStep("step-2")
		step.Error = err.Error()
		_ = o.actionPlan.UpdateStepStatus("step-2", StepStatusFailed) //#nosec G104 -- Status update errors handled at workflow level
		if autoOutput != nil {
			autoOutput.AddStepResult(StepResult{
				ID:          "step-2",
				Type:        "spec:lock",
				Status:      "failed",
				StartedAt:   step2Start,
				CompletedAt: time.Now(),
				Duration:    time.Since(step2Start),
				Error:       err.Error(),
			})
		}
		return nil, fmt.Errorf("generate spec lock: %w", err)
	}
	result.SpecLock = specLock
	if err := o.actionPlan.UpdateStepStatus("step-2", StepStatusCompleted); err != nil {
		return nil, fmt.Errorf("update step status: %w", err)
	}
	step2Cost := 0.01 // Locking is cheap
	if autoOutput != nil {
		autoOutput.AddStepResult(StepResult{
			ID:          "step-2",
			Type:        "spec:lock",
			Status:      "completed",
			StartedAt:   step2Start,
			CompletedAt: time.Now(),
			Duration:    time.Since(step2Start),
			CostUSD:     step2Cost,
		})
	}
	completedSteps++
	totalCost += step2Cost
	fmt.Printf("✅ Spec locked: %d features\n\n", len(specLock.Features))

	// Generate and save patch for step 2
	if err := o.generateAndSavePatch("step-2", "spec:lock", "Lock specification", step2Snapshot); err != nil {
		fmt.Printf("⚠️  Patch generation warning: %v\n", err)
	}

	// Pre-flight: Check budget for plan generation
	if o.router != nil {
		budget := o.router.GetBudget()
		estimatedCost := EstimatePlanGenerationCost(len(productSpec.Features), 0.01) // $0.01 per MTok typical
		warning, err := CheckBudgetWithWarning(budget, estimatedCost, "plan generation")
		if err != nil {
			return nil, fmt.Errorf("budget check failed: %w", err)
		}
		if warning != "" {
			fmt.Printf("%s\n\n", warning)
		}
	}

	// Step 3: Generate execution plan
	step3, _ := o.actionPlan.GetStep("step-3")

	// Check policy before executing step
	allowed, policyEvent, err = o.checkPolicy(ctx, o.policyChecker, step3, 2, completedSteps, totalCost, executionStart)
	if err != nil {
		return nil, fmt.Errorf("step-3 policy check: %w", err)
	}
	if autoOutput != nil && policyEvent != nil {
		autoOutput.AddPolicy(*policyEvent)
	}
	if !allowed {
		fmt.Printf("🚫 Step 3 blocked by policy: %s\n", policyEvent.Reason)
		if autoOutput != nil {
			autoOutput.SetPartial()
		}
		result.Success = false
		result.Duration = time.Since(start)
		return result, fmt.Errorf("step-3 blocked by policy: %s", policyEvent.Reason)
	}

	step3Start := time.Now()
	if err := o.actionPlan.UpdateStepStatus("step-3", StepStatusInProgress); err != nil {
		return nil, fmt.Errorf("update step status: %w", err)
	}

	// Capture snapshot before step
	step3Snapshot, err := o.captureSnapshot()
	if err != nil {
		fmt.Printf("⚠️  Failed to capture snapshot: %v\n", err)
	}

	fmt.Println("📋 Generating execution plan...")
	execPlan, err := o.generatePlan(ctx, productSpec, specLock)
	if err != nil {
		step, _ := o.actionPlan.GetStep("step-3")
		step.Error = err.Error()
		_ = o.actionPlan.UpdateStepStatus("step-3", StepStatusFailed) //#nosec G104 -- Status update errors handled at workflow level
		if autoOutput != nil {
			autoOutput.AddStepResult(StepResult{
				ID:          "step-3",
				Type:        "plan:gen",
				Status:      "failed",
				StartedAt:   step3Start,
				CompletedAt: time.Now(),
				Duration:    time.Since(step3Start),
				Error:       err.Error(),
			})
		}
		return nil, fmt.Errorf("generate plan: %w", err)
	}
	result.Plan = execPlan
	if err := o.actionPlan.UpdateStepStatus("step-3", StepStatusCompleted); err != nil {
		return nil, fmt.Errorf("update step status: %w", err)
	}
	step3Cost := EstimatePlanGenerationCost(len(productSpec.Features), 0.01)
	if autoOutput != nil {
		autoOutput.AddStepResult(StepResult{
			ID:          "step-3",
			Type:        "plan:gen",
			Status:      "completed",
			StartedAt:   step3Start,
			CompletedAt: time.Now(),
			Duration:    time.Since(step3Start),
			CostUSD:     step3Cost,
		})
	}
	completedSteps++
	totalCost += step3Cost
	fmt.Printf("✅ Plan created: %d tasks\n\n", len(execPlan.Tasks))

	// Trigger plan created hook
	o.triggerHook(ctx, hooks.EventPlanCreated, workflowID, map[string]interface{}{
		"steps": len(execPlan.Tasks),
	})

	// Generate and save patch for step 3
	if err := o.generateAndSavePatch("step-3", "plan:gen", "Generate execution plan", step3Snapshot); err != nil {
		fmt.Printf("⚠️  Patch generation warning: %v\n", err)
	}

	// Apply scope filtering if specified
	if len(o.config.ScopePatterns) > 0 {
		scope, err := NewScope(o.config.ScopePatterns, o.config.IncludeDependencies)
		if err != nil {
			return nil, fmt.Errorf("invalid scope patterns: %w", err)
		}

		// Estimate impact before filtering
		matched, total := scope.EstimateImpact(execPlan, productSpec)
		fmt.Printf("🎯 Applying scope filter: %s\n", scope.Summary())
		fmt.Printf("   Matched: %d/%d tasks\n\n", matched, total)

		// Filter the plan
		execPlan = scope.FilterPlan(execPlan, productSpec)
		result.Plan = execPlan
		fmt.Printf("✅ Filtered plan: %d tasks\n\n", len(execPlan.Tasks))
	}

	// Save spec, plan, and action plan to output directory if specified
	if o.config.OutputDir != "" {
		if err := o.saveOutputFiles(productSpec, specLock, execPlan, o.actionPlan); err != nil {
			fmt.Printf("⚠️  Warning: failed to save output files: %v\n\n", err)
		}
	}

	// Step 4: Approval gate (if enabled)
	if o.config.RequireApproval && !o.config.DryRun {
		approved, err := ShowApprovalGate(execPlan, productSpec)
		if err != nil {
			return nil, fmt.Errorf("approval gate: %w", err)
		}
		if !approved {
			return result, fmt.Errorf("plan not approved by user")
		}
		fmt.Println()
	}

	if o.config.DryRun {
		fmt.Println("🏁 Dry run complete (no execution)")
		result.Success = true
		result.Duration = time.Since(start)
		if autoOutput != nil {
			autoOutput.SetCompleted()
		}
		return result, nil
	}

	// Pre-flight: Check budget for task execution
	if o.router != nil {
		budget := o.router.GetBudget()
		estimatedCost := EstimateTaskExecutionCost(len(execPlan.Tasks), 0.01) // $0.01 per MTok typical
		warning, err := CheckBudgetWithWarning(budget, estimatedCost, "task execution")
		if err != nil {
			return nil, fmt.Errorf("budget check failed: %w", err)
		}
		if warning != "" {
			fmt.Printf("%s\n\n", warning)
		}

		// Check per-task budget if configured
		if o.config.MaxCostPerTask > 0 {
			perTaskEstimate := estimatedCost / float64(len(execPlan.Tasks))
			if err := CheckPerTaskBudget(perTaskEstimate, o.config.MaxCostPerTask, "average"); err != nil {
				fmt.Printf("⚠️  Warning: %v\n\n", err)
			}
		}
	}

	// Step 4: Execute plan
	step4, _ := o.actionPlan.GetStep("step-4")

	// Check policy before executing step
	allowed, policyEvent, err = o.checkPolicy(ctx, o.policyChecker, step4, 3, completedSteps, totalCost, executionStart)
	if err != nil {
		return nil, fmt.Errorf("step-4 policy check: %w", err)
	}
	if autoOutput != nil && policyEvent != nil {
		autoOutput.AddPolicy(*policyEvent)
	}
	if !allowed {
		fmt.Printf("🚫 Step 4 blocked by policy: %s\n", policyEvent.Reason)
		if autoOutput != nil {
			autoOutput.SetPartial()
		}
		result.Success = false
		result.Duration = time.Since(start)
		result.TotalCost = totalCost
		return result, fmt.Errorf("step-4 blocked by policy: %s", policyEvent.Reason)
	}

	step4Start := time.Now()
	if err := o.actionPlan.UpdateStepStatus("step-4", StepStatusInProgress); err != nil {
		return nil, fmt.Errorf("update step status: %w", err)
	}

	// Capture snapshot before step
	step4Snapshot, err := o.captureSnapshot()
	if err != nil {
		fmt.Printf("⚠️  Failed to capture snapshot: %v\n", err)
	}

	fmt.Println("🚀 Executing plan...")

	// Get initial budget before execution
	initialBudget := o.router.GetBudget()

	executor := NewTaskExecutor(nil, o.config, productSpec, o.actionPlan, o.router)
	execStats, err := executor.Execute(ctx, execPlan)
	if err != nil {
		step, _ := o.actionPlan.GetStep("step-4")
		step.Error = err.Error()
		_ = o.actionPlan.UpdateStepStatus("step-4", StepStatusFailed) //#nosec G104 -- Status update errors handled at workflow level
		result.Success = false
		result.TasksExecuted = execStats.Executed
		result.TasksFailed = execStats.Failed
		result.Duration = time.Since(start)
		result.Errors = append(result.Errors, err)
		if autoOutput != nil {
			autoOutput.AddStepResult(StepResult{
				ID:          "step-4",
				Type:        "build:run",
				Status:      "failed",
				StartedAt:   step4Start,
				CompletedAt: time.Now(),
				Duration:    time.Since(step4Start),
				Error:       err.Error(),
			})
			autoOutput.SetFailed()
		}
		return result, fmt.Errorf("execution failed: %w", err)
	}

	// Get final budget after execution
	finalBudget := o.router.GetBudget()
	executionCost := finalBudget.SpentUSD - initialBudget.SpentUSD

	// Mark step-4 as completed
	if err := o.actionPlan.UpdateStepStatus("step-4", StepStatusCompleted); err != nil {
		return nil, fmt.Errorf("update step status: %w", err)
	}

	// Add step-4 result to AutoOutput
	if autoOutput != nil {
		autoOutput.AddStepResult(StepResult{
			ID:          "step-4",
			Type:        "build:run",
			Status:      "completed",
			StartedAt:   step4Start,
			CompletedAt: time.Now(),
			Duration:    time.Since(step4Start),
			CostUSD:     executionCost,
		})
	}

	// Update result with execution stats and cost
	result.Success = execStats.Success
	result.TasksExecuted = execStats.Executed
	result.TasksFailed = execStats.Failed
	result.TotalCost = execStats.TotalCost + executionCost // Include spec generation + execution costs
	result.Duration = time.Since(start)

	// Print cost summary
	if result.TotalCost > 0 {
		fmt.Printf("\n💰 Cost Summary:\n")
		fmt.Printf("   Spec generation: $%.4f\n", initialBudget.SpentUSD)
		fmt.Printf("   Task execution:  $%.4f\n", executionCost)
		fmt.Printf("   Total cost:      $%.4f\n", result.TotalCost)
		fmt.Printf("   Remaining:       $%.2f / $%.2f\n", finalBudget.RemainingUSD, finalBudget.LimitUSD)
	}

	// Generate and save patch for step 4
	if err := o.generateAndSavePatch("step-4", "build:run", "Execute plan", step4Snapshot); err != nil {
		fmt.Printf("⚠️  Patch generation warning: %v\n", err)
	}

	// Finalize AutoOutput if enabled
	if autoOutput != nil {
		autoOutput.SetCompleted()
	}

	// Log workflow completion
	if o.tracer != nil {
		o.tracer.LogWorkflowComplete(result.Success, result.Duration, result.TotalCost) //#nosec G104 -- Logging errors not critical
		o.tracer.Close()                                                                //#nosec G104 -- Logging close errors not critical
	}

	// Trigger workflow complete hook
	o.triggerHook(ctx, hooks.EventWorkflowComplete, workflowID, map[string]interface{}{
		"duration": result.Duration.String(),
		"cost":     result.TotalCost,
		"success":  result.Success,
	})

	return result, nil
}

// checkPolicy validates a step against policy constraints.
// Returns (allowed bool, policyEvent *PolicyEvent, error).
func (o *Orchestrator) checkPolicy(
	ctx context.Context,
	policyChecker PolicyChecker,
	step *ActionStep,
	stepIndex int,
	completedSteps int,
	totalCost float64,
	executionStart time.Time,
) (bool, *PolicyEvent, error) {
	if policyChecker == nil {
		return true, nil, nil // No policy checker, allow by default
	}

	// Create policy context with current execution state
	policyCtx := NewPolicyContext(step, o.actionPlan, stepIndex)
	policyCtx.CompletedSteps = completedSteps
	policyCtx.TotalCostSoFar = totalCost
	policyCtx.ExecutionStartTime = executionStart

	// Add policy context to Go context
	ctx = context.WithValue(ctx, "policy_context", policyCtx)

	// Check the policy
	result, err := policyChecker.CheckStep(ctx, step)
	if err != nil {
		return false, nil, fmt.Errorf("policy check failed: %w", err)
	}

	// Create policy event for audit trail
	policyEvent := &PolicyEvent{
		StepID:      step.ID,
		Timestamp:   time.Now(),
		CheckerName: policyChecker.Name(),
		Allowed:     result.Allowed,
		Reason:      result.Reason,
		Warnings:    result.Warnings,
		Metadata:    result.Metadata,
	}

	// Log warnings
	for _, warning := range result.Warnings {
		fmt.Printf("⚠️  Policy warning: %s\n", warning)
	}

	// Log to tracer
	if o.tracer != nil {
		o.tracer.LogPolicyCheck(step.ID, result.Allowed, result.Reason, result.Metadata) //#nosec G104 -- Logging errors not critical
	}

	return result.Allowed, policyEvent, nil
}

// captureSnapshot captures the current working directory state before a step
func (o *Orchestrator) captureSnapshot() (map[string]string, error) {
	if o.patchGenerator == nil {
		return nil, nil // Patch generation not enabled
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	snapshot, err := patch.CaptureDirectorySnapshot(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to capture snapshot: %w", err)
	}

	return snapshot, nil
}

// generateAndSavePatch generates a patch from before/after snapshots and saves it
func (o *Orchestrator) generateAndSavePatch(stepID, stepType, description string, beforeSnapshot map[string]string) error {
	if o.patchGenerator == nil || o.patchWriter == nil {
		return nil // Patch generation not enabled
	}

	// Get workflow ID from tracer or generate one
	workflowID := "auto"
	if o.tracer != nil {
		workflowID = o.tracer.GetWorkflowID()
	}

	// Generate patch
	patchData, err := o.patchGenerator.GeneratePatch(stepID, stepType, workflowID, description, beforeSnapshot)
	if err != nil {
		// Log warning but don't fail the step
		fmt.Printf("⚠️  Failed to generate patch for %s: %v\n", stepID, err)
		return nil
	}

	// Skip empty patches
	if patchData.IsEmpty() {
		return nil
	}

	// Save patch
	patchPath, err := o.patchWriter.WritePatch(patchData)
	if err != nil {
		// Log warning but don't fail the step
		fmt.Printf("⚠️  Failed to save patch for %s: %v\n", stepID, err)
		return nil
	}

	fmt.Printf("💾 Saved patch: %s (%d files, +%d -%d)\n", patchPath, patchData.FilesChanged, patchData.Insertions, patchData.Deletions)
	return nil
}

// generateSpecLock creates a locked specification with hashes
func (o *Orchestrator) generateSpecLock(productSpec *spec.ProductSpec) (*spec.SpecLock, error) {
	return spec.GenerateSpecLock(*productSpec, "1.0.0")
}

// generatePlan creates an execution plan from the spec and lock
func (o *Orchestrator) generatePlan(ctx context.Context, productSpec *spec.ProductSpec, specLock *spec.SpecLock) (*plan.Plan, error) {
	opts := plan.GenerateOptions{
		SpecLock:           specLock,
		EstimateComplexity: true,
	}
	return plan.Generate(ctx, productSpec, opts)
}

// executeResume resumes execution from a checkpoint
func (o *Orchestrator) executeResume(ctx context.Context, start time.Time) (*Result, error) {
	result := &Result{
		Success: false,
		Errors:  []error{},
	}

	// Load checkpoint
	fmt.Printf("🔄 Resuming from checkpoint: %s\n", o.config.ResumeFrom)
	checkpointMgr := checkpoint.NewManager(".specular/checkpoints", true, 30*time.Second)
	cpState, err := checkpointMgr.Load(o.config.ResumeFrom)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Restore goal from checkpoint
	goal, _ := cpState.GetMetadata("goal")
	product, _ := cpState.GetMetadata("product")
	fmt.Printf("📋 Resuming: %s\n", product)
	fmt.Printf("   Goal: %s\n", goal)

	// Load spec JSON from checkpoint
	specJSON, ok := cpState.GetMetadata("spec_json")
	if !ok {
		return nil, fmt.Errorf("checkpoint missing spec data")
	}
	var productSpec spec.ProductSpec
	if err := json.Unmarshal([]byte(specJSON), &productSpec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec from checkpoint: %w", err)
	}
	result.Spec = &productSpec

	// Load plan JSON from checkpoint
	planJSON, ok := cpState.GetMetadata("plan_json")
	if !ok {
		return nil, fmt.Errorf("checkpoint missing plan data")
	}
	var execPlan plan.Plan
	if err := json.Unmarshal([]byte(planJSON), &execPlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan from checkpoint: %w", err)
	}
	result.Plan = &execPlan

	// Load action plan JSON from checkpoint (optional for backwards compatibility)
	var actionPlan *ActionPlan
	if actionPlanJSON, ok := cpState.GetMetadata("action_plan_json"); ok {
		actionPlan = &ActionPlan{}
		if err := json.Unmarshal([]byte(actionPlanJSON), actionPlan); err != nil {
			fmt.Printf("Warning: failed to unmarshal action plan from checkpoint: %v\n", err)
			// Create default action plan if loading fails
			actionPlan = CreateDefaultActionPlan(goal, "")
		}
	} else {
		// Create default action plan for backwards compatibility
		actionPlan = CreateDefaultActionPlan(goal, "")
	}
	result.ActionPlan = actionPlan

	// Get task completion status
	completed := cpState.GetCompletedTasks()
	pending := cpState.GetPendingTasks()
	failed := cpState.GetFailedTasks()

	fmt.Printf("\n📊 Checkpoint status:\n")
	fmt.Printf("   ✓ Completed: %d\n", len(completed))
	fmt.Printf("   ⏳ Pending:   %d\n", len(pending))
	if len(failed) > 0 {
		fmt.Printf("   ✗ Failed:    %d\n", len(failed))
	}
	fmt.Println()

	// Filter plan to only include pending and failed tasks
	filteredTasks := []plan.Task{}
	completedMap := make(map[string]bool)
	for _, taskID := range completed {
		completedMap[taskID] = true
	}

	for _, task := range execPlan.Tasks {
		if !completedMap[task.ID.String()] {
			filteredTasks = append(filteredTasks, task)
		}
	}

	// Create filtered plan
	filteredPlan := &plan.Plan{
		Tasks: filteredTasks,
	}

	fmt.Printf("🚀 Resuming execution (%d tasks remaining)...\n", len(filteredTasks))

	// Get initial budget before execution
	initialBudget := o.router.GetBudget()

	// Execute remaining tasks
	executor := NewTaskExecutor(nil, o.config, &productSpec, actionPlan, o.router)
	execStats, err := executor.ExecuteWithCheckpoint(ctx, filteredPlan, cpState, checkpointMgr)
	if err != nil {
		result.Success = false
		result.TasksExecuted = execStats.Executed
		result.TasksFailed = execStats.Failed
		result.Duration = time.Since(start)
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("resumed execution failed: %w", err)
	}

	// Get final budget after execution
	finalBudget := o.router.GetBudget()
	executionCost := finalBudget.SpentUSD - initialBudget.SpentUSD

	// Update result
	result.Success = execStats.Success
	result.TasksExecuted = len(completed) + execStats.Executed // Include previously completed tasks
	result.TasksFailed = execStats.Failed
	result.TotalCost = executionCost
	result.Duration = time.Since(start)

	return result, nil
}

// saveOutputFiles saves spec, lock, plan, and action plan to the output directory
func (o *Orchestrator) saveOutputFiles(productSpec *spec.ProductSpec, specLock *spec.SpecLock, execPlan *plan.Plan, actionPlan *ActionPlan) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(o.config.OutputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save spec as YAML
	specYAML, err := yaml.Marshal(productSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %w", err)
	}
	specPath := filepath.Join(o.config.OutputDir, "spec.yaml")
	if err := os.WriteFile(specPath, specYAML, 0o600); err != nil {
		return fmt.Errorf("failed to write spec file: %w", err)
	}

	// Save spec lock as JSON
	lockJSON, err := json.MarshalIndent(specLock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal spec lock: %w", err)
	}
	lockPath := filepath.Join(o.config.OutputDir, "spec.lock.json")
	if err := os.WriteFile(lockPath, lockJSON, 0o600); err != nil {
		return fmt.Errorf("failed to write spec lock file: %w", err)
	}

	// Save plan as JSON
	planJSON, err := json.MarshalIndent(execPlan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}
	planPath := filepath.Join(o.config.OutputDir, "plan.json")
	if err := os.WriteFile(planPath, planJSON, 0o600); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	// Save action plan as JSON
	actionPlanJSON, err := json.MarshalIndent(actionPlan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal action plan: %w", err)
	}
	actionPlanPath := filepath.Join(o.config.OutputDir, "action-plan.json")
	if err := os.WriteFile(actionPlanPath, actionPlanJSON, 0o600); err != nil {
		return fmt.Errorf("failed to write action plan file: %w", err)
	}

	fmt.Printf("📁 Saved output files to: %s\n", o.config.OutputDir)
	fmt.Printf("   - spec.yaml\n")
	fmt.Printf("   - spec.lock.json\n")
	fmt.Printf("   - plan.json\n")
	fmt.Printf("   - action-plan.json\n\n")

	return nil
}
