// Package benchmark provides performance measurement utilities for Specular CLI.
package benchmark

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/safeutil"
)

// StartupResult contains the results of a startup time measurement.
type StartupResult struct {
	Command     string
	Duration    time.Duration
	Iterations  int
	Min         time.Duration
	Max         time.Duration
	Avg         time.Duration
	P50         time.Duration
	P95         time.Duration
	P99         time.Duration
	StdDev      time.Duration
	MemoryBytes int64
}

// BinaryInfo contains information about the compiled binary.
type BinaryInfo struct {
	Path         string
	Size         int64
	SizeHuman    string
	GOOS         string
	GOARCH       string
	GoVersion    string
	BuildFlags   string
	Dependencies int
}

// MeasureStartup measures the startup time of a command over multiple iterations.
func MeasureStartup(binary string, args []string, iterations int) (*StartupResult, error) {
	if iterations < 1 {
		iterations = 10
	}

	durations := make([]time.Duration, 0, iterations)
	cmd := append([]string{binary}, args...)
	cmdStr := strings.Join(cmd, " ")

	for i := 0; i < iterations; i++ {
		start := time.Now()
		c, err := safeutil.SafeCommand(context.Background(), binary, args...)
		if err != nil {
			return nil, fmt.Errorf("prepare benchmark command: %w", err)
		}
		c.Stdout = nil
		c.Stderr = nil
		if runErr := c.Run(); runErr != nil {
			return nil, fmt.Errorf("command failed: %w", runErr)
		}
		durations = append(durations, time.Since(start))
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	result := &StartupResult{
		Command:    cmdStr,
		Iterations: iterations,
		Min:        durations[0],
		Max:        durations[len(durations)-1],
	}

	// Calculate average
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	result.Avg = total / time.Duration(iterations)
	result.Duration = result.Avg

	// Calculate percentiles
	result.P50 = durations[len(durations)*50/100]
	result.P95 = durations[len(durations)*95/100]
	result.P99 = durations[len(durations)*99/100]

	// Calculate standard deviation
	var sumSquares float64
	avgNs := float64(result.Avg.Nanoseconds())
	for _, d := range durations {
		diff := float64(d.Nanoseconds()) - avgNs
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(iterations)
	result.StdDev = time.Duration(sqrt(variance))

	return result, nil
}

// GetBinaryInfo returns information about a binary file.
func GetBinaryInfo(path string) (*BinaryInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat binary: %w", err)
	}

	size := info.Size()
	return &BinaryInfo{
		Path:      path,
		Size:      size,
		SizeHuman: humanBytes(size),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		GoVersion: runtime.Version(),
	}, nil
}

// RunBenchmarkSuite runs a suite of startup benchmarks.
func RunBenchmarkSuite(binary string, iterations int) ([]StartupResult, error) {
	commands := [][]string{
		{"version"},
		{"--help"},
		{"doctor", "--help"},
		{"config", "view", "--help"},
		{"plugin", "list", "--help"},
	}

	results := make([]StartupResult, 0, len(commands))
	for _, args := range commands {
		result, err := MeasureStartup(binary, args, iterations)
		if err != nil {
			// Skip failing commands
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// FormatResult formats a StartupResult for display.
func FormatResult(r *StartupResult) string {
	return fmt.Sprintf(`Command:     %s
Iterations:  %d
Average:     %s
Min:         %s
Max:         %s
P50:         %s
P95:         %s
P99:         %s
Std Dev:     %s
`,
		r.Command,
		r.Iterations,
		r.Avg,
		r.Min,
		r.Max,
		r.P50,
		r.P95,
		r.P99,
		r.StdDev,
	)
}

// FormatBinaryInfo formats BinaryInfo for display.
func FormatBinaryInfo(b *BinaryInfo) string {
	return fmt.Sprintf(`Binary:      %s
Size:        %s (%d bytes)
Platform:    %s/%s
Go Version:  %s
`,
		filepath.Base(b.Path),
		b.SizeHuman,
		b.Size,
		b.GOOS,
		b.GOARCH,
		b.GoVersion,
	)
}

// humanBytes converts bytes to human-readable format.
func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// sqrt calculates square root (simple Newton-Raphson).
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x == 0 {
		return 0
	}
	z := x / 2
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
