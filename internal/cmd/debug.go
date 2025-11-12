package cmd

import (
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debugging and diagnostic utilities",
	Long: `Debugging and diagnostic utilities for troubleshooting Specular.

Commands:
  status    Show environment and project status
  context   Detect and display environment setup
  doctor    Run system diagnostics and health checks
  logs      Show or tail CLI logs
  explain   Explain routing decisions for a workflow

These tools help you:
  • Understand your environment configuration
  • Diagnose issues with providers and dependencies
  • View execution logs and traces
  • Analyze routing and model selection decisions

Examples:
  specular debug status
  specular debug doctor
  specular debug logs --follow
  specular debug context --format json
  specular debug explain auto-1762811730`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	// Group utility commands under debug namespace
	debugCmd.AddCommand(statusCmd)
	debugCmd.AddCommand(contextCmd)
	debugCmd.AddCommand(doctorCmd)
	debugCmd.AddCommand(logsCmd)
	debugCmd.AddCommand(explainCmd)

	// Add debug command to root
	rootCmd.AddCommand(debugCmd)
}
