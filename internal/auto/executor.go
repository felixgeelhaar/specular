package auto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	"github.com/felixgeelhaar/specular/internal/exec"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/progress"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// TaskExecutor handles execution of tasks from a plan
type TaskExecutor struct {
	policy       *policy.Policy
	config       Config
	spec         *spec.ProductSpec
	actionPlan   *ActionPlan
	router       interface{ GetBudget() *router.Budget } // Use interface for testability
	progressFunc func(taskID, status string, err error)
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(pol *policy.Policy, cfg Config, s *spec.ProductSpec, actionPlan *ActionPlan, r interface{ GetBudget() *router.Budget }) *TaskExecutor {
	return &TaskExecutor{
		policy:     pol,
		config:     cfg,
		spec:       s,
		actionPlan: actionPlan,
		router:     r,
	}
}

// SetProgressCallback sets a callback function for progress updates
func (te *TaskExecutor) SetProgressCallback(fn func(taskID, status string, err error)) {
	te.progressFunc = fn
}

// Execute runs all tasks in the plan with progress tracking and error handling
func (te *TaskExecutor) Execute(ctx context.Context, p *plan.Plan) (*ExecutionStats, error) {
	stats := &ExecutionStats{
		TotalTasks: len(p.Tasks),
		StartTime:  time.Now(),
	}

	// Load or create default policy if not provided
	pol := te.policy
	if pol == nil {
		pol = policy.DefaultPolicy()
	}

	// Setup progress indicator
	progressIndicator := progress.NewIndicator(progress.Config{
		Writer:      os.Stdout,
		ShowSpinner: !te.config.Verbose, // Hide spinner in verbose mode
	})

	// Setup checkpoint for resume capability
	checkpointMgr := checkpoint.NewManager(".specular/checkpoints", true, 30*time.Second)
	cpState := checkpoint.NewState(fmt.Sprintf("auto-%d", time.Now().Unix()))
	cpState.SetMetadata("goal", te.config.Goal)
	cpState.SetMetadata("product", te.spec.Product)

	// Save spec, plan, and action plan JSON for resume capability
	if specJSON, err := json.Marshal(te.spec); err == nil {
		cpState.SetMetadata("spec_json", string(specJSON))
	}
	if planJSON, err := json.Marshal(p); err == nil {
		cpState.SetMetadata("plan_json", string(planJSON))
	}
	if te.actionPlan != nil {
		if actionPlanJSON, err := json.Marshal(te.actionPlan); err == nil {
			cpState.SetMetadata("action_plan_json", string(actionPlanJSON))
		}
	}

	// Initialize tasks in checkpoint
	for _, task := range p.Tasks {
		cpState.UpdateTask(task.ID.String(), "pending", nil)
	}

	// Save initial checkpoint
	if err := checkpointMgr.Save(cpState); err != nil && te.config.Verbose {
		fmt.Printf("Warning: failed to save checkpoint: %v\n", err)
	}

	// Set state in progress indicator
	progressIndicator.SetState(cpState)

	// Create executor
	executor := &exec.Executor{
		Policy:      pol,
		DryRun:      te.config.DryRun,
		ManifestDir: ".specular/manifests",
		ImageCache:  nil, // TODO: Add cache support
		Verbose:     te.config.Verbose,
	}

	// Start progress indicator
	if !te.config.Verbose {
		progressIndicator.Start()
		defer progressIndicator.Stop()
	}

	// Track cost before execution (if router available)
	var initialSpent float64
	if te.router != nil {
		initialBudget := te.router.GetBudget()
		initialSpent = initialBudget.SpentUSD
	}

	// Execute plan with retry logic
	var execResult *exec.ExecutionResult
	var execErr error

	for attempt := 1; attempt <= te.config.MaxRetries; attempt++ {
		if te.config.Verbose {
			fmt.Printf("\n🚀 Execution attempt %d/%d...\n", attempt, te.config.MaxRetries)
		}

		execResult, execErr = executor.Execute(p)

		if execErr == nil && execResult.FailedTasks == 0 {
			// Success - all tasks completed
			break
		}

		// Check if we should retry
		if attempt < te.config.MaxRetries {
			if te.config.Verbose {
				fmt.Printf("⚠️  Attempt %d failed, retrying in %v...\n", attempt, te.config.RetryDelay)
			}

			// Wait before retry
			select {
			case <-ctx.Done():
				return stats, ctx.Err()
			case <-time.After(te.config.RetryDelay):
			}
		}
	}

	// Stop progress indicator before final processing
	if !te.config.Verbose {
		progressIndicator.Stop()
	}

	// Handle execution error
	if execErr != nil {
		cpState.Status = "failed"
		checkpointMgr.Save(cpState) //#nosec G104 -- Best effort checkpoint save
		return stats, fmt.Errorf("execution failed: %w", execErr)
	}

	// Track cost after execution (if router available)
	if te.router != nil {
		finalBudget := te.router.GetBudget()
		stats.TotalCost = finalBudget.SpentUSD - initialSpent
	}

	// Update stats from execution result
	stats.Executed = execResult.SuccessTasks
	stats.Failed = execResult.FailedTasks
	stats.Skipped = execResult.SkippedTasks
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	stats.TaskResults = execResult.TaskResults

	// Update checkpoint with results
	for taskID, taskResult := range execResult.TaskResults {
		if taskResult.ExitCode == 0 {
			cpState.UpdateTask(taskID, "completed", nil)
			if te.progressFunc != nil {
				te.progressFunc(taskID, "completed", nil)
			}
		} else {
			cpState.UpdateTask(taskID, "failed", taskResult.Error)
			if te.progressFunc != nil {
				te.progressFunc(taskID, "failed", taskResult.Error)
			}
		}
	}

	// Mark checkpoint as completed or failed
	if execResult.FailedTasks > 0 {
		cpState.Status = "failed"
		stats.Success = false
	} else {
		cpState.Status = "completed"
		stats.Success = true
	}

	// Save final checkpoint
	if err := checkpointMgr.Save(cpState); err != nil && te.config.Verbose {
		fmt.Printf("Warning: failed to save final checkpoint: %v\n", err)
	}

	// Print summary if not in verbose mode (verbose mode prints as it goes)
	if !te.config.Verbose {
		fmt.Printf("\n")
		fmt.Printf("📊 Execution Summary:\n")
		fmt.Printf("   Total tasks:   %d\n", stats.TotalTasks)
		fmt.Printf("   ✓ Completed:   %d\n", stats.Executed)
		if stats.Failed > 0 {
			fmt.Printf("   ✗ Failed:      %d\n", stats.Failed)
		}
		if stats.Skipped > 0 {
			fmt.Printf("   ⊘ Skipped:     %d\n", stats.Skipped)
		}
		fmt.Printf("   Duration:      %v\n", stats.Duration)
	}

	// Return error if any tasks failed
	if stats.Failed > 0 {
		return stats, fmt.Errorf("%d tasks failed", stats.Failed)
	}

	return stats, nil
}

// ExecutionStats contains statistics about task execution
type ExecutionStats struct {
	TotalTasks  int
	Executed    int
	Failed      int
	Skipped     int
	Success     bool
	TotalCost   float64 // Total cost in USD for AI operations
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	TaskResults map[string]*exec.Result
}

// ExecuteWithCheckpoint runs tasks with an existing checkpoint state (for resume)
func (te *TaskExecutor) ExecuteWithCheckpoint(ctx context.Context, p *plan.Plan, cpState *checkpoint.State, checkpointMgr *checkpoint.Manager) (*ExecutionStats, error) {
	stats := &ExecutionStats{
		TotalTasks: len(p.Tasks),
		StartTime:  time.Now(),
	}

	// Load or create default policy if not provided
	pol := te.policy
	if pol == nil {
		pol = policy.DefaultPolicy()
	}

	// Setup progress indicator
	progressIndicator := progress.NewIndicator(progress.Config{
		Writer:      os.Stdout,
		ShowSpinner: !te.config.Verbose,
	})

	// Set state in progress indicator
	progressIndicator.SetState(cpState)

	// Create executor
	executor := &exec.Executor{
		Policy:      pol,
		DryRun:      te.config.DryRun,
		ManifestDir: ".specular/manifests",
		ImageCache:  nil,
		Verbose:     te.config.Verbose,
	}

	// Start progress indicator
	if !te.config.Verbose {
		progressIndicator.Start()
		defer progressIndicator.Stop()
	}

	// Track cost before execution (if router available)
	var initialSpent float64
	if te.router != nil {
		initialBudget := te.router.GetBudget()
		initialSpent = initialBudget.SpentUSD
	}

	// Execute plan with retry logic
	var execResult *exec.ExecutionResult
	var execErr error

	for attempt := 1; attempt <= te.config.MaxRetries; attempt++ {
		if te.config.Verbose {
			fmt.Printf("\n🚀 Execution attempt %d/%d...\n", attempt, te.config.MaxRetries)
		}

		execResult, execErr = executor.Execute(p)

		if execErr == nil && execResult.FailedTasks == 0 {
			// Success - all tasks completed
			break
		}

		// Check if we should retry
		if attempt < te.config.MaxRetries {
			if te.config.Verbose {
				fmt.Printf("⚠️  Attempt %d failed, retrying in %v...\n", attempt, te.config.RetryDelay)
			}

			// Wait before retry
			select {
			case <-ctx.Done():
				return stats, ctx.Err()
			case <-time.After(te.config.RetryDelay):
			}
		}
	}

	// Stop progress indicator before final processing
	if !te.config.Verbose {
		progressIndicator.Stop()
	}

	// Handle execution error
	if execErr != nil {
		cpState.Status = "failed"
		checkpointMgr.Save(cpState) //#nosec G104 -- Best effort checkpoint save
		return stats, fmt.Errorf("execution failed: %w", execErr)
	}

	// Track cost after execution (if router available)
	if te.router != nil {
		finalBudget := te.router.GetBudget()
		stats.TotalCost = finalBudget.SpentUSD - initialSpent
	}

	// Update stats from execution result
	stats.Executed = execResult.SuccessTasks
	stats.Failed = execResult.FailedTasks
	stats.Skipped = execResult.SkippedTasks
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	stats.TaskResults = execResult.TaskResults

	// Update checkpoint with results
	for taskID, taskResult := range execResult.TaskResults {
		if taskResult.ExitCode == 0 {
			cpState.UpdateTask(taskID, "completed", nil)
			if te.progressFunc != nil {
				te.progressFunc(taskID, "completed", nil)
			}
		} else {
			cpState.UpdateTask(taskID, "failed", taskResult.Error)
			if te.progressFunc != nil {
				te.progressFunc(taskID, "failed", taskResult.Error)
			}
		}
	}

	// Mark checkpoint as completed or failed
	if execResult.FailedTasks > 0 {
		cpState.Status = "failed"
		stats.Success = false
	} else {
		cpState.Status = "completed"
		stats.Success = true
	}

	// Save final checkpoint
	if err := checkpointMgr.Save(cpState); err != nil && te.config.Verbose {
		fmt.Printf("Warning: failed to save final checkpoint: %v\n", err)
	}

	// Print summary if not in verbose mode
	if !te.config.Verbose {
		fmt.Printf("\n")
		fmt.Printf("📊 Execution Summary:\n")
		fmt.Printf("   Total tasks:   %d\n", stats.TotalTasks)
		fmt.Printf("   ✓ Completed:   %d\n", stats.Executed)
		if stats.Failed > 0 {
			fmt.Printf("   ✗ Failed:      %d\n", stats.Failed)
		}
		if stats.Skipped > 0 {
			fmt.Printf("   ⊘ Skipped:     %d\n", stats.Skipped)
		}
		fmt.Printf("   Duration:      %v\n", stats.Duration)
	}

	// Return error if any tasks failed
	if stats.Failed > 0 {
		return stats, fmt.Errorf("%d tasks failed", stats.Failed)
	}

	return stats, nil
}
