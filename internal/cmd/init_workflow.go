package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/templates/workflows"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var (
	workflowVars      []string
	workflowOutputDir string
	workflowDryRun    bool
	workflowFlag      string
)

var initWorkflowCmd = &cobra.Command{
	Use:   "workflow <template-id>",
	Short: "Initialize project with workflow template",
	Long: `Generate CI/CD, microservice, or project structure from workflow templates.

Workflow templates provide pre-configured files and configurations for common
project patterns like CI pipelines, microservices, data pipelines, and monorepos.

Examples:
  # Generate CI pipeline
  specular init workflow ci-pipeline --var project_name=myapp --var language=go

  # Generate microservice structure
  specular init workflow microservice --var service_name=api

  # Preview files without creating
  specular init workflow ci-pipeline --var project_name=test --dry-run

  # Generate to specific directory
  specular init workflow monorepo --output-dir ./projects/new-project
`,
	Args: cobra.ExactArgs(1),
	RunE: runInitWorkflow,
}

var initWorkflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available workflow templates",
	Long: `List all available workflow templates with descriptions.

Shows templates organized by category (CI, Microservice, Data, Monorepo).
Use 'specular init workflow <template-id> --help' for template-specific details.

Examples:
  # List all templates
  specular init workflow list

  # Get details about a specific template
  specular init workflow ci-pipeline --help
`,
	RunE: runInitWorkflowList,
}

var initWorkflowShowCmd = &cobra.Command{
	Use:   "show <template-id>",
	Short: "Show details about a workflow template",
	Long: `Display detailed information about a workflow template including
variables, files, and usage examples.

Examples:
  specular init workflow show ci-pipeline
  specular init workflow show microservice
`,
	Args: cobra.ExactArgs(1),
	RunE: runInitWorkflowShow,
}

func init() {
	// Workflow command flags
	initWorkflowCmd.Flags().StringArrayVar(&workflowVars, "var", []string{}, "Set template variable (repeatable, format: name=value)")
	initWorkflowCmd.Flags().StringVar(&workflowOutputDir, "output-dir", ".", "Output directory for generated files")
	initWorkflowCmd.Flags().BoolVar(&workflowDryRun, "dry-run", false, "Preview files without creating")

	// Add --workflow flag to main init command
	initCmd.Flags().StringVar(&workflowFlag, "workflow", "", "Generate workflow template (shorthand for 'init workflow <id>')")

	// Add subcommands
	initWorkflowCmd.AddCommand(initWorkflowListCmd)
	initWorkflowCmd.AddCommand(initWorkflowShowCmd)
	initCmd.AddCommand(initWorkflowCmd)
}

func runInitWorkflow(cmd *cobra.Command, args []string) error {
	templateID := args[0]

	// Load registry
	registry, err := workflows.NewRegistry()
	if err != nil {
		return ux.FormatError(err, "loading workflow templates")
	}

	// Get template
	tmpl, ok := registry.Get(templateID)
	if !ok {
		// Show similar templates
		ids := registry.GetIDs()
		suggestions := findSimilarTemplates(templateID, ids)
		if len(suggestions) > 0 {
			return fmt.Errorf("workflow template '%s' not found. Did you mean: %s?", templateID, strings.Join(suggestions, ", "))
		}
		return fmt.Errorf("workflow template '%s' not found. Run 'specular init workflow list' to see available templates", templateID)
	}

	// Parse variables
	variables := make(map[string]string)
	for _, v := range workflowVars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid variable format '%s': expected name=value", v)
		}
		variables[parts[0]] = parts[1]
	}

	// Resolve output directory
	outputDir, err := filepath.Abs(workflowOutputDir)
	if err != nil {
		return ux.FormatError(err, "resolving output directory")
	}

	// Check for missing required variables
	missing := checkMissingVariables(tmpl, variables)
	if len(missing) > 0 {
		return fmt.Errorf("missing required variables: %s\nUse --var name=value to provide them", strings.Join(missing, ", "))
	}

	// Generate workflow
	config := workflows.GenerateConfig{
		OutputDir: outputDir,
		Variables: variables,
		DryRun:    workflowDryRun,
	}

	result, err := tmpl.Generate(config)
	if err != nil {
		return ux.FormatError(err, "generating workflow")
	}

	// Print result
	result.PrintResult(cmd.OutOrStdout(), workflowDryRun)

	if workflowDryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "\nRun without --dry-run to create these files.")
	}

	return nil
}

func runInitWorkflowList(cmd *cobra.Command, args []string) error {
	registry, err := workflows.NewRegistry()
	if err != nil {
		return ux.FormatError(err, "loading workflow templates")
	}

	fmt.Fprintln(cmd.OutOrStdout(), "")
	fmt.Fprintln(cmd.OutOrStdout(), "Available Workflow Templates")
	fmt.Fprintln(cmd.OutOrStdout(), "============================")
	fmt.Fprintln(cmd.OutOrStdout(), "")

	// Group by category
	categories := []workflows.Category{
		workflows.CategoryCI,
		workflows.CategoryMicroservice,
		workflows.CategoryData,
		workflows.CategoryMonorepo,
	}

	for _, cat := range categories {
		templates := registry.ListByCategory(cat)
		if len(templates) == 0 {
			continue
		}

		// Sort templates by ID
		sort.Slice(templates, func(i, j int) bool {
			return templates[i].ID < templates[j].ID
		})

		fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", formatCategoryName(cat))
		for _, tmpl := range templates {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-20s %s\n", tmpl.ID, tmpl.Description)
			if len(tmpl.Tags) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-20s Tags: %s\n", "", strings.Join(tmpl.Tags, ", "))
			}
		}
		fmt.Fprintln(cmd.OutOrStdout(), "")
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Usage:")
	fmt.Fprintln(cmd.OutOrStdout(), "  specular init workflow <template-id> --var name=value")
	fmt.Fprintln(cmd.OutOrStdout(), "")
	fmt.Fprintln(cmd.OutOrStdout(), "For template details:")
	fmt.Fprintln(cmd.OutOrStdout(), "  specular init workflow show <template-id>")
	fmt.Fprintln(cmd.OutOrStdout(), "")

	return nil
}

func runInitWorkflowShow(cmd *cobra.Command, args []string) error {
	templateID := args[0]

	registry, err := workflows.NewRegistry()
	if err != nil {
		return ux.FormatError(err, "loading workflow templates")
	}

	tmpl, ok := registry.Get(templateID)
	if !ok {
		return fmt.Errorf("workflow template '%s' not found", templateID)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "")
	fmt.Fprint(cmd.OutOrStdout(), tmpl.FormatHelp())
	fmt.Fprintln(cmd.OutOrStdout(), "")

	// Show example usage
	fmt.Fprintln(cmd.OutOrStdout(), "Example usage:")
	example := fmt.Sprintf("  specular init workflow %s", tmpl.ID)
	for _, v := range tmpl.GetRequiredVariables() {
		example += fmt.Sprintf(" --var %s=<value>", v.Name)
	}
	fmt.Fprintln(cmd.OutOrStdout(), example)
	fmt.Fprintln(cmd.OutOrStdout(), "")

	return nil
}

// checkMissingVariables returns the names of required variables that are missing
func checkMissingVariables(tmpl *workflows.WorkflowTemplate, provided map[string]string) []string {
	var missing []string
	for _, v := range tmpl.Variables {
		if v.Required {
			if _, ok := provided[v.Name]; !ok && v.Default == "" {
				missing = append(missing, v.Name)
			}
		}
	}
	return missing
}

// findSimilarTemplates finds templates with similar IDs
func findSimilarTemplates(target string, ids []string) []string {
	var similar []string
	target = strings.ToLower(target)

	for _, id := range ids {
		// Check if target is a substring or similar
		if strings.Contains(strings.ToLower(id), target) ||
			strings.Contains(target, strings.ToLower(id)) ||
			levenshteinDistance(target, strings.ToLower(id)) <= 3 {
			similar = append(similar, id)
		}
	}

	// Limit to 3 suggestions
	if len(similar) > 3 {
		similar = similar[:3]
	}

	return similar
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create distance matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// formatCategoryName formats a category for display
func formatCategoryName(cat workflows.Category) string {
	switch cat {
	case workflows.CategoryCI:
		return "CI/CD Pipelines"
	case workflows.CategoryMicroservice:
		return "Microservices"
	case workflows.CategoryData:
		return "Data Pipelines"
	case workflows.CategoryMonorepo:
		return "Monorepo"
	default:
		return string(cat)
	}
}

// RunWorkflowFromFlag handles the --workflow flag on init command
// Called from runInit when --workflow is specified
func RunWorkflowFromFlag(cmd *cobra.Command, workflowID string) error {
	// Set up for workflow generation
	registry, err := workflows.NewRegistry()
	if err != nil {
		return ux.FormatError(err, "loading workflow templates")
	}

	tmpl, ok := registry.Get(workflowID)
	if !ok {
		ids := registry.GetIDs()
		suggestions := findSimilarTemplates(workflowID, ids)
		if len(suggestions) > 0 {
			return fmt.Errorf("workflow template '%s' not found. Did you mean: %s?", workflowID, strings.Join(suggestions, ", "))
		}
		return fmt.Errorf("workflow template '%s' not found", workflowID)
	}

	// Parse variables from --var flags
	variables := make(map[string]string)
	for _, v := range workflowVars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}

	// Check for missing required variables
	missing := checkMissingVariables(tmpl, variables)
	if len(missing) > 0 {
		return fmt.Errorf("missing required variables: %s\nUse --var name=value to provide them", strings.Join(missing, ", "))
	}

	// Get output directory
	outputDir := "."
	if workflowOutputDir != "" && workflowOutputDir != "." {
		outputDir = workflowOutputDir
	}
	absDir, err := filepath.Abs(outputDir)
	if err != nil {
		return ux.FormatError(err, "resolving output directory")
	}

	// Generate workflow
	config := workflows.GenerateConfig{
		OutputDir: absDir,
		Variables: variables,
		DryRun:    initDryRun, // Use init's --dry-run flag
	}

	result, err := tmpl.Generate(config)
	if err != nil {
		return ux.FormatError(err, "generating workflow")
	}

	// Print result
	result.PrintResult(os.Stdout, initDryRun)

	if initDryRun {
		fmt.Println("\nRun without --dry-run to create these files.")
	}

	return nil
}

// GetWorkflowIDs returns all workflow template IDs for shell completion
func GetWorkflowIDs() []string {
	registry, err := workflows.NewRegistry()
	if err != nil {
		return nil
	}
	return registry.GetIDs()
}
