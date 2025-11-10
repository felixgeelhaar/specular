package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	"github.com/felixgeelhaar/specular/internal/exec"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/progress"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Execute build with policy enforcement",
	Long: `Execute the build process in a Docker sandbox with strict policy enforcement.
All execution passes through guardrail checks including Docker-only enforcement,
linting, testing, and security scanning.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		defaults := ux.NewPathDefaults()
		planFile := cmd.Flags().Lookup("plan").Value.String()
		policyFile := cmd.Flags().Lookup("policy").Value.String()
		dryRun := cmd.Flags().Lookup("dry-run").Value.String() == "true"
		manifestDir := cmd.Flags().Lookup("manifest-dir").Value.String()
		resume := cmd.Flags().Lookup("resume").Value.String() == "true"
		checkpointDir := cmd.Flags().Lookup("checkpoint-dir").Value.String()
		checkpointID := cmd.Flags().Lookup("checkpoint-id").Value.String()

		// Use smart defaults if not changed
		if !cmd.Flags().Changed("plan") {
			planFile = defaults.PlanFile()
		}
		if !cmd.Flags().Changed("policy") {
			policyFile = defaults.PolicyFile()
		}
		if !cmd.Flags().Changed("manifest-dir") {
			manifestDir = defaults.ManifestDir()
		}
		if !cmd.Flags().Changed("checkpoint-dir") {
			checkpointDir = defaults.CheckpointDir()
		}

		// Validate plan file exists with helpful error
		if err := ux.ValidateRequiredFile(planFile, "Plan file", "specular plan"); err != nil {
			return ux.EnhanceError(err)
		}

		// Load plan
		p, err := plan.LoadPlan(planFile)
		if err != nil {
			return ux.FormatError(err, "loading plan file")
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

		// Setup checkpoint manager
		checkpointMgr := checkpoint.NewManager(checkpointDir, true, 30*time.Second)
		var cpState *checkpoint.State

		// Generate operation ID from plan file if not provided
		if checkpointID == "" {
			checkpointID = fmt.Sprintf("build-%s-%d", planFile, time.Now().Unix())
		}

		// Initialize progress indicator
		progressIndicator := progress.NewIndicator(progress.Config{
			Writer:      os.Stdout,
			ShowSpinner: true,
		})

		// Handle resume if requested
		if resume {
			if checkpointMgr.Exists(checkpointID) {
				cpState, err = checkpointMgr.Load(checkpointID)
				if err != nil {
					return fmt.Errorf("failed to load checkpoint: %w", err)
				}

				// Use progress indicator for formatted resume info
				progressIndicator.SetState(cpState)
				progressIndicator.PrintResumeInfo()
			} else {
				fmt.Printf("No checkpoint found for: %s\n", checkpointID)
				fmt.Println("Starting fresh execution...")
				cpState = checkpoint.NewState(checkpointID)
			}
		} else {
			cpState = checkpoint.NewState(checkpointID)
		}

		// Set state in progress indicator
		progressIndicator.SetState(cpState)

		// Store metadata
		cpState.SetMetadata("plan", planFile)
		cpState.SetMetadata("policy", policyFile)
		cpState.SetMetadata("dry_run", fmt.Sprintf("%v", dryRun))

		// Initialize tasks in checkpoint state
		for _, task := range p.Tasks {
			if _, exists := cpState.Tasks[task.ID.String()]; !exists {
				cpState.UpdateTask(task.ID.String(), "pending", nil)
			}
		}

		// Save initial checkpoint
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save initial checkpoint: %v\n", saveErr)
		}

		// Initialize image cache
		verbose := cmd.Flags().Lookup("verbose").Value.String() == "true"
		enableCache := cmd.Flags().Lookup("enable-cache").Value.String() == "true"
		cacheDir := cmd.Flags().Lookup("cache-dir").Value.String()
		cacheMaxAgeStr := cmd.Flags().Lookup("cache-max-age").Value.String()
		cacheMaxAge, parseErr := time.ParseDuration(cacheMaxAgeStr)
		if parseErr != nil {
			cacheMaxAge = 7 * 24 * time.Hour // default
		}

		var imageCache *exec.ImageCache
		if enableCache {
			imageCache = exec.NewImageCache(cacheDir, cacheMaxAge)
			if loadErr := imageCache.LoadManifest(); loadErr != nil {
				fmt.Printf("Warning: failed to load cache manifest: %v\n", loadErr)
			}
		}

		// Create executor with checkpoint support
		executor := &exec.Executor{
			Policy:      pol,
			DryRun:      dryRun,
			ManifestDir: manifestDir,
			ImageCache:  imageCache,
			Verbose:     verbose,
		}

		// Execute plan
		fmt.Printf("Executing plan with %d tasks...\n\n", len(p.Tasks))

		// Start progress indicator
		progressIndicator.Start()
		defer progressIndicator.Stop()

		result, err := executor.Execute(p)
		if err != nil {
			// Stop progress indicator before error handling
			progressIndicator.Stop()

			// Update checkpoint with failure
			cpState.Status = "failed"
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}
			return fmt.Errorf("execution failed: %w", err)
		}

		// Stop progress indicator
		progressIndicator.Stop()

		// Update checkpoint with results and progress indicator
		for taskID, taskResult := range result.TaskResults {
			if taskResult.ExitCode == 0 {
				progressIndicator.UpdateTask(taskID, "completed", nil)
			} else {
				progressIndicator.UpdateTask(taskID, "failed", taskResult.Error)
			}
		}

		// Mark as completed
		cpState.Status = "completed"
		if finalSaveErr := checkpointMgr.Save(cpState); finalSaveErr != nil {
			fmt.Printf("Warning: failed to save final checkpoint: %v\n", finalSaveErr)
		}

		// Print summary using progress indicator
		progressIndicator.PrintSummary()

		// Check for failures
		if result.FailedTasks > 0 {
			return fmt.Errorf("execution completed with %d failed tasks", result.FailedTasks)
		}

		fmt.Println("\n✓ All tasks completed successfully")

		// Clean up checkpoint on success unless user wants to keep it
		keepCheckpoint := cmd.Flags().Lookup("keep-checkpoint").Value.String() == "true"
		if !keepCheckpoint {
			if deleteErr := checkpointMgr.Delete(checkpointID); deleteErr != nil {
				fmt.Printf("Warning: failed to delete checkpoint: %v\n", deleteErr)
			} else {
				fmt.Printf("Checkpoint cleaned up: %s\n", checkpointID)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().String("plan", "plan.json", "Plan file to execute")
	buildCmd.Flags().String("policy", ".specular/policy.yaml", "Policy file for enforcement")
	buildCmd.Flags().Bool("dry-run", false, "Show what would be executed without running")
	buildCmd.Flags().String("manifest-dir", ".specular/runs", "Directory for run manifests")
	buildCmd.Flags().String("fail-on", "", "Fail on conditions (comma-separated: drift,lint,test,security)")
	buildCmd.Flags().Bool("resume", false, "Resume from previous checkpoint")
	buildCmd.Flags().String("checkpoint-dir", ".specular/checkpoints", "Directory for checkpoints")
	buildCmd.Flags().String("checkpoint-id", "", "Checkpoint ID (auto-generated if not provided)")
	buildCmd.Flags().Bool("keep-checkpoint", false, "Keep checkpoint after successful completion")
	buildCmd.Flags().Bool("enable-cache", true, "Enable Docker image caching")
	buildCmd.Flags().String("cache-dir", ".specular/cache", "Directory for image cache")
	buildCmd.Flags().Duration("cache-max-age", 7*24*time.Hour, "Maximum cache age")
	buildCmd.Flags().Bool("verbose", false, "Verbose output")
}
