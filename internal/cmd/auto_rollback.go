package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/patch"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback [workflow-id] [step-id]",
	Short: "Rollback changes made by auto mode",
	Long: `Rollback changes made by specular auto mode using saved patches.

Examples:
  # List patches for a workflow
  specular auto rollback auto-1762811730 --list

  # Rollback a single step
  specular auto rollback auto-1762811730 step-2

  # Rollback to a specific step (reverts all steps after it)
  specular auto rollback auto-1762811730 --to step-2

  # Rollback all steps in a workflow
  specular auto rollback auto-1762811730 --all

  # Verify rollback safety before applying
  specular auto rollback auto-1762811730 step-2 --dry-run`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowID := args[0]
		stepID := ""
		if len(args) > 1 {
			stepID = args[1]
		}

		// Parse flags
		listPatches, _ := cmd.Flags().GetBool("list")
		rollbackAll, _ := cmd.Flags().GetBool("all")
		rollbackTo, _ := cmd.Flags().GetString("to")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Get working directory and patch directory
		workingDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		homeDir, _ := os.UserHomeDir()
		patchDir := filepath.Join(homeDir, ".specular", "patches")

		rollback := patch.NewRollback(workingDir, patchDir)

		// Handle --list
		if listPatches {
			return listPatchesForWorkflow(workflowID, patchDir)
		}

		// Handle --all
		if rollbackAll {
			return rollbackAllSteps(rollback, workflowID, dryRun)
		}

		// Handle --to
		if rollbackTo != "" {
			return rollbackToStep(rollback, workflowID, rollbackTo, dryRun)
		}

		// Handle single step rollback
		if stepID == "" {
			return fmt.Errorf("step-id required (or use --list, --all, or --to)")
		}

		return rollbackSingleStep(rollback, workflowID, stepID, dryRun)
	},
}

func init() {
	rollbackCmd.Flags().Bool("list", false, "List available patches for the workflow")
	rollbackCmd.Flags().Bool("all", false, "Rollback all steps in the workflow")
	rollbackCmd.Flags().String("to", "", "Rollback to a specific step (reverts all steps after it)")
	rollbackCmd.Flags().Bool("dry-run", false, "Verify rollback safety without applying changes")

	autoCmd.AddCommand(rollbackCmd)
}

// listPatchesForWorkflow lists all patches for a workflow
func listPatchesForWorkflow(workflowID, patchDir string) error {
	writer := patch.NewWriter(patchDir)
	patches, err := writer.ListPatches(workflowID)
	if err != nil {
		return fmt.Errorf("failed to list patches: %w", err)
	}

	if len(patches) == 0 {
		fmt.Printf("No patches found for workflow %s\n", workflowID)
		fmt.Printf("Patches are saved when using --save-patches flag\n")
		return nil
	}

	fmt.Printf("📋 Patches for workflow %s:\n\n", workflowID)
	for _, p := range patches {
		fmt.Printf("  %s (%s)\n", p.StepID, p.StepType)
		fmt.Printf("    %s\n", p.Description)
		fmt.Printf("    Files: %d, Changes: +%d -%d\n", p.FilesChanged, p.Insertions, p.Deletions)
		fmt.Printf("    Created: %s\n\n", p.Timestamp.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("Total: %d patches\n", len(patches))
	fmt.Println("\nUse 'specular auto rollback <workflow-id> <step-id>' to rollback a specific step")
	return nil
}

// rollbackSingleStep rolls back a single step
func rollbackSingleStep(rollback *patch.Rollback, workflowID, stepID string, dryRun bool) error {
	fmt.Printf("🔄 Rolling back step: %s\n", stepID)

	// Verify safety
	safe, warnings, err := rollback.VerifyRollbackSafety(workflowID, stepID)
	if err != nil {
		return fmt.Errorf("failed to verify rollback safety: %w", err)
	}

	if len(warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, warning := range warnings {
			fmt.Printf("   - %s\n", warning)
		}
		fmt.Println()
	}

	if !safe {
		fmt.Println("⚠️  Rollback may not be safe due to conflicts")
		fmt.Println("Use --dry-run to see details without applying changes")
		if !dryRun {
			fmt.Print("\nContinue anyway? [y/N]: ")
			var response string
			if _, err := fmt.Scanln(&response); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
				return err
			}
			if response != "y" && response != "Y" {
				fmt.Println("Rollback cancelled")
				return nil
			}
		}
	}

	if dryRun {
		fmt.Println("✅ Dry-run complete. Use without --dry-run to apply rollback")
		return nil
	}

	// Apply rollback
	if err := rollback.RollbackStep(workflowID, stepID); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Printf("✅ Successfully rolled back step: %s\n", stepID)
	return nil
}

// rollbackToStep rolls back all steps after the target step
func rollbackToStep(rollback *patch.Rollback, workflowID, targetStepID string, dryRun bool) error {
	fmt.Printf("🔄 Rolling back to step: %s\n", targetStepID)
	fmt.Println("   (This will revert all steps after this one)")

	if dryRun {
		fmt.Println("\n✅ Dry-run mode: no changes will be applied")
	}

	result, err := rollback.RollbackToStep(workflowID, targetStepID)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	// Print results
	fmt.Printf("\n📊 Rollback Summary:\n")
	fmt.Printf("   Steps reverted: %d\n", result.StepsReverted)

	if len(result.Errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for _, errMsg := range result.Errors {
			fmt.Printf("   - %s\n", errMsg)
		}
	}

	if len(result.Conflicts) > 0 {
		fmt.Println("\n⚠️  Conflicts:")
		for _, conflict := range result.Conflicts {
			fmt.Printf("   - %s\n", conflict)
		}
	}

	if result.Success {
		fmt.Println("\n✅ Rollback completed successfully")
	} else {
		fmt.Println("\n⚠️  Rollback completed with errors")
	}

	return nil
}

// rollbackAllSteps rolls back all steps in the workflow
func rollbackAllSteps(rollback *patch.Rollback, workflowID string, dryRun bool) error {
	fmt.Printf("🔄 Rolling back all steps for workflow: %s\n", workflowID)
	fmt.Println("   ⚠️  This will revert all changes made by this workflow")

	if !dryRun {
		fmt.Print("\nAre you sure? [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			return err
		}
		if response != "y" && response != "Y" {
			fmt.Println("Rollback cancelled")
			return nil
		}
	}

	result, err := rollback.RollbackAll(workflowID)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	// Print results
	fmt.Printf("\n📊 Rollback Summary:\n")
	fmt.Printf("   Steps reverted: %d\n", result.StepsReverted)

	if len(result.Errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for _, errMsg := range result.Errors {
			fmt.Printf("   - %s\n", errMsg)
		}
	}

	if len(result.Conflicts) > 0 {
		fmt.Println("\n⚠️  Conflicts:")
		for _, conflict := range result.Conflicts {
			fmt.Printf("   - %s\n", conflict)
		}
	}

	if result.Success {
		fmt.Println("\n✅ Rollback completed successfully")
	} else {
		fmt.Println("\n⚠️  Rollback completed with errors")
	}

	return nil
}
