package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var interviewCmd = &cobra.Command{
	Use:   "interview",
	Short: "Interactive interview mode to generate spec from Q&A",
	Long: `Launch an interactive interview session that guides you through
creating a best-practice specification from natural language inputs.

Supports presets (saas-api, mobile-app, internal-tool) and strict mode
for enhanced validation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out, _ := cmd.Flags().GetString("out")
		preset, _ := cmd.Flags().GetString("preset")
		strict, _ := cmd.Flags().GetBool("strict")
		tui, _ := cmd.Flags().GetBool("tui")

		// TODO: Implement interview logic
		fmt.Printf("Interview mode (out=%s, preset=%s, strict=%v, tui=%v)\n", out, preset, strict, tui)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(interviewCmd)

	interviewCmd.Flags().StringP("out", "o", ".aidv/spec.yaml", "Output path for generated spec")
	interviewCmd.Flags().String("preset", "", "Use a preset template (saas-api|mobile-app|internal-tool)")
	interviewCmd.Flags().Bool("strict", false, "Enable strict validation mode")
	interviewCmd.Flags().Bool("tui", false, "Use terminal UI mode")
}
