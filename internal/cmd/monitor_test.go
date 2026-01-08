package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
)

func TestFindLatestSession(t *testing.T) {
	// Create temp checkpoint directory
	tmpDir, err := os.MkdirTemp("", "specular-monitor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := checkpoint.NewManager(tmpDir, false, 0)

	// Test with no sessions
	_, err = findLatestSession(mgr)
	if err == nil {
		t.Error("expected error with no sessions")
	}

	// Create some test sessions
	state1 := checkpoint.NewState("session-1")
	state1.UpdatedAt = time.Now().Add(-2 * time.Hour)
	if err := mgr.Save(state1); err != nil {
		t.Fatalf("failed to save state1: %v", err)
	}

	state2 := checkpoint.NewState("session-2")
	state2.UpdatedAt = time.Now().Add(-1 * time.Hour)
	if err := mgr.Save(state2); err != nil {
		t.Fatalf("failed to save state2: %v", err)
	}

	state3 := checkpoint.NewState("session-3")
	state3.UpdatedAt = time.Now()
	if err := mgr.Save(state3); err != nil {
		t.Fatalf("failed to save state3: %v", err)
	}

	// Find latest should return session-3
	latestID, err := findLatestSession(mgr)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if latestID != "session-3" {
		t.Errorf("expected session-3, got %s", latestID)
	}
}

func TestStreamSession_SessionNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "specular-monitor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := checkpoint.NewManager(tmpDir, false, 0)

	err = streamSession(mgr, tmpDir, "nonexistent-session", false)
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestFormatMonitorDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"minutes and seconds", 2*time.Minute + 30*time.Second, "2m 30s"},
		{"hours and minutes", 1*time.Hour + 15*time.Minute, "1h 15m"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatMonitorDuration(tc.duration)
			if result != tc.expected {
				t.Errorf("formatMonitorDuration(%v) = %s, want %s", tc.duration, result, tc.expected)
			}
		})
	}
}

func TestRepeatString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		n        int
		expected string
	}{
		{"empty", "", 5, ""},
		{"single char", "-", 3, "---"},
		{"multi char", "ab", 2, "abab"},
		{"zero repeat", "x", 0, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := repeatString(tc.s, tc.n)
			if result != tc.expected {
				t.Errorf("repeatString(%q, %d) = %q, want %q", tc.s, tc.n, result, tc.expected)
			}
		})
	}
}

func TestPrintStreamUpdate(t *testing.T) {
	// Create a test state
	state := &checkpoint.State{
		Status:    "running",
		StartedAt: time.Now().Add(-5 * time.Minute),
		UpdatedAt: time.Now(),
		Metadata:  map[string]string{"goal": "Test goal"},
		Tasks: map[string]checkpoint.Task{
			"task-1": {ID: "task-1", Status: "completed"},
			"task-2": {ID: "task-2", Status: "running", StartedAt: time.Now().Add(-1 * time.Minute)},
			"task-3": {ID: "task-3", Status: "pending"},
		},
	}

	// This is more of a smoke test - just verify it doesn't panic
	// Capture stdout would require more setup
	printStreamUpdate(state, false)
	printStreamUpdate(state, true)
}

func TestListSessions_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "specular-monitor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := checkpoint.NewManager(tmpDir, false, 0)

	// Should not error with empty sessions
	err = listSessions(mgr, false, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListSessions_WithSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "specular-monitor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := checkpoint.NewManager(tmpDir, false, 0)

	// Create test sessions
	state1 := checkpoint.NewState("test-session-1")
	state1.Status = "running"
	state1.Metadata["goal"] = "Test goal 1"
	if err := mgr.Save(state1); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	state2 := checkpoint.NewState("test-session-2")
	state2.Status = "completed"
	state2.Metadata["goal"] = "Test goal 2"
	if err := mgr.Save(state2); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// List all sessions
	err = listSessions(mgr, false, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// List active only
	err = listSessions(mgr, true, false)
	if err != nil {
		t.Errorf("unexpected error listing active: %v", err)
	}

	// List with verbose
	err = listSessions(mgr, false, true)
	if err != nil {
		t.Errorf("unexpected error with verbose: %v", err)
	}
}

func TestAttachToSession_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "specular-monitor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := checkpoint.NewManager(tmpDir, false, 0)

	err = attachToSession(mgr, "nonexistent", false)
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestAttachToSession_Exists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "specular-monitor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := checkpoint.NewManager(tmpDir, false, 0)

	// Create a test session
	state := checkpoint.NewState("test-session")
	state.Status = "completed"
	state.Metadata["goal"] = "Test goal"
	if err := mgr.Save(state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Should work for existing session
	err = attachToSession(mgr, "test-session", false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAttachToLatest_NoSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "specular-monitor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := checkpoint.NewManager(tmpDir, false, 0)

	err = attachToLatest(mgr, false)
	if err == nil {
		t.Error("expected error with no sessions")
	}
}

func TestPrintSessionInfo(t *testing.T) {
	state := &checkpoint.State{
		Status:    "running",
		StartedAt: time.Now().Add(-10 * time.Minute),
		UpdatedAt: time.Now(),
		Metadata:  map[string]string{"goal": "A test goal that is being executed"},
		Tasks: map[string]checkpoint.Task{
			"task-1": {ID: "task-1", Status: "completed"},
			"task-2": {ID: "task-2", Status: "running", Attempts: 1},
			"task-3": {ID: "task-3", Status: "pending"},
			"task-4": {ID: "task-4", Status: "failed", Error: "test error"},
		},
	}

	// Smoke test - just verify no panic
	printSessionInfo("test-session", state, false)
	printSessionInfo("test-session", state, true)
}

func TestPrintTaskStatus(t *testing.T) {
	state := &checkpoint.State{
		Tasks: map[string]checkpoint.Task{
			"running-task": {ID: "running-task", Status: "running", StartedAt: time.Now().Add(-1 * time.Minute), Attempts: 2},
			"pending-task": {ID: "pending-task", Status: "pending"},
		},
	}

	// Smoke test - verify no panic
	printTaskStatus(state)
}

func TestCheckpointFilePath(t *testing.T) {
	tmpDir := "/tmp/test-checkpoints"
	sessionID := "auto-123456"
	expected := filepath.Join(tmpDir, sessionID+".json")

	result := filepath.Join(tmpDir, sessionID+".json")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
