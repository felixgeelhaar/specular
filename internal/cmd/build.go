package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	"github.com/felixgeelhaar/specular/internal/exec"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/progress"
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
		resume, _ := cmd.Flags().GetBool("resume")
		checkpointDir, _ := cmd.Flags().GetString("checkpoint-dir")
		checkpointID, _ := cmd.Flags().GetString("checkpoint-id")

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
			if _, exists := cpState.Tasks[task.ID]; !exists {
				cpState.UpdateTask(task.ID, "pending", nil)
			}
		}

		// Save initial checkpoint
		if err := checkpointMgr.Save(cpState); err != nil {
			fmt.Printf("Warning: failed to save initial checkpoint: %v\n", err)
		}

		// Initialize image cache
		verbose, _ := cmd.Flags().GetBool("verbose")
		enableCache, _ := cmd.Flags().GetBool("enable-cache")
		cacheDir, _ := cmd.Flags().GetString("cache-dir")
		cacheMaxAge, _ := cmd.Flags().GetDuration("cache-max-age")

		var imageCache *exec.ImageCache
		if enableCache {
			imageCache = exec.NewImageCache(cacheDir, cacheMaxAge)
			if err := imageCache.LoadManifest(); err != nil {
				fmt.Printf("Warning: failed to load cache manifest: %v\n", err)
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
		if err := checkpointMgr.Save(cpState); err != nil {
			fmt.Printf("Warning: failed to save final checkpoint: %v\n", err)
		}

		// Print summary using progress indicator
		progressIndicator.PrintSummary()

		// Check for failures
		if result.FailedTasks > 0 {
			return fmt.Errorf("execution completed with %d failed tasks", result.FailedTasks)
		}

		fmt.Println("\n✓ All tasks completed successfully")

		// Clean up checkpoint on success unless user wants to keep it
		keepCheckpoint, _ := cmd.Flags().GetBool("keep-checkpoint")
		if !keepCheckpoint {
			if err := checkpointMgr.Delete(checkpointID); err != nil {
				fmt.Printf("Warning: failed to delete checkpoint: %v\n", err)
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
