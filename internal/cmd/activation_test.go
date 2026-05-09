package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActivationMarkerRoundTrip(t *testing.T) {
	specDir := filepath.Join(t.TempDir(), ".specular")
	start := time.Now().Add(-90 * time.Second).Truncate(time.Second)

	if err := writeActivationStart(specDir, start); err != nil {
		t.Fatalf("writeActivationStart: %v", err)
	}

	marker, err := readActivationMarker(specDir)
	if err != nil {
		t.Fatalf("readActivationMarker: %v", err)
	}
	if !marker.StartedAt.Equal(start) {
		t.Errorf("StartedAt mismatch: got %v want %v", marker.StartedAt, start)
	}
	if !marker.InitCompleteAt.IsZero() {
		t.Errorf("InitCompleteAt should be zero before completion, got %v", marker.InitCompleteAt)
	}
}

func TestMarkActivationInitCompletePreservesStart(t *testing.T) {
	specDir := filepath.Join(t.TempDir(), ".specular")
	start := time.Now().Add(-2 * time.Minute).Truncate(time.Second)

	if err := writeActivationStart(specDir, start); err != nil {
		t.Fatalf("writeActivationStart: %v", err)
	}

	completed := time.Now().Truncate(time.Second)
	if err := markActivationInitComplete(specDir, completed); err != nil {
		t.Fatalf("markActivationInitComplete: %v", err)
	}

	marker, err := readActivationMarker(specDir)
	if err != nil {
		t.Fatalf("readActivationMarker: %v", err)
	}
	if !marker.StartedAt.Equal(start) {
		t.Errorf("StartedAt was overwritten: got %v want %v", marker.StartedAt, start)
	}
	if !marker.InitCompleteAt.Equal(completed) {
		t.Errorf("InitCompleteAt mismatch: got %v want %v", marker.InitCompleteAt, completed)
	}
}

func TestMarkActivationInitCompleteCreatesIfMissing(t *testing.T) {
	specDir := filepath.Join(t.TempDir(), ".specular")
	completed := time.Now().Truncate(time.Second)

	if err := markActivationInitComplete(specDir, completed); err != nil {
		t.Fatalf("markActivationInitComplete: %v", err)
	}

	marker, err := readActivationMarker(specDir)
	if err != nil {
		t.Fatalf("readActivationMarker: %v", err)
	}
	if !marker.StartedAt.Equal(completed) {
		t.Errorf("StartedAt should default to completed when marker missing: got %v", marker.StartedAt)
	}
	if !marker.InitCompleteAt.Equal(completed) {
		t.Errorf("InitCompleteAt mismatch: got %v want %v", marker.InitCompleteAt, completed)
	}
}

func TestRecordFirstSuccessIfPendingPersistsTimestamp(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(chdir(t, tmp))

	specDir := filepath.Join(tmp, ".specular")
	start := time.Now().Add(-30 * time.Second)
	if err := writeActivationStart(specDir, start); err != nil {
		t.Fatalf("writeActivationStart: %v", err)
	}

	recordFirstSuccessIfPending(t.Context(), "plan")

	marker, err := readActivationMarker(specDir)
	if err != nil {
		t.Fatalf("readActivationMarker: %v", err)
	}
	if marker.FirstSuccessAt.IsZero() {
		t.Fatalf("FirstSuccessAt was not recorded")
	}
}

func TestRecordFirstSuccessIfPendingSkipsExcludedCommand(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(chdir(t, tmp))

	specDir := filepath.Join(tmp, ".specular")
	start := time.Now().Add(-30 * time.Second)
	if err := writeActivationStart(specDir, start); err != nil {
		t.Fatalf("writeActivationStart: %v", err)
	}

	recordFirstSuccessIfPending(t.Context(), "init")
	recordFirstSuccessIfPending(t.Context(), "help")
	recordFirstSuccessIfPending(t.Context(), "version")

	marker, err := readActivationMarker(specDir)
	if err != nil {
		t.Fatalf("readActivationMarker: %v", err)
	}
	if !marker.FirstSuccessAt.IsZero() {
		t.Fatalf("FirstSuccessAt should remain zero for excluded commands, got %v", marker.FirstSuccessAt)
	}
}

func TestRecordFirstSuccessIsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(chdir(t, tmp))

	specDir := filepath.Join(tmp, ".specular")
	start := time.Now().Add(-30 * time.Second)
	if err := writeActivationStart(specDir, start); err != nil {
		t.Fatalf("writeActivationStart: %v", err)
	}

	recordFirstSuccessIfPending(t.Context(), "plan")
	first, err := readActivationMarker(specDir)
	if err != nil {
		t.Fatalf("read after first call: %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	recordFirstSuccessIfPending(t.Context(), "build")
	second, err := readActivationMarker(specDir)
	if err != nil {
		t.Fatalf("read after second call: %v", err)
	}

	if !first.FirstSuccessAt.Equal(second.FirstSuccessAt) {
		t.Errorf("FirstSuccessAt should not change on subsequent calls: first=%v second=%v",
			first.FirstSuccessAt, second.FirstSuccessAt)
	}
}

func TestRecordFirstSuccessIfPendingNoMarker(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(chdir(t, tmp))

	// No .specular directory at all — should be a silent no-op.
	recordFirstSuccessIfPending(t.Context(), "plan")
}

func TestRecordFirstSuccessIfPendingCorruptMarker(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(chdir(t, tmp))

	specDir := filepath.Join(tmp, ".specular")
	if err := os.MkdirAll(specDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(activationMarkerPath(specDir), []byte("not json"), 0o600); err != nil {
		t.Fatalf("write corrupt marker: %v", err)
	}

	// Must not panic and must not write a partial marker.
	recordFirstSuccessIfPending(t.Context(), "plan")

	data, err := os.ReadFile(activationMarkerPath(specDir))
	if err != nil {
		t.Fatalf("read after corrupt: %v", err)
	}
	if string(data) != "not json" {
		t.Errorf("corrupt marker should be left untouched, got %q", string(data))
	}
}

func TestActivationMarkerJSONStable(t *testing.T) {
	specDir := filepath.Join(t.TempDir(), ".specular")
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := writeActivationStart(specDir, start); err != nil {
		t.Fatalf("writeActivationStart: %v", err)
	}

	data, err := os.ReadFile(activationMarkerPath(specDir))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["started_at"]; !ok {
		t.Errorf("marker is missing started_at field: %s", string(data))
	}
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return func() { _ = os.Chdir(prev) }
}
