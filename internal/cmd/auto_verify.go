package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/attestation"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <attestation-file>",
	Short: "Verify cryptographic attestation of workflow execution",
	Long: `Verify the cryptographic signature and provenance data in an attestation file.

Attestations provide cryptographic proof that a workflow was executed by a specific
identity, with verifiable hashes of the execution plan and output.

Examples:
  # Verify an attestation
  specular auto verify ~/.specular/attestations/auto-1762811730.attestation.json

  # Verify with strict options
  specular auto verify attestation.json --max-age 24h --require-clean-git

  # Verify with allowed identities
  specular auto verify attestation.json --allowed-identity user@example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		attestationPath := args[0]

		// Parse flags
		maxAge, _ := cmd.Flags().GetDuration("max-age")
		requireCleanGit, _ := cmd.Flags().GetBool("require-clean-git")
		allowedIdentities, _ := cmd.Flags().GetStringSlice("allowed-identity")
		verifyHashes, _ := cmd.Flags().GetBool("verify-hashes")
		planPath, _ := cmd.Flags().GetString("plan")
		outputPath, _ := cmd.Flags().GetString("output")

		fmt.Printf("🔍 Verifying attestation: %s\n\n", attestationPath)

		// Read attestation file
		attestationData, err := os.ReadFile(attestationPath)
		if err != nil {
			return fmt.Errorf("failed to read attestation: %w", err)
		}

		// Parse attestation
		att, err := attestation.FromJSON(attestationData)
		if err != nil {
			return fmt.Errorf("failed to parse attestation: %w", err)
		}

		// Display attestation info
		fmt.Println("📋 Attestation Information:")
		fmt.Printf("   Version:     %s\n", att.Version)
		fmt.Printf("   Workflow ID: %s\n", att.WorkflowID)
		fmt.Printf("   Goal:        %s\n", att.Goal)
		fmt.Printf("   Status:      %s\n", att.Status)
		fmt.Printf("   Duration:    %s\n", att.Duration)
		fmt.Printf("   Signed by:   %s\n", att.SignedBy)
		fmt.Printf("   Signed at:   %s\n", att.SignedAt.Format(time.RFC3339))
		fmt.Println()

		fmt.Println("🖥️  Provenance:")
		fmt.Printf("   Hostname: %s\n", att.Provenance.Hostname)
		fmt.Printf("   Platform: %s/%s\n", att.Provenance.Platform, att.Provenance.Arch)
		fmt.Printf("   Specular: %s\n", att.Provenance.SpecularVersion)
		fmt.Printf("   Profile:  %s\n", att.Provenance.Profile)
		if att.Provenance.GitRepo != "" {
			fmt.Printf("   Git:      %s@%s", att.Provenance.GitRepo, att.Provenance.GitCommit[:8])
			if att.Provenance.GitDirty {
				fmt.Printf(" (dirty)")
			}
			fmt.Println()
		}
		fmt.Printf("   Cost:     $%.4f\n", att.Provenance.TotalCost)
		fmt.Printf("   Tasks:    %d executed, %d failed\n", att.Provenance.TasksExecuted, att.Provenance.TasksFailed)
		fmt.Println()

		// Create verifier with options
		var opts []attestation.VerifierOption
		if maxAge > 0 {
			opts = append(opts, attestation.WithMaxAge(maxAge))
		}
		if requireCleanGit {
			opts = append(opts, attestation.WithRequireGitClean(true))
		}
		if len(allowedIdentities) > 0 {
			opts = append(opts, attestation.WithAllowedIdentities(allowedIdentities))
		}

		verifier := attestation.NewStandardVerifier(opts...)

		// Verify signature
		fmt.Println("🔐 Verifying signature...")
		if err := verifier.Verify(att); err != nil {
			fmt.Printf("❌ Signature verification failed: %v\n", err)
			return fmt.Errorf("signature verification failed")
		}
		fmt.Println("✅ Signature valid")
		fmt.Println()

		// Verify provenance
		fmt.Println("📊 Verifying provenance...")
		if err := verifier.VerifyProvenance(att); err != nil {
			fmt.Printf("❌ Provenance verification failed: %v\n", err)
			return fmt.Errorf("provenance verification failed")
		}
		fmt.Println("✅ Provenance valid")
		fmt.Println()

		// Verify hashes if requested
		if verifyHashes {
			if planPath == "" || outputPath == "" {
				return fmt.Errorf("--plan and --output required when --verify-hashes is set")
			}

			fmt.Println("🔢 Verifying hashes...")

			// Read plan file
			planData, err := os.ReadFile(planPath)
			if err != nil {
				return fmt.Errorf("failed to read plan file: %w", err)
			}

			// Read output file
			outputData, err := os.ReadFile(outputPath)
			if err != nil {
				return fmt.Errorf("failed to read output file: %w", err)
			}

			// Verify hashes
			if err := verifier.VerifyHashes(att, planData, outputData); err != nil {
				fmt.Printf("❌ Hash verification failed: %v\n", err)
				return fmt.Errorf("hash verification failed")
			}
			fmt.Println("✅ Hashes valid")
			fmt.Println()
		}

		fmt.Println("🎉 Attestation verified successfully!")
		return nil
	},
}

func init() {
	verifyCmd.Flags().Duration("max-age", 0, "Maximum age for attestation (e.g., 24h, 7d)")
	verifyCmd.Flags().Bool("require-clean-git", false, "Require clean git status in provenance")
	verifyCmd.Flags().StringSlice("allowed-identity", []string{}, "Allowed signer identities (can be used multiple times)")
	verifyCmd.Flags().Bool("verify-hashes", false, "Verify plan and output hashes")
	verifyCmd.Flags().String("plan", "", "Path to plan JSON file for hash verification")
	verifyCmd.Flags().String("output", "", "Path to output JSON file for hash verification")

	autoCmd.AddCommand(verifyCmd)
}
