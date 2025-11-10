package auto

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/spec"
)

func TestNewTaskExecutor(t *testing.T) {
	pol := policy.DefaultPolicy()
	cfg := DefaultConfig()
	cfg.Goal = "Test goal"

	productSpec := &spec.ProductSpec{
		Product: "TestProduct",
	}

	executor := NewTaskExecutor(pol, cfg, productSpec)

	if executor == nil {
		t.Fatal("NewTaskExecutor returned nil")
	}
	if executor.policy != pol {
		t.Error("Executor policy was not set correctly")
	}
	if executor.config.Goal != "Test goal" {
		t.Errorf("Executor config.Goal = %s, want %s", executor.config.Goal, "Test goal")
	}
	if executor.spec != productSpec {
		t.Error("Executor spec was not set correctly")
	}
	if executor.progressFunc != nil {
		t.Error("Executor progressFunc should be nil by default")
	}
}

func TestNewTaskExecutor_NilPolicy(t *testing.T) {
	cfg := DefaultConfig()
	productSpec := &spec.ProductSpec{
		Product: "TestProduct",
	}

	executor := NewTaskExecutor(nil, cfg, productSpec)

	if executor == nil {
		t.Fatal("NewTaskExecutor returned nil")
	}
	if executor.policy != nil {
		t.Error("Executor policy should be nil when passed nil")
	}
}

func TestNewTaskExecutor_ConfigPreservation(t *testing.T) {
	pol := policy.DefaultPolicy()
	cfg := Config{
		Goal:             "Build a REST API",
		RequireApproval:  false,
		MaxCostUSD:       10.0,
		MaxCostPerTask:   2.0,
		MaxRetries:       5,
		RetryDelay:       time.Second * 5,
		TimeoutMinutes:   60,
		TaskTimeout:      time.Minute * 10,
		PolicyPath:       "custom/policy.yaml",
		FallbackToManual: false,
		Verbose:          true,
		DryRun:           true,
	}
	productSpec := &spec.ProductSpec{
		Product: "TestProduct",
	}

	executor := NewTaskExecutor(pol, cfg, productSpec)

	// Verify all config fields are preserved
	if executor.config.Goal != cfg.Goal {
		t.Errorf("Goal = %s, want %s", executor.config.Goal, cfg.Goal)
	}
	if executor.config.RequireApproval != cfg.RequireApproval {
		t.Errorf("RequireApproval = %v, want %v", executor.config.RequireApproval, cfg.RequireApproval)
	}
	if executor.config.MaxCostUSD != cfg.MaxCostUSD {
		t.Errorf("MaxCostUSD = %f, want %f", executor.config.MaxCostUSD, cfg.MaxCostUSD)
	}
	if executor.config.MaxCostPerTask != cfg.MaxCostPerTask {
		t.Errorf("MaxCostPerTask = %f, want %f", executor.config.MaxCostPerTask, cfg.MaxCostPerTask)
	}
	if executor.config.MaxRetries != cfg.MaxRetries {
		t.Errorf("MaxRetries = %d, want %d", executor.config.MaxRetries, cfg.MaxRetries)
	}
	if executor.config.RetryDelay != cfg.RetryDelay {
		t.Errorf("RetryDelay = %v, want %v", executor.config.RetryDelay, cfg.RetryDelay)
	}
	if executor.config.TimeoutMinutes != cfg.TimeoutMinutes {
		t.Errorf("TimeoutMinutes = %d, want %d", executor.config.TimeoutMinutes, cfg.TimeoutMinutes)
	}
	if executor.config.TaskTimeout != cfg.TaskTimeout {
		t.Errorf("TaskTimeout = %v, want %v", executor.config.TaskTimeout, cfg.TaskTimeout)
	}
	if executor.config.PolicyPath != cfg.PolicyPath {
		t.Errorf("PolicyPath = %s, want %s", executor.config.PolicyPath, cfg.PolicyPath)
	}
	if executor.config.FallbackToManual != cfg.FallbackToManual {
		t.Errorf("FallbackToManual = %v, want %v", executor.config.FallbackToManual, cfg.FallbackToManual)
	}
	if executor.config.Verbose != cfg.Verbose {
		t.Errorf("Verbose = %v, want %v", executor.config.Verbose, cfg.Verbose)
	}
	if executor.config.DryRun != cfg.DryRun {
		t.Errorf("DryRun = %v, want %v", executor.config.DryRun, cfg.DryRun)
	}
}

func TestSetProgressCallback(t *testing.T) {
	executor := NewTaskExecutor(nil, DefaultConfig(), &spec.ProductSpec{})

	if executor.progressFunc != nil {
		t.Error("Progress function should be nil initially")
	}

	// Create a test callback
	called := false
	var capturedTaskID, capturedStatus string
	var capturedErr error

	callback := func(taskID, status string, err error) {
		called = true
		capturedTaskID = taskID
		capturedStatus = status
		capturedErr = err
	}

	executor.SetProgressCallback(callback)

	if executor.progressFunc == nil {
		t.Fatal("Progress function was not set")
	}

	// Test the callback
	executor.progressFunc("task-001", "completed", nil)

	if !called {
		t.Error("Callback was not called")
	}
	if capturedTaskID != "task-001" {
		t.Errorf("Captured taskID = %s, want %s", capturedTaskID, "task-001")
	}
	if capturedStatus != "completed" {
		t.Errorf("Captured status = %s, want %s", capturedStatus, "completed")
	}
	if capturedErr != nil {
		t.Errorf("Captured error = %v, want nil", capturedErr)
	}
}

func TestSetProgressCallback_WithError(t *testing.T) {
	executor := NewTaskExecutor(nil, DefaultConfig(), &spec.ProductSpec{})

	var capturedErr error
	callback := func(taskID, status string, err error) {
		capturedErr = err
	}

	executor.SetProgressCallback(callback)

	// Test with error
	testErr := &testError{msg: "test error"}
	executor.progressFunc("task-001", "failed", testErr)

	if capturedErr == nil {
		t.Error("Error was not captured")
	}
	if capturedErr.Error() != "test error" {
		t.Errorf("Captured error = %v, want 'test error'", capturedErr)
	}
}

func TestSetProgressCallback_Multiple(t *testing.T) {
	executor := NewTaskExecutor(nil, DefaultConfig(), &spec.ProductSpec{})

	callCount := 0
	callback := func(taskID, status string, err error) {
		callCount++
	}

	executor.SetProgressCallback(callback)

	// Multiple calls
	executor.progressFunc("task-001", "pending", nil)
	executor.progressFunc("task-001", "in_progress", nil)
	executor.progressFunc("task-001", "completed", nil)

	if callCount != 3 {
		t.Errorf("Callback call count = %d, want 3", callCount)
	}
}

func TestExecutionStats_InitialState(t *testing.T) {
	stats := &ExecutionStats{
		TotalTasks: 5,
		StartTime:  time.Now(),
	}

	if stats.TotalTasks != 5 {
		t.Errorf("TotalTasks = %d, want 5", stats.TotalTasks)
	}
	if stats.Executed != 0 {
		t.Errorf("Executed = %d, want 0", stats.Executed)
	}
	if stats.Failed != 0 {
		t.Errorf("Failed = %d, want 0", stats.Failed)
	}
	if stats.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", stats.Skipped)
	}
	if stats.Success {
		t.Error("Success should be false initially")
	}
	if !stats.StartTime.IsZero() && stats.EndTime.IsZero() {
		// Expected: StartTime set, EndTime not set
	} else if stats.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
	if stats.Duration != 0 {
		t.Errorf("Duration = %v, want 0", stats.Duration)
	}
}

func TestExecutionStats_SuccessCase(t *testing.T) {
	start := time.Now()
	stats := &ExecutionStats{
		TotalTasks: 3,
		Executed:   3,
		Failed:     0,
		Skipped:    0,
		Success:    true,
		StartTime:  start,
		EndTime:    start.Add(time.Second * 10),
		Duration:   time.Second * 10,
	}

	if stats.TotalTasks != 3 {
		t.Errorf("TotalTasks = %d, want 3", stats.TotalTasks)
	}
	if stats.Executed != 3 {
		t.Errorf("Executed = %d, want 3", stats.Executed)
	}
	if stats.Failed != 0 {
		t.Errorf("Failed = %d, want 0", stats.Failed)
	}
	if !stats.Success {
		t.Error("Success should be true")
	}
	if stats.Duration != time.Second*10 {
		t.Errorf("Duration = %v, want 10s", stats.Duration)
	}
}

func TestExecutionStats_PartialFailure(t *testing.T) {
	stats := &ExecutionStats{
		TotalTasks: 5,
		Executed:   3,
		Failed:     2,
		Skipped:    0,
		Success:    false,
	}

	if stats.Executed+stats.Failed != stats.TotalTasks {
		t.Errorf("Executed(%d) + Failed(%d) != TotalTasks(%d)",
			stats.Executed, stats.Failed, stats.TotalTasks)
	}
	if stats.Success {
		t.Error("Success should be false when there are failures")
	}
}

func TestExecutionStats_WithSkipped(t *testing.T) {
	stats := &ExecutionStats{
		TotalTasks: 5,
		Executed:   2,
		Failed:     1,
		Skipped:    2,
		Success:    false,
	}

	if stats.Executed+stats.Failed+stats.Skipped != stats.TotalTasks {
		t.Errorf("Executed(%d) + Failed(%d) + Skipped(%d) != TotalTasks(%d)",
			stats.Executed, stats.Failed, stats.Skipped, stats.TotalTasks)
	}
	if stats.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", stats.Skipped)
	}
}

// Helper types for testing

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
