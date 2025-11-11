package auto

import (
	"context"
	"testing"
	"time"
)

// mockPolicyChecker is a test implementation of PolicyChecker
type mockPolicyChecker struct {
	name           string
	checkFunc      func(ctx context.Context, step *ActionStep) (*PolicyResult, error)
	checkCallCount int
}

func (m *mockPolicyChecker) CheckStep(ctx context.Context, step *ActionStep) (*PolicyResult, error) {
	m.checkCallCount++
	if m.checkFunc != nil {
		return m.checkFunc(ctx, step)
	}
	return &PolicyResult{
		Allowed:  true,
		Warnings: []string{},
		Metadata: make(map[string]interface{}),
	}, nil
}

func (m *mockPolicyChecker) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

// Test that max steps policy check is called before each step
func TestMaxStepsPolicy_CallsBeforeEachStep(t *testing.T) {
	mockChecker := &mockPolicyChecker{
		name: "max_steps",
		checkFunc: func(ctx context.Context, step *ActionStep) (*PolicyResult, error) {
			// Allow all steps
			return &PolicyResult{
				Allowed:  true,
				Warnings: []string{},
				Metadata: make(map[string]interface{}),
			}, nil
		},
	}

	// Create a simple orchestrator setup (without actual execution)
	config := DefaultConfig()
	config.Goal = "Test goal"
	config.DryRun = true // Use dry run to avoid actual execution

	o := &Orchestrator{
		config:        config,
		policyChecker: mockChecker,
	}

	// Create action plan
	o.actionPlan = CreateDefaultActionPlan("Test goal", "default")

	// Verify that the policy checker would be called
	// This is a simple structural test
	if o.policyChecker == nil {
		t.Fatal("Expected policy checker to be set")
	}

	if o.policyChecker.Name() != "max_steps" {
		t.Errorf("Expected checker name 'max_steps', got '%s'", o.policyChecker.Name())
	}
}

// Test that policy denial stops execution
func TestMaxStepsPolicy_DenialStopsExecution(t *testing.T) {
	denialReason := "maximum step count exceeded: 3 > 2 limit"
	stepsDenied := 0

	mockChecker := &mockPolicyChecker{
		name: "max_steps",
		checkFunc: func(ctx context.Context, step *ActionStep) (*PolicyResult, error) {
			// Get policy context from context
			policyCtx, ok := ctx.Value("policy_context").(*PolicyContext)
			if !ok {
				return &PolicyResult{Allowed: true}, nil
			}

			// Deny on step 3 (completed steps = 2, trying to execute step 3)
			if policyCtx.CompletedSteps >= 2 {
				stepsDenied++
				return &PolicyResult{
					Allowed: false,
					Reason:  denialReason,
				}, nil
			}

			return &PolicyResult{
				Allowed: true,
			}, nil
		},
	}

	// Test that checkPolicy returns denial correctly
	o := &Orchestrator{
		policyChecker: mockChecker,
	}

	// Create a mock action plan
	actionPlan := CreateDefaultActionPlan("Test goal", "default")
	o.actionPlan = actionPlan

	step3, _ := actionPlan.GetStep("step-3")

	// Simulate checking step 3 after completing 2 steps
	allowed, policyEvent, err := o.checkPolicy(
		context.Background(),
		mockChecker,
		step3,
		2,          // stepIndex
		2,          // completedSteps
		0.05,       // totalCost
		time.Now(), // executionStart
	)

	if err != nil {
		t.Fatalf("checkPolicy returned error: %v", err)
	}

	if allowed {
		t.Error("Expected policy to deny step 3, but it was allowed")
	}

	if policyEvent == nil {
		t.Fatal("Expected policy event to be returned")
	}

	if policyEvent.Allowed {
		t.Error("Expected policy event to show denied")
	}

	if policyEvent.Reason != denialReason {
		t.Errorf("Expected denial reason '%s', got '%s'", denialReason, policyEvent.Reason)
	}

	if stepsDenied != 1 {
		t.Errorf("Expected 1 step to be denied, got %d", stepsDenied)
	}
}

// Test that policy context is updated correctly
func TestMaxStepsPolicy_PolicyContextUpdates(t *testing.T) {
	var capturedContexts []*PolicyContext

	mockChecker := &mockPolicyChecker{
		name: "max_steps",
		checkFunc: func(ctx context.Context, step *ActionStep) (*PolicyResult, error) {
			policyCtx, ok := ctx.Value("policy_context").(*PolicyContext)
			if ok {
				// Capture a copy
				captured := *policyCtx
				capturedContexts = append(capturedContexts, &captured)
			}
			return &PolicyResult{Allowed: true}, nil
		},
	}

	o := &Orchestrator{
		policyChecker: mockChecker,
	}

	actionPlan := CreateDefaultActionPlan("Test goal", "default")
	o.actionPlan = actionPlan

	executionStart := time.Now()

	// Simulate checking steps with increasing completed counts
	for i := 0; i < 4; i++ {
		stepID := ""
		switch i {
		case 0:
			stepID = "step-1"
		case 1:
			stepID = "step-2"
		case 2:
			stepID = "step-3"
		case 3:
			stepID = "step-4"
		}

		step, _ := actionPlan.GetStep(stepID)
		_, _, err := o.checkPolicy(
			context.Background(),
			mockChecker,
			step,
			i,               // stepIndex
			i,               // completedSteps
			float64(i)*0.01, // totalCost
			executionStart,
		)

		if err != nil {
			t.Fatalf("checkPolicy for step %d returned error: %v", i+1, err)
		}
	}

	// Verify we captured 4 contexts
	if len(capturedContexts) != 4 {
		t.Fatalf("Expected 4 captured contexts, got %d", len(capturedContexts))
	}

	// Verify completed steps progression
	for i, ctx := range capturedContexts {
		if ctx.CompletedSteps != i {
			t.Errorf("Step %d: expected %d completed steps, got %d", i+1, i, ctx.CompletedSteps)
		}
	}

	// Verify cost accumulation
	for i, ctx := range capturedContexts {
		expectedCost := float64(i) * 0.01
		if ctx.TotalCostSoFar != expectedCost {
			t.Errorf("Step %d: expected cost %.4f, got %.4f", i+1, expectedCost, ctx.TotalCostSoFar)
		}
	}

	// Verify execution start time is consistent
	for i, ctx := range capturedContexts {
		if !ctx.ExecutionStartTime.Equal(executionStart) {
			t.Errorf("Step %d: execution start time mismatch", i+1)
		}
	}
}

// Test that warnings are collected and reported
func TestMaxStepsPolicy_WarningsReported(t *testing.T) {
	warnings := []string{
		"Approaching step limit: 2 steps remaining of 4 maximum",
		"Approaching step limit: 1 steps remaining of 4 maximum",
	}
	warningIndex := 0

	mockChecker := &mockPolicyChecker{
		name: "max_steps",
		checkFunc: func(ctx context.Context, step *ActionStep) (*PolicyResult, error) {
			policyCtx, ok := ctx.Value("policy_context").(*PolicyContext)
			if !ok {
				return &PolicyResult{Allowed: true}, nil
			}

			result := &PolicyResult{
				Allowed:  true,
				Warnings: []string{},
				Metadata: make(map[string]interface{}),
			}

			// Add warnings when approaching limit (steps 2 and 3 out of 4)
			if policyCtx.CompletedSteps >= 2 && policyCtx.CompletedSteps < 4 {
				result.Warnings = append(result.Warnings, warnings[warningIndex])
				warningIndex++
			}

			return result, nil
		},
	}

	o := &Orchestrator{
		policyChecker: mockChecker,
	}

	actionPlan := CreateDefaultActionPlan("Test goal", "default")
	o.actionPlan = actionPlan

	// Check steps 3 and 4 (after completing 2 and 3 steps)
	for i := 2; i < 4; i++ {
		stepID := ""
		if i == 2 {
			stepID = "step-3"
		} else {
			stepID = "step-4"
		}

		step, _ := actionPlan.GetStep(stepID)
		allowed, policyEvent, err := o.checkPolicy(
			context.Background(),
			mockChecker,
			step,
			i,          // stepIndex
			i,          // completedSteps
			0.05,       // totalCost
			time.Now(), // executionStart
		)

		if err != nil {
			t.Fatalf("checkPolicy for step %d returned error: %v", i+1, err)
		}

		if !allowed {
			t.Errorf("Step %d should be allowed but was denied", i+1)
		}

		if policyEvent == nil {
			t.Fatalf("Expected policy event for step %d", i+1)
		}

		if len(policyEvent.Warnings) == 0 {
			t.Errorf("Expected warnings for step %d", i+1)
		}
	}
}

// Test that partial completion is tracked
func TestMaxStepsPolicy_PartialCompletion(t *testing.T) {
	// This test verifies the structure is in place for partial completion
	// Actual partial completion logic happens in Execute()

	config := DefaultConfig()
	config.Goal = "Test goal"
	config.JSONOutput = true

	autoOutput := NewAutoOutput("Test goal", "default")

	// Simulate a policy denial
	autoOutput.AddPolicy(PolicyEvent{
		StepID:      "step-3",
		Timestamp:   time.Now(),
		CheckerName: "max_steps",
		Allowed:     false,
		Reason:      "maximum step count exceeded: 3 > 2 limit",
		Metadata: map[string]interface{}{
			"completed_steps": 2,
			"max_steps":       2,
		},
	})

	// Mark as partial
	autoOutput.SetPartial()

	if autoOutput.Status != "partial" {
		t.Errorf("Expected status 'partial', got '%s'", autoOutput.Status)
	}

	if len(autoOutput.Audit.Policies) != 1 {
		t.Errorf("Expected 1 policy event, got %d", len(autoOutput.Audit.Policies))
	}

	policyEvent := autoOutput.Audit.Policies[0]
	if policyEvent.Allowed {
		t.Error("Expected policy event to show denied")
	}

	if policyEvent.Reason != "maximum step count exceeded: 3 > 2 limit" {
		t.Errorf("Unexpected denial reason: %s", policyEvent.Reason)
	}
}

// Test that no policy checker allows all steps
func TestMaxStepsPolicy_NoPolicyCheckerAllowsAll(t *testing.T) {
	o := &Orchestrator{
		policyChecker: nil, // No policy checker
	}

	actionPlan := CreateDefaultActionPlan("Test goal", "default")
	o.actionPlan = actionPlan

	step1, _ := actionPlan.GetStep("step-1")

	allowed, policyEvent, err := o.checkPolicy(
		context.Background(),
		nil, // Explicitly nil
		step1,
		0,
		0,
		0.0,
		time.Now(),
	)

	if err != nil {
		t.Fatalf("checkPolicy with nil checker returned error: %v", err)
	}

	if !allowed {
		t.Error("Expected nil policy checker to allow all steps")
	}

	if policyEvent != nil {
		t.Error("Expected nil policy event when no checker is provided")
	}
}
