package checkpoint

import (
	"fmt"
	"testing"
)

func TestStateTaskTrackingAndProgress(t *testing.T) {
	state := NewState("state-progress")

	state.UpdateTask("task-running", "running", nil)
	state.UpdateTask("task-complete", "completed", nil)
	state.UpdateTask("task-failed", "failed", fmt.Errorf("boom"))

	pending := state.GetPendingTasks()
	if len(pending) != 1 || pending[0] != "task-running" {
		t.Fatalf("expected running task to be pending, got %v", pending)
	}

	completed := state.GetCompletedTasks()
	if len(completed) != 1 || completed[0] != "task-complete" {
		t.Fatalf("expected one completed task, got %v", completed)
	}

	failed := state.GetFailedTasks()
	if len(failed) != 1 || failed[0] != "task-failed" {
		t.Fatalf("expected one failed task, got %v", failed)
	}

	progress := state.Progress()
	if progress <= 0 || progress >= 1 {
		t.Fatalf("expected partial progress, got %f", progress)
	}

	state.AddArtifact("task-complete", "/tmp/artifact.txt")
	if len(state.Tasks["task-complete"].Artifacts) != 1 {
		t.Fatalf("expected artifact recorded, got %v", state.Tasks["task-complete"].Artifacts)
	}

	state.SetMetadata("env", "test")
	if val, ok := state.GetMetadata("env"); !ok || val != "test" {
		t.Fatalf("expected metadata env=test, got %s (exists=%v)", val, ok)
	}

	if state.IsComplete() {
		t.Fatal("state should not be complete yet")
	}

	state.UpdateTask("task-running", "completed", nil)
	state.UpdateTask("task-failed", "skipped", nil)

	if !state.IsComplete() {
		t.Fatal("state should be marked complete when no pending tasks remain")
	}
}
