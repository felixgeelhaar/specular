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
	Short: "Show and verify Agent Provenance Protocol documents",
	Long: `Inspect open Agent Provenance Protocol envelopes (PRODUCT_INTENT §9 / P1 #3).

specular.provenance/v1 maps existing session attestation.Provenance fields
into a stable, agent-neutral document. session attest emits
.specular/sessions/<id>.provenance.json beside the attestation.
provenance verify checks schema/required fields (not cryptographic
signatures — use specular auto verify for those).

Examples:
  specular provenance show
  specular provenance show auth
  specular provenance show --json
  specular provenance verify
  specular provenance verify auth --json
`,
}

var provenanceShowCmd = &cobra.Command{
	Use:   "show [session-id|path]",
	Short: "Show provenance for a session (default: latest attestation)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runProvenanceShow,
}

var provenanceVerifyCmd = &cobra.Command{
	Use:   "verify [session-id|path]",
	Short: "Verify Agent Provenance Protocol schema (not signatures)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runProvenanceVerify,
}

func runProvenanceShow(cmd *cobra.Command, args []string) error {
	root, err := provenanceProjectRoot(cmd)
	if err != nil {
		return err
	}
	target := ""
	if len(args) == 1 {
		target = args[0]
	}
	doc, err := provenance.Resolve(root, target)
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

func runProvenanceVerify(cmd *cobra.Command, args []string) error {
	root, err := provenanceProjectRoot(cmd)
	if err != nil {
		return err
	}
	target := ""
	if len(args) == 1 {
		target = args[0]
	}
	doc, err := provenance.Resolve(root, target)
	if err != nil {
		return err
	}
	res := provenance.Validate(doc)
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if encErr := enc.Encode(res); encErr != nil {
			return encErr
		}
	} else {
		fmt.Print(provenance.FormatVerifyHuman(res))
	}
	if !res.OK {
		return fmt.Errorf("provenance verify failed (%d error(s))", len(res.Errors))
	}
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
	provenanceVerifyCmd.Flags().Bool("json", false, "Emit verify result as JSON")
	provenanceCmd.AddCommand(provenanceShowCmd)
	provenanceCmd.AddCommand(provenanceVerifyCmd)
	rootCmd.AddCommand(provenanceCmd)
}
