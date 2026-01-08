package workflows

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	if len(registry.templates) == 0 {
		t.Error("Registry should have templates loaded")
	}
}

func TestRegistryGet(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	// Test getting existing template
	tmpl, ok := registry.Get("ci-pipeline")
	if !ok {
		t.Error("Get() should find ci-pipeline template")
	}
	if tmpl.ID != "ci-pipeline" {
		t.Errorf("Template ID = %s, want ci-pipeline", tmpl.ID)
	}

	// Test getting non-existing template
	_, ok = registry.Get("non-existent")
	if ok {
		t.Error("Get() should not find non-existent template")
	}
}

func TestRegistryList(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	templates := registry.List()
	if len(templates) == 0 {
		t.Error("List() should return templates")
	}

	// Verify all templates have required fields
	for _, tmpl := range templates {
		if tmpl.ID == "" {
			t.Error("Template should have ID")
		}
		if tmpl.Name == "" {
			t.Error("Template should have Name")
		}
		if tmpl.Category == "" {
			t.Error("Template should have Category")
		}
	}
}

func TestRegistryListByCategory(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	// Test CI category
	ciTemplates := registry.ListByCategory(CategoryCI)
	for _, tmpl := range ciTemplates {
		if tmpl.Category != CategoryCI {
			t.Errorf("Template %s has category %s, want %s", tmpl.ID, tmpl.Category, CategoryCI)
		}
	}

	// Test microservice category
	msTemplates := registry.ListByCategory(CategoryMicroservice)
	for _, tmpl := range msTemplates {
		if tmpl.Category != CategoryMicroservice {
			t.Errorf("Template %s has category %s, want %s", tmpl.ID, tmpl.Category, CategoryMicroservice)
		}
	}
}

func TestRegistryListByTag(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	dockerTemplates := registry.ListByTag("docker")
	for _, tmpl := range dockerTemplates {
		hasTag := false
		for _, tag := range tmpl.Tags {
			if strings.EqualFold(tag, "docker") {
				hasTag = true
				break
			}
		}
		if !hasTag {
			t.Errorf("Template %s should have docker tag", tmpl.ID)
		}
	}
}

func TestRegistryGetIDs(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	ids := registry.GetIDs()
	if len(ids) == 0 {
		t.Error("GetIDs() should return template IDs")
	}

	// All IDs should be non-empty
	for _, id := range ids {
		if id == "" {
			t.Error("Template ID should not be empty")
		}
	}
}

func TestWorkflowTemplateGetRequiredVariables(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tmpl, ok := registry.Get("ci-pipeline")
	if !ok {
		t.Fatal("Template ci-pipeline should exist")
	}

	required := tmpl.GetRequiredVariables()
	if len(required) == 0 {
		t.Error("CI pipeline should have required variables")
	}

	// All returned variables should be required
	for _, v := range required {
		if !v.Required {
			t.Errorf("Variable %s should be required", v.Name)
		}
	}
}

func TestWorkflowTemplateFormatHelp(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tmpl, ok := registry.Get("ci-pipeline")
	if !ok {
		t.Fatal("Template ci-pipeline should exist")
	}

	help := tmpl.FormatHelp()

	if !strings.Contains(help, tmpl.Name) {
		t.Error("Help should contain template name")
	}
	if !strings.Contains(help, tmpl.Description) {
		t.Error("Help should contain template description")
	}
	if !strings.Contains(help, "Category:") {
		t.Error("Help should show category")
	}
	if !strings.Contains(help, "Files:") {
		t.Error("Help should list files")
	}
	if !strings.Contains(help, "Variables:") {
		t.Error("Help should list variables")
	}
}

func TestWorkflowTemplateGenerate_DryRun(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tmpl, ok := registry.Get("ci-pipeline")
	if !ok {
		t.Fatal("Template ci-pipeline should exist")
	}

	config := GenerateConfig{
		OutputDir: "/tmp/test",
		Variables: map[string]string{
			"project_name": "test-project",
			"language":     "go",
		},
		DryRun: true,
	}

	result, err := tmpl.Generate(config)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(result.FilesCreated) == 0 {
		t.Error("Should have files to create")
	}
	if result.TotalBytes == 0 {
		t.Error("Should have bytes to write")
	}
}

func TestWorkflowTemplateGenerate_MissingRequired(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tmpl, ok := registry.Get("ci-pipeline")
	if !ok {
		t.Fatal("Template ci-pipeline should exist")
	}

	config := GenerateConfig{
		OutputDir: "/tmp/test",
		Variables: map[string]string{
			// Missing required variables
		},
		DryRun: true,
	}

	_, err = tmpl.Generate(config)
	if err == nil {
		t.Error("Generate() should error on missing required variables")
	}
}

func TestWorkflowTemplateGenerate_RealFiles(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tmpl, ok := registry.Get("ci-pipeline")
	if !ok {
		t.Fatal("Template ci-pipeline should exist")
	}

	// Create temp directory
	tmpDir := t.TempDir()

	config := GenerateConfig{
		OutputDir: tmpDir,
		Variables: map[string]string{
			"project_name":          "test-project",
			"language":              "go",
			"deploy_target":         "docker",
			"enable_security_scan":  "true",
		},
		DryRun: false,
	}

	result, err := tmpl.Generate(config)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify files were created
	for _, file := range result.FilesCreated {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("File %s should exist", file)
		}
	}

	// Verify CI workflow content
	ciPath := filepath.Join(tmpDir, ".github/workflows/ci.yaml")
	content, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("Failed to read CI workflow: %v", err)
	}

	if !strings.Contains(string(content), "test-project") {
		t.Error("CI workflow should contain project name")
	}
	if !strings.Contains(string(content), "setup-go") {
		t.Error("CI workflow should contain Go setup for Go projects")
	}
}

func TestGenerateResultPrintResult(t *testing.T) {
	result := &GenerateResult{
		FilesCreated: []string{"/tmp/file1.yaml", "/tmp/file2.yaml"},
		FilesSkipped: []string{"/tmp/skipped.yaml"},
		Errors:       []error{},
		TotalBytes:   1024,
		TemplateID:   "test-template",
		TemplateName: "Test Template",
	}

	var buf bytes.Buffer
	result.PrintResult(&buf, false)

	output := buf.String()

	if !strings.Contains(output, "Test Template") {
		t.Error("Output should contain template name")
	}
	if !strings.Contains(output, "file1.yaml") {
		t.Error("Output should list created files")
	}
	if !strings.Contains(output, "skipped.yaml") {
		t.Error("Output should list skipped files")
	}
	if !strings.Contains(output, "2 files") {
		t.Error("Output should show file count")
	}
}

func TestGenerateResultPrintResult_DryRun(t *testing.T) {
	result := &GenerateResult{
		FilesCreated: []string{"/tmp/file1.yaml"},
		TotalBytes:   512,
		TemplateID:   "test-template",
		TemplateName: "Test Template",
	}

	var buf bytes.Buffer
	result.PrintResult(&buf, true)

	output := buf.String()

	if !strings.Contains(output, "Would generate") {
		t.Error("Dry run output should indicate simulation")
	}
}

func TestAvailableWorkflows(t *testing.T) {
	output := AvailableWorkflows()

	if !strings.Contains(output, "Available workflow templates") {
		t.Error("Output should have header")
	}
	// Should contain category headers
	for _, cat := range []string{"Ci", "Microservice", "Data", "Monorepo"} {
		if !strings.Contains(output, cat) {
			t.Errorf("Output should contain category: %s", cat)
		}
	}
}

func TestMicroserviceTemplate(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tmpl, ok := registry.Get("microservice")
	if !ok {
		t.Fatal("Template microservice should exist")
	}

	if tmpl.Category != CategoryMicroservice {
		t.Errorf("Category = %s, want microservice", tmpl.Category)
	}

	// Should have Dockerfile
	hasDockerfile := false
	for _, file := range tmpl.Files {
		if file.Path == "Dockerfile" {
			hasDockerfile = true
			break
		}
	}
	if !hasDockerfile {
		t.Error("Microservice template should have Dockerfile")
	}
}

func TestDataPipelineTemplate(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tmpl, ok := registry.Get("data-pipeline")
	if !ok {
		t.Fatal("Template data-pipeline should exist")
	}

	if tmpl.Category != CategoryData {
		t.Errorf("Category = %s, want data", tmpl.Category)
	}
}

func TestMonorepoTemplate(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tmpl, ok := registry.Get("monorepo")
	if !ok {
		t.Fatal("Template monorepo should exist")
	}

	if tmpl.Category != CategoryMonorepo {
		t.Errorf("Category = %s, want monorepo", tmpl.Category)
	}

	// Should have package.json
	hasPackageJSON := false
	for _, file := range tmpl.Files {
		if file.Path == "package.json" {
			hasPackageJSON = true
			break
		}
	}
	if !hasPackageJSON {
		t.Error("Monorepo template should have package.json")
	}
}
