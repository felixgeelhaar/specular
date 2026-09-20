package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/policylibrary"
)

var policyLibraryCmd = &cobra.Command{
	Use:   "library",
	Short: "Browse and install open control-mapping seeds",
	Long: `Open policy library: framework control → Specular evidence mappings.

Seeds cover SOC 2, ISO/IEC 42001, EU AI Act, and NIST AI RMF. Install a
fragment into .specular/policies/ and pass it to bundle create --policy.

No Pro license required — the library is the open-core moat.

Examples:
  specular policy library list
  specular policy library show soc2-cc8.1
  specular policy library install soc2-cc8.1
  specular bundle create --policy .specular/policies/soc2-cc8.1.yaml evidence.sbundle.tgz
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var policyLibraryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List embedded policy library seeds",
	RunE:  runPolicyLibraryList,
}

var policyLibraryShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show one library seed (YAML)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPolicyLibraryShow,
}

var policyLibraryInstallCmd = &cobra.Command{
	Use:   "install <id>",
	Short: "Install a library seed into .specular/policies/",
	Args:  cobra.ExactArgs(1),
	RunE:  runPolicyLibraryInstall,
}

func runPolicyLibraryList(cmd *cobra.Command, args []string) error {
	entries, err := policylibrary.List()
	if err != nil {
		return err
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		type row struct {
			ID        string `json:"id"`
			Framework string `json:"framework"`
			Control   string `json:"control"`
			Title     string `json:"title"`
			Version   string `json:"version"`
		}
		rows := make([]row, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, row{
				ID: e.ID, Framework: e.Framework, Control: e.Control,
				Title: e.Title, Version: e.Version,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tFRAMEWORK\tCONTROL\tTITLE")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.ID, e.Framework, e.Control, e.Title)
	}
	return w.Flush()
}

func runPolicyLibraryShow(_ *cobra.Command, args []string) error {
	e, err := policylibrary.Get(args[0])
	if err != nil {
		return err
	}
	if _, writeErr := os.Stdout.Write(e.Raw); writeErr != nil {
		return writeErr
	}
	if len(e.Raw) == 0 || e.Raw[len(e.Raw)-1] != '\n' {
		fmt.Println()
	}
	return nil
}

func runPolicyLibraryInstall(cmd *cobra.Command, args []string) error {
	id := args[0]
	path, _ := cmd.Flags().GetString("path")
	force, _ := cmd.Flags().GetBool("force")
	if path == "" {
		path = policylibrary.DefaultInstallPath(".", id)
	}
	if err := policylibrary.Install(id, path, force); err != nil {
		return err
	}
	e, _ := policylibrary.Get(id)
	fmt.Printf("✓ Installed %s → %s\n", id, path)
	fmt.Println("\nEvidence loop:")
	fmt.Println("  specular session wait --attest --gate")
	fmt.Printf("  specular bundle create --policy %s --include .specular/sessions/*.attestation.json evidence.sbundle.tgz\n", path)
	if e.Framework != "" {
		fmt.Printf("\nMapped: %s %s — %s\n", e.Framework, e.Control, e.Title)
	}
	return nil
}

func init() {
	policyCmd.AddCommand(policyLibraryCmd)
	policyLibraryCmd.AddCommand(policyLibraryListCmd)
	policyLibraryCmd.AddCommand(policyLibraryShowCmd)
	policyLibraryCmd.AddCommand(policyLibraryInstallCmd)

	policyLibraryListCmd.Flags().Bool("json", false, "Emit JSON")
	policyLibraryInstallCmd.Flags().String("path", "", "Install path (default: .specular/policies/<id>.yaml)")
	policyLibraryInstallCmd.Flags().Bool("force", false, "Overwrite existing file")
}
