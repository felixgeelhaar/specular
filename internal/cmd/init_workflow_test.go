package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/templates/workflows"
)

func TestInitWorkflowCommand(t *testing.T) {
	// Test that workflow command is registered
	found := false
	for _, c := range initCmd.Commands() {
		if c.Use == "workflow <template-id>" {
			found = true
			break
		}
	}
	if !found {
		t.Error("workflow command not registered under init")
	}
}

func TestInitWorkflowListCommand(t *testing.T) {
	// Test that list subcommand is registered
	var workflowCmd *cobra.Command
	for _, c := range initCmd.Commands() {
		if strings.HasPrefix(c.Use, "workflow") {
			workflowCmd = c
			break
		}
	}
	if workflowCmd == nil {
		t.Fatal("workflow command not found")
	}

	found := false
	for _, c := range workflowCmd.Commands() {
		if c.Use == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Error("list command not registered under workflow")
	}
}

func TestInitWorkflowShowCommand(t *testing.T) {
	var workflowCmd *cobra.Command
	for _, c := range initCmd.Commands() {
		if strings.HasPrefix(c.Use, "workflow") {
			workflowCmd = c
			break
		}
	}
	if workflowCmd == nil {
		t.Fatal("workflow command not found")
	}

	found := false
	for _, c := range workflowCmd.Commands() {
		if c.Use == "show <template-id>" {
			found = true
			break
		}
	}
	if !found {
		t.Error("show command not registered under workflow")
	}
}

func TestInitWorkflowFlags(t *testing.T) {
	var workflowCmd *cobra.Command
	for _, c := range initCmd.Commands() {
		if strings.HasPrefix(c.Use, "workflow") {
			workflowCmd = c
			break
		}
	}
	if workflowCmd == nil {
		t.Fatal("workflow command not found")
	}

	// Check flags
	if workflowCmd.Flags().Lookup("var") == nil {
		t.Error("Missing --var flag")
	}
	if workflowCmd.Flags().Lookup("output-dir") == nil {
		t.Error("Missing --output-dir flag")
	}
	if workflowCmd.Flags().Lookup("dry-run") == nil {
		t.Error("Missing --dry-run flag")
	}
}

func TestInitWorkflowFlag(t *testing.T) {
	// Check that --workflow flag is on init command
	if initCmd.Flags().Lookup("workflow") == nil {
		t.Error("Missing --workflow flag on init command")
	}
}

func TestCheckMissingVariables(t *testing.T) {
	tmpl := &workflows.WorkflowTemplate{
		Variables: []workflows.TemplateVariable{
			{Name: "required1", Required: true},
			{Name: "required2", Required: true, Default: "default"},
			{Name: "optional1", Required: false},
		},
	}

	tests := []struct {
		name     string
		provided map[string]string
		expected []string
	}{
		{
			name:     "all provided",
			provided: map[string]string{"required1": "value1"},
			expected: []string{},
		},
		{
			name:     "missing required",
			provided: map[string]string{},
			expected: []string{"required1"},
		},
		{
			name:     "default not missing",
			provided: map[string]string{"required1": "value"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missing := checkMissingVariables(tmpl, tt.provided)
			if len(missing) != len(tt.expected) {
				t.Errorf("Expected %d missing, got %d: %v", len(tt.expected), len(missing), missing)
			}
		})
	}
}

func TestFindSimilarTemplates(t *testing.T) {
	ids := []string{"ci-pipeline", "data-pipeline", "microservice", "monorepo"}

	tests := []struct {
		target   string
		expected int // minimum expected matches
	}{
		{"ci", 1},       // should match ci-pipeline
		{"pipeline", 2}, // should match ci-pipeline and data-pipeline
		{"micro", 1},    // should match microservice
		{"xyz", 0},      // should match nothing
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			similar := findSimilarTemplates(tt.target, ids)
			if len(similar) < tt.expected {
				t.Errorf("Expected at least %d matches for '%s', got %d: %v", tt.expected, tt.target, len(similar), similar)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1, s2   string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			dist := levenshteinDistance(tt.s1, tt.s2)
			if dist != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, dist, tt.expected)
			}
		})
	}
}

func TestFormatCategoryName(t *testing.T) {
	tests := []struct {
		cat      workflows.Category
		expected string
	}{
		{workflows.CategoryCI, "CI/CD Pipelines"},
		{workflows.CategoryMicroservice, "Microservices"},
		{workflows.CategoryData, "Data Pipelines"},
		{workflows.CategoryMonorepo, "Monorepo"},
	}

	for _, tt := range tests {
		t.Run(string(tt.cat), func(t *testing.T) {
			result := formatCategoryName(tt.cat)
			if result != tt.expected {
				t.Errorf("formatCategoryName(%s) = %q, want %q", tt.cat, result, tt.expected)
			}
		})
	}
}

func TestGetWorkflowIDs(t *testing.T) {
	ids := GetWorkflowIDs()
	if len(ids) == 0 {
		t.Error("GetWorkflowIDs() returned empty list")
	}

	// Should contain known templates
	hasCI := false
	for _, id := range ids {
		if id == "ci-pipeline" {
			hasCI = true
			break
		}
	}
	if !hasCI {
		t.Error("Expected ci-pipeline in workflow IDs")
	}
}

func TestRunInitWorkflowList(t *testing.T) {
	var buf bytes.Buffer
	initWorkflowListCmd.SetOut(&buf)

	err := runInitWorkflowList(initWorkflowListCmd, []string{})
	if err != nil {
		t.Fatalf("runInitWorkflowList() error = %v", err)
	}

	output := buf.String()

	// Check for expected content
	if !strings.Contains(output, "Available Workflow Templates") {
		t.Error("Output should contain header")
	}
	if !strings.Contains(output, "CI/CD Pipelines") {
		t.Error("Output should contain CI/CD category")
	}
	if !strings.Contains(output, "specular init workflow") {
		t.Error("Output should contain usage example")
	}
}

func TestRunInitWorkflowShow(t *testing.T) {
	var buf bytes.Buffer
	initWorkflowShowCmd.SetOut(&buf)

	err := runInitWorkflowShow(initWorkflowShowCmd, []string{"ci-pipeline"})
	if err != nil {
		t.Fatalf("runInitWorkflowShow() error = %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "ci-pipeline") {
		t.Error("Output should contain template ID")
	}
	if !strings.Contains(output, "Example usage") {
		t.Error("Output should contain example usage")
	}
}

func TestRunInitWorkflowShowNotFound(t *testing.T) {
	var buf bytes.Buffer
	initWorkflowShowCmd.SetOut(&buf)

	err := runInitWorkflowShow(initWorkflowShowCmd, []string{"nonexistent-template"})
	if err == nil {
		t.Error("Expected error for nonexistent template")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

func TestRunInitWorkflow_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore flags
	oldVars := workflowVars
	oldDir := workflowOutputDir
	oldDryRun := workflowDryRun
	defer func() {
		workflowVars = oldVars
		workflowOutputDir = oldDir
		workflowDryRun = oldDryRun
	}()

	workflowVars = []string{"project_name=testproject", "language=go"}
	workflowOutputDir = tmpDir
	workflowDryRun = true

	var buf bytes.Buffer
	initWorkflowCmd.SetOut(&buf)

	err := runInitWorkflow(initWorkflowCmd, []string{"ci-pipeline"})
	if err != nil {
		t.Fatalf("runInitWorkflow() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Would generate") {
		t.Error("Dry run output should indicate simulation")
	}

	// Verify no files were created
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) > 0 {
		t.Error("Dry run should not create any files")
	}
}

func TestRunInitWorkflow_MissingVars(t *testing.T) {
	// Save and restore flags
	oldVars := workflowVars
	oldDir := workflowOutputDir
	oldDryRun := workflowDryRun
	defer func() {
		workflowVars = oldVars
		workflowOutputDir = oldDir
		workflowDryRun = oldDryRun
	}()

	workflowVars = []string{} // No variables provided
	workflowOutputDir = "."
	workflowDryRun = true

	err := runInitWorkflow(initWorkflowCmd, []string{"ci-pipeline"})
	if err == nil {
		t.Error("Expected error for missing required variables")
	}
	if !strings.Contains(err.Error(), "missing required variables") {
		t.Errorf("Error should mention missing variables, got: %v", err)
	}
}

func TestRunInitWorkflow_NotFound(t *testing.T) {
	err := runInitWorkflow(initWorkflowCmd, []string{"nonexistent"})
	if err == nil {
		t.Error("Expected error for nonexistent template")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

func TestRunInitWorkflow_RealFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore flags
	oldVars := workflowVars
	oldDir := workflowOutputDir
	oldDryRun := workflowDryRun
	defer func() {
		workflowVars = oldVars
		workflowOutputDir = oldDir
		workflowDryRun = oldDryRun
	}()

	workflowVars = []string{"project_name=testproject", "language=go", "deploy_target=docker", "enable_security_scan=true"}
	workflowOutputDir = tmpDir
	workflowDryRun = false

	var buf bytes.Buffer
	initWorkflowCmd.SetOut(&buf)

	err := runInitWorkflow(initWorkflowCmd, []string{"ci-pipeline"})
	if err != nil {
		t.Fatalf("runInitWorkflow() error = %v", err)
	}

	// Verify files were created
	ciPath := filepath.Join(tmpDir, ".github/workflows/ci.yaml")
	if _, statErr := os.Stat(ciPath); os.IsNotExist(statErr) {
		t.Error("Expected CI workflow file to be created")
	}
}

func TestRunInitWorkflow_InvalidVarFormat(t *testing.T) {
	// Save and restore flags
	oldVars := workflowVars
	defer func() {
		workflowVars = oldVars
	}()

	workflowVars = []string{"invalid-format-no-equals"}

	err := runInitWorkflow(initWorkflowCmd, []string{"ci-pipeline"})
	if err == nil {
		t.Error("Expected error for invalid variable format")
	}
	if !strings.Contains(err.Error(), "invalid variable format") {
		t.Errorf("Error should mention invalid format, got: %v", err)
	}
}
