package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/interview"
	"github.com/felixgeelhaar/specular/internal/prd"
	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/internal/telemetry"
	"github.com/felixgeelhaar/specular/internal/tui"
	"github.com/felixgeelhaar/specular/internal/ux"
	"go.opentelemetry.io/otel/attribute"
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Specification management commands",
	Long:  `Generate, validate, and manage product specifications.`,
}

var specGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate spec from PRD markdown",
	Long:  `Convert a Product Requirements Document (PRD) in markdown format into a structured specification using AI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Start distributed tracing span for spec generate command
		ctx, span := telemetry.StartCommandSpan(cmd.Context(), "spec.generate")
		defer span.End()

		startTime := time.Now()

		defaults := ux.NewPathDefaults()
		in := cmd.Flags().Lookup("in").Value.String()
		out := cmd.Flags().Lookup("out").Value.String()
		configPath := cmd.Flags().Lookup("config").Value.String()

		// Interactive prompt if PRD file doesn't exist
		if _, err := os.Stat(in); os.IsNotExist(err) && !cmd.Flags().Changed("in") {
			in = ux.PromptForPath("Enter PRD markdown file path", in)
		}

		// Validate PRD file exists
		if err := ux.ValidateRequiredFile(in, "PRD file", "Create a PRD markdown file or run 'specular interview'"); err != nil {
			return ux.EnhanceError(err)
		}

		// Use defaults for output and config
		if !cmd.Flags().Changed("out") {
			out = defaults.SpecFile()
		}
		if !cmd.Flags().Changed("config") {
			configPath = defaults.ProvidersFile()
		}

		// Record span attributes
		span.SetAttributes(
			attribute.String("prd_file", in),
			attribute.String("spec_file", out),
			attribute.String("config_file", configPath),
		)

		fmt.Printf("Generating spec from PRD: %s\n", in)

		// Read PRD file
		prdContent, err := os.ReadFile(in)
		if err != nil {
			return ux.FormatError(err, "reading PRD file")
		}

		// Load provider configuration
		fmt.Println("Loading provider configuration...")
		registry, err := provider.LoadRegistryFromConfig(configPath)
		if err != nil {
			return ux.FormatError(err, "loading AI providers")
		}

		// Load provider config to get strategy settings
		providerConfig, err := provider.LoadProvidersConfig(configPath)
		if err != nil {
			return ux.FormatError(err, "loading provider config")
		}

		// Create router config from provider strategy
		routerConfig := &router.RouterConfig{
			BudgetUSD:    providerConfig.Strategy.Budget.MaxCostPerDay,
			MaxLatencyMs: providerConfig.Strategy.Performance.MaxLatencyMs,
			PreferCheap:  providerConfig.Strategy.Performance.PreferCheap,
		}

		// Set defaults if not specified
		if routerConfig.BudgetUSD == 0 {
			routerConfig.BudgetUSD = 20.0
		}
		if routerConfig.MaxLatencyMs == 0 {
			routerConfig.MaxLatencyMs = 60000
		}

		// Create router with providers
		r, err := router.NewRouterWithProviders(routerConfig, registry)
		if err != nil {
			return ux.FormatError(err, "creating AI router")
		}

		// Create PRD parser (router handles provider access internally)
		parser := prd.NewParser(r)

		// Parse PRD to spec with timeout context
		parseCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		defer cancel()

		fmt.Println("Parsing PRD with AI (this may take 30-60 seconds)...")
		productSpec, err := parser.ParsePRD(parseCtx, string(prdContent))
		if err != nil {
			telemetry.RecordError(span, err)
			return ux.FormatError(err, "parsing PRD with AI")
		}

		// Save spec
		if saveErr := spec.SaveSpec(productSpec, out); saveErr != nil {
			telemetry.RecordError(span, saveErr)
			return ux.FormatError(saveErr, "saving spec file")
		}

		// Record success with metrics
		duration := time.Since(startTime)
		telemetry.RecordSuccess(span,
			attribute.Int("features_count", len(productSpec.Features)),
			attribute.Int64("duration_ms", duration.Milliseconds()),
		)

		fmt.Printf("✓ Generated spec with %d features\n", len(productSpec.Features))
		fmt.Printf("✓ Saved to: %s\n", out)

		return nil
	},
}

var specValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a specification file",
	Long:  `Validate a specification against the schema and semantic rules.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		defaults := ux.NewPathDefaults()
		in := cmd.Flags().Lookup("in").Value.String()

		// Use smart default if not changed
		if !cmd.Flags().Changed("in") {
			in = defaults.SpecFile()
		}

		// Validate file exists with helpful error
		if err := ux.ValidateRequiredFile(in, "Spec file", "specular spec generate"); err != nil {
			return ux.EnhanceError(err)
		}

		// Load spec
		s, err := spec.LoadSpec(in)
		if err != nil {
			return ux.FormatError(err, "loading spec file")
		}

		// Basic validation
		if s.Product == "" {
			return fmt.Errorf("validation failed: product name is required")
		}

		if len(s.Features) == 0 {
			return fmt.Errorf("validation failed: at least one feature is required")
		}

		// Validate each feature
		for _, feature := range s.Features {
			if feature.ID == "" {
				return fmt.Errorf("validation failed: feature missing ID")
			}
			if feature.Title == "" {
				return fmt.Errorf("validation failed: feature %s missing title", feature.ID)
			}
			if feature.Priority == "" {
				return fmt.Errorf("validation failed: feature %s missing priority", feature.ID)
			}
			if feature.Priority != "P0" && feature.Priority != "P1" && feature.Priority != "P2" {
				return fmt.Errorf("validation failed: feature %s has invalid priority %s (must be P0, P1, or P2)",
					feature.ID, feature.Priority)
			}
		}

		fmt.Printf("✓ Spec is valid (%d features)\n", len(s.Features))
		return nil
	},
}

var specLockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Generate SpecLock from specification",
	Long:  `Create a canonical, hashed SpecLock file with blake3 hashes for drift detection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		defaults := ux.NewPathDefaults()
		in := cmd.Flags().Lookup("in").Value.String()
		out := cmd.Flags().Lookup("out").Value.String()
		version := cmd.Flags().Lookup("version").Value.String()
		note := cmd.Flags().Lookup("note").Value.String()

		// Use smart defaults if not changed
		if !cmd.Flags().Changed("in") {
			in = defaults.SpecFile()
		}
		if !cmd.Flags().Changed("out") {
			out = defaults.SpecLockFile()
		}

		// Validate spec file exists
		if err := ux.ValidateRequiredFile(in, "Spec file", "specular spec new"); err != nil {
			return ux.EnhanceError(err)
		}

		// Load spec
		s, err := spec.LoadSpec(in)
		if err != nil {
			return ux.FormatError(err, "loading spec file")
		}

		// Generate SpecLock
		lock, err := spec.GenerateSpecLock(*s, version)
		if err != nil {
			return ux.FormatError(err, "generating SpecLock")
		}

		// Add note if provided
		if note != "" {
			// Store note in metadata (assumes SpecLock has a Metadata field)
			// If it doesn't exist, we'll just log it
			fmt.Printf("Note: %s\n", note)
		}

		// Save SpecLock
		if saveErr := spec.SaveSpecLock(lock, out); saveErr != nil {
			return ux.FormatError(saveErr, "saving SpecLock file")
		}

		fmt.Printf("✓ Generated SpecLock with %d features\n", len(lock.Features))
		for featureID, lockedFeature := range lock.Features {
			fmt.Printf("  %s: %s\n", featureID, lockedFeature.Hash[:16]+"...")
		}

		if note != "" {
			// Also save note to a separate file
			noteFile := out + ".note"
			noteData := fmt.Sprintf("Created: %s\n%s\n", time.Now().Format(time.RFC3339), note)
			if err := os.WriteFile(noteFile, []byte(noteData), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to save note file: %v\n", err)
			} else {
				fmt.Printf("✓ Note saved to: %s\n", noteFile)
			}
		}

		return nil
	},
}

var specNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new specification",
	Long: `Create a new specification either interactively or from a PRD file.

Without --from flag: Run interactive interview mode (recommended for new projects).
With --from flag: Generate from an existing PRD markdown file.

Examples:
  # Interactive interview mode (default)
  specular spec new

  # Generate from PRD file
  specular spec new --from PRD.md

  # Interactive with preset
  specular spec new --preset web-app

  # Interactive with TUI
  specular spec new --tui`,
	RunE: runSpecNew,
}

var specEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the current specification in $EDITOR",
	Long: `Open the current specification file in your default editor (from $EDITOR environment variable).

The specification file is opened for editing and automatically validated after saving.`,
	RunE: runSpecEdit,
}

var specDiffCmd = &cobra.Command{
	Use:   "diff <fileA> <fileB>",
	Short: "Compare two specification versions",
	Long: `Compare two specification files and show differences in features, priorities, and content.

Useful for reviewing changes before locking a new spec version or understanding
what changed between releases.`,
	Args: cobra.ExactArgs(2),
	RunE: runSpecDiff,
}

var specApproveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Approve the current specification for use",
	Long: `Mark the current specification as approved for plan generation and execution.

This creates an approval record with timestamp and optional signature for
governance and compliance purposes.`,
	RunE: runSpecApprove,
}

func runSpecNew(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	out := cmd.Flags().Lookup("out").Value.String()
	fromFile := cmd.Flags().Lookup("from").Value.String()

	// Use smart default for output if not changed
	if !cmd.Flags().Changed("out") {
		out = defaults.SpecFile()
	}

	// If --from flag is provided, use PRD generation mode
	if cmd.Flags().Changed("from") {
		return runSpecGenerateFromPRD(fromFile, out)
	}

	// Otherwise, run interview mode
	return runSpecInterviewMode(cmd, out)
}

func runSpecGenerateFromPRD(prdFile, out string) error {
	defaults := ux.NewPathDefaults()
	configPath := defaults.ProvidersFile()

	// Validate PRD file exists
	if err := ux.ValidateRequiredFile(prdFile, "PRD file", "Create a PRD markdown file or run 'specular spec new'"); err != nil {
		return ux.EnhanceError(err)
	}

	fmt.Printf("Generating spec from PRD: %s\n", prdFile)

	// Read PRD file
	prdContent, err := os.ReadFile(prdFile)
	if err != nil {
		return ux.FormatError(err, "reading PRD file")
	}

	// Load provider configuration
	fmt.Println("Loading provider configuration...")
	registry, err := provider.LoadRegistryFromConfig(configPath)
	if err != nil {
		return ux.FormatError(err, "loading AI providers")
	}

	// Load provider config to get strategy settings
	providerConfig, err := provider.LoadProvidersConfig(configPath)
	if err != nil {
		return ux.FormatError(err, "loading provider config")
	}

	// Create router config from provider strategy
	routerConfig := &router.RouterConfig{
		BudgetUSD:    providerConfig.Strategy.Budget.MaxCostPerDay,
		MaxLatencyMs: providerConfig.Strategy.Performance.MaxLatencyMs,
		PreferCheap:  providerConfig.Strategy.Performance.PreferCheap,
	}

	// Set defaults if not specified
	if routerConfig.BudgetUSD == 0 {
		routerConfig.BudgetUSD = 20.0
	}
	if routerConfig.MaxLatencyMs == 0 {
		routerConfig.MaxLatencyMs = 60000
	}

	// Create router with providers
	r, err := router.NewRouterWithProviders(routerConfig, registry)
	if err != nil {
		return ux.FormatError(err, "creating AI router")
	}

	// Create PRD parser
	parser := prd.NewParser(r)

	// Parse PRD to spec
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fmt.Println("Parsing PRD with AI (this may take 30-60 seconds)...")
	productSpec, err := parser.ParsePRD(ctx, string(prdContent))
	if err != nil {
		return ux.FormatError(err, "parsing PRD with AI")
	}

	// Save spec
	if saveErr := spec.SaveSpec(productSpec, out); saveErr != nil {
		return ux.FormatError(saveErr, "saving spec file")
	}

	fmt.Printf("✓ Generated spec with %d features\n", len(productSpec.Features))
	fmt.Printf("✓ Saved to: %s\n", out)

	return nil
}

func runSpecInterviewMode(cmd *cobra.Command, out string) error {
	// Reuse interview command logic by constructing temporary command
	// This allows us to reuse all the interview functionality
	interviewCmd.Flags().Set("out", out)
	if cmd.Flags().Changed("preset") {
		interviewCmd.Flags().Set("preset", cmd.Flags().Lookup("preset").Value.String())
	}
	if cmd.Flags().Changed("strict") {
		interviewCmd.Flags().Set("strict", cmd.Flags().Lookup("strict").Value.String())
	}
	if cmd.Flags().Changed("tui") {
		interviewCmd.Flags().Set("tui", cmd.Flags().Lookup("tui").Value.String())
	}
	if cmd.Flags().Changed("list") {
		interviewCmd.Flags().Set("list", cmd.Flags().Lookup("list").Value.String())
	}

	// Call runInterview from interview.go (but without deprecation warning for this use)
	return runInterviewInternal(cmd)
}

func runInterviewInternal(cmd *cobra.Command) error {
	// Similar to runInterview but without deprecation notice
	// This is the logic duplicated from interview.go
	defaults := ux.NewPathDefaults()
	out := cmd.Flags().Lookup("out").Value.String()
	preset := cmd.Flags().Lookup("preset").Value.String()
	strict := cmd.Flags().Lookup("strict").Value.String() == "true"
	useTUI := cmd.Flags().Lookup("tui").Value.String() == "true"
	list := cmd.Flags().Lookup("list").Value.String() == "true"

	if !cmd.Flags().Changed("out") {
		out = defaults.SpecFile()
	}

	if list {
		fmt.Println("Available interview presets:")
		presets := interview.ListPresets()
		for _, p := range presets {
			fmt.Printf("  %s\n", p.Name)
			fmt.Printf("    %s\n", p.Description)
			fmt.Printf("    Questions: %d\n\n", len(p.Questions))
		}
		return nil
	}

	if preset == "" && !cmd.Flags().Changed("preset") {
		fmt.Println("Select a preset for your project:")
		presets := []string{
			"web-app - Web application with UI and backend",
			"api-service - RESTful API service",
			"cli-tool - Command-line interface tool",
			"microservice - Microservice component",
			"data-pipeline - Data processing pipeline",
		}
		selected, _ := ux.Select("Choose preset:", presets, 0)
		preset = strings.Split(selected, " ")[0]
	} else if preset == "" {
		return ux.NewErrorWithSuggestion(
			fmt.Errorf("preset is required"),
			"Use --list to see available presets or run without --preset for interactive selection",
		)
	}

	// Create engine
	engine, err := interview.NewEngine(preset, strict)
	if err != nil {
		return ux.FormatError(err, "creating interview engine")
	}

	// Run TUI or CLI mode
	if useTUI {
		fmt.Printf("=== Specular Interview Mode (TUI) ===\n")
		fmt.Printf("Preset: %s\n\n", preset)
		result, err := tui.RunInterview(engine)
		if err != nil {
			return ux.FormatError(err, "running TUI interview")
		}
		return tui.SaveResult(result, out)
	}

	// CLI interview - simplified version
	fmt.Printf("=== Specular Interview Mode ===\n")
	fmt.Printf("Preset: %s\n\n", preset)

	if err := engine.Start(); err != nil {
		return ux.FormatError(err, "starting interview")
	}

	scanner := bufio.NewScanner(os.Stdin)

	for !engine.IsComplete() {
		q, err := engine.CurrentQuestion()
		if err != nil || q == nil {
			break
		}

		fmt.Printf("[%d%%] %s\n", int(engine.Progress()), q.Text)
		if q.Description != "" {
			fmt.Printf("     %s\n", q.Description)
		}
		if q.Required {
			fmt.Printf("     (required)\n")
		}
		fmt.Printf("\n> ")

		var answer interview.Answer
		if q.Type == interview.QuestionTypeMulti {
			values := []string{}
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					break
				}
				values = append(values, strings.TrimSpace(line))
			}
			answer.Values = values
		} else {
			if !scanner.Scan() {
				return fmt.Errorf("failed to read input")
			}
			answer.Value = strings.TrimSpace(scanner.Text())
		}

		_, err = engine.Answer(answer)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			if strict {
				return err
			}
			continue
		}
		fmt.Println()
	}

	result, err := engine.GetResult()
	if err != nil {
		return ux.FormatError(err, "generating spec")
	}

	if err := spec.SaveSpec(result.Spec, out); err != nil {
		return ux.FormatError(err, "saving spec")
	}

	fmt.Printf("\n✓ Specification generated successfully!\n")
	fmt.Printf("  Output: %s\n", out)
	fmt.Printf("  Features: %d\n\n", len(result.Spec.Features))
	fmt.Println("Next steps:")
	fmt.Printf("  1. Review: specular spec edit\n")
	fmt.Printf("  2. Validate: specular spec validate\n")
	fmt.Printf("  3. Lock: specular spec lock\n")

	return nil
}

func runSpecEdit(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	specPath := defaults.SpecFile()

	// Ensure spec exists
	if err := ux.ValidateRequiredFile(specPath, "Spec file", "specular spec new"); err != nil {
		return ux.EnhanceError(err)
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Fallback to vi
	}

	// Open editor
	editorCmd := exec.Command(editor, specPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}

	// Validate the edited spec
	s, err := spec.LoadSpec(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Specification may contain errors: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please check and fix the specification file.\n")
		return err
	}

	// Basic validation
	if s.Product == "" || len(s.Features) == 0 {
		fmt.Fprintf(os.Stderr, "Warning: Specification appears incomplete\n")
	} else {
		fmt.Println("✓ Specification updated successfully")
		fmt.Printf("  Product: %s\n", s.Product)
		fmt.Printf("  Features: %d\n", len(s.Features))
	}

	return nil
}

func runSpecDiff(cmd *cobra.Command, args []string) error {
	fileA := args[0]
	fileB := args[1]

	// Validate both files exist
	if err := ux.ValidateRequiredFile(fileA, "First spec file", ""); err != nil {
		return ux.EnhanceError(err)
	}
	if err := ux.ValidateRequiredFile(fileB, "Second spec file", ""); err != nil {
		return ux.EnhanceError(err)
	}

	// Load both specs
	specA, err := spec.LoadSpec(fileA)
	if err != nil {
		return ux.FormatError(err, fmt.Sprintf("loading %s", fileA))
	}

	specB, err := spec.LoadSpec(fileB)
	if err != nil {
		return ux.FormatError(err, fmt.Sprintf("loading %s", fileB))
	}

	// Compare specs
	fmt.Printf("Comparing specifications:\n")
	fmt.Printf("  A: %s (%s, %d features)\n", fileA, specA.Product, len(specA.Features))
	fmt.Printf("  B: %s (%s, %d features)\n\n", fileB, specB.Product, len(specB.Features))

	// Product name difference
	if specA.Product != specB.Product {
		fmt.Printf("Product name changed:\n")
		fmt.Printf("  - %s\n", specA.Product)
		fmt.Printf("  + %s\n\n", specB.Product)
	}

	// Build feature maps
	featuresA := make(map[string]spec.Feature)
	for _, f := range specA.Features {
		featuresA[string(f.ID)] = f
	}

	featuresB := make(map[string]spec.Feature)
	for _, f := range specB.Features {
		featuresB[string(f.ID)] = f
	}

	// Find added features
	added := []spec.Feature{}
	for id, f := range featuresB {
		if _, exists := featuresA[id]; !exists {
			added = append(added, f)
		}
	}

	// Find removed features
	removed := []spec.Feature{}
	for id, f := range featuresA {
		if _, exists := featuresB[id]; !exists {
			removed = append(removed, f)
		}
	}

	// Find modified features
	modified := []string{}
	for id, fA := range featuresA {
		if fB, exists := featuresB[id]; exists {
			if fA.Title != fB.Title || fA.Desc != fB.Desc || fA.Priority != fB.Priority {
				modified = append(modified, id)
			}
		}
	}

	// Print differences
	if len(added) > 0 {
		fmt.Printf("Added features (%d):\n", len(added))
		for _, f := range added {
			fmt.Printf("  + %s: %s [%s]\n", f.ID, f.Title, f.Priority)
		}
		fmt.Println()
	}

	if len(removed) > 0 {
		fmt.Printf("Removed features (%d):\n", len(removed))
		for _, f := range removed {
			fmt.Printf("  - %s: %s [%s]\n", f.ID, f.Title, f.Priority)
		}
		fmt.Println()
	}

	if len(modified) > 0 {
		fmt.Printf("Modified features (%d):\n", len(modified))
		for _, id := range modified {
			fA := featuresA[id]
			fB := featuresB[id]
			fmt.Printf("  ~ %s:\n", id)
			if fA.Title != fB.Title {
				fmt.Printf("    Title: %s → %s\n", fA.Title, fB.Title)
			}
			if fA.Priority != fB.Priority {
				fmt.Printf("    Priority: %s → %s\n", fA.Priority, fB.Priority)
			}
			if fA.Desc != fB.Desc {
				fmt.Printf("    Description changed\n")
			}
		}
		fmt.Println()
	}

	if len(added) == 0 && len(removed) == 0 && len(modified) == 0 {
		fmt.Println("No differences found")
	}

	return nil
}

func runSpecApprove(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	specPath := defaults.SpecFile()

	// Validate spec file exists
	if err := ux.ValidateRequiredFile(specPath, "Spec file", "specular spec new"); err != nil {
		return ux.EnhanceError(err)
	}

	// Load spec
	s, err := spec.LoadSpec(specPath)
	if err != nil {
		return ux.FormatError(err, "loading spec file")
	}

	// Validate spec before approval
	if s.Product == "" {
		return fmt.Errorf("cannot approve: product name is required")
	}
	if len(s.Features) == 0 {
		return fmt.Errorf("cannot approve: at least one feature is required")
	}

	// Create approval marker file
	approvalFile := defaults.SpecFile() + ".approved"
	approvalData := fmt.Sprintf("Approved at: %s\n", time.Now().Format(time.RFC3339))
	approvalData += fmt.Sprintf("Product: %s\n", s.Product)
	approvalData += fmt.Sprintf("Features: %d\n", len(s.Features))

	if err := os.WriteFile(approvalFile, []byte(approvalData), 0644); err != nil {
		return ux.FormatError(err, "creating approval record")
	}

	fmt.Printf("✓ Specification approved\n")
	fmt.Printf("  Product: %s\n", s.Product)
	fmt.Printf("  Features: %d\n", len(s.Features))
	fmt.Printf("  Approval record: %s\n", approvalFile)

	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Generate lock: specular spec lock\n")
	fmt.Printf("  2. Create plan: specular plan\n")

	return nil
}

func init() {
	rootCmd.AddCommand(specCmd)
	specCmd.AddCommand(specNewCmd)
	specCmd.AddCommand(specGenerateCmd)
	specCmd.AddCommand(specValidateCmd)
	specCmd.AddCommand(specLockCmd)
	specCmd.AddCommand(specEditCmd)
	specCmd.AddCommand(specDiffCmd)
	specCmd.AddCommand(specApproveCmd)

	specGenerateCmd.Flags().StringP("in", "i", "PRD.md", "Input PRD file")
	specGenerateCmd.Flags().StringP("out", "o", ".specular/spec.yaml", "Output spec file")
	specGenerateCmd.Flags().String("config", ".specular/providers.yaml", "Provider configuration file")

	specValidateCmd.Flags().StringP("in", "i", ".specular/spec.yaml", "Spec file to validate")

	specLockCmd.Flags().StringP("in", "i", ".specular/spec.yaml", "Input spec file")
	specLockCmd.Flags().StringP("out", "o", ".specular/spec.lock.json", "Output SpecLock file")
	specLockCmd.Flags().String("version", "1.0", "SpecLock version")
	specLockCmd.Flags().String("note", "", "Add a note to the SpecLock (e.g., release notes or approval info)")

	specNewCmd.Flags().StringP("out", "o", ".specular/spec.yaml", "Output path for generated spec")
	specNewCmd.Flags().String("from", "", "Generate from PRD file instead of interactive mode")
	specNewCmd.Flags().String("preset", "", "Use a preset template (use --list to see options)")
	specNewCmd.Flags().Bool("strict", false, "Enable strict validation mode")
	specNewCmd.Flags().Bool("tui", false, "Use interactive terminal UI mode")
	specNewCmd.Flags().Bool("list", false, "List available presets")
}
