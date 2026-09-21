package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/evidence"
	"github.com/felixgeelhaar/specular/internal/gate"
)

// changeExplainCmd is the top-level PRODUCT_INTENT §20 surface:
// "Why did Specular make this decision?" Routing explain remains under
// `specular debug explain`.
var changeExplainCmd = &cobra.Command{
	Use:   "explain [evidence-id]",
	Short: "Explain a gate ALLOW / DENY decision",
	Long: `Explain why Specular allowed or denied a change.

Default: load the latest Change Evidence Graph record written by
specular gate. Pass an evidence id (ev_…) to explain a specific record.
With --fresh, re-run the gate (and persist evidence) before explaining.

Human text is an auditor-facing AI CHANGE RECORD (PRODUCT_INTENT §19).
--json emits the unchanged machine-readable evidence record.

Routing / model-selection explainability remains at:
  specular debug explain <checkpoint-id>

See docs/PRODUCT_INTENT.md §7 (Change Evidence Graph) and §20 (Explain).

Examples:
  specular explain
  specular explain ev_abc123
  specular explain --fresh
  specular explain --json
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runChangeExplain,
}

func runChangeExplain(cmd *cobra.Command, args []string) error {
	projectRoot, _ := cmd.Flags().GetString("project-root")
	jsonOut, _ := cmd.Flags().GetBool("json")
	fresh, _ := cmd.Flags().GetBool("fresh")
	strictSpec, _ := cmd.Flags().GetBool("strict-spec")
	policyPath, _ := cmd.Flags().GetString("policy")

	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		projectRoot = cwd
	}

	var rec *evidence.Record
	var err error
	switch {
	case fresh:
		res, evalErr := gate.Evaluate(gate.Options{
			ProjectRoot: projectRoot,
			PolicyPath:  policyPath,
			StrictSpec:  strictSpec,
		})
		if evalErr != nil {
			return evalErr
		}
		rec, err = evidence.NewFromGate(projectRoot, res)
		if err != nil {
			return err
		}
		if writeErr := evidence.Write(projectRoot, rec); writeErr != nil {
			return writeErr
		}
	case len(args) == 1:
		rec, err = evidence.Load(projectRoot, args[0])
	default:
		rec, err = evidence.LoadLatest(projectRoot)
	}
	if err != nil {
		return err
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rec)
	}
	fmt.Print(evidence.FormatExplain(rec))
	return nil
}

func init() {
	changeExplainCmd.Flags().String("project-root", "", "Repository root (default: cwd)")
	changeExplainCmd.Flags().String("policy", "", "Policy file when using --fresh")
	changeExplainCmd.Flags().Bool("strict-spec", false, "Require specs when using --fresh")
	changeExplainCmd.Flags().Bool("fresh", false, "Re-run specular gate before explaining")
	changeExplainCmd.Flags().Bool("json", false, "Emit the evidence record as JSON")
	rootCmd.AddCommand(changeExplainCmd)
}
