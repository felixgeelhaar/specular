package autopolicy

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/auto"
)

func TestCostLimitChecker(t *testing.T) {
	t.Run("allows step within per-step limit", func(t *testing.T) {
		checker := NewCostLimitChecker(5.0, 1.0)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate, // Estimated $0.50
		}

		result, err := checker.CheckStep(context.Background(), step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step to be allowed, got denied: %s", result.Reason)
		}
	})

	t.Run("denies step exceeding per-step limit", func(t *testing.T) {
		checker := NewCostLimitChecker(5.0, 0.10)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate, // Estimated $0.50 > $0.10 limit
		}

		result, err := checker.CheckStep(context.Background(), step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected step to be denied, got allowed")
		}
	})

	t.Run("denies step exceeding total budget", func(t *testing.T) {
		checker := NewCostLimitChecker(1.0, 2.0)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeBuildRun, // Estimated $1.00
		}

		policyCtx := &PolicyContext{
			CurrentStep:    step,
			TotalCostSoFar: 0.50, // Already spent $0.50
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := checker.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected step to be denied, got allowed")
		}
	})

	t.Run("warns when approaching budget limit", func(t *testing.T) {
		checker := NewCostLimitChecker(1.0, 2.0)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate, // Estimated $0.50
		}

		policyCtx := &PolicyContext{
			CurrentStep:    step,
			TotalCostSoFar: 0.40, // $0.40 + $0.50 = $0.90, leaving $0.10 (< 20% threshold)
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := checker.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step to be allowed, got denied: %s", result.Reason)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected warning about approaching limit")
		}
	})
}

func TestTimeoutChecker(t *testing.T) {
	t.Run("allows step with sufficient time", func(t *testing.T) {
		checker := NewTimeoutChecker(30*time.Minute, 5*time.Minute)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}

		policyCtx := &PolicyContext{
			CurrentStep:        step,
			ExecutionStartTime: time.Now().Add(-5 * time.Minute), // Started 5 min ago
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := checker.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step to be allowed, got denied: %s", result.Reason)
		}
	})

	t.Run("denies step when timeout exceeded", func(t *testing.T) {
		checker := NewTimeoutChecker(10*time.Minute, 5*time.Minute)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}

		policyCtx := &PolicyContext{
			CurrentStep:        step,
			ExecutionStartTime: time.Now().Add(-15 * time.Minute), // Started 15 min ago
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := checker.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected step to be denied, got allowed")
		}
	})

	t.Run("warns when approaching timeout", func(t *testing.T) {
		checker := NewTimeoutChecker(10*time.Minute, 5*time.Minute)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}

		policyCtx := &PolicyContext{
			CurrentStep:        step,
			ExecutionStartTime: time.Now().Add(-9 * time.Minute), // 1 min remaining (< 20%)
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := checker.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step to be allowed, got denied: %s", result.Reason)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected warning about approaching timeout")
		}
	})
}

func TestStepTypeChecker(t *testing.T) {
	t.Run("allows all types when no restrictions", func(t *testing.T) {
		checker := NewStepTypeChecker(nil, nil)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeBuildRun,
		}

		result, err := checker.CheckStep(context.Background(), step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step to be allowed, got denied: %s", result.Reason)
		}
	})

	t.Run("denies blocked type", func(t *testing.T) {
		checker := NewStepTypeChecker(nil, []auto.StepType{auto.StepTypeBuildRun})
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeBuildRun,
		}

		result, err := checker.CheckStep(context.Background(), step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected step to be denied, got allowed")
		}
	})

	t.Run("allows only whitelisted types", func(t *testing.T) {
		checker := NewStepTypeChecker(
			[]auto.StepType{auto.StepTypeSpecUpdate, auto.StepTypePlanGen},
			nil,
		)

		// Allowed type
		step1 := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}
		result, err := checker.CheckStep(context.Background(), step1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected whitelisted step to be allowed, got denied: %s", result.Reason)
		}

		// Not allowed type
		step2 := &auto.ActionStep{
			ID:   "test-2",
			Type: auto.StepTypeBuildRun,
		}
		result, err = checker.CheckStep(context.Background(), step2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected non-whitelisted step to be denied, got allowed")
		}
	})

	t.Run("blacklist takes precedence over whitelist", func(t *testing.T) {
		checker := NewStepTypeChecker(
			[]auto.StepType{auto.StepTypeSpecUpdate, auto.StepTypeBuildRun},
			[]auto.StepType{auto.StepTypeBuildRun},
		)

		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeBuildRun,
		}
		result, err := checker.CheckStep(context.Background(), step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected blacklisted step to be denied, got allowed")
		}
	})
}

func TestMaxStepsChecker(t *testing.T) {
	t.Run("allows step within limit", func(t *testing.T) {
		checker := NewMaxStepsChecker(5)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}

		policyCtx := &PolicyContext{
			CurrentStep:    step,
			CompletedSteps: 2, // 2 completed + 1 current = 3 total
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := checker.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step to be allowed, got denied: %s", result.Reason)
		}
	})

	t.Run("denies step exceeding limit", func(t *testing.T) {
		checker := NewMaxStepsChecker(3)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}

		policyCtx := &PolicyContext{
			CurrentStep:    step,
			CompletedSteps: 3, // 3 completed + 1 current = 4 total > 3 limit
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := checker.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected step to be denied, got allowed")
		}
	})

	t.Run("warns when approaching limit", func(t *testing.T) {
		checker := NewMaxStepsChecker(5)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}

		policyCtx := &PolicyContext{
			CurrentStep:    step,
			CompletedSteps: 3, // 3 completed + 1 current = 4 total, 1 remaining
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := checker.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step to be allowed, got denied: %s", result.Reason)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected warning about approaching limit")
		}
	})
}

func TestMaxRetriesChecker(t *testing.T) {
	t.Run("allows step within retry limit", func(t *testing.T) {
		checker := NewMaxRetriesChecker(3)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}

		checker.RecordRetry(step.ID)
		checker.RecordRetry(step.ID) // 2 retries

		result, err := checker.CheckStep(context.Background(), step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step to be allowed, got denied: %s", result.Reason)
		}
	})

	t.Run("denies step exceeding retry limit", func(t *testing.T) {
		checker := NewMaxRetriesChecker(2)
		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}

		checker.RecordRetry(step.ID)
		checker.RecordRetry(step.ID) // 2 retries, at limit

		result, err := checker.CheckStep(context.Background(), step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected step to be denied, got allowed")
		}
	})

	t.Run("tracks retries per step", func(t *testing.T) {
		checker := NewMaxRetriesChecker(2)
		step1 := &auto.ActionStep{ID: "step-1", Type: auto.StepTypeSpecUpdate}
		step2 := &auto.ActionStep{ID: "step-2", Type: auto.StepTypePlanGen}

		checker.RecordRetry(step1.ID)
		checker.RecordRetry(step1.ID) // step-1 at limit

		// step-1 should be denied
		result, err := checker.CheckStep(context.Background(), step1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected step-1 to be denied")
		}

		// step-2 should be allowed (independent retry count)
		result, err = checker.CheckStep(context.Background(), step2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step-2 to be allowed, got denied: %s", result.Reason)
		}
	})
}

func TestCompositeChecker(t *testing.T) {
	t.Run("allows when all checkers pass", func(t *testing.T) {
		composite := NewCompositeChecker(
			NewCostLimitChecker(5.0, 2.0),
			NewStepTypeChecker(nil, nil),
			NewMaxStepsChecker(10),
		)

		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}

		policyCtx := &PolicyContext{
			CurrentStep:    step,
			TotalCostSoFar: 0.5,
			CompletedSteps: 2,
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := composite.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step to be allowed, got denied: %s", result.Reason)
		}
	})

	t.Run("denies when any checker fails", func(t *testing.T) {
		composite := NewCompositeChecker(
			NewCostLimitChecker(5.0, 2.0),
			NewStepTypeChecker(nil, []auto.StepType{auto.StepTypeBuildRun}), // Block build:run
			NewMaxStepsChecker(10),
		)

		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeBuildRun, // This is blocked
		}

		policyCtx := &PolicyContext{
			CurrentStep:    step,
			TotalCostSoFar: 0.5,
			CompletedSteps: 2,
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := composite.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected step to be denied, got allowed")
		}
	})

	t.Run("collects warnings from all checkers", func(t *testing.T) {
		composite := NewCompositeChecker(
			NewCostLimitChecker(1.0, 2.0),                    // Will warn about approaching limit
			NewTimeoutChecker(10*time.Minute, 5*time.Minute), // Will warn about timeout
		)

		step := &auto.ActionStep{
			ID:   "test-1",
			Type: auto.StepTypeSpecUpdate,
		}

		policyCtx := &PolicyContext{
			CurrentStep:        step,
			TotalCostSoFar:     0.40, // Close to $1.00 limit
			CompletedSteps:     2,
			ExecutionStartTime: time.Now().Add(-9 * time.Minute), // Close to 10 min timeout
		}
		ctx := context.WithValue(context.Background(), "policy_context", policyCtx)

		result, err := composite.CheckStep(ctx, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected step to be allowed, got denied: %s", result.Reason)
		}
		if len(result.Warnings) < 2 {
			t.Errorf("expected at least 2 warnings, got %d", len(result.Warnings))
		}
	})
}

func TestPolicyResult(t *testing.T) {
	t.Run("NewAllowedResult creates allowed result", func(t *testing.T) {
		result := NewAllowedResult()
		if !result.Allowed {
			t.Error("expected Allowed to be true")
		}
		if result.Reason != "" {
			t.Error("expected Reason to be empty")
		}
		if len(result.Warnings) != 0 {
			t.Error("expected no warnings")
		}
	})

	t.Run("NewDeniedResult creates denied result", func(t *testing.T) {
		result := NewDeniedResult("test reason")
		if result.Allowed {
			t.Error("expected Allowed to be false")
		}
		if result.Reason != "test reason" {
			t.Errorf("expected Reason to be 'test reason', got '%s'", result.Reason)
		}
	})

	t.Run("AddWarning adds warnings", func(t *testing.T) {
		result := NewAllowedResult()
		result.AddWarning("warning 1")
		result.AddWarning("warning 2")

		if len(result.Warnings) != 2 {
			t.Errorf("expected 2 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("SetMetadata sets metadata", func(t *testing.T) {
		result := NewAllowedResult()
		result.SetMetadata("key1", "value1")
		result.SetMetadata("key2", 42)

		if result.Metadata["key1"] != "value1" {
			t.Error("metadata key1 not set correctly")
		}
		if result.Metadata["key2"] != 42 {
			t.Error("metadata key2 not set correctly")
		}
	})
}

func TestPolicyContext(t *testing.T) {
	t.Run("NewPolicyContext creates context", func(t *testing.T) {
		step := &auto.ActionStep{ID: "test-1"}
		plan := &auto.ActionPlan{Steps: []auto.ActionStep{{}, {}, {}}}
		ctx := NewPolicyContext(step, plan, 1)

		if ctx.CurrentStep != step {
			t.Error("CurrentStep not set correctly")
		}
		if ctx.Plan != plan {
			t.Error("Plan not set correctly")
		}
		if ctx.StepIndex != 1 {
			t.Error("StepIndex not set correctly")
		}
	})

	t.Run("ElapsedTime calculates duration", func(t *testing.T) {
		ctx := &PolicyContext{
			ExecutionStartTime: time.Now().Add(-5 * time.Minute),
		}

		elapsed := ctx.ElapsedTime()
		if elapsed < 4*time.Minute || elapsed > 6*time.Minute {
			t.Errorf("expected elapsed time around 5 minutes, got %s", elapsed)
		}
	})

	t.Run("RemainingSteps calculates remaining", func(t *testing.T) {
		plan := &auto.ActionPlan{
			Steps: make([]auto.ActionStep, 5),
		}
		ctx := &PolicyContext{
			Plan:           plan,
			CompletedSteps: 2,
			FailedSteps:    1,
		}

		// 5 total - 2 completed - 1 failed - 1 current = 1 remaining
		remaining := ctx.RemainingSteps()
		if remaining != 1 {
			t.Errorf("expected 1 remaining step, got %d", remaining)
		}
	})
}
