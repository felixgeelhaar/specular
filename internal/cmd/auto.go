package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/auto"
	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/router"
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

Examples:
  specular auto "Build a REST API for user management"
  specular auto --dry-run "Create a React dashboard"
  specular auto --no-approval "Add authentication to my app"
  specular auto --resume auto-1762811730
`,
	Args: func(cmd *cobra.Command, args []string) error {
		resumeFrom, _ := cmd.Flags().GetString("resume")
		if resumeFrom == "" && len(args) < 1 {
			return fmt.Errorf("requires a goal argument when not resuming")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse flags
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		noApproval, _ := cmd.Flags().GetBool("no-approval")
		maxCost, _ := cmd.Flags().GetFloat64("max-cost")
		maxCostPerTask, _ := cmd.Flags().GetFloat64("max-cost-per-task")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")
		timeoutMinutes, _ := cmd.Flags().GetInt("timeout")
		verbose, _ := cmd.Flags().GetBool("verbose")
		resumeFrom, _ := cmd.Flags().GetString("resume")

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

		// Build auto config
		config := auto.Config{
			Goal:             goal,
			RequireApproval:  !noApproval,
			MaxCostUSD:       maxCost,
			MaxCostPerTask:   maxCostPerTask,
			MaxRetries:       maxRetries,
			TimeoutMinutes:   timeoutMinutes,
			Verbose:          verbose,
			DryRun:           dryRun,
			ResumeFrom:       resumeFrom,
		}

		// Create orchestrator and execute
		orchestrator := auto.NewOrchestrator(r, config)
		result, err := orchestrator.Execute(cmd.Context())
		if err != nil {
			return fmt.Errorf("auto mode failed: %w", err)
		}

		// Print results
		fmt.Println()
		fmt.Printf("✅ Auto mode completed in %s\n", result.Duration)
		fmt.Printf("   Total cost: $%.4f\n", result.TotalCost)
		fmt.Printf("   Tasks executed: %d\n", result.TasksExecuted)
		if result.TasksFailed > 0 {
			fmt.Printf("   Tasks failed: %d\n", result.TasksFailed)
		}

		return nil
	},
}

func init() {
	autoCmd.Flags().Bool("dry-run", false, "Generate spec and plan but don't execute")
	autoCmd.Flags().Bool("no-approval", false, "Skip approval gate (auto-approve plan)")
	autoCmd.Flags().Float64("max-cost", 5.0, "Maximum cost in USD for entire workflow")
	autoCmd.Flags().Float64("max-cost-per-task", 1.0, "Maximum cost in USD per task")
	autoCmd.Flags().Int("max-retries", 3, "Maximum retries per failed task")
	autoCmd.Flags().Int("timeout", 30, "Timeout in minutes for entire workflow")
	autoCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	autoCmd.Flags().String("resume", "", "Resume from checkpoint (e.g., auto-1762811730)")

	rootCmd.AddCommand(autoCmd)
}
