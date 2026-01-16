package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFileWriter(t *testing.T) {
	w := NewFileWriter("/tmp/test", true, false)

	if w.BaseDir != "/tmp/test" {
		t.Errorf("BaseDir = %q, want %q", w.BaseDir, "/tmp/test")
	}
	if !w.Verbose {
		t.Error("Verbose should be true")
	}
	if w.DryRun {
		t.Error("DryRun should be false")
	}
}

func TestFileWriter_WriteFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	w := NewFileWriter(tmpDir, false, false)

	file := &GeneratedFile{
		Path:    "subdir/test.go",
		Content: "package test\n",
	}

	written, err := w.WriteFile(file)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !written {
		t.Error("WriteFile() should return true when file is written")
	}

	// Verify file exists
	fullPath := filepath.Join(tmpDir, "subdir/test.go")
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(content) != "package test\n" {
		t.Errorf("File content = %q, want %q", string(content), "package test\n")
	}
}

func TestFileWriter_WriteFile_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	w := NewFileWriter(tmpDir, false, true) // DryRun = true

	file := &GeneratedFile{
		Path:    "test.go",
		Content: "package test\n",
	}

	written, err := w.WriteFile(file)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if written {
		t.Error("WriteFile() should return false in dry-run mode")
	}

	// Verify file does NOT exist
	fullPath := filepath.Join(tmpDir, "test.go")
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("File should not exist in dry-run mode")
	}
}

func TestFileWriter_WriteFile_EmptyPath(t *testing.T) {
	w := NewFileWriter("/tmp", false, false)

	file := &GeneratedFile{
		Path:    "",
		Content: "content",
	}

	_, err := w.WriteFile(file)
	if err == nil {
		t.Error("WriteFile() should error on empty path")
	}
}

func TestFileWriter_WriteFile_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	w := NewFileWriter(tmpDir, false, false)

	// Attempt path traversal
	file := &GeneratedFile{
		Path:    "../../../etc/passwd",
		Content: "malicious content",
	}

	_, err := w.WriteFile(file)
	if err == nil {
		t.Error("WriteFile() should error on path traversal attempt")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Errorf("Error should mention path traversal, got: %v", err)
	}
}

func TestFileWriter_WriteFiles(t *testing.T) {
	tmpDir := t.TempDir()
	w := NewFileWriter(tmpDir, false, false)

	files := []GeneratedFile{
		{Path: "file1.go", Content: "package one\n"},
		{Path: "subdir/file2.go", Content: "package two\n"},
		{Path: "deep/nested/file3.go", Content: "package three\n"},
	}

	results, err := w.WriteFiles(files)
	if err != nil {
		t.Fatalf("WriteFiles() error = %v", err)
	}

	if len(results) != 3 {
		t.Errorf("WriteFiles() returned %d results, want 3", len(results))
	}

	for _, f := range results {
		if !f.Written {
			t.Errorf("File %s was not marked as written", f.Path)
		}

		// Verify file exists
		fullPath := filepath.Join(tmpDir, f.Path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("File %s does not exist", f.Path)
		}
	}
}

func TestFileWriter_WrittenFiles(t *testing.T) {
	tmpDir := t.TempDir()
	w := NewFileWriter(tmpDir, false, false)

	files := []GeneratedFile{
		{Path: "file1.go", Content: "package one\n"},
		{Path: "file2.go", Content: "package two\n"},
	}

	_, err := w.WriteFiles(files)
	if err != nil {
		t.Fatalf("WriteFiles() error = %v", err)
	}

	written := w.WrittenFiles()
	if len(written) != 2 {
		t.Errorf("WrittenFiles() returned %d files, want 2", len(written))
	}
}

func TestFileWriter_CleanupWritten(t *testing.T) {
	tmpDir := t.TempDir()
	w := NewFileWriter(tmpDir, false, false)

	files := []GeneratedFile{
		{Path: "file1.go", Content: "package one\n"},
		{Path: "file2.go", Content: "package two\n"},
	}

	_, err := w.WriteFiles(files)
	if err != nil {
		t.Fatalf("WriteFiles() error = %v", err)
	}

	// Verify files exist
	for _, f := range files {
		fullPath := filepath.Join(tmpDir, f.Path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Fatalf("File %s should exist before cleanup", f.Path)
		}
	}

	// Cleanup
	if err := w.CleanupWritten(); err != nil {
		t.Fatalf("CleanupWritten() error = %v", err)
	}

	// Verify files are removed
	for _, f := range files {
		fullPath := filepath.Join(tmpDir, f.Path)
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Errorf("File %s should not exist after cleanup", f.Path)
		}
	}

	// WrittenFiles should be empty
	if len(w.WrittenFiles()) != 0 {
		t.Error("WrittenFiles() should be empty after cleanup")
	}
}

func TestFileWriter_EnsureBaseDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new", "nested", "dir")

	w := NewFileWriter(newDir, false, false)

	if err := w.EnsureBaseDir(); err != nil {
		t.Fatalf("EnsureBaseDir() error = %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("Base directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Base path is not a directory")
	}
}

func TestFileWriter_EnsureBaseDir_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "should", "not", "exist")

	w := NewFileWriter(newDir, false, true) // DryRun = true

	if err := w.EnsureBaseDir(); err != nil {
		t.Fatalf("EnsureBaseDir() error = %v", err)
	}

	// Directory should NOT exist in dry-run mode
	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		t.Error("Directory should not exist in dry-run mode")
	}
}

func TestIsSubPath(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{
			name:   "child is subpath",
			parent: "/home/user/project",
			child:  "/home/user/project/src/main.go",
			want:   true,
		},
		{
			name:   "child equals parent",
			parent: "/home/user/project",
			child:  "/home/user/project",
			want:   true,
		},
		{
			name:   "child is outside parent",
			parent: "/home/user/project",
			child:  "/home/user/other/file.go",
			want:   false,
		},
		{
			name:   "path traversal attempt",
			parent: "/home/user/project",
			child:  "/home/user/project/../../../etc/passwd",
			want:   false,
		},
		{
			name:   "similar prefix but not subpath",
			parent: "/home/user/project",
			child:  "/home/user/project-other/file.go",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubPath(tt.parent, tt.child)
			if got != tt.want {
				t.Errorf("isSubPath(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
			}
		})
	}
}

func TestGeneratedFile_ToRecord(t *testing.T) {
	file := GeneratedFile{
		Path:     "cmd/main.go",
		Content:  "package main",
		Language: "go",
		Hash:     "abc123",
		Size:     12,
		Written:  true,
	}

	record := file.ToRecord()

	if record.Path != file.Path {
		t.Errorf("ToRecord().Path = %q, want %q", record.Path, file.Path)
	}
	if record.Hash != file.Hash {
		t.Errorf("ToRecord().Hash = %q, want %q", record.Hash, file.Hash)
	}
	if record.Language != file.Language {
		t.Errorf("ToRecord().Language = %q, want %q", record.Language, file.Language)
	}
	if record.Size != file.Size {
		t.Errorf("ToRecord().Size = %d, want %d", record.Size, file.Size)
	}
}

func TestWriteGeneratedFile_Convenience(t *testing.T) {
	tmpDir := t.TempDir()

	err := WriteGeneratedFile(tmpDir, "test.txt", "hello world", false)
	if err != nil {
		t.Fatalf("WriteGeneratedFile() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("File content = %q, want %q", string(content), "hello world")
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !FileExists(tmpDir, "exists.txt") {
		t.Error("FileExists() should return true for existing file")
	}

	if FileExists(tmpDir, "nonexistent.txt") {
		t.Error("FileExists() should return false for non-existing file")
	}
}

func TestReadExistingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	expectedContent := "test content"
	if err := os.WriteFile(testFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	content, err := ReadExistingFile(tmpDir, "test.txt")
	if err != nil {
		t.Fatalf("ReadExistingFile() error = %v", err)
	}
	if content != expectedContent {
		t.Errorf("ReadExistingFile() = %q, want %q", content, expectedContent)
	}

	// Non-existing file
	_, err = ReadExistingFile(tmpDir, "nonexistent.txt")
	if err == nil {
		t.Error("ReadExistingFile() should error for non-existing file")
	}
}
