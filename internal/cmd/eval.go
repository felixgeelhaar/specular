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
		planFile := cmd.Flags().Lookup("plan").Value.String()
		lockFile := cmd.Flags().Lookup("lock").Value.String()
		specFile := cmd.Flags().Lookup("spec").Value.String()
		policyFile := cmd.Flags().Lookup("policy").Value.String()
		reportFile := cmd.Flags().Lookup("report").Value.String()
		failOnDrift := cmd.Flags().Lookup("fail-on-drift").Value.String() == "true"
		projectRoot := cmd.Flags().Lookup("project-root").Value.String()
		apiSpecPath := cmd.Flags().Lookup("api-spec").Value.String()
		ignoreGlobs, _ := cmd.Flags().GetStringSlice("ignore") //nolint:errcheck // Acceptable to ignore array return
		resume := cmd.Flags().Lookup("resume").Value.String() == "true"
		checkpointDir := cmd.Flags().Lookup("checkpoint-dir").Value.String()
		checkpointID := cmd.Flags().Lookup("checkpoint-id").Value.String()

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
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}

			polQualityGate, err := policy.LoadPolicy(policyFile)
			if err != nil {
				progressIndicator.UpdateTask("quality-gate", "failed", err)
				if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
					fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
				}
				return fmt.Errorf("failed to load policy: %w", err)
			}

			fmt.Println("Running quality gate checks...")
			gateReport, err := eval.RunEvalGate(eval.GateOptions{
				Policy:      polQualityGate,
				ProjectRoot: projectRoot,
				Verbose:     false,
			})
			if err != nil {
				progressIndicator.UpdateTask("quality-gate", "failed", err)
				if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
					fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
				}
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
				if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
					fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
				}
				return fmt.Errorf("quality gate failed with %d failed checks", gateReport.TotalFailed)
			}

			progressIndicator.UpdateTask("quality-gate", "completed", nil)
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}
		} else if policyFile == "" {
			progressIndicator.UpdateTask("quality-gate", "skipped", nil)
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}
		} else {
			fmt.Println("✓ Quality gate check already completed (skipping)")
		}

		// Detect plan drift
		if cpState.Tasks["plan-drift"].Status != "completed" {
			progressIndicator.UpdateTask("plan-drift", "running", nil)
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}

			fmt.Println("Detecting plan drift...")
			planDrift := drift.DetectPlanDrift(lock, p)

			progressIndicator.UpdateTask("plan-drift", "completed", nil)
			cpState.SetMetadata("plan_drift_count", fmt.Sprintf("%d", len(planDrift)))
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}
		} else {
			fmt.Println("✓ Plan drift check already completed (skipping)")
		}

		// Detect code drift
		if cpState.Tasks["code-drift"].Status != "completed" {
			progressIndicator.UpdateTask("code-drift", "running", nil)
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}

			fmt.Println("Detecting code drift...")
			codeDrift := drift.DetectCodeDrift(s, lock, drift.CodeDriftOptions{
				ProjectRoot: projectRoot,
				APISpecPath: apiSpecPath,
				IgnoreGlobs: ignoreGlobs,
			})

			progressIndicator.UpdateTask("code-drift", "completed", nil)
			cpState.SetMetadata("code_drift_count", fmt.Sprintf("%d", len(codeDrift)))
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}
		} else {
			fmt.Println("✓ Code drift check already completed (skipping)")
		}

		// Detect infrastructure drift
		var infraDrift []drift.Finding
		if cpState.Tasks["infra-drift"].Status != "completed" {
			progressIndicator.UpdateTask("infra-drift", "running", nil)
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}

			fmt.Println("Detecting infrastructure drift...")
			if policyFile != "" {
				polInfra, err := policy.LoadPolicy(policyFile)
				if err != nil {
					progressIndicator.UpdateTask("infra-drift", "failed", err)
					if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
						fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
					}
					return fmt.Errorf("failed to load policy: %w", err)
				}

				// Build task images map from plan
				// Note: Currently plan.Task doesn't have Image field, so this will be empty
				// This is a placeholder for future enhancement when task images are tracked
				taskImages := make(map[string]string)
				// Future: when plan.Task has Image field, populate taskImages here

				infraDrift = drift.DetectInfraDrift(drift.InfraDriftOptions{
					Policy:     polInfra,
					TaskImages: taskImages,
				})
			}

			progressIndicator.UpdateTask("infra-drift", "completed", nil)
			cpState.SetMetadata("infra_drift_count", fmt.Sprintf("%d", len(infraDrift)))
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}
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
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}

		report := drift.GenerateReport(planDrift, codeDrift, infraDrift)

		progressIndicator.UpdateTask("report-generation", "completed", nil)
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}

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
		if errSARIF := drift.SaveSARIF(sarif, reportFile); errSARIF != nil {
			return fmt.Errorf("failed to save SARIF report: %w", errSARIF)
		}
		fmt.Printf("✓ SARIF report saved to %s\n", reportFile)

		// Mark evaluation as completed
		cpState.Status = "completed"
		if errCP := checkpointMgr.Save(cpState); errCP != nil {
			fmt.Printf("Warning: failed to save final checkpoint: %v\n", errCP)
		}

		// Fail if requested and drift detected
		if failOnDrift && report.HasErrors() {
			cpState.Status = "failed"
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}
			return fmt.Errorf("drift detection failed with %d errors", report.Summary.Errors)
		}

		if report.IsClean() {
			fmt.Println("✓ No drift detected")
		}

		// Clean up checkpoint on success unless user wants to keep it
		keepCheckpoint := cmd.Flags().Lookup("keep-checkpoint").Value.String() == "true"
		if !keepCheckpoint && cpState.Status == "completed" {
			if errDel := checkpointMgr.Delete(checkpointID); errDel != nil {
				fmt.Printf("Warning: failed to delete checkpoint: %v\n", errDel)
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
