package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/attestation"
	"github.com/felixgeelhaar/specular/internal/auto"
	"github.com/felixgeelhaar/specular/internal/autopolicy"
	"github.com/felixgeelhaar/specular/internal/hooks"
	"github.com/felixgeelhaar/specular/internal/profiles"
	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/trace"
	"github.com/felixgeelhaar/specular/internal/tui"
)

var autoCmd = &cobra.Command{
	Use:   "auto <goal>",
	Short: "Autonomous mode: from goal to working code",
	Long: `Run Specular in autonomous agent mode. Provide a natural language goal,
and Specular will:
  1. Generate a structured specification
  2. Create a locked spec with hashes
  3. Generate an execution plan
  4. Show approval gate (if enabled)
  5. Execute the plan (Phase 2 - coming soon)

This is similar to Claude Code's autonomous workflow but with Specular's
specification-driven approach and policy enforcement.

Profiles:
  Profiles enable environment-specific configurations. Use --profile to select
  a profile or --list-profiles to see available profiles.

  Built-in profiles:
    default - Interactive development (balanced safety and flexibility)
    ci      - Non-interactive CI/CD pipelines (auto-approve, JSON output)
    strict  - Maximum safety (approve all steps, strict limits)

Exit Codes:
  0  Success - Execution completed successfully
  1  General error - Unexpected runtime error
  2  Usage error - Invalid CLI usage (bad flags, missing args)
  3  Policy violation - Operation blocked by policy
  4  Drift detected - Specification drift requires intervention
  5  Auth error - Authentication or permission failure
  6  Network error - Network connectivity issue

Scope Filtering:
  Filter execution to specific features or paths using --scope:

  Patterns:
    feature:ID          Match by exact feature ID (e.g., feature:feat-1)
    feature:pattern*    Match feature titles with glob (e.g., feature:User*)
    /api/path/*         Match API paths with glob (e.g., /api/users/*)
    @tag                Match by feature tag (future)

  Multiple patterns are combined with OR logic. By default, dependencies
  of matched tasks are included. Use --include-dependencies=false to disable.

Examples:
  specular auto "Build a REST API for user management"
  specular auto --profile ci "Create a React dashboard"
  specular auto --profile strict --dry-run "Add authentication"
  specular auto --scope feature:feat-1 "Execute only feature 1"
  specular auto --scope "feature:User*" --scope "/api/auth/*" "Execute user features"
  specular auto --list-profiles
  specular auto --resume auto-1762811730
`,
	Args: func(cmd *cobra.Command, args []string) error {
		listProfiles, _ := cmd.Flags().GetBool("list-profiles")
		resumeFrom, _ := cmd.Flags().GetString("resume")

		// Allow no goal if listing profiles or resuming
		if !listProfiles && resumeFrom == "" && len(args) < 1 {
			return fmt.Errorf("invalid argument: requires a goal argument when not resuming or listing profiles")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse flags
		listProfiles, _ := cmd.Flags().GetBool("list-profiles")
		profileName, _ := cmd.Flags().GetString("profile")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		noApproval, _ := cmd.Flags().GetBool("no-approval")
		maxCost, _ := cmd.Flags().GetFloat64("max-cost")
		maxCostPerTask, _ := cmd.Flags().GetFloat64("max-cost-per-task")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")
		maxSteps, _ := cmd.Flags().GetInt("max-steps")
		timeoutMinutes, _ := cmd.Flags().GetInt("timeout")
		verbose, _ := cmd.Flags().GetBool("verbose")
		resumeFrom, _ := cmd.Flags().GetString("resume")
		outputDir, _ := cmd.Flags().GetString("output")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		scopePatterns, _ := cmd.Flags().GetStringSlice("scope")
		includeDependencies, _ := cmd.Flags().GetBool("include-dependencies")
		useTUI, _ := cmd.Flags().GetBool("tui")
		enableTrace, _ := cmd.Flags().GetBool("trace")
		savePatches, _ := cmd.Flags().GetBool("save-patches")
		enableAttest, _ := cmd.Flags().GetBool("attest")

		// Handle --list-profiles
		if listProfiles {
			return listAvailableProfiles()
		}

		// Load profile
		loader := profiles.NewLoader()
		if profileName == "" {
			profileName = "default"
		}

		profile, err := loader.Load(profileName)
		if err != nil {
			return fmt.Errorf("failed to load profile %q: %w", profileName, err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Using profile: %s (%s)\n", profile.Name, profile.Description)
		}

		// Build goal from args (required unless resuming)
		goal := ""
		if resumeFrom == "" {
			for i, arg := range args {
				if i > 0 {
					goal += " "
				}
				goal += arg
			}
		}

		// Load provider registry
		providerConfigPath := ".specular/providers.yaml"
		registry, err := provider.LoadRegistryFromConfig(providerConfigPath)
		if err != nil {
			return fmt.Errorf("failed to load providers: %w", err)
		}

		if verbose {
			providerNames := registry.List()
			fmt.Fprintf(os.Stderr, "Loaded %d provider(s)\n", len(providerNames))
		}

		// Create router config
		routerConfig := &router.RouterConfig{
			BudgetUSD:    maxCost,
			MaxLatencyMs: 60000,
			PreferCheap:  true, // Prefer cheaper models for auto mode
		}

		// Create router
		r, err := router.NewRouterWithProviders(routerConfig, registry)
		if err != nil {
			return fmt.Errorf("failed to create router: %w", err)
		}

		if verbose {
			budget := r.GetBudget()
			fmt.Fprintf(os.Stderr, "Router initialized: budget=$%.2f\n", budget.LimitUSD)
		}

		// Merge CLI flags with profile
		cliFlags := &profiles.CLIFlags{}

		// Only set flags that were explicitly provided
		if cmd.Flags().Changed("no-approval") {
			requireApproval := !noApproval
			cliFlags.RequireApproval = &requireApproval
		}
		if cmd.Flags().Changed("max-cost") {
			cliFlags.MaxCostUSD = &maxCost
		}
		if cmd.Flags().Changed("max-cost-per-task") {
			cliFlags.MaxCostPerTask = &maxCostPerTask
		}
		if cmd.Flags().Changed("max-retries") {
			cliFlags.MaxRetries = &maxRetries
		}
		if cmd.Flags().Changed("max-steps") {
			cliFlags.MaxSteps = &maxSteps
		}
		if cmd.Flags().Changed("timeout") {
			timeout := time.Duration(timeoutMinutes) * time.Minute
			cliFlags.Timeout = &timeout
		}

		// Merge profile with CLI overrides
		effectiveProfile := profiles.MergeWithCLIFlags(profile, cliFlags)

		if verbose {
			fmt.Fprintf(os.Stderr, "Effective config: max_steps=%d, timeout=%s, max_cost=$%.2f\n",
				effectiveProfile.Safety.MaxSteps,
				effectiveProfile.Safety.Timeout,
				effectiveProfile.Safety.MaxCostUSD)
		}

		// Build auto config from effective profile
		config := auto.Config{
			Goal:                goal,
			RequireApproval:     effectiveProfile.Approvals.Interactive && effectiveProfile.Approvals.Mode != profiles.ApprovalModeNone,
			MaxCostUSD:          effectiveProfile.Safety.MaxCostUSD,
			MaxCostPerTask:      effectiveProfile.Safety.MaxCostPerTask,
			MaxRetries:          effectiveProfile.Safety.MaxRetries,
			TimeoutMinutes:      int(effectiveProfile.Safety.Timeout.Minutes()),
			Verbose:             verbose,
			DryRun:              dryRun,
			ResumeFrom:          resumeFrom,
			OutputDir:           outputDir,
			JSONOutput:          jsonOutput,
			ScopePatterns:       scopePatterns,
			IncludeDependencies: includeDependencies,
		}

		// Create orchestrator
		orchestrator := auto.NewOrchestrator(r, config)

		// Handle TUI mode
		var tuiAdapter *tui.Adapter
		if useTUI {
			// Initialize TUI
			tuiAdapter = tui.NewAdapter(goal, profileName)
			if err := tuiAdapter.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to start TUI, falling back to text mode: %v\n", err)
				tuiAdapter = nil
			} else {
				defer tuiAdapter.Stop()

				// Create hook registry and register TUI hook
				registry := hooks.NewRegistry()
				tuiHook := tui.NewHook(tuiAdapter)
				if err := registry.Register(tuiHook); err != nil {
					fmt.Fprintf(os.Stderr, "⚠️  Failed to register TUI hook: %v\n", err)
				} else {
					// Set hook registry on orchestrator for real-time updates
					orchestrator.SetHookRegistry(registry)
					fmt.Println("📺 TUI mode enabled with real-time updates")
				}
			}
		}

		// Set policy checker from profile if available
		if effectiveProfile != nil {
			policyChecker := autopolicy.NewCheckerFromProfile(effectiveProfile)
			// Wrap the autopolicy checker to match auto.PolicyChecker interface
			orchestrator.SetPolicyChecker(newPolicyCheckerAdapter(policyChecker))
		}

		// Set trace logger if enabled
		if enableTrace {
			traceConfig := trace.DefaultConfig()
			traceConfig.Enabled = true
			tracer, err := trace.NewLogger(traceConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to initialize trace logging: %v\n", err)
			} else {
				orchestrator.SetTracer(tracer)
				fmt.Printf("📝 Trace logging enabled: %s\n", tracer.GetLogPath())
			}
		}

		// Set patch generator if enabled
		if savePatches {
			workingDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to get working directory: %v\n", err)
			} else {
				homeDir, _ := os.UserHomeDir()
				patchDir := filepath.Join(homeDir, ".specular", "patches")
				orchestrator.SetPatchGenerator(workingDir, patchDir)
				fmt.Printf("💾 Patch generation enabled: %s\n", patchDir)
			}
		}

		// Execute workflow
		result, err := orchestrator.Execute(cmd.Context())
		if err != nil {
			return fmt.Errorf("auto mode failed: %w", err)
		}

		// Generate attestation if enabled
		if enableAttest {
			if err := generateAttestation(result, &config, outputDir); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to generate attestation: %v\n", err)
			}
		}

		// Print results based on output format
		if jsonOutput {
			// Output JSON format
			if result.AutoOutput != nil {
				jsonData, err := result.AutoOutput.ToJSON()
				if err != nil {
					return fmt.Errorf("failed to serialize JSON output: %w", err)
				}
				fmt.Println(string(jsonData))
			} else {
				return fmt.Errorf("JSON output not available")
			}
		} else {
			// Output text format (default)
			fmt.Println()
			fmt.Printf("✅ Auto mode completed in %s\n", result.Duration)
			fmt.Printf("   Total cost: $%.4f\n", result.TotalCost)
			fmt.Printf("   Tasks executed: %d\n", result.TasksExecuted)
			if result.TasksFailed > 0 {
				fmt.Printf("   Tasks failed: %d\n", result.TasksFailed)
			}
		}

		return nil
	},
}

// policyCheckerAdapter adapts autopolicy.PolicyChecker to auto.PolicyChecker
type policyCheckerAdapter struct {
	checker autopolicy.PolicyChecker
}

// newPolicyCheckerAdapter creates an adapter that wraps autopolicy.PolicyChecker
func newPolicyCheckerAdapter(checker autopolicy.PolicyChecker) auto.PolicyChecker {
	return &policyCheckerAdapter{checker: checker}
}

// CheckStep implements auto.PolicyChecker
func (a *policyCheckerAdapter) CheckStep(ctx context.Context, step *auto.ActionStep) (*auto.PolicyResult, error) {
	// Call the autopolicy checker
	result, err := a.checker.CheckStep(ctx, step)
	if err != nil {
		return nil, err
	}

	// Convert autopolicy.PolicyResult to auto.PolicyResult
	return &auto.PolicyResult{
		Allowed:  result.Allowed,
		Reason:   result.Reason,
		Warnings: result.Warnings,
		Metadata: result.Metadata,
	}, nil
}

// Name implements auto.PolicyChecker
func (a *policyCheckerAdapter) Name() string {
	return a.checker.Name()
}

func init() {
	// Profile flags
	autoCmd.Flags().StringP("profile", "p", "", "Profile to use (default, ci, strict, or custom)")
	autoCmd.Flags().Bool("list-profiles", false, "List available profiles and exit")

	// Execution flags
	autoCmd.Flags().Bool("dry-run", false, "Generate spec and plan but don't execute")
	autoCmd.Flags().Bool("no-approval", false, "Skip approval gate (auto-approve plan)")
	autoCmd.Flags().String("resume", "", "Resume from checkpoint (e.g., auto-1762811730)")
	autoCmd.Flags().StringP("output", "o", "", "Output directory to save spec and plan files")
	autoCmd.Flags().Bool("save-patches", false, "Save patches for each step to enable rollback")
	autoCmd.Flags().Bool("attest", false, "Generate cryptographic attestation of workflow execution")

	// Safety limit flags (override profile settings)
	autoCmd.Flags().Float64("max-cost", 0, "Maximum cost in USD for entire workflow (overrides profile)")
	autoCmd.Flags().Float64("max-cost-per-task", 0, "Maximum cost in USD per task (overrides profile)")
	autoCmd.Flags().Int("max-retries", 0, "Maximum retries per failed task (overrides profile)")
	autoCmd.Flags().Int("max-steps", 0, "Maximum number of workflow steps (overrides profile)")
	autoCmd.Flags().Int("timeout", 0, "Timeout in minutes for entire workflow (overrides profile)")

	// Output flags
	autoCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	autoCmd.Flags().Bool("json", false, "Output results in JSON format (for CI/CD integration)")
	autoCmd.Flags().Bool("tui", false, "Enable interactive TUI mode")
	autoCmd.Flags().Bool("trace", false, "Enable detailed trace logging to ~/.specular/logs")

	// Scope filtering flags
	autoCmd.Flags().StringSliceP("scope", "s", []string{}, "Filter execution scope (can be used multiple times)")
	autoCmd.Flags().Bool("include-dependencies", true, "Include dependencies of scoped tasks")

	rootCmd.AddCommand(autoCmd)
}

// generateAttestation creates and saves a cryptographic attestation
func generateAttestation(result *auto.Result, config *auto.Config, outputDir string) error {
	// Get user identity (use hostname as fallback)
	identity := os.Getenv("USER")
	if identity == "" {
		hostname, _ := os.Hostname()
		identity = hostname
	}

	// Create signer
	signer, err := attestation.NewEphemeralSigner(identity)
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	// Create generator (use a version string)
	generator := attestation.NewGenerator(signer, "1.0.0") // TODO: Use actual version

	// Get plan and output JSON
	var planJSON []byte
	var outputJSON []byte

	if result.AutoOutput != nil {
		// Get output JSON
		outputJSON, err = result.AutoOutput.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize output: %w", err)
		}

		// Get plan JSON (if available from Result.Plan)
		if result.Plan != nil {
			planJSON, _ = json.Marshal(result.Plan)
		}
	}

	// Generate attestation
	att, err := generator.Generate(result, config, planJSON, outputJSON)
	if err != nil {
		return fmt.Errorf("failed to generate attestation: %w", err)
	}

	// Determine workflow ID
	workflowID := "unknown"
	if result.AutoOutput != nil {
		workflowID = result.AutoOutput.Audit.CheckpointID
	}

	// Determine output path
	var attestPath string
	if outputDir != "" {
		attestPath = filepath.Join(outputDir, fmt.Sprintf("%s.attestation.json", workflowID))
	} else {
		homeDir, _ := os.UserHomeDir()
		attestPath = filepath.Join(homeDir, ".specular", "attestations", fmt.Sprintf("%s.attestation.json", workflowID))
		if err := os.MkdirAll(filepath.Dir(attestPath), 0750); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create attestation directory: %v\n", err)
		}
	}

	// Save attestation
	attestJSON, err := att.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize attestation: %w", err)
	}

	if err := os.WriteFile(attestPath, attestJSON, 0600); err != nil {
		return fmt.Errorf("failed to write attestation: %w", err)
	}

	fmt.Printf("🔐 Generated attestation: %s\n", attestPath)
	fmt.Printf("   Signed by: %s\n", att.SignedBy)
	fmt.Printf("   Plan hash: %s\n", att.PlanHash[:16]+"...")
	fmt.Printf("   Output hash: %s\n", att.OutputHash[:16]+"...")

	return nil
}

// listAvailableProfiles lists all available profiles from all sources.
func listAvailableProfiles() error {
	loader := profiles.NewLoader()
	profileNames, err := loader.List()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	fmt.Println("Available profiles:")
	fmt.Println()

	// Load and display each profile
	for _, name := range profileNames {
		profile, err := loader.Load(name)
		if err != nil {
			fmt.Printf("  ❌ %s (error: %v)\n", name, err)
			continue
		}

		// Determine source
		source := "built-in"
		if name != "default" && name != "ci" && name != "strict" {
			source = "custom"
		}

		fmt.Printf("  %s (%s)\n", name, source)
		fmt.Printf("     %s\n", profile.Description)
		fmt.Printf("     Approval: %s, Max steps: %d, Timeout: %s, Max cost: $%.2f\n",
			profile.Approvals.Mode,
			profile.Safety.MaxSteps,
			profile.Safety.Timeout,
			profile.Safety.MaxCostUSD)
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  specular auto --profile <name> \"your goal\"")
	fmt.Println()
	fmt.Println("Create custom profiles in:")
	fmt.Println("  - Project: ./auto.profiles.yaml")
	fmt.Println("  - User:    ~/.specular/auto.profiles.yaml")

	return nil
}
