package patch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiffGeneratorAddedFile tests diff generation for added files
func TestDiffGeneratorAddedFile(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewDiffGenerator(tmpDir)

	// Create a new file
	testFile := filepath.Join(tmpDir, "new.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate patch with empty snapshot
	snapshot := make(map[string]string)
	patch, err := gen.GeneratePatch("step-1", "spec:update", "test-workflow", "Added file", snapshot)
	if err != nil {
		t.Fatalf("Failed to generate patch: %v", err)
	}

	if len(patch.Files) != 1 {
		t.Fatalf("Expected 1 file patch, got %d", len(patch.Files))
	}

	filePatch := patch.Files[0]
	if filePatch.Status != FileStatusAdded {
		t.Errorf("Expected status %s, got %s", FileStatusAdded, filePatch.Status)
	}

	if filePatch.Path != "new.txt" {
		t.Errorf("Expected path new.txt, got %s", filePatch.Path)
	}

	if filePatch.NewContent != content {
		t.Errorf("Expected content %q, got %q", content, filePatch.NewContent)
	}

	if filePatch.Insertions != 3 {
		t.Errorf("Expected 3 insertions, got %d", filePatch.Insertions)
	}

	if filePatch.Deletions != 0 {
		t.Errorf("Expected 0 deletions, got %d", filePatch.Deletions)
	}
}

// TestDiffGeneratorModifiedFile tests diff generation for modified files
func TestDiffGeneratorModifiedFile(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewDiffGenerator(tmpDir)

	// Create file with initial content
	testFile := filepath.Join(tmpDir, "modified.txt")
	oldContent := "line1\nline2\nline3\n"
	newContent := "line1\nline2 modified\nline3\nline4\n"

	// Snapshot before modification
	snapshot := map[string]string{
		"modified.txt": oldContent,
	}

	// Write modified content
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Generate patch
	patch, err := gen.GeneratePatch("step-1", "spec:update", "test-workflow", "Modified file", snapshot)
	if err != nil {
		t.Fatalf("Failed to generate patch: %v", err)
	}

	if len(patch.Files) != 1 {
		t.Fatalf("Expected 1 file patch, got %d", len(patch.Files))
	}

	filePatch := patch.Files[0]
	if filePatch.Status != FileStatusModified {
		t.Errorf("Expected status %s, got %s", FileStatusModified, filePatch.Status)
	}

	if filePatch.OldContent != oldContent {
		t.Errorf("Expected old content %q, got %q", oldContent, filePatch.OldContent)
	}

	if filePatch.NewContent != newContent {
		t.Errorf("Expected new content %q, got %q", newContent, filePatch.NewContent)
	}

	// Check that diff contains expected markers
	if !strings.Contains(filePatch.Diff, "--- a/modified.txt") {
		t.Error("Diff should contain old file marker")
	}

	if !strings.Contains(filePatch.Diff, "+++ b/modified.txt") {
		t.Error("Diff should contain new file marker")
	}
}

// TestDiffGeneratorDeletedFile tests diff generation for deleted files
func TestDiffGeneratorDeletedFile(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewDiffGenerator(tmpDir)

	oldContent := "line1\nline2\nline3\n"

	// Snapshot with file that will be deleted
	snapshot := map[string]string{
		"deleted.txt": oldContent,
	}

	// Generate patch (file doesn't exist in tmpDir)
	patch, err := gen.GeneratePatch("step-1", "spec:update", "test-workflow", "Deleted file", snapshot)
	if err != nil {
		t.Fatalf("Failed to generate patch: %v", err)
	}

	if len(patch.Files) != 1 {
		t.Fatalf("Expected 1 file patch, got %d", len(patch.Files))
	}

	filePatch := patch.Files[0]
	if filePatch.Status != FileStatusDeleted {
		t.Errorf("Expected status %s, got %s", FileStatusDeleted, filePatch.Status)
	}

	if filePatch.OldContent != oldContent {
		t.Errorf("Expected old content %q, got %q", oldContent, filePatch.OldContent)
	}

	if filePatch.Deletions != 3 {
		t.Errorf("Expected 3 deletions, got %d", filePatch.Deletions)
	}

	if filePatch.Insertions != 0 {
		t.Errorf("Expected 0 insertions, got %d", filePatch.Insertions)
	}
}

// TestDiffGeneratorMultipleFiles tests diff generation with multiple changes
func TestDiffGeneratorMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewDiffGenerator(tmpDir)

	// Create files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	// file3.txt intentionally not created (will be detected as deleted)

	if err := os.WriteFile(file1, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	if err := os.WriteFile(file2, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Snapshot
	snapshot := map[string]string{
		"file2.txt": "old content",
		"file3.txt": "deleted content",
	}

	// Generate patch
	patch, err := gen.GeneratePatch("step-1", "spec:update", "test-workflow", "Multiple changes", snapshot)
	if err != nil {
		t.Fatalf("Failed to generate patch: %v", err)
	}

	if len(patch.Files) != 3 {
		t.Fatalf("Expected 3 file patches, got %d", len(patch.Files))
	}

	// Verify statistics
	if patch.FilesChanged != 3 {
		t.Errorf("Expected 3 files changed, got %d", patch.FilesChanged)
	}

	// Check each file status
	statusCount := make(map[FileStatus]int)
	for _, filePatch := range patch.Files {
		statusCount[filePatch.Status]++
	}

	if statusCount[FileStatusAdded] != 1 {
		t.Errorf("Expected 1 added file, got %d", statusCount[FileStatusAdded])
	}

	if statusCount[FileStatusModified] != 1 {
		t.Errorf("Expected 1 modified file, got %d", statusCount[FileStatusModified])
	}

	if statusCount[FileStatusDeleted] != 1 {
		t.Errorf("Expected 1 deleted file, got %d", statusCount[FileStatusDeleted])
	}
}

// TestDiffGeneratorEmptyPatch tests when no changes are made
func TestDiffGeneratorEmptyPatch(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewDiffGenerator(tmpDir)

	// Create file
	testFile := filepath.Join(tmpDir, "unchanged.txt")
	content := "content"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Snapshot with same content
	snapshot := map[string]string{
		"unchanged.txt": content,
	}

	// Generate patch
	patch, err := gen.GeneratePatch("step-1", "spec:update", "test-workflow", "No changes", snapshot)
	if err != nil {
		t.Fatalf("Failed to generate patch: %v", err)
	}

	if !patch.IsEmpty() {
		t.Error("Expected empty patch")
	}

	if patch.FilesChanged != 0 {
		t.Errorf("Expected 0 files changed, got %d", patch.FilesChanged)
	}
}

// TestCountChanges tests insertion and deletion counting
func TestCountChanges(t *testing.T) {
	gen := NewDiffGenerator("")

	tests := []struct {
		name       string
		oldContent string
		newContent string
		insertions int
		deletions  int
	}{
		{
			name:       "add lines",
			oldContent: "line1\n",
			newContent: "line1\nline2\nline3\n",
			insertions: 2,
			deletions:  0,
		},
		{
			name:       "remove lines",
			oldContent: "line1\nline2\nline3\n",
			newContent: "line1\n",
			insertions: 0,
			deletions:  2,
		},
		{
			name:       "modify lines",
			oldContent: "line1\nline2\nline3\n",
			newContent: "line1\nmodified\nline3\n",
			insertions: 1,
			deletions:  1,
		},
		{
			name:       "empty to content",
			oldContent: "",
			newContent: "line1\nline2\n",
			insertions: 2,
			deletions:  0,
		},
		{
			name:       "content to empty",
			oldContent: "line1\nline2\n",
			newContent: "",
			insertions: 0,
			deletions:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insertions, deletions := gen.countChanges(tt.oldContent, tt.newContent)

			if insertions != tt.insertions {
				t.Errorf("Expected %d insertions, got %d", tt.insertions, insertions)
			}

			if deletions != tt.deletions {
				t.Errorf("Expected %d deletions, got %d", tt.deletions, deletions)
			}
		})
	}
}

// TestGenerateUnifiedDiff tests unified diff generation
func TestGenerateUnifiedDiff(t *testing.T) {
	gen := NewDiffGenerator("")

	oldContent := "line1\nline2\nline3\n"
	newContent := "line1\nline2 modified\nline3\nline4\n"

	diff := gen.generateUnifiedDiff("test.txt", oldContent, newContent)

	// Check diff format
	if !strings.Contains(diff, "--- a/test.txt") {
		t.Error("Diff should contain old file marker")
	}

	if !strings.Contains(diff, "+++ b/test.txt") {
		t.Error("Diff should contain new file marker")
	}

	if !strings.Contains(diff, "@@") {
		t.Error("Diff should contain hunk header")
	}

	// The diff library does character-level diffs, so changes are more granular
	// Check that it contains markers for modifications
	if !strings.Contains(diff, "+ modified") && !strings.Contains(diff, "+line2 modified") {
		t.Errorf("Diff should contain modification marker, got:\n%s", diff)
	}

	if !strings.Contains(diff, "+line4") {
		t.Errorf("Diff should contain new line, got:\n%s", diff)
	}

	// Verify it's a valid unified diff format
	if !strings.Contains(diff, " line1") {
		t.Error("Diff should contain context lines")
	}
}
