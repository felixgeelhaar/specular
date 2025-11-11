package patch

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffGenerator generates diffs for file changes
type DiffGenerator struct {
	workingDir string
	dmp        *diffmatchpatch.DiffMatchPatch
}

// NewDiffGenerator creates a new diff generator
func NewDiffGenerator(workingDir string) *DiffGenerator {
	return &DiffGenerator{
		workingDir: workingDir,
		dmp:        diffmatchpatch.New(),
	}
}

// GeneratePatch creates a patch for file changes
func (g *DiffGenerator) GeneratePatch(stepID, stepType, workflowID, description string, fileSnapshots map[string]string) (*Patch, error) {
	patch := &Patch{
		StepID:      stepID,
		StepType:    stepType,
		Timestamp:   time.Now(),
		WorkflowID:  workflowID,
		Description: description,
		Files:       []FilePatch{},
	}

	// Track files that existed before (in snapshot)
	processedFiles := make(map[string]bool)

	// Check for modified and deleted files
	for path, oldContent := range fileSnapshots {
		processedFiles[path] = true

		fullPath := filepath.Join(g.workingDir, path)
		newContentBytes, err := os.ReadFile(fullPath)

		if os.IsNotExist(err) {
			// File was deleted
			filePatch := g.createDeletedFilePatch(path, oldContent)
			patch.Files = append(patch.Files, filePatch)
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", path, err)
		}

		newContent := string(newContentBytes)

		// Check if file was modified
		if oldContent != newContent {
			filePatch := g.createModifiedFilePatch(path, oldContent, newContent)
			patch.Files = append(patch.Files, filePatch)
		}
	}

	// Check for added files
	err := filepath.Walk(g.workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip hidden files and directories
		if strings.Contains(path, "/.") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(g.workingDir, path)
		if err != nil {
			return err
		}

		// Skip if already processed
		if processedFiles[relPath] {
			return nil
		}

		// This is a new file
		newContentBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		filePatch := g.createAddedFilePatch(relPath, string(newContentBytes))
		patch.Files = append(patch.Files, filePatch)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Calculate statistics
	patch.CalculateStats()

	return patch, nil
}

// createAddedFilePatch creates a patch for a newly added file
func (g *DiffGenerator) createAddedFilePatch(path, newContent string) FilePatch {
	diff := g.generateUnifiedDiff(path, "", newContent)
	insertions := countLines(newContent)

	return FilePatch{
		Path:       path,
		Status:     FileStatusAdded,
		NewContent: newContent,
		Diff:       diff,
		Insertions: insertions,
		Deletions:  0,
	}
}

// createModifiedFilePatch creates a patch for a modified file
func (g *DiffGenerator) createModifiedFilePatch(path, oldContent, newContent string) FilePatch {
	diff := g.generateUnifiedDiff(path, oldContent, newContent)
	insertions, deletions := g.countChanges(oldContent, newContent)

	return FilePatch{
		Path:       path,
		Status:     FileStatusModified,
		OldContent: oldContent,
		NewContent: newContent,
		Diff:       diff,
		Insertions: insertions,
		Deletions:  deletions,
	}
}

// createDeletedFilePatch creates a patch for a deleted file
func (g *DiffGenerator) createDeletedFilePatch(path, oldContent string) FilePatch {
	diff := g.generateUnifiedDiff(path, oldContent, "")
	deletions := countLines(oldContent)

	return FilePatch{
		Path:       path,
		Status:     FileStatusDeleted,
		OldContent: oldContent,
		Diff:       diff,
		Insertions: 0,
		Deletions:  deletions,
	}
}

// generateUnifiedDiff generates a unified diff in standard format
func (g *DiffGenerator) generateUnifiedDiff(path, oldContent, newContent string) string {
	diffs := g.dmp.DiffMain(oldContent, newContent, false)

	var buf bytes.Buffer

	// Write diff header
	fmt.Fprintf(&buf, "--- a/%s\n", path)
	fmt.Fprintf(&buf, "+++ b/%s\n", path)

	// Convert to unified diff format
	oldLine, newLine := 1, 1
	var hunkLines []string
	var hunkOldStart, hunkNewStart int
	var hunkOldCount, hunkNewCount int

	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")
		if diff.Text != "" && !strings.HasSuffix(diff.Text, "\n") {
			// Last line without newline
		} else {
			lines = lines[:len(lines)-1] // Remove empty last element
		}

		for i, line := range lines {
			switch diff.Type {
			case diffmatchpatch.DiffEqual:
				if len(hunkLines) == 0 {
					hunkOldStart = oldLine
					hunkNewStart = newLine
				}
				hunkLines = append(hunkLines, " "+line)
				hunkOldCount++
				hunkNewCount++
				oldLine++
				newLine++

			case diffmatchpatch.DiffDelete:
				if len(hunkLines) == 0 {
					hunkOldStart = oldLine
					hunkNewStart = newLine
				}
				hunkLines = append(hunkLines, "-"+line)
				hunkOldCount++
				oldLine++

			case diffmatchpatch.DiffInsert:
				if len(hunkLines) == 0 {
					hunkOldStart = oldLine
					hunkNewStart = newLine
				}
				hunkLines = append(hunkLines, "+"+line)
				hunkNewCount++
				newLine++
			}

			// Add newline handling
			if i < len(lines)-1 || strings.HasSuffix(diff.Text, "\n") {
				// Continue to next line
			}
		}
	}

	// Write hunk if there are changes
	if len(hunkLines) > 0 {
		fmt.Fprintf(&buf, "@@ -%d,%d +%d,%d @@\n", hunkOldStart, hunkOldCount, hunkNewStart, hunkNewCount)
		for _, line := range hunkLines {
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// countChanges counts insertions and deletions
// This counts affected lines, not the granular character changes
func (g *DiffGenerator) countChanges(oldContent, newContent string) (insertions, deletions int) {
	// Count lines in old and new content
	oldLines := countLines(oldContent)
	newLines := countLines(newContent)

	// Simple heuristic: if content changed, count the difference
	if oldLines > newLines {
		deletions = oldLines - newLines
	} else if newLines > oldLines {
		insertions = newLines - oldLines
	} else if oldContent != newContent {
		// Same number of lines but content changed - count as 1 insertion and 1 deletion per changed line
		// For simplicity, we'll say at least 1 line was modified
		insertions = 1
		deletions = 1
	}

	return insertions, deletions
}

// CaptureFileSnapshot captures the current state of files
func CaptureFileSnapshot(workingDir string, paths []string) (map[string]string, error) {
	snapshot := make(map[string]string)

	for _, path := range paths {
		fullPath := filepath.Join(workingDir, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				// File doesn't exist yet, record as empty
				snapshot[path] = ""
				continue
			}
			return nil, fmt.Errorf("failed to read file %s: %w", path, err)
		}

		snapshot[path] = string(content)
	}

	return snapshot, nil
}

// CaptureDirectorySnapshot captures all files in a directory
func CaptureDirectorySnapshot(workingDir string) (map[string]string, error) {
	snapshot := make(map[string]string)

	err := filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip hidden files and directories
		if strings.Contains(path, "/.") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(workingDir, path)
		if err != nil {
			return err
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		snapshot[relPath] = string(content)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// countLines counts the number of lines in a string
// Empty string = 0 lines
// String ending with \n = number of \n
// String not ending with \n = number of \n + 1
func countLines(content string) int {
	if content == "" {
		return 0
	}

	count := strings.Count(content, "\n")

	// If content doesn't end with newline, add 1 for the last line
	if !strings.HasSuffix(content, "\n") {
		count++
	}

	return count
}
