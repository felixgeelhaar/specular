package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/evidence"
	"github.com/felixgeelhaar/specular/internal/gate"
)

var gateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Evaluate a proposed change (ALLOW / DENY)",
	Long: `Specular's primary change-control primitive.

Discovers the working-tree change, summarizes provenance, evaluates drift
when Specular specs exist, and runs policy verification when a policy file
is present. Prints an explainable ALLOW / DENY verdict and persists a
Change Evidence Graph record under .specular/evidence/ (unless
--no-evidence). Inspect with specular explain / specular evidence show.

Output formats:
  text       Human board (default)
  json       Machine-readable Result
  markdown   PR / GITHUB_STEP_SUMMARY body (stable "## Specular Change Control" marker)

--github-annotations emits ::error/::warning/::notice workflow commands for
CI file annotations (PRODUCT_INTENT §21).

Brownfield: repositories without .specular/spec skip drift (unless
--strict-spec). Missing policy skips verification. Unattested provenance
is reported explicitly — never silently treated as verified.

Exit codes:
  0  ALLOW
  3  DENY (policy)
  4  DENY (drift)
  1  other errors

See docs/PRODUCT_INTENT.md.

Examples:
  specular gate
  specular gate --json
  specular gate --format markdown
  specular gate --format markdown --github-annotations
  specular gate --strict-spec
  specular gate --policy .specular/policy.yaml
  specular gate --no-evidence
`,
	Args: cobra.NoArgs,
	RunE: runGate,
}

func runGate(cmd *cobra.Command, _ []string) error {
	projectRoot, _ := cmd.Flags().GetString("project-root")
	policyPath, _ := cmd.Flags().GetString("policy")
	reportFile, _ := cmd.Flags().GetString("report")
	strictSpec, _ := cmd.Flags().GetBool("strict-spec")
	jsonOut, _ := cmd.Flags().GetBool("json")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noEvidence, _ := cmd.Flags().GetBool("no-evidence")
	format, _ := cmd.Flags().GetString("format")
	ghAnnotations, _ := cmd.Flags().GetBool("github-annotations")

	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		projectRoot = cwd
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if jsonOut {
		format = "json"
	}
	if format == "" {
		format = "text"
	}

	res, err := gate.Evaluate(gate.Options{
		ProjectRoot: projectRoot,
		PolicyPath:  policyPath,
		ReportFile:  reportFile,
		StrictSpec:  strictSpec,
	})
	if err != nil {
		return err
	}

	evidenceID := ""
	if !noEvidence {
		if rec, recErr := evidence.NewFromGate(projectRoot, res); recErr == nil {
			if writeErr := evidence.Write(projectRoot, rec); writeErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not persist evidence: %v\n", writeErr)
			} else {
				evidenceID = rec.ID
				if !quiet && format == "text" {
					fmt.Fprintf(os.Stderr, "Evidence: %s (.specular/evidence/)\n", rec.ID)
				}
			}
		}
	}

	if ghAnnotations {
		fmt.Fprint(os.Stderr, gate.FormatGitHubAnnotations(gate.AnnotationsFromResult(res)))
	}

	if !quiet {
		switch format {
		case "json":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(res)
		case "markdown", "md":
			fmt.Print(gate.FormatMarkdownWith(res, gate.FormatMarkdownOptions{
				EvidenceID: evidenceID,
			}))
		default:
			fmt.Print(gate.FormatText(res))
		}
	}

	if res.Verdict == gate.Deny {
		if res.Drift.Status == gate.StatusFail {
			return fmt.Errorf("drift detection failed: %s", res.Reason)
		}
		if res.Policy.Status == gate.StatusFail {
			return fmt.Errorf("policy violation: %s", res.Reason)
		}
		if res.Risk.Enforced && len(res.Risk.Missing) > 0 {
			return fmt.Errorf("policy violation: %s", res.Reason)
		}
		return fmt.Errorf("gate denied: %s", res.Reason)
	}
	return nil
}

func init() {
	gateCmd.Flags().String("project-root", "", "Repository root (default: cwd)")
	gateCmd.Flags().String("policy", "", "Policy file (default: .specular/policy.yaml if present)")
	gateCmd.Flags().String("report", "drift.sarif", "Drift SARIF output path when drift runs")
	gateCmd.Flags().Bool("strict-spec", false, "Fail when Specular spec/plan/lock are missing")
	gateCmd.Flags().String("format", "text", "Output format: text, json, markdown")
	gateCmd.Flags().Bool("json", false, "Emit machine-readable JSON (alias for --format json)")
	gateCmd.Flags().Bool("github-annotations", false, "Emit GitHub Actions ::error/::warning annotations to stderr")
	gateCmd.Flags().BoolP("quiet", "q", false, "Suppress human board (exit code still set)")
	gateCmd.Flags().Bool("no-evidence", false, "Skip writing .specular/evidence/ record")
	rootCmd.AddCommand(gateCmd)
}
