package patch

import (
	"fmt"
	"os"
	"path/filepath"
)

// Writer handles writing patches to disk
type Writer struct {
	patchDir string
}

// NewWriter creates a new patch writer
func NewWriter(patchDir string) *Writer {
	return &Writer{
		patchDir: patchDir,
	}
}

// WritePatch writes a patch to disk
func (w *Writer) WritePatch(patch *Patch) (string, error) {
	// Create patches directory if it doesn't exist
	if err := os.MkdirAll(w.patchDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create patch directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("%s_%s.patch.json", patch.WorkflowID, patch.StepID)
	patchPath := filepath.Join(w.patchDir, filename)

	// Serialize patch
	jsonData, err := patch.ToJSON()
	if err != nil {
		return "", fmt.Errorf("failed to serialize patch: %w", err)
	}

	// Write to file
	if err := os.WriteFile(patchPath, jsonData, 0600); err != nil {
		return "", fmt.Errorf("failed to write patch file: %w", err)
	}

	return patchPath, nil
}

// ReadPatch reads a patch from disk
func (w *Writer) ReadPatch(workflowID, stepID string) (*Patch, error) {
	filename := fmt.Sprintf("%s_%s.patch.json", workflowID, stepID)
	patchPath := filepath.Join(w.patchDir, filename)

	data, err := os.ReadFile(patchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read patch file: %w", err)
	}

	patch, err := FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse patch: %w", err)
	}

	return patch, nil
}

// ListPatches lists all patches for a workflow
func (w *Writer) ListPatches(workflowID string) ([]*PatchMetadata, error) {
	pattern := filepath.Join(w.patchDir, fmt.Sprintf("%s_*.patch.json", workflowID))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob patches: %w", err)
	}

	var patches []*PatchMetadata
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip unreadable files
		}

		patch, err := FromJSON(data)
		if err != nil {
			continue // Skip unparseable files
		}

		patches = append(patches, patch.GetMetadata())
	}

	return patches, nil
}

// DeletePatch deletes a patch file
func (w *Writer) DeletePatch(workflowID, stepID string) error {
	filename := fmt.Sprintf("%s_%s.patch.json", workflowID, stepID)
	patchPath := filepath.Join(w.patchDir, filename)

	if err := os.Remove(patchPath); err != nil {
		return fmt.Errorf("failed to delete patch: %w", err)
	}

	return nil
}

// GetPatchPath returns the file path for a patch
func (w *Writer) GetPatchPath(workflowID, stepID string) string {
	filename := fmt.Sprintf("%s_%s.patch.json", workflowID, stepID)
	return filepath.Join(w.patchDir, filename)
}

// PatchExists checks if a patch file exists
func (w *Writer) PatchExists(workflowID, stepID string) bool {
	patchPath := w.GetPatchPath(workflowID, stepID)
	_, err := os.Stat(patchPath)
	return err == nil
}
