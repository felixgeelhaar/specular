package ux

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// FileChangeType represents the type of file change
type FileChangeType int

const (
	// FileCreate indicates a file will be created
	FileCreate FileChangeType = iota
	// FileModify indicates a file will be modified
	FileModify
	// FileDelete indicates a file will be deleted
	FileDelete
)

// String returns a string representation of the file change type
func (t FileChangeType) String() string {
	switch t {
	case FileCreate:
		return "create"
	case FileModify:
		return "modify"
	case FileDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// FileChange represents a change to a file
type FileChange struct {
	Type        FileChangeType
	Path        string
	Before      string // Content before change (empty for create)
	After       string // Content after change (empty for delete)
	SizeBefore  int64
	SizeAfter   int64
	Description string
}

// DryRunResult represents the result of a dry-run operation
type DryRunResult struct {
	Command       string
	Description   string
	Timestamp     time.Time
	WouldCreate   []FileChange
	WouldModify   []FileChange
	WouldDelete   []FileChange
	EstimatedCost float64
	Duration      time.Duration
	Metadata      map[string]interface{}
}

// NewDryRunResult creates a new DryRunResult
func NewDryRunResult(command string) *DryRunResult {
	return &DryRunResult{
		Command:     command,
		Timestamp:   time.Now(),
		WouldCreate: []FileChange{},
		WouldModify: []FileChange{},
		WouldDelete: []FileChange{},
		Metadata:    make(map[string]interface{}),
	}
}

// AddCreate adds a file that would be created
func (r *DryRunResult) AddCreate(path string, content string, description string) {
	r.WouldCreate = append(r.WouldCreate, FileChange{
		Type:        FileCreate,
		Path:        path,
		After:       content,
		SizeAfter:   int64(len(content)),
		Description: description,
	})
}

// AddModify adds a file that would be modified
func (r *DryRunResult) AddModify(path string, before, after string, description string) {
	r.WouldModify = append(r.WouldModify, FileChange{
		Type:        FileModify,
		Path:        path,
		Before:      before,
		After:       after,
		SizeBefore:  int64(len(before)),
		SizeAfter:   int64(len(after)),
		Description: description,
	})
}

// AddDelete adds a file that would be deleted
func (r *DryRunResult) AddDelete(path string, content string, description string) {
	r.WouldDelete = append(r.WouldDelete, FileChange{
		Type:        FileDelete,
		Path:        path,
		Before:      content,
		SizeBefore:  int64(len(content)),
		Description: description,
	})
}

// TotalChanges returns the total number of changes
func (r *DryRunResult) TotalChanges() int {
	return len(r.WouldCreate) + len(r.WouldModify) + len(r.WouldDelete)
}

// HasChanges returns true if there are any changes
func (r *DryRunResult) HasChanges() bool {
	return r.TotalChanges() > 0
}

// TotalSizeChange returns the net size change in bytes
func (r *DryRunResult) TotalSizeChange() int64 {
	var change int64

	// Created files add size
	for _, c := range r.WouldCreate {
		change += c.SizeAfter
	}

	// Modified files change size
	for _, c := range r.WouldModify {
		change += c.SizeAfter - c.SizeBefore
	}

	// Deleted files remove size
	for _, c := range r.WouldDelete {
		change -= c.SizeBefore
	}

	return change
}

// PrintSummary prints a summary of the dry-run result
func (r *DryRunResult) PrintSummary(w io.Writer) {
	styles := getDryRunStyles(true)

	// Header
	fmt.Fprintf(w, "\n%s\n", styles.Header.Render("DRY RUN SUMMARY"))
	fmt.Fprintf(w, "Command: %s\n", r.Command)
	if r.Description != "" {
		fmt.Fprintf(w, "Description: %s\n", r.Description)
	}
	fmt.Fprintf(w, "Timestamp: %s\n\n", r.Timestamp.Format(time.RFC3339))

	// Changes summary
	if !r.HasChanges() {
		fmt.Fprintf(w, "%s\n", styles.Info.Render("No changes would be made."))
		return
	}

	fmt.Fprintf(w, "Changes:\n")
	if len(r.WouldCreate) > 0 {
		fmt.Fprintf(w, "  %s %d file(s)\n", styles.Create.Render("+ Create:"), len(r.WouldCreate))
	}
	if len(r.WouldModify) > 0 {
		fmt.Fprintf(w, "  %s %d file(s)\n", styles.Modify.Render("~ Modify:"), len(r.WouldModify))
	}
	if len(r.WouldDelete) > 0 {
		fmt.Fprintf(w, "  %s %d file(s)\n", styles.Delete.Render("- Delete:"), len(r.WouldDelete))
	}

	// Size change
	sizeChange := r.TotalSizeChange()
	if sizeChange != 0 {
		sign := "+"
		if sizeChange < 0 {
			sign = ""
		}
		fmt.Fprintf(w, "\nSize change: %s%s\n", sign, formatBytes(sizeChange))
	}

	// Estimated cost
	if r.EstimatedCost > 0 {
		fmt.Fprintf(w, "Estimated cost: $%.4f\n", r.EstimatedCost)
	}

	fmt.Fprintf(w, "\n")
}

// PrintDiff prints a detailed diff of the dry-run result
func (r *DryRunResult) PrintDiff(w io.Writer, useColor bool) {
	styles := getDryRunStyles(useColor)

	// Print created files
	for _, c := range r.WouldCreate {
		fmt.Fprintf(w, "\n%s %s\n", styles.Create.Render("+++ CREATE"), c.Path)
		if c.Description != "" {
			fmt.Fprintf(w, "    %s\n", styles.Description.Render(c.Description))
		}
		if c.After != "" {
			printContentPreview(w, c.After, "+", styles.AddLine, useColor)
		}
		fmt.Fprintf(w, "    Size: %s\n", formatBytes(c.SizeAfter))
	}

	// Print modified files
	for _, c := range r.WouldModify {
		fmt.Fprintf(w, "\n%s %s\n", styles.Modify.Render("~~~ MODIFY"), c.Path)
		if c.Description != "" {
			fmt.Fprintf(w, "    %s\n", styles.Description.Render(c.Description))
		}
		printUnifiedDiff(w, c.Before, c.After, styles, useColor)
		fmt.Fprintf(w, "    Size: %s -> %s (%s)\n",
			formatBytes(c.SizeBefore),
			formatBytes(c.SizeAfter),
			formatSizeDiff(c.SizeAfter-c.SizeBefore))
	}

	// Print deleted files
	for _, c := range r.WouldDelete {
		fmt.Fprintf(w, "\n%s %s\n", styles.Delete.Render("--- DELETE"), c.Path)
		if c.Description != "" {
			fmt.Fprintf(w, "    %s\n", styles.Description.Render(c.Description))
		}
		if c.Before != "" {
			printContentPreview(w, c.Before, "-", styles.RemoveLine, useColor)
		}
		fmt.Fprintf(w, "    Size: %s\n", formatBytes(c.SizeBefore))
	}
}

// PrintFileList prints a simple list of files that would be changed
func (r *DryRunResult) PrintFileList(w io.Writer) {
	if len(r.WouldCreate) > 0 {
		fmt.Fprintf(w, "\nFiles to create:\n")
		for _, c := range r.WouldCreate {
			fmt.Fprintf(w, "  + %s\n", c.Path)
		}
	}

	if len(r.WouldModify) > 0 {
		fmt.Fprintf(w, "\nFiles to modify:\n")
		for _, c := range r.WouldModify {
			fmt.Fprintf(w, "  ~ %s\n", c.Path)
		}
	}

	if len(r.WouldDelete) > 0 {
		fmt.Fprintf(w, "\nFiles to delete:\n")
		for _, c := range r.WouldDelete {
			fmt.Fprintf(w, "  - %s\n", c.Path)
		}
	}
}

// AffectedPaths returns all paths that would be affected
func (r *DryRunResult) AffectedPaths() []string {
	paths := make([]string, 0, r.TotalChanges())

	for _, c := range r.WouldCreate {
		paths = append(paths, c.Path)
	}
	for _, c := range r.WouldModify {
		paths = append(paths, c.Path)
	}
	for _, c := range r.WouldDelete {
		paths = append(paths, c.Path)
	}

	return paths
}

// AffectedDirectories returns all directories that would be affected
func (r *DryRunResult) AffectedDirectories() []string {
	dirSet := make(map[string]bool)

	for _, path := range r.AffectedPaths() {
		dir := filepath.Dir(path)
		dirSet[dir] = true
	}

	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}

	return dirs
}

// DryRunStyles holds styles for dry-run output
type DryRunStyles struct {
	Header      lipgloss.Style
	Create      lipgloss.Style
	Modify      lipgloss.Style
	Delete      lipgloss.Style
	Info        lipgloss.Style
	Description lipgloss.Style
	AddLine     lipgloss.Style
	RemoveLine  lipgloss.Style
	ContextLine lipgloss.Style
}

// getDryRunStyles returns styles for dry-run output
func getDryRunStyles(useColor bool) DryRunStyles {
	if !useColor {
		return DryRunStyles{
			Header:      lipgloss.NewStyle(),
			Create:      lipgloss.NewStyle(),
			Modify:      lipgloss.NewStyle(),
			Delete:      lipgloss.NewStyle(),
			Info:        lipgloss.NewStyle(),
			Description: lipgloss.NewStyle(),
			AddLine:     lipgloss.NewStyle(),
			RemoveLine:  lipgloss.NewStyle(),
			ContextLine: lipgloss.NewStyle(),
		}
	}

	return DryRunStyles{
		Header:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
		Create:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")), // Green
		Modify:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11")), // Yellow
		Delete:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),  // Red
		Info:        lipgloss.NewStyle().Foreground(lipgloss.Color("8")),             // Gray
		Description: lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("8")),
		AddLine:     lipgloss.NewStyle().Foreground(lipgloss.Color("10")), // Green
		RemoveLine:  lipgloss.NewStyle().Foreground(lipgloss.Color("9")),  // Red
		ContextLine: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),  // Gray
	}
}

// printContentPreview prints a preview of content with line prefix
func printContentPreview(w io.Writer, content string, prefix string, style lipgloss.Style, useColor bool) {
	lines := strings.Split(content, "\n")
	maxLines := 10

	if len(lines) > maxLines {
		for i := 0; i < maxLines/2; i++ {
			if useColor {
				fmt.Fprintf(w, "    %s\n", style.Render(prefix+" "+lines[i]))
			} else {
				fmt.Fprintf(w, "    %s %s\n", prefix, lines[i])
			}
		}
		fmt.Fprintf(w, "    ... (%d more lines) ...\n", len(lines)-maxLines)
		for i := len(lines) - maxLines/2; i < len(lines); i++ {
			if useColor {
				fmt.Fprintf(w, "    %s\n", style.Render(prefix+" "+lines[i]))
			} else {
				fmt.Fprintf(w, "    %s %s\n", prefix, lines[i])
			}
		}
	} else {
		for _, line := range lines {
			if useColor {
				fmt.Fprintf(w, "    %s\n", style.Render(prefix+" "+line))
			} else {
				fmt.Fprintf(w, "    %s %s\n", prefix, line)
			}
		}
	}
}

// printUnifiedDiff prints a simple unified diff
func printUnifiedDiff(w io.Writer, before, after string, styles DryRunStyles, useColor bool) {
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")

	// Simple line-by-line comparison (not a true diff algorithm)
	maxLines := max(len(beforeLines), len(afterLines))
	previewLines := 10

	if maxLines > previewLines*2 {
		fmt.Fprintf(w, "    @@ Changes preview (showing first and last %d lines) @@\n", previewLines)
	}

	for i := 0; i < min(previewLines, maxLines); i++ {
		printDiffLine(w, i, beforeLines, afterLines, styles, useColor)
	}

	if maxLines > previewLines*2 {
		fmt.Fprintf(w, "    ... (%d more lines) ...\n", maxLines-previewLines*2)
		for i := maxLines - previewLines; i < maxLines; i++ {
			printDiffLine(w, i, beforeLines, afterLines, styles, useColor)
		}
	} else if maxLines > previewLines {
		for i := previewLines; i < maxLines; i++ {
			printDiffLine(w, i, beforeLines, afterLines, styles, useColor)
		}
	}
}

// printDiffLine prints a single diff line
func printDiffLine(w io.Writer, i int, beforeLines, afterLines []string, styles DryRunStyles, useColor bool) {
	beforeLine := ""
	afterLine := ""

	if i < len(beforeLines) {
		beforeLine = beforeLines[i]
	}
	if i < len(afterLines) {
		afterLine = afterLines[i]
	}

	if beforeLine == afterLine {
		// Unchanged line
		if useColor {
			fmt.Fprintf(w, "    %s\n", styles.ContextLine.Render("  "+afterLine))
		} else {
			fmt.Fprintf(w, "      %s\n", afterLine)
		}
	} else if beforeLine == "" && afterLine != "" {
		// Added line
		if useColor {
			fmt.Fprintf(w, "    %s\n", styles.AddLine.Render("+ "+afterLine))
		} else {
			fmt.Fprintf(w, "    + %s\n", afterLine)
		}
	} else if beforeLine != "" && afterLine == "" {
		// Removed line
		if useColor {
			fmt.Fprintf(w, "    %s\n", styles.RemoveLine.Render("- "+beforeLine))
		} else {
			fmt.Fprintf(w, "    - %s\n", beforeLine)
		}
	} else {
		// Changed line
		if useColor {
			fmt.Fprintf(w, "    %s\n", styles.RemoveLine.Render("- "+beforeLine))
			fmt.Fprintf(w, "    %s\n", styles.AddLine.Render("+ "+afterLine))
		} else {
			fmt.Fprintf(w, "    - %s\n", beforeLine)
			fmt.Fprintf(w, "    + %s\n", afterLine)
		}
	}
}

// formatSizeDiff formats a size difference with sign
func formatSizeDiff(diff int64) string {
	if diff == 0 {
		return "no change"
	}
	sign := "+"
	if diff < 0 {
		sign = ""
	}
	return sign + formatBytes(diff)
}

// formatBytes is a helper to format bytes (reuse from command_tui.go)
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	absBytes := bytes
	sign := ""
	if bytes < 0 {
		absBytes = -bytes
		sign = "-"
	}

	switch {
	case absBytes >= GB:
		return fmt.Sprintf("%s%.1f GB", sign, float64(absBytes)/GB)
	case absBytes >= MB:
		return fmt.Sprintf("%s%.1f MB", sign, float64(absBytes)/MB)
	case absBytes >= KB:
		return fmt.Sprintf("%s%.1f KB", sign, float64(absBytes)/KB)
	default:
		return fmt.Sprintf("%s%d B", sign, absBytes)
	}
}
