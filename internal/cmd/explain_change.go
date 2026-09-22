package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/evidence"
	"github.com/felixgeelhaar/specular/internal/gate"
)

// changeExplainCmd is the top-level PRODUCT_INTENT §20 surface:
// "Why did Specular make this decision?" Routing explain remains under
// `specular debug explain`.
var changeExplainCmd = &cobra.Command{
	Use:   "explain [evidence-id|commit-prefix]",
	Short: "Explain a gate ALLOW / DENY decision",
	Long: `Explain why Specular allowed or denied a change.

Default: load the latest Change Evidence Graph record written by
specular gate. Pass an evidence id (ev_…) or a git commit prefix (e.g.
abc123) to select a record. With --fresh, re-run the gate (and persist
evidence) before explaining.

Graph filters (newest match):
  --file <substr>     root / drift finding paths (evidence list --path)
  --control <substr>  failed checks / exception policy / soft-ALLOW bind
  --commit <prefix>   record / gate.change commit SHA prefix
  --session <id>      exact gate.provenance.sessions[] match (fleet board id)

Human text is an auditor-facing AI CHANGE RECORD (PRODUCT_INTENT §19).
--json emits the unchanged machine-readable evidence record.

Routing / model-selection explainability remains at:
  specular debug explain <checkpoint-id>

See docs/PRODUCT_INTENT.md §7 (Change Evidence Graph) and §20 (Explain).

Examples:
  specular explain
  specular explain ev_abc123
  specular explain abc123
  specular explain --commit abc123
  specular explain --session auth
  specular explain --file internal/auth/token.go
  specular explain --control SEC-17
  specular explain --fresh --policy-file .specular/policy.yaml
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
	policyPath := explainPolicyFile(cmd)
	fileFilter, _ := cmd.Flags().GetString("file")
	controlFilter, _ := cmd.Flags().GetString("control")
	commitFilter, _ := cmd.Flags().GetString("commit")
	sessionFilter, _ := cmd.Flags().GetString("session")

	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		projectRoot = cwd
	}

	fileFilter = strings.TrimSpace(fileFilter)
	controlFilter = strings.TrimSpace(controlFilter)
	commitFilter = strings.TrimSpace(commitFilter)
	sessionFilter = strings.TrimSpace(sessionFilter)

	// Positional short SHA → --commit (PRODUCT_INTENT §20: explain abc123).
	if len(args) == 1 && commitFilter == "" && looksLikeCommitPrefix(args[0]) {
		commitFilter = args[0]
		args = nil
	}

	if err := explainGraphFiltersExclusive([]explainNamedFilter{
		{name: "file", value: fileFilter},
		{name: "control", value: controlFilter},
		{name: "commit", value: commitFilter},
		{name: "session", value: sessionFilter},
	}, fresh, args); err != nil {
		return err
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
	case fileFilter != "":
		rec, err = loadEvidenceByFilter(projectRoot, evidence.ListFilter{
			PathContains: fileFilter,
			Limit:        1,
		}, "file", fileFilter)
	case controlFilter != "":
		rec, err = loadEvidenceByFilter(projectRoot, evidence.ListFilter{
			ControlContains: controlFilter,
			Limit:           1,
		}, "control", controlFilter)
	case commitFilter != "":
		rec, err = loadEvidenceByFilter(projectRoot, evidence.ListFilter{
			CommitPrefix: commitFilter,
			Limit:        1,
		}, "commit", commitFilter)
	case sessionFilter != "":
		rec, err = loadEvidenceByFilter(projectRoot, evidence.ListFilter{
			Session: sessionFilter,
			Limit:   1,
		}, "session", sessionFilter)
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

type explainNamedFilter struct {
	name  string
	value string
}

func explainGraphFiltersExclusive(filters []explainNamedFilter, fresh bool, args []string) error {
	var active []string
	for _, f := range filters {
		if f.value != "" {
			active = append(active, "--"+f.name)
		}
	}
	if len(active) > 1 {
		return fmt.Errorf("explain: %s are mutually exclusive", strings.Join(active, ", "))
	}
	if len(active) == 0 {
		return nil
	}
	flag := active[0]
	if fresh {
		return fmt.Errorf("explain: %s cannot be combined with --fresh", flag)
	}
	if len(args) == 1 {
		return fmt.Errorf("explain: %s cannot be combined with evidence-id", flag)
	}
	return nil
}

// looksLikeCommitPrefix reports whether s is a hex SHA prefix (not an ev_ id).
func looksLikeCommitPrefix(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(strings.ToLower(s), "ev_") {
		return false
	}
	if len(s) < 4 || len(s) > 40 {
		return false
	}
	for _, r := range s {
		if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
			return false
		}
	}
	return true
}

// explainPolicyFile resolves --policy-file, falling back to deprecated --policy.
func explainPolicyFile(cmd *cobra.Command) string {
	if cmd.Flags().Changed("policy-file") {
		v, _ := cmd.Flags().GetString("policy-file")
		return strings.TrimSpace(v)
	}
	if cmd.Flags().Changed("policy") {
		v, _ := cmd.Flags().GetString("policy")
		return strings.TrimSpace(v)
	}
	v, _ := cmd.Flags().GetString("policy-file")
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	v, _ = cmd.Flags().GetString("policy")
	return strings.TrimSpace(v)
}

func loadEvidenceByFilter(projectRoot string, filter evidence.ListFilter, flag, substr string) (*evidence.Record, error) {
	recs, err := evidence.List(projectRoot, filter)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		listFlag := flag
		if flag == "file" {
			listFlag = "path"
		}
		return nil, fmt.Errorf("explain: no evidence matches --%s %q (try: specular evidence list --%s %q)", flag, substr, listFlag, substr)
	}
	return recs[0], nil
}

// loadEvidenceByFile kept for tests; prefer loadEvidenceByFilter.
func loadEvidenceByFile(projectRoot, substr string) (*evidence.Record, error) {
	return loadEvidenceByFilter(projectRoot, evidence.ListFilter{
		PathContains: substr,
		Limit:        1,
	}, "file", substr)
}

func init() {
	changeExplainCmd.Flags().String("project-root", "", "Repository root (default: cwd)")
	changeExplainCmd.Flags().String("policy-file", "", "Policy file when using --fresh")
	changeExplainCmd.Flags().String("policy", "", "Deprecated alias for --policy-file")
	_ = changeExplainCmd.Flags().MarkDeprecated("policy", "use --policy-file")
	changeExplainCmd.Flags().Bool("strict-spec", false, "Require specs when using --fresh")
	changeExplainCmd.Flags().Bool("fresh", false, "Re-run specular gate before explaining")
	changeExplainCmd.Flags().String("file", "", "Explain newest evidence whose paths contain this substring (PRODUCT_INTENT §20)")
	changeExplainCmd.Flags().String("control", "", "Explain newest evidence matching failed check / exception policy / soft-ALLOW bind (PRODUCT_INTENT §20)")
	changeExplainCmd.Flags().String("commit", "", "Explain newest evidence whose commit SHA starts with this prefix (PRODUCT_INTENT §20)")
	changeExplainCmd.Flags().String("session", "", "Explain newest evidence for this session id (gate.provenance.sessions[]; PRODUCT_INTENT §20)")
	changeExplainCmd.Flags().Bool("json", false, "Emit the evidence record as JSON")
	rootCmd.AddCommand(changeExplainCmd)
}
