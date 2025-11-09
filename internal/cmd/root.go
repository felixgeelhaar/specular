package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Global flag variables
var (
	// Output control
	verbose bool
	quiet   bool
	format  string
	noColor bool

	// AI behavior
	explain bool
	trace   string

	// Configuration
	specularHome string
	logLevel     string
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
	// Environment variable defaults
	if specularHome == "" {
		specularHome = os.Getenv("SPECULAR_HOME")
	}
	if logLevel == "" {
		logLevel = os.Getenv("SPECULAR_LOG_LEVEL")
		if logLevel == "" {
			logLevel = "info"
		}
	}
	if os.Getenv("SPECULAR_NO_COLOR") == "true" {
		noColor = true
	}

	// Output control flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().StringVar(&format, "format", "text", "Output format (text, json, yaml)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", noColor, "Disable colored output")

	// AI behavior flags
	rootCmd.PersistentFlags().BoolVar(&explain, "explain", false, "Show AI reasoning and decision-making process")
	rootCmd.PersistentFlags().StringVar(&trace, "trace", "", "Distributed tracing ID for debugging")

	// Configuration flags
	rootCmd.PersistentFlags().StringVar(&specularHome, "home", specularHome, "Override .specular directory location")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", logLevel, "Log level (debug, info, warn, error)")
}
