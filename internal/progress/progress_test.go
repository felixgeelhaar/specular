package progress

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
)

func TestNewIndicator(t *testing.T) {
	// Save and clear CI environment variables to test non-CI behavior
	origCI := os.Getenv("CI")
	origGHA := os.Getenv("GITHUB_ACTIONS")
	os.Unsetenv("CI")
	os.Unsetenv("GITHUB_ACTIONS")
	defer func() {
		if origCI != "" {
			os.Setenv("CI", origCI)
		}
		if origGHA != "" {
			os.Setenv("GITHUB_ACTIONS", origGHA)
		}
	}()

	buf := &bytes.Buffer{}
	ind := NewIndicator(Config{
		Writer:      buf,
		ShowSpinner: true,
		IsCI:        false,
	})

	if ind == nil {
		t.Fatal("Expected indicator to be created")
	}

	if ind.writer != buf {
		t.Error("Writer not set correctly")
	}

	if !ind.showSpinner {
		t.Error("Spinner should be enabled")
	}
}

func TestNewIndicatorCIMode(t *testing.T) {
	buf := &bytes.Buffer{}
	ind := NewIndicator(Config{
		Writer:      buf,
		ShowSpinner: true,
		IsCI:        true,
	})

	if ind.showSpinner {
		t.Error("Spinner should be disabled in CI mode")
	}

	if !ind.isCI {
		t.Error("IsCI should be true")
	}
}

func TestSetState(t *testing.T) {
	buf := &bytes.Buffer{}
	ind := NewIndicator(Config{Writer: buf})

	state := checkpoint.NewState("test-operation")
	ind.SetState(state)

	if ind.state == nil {
		t.Error("State should be set")
	}

	if ind.state.OperationID != "test-operation" {
		t.Errorf("Expected operation ID 'test-operation', got %s", ind.state.OperationID)
	}
}

func TestUpdateTask(t *testing.T) {
	buf := &bytes.Buffer{}
	ind := NewIndicator(Config{
		Writer: buf,
		IsCI:   true, // CI mode for predictable output
	})

	state := checkpoint.NewState("test-operation")
	state.UpdateTask("task1", "pending", nil)
	ind.SetState(state)

	// Update to running
	ind.UpdateTask("task1", "running", nil)

	if state.Tasks["task1"].Status != "running" {
		t.Errorf("Expected task1 status 'running', got %s", state.Tasks["task1"].Status)
	}

	// Check CI output
	output := buf.String()
	if !strings.Contains(output, "task1") {
		t.Error("Output should contain task1")
	}
	if !strings.Contains(output, "running") {
		t.Error("Output should contain running status")
	}

	// Update to completed
	buf.Reset()
	ind.UpdateTask("task1", "completed", nil)

	output = buf.String()
	if !strings.Contains(output, "✓") {
		t.Error("Output should contain success symbol")
	}
}

func TestUpdateTaskWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	ind := NewIndicator(Config{
		Writer: buf,
		IsCI:   true,
	})

	state := checkpoint.NewState("test-operation")
	state.UpdateTask("task1", "pending", nil)
	ind.SetState(state)

	// Update to failed with error
	testErr := fmt.Errorf("test error")
	ind.UpdateTask("task1", "failed", testErr)

	if state.Tasks["task1"].Status != "failed" {
		t.Errorf("Expected task1 status 'failed', got %s", state.Tasks["task1"].Status)
	}

	output := buf.String()
	if !strings.Contains(output, "✗") {
		t.Error("Output should contain failure symbol")
	}
	if !strings.Contains(output, "test error") {
		t.Error("Output should contain error message")
	}
}

func TestPrintSummary(t *testing.T) {
	buf := &bytes.Buffer{}
	ind := NewIndicator(Config{Writer: buf})

	state := checkpoint.NewState("test-operation")
	state.UpdateTask("task1", "running", nil)
	state.UpdateTask("task1", "completed", nil)
	state.UpdateTask("task2", "running", nil)
	state.UpdateTask("task2", "completed", nil)
	state.UpdateTask("task3", "running", nil)
	state.UpdateTask("task3", "failed", fmt.Errorf("test error"))

	ind.SetState(state)

	// Small delay to ensure elapsed time is measurable
	time.Sleep(10 * time.Millisecond)

	ind.PrintSummary()

	output := buf.String()

	// Check for key elements
	if !strings.Contains(output, "Execution Summary") {
		t.Error("Output should contain 'Execution Summary'")
	}

	if !strings.Contains(output, "Total Tasks:") {
		t.Error("Output should contain 'Total Tasks'")
	}

	if !strings.Contains(output, "Completed:") {
		t.Error("Output should contain 'Completed'")
	}

	if !strings.Contains(output, "Failed:") {
		t.Error("Output should contain 'Failed'")
	}

	if !strings.Contains(output, "Failed Tasks:") {
		t.Error("Output should contain failed tasks section")
	}

	if !strings.Contains(output, "task3") {
		t.Error("Output should list failed task3")
	}

	if !strings.Contains(output, "test error") {
		t.Error("Output should contain error message")
	}

	// Check success rate calculation
	if !strings.Contains(output, "Success Rate") {
		t.Error("Output should contain success rate")
	}
}

func TestPrintResumeInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	ind := NewIndicator(Config{Writer: buf})

	state := checkpoint.NewState("test-checkpoint-id")
	state.UpdateTask("task1", "running", nil)
	state.UpdateTask("task1", "completed", nil)
	state.UpdateTask("task2", "pending", nil)
	state.UpdateTask("task3", "running", nil)
	state.UpdateTask("task3", "failed", fmt.Errorf("error"))

	ind.SetState(state)
	ind.PrintResumeInfo()

	output := buf.String()

	if !strings.Contains(output, "Resuming:") {
		t.Error("Output should contain 'Resuming:'")
	}

	if !strings.Contains(output, "test-checkpoint-id") {
		t.Error("Output should contain operation ID")
	}

	if !strings.Contains(output, "Completed:") {
		t.Error("Output should show completed count")
	}

	if !strings.Contains(output, "Pending:") {
		t.Error("Output should show pending count")
	}

	if !strings.Contains(output, "Failed:") {
		t.Error("Output should show failed count")
	}

	if !strings.Contains(output, "Progress:") {
		t.Error("Output should show progress percentage")
	}
}

func TestStreamWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	sw := NewStreamWriter(buf, "[TEST]")

	// Write a complete line
	n, err := sw.Write([]byte("Hello, World!\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 14 {
		t.Errorf("Expected to write 14 bytes, wrote %d", n)
	}

	output := buf.String()
	if !strings.Contains(output, "[TEST] Hello, World!") {
		t.Errorf("Expected prefixed output, got: %s", output)
	}

	// Write partial line
	buf.Reset()
	sw.Write([]byte("Partial"))

	// Nothing should be written yet
	if buf.Len() > 0 {
		t.Error("Partial line should not be written")
	}

	// Complete the line
	sw.Write([]byte(" line\n"))

	output = buf.String()
	if !strings.Contains(output, "[TEST] Partial line") {
		t.Errorf("Expected complete prefixed line, got: %s", output)
	}
}

func TestStreamWriterFlush(t *testing.T) {
	buf := &bytes.Buffer{}
	sw := NewStreamWriter(buf, "[PREFIX]")

	// Write incomplete line
	sw.Write([]byte("Incomplete"))

	// Flush should write it
	err := sw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[PREFIX] Incomplete") {
		t.Errorf("Expected flushed output, got: %s", output)
	}

	// Second flush should be a no-op
	buf.Reset()
	err = sw.Flush()
	if err != nil {
		t.Fatalf("Second flush failed: %v", err)
	}

	if buf.Len() > 0 {
		t.Error("Second flush should write nothing")
	}
}

func TestStreamWriterMultipleLines(t *testing.T) {
	buf := &bytes.Buffer{}
	sw := NewStreamWriter(buf, "[LOG]")

	input := "Line 1\nLine 2\nLine 3\n"
	sw.Write([]byte(input))

	output := buf.String()

	expectedLines := []string{
		"[LOG] Line 1",
		"[LOG] Line 2",
		"[LOG] Line 3",
	}

	for _, expected := range expectedLines {
		if !strings.Contains(output, expected) {
			t.Errorf("Output missing line: %s\nGot: %s", expected, output)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m5s"},
		{3665 * time.Second, "1h1m5s"},
		{3600 * time.Second, "1h0m0s"},
		{90 * time.Second, "1m30s"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tt.duration, result, tt.expected)
		}
	}
}

func TestBarIndicator(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewBarIndicator(buf, 10)

	if bar.total != 10 {
		t.Errorf("Expected total 10, got %d", bar.total)
	}

	// Increment with success
	bar.Increment(true)
	if bar.completed != 1 {
		t.Errorf("Expected completed 1, got %d", bar.completed)
	}

	// Increment with failure
	bar.Increment(false)
	if bar.failed != 1 {
		t.Errorf("Expected failed 1, got %d", bar.failed)
	}

	// Check output contains progress elements
	output := buf.String()
	if !strings.Contains(output, "█") || !strings.Contains(output, "░") {
		t.Error("Output should contain progress bar characters")
	}

	if !strings.Contains(output, "✓") {
		t.Error("Output should contain success symbol")
	}

	if !strings.Contains(output, "✗") {
		t.Error("Output should contain failure symbol")
	}
}

func TestBarIndicatorFinish(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewBarIndicator(buf, 5)

	for i := 0; i < 5; i++ {
		bar.Increment(true)
	}

	initialLen := buf.Len()

	bar.Finish()

	// Finish should add a newline
	if buf.Len() <= initialLen {
		t.Error("Finish should add content to buffer")
	}

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Error("Output should end with newline after Finish")
	}
}

func TestIndicatorProgress(t *testing.T) {
	buf := &bytes.Buffer{}
	ind := NewIndicator(Config{
		Writer: buf,
		IsCI:   false,
	})

	state := checkpoint.NewState("progress-test")

	// Add 10 tasks
	for i := 1; i <= 10; i++ {
		taskID := fmt.Sprintf("task%d", i)
		state.UpdateTask(taskID, "pending", nil)
	}

	ind.SetState(state)

	// Complete 5 tasks
	for i := 1; i <= 5; i++ {
		taskID := fmt.Sprintf("task%d", i)
		ind.UpdateTask(taskID, "running", nil)
		ind.UpdateTask(taskID, "completed", nil)
	}

	// Progress should be 50%
	progress := state.Progress()
	if progress != 0.5 {
		t.Errorf("Expected progress 0.5, got %.2f", progress)
	}

	// Complete remaining tasks
	for i := 6; i <= 10; i++ {
		taskID := fmt.Sprintf("task%d", i)
		ind.UpdateTask(taskID, "running", nil)
		ind.UpdateTask(taskID, "completed", nil)
	}

	// Progress should be 100%
	progress = state.Progress()
	if progress != 1.0 {
		t.Errorf("Expected progress 1.0, got %.2f", progress)
	}
}

func TestIndicatorNilState(t *testing.T) {
	buf := &bytes.Buffer{}
	ind := NewIndicator(Config{Writer: buf})

	// Should not panic with nil state
	ind.PrintSummary()
	ind.PrintResumeInfo()

	// Output should be minimal or empty
	output := buf.String()
	if strings.Contains(output, "Execution Summary") {
		t.Error("Should not print summary with nil state")
	}
}

// ======== MultiStepIndicator Tests ========

func TestNewMultiStepIndicator(t *testing.T) {
	buf := &bytes.Buffer{}
	steps := []string{"Step 1", "Step 2", "Step 3"}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer:      buf,
		Steps:       steps,
		ShowSpinner: false,
		IsCI:        true,
	})

	if m == nil {
		t.Fatal("Expected multi-step indicator to be created")
	}

	if len(m.steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(m.steps))
	}

	// All steps should start as pending
	for i, step := range m.steps {
		if step.Status != StepPending {
			t.Errorf("Step %d should be pending, got %s", i, step.Status)
		}
		if step.Name != steps[i] {
			t.Errorf("Step %d name = %s, want %s", i, step.Name, steps[i])
		}
	}
}

func TestMultiStepIndicatorStartStep(t *testing.T) {
	buf := &bytes.Buffer{}
	steps := []string{"Build", "Test", "Deploy"}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer: buf,
		Steps:  steps,
		IsCI:   true,
	})

	m.Start()
	m.StartStep("Build")

	if m.current != 0 {
		t.Errorf("Current step should be 0, got %d", m.current)
	}

	if m.steps[0].Status != StepRunning {
		t.Errorf("Step should be running, got %s", m.steps[0].Status)
	}

	output := buf.String()
	if !strings.Contains(output, "Build") {
		t.Error("Output should contain step name 'Build'")
	}
}

func TestMultiStepIndicatorCompleteStep(t *testing.T) {
	buf := &bytes.Buffer{}
	steps := []string{"Build", "Test"}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer: buf,
		Steps:  steps,
		IsCI:   true,
	})

	m.Start()
	m.StartStep("Build")
	time.Sleep(10 * time.Millisecond)
	m.CompleteStep(nil)

	if m.steps[0].Status != StepCompleted {
		t.Errorf("Step should be completed, got %s", m.steps[0].Status)
	}

	if m.steps[0].Duration == 0 {
		t.Error("Step duration should be recorded")
	}

	output := buf.String()
	if !strings.Contains(output, "✓") {
		t.Error("Output should contain success symbol")
	}
}

func TestMultiStepIndicatorCompleteStepWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	steps := []string{"Build", "Test"}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer: buf,
		Steps:  steps,
		IsCI:   true,
	})

	m.Start()
	m.StartStep("Build")
	m.CompleteStep(fmt.Errorf("build failed"))

	if m.steps[0].Status != StepFailed {
		t.Errorf("Step should be failed, got %s", m.steps[0].Status)
	}

	if m.steps[0].Error == nil {
		t.Error("Step error should be recorded")
	}

	output := buf.String()
	if !strings.Contains(output, "✗") {
		t.Error("Output should contain failure symbol")
	}
	if !strings.Contains(output, "build failed") {
		t.Error("Output should contain error message")
	}
}

func TestMultiStepIndicatorSkipStep(t *testing.T) {
	buf := &bytes.Buffer{}
	steps := []string{"Build", "Test", "Deploy"}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer: buf,
		Steps:  steps,
		IsCI:   true,
	})

	m.Start()
	m.SkipStep("Test", "no tests found")

	if m.steps[1].Status != StepSkipped {
		t.Errorf("Step should be skipped, got %s", m.steps[1].Status)
	}

	output := buf.String()
	if !strings.Contains(output, "⊘") {
		t.Error("Output should contain skip symbol")
	}
	if !strings.Contains(output, "no tests found") {
		t.Error("Output should contain skip reason")
	}
}

func TestMultiStepIndicatorStartNextStep(t *testing.T) {
	buf := &bytes.Buffer{}
	steps := []string{"Build", "Test", "Deploy"}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer: buf,
		Steps:  steps,
		IsCI:   true,
	})

	m.Start()

	// Start first step
	ok := m.StartNextStep()
	if !ok {
		t.Error("Should find next step")
	}
	if m.CurrentStep() != "Build" {
		t.Errorf("Current step should be 'Build', got %s", m.CurrentStep())
	}

	// Complete and start next
	m.CompleteStep(nil)
	ok = m.StartNextStep()
	if !ok {
		t.Error("Should find next step")
	}
	if m.CurrentStep() != "Test" {
		t.Errorf("Current step should be 'Test', got %s", m.CurrentStep())
	}

	// Complete all
	m.CompleteStep(nil)
	m.StartNextStep()
	m.CompleteStep(nil)

	// No more steps
	ok = m.StartNextStep()
	if ok {
		t.Error("Should not find next step when all completed")
	}
}

func TestMultiStepIndicatorProgress(t *testing.T) {
	buf := &bytes.Buffer{}
	steps := []string{"Step1", "Step2", "Step3", "Step4"}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer: buf,
		Steps:  steps,
		IsCI:   true,
	})

	// Initial progress
	if m.Progress() != 0 {
		t.Errorf("Initial progress should be 0, got %f", m.Progress())
	}

	// Complete 2 steps
	m.StartStep("Step1")
	m.CompleteStep(nil)
	m.StartStep("Step2")
	m.CompleteStep(nil)

	// 2/4 = 0.5
	progress := m.Progress()
	if progress != 0.5 {
		t.Errorf("Progress should be 0.5, got %f", progress)
	}

	// Complete all
	m.StartStep("Step3")
	m.CompleteStep(nil)
	m.StartStep("Step4")
	m.CompleteStep(nil)

	if m.Progress() != 1.0 {
		t.Errorf("Progress should be 1.0, got %f", m.Progress())
	}
}

func TestMultiStepIndicatorHasFailures(t *testing.T) {
	buf := &bytes.Buffer{}
	steps := []string{"Build", "Test"}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer: buf,
		Steps:  steps,
		IsCI:   true,
	})

	// Initially no failures
	if m.HasFailures() {
		t.Error("Should not have failures initially")
	}

	// Complete one successfully
	m.StartStep("Build")
	m.CompleteStep(nil)

	if m.HasFailures() {
		t.Error("Should not have failures after success")
	}

	// Fail one
	m.StartStep("Test")
	m.CompleteStep(fmt.Errorf("test failed"))

	if !m.HasFailures() {
		t.Error("Should have failures after error")
	}
}

func TestMultiStepIndicatorPrintSummary(t *testing.T) {
	buf := &bytes.Buffer{}
	steps := []string{"Build", "Test", "Deploy"}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer: buf,
		Steps:  steps,
		IsCI:   true,
	})

	m.Start()

	// Complete some steps
	m.StartStep("Build")
	m.CompleteStep(nil)

	m.StartStep("Test")
	m.CompleteStep(fmt.Errorf("test failed"))

	m.SkipStep("Deploy", "test failed")

	m.PrintSummary()

	output := buf.String()

	if !strings.Contains(output, "Summary") {
		t.Error("Should contain 'Summary'")
	}
	if !strings.Contains(output, "Completed: 1") {
		t.Error("Should show 1 completed")
	}
	if !strings.Contains(output, "Failed:    1") {
		t.Error("Should show 1 failed")
	}
	if !strings.Contains(output, "Skipped:   1") {
		t.Error("Should show 1 skipped")
	}
	if !strings.Contains(output, "Failed steps:") {
		t.Error("Should list failed steps")
	}
	if !strings.Contains(output, "Test") {
		t.Error("Should list Test as failed")
	}
}

func TestMultiStepIndicatorEmptySteps(t *testing.T) {
	buf := &bytes.Buffer{}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer: buf,
		Steps:  []string{},
		IsCI:   true,
	})

	// Should not panic with empty steps
	m.Start()
	m.PrintSummary()

	// Progress should be 0 for empty steps
	if m.Progress() != 0 {
		t.Errorf("Progress for empty steps should be 0, got %f", m.Progress())
	}

	// HasFailures should be false
	if m.HasFailures() {
		t.Error("Empty steps should have no failures")
	}
}

func TestMultiStepIndicatorStop(t *testing.T) {
	buf := &bytes.Buffer{}
	steps := []string{"Step1", "Step2"}

	m := NewMultiStepIndicator(MultiStepConfig{
		Writer:      buf,
		Steps:       steps,
		ShowSpinner: true,
		IsCI:        false, // Enable spinner
	})

	m.Start()

	// Stop should not panic and should be idempotent
	m.Stop()
	m.Stop() // Second call should be safe
}

func TestStepStatusConstants(t *testing.T) {
	// Verify step status constants are distinct
	statuses := []StepStatus{
		StepPending,
		StepRunning,
		StepCompleted,
		StepFailed,
		StepSkipped,
	}

	seen := make(map[StepStatus]bool)
	for _, status := range statuses {
		if seen[status] {
			t.Errorf("Duplicate status: %s", status)
		}
		seen[status] = true
	}

	// Verify expected values
	if StepPending != "pending" {
		t.Errorf("StepPending = %s, want pending", StepPending)
	}
	if StepRunning != "running" {
		t.Errorf("StepRunning = %s, want running", StepRunning)
	}
	if StepCompleted != "completed" {
		t.Errorf("StepCompleted = %s, want completed", StepCompleted)
	}
	if StepFailed != "failed" {
		t.Errorf("StepFailed = %s, want failed", StepFailed)
	}
	if StepSkipped != "skipped" {
		t.Errorf("StepSkipped = %s, want skipped", StepSkipped)
	}
}
