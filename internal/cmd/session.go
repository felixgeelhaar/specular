package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	"github.com/felixgeelhaar/specular/internal/gate"
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
  specular session integrate claude-code
  specular session integrate cursor
  specular session integrate codex
  specular session integrate gemini
  specular session start --harness claude-code "Add /healthz endpoint"
  specular session start --name auth --harness codex "Harden JWT validation"
  specular session status --watch
  specular session wait auth ratelimit
  specular session restart auth --harness gemini --force
  specular session logs auth --follow
  cd "$(specular session open auth)"
  specular session fork auth --name auth-alt
  specular session stop auth
  specular session prune --delete-branch
  specular session diff auth --stat
  specular session exec auth -- go test ./...
  specular session commit auth
  specular session sync auth
  specular session push auth --pr
  specular session merge auth
  specular session cherry-pick review --from auth
  specular session attest auth
  specular session batch fleet.yaml
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sessionStartCmd = &cobra.Command{
	Use:   "start [goal]",
	Short: "Start a parallel agent session in an isolated worktree",
	Long: `Start a coding-agent session as a managed background process.

Creates (or reuses) a Git worktree under .specular/worktrees/<name> and
launches the selected harness:

  specular-auto  Specular's governed auto pipeline (default)
  claude-code    Anthropic Claude Code CLI (agentic --print)
  codex          OpenAI Codex CLI (exec --full-auto)
  gemini         Google Gemini CLI (--prompt)

Harness + worktree identity are recorded for the outer-loop drift gate.

Native harnesses auto-enable --governed when .specular/policy.yaml (or
policies.yaml) is present. Pass --no-governed to keep skip-permissions /
full-auto even with a policy file.

Fleet launch (CI-native vs Xirp's Mac grid):

  specular session start --manifest fleet.yaml
  specular session wait
`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		manifestPath, _ := cmd.Flags().GetString("manifest")
		if manifestPath != "" {
			return runSessionManifest(cmd, manifestPath)
		}
		if len(args) < 1 {
			return fmt.Errorf("goal is required (or pass --manifest)")
		}
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
		governed, _ := cmd.Flags().GetBool("governed")
		noGoverned, _ := cmd.Flags().GetBool("no-governed")
		jsonOut, _ := cmd.Flags().GetBool("json")
		if governed && noGoverned {
			return fmt.Errorf("session: --governed and --no-governed are mutually exclusive")
		}

		rec, err := mgr.Start(cmd.Context(), session.StartOptions{
			Goal:         goal,
			Name:         name,
			Harness:      harness,
			Profile:      profile,
			NoApproval:   true,
			Detach:       !foreground,
			SkipWorktree: noWorktree,
			Governed:     governed,
			NoGoverned:   noGoverned,
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
		if rec.Governed {
			fmt.Printf("  Governed: true\n")
		}
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

var sessionBatchCmd = &cobra.Command{
	Use:   "batch <manifest>",
	Short: "Start a fleet of sessions from a YAML/JSON manifest",
	Long: `Launch multiple parallel sessions from a fleet manifest.

Manifest is a YAML/JSON array of entries, or an object with a "sessions" key:

  - name: auth
    harness: claude-code
    goal: Harden JWT validation
  - name: ratelimit
    harness: codex
    goal: Add rate limiting
  - name: review
    harness: gemini
    goal: Review auth + ratelimit diffs
    dependsOn: [auth, ratelimit]

Entries with dependsOn stay queued until every parent completes successfully.
A failed parent aborts the remaining chain.

Then:

  specular session batch fleet.yaml
  specular session wait
  specular session diff auth --against ratelimit
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionManifest(cmd, args[0])
	},
}

func runSessionManifest(cmd *cobra.Command, manifestPath string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	mgr, err := session.NewManager(cwd)
	if err != nil {
		return err
	}
	data, readErr := os.ReadFile(manifestPath)
	if readErr != nil {
		return fmt.Errorf("read manifest: %w", readErr)
	}
	entries, parseErr := session.ParseManifest(data)
	if parseErr != nil {
		return parseErr
	}
	harness, _ := cmd.Flags().GetString("harness")
	profile, _ := cmd.Flags().GetString("profile")
	noWorktree, _ := cmd.Flags().GetBool("no-worktree")
	governed, _ := cmd.Flags().GetBool("governed")
	noGoverned, _ := cmd.Flags().GetBool("no-governed")
	jsonOut, _ := cmd.Flags().GetBool("json")
	if governed && noGoverned {
		return fmt.Errorf("session: --governed and --no-governed are mutually exclusive")
	}

	started, startErr := mgr.StartMany(cmd.Context(), entries, session.StartOptions{
		Harness:      harness,
		Profile:      profile,
		SkipWorktree: noWorktree,
		Governed:     governed,
		NoGoverned:   noGoverned,
	})
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(started)
		return startErr
	}
	ids := make([]string, 0, len(started))
	for _, rec := range started {
		gov := ""
		if rec.Governed {
			gov = " governed=true"
		}
		fmt.Printf("Started session %s (%s) harness=%s%s\n", rec.ID, rec.Status, rec.Harness, gov)
		ids = append(ids, rec.ID)
	}
	if len(ids) > 0 {
		fmt.Printf("\n  specular session wait %s\n", strings.Join(ids, " "))
		fmt.Printf("  specular session status\n")
	}
	return startErr
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed agent sessions",
	Long: `List Specular-managed sessions from .specular/sessions.

Use --checkpoints to also show legacy auto checkpoint sessions.
Trust filters (--verdict/--soft-allow/--risk/--protocol/--attested/--governed/--harness)
match status board columns (evidence list parity).`,
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
		filter, ferr := sessionBoardFilter(cmd)
		if ferr != nil {
			return ferr
		}
		evMap := session.EvidenceMapFor(list, mgr.Store().Dir(), mgr.RepoRoot())
		list = session.FilterSessions(list, evMap, filter)

		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(list)
		}

		if len(list) == 0 && !includeCheckpoints {
			if filter.Active() {
				fmt.Println("No managed sessions match filters.")
				return nil
			}
			fmt.Println("No managed sessions. Start one with: specular session start \"your goal\"")
			return nil
		}

		if len(list) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tHARNESS\tGOV\tATTEST\tAPP\tPROTO\tCOMMIT\tGATE\tEVID\tSOFT\tRISK\tWORKTREE\tPID\tGOAL")
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
				gov := session.YesDash(s.Governed)
				ev := evMap[s.ID]
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					s.ID, s.Status, s.Harness, gov, session.YesDash(ev.Attested), session.YesDash(ev.App), session.YesDash(ev.Protocol), session.DashOr(ev.Commit), session.DashOr(ev.Verdict), session.DashOr(ev.EvidenceID), session.YesDash(ev.SoftAllow), session.DashOr(ev.Risk), wt, pid, goal)
			}
			_ = w.Flush()
			if hints := session.FormatSoftAllowBoardHints(list, evMap); hints != "" {
				fmt.Print("\n" + hints)
			}
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
	Long: `Show session details, worktree, harness, and sibling attestation/APP paths.

When a Change Evidence Graph record lists this session, also shows GATE / SOFT /
RISK / PROTO / evidence id and DENY Next steps, with jumps to explain --session /
evidence show; soft-ALLOW overrules jump to approvals show / list --status open.`,
	Args: cobra.ExactArgs(1),
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
				gateEv := session.GateDetailsFor(mgr.Store().Dir(), mgr.RepoRoot(), *rec)
				if jsonOut {
					out := sessionShowJSON{Record: *rec}
					if gateEv.HasSurface() {
						out.Evidence = &gateEv
					}
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(out)
				}
				fmt.Printf("Session: %s\n\n", rec.ID)
				fmt.Printf("Status:     %s\n", rec.Status)
				fmt.Printf("Goal:       %s\n", rec.Goal)
				fmt.Printf("Harness:    %s\n", rec.Harness)
				if rec.Governed {
					fmt.Printf("Governed:   true\n")
				}
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
				attRel := filepath.Join(".specular", "sessions", rec.ID+".attestation.json")
				provRel := filepath.Join(".specular", "sessions", rec.ID+".provenance.json")
				if _, err := os.Stat(filepath.Join(cwd, attRel)); err == nil {
					fmt.Printf("Attest:     %s\n", filepath.ToSlash(attRel))
				}
				if _, err := os.Stat(filepath.Join(cwd, provRel)); err == nil {
					fmt.Printf("Provenance: %s\n", filepath.ToSlash(provRel))
					fmt.Printf("Verify:     specular provenance verify %s\n", rec.ID)
				}
				printSessionGateBlock(rec.ID, gateEv)
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

// sessionShowJSON extends the session record with optional newest-gate evidence.
type sessionShowJSON struct {
	session.Record
	Evidence *session.GateDetails `json:"evidence,omitempty"`
}

func printSessionGateBlock(sessionID string, ev session.GateDetails) {
	if ev.Verdict == "" && ev.EvidenceID == "" {
		return
	}
	if ev.Commit != "" {
		fmt.Printf("Commit:     %s\n", ev.Commit)
	}
	fmt.Printf("Gate:       %s\n", session.DashOr(ev.Verdict))
	fmt.Printf("Soft:       %s\n", session.YesDash(ev.SoftAllow))
	fmt.Printf("Risk:       %s\n", session.DashOr(ev.Risk))
	fmt.Printf("Protocol:   %s\n", session.YesDash(ev.Protocol))
	if ev.EvidenceID != "" {
		fmt.Printf("Evidence:   %s\n", ev.EvidenceID)
		fmt.Printf("Explain:    specular explain --session %s\n", sessionID)
		fmt.Printf("            specular evidence show %s\n", ev.EvidenceID)
	}
	for _, id := range ev.SoftAllowIDs {
		fmt.Printf("Approval:   specular approvals show %s\n", id)
	}
	if len(ev.SoftAllowIDs) > 0 {
		fmt.Printf("            %s\n", gate.SoftAllowListHint(ev.EvidenceID))
		fmt.Printf("            %s\n", gate.SoftAllowPendingHint)
		fmt.Printf("            %s\n", gate.SoftAllowDoctorHint)
	}
	if len(ev.NextSteps) > 0 {
		fmt.Println("\nNext steps:")
		for _, step := range ev.NextSteps {
			fmt.Printf("  • %s\n", step)
		}
	}
}

func sessionBoardFilter(cmd *cobra.Command) (session.BoardFilter, error) {
	var f session.BoardFilter
	verdict, _ := cmd.Flags().GetString("verdict")
	f.Verdict = strings.TrimSpace(verdict)
	harness, _ := cmd.Flags().GetString("harness")
	f.Harness = strings.TrimSpace(harness)
	risk, _ := cmd.Flags().GetString("risk")
	f.RiskLevel = strings.TrimSpace(risk)
	if cmd.Flags().Changed("soft-allow") {
		v, _ := cmd.Flags().GetBool("soft-allow")
		f.SoftAllow = &v
	}
	if cmd.Flags().Changed("attested") {
		v, _ := cmd.Flags().GetBool("attested")
		f.Attested = &v
	}
	if cmd.Flags().Changed("governed") {
		v, _ := cmd.Flags().GetBool("governed")
		f.Governed = &v
	}
	if cmd.Flags().Changed("protocol") {
		v, _ := cmd.Flags().GetBool("protocol")
		f.Protocol = &v
	}
	if err := f.Validate(); err != nil {
		return f, err
	}
	return f, nil
}

func addSessionBoardFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("verdict", "", "Only sessions whose newest evidence gate is ALLOW or DENY")
	cmd.Flags().String("harness", "", "Only sessions whose harness contains this substring")
	cmd.Flags().String("risk", "", "Only sessions whose newest evidence risk is NONE|LOW|MEDIUM|HIGH|CRITICAL")
	cmd.Flags().Bool("soft-allow", false, "Only sessions whose newest evidence has (or lacks, with =false) soft-ALLOW overrules")
	cmd.Flags().Bool("attested", false, "Only sessions with (or without, =false) sibling attestation")
	cmd.Flags().Bool("governed", false, "Only sessions with (or without, =false) GOV=yes")
	cmd.Flags().Bool("protocol", false, "Only sessions whose newest evidence has (or lacks, =false) APP protocol schema+bound")
}

var sessionStopCmd = &cobra.Command{
	Use:   "stop [session-id...]",
	Short: "Stop running agent session(s)",
	Long: `Stop one or more managed sessions (Xirp grid kill analogue).

  specular session stop auth
  specular session stop auth ratelimit
  specular session stop --all
`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := session.NewManager(cwd)
		if err != nil {
			return err
		}
		all, _ := cmd.Flags().GetBool("all")
		jsonOut, _ := cmd.Flags().GetBool("json")
		if all && len(args) > 0 {
			return fmt.Errorf("session: pass session IDs or --all, not both")
		}
		if !all && len(args) == 0 {
			return fmt.Errorf("session: session ID required (or pass --all)")
		}

		var stopped []session.Record
		var stopErr error
		if all {
			stopped, stopErr = mgr.StopAll()
		} else {
			stopped, stopErr = mgr.StopMany(args)
		}
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(stopped)
			return stopErr
		}
		if len(stopped) == 0 {
			fmt.Println("No sessions to stop.")
			return stopErr
		}
		for _, rec := range stopped {
			fmt.Printf("Stopped session %s\n", rec.ID)
		}
		return stopErr
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

Use --watch to refresh periodically — the CLI equivalent of a session minimap.
With --json, emit {summary, sessions, evidence} for dashboards (not a bare array).
Evidence includes ATTEST/APP/PROTO/COMMIT plus GATE verdict/EVID (evidenceId), SOFT
(soft-ALLOW overrules), and RISK from newest graph records.

Trust filters (--verdict/--soft-allow/--risk/--protocol/--attested/--governed/--harness)
narrow the board (evidence list parity); summary counts reflect the filtered set.`,
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
		filter, ferr := sessionBoardFilter(cmd)
		if ferr != nil {
			return ferr
		}
		if interval <= 0 {
			interval = 2 * time.Second
		}

		printOnce := func() error {
			list, listErr := mgr.List()
			if listErr != nil {
				return listErr
			}
			evMap := session.EvidenceMapFor(list, mgr.Store().Dir(), mgr.RepoRoot())
			list = session.FilterSessions(list, evMap, filter)
			board := session.BuildStatusBoardWithEvidence(list, mgr.Store().Dir(), mgr.RepoRoot())
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(board)
			}
			if len(board.Sessions) == 0 {
				if filter.Active() {
					fmt.Println("No managed sessions match filters.")
					return nil
				}
				fmt.Println("No managed sessions.")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tHARNESS\tGOV\tATTEST\tAPP\tPROTO\tCOMMIT\tGATE\tEVID\tSOFT\tRISK\tPID\tBRANCH\tGOAL")
			for _, s := range board.Sessions {
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
				gov := session.YesDash(s.Governed)
				ev := board.Evidence[s.ID]
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					s.ID, s.Status, s.Harness, gov, session.YesDash(ev.Attested), session.YesDash(ev.App), session.YesDash(ev.Protocol), session.DashOr(ev.Commit), session.DashOr(ev.Verdict), session.DashOr(ev.EvidenceID), session.YesDash(ev.SoftAllow), session.DashOr(ev.Risk), pid, branch, goal)
			}
			_ = w.Flush()
			fmt.Printf("\nworking=%d  queued=%d  completed=%d  failed=%d  stopped=%d  total=%d\n",
				board.Summary.Working, board.Summary.Queued, board.Summary.Completed,
				board.Summary.Failed, board.Summary.Stopped, board.Summary.Total)
			if hints := session.FormatSoftAllowBoardHints(board.Sessions, board.Evidence); hints != "" {
				fmt.Print("\n" + hints)
			}
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

After wait (and optional --attest/--gate/--bundle), the human board and
--json output match session status trust columns (GOV…RISK + EXIT), so
fleet→gate evidence is visible without a separate status call.

Trust filters (--verdict/--soft-allow/--risk/--protocol/--attested/--governed/--harness)
narrow the emitted board only (wait/attest/gate still cover the full waited set).

Examples:
  specular session wait
  specular session wait auth ratelimit
  specular session wait --any auth ratelimit
  specular session wait --timeout 10m && specular gate
  specular session wait --timeout 45m --stop
  specular session wait --attest auth ratelimit
  specular session wait --attest --gate
  specular session wait --gate --require-attested --require-protocol
  specular session wait --bundle --require-governed --policy .specular/policies/soc2-cc8.1.yaml
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
		stopOnTimeout, _ := cmd.Flags().GetBool("stop")
		jsonOut, _ := cmd.Flags().GetBool("json")
		doAttest, _ := cmd.Flags().GetBool("attest")
		doGate, _ := cmd.Flags().GetBool("gate")
		doBundle, _ := cmd.Flags().GetBool("bundle")
		bundleOut, _ := cmd.Flags().GetString("bundle-out")
		policies, _ := cmd.Flags().GetStringSlice("policy")
		requireAttested, _ := cmd.Flags().GetBool("require-attested")
		requireProtocol, _ := cmd.Flags().GetBool("require-protocol")
		requireGoverned, _ := cmd.Flags().GetBool("require-governed")
		if stopOnTimeout && timeout <= 0 {
			return fmt.Errorf("session: --stop requires --timeout")
		}
		if (requireAttested || requireProtocol || requireGoverned) && !doGate && !doBundle {
			return fmt.Errorf("session: --require-* requires --gate or --bundle")
		}
		filter, ferr := sessionBoardFilter(cmd)
		if ferr != nil {
			return ferr
		}

		postOpts := sessionWaitPostOptions{
			Ctx: cmd.Context(), Mgr: mgr,
			Attest: doAttest, Gate: doGate, Bundle: doBundle,
			BundleOut: bundleOut, Policies: policies,
			RequireAttested: requireAttested,
			RequireProtocol: requireProtocol,
			RequireGoverned: requireGoverned,
			Quiet:           jsonOut,
		}

		recs, waitErr := mgr.Wait(cmd.Context(), args, session.WaitOptions{
			Timeout:       timeout,
			Interval:      interval,
			Any:           anyDone,
			StopOnTimeout: stopOnTimeout,
		})
		postOpts.Recs = recs
		if waitErr != nil {
			_ = emitSessionWaitBoard(mgr, filterWaitRecs(mgr, recs, filter), jsonOut, filter.Active())
			return waitErr
		}
		if len(recs) == 0 && !doGate && !doBundle && !doAttest {
			if !jsonOut {
				fmt.Println("No active sessions to wait for.")
			} else {
				_ = emitSessionWaitBoard(mgr, recs, true, false)
			}
			return nil
		}
		postErr := runSessionWaitPost(postOpts)
		if emitErr := emitSessionWaitBoard(mgr, filterWaitRecs(mgr, recs, filter), jsonOut, filter.Active()); emitErr != nil && postErr == nil {
			return emitErr
		}
		return postErr
	},
}

func filterWaitRecs(mgr *session.Manager, recs []session.Record, filter session.BoardFilter) []session.Record {
	if !filter.Active() {
		return recs
	}
	evMap := session.EvidenceMapFor(recs, mgr.Store().Dir(), mgr.RepoRoot())
	return session.FilterSessions(recs, evMap, filter)
}

// emitSessionWaitBoard prints or encodes the waited sessions with status-board
// trust columns (and EXIT). Evidence is refreshed from disk so --gate results show.
func emitSessionWaitBoard(mgr *session.Manager, recs []session.Record, jsonOut, filtered bool) error {
	board := session.BuildStatusBoardWithEvidence(recs, mgr.Store().Dir(), mgr.RepoRoot())
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(board)
	}
	if len(board.Sessions) == 0 {
		if filtered {
			fmt.Println("No waited sessions match filters.")
			return nil
		}
		fmt.Println("No active sessions to wait for.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tHARNESS\tGOV\tATTEST\tAPP\tPROTO\tCOMMIT\tGATE\tEVID\tSOFT\tRISK\tEXIT")
	for _, s := range board.Sessions {
		exit := "-"
		if s.ExitCode != nil {
			exit = fmt.Sprintf("%d", *s.ExitCode)
		}
		gov := session.YesDash(s.Governed)
		ev := board.Evidence[s.ID]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			s.ID, s.Status, s.Harness, gov, session.YesDash(ev.Attested), session.YesDash(ev.App),
			session.YesDash(ev.Protocol), session.DashOr(ev.Commit), session.DashOr(ev.Verdict),
			session.DashOr(ev.EvidenceID), session.YesDash(ev.SoftAllow), session.DashOr(ev.Risk), exit)
	}
	_ = w.Flush()
	fmt.Printf("\nworking=%d  queued=%d  completed=%d  failed=%d  stopped=%d  total=%d\n",
		board.Summary.Working, board.Summary.Queued, board.Summary.Completed,
		board.Summary.Failed, board.Summary.Stopped, board.Summary.Total)
	if hints := session.FormatSoftAllowBoardHints(board.Sessions, board.Evidence); hints != "" {
		fmt.Print("\n" + hints)
	}
	return nil
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
  specular session restart demo --no-governed --force
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
		governedChanged := cmd.Flags().Changed("governed")
		governed, _ := cmd.Flags().GetBool("governed")
		noGovernedChanged := cmd.Flags().Changed("no-governed")
		noGoverned, _ := cmd.Flags().GetBool("no-governed")
		if governedChanged && noGovernedChanged {
			return fmt.Errorf("session: --governed and --no-governed are mutually exclusive")
		}

		rec, err := mgr.Restart(cmd.Context(), args[0], session.RestartOptions{
			Harness:       harness,
			Goal:          goal,
			Profile:       profile,
			Force:         force,
			Detach:        !foreground,
			NoApproval:    true,
			Governed:      governed,
			UseGoverned:   governedChanged,
			NoGoverned:    noGoverned,
			UseNoGoverned: noGovernedChanged,
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

var sessionRmCmd = &cobra.Command{
	Use:   "rm <session-id> [session-id...]",
	Short: "Remove session records (and worktrees by default)",
	Long: `Delete finished session records, logs, and exit sidecars.

By default also removes the associated Git worktree. Use --force to stop
a still-running session before removal.

Examples:
  specular session rm demo
  specular session rm auth ratelimit --delete-branch
  specular session rm stuck --force
  specular session rm demo --keep-worktree
`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := session.NewManager(cwd)
		if err != nil {
			return err
		}
		force, _ := cmd.Flags().GetBool("force")
		keepWT, _ := cmd.Flags().GetBool("keep-worktree")
		delBranch, _ := cmd.Flags().GetBool("delete-branch")
		jsonOut, _ := cmd.Flags().GetBool("json")

		var removed []session.Record
		for _, id := range args {
			rec, rmErr := mgr.Remove(cmd.Context(), id, session.RemoveOptions{
				Force:        force,
				KeepWorktree: keepWT,
				DeleteBranch: delBranch,
			})
			if rmErr != nil {
				return rmErr
			}
			if rec != nil {
				removed = append(removed, *rec)
			}
		}
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(removed)
		}
		for _, rec := range removed {
			fmt.Printf("Removed session %s (%s)\n", rec.ID, rec.Status)
		}
		return nil
	},
}

var sessionPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove finished sessions (optional age filter)",
	Long: `Prune terminal sessions (completed, failed, stopped, idle).

Closes the scriptable parallel loop: start → wait → drift → prune.

Examples:
  specular session prune
  specular session prune --older-than 24h
  specular session prune --delete-branch
  specular session prune --keep-worktree
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := session.NewManager(cwd)
		if err != nil {
			return err
		}
		older, _ := cmd.Flags().GetDuration("older-than")
		keepWT, _ := cmd.Flags().GetBool("keep-worktree")
		delBranch, _ := cmd.Flags().GetBool("delete-branch")
		jsonOut, _ := cmd.Flags().GetBool("json")

		removed, pruneErr := mgr.Prune(cmd.Context(), session.PruneOptions{
			OlderThan:    older,
			KeepWorktree: keepWT,
			DeleteBranch: delBranch,
		})
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(removed)
			return pruneErr
		}
		if len(removed) == 0 {
			fmt.Println("No finished sessions to prune.")
			return pruneErr
		}
		for _, rec := range removed {
			fmt.Printf("Pruned session %s (%s)\n", rec.ID, rec.Status)
		}
		fmt.Printf("Removed %d session(s).\n", len(removed))
		return pruneErr
	},
}

var sessionDiffCmd = &cobra.Command{
	Use:   "diff <session-id>",
	Short: "Show Git changes for a session worktree",
	Long: `Show changes in a session worktree versus a base ref (default: main/master/HEAD).

Compare two sessions with --against. This is Specular's scriptable stand-in
for Xirp's per-session Git changes panel.

Examples:
  specular session diff demo
  specular session diff demo --name-only
  specular session diff demo --patch
  specular session diff demo --base origin/main
  specular session diff demo --against demo-2 --stat
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
		base, _ := cmd.Flags().GetString("base")
		against, _ := cmd.Flags().GetString("against")
		stat, _ := cmd.Flags().GetBool("stat")
		nameOnly, _ := cmd.Flags().GetBool("name-only")
		patch, _ := cmd.Flags().GetBool("patch")
		jsonOut, _ := cmd.Flags().GetBool("json")

		res, diffErr := mgr.Diff(cmd.Context(), args[0], session.DiffOptions{
			Base:     base,
			Against:  against,
			Stat:     stat,
			NameOnly: nameOnly,
			Patch:    patch,
		})
		if diffErr != nil {
			return diffErr
		}
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		}
		if against != "" {
			fmt.Printf("# session %s .. %s\n", res.SessionID, res.AgainstID)
		} else {
			fmt.Printf("# session %s vs %s\n", res.SessionID, res.Base)
		}
		if strings.TrimSpace(res.Output) == "" {
			fmt.Println("(no changes)")
			return nil
		}
		fmt.Print(res.Output)
		if !strings.HasSuffix(res.Output, "\n") {
			fmt.Println()
		}
		return nil
	},
}

var sessionExecCmd = &cobra.Command{
	Use:   "exec <session-id> [--] <command>...",
	Short: "Run a command in a session worktree",
	Long: `Run a non-interactive command with cwd set to the session worktree.

This is the CI-native substitute for a per-session terminal (Xirp's PTY grid):
tests, lint, and hooks run in isolation without shell cd gymnastics.

  specular session exec auth -- go test ./...
  specular session exec auth -- make lint
  specular session exec --json auth -- git status --porcelain

The child process exit code is propagated (CI fails when tests fail).
Put session flags before the session id; use -- before commands that take flags.
`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := session.NewManager(cwd)
		if err != nil {
			return err
		}
		id := args[0]
		argv := args[1:]
		if len(argv) > 0 && argv[0] == "--" {
			argv = argv[1:]
		}
		if len(argv) == 0 {
			return fmt.Errorf("command is required after session id (e.g. session exec %s -- go test ./...)", id)
		}
		appendLog, _ := cmd.Flags().GetBool("log")
		jsonOut, _ := cmd.Flags().GetBool("json")

		res, execErr := mgr.Exec(cmd.Context(), id, argv, session.ExecOptions{AppendLog: appendLog})
		if execErr != nil {
			return execErr
		}
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if encErr := enc.Encode(res); encErr != nil {
				return encErr
			}
		}
		if res.ExitCode != 0 {
			os.Exit(res.ExitCode)
		}
		return nil
	},
}

var sessionCommitCmd = &cobra.Command{
	Use:   "commit <session-id>",
	Short: "Commit changes in a session worktree",
	Long: `Stage and commit changes in a session's isolated worktree.

Default message embeds session id, harness, and goal for provenance.
Use --all to include untracked files; without it only tracked modifications
are staged. Refuses while the session is still running unless --force.

  specular session commit auth
  specular session commit auth -m "Harden JWT validation"
  specular session commit auth --all
  specular session commit auth --json
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
		msg, _ := cmd.Flags().GetString("message")
		all, _ := cmd.Flags().GetBool("all")
		allowEmpty, _ := cmd.Flags().GetBool("allow-empty")
		force, _ := cmd.Flags().GetBool("force")
		jsonOut, _ := cmd.Flags().GetBool("json")

		res, commitErr := mgr.Commit(cmd.Context(), args[0], session.CommitOptions{
			Message:    msg,
			All:        all,
			AllowEmpty: allowEmpty,
			Force:      force,
		})
		if commitErr != nil {
			return commitErr
		}
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		}
		fmt.Printf("Committed session %s\n", res.SessionID)
		fmt.Printf("  SHA:     %s\n", res.SHA)
		if res.Branch != "" {
			fmt.Printf("  Branch:  %s\n", res.Branch)
		}
		fmt.Printf("  Message: %s\n", res.Message)
		return nil
	},
}

var sessionSyncCmd = &cobra.Command{
	Use:   "sync <session-id>",
	Short: "Rebase (or merge) a session worktree onto a base ref",
	Long: `Bring a session branch up to date with the repo base (default: main/master/HEAD).

Default strategy is rebase. Use --merge for a merge commit. Dirty trees are
refused unless --autostash. Conflicts abort the operation and report paths
(CI-scriptable non-zero exit).

With --fetch, runs git fetch first and defaults --onto to origin/<base>
so fleets sync onto the remote tip rather than a stale local main.

  specular session sync auth
  specular session sync auth --fetch
  specular session sync auth --fetch --remote upstream
  specular session sync auth --onto origin/main
  specular session sync auth --merge
  specular session sync auth --autostash --json
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
		onto, _ := cmd.Flags().GetString("onto")
		merge, _ := cmd.Flags().GetBool("merge")
		autostash, _ := cmd.Flags().GetBool("autostash")
		force, _ := cmd.Flags().GetBool("force")
		fetch, _ := cmd.Flags().GetBool("fetch")
		remote, _ := cmd.Flags().GetString("remote")
		jsonOut, _ := cmd.Flags().GetBool("json")

		res, syncErr := mgr.Sync(cmd.Context(), args[0], session.SyncOptions{
			Onto:      onto,
			Merge:     merge,
			Autostash: autostash,
			Force:     force,
			Fetch:     fetch,
			Remote:    remote,
		})
		if jsonOut && res != nil {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(res)
		}
		if syncErr != nil {
			if !jsonOut && res != nil && len(res.Conflicts) > 0 {
				fmt.Fprintf(os.Stderr, "Conflicts:\n")
				for _, p := range res.Conflicts {
					fmt.Fprintf(os.Stderr, "  %s\n", p)
				}
			}
			return syncErr
		}
		if jsonOut {
			return nil
		}
		fmt.Printf("Synced session %s\n", res.SessionID)
		fmt.Printf("  Strategy: %s onto %s\n", res.Strategy, res.Onto)
		if res.Fetched {
			fmt.Printf("  Fetched:  %s\n", res.Remote)
		}
		fmt.Printf("  Before:   %s\n", res.BeforeSHA)
		fmt.Printf("  After:    %s\n", res.AfterSHA)
		if res.Stashed {
			fmt.Printf("  Stash:    popped\n")
		}
		return nil
	},
}

var sessionPushCmd = &cobra.Command{
	Use:   "push <session-id>",
	Short: "Push a session worktree branch (optional --pr)",
	Long: `Publish the session branch to the remote and optionally open a pull request.

Uses git push -u to the session worktree branch. With --pr, runs gh pr create
with harness/goal provenance in the body (requires gh on PATH).

  specular session push auth
  specular session push auth --pr
  specular session push auth --pr --base main --title "Harden JWT"
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
		remote, _ := cmd.Flags().GetString("remote")
		force, _ := cmd.Flags().GetBool("force")
		createPR, _ := cmd.Flags().GetBool("pr")
		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")
		base, _ := cmd.Flags().GetString("base")
		jsonOut, _ := cmd.Flags().GetBool("json")

		res, pushErr := mgr.Push(cmd.Context(), args[0], session.PushOptions{
			Remote:   remote,
			Force:    force,
			CreatePR: createPR,
			PRTitle:  title,
			PRBody:   body,
			Base:     base,
		})
		if jsonOut && res != nil {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(res)
		}
		if pushErr != nil {
			return pushErr
		}
		if jsonOut {
			return nil
		}
		fmt.Printf("Pushed session %s\n", res.SessionID)
		fmt.Printf("  Remote: %s\n", res.Remote)
		fmt.Printf("  Branch: %s\n", res.Branch)
		fmt.Printf("  SHA:    %s\n", res.SHA)
		if res.PRURL != "" {
			fmt.Printf("  PR:     %s\n", res.PRURL)
		}
		return nil
	},
}

var sessionMergeCmd = &cobra.Command{
	Use:   "merge <session-id>",
	Short: "Merge a session worktree branch into a base branch",
	Long: `Land a session branch into the primary checkout (local merge, no gh).

Checks out --into (default main/master) at the repo root and merges the
session worktree branch with a provenance-aware message. Requires a clean
primary working tree. On conflict, aborts and reports conflicted paths.

  specular session merge auth
  specular session merge auth --into main --no-ff
  specular session merge auth --ff-only
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
		into, _ := cmd.Flags().GetString("into")
		message, _ := cmd.Flags().GetString("message")
		ffOnly, _ := cmd.Flags().GetBool("ff-only")
		noFF, _ := cmd.Flags().GetBool("no-ff")
		force, _ := cmd.Flags().GetBool("force")
		jsonOut, _ := cmd.Flags().GetBool("json")
		if ffOnly && noFF {
			return fmt.Errorf("session: --ff-only and --no-ff are mutually exclusive")
		}

		res, mergeErr := mgr.Merge(cmd.Context(), args[0], session.MergeOptions{
			Into:    into,
			Message: message,
			FFOnly:  ffOnly,
			NoFF:    noFF,
			Force:   force,
		})
		if jsonOut && res != nil {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(res)
		}
		if mergeErr != nil {
			return mergeErr
		}
		if jsonOut {
			return nil
		}
		fmt.Printf("Merged session %s\n", res.SessionID)
		fmt.Printf("  Branch:   %s\n", res.Branch)
		fmt.Printf("  Into:     %s\n", res.Into)
		fmt.Printf("  Strategy: %s\n", res.Strategy)
		fmt.Printf("  Before:   %s\n", res.BeforeSHA)
		fmt.Printf("  After:    %s\n", res.AfterSHA)
		return nil
	},
}

var sessionCherryPickCmd = &cobra.Command{
	Use:   "cherry-pick <session-id>",
	Short: "Apply a commit from another session into this worktree",
	Long: `Cherry-pick a source session's HEAD (or --sha) into the target worktree.

Useful for dependsOn handoff: review session pulls the implement tip without
a full merge. On conflict, aborts and reports paths.

  specular session cherry-pick review --from auth
  specular session cherry-pick review --from auth --sha abc1234
  specular session cherry-pick review --from auth --no-commit
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
		from, _ := cmd.Flags().GetString("from")
		sha, _ := cmd.Flags().GetString("sha")
		noCommit, _ := cmd.Flags().GetBool("no-commit")
		force, _ := cmd.Flags().GetBool("force")
		jsonOut, _ := cmd.Flags().GetBool("json")
		if strings.TrimSpace(from) == "" {
			return fmt.Errorf("session: --from <session-id> is required")
		}

		res, pickErr := mgr.CherryPick(cmd.Context(), args[0], session.CherryPickOptions{
			From:     from,
			SHA:      sha,
			NoCommit: noCommit,
			Force:    force,
		})
		if jsonOut && res != nil {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(res)
		}
		if pickErr != nil {
			if !jsonOut && res != nil && len(res.Conflicts) > 0 {
				fmt.Fprintf(os.Stderr, "Conflicts:\n")
				for _, p := range res.Conflicts {
					fmt.Fprintf(os.Stderr, "  %s\n", p)
				}
			}
			return pickErr
		}
		if jsonOut {
			return nil
		}
		fmt.Printf("Cherry-picked into session %s\n", res.SessionID)
		fmt.Printf("  From:   %s\n", res.FromSessionID)
		fmt.Printf("  Commit: %s\n", res.Commit)
		fmt.Printf("  Before: %s\n", res.BeforeSHA)
		fmt.Printf("  After:  %s\n", res.AfterSHA)
		if res.NoCommit {
			fmt.Printf("  Note:   --no-commit (changes staged/applied only)\n")
		}
		return nil
	},
}

var sessionAttestCmd = &cobra.Command{
	Use:   "attest <session-id>",
	Short: "Write a signed attestation with harness/worktree provenance",
	Long: `Emit outer-loop evidence for a managed session — including native
Claude Code, Codex, and Gemini runs.

Writes .specular/sessions/<id>.attestation.json and a sibling
.specular/sessions/<id>.provenance.json (Agent Provenance Protocol v1).
Signatures: specular auto verify. Schema: specular provenance verify.

  specular session attest auth
  specular session wait --attest auth ratelimit
  specular auto verify .specular/sessions/auth.attestation.json
  specular provenance verify auth
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
		out, _ := cmd.Flags().GetString("output")
		force, _ := cmd.Flags().GetBool("force")
		jsonOut, _ := cmd.Flags().GetBool("json")
		res, attestErr := mgr.Attest(cmd.Context(), args[0], session.AttestOptions{
			OutputPath: out,
			Force:      force,
		})
		if attestErr != nil {
			return attestErr
		}
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		}
		fmt.Printf("Attested session %s\n", res.SessionID)
		fmt.Printf("  Harness:    %s\n", res.Harness)
		if res.Governed {
			fmt.Printf("  Governed:   true\n")
		}
		fmt.Printf("  Status:     %s\n", res.Status)
		fmt.Printf("  Attest:     %s\n", res.Path)
		if res.ProvenancePath != "" {
			fmt.Printf("  Provenance: %s\n", res.ProvenancePath)
		}
		fmt.Printf("\n  specular auto verify %s\n", res.Path)
		if res.ProvenancePath != "" {
			fmt.Printf("  specular provenance verify %s\n", res.SessionID)
		}
		return nil
	},
}

var sessionIntegrateCmd = &cobra.Command{
	Use:   "integrate <harness>",
	Short: "Install native agent hooks for session attest + gate",
	Long: `Write a minimal native coding-agent hook into the repo so Stop/completion
calls existing Specular surfaces — session attest (harness provenance) and gate.

This is the PRODUCT_INTENT P1 #4 starter: Level-2 integrated provenance without
inventing a new protocol. Pass --enforce for Level-3 fail-closed Stop hooks
(attest + gate --require-attested --require-protocol; non-zero on DENY).
Add --require-governed with --enforce to also pass --require-governed (safer
native launch required). Prefer managed sessions:

  specular session integrate claude-code
  specular session integrate claude-code --dry-run
  specular session integrate claude-code --enforce --force
  specular session integrate claude-code --enforce --require-governed --force
  specular session integrate cursor
  specular session integrate cursor --dry-run
  specular session integrate codex
  specular session integrate gemini
  specular session start --harness claude-code --governed "Harden JWT validation"

Supported harnesses: claude-code (alias: claude), cursor (alias: cursor-agent),
codex (alias: codex-cli), gemini (alias: gemini-cli).

Installs (claude-code):
  .claude/hooks/specular-session-stop.sh
  .claude/settings.json  (merges hooks.Stop; preserves other settings)

Installs (cursor):
  .cursor/hooks/specular-session-stop.sh
  .cursor/hooks.json  (merges hooks.stop; preserves other hooks / version)

Installs (codex):
  .codex/hooks/specular-session-stop.sh
  .codex/hooks.json  (merges hooks.Stop; preserves other hooks)

Installs (gemini):
  .gemini/hooks/specular-session-stop.sh
  .gemini/settings.json  (merges hooks.SessionEnd + hooksConfig.enabled)

session start exports SPECULAR_SESSION_ID and SPECULAR_SESSION_HARNESS so the
Stop/SessionEnd hook can attest the right record.
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")
		enforce, _ := cmd.Flags().GetBool("enforce")
		requireGoverned, _ := cmd.Flags().GetBool("require-governed")
		jsonOut, _ := cmd.Flags().GetBool("json")
		res, integrateErr := session.Integrate(session.IntegrateOptions{
			Harness:         args[0],
			Root:            cwd,
			DryRun:          dryRun,
			Force:           force,
			Enforce:         enforce,
			RequireGoverned: requireGoverned,
		})
		if integrateErr != nil {
			return integrateErr
		}
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		}
		mode := "Installed"
		if res.DryRun {
			mode = "Dry-run"
		}
		fmt.Printf("%s native hooks for harness %s\n", mode, res.Harness)
		for _, f := range res.Files {
			fmt.Printf("  [%s] %s\n", f.Action, f.Path)
			if res.DryRun && f.Content != "" {
				fmt.Printf("---- %s ----\n%s", f.Path, f.Content)
				if !strings.HasSuffix(f.Content, "\n") {
					fmt.Println()
				}
			}
		}
		if len(res.NextSteps) > 0 && !res.DryRun {
			fmt.Println("\nNext:")
			for _, step := range res.NextSteps {
				fmt.Printf("  • %s\n", step)
			}
		}
		return nil
	},
}

func attestSessionRecords(ctx context.Context, mgr *session.Manager, recs []session.Record, print bool) error {
	var first error
	for _, rec := range recs {
		if rec.Status != session.StatusCompleted && rec.Status != session.StatusFailed && rec.Status != session.StatusStopped {
			continue
		}
		res, err := mgr.Attest(ctx, rec.ID, session.AttestOptions{})
		if err != nil {
			if first == nil {
				first = err
			}
			if print {
				fmt.Fprintf(os.Stderr, "attest %s: %v\n", rec.ID, err)
			}
			continue
		}
		if print {
			fmt.Printf("Attested session %s → %s\n", res.SessionID, res.Path)
		}
	}
	return first
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
	sessionStartCmd.Flags().Bool("governed", false, "Safer native launch (no skip-permissions/full-auto) + governance preamble")
	sessionStartCmd.Flags().Bool("no-governed", false, "Disable auto-governed even when .specular/policy.yaml is present")
	sessionStartCmd.Flags().Bool("json", false, "Emit JSON")
	sessionStartCmd.Flags().String("manifest", "", "Start a fleet from a YAML/JSON manifest file")

	sessionBatchCmd.Flags().String("harness", "", "Default harness when an entry omits harness")
	sessionBatchCmd.Flags().String("profile", "", "Default profile when an entry omits profile")
	sessionBatchCmd.Flags().Bool("no-worktree", false, "Run all entries in the current checkout")
	sessionBatchCmd.Flags().Bool("governed", false, "Default governed=true for native harness entries")
	sessionBatchCmd.Flags().Bool("no-governed", false, "Disable auto-governed even when a policy file is present")
	sessionBatchCmd.Flags().Bool("json", false, "Emit JSON")

	sessionListCmd.Flags().Bool("checkpoints", false, "Also list legacy auto checkpoints")
	sessionListCmd.Flags().Bool("json", false, "Emit JSON")
	addSessionBoardFilterFlags(sessionListCmd)

	sessionShowCmd.Flags().BoolP("verbose", "v", false, "Show log tail / task details")
	sessionShowCmd.Flags().Bool("json", false, "Emit JSON")

	sessionStopCmd.Flags().Bool("all", false, "Stop every non-terminal session")
	sessionStopCmd.Flags().Bool("json", false, "Emit JSON")

	sessionLogsCmd.Flags().Bool("follow", false, "Follow log output")
	sessionForkCmd.Flags().String("name", "", "Name for the forked session")
	sessionForkCmd.Flags().Bool("start", false, "Start the forked session immediately")
	sessionForkCmd.Flags().Bool("json", false, "Emit JSON")

	sessionHarnessesCmd.Flags().Bool("json", false, "Emit JSON")
	sessionStatusCmd.Flags().Bool("watch", false, "Refresh the status board until interrupted")
	sessionStatusCmd.Flags().Duration("interval", 2*time.Second, "Refresh interval for --watch")
	sessionStatusCmd.Flags().Bool("json", false, "Emit JSON")
	addSessionBoardFilterFlags(sessionStatusCmd)
	sessionOpenCmd.Flags().Bool("shell", false, "Print a cd command instead of the bare path")
	sessionOpenCmd.Flags().Bool("editor", false, "Open the worktree in $EDITOR")

	sessionWaitCmd.Flags().Duration("timeout", 0, "Maximum time to wait (0 = no limit)")
	sessionWaitCmd.Flags().Duration("interval", 500*time.Millisecond, "Poll interval")
	sessionWaitCmd.Flags().Bool("any", false, "Return when any named session finishes")
	sessionWaitCmd.Flags().Bool("stop", false, "Stop still-running sessions when --timeout fires")
	sessionWaitCmd.Flags().Bool("attest", false, "Write session attestations after wait succeeds")
	sessionWaitCmd.Flags().Bool("gate", false, "Run product specular gate after wait (provenance/drift/policy; fail-on-DENY)")
	sessionWaitCmd.Flags().Bool("bundle", false, "Package attestations + APP docs + evidence graph + drift (+ policies) into a bundle (implies --gate)")
	sessionWaitCmd.Flags().String("bundle-out", "session-evidence.sbundle.tgz", "Output path for --bundle")
	sessionWaitCmd.Flags().StringSlice("policy", nil, "Policy files to include when using --bundle")
	sessionWaitCmd.Flags().Bool("require-attested", false, "With --gate/--bundle: DENY when unattested (mirrors gate --require-attested)")
	sessionWaitCmd.Flags().Bool("require-protocol", false, "With --gate/--bundle: DENY when APP docs missing/invalid/unbound")
	sessionWaitCmd.Flags().Bool("require-governed", false, "With --gate/--bundle: DENY when no governed session")
	sessionWaitCmd.Flags().Bool("json", false, "Emit JSON")
	addSessionBoardFilterFlags(sessionWaitCmd)

	sessionRestartCmd.Flags().String("harness", "", "Switch harness on restart")
	sessionRestartCmd.Flags().String("goal", "", "Override goal on restart")
	sessionRestartCmd.Flags().String("profile", "", "Override auto profile on restart")
	sessionRestartCmd.Flags().Bool("force", false, "Stop a still-running session before restart")
	sessionRestartCmd.Flags().Bool("foreground", false, "Run in the foreground instead of detaching")
	sessionRestartCmd.Flags().Bool("governed", false, "Safer native launch on restart (omit to keep prior setting)")
	sessionRestartCmd.Flags().Bool("no-governed", false, "Disable auto-governed on restart even when a policy file is present")
	sessionRestartCmd.Flags().Bool("json", false, "Emit JSON")

	sessionRmCmd.Flags().Bool("force", false, "Stop a still-running session before removal")
	sessionRmCmd.Flags().Bool("keep-worktree", false, "Leave the Git worktree in place")
	sessionRmCmd.Flags().Bool("delete-branch", false, "Also delete the managed worktree branch")
	sessionRmCmd.Flags().Bool("json", false, "Emit JSON")

	sessionPruneCmd.Flags().Duration("older-than", 0, "Only prune sessions older than this duration (0 = all finished)")
	sessionPruneCmd.Flags().Bool("keep-worktree", false, "Leave Git worktrees in place")
	sessionPruneCmd.Flags().Bool("delete-branch", false, "Also delete managed worktree branches")
	sessionPruneCmd.Flags().Bool("json", false, "Emit JSON")

	sessionDiffCmd.Flags().String("base", "", "Base ref to compare against (default: main/master/HEAD)")
	sessionDiffCmd.Flags().String("against", "", "Compare against another session's HEAD")
	sessionDiffCmd.Flags().Bool("stat", false, "Show --stat summary (default when neither --name-only nor --patch)")
	sessionDiffCmd.Flags().Bool("name-only", false, "List changed file paths only")
	sessionDiffCmd.Flags().Bool("patch", false, "Show full unified diff")
	sessionDiffCmd.Flags().Bool("json", false, "Emit JSON")

	sessionExecCmd.Flags().SetInterspersed(false)
	sessionExecCmd.Flags().Bool("log", false, "Append command output to the session log file")
	sessionExecCmd.Flags().Bool("json", false, "Emit JSON result (after command stdout/stderr)")

	sessionCommitCmd.Flags().StringP("message", "m", "", "Commit message (default: provenance-aware)")
	sessionCommitCmd.Flags().Bool("all", false, "Stage untracked files too (git add -A)")
	sessionCommitCmd.Flags().Bool("allow-empty", false, "Allow an empty commit")
	sessionCommitCmd.Flags().Bool("force", false, "Commit even if the session is still running")
	sessionCommitCmd.Flags().Bool("json", false, "Emit JSON")

	sessionSyncCmd.Flags().String("onto", "", "Base ref to sync onto (default: main/master/HEAD, or origin/<base> with --fetch)")
	sessionSyncCmd.Flags().Bool("merge", false, "Merge instead of rebase")
	sessionSyncCmd.Flags().Bool("autostash", false, "Stash dirty changes before sync and pop after")
	sessionSyncCmd.Flags().Bool("fetch", false, "git fetch before sync; default onto becomes origin/<base>")
	sessionSyncCmd.Flags().String("remote", "origin", "Remote to fetch when using --fetch")
	sessionSyncCmd.Flags().Bool("force", false, "Sync even if the session is still running")
	sessionSyncCmd.Flags().Bool("json", false, "Emit JSON")

	sessionPushCmd.Flags().String("remote", "origin", "Git remote to push to")
	sessionPushCmd.Flags().Bool("pr", false, "Open a pull request with gh after push")
	sessionPushCmd.Flags().String("title", "", "PR title (default: session id + goal)")
	sessionPushCmd.Flags().String("body", "", "PR body (default: harness/goal provenance)")
	sessionPushCmd.Flags().String("base", "", "PR base branch for gh --base")
	sessionPushCmd.Flags().Bool("force", false, "Push even if the session is still running")
	sessionPushCmd.Flags().Bool("json", false, "Emit JSON")

	sessionMergeCmd.Flags().String("into", "", "Target branch (default: main/master/HEAD)")
	sessionMergeCmd.Flags().StringP("message", "m", "", "Merge commit message (default: provenance-aware)")
	sessionMergeCmd.Flags().Bool("ff-only", false, "Require a fast-forward merge")
	sessionMergeCmd.Flags().Bool("no-ff", false, "Always create a merge commit")
	sessionMergeCmd.Flags().Bool("force", false, "Merge even if the session is still running")
	sessionMergeCmd.Flags().Bool("json", false, "Emit JSON")

	sessionCherryPickCmd.Flags().String("from", "", "Source session ID to cherry-pick from (required)")
	sessionCherryPickCmd.Flags().String("sha", "", "Commit SHA to pick (default: source session HEAD)")
	sessionCherryPickCmd.Flags().Bool("no-commit", false, "Apply without creating a commit")
	sessionCherryPickCmd.Flags().Bool("force", false, "Cherry-pick even if the target session is still running")
	sessionCherryPickCmd.Flags().Bool("json", false, "Emit JSON")

	sessionAttestCmd.Flags().String("output", "", "Attestation output path (default: .specular/sessions/<id>.attestation.json)")
	sessionAttestCmd.Flags().Bool("force", false, "Attest even if the session is still running")
	sessionAttestCmd.Flags().Bool("json", false, "Emit JSON")

	sessionIntegrateCmd.Flags().Bool("dry-run", false, "Print planned hook/config writes without changing the repo")
	sessionIntegrateCmd.Flags().Bool("force", false, "Overwrite an existing Specular Stop hook script")
	sessionIntegrateCmd.Flags().Bool("enforce", false, "Install fail-closed Stop hooks (attest + gate --require-attested/--require-protocol)")
	sessionIntegrateCmd.Flags().Bool("require-governed", false, "With --enforce, also pass gate --require-governed (requires governed session)")
	sessionIntegrateCmd.Flags().Bool("json", false, "Emit JSON")

	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionBatchCmd)
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
	sessionCmd.AddCommand(sessionRmCmd)
	sessionCmd.AddCommand(sessionPruneCmd)
	sessionCmd.AddCommand(sessionDiffCmd)
	sessionCmd.AddCommand(sessionExecCmd)
	sessionCmd.AddCommand(sessionCommitCmd)
	sessionCmd.AddCommand(sessionSyncCmd)
	sessionCmd.AddCommand(sessionPushCmd)
	sessionCmd.AddCommand(sessionMergeCmd)
	sessionCmd.AddCommand(sessionCherryPickCmd)
	sessionCmd.AddCommand(sessionAttestCmd)
	sessionCmd.AddCommand(sessionIntegrateCmd)
	rootCmd.AddCommand(sessionCmd)
}
