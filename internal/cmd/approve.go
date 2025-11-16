package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// Approval represents a governance approval signature
type Approval struct {
	Artifact    string    `json:"artifact"`    // spec, plan, or bundle
	Path        string    `json:"path"`        // file path
	Hash        string    `json:"hash"`        // SHA256 hash of content
	ApprovedBy  string    `json:"approved_by"` // user/email
	ApprovedAt  time.Time `json:"approved_at"` // timestamp
	Comment     string    `json:"comment"`     // optional comment
	Environment string    `json:"environment"` // dev, staging, prod
}

var approveCmd = &cobra.Command{
	Use:   "approve [artifact]",
	Short: "Approve an artifact with governance signature",
	Long: `Add a governance approval signature to a spec, plan, or bundle.

Approvals create a cryptographic signature (SHA256 hash) of the artifact
along with metadata about who approved it and when.

Artifacts:
  spec      Approve a specification file
  plan      Approve an execution plan
  bundle    Approve a complete bundle (spec + plan)

The approval is stored in .specular/approvals/ and includes:
- SHA256 hash of the artifact
- Approver identity (from USER env var or --approver flag)
- Timestamp
- Optional comment
- Environment (dev, staging, prod)

Examples:
  specular approve spec --file spec.yaml
  specular approve plan --file plan.yaml --approver alice@example.com
  specular approve bundle --file bundle.tar.gz --comment "Reviewed and approved" --env prod`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		artifactType := args[0]

		// Validate artifact type
		validTypes := map[string]bool{
			"spec":   true,
			"plan":   true,
			"bundle": true,
		}

		if !validTypes[artifactType] {
			return fmt.Errorf("invalid artifact type '%s'. Valid types: spec, plan, bundle", artifactType)
		}

		// Get flags
		filePath, _ := cmd.Flags().GetString("file")
		approver, _ := cmd.Flags().GetString("approver")
		comment, _ := cmd.Flags().GetString("comment")
		environment, _ := cmd.Flags().GetString("env")

		// Validate required flags
		if filePath == "" {
			return fmt.Errorf("--file is required")
		}

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}

		// Read file and compute hash
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		hash := sha256.Sum256(data)
		hashStr := hex.EncodeToString(hash[:])

		// Get approver identity
		if approver == "" {
			approver = os.Getenv("USER")
			if approver == "" {
				approver = "unknown"
			}
		}

		// Create approval
		approval := Approval{
			Artifact:    artifactType,
			Path:        filePath,
			Hash:        hashStr,
			ApprovedBy:  approver,
			ApprovedAt:  time.Now(),
			Comment:     comment,
			Environment: environment,
		}

		// Create approvals directory
		approvalsDir := ".specular/approvals"
		if err := os.MkdirAll(approvalsDir, 0755); err != nil {
			return fmt.Errorf("failed to create approvals directory: %w", err)
		}

		// Generate approval filename
		timestamp := approval.ApprovedAt.Format("20060102-150405")
		approvalFile := filepath.Join(approvalsDir, fmt.Sprintf("%s-%s-%s.json", artifactType, timestamp, approver))

		// Save approval
		approvalData, err := json.MarshalIndent(approval, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal approval: %w", err)
		}

		if err := os.WriteFile(approvalFile, approvalData, 0644); err != nil {
			return fmt.Errorf("failed to write approval: %w", err)
		}

		// Display approval details
		fmt.Printf("✅ Approved %s: %s\n", artifactType, filePath)
		fmt.Println()
		fmt.Println("Approval Details:")
		fmt.Printf("  Hash: %s\n", hashStr[:16]+"...")
		fmt.Printf("  Approved by: %s\n", approver)
		fmt.Printf("  Approved at: %s\n", approval.ApprovedAt.Format("2006-01-02 15:04:05"))
		if comment != "" {
			fmt.Printf("  Comment: %s\n", comment)
		}
		if environment != "" {
			fmt.Printf("  Environment: %s\n", environment)
		}
		fmt.Println()
		fmt.Printf("Approval saved to: %s\n", approvalFile)

		return nil
	},
}

func init() {
	// Flags for approve command
	approveCmd.Flags().String("file", "", "Path to artifact file (required)")
	approveCmd.Flags().String("approver", "", "Approver identity (default: $USER)")
	approveCmd.Flags().String("comment", "", "Approval comment")
	approveCmd.Flags().String("env", "dev", "Environment (dev, staging, prod)")

	rootCmd.AddCommand(approveCmd)
}
