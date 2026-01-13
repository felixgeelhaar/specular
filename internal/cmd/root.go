// Package cmd implements the Specular CLI command structure using Cobra.
//
// This package follows Cobra's recommended pattern of using a package-level
// rootCmd variable with init() functions for command registration. While this
// uses a global variable, it's the standard Cobra approach that ensures all
// subcommands are registered before Execute() is called.
//
// For testing, use ExecuteContext(ctx) which provides a clean entry point.
// The CommandContext abstraction in flags.go provides dependency injection
// for command-level dependencies.
//
// Commands are organized as:
//   - root.go: Root command and global flags
//   - auto.go, plan.go, etc.: Subcommands with their own flags
//   - flags.go: CommandContext for flag extraction
//   - errors.go: User-friendly error handling
package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	_ "github.com/felixgeelhaar/specular/internal/provider/native/ollama"
)

// rootCmd is the base command for the Specular CLI.
//
// This follows Cobra's recommended pattern of a package-level variable.
// Subcommands register themselves via init() functions using rootCmd.AddCommand().
// Testing is done through ExecuteContext() which wraps this command.
var rootCmd = &cobra.Command{
	Use:   "specular",
	Short: "AI-Native Spec and Build Assistant",
	Long: `╔══════════════════════════════════════════════════════════════╗
║                      [ specular ]                            ║
║            AI-Native Spec and Build Assistant                ║
╚══════════════════════════════════════════════════════════════╝

specular is a CLI tool that enables spec-first, policy-enforced software
development using AI. It transforms natural language product requirements into
structured specifications, executable plans, and production-ready code while
maintaining traceability and enforcing organizational guardrails.`,
	SilenceUsage:  true, // Don't show usage on errors - it's noise
	SilenceErrors: true, // main.go handles error printing
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// ExecuteContext runs the root command with the given context
func ExecuteContext(ctx context.Context) error {
	// Skip observability setup for fast commands to improve startup time
	if !isFastCommand() {
		cleanup := setupObservability(ctx)
		defer cleanup()
	}

	return rootCmd.ExecuteContext(ctx)
}

// isFastCommand checks if the current invocation is a "fast" command
// that doesn't need observability (logging, metrics, telemetry).
// This significantly improves startup time for simple commands.
func isFastCommand() bool {
	args := os.Args[1:]
	if len(args) == 0 {
		return false
	}

	// Commands that don't need observability
	fastCommands := map[string]bool{
		"version":    true,
		"completion": true,
		"help":       true,
		"__complete": true, // Cobra completion helper
	}

	// Check first argument
	if fastCommands[args[0]] {
		return true
	}

	// Check for --help or -h flag anywhere
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}

	return false
}

func init() {
	// Get default values from environment variables for flag defaults
	specularHome := os.Getenv("SPECULAR_HOME")

	logLevel := os.Getenv("SPECULAR_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	noColor := os.Getenv("SPECULAR_NO_COLOR") == "true"

	// Output control flags
	// Note: Commands should use NewCommandContext(cmd) to access these values
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().String("format", "text", "Output format (text, json, yaml)")
	rootCmd.PersistentFlags().Bool("no-color", noColor, "Disable colored output")

	// AI behavior flags
	rootCmd.PersistentFlags().Bool("explain", false, "Show AI reasoning and decision-making process")
	rootCmd.PersistentFlags().String("trace", "", "Distributed tracing ID for debugging")

	// Configuration flags
	rootCmd.PersistentFlags().String("home", specularHome, "Override .specular directory location")
	rootCmd.PersistentFlags().String("log-level", logLevel, "Log level (debug, info, warn, error)")
}
