package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
)

var checkpointCmd = &cobra.Command{
	Use:   "checkpoint",
	Short: "Manage workflow checkpoints for resumable execution",
	Long: `Manage workflow checkpoints for resumable execution.

Checkpoints are created automatically during autonomous mode execution and can be
used to resume interrupted or failed workflows.

Commands:
  list     List all available checkpoints
  show     Show detailed information about a checkpoint

To resume a checkpoint, use: specular auto --resume <checkpoint-id>

Examples:
  specular checkpoint list
  specular checkpoint show auto-1762811730
  specular auto --resume auto-1762811730`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var checkpointListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available checkpoints",
	Long: `List all checkpoints saved in .specular/checkpoints directory.

Shows checkpoint ID, status, created time, and task completion.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create checkpoint manager
		checkpointMgr := checkpoint.NewManager(".specular/checkpoints", false, 0)

		// List all checkpoints
		checkpointIDs, err := checkpointMgr.List()
		if err != nil {
			return fmt.Errorf("failed to list checkpoints: %w", err)
		}

		if len(checkpointIDs) == 0 {
			fmt.Println("No checkpoints found.")
			return nil
		}

		// Load and display each checkpoint
		type checkpointInfo struct {
			ID          string
			Status      string
			StartedAt   time.Time
			Product     string
			Goal        string
			Completed   int
			Total       int
			FailedTasks int
		}

		var checkpoints []checkpointInfo

		for _, id := range checkpointIDs {
			cpState, err := checkpointMgr.Load(id)
			if err != nil {
				continue // Skip invalid checkpoints
			}

			product, _ := cpState.GetMetadata("product")
			goal, _ := cpState.GetMetadata("goal")

			completed := len(cpState.GetCompletedTasks())
			failed := len(cpState.GetFailedTasks())
			total := len(cpState.Tasks)

			checkpoints = append(checkpoints, checkpointInfo{
				ID:          id,
				Status:      cpState.Status,
				StartedAt:   cpState.StartedAt,
				Product:     product,
				Goal:        goal,
				Completed:   completed,
				Total:       total,
				FailedTasks: failed,
			})
		}

		// Sort by started time (newest first)
		sort.Slice(checkpoints, func(i, j int) bool {
			return checkpoints[i].StartedAt.After(checkpoints[j].StartedAt)
		})

		// Print header
		fmt.Println("Checkpoints:")
		fmt.Println()

		// Print checkpoints
		for _, cp := range checkpoints {
			statusIcon := "📦"
			switch cp.Status {
			case "completed":
				statusIcon = "✅"
			case "failed":
				statusIcon = "✗"
			case "running":
				statusIcon = "⏳"
			}

			fmt.Printf("%s %s\n", statusIcon, cp.ID)
			fmt.Printf("   Status:   %s\n", cp.Status)
			fmt.Printf("   Product:  %s\n", cp.Product)
			fmt.Printf("   Started:  %s\n", cp.StartedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("   Progress: %d/%d tasks", cp.Completed, cp.Total)
			if cp.FailedTasks > 0 {
				fmt.Printf(" (%d failed)", cp.FailedTasks)
			}
			fmt.Println()
			fmt.Println()
		}

		return nil
	},
}

var checkpointShowCmd = &cobra.Command{
	Use:   "show <checkpoint-id>",
	Short: "Show detailed information about a checkpoint",
	Long: `Show detailed information about a specific checkpoint.

Displays checkpoint metadata, task status, and execution details.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		checkpointID := args[0]

		// Create checkpoint manager
		checkpointMgr := checkpoint.NewManager(".specular/checkpoints", false, 0)

		// Load checkpoint
		cpState, err := checkpointMgr.Load(checkpointID)
		if err != nil {
			return fmt.Errorf("failed to load checkpoint: %w", err)
		}

		// Get metadata
		product, _ := cpState.GetMetadata("product")
		goal, _ := cpState.GetMetadata("goal")

		// Get task lists
		completed := cpState.GetCompletedTasks()
		pending := cpState.GetPendingTasks()
		failed := cpState.GetFailedTasks()

		// Print checkpoint info
		fmt.Printf("Checkpoint: %s\n\n", checkpointID)
		fmt.Printf("Status:     %s\n", cpState.Status)
		fmt.Printf("Product:    %s\n", product)
		fmt.Printf("Goal:       %s\n", goal)
		fmt.Printf("Started:    %s\n", cpState.StartedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:    %s\n", cpState.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Duration:   %s\n", cpState.UpdatedAt.Sub(cpState.StartedAt).Round(time.Second))
		fmt.Println()

		// Print task summary
		fmt.Printf("Tasks:\n")
		fmt.Printf("  ✓ Completed: %d\n", len(completed))
		fmt.Printf("  ⏳ Pending:   %d\n", len(pending))
		if len(failed) > 0 {
			fmt.Printf("  ✗ Failed:    %d\n", len(failed))
		}
		fmt.Println()

		// Show task details in verbose mode
		verbose, _ := cmd.Flags().GetBool("verbose")
		if verbose {
			// Show spec and plan if available
			if specJSON, ok := cpState.GetMetadata("spec_json"); ok && specJSON != "" {
				fmt.Println("📄 Spec available (use --json to view)")
			}
			if planJSON, ok := cpState.GetMetadata("plan_json"); ok && planJSON != "" {
				fmt.Println("📋 Plan available (use --json to view)")
			}
			fmt.Println()

			// Show task details
			if len(cpState.Tasks) > 0 {
				fmt.Println("Task Details:")
				for id, task := range cpState.Tasks {
					statusIcon := "⏳"
					switch task.Status {
					case "completed":
						statusIcon = "✓"
					case "failed":
						statusIcon = "✗"
					case "pending":
						statusIcon = "○"
					}
					fmt.Printf("  %s %s (%s)\n", statusIcon, id, task.Status)
					if task.Error != "" {
						fmt.Printf("      Error: %s\n", task.Error)
					}
				}
			}
		}

		// Show JSON output if requested
		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			jsonData, err := json.MarshalIndent(cpState, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal checkpoint: %w", err)
			}
			fmt.Println()
			fmt.Println("JSON:")
			fmt.Println(string(jsonData))
		}

		return nil
	},
}

func init() {
	// Add show command flags
	checkpointShowCmd.Flags().BoolP("verbose", "v", false, "Show detailed task information")
	checkpointShowCmd.Flags().Bool("json", false, "Output checkpoint as JSON")

	// Add subcommands
	checkpointCmd.AddCommand(checkpointListCmd)
	checkpointCmd.AddCommand(checkpointShowCmd)

	// Add checkpoint command to root
	rootCmd.AddCommand(checkpointCmd)
}
