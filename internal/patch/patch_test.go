package patch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPatchSerialization tests patch JSON serialization
func TestPatchSerialization(t *testing.T) {
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
				Diff:       "--- a/test.txt\n+++ b/test.txt\n@@ -1,1 +1,1 @@\n-old\n+new\n",
				Insertions: 1,
				Deletions:  1,
			},
		},
		FilesChanged: 1,
		Insertions:   1,
		Deletions:    1,
	}

	// Serialize
	jsonData, err := patch.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize patch: %v", err)
	}

	// Deserialize
	parsed, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize patch: %v", err)
	}

	// Verify
	if parsed.StepID != patch.StepID {
		t.Errorf("Expected StepID %s, got %s", patch.StepID, parsed.StepID)
	}

	if parsed.FilesChanged != 1 {
		t.Errorf("Expected 1 file changed, got %d", parsed.FilesChanged)
	}

	if len(parsed.Files) != 1 {
		t.Fatalf("Expected 1 file patch, got %d", len(parsed.Files))
	}

	if parsed.Files[0].Status != FileStatusModified {
		t.Errorf("Expected status %s, got %s", FileStatusModified, parsed.Files[0].Status)
	}
}

// TestPatchMetadata tests metadata extraction
func TestPatchMetadata(t *testing.T) {
	patch := &Patch{
		StepID:       "step-1",
		StepType:     "spec:update",
		Timestamp:    time.Now(),
		WorkflowID:   "test-workflow",
		Description:  "Test patch",
		FilesChanged: 3,
		Insertions:   10,
		Deletions:    5,
	}

	metadata := patch.GetMetadata()

	if metadata.StepID != patch.StepID {
		t.Errorf("Expected StepID %s, got %s", patch.StepID, metadata.StepID)
	}

	if metadata.FilesChanged != 3 {
		t.Errorf("Expected 3 files changed, got %d", metadata.FilesChanged)
	}

	if metadata.Insertions != 10 {
		t.Errorf("Expected 10 insertions, got %d", metadata.Insertions)
	}

	if metadata.Deletions != 5 {
		t.Errorf("Expected 5 deletions, got %d", metadata.Deletions)
	}
}

// TestPatchIsEmpty tests empty patch detection
func TestPatchIsEmpty(t *testing.T) {
	emptyPatch := &Patch{
		Files: []FilePatch{},
	}

	if !emptyPatch.IsEmpty() {
		t.Error("Expected patch to be empty")
	}

	nonEmptyPatch := &Patch{
		Files: []FilePatch{
			{Path: "test.txt", Status: FileStatusAdded},
		},
	}

	if nonEmptyPatch.IsEmpty() {
		t.Error("Expected patch to not be empty")
	}
}

// TestCalculateStats tests statistics calculation
func TestCalculateStats(t *testing.T) {
	patch := &Patch{
		Files: []FilePatch{
			{
				Path:       "file1.txt",
				Status:     FileStatusModified,
				Insertions: 5,
				Deletions:  3,
			},
			{
				Path:       "file2.txt",
				Status:     FileStatusAdded,
				Insertions: 10,
				Deletions:  0,
			},
			{
				Path:       "file3.txt",
				Status:     FileStatusDeleted,
				Insertions: 0,
				Deletions:  7,
			},
		},
	}

	patch.CalculateStats()

	if patch.FilesChanged != 3 {
		t.Errorf("Expected 3 files changed, got %d", patch.FilesChanged)
	}

	if patch.Insertions != 15 {
		t.Errorf("Expected 15 insertions, got %d", patch.Insertions)
	}

	if patch.Deletions != 10 {
		t.Errorf("Expected 10 deletions, got %d", patch.Deletions)
	}
}

// TestFileStatuses tests different file status types
func TestFileStatuses(t *testing.T) {
	statuses := []FileStatus{
		FileStatusAdded,
		FileStatusModified,
		FileStatusDeleted,
		FileStatusRenamed,
	}

	for _, status := range statuses {
		filePatch := FilePatch{
			Path:   "test.txt",
			Status: status,
		}

		// Serialize
		data, err := json.Marshal(filePatch)
		if err != nil {
			t.Fatalf("Failed to marshal file patch with status %s: %v", status, err)
		}

		// Deserialize
		var parsed FilePatch
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Failed to unmarshal file patch: %v", err)
		}

		if parsed.Status != status {
			t.Errorf("Expected status %s, got %s", status, parsed.Status)
		}
	}
}

// TestCaptureFileSnapshot tests file snapshot capturing
func TestCaptureFileSnapshot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")

	if err := os.WriteFile(testFile1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := os.WriteFile(testFile2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Capture snapshot
	snapshot, err := CaptureFileSnapshot(tmpDir, []string{"test1.txt", "test2.txt"})
	if err != nil {
		t.Fatalf("Failed to capture snapshot: %v", err)
	}

	if len(snapshot) != 2 {
		t.Fatalf("Expected 2 files in snapshot, got %d", len(snapshot))
	}

	if snapshot["test1.txt"] != "content1" {
		t.Errorf("Expected content1, got %s", snapshot["test1.txt"])
	}

	if snapshot["test2.txt"] != "content2" {
		t.Errorf("Expected content2, got %s", snapshot["test2.txt"])
	}
}

// TestCaptureDirectorySnapshot tests directory snapshot capturing
func TestCaptureDirectorySnapshot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	subDir := filepath.Join(tmpDir, "subdir")
	testFile3 := filepath.Join(subDir, "test3.txt")

	if err := os.WriteFile(testFile1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := os.WriteFile(testFile2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	if err := os.WriteFile(testFile3, []byte("content3"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Capture snapshot
	snapshot, err := CaptureDirectorySnapshot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to capture snapshot: %v", err)
	}

	if len(snapshot) != 3 {
		t.Fatalf("Expected 3 files in snapshot, got %d", len(snapshot))
	}

	if snapshot["test1.txt"] != "content1" {
		t.Errorf("Expected content1, got %s", snapshot["test1.txt"])
	}

	if snapshot["subdir/test3.txt"] != "content3" {
		t.Errorf("Expected content3, got %s", snapshot["subdir/test3.txt"])
	}
}
