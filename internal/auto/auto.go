package auto

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
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

	// Step 3: Generate execution plan
	fmt.Println("📋 Generating execution plan...")
	execPlan, err := o.generatePlan(ctx, productSpec, specLock)
	if err != nil {
		return nil, fmt.Errorf("generate plan: %w", err)
	}
	result.Plan = execPlan
	fmt.Printf("✅ Plan created: %d tasks\n\n", len(execPlan.Tasks))

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
