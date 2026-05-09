package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/ux"
)

// CommandContext holds all command-line flags and configuration
// that were previously global variables. This enables:
// - Better testability (no global state interference)
// - Concurrent command execution
// - Explicit dependencies
type CommandContext struct {
	// Output control
	Verbose bool
	Quiet   bool
	Format  string
	NoColor bool

	// AI behavior
	Explain bool
	Trace   string

	// Configuration
	SpecularHome string
	LogLevel     string

	// Interactive mode
	// IsInteractive indicates if the terminal supports interactive prompts.
	// This is automatically detected based on stdin being a terminal and
	// the absence of CI environment variables.
	IsInteractive bool
}

// NewCommandContext extracts command context from cobra.Command flags.
// Commands should call this in their RunE function to get their configuration:
//
//	func runCommand(cmd *cobra.Command, args []string) error {
//		ctx, err := NewCommandContext(cmd)
//		if err != nil {
//			return fmt.Errorf("failed to create command context: %w", err)
//		}
//		// Use ctx.Verbose, ctx.Format, etc.
//	}
func NewCommandContext(cmd *cobra.Command) (*CommandContext, error) {
	// Extract all persistent flags from the command
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return nil, err
	}

	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, err
	}

	noColor, err := cmd.Flags().GetBool("no-color")
	if err != nil {
		return nil, err
	}

	explain, err := cmd.Flags().GetBool("explain")
	if err != nil {
		return nil, err
	}

	trace, err := cmd.Flags().GetString("trace")
	if err != nil {
		return nil, err
	}

	specularHome, err := cmd.Flags().GetString("home")
	if err != nil {
		return nil, err
	}

	logLevel, err := cmd.Flags().GetString("log-level")
	if err != nil {
		return nil, err
	}

	// Detect interactive mode based on environment and terminal state
	interactiveCfg := ux.NewInteractiveConfig()
	interactiveCfg.Quiet = quiet
	isInteractive := ux.ShouldPrompt(interactiveCfg)

	return &CommandContext{
		Verbose:       verbose,
		Quiet:         quiet,
		Format:        format,
		NoColor:       noColor,
		Explain:       explain,
		Trace:         trace,
		SpecularHome:  specularHome,
		LogLevel:      logLevel,
		IsInteractive: isInteractive,
	}, nil
}

// ValidateGlobalFlags validates global CLI flags before command execution.
func ValidateGlobalFlags(cmd *cobra.Command) error {
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return err
	}

	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return err
	}

	if verbose && quiet {
		return fmt.Errorf("--verbose and --quiet cannot be used together")
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}

	allowedFormats := map[string]struct{}{
		"text": {},
		"json": {},
		"yaml": {},
	}
	if _, ok := allowedFormats[strings.ToLower(format)]; !ok {
		return fmt.Errorf("invalid --format value %q (allowed: text, json, yaml)", format)
	}

	logLevel, err := cmd.Flags().GetString("log-level")
	if err != nil {
		return err
	}

	allowedLogLevels := map[string]struct{}{
		"debug": {},
		"info":  {},
		"warn":  {},
		"error": {},
	}
	if _, ok := allowedLogLevels[strings.ToLower(logLevel)]; !ok {
		return fmt.Errorf("invalid --log-level value %q (allowed: debug, info, warn, error)", logLevel)
	}

	return nil
}
