package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/evidence"
)

var evidenceCmd = &cobra.Command{
	Use:   "evidence",
	Short: "Show Change Evidence Graph records",
	Long: `Inspect durable evidence records written by specular gate.

Records live under .specular/evidence/ (schema specular.evidence/v1).
This is the first Change Evidence Graph surface — later versions add
intent/session/approval edges. See docs/PRODUCT_INTENT.md §7.

Examples:
  specular evidence list
  specular evidence show
  specular evidence show ev_abc123
  specular evidence show --json
`,
}

var evidenceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List evidence record IDs (newest first)",
	Args:  cobra.NoArgs,
	RunE:  runEvidenceList,
}

var evidenceShowCmd = &cobra.Command{
	Use:   "show [evidence-id]",
	Short: "Show one evidence record (default: latest)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runEvidenceShow,
}

func runEvidenceList(cmd *cobra.Command, _ []string) error {
	root, err := evidenceProjectRoot(cmd)
	if err != nil {
		return err
	}
	ids, err := evidence.ListIDs(root)
	if err != nil {
		return err
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(ids)
	}
	if len(ids) == 0 {
		fmt.Println("No evidence records. Run: specular gate")
		return nil
	}
	for _, id := range ids {
		fmt.Println(id)
	}
	return nil
}

func runEvidenceShow(cmd *cobra.Command, args []string) error {
	root, err := evidenceProjectRoot(cmd)
	if err != nil {
		return err
	}
	var rec *evidence.Record
	if len(args) == 1 {
		rec, err = evidence.Load(root, args[0])
	} else {
		rec, err = evidence.LoadLatest(root)
	}
	if err != nil {
		return err
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rec)
	}
	fmt.Print(evidence.FormatExplain(rec))
	return nil
}

func evidenceProjectRoot(cmd *cobra.Command) (string, error) {
	root, _ := cmd.Flags().GetString("project-root")
	if root != "" {
		return root, nil
	}
	return os.Getwd()
}

func init() {
	evidenceCmd.PersistentFlags().String("project-root", "", "Repository root (default: cwd)")
	evidenceListCmd.Flags().Bool("json", false, "Emit JSON array of IDs")
	evidenceShowCmd.Flags().Bool("json", false, "Emit the evidence record as JSON")
	evidenceCmd.AddCommand(evidenceListCmd)
	evidenceCmd.AddCommand(evidenceShowCmd)
	rootCmd.AddCommand(evidenceCmd)
}
