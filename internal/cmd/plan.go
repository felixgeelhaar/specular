package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Build execution plan from spec",
	Long: `Generate a task DAG (Directed Acyclic Graph) from a specification.
The plan includes task dependencies, priorities, skill requirements, and
expected hashes for drift detection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		defaults := ux.NewPathDefaults()
		specPath := cmd.Flags().Lookup("in").Value.String()
		lockPath := cmd.Flags().Lookup("lock").Value.String()
		out := cmd.Flags().Lookup("out").Value.String()
		estimate := cmd.Flags().Lookup("estimate").Value.String() == "true"

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
		if err := ux.ValidateRequiredFile(specPath, "Spec file", "specular spec generate"); err != nil {
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

		p, err := plan.Generate(s, opts)
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

		return nil
	},
}

func init() {
	rootCmd.AddCommand(planCmd)

	planCmd.Flags().StringP("in", "i", ".specular/spec.yaml", "Input spec file")
	planCmd.Flags().String("lock", ".specular/spec.lock.json", "Input SpecLock file")
	planCmd.Flags().StringP("out", "o", "plan.json", "Output plan file")
	planCmd.Flags().Bool("estimate", true, "Estimate task complexity")
}
