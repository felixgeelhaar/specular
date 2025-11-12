package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetTraceFiles(t *testing.T) {
	// Create a temporary log directory
	tmpDir := t.TempDir()

	// Create some test trace files
	trace1 := filepath.Join(tmpDir, "trace_test-1.json")
	trace2 := filepath.Join(tmpDir, "trace_test-2.json")
	trace3 := filepath.Join(tmpDir, "trace_test-3.json")
	otherFile := filepath.Join(tmpDir, "other.log")

	// Write test files with different timestamps
	if err := os.WriteFile(trace1, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := os.WriteFile(trace2, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := os.WriteFile(trace3, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a non-trace file (should be ignored)
	if err := os.WriteFile(otherFile, []byte("log"), 0644); err != nil {
		t.Fatal(err)
	}

	// Get trace files
	traces, err := getTraceFiles(tmpDir)
	if err != nil {
		t.Fatalf("getTraceFiles() error = %v", err)
	}

	// Should only get trace_*.json files
	if len(traces) != 3 {
		t.Errorf("getTraceFiles() count = %d, want 3", len(traces))
	}

	// Should be sorted by creation time (newest first)
	if len(traces) > 0 && traces[0].ID != "test-3" {
		t.Errorf("First trace ID = %s, want test-3 (newest)", traces[0].ID)
	}

	// Verify trace IDs are extracted correctly
	expectedIDs := map[string]bool{
		"test-1": true,
		"test-2": true,
		"test-3": true,
	}

	for _, trace := range traces {
		if !expectedIDs[trace.ID] {
			t.Errorf("Unexpected trace ID: %s", trace.ID)
		}

		// Verify path is correct
		expectedPath := filepath.Join(tmpDir, "trace_"+trace.ID+".json")
		if trace.Path != expectedPath {
			t.Errorf("Trace path = %s, want %s", trace.Path, expectedPath)
		}

		// Verify size is set
		if trace.Size == 0 {
			t.Errorf("Trace %s size should not be 0", trace.ID)
		}

		// Verify creation time is set
		if trace.CreatedAt.IsZero() {
			t.Errorf("Trace %s creation time should not be zero", trace.ID)
		}
	}
}

func TestGetTraceFiles_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	traces, err := getTraceFiles(tmpDir)
	if err != nil {
		t.Fatalf("getTraceFiles() error = %v", err)
	}

	if len(traces) != 0 {
		t.Errorf("getTraceFiles() count = %d, want 0", len(traces))
	}
}

func TestGetTraceFiles_NonexistentDirectory(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent")

	_, err := getTraceFiles(nonexistent)
	if err == nil {
		t.Error("getTraceFiles() should error on nonexistent directory")
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want string
	}{
		{
			name: "bytes",
			size: 512,
			want: "512 B",
		},
		{
			name: "kilobytes",
			size: 2048,
			want: "2.00 KB",
		},
		{
			name: "megabytes",
			size: 5 * 1024 * 1024,
			want: "5.00 MB",
		},
		{
			name: "gigabytes",
			size: 3 * 1024 * 1024 * 1024,
			want: "3.00 GB",
		},
		{
			name: "fractional KB",
			size: 1536,
			want: "1.50 KB",
		},
		{
			name: "fractional MB",
			size: 7*1024*1024 + 512*1024,
			want: "7.50 MB",
		},
		{
			name: "zero",
			size: 0,
			want: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFileSize(tt.size)
			if got != tt.want {
				t.Errorf("formatFileSize(%d) = %s, want %s", tt.size, got, tt.want)
			}
		})
	}
}

func TestExtractField(t *testing.T) {
	event := map[string]interface{}{
		"timestamp": "2025-11-11T12:00:00Z",
		"level":     "info",
		"message":   "test message",
		"type":      "workflow_start",
		"number":    42,
		"bool":      true,
	}

	tests := []struct {
		name  string
		field string
		want  string
	}{
		{
			name:  "string field",
			field: "message",
			want:  "test message",
		},
		{
			name:  "timestamp field",
			field: "timestamp",
			want:  "2025-11-11T12:00:00Z",
		},
		{
			name:  "level field",
			field: "level",
			want:  "info",
		},
		{
			name:  "type field",
			field: "type",
			want:  "workflow_start",
		},
		{
			name:  "number field",
			field: "number",
			want:  "42",
		},
		{
			name:  "bool field",
			field: "bool",
			want:  "true",
		},
		{
			name:  "nonexistent field",
			field: "nonexistent",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractField(event, tt.field)
			if got != tt.want {
				t.Errorf("extractField(%s) = %s, want %s", tt.field, got, tt.want)
			}
		})
	}
}

func TestExtractField_EmptyEvent(t *testing.T) {
	event := map[string]interface{}{}

	got := extractField(event, "any_field")
	if got != "" {
		t.Errorf("extractField() on empty event = %s, want empty string", got)
	}
}

func TestTailFile(t *testing.T) {
	// Create a temporary file with multiple lines
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	content := `{"line": 1}
{"line": 2}
{"line": 3}
{"line": 4}
{"line": 5}
{"line": 6}
{"line": 7}
{"line": 8}
{"line": 9}
{"line": 10}
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Test tailing last 3 lines
	// Note: tailFile prints to stdout, so we can't easily capture the output
	// We'll just verify it doesn't error
	err := tailFile(testFile, 3)
	if err != nil {
		t.Errorf("tailFile() error = %v", err)
	}
}

func TestTailFile_FewerLinesThanRequested(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	content := `{"line": 1}
{"line": 2}
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Request more lines than exist
	err := tailFile(testFile, 10)
	if err != nil {
		t.Errorf("tailFile() error = %v", err)
	}
}

func TestTailFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.log")

	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	err := tailFile(testFile, 10)
	if err != nil {
		t.Errorf("tailFile() error = %v", err)
	}
}

func TestTailFile_NonexistentFile(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.log")

	err := tailFile(nonexistent, 10)
	if err == nil {
		t.Error("tailFile() should error on nonexistent file")
	}
}

func TestGetLogDirectory(t *testing.T) {
	logDir := getLogDirectory()

	// Should contain .specular/logs
	if logDir == "" {
		t.Error("getLogDirectory() should not return empty string")
	}

	// Should end with logs
	if !filepath.IsAbs(logDir) {
		t.Error("getLogDirectory() should return absolute path")
	}
}
