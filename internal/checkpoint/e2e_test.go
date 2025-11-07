package checkpoint_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
)

// TestE2ECheckpointResume tests the full checkpoint/resume workflow
func TestE2ECheckpointResume(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointDir := filepath.Join(tmpDir, ".specular", "checkpoints")

	// Scenario: Simulate a build with 10 tasks that gets interrupted after task 5
	operationID := "e2e-build-test"

	// Phase 1: Initial execution (tasks 1-5 complete, then interrupted)
	t.Run("Phase1_InitialExecution", func(t *testing.T) {
		mgr := checkpoint.NewManager(checkpointDir, true, 30*time.Second)
		state := checkpoint.NewState(operationID)

		// Set metadata
		state.SetMetadata("plan", "test-plan.json")
		state.SetMetadata("policy", ".specular/policy.yaml")

		// Initialize all tasks as pending
		for i := 1; i <= 10; i++ {
			taskID := fmt.Sprintf("task%d", i)
			state.UpdateTask(taskID, "pending", nil)
		}

		// Save initial checkpoint
		if err := mgr.Save(state); err != nil {
			t.Fatalf("Failed to save initial checkpoint: %v", err)
		}

		// Execute tasks 1-5 successfully
		for i := 1; i <= 5; i++ {
			taskID := fmt.Sprintf("task%d", i)
			state.UpdateTask(taskID, "running", nil)
			time.Sleep(10 * time.Millisecond) // Simulate work
			state.UpdateTask(taskID, "completed", nil)
			mgr.Save(state)
		}

		// Simulate task 6 starting but failing
		state.UpdateTask("task6", "running", nil)
		mgr.Save(state)
		state.UpdateTask("task6", "failed", fmt.Errorf("simulated network error"))
		mgr.Save(state)

		// At this point: 5 completed, 1 failed, 4 pending
		// System crashes/interrupted here
		completed := len(state.GetCompletedTasks())
		failed := len(state.GetFailedTasks())
		pending := len(state.GetPendingTasks())

		if completed != 5 {
			t.Errorf("Expected 5 completed tasks, got %d", completed)
		}
		if failed != 1 {
			t.Errorf("Expected 1 failed task, got %d", failed)
		}
		if pending != 4 {
			t.Errorf("Expected 4 pending tasks, got %d", pending)
		}

		// Verify checkpoint exists
		if !mgr.Exists(operationID) {
			t.Fatal("Checkpoint should exist after initial execution")
		}
	})

	// Phase 2: Resume and complete remaining tasks
	t.Run("Phase2_Resume", func(t *testing.T) {
		mgr := checkpoint.NewManager(checkpointDir, true, 30*time.Second)

		// Load checkpoint
		state, err := mgr.Load(operationID)
		if err != nil {
			t.Fatalf("Failed to load checkpoint: %v", err)
		}

		// Verify metadata persisted
		plan, ok := state.GetMetadata("plan")
		if !ok || plan != "test-plan.json" {
			t.Errorf("Expected plan metadata to be 'test-plan.json', got %s", plan)
		}

		// Verify state is correct
		completed := state.GetCompletedTasks()
		if len(completed) != 5 {
			t.Errorf("Expected 5 completed tasks on resume, got %d", len(completed))
		}

		failed := state.GetFailedTasks()
		if len(failed) != 1 {
			t.Errorf("Expected 1 failed task on resume, got %d", len(failed))
		}

		// Check progress
		progress := state.Progress()
		expectedProgress := 0.5 // 5/10 completed
		if progress != expectedProgress {
			t.Errorf("Expected progress %.2f, got %.2f", expectedProgress, progress)
		}

		// Retry failed task (task6)
		state.UpdateTask("task6", "pending", nil)
		state.UpdateTask("task6", "running", nil)
		time.Sleep(10 * time.Millisecond)
		state.UpdateTask("task6", "completed", nil)
		mgr.Save(state)

		// Verify retry attempt count
		task6 := state.Tasks["task6"]
		if task6.Attempts != 2 {
			t.Errorf("Expected task6 to have 2 attempts, got %d", task6.Attempts)
		}

		// Complete remaining tasks (7-10)
		for i := 7; i <= 10; i++ {
			taskID := fmt.Sprintf("task%d", i)
			state.UpdateTask(taskID, "running", nil)
			time.Sleep(10 * time.Millisecond)
			state.UpdateTask(taskID, "completed", nil)
			mgr.Save(state)
		}

		// Verify all tasks completed
		if !state.IsComplete() {
			t.Error("State should be complete after all tasks finished")
		}

		// Final progress should be 100%
		finalProgress := state.Progress()
		if finalProgress != 1.0 {
			t.Errorf("Expected final progress 1.0, got %.2f", finalProgress)
		}

		// Mark operation as completed
		state.Status = "completed"
		mgr.Save(state)
	})

	// Phase 3: Verify checkpoint can be cleaned up
	t.Run("Phase3_Cleanup", func(t *testing.T) {
		mgr := checkpoint.NewManager(checkpointDir, true, 30*time.Second)

		// Verify checkpoint exists
		if !mgr.Exists(operationID) {
			t.Fatal("Checkpoint should exist before cleanup")
		}

		// Delete checkpoint
		if err := mgr.Delete(operationID); err != nil {
			t.Fatalf("Failed to delete checkpoint: %v", err)
		}

		// Verify checkpoint is gone
		if mgr.Exists(operationID) {
			t.Error("Checkpoint should not exist after deletion")
		}
	})
}

// TestE2EMultipleCheckpoints tests managing multiple concurrent checkpoints
func TestE2EMultipleCheckpoints(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointDir := filepath.Join(tmpDir, ".specular", "checkpoints")
	mgr := checkpoint.NewManager(checkpointDir, true, 30*time.Second)

	// Create multiple checkpoints for different operations
	operations := []string{
		"build-project-a",
		"build-project-b",
		"eval-project-a",
	}

	for _, opID := range operations {
		state := checkpoint.NewState(opID)
		state.SetMetadata("operation", opID)

		// Add some tasks
		for i := 1; i <= 5; i++ {
			taskID := fmt.Sprintf("task%d", i)
			state.UpdateTask(taskID, "pending", nil)
		}

		// Complete some tasks
		state.UpdateTask("task1", "running", nil)
		state.UpdateTask("task1", "completed", nil)
		state.UpdateTask("task2", "running", nil)
		state.UpdateTask("task2", "completed", nil)

		if err := mgr.Save(state); err != nil {
			t.Fatalf("Failed to save checkpoint for %s: %v", opID, err)
		}
	}

	// List all checkpoints
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(list) != len(operations) {
		t.Errorf("Expected %d checkpoints, got %d", len(operations), len(list))
	}

	// Verify we can load each checkpoint independently
	for _, opID := range operations {
		state, err := mgr.Load(opID)
		if err != nil {
			t.Errorf("Failed to load checkpoint %s: %v", opID, err)
		}

		metadata, ok := state.GetMetadata("operation")
		if !ok || metadata != opID {
			t.Errorf("Checkpoint %s has incorrect metadata", opID)
		}

		// Should have 2 completed tasks
		completed := len(state.GetCompletedTasks())
		if completed != 2 {
			t.Errorf("Checkpoint %s should have 2 completed tasks, got %d", opID, completed)
		}
	}

	// Clean up all checkpoints
	for _, opID := range operations {
		if err := mgr.Delete(opID); err != nil {
			t.Errorf("Failed to delete checkpoint %s: %v", opID, err)
		}
	}

	// Verify all cleaned up
	list, err = mgr.List()
	if err != nil {
		t.Fatalf("Failed to list checkpoints after cleanup: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("Expected 0 checkpoints after cleanup, got %d", len(list))
	}
}

// TestE2ECheckpointJSONFormat tests that checkpoint JSON is valid and readable
func TestE2ECheckpointJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointDir := filepath.Join(tmpDir, ".specular", "checkpoints")
	mgr := checkpoint.NewManager(checkpointDir, true, 30*time.Second)

	operationID := "format-test"
	state := checkpoint.NewState(operationID)

	// Add metadata
	state.SetMetadata("plan", "plan.json")
	state.SetMetadata("policy", ".specular/policy.yaml")

	// Add tasks with various states
	state.UpdateTask("task1", "running", nil)
	state.UpdateTask("task1", "completed", nil)

	state.UpdateTask("task2", "running", nil)
	state.UpdateTask("task2", "failed", fmt.Errorf("test error"))

	state.UpdateTask("task3", "pending", nil)

	state.UpdateTask("task4", "running", nil)
	state.UpdateTask("task4", "skipped", nil)

	// Add artifacts
	state.AddArtifact("task1", "/path/to/artifact1.txt")
	state.AddArtifact("task1", "/path/to/artifact2.txt")

	// Save checkpoint
	if err := mgr.Save(state); err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Read raw JSON file
	checkpointPath := filepath.Join(checkpointDir, fmt.Sprintf("%s.json", operationID))
	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		t.Fatalf("Failed to read checkpoint file: %v", err)
	}

	// Verify it's valid JSON
	var rawState map[string]interface{}
	if err := json.Unmarshal(data, &rawState); err != nil {
		t.Fatalf("Checkpoint file is not valid JSON: %v", err)
	}

	// Verify expected fields exist
	expectedFields := []string{"version", "operation_id", "started_at", "updated_at", "status", "tasks", "metadata"}
	for _, field := range expectedFields {
		if _, ok := rawState[field]; !ok {
			t.Errorf("Checkpoint JSON missing required field: %s", field)
		}
	}

	// Verify tasks have correct structure
	tasks, ok := rawState["tasks"].(map[string]interface{})
	if !ok {
		t.Fatal("Tasks field is not a map")
	}

	// Check task1 has artifacts
	task1, ok := tasks["task1"].(map[string]interface{})
	if !ok {
		t.Fatal("task1 is not a map")
	}

	artifacts, ok := task1["artifacts"].([]interface{})
	if !ok || len(artifacts) != 2 {
		t.Errorf("task1 should have 2 artifacts, got %d", len(artifacts))
	}

	// Check task2 has error
	task2, ok := tasks["task2"].(map[string]interface{})
	if !ok {
		t.Fatal("task2 is not a map")
	}

	errorMsg, ok := task2["error"].(string)
	if !ok || errorMsg != "test error" {
		t.Errorf("task2 should have error 'test error', got '%s'", errorMsg)
	}

	// Verify version
	version, ok := rawState["version"].(string)
	if !ok || version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", version)
	}

	t.Logf("Checkpoint JSON format validated successfully")
	t.Logf("Checkpoint content:\n%s", string(data))
}

// TestE2ECheckpointConcurrentAccess tests concurrent read/write to checkpoints
func TestE2ECheckpointConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointDir := filepath.Join(tmpDir, ".specular", "checkpoints")
	mgr := checkpoint.NewManager(checkpointDir, true, 30*time.Second)

	operationID := "concurrent-test"
	state := checkpoint.NewState(operationID)

	// Initialize tasks
	for i := 1; i <= 20; i++ {
		taskID := fmt.Sprintf("task%d", i)
		state.UpdateTask(taskID, "pending", nil)
	}

	// Save initial state
	if err := mgr.Save(state); err != nil {
		t.Fatalf("Failed to save initial checkpoint: %v", err)
	}

	// Simulate concurrent task updates
	// In real scenario, this would be multiple goroutines updating different tasks
	// For testing, we'll sequentially update and save
	for i := 1; i <= 20; i++ {
		taskID := fmt.Sprintf("task%d", i)

		// Load current state
		currentState, err := mgr.Load(operationID)
		if err != nil {
			t.Fatalf("Failed to load checkpoint: %v", err)
		}

		// Update task
		currentState.UpdateTask(taskID, "running", nil)
		time.Sleep(5 * time.Millisecond)
		currentState.UpdateTask(taskID, "completed", nil)

		// Save updated state
		if err := mgr.Save(currentState); err != nil {
			t.Fatalf("Failed to save checkpoint: %v", err)
		}
	}

	// Verify final state
	finalState, err := mgr.Load(operationID)
	if err != nil {
		t.Fatalf("Failed to load final checkpoint: %v", err)
	}

	if !finalState.IsComplete() {
		t.Error("All tasks should be completed")
	}

	if len(finalState.GetCompletedTasks()) != 20 {
		t.Errorf("Expected 20 completed tasks, got %d", len(finalState.GetCompletedTasks()))
	}
}
