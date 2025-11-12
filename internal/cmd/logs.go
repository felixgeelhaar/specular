package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/ux"
)

var (
	logsTail   bool
	logsTrace  string
	logsFollow bool
	logsLines  int
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show or tail CLI logs",
	Long: `View Specular CLI logs and trace events.

Logs are stored in ~/.specular/logs/ with each workflow execution
getting its own trace file.

Examples:
  # Show recent logs
  specular logs

  # Show last 50 log entries
  specular logs --lines 50

  # Show a specific trace log
  specular logs --trace <trace-id>

  # Follow logs in real-time
  specular logs --follow

  # List all available trace logs
  specular logs --list
`,
	RunE: runLogs,
}

var logsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available trace logs",
	Long:  `Display a list of all trace logs stored in ~/.specular/logs/`,
	RunE:  runLogsList,
}

func init() {
	logsCmd.Flags().BoolVar(&logsTail, "tail", false, "Show only recent log entries")
	logsCmd.Flags().StringVar(&logsTrace, "trace", "", "Show specific trace log by ID")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output in real-time")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 20, "Number of recent lines to show")

	logsCmd.AddCommand(logsListCmd)
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	// Get log directory
	logDir := getLogDirectory()

	// Check if log directory exists
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return fmt.Errorf("no logs found at %s", logDir)
	}

	// If specific trace requested, show that trace
	if logsTrace != "" {
		return showTraceLog(cmdCtx, logDir, logsTrace)
	}

	// If follow mode, tail the latest log file
	if logsFollow {
		return followLogs(logDir)
	}

	// Show recent logs from latest file
	return showRecentLogs(cmdCtx, logDir, logsLines)
}

func runLogsList(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	logDir := getLogDirectory()

	// Get all trace files
	traces, err := getTraceFiles(logDir)
	if err != nil {
		return ux.FormatError(err, "listing trace logs")
	}

	if len(traces) == 0 {
		fmt.Println("No trace logs found")
		return nil
	}

	// For JSON/YAML output
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(traces)
	}

	// Text output
	fmt.Printf("Trace logs in %s:\n\n", logDir)
	for _, t := range traces {
		fmt.Printf("  %s  %s  (%s)\n",
			t.CreatedAt.Format("2006-01-02 15:04:05"),
			t.ID,
			formatFileSize(t.Size))
	}
	fmt.Printf("\nTotal: %d trace logs\n", len(traces))

	return nil
}

func getLogDirectory() string {
	// Match the default from trace.DefaultConfig()
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".specular", "logs")
}

// TraceFileInfo holds information about a trace log file
type TraceFileInfo struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

func getTraceFiles(logDir string) ([]TraceFileInfo, error) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	var traces []TraceFileInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only include trace_*.json files
		if !strings.HasPrefix(entry.Name(), "trace_") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Extract trace ID from filename
		id := strings.TrimPrefix(entry.Name(), "trace_")
		id = strings.TrimSuffix(id, ".json")

		path := filepath.Join(logDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		traces = append(traces, TraceFileInfo{
			ID:        id,
			Path:      path,
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	// Sort by creation time (newest first)
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].CreatedAt.After(traces[j].CreatedAt)
	})

	return traces, nil
}

func showRecentLogs(cmdCtx *CommandContext, logDir string, numLines int) error {
	traces, err := getTraceFiles(logDir)
	if err != nil {
		return err
	}

	if len(traces) == 0 {
		fmt.Println("No trace logs found")
		return nil
	}

	// Use the most recent trace file
	latestTrace := traces[0]

	fmt.Printf("Showing last %d lines from trace %s:\n\n", numLines, latestTrace.ID)

	return tailFile(latestTrace.Path, numLines)
}

func showTraceLog(cmdCtx *CommandContext, logDir string, traceID string) error {
	tracePath := filepath.Join(logDir, fmt.Sprintf("trace_%s.json", traceID))

	if _, err := os.Stat(tracePath); os.IsNotExist(err) {
		return fmt.Errorf("trace log not found: %s", traceID)
	}

	fmt.Printf("Trace log: %s\n\n", traceID)

	// Read and display the entire file
	file, err := os.Open(tracePath)
	if err != nil {
		return fmt.Errorf("failed to open trace log: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Try to pretty-print JSON
		if strings.TrimSpace(line) != "" {
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(line), &event); err == nil {
				prettyJSON, _ := json.MarshalIndent(event, "", "  ")
				fmt.Printf("[%d] %s\n", lineNum, prettyJSON)
			} else {
				fmt.Printf("[%d] %s\n", lineNum, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading trace log: %w", err)
	}

	return nil
}

func followLogs(logDir string) error {
	traces, err := getTraceFiles(logDir)
	if err != nil {
		return err
	}

	if len(traces) == 0 {
		fmt.Println("No trace logs found. Waiting for new logs...")
		// Could implement watching for new files here
		return nil
	}

	latestTrace := traces[0]

	fmt.Printf("Following trace %s (Ctrl+C to stop):\n\n", latestTrace.ID)

	file, err := os.Open(latestTrace.Path)
	if err != nil {
		return fmt.Errorf("failed to open trace log: %w", err)
	}
	defer file.Close()

	// Seek to end of file
	if _, err := file.Seek(0, 2); err != nil {
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	// Read new lines as they appear
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for {
		if scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Try to pretty-print JSON
			if strings.TrimSpace(line) != "" {
				var event map[string]interface{}
				if err := json.Unmarshal([]byte(line), &event); err == nil {
					// Format timestamp if present
					if ts, ok := event["timestamp"].(string); ok {
						t, err := time.Parse(time.RFC3339, ts)
						if err == nil {
							event["timestamp"] = t.Format("15:04:05")
						}
					}
					prettyJSON, _ := json.MarshalIndent(event, "", "  ")
					fmt.Println(prettyJSON)
				} else {
					fmt.Println(line)
				}
			}
		}

		// Check for errors
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading log: %w", err)
		}

		// Sleep briefly before checking for more data
		time.Sleep(100 * time.Millisecond)
	}
}

func tailFile(path string, numLines int) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read all lines
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Get last N lines
	start := 0
	if len(lines) > numLines {
		start = len(lines) - numLines
	}

	// Print last N lines
	for i, line := range lines[start:] {
		if strings.TrimSpace(line) != "" {
			// Try to pretty-print JSON
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(line), &event); err == nil {
				// Extract key fields for compact display
				timestamp := extractField(event, "timestamp")
				message := extractField(event, "message")
				eventType := extractField(event, "type")

				if timestamp != "" && message != "" {
					fmt.Printf("[%s] %s: %s\n", timestamp, eventType, message)
				} else {
					prettyJSON, _ := json.MarshalIndent(event, "", "  ")
					fmt.Printf("[%d] %s\n", start+i+1, prettyJSON)
				}
			} else {
				fmt.Printf("[%d] %s\n", start+i+1, line)
			}
		}
	}

	return nil
}

func extractField(event map[string]interface{}, field string) string {
	if val, ok := event[field]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
