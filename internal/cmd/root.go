package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ai-dev",
	Short: "AI-Native Spec and Build Assistant",
	Long: `ai-dev is a CLI tool that enables spec-first, policy-enforced software development using AI.
It transforms natural language product requirements into structured specifications,
executable plans, and production-ready code while maintaining traceability and
enforcing organizational guardrails.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ai-dev/config.yaml)")
}
