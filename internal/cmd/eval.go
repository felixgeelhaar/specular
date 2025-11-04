package cmd

import (
	"fmt"

	"github.com/felixgeelhaar/ai-dev/internal/drift"
	"github.com/felixgeelhaar/ai-dev/internal/plan"
	"github.com/felixgeelhaar/ai-dev/internal/spec"
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
		reportFile, _ := cmd.Flags().GetString("report")
		failOnDrift, _ := cmd.Flags().GetBool("fail-on-drift")
		projectRoot, _ := cmd.Flags().GetString("project-root")
		apiSpecPath, _ := cmd.Flags().GetString("api-spec")
		ignoreGlobs, _ := cmd.Flags().GetStringSlice("ignore")

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

		// Detect plan drift
		fmt.Println("Detecting plan drift...")
		planDrift := drift.DetectPlanDrift(lock, p)

		// Detect code drift
		fmt.Println("Detecting code drift...")
		codeDrift := drift.DetectCodeDrift(s, lock, drift.CodeDriftOptions{
			ProjectRoot: projectRoot,
			APISpecPath: apiSpecPath,
			IgnoreGlobs: ignoreGlobs,
		})

		// TODO: Infrastructure drift detection (requires implementation)
		var infraDrift []drift.Finding

		// Generate report
		report := drift.GenerateReport(planDrift, codeDrift, infraDrift)

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

		// Generate SARIF output
		sarif := report.ToSARIF()
		if err := drift.SaveSARIF(sarif, reportFile); err != nil {
			return fmt.Errorf("failed to save SARIF report: %w", err)
		}
		fmt.Printf("✓ SARIF report saved to %s\n", reportFile)

		// Fail if requested and drift detected
		if failOnDrift && report.HasErrors() {
			return fmt.Errorf("drift detection failed with %d errors", report.Summary.Errors)
		}

		if report.IsClean() {
			fmt.Println("✓ No drift detected")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(evalCmd)

	evalCmd.Flags().String("plan", "plan.json", "Plan file to evaluate")
	evalCmd.Flags().String("lock", ".aidv/spec.lock.json", "SpecLock file")
	evalCmd.Flags().String("spec", ".aidv/spec.yaml", "Spec file for code drift detection")
	evalCmd.Flags().String("report", "drift.sarif", "Output report file (SARIF format)")
	evalCmd.Flags().Bool("fail-on-drift", false, "Exit with error if drift is detected")
	evalCmd.Flags().String("project-root", ".", "Project root directory")
	evalCmd.Flags().String("api-spec", "", "Path to OpenAPI spec file")
	evalCmd.Flags().StringSlice("ignore", []string{}, "Glob patterns to ignore (e.g., *.test.js)")
}
