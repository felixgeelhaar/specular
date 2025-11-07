package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	"github.com/felixgeelhaar/specular/internal/drift"
	"github.com/felixgeelhaar/specular/internal/eval"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/progress"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Run evaluation and drift detection",
	Long: `Execute comprehensive evaluation including:
- Plan drift detection (spec hash mismatches)
- Code drift detection (contract tests, API conformance)
- Infrastructure drift (policy violations)
- Test execution and coverage analysis
- Security scanning

Results are output in SARIF format for integration with CI/CD tools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planFile, _ := cmd.Flags().GetString("plan")
		lockFile, _ := cmd.Flags().GetString("lock")
		specFile, _ := cmd.Flags().GetString("spec")
		policyFile, _ := cmd.Flags().GetString("policy")
		reportFile, _ := cmd.Flags().GetString("report")
		failOnDrift, _ := cmd.Flags().GetBool("fail-on-drift")
		projectRoot, _ := cmd.Flags().GetString("project-root")
		apiSpecPath, _ := cmd.Flags().GetString("api-spec")
		ignoreGlobs, _ := cmd.Flags().GetStringSlice("ignore")
		resume, _ := cmd.Flags().GetBool("resume")
		checkpointDir, _ := cmd.Flags().GetString("checkpoint-dir")
		checkpointID, _ := cmd.Flags().GetString("checkpoint-id")

		// Setup checkpoint manager
		checkpointMgr := checkpoint.NewManager(checkpointDir, true, 30*time.Second)
		var cpState *checkpoint.State

		// Generate operation ID if not provided
		if checkpointID == "" {
			checkpointID = fmt.Sprintf("eval-%s-%d", planFile, time.Now().Unix())
		}

		// Initialize progress indicator
		progressIndicator := progress.NewIndicator(progress.Config{
			Writer:      os.Stdout,
			ShowSpinner: true,
		})

		// Handle resume if requested
		if resume {
			if checkpointMgr.Exists(checkpointID) {
				var err error
				cpState, err = checkpointMgr.Load(checkpointID)
				if err != nil {
					return fmt.Errorf("failed to load checkpoint: %w", err)
				}

				// Use progress indicator for formatted resume info
				progressIndicator.SetState(cpState)
				progressIndicator.PrintResumeInfo()
			} else {
				fmt.Printf("No checkpoint found for: %s\n", checkpointID)
				fmt.Println("Starting fresh evaluation...")
				cpState = checkpoint.NewState(checkpointID)
			}
		} else {
			cpState = checkpoint.NewState(checkpointID)
		}

		// Set state in progress indicator
		progressIndicator.SetState(cpState)

		// Store metadata
		cpState.SetMetadata("plan", planFile)
		cpState.SetMetadata("lock", lockFile)
		cpState.SetMetadata("spec", specFile)
		cpState.SetMetadata("policy", policyFile)

		// Initialize check tasks
		checks := []string{"quality-gate", "plan-drift", "code-drift", "infra-drift", "report-generation"}
		for _, checkID := range checks {
			if _, exists := cpState.Tasks[checkID]; !exists {
				progressIndicator.UpdateTask(checkID, "pending", nil)
			}
		}

		// Save initial checkpoint
		if err := checkpointMgr.Save(cpState); err != nil {
			fmt.Printf("Warning: failed to save initial checkpoint: %v\n", err)
		}

		// Start progress indicator
		progressIndicator.Start()
		defer progressIndicator.Stop()

		// Load plan
		p, err := plan.LoadPlan(planFile)
		if err != nil {
			return fmt.Errorf("failed to load plan: %w", err)
		}

		// Load SpecLock
		lock, err := spec.LoadSpecLock(lockFile)
		if err != nil {
			return fmt.Errorf("failed to load SpecLock: %w", err)
		}

		// Load spec for code drift detection
		s, err := spec.LoadSpec(specFile)
		if err != nil {
			return fmt.Errorf("failed to load spec: %w", err)
		}

		// Run eval gate if policy is provided
		if policyFile != "" && cpState.Tasks["quality-gate"].Status != "completed" {
			progressIndicator.UpdateTask("quality-gate", "running", nil)
			checkpointMgr.Save(cpState)

			pol, err := policy.LoadPolicy(policyFile)
			if err != nil {
				progressIndicator.UpdateTask("quality-gate", "failed", err)
				checkpointMgr.Save(cpState)
				return fmt.Errorf("failed to load policy: %w", err)
			}

			fmt.Println("Running quality gate checks...")
			gateReport, err := eval.RunEvalGate(eval.GateOptions{
				Policy:      pol,
				ProjectRoot: projectRoot,
				Verbose:     false,
			})
			if err != nil {
				progressIndicator.UpdateTask("quality-gate", "failed", err)
				checkpointMgr.Save(cpState)
				return fmt.Errorf("eval gate failed: %w", err)
			}

			// Print gate results
			fmt.Printf("\nQuality Gate Results:\n")
			fmt.Printf("  Total Checks: %d\n", len(gateReport.Checks))
			fmt.Printf("  Passed:       %d\n", gateReport.TotalPassed)
			fmt.Printf("  Failed:       %d\n", gateReport.TotalFailed)
			fmt.Printf("  Skipped:      %d\n", gateReport.TotalSkipped)
			fmt.Printf("  Duration:     %s\n\n", gateReport.Duration)

			for _, check := range gateReport.Checks {
				status := "✓"
				if !check.Passed {
					status = "✗"
				}
				fmt.Printf("  %s %s: %s (%.2fs)\n", status, check.Name, check.Message, check.Duration.Seconds())
			}
			fmt.Println()

			// Fail early if gate failed
			if !gateReport.AllPassed {
				progressIndicator.UpdateTask("quality-gate", "failed", fmt.Errorf("quality gate failed with %d failed checks", gateReport.TotalFailed))
				checkpointMgr.Save(cpState)
				return fmt.Errorf("quality gate failed with %d failed checks", gateReport.TotalFailed)
			}

			progressIndicator.UpdateTask("quality-gate", "completed", nil)
			checkpointMgr.Save(cpState)
		} else if policyFile == "" {
			progressIndicator.UpdateTask("quality-gate", "skipped", nil)
			checkpointMgr.Save(cpState)
		} else {
			fmt.Println("✓ Quality gate check already completed (skipping)")
		}

		// Detect plan drift
		if cpState.Tasks["plan-drift"].Status != "completed" {
			progressIndicator.UpdateTask("plan-drift", "running", nil)
			checkpointMgr.Save(cpState)

			fmt.Println("Detecting plan drift...")
			planDrift := drift.DetectPlanDrift(lock, p)

			progressIndicator.UpdateTask("plan-drift", "completed", nil)
			cpState.SetMetadata("plan_drift_count", fmt.Sprintf("%d", len(planDrift)))
			checkpointMgr.Save(cpState)
		} else {
			fmt.Println("✓ Plan drift check already completed (skipping)")
		}

		// Detect code drift
		if cpState.Tasks["code-drift"].Status != "completed" {
			progressIndicator.UpdateTask("code-drift", "running", nil)
			checkpointMgr.Save(cpState)

			fmt.Println("Detecting code drift...")
			codeDrift := drift.DetectCodeDrift(s, lock, drift.CodeDriftOptions{
				ProjectRoot: projectRoot,
				APISpecPath: apiSpecPath,
				IgnoreGlobs: ignoreGlobs,
			})

			progressIndicator.UpdateTask("code-drift", "completed", nil)
			cpState.SetMetadata("code_drift_count", fmt.Sprintf("%d", len(codeDrift)))
			checkpointMgr.Save(cpState)
		} else {
			fmt.Println("✓ Code drift check already completed (skipping)")
		}

		// Detect infrastructure drift
		var infraDrift []drift.Finding
		if cpState.Tasks["infra-drift"].Status != "completed" {
			progressIndicator.UpdateTask("infra-drift", "running", nil)
			checkpointMgr.Save(cpState)

			fmt.Println("Detecting infrastructure drift...")
			if policyFile != "" {
				pol, err := policy.LoadPolicy(policyFile)
				if err != nil {
					progressIndicator.UpdateTask("infra-drift", "failed", err)
					checkpointMgr.Save(cpState)
					return fmt.Errorf("failed to load policy: %w", err)
				}

				// Build task images map from plan
				// Note: Currently plan.Task doesn't have Image field, so this will be empty
				// This is a placeholder for future enhancement when task images are tracked
				taskImages := make(map[string]string)
				// Future: when plan.Task has Image field, populate taskImages here

				infraDrift = drift.DetectInfraDrift(drift.InfraDriftOptions{
					Policy:     pol,
					TaskImages: taskImages,
				})
			}

			progressIndicator.UpdateTask("infra-drift", "completed", nil)
			cpState.SetMetadata("infra_drift_count", fmt.Sprintf("%d", len(infraDrift)))
			checkpointMgr.Save(cpState)
		} else {
			fmt.Println("✓ Infrastructure drift check already completed (skipping)")
		}

		// Get drift results from checkpoint metadata if checks were skipped
		planDrift := drift.DetectPlanDrift(lock, p)
		codeDrift := drift.DetectCodeDrift(s, lock, drift.CodeDriftOptions{
			ProjectRoot: projectRoot,
			APISpecPath: apiSpecPath,
			IgnoreGlobs: ignoreGlobs,
		})

		// Generate report
		progressIndicator.UpdateTask("report-generation", "running", nil)
		checkpointMgr.Save(cpState)

		report := drift.GenerateReport(planDrift, codeDrift, infraDrift)

		progressIndicator.UpdateTask("report-generation", "completed", nil)
		checkpointMgr.Save(cpState)

		// Print summary
		fmt.Printf("\nDrift Detection Summary:\n")
		fmt.Printf("  Total Findings: %d\n", report.Summary.TotalFindings)
		fmt.Printf("  Errors:        %d\n", report.Summary.Errors)
		fmt.Printf("  Warnings:      %d\n", report.Summary.Warnings)
		fmt.Printf("  Info:          %d\n", report.Summary.Info)
		fmt.Println()

		// Print findings
		if len(planDrift) > 0 {
			fmt.Println("Plan Drift:")
			for _, f := range planDrift {
				fmt.Printf("  [%s] %s: %s\n", f.Severity, f.Code, f.Message)
			}
		}

		if len(codeDrift) > 0 {
			fmt.Println("\nCode Drift:")
			for _, f := range codeDrift {
				fmt.Printf("  [%s] %s: %s (feature: %s)\n", f.Severity, f.Code, f.Message, f.FeatureID)
			}
		}

		if len(infraDrift) > 0 {
			fmt.Println("\nInfrastructure Drift:")
			for _, f := range infraDrift {
				fmt.Printf("  [%s] %s: %s\n", f.Severity, f.Code, f.Message)
			}
		}

		// Generate SARIF output
		sarif := report.ToSARIF()
		if err := drift.SaveSARIF(sarif, reportFile); err != nil {
			return fmt.Errorf("failed to save SARIF report: %w", err)
		}
		fmt.Printf("✓ SARIF report saved to %s\n", reportFile)

		// Mark evaluation as completed
		cpState.Status = "completed"
		if err := checkpointMgr.Save(cpState); err != nil {
			fmt.Printf("Warning: failed to save final checkpoint: %v\n", err)
		}

		// Fail if requested and drift detected
		if failOnDrift && report.HasErrors() {
			cpState.Status = "failed"
			checkpointMgr.Save(cpState)
			return fmt.Errorf("drift detection failed with %d errors", report.Summary.Errors)
		}

		if report.IsClean() {
			fmt.Println("✓ No drift detected")
		}

		// Clean up checkpoint on success unless user wants to keep it
		keepCheckpoint, _ := cmd.Flags().GetBool("keep-checkpoint")
		if !keepCheckpoint && cpState.Status == "completed" {
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
	rootCmd.AddCommand(evalCmd)

	evalCmd.Flags().String("plan", "plan.json", "Plan file to evaluate")
	evalCmd.Flags().String("lock", ".specular/spec.lock.json", "SpecLock file")
	evalCmd.Flags().String("spec", ".specular/spec.yaml", "Spec file for code drift detection")
	evalCmd.Flags().String("policy", "", "Policy file for infrastructure drift detection")
	evalCmd.Flags().String("report", "drift.sarif", "Output report file (SARIF format)")
	evalCmd.Flags().Bool("fail-on-drift", false, "Exit with error if drift is detected")
	evalCmd.Flags().String("project-root", ".", "Project root directory")
	evalCmd.Flags().String("api-spec", "", "Path to OpenAPI spec file")
	evalCmd.Flags().StringSlice("ignore", []string{}, "Glob patterns to ignore (e.g., *.test.js)")
	evalCmd.Flags().Bool("resume", false, "Resume from previous checkpoint")
	evalCmd.Flags().String("checkpoint-dir", ".specular/checkpoints", "Directory for checkpoints")
	evalCmd.Flags().String("checkpoint-id", "", "Checkpoint ID (auto-generated if not provided)")
	evalCmd.Flags().Bool("keep-checkpoint", false, "Keep checkpoint after successful completion")
}
