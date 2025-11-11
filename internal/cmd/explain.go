package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/explain"
)

var explainCmd = &cobra.Command{
	Use:   "explain <checkpoint-id>",
	Short: "Explain routing decisions for a workflow",
	Long: `Explain how routing decisions were made during a workflow execution.

This command analyzes a completed workflow and explains:
  - Which providers and models were selected for each step
  - Why those selections were made (routing strategy, costs, signals)
  - Budget utilization and cost breakdown by provider
  - Overall routing strategy and performance

This is useful for:
  - Understanding why specific models were chosen
  - Debugging unexpected routing behavior
  - Optimizing routing strategy and costs
  - Auditing model selection decisions

Examples:
  # Explain routing for a completed workflow
  specular explain auto-1762811730

  # Output as JSON for programmatic analysis
  specular explain auto-1762811730 --format json

  # Output as Markdown for documentation
  specular explain auto-1762811730 --format markdown

  # Compact summary format
  specular explain auto-1762811730 --format compact`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		checkpointID := args[0]

		// Parse flags
		format, _ := cmd.Flags().GetString("format")
		outputFile, _ := cmd.Flags().GetString("output")

		// Get checkpoint directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		checkpointDir := filepath.Join(homeDir, ".specular", "checkpoints")

		// Create explainer
		explainer := explain.NewExplainer(checkpointDir)

		// Generate explanation
		fmt.Printf("🔍 Analyzing routing decisions for workflow: %s\n\n", checkpointID)

		explanation, err := explainer.Explain(checkpointID)
		if err != nil {
			return fmt.Errorf("failed to generate explanation: %w", err)
		}

		// Format output
		formatter := explain.NewFormatter(true) // Enable colors
		var output string

		switch format {
		case "json":
			output, err = formatter.FormatJSON(explanation)
			if err != nil {
				return fmt.Errorf("failed to format JSON: %w", err)
			}
		case "markdown", "md":
			output = formatter.FormatMarkdown(explanation)
		case "compact":
			output = formatter.FormatCompact(explanation)
		case "text", "":
			output = formatter.FormatText(explanation)
		default:
			return fmt.Errorf("unknown format: %s (use: text, json, markdown, compact)", format)
		}

		// Write output
		if outputFile != "" {
			if err := os.WriteFile(outputFile, []byte(output), 0600); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("✅ Explanation written to: %s\n", outputFile)
		} else {
			fmt.Println(output)
		}

		return nil
	},
}

func init() {
	explainCmd.Flags().StringP("format", "f", "text", "Output format (text, json, markdown, compact)")
	explainCmd.Flags().StringP("output", "o", "", "Write output to file instead of stdout")

	rootCmd.AddCommand(explainCmd)
}
