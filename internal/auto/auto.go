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

	// Step 5: Execute plan (TODO: Phase 2)
	fmt.Println("🚀 Plan execution...")
	fmt.Println("⚠️  Task execution not yet implemented (Phase 2)")
	fmt.Println("    For now, auto mode generates spec, lock, and plan only.")

	result.Success = true
	result.Duration = time.Since(start)
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
