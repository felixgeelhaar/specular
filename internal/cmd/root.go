package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "specular",
	Short: "AI-Native Spec and Build Assistant",
	Long: `
  ╔══════════════════════════════════════════════════════════════╗
  ║                      [ specular ]                            ║
  ║            AI-Native Spec and Build Assistant                ║
  ╚══════════════════════════════════════════════════════════════╝

specular is a CLI tool that enables spec-first, policy-enforced software
development using AI. It transforms natural language product requirements into
structured specifications, executable plans, and production-ready code while
maintaining traceability and enforcing organizational guardrails.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
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
