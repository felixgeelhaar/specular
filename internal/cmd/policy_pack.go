package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/policylibrary"
)

var policyPackCmd = &cobra.Command{
	Use:   "pack",
	Short: "List, show, and apply executable control packs",
	Long: `Executable control packs (PRODUCT_INTENT §15): open framework control
→ Specular policy/evidence mappings shipped as embedded seeds.

Packs are the same seeds as "policy library"; this surface is the
product-facing control-pack CLI (list / show / apply).

Apply writes a pack fragment into .specular/policies/ for use with
bundle create --policy and gate --policy. Specular does not claim that
applying a pack means an organization is compliant.

Examples:
  specular policy pack list
  specular policy pack show soc2-cc8.1
  specular policy pack show soc2-cc8.1 --json
  specular policy pack apply soc2-cc8.1 --dry-run
  specular policy pack apply soc2-cc8.1
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var policyPackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available control packs",
	RunE:  runPolicyPackList,
}

var policyPackShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show one control pack (human or --json)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPolicyPackShow,
}

var policyPackApplyCmd = &cobra.Command{
	Use:   "apply <id>",
	Short: "Apply a control pack into .specular/policies/",
	Args:  cobra.ExactArgs(1),
	RunE:  runPolicyPackApply,
}

func runPolicyPackList(cmd *cobra.Command, args []string) error {
	entries, err := policylibrary.List()
	if err != nil {
		return err
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		type row struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Summary  string `json:"summary"`
			Framework string `json:"framework"`
			Control  string `json:"control"`
			Version  string `json:"version"`
		}
		rows := make([]row, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, row{
				ID: e.ID, Title: e.Title, Summary: compactSummary(e.Summary),
				Framework: e.Framework, Control: e.Control, Version: e.Version,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSUMMARY")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\n", e.ID, e.Title, truncateSummary(e.Summary, 72))
	}
	return w.Flush()
}

func runPolicyPackShow(cmd *cobra.Command, args []string) error {
	e, err := policylibrary.Get(args[0])
	if err != nil {
		return err
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		type artifact struct {
			Kind string `json:"kind"`
			Path string `json:"path"`
			Why  string `json:"why,omitempty"`
		}
		out := struct {
			ID               string     `json:"id"`
			Framework        string     `json:"framework"`
			Control          string     `json:"control"`
			Title            string     `json:"title"`
			Summary          string     `json:"summary"`
			Version          string     `json:"version"`
			Artifacts        []artifact `json:"artifacts"`
			EvidenceCommands []string   `json:"evidence_commands,omitempty"`
		}{
			ID: e.ID, Framework: e.Framework, Control: e.Control,
			Title: e.Title, Summary: compactSummary(e.Summary), Version: e.Version,
			EvidenceCommands: e.EvidenceCommands,
		}
		for _, a := range e.Artifacts {
			out.Artifacts = append(out.Artifacts, artifact{Kind: a.Kind, Path: a.Path, Why: a.Why})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("ID:          %s\n", e.ID)
	fmt.Printf("Framework:   %s\n", e.Framework)
	fmt.Printf("Control:     %s\n", e.Control)
	fmt.Printf("Title:       %s\n", e.Title)
	fmt.Printf("Version:     %s\n", e.Version)
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Println(" ", compactSummary(e.Summary))
	fmt.Println()
	fmt.Println("Artifacts:")
	for _, a := range e.Artifacts {
		fmt.Printf("  - %s → %s\n", a.Kind, a.Path)
		if a.Why != "" {
			fmt.Printf("      %s\n", a.Why)
		}
	}
	if len(e.EvidenceCommands) > 0 {
		fmt.Println()
		fmt.Println("Evidence commands:")
		for _, c := range e.EvidenceCommands {
			fmt.Printf("  %s\n", c)
		}
	}
	fmt.Println()
	fmt.Printf("Apply: specular policy pack apply %s\n", e.ID)
	return nil
}

func runPolicyPackApply(cmd *cobra.Command, args []string) error {
	id := args[0]
	path, _ := cmd.Flags().GetString("path")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	jsonOut, _ := cmd.Flags().GetBool("json")
	if path == "" {
		path = policylibrary.DefaultInstallPath(".", id)
	}

	res, err := policylibrary.Apply(id, path, policylibrary.ApplyOptions{
		Force:  force,
		DryRun: dryRun,
	})
	if err != nil {
		return err
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}

	e, _ := policylibrary.Get(id)
	if dryRun {
		action := "would create"
		if res.Exists {
			action = "would overwrite"
		}
		fmt.Printf("✓ Dry-run: %s %s → %s (%d bytes)\n", action, id, res.Path, res.Bytes)
		fmt.Println("Run without --dry-run to write the pack fragment.")
		return nil
	}

	fmt.Printf("✓ Applied %s → %s\n", id, res.Path)
	fmt.Println("\nEvidence loop:")
	fmt.Println("  specular session wait --attest --gate")
	fmt.Printf("  specular bundle create --policy %s --include .specular/sessions/*.attestation.json evidence.sbundle.tgz\n", res.Path)
	if e.Framework != "" {
		fmt.Printf("\nMapped: %s %s — %s\n", e.Framework, e.Control, e.Title)
	}
	fmt.Println("\nNote: applying a control pack implements Specular controls and evidence")
	fmt.Println("paths; it does not certify organizational compliance.")
	return nil
}

func compactSummary(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncateSummary(s string, max int) string {
	s = compactSummary(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	if max < 4 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func init() {
	policyCmd.AddCommand(policyPackCmd)
	policyPackCmd.AddCommand(policyPackListCmd)
	policyPackCmd.AddCommand(policyPackShowCmd)
	policyPackCmd.AddCommand(policyPackApplyCmd)

	policyPackListCmd.Flags().Bool("json", false, "Emit JSON")
	policyPackShowCmd.Flags().Bool("json", false, "Emit JSON")
	policyPackApplyCmd.Flags().String("path", "", "Destination path (default: .specular/policies/<id>.yaml)")
	policyPackApplyCmd.Flags().Bool("force", false, "Overwrite existing file")
	policyPackApplyCmd.Flags().Bool("dry-run", false, "Show what would be written without writing")
	policyPackApplyCmd.Flags().Bool("json", false, "Emit JSON result")
}
