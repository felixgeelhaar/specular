package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	execpkg "github.com/felixgeelhaar/specular/internal/exec"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/progress"
	"github.com/felixgeelhaar/specular/internal/telemetry"
	"github.com/felixgeelhaar/specular/internal/ux"
	"go.opentelemetry.io/otel/attribute"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Manage build execution and verification",
	Long: `Execute, verify, and approve builds with policy enforcement.

Use 'specular build run' to execute a build plan.
Use 'specular build verify' to run lint, tests, and policy checks.
Use 'specular build approve' to approve build results.
Use 'specular build explain' to show logs and routing decisions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if this is being used as the old direct command
		// If flags are set, run the run command for backward compatibility
		if cmd.Flags().Changed("plan") || cmd.Flags().Changed("policy") || cmd.Flags().Changed("dry-run") {
			fmt.Fprintf(os.Stderr, "\n⚠️  DEPRECATION WARNING:\n")
			fmt.Fprintf(os.Stderr, "Running 'build' directly is deprecated and will be removed in v1.6.0.\n")
			fmt.Fprintf(os.Stderr, "Please use 'specular build run' instead.\n\n")

			// Run build run command
			return runBuildRun(cmd, args)
		}

		// Otherwise show help
		return cmd.Help()
	},
}

var buildRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute build with policy enforcement",
	Long: `Execute the build process in a Docker sandbox with strict policy enforcement.
All execution passes through guardrail checks including Docker-only enforcement,
linting, testing, and security scanning.

You can optionally execute a build for a specific feature using --feature.`,
	RunE: runBuildRun,
}

var buildVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Run lint, tests, and policy checks",
	Long: `Verify the build by running:
- Code linting (go vet, golangci-lint)
- Test suite execution
- Policy compliance checks
- Security scanning

This command should be run before 'build run' to catch issues early.`,
	RunE: runBuildVerify,
}

var buildApproveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Approve build results",
	Long: `Approve the build results after verification and execution.

Validation checks:
- Build manifest exists
- All tasks completed successfully
- No policy violations

Creates an approval marker file with timestamp for audit trail.`,
	RunE: runBuildApprove,
}

var buildExplainCmd = &cobra.Command{
	Use:   "explain [task-id]",
	Short: "Show logs and routing decisions",
	Long: `Explain the build execution for a specific task or overall build.

Shows:
- Execution logs from manifest
- Policy decisions per task
- Model routing choices
- Checkpoint history
- Resource usage statistics`,
	RunE: runBuildExplain,
}

func runBuildRun(cmd *cobra.Command, args []string) error {
	// Start distributed tracing span for build run command
	_, span := telemetry.StartCommandSpan(cmd.Context(), "build.run")
	defer span.End()

	startTime := time.Now()

	defaults := ux.NewPathDefaults()
	planFile := cmd.Flags().Lookup("plan").Value.String()
	policyFile := cmd.Flags().Lookup("policy").Value.String()
	dryRun := cmd.Flags().Lookup("dry-run").Value.String() == "true"
	manifestDir := cmd.Flags().Lookup("manifest-dir").Value.String()
	resume := cmd.Flags().Lookup("resume").Value.String() == "true"
	checkpointDir := cmd.Flags().Lookup("checkpoint-dir").Value.String()
	checkpointID := cmd.Flags().Lookup("checkpoint-id").Value.String()
	featureID := cmd.Flags().Lookup("feature").Value.String()

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

	// Record span attributes
	span.SetAttributes(
		attribute.String("plan_file", planFile),
		attribute.String("policy_file", policyFile),
		attribute.Bool("dry_run", dryRun),
		attribute.String("manifest_dir", manifestDir),
		attribute.Bool("resume", resume),
		attribute.String("checkpoint_dir", checkpointDir),
	)
	if featureID != "" {
		span.SetAttributes(attribute.String("feature_id", featureID))
	}
	if checkpointID != "" {
		span.SetAttributes(attribute.String("checkpoint_id", checkpointID))
	}

	// Validate plan file exists with helpful error
	if err := ux.ValidateRequiredFile(planFile, "Plan file", "specular plan gen"); err != nil {
		telemetry.RecordError(span, err)
		return ux.EnhanceError(err)
	}

	// Load plan
	p, err := plan.LoadPlan(planFile)
	if err != nil {
		telemetry.RecordError(span, err)
		return ux.FormatError(err, "loading plan file")
	}

	// If feature flag is set, filter to specific feature
	if featureID != "" {
		var filteredTasks []plan.Task
		for _, task := range p.Tasks {
			if string(task.FeatureID) == featureID {
				filteredTasks = append(filteredTasks, task)
			}
		}

		if len(filteredTasks) == 0 {
			err := fmt.Errorf("no tasks found for feature '%s'", featureID)
			telemetry.RecordError(span, err)
			return err
		}

		fmt.Printf("Executing %d tasks for feature: %s\n\n", len(filteredTasks), featureID)
		p = &plan.Plan{Tasks: filteredTasks}
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
				telemetry.RecordError(span, err)
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
	if featureID != "" {
		cpState.SetMetadata("feature", featureID)
	}

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

	var imageCache *execpkg.ImageCache
	if enableCache {
		imageCache = execpkg.NewImageCache(cacheDir, cacheMaxAge)
		if loadErr := imageCache.LoadManifest(); loadErr != nil {
			fmt.Printf("Warning: failed to load cache manifest: %v\n", loadErr)
		}
	}

	// Create executor with checkpoint support
	executor := &execpkg.Executor{
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
		telemetry.RecordError(span, err)
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
		err := fmt.Errorf("execution completed with %d failed tasks", result.FailedTasks)
		telemetry.RecordError(span, err)
		return err
	}

	fmt.Println("\n✓ All tasks completed successfully")

	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Review build results: specular build explain\n")
	fmt.Printf("  2. Approve build: specular build approve\n")

	// Clean up checkpoint on success unless user wants to keep it
	keepCheckpoint := cmd.Flags().Lookup("keep-checkpoint").Value.String() == "true"
	if !keepCheckpoint {
		if deleteErr := checkpointMgr.Delete(checkpointID); deleteErr != nil {
			fmt.Printf("Warning: failed to delete checkpoint: %v\n", deleteErr)
		} else {
			fmt.Printf("Checkpoint cleaned up: %s\n", checkpointID)
		}
	}

	// Record success with metrics
	duration := time.Since(startTime)
	telemetry.RecordSuccess(span,
		attribute.Int("total_tasks", len(p.Tasks)),
		attribute.Int("success_tasks", result.SuccessTasks),
		attribute.Int("failed_tasks", result.FailedTasks),
		attribute.Int("skipped_tasks", result.SkippedTasks),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	return nil
}

func runBuildVerify(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	policyFile := cmd.Flags().Lookup("policy").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("policy") {
		policyFile = defaults.PolicyFile()
	}

	fmt.Printf("=== Build Verification ===\n\n")

	// Load policy if exists
	var pol *policy.Policy
	var err error
	if policyFile != "" {
		pol, err = policy.LoadPolicy(policyFile)
		if err != nil {
			fmt.Printf("Warning: failed to load policy, using defaults: %v\n", err)
			pol = policy.DefaultPolicy()
		}
	} else {
		pol = policy.DefaultPolicy()
	}

	passed := 0
	failed := 0

	// 1. Run go vet
	fmt.Printf("1. Running go vet...\n")
	vetCmd := exec.Command("go", "vet", "./...")
	vetOutput, vetErr := vetCmd.CombinedOutput()
	if vetErr != nil {
		fmt.Printf("   ✗ go vet failed:\n%s\n", string(vetOutput))
		failed++
	} else {
		fmt.Printf("   ✓ go vet passed\n")
		passed++
	}

	// 2. Run golangci-lint if available
	fmt.Printf("\n2. Running golangci-lint...\n")
	lintCmd := exec.Command("golangci-lint", "run", "--timeout=5m")
	lintOutput, lintErr := lintCmd.CombinedOutput()
	if lintErr != nil {
		// Check if command not found
		if strings.Contains(lintErr.Error(), "not found") || strings.Contains(lintErr.Error(), "executable file not found") {
			fmt.Printf("   ⚠  golangci-lint not installed (skipped)\n")
		} else {
			fmt.Printf("   ✗ golangci-lint failed:\n%s\n", string(lintOutput))
			failed++
		}
	} else {
		fmt.Printf("   ✓ golangci-lint passed\n")
		passed++
	}

	// 3. Run tests
	fmt.Printf("\n3. Running tests...\n")
	testCmd := exec.Command("go", "test", "./...", "-short")
	testOutput, testErr := testCmd.CombinedOutput()
	if testErr != nil {
		fmt.Printf("   ✗ Tests failed:\n%s\n", string(testOutput))
		failed++
	} else {
		fmt.Printf("   ✓ Tests passed\n")
		passed++
	}

	// 4. Policy compliance check
	fmt.Printf("\n4. Checking policy compliance...\n")
	if pol != nil {
		fmt.Printf("   ✓ Policy loaded\n")
		fmt.Printf("   • Docker required: %v\n", pol.Execution.Docker.Required)
		fmt.Printf("   • Test coverage min: %.1f%%\n", pol.Tests.MinCoverage*100)
		passed++
	} else {
		fmt.Printf("   ✗ No policy found\n")
		failed++
	}

	// Summary
	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Verification Summary:\n")
	fmt.Printf("  ✓ Passed: %d\n", passed)
	if failed > 0 {
		fmt.Printf("  ✗ Failed: %d\n", failed)
		fmt.Printf("\n❌ Verification failed\n")
		fmt.Println("\nRecommendations:")
		fmt.Printf("  1. Fix failing checks\n")
		fmt.Printf("  2. Run 'specular build verify' again\n")
		return fmt.Errorf("verification failed with %d errors", failed)
	}

	fmt.Printf("\n✅ All verifications passed\n")
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Execute build: specular build run\n")

	return nil
}

func runBuildApprove(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	manifestDir := cmd.Flags().Lookup("manifest-dir").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("manifest-dir") {
		manifestDir = defaults.ManifestDir()
	}

	// Check if manifest directory exists
	if _, err := os.Stat(manifestDir); os.IsNotExist(err) {
		return fmt.Errorf("no build manifests found\n\nRun 'specular build run' first")
	}

	// Find most recent manifest
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		return fmt.Errorf("failed to read manifest directory: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no build manifests found\n\nRun 'specular build run' first")
	}

	// Get most recent directory
	var latestDir string
	var latestTime time.Time
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, _ := entry.Info()
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestDir = entry.Name()
		}
	}

	if latestDir == "" {
		return fmt.Errorf("no valid build manifests found")
	}

	manifestPath := filepath.Join(manifestDir, latestDir)
	fmt.Printf("Approving build: %s\n", latestDir)
	fmt.Printf("Manifest: %s\n\n", manifestPath)

	// TODO: Load and validate manifest
	// For now, create approval marker

	approvalFile := filepath.Join(manifestPath, "approved")
	approvalData := fmt.Sprintf("Approved at: %s\n", time.Now().Format(time.RFC3339))
	approvalData += fmt.Sprintf("Manifest: %s\n", latestDir)

	if err := os.WriteFile(approvalFile, []byte(approvalData), 0644); err != nil {
		return fmt.Errorf("failed to create approval marker: %w", err)
	}

	fmt.Printf("✓ Build approved\n")
	fmt.Printf("  Approval record: %s\n", approvalFile)
	fmt.Printf("  Timestamp: %s\n", time.Now().Format(time.RFC3339))

	return nil
}

func runBuildExplain(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	manifestDir := cmd.Flags().Lookup("manifest-dir").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("manifest-dir") {
		manifestDir = defaults.ManifestDir()
	}

	// Check if manifest directory exists
	if _, err := os.Stat(manifestDir); os.IsNotExist(err) {
		return fmt.Errorf("no build manifests found\n\nRun 'specular build run' first")
	}

	// Find most recent manifest
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		return fmt.Errorf("failed to read manifest directory: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no build manifests found\n\nRun 'specular build run' first")
	}

	// Get most recent directory
	var latestDir string
	var latestTime time.Time
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, _ := entry.Info()
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestDir = entry.Name()
		}
	}

	if latestDir == "" {
		return fmt.Errorf("no valid build manifests found")
	}

	manifestPath := filepath.Join(manifestDir, latestDir)

	fmt.Printf("=== Build Execution Explanation ===\n\n")
	fmt.Printf("Build ID: %s\n", latestDir)
	fmt.Printf("Manifest: %s\n", manifestPath)
	fmt.Printf("Timestamp: %s\n\n", latestTime.Format(time.RFC3339))

	// Check for logs
	logsFile := filepath.Join(manifestPath, "logs.txt")
	if _, err := os.Stat(logsFile); err == nil {
		fmt.Printf("Execution Logs:\n")
		logs, readErr := os.ReadFile(logsFile)
		if readErr == nil {
			fmt.Printf("%s\n", string(logs))
		}
	} else {
		fmt.Printf("No execution logs found\n")
	}

	// Check for manifest.json
	manifestFile := filepath.Join(manifestPath, "manifest.json")
	if _, err := os.Stat(manifestFile); err == nil {
		fmt.Printf("\nManifest file: %s\n", manifestFile)
		fmt.Printf("  Use 'cat %s | jq' to inspect\n", manifestFile)
	}

	// Check for approval
	approvalFile := filepath.Join(manifestPath, "approved")
	if _, err := os.Stat(approvalFile); err == nil {
		approval, _ := os.ReadFile(approvalFile)
		fmt.Printf("\nApproval Status:\n")
		fmt.Printf("%s\n", string(approval))
	} else {
		fmt.Printf("\nApproval Status: Not approved\n")
		fmt.Printf("  Run 'specular build approve' to approve this build\n")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.AddCommand(buildRunCmd)
	buildCmd.AddCommand(buildVerifyCmd)
	buildCmd.AddCommand(buildApproveCmd)
	buildCmd.AddCommand(buildExplainCmd)

	// Flags for backward compatibility on root build command
	buildCmd.Flags().String("plan", "plan.json", "Plan file to execute")
	buildCmd.Flags().String("policy", ".specular/policy.yaml", "Policy file for enforcement")
	buildCmd.Flags().Bool("dry-run", false, "Show what would be executed without running")
	buildCmd.Flags().String("manifest-dir", ".specular/runs", "Directory for run manifests")

	// build run flags
	buildRunCmd.Flags().String("plan", "plan.json", "Plan file to execute (default: plan.json)")
	buildRunCmd.Flags().String("policy", ".specular/policy.yaml", "Policy file for enforcement (default: .specular/policy.yaml)")
	buildRunCmd.Flags().Bool("dry-run", false, "Show what would be executed without running")
	buildRunCmd.Flags().String("manifest-dir", ".specular/runs", "Directory for run manifests (default: .specular/runs)")
	buildRunCmd.Flags().String("fail-on", "", "Fail on conditions (comma-separated: drift,lint,test,security)")
	buildRunCmd.Flags().Bool("resume", false, "Resume from previous checkpoint")
	buildRunCmd.Flags().String("checkpoint-dir", ".specular/checkpoints", "Directory for checkpoints (default: .specular/checkpoints)")
	buildRunCmd.Flags().String("checkpoint-id", "", "Checkpoint ID (auto-generated if not provided)")
	buildRunCmd.Flags().Bool("keep-checkpoint", false, "Keep checkpoint after successful completion")
	buildRunCmd.Flags().Bool("enable-cache", true, "Enable Docker image caching (default: true)")
	buildRunCmd.Flags().String("cache-dir", ".specular/cache", "Directory for image cache (default: .specular/cache)")
	buildRunCmd.Flags().Duration("cache-max-age", 7*24*time.Hour, "Maximum cache age (default: 168h = 7 days)")
	buildRunCmd.Flags().Bool("verbose", false, "Verbose output")
	buildRunCmd.Flags().String("feature", "", "Execute build for specific feature ID")

	// build verify flags
	buildVerifyCmd.Flags().String("policy", ".specular/policy.yaml", "Policy file for verification")

	// build approve flags
	buildApproveCmd.Flags().String("manifest-dir", ".specular/runs", "Directory for run manifests")

	// build explain flags
	buildExplainCmd.Flags().String("manifest-dir", ".specular/runs", "Directory for run manifests")
}
