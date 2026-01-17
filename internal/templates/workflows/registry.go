// Package workflows provides workflow templates for common project types.
// It includes templates for CI/CD pipelines, data pipelines, microservices,
// and monorepo configurations.
package workflows

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var workflowFS embed.FS

// Category represents a workflow template category
type Category string

const (
	// CategoryCI represents CI/CD pipeline templates
	CategoryCI Category = "ci"
	// CategoryData represents data pipeline templates
	CategoryData Category = "data"
	// CategoryMicroservice represents microservice templates
	CategoryMicroservice Category = "microservice"
	// CategoryMonorepo represents monorepo templates
	CategoryMonorepo Category = "monorepo"
)

// TemplateFile represents a file to be generated from a template
type TemplateFile struct {
	Path        string `yaml:"path"`
	Template    string `yaml:"template"`
	Description string `yaml:"description"`
	Optional    bool   `yaml:"optional,omitempty"`
}

// TemplateVariable represents a variable required by a template
type TemplateVariable struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Type        string   `yaml:"type"` // string, boolean, choice
	Default     string   `yaml:"default,omitempty"`
	Required    bool     `yaml:"required"`
	Choices     []string `yaml:"choices,omitempty"`
}

// WorkflowTemplate represents a complete workflow template
type WorkflowTemplate struct {
	ID          string             `yaml:"id"`
	Name        string             `yaml:"name"`
	Description string             `yaml:"description"`
	Category    Category           `yaml:"category"`
	Tags        []string           `yaml:"tags"`
	Files       []TemplateFile     `yaml:"files"`
	Variables   []TemplateVariable `yaml:"variables"`
}

// Registry holds available workflow templates
type Registry struct {
	templates map[string]*WorkflowTemplate
}

// NewRegistry creates a new workflow template registry
func NewRegistry() (*Registry, error) {
	r := &Registry{
		templates: make(map[string]*WorkflowTemplate),
	}

	// Load embedded YAML files
	entries, err := workflowFS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read workflow templates: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		data, readErr := workflowFS.ReadFile(entry.Name())
		if readErr != nil {
			return nil, fmt.Errorf("read template %s: %w", entry.Name(), readErr)
		}

		var tmpl WorkflowTemplate
		if parseErr := yaml.Unmarshal(data, &tmpl); parseErr != nil {
			return nil, fmt.Errorf("parse template %s: %w", entry.Name(), parseErr)
		}

		r.templates[tmpl.ID] = &tmpl
	}

	return r, nil
}

// Get returns a workflow template by ID
func (r *Registry) Get(id string) (*WorkflowTemplate, bool) {
	tmpl, ok := r.templates[id]
	return tmpl, ok
}

// List returns all available workflow templates
func (r *Registry) List() []*WorkflowTemplate {
	result := make([]*WorkflowTemplate, 0, len(r.templates))
	for _, tmpl := range r.templates {
		result = append(result, tmpl)
	}
	return result
}

// ListByCategory returns workflow templates in a specific category
func (r *Registry) ListByCategory(category Category) []*WorkflowTemplate {
	var result []*WorkflowTemplate
	for _, tmpl := range r.templates {
		if tmpl.Category == category {
			result = append(result, tmpl)
		}
	}
	return result
}

// ListByTag returns workflow templates matching a tag
func (r *Registry) ListByTag(tag string) []*WorkflowTemplate {
	var result []*WorkflowTemplate
	for _, tmpl := range r.templates {
		for _, t := range tmpl.Tags {
			if strings.EqualFold(t, tag) {
				result = append(result, tmpl)
				break
			}
		}
	}
	return result
}

// GetIDs returns all template IDs
func (r *Registry) GetIDs() []string {
	ids := make([]string, 0, len(r.templates))
	for id := range r.templates {
		ids = append(ids, id)
	}
	return ids
}

// GenerateConfig holds configuration for generating workflow files
type GenerateConfig struct {
	OutputDir   string
	Variables   map[string]string
	DryRun      bool
	Interactive bool
}

// GenerateResult holds the result of generating workflow files
type GenerateResult struct {
	FilesCreated []string
	FilesSkipped []string
	Errors       []error
	TotalBytes   int64
	TemplateID   string
	TemplateName string
}

// applyVariableDefaults applies default values to missing variables
func (t *WorkflowTemplate) applyVariableDefaults(config *GenerateConfig) error {
	// Validate required variables
	for _, v := range t.Variables {
		if v.Required {
			if _, ok := config.Variables[v.Name]; !ok {
				if v.Default != "" {
					config.Variables[v.Name] = v.Default
				} else {
					return fmt.Errorf("missing required variable: %s", v.Name)
				}
			}
		}
	}

	// Apply defaults for missing optional variables
	for _, v := range t.Variables {
		if _, ok := config.Variables[v.Name]; !ok && v.Default != "" {
			config.Variables[v.Name] = v.Default
		}
	}
	return nil
}

// generateFile generates a single file from a template
func generateFile(file TemplateFile, config GenerateConfig, result *GenerateResult) {
	outputPath := filepath.Join(config.OutputDir, file.Path)

	// Skip optional files if not needed
	if file.Optional {
		if skip, ok := config.Variables["skip_"+filepath.Base(file.Path)]; ok && skip == "true" {
			result.FilesSkipped = append(result.FilesSkipped, outputPath)
			return
		}
	}

	// Parse and execute template
	tmpl, err := template.New(file.Path).Parse(file.Template)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("parse template for %s: %w", file.Path, err))
		return
	}

	var content strings.Builder
	if execErr := tmpl.Execute(&content, config.Variables); execErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("execute template for %s: %w", file.Path, execErr))
		return
	}

	if config.DryRun {
		result.FilesCreated = append(result.FilesCreated, outputPath)
		result.TotalBytes += int64(content.Len())
		return
	}

	// Create directory if needed
	dir := filepath.Dir(outputPath)
	if mkdirErr := os.MkdirAll(dir, 0750); mkdirErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("create directory %s: %w", dir, mkdirErr))
		return
	}

	// Write file
	if writeErr := os.WriteFile(outputPath, []byte(content.String()), 0600); writeErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("write file %s: %w", outputPath, writeErr))
		return
	}

	result.FilesCreated = append(result.FilesCreated, outputPath)
	result.TotalBytes += int64(content.Len())
}

// Generate creates workflow files from a template
func (t *WorkflowTemplate) Generate(config GenerateConfig) (*GenerateResult, error) {
	result := &GenerateResult{
		FilesCreated: make([]string, 0),
		FilesSkipped: make([]string, 0),
		Errors:       make([]error, 0),
		TemplateID:   t.ID,
		TemplateName: t.Name,
	}

	if err := t.applyVariableDefaults(&config); err != nil {
		return nil, err
	}

	// Generate each file
	for _, file := range t.Files {
		generateFile(file, config, result)
	}

	return result, nil
}

// PrintResult prints the generation result to a writer
func (r *GenerateResult) PrintResult(w io.Writer, dryRun bool) {
	prefix := "Generated"
	if dryRun {
		prefix = "Would generate"
	}

	_, _ = fmt.Fprintf(w, "\n%s workflow: %s (%s)\n\n", prefix, r.TemplateName, r.TemplateID)

	if len(r.FilesCreated) > 0 {
		_, _ = fmt.Fprintf(w, "Files %s:\n", strings.ToLower(prefix))
		for _, file := range r.FilesCreated {
			_, _ = fmt.Fprintf(w, "  + %s\n", file)
		}
	}

	if len(r.FilesSkipped) > 0 {
		_, _ = fmt.Fprintln(w, "\nFiles skipped:")
		for _, file := range r.FilesSkipped {
			_, _ = fmt.Fprintf(w, "  - %s\n", file)
		}
	}

	if len(r.Errors) > 0 {
		_, _ = fmt.Fprintln(w, "\nErrors:")
		for _, err := range r.Errors {
			_, _ = fmt.Fprintf(w, "  ! %v\n", err)
		}
	}

	_, _ = fmt.Fprintf(w, "\nTotal: %d files, %d bytes\n", len(r.FilesCreated), r.TotalBytes)
}

// GetRequiredVariables returns the list of required variables
func (t *WorkflowTemplate) GetRequiredVariables() []TemplateVariable {
	var required []TemplateVariable
	for _, v := range t.Variables {
		if v.Required {
			required = append(required, v)
		}
	}
	return required
}

// FormatHelp returns a formatted help string for the template
func (t *WorkflowTemplate) FormatHelp() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s (%s)\n", t.Name, t.ID))
	sb.WriteString(fmt.Sprintf("  %s\n\n", t.Description))

	if len(t.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(t.Tags, ", ")))
	}

	sb.WriteString(fmt.Sprintf("  Category: %s\n\n", t.Category))

	if len(t.Files) > 0 {
		sb.WriteString("  Files:\n")
		for _, file := range t.Files {
			optional := ""
			if file.Optional {
				optional = " (optional)"
			}
			sb.WriteString(fmt.Sprintf("    • %s%s\n", file.Path, optional))
			if file.Description != "" {
				sb.WriteString(fmt.Sprintf("      %s\n", file.Description))
			}
		}
		sb.WriteString("\n")
	}

	if len(t.Variables) > 0 {
		sb.WriteString("  Variables:\n")
		for _, v := range t.Variables {
			required := ""
			if v.Required {
				required = " (required)"
			}
			sb.WriteString(fmt.Sprintf("    • %s%s\n", v.Name, required))
			if v.Description != "" {
				sb.WriteString(fmt.Sprintf("      %s\n", v.Description))
			}
			if v.Default != "" {
				sb.WriteString(fmt.Sprintf("      Default: %s\n", v.Default))
			}
			if len(v.Choices) > 0 {
				sb.WriteString(fmt.Sprintf("      Choices: %s\n", strings.Join(v.Choices, ", ")))
			}
		}
	}

	return sb.String()
}

// AvailableWorkflows returns a formatted list of available workflows
func AvailableWorkflows() string {
	registry, err := NewRegistry()
	if err != nil {
		return "Error loading workflow templates"
	}

	var sb strings.Builder
	sb.WriteString("Available workflow templates:\n\n")

	// Group by category
	categories := []Category{CategoryCI, CategoryData, CategoryMicroservice, CategoryMonorepo}
	for _, cat := range categories {
		templates := registry.ListByCategory(cat)
		if len(templates) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("%s:\n", cases.Title(language.English).String(string(cat))))
		for _, tmpl := range templates {
			sb.WriteString(fmt.Sprintf("  • %s - %s\n", tmpl.ID, tmpl.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
