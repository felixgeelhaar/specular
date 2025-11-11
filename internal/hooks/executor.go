package hooks

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Executor executes hooks
type Executor struct {
	// maxConcurrency limits concurrent hook execution
	maxConcurrency int

	// defaultTimeout is used if hook doesn't specify one
	defaultTimeout time.Duration
}

// NewExecutor creates a new hook executor
func NewExecutor() *Executor {
	return &Executor{
		maxConcurrency: 10, // Allow up to 10 hooks to run concurrently
		defaultTimeout: DefaultTimeout,
	}
}

// ExecuteAll executes all hooks for an event
func (e *Executor) ExecuteAll(ctx context.Context, hooks []Hook, event *Event) []ExecutionResult {
	if len(hooks) == 0 {
		return nil
	}

	results := make([]ExecutionResult, len(hooks))
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrency
	sem := make(chan struct{}, e.maxConcurrency)

	for i, hook := range hooks {
		wg.Add(1)
		go func(index int, h Hook) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Execute hook with timeout
			results[index] = e.Execute(ctx, h, event)
		}(i, hook)
	}

	wg.Wait()
	return results
}

// Execute executes a single hook
func (e *Executor) Execute(ctx context.Context, hook Hook, event *Event) ExecutionResult {
	result := ExecutionResult{
		HookName:  hook.Name(),
		EventType: event.Type,
		Timestamp: time.Now(),
	}

	// Create context with timeout
	timeout := e.defaultTimeout
	hookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Measure execution time
	start := time.Now()

	// Execute hook
	err := hook.Execute(hookCtx, event)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	return result
}

// ExecuteAsync executes a hook asynchronously
func (e *Executor) ExecuteAsync(ctx context.Context, hook Hook, event *Event) <-chan ExecutionResult {
	resultChan := make(chan ExecutionResult, 1)

	go func() {
		result := e.Execute(ctx, hook, event)
		resultChan <- result
		close(resultChan)
	}()

	return resultChan
}

// SetMaxConcurrency sets the maximum number of concurrent hook executions
func (e *Executor) SetMaxConcurrency(max int) {
	if max < 1 {
		max = 1
	}
	e.maxConcurrency = max
}

// SetDefaultTimeout sets the default timeout for hook execution
func (e *Executor) SetDefaultTimeout(timeout time.Duration) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	e.defaultTimeout = timeout
}

// HandleResults processes hook execution results based on failure modes
func HandleResults(results []ExecutionResult, failureMode string, logger Logger) error {
	if len(results) == 0 {
		return nil
	}

	var failures []ExecutionResult
	for _, result := range results {
		if !result.Success {
			failures = append(failures, result)
		}
	}

	if len(failures) == 0 {
		return nil
	}

	// Log failures
	for _, failure := range failures {
		msg := fmt.Sprintf("Hook %s failed for event %s: %s (took %s)",
			failure.HookName, failure.EventType, failure.Error, failure.Duration)

		switch failureMode {
		case "ignore":
			if logger != nil {
				logger.Debug(msg)
			}
		case "warn":
			if logger != nil {
				logger.Warn(msg)
			}
		case "fail":
			if logger != nil {
				logger.Error(msg)
			}
			return fmt.Errorf("hook %s failed: %s", failure.HookName, failure.Error)
		}
	}

	// Only fail if mode is "fail"
	if failureMode == "fail" {
		return fmt.Errorf("%d hook(s) failed", len(failures))
	}

	return nil
}

// Logger interface for logging hook results
type Logger interface {
	Debug(msg string)
	Warn(msg string)
	Error(msg string)
}
