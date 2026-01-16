package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/configval"
	"github.com/felixgeelhaar/specular/internal/ux"
	"github.com/felixgeelhaar/specular/internal/validate"
)

var configValidateCmd = &cobra.Command{
	Use:     "validate [files...]",
	Aliases: []string{"check", "lint"},
	Short:   "Validate configuration files",
	Long: `Validate YAML configuration files against schemas with detailed error reporting.

If no files are specified, validates all configuration files in .specular/ directory.

Supports the following configuration types:
  - providers.yaml  (provider configuration)
  - routing.yaml     (routing configuration)
  - spec.yaml       (specification files)
  - policy.yaml     (policy configuration)
  - slo.yaml        (SLO definitions)

Examples:
  # Validate all configs in .specular/
  specular config validate

  # Validate specific files
  specular config validate .specular/providers.yaml .specular/routing.yaml

  # Strict mode (warnings become errors)
  specular config validate --strict

  # JSON output for CI integration
  specular config validate --format json
`,
	RunE: runConfigValidate,
}

var configFixCmd = &cobra.Command{
	Use:   "fix [file]",
	Short: "Auto-fix configuration issues",
	Long: `Apply automatic fixes to configuration files based on validation errors.

Fixes include:
  - Adding missing required fields with default values
  - Correcting invalid enum values to closest matches
  - Setting proper types for mistyped values

Examples:
  # Preview fixes without applying
  specular config fix .specular/providers.yaml --dry-run

  # Apply fixes with backup
  specular config fix .specular/providers.yaml --backup

  # Interactive mode - confirm each fix
  specular config fix .specular/providers.yaml --interactive
`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigFix,
}

func init() {
	// Validate command flags
	configValidateCmd.Flags().Bool("strict", false, "Treat warnings as errors")
	configValidateCmd.Flags().String("format", "text", "Output format: text, json, sarif")
	configValidateCmd.Flags().String("schema", "", "Custom schema file (overrides auto-detection)")

	// Fix command flags
	configFixCmd.Flags().Bool("dry-run", false, "Show fixes without applying")
	configFixCmd.Flags().Bool("backup", true, "Create backup before fixing")
	configFixCmd.Flags().Bool("interactive", false, "Confirm each fix interactively")

	// Add to config command
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configFixCmd)
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	strict, _ := cmd.Flags().GetBool("strict")
	format, _ := cmd.Flags().GetString("format")
	customSchema, _ := cmd.Flags().GetString("schema")

	if customSchema != "" {
		return fmt.Errorf("custom schema not yet supported")
	}

	// Determine files to validate
	var files []string
	if len(args) > 0 {
		files = args
	} else {
		// Auto-discover config files in .specular/
		discovered, err := discoverConfigFiles()
		if err != nil {
			return ux.FormatError(err, "discovering config files")
		}
		files = discovered
	}

	if len(files) == 0 {
		fmt.Println("No configuration files found to validate.")
		return nil
	}

	// Validate each file
	var allResults []ConfigValidationOutput
	hasErrors := false
	hasWarnings := false

	for _, file := range files {
		ctx, err := configval.NewValidationContext(file)
		if err != nil {
			return ux.FormatError(err, fmt.Sprintf("reading %s", file))
		}

		result := configval.ValidateConfig(ctx)

		output := ConfigValidationOutput{
			File:     file,
			Valid:    !result.HasErrors(),
			Errors:   convertValidationIssues(result.Errors()),
			Warnings: convertValidationIssues(result.Warnings()),
		}
		allResults = append(allResults, output)

		if result.HasErrors() {
			hasErrors = true
		}
		if result.HasWarnings() {
			hasWarnings = true
		}
	}

	// Output results
	switch format {
	case "json":
		return outputValidationJSON(allResults)
	case "sarif":
		return outputValidationSARIF(allResults)
	default:
		printValidationText(allResults, cmd.OutOrStdout())
	}

	// Determine exit status
	if hasErrors {
		return fmt.Errorf("validation failed with errors")
	}
	if strict && hasWarnings {
		return fmt.Errorf("validation failed with warnings (strict mode)")
	}

	return nil
}

func runConfigFix(cmd *cobra.Command, args []string) error {
	file := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	backup, _ := cmd.Flags().GetBool("backup")
	interactive, _ := cmd.Flags().GetBool("interactive")

	// Create validation context
	ctx, err := configval.NewValidationContext(file)
	if err != nil {
		return ux.FormatError(err, fmt.Sprintf("reading %s", file))
	}

	// Get fix suggestions
	fixResult := configval.SuggestFixes(ctx)
	if len(fixResult.Suggestions) == 0 {
		fmt.Printf("No fixes needed for %s\n", file)
		return nil
	}

	// Print suggestions
	configval.PrintFixSuggestions(fixResult, cmd.OutOrStdout())

	if dryRun {
		fmt.Println("\nDry run - no changes made.")
		return nil
	}

	// Filter suggestions for application
	suggestions := fixResult.Suggestions

	// Interactive mode - confirm each fix
	if interactive {
		suggestions = confirmFixSuggestions(suggestions)
		if len(suggestions) == 0 {
			fmt.Println("No fixes selected.")
			return nil
		}
	}

	// Create backup if requested
	if backup {
		backupPath := file + ".bak"
		if err := copyConfigFile(file, backupPath); err != nil {
			return ux.FormatError(err, "creating backup")
		}
		fmt.Printf("Backup created: %s\n", backupPath)
	}

	// Apply fixes (only auto-fixable ones unless interactive)
	autoOnly := !interactive
	applyResult, err := configval.ApplyFixes(ctx, suggestions, autoOnly)
	if err != nil {
		return ux.FormatError(err, "applying fixes")
	}

	// Print result
	configval.PrintFixResult(applyResult, cmd.OutOrStdout())

	return nil
}

// ConfigValidationOutput represents the output for a single file validation
type ConfigValidationOutput struct {
	File     string              `json:"file"`
	Valid    bool                `json:"valid"`
	Errors   []ConfigIssueOutput `json:"errors,omitempty"`
	Warnings []ConfigIssueOutput `json:"warnings,omitempty"`
}

// ConfigIssueOutput represents a validation issue for output
type ConfigIssueOutput struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// discoverConfigFiles finds config files in .specular/ directory
func discoverConfigFiles() ([]string, error) {
	specularDir := ".specular"
	if _, err := os.Stat(specularDir); os.IsNotExist(err) {
		// Try current directory for common config files
		return discoverConfigFilesInDir(".")
	}
	return discoverConfigFilesInDir(specularDir)
}

func discoverConfigFilesInDir(dir string) ([]string, error) {
	configPatterns := []string{
		"providers.yaml", "providers.yml",
		"routing.yaml", "router.yml",
		"spec.yaml", "spec.yml",
		"policy.yaml", "policy.yml",
		"slo.yaml", "slo.yml",
		"config.yaml", "config.yml",
	}

	var files []string
	for _, pattern := range configPatterns {
		path := filepath.Join(dir, pattern)
		if _, err := os.Stat(path); err == nil {
			files = append(files, path)
		}
	}

	return files, nil
}

// convertValidationIssues converts validate.ValidationIssue to ConfigIssueOutput
func convertValidationIssues(issues []validate.ValidationIssue) []ConfigIssueOutput {
	var result []ConfigIssueOutput
	for _, issue := range issues {
		result = append(result, ConfigIssueOutput{
			Field:   issue.Field,
			Message: issue.Message,
			Value:   issue.Value,
		})
	}
	return result
}

// printValidationText prints validation results as text
func printValidationText(results []ConfigValidationOutput, w io.Writer) {
	for _, r := range results {
		if r.Valid && len(r.Warnings) == 0 {
			fmt.Fprintf(w, "OK %s\n", r.File)
			continue
		}

		fmt.Fprintf(w, "\n%s:\n", r.File)

		for _, e := range r.Errors {
			fmt.Fprintf(w, "  ERROR [%s]: %s", e.Field, e.Message)
			if e.Value != "" {
				fmt.Fprintf(w, " (got: %s)", e.Value)
			}
			fmt.Fprintln(w)
		}

		for _, e := range r.Warnings {
			fmt.Fprintf(w, "  WARN  [%s]: %s", e.Field, e.Message)
			if e.Value != "" {
				fmt.Fprintf(w, " (got: %s)", e.Value)
			}
			fmt.Fprintln(w)
		}
	}

	// Summary
	totalErrors := 0
	totalWarnings := 0
	validFiles := 0
	for _, r := range results {
		totalErrors += len(r.Errors)
		totalWarnings += len(r.Warnings)
		if r.Valid {
			validFiles++
		}
	}

	fmt.Fprintf(w, "\nSummary: %d/%d files valid", validFiles, len(results))
	if totalErrors > 0 {
		fmt.Fprintf(w, ", %d errors", totalErrors)
	}
	if totalWarnings > 0 {
		fmt.Fprintf(w, ", %d warnings", totalWarnings)
	}
	fmt.Fprintln(w)
}

// outputValidationJSON prints validation results as JSON
func outputValidationJSON(results []ConfigValidationOutput) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// ConfigSARIFOutput represents SARIF format output
type ConfigSARIFOutput struct {
	Version string           `json:"version"`
	Schema  string           `json:"$schema"`
	Runs    []ConfigSARIFRun `json:"runs"`
}

// ConfigSARIFRun represents a SARIF run
type ConfigSARIFRun struct {
	Tool    ConfigSARIFTool     `json:"tool"`
	Results []ConfigSARIFResult `json:"results"`
}

// ConfigSARIFTool represents a SARIF tool
type ConfigSARIFTool struct {
	Driver ConfigSARIFDriver `json:"driver"`
}

// ConfigSARIFDriver represents a SARIF driver
type ConfigSARIFDriver struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ConfigSARIFResult represents a SARIF result
type ConfigSARIFResult struct {
	RuleID    string                `json:"ruleId"`
	Level     string                `json:"level"`
	Message   ConfigSARIFMessage    `json:"message"`
	Locations []ConfigSARIFLocation `json:"locations"`
}

// ConfigSARIFMessage represents a SARIF message
type ConfigSARIFMessage struct {
	Text string `json:"text"`
}

// ConfigSARIFLocation represents a SARIF location
type ConfigSARIFLocation struct {
	PhysicalLocation ConfigSARIFPhysicalLocation `json:"physicalLocation"`
}

// ConfigSARIFPhysicalLocation represents a SARIF physical location
type ConfigSARIFPhysicalLocation struct {
	ArtifactLocation ConfigSARIFArtifactLocation `json:"artifactLocation"`
}

// ConfigSARIFArtifactLocation represents a SARIF artifact location
type ConfigSARIFArtifactLocation struct {
	URI string `json:"uri"`
}

// outputValidationSARIF prints validation results in SARIF format
func outputValidationSARIF(results []ConfigValidationOutput) error {
	sarif := ConfigSARIFOutput{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Runs: []ConfigSARIFRun{
			{
				Tool: ConfigSARIFTool{
					Driver: ConfigSARIFDriver{
						Name:    "specular-config-validate",
						Version: "1.4.0",
					},
				},
				Results: []ConfigSARIFResult{},
			},
		},
	}

	for _, r := range results {
		for _, e := range r.Errors {
			sarif.Runs[0].Results = append(sarif.Runs[0].Results, ConfigSARIFResult{
				RuleID: "config-error",
				Level:  "error",
				Message: ConfigSARIFMessage{
					Text: fmt.Sprintf("[%s] %s", e.Field, e.Message),
				},
				Locations: []ConfigSARIFLocation{
					{
						PhysicalLocation: ConfigSARIFPhysicalLocation{
							ArtifactLocation: ConfigSARIFArtifactLocation{
								URI: r.File,
							},
						},
					},
				},
			})
		}
		for _, e := range r.Warnings {
			sarif.Runs[0].Results = append(sarif.Runs[0].Results, ConfigSARIFResult{
				RuleID: "config-warning",
				Level:  "warning",
				Message: ConfigSARIFMessage{
					Text: fmt.Sprintf("[%s] %s", e.Field, e.Message),
				},
				Locations: []ConfigSARIFLocation{
					{
						PhysicalLocation: ConfigSARIFPhysicalLocation{
							ArtifactLocation: ConfigSARIFArtifactLocation{
								URI: r.File,
							},
						},
					},
				},
			})
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sarif)
}

// confirmFixSuggestions interactively confirms which fixes to apply
func confirmFixSuggestions(suggestions []configval.FixSuggestion) []configval.FixSuggestion {
	var confirmed []configval.FixSuggestion

	for _, s := range suggestions {
		fmt.Printf("\nFix: %s\n", s.Description)
		fmt.Printf("  Field: %s\n", s.Field)
		if s.OldValue != nil {
			fmt.Printf("  Current: %v\n", s.OldValue)
		}
		fmt.Printf("  New: %v\n", s.NewValue)
		fmt.Print("Apply this fix? [y/N]: ")

		var response string
		fmt.Scanln(&response)
		if strings.EqualFold(response, "y") || strings.EqualFold(response, "yes") {
			confirmed = append(confirmed, s)
		}
	}

	return confirmed
}

// copyConfigFile creates a copy of a file
func copyConfigFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}
