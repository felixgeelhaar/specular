package cmd

import (
	"fmt"

	"github.com/felixgeelhaar/ai-dev/internal/exec"
	"github.com/felixgeelhaar/ai-dev/internal/plan"
	"github.com/felixgeelhaar/ai-dev/internal/policy"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Execute build with policy enforcement",
	Long: `Execute the build process in a Docker sandbox with strict policy enforcement.
All execution passes through guardrail checks including Docker-only enforcement,
linting, testing, and security scanning.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planFile, _ := cmd.Flags().GetString("plan")
		policyFile, _ := cmd.Flags().GetString("policy")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		manifestDir, _ := cmd.Flags().GetString("manifest-dir")

		// Load plan
		p, err := plan.LoadPlan(planFile)
		if err != nil {
			return fmt.Errorf("failed to load plan: %w", err)
		}

		// Load or create default policy
		var pol *policy.Policy
		if policyFile != "" {
			pol, err = policy.LoadPolicy(policyFile)
			if err != nil {
				fmt.Printf("Warning: failed to load policy, using defaults: %v\n", err)
				pol = policy.DefaultPolicy()
			}
		} else {
			pol = policy.DefaultPolicy()
		}

		// Create executor
		executor := &exec.Executor{
			Policy:      pol,
			DryRun:      dryRun,
			ManifestDir: manifestDir,
		}

		// Execute plan
		fmt.Printf("Executing plan with %d tasks...\n\n", len(p.Tasks))
		result, err := executor.Execute(p)
		if err != nil {
			return fmt.Errorf("execution failed: %w", err)
		}

		// Print summary
		result.PrintSummary()

		// Check for failures
		if result.FailedTasks > 0 {
			return fmt.Errorf("execution completed with %d failed tasks", result.FailedTasks)
		}

		fmt.Println("\n✓ All tasks completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().String("plan", "plan.json", "Plan file to execute")
	buildCmd.Flags().String("policy", ".aidv/policy.yaml", "Policy file for enforcement")
	buildCmd.Flags().Bool("dry-run", false, "Show what would be executed without running")
	buildCmd.Flags().String("manifest-dir", ".aidv/runs", "Directory for run manifests")
	buildCmd.Flags().String("fail-on", "", "Fail on conditions (comma-separated: drift,lint,test,security)")
}
