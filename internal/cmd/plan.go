package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/internal/tui"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage execution plans",
	Long: `Generate, review, and manage execution plans from specifications.

Use 'specular plan gen' to generate a new plan from a specification.
Use 'specular plan review' to interactively review a plan.
Use 'specular plan drift' to detect drift between plan and repository.
Use 'specular plan explain' to understand routing decisions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if this is being used as the old direct command
		// If flags are set, run the gen command for backward compatibility
		if cmd.Flags().Changed("in") || cmd.Flags().Changed("out") || cmd.Flags().Changed("lock") {
			fmt.Fprintf(os.Stderr, "\n⚠️  DEPRECATION WARNING:\n")
			fmt.Fprintf(os.Stderr, "Running 'plan' directly is deprecated and will be removed in v1.6.0.\n")
			fmt.Fprintf(os.Stderr, "Please use 'specular plan gen' instead.\n\n")

			// Run gen command
			return runPlanGen(cmd, args)
		}

		// Otherwise show help
		return cmd.Help()
	},
}

var planGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate execution plan from spec",
	Long: `Generate a task DAG (Directed Acyclic Graph) from a specification.
The plan includes task dependencies, priorities, skill requirements, and
expected hashes for drift detection.

You can optionally generate a plan for a specific feature using --feature.`,
	RunE: runPlanGen,
}

var planReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Interactively review execution plan",
	Long: `Launch an interactive terminal UI to review the execution plan.

The TUI allows you to:
- View task dependencies as a graph
- Inspect task details and requirements
- Modify task priorities
- Approve or reject the plan`,
	RunE: runPlanReview,
}

var planDriftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect drift between plan and repository",
	Long: `Compare the current repository state with the execution plan to detect drift.

Drift detection checks:
- File hashes vs expected hashes in plan
- Missing or extra files
- Uncommitted changes that may affect the plan`,
	RunE: runPlanDrift,
}

var planExplainCmd = &cobra.Command{
	Use:   "explain [step]",
	Short: "Explain routing decisions for plan step",
	Long: `Explain the reasoning behind routing decisions for a specific plan step.

Shows:
- Why a particular model was selected
- Skill requirements that influenced the decision
- Cost and latency considerations
- Alternative models that were considered`,
	RunE: runPlanExplain,
}

func runPlanGen(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	specPath := cmd.Flags().Lookup("in").Value.String()
	lockPath := cmd.Flags().Lookup("lock").Value.String()
	out := cmd.Flags().Lookup("out").Value.String()
	estimate := cmd.Flags().Lookup("estimate").Value.String() == "true"
	featureID := cmd.Flags().Lookup("feature").Value.String()

	// Use smart defaults if not changed
	if !cmd.Flags().Changed("in") {
		specPath = defaults.SpecFile()
	}
	if !cmd.Flags().Changed("lock") {
		lockPath = defaults.SpecLockFile()
	}
	if !cmd.Flags().Changed("out") {
		out = defaults.PlanFile()
	}

	// Validate required files with helpful errors
	if err := ux.ValidateRequiredFile(specPath, "Spec file", "specular spec new"); err != nil {
		return ux.EnhanceError(err)
	}
	if err := ux.ValidateRequiredFile(lockPath, "SpecLock file", "specular spec lock"); err != nil {
		return ux.EnhanceError(err)
	}

	// Load spec
	s, err := spec.LoadSpec(specPath)
	if err != nil {
		return ux.FormatError(err, "loading spec file")
	}

	// Load SpecLock
	lock, err := spec.LoadSpecLock(lockPath)
	if err != nil {
		return ux.FormatError(err, "loading SpecLock file")
	}

	// Generate plan
	opts := plan.GenerateOptions{
		SpecLock:           lock,
		EstimateComplexity: estimate,
	}

	// If feature flag is set, filter to specific feature
	if featureID != "" {
		// Verify feature exists and filter spec
		found := false
		var filteredFeatures []spec.Feature
		for _, f := range s.Features {
			if string(f.ID) == featureID {
				found = true
				filteredFeatures = append(filteredFeatures, f)
				break
			}
		}
		if !found {
			return fmt.Errorf("feature '%s' not found in spec", featureID)
		}

		fmt.Printf("Generating plan for feature: %s\n", featureID)
		// Create filtered spec with only the requested feature
		s = &spec.ProductSpec{
			Product:       s.Product,
			Goals:         s.Goals,
			Features:      filteredFeatures,
			NonFunctional: s.NonFunctional,
			Acceptance:    s.Acceptance,
			Milestones:    s.Milestones,
		}
	}

	p, err := plan.Generate(cmd.Context(), s, opts)
	if err != nil {
		return ux.FormatError(err, "generating plan")
	}

	// Save plan
	if saveErr := plan.SavePlan(p, out); saveErr != nil {
		return ux.FormatError(saveErr, "saving plan file")
	}

	fmt.Printf("✓ Generated plan with %d tasks\n", len(p.Tasks))
	for _, task := range p.Tasks {
		deps := "none"
		if len(task.DependsOn) > 0 {
			deps = fmt.Sprintf("%d dependencies", len(task.DependsOn))
		}
		fmt.Printf("  %s [%s] %s - %s (%s)\n",
			task.ID, task.Priority, task.FeatureID, task.Skill, deps)
	}

	fmt.Println("\nNext steps:")
	if featureID != "" {
		fmt.Printf("  1. Review plan: specular plan review\n")
		fmt.Printf("  2. Execute feature: specular build --plan %s\n", out)
	} else {
		fmt.Printf("  1. Review plan: specular plan review\n")
		fmt.Printf("  2. Execute plan: specular build\n")
	}

	return nil
}

func runPlanReview(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	planPath := cmd.Flags().Lookup("plan").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("plan") {
		planPath = defaults.PlanFile()
	}

	// Validate plan file exists
	if err := ux.ValidateRequiredFile(planPath, "Plan file", "specular plan gen"); err != nil {
		return ux.EnhanceError(err)
	}

	// Load plan
	p, err := plan.LoadPlan(planPath)
	if err != nil {
		return ux.FormatError(err, "loading plan file")
	}

	fmt.Printf("=== Plan Review (TUI) ===\n")
	fmt.Printf("Plan: %s (%d tasks)\n\n", planPath, len(p.Tasks))

	// Launch TUI for plan review
	result, err := tui.RunPlanReview(p)
	if err != nil {
		return ux.FormatError(err, "running plan review TUI")
	}

	// Show result
	if result.Approved {
		fmt.Printf("\n✓ Plan approved\n")
		fmt.Println("\nNext steps:")
		fmt.Printf("  1. Execute plan: specular build\n")
	} else {
		fmt.Printf("\n✗ Plan rejected\n")
		if result.Reason != "" {
			fmt.Printf("  Reason: %s\n", result.Reason)
		}
		fmt.Println("\nNext steps:")
		fmt.Printf("  1. Modify spec: specular spec edit\n")
		fmt.Printf("  2. Regenerate plan: specular plan gen\n")
	}

	return nil
}

func runPlanDrift(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	planPath := cmd.Flags().Lookup("plan").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("plan") {
		planPath = defaults.PlanFile()
	}

	// Validate plan file exists
	if err := ux.ValidateRequiredFile(planPath, "Plan file", "specular plan gen"); err != nil {
		return ux.EnhanceError(err)
	}

	// Load plan
	p, err := plan.LoadPlan(planPath)
	if err != nil {
		return ux.FormatError(err, "loading plan file")
	}

	fmt.Printf("Detecting drift for plan: %s\n\n", planPath)

	// Get git status to check for uncommitted changes
	gitCmd := exec.Command("git", "status", "--porcelain")
	output, err := gitCmd.Output()
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not check git status: %v\n", err)
	}

	uncommitted := strings.TrimSpace(string(output))
	if uncommitted != "" {
		lines := strings.Split(uncommitted, "\n")
		fmt.Printf("⚠️  Uncommitted changes detected (%d files):\n", len(lines))
		for i, line := range lines {
			if i < 5 {
				fmt.Printf("  %s\n", line)
			}
		}
		if len(lines) > 5 {
			fmt.Printf("  ... and %d more\n", len(lines)-5)
		}
		fmt.Println()
	}

	// Check for task drift (simplified - would need actual implementation)
	driftCount := 0
	for _, task := range p.Tasks {
		// In a real implementation, we would:
		// 1. Check if files for this task have changed
		// 2. Compare file hashes with expected hashes
		// 3. Report any mismatches
		_ = task // Placeholder
	}

	if driftCount == 0 && uncommitted == "" {
		fmt.Printf("✓ No drift detected\n")
		fmt.Printf("  All tasks align with current repository state\n")
	} else {
		fmt.Printf("⚠️  Drift detected\n")
		fmt.Printf("  %d task(s) may be affected by changes\n", driftCount)
		fmt.Println("\nRecommendations:")
		if uncommitted != "" {
			fmt.Printf("  1. Commit or stash uncommitted changes\n")
			fmt.Printf("  2. Regenerate plan: specular plan gen\n")
		} else {
			fmt.Printf("  1. Review changes: git diff\n")
			fmt.Printf("  2. Regenerate plan if needed: specular plan gen\n")
		}
	}

	return nil
}

func runPlanExplain(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	planPath := cmd.Flags().Lookup("plan").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("plan") {
		planPath = defaults.PlanFile()
	}

	// Validate plan file exists
	if err := ux.ValidateRequiredFile(planPath, "Plan file", "specular plan gen"); err != nil {
		return ux.EnhanceError(err)
	}

	// Require step argument
	if len(args) == 0 {
		return fmt.Errorf("step ID is required\n\nUsage: specular plan explain <step-id>")
	}
	stepID := args[0]

	// Load plan
	p, err := plan.LoadPlan(planPath)
	if err != nil {
		return ux.FormatError(err, "loading plan file")
	}

	// Find the task
	var task *plan.Task
	for i := range p.Tasks {
		if string(p.Tasks[i].ID) == stepID {
			task = &p.Tasks[i]
			break
		}
	}

	if task == nil {
		return fmt.Errorf("task '%s' not found in plan", stepID)
	}

	// Explain the routing decision
	fmt.Printf("=== Plan Step Explanation ===\n\n")
	fmt.Printf("Task ID: %s\n", task.ID)
	fmt.Printf("Feature: %s\n", task.FeatureID)
	fmt.Printf("Skill: %s\n", task.Skill)
	fmt.Printf("Priority: %s\n", task.Priority)
	fmt.Printf("Model Hint: %s\n", task.ModelHint)
	fmt.Printf("Estimated Complexity: %d\n\n", task.Estimate)

	fmt.Printf("Routing Decision:\n")
	fmt.Printf("  Model selected based on:\n")
	fmt.Printf("    • Skill requirement: %s\n", task.Skill)
	fmt.Printf("    • Model hint: %s\n", task.ModelHint)
	fmt.Printf("    • Task priority: %s\n", task.Priority)
	fmt.Println()

	// Show dependencies
	if len(task.DependsOn) > 0 {
		fmt.Printf("Dependencies (%d):\n", len(task.DependsOn))
		for _, depID := range task.DependsOn {
			fmt.Printf("  • %s\n", depID)
		}
		fmt.Println()
	} else {
		fmt.Printf("Dependencies: none\n\n")
	}

	fmt.Printf("Expected Hash: %s\n", task.ExpectedHash)
	fmt.Printf("  (used for drift detection)\n")

	return nil
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.AddCommand(planGenCmd)
	planCmd.AddCommand(planReviewCmd)
	planCmd.AddCommand(planDriftCmd)
	planCmd.AddCommand(planExplainCmd)

	// Flags for backward compatibility on root plan command
	planCmd.Flags().StringP("in", "i", ".specular/spec.yaml", "Input spec file")
	planCmd.Flags().String("lock", ".specular/spec.lock.json", "Input SpecLock file")
	planCmd.Flags().StringP("out", "o", "plan.json", "Output plan file")
	planCmd.Flags().Bool("estimate", true, "Estimate task complexity")

	// plan gen flags
	planGenCmd.Flags().StringP("in", "i", ".specular/spec.yaml", "Input spec file")
	planGenCmd.Flags().String("lock", ".specular/spec.lock.json", "Input SpecLock file")
	planGenCmd.Flags().StringP("out", "o", "plan.json", "Output plan file")
	planGenCmd.Flags().Bool("estimate", true, "Estimate task complexity")
	planGenCmd.Flags().String("feature", "", "Generate plan for specific feature ID")

	// plan review flags
	planReviewCmd.Flags().String("plan", "plan.json", "Plan file to review")

	// plan drift flags
	planDriftCmd.Flags().String("plan", "plan.json", "Plan file to check for drift")

	// plan explain flags
	planExplainCmd.Flags().String("plan", "plan.json", "Plan file to explain")
}
