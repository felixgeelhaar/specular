package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/validate"
)

func TestDiscoverConfigFiles(t *testing.T) {
	// Create temp directory with config files
	tmpDir := t.TempDir()
	specularDir := filepath.Join(tmpDir, ".specular")
	if err := os.MkdirAll(specularDir, 0755); err != nil {
		t.Fatalf("Failed to create .specular dir: %v", err)
	}

	// Create test config files
	testFiles := []string{"providers.yaml", "router.yaml", "policy.yaml"}
	for _, f := range testFiles {
		path := filepath.Join(specularDir, f)
		if err := os.WriteFile(path, []byte("test: value"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	files, err := discoverConfigFiles()
	if err != nil {
		t.Fatalf("discoverConfigFiles() error = %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}
}

func TestDiscoverConfigFilesInDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{"providers.yaml", "spec.yaml", "other.txt"}
	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test: value"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	files, err := discoverConfigFilesInDir(tmpDir)
	if err != nil {
		t.Fatalf("discoverConfigFilesInDir() error = %v", err)
	}

	// Should only find .yaml config files
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d: %v", len(files), files)
	}
}

func TestPrintValidationText(t *testing.T) {
	results := []ConfigValidationOutput{
		{
			File:  "test.yaml",
			Valid: true,
		},
		{
			File:  "invalid.yaml",
			Valid: false,
			Errors: []ConfigIssueOutput{
				{Field: "name", Message: "required field missing"},
			},
			Warnings: []ConfigIssueOutput{
				{Field: "description", Message: "recommended field missing"},
			},
		},
	}

	var buf bytes.Buffer
	printValidationText(results, &buf)
	output := buf.String()

	if !strings.Contains(output, "OK test.yaml") {
		t.Error("Output should contain OK for valid file")
	}
	if !strings.Contains(output, "invalid.yaml") {
		t.Error("Output should contain invalid file")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("Output should contain ERROR")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("Output should contain WARN")
	}
	if !strings.Contains(output, "Summary:") {
		t.Error("Output should contain summary")
	}
}

func TestOutputValidationJSON(t *testing.T) {
	results := []ConfigValidationOutput{
		{
			File:  "test.yaml",
			Valid: true,
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputValidationJSON(results)
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("outputValidationJSON() error = %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var parsed []ConfigValidationOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(parsed) != 1 {
		t.Errorf("Expected 1 result, got %d", len(parsed))
	}
	if parsed[0].File != "test.yaml" {
		t.Errorf("Expected file 'test.yaml', got '%s'", parsed[0].File)
	}
}

func TestOutputValidationSARIF(t *testing.T) {
	results := []ConfigValidationOutput{
		{
			File:  "test.yaml",
			Valid: false,
			Errors: []ConfigIssueOutput{
				{Field: "name", Message: "required field"},
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputValidationSARIF(results)
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("outputValidationSARIF() error = %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var sarif ConfigSARIFOutput
	if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
		t.Fatalf("Failed to parse SARIF output: %v", err)
	}

	if sarif.Version != "2.1.0" {
		t.Errorf("Expected SARIF version 2.1.0, got %s", sarif.Version)
	}
	if len(sarif.Runs) != 1 {
		t.Fatalf("Expected 1 run, got %d", len(sarif.Runs))
	}
	if len(sarif.Runs[0].Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(sarif.Runs[0].Results))
	}
}

func TestCopyConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	content := []byte("test content")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	if err := copyConfigFile(src, dst); err != nil {
		t.Fatalf("copyConfigFile() error = %v", err)
	}

	// Verify destination content
	readContent, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("Failed to read destination: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Content mismatch: got %s, want %s", readContent, content)
	}
}

func TestConfigValidateCommand(t *testing.T) {
	// Test that command is registered
	cmd := configCmd.Commands()
	found := false
	for _, c := range cmd {
		if c.Use == "validate [files...]" {
			found = true
			break
		}
	}
	if !found {
		t.Error("validate command not registered under config")
	}
}

func TestConfigFixCommand(t *testing.T) {
	// Test that command is registered
	cmd := configCmd.Commands()
	found := false
	for _, c := range cmd {
		if c.Use == "fix [file]" {
			found = true
			break
		}
	}
	if !found {
		t.Error("fix command not registered under config")
	}
}

func TestConfigValidateAliases(t *testing.T) {
	for _, c := range configCmd.Commands() {
		if c.Use == "validate [files...]" {
			if len(c.Aliases) != 2 {
				t.Errorf("Expected 2 aliases, got %d", len(c.Aliases))
			}
			// Check for 'check' and 'lint' aliases
			hasCheck := false
			hasLint := false
			for _, a := range c.Aliases {
				if a == "check" {
					hasCheck = true
				}
				if a == "lint" {
					hasLint = true
				}
			}
			if !hasCheck {
				t.Error("Missing 'check' alias")
			}
			if !hasLint {
				t.Error("Missing 'lint' alias")
			}
			break
		}
	}
}

func TestConfigValidateFlags(t *testing.T) {
	for _, c := range configCmd.Commands() {
		if c.Use == "validate [files...]" {
			// Check flags
			if c.Flags().Lookup("strict") == nil {
				t.Error("Missing --strict flag")
			}
			if c.Flags().Lookup("format") == nil {
				t.Error("Missing --format flag")
			}
			if c.Flags().Lookup("schema") == nil {
				t.Error("Missing --schema flag")
			}
			break
		}
	}
}

func TestConfigFixFlags(t *testing.T) {
	for _, c := range configCmd.Commands() {
		if c.Use == "fix [file]" {
			// Check flags
			if c.Flags().Lookup("dry-run") == nil {
				t.Error("Missing --dry-run flag")
			}
			if c.Flags().Lookup("backup") == nil {
				t.Error("Missing --backup flag")
			}
			if c.Flags().Lookup("interactive") == nil {
				t.Error("Missing --interactive flag")
			}
			break
		}
	}
}

func TestConvertValidationIssues(t *testing.T) {
	// Test the conversion function with actual validate.ValidationIssue
	issues := []validate.ValidationIssue{
		{Field: "name", Message: "required", Value: ""},
		{Field: "type", Message: "invalid", Value: "bad"},
	}

	result := convertValidationIssues(issues)

	if len(result) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(result))
	}
	if result[0].Field != "name" {
		t.Errorf("Expected field 'name', got '%s'", result[0].Field)
	}
	if result[1].Message != "invalid" {
		t.Errorf("Expected message 'invalid', got '%s'", result[1].Message)
	}
}
