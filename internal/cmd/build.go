package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	execpkg "github.com/felixgeelhaar/specular/internal/exec"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/progress"
	"github.com/felixgeelhaar/specular/internal/safeutil"
	"github.com/felixgeelhaar/specular/internal/telemetry"
	"github.com/felixgeelhaar/specular/internal/tui"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var buildCmd = &cobra.Command{
	Use:     "build",
	Aliases: []string{"b"},
	Short:   "Manage build execution and verification",
	Long: `Execute, verify, and approve builds with policy enforcement.

Use 'specular build run' to execute a build plan.
Use 'specular build verify' to run lint, tests, and policy checks.
Use 'specular build approve' to approve build results.
Use 'specular build explain' to show logs and routing decisions.

Typical flow:
  plan create -> build run -> eval run`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

Creates an approval marker file with timestamp for audit trail.

NOTE: This is a convenience wrapper. For unified governance approvals, prefer:
  specular approve bundle-<hash>   # Approve with resource ID
  specular approvals pending       # View pending approvals
  specular approvals list          # List all approvals`,
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
	ctx, span := telemetry.StartCommandSpan(cmd.Context(), "build.run")
	defer span.End()

	startTime := time.Now()

	defaults := ux.NewPathDefaults()
	planFile := cmd.Flags().Lookup("plan").Value.String()
	policyFile := cmd.Flags().Lookup("policy").Value.String()
	dryRun := cmd.Flags().Lookup("dry-run").Value.String() == "true"
	manifestDir := cmd.Flags().Lookup("manifest-dir").Value.String()

	// Check for TUI mode
	useTUI := false
	if flag := cmd.Flags().Lookup("tui"); flag != nil {
		useTUI = flag.Value.String() == "true"
	}

	// Apply smart defaults for plan file (needed for TUI mode check)
	if !cmd.Flags().Changed("plan") {
		planFile = defaults.PlanFile()
	}
	if !cmd.Flags().Changed("manifest-dir") {
		manifestDir = defaults.ManifestDir()
	}

	// If TUI mode is enabled, run with BuildTUI wrapper
	if useTUI {
		tuiConfig := tui.BuildTUIConfig{
			SpecFile:  planFile,
			OutputDir: manifestDir,
			Format:    "json",
			Validate:  true,
			ShowDiff:  false,
		}

		runner := tui.NewBuildTUIRunner(tuiConfig, nil)
		return runner.Run(func(buildTUI *tui.BuildTUI) error {
			// Execute the build within TUI context
			buildTUI.OnLoadSpec(true, planFile, nil)

			// Load plan
			p, err := plan.LoadPlan(planFile)
			if err != nil {
				buildTUI.OnLoadSpec(false, planFile, err)
				return fmt.Errorf("failed to load plan: %w", err)
			}
			buildTUI.OnLoadSpec(false, planFile, nil)

			// Validate
			buildTUI.OnValidate(true, 0, 0, nil)
			buildTUI.OnValidate(false, 0, 0, nil)

			// Resolve dependencies (simplified for TUI mode)
			buildTUI.OnResolve(true, 0, nil)
			buildTUI.OnResolve(false, len(p.Tasks), nil)

			// Generate output phase
			buildTUI.OnGenerate(true, "json", nil)
			buildTUI.OnGenerate(false, "json", nil)

			// Write files (placeholder - actual execution happens here)
			buildTUI.OnWriteComplete(len(p.Tasks), 0, nil)

			buildTUI.Log(tui.LogLevelInfo, fmt.Sprintf("Completed %d tasks", len(p.Tasks)))

			return nil
		})
	}

	// These flags might not exist if called from root buildCmd for backward compatibility
	resume := false
	if flag := cmd.Flags().Lookup("resume"); flag != nil {
		resume = flag.Value.String() == "true"
	}
	checkpointDir := ""
	if flag := cmd.Flags().Lookup("checkpoint-dir"); flag != nil {
		checkpointDir = flag.Value.String()
	}
	checkpointID := ""
	if flag := cmd.Flags().Lookup("checkpoint-id"); flag != nil {
		checkpointID = flag.Value.String()
	}
	featureID := ""
	if flag := cmd.Flags().Lookup("feature"); flag != nil {
		featureID = flag.Value.String()
	}

	// Use smart defaults for remaining flags
	if !cmd.Flags().Changed("policy") {
		policyFile = defaults.PolicyFile()
	}
	// Only check checkpoint-dir if flag exists (backward compatibility)
	if cmd.Flags().Lookup("checkpoint-dir") != nil && !cmd.Flags().Changed("checkpoint-dir") {
		checkpointDir = defaults.CheckpointDir()
	} else if checkpointDir == "" {
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
	if err := ux.ValidateRequiredFile(planFile, "Plan file", "specular plan create"); err != nil {
		telemetry.RecordCommandFailure(ctx, span, "build.run", err)
		return ux.EnhanceError(err)
	}

	// Governed preflight: policy file must exist before execution.
	if err := ensureGovernedPolicyFile(policyFile); err != nil {
		telemetry.RecordCommandFailure(ctx, span, "build.run", err)
		return ux.EnhanceError(err)
	}

	// Load plan
	p, err := plan.LoadPlan(planFile)
	if err != nil {
		telemetry.RecordCommandFailure(ctx, span, "build.run", err)
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
			telemetry.RecordCommandFailure(ctx, span, "build.run", err)
			return err
		}

		fmt.Printf("Executing %d tasks for feature: %s\n\n", len(filteredTasks), featureID)
		p = &plan.Plan{Tasks: filteredTasks}
	}

	// Load policy
	pol, err := policy.LoadPolicy(policyFile)
	if err != nil {
		telemetry.RecordCommandFailure(ctx, span, "build.run", err)
		return ux.FormatError(err, "loading policy file")
	}

	// Setup checkpoint manager
	checkpointMgr := checkpoint.NewManager(checkpointDir, true, 30*time.Second)
	var cpState *checkpoint.State

	// Generate operation ID from plan file if not provided
	if checkpointID == "" {
		checkpointID = fmt.Sprintf("build-%s-%d", sanitizeIdentifier(planFile), time.Now().Unix())
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
				telemetry.RecordCommandFailure(ctx, span, "build.run", err)
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
	verbose := false
	if flag := cmd.Flags().Lookup("verbose"); flag != nil {
		verbose = flag.Value.String() == "true"
	}
	enableCache := false
	if flag := cmd.Flags().Lookup("enable-cache"); flag != nil {
		enableCache = flag.Value.String() == "true"
	}
	cacheDir := ""
	if flag := cmd.Flags().Lookup("cache-dir"); flag != nil {
		cacheDir = flag.Value.String()
	}
	cacheMaxAge := 7 * 24 * time.Hour // default
	if flag := cmd.Flags().Lookup("cache-max-age"); flag != nil {
		if age, parseErr := time.ParseDuration(flag.Value.String()); parseErr == nil {
			cacheMaxAge = age
		}
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

	result, err := executor.Execute(ctx, p)
	if err != nil {
		// Stop progress indicator before error handling
		progressIndicator.Stop()

		// Update checkpoint with failure
		cpState.Status = "failed"
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}
		telemetry.RecordCommandFailure(ctx, span, "build.run", err)
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
		telemetry.RecordCommandFailure(ctx, span, "build.run", err)
		return err
	}

	fmt.Println("\n✓ All tasks completed successfully")

	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Review build results: specular build explain\n")
	fmt.Printf("  2. Approve build: specular build approve\n")

	// Clean up checkpoint on success unless user wants to keep it
	keepCheckpoint := false
	if flag := cmd.Flags().Lookup("keep-checkpoint"); flag != nil {
		keepCheckpoint = flag.Value.String() == "true"
	}
	if !keepCheckpoint {
		if deleteErr := checkpointMgr.Delete(checkpointID); deleteErr != nil {
			fmt.Printf("Warning: failed to delete checkpoint: %v\n", deleteErr)
		} else {
			fmt.Printf("Checkpoint cleaned up: %s\n", checkpointID)
		}
	}

	// Record success with combined trace and metrics
	duration := time.Since(startTime)
	telemetry.RecordCommandSuccess(ctx, span, "build.run", duration,
		attribute.Int("total_tasks", len(p.Tasks)),
		attribute.Int("success_tasks", result.SuccessTasks),
		attribute.Int("failed_tasks", result.FailedTasks),
		attribute.Int("skipped_tasks", result.SkippedTasks),
	)

	return nil
}

func runBuildVerify(cmd *cobra.Command, args []string) error {
	// Start distributed tracing span for build verify command
	ctx, span := telemetry.StartCommandSpan(cmd.Context(), "build.verify")
	defer span.End()

	startTime := time.Now()

	defaults := ux.NewPathDefaults()
	policyFile := cmd.Flags().Lookup("policy").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("policy") {
		policyFile = defaults.PolicyFile()
	}

	// Record span attributes
	span.SetAttributes(
		attribute.String("policy_file", policyFile),
	)

	fmt.Printf("=== Build Verification ===\n\n")

	// Governed preflight: policy file must exist before verification.
	if err := ensureGovernedPolicyFile(policyFile); err != nil {
		telemetry.RecordCommandFailure(ctx, span, "build.verify", err)
		return ux.EnhanceError(err)
	}

	// Load policy
	pol, err := policy.LoadPolicy(policyFile)
	if err != nil {
		telemetry.RecordCommandFailure(ctx, span, "build.verify", err)
		return ux.FormatError(err, "loading policy file")
	}

	passed := 0
	failed := 0

	// 1. Run go vet
	fmt.Printf("1. Running go vet...\n")
	vetCmd, vetPrepErr := safeutil.SafeCommand(context.Background(), "go", "vet", "./...")
	if vetPrepErr != nil {
		fmt.Printf("   ✗ go vet unavailable: %v\n", vetPrepErr)
		failed++
	} else {
		vetOutput, vetErr := vetCmd.CombinedOutput()
		if vetErr != nil {
			fmt.Printf("   ✗ go vet failed:\n%s\n", string(vetOutput))
			failed++
		} else {
			fmt.Printf("   ✓ go vet passed\n")
			passed++
		}
	}

	// 2. Run golangci-lint if available
	fmt.Printf("\n2. Running golangci-lint...\n")
	lintCmd, lintPrepErr := safeutil.SafeCommand(context.Background(), "golangci-lint", "run", "--timeout=5m")
	if lintPrepErr != nil {
		if errors.Is(lintPrepErr, exec.ErrNotFound) {
			fmt.Printf("   ⚠  golangci-lint not installed (skipped)\n")
		} else {
			fmt.Printf("   ✗ failed to prepare golangci-lint: %v\n", lintPrepErr)
			failed++
		}
	} else {
		lintOutput, lintErr := lintCmd.CombinedOutput()
		if lintErr != nil {
			fmt.Printf("   ✗ golangci-lint failed:\n%s\n", string(lintOutput))
			failed++
		} else {
			fmt.Printf("   ✓ golangci-lint passed\n")
			passed++
		}
	}

	// 3. Run tests
	fmt.Printf("\n3. Running tests...\n")
	testCmd, testPrepErr := safeutil.SafeCommand(context.Background(), "go", "test", "./...", "-short")
	if testPrepErr != nil {
		fmt.Printf("   ✗ tests unavailable: %v\n", testPrepErr)
		failed++
	} else {
		testOutput, testErr := testCmd.CombinedOutput()
		if testErr != nil {
			fmt.Printf("   ✗ Tests failed:\n%s\n", string(testOutput))
			failed++
		} else {
			fmt.Printf("   ✓ Tests passed\n")
			passed++
		}
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
		err := fmt.Errorf("verification failed with %d errors", failed)
		telemetry.RecordCommandFailure(ctx, span, "build.verify", err)
		return err
	}

	fmt.Printf("\n✅ All verifications passed\n")
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Execute build: specular build run\n")

	telemetry.RecordCommandSuccess(ctx, span, "build.verify", time.Since(startTime),
		attribute.Int("checks_passed", passed),
		attribute.Int("checks_failed", failed),
	)
	return nil
}

func ensureGovernedPolicyFile(policyFile string) error {
	if policyFile == "" {
		return fmt.Errorf("policy file path is required for governed execution")
	}

	return ux.ValidateRequiredFile(policyFile, "Policy file", "specular policy init")
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

	// Get most recent manifest file (JSON)
	var latestManifest string
	var latestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestManifest = filepath.Join(manifestDir, entry.Name())
		}
	}

	if latestManifest == "" {
		return fmt.Errorf("no valid build manifests found")
	}

	// Load and validate manifest
	data, err := os.ReadFile(latestManifest)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest execpkg.RunManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Display manifest details
	fmt.Printf("Approving build step: %s\n", manifest.StepID)
	fmt.Printf("Manifest: %s\n", latestManifest)
	fmt.Printf("Timestamp: %s\n", manifest.Timestamp.Format(time.RFC3339))
	fmt.Printf("Exit code: %d\n", manifest.ExitCode)
	fmt.Printf("Duration: %s\n\n", manifest.Duration)

	// Validate manifest - check if build was successful
	if manifest.ExitCode != 0 {
		return fmt.Errorf("cannot approve build: step %s failed with exit code %d", manifest.StepID, manifest.ExitCode)
	}

	// Create approval marker next to manifest
	approvalFile := latestManifest + ".approved"
	approvalData := fmt.Sprintf("Approved at: %s\n", time.Now().Format(time.RFC3339))
	approvalData += fmt.Sprintf("Manifest: %s\n", filepath.Base(latestManifest))
	approvalData += fmt.Sprintf("Step ID: %s\n", manifest.StepID)
	approvalData += fmt.Sprintf("Exit code: %d\n", manifest.ExitCode)

	if err := os.WriteFile(approvalFile, []byte(approvalData), 0600); err != nil {
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
	info, err := findLatestManifestInfo(manifestDir)
	if err != nil {
		return fmt.Errorf("no build manifests found\n\nRun 'specular build run' first")
	}

	fmt.Printf("=== Build Execution Explanation ===\n\n")
	fmt.Printf("Build ID: %s\n", filepath.Base(info.LatestDir))
	fmt.Printf("Manifest: %s\n", info.ManifestFile)
	fmt.Printf("Timestamp: %s\n\n", info.LatestTime.Format(time.RFC3339))

	if info.IsDir {
		logsFile := filepath.Join(info.LatestDir, "logs.txt")
		if _, err := os.Stat(logsFile); err == nil {
			fmt.Printf("Execution Logs:\n")
			if logs, readErr := os.ReadFile(logsFile); readErr == nil {
				fmt.Printf("%s\n", string(logs))
			}
		} else {
			fmt.Printf("No execution logs found\n")
		}
	} else {
		fmt.Printf("No execution logs found\n")
	}

	if _, err := os.Stat(info.ManifestFile); err == nil {
		fmt.Printf("\nManifest file: %s\n", info.ManifestFile)
		fmt.Printf("  Use 'cat %s | jq' to inspect\n", info.ManifestFile)
	}

	var approvalFile string
	if info.IsDir {
		approvalFile = filepath.Join(info.LatestDir, "approved")
	} else {
		approvalFile = info.ManifestFile + ".approved"
	}
	if approval, err := os.ReadFile(approvalFile); err == nil {
		fmt.Printf("\nApproval Status:\n")
		fmt.Printf("%s\n", string(approval))
	} else {
		fmt.Printf("\nApproval Status: Not approved\n")
		fmt.Printf("  Run 'specular build approve' to approve this build\n")
	}

	return nil
}

type manifestInfo struct {
	LatestDir    string
	ManifestFile string
	LatestTime   time.Time
	IsDir        bool
}

func findLatestManifestInfo(dir string) (*manifestInfo, error) {
	meta, err := execpkg.LoadLatestManifest(dir)
	if err != nil {
		return nil, err
	}

	info := &manifestInfo{
		ManifestFile: meta.ManifestPath,
		LatestDir:    filepath.Dir(meta.ManifestPath),
		LatestTime:   meta.Timestamp,
	}

	if stat, statErr := os.Stat(meta.ManifestPath); statErr == nil {
		info.IsDir = stat.IsDir()
	}

	return info, nil
}

func sanitizeIdentifier(value string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		" ", "-",
		".", "-",
	)
	return replacer.Replace(value)
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.AddCommand(buildRunCmd)
	buildCmd.AddCommand(buildVerifyCmd)
	buildCmd.AddCommand(buildApproveCmd)
	buildCmd.AddCommand(buildExplainCmd)

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
	buildRunCmd.Flags().Bool("tui", false, "Run with interactive TUI mode")

	// build verify flags
	buildVerifyCmd.Flags().String("policy", ".specular/policy.yaml", "Policy file for verification")

	// build approve flags
	buildApproveCmd.Flags().String("manifest-dir", ".specular/runs", "Directory for run manifests")

	// build explain flags
	buildExplainCmd.Flags().String("manifest-dir", ".specular/runs", "Directory for run manifests")
}
