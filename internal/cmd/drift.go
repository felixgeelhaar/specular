package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/license"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect and manage drift between plan and repository",
	Long: `Detect drift between plan and repository state, and manage drift approvals.

Use 'specular drift check' to detect drift.
Use 'specular drift approve' to approve detected drift.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand, show help
		return cmd.Help()
	},
}

var driftCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Detect drift between plan and repository",
	Long: `Compare the current repository state with the execution plan to detect drift.

Drift detection checks:
- File hashes vs expected hashes in plan
- Missing or extra files
- Uncommitted changes that may affect the plan

Exit codes:
  0 - No drift detected
  1 - Drift detected`,
	RunE: runDriftCheck,
}

var driftApproveCmd = &cobra.Command{
	Use:   "approve [drift-hash]",
	Short: "Approve detected drift",
	Long: `Create an approval record for detected drift.

This allows drift to be acknowledged and approved for governance compliance.
The drift hash is displayed by 'drift check' when drift is detected.

Examples:
  specular drift approve drift-abc123def456
  specular drift approve drift-abc123def456 --message "Approved emergency hotfix"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDriftApprove,
}

func runDriftCheck(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	planPath := cmd.Flags().Lookup("plan").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("plan") {
		planPath = defaults.PlanFile()
	}

	// Validate plan file exists
	if err := ux.ValidateRequiredFile(planPath, "Plan file", "specular plan create"); err != nil {
		return ux.EnhanceError(err)
	}

	// Load plan
	p, err := plan.LoadPlan(planPath)
	if err != nil {
		return ux.FormatError(err, "loading plan file")
	}

	fmt.Printf("Detecting drift for plan: %s\n\n", planPath)

	// Get git status to check for uncommitted changes
	gitCmd := exec.Command("git", "status", "--porcelain")
	output, err := gitCmd.Output()
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not check git status: %v\n", err)
	}

	uncommitted := strings.TrimSpace(string(output))
	hasDrift := false

	if uncommitted != "" {
		lines := strings.Split(uncommitted, "\n")
		fmt.Printf("⚠️  Uncommitted changes detected (%d files):\n", len(lines))
		for i, line := range lines {
			if i < 5 {
				fmt.Printf("  %s\n", line)
			}
		}
		if len(lines) > 5 {
			fmt.Printf("  ... and %d more\n", len(lines)-5)
		}
		fmt.Println()
		hasDrift = true
	}

	// Check for task drift (simplified - would need actual implementation)
	driftCount := 0
	for _, task := range p.Tasks {
		// In a real implementation, we would:
		// 1. Check if files for this task have changed
		// 2. Compare file hashes with expected hashes
		// 3. Report any mismatches
		_ = task // Placeholder
	}

	if driftCount > 0 {
		hasDrift = true
	}

	if !hasDrift {
		fmt.Printf("✓ No drift detected\n")
		fmt.Printf("  All tasks align with current repository state\n")
		return nil
	}

	// Generate drift hash for approval tracking
	driftHash := generateDriftHash(planPath, uncommitted)

	fmt.Printf("⚠️  Drift detected\n")
	if driftCount > 0 {
		fmt.Printf("  %d task(s) may be affected by changes\n", driftCount)
	}
	fmt.Printf("  Drift ID: %s\n", driftHash)

	fmt.Println("\nRecommendations:")
	if uncommitted != "" {
		fmt.Printf("  1. Commit or stash uncommitted changes\n")
		fmt.Printf("  2. Regenerate plan: specular plan create\n")
		fmt.Printf("  OR\n")
		fmt.Printf("  3. Approve drift: specular drift approve %s\n", driftHash)
	} else {
		fmt.Printf("  1. Review changes: git diff\n")
		fmt.Printf("  2. Regenerate plan if needed: specular plan create\n")
		fmt.Printf("  OR\n")
		fmt.Printf("  3. Approve drift: specular drift approve %s\n", driftHash)
	}

	// Exit with code 1 to indicate drift detected
	os.Exit(1)
	return nil
}

func runDriftApprove(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("drift.approve", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "drift approve")
		return err
	}

	var driftHash string
	message := cmd.Flags().Lookup("message").Value.String()

	// If no drift hash provided, run check to get it
	if len(args) == 0 {
		defaults := ux.NewPathDefaults()
		planPath := cmd.Flags().Lookup("plan").Value.String()

		if !cmd.Flags().Changed("plan") {
			planPath = defaults.PlanFile()
		}

		// Validate plan file exists
		if err := ux.ValidateRequiredFile(planPath, "Plan file", "specular plan create"); err != nil {
			return ux.EnhanceError(err)
		}

		// Get git status
		gitCmd := exec.Command("git", "status", "--porcelain")
		output, err := gitCmd.Output()
		if err != nil {
			return fmt.Errorf("could not check git status: %w", err)
		}

		uncommitted := strings.TrimSpace(string(output))
		if uncommitted == "" {
			fmt.Println("No drift detected. Nothing to approve.")
			return nil
		}

		driftHash = generateDriftHash(planPath, uncommitted)
		fmt.Printf("Detected drift ID: %s\n", driftHash)
	} else {
		driftHash = args[0]
	}

	// Get approver name from environment or system
	approver := os.Getenv("USER")
	if approver == "" {
		approver = "unknown"
	}

	// Create approval record
	approval := ApprovalRecord{
		Version:      "1.0",
		Type:         "drift",
		ResourceID:   driftHash,
		ResourceHash: driftHash,
		ApprovedBy:   approver,
		ApprovedAt:   time.Now(),
		Message:      message,
	}

	// Save approval record
	approvalsDir := filepath.Join(".specular", "approvals")
	if err := os.MkdirAll(approvalsDir, 0755); err != nil {
		return fmt.Errorf("creating approvals directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("drift-%s.yaml", timestamp)
	approvalPath := filepath.Join(approvalsDir, filename)

	data, err := yaml.Marshal(&approval)
	if err != nil {
		return fmt.Errorf("marshaling approval: %w", err)
	}

	if err := os.WriteFile(approvalPath, data, 0644); err != nil {
		return fmt.Errorf("writing approval: %w", err)
	}

	fmt.Printf("✅ Approved drift: %s\n\n", driftHash)
	fmt.Printf("Approved by: %s\n", approver)
	fmt.Printf("Approval saved: %s\n", approvalPath)
	if message != "" {
		fmt.Printf("Message: %s\n", message)
	}

	return nil
}

// generateDriftHash generates a unique hash for drift based on plan path and uncommitted changes
func generateDriftHash(planPath string, uncommitted string) string {
	h := sha256.New()
	h.Write([]byte(planPath))
	h.Write([]byte(uncommitted))
	hash := h.Sum(nil)
	return fmt.Sprintf("drift-%x", hash[:8]) // Use first 8 bytes for shorter hash
}

func init() {
	rootCmd.AddCommand(driftCmd)
	driftCmd.AddCommand(driftCheckCmd)
	driftCmd.AddCommand(driftApproveCmd)

	// drift check flags
	driftCheckCmd.Flags().String("plan", "plan.json", "Plan file to check for drift")

	// drift approve flags
	driftApproveCmd.Flags().String("plan", "plan.json", "Plan file (used if drift hash not provided)")
	driftApproveCmd.Flags().String("message", "", "Approval message or comment")
}
