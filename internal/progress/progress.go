package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
)

// Indicator provides progress tracking and display for long-running operations
type Indicator struct {
	writer      io.Writer
	state       *checkpoint.State
	startTime   time.Time
	lastUpdate  time.Time
	mu          sync.Mutex
	showSpinner bool
	spinnerIdx  int
	stopChan    chan struct{}
	stopOnce    sync.Once // Ensures Stop() is only called once
	isCI        bool
}

// Config holds configuration for progress indicator
type Config struct {
	Writer      io.Writer
	ShowSpinner bool
	IsCI        bool // Set to true in CI/CD environments to disable fancy output
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewIndicator creates a new progress indicator
func NewIndicator(cfg Config) *Indicator {
	if cfg.Writer == nil {
		cfg.Writer = os.Stdout
	}

	// Auto-detect CI environment
	if !cfg.IsCI {
		cfg.IsCI = os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
	}

	return &Indicator{
		writer:      cfg.Writer,
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		showSpinner: cfg.ShowSpinner && !cfg.IsCI,
		stopChan:    make(chan struct{}),
		isCI:        cfg.IsCI,
	}
}

// SetState sets the checkpoint state to track
func (p *Indicator) SetState(state *checkpoint.State) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
}

// Start begins the progress indicator display
func (p *Indicator) Start() {
	if p.showSpinner {
		go p.spinnerLoop()
	}
}

// Stop stops the progress indicator
func (p *Indicator) Stop() {
	p.stopOnce.Do(func() {
		if p.showSpinner {
			close(p.stopChan)
			// Clear spinner line
			fmt.Fprintf(p.writer, "\r%s\r", strings.Repeat(" ", 80))
		}
	})
}

// spinnerLoop runs the spinner animation
func (p *Indicator) spinnerLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.mu.Lock()
			if p.state != nil {
				p.renderProgress()
			}
			p.spinnerIdx = (p.spinnerIdx + 1) % len(spinnerFrames)
			p.mu.Unlock()
		}
	}
}

// renderProgress renders the current progress state
func (p *Indicator) renderProgress() {
	if p.state == nil {
		return
	}

	progress := p.state.Progress()
	completed := len(p.state.GetCompletedTasks())
	failed := len(p.state.GetFailedTasks())
	total := len(p.state.Tasks)
	elapsed := time.Since(p.startTime)

	// Calculate ETA
	var eta string
	if progress > 0 && progress < 1.0 {
		totalEstimated := time.Duration(float64(elapsed) / progress)
		remaining := totalEstimated - elapsed
		eta = fmt.Sprintf(" | ETA: %s", formatDuration(remaining))
	}

	// Build progress bar
	barWidth := 30
	filled := int(float64(barWidth) * progress)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Spinner frame
	spinner := spinnerFrames[p.spinnerIdx]

	// Status line
	statusLine := fmt.Sprintf("\r%s [%s] %.1f%% | %d/%d tasks | ✓ %d | ✗ %d | %s%s",
		spinner,
		bar,
		progress*100,
		completed+failed,
		total,
		completed,
		failed,
		formatDuration(elapsed),
		eta,
	)

	fmt.Fprint(p.writer, statusLine)
}

// UpdateTask updates progress for a specific task
func (p *Indicator) UpdateTask(taskID, status string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state != nil {
		p.state.UpdateTask(taskID, status, err)
		p.lastUpdate = time.Now()

		// In CI mode, print status updates immediately
		if p.isCI {
			p.printTaskStatus(taskID, status, err)
		}
	}
}

// printTaskStatus prints task status in CI-friendly format
func (p *Indicator) printTaskStatus(taskID, status string, err error) {
	symbol := "⟲"
	switch status {
	case "running":
		symbol = "▶"
	case "completed":
		symbol = "✓"
	case "failed":
		symbol = "✗"
	case "skipped":
		symbol = "⊘"
	}

	msg := fmt.Sprintf("%s %s [%s]", symbol, taskID, status)
	if err != nil {
		msg += fmt.Sprintf(" - %v", err)
	}

	fmt.Fprintln(p.writer, msg)
}

// PrintSummary prints final execution summary
func (p *Indicator) PrintSummary() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state == nil {
		return
	}

	fmt.Fprintln(p.writer)
	fmt.Fprintln(p.writer, "═══════════════════════════════════════════════════════════")
	fmt.Fprintln(p.writer, "Execution Summary")
	fmt.Fprintln(p.writer, "═══════════════════════════════════════════════════════════")

	completed := p.state.GetCompletedTasks()
	failed := p.state.GetFailedTasks()
	total := len(p.state.Tasks)
	elapsed := time.Since(p.startTime)

	fmt.Fprintf(p.writer, "Total Tasks:     %d\n", total)
	fmt.Fprintf(p.writer, "Completed:       %d ✓\n", len(completed))
	fmt.Fprintf(p.writer, "Failed:          %d ✗\n", len(failed))
	fmt.Fprintf(p.writer, "Success Rate:    %.1f%%\n", float64(len(completed))/float64(total)*100)
	fmt.Fprintf(p.writer, "Total Time:      %s\n", formatDuration(elapsed))

	if len(completed) > 0 {
		avgTime := elapsed / time.Duration(len(completed))
		fmt.Fprintf(p.writer, "Avg Time/Task:   %s\n", formatDuration(avgTime))
	}

	fmt.Fprintln(p.writer, "═══════════════════════════════════════════════════════════")

	// Print failed tasks details
	if len(failed) > 0 {
		fmt.Fprintln(p.writer)
		fmt.Fprintln(p.writer, "Failed Tasks:")
		for _, taskID := range failed {
			task := p.state.Tasks[taskID]
			fmt.Fprintf(p.writer, "  ✗ %s", taskID)
			if task.Error != "" {
				fmt.Fprintf(p.writer, " - %s", task.Error)
			}
			fmt.Fprintln(p.writer)
		}
	}
}

// PrintResumeInfo prints information when resuming from checkpoint
func (p *Indicator) PrintResumeInfo() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state == nil {
		return
	}

	completed := len(p.state.GetCompletedTasks())
	failed := len(p.state.GetFailedTasks())
	pending := len(p.state.GetPendingTasks())

	fmt.Fprintln(p.writer, "─────────────────────────────────────────────────────────")
	fmt.Fprintf(p.writer, "Resuming: %s\n", p.state.OperationID)
	fmt.Fprintln(p.writer, "─────────────────────────────────────────────────────────")
	fmt.Fprintf(p.writer, "  Completed:  %d tasks ✓\n", completed)
	fmt.Fprintf(p.writer, "  Pending:    %d tasks ⟲\n", pending)
	fmt.Fprintf(p.writer, "  Failed:     %d tasks ✗\n", failed)
	fmt.Fprintf(p.writer, "  Progress:   %.1f%%\n", p.state.Progress()*100)
	fmt.Fprintln(p.writer, "─────────────────────────────────────────────────────────")
	fmt.Fprintln(p.writer)
}

// StreamWriter wraps an io.Writer to stream output with prefixes
type StreamWriter struct {
	writer io.Writer
	prefix string
	buffer []byte
}

// NewStreamWriter creates a new stream writer with a prefix
func NewStreamWriter(w io.Writer, prefix string) *StreamWriter {
	return &StreamWriter{
		writer: w,
		prefix: prefix,
		buffer: make([]byte, 0, 4096),
	}
}

// Write implements io.Writer
func (sw *StreamWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	sw.buffer = append(sw.buffer, p...)

	// Process complete lines
	for {
		idx := strings.IndexByte(string(sw.buffer), '\n')
		if idx == -1 {
			break
		}

		line := sw.buffer[:idx]
		sw.buffer = sw.buffer[idx+1:]

		// Write prefixed line
		_, err = fmt.Fprintf(sw.writer, "%s %s\n", sw.prefix, string(line))
		if err != nil {
			return
		}
	}

	return
}

// Flush writes any remaining buffered content
func (sw *StreamWriter) Flush() error {
	if len(sw.buffer) > 0 {
		_, err := fmt.Fprintf(sw.writer, "%s %s\n", sw.prefix, string(sw.buffer))
		sw.buffer = sw.buffer[:0]
		return err
	}
	return nil
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// BarIndicator provides a simple progress bar without animation
type BarIndicator struct {
	writer    io.Writer
	total     int
	completed int
	failed    int
	startTime time.Time
	mu        sync.Mutex
}

// NewBarIndicator creates a simple progress bar
func NewBarIndicator(w io.Writer, total int) *BarIndicator {
	if w == nil {
		w = os.Stdout
	}
	return &BarIndicator{
		writer:    w,
		total:     total,
		startTime: time.Now(),
	}
}

// Increment increments the progress counter
func (b *BarIndicator) Increment(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if success {
		b.completed++
	} else {
		b.failed++
	}

	b.render()
}

// render draws the progress bar
func (b *BarIndicator) render() {
	progress := float64(b.completed+b.failed) / float64(b.total)
	barWidth := 40
	filled := int(float64(barWidth) * progress)

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	elapsed := time.Since(b.startTime)

	fmt.Fprintf(b.writer, "\r[%s] %.0f%% | %d/%d | ✓ %d | ✗ %d | %s",
		bar,
		progress*100,
		b.completed+b.failed,
		b.total,
		b.completed,
		b.failed,
		formatDuration(elapsed),
	)
}

// Finish completes the progress bar
func (b *BarIndicator) Finish() {
	b.mu.Lock()
	defer b.mu.Unlock()

	fmt.Fprintln(b.writer)
}
