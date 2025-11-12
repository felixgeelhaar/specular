package auto

import (
	"testing"
	"time"
)

// TestPolicyContext_ElapsedTime tests the ElapsedTime method
func TestPolicyContext_ElapsedTime(t *testing.T) {
	// Create a policy context with a known start time
	startTime := time.Now().Add(-5 * time.Second)
	ctx := &PolicyContext{
		ExecutionStartTime: startTime,
	}

	elapsed := ctx.ElapsedTime()

	// Should be approximately 5 seconds (allow some tolerance for test execution time)
	if elapsed < 4*time.Second || elapsed > 6*time.Second {
		t.Errorf("ElapsedTime() = %v, expected approximately 5 seconds", elapsed)
	}
}

// TestPolicyContext_ElapsedTime_Zero tests ElapsedTime with zero start time
func TestPolicyContext_ElapsedTime_Zero(t *testing.T) {
	ctx := &PolicyContext{
		ExecutionStartTime: time.Time{}, // Zero time
	}

	elapsed := ctx.ElapsedTime()

	// With zero time, elapsed should be very large (time since epoch)
	// This tests the method works even with uninitialized time
	if elapsed <= 0 {
		t.Errorf("ElapsedTime() with zero time = %v, expected positive duration", elapsed)
	}
}

// TestPolicyContext_ElapsedTime_Recent tests ElapsedTime with recent start
func TestPolicyContext_ElapsedTime_Recent(t *testing.T) {
	// Create a policy context with current time (just started)
	ctx := &PolicyContext{
		ExecutionStartTime: time.Now(),
	}

	elapsed := ctx.ElapsedTime()

	// Should be very small (less than 1 second for test execution)
	if elapsed > time.Second {
		t.Errorf("ElapsedTime() = %v, expected less than 1 second for recent start", elapsed)
	}
}

// TestPolicyContext_RemainingSteps tests the RemainingSteps method
func TestPolicyContext_RemainingSteps(t *testing.T) {
	// Create an action plan with 4 steps
	plan := CreateDefaultActionPlan("Test goal", "default")

	ctx := &PolicyContext{
		Plan:           plan,
		CompletedSteps: 2,
		FailedSteps:    0,
	}

	remaining := ctx.RemainingSteps()

	// Total steps: 4
	// Completed: 2
	// Failed: 0
	// Current: 1 (being evaluated)
	// Remaining: 4 - 2 - 0 - 1 = 1
	expected := 1
	if remaining != expected {
		t.Errorf("RemainingSteps() = %d, expected %d", remaining, expected)
	}
}

// TestPolicyContext_RemainingSteps_NoPlan tests RemainingSteps with nil plan
func TestPolicyContext_RemainingSteps_NoPlan(t *testing.T) {
	ctx := &PolicyContext{
		Plan:           nil,
		CompletedSteps: 2,
		FailedSteps:    0,
	}

	remaining := ctx.RemainingSteps()

	// With nil plan, should return 0
	if remaining != 0 {
		t.Errorf("RemainingSteps() with nil plan = %d, expected 0", remaining)
	}
}

// TestPolicyContext_RemainingSteps_AllCompleted tests RemainingSteps when all done
func TestPolicyContext_RemainingSteps_AllCompleted(t *testing.T) {
	plan := CreateDefaultActionPlan("Test goal", "default")

	ctx := &PolicyContext{
		Plan:           plan,
		CompletedSteps: 3, // 3 steps completed
		FailedSteps:    0,
	}

	remaining := ctx.RemainingSteps()

	// Total steps: 4
	// Completed: 3
	// Failed: 0
	// Current: 1 (the 4th step being evaluated)
	// Remaining: 4 - 3 - 0 - 1 = 0
	expected := 0
	if remaining != expected {
		t.Errorf("RemainingSteps() = %d, expected %d", remaining, expected)
	}
}

// TestPolicyContext_RemainingSteps_WithFailures tests RemainingSteps with some failures
func TestPolicyContext_RemainingSteps_WithFailures(t *testing.T) {
	plan := CreateDefaultActionPlan("Test goal", "default")

	ctx := &PolicyContext{
		Plan:           plan,
		CompletedSteps: 1,
		FailedSteps:    1,
	}

	remaining := ctx.RemainingSteps()

	// Total steps: 4
	// Completed: 1
	// Failed: 1
	// Current: 1 (being evaluated)
	// Remaining: 4 - 1 - 1 - 1 = 1
	expected := 1
	if remaining != expected {
		t.Errorf("RemainingSteps() = %d, expected %d", remaining, expected)
	}
}

// TestPolicyContext_RemainingSteps_JustStarted tests RemainingSteps at beginning
func TestPolicyContext_RemainingSteps_JustStarted(t *testing.T) {
	plan := CreateDefaultActionPlan("Test goal", "default")

	ctx := &PolicyContext{
		Plan:           plan,
		CompletedSteps: 0,
		FailedSteps:    0,
	}

	remaining := ctx.RemainingSteps()

	// Total steps: 4
	// Completed: 0
	// Failed: 0
	// Current: 1 (first step being evaluated)
	// Remaining: 4 - 0 - 0 - 1 = 3
	expected := 3
	if remaining != expected {
		t.Errorf("RemainingSteps() = %d, expected %d", remaining, expected)
	}
}

// TestNewPolicyContext tests the NewPolicyContext constructor
func TestNewPolicyContext(t *testing.T) {
	plan := CreateDefaultActionPlan("Test goal", "default")
	step, _ := plan.GetStep("step-1")
	stepIndex := 0

	ctx := NewPolicyContext(step, plan, stepIndex)

	if ctx == nil {
		t.Fatal("NewPolicyContext returned nil")
	}

	if ctx.CurrentStep != step {
		t.Error("CurrentStep was not set correctly")
	}

	if ctx.Plan != plan {
		t.Error("Plan was not set correctly")
	}

	if ctx.StepIndex != stepIndex {
		t.Errorf("StepIndex = %d, expected %d", ctx.StepIndex, stepIndex)
	}

	// Check defaults
	if ctx.TotalCostSoFar != 0 {
		t.Errorf("TotalCostSoFar = %f, expected 0", ctx.TotalCostSoFar)
	}

	if ctx.CompletedSteps != 0 {
		t.Errorf("CompletedSteps = %d, expected 0", ctx.CompletedSteps)
	}

	if ctx.FailedSteps != 0 {
		t.Errorf("FailedSteps = %d, expected 0", ctx.FailedSteps)
	}

	// ExecutionStartTime should be set to approximately now
	if time.Since(ctx.ExecutionStartTime) > time.Second {
		t.Error("ExecutionStartTime was not set to current time")
	}
}

// TestNewPolicyContext_MultipleSteps tests NewPolicyContext with different steps
func TestNewPolicyContext_MultipleSteps(t *testing.T) {
	plan := CreateDefaultActionPlan("Test goal", "default")

	testCases := []struct {
		stepID    string
		stepIndex int
	}{
		{"step-1", 0},
		{"step-2", 1},
		{"step-3", 2},
		{"step-4", 3},
	}

	for _, tc := range testCases {
		t.Run(tc.stepID, func(t *testing.T) {
			step, _ := plan.GetStep(tc.stepID)
			ctx := NewPolicyContext(step, plan, tc.stepIndex)

			if ctx.StepIndex != tc.stepIndex {
				t.Errorf("StepIndex = %d, expected %d", ctx.StepIndex, tc.stepIndex)
			}

			if ctx.CurrentStep.ID != tc.stepID {
				t.Errorf("CurrentStep.ID = %s, expected %s", ctx.CurrentStep.ID, tc.stepID)
			}
		})
	}
}
