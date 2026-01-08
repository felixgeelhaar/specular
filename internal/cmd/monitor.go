package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor [session-id]",
	Short: "Monitor and attach to running auto sessions",
	Long: `Monitor running Specular auto sessions or attach to view real-time progress.

Without arguments, lists all available sessions. With a session-id, attaches to
that session to view progress in real-time.

The monitor command provides visibility into:
  • Running sessions and their current status
  • Session history and completion status
  • Real-time progress when attached to active sessions

Use --follow to stream live updates as the session progresses.

Examples:
  specular monitor                           # List all sessions
  specular monitor auto-1762811730           # Attach to specific session
  specular monitor --latest                  # Attach to most recent session
  specular monitor --active                  # List only active sessions
  specular monitor --follow auto-1762811730  # Stream live updates
  specular monitor --follow --latest         # Follow most recent session
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse flags
		latest, _ := cmd.Flags().GetBool("latest")
		activeOnly, _ := cmd.Flags().GetBool("active")
		verbose, _ := cmd.Flags().GetBool("verbose")
		follow, _ := cmd.Flags().GetBool("follow")

		// Get checkpoint directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		checkpointDir := filepath.Join(homeDir, ".specular", "checkpoints")

		// Create checkpoint manager
		mgr := checkpoint.NewManager(checkpointDir, false, 0)

		// Handle --follow mode
		if follow {
			var sessionID string
			if latest {
				sessionID, err = findLatestSession(mgr)
				if err != nil {
					return err
				}
			} else if len(args) > 0 {
				sessionID = args[0]
			} else {
				return fmt.Errorf("--follow requires a session-id or --latest flag")
			}
			return streamSession(mgr, checkpointDir, sessionID, verbose)
		}

		// If --latest flag, find and attach to most recent session
		if latest {
			return attachToLatest(mgr, verbose)
		}

		// If session-id provided, attach to that session
		if len(args) > 0 {
			return attachToSession(mgr, args[0], verbose)
		}

		// Otherwise, list sessions
		return listSessions(mgr, activeOnly, verbose)
	},
}

func init() {
	monitorCmd.Flags().Bool("latest", false, "Attach to the most recent session")
	monitorCmd.Flags().Bool("active", false, "Show only active (running) sessions")
	monitorCmd.Flags().BoolP("verbose", "v", false, "Show verbose session details")
	monitorCmd.Flags().BoolP("follow", "f", false, "Stream live updates (like tail -f)")

	rootCmd.AddCommand(monitorCmd)
}

// listSessions lists all available sessions
func listSessions(mgr *checkpoint.Manager, activeOnly, verbose bool) error {
	sessions, err := mgr.List()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found")
		fmt.Println()
		fmt.Println("Start a new session with:")
		fmt.Println("  specular auto \"your goal here\"")
		return nil
	}

	// Load session states for sorting and filtering
	type sessionInfo struct {
		id        string
		state     *checkpoint.State
		updatedAt time.Time
		isActive  bool
	}

	var sessionInfos []sessionInfo
	for _, id := range sessions {
		state, err := mgr.Load(id)
		if err != nil {
			continue
		}

		isActive := state.Status == "running" || state.Status == "pending"
		if activeOnly && !isActive {
			continue
		}

		sessionInfos = append(sessionInfos, sessionInfo{
			id:        id,
			state:     state,
			updatedAt: state.UpdatedAt,
			isActive:  isActive,
		})
	}

	// Sort by update time (most recent first)
	sort.Slice(sessionInfos, func(i, j int) bool {
		return sessionInfos[i].updatedAt.After(sessionInfos[j].updatedAt)
	})

	if len(sessionInfos) == 0 {
		if activeOnly {
			fmt.Println("No active sessions found")
		} else {
			fmt.Println("No sessions found")
		}
		return nil
	}

	// Print header
	if activeOnly {
		fmt.Println("Active Sessions:")
	} else {
		fmt.Println("All Sessions:")
	}
	fmt.Println()

	// Print sessions
	for _, info := range sessionInfos {
		printSessionInfo(info.id, info.state, verbose)
	}

	// Print help
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  specular monitor <session-id>  Attach to a session")
	fmt.Println("  specular monitor --latest      Attach to most recent session")
	fmt.Println("  specular monitor --active      Show only active sessions")

	return nil
}

// attachToLatest attaches to the most recent session
func attachToLatest(mgr *checkpoint.Manager, verbose bool) error {
	sessions, err := mgr.List()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		return fmt.Errorf("no sessions found")
	}

	// Find most recent session
	var latestID string
	var latestTime time.Time

	for _, id := range sessions {
		state, err := mgr.Load(id)
		if err != nil {
			continue
		}

		if latestID == "" || state.UpdatedAt.After(latestTime) {
			latestID = id
			latestTime = state.UpdatedAt
		}
	}

	if latestID == "" {
		return fmt.Errorf("no valid sessions found")
	}

	return attachToSession(mgr, latestID, verbose)
}

// attachToSession attaches to a specific session
func attachToSession(mgr *checkpoint.Manager, sessionID string, verbose bool) error {
	// Check if session exists
	if !mgr.Exists(sessionID) {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Load session state
	state, err := mgr.Load(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Print session header
	fmt.Printf("📺 Monitoring Session: %s\n", sessionID)
	fmt.Println(repeatString("─", 60))
	printSessionInfo(sessionID, state, verbose)

	// Check if session is active
	isActive := state.Status == "running" || state.Status == "pending"

	if !isActive {
		fmt.Println()
		fmt.Printf("ℹ️  Session is %s (not running)\n", state.Status)
		fmt.Println()
		fmt.Println("To view session details:")
		fmt.Printf("  specular auto explain %s\n", sessionID)
		fmt.Println()
		fmt.Println("To resume a paused session:")
		fmt.Printf("  specular auto --resume %s \"continue goal\"\n", sessionID)
		return nil
	}

	// For active sessions, show live monitoring message
	fmt.Println()
	fmt.Println("🔄 Session is active")
	fmt.Println()
	fmt.Println("Live monitoring is best experienced with TUI mode:")
	fmt.Printf("  specular auto --tui --resume %s \"continue goal\"\n", sessionID)
	fmt.Println()

	// Show current task status
	fmt.Println("Current Task Status:")
	printTaskStatus(state)

	return nil
}

// printSessionInfo prints formatted session information
func printSessionInfo(id string, state *checkpoint.State, verbose bool) {
	// Status icon
	statusIcon := "⏸️"
	switch state.Status {
	case "completed":
		statusIcon = "✅"
	case "failed":
		statusIcon = "❌"
	case "running":
		statusIcon = "🔄"
	case "pending":
		statusIcon = "⏳"
	}

	// Print basic info
	fmt.Printf("  %s %s\n", statusIcon, id)
	fmt.Printf("     Status:  %s\n", state.Status)
	fmt.Printf("     Started: %s\n", state.StartedAt.Format("2006-01-02 15:04:05"))

	// Show elapsed time for running sessions
	if state.Status == "running" {
		elapsed := time.Since(state.StartedAt)
		fmt.Printf("     Elapsed: %s\n", formatMonitorDuration(elapsed))
	} else {
		fmt.Printf("     Updated: %s\n", state.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	// Goal from metadata
	if goal, ok := state.Metadata["goal"]; ok {
		goalStr := fmt.Sprintf("%v", goal)
		if len(goalStr) > 50 {
			goalStr = goalStr[:47] + "..."
		}
		fmt.Printf("     Goal:    %s\n", goalStr)
	}

	// Task summary
	var pending, running, completed, failed int
	for _, task := range state.Tasks {
		switch task.Status {
		case "pending":
			pending++
		case "running":
			running++
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	total := len(state.Tasks)
	fmt.Printf("     Tasks:   %d/%d completed", completed, total)
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	if running > 0 {
		fmt.Printf(", %d running", running)
	}
	fmt.Println()

	// Verbose output
	if verbose {
		fmt.Println()
		fmt.Println("     Task Details:")
		for taskID, task := range state.Tasks {
			taskIcon := "○"
			switch task.Status {
			case "completed":
				taskIcon = "✓"
			case "failed":
				taskIcon = "✗"
			case "running":
				taskIcon = "⟳"
			}
			fmt.Printf("       %s %s - %s\n", taskIcon, taskID, task.Status)
			if task.Error != "" {
				fmt.Printf("         Error: %s\n", task.Error)
			}
		}
	}

	fmt.Println()
}

// printTaskStatus prints the current task status for active sessions
func printTaskStatus(state *checkpoint.State) {
	for taskID, task := range state.Tasks {
		if task.Status == "running" {
			fmt.Printf("  🔄 Running: %s\n", taskID)
			if task.Attempts > 1 {
				fmt.Printf("     Attempt: %d\n", task.Attempts)
			}
			if !task.StartedAt.IsZero() {
				elapsed := time.Since(task.StartedAt)
				fmt.Printf("     Elapsed: %s\n", formatMonitorDuration(elapsed))
			}
		}
	}

	// Show next pending tasks
	var pendingTasks []string
	for taskID, task := range state.Tasks {
		if task.Status == "pending" {
			pendingTasks = append(pendingTasks, taskID)
		}
	}

	if len(pendingTasks) > 0 {
		fmt.Println()
		fmt.Println("  Pending:")
		for i, taskID := range pendingTasks {
			if i >= 3 {
				fmt.Printf("    ... and %d more\n", len(pendingTasks)-3)
				break
			}
			fmt.Printf("    ○ %s\n", taskID)
		}
	}
}

// formatDuration formats a duration for display (local copy to avoid import cycles)
func formatMonitorDuration(d time.Duration) string {
	d = d.Round(time.Second)

	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// repeatString repeats a string n times
func repeatString(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// findLatestSession finds the most recent session ID
func findLatestSession(mgr *checkpoint.Manager) (string, error) {
	sessions, err := mgr.List()
	if err != nil {
		return "", fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions found")
	}

	var latestID string
	var latestTime time.Time

	for _, id := range sessions {
		state, err := mgr.Load(id)
		if err != nil {
			continue
		}

		if latestID == "" || state.UpdatedAt.After(latestTime) {
			latestID = id
			latestTime = state.UpdatedAt
		}
	}

	if latestID == "" {
		return "", fmt.Errorf("no valid sessions found")
	}

	return latestID, nil
}

// streamSession watches a session's checkpoint file and streams updates
func streamSession(mgr *checkpoint.Manager, checkpointDir, sessionID string, verbose bool) error {
	// Verify session exists
	if !mgr.Exists(sessionID) {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the checkpoint directory (not just the file, as file writes may recreate it)
	if err := watcher.Add(checkpointDir); err != nil {
		return fmt.Errorf("failed to watch checkpoint directory: %w", err)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Print initial state
	fmt.Printf("📺 Following Session: %s\n", sessionID)
	fmt.Println(repeatString("─", 60))
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Print current state
	state, err := mgr.Load(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}
	printStreamUpdate(state, verbose)

	// Track last update time to avoid duplicate prints
	lastUpdate := state.UpdatedAt

	// Watch for changes
	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			fmt.Println("👋 Stopped following session")
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only react to writes on our checkpoint file
			if event.Op&fsnotify.Write == fsnotify.Write {
				if filepath.Base(event.Name) == fmt.Sprintf("%s.json", sessionID) {
					// Small delay to ensure file is fully written
					time.Sleep(50 * time.Millisecond)

					newState, err := mgr.Load(sessionID)
					if err != nil {
						continue // Ignore transient read errors
					}

					// Only print if actually updated
					if newState.UpdatedAt.After(lastUpdate) {
						clearTerminal()
						fmt.Printf("📺 Following Session: %s\n", sessionID)
						fmt.Println(repeatString("─", 60))
						fmt.Println("Press Ctrl+C to stop")
						fmt.Println()
						printStreamUpdate(newState, verbose)
						lastUpdate = newState.UpdatedAt

						// Check if session completed
						if newState.Status == "completed" || newState.Status == "failed" {
							fmt.Println()
							if newState.Status == "completed" {
								fmt.Println("✅ Session completed")
							} else {
								fmt.Println("❌ Session failed")
							}
							return nil
						}
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			// Log but don't fail on watcher errors
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}

// printStreamUpdate prints a condensed session update for streaming mode
func printStreamUpdate(state *checkpoint.State, verbose bool) {
	// Status
	statusIcon := "⏸️"
	switch state.Status {
	case "completed":
		statusIcon = "✅"
	case "failed":
		statusIcon = "❌"
	case "running":
		statusIcon = "🔄"
	case "pending":
		statusIcon = "⏳"
	}

	fmt.Printf("%s Status: %s\n", statusIcon, state.Status)

	// Elapsed time
	elapsed := time.Since(state.StartedAt)
	fmt.Printf("⏱️  Elapsed: %s\n", formatMonitorDuration(elapsed))
	fmt.Printf("📝 Updated: %s\n", state.UpdatedAt.Format("15:04:05"))

	// Goal
	if goal, ok := state.Metadata["goal"]; ok {
		goalStr := fmt.Sprintf("%v", goal)
		if len(goalStr) > 60 {
			goalStr = goalStr[:57] + "..."
		}
		fmt.Printf("🎯 Goal: %s\n", goalStr)
	}

	fmt.Println()

	// Task summary
	var pending, running, completed, failed int
	var currentTask string
	for taskID, task := range state.Tasks {
		switch task.Status {
		case "pending":
			pending++
		case "running":
			running++
			currentTask = taskID
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	total := len(state.Tasks)
	fmt.Printf("📊 Progress: %d/%d tasks completed\n", completed, total)

	// Progress bar
	if total > 0 {
		progress := float64(completed) / float64(total)
		barWidth := 30
		filledWidth := int(progress * float64(barWidth))
		bar := ""
		for i := 0; i < barWidth; i++ {
			if i < filledWidth {
				bar += "█"
			} else {
				bar += "░"
			}
		}
		fmt.Printf("   [%s] %.0f%%\n", bar, progress*100)
	}

	// Current task
	if currentTask != "" {
		fmt.Printf("\n🔄 Current: %s\n", currentTask)
		if task, ok := state.Tasks[currentTask]; ok && !task.StartedAt.IsZero() {
			taskElapsed := time.Since(task.StartedAt)
			fmt.Printf("   Running for: %s\n", formatMonitorDuration(taskElapsed))
		}
	}

	// Failed tasks
	if failed > 0 {
		fmt.Printf("\n❌ Failed: %d task(s)\n", failed)
		if verbose {
			for taskID, task := range state.Tasks {
				if task.Status == "failed" {
					fmt.Printf("   • %s: %s\n", taskID, task.Error)
				}
			}
		}
	}

	// Verbose: show all tasks
	if verbose && total > 0 {
		fmt.Println("\n📋 All Tasks:")
		for taskID, task := range state.Tasks {
			icon := "○"
			switch task.Status {
			case "completed":
				icon = "✓"
			case "failed":
				icon = "✗"
			case "running":
				icon = "⟳"
			}
			fmt.Printf("   %s %s - %s\n", icon, taskID, task.Status)
		}
	}
}

// clearTerminal clears the terminal screen
func clearTerminal() {
	// ANSI escape code to clear screen and move cursor to top-left
	fmt.Print("\033[2J\033[H")
}
