package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BuildTUIConfig configures the build TUI
type BuildTUIConfig struct {
	SpecFile  string
	OutputDir string
	Format    string
	Validate  bool
	ShowDiff  bool
}

// BuildTUI provides a TUI wrapper for the build command
type BuildTUI struct {
	adapter *CommandTUIAdapter
	config  BuildTUIConfig
	output  io.Writer
}

// BuildPhases defines the phases of a build operation
var BuildPhases = []string{
	"Load spec file",
	"Validate spec",
	"Resolve dependencies",
	"Generate output",
	"Write files",
}

// NewBuildTUI creates a new build TUI
func NewBuildTUI(config BuildTUIConfig, output io.Writer) *BuildTUI {
	if output == nil {
		output = os.Stdout
	}

	tuiConfig := CommandTUIConfig{
		Command:     CommandBuild,
		Title:       "Build",
		Description: fmt.Sprintf("Building %s", filepath.Base(config.SpecFile)),
		Phases:      BuildPhases,
		ShowLogs:    true,
		ShowMetrics: true,
		Interactive: true,
	}

	return &BuildTUI{
		adapter: NewCommandTUIAdapter(tuiConfig, output),
		config:  config,
		output:  output,
	}
}

// Start starts the build TUI
func (b *BuildTUI) Start() error {
	return b.adapter.Start()
}

// Stop stops the build TUI
func (b *BuildTUI) Stop() {
	b.adapter.Stop()
}

// handlePhaseEvent is a helper for common phase event handling pattern.
func (b *BuildTUI) handlePhaseEvent(
	phaseIdx int,
	phaseName string,
	started bool,
	err error,
	startMsg string,
	successMsg string,
	failMsgPrefix string,
	details map[string]interface{},
) {
	if started {
		b.adapter.StartPhase(phaseIdx, phaseName)
		b.adapter.Log(LogLevelInfo, startMsg)
	} else if err != nil {
		b.adapter.FailPhase(phaseIdx, phaseName, err)
		b.adapter.Log(LogLevelError, fmt.Sprintf("%s: %v", failMsgPrefix, err))
	} else {
		b.adapter.CompletePhase(phaseIdx, phaseName, details)
		b.adapter.Log(LogLevelInfo, successMsg)
	}
}

// OnLoadSpec signals that spec loading has started/completed
func (b *BuildTUI) OnLoadSpec(started bool, specPath string, err error) {
	b.handlePhaseEvent(
		0, "Load spec file", started, err,
		fmt.Sprintf("Loading spec from %s", specPath),
		"Spec loaded successfully",
		"Failed to load spec",
		map[string]interface{}{"path": specPath},
	)
}

// OnValidate signals that validation has started/completed
func (b *BuildTUI) OnValidate(started bool, warnings int, errors int, err error) {
	if started {
		b.adapter.StartPhase(1, "Validate spec")
		b.adapter.Log(LogLevelInfo, "Validating spec...")
	} else if err != nil {
		b.adapter.FailPhase(1, "Validate spec", err)
		b.adapter.Log(LogLevelError, fmt.Sprintf("Validation failed: %v", err))
	} else {
		details := map[string]interface{}{
			"warnings": warnings,
			"errors":   errors,
		}
		b.adapter.CompletePhase(1, "Validate spec", details)

		if warnings > 0 {
			b.adapter.Log(LogLevelWarn, fmt.Sprintf("Validation passed with %d warnings", warnings))
		} else {
			b.adapter.Log(LogLevelInfo, "Validation passed")
		}
	}
}

// OnResolve signals that dependency resolution has started/completed
func (b *BuildTUI) OnResolve(started bool, dependencies int, err error) {
	if started {
		b.adapter.StartPhase(2, "Resolve dependencies")
		b.adapter.Log(LogLevelInfo, "Resolving dependencies...")
	} else if err != nil {
		b.adapter.FailPhase(2, "Resolve dependencies", err)
		b.adapter.Log(LogLevelError, fmt.Sprintf("Resolution failed: %v", err))
	} else {
		b.adapter.CompletePhase(2, "Resolve dependencies", map[string]interface{}{
			"count": dependencies,
		})
		b.adapter.Log(LogLevelInfo, fmt.Sprintf("Resolved %d dependencies", dependencies))
	}
}

// OnGenerate signals that output generation has started/completed
func (b *BuildTUI) OnGenerate(started bool, format string, err error) {
	b.handlePhaseEvent(
		3, "Generate output", started, err,
		fmt.Sprintf("Generating %s output...", format),
		"Output generated",
		"Generation failed",
		map[string]interface{}{"format": format},
	)
}

// OnWriteFile signals that a file is being written
func (b *BuildTUI) OnWriteFile(path string, size int64, err error) {
	if err != nil {
		b.adapter.Log(LogLevelError, fmt.Sprintf("Failed to write %s: %v", filepath.Base(path), err))
	} else {
		b.adapter.AddOutput(OutputItem{
			Name:    filepath.Base(path),
			Path:    path,
			Size:    size,
			Created: time.Now(),
		})
		b.adapter.Log(LogLevelInfo, fmt.Sprintf("Wrote %s", filepath.Base(path)))
	}
}

// OnWriteComplete signals that all files have been written
func (b *BuildTUI) OnWriteComplete(fileCount int, totalSize int64, err error) {
	if err != nil {
		b.adapter.FailPhase(4, "Write files", err)
	} else {
		b.adapter.CompletePhase(4, "Write files", map[string]interface{}{
			"files": fileCount,
			"size":  formatBytes(totalSize),
		})
		b.adapter.Log(LogLevelInfo, fmt.Sprintf("Wrote %d files (%s)", fileCount, formatBytes(totalSize)))
	}
}

// OnComplete signals that the build has completed
func (b *BuildTUI) OnComplete(success bool, err error) {
	b.adapter.Done(success, err)
}

// UpdateMetric updates a build metric
func (b *BuildTUI) UpdateMetric(key string, value interface{}) {
	b.adapter.UpdateMetric(key, value)
}

// Log adds a log entry
func (b *BuildTUI) Log(level LogLevel, message string) {
	b.adapter.Log(level, message)
}

// BuildTUIRunner wraps a build operation with TUI
type BuildTUIRunner struct {
	tui    *BuildTUI
	config BuildTUIConfig
}

// NewBuildTUIRunner creates a new build TUI runner
func NewBuildTUIRunner(config BuildTUIConfig, output io.Writer) *BuildTUIRunner {
	return &BuildTUIRunner{
		tui:    NewBuildTUI(config, output),
		config: config,
	}
}

// Run executes the build with TUI wrapper
func (r *BuildTUIRunner) Run(buildFn func(*BuildTUI) error) error {
	if err := r.tui.Start(); err != nil {
		return fmt.Errorf("failed to start TUI: %w", err)
	}
	defer r.tui.Stop()

	// Execute the build function with TUI hooks
	err := buildFn(r.tui)

	// Signal completion
	r.tui.OnComplete(err == nil, err)

	return err
}
