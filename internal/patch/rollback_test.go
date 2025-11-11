package patch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWriterWriteAndReadPatch tests writing and reading patches
func TestWriterWriteAndReadPatch(t *testing.T) {
	patchDir := t.TempDir()
	writer := NewWriter(patchDir)

	patch := &Patch{
		StepID:      "step-1",
		StepType:    "spec:update",
		Timestamp:   time.Now(),
		WorkflowID:  "test-workflow",
		Description: "Test patch",
		Files: []FilePatch{
			{
				Path:       "test.txt",
				Status:     FileStatusModified,
				OldContent: "old",
				NewContent: "new",
				Diff:       "diff content",
				Insertions: 1,
				Deletions:  1,
			},
		},
		FilesChanged: 1,
		Insertions:   1,
		Deletions:    1,
	}

	// Write patch
	patchPath, err := writer.WritePatch(patch)
	if err != nil {
		t.Fatalf("Failed to write patch: %v", err)
	}

	if patchPath == "" {
		t.Error("Expected non-empty patch path")
	}

	// Verify file exists
	if _, err := os.Stat(patchPath); os.IsNotExist(err) {
		t.Errorf("Patch file not created at %s", patchPath)
	}

	// Read patch back
	readPatch, err := writer.ReadPatch("test-workflow", "step-1")
	if err != nil {
		t.Fatalf("Failed to read patch: %v", err)
	}

	// Verify content
	if readPatch.StepID != patch.StepID {
		t.Errorf("Expected StepID %s, got %s", patch.StepID, readPatch.StepID)
	}

	if len(readPatch.Files) != 1 {
		t.Fatalf("Expected 1 file patch, got %d", len(readPatch.Files))
	}

	if readPatch.Files[0].Status != FileStatusModified {
		t.Errorf("Expected status %s, got %s", FileStatusModified, readPatch.Files[0].Status)
	}
}

// TestWriterListPatches tests listing patches
func TestWriterListPatches(t *testing.T) {
	patchDir := t.TempDir()
	writer := NewWriter(patchDir)

	// Write multiple patches
	for i := 1; i <= 3; i++ {
		patch := &Patch{
			StepID:       "step-" + string(rune('0'+i)),
			StepType:     "spec:update",
			Timestamp:    time.Now(),
			WorkflowID:   "test-workflow",
			Description:  "Test patch",
			FilesChanged: i,
		}

		if _, err := writer.WritePatch(patch); err != nil {
			t.Fatalf("Failed to write patch %d: %v", i, err)
		}
	}

	// List patches
	patches, err := writer.ListPatches("test-workflow")
	if err != nil {
		t.Fatalf("Failed to list patches: %v", err)
	}

	if len(patches) != 3 {
		t.Fatalf("Expected 3 patches, got %d", len(patches))
	}
}

// TestWriterPatchExists tests patch existence check
func TestWriterPatchExists(t *testing.T) {
	patchDir := t.TempDir()
	writer := NewWriter(patchDir)

	// Check non-existent patch
	if writer.PatchExists("test-workflow", "step-1") {
		t.Error("Patch should not exist")
	}

	// Write patch
	patch := &Patch{
		StepID:     "step-1",
		WorkflowID: "test-workflow",
	}

	if _, err := writer.WritePatch(patch); err != nil {
		t.Fatalf("Failed to write patch: %v", err)
	}

	// Check existing patch
	if !writer.PatchExists("test-workflow", "step-1") {
		t.Error("Patch should exist")
	}
}

// TestWriterDeletePatch tests patch deletion
func TestWriterDeletePatch(t *testing.T) {
	patchDir := t.TempDir()
	writer := NewWriter(patchDir)

	// Write patch
	patch := &Patch{
		StepID:     "step-1",
		WorkflowID: "test-workflow",
	}

	if _, err := writer.WritePatch(patch); err != nil {
		t.Fatalf("Failed to write patch: %v", err)
	}

	// Verify exists
	if !writer.PatchExists("test-workflow", "step-1") {
		t.Error("Patch should exist before deletion")
	}

	// Delete patch
	if err := writer.DeletePatch("test-workflow", "step-1"); err != nil {
		t.Fatalf("Failed to delete patch: %v", err)
	}

	// Verify deleted
	if writer.PatchExists("test-workflow", "step-1") {
		t.Error("Patch should not exist after deletion")
	}
}

// TestRollbackAddedFile tests rolling back an added file
func TestRollbackAddedFile(t *testing.T) {
	workingDir := t.TempDir()
	patchDir := t.TempDir()
	writer := NewWriter(patchDir)
	rollback := NewRollback(workingDir, patchDir)

	// Create a file
	testFile := filepath.Join(workingDir, "added.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create patch for added file
	patch := &Patch{
		StepID:     "step-1",
		WorkflowID: "test-workflow",
		Files: []FilePatch{
			{
				Path:       "added.txt",
				Status:     FileStatusAdded,
				NewContent: "content",
			},
		},
	}

	if _, err := writer.WritePatch(patch); err != nil {
		t.Fatalf("Failed to write patch: %v", err)
	}

	// Rollback
	if err := rollback.RollbackStep("test-workflow", "step-1"); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}
}

// TestRollbackModifiedFile tests rolling back a modified file
func TestRollbackModifiedFile(t *testing.T) {
	workingDir := t.TempDir()
	patchDir := t.TempDir()
	writer := NewWriter(patchDir)
	rollback := NewRollback(workingDir, patchDir)

	// Create a file with new content
	testFile := filepath.Join(workingDir, "modified.txt")
	newContent := "new content"
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create patch for modified file
	oldContent := "old content"
	patch := &Patch{
		StepID:     "step-1",
		WorkflowID: "test-workflow",
		Files: []FilePatch{
			{
				Path:       "modified.txt",
				Status:     FileStatusModified,
				OldContent: oldContent,
				NewContent: newContent,
			},
		},
	}

	if _, err := writer.WritePatch(patch); err != nil {
		t.Fatalf("Failed to write patch: %v", err)
	}

	// Rollback
	if err := rollback.RollbackStep("test-workflow", "step-1"); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify file was restored to old content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != oldContent {
		t.Errorf("Expected content %q, got %q", oldContent, string(content))
	}
}

// TestRollbackDeletedFile tests rolling back a deleted file
func TestRollbackDeletedFile(t *testing.T) {
	workingDir := t.TempDir()
	patchDir := t.TempDir()
	writer := NewWriter(patchDir)
	rollback := NewRollback(workingDir, patchDir)

	testFile := filepath.Join(workingDir, "deleted.txt")

	// Create patch for deleted file
	oldContent := "old content"
	patch := &Patch{
		StepID:     "step-1",
		WorkflowID: "test-workflow",
		Files: []FilePatch{
			{
				Path:       "deleted.txt",
				Status:     FileStatusDeleted,
				OldContent: oldContent,
			},
		},
	}

	if _, err := writer.WritePatch(patch); err != nil {
		t.Fatalf("Failed to write patch: %v", err)
	}

	// Rollback
	if err := rollback.RollbackStep("test-workflow", "step-1"); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify file was recreated
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != oldContent {
		t.Errorf("Expected content %q, got %q", oldContent, string(content))
	}
}

// TestRollbackMultipleSteps tests rolling back multiple steps
func TestRollbackMultipleSteps(t *testing.T) {
	workingDir := t.TempDir()
	patchDir := t.TempDir()
	writer := NewWriter(patchDir)
	rollback := NewRollback(workingDir, patchDir)

	// Create multiple patches
	for i := 1; i <= 3; i++ {
		testFile := filepath.Join(workingDir, "file.txt")
		content := "content" + string(rune('0'+i))
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file for step %d: %v", i, err)
		}

		patch := &Patch{
			StepID:     "step-" + string(rune('0'+i)),
			WorkflowID: "test-workflow",
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
			Files: []FilePatch{
				{
					Path:       "file.txt",
					Status:     FileStatusModified,
					OldContent: "content" + string(rune('0'+i-1)),
					NewContent: content,
				},
			},
		}

		if _, err := writer.WritePatch(patch); err != nil {
			t.Fatalf("Failed to write patch %d: %v", i, err)
		}
	}

	// Rollback to step-1
	result, err := rollback.RollbackToStep("test-workflow", "step-1")
	if err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	if !result.Success {
		t.Error("Expected rollback to succeed")
	}

	if result.StepsReverted != 2 {
		t.Errorf("Expected 2 steps reverted, got %d", result.StepsReverted)
	}
}

// TestVerifyRollbackSafety tests rollback safety verification
func TestVerifyRollbackSafety(t *testing.T) {
	workingDir := t.TempDir()
	patchDir := t.TempDir()
	writer := NewWriter(patchDir)
	rollback := NewRollback(workingDir, patchDir)

	// Create file with expected content
	testFile := filepath.Join(workingDir, "safe.txt")
	expectedContent := "expected content"
	if err := os.WriteFile(testFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create patch
	patch := &Patch{
		StepID:     "step-1",
		WorkflowID: "test-workflow",
		Files: []FilePatch{
			{
				Path:       "safe.txt",
				Status:     FileStatusModified,
				OldContent: "old content",
				NewContent: expectedContent,
			},
		},
	}

	if _, err := writer.WritePatch(patch); err != nil {
		t.Fatalf("Failed to write patch: %v", err)
	}

	// Verify safety (should be safe)
	safe, warnings, err := rollback.VerifyRollbackSafety("test-workflow", "step-1")
	if err != nil {
		t.Fatalf("Failed to verify safety: %v", err)
	}

	if !safe {
		t.Error("Expected rollback to be safe")
	}

	if len(warnings) != 0 {
		t.Errorf("Expected no warnings, got %d", len(warnings))
	}

	// Modify file to create conflict
	conflictContent := "conflict content"
	if err := os.WriteFile(testFile, []byte(conflictContent), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Verify safety again (should have warnings)
	safe, warnings, err = rollback.VerifyRollbackSafety("test-workflow", "step-1")
	if err != nil {
		t.Fatalf("Failed to verify safety: %v", err)
	}

	if safe {
		t.Error("Expected rollback to be unsafe")
	}

	if len(warnings) == 0 {
		t.Error("Expected warnings about modified file")
	}
}
