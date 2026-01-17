package cmd

import (
	"github.com/spf13/cobra"
)

// driftCmd is a top-level alias for 'eval drift' for convenience.
// This provides a shorter path to the drift detection functionality
// while keeping 'eval drift' as the canonical location.
var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect drift between spec/plan and implementation (alias for 'eval drift')",
	Long: `Detect drift between specification/plan and actual implementation.

This is a convenience alias for 'specular eval drift'. The canonical command is:
  specular eval drift

Drift detection compares:
  • Locked spec features vs implemented functionality
  • Plan tasks vs actual file changes
  • Expected outputs vs generated artifacts

Examples:
  specular drift                    # Run drift detection
  specular drift --format json      # Output as JSON
  specular eval drift               # Canonical command

See 'specular eval drift --help' for all options.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Delegate to eval drift command
		evalDriftCmd.SetArgs(args)
		return evalDriftCmd.Execute()
	},
}

func init() {
	rootCmd.AddCommand(driftCmd)

	// Copy flags from evalDriftCmd to ensure consistency
	// Note: Flags are inherited when we execute the actual command
}
