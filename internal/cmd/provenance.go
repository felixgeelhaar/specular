package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/provenance"
)

var provenanceCmd = &cobra.Command{
	Use:   "provenance",
	Short: "Show Agent Provenance Protocol documents",
	Long: `Inspect open Agent Provenance Protocol envelopes (PRODUCT_INTENT §9 / P1 #3).

specular.provenance/v1 maps existing session attestation.Provenance fields
into a stable, agent-neutral document. This is a format + emit/consume path —
not a Specular control plane.

Documents are projected from .specular/sessions/<id>.attestation.json.

Examples:
  specular provenance show
  specular provenance show auth
  specular provenance show --json
`,
}

var provenanceShowCmd = &cobra.Command{
	Use:   "show [session-id]",
	Short: "Show provenance for a session (default: latest attestation)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runProvenanceShow,
}

func runProvenanceShow(cmd *cobra.Command, args []string) error {
	root, err := provenanceProjectRoot(cmd)
	if err != nil {
		return err
	}
	var doc *provenance.Document
	if len(args) == 1 {
		doc, err = provenance.LoadSession(root, args[0])
	} else {
		doc, err = provenance.LoadLatest(root)
	}
	if err != nil {
		return err
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(doc)
	}
	fmt.Print(provenance.FormatHuman(doc))
	return nil
}

func provenanceProjectRoot(cmd *cobra.Command) (string, error) {
	root, _ := cmd.Flags().GetString("project-root")
	if root != "" {
		return root, nil
	}
	return os.Getwd()
}

func init() {
	provenanceCmd.PersistentFlags().String("project-root", "", "Repository root (default: cwd)")
	provenanceShowCmd.Flags().Bool("json", false, "Emit the provenance document as JSON")
	provenanceCmd.AddCommand(provenanceShowCmd)
	rootCmd.AddCommand(provenanceCmd)
}
