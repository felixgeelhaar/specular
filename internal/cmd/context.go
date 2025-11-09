package cmd

import (
	"github.com/spf13/cobra"
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

	return &CommandContext{
		Verbose:      verbose,
		Quiet:        quiet,
		Format:       format,
		NoColor:      noColor,
		Explain:      explain,
		Trace:        trace,
		SpecularHome: specularHome,
		LogLevel:     logLevel,
	}, nil
}
