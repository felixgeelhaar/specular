package ux

import (
	"bytes"
	"strings"
	"testing"
)

func TestFileChangeType_String(t *testing.T) {
	tests := []struct {
		cType FileChangeType
		want  string
	}{
		{FileCreate, "create"},
		{FileModify, "modify"},
		{FileDelete, "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.cType.String(); got != tt.want {
				t.Errorf("FileChangeType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewDryRunResult(t *testing.T) {
	result := NewDryRunResult("test-command")

	if result.Command != "test-command" {
		t.Errorf("Command = %q, want %q", result.Command, "test-command")
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if len(result.WouldCreate) != 0 {
		t.Error("WouldCreate should be empty")
	}
	if len(result.WouldModify) != 0 {
		t.Error("WouldModify should be empty")
	}
	if len(result.WouldDelete) != 0 {
		t.Error("WouldDelete should be empty")
	}
}

func TestDryRunResult_AddCreate(t *testing.T) {
	result := NewDryRunResult("test")
	result.AddCreate("test.txt", "content", "test file")

	if len(result.WouldCreate) != 1 {
		t.Fatalf("Expected 1 create, got %d", len(result.WouldCreate))
	}

	c := result.WouldCreate[0]
	if c.Path != "test.txt" {
		t.Errorf("Path = %q, want %q", c.Path, "test.txt")
	}
	if c.After != "content" {
		t.Errorf("After = %q, want %q", c.After, "content")
	}
	if c.Description != "test file" {
		t.Errorf("Description = %q, want %q", c.Description, "test file")
	}
	if c.SizeAfter != 7 {
		t.Errorf("SizeAfter = %d, want %d", c.SizeAfter, 7)
	}
}

func TestDryRunResult_AddModify(t *testing.T) {
	result := NewDryRunResult("test")
	result.AddModify("test.txt", "before", "after content", "modified file")

	if len(result.WouldModify) != 1 {
		t.Fatalf("Expected 1 modify, got %d", len(result.WouldModify))
	}

	c := result.WouldModify[0]
	if c.Path != "test.txt" {
		t.Errorf("Path = %q, want %q", c.Path, "test.txt")
	}
	if c.Before != "before" {
		t.Errorf("Before = %q, want %q", c.Before, "before")
	}
	if c.After != "after content" {
		t.Errorf("After = %q, want %q", c.After, "after content")
	}
	if c.SizeBefore != 6 {
		t.Errorf("SizeBefore = %d, want %d", c.SizeBefore, 6)
	}
	if c.SizeAfter != 13 {
		t.Errorf("SizeAfter = %d, want %d", c.SizeAfter, 13)
	}
}

func TestDryRunResult_AddDelete(t *testing.T) {
	result := NewDryRunResult("test")
	result.AddDelete("test.txt", "content", "deleted file")

	if len(result.WouldDelete) != 1 {
		t.Fatalf("Expected 1 delete, got %d", len(result.WouldDelete))
	}

	c := result.WouldDelete[0]
	if c.Path != "test.txt" {
		t.Errorf("Path = %q, want %q", c.Path, "test.txt")
	}
	if c.Before != "content" {
		t.Errorf("Before = %q, want %q", c.Before, "content")
	}
	if c.SizeBefore != 7 {
		t.Errorf("SizeBefore = %d, want %d", c.SizeBefore, 7)
	}
}

func TestDryRunResult_TotalChanges(t *testing.T) {
	result := NewDryRunResult("test")
	result.AddCreate("a.txt", "a", "")
	result.AddModify("b.txt", "b", "bb", "")
	result.AddDelete("c.txt", "c", "")

	if got := result.TotalChanges(); got != 3 {
		t.Errorf("TotalChanges() = %d, want %d", got, 3)
	}
}

func TestDryRunResult_HasChanges(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*DryRunResult)
		expect bool
	}{
		{
			name:   "no changes",
			setup:  func(r *DryRunResult) {},
			expect: false,
		},
		{
			name: "with create",
			setup: func(r *DryRunResult) {
				r.AddCreate("a.txt", "a", "")
			},
			expect: true,
		},
		{
			name: "with modify",
			setup: func(r *DryRunResult) {
				r.AddModify("a.txt", "a", "aa", "")
			},
			expect: true,
		},
		{
			name: "with delete",
			setup: func(r *DryRunResult) {
				r.AddDelete("a.txt", "a", "")
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewDryRunResult("test")
			tt.setup(result)
			if got := result.HasChanges(); got != tt.expect {
				t.Errorf("HasChanges() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestDryRunResult_TotalSizeChange(t *testing.T) {
	result := NewDryRunResult("test")

	// Create adds 10 bytes
	result.AddCreate("a.txt", "0123456789", "")

	// Modify changes from 5 to 15 bytes (+10)
	result.AddModify("b.txt", "12345", "123456789012345", "")

	// Delete removes 5 bytes (-5)
	result.AddDelete("c.txt", "12345", "")

	// Total: 10 + 10 - 5 = 15
	if got := result.TotalSizeChange(); got != 15 {
		t.Errorf("TotalSizeChange() = %d, want %d", got, 15)
	}
}

func TestDryRunResult_PrintSummary(t *testing.T) {
	result := NewDryRunResult("test-command")
	result.Description = "Test operation"
	result.AddCreate("a.txt", "content", "")
	result.AddModify("b.txt", "old", "new", "")
	result.AddDelete("c.txt", "content", "")
	result.EstimatedCost = 0.05

	var buf bytes.Buffer
	result.PrintSummary(&buf)

	output := buf.String()

	if !strings.Contains(output, "DRY RUN SUMMARY") {
		t.Error("Output should contain header")
	}
	if !strings.Contains(output, "test-command") {
		t.Error("Output should contain command name")
	}
	if !strings.Contains(output, "Test operation") {
		t.Error("Output should contain description")
	}
	if !strings.Contains(output, "Create:") {
		t.Error("Output should contain create count")
	}
	if !strings.Contains(output, "Modify:") {
		t.Error("Output should contain modify count")
	}
	if !strings.Contains(output, "Delete:") {
		t.Error("Output should contain delete count")
	}
	if !strings.Contains(output, "0.0500") {
		t.Error("Output should contain estimated cost")
	}
}

func TestDryRunResult_PrintSummary_NoChanges(t *testing.T) {
	result := NewDryRunResult("test-command")

	var buf bytes.Buffer
	result.PrintSummary(&buf)

	output := buf.String()

	if !strings.Contains(output, "No changes would be made") {
		t.Error("Output should indicate no changes")
	}
}

func TestDryRunResult_PrintDiff(t *testing.T) {
	result := NewDryRunResult("test")
	result.AddCreate("new.txt", "new content", "Creating new file")
	result.AddModify("mod.txt", "old line", "new line", "Updating content")
	result.AddDelete("old.txt", "deleted content", "Removing old file")

	var buf bytes.Buffer
	result.PrintDiff(&buf, false)

	output := buf.String()

	if !strings.Contains(output, "CREATE") {
		t.Error("Output should contain CREATE")
	}
	if !strings.Contains(output, "new.txt") {
		t.Error("Output should contain created file path")
	}
	if !strings.Contains(output, "MODIFY") {
		t.Error("Output should contain MODIFY")
	}
	if !strings.Contains(output, "mod.txt") {
		t.Error("Output should contain modified file path")
	}
	if !strings.Contains(output, "DELETE") {
		t.Error("Output should contain DELETE")
	}
	if !strings.Contains(output, "old.txt") {
		t.Error("Output should contain deleted file path")
	}
}

func TestDryRunResult_PrintDiff_WithColor(t *testing.T) {
	result := NewDryRunResult("test")
	result.AddCreate("test.txt", "content", "")

	var buf bytes.Buffer
	result.PrintDiff(&buf, true)

	output := buf.String()

	// With color, output should have ANSI escape codes
	if !strings.Contains(output, "CREATE") {
		t.Error("Output should contain CREATE")
	}
}

func TestDryRunResult_PrintFileList(t *testing.T) {
	result := NewDryRunResult("test")
	result.AddCreate("new.txt", "", "")
	result.AddModify("mod.txt", "", "", "")
	result.AddDelete("old.txt", "", "")

	var buf bytes.Buffer
	result.PrintFileList(&buf)

	output := buf.String()

	if !strings.Contains(output, "+ new.txt") {
		t.Error("Output should contain create marker")
	}
	if !strings.Contains(output, "~ mod.txt") {
		t.Error("Output should contain modify marker")
	}
	if !strings.Contains(output, "- old.txt") {
		t.Error("Output should contain delete marker")
	}
}

func TestDryRunResult_AffectedPaths(t *testing.T) {
	result := NewDryRunResult("test")
	result.AddCreate("a.txt", "", "")
	result.AddModify("b.txt", "", "", "")
	result.AddDelete("c.txt", "", "")

	paths := result.AffectedPaths()

	if len(paths) != 3 {
		t.Errorf("Expected 3 paths, got %d", len(paths))
	}

	expected := map[string]bool{"a.txt": true, "b.txt": true, "c.txt": true}
	for _, p := range paths {
		if !expected[p] {
			t.Errorf("Unexpected path: %s", p)
		}
	}
}

func TestDryRunResult_AffectedDirectories(t *testing.T) {
	result := NewDryRunResult("test")
	result.AddCreate("dir1/a.txt", "", "")
	result.AddModify("dir1/b.txt", "", "", "")
	result.AddDelete("dir2/c.txt", "", "")

	dirs := result.AffectedDirectories()

	if len(dirs) != 2 {
		t.Errorf("Expected 2 directories, got %d", len(dirs))
	}

	expected := map[string]bool{"dir1": true, "dir2": true}
	for _, d := range dirs {
		if !expected[d] {
			t.Errorf("Unexpected directory: %s", d)
		}
	}
}

func TestFormatBytes_DryRun(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{-1024, "-1.0 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatSizeDiff(t *testing.T) {
	tests := []struct {
		diff int64
		want string
	}{
		{0, "no change"},
		{1024, "+1.0 KB"},
		{-1024, "-1.0 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSizeDiff(tt.diff)
			if got != tt.want {
				t.Errorf("formatSizeDiff(%d) = %q, want %q", tt.diff, got, tt.want)
			}
		})
	}
}

func TestGetDryRunStyles(t *testing.T) {
	// With color
	stylesColor := getDryRunStyles(true)
	if stylesColor.Header.GetBold() != true {
		t.Error("Header should be bold with color")
	}

	// Without color
	stylesNoColor := getDryRunStyles(false)
	if stylesNoColor.Header.GetBold() {
		t.Error("Header should not be bold without color")
	}
}
