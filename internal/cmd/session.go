package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

  Inner loop — start/stop parallel Claude Code, Codex, Gemini, or specular-auto
               runs in isolated Git worktrees (Xirp-competitive session control)
  Outer loop — harness + worktree provenance flows into attestations and the drift gate

Examples:
  specular session start --harness claude-code "Add /healthz endpoint"
  specular session start --name auth --harness codex "Harden JWT validation"
  specular session status --watch
  specular session wait auth ratelimit
  specular session restart auth --harness gemini --force
  specular session logs auth --follow
  cd "$(specular session open auth)"
  specular session fork auth --name auth-alt
  specular session stop auth
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sessionStartCmd = &cobra.Command{
	Use:   "start <goal>",
	Short: "Start a parallel agent session in an isolated worktree",
	Long: `Start a coding-agent session as a managed background process.

Creates (or reuses) a Git worktree under .specular/worktrees/<name> and
launches the selected harness:

  specular-auto  Specular's governed auto pipeline (default)
  claude-code    Anthropic Claude Code CLI (agentic --print)
  codex          OpenAI Codex CLI (exec --full-auto)
  gemini         Google Gemini CLI (--prompt)

Harness + worktree identity are recorded for the outer-loop drift gate.
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
		fmt.Printf("\n  specular session logs %s --follow\n", rec.ID)
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

var sessionLogsCmd = &cobra.Command{
	Use:   "logs <session-id>",
	Short: "Show or follow a session log",
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
		rec, err := mgr.Get(args[0])
		if err != nil {
			return err
		}
		if rec.LogPath == "" {
			return fmt.Errorf("session %s has no log path", rec.ID)
		}
		follow, _ := cmd.Flags().GetBool("follow")
		if !follow {
			b, readErr := os.ReadFile(rec.LogPath)
			if readErr != nil {
				if os.IsNotExist(readErr) {
					fmt.Println("(log empty)")
					return nil
				}
				return readErr
			}
			fmt.Print(string(b))
			return nil
		}
		return followFile(cmd.Context(), rec.LogPath)
	},
}

var sessionForkCmd = &cobra.Command{
	Use:   "fork <session-id>",
	Short: "Fork a session onto a new isolated worktree",
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
		name, _ := cmd.Flags().GetString("name")
		startNow, _ := cmd.Flags().GetBool("start")
		jsonOut, _ := cmd.Flags().GetBool("json")

		rec, err := mgr.Fork(cmd.Context(), args[0], name)
		if err != nil {
			return err
		}
		if startNow {
			rec, err = mgr.Start(cmd.Context(), session.StartOptions{
				Goal:         rec.Goal,
				Name:         rec.ID,
				Harness:      rec.Harness,
				Profile:      rec.Profile,
				NoApproval:   true,
				Detach:       true,
				SkipWorktree: false,
			})
			if err != nil && rec == nil {
				return err
			}
		}
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(rec)
		}
		fmt.Printf("Forked session %s\n", rec.ID)
		fmt.Printf("  Status:   %s\n", rec.Status)
		fmt.Printf("  Worktree: %s (%s)\n", rec.WorktreePath, rec.WorktreeBranch)
		return err
	},
}

var sessionHarnessesCmd = &cobra.Command{
	Use:   "harnesses",
	Short: "List supported coding-agent harnesses and PATH availability",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		list := session.ProbeHarnesses()
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(list)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tKIND\tBINARY\tAVAILABLE")
		for _, h := range list {
			avail := "no"
			if h.Available {
				avail = "yes"
			}
			bin := h.Binary
			if bin == "" {
				bin = "(self)"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", h.Name, h.Kind, bin, avail)
		}
		return w.Flush()
	},
}

var sessionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Live multi-session overview (Xirp-style control surface)",
	Long: `Show a compact status board for all managed sessions.

Use --watch to refresh periodically — the CLI equivalent of a session minimap.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := session.NewManager(cwd)
		if err != nil {
			return err
		}
		watch, _ := cmd.Flags().GetBool("watch")
		interval, _ := cmd.Flags().GetDuration("interval")
		jsonOut, _ := cmd.Flags().GetBool("json")
		if interval <= 0 {
			interval = 2 * time.Second
		}

		printOnce := func() error {
			list, listErr := mgr.List()
			if listErr != nil {
				return listErr
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(list)
			}
			if len(list) == 0 {
				fmt.Println("No managed sessions.")
				return nil
			}
			working, done, failed, stopped := 0, 0, 0, 0
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tHARNESS\tPID\tBRANCH\tGOAL")
			for _, s := range list {
				switch s.Status {
				case session.StatusWorking, session.StatusIdle, session.StatusWaiting:
					working++
				case session.StatusCompleted:
					done++
				case session.StatusFailed:
					failed++
				case session.StatusStopped:
					stopped++
				}
				goal := s.Goal
				if len(goal) > 40 {
					goal = goal[:37] + "..."
				}
				pid := "-"
				if s.PID > 0 {
					pid = fmt.Sprintf("%d", s.PID)
				}
				branch := s.WorktreeBranch
				if branch == "" {
					branch = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", s.ID, s.Status, s.Harness, pid, branch, goal)
			}
			_ = w.Flush()
			fmt.Printf("\nworking=%d  completed=%d  failed=%d  stopped=%d  total=%d\n",
				working, done, failed, stopped, len(list))
			return nil
		}

		if !watch {
			return printOnce()
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		clear := func() { fmt.Print("\033[H\033[2J") }
		for {
			clear()
			fmt.Printf("specular session status  (refresh %s, Ctrl-C to exit)\n\n", interval)
			if err := printOnce(); err != nil {
				return err
			}
			select {
			case <-cmd.Context().Done():
				return nil
			case <-ticker.C:
			}
		}
	},
}

var sessionOpenCmd = &cobra.Command{
	Use:   "open <session-id>",
	Short: "Print the session worktree path (for cd / editors)",
	Long: `Print the isolated worktree path for a session.

  specular session open demo           # path only
  cd "$(specular session open demo)"   # enter worktree
  specular session open demo --shell   # prints: cd /path/to/worktree
  specular session open demo --editor  # open worktree in $EDITOR
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := session.NewManager(cwd)
		if err != nil {
			return err
		}
		rec, err := mgr.Get(args[0])
		if err != nil {
			return err
		}
		path := rec.WorktreePath
		if path == "" {
			return fmt.Errorf("session %s has no worktree (started with --no-worktree?)", rec.ID)
		}
		editor, _ := cmd.Flags().GetBool("editor")
		if editor {
			ed := os.Getenv("EDITOR")
			if ed == "" {
				ed = os.Getenv("VISUAL")
			}
			if ed == "" {
				return fmt.Errorf("EDITOR (or VISUAL) is not set")
			}
			edCmd := exec.Command(ed, path) //nolint:gosec // intentional user $EDITOR
			edCmd.Stdin = os.Stdin
			edCmd.Stdout = os.Stdout
			edCmd.Stderr = os.Stderr
			return edCmd.Run()
		}
		shell, _ := cmd.Flags().GetBool("shell")
		if shell {
			fmt.Printf("cd %q\n", path)
			return nil
		}
		fmt.Println(path)
		return nil
	},
}

var sessionWaitCmd = &cobra.Command{
	Use:   "wait [session-id...]",
	Short: "Block until sessions finish (scriptable parallel gate)",
	Long: `Wait for managed sessions to reach a terminal status.

With no IDs, waits for every currently active (working/waiting) session.
Exit non-zero if any waited session fails or is stopped, or on timeout.

Examples:
  specular session wait
  specular session wait auth ratelimit
  specular session wait --any auth ratelimit
  specular session wait --timeout 10m && specular eval drift --fail-on-change
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := session.NewManager(cwd)
		if err != nil {
			return err
		}
		timeout, _ := cmd.Flags().GetDuration("timeout")
		interval, _ := cmd.Flags().GetDuration("interval")
		anyDone, _ := cmd.Flags().GetBool("any")
		jsonOut, _ := cmd.Flags().GetBool("json")

		recs, waitErr := mgr.Wait(cmd.Context(), args, session.WaitOptions{
			Timeout:  timeout,
			Interval: interval,
			Any:      anyDone,
		})
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(recs)
			return waitErr
		}
		if len(recs) == 0 {
			fmt.Println("No active sessions to wait for.")
			return waitErr
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tHARNESS\tEXIT")
		for _, s := range recs {
			exit := "-"
			if s.ExitCode != nil {
				exit = fmt.Sprintf("%d", *s.ExitCode)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Status, s.Harness, exit)
		}
		_ = w.Flush()
		return waitErr
	},
}

var sessionRestartCmd = &cobra.Command{
	Use:   "restart <session-id>",
	Short: "Re-launch a session in its worktree (optional harness swap)",
	Long: `Restart a managed session, reusing its isolated worktree.

Optionally switch harness or goal — Specular's CLI analogue of swapping
agents without losing project isolation.

Examples:
  specular session restart demo
  specular session restart demo --harness gemini
  specular session restart demo --force --goal "Retry with tests"
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := session.NewManager(cwd)
		if err != nil {
			return err
		}
		harness, _ := cmd.Flags().GetString("harness")
		goal, _ := cmd.Flags().GetString("goal")
		profile, _ := cmd.Flags().GetString("profile")
		force, _ := cmd.Flags().GetBool("force")
		foreground, _ := cmd.Flags().GetBool("foreground")
		jsonOut, _ := cmd.Flags().GetBool("json")

		rec, err := mgr.Restart(cmd.Context(), args[0], session.RestartOptions{
			Harness:    harness,
			Goal:       goal,
			Profile:    profile,
			Force:      force,
			Detach:     !foreground,
			NoApproval: true,
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
		fmt.Printf("Restarted session %s\n", rec.ID)
		fmt.Printf("  Status:   %s\n", rec.Status)
		fmt.Printf("  Harness:  %s\n", rec.Harness)
		if rec.WorktreePath != "" {
			fmt.Printf("  Worktree: %s (%s)\n", rec.WorktreePath, rec.WorktreeBranch)
		}
		if rec.PID > 0 {
			fmt.Printf("  PID:      %d\n", rec.PID)
		}
		return err
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

func followFile(ctx context.Context, path string) error {
	var offset int64
	if st, err := os.Stat(path); err == nil {
		// Print existing content first.
		b, readErr := os.ReadFile(path)
		if readErr == nil {
			fmt.Print(string(b))
			offset = st.Size()
		}
	}
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			f, err := os.Open(path)
			if err != nil {
				continue
			}
			st, statErr := f.Stat()
			if statErr != nil {
				_ = f.Close()
				continue
			}
			if st.Size() > offset {
				if _, seekErr := f.Seek(offset, 0); seekErr == nil {
					buf := make([]byte, st.Size()-offset)
					n, _ := f.Read(buf)
					if n > 0 {
						fmt.Print(string(buf[:n]))
						offset += int64(n)
					}
				}
			}
			_ = f.Close()
		}
	}
}

func init() {
	sessionStartCmd.Flags().String("name", "", "Session / worktree name (default: sess-<timestamp>)")
	sessionStartCmd.Flags().String("harness", "specular-auto", "Harness: specular-auto|claude-code|codex|gemini")
	sessionStartCmd.Flags().String("profile", "ci", "Auto profile (specular-auto only)")
	sessionStartCmd.Flags().Bool("no-worktree", false, "Run in the current checkout (not isolated)")
	sessionStartCmd.Flags().Bool("foreground", false, "Run in the foreground instead of detaching")
	sessionStartCmd.Flags().Bool("json", false, "Emit JSON")

	sessionListCmd.Flags().Bool("checkpoints", false, "Also list legacy auto checkpoints")
	sessionListCmd.Flags().Bool("json", false, "Emit JSON")

	sessionShowCmd.Flags().BoolP("verbose", "v", false, "Show log tail / task details")
	sessionShowCmd.Flags().Bool("json", false, "Emit JSON")

	sessionStopCmd.Flags().Bool("json", false, "Emit JSON")

	sessionLogsCmd.Flags().Bool("follow", false, "Follow log output")
	sessionForkCmd.Flags().String("name", "", "Name for the forked session")
	sessionForkCmd.Flags().Bool("start", false, "Start the forked session immediately")
	sessionForkCmd.Flags().Bool("json", false, "Emit JSON")

	sessionHarnessesCmd.Flags().Bool("json", false, "Emit JSON")
	sessionStatusCmd.Flags().Bool("watch", false, "Refresh the status board until interrupted")
	sessionStatusCmd.Flags().Duration("interval", 2*time.Second, "Refresh interval for --watch")
	sessionStatusCmd.Flags().Bool("json", false, "Emit JSON")
	sessionOpenCmd.Flags().Bool("shell", false, "Print a cd command instead of the bare path")
	sessionOpenCmd.Flags().Bool("editor", false, "Open the worktree in $EDITOR")

	sessionWaitCmd.Flags().Duration("timeout", 0, "Maximum time to wait (0 = no limit)")
	sessionWaitCmd.Flags().Duration("interval", 500*time.Millisecond, "Poll interval")
	sessionWaitCmd.Flags().Bool("any", false, "Return when any named session finishes")
	sessionWaitCmd.Flags().Bool("json", false, "Emit JSON")

	sessionRestartCmd.Flags().String("harness", "", "Switch harness on restart")
	sessionRestartCmd.Flags().String("goal", "", "Override goal on restart")
	sessionRestartCmd.Flags().String("profile", "", "Override auto profile on restart")
	sessionRestartCmd.Flags().Bool("force", false, "Stop a still-running session before restart")
	sessionRestartCmd.Flags().Bool("foreground", false, "Run in the foreground instead of detaching")
	sessionRestartCmd.Flags().Bool("json", false, "Emit JSON")

	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionShowCmd)
	sessionCmd.AddCommand(sessionStopCmd)
	sessionCmd.AddCommand(sessionLogsCmd)
	sessionCmd.AddCommand(sessionForkCmd)
	sessionCmd.AddCommand(sessionHarnessesCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
	sessionCmd.AddCommand(sessionOpenCmd)
	sessionCmd.AddCommand(sessionWaitCmd)
	sessionCmd.AddCommand(sessionRestartCmd)
	rootCmd.AddCommand(sessionCmd)
}
