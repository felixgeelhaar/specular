package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	"github.com/felixgeelhaar/specular/internal/session"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage parallel agent sessions (inner loop + governance)",
	Long: `Manage Specular agent sessions across both loops:

  Inner loop — start/stop parallel auto runs in isolated Git worktrees
  Outer loop — harness + worktree provenance flows into attestations and the drift gate

Sessions are first-class tracked runs (not just checkpoints). Use
'session start' to launch a goal in the background; 'session list' to
see working/completed/failed; 'session stop' to terminate.

Legacy auto checkpoints under .specular/checkpoints remain visible via
'session list --checkpoints'.

Examples:
  specular session start "Add /healthz endpoint"
  specular session start --name auth-fix --harness claude-code "Harden JWT validation"
  specular session list
  specular session show auth-fix
  specular session stop auth-fix
  specular auto --resume <checkpoint-id>
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sessionStartCmd = &cobra.Command{
	Use:   "start <goal>",
	Short: "Start a parallel agent session in an isolated worktree",
	Long: `Start a Specular auto run as a managed background session.

Creates (or reuses) a Git worktree under .specular/worktrees/<name>,
launches 'specular auto' detached with harness provenance, and records
the session under .specular/sessions/.

This is the inner-loop orchestration surface: many sessions can run in
parallel; each still hits the same outer-loop drift/policy gate when its
changes land.
`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := args[0]
		for i := 1; i < len(args); i++ {
			goal += " " + args[i]
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := session.NewManager(cwd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		harness, _ := cmd.Flags().GetString("harness")
		profile, _ := cmd.Flags().GetString("profile")
		noWorktree, _ := cmd.Flags().GetBool("no-worktree")
		foreground, _ := cmd.Flags().GetBool("foreground")
		jsonOut, _ := cmd.Flags().GetBool("json")

		rec, err := mgr.Start(cmd.Context(), session.StartOptions{
			Goal:         goal,
			Name:         name,
			Harness:      harness,
			Profile:      profile,
			NoApproval:   true,
			Detach:       !foreground,
			SkipWorktree: noWorktree,
		})
		if err != nil && rec == nil {
			return err
		}

		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(rec)
			return err
		}

		fmt.Printf("Started session %s\n", rec.ID)
		fmt.Printf("  Status:   %s\n", rec.Status)
		fmt.Printf("  Harness:  %s\n", rec.Harness)
		if rec.WorktreePath != "" {
			fmt.Printf("  Worktree: %s (%s)\n", rec.WorktreePath, rec.WorktreeBranch)
		}
		if rec.PID > 0 {
			fmt.Printf("  PID:      %d\n", rec.PID)
		}
		if rec.LogPath != "" {
			fmt.Printf("  Log:      %s\n", rec.LogPath)
		}
		fmt.Printf("\n  specular session show %s\n", rec.ID)
		fmt.Printf("  specular session stop %s\n", rec.ID)
		return err
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed agent sessions",
	Long: `List Specular-managed sessions from .specular/sessions.

Use --checkpoints to also show legacy auto checkpoint sessions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		includeCheckpoints, _ := cmd.Flags().GetBool("checkpoints")
		jsonOut, _ := cmd.Flags().GetBool("json")

		mgr, err := session.NewManager(cwd)
		if err != nil {
			// Not a git repo — fall back to checkpoints only.
			if includeCheckpoints {
				return listCheckpointSessions(jsonOut)
			}
			return err
		}

		list, err := mgr.List()
		if err != nil {
			return err
		}

		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(list)
		}

		if len(list) == 0 && !includeCheckpoints {
			fmt.Println("No managed sessions. Start one with: specular session start \"your goal\"")
			return nil
		}

		if len(list) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tHARNESS\tWORKTREE\tPID\tGOAL")
			for _, s := range list {
				goal := s.Goal
				if len(goal) > 48 {
					goal = goal[:45] + "..."
				}
				pid := "-"
				if s.PID > 0 {
					pid = fmt.Sprintf("%d", s.PID)
				}
				wt := s.WorktreeName
				if wt == "" {
					wt = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", s.ID, s.Status, s.Harness, wt, pid, goal)
			}
			_ = w.Flush()
		}

		if includeCheckpoints {
			fmt.Println()
			fmt.Println("Legacy checkpoints:")
			return listCheckpointSessions(false)
		}
		return nil
	},
}

var sessionShowCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Show detailed information about a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		jsonOut, _ := cmd.Flags().GetBool("json")
		verbose, _ := cmd.Flags().GetBool("verbose")

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		if mgr, err := session.NewManager(cwd); err == nil {
			if rec, err := mgr.Get(id); err == nil {
				if jsonOut {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(rec)
				}
				fmt.Printf("Session: %s\n\n", rec.ID)
				fmt.Printf("Status:     %s\n", rec.Status)
				fmt.Printf("Goal:       %s\n", rec.Goal)
				fmt.Printf("Harness:    %s\n", rec.Harness)
				fmt.Printf("Profile:    %s\n", rec.Profile)
				if rec.WorktreePath != "" {
					fmt.Printf("Worktree:   %s\n", rec.WorktreePath)
					fmt.Printf("Branch:     %s\n", rec.WorktreeBranch)
				}
				if rec.PID > 0 {
					fmt.Printf("PID:        %d\n", rec.PID)
				}
				fmt.Printf("Created:    %s\n", rec.CreatedAt.Format(time.RFC3339))
				fmt.Printf("Updated:    %s\n", rec.UpdatedAt.Format(time.RFC3339))
				if rec.LogPath != "" {
					fmt.Printf("Log:        %s\n", rec.LogPath)
				}
				if rec.Error != "" {
					fmt.Printf("Error:      %s\n", rec.Error)
				}
				if verbose && rec.LogPath != "" {
					if b, err := os.ReadFile(rec.LogPath); err == nil && len(b) > 0 {
						fmt.Println("\n--- log (tail) ---")
						fmt.Print(tailBytes(b, 4000))
					}
				}
				return nil
			}
		}

		// Fall back to checkpoint show
		return showCheckpointSession(id, jsonOut, verbose)
	},
}

var sessionStopCmd = &cobra.Command{
	Use:   "stop <session-id>",
	Short: "Stop a running agent session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := session.NewManager(cwd)
		if err != nil {
			return err
		}
		rec, err := mgr.Stop(args[0])
		if err != nil {
			return err
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(rec)
		}
		fmt.Printf("Stopped session %s\n", rec.ID)
		return nil
	},
}

func listCheckpointSessions(asJSON bool) error {
	checkpointMgr := checkpoint.NewManager(".specular/checkpoints", false, 0)
	checkpointIDs, err := checkpointMgr.List()
	if err != nil {
		return fmt.Errorf("failed to list checkpoints: %w", err)
	}
	if len(checkpointIDs) == 0 {
		fmt.Println("No checkpoint sessions found.")
		return nil
	}

	type sessionInfo struct {
		ID        string    `json:"id"`
		Status    string    `json:"status"`
		StartedAt time.Time `json:"startedAt"`
		Product   string    `json:"product"`
		Goal      string    `json:"goal"`
		Completed int       `json:"completed"`
		Total     int       `json:"total"`
		Failed    int       `json:"failed"`
	}

	var sessions []sessionInfo
	for _, id := range checkpointIDs {
		cpState, err := checkpointMgr.Load(id)
		if err != nil {
			continue
		}
		product, _ := cpState.GetMetadata("product")
		goal, _ := cpState.GetMetadata("goal")
		sessions = append(sessions, sessionInfo{
			ID:        id,
			Status:    cpState.Status,
			StartedAt: cpState.StartedAt,
			Product:   product,
			Goal:      goal,
			Completed: len(cpState.GetCompletedTasks()),
			Total:     len(cpState.Tasks),
			Failed:    len(cpState.GetFailedTasks()),
		})
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartedAt.After(sessions[j].StartedAt)
	})

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sessions)
	}

	for _, s := range sessions {
		fmt.Printf("%s  %s  %d/%d  %s\n", s.ID, s.Status, s.Completed, s.Total, s.Goal)
	}
	return nil
}

func showCheckpointSession(id string, asJSON, verbose bool) error {
	checkpointMgr := checkpoint.NewManager(".specular/checkpoints", false, 0)
	cpState, err := checkpointMgr.Load(id)
	if err != nil {
		return fmt.Errorf("session not found in registry or checkpoints: %s", id)
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cpState)
	}
	product, _ := cpState.GetMetadata("product")
	goal, _ := cpState.GetMetadata("goal")
	fmt.Printf("Checkpoint session: %s\n\n", id)
	fmt.Printf("Status:     %s\n", cpState.Status)
	fmt.Printf("Product:    %s\n", product)
	fmt.Printf("Goal:       %s\n", goal)
	fmt.Printf("Started:    %s\n", cpState.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:    %s\n", cpState.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Progress:   %d/%d completed\n", len(cpState.GetCompletedTasks()), len(cpState.Tasks))
	if verbose {
		for tid, task := range cpState.Tasks {
			fmt.Printf("  %s (%s)\n", tid, task.Status)
		}
	}
	return nil
}

func tailBytes(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[len(b)-n:])
}

func init() {
	sessionStartCmd.Flags().String("name", "", "Session / worktree name (default: sess-<timestamp>)")
	sessionStartCmd.Flags().String("harness", "specular-auto", "Harness label recorded in attestation provenance")
	sessionStartCmd.Flags().String("profile", "ci", "Auto profile to use")
	sessionStartCmd.Flags().Bool("no-worktree", false, "Run in the current checkout (not isolated)")
	sessionStartCmd.Flags().Bool("foreground", false, "Run in the foreground instead of detaching")
	sessionStartCmd.Flags().Bool("json", false, "Emit JSON")

	sessionListCmd.Flags().Bool("checkpoints", false, "Also list legacy auto checkpoints")
	sessionListCmd.Flags().Bool("json", false, "Emit JSON")

	sessionShowCmd.Flags().BoolP("verbose", "v", false, "Show log tail / task details")
	sessionShowCmd.Flags().Bool("json", false, "Emit JSON")

	sessionStopCmd.Flags().Bool("json", false, "Emit JSON")

	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionShowCmd)
	sessionCmd.AddCommand(sessionStopCmd)
	rootCmd.AddCommand(sessionCmd)
}
