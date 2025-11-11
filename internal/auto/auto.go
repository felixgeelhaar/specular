package auto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
	"gopkg.in/yaml.v3"
)

// Orchestrator manages the autonomous workflow
type Orchestrator struct {
	router *router.Router
	config Config
	parser *GoalParser
}

// NewOrchestrator creates a new orchestrator with the given router and config
func NewOrchestrator(r *router.Router, config Config) *Orchestrator {
	return &Orchestrator{
		router: r,
		config: config,
		parser: NewGoalParser(r),
	}
}

// Execute runs the complete autonomous workflow
func (o *Orchestrator) Execute(ctx context.Context) (*Result, error) {
	start := time.Now()
	result := &Result{
		Success: false,
		Errors:  []error{},
	}

	// Check if resuming from checkpoint
	if o.config.ResumeFrom != "" {
		return o.executeResume(ctx, start)
	}

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
	fmt.Println("🤖 Generating specification from goal...")
	productSpec, err := o.parser.ParseGoal(ctx, o.config.Goal)
	if err != nil {
		return nil, fmt.Errorf("parse goal: %w", err)
	}
	result.Spec = productSpec
	fmt.Printf("✅ Generated spec: %s\n", productSpec.Product)
	fmt.Printf("   Features: %d\n\n", len(productSpec.Features))

	// Step 2: Generate spec lock
	fmt.Println("🔒 Locking specification...")
	specLock, err := o.generateSpecLock(productSpec)
	if err != nil {
		return nil, fmt.Errorf("generate spec lock: %w", err)
	}
	result.SpecLock = specLock
	fmt.Printf("✅ Spec locked: %d features\n\n", len(specLock.Features))

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
	fmt.Println("📋 Generating execution plan...")
	execPlan, err := o.generatePlan(ctx, productSpec, specLock)
	if err != nil {
		return nil, fmt.Errorf("generate plan: %w", err)
	}
	result.Plan = execPlan
	fmt.Printf("✅ Plan created: %d tasks\n\n", len(execPlan.Tasks))

	// Save spec and plan to output directory if specified
	if o.config.OutputDir != "" {
		if err := o.saveOutputFiles(productSpec, specLock, execPlan); err != nil {
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

	// Step 5: Execute plan
	fmt.Println("🚀 Executing plan...")

	// Get initial budget before execution
	initialBudget := o.router.GetBudget()

	executor := NewTaskExecutor(nil, o.config, productSpec, o.router)
	execStats, err := executor.Execute(ctx, execPlan)
	if err != nil {
		result.Success = false
		result.TasksExecuted = execStats.Executed
		result.TasksFailed = execStats.Failed
		result.Duration = time.Since(start)
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("execution failed: %w", err)
	}

	// Get final budget after execution
	finalBudget := o.router.GetBudget()
	executionCost := finalBudget.SpentUSD - initialBudget.SpentUSD

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

	return result, nil
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
	executor := NewTaskExecutor(nil, o.config, &productSpec, o.router)
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

// saveOutputFiles saves spec, lock, and plan to the output directory
func (o *Orchestrator) saveOutputFiles(productSpec *spec.ProductSpec, specLock *spec.SpecLock, execPlan *plan.Plan) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(o.config.OutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save spec as YAML
	specYAML, err := yaml.Marshal(productSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %w", err)
	}
	specPath := filepath.Join(o.config.OutputDir, "spec.yaml")
	if err := os.WriteFile(specPath, specYAML, 0o644); err != nil {
		return fmt.Errorf("failed to write spec file: %w", err)
	}

	// Save spec lock as JSON
	lockJSON, err := json.MarshalIndent(specLock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal spec lock: %w", err)
	}
	lockPath := filepath.Join(o.config.OutputDir, "spec.lock.json")
	if err := os.WriteFile(lockPath, lockJSON, 0o644); err != nil {
		return fmt.Errorf("failed to write spec lock file: %w", err)
	}

	// Save plan as JSON
	planJSON, err := json.MarshalIndent(execPlan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}
	planPath := filepath.Join(o.config.OutputDir, "plan.json")
	if err := os.WriteFile(planPath, planJSON, 0o644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	fmt.Printf("📁 Saved output files to: %s\n", o.config.OutputDir)
	fmt.Printf("   - spec.yaml\n")
	fmt.Printf("   - spec.lock.json\n")
	fmt.Printf("   - plan.json\n\n")

	return nil
}
