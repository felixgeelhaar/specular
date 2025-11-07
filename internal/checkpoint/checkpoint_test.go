package checkpoint

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	operationID := "test-operation"
	state := NewState(operationID)

	if state.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", state.Version)
	}
	if state.OperationID != operationID {
		t.Errorf("expected operation ID %s, got %s", operationID, state.OperationID)
	}
	if state.Status != "running" {
		t.Errorf("expected status running, got %s", state.Status)
	}
	if state.Tasks == nil {
		t.Error("tasks map should be initialized")
	}
	if state.Metadata == nil {
		t.Error("metadata map should be initialized")
	}
}

func TestManagerSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir, false, 0)

	// Create a state
	state := NewState("test-save-load")
	state.UpdateTask("task1", "completed", nil)
	state.UpdateTask("task2", "running", nil)
	state.SetMetadata("key1", "value1")

	// Save the state
	if err := manager.Save(state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Load the state
	loaded, err := manager.Load("test-save-load")
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	// Verify
	if loaded.OperationID != state.OperationID {
		t.Errorf("expected operation ID %s, got %s", state.OperationID, loaded.OperationID)
	}
	if len(loaded.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(loaded.Tasks))
	}
	if loaded.Tasks["task1"].Status != "completed" {
		t.Errorf("expected task1 status completed, got %s", loaded.Tasks["task1"].Status)
	}
	if value, ok := loaded.GetMetadata("key1"); !ok || value != "value1" {
		t.Errorf("expected metadata key1=value1, got %s (exists: %v)", value, ok)
	}
}

func TestManagerExists(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir, false, 0)

	operationID := "test-exists"

	// Should not exist initially
	if manager.Exists(operationID) {
		t.Error("checkpoint should not exist initially")
	}

	// Create and save
	state := NewState(operationID)
	if err := manager.Save(state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Should exist now
	if !manager.Exists(operationID) {
		t.Error("checkpoint should exist after save")
	}
}

func TestManagerDelete(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir, false, 0)

	operationID := "test-delete"

	// Create and save
	state := NewState(operationID)
	if err := manager.Save(state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Delete
	if err := manager.Delete(operationID); err != nil {
		t.Fatalf("failed to delete checkpoint: %v", err)
	}

	// Should not exist
	if manager.Exists(operationID) {
		t.Error("checkpoint should not exist after delete")
	}

	// Deleting non-existent should not error
	if err := manager.Delete("nonexistent"); err != nil {
		t.Errorf("deleting nonexistent checkpoint should not error: %v", err)
	}
}

func TestManagerList(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir, false, 0)

	// Initially empty
	list, err := manager.List()
	if err != nil {
		t.Fatalf("failed to list checkpoints: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 checkpoints, got %d", len(list))
	}

	// Create multiple checkpoints
	operations := []string{"op1", "op2", "op3"}
	for _, opID := range operations {
		state := NewState(opID)
		if err := manager.Save(state); err != nil {
			t.Fatalf("failed to save state %s: %v", opID, err)
		}
	}

	// List should contain all
	list, err = manager.List()
	if err != nil {
		t.Fatalf("failed to list checkpoints: %v", err)
	}
	if len(list) != len(operations) {
		t.Errorf("expected %d checkpoints, got %d", len(operations), len(list))
	}

	// Verify all operation IDs are present
	found := make(map[string]bool)
	for _, opID := range list {
		found[opID] = true
	}
	for _, opID := range operations {
		if !found[opID] {
			t.Errorf("operation ID %s not found in list", opID)
		}
	}
}

func TestStateUpdateTask(t *testing.T) {
	state := NewState("test-update")

	// Create new task
	state.UpdateTask("task1", "running", nil)
	task := state.Tasks["task1"]

	if task.ID != "task1" {
		t.Errorf("expected task ID task1, got %s", task.ID)
	}
	if task.Status != "running" {
		t.Errorf("expected status running, got %s", task.Status)
	}
	if task.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", task.Attempts)
	}
	if task.StartedAt.IsZero() {
		t.Error("started_at should be set when transitioning to running")
	}

	// Complete the task
	state.UpdateTask("task1", "completed", nil)
	task = state.Tasks["task1"]

	if task.Status != "completed" {
		t.Errorf("expected status completed, got %s", task.Status)
	}
	if task.CompletedAt.IsZero() {
		t.Error("completed_at should be set when completing")
	}

	// Fail a task with error
	testErr := fmt.Errorf("test error")
	state.UpdateTask("task2", "failed", testErr)
	task = state.Tasks["task2"]

	if task.Status != "failed" {
		t.Errorf("expected status failed, got %s", task.Status)
	}
	if task.Error != "test error" {
		t.Errorf("expected error 'test error', got %s", task.Error)
	}
}

func TestStateGetPendingTasks(t *testing.T) {
	state := NewState("test-pending")

	state.UpdateTask("task1", "pending", nil)
	state.UpdateTask("task2", "running", nil)
	state.UpdateTask("task3", "completed", nil)
	state.UpdateTask("task4", "failed", nil)

	pending := state.GetPendingTasks()

	if len(pending) != 2 {
		t.Errorf("expected 2 pending tasks, got %d", len(pending))
	}

	// Verify task1 and task2 are in pending
	found := make(map[string]bool)
	for _, id := range pending {
		found[id] = true
	}
	if !found["task1"] || !found["task2"] {
		t.Error("expected task1 and task2 in pending tasks")
	}
}

func TestStateGetCompletedTasks(t *testing.T) {
	state := NewState("test-completed")

	state.UpdateTask("task1", "completed", nil)
	state.UpdateTask("task2", "completed", nil)
	state.UpdateTask("task3", "running", nil)

	completed := state.GetCompletedTasks()

	if len(completed) != 2 {
		t.Errorf("expected 2 completed tasks, got %d", len(completed))
	}
}

func TestStateGetFailedTasks(t *testing.T) {
	state := NewState("test-failed")

	state.UpdateTask("task1", "failed", fmt.Errorf("error1"))
	state.UpdateTask("task2", "completed", nil)
	state.UpdateTask("task3", "failed", fmt.Errorf("error2"))

	failed := state.GetFailedTasks()

	if len(failed) != 2 {
		t.Errorf("expected 2 failed tasks, got %d", len(failed))
	}
}

func TestStateIsComplete(t *testing.T) {
	state := NewState("test-complete")

	// Empty state is not complete
	if state.IsComplete() {
		t.Error("empty state should not be complete")
	}

	// All completed
	state.UpdateTask("task1", "completed", nil)
	state.UpdateTask("task2", "completed", nil)
	if !state.IsComplete() {
		t.Error("state with all completed tasks should be complete")
	}

	// Mix of completed and skipped
	state.UpdateTask("task3", "skipped", nil)
	if !state.IsComplete() {
		t.Error("state with completed and skipped tasks should be complete")
	}

	// Add a pending task
	state.UpdateTask("task4", "pending", nil)
	if state.IsComplete() {
		t.Error("state with pending tasks should not be complete")
	}
}

func TestStateProgress(t *testing.T) {
	state := NewState("test-progress")

	// Empty state has 0% progress
	if progress := state.Progress(); progress != 0.0 {
		t.Errorf("expected 0%% progress, got %.2f", progress*100)
	}

	// Add tasks
	state.UpdateTask("task1", "completed", nil)
	state.UpdateTask("task2", "pending", nil)
	state.UpdateTask("task3", "running", nil)
	state.UpdateTask("task4", "completed", nil)

	// 2/4 completed = 50%
	progress := state.Progress()
	expected := 0.5
	if progress != expected {
		t.Errorf("expected %.2f progress, got %.2f", expected, progress)
	}

	// Complete all
	state.UpdateTask("task2", "completed", nil)
	state.UpdateTask("task3", "completed", nil)

	// 4/4 = 100%
	progress = state.Progress()
	if progress != 1.0 {
		t.Errorf("expected 100%% progress, got %.2f", progress*100)
	}
}

func TestStateAddArtifact(t *testing.T) {
	state := NewState("test-artifact")

	state.UpdateTask("task1", "running", nil)
	state.AddArtifact("task1", "/path/to/artifact1.txt")
	state.AddArtifact("task1", "/path/to/artifact2.txt")

	task := state.Tasks["task1"]
	if len(task.Artifacts) != 2 {
		t.Errorf("expected 2 artifacts, got %d", len(task.Artifacts))
	}
	if task.Artifacts[0] != "/path/to/artifact1.txt" {
		t.Errorf("expected artifact1.txt, got %s", task.Artifacts[0])
	}

	// Adding artifact to non-existent task should not panic
	state.AddArtifact("nonexistent", "/path/to/artifact.txt")
}

func TestStateMetadata(t *testing.T) {
	state := NewState("test-metadata")

	// Set metadata
	state.SetMetadata("plan", "plan.json")
	state.SetMetadata("policy", "policy.yaml")

	// Get metadata
	if value, ok := state.GetMetadata("plan"); !ok || value != "plan.json" {
		t.Errorf("expected plan=plan.json, got %s (exists: %v)", value, ok)
	}

	// Non-existent key
	if _, ok := state.GetMetadata("nonexistent"); ok {
		t.Error("nonexistent key should return false")
	}
}

func TestManagerSaveCreatesDirectory(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "nested", "checkpoint", "dir")
	manager := NewManager(tmpDir, false, 0)

	state := NewState("test-create-dir")

	// Directory doesn't exist yet
	if _, err := os.Stat(tmpDir); err == nil {
		t.Fatal("directory should not exist yet")
	}

	// Save should create directory
	if err := manager.Save(state); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Directory should now exist
	if _, err := os.Stat(tmpDir); err != nil {
		t.Fatalf("directory should exist after save: %v", err)
	}
}

func TestManagerLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir, false, 0)

	_, err := manager.Load("nonexistent")
	if err == nil {
		t.Error("loading nonexistent checkpoint should return error")
	}
}

func TestManagerSaveNilState(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir, false, 0)

	err := manager.Save(nil)
	if err == nil {
		t.Error("saving nil state should return error")
	}
}

func TestStateTimestamps(t *testing.T) {
	state := NewState("test-timestamps")

	// Record initial time
	initialUpdated := state.UpdatedAt

	// Wait a bit to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	// Update task
	state.UpdateTask("task1", "running", nil)

	// UpdatedAt should be newer
	if !state.UpdatedAt.After(initialUpdated) {
		t.Error("UpdatedAt should be updated after task update")
	}

	// StartedAt should be set for the task
	task := state.Tasks["task1"]
	if task.StartedAt.IsZero() {
		t.Error("task StartedAt should be set when moving to running")
	}
}

func TestStateRetryAttempts(t *testing.T) {
	state := NewState("test-retry")

	// First attempt
	state.UpdateTask("task1", "running", nil)
	if state.Tasks["task1"].Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", state.Tasks["task1"].Attempts)
	}

	// Fail and retry
	state.UpdateTask("task1", "failed", fmt.Errorf("temporary error"))
	state.UpdateTask("task1", "pending", nil)
	state.UpdateTask("task1", "running", nil)

	if state.Tasks["task1"].Attempts != 2 {
		t.Errorf("expected 2 attempts after retry, got %d", state.Tasks["task1"].Attempts)
	}
}
