package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/baseline"
	"github.com/felixgeelhaar/specular/internal/evidence"
	"github.com/felixgeelhaar/specular/internal/gate"
)

var baselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Acknowledge current state for progressive governance",
	Long: `Capture and inspect an explicit Specular baseline (PRODUCT_INTENT §25).

A baseline means: this is the acknowledged current state.
It does not mean: this state is good.

New drift from the baseline can then be governed without remediating all
historical debt immediately. Captures write .specular/baseline.yaml
(reviewable YAML) and a drift-baseline.json pointer for approval workflows.

Examples:
  specular baseline capture
  specular baseline capture --note "pilot week-1 acknowledge"
  specular baseline show
  specular baseline status
  specular baseline status --json
`,
}

var baselineCaptureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Record the current gate result as the acknowledged baseline",
	Args:  cobra.NoArgs,
	RunE:  runBaselineCapture,
}

var baselineShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the current baseline document",
	Args:  cobra.NoArgs,
	RunE:  runBaselineShow,
}

var baselineStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Compare current gate result against the baseline",
	Args:  cobra.NoArgs,
	RunE:  runBaselineStatus,
}

func runBaselineCapture(cmd *cobra.Command, _ []string) error {
	root, err := baselineRoot(cmd)
	if err != nil {
		return err
	}
	note, _ := cmd.Flags().GetString("note")
	strictSpec, _ := cmd.Flags().GetBool("strict-spec")
	policyPath, _ := cmd.Flags().GetString("policy")
	jsonOut, _ := cmd.Flags().GetBool("json")

	res, err := gate.Evaluate(gate.Options{
		ProjectRoot: root,
		PolicyPath:  policyPath,
		StrictSpec:  strictSpec,
	})
	if err != nil {
		return err
	}

	evidenceID := ""
	if rec, recErr := evidence.NewFromGate(root, res); recErr == nil {
		if writeErr := evidence.Write(root, rec); writeErr == nil {
			evidenceID = rec.ID
		}
	}

	posture := baseline.DefaultPosture(baseline.DetectApprovalsLabel(root))
	doc, err := baseline.Capture(root, res, posture, note, evidenceID)
	if err != nil {
		return err
	}
	if err := baseline.Write(root, doc); err != nil {
		return err
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(doc)
	}
	fmt.Print(baseline.FormatText(doc))
	fmt.Println("Captured. Review .specular/baseline.yaml before treating it as policy.")
	return nil
}

func runBaselineShow(cmd *cobra.Command, _ []string) error {
	root, err := baselineRoot(cmd)
	if err != nil {
		return err
	}
	doc, err := baseline.Load(root)
	if err != nil {
		return err
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(doc)
	}
	fmt.Print(baseline.FormatText(doc))
	return nil
}

func runBaselineStatus(cmd *cobra.Command, _ []string) error {
	root, err := baselineRoot(cmd)
	if err != nil {
		return err
	}
	strictSpec, _ := cmd.Flags().GetBool("strict-spec")
	policyPath, _ := cmd.Flags().GetString("policy")
	jsonOut, _ := cmd.Flags().GetBool("json")

	doc, loadErr := baseline.Load(root)
	res, evalErr := gate.Evaluate(gate.Options{
		ProjectRoot: root,
		PolicyPath:  policyPath,
		StrictSpec:  strictSpec,
	})
	if evalErr != nil {
		return evalErr
	}
	diff := baseline.Compare(doc, res)
	if loadErr != nil {
		diff = baseline.Diff{HasBaseline: false, Summary: "no baseline — run specular baseline capture"}
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(diff)
	}
	fmt.Print(baseline.FormatDiff(diff))
	if diff.Changed {
		return fmt.Errorf("baseline status: %s", diff.Summary)
	}
	return nil
}

func baselineRoot(cmd *cobra.Command) (string, error) {
	root, _ := cmd.Flags().GetString("project-root")
	if root != "" {
		return root, nil
	}
	return os.Getwd()
}

func init() {
	baselineCmd.PersistentFlags().String("project-root", "", "Repository root (default: cwd)")
	baselineCaptureCmd.Flags().String("note", "", "Optional reviewer note")
	baselineCaptureCmd.Flags().String("policy", "", "Policy file for gate evaluation")
	baselineCaptureCmd.Flags().Bool("strict-spec", false, "Require specs during capture evaluation")
	baselineCaptureCmd.Flags().Bool("json", false, "Emit baseline document as JSON")
	baselineShowCmd.Flags().Bool("json", false, "Emit baseline document as JSON")
	baselineStatusCmd.Flags().String("policy", "", "Policy file for live gate evaluation")
	baselineStatusCmd.Flags().Bool("strict-spec", false, "Require specs during status evaluation")
	baselineStatusCmd.Flags().Bool("json", false, "Emit status diff as JSON")
	baselineCmd.AddCommand(baselineCaptureCmd)
	baselineCmd.AddCommand(baselineShowCmd)
	baselineCmd.AddCommand(baselineStatusCmd)
	rootCmd.AddCommand(baselineCmd)
}
