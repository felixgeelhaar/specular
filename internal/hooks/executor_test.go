package hooks

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// SlowHook simulates a slow hook execution
type SlowHook struct {
	name       string
	eventTypes []EventType
	enabled    bool
	duration   time.Duration
	executed   *atomic.Bool
}

func NewSlowHook(name string, duration time.Duration) *SlowHook {
	return &SlowHook{
		name:       name,
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    true,
		duration:   duration,
		executed:   &atomic.Bool{},
	}
}

func (h *SlowHook) Name() string            { return h.name }
func (h *SlowHook) EventTypes() []EventType { return h.eventTypes }
func (h *SlowHook) Enabled() bool           { return h.enabled }
func (h *SlowHook) Execute(ctx context.Context, event *Event) error {
	h.executed.Store(true)
	select {
	case <-time.After(h.duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *SlowHook) WasExecuted() bool {
	return h.executed.Load()
}

// FailingHook always fails
type FailingHook struct {
	name       string
	eventTypes []EventType
	enabled    bool
}

func (h *FailingHook) Name() string            { return h.name }
func (h *FailingHook) EventTypes() []EventType { return h.eventTypes }
func (h *FailingHook) Enabled() bool           { return h.enabled }
func (h *FailingHook) Execute(ctx context.Context, event *Event) error {
	return fmt.Errorf("hook failed")
}

func TestExecutorExecute(t *testing.T) {
	executor := NewExecutor()

	hook := &MockHook{
		name:       "test-hook",
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    true,
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)
	result := executor.Execute(context.Background(), hook, event)

	if !result.Success {
		t.Errorf("Execution should succeed, got error: %s", result.Error)
	}

	if result.HookName != "test-hook" {
		t.Errorf("HookName mismatch: got %s, want test-hook", result.HookName)
	}

	if result.EventType != EventWorkflowStart {
		t.Errorf("EventType mismatch: got %s, want %s", result.EventType, EventWorkflowStart)
	}

	if result.Duration < 0 {
		t.Error("Duration should be >= 0")
	}

	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestExecutorExecuteFailure(t *testing.T) {
	executor := NewExecutor()

	hook := &FailingHook{
		name:       "failing-hook",
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    true,
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)
	result := executor.Execute(context.Background(), hook, event)

	if result.Success {
		t.Error("Execution should fail")
	}

	if result.Error == "" {
		t.Error("Error message should not be empty")
	}

	if result.Error != "hook failed" {
		t.Errorf("Error message mismatch: got %s, want 'hook failed'", result.Error)
	}
}

func TestExecutorExecuteTimeout(t *testing.T) {
	executor := NewExecutor()
	executor.SetDefaultTimeout(100 * time.Millisecond)

	hook := NewSlowHook("slow-hook", 500*time.Millisecond)

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)
	result := executor.Execute(context.Background(), hook, event)

	if result.Success {
		t.Error("Execution should fail due to timeout")
	}

	if result.Error == "" {
		t.Error("Error message should not be empty")
	}

	// Check that hook was executed (started)
	if !hook.WasExecuted() {
		t.Error("Hook should have been executed")
	}
}

func TestExecutorExecuteAll(t *testing.T) {
	executor := NewExecutor()

	hooks := []Hook{
		&MockHook{name: "hook-1", eventTypes: []EventType{EventWorkflowStart}, enabled: true},
		&MockHook{name: "hook-2", eventTypes: []EventType{EventWorkflowStart}, enabled: true},
		&MockHook{name: "hook-3", eventTypes: []EventType{EventWorkflowStart}, enabled: true},
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)
	results := executor.ExecuteAll(context.Background(), hooks, event)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Check all hooks executed successfully
	for i, result := range results {
		if !result.Success {
			t.Errorf("Hook %d should succeed, got error: %s", i, result.Error)
		}
	}
}

func TestExecutorExecuteAllEmpty(t *testing.T) {
	executor := NewExecutor()

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)
	results := executor.ExecuteAll(context.Background(), []Hook{}, event)

	if results != nil {
		t.Errorf("Expected nil results for empty hooks, got %d results", len(results))
	}
}

func TestExecutorExecuteAllMixed(t *testing.T) {
	executor := NewExecutor()

	hooks := []Hook{
		&MockHook{name: "success-1", eventTypes: []EventType{EventWorkflowStart}, enabled: true},
		&FailingHook{name: "failure", eventTypes: []EventType{EventWorkflowStart}, enabled: true},
		&MockHook{name: "success-2", eventTypes: []EventType{EventWorkflowStart}, enabled: true},
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)
	results := executor.ExecuteAll(context.Background(), hooks, event)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Check results
	successCount := 0
	failureCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	if successCount != 2 {
		t.Errorf("Expected 2 successful hooks, got %d", successCount)
	}

	if failureCount != 1 {
		t.Errorf("Expected 1 failed hook, got %d", failureCount)
	}
}

func TestExecutorConcurrency(t *testing.T) {
	executor := NewExecutor()
	executor.SetMaxConcurrency(3)

	// Create 10 slow hooks
	hooks := make([]Hook, 10)
	for i := 0; i < 10; i++ {
		hooks[i] = NewSlowHook(fmt.Sprintf("hook-%d", i), 50*time.Millisecond)
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)
	start := time.Now()
	results := executor.ExecuteAll(context.Background(), hooks, event)
	duration := time.Since(start)

	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}

	// With max concurrency of 3, 10 hooks should take roughly 200ms (10/3 = 4 batches, 4*50ms)
	// Allow some overhead
	if duration < 150*time.Millisecond {
		t.Errorf("Execution too fast, concurrency limit may not be working: %v", duration)
	}

	if duration > 300*time.Millisecond {
		t.Errorf("Execution too slow: %v", duration)
	}

	// All hooks should succeed
	for _, result := range results {
		if !result.Success {
			t.Errorf("Hook %s failed: %s", result.HookName, result.Error)
		}
	}
}

func TestExecutorExecuteAsync(t *testing.T) {
	executor := NewExecutor()

	hook := &MockHook{
		name:       "test-hook",
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    true,
	}

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)
	resultChan := executor.ExecuteAsync(context.Background(), hook, event)

	// Should receive result
	select {
	case result := <-resultChan:
		if !result.Success {
			t.Errorf("Execution should succeed, got error: %s", result.Error)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for result")
	}

	// Channel should be closed
	select {
	case _, ok := <-resultChan:
		if ok {
			t.Error("Channel should be closed")
		}
	default:
		t.Error("Channel should be closed")
	}
}

func TestExecutorSetMaxConcurrency(t *testing.T) {
	executor := NewExecutor()

	executor.SetMaxConcurrency(5)
	if executor.maxConcurrency != 5 {
		t.Errorf("MaxConcurrency mismatch: got %d, want 5", executor.maxConcurrency)
	}

	// Test minimum value
	executor.SetMaxConcurrency(0)
	if executor.maxConcurrency != 1 {
		t.Errorf("MaxConcurrency should be at least 1, got %d", executor.maxConcurrency)
	}

	executor.SetMaxConcurrency(-5)
	if executor.maxConcurrency != 1 {
		t.Errorf("MaxConcurrency should be at least 1, got %d", executor.maxConcurrency)
	}
}

func TestExecutorSetDefaultTimeout(t *testing.T) {
	executor := NewExecutor()

	executor.SetDefaultTimeout(5 * time.Second)
	if executor.defaultTimeout != 5*time.Second {
		t.Errorf("DefaultTimeout mismatch: got %v, want 5s", executor.defaultTimeout)
	}

	// Test invalid value
	executor.SetDefaultTimeout(0)
	if executor.defaultTimeout != DefaultTimeout {
		t.Errorf("DefaultTimeout should reset to default, got %v", executor.defaultTimeout)
	}

	executor.SetDefaultTimeout(-1 * time.Second)
	if executor.defaultTimeout != DefaultTimeout {
		t.Errorf("DefaultTimeout should reset to default, got %v", executor.defaultTimeout)
	}
}

func TestHandleResults(t *testing.T) {
	tests := []struct {
		name          string
		results       []ExecutionResult
		failureMode   string
		shouldError   bool
		expectedError string
	}{
		{
			name:        "no results",
			results:     []ExecutionResult{},
			failureMode: "fail",
			shouldError: false,
		},
		{
			name: "all success",
			results: []ExecutionResult{
				{HookName: "hook-1", Success: true},
				{HookName: "hook-2", Success: true},
			},
			failureMode: "fail",
			shouldError: false,
		},
		{
			name: "failure with ignore mode",
			results: []ExecutionResult{
				{HookName: "hook-1", Success: false, Error: "failed"},
			},
			failureMode: "ignore",
			shouldError: false,
		},
		{
			name: "failure with warn mode",
			results: []ExecutionResult{
				{HookName: "hook-1", Success: false, Error: "failed"},
			},
			failureMode: "warn",
			shouldError: false,
		},
		{
			name: "failure with fail mode",
			results: []ExecutionResult{
				{HookName: "hook-1", Success: false, Error: "failed"},
			},
			failureMode:   "fail",
			shouldError:   true,
			expectedError: "hook hook-1 failed: failed",
		},
		{
			name: "multiple failures with fail mode",
			results: []ExecutionResult{
				{HookName: "hook-1", Success: false, Error: "error1"},
				{HookName: "hook-2", Success: false, Error: "error2"},
			},
			failureMode:   "fail",
			shouldError:   true,
			expectedError: "hook hook-1 failed: error1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HandleResults(tt.results, tt.failureMode, nil)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.shouldError && err != nil && tt.expectedError != "" {
				if err.Error() != tt.expectedError {
					t.Errorf("Error message mismatch: got %s, want %s", err.Error(), tt.expectedError)
				}
			}
		})
	}
}
