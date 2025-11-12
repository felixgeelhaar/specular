package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage workflow sessions and checkpoints",
	Long: `Manage workflow sessions and checkpoints for resumable execution.

Sessions are created automatically during autonomous mode execution and can be
used to resume interrupted or failed workflows.

Commands:
  list     List all available sessions
  show     Show detailed information about a session

To resume a session, use: specular auto --resume <session-id>

Examples:
  specular session list
  specular session show auto-1762811730
  specular auto --resume auto-1762811730`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available sessions",
	Long: `List all sessions saved in .specular/checkpoints directory.

Shows session ID, status, created time, and task completion.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create checkpoint manager
		checkpointMgr := checkpoint.NewManager(".specular/checkpoints", false, 0)

		// List all checkpoints
		checkpointIDs, err := checkpointMgr.List()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		if len(checkpointIDs) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}

		// Load and display each checkpoint
		type sessionInfo struct {
			ID          string
			Status      string
			StartedAt   time.Time
			Product     string
			Goal        string
			Completed   int
			Total       int
			FailedTasks int
		}

		var sessions []sessionInfo

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

			sessions = append(sessions, sessionInfo{
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
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].StartedAt.After(sessions[j].StartedAt)
		})

		// Print header
		fmt.Println("Sessions:")
		fmt.Println()

		// Print sessions
		for _, s := range sessions {
			statusIcon := "📦"
			switch s.Status {
			case "completed":
				statusIcon = "✅"
			case "failed":
				statusIcon = "✗"
			case "running":
				statusIcon = "⏳"
			}

			fmt.Printf("%s %s\n", statusIcon, s.ID)
			fmt.Printf("   Status:   %s\n", s.Status)
			fmt.Printf("   Product:  %s\n", s.Product)
			fmt.Printf("   Started:  %s\n", s.StartedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("   Progress: %d/%d tasks", s.Completed, s.Total)
			if s.FailedTasks > 0 {
				fmt.Printf(" (%d failed)", s.FailedTasks)
			}
			fmt.Println()
			fmt.Println()
		}

		return nil
	},
}

var sessionShowCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Show detailed information about a session",
	Long: `Show detailed information about a specific session.

Displays session metadata, task status, and execution details.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]

		// Create checkpoint manager
		checkpointMgr := checkpoint.NewManager(".specular/checkpoints", false, 0)

		// Load checkpoint
		cpState, err := checkpointMgr.Load(sessionID)
		if err != nil {
			return fmt.Errorf("failed to load session: %w", err)
		}

		// Get metadata
		product, _ := cpState.GetMetadata("product")
		goal, _ := cpState.GetMetadata("goal")

		// Get task lists
		completed := cpState.GetCompletedTasks()
		pending := cpState.GetPendingTasks()
		failed := cpState.GetFailedTasks()

		// Print session info
		fmt.Printf("Session: %s\n\n", sessionID)
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
				return fmt.Errorf("failed to marshal session: %w", err)
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
	sessionShowCmd.Flags().BoolP("verbose", "v", false, "Show detailed task information")
	sessionShowCmd.Flags().Bool("json", false, "Output session as JSON")

	// Add subcommands
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionShowCmd)

	// Add session command to root
	rootCmd.AddCommand(sessionCmd)
}
