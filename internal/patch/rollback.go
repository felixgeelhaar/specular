package patch

import (
	"fmt"
	"os"
	"path/filepath"
)

// Rollback handles applying patches in reverse
type Rollback struct {
	workingDir string
	writer     *Writer
}

// NewRollback creates a new rollback handler
func NewRollback(workingDir string, patchDir string) *Rollback {
	return &Rollback{
		workingDir: workingDir,
		writer:     NewWriter(patchDir),
	}
}

// RollbackResult contains the result of a rollback operation
type RollbackResult struct {
	Success       bool     `json:"success"`
	StepsReverted int      `json:"stepsReverted"`
	Errors        []string `json:"errors"`
	Conflicts     []string `json:"conflicts"`
}

// RollbackStep rolls back a single step by applying its patch in reverse
func (r *Rollback) RollbackStep(workflowID, stepID string) error {
	// Read the patch
	patch, err := r.writer.ReadPatch(workflowID, stepID)
	if err != nil {
		return fmt.Errorf("failed to read patch: %w", err)
	}

	// Apply each file patch in reverse
	for _, filePatch := range patch.Files {
		if err := r.rollbackFile(filePatch); err != nil {
			return fmt.Errorf("failed to rollback file %s: %w", filePatch.Path, err)
		}
	}

	return nil
}

// RollbackToStep rolls back all steps after the specified step
func (r *Rollback) RollbackToStep(workflowID, targetStepID string) (*RollbackResult, error) {
	result := &RollbackResult{
		Success:   true,
		Errors:    []string{},
		Conflicts: []string{},
	}

	// List all patches for the workflow
	patches, err := r.writer.ListPatches(workflowID)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to list patches: %v", err))
		return result, err
	}

	// Find patches to rollback (all patches after target step)
	var patchesToRollback []*PatchMetadata
	foundTarget := false

	// Patches are in chronological order, we need to rollback in reverse
	for i := len(patches) - 1; i >= 0; i-- {
		patch := patches[i]

		if patch.StepID == targetStepID {
			foundTarget = true
			break
		}

		patchesToRollback = append(patchesToRollback, patch)
	}

	if !foundTarget && targetStepID != "" {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("target step %s not found", targetStepID))
		return result, fmt.Errorf("target step not found")
	}

	// Rollback each patch in reverse order
	for _, patch := range patchesToRollback {
		if err := r.RollbackStep(workflowID, patch.StepID); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("step %s: %v", patch.StepID, err))
			continue
		}

		result.StepsReverted++
	}

	return result, nil
}

// RollbackAll rolls back all steps for a workflow
func (r *Rollback) RollbackAll(workflowID string) (*RollbackResult, error) {
	return r.RollbackToStep(workflowID, "")
}

// rollbackFile applies a file patch in reverse
func (r *Rollback) rollbackFile(filePatch FilePatch) error {
	fullPath := filepath.Join(r.workingDir, filePatch.Path)

	switch filePatch.Status {
	case FileStatusAdded:
		// File was added, so delete it
		return r.deleteFile(fullPath)

	case FileStatusModified:
		// File was modified, restore old content
		return r.restoreFile(fullPath, filePatch.OldContent)

	case FileStatusDeleted:
		// File was deleted, restore it
		return r.recreateFile(fullPath, filePatch.OldContent)

	case FileStatusRenamed:
		// File was renamed, rename it back
		oldFullPath := filepath.Join(r.workingDir, filePatch.OldPath)
		return r.renameFile(fullPath, oldFullPath)

	default:
		return fmt.Errorf("unknown file status: %s", filePatch.Status)
	}
}

// deleteFile deletes a file
func (r *Rollback) deleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, that's okay
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// restoreFile restores a file to its old content
func (r *Rollback) restoreFile(path, oldContent string) error {
	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Write old content
	if err := os.WriteFile(path, []byte(oldContent), 0600); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}

// recreateFile recreates a deleted file
func (r *Rollback) recreateFile(path, content string) error {
	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Write content
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to recreate file: %w", err)
	}

	return nil
}

// renameFile renames a file back to its old name
func (r *Rollback) renameFile(newPath, oldPath string) error {
	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(oldPath), 0750); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Rename file
	if err := os.Rename(newPath, oldPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// VerifyRollbackSafety checks if rollback can be safely performed
func (r *Rollback) VerifyRollbackSafety(workflowID, stepID string) (bool, []string, error) {
	var warnings []string

	// Read the patch
	patch, err := r.writer.ReadPatch(workflowID, stepID)
	if err != nil {
		return false, warnings, fmt.Errorf("failed to read patch: %w", err)
	}

	// Check each file for potential conflicts
	for _, filePatch := range patch.Files {
		fullPath := filepath.Join(r.workingDir, filePatch.Path)

		switch filePatch.Status {
		case FileStatusAdded:
			// Check if file still exists
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				warnings = append(warnings, fmt.Sprintf("file %s no longer exists", filePatch.Path))
			}

		case FileStatusModified:
			// Check if current content matches what we expect
			currentContent, err := os.ReadFile(fullPath)
			if err != nil {
				if os.IsNotExist(err) {
					warnings = append(warnings, fmt.Sprintf("file %s no longer exists", filePatch.Path))
					continue
				}
				return false, warnings, fmt.Errorf("failed to read file %s: %w", fullPath, err)
			}

			// Warn if content has changed since patch was created
			if string(currentContent) != filePatch.NewContent {
				warnings = append(warnings, fmt.Sprintf("file %s has been modified since patch was created", filePatch.Path))
			}

		case FileStatusDeleted:
			// Check if file has been recreated
			if _, err := os.Stat(fullPath); err == nil {
				warnings = append(warnings, fmt.Sprintf("file %s has been recreated", filePatch.Path))
			}
		}
	}

	// Return true if no critical errors, but include warnings
	return len(warnings) == 0, warnings, nil
}
