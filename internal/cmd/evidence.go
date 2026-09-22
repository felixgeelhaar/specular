package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/evidence"
	"github.com/felixgeelhaar/specular/internal/gate"
)

var evidenceCmd = &cobra.Command{
	Use:   "evidence",
	Short: "Show Change Evidence Graph records",
	Long: `Inspect durable evidence records written by specular gate.

Records live under .specular/evidence/ (schema specular.evidence/v1).
This is the first Change Evidence Graph surface — later versions add
intent/session/approval edges. See docs/PRODUCT_INTENT.md §7.

Human show output is an auditor-facing AI CHANGE RECORD (PRODUCT_INTENT §19).
--json emits the unchanged machine-readable record.

Examples:
  specular evidence list
  specular evidence list --verdict DENY --since 24h --limit 20
  specular evidence list --path internal/auth --json
  specular evidence list --risk HIGH --verdict DENY
  specular evidence list --session auth --harness claude
  specular evidence list --soft-allow --verdict ALLOW
  specular evidence list --attested=false
  specular evidence list --governed
  specular evidence list --protocol=false
  specular evidence show
  specular evidence show ev_abc123
  specular evidence show --json
`,
}

var evidenceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List evidence records (newest first)",
	Long: `List local Change Evidence Graph record IDs under .specular/evidence/.

Filters (combinable):
  --verdict ALLOW|DENY              Gate decision
  --since <dur|RFC3339>             CreatedAt at or after (e.g. 24h, 2026-09-01T00:00:00Z)
  --path <substr>                   Root or drift finding path contains substring
  --risk NONE|LOW|MEDIUM|HIGH|CRITICAL  Gate risk level (empty risk treated as NONE)
  --session <id>                    Exact match on gate.provenance.sessions[]
  --harness <substr>                Case-insensitive substring on harnesses[]
  --soft-allow[=true|false]         Exception soft-ALLOW overrules present / absent
  --attested[=true|false]           Gate provenance attested / unattested
  --governed[=true|false]           Gate provenance governed / ungoverned
  --protocol[=true|false]           APP docs present+schema+bound / missing, invalid, or unbound
  --limit N                         Cap results after sorting (newest first)

--json emits a JSON array of matching IDs.
`,
	Args: cobra.NoArgs,
	RunE: runEvidenceList,
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
	filter, err := evidenceListFilter(cmd)
	if err != nil {
		return err
	}
	recs, err := evidence.List(root, filter)
	if err != nil {
		return err
	}
	ids := make([]string, len(recs))
	for i, rec := range recs {
		ids[i] = rec.ID
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(ids)
	}
	if len(ids) == 0 {
		if filter.Active() {
			fmt.Println("No evidence records match filters.")
			return nil
		}
		fmt.Println("No evidence records. Run: specular gate")
		return nil
	}
	for _, id := range ids {
		fmt.Println(id)
	}
	return nil
}

func evidenceListFilter(cmd *cobra.Command) (evidence.ListFilter, error) {
	var f evidence.ListFilter
	verdict, _ := cmd.Flags().GetString("verdict")
	if verdict != "" {
		f.Verdict = gate.Verdict(strings.ToUpper(strings.TrimSpace(verdict)))
	}
	sinceStr, _ := cmd.Flags().GetString("since")
	if sinceStr != "" {
		since, err := evidence.ParseSince(sinceStr, time.Now().UTC())
		if err != nil {
			return f, err
		}
		f.Since = since
	}
	pathSub, _ := cmd.Flags().GetString("path")
	f.PathContains = strings.TrimSpace(pathSub)
	risk, _ := cmd.Flags().GetString("risk")
	f.RiskLevel = strings.TrimSpace(risk)
	session, _ := cmd.Flags().GetString("session")
	f.Session = strings.TrimSpace(session)
	harness, _ := cmd.Flags().GetString("harness")
	f.Harness = strings.TrimSpace(harness)
	if cmd.Flags().Changed("soft-allow") {
		v, _ := cmd.Flags().GetBool("soft-allow")
		f.SoftAllow = &v
	}
	if cmd.Flags().Changed("attested") {
		v, _ := cmd.Flags().GetBool("attested")
		f.Attested = &v
	}
	if cmd.Flags().Changed("governed") {
		v, _ := cmd.Flags().GetBool("governed")
		f.Governed = &v
	}
	if cmd.Flags().Changed("protocol") {
		v, _ := cmd.Flags().GetBool("protocol")
		f.Protocol = &v
	}
	limit, _ := cmd.Flags().GetInt("limit")
	f.Limit = limit
	return f, nil
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
	evidenceListCmd.Flags().String("verdict", "", "Filter by gate verdict (ALLOW or DENY)")
	evidenceListCmd.Flags().String("since", "", "Only records at or after time (duration like 24h, or RFC3339)")
	evidenceListCmd.Flags().String("path", "", "Only records whose root or finding paths contain substring")
	evidenceListCmd.Flags().String("risk", "", "Filter by gate risk level (NONE|LOW|MEDIUM|HIGH|CRITICAL)")
	evidenceListCmd.Flags().String("session", "", "Exact match on gate provenance session id")
	evidenceListCmd.Flags().String("harness", "", "Substring match on gate provenance harness label")
	evidenceListCmd.Flags().Bool("soft-allow", false, "Filter by exception soft-ALLOW overrules (--soft-allow / --soft-allow=false)")
	evidenceListCmd.Flags().Bool("attested", false, "Filter by attested provenance (--attested / --attested=false)")
	evidenceListCmd.Flags().Bool("governed", false, "Filter by governed provenance (--governed / --governed=false)")
	evidenceListCmd.Flags().Bool("protocol", false, "Filter by APP protocol docs OK schema+bound (--protocol / --protocol=false)")
	evidenceListCmd.Flags().Int("limit", 0, "Maximum number of records to return (0 = all)")
	evidenceShowCmd.Flags().Bool("json", false, "Emit the evidence record as JSON")
	evidenceCmd.AddCommand(evidenceListCmd)
	evidenceCmd.AddCommand(evidenceShowCmd)
	rootCmd.AddCommand(evidenceCmd)
}
