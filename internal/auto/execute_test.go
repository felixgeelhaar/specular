package auto

import (
	"fmt"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/domain"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// Test helper functions

func createTestProductSpec() *spec.ProductSpec {
	return &spec.ProductSpec{
		Product: "Test Product",
		Goals:   []string{"Build a test product"},
		Features: []spec.Feature{
			{
				ID:       domain.FeatureID("test-feature"),
				Title:    "Test Feature",
				Desc:     "Test feature description",
				Priority: "P0",
				Success:  []string{"Feature works correctly"},
				Trace:    []string{"Implement feature"},
			},
		},
	}
}

func createTestPlan() *plan.Plan {
	return &plan.Plan{
		Tasks: []plan.Task{
			{
				ID:        domain.TaskID("test-task-1"),
				FeatureID: domain.FeatureID("test-feature"),
				Skill:     "test-skill",
				Priority:  "P0",
			},
		},
	}
}

// TestResult_Structure tests the Result struct initialization
func TestResult_Structure(t *testing.T) {
	result := &Result{
		Success:  true,
		Errors:   []error{},
		Duration: 5 * time.Second,
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if len(result.Errors) != 0 {
		t.Error("Expected empty Errors slice")
	}

	if result.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want %v", result.Duration, 5*time.Second)
	}
}

// TestResult_WithSpec tests Result with spec attached
func TestResult_WithSpec(t *testing.T) {
	productSpec := createTestProductSpec()
	result := &Result{
		Success: true,
		Spec:    productSpec,
	}

	if result.Spec == nil {
		t.Fatal("Expected Spec to be set")
	}

	if result.Spec.Product != "Test Product" {
		t.Errorf("Spec.Product = %s, want Test Product", result.Spec.Product)
	}
}

// TestResult_WithPlan tests Result with plan attached
func TestResult_WithPlan(t *testing.T) {
	execPlan := createTestPlan()
	result := &Result{
		Success: true,
		Plan:    execPlan,
	}

	if result.Plan == nil {
		t.Fatal("Expected Plan to be set")
	}

	if len(result.Plan.Tasks) != 1 {
		t.Errorf("Plan.Tasks length = %d, want 1", len(result.Plan.Tasks))
	}
}

// TestResult_WithActionPlan tests Result with action plan attached
func TestResult_WithActionPlan(t *testing.T) {
	actionPlan := CreateDefaultActionPlan("Test goal", "default")
	result := &Result{
		Success:    true,
		ActionPlan: actionPlan,
	}

	if result.ActionPlan == nil {
		t.Fatal("Expected ActionPlan to be set")
	}

	if len(result.ActionPlan.Steps) != 4 {
		t.Errorf("ActionPlan.Steps length = %d, want 4", len(result.ActionPlan.Steps))
	}
}

// TestResult_WithAutoOutput tests Result with JSON output
func TestResult_WithAutoOutput(t *testing.T) {
	autoOutput := NewAutoOutput("Test goal", "default")
	result := &Result{
		Success:    true,
		AutoOutput: autoOutput,
	}

	if result.AutoOutput == nil {
		t.Fatal("Expected AutoOutput to be set")
	}

	if result.AutoOutput.Goal != "Test goal" {
		t.Errorf("AutoOutput.Goal = %s, want Test goal", result.AutoOutput.Goal)
	}

	if result.AutoOutput.Status != "in_progress" {
		t.Errorf("AutoOutput.Status = %s, want in_progress", result.AutoOutput.Status)
	}
}

// TestResult_WithErrors tests Result with errors
func TestResult_WithErrors(t *testing.T) {
	err1 := fmt.Errorf("first error")
	err2 := fmt.Errorf("second error")

	result := &Result{
		Success: false,
		Errors:  []error{err1, err2},
	}

	if result.Success {
		t.Error("Expected Success to be false")
	}

	if len(result.Errors) != 2 {
		t.Errorf("Errors length = %d, want 2", len(result.Errors))
	}

	if result.Errors[0].Error() != "first error" {
		t.Errorf("First error = %s, want 'first error'", result.Errors[0].Error())
	}
}

// TestConfig_DryRunMode tests DryRun configuration
func TestConfig_DryRunMode(t *testing.T) {
	config := DefaultConfig()
	config.DryRun = true
	config.Goal = "Test goal"

	if !config.DryRun {
		t.Error("Expected DryRun to be true")
	}

	// Verify dry run doesn't conflict with other settings
	config.RequireApproval = true
	if !config.DryRun {
		t.Error("DryRun should remain true even with RequireApproval")
	}
}

// TestConfig_DryRunWithOutput tests DryRun with output directory
func TestConfig_DryRunWithOutput(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.DryRun = true
	config.OutputDir = tmpDir
	config.Goal = "Test goal"

	// Verify configuration is valid
	if !config.DryRun {
		t.Error("Expected DryRun to be true")
	}

	if config.OutputDir != tmpDir {
		t.Errorf("OutputDir = %s, want %s", config.OutputDir, tmpDir)
	}
}

// TestConfig_DryRunWithJSONOutput tests DryRun with JSON output
func TestConfig_DryRunWithJSONOutput(t *testing.T) {
	config := DefaultConfig()
	config.DryRun = true
	config.JSONOutput = true
	config.Goal = "Test goal"

	if !config.DryRun {
		t.Error("Expected DryRun to be true")
	}

	if !config.JSONOutput {
		t.Error("Expected JSONOutput to be true")
	}
}

// TestConfig_DryRunWithScope tests DryRun with scope filtering
func TestConfig_DryRunWithScope(t *testing.T) {
	config := DefaultConfig()
	config.DryRun = true
	config.ScopePatterns = []string{"auth*", "user-*"}
	config.IncludeDependencies = true
	config.Goal = "Test goal"

	if !config.DryRun {
		t.Error("Expected DryRun to be true")
	}

	if len(config.ScopePatterns) != 2 {
		t.Errorf("ScopePatterns length = %d, want 2", len(config.ScopePatterns))
	}

	if !config.IncludeDependencies {
		t.Error("Expected IncludeDependencies to be true")
	}
}

// TestAutoOutput_SetCompleted tests AutoOutput completion status
func TestAutoOutput_SetCompleted(t *testing.T) {
	autoOutput := NewAutoOutput("Test goal", "default")

	// Initially in_progress
	if autoOutput.Status != "in_progress" {
		t.Errorf("Initial status = %s, want in_progress", autoOutput.Status)
	}

	// Set to completed
	autoOutput.SetCompleted()

	if autoOutput.Status != "completed" {
		t.Errorf("Status after SetCompleted = %s, want completed", autoOutput.Status)
	}
}

// TestAutoOutput_SetPartial tests AutoOutput partial status
func TestAutoOutput_SetPartial(t *testing.T) {
	autoOutput := NewAutoOutput("Test goal", "default")

	// Set to partial
	autoOutput.SetPartial()

	if autoOutput.Status != "partial" {
		t.Errorf("Status after SetPartial = %s, want partial", autoOutput.Status)
	}
}

// TestAutoOutput_AddStepResult tests adding step results
func TestAutoOutput_AddStepResult(t *testing.T) {
	autoOutput := NewAutoOutput("Test goal", "default")

	stepResult := StepResult{
		ID:          "step-1",
		Type:        "spec:update",
		Status:      "completed",
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
		Duration:    1 * time.Second,
		CostUSD:     0.05,
	}

	autoOutput.AddStepResult(stepResult)

	if len(autoOutput.Steps) != 1 {
		t.Errorf("Steps length = %d, want 1", len(autoOutput.Steps))
	}

	if autoOutput.Steps[0].ID != "step-1" {
		t.Errorf("Step ID = %s, want step-1", autoOutput.Steps[0].ID)
	}

	if autoOutput.Steps[0].Type != "spec:update" {
		t.Errorf("Step Type = %s, want spec:update", autoOutput.Steps[0].Type)
	}
}
