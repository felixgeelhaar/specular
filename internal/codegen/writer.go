package codegen

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileWriter writes generated files to disk
type FileWriter struct {
	// BaseDir is the base directory for all output
	BaseDir string

	// Verbose enables detailed logging
	Verbose bool

	// DryRun prevents actual file writes
	DryRun bool

	// written tracks files that have been written
	written []string
}

// NewFileWriter creates a new file writer
func NewFileWriter(baseDir string, verbose, dryRun bool) *FileWriter {
	return &FileWriter{
		BaseDir: baseDir,
		Verbose: verbose,
		DryRun:  dryRun,
		written: []string{},
	}
}

// WriteFiles writes all generated files to disk
func (w *FileWriter) WriteFiles(files []GeneratedFile) ([]GeneratedFile, error) {
	results := make([]GeneratedFile, len(files))

	for i, file := range files {
		results[i] = file

		written, err := w.WriteFile(&results[i])
		if err != nil {
			return results, fmt.Errorf("failed to write %s: %w", file.Path, err)
		}

		results[i].Written = written
	}

	return results, nil
}

// WriteFile writes a single file to disk
func (w *FileWriter) WriteFile(file *GeneratedFile) (bool, error) {
	if file.Path == "" {
		return false, fmt.Errorf("file path is required")
	}

	// Compute full path
	fullPath := filepath.Join(w.BaseDir, file.Path)

	// Normalize the path to prevent directory traversal
	fullPath = filepath.Clean(fullPath)

	// Security check: ensure path is within base directory
	if !isSubPath(w.BaseDir, fullPath) {
		return false, fmt.Errorf("path traversal detected: %s is outside %s", fullPath, w.BaseDir)
	}

	if w.DryRun {
		if w.Verbose {
			fmt.Printf("  [dry-run] Would write: %s (%d bytes)\n", fullPath, len(file.Content))
		}
		return false, nil
	}

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(file.Content), 0644); err != nil {
		return false, fmt.Errorf("failed to write file: %w", err)
	}

	w.written = append(w.written, fullPath)

	if w.Verbose {
		fmt.Printf("  ✓ Written: %s (%d bytes)\n", fullPath, len(file.Content))
	}

	return true, nil
}

// WrittenFiles returns the list of files that were written
func (w *FileWriter) WrittenFiles() []string {
	return w.written
}

// CleanupWritten removes all files that were written
func (w *FileWriter) CleanupWritten() error {
	for _, path := range w.written {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}
	w.written = nil
	return nil
}

// EnsureBaseDir creates the base directory if it doesn't exist
func (w *FileWriter) EnsureBaseDir() error {
	if w.DryRun {
		return nil
	}
	return os.MkdirAll(w.BaseDir, 0755)
}

// isSubPath checks if child is a subdirectory of parent
func isSubPath(parent, child string) bool {
	// Convert to absolute paths for reliable comparison
	absParent, err := filepath.Abs(parent)
	if err != nil {
		return false
	}
	absChild, err := filepath.Abs(child)
	if err != nil {
		return false
	}

	absParent = filepath.Clean(absParent)
	absChild = filepath.Clean(absChild)

	// Add trailing separator to parent to ensure proper matching
	parentWithSep := absParent + string(filepath.Separator)

	// Check if child starts with parent path
	return absChild == absParent || len(absChild) > len(parentWithSep) && absChild[:len(parentWithSep)] == parentWithSep
}

// WriteManifest writes a generation manifest for tracking
type GenerationManifest struct {
	// Timestamp of generation
	Timestamp string `json:"timestamp"`

	// TaskID that triggered generation
	TaskID string `json:"task_id"`

	// FeatureID being implemented
	FeatureID string `json:"feature_id"`

	// Files contains all generated file records
	Files []GeneratedFileRecord `json:"files"`

	// AIMetadata contains generation metadata
	AIMetadata AIGenerationMetadata `json:"ai_metadata"`
}

// GeneratedFileRecord is a manifest entry for a generated file
type GeneratedFileRecord struct {
	// Path is the relative file path
	Path string `json:"path"`

	// Hash is the SHA-256 hash of the content
	Hash string `json:"hash"`

	// Language is the programming language
	Language string `json:"language"`

	// Size is the file size in bytes
	Size int `json:"size"`
}

// ToRecord converts a GeneratedFile to a manifest record
func (f *GeneratedFile) ToRecord() GeneratedFileRecord {
	return GeneratedFileRecord{
		Path:     f.Path,
		Hash:     f.Hash,
		Language: f.Language,
		Size:     f.Size,
	}
}

// WriteGeneratedFile is a convenience function for writing a single file
func WriteGeneratedFile(baseDir, path, content string, dryRun bool) error {
	writer := NewFileWriter(baseDir, false, dryRun)
	file := &GeneratedFile{
		Path:    path,
		Content: content,
	}
	_, err := writer.WriteFile(file)
	return err
}

// FileExists checks if a file exists at the given path
func FileExists(baseDir, path string) bool {
	fullPath := filepath.Join(baseDir, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// ReadExistingFile reads an existing file for context
func ReadExistingFile(baseDir, path string) (string, error) {
	fullPath := filepath.Join(baseDir, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
