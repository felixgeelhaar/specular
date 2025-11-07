package cmd

import (
	"fmt"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Build execution plan from spec",
	Long: `Generate a task DAG (Directed Acyclic Graph) from a specification.
The plan includes task dependencies, priorities, skill requirements, and
expected hashes for drift detection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		specPath, _ := cmd.Flags().GetString("in")
		lockPath, _ := cmd.Flags().GetString("lock")
		out, _ := cmd.Flags().GetString("out")
		estimate, _ := cmd.Flags().GetBool("estimate")

		// Load spec
		s, err := spec.LoadSpec(specPath)
		if err != nil {
			return fmt.Errorf("failed to load spec: %w", err)
		}

		// Load SpecLock
		lock, err := spec.LoadSpecLock(lockPath)
		if err != nil {
			return fmt.Errorf("failed to load SpecLock: %w (run 'ai-dev spec lock' first)", err)
		}

		// Generate plan
		opts := plan.GenerateOptions{
			SpecLock:           lock,
			EstimateComplexity: estimate,
		}

		p, err := plan.Generate(s, opts)
		if err != nil {
			return fmt.Errorf("failed to generate plan: %w", err)
		}

		// Save plan
		if err := plan.SavePlan(p, out); err != nil {
			return fmt.Errorf("failed to save plan: %w", err)
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
