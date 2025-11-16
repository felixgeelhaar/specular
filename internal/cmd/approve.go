package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/license"
)

var approveCmd = &cobra.Command{
	Use:   "approve <resource>",
	Short: "Approve bundle, drift, or other governance resource",
	Long: `Create an approval record for a governance resource.

Resources can be:
  • Bundle ID (from bundle create)
  • Drift hash (from drift check)
  • Policy change (from policy diff)

Approvals are stored in .specular/approvals/ with timestamps and approver info.

Examples:
  specular approve bundle-abc123
  specular approve drift-def456
  specular approve policy-change --message "Approved security update"`,
	Args: cobra.ExactArgs(1),
	RunE: runApprove,
}

var approvalsCmd = &cobra.Command{
	Use:   "approvals",
	Short: "Manage approval records",
	Long:  `List and manage approval records for bundles, drift, and policies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var approvalsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all approval records",
	Long: `Display all approval records with details.

Shows:
  • Approval type (bundle, drift, policy)
  • Resource ID
  • Approver name and timestamp
  • Approval message/comment
  • Approval status`,
	RunE: runApprovalsList,
}

var approvalsPendingCmd = &cobra.Command{
	Use:   "pending",
	Short: "Show pending approvals",
	Long: `Display resources that require approval but don't have one yet.

Checks:
  • Unapproved policy changes (from policy diff)
  • Unapproved bundles (from bundle create)
  • Unapproved drift (from drift check)

Exit codes:
  0: No pending approvals
  1: Pending approvals found`,
	RunE: runApprovalsPending,
}

// ApprovalRecord represents a governance approval record
type ApprovalRecord struct {
	Version      string    `yaml:"version"`
	Type         string    `yaml:"type"` // "bundle", "drift", "policy", "plan"
	ResourceID   string    `yaml:"resource_id"`
	ResourceHash string    `yaml:"resource_hash,omitempty"`
	ApprovedBy   string    `yaml:"approved_by"`
	ApprovedAt   time.Time `yaml:"approved_at"`
	Message      string    `yaml:"message,omitempty"`
	Metadata     map[string]string `yaml:"metadata,omitempty"`
}

func runApprove(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("approvals.create", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "approve")
		return err
	}

	resourceID := args[0]
	message := cmd.Flags().Lookup("message").Value.String()

	// Determine resource type from ID prefix
	var resourceType string
	switch {
	case strings.HasPrefix(resourceID, "bundle-"):
		resourceType = "bundle"
	case strings.HasPrefix(resourceID, "drift-"):
		resourceType = "drift"
	case strings.HasPrefix(resourceID, "policy-"):
		resourceType = "policy"
	case strings.HasPrefix(resourceID, "plan-"):
		resourceType = "plan"
	default:
		return fmt.Errorf("unknown resource type: %s (expected bundle-*, drift-*, policy-*, or plan-*)", resourceID)
	}

	// Get approver name from environment or system
	approver := os.Getenv("USER")
	if approver == "" {
		approver = "unknown"
	}

	// Create approval record
	approval := ApprovalRecord{
		Version:    "1.0",
		Type:       resourceType,
		ResourceID: resourceID,
		ApprovedBy: approver,
		ApprovedAt: time.Now(),
		Message:    message,
	}

	// Save approval record
	approvalsDir := filepath.Join(".specular", "approvals")
	if err := os.MkdirAll(approvalsDir, 0755); err != nil {
		return fmt.Errorf("creating approvals directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.yaml", resourceType, timestamp)
	approvalPath := filepath.Join(approvalsDir, filename)

	data, err := yaml.Marshal(&approval)
	if err != nil {
		return fmt.Errorf("marshaling approval: %w", err)
	}

	if err := os.WriteFile(approvalPath, data, 0644); err != nil {
		return fmt.Errorf("writing approval: %w", err)
	}

	fmt.Printf("✅ Approved %s: %s\n\n", resourceType, resourceID)
	fmt.Printf("Approved by: %s\n", approver)
	fmt.Printf("Approval saved: %s\n", approvalPath)
	if message != "" {
		fmt.Printf("Message: %s\n", message)
	}

	return nil
}

func runApprovalsList(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("approvals.list", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "approvals list")
		return err
	}

	approvalsDir := filepath.Join(".specular", "approvals")

	// Check if approvals directory exists
	if _, err := os.Stat(approvalsDir); os.IsNotExist(err) {
		fmt.Println("No approval records found.")
		fmt.Println("\nRun 'specular governance init' to create the governance workspace.")
		return nil
	}

	// Read all approval files
	entries, err := os.ReadDir(approvalsDir)
	if err != nil {
		return fmt.Errorf("reading approvals directory: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No approval records found.")
		return nil
	}

	fmt.Println("=== Approval Records ===\n")

	approvalsByType := make(map[string][]ApprovalRecord)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		approvalPath := filepath.Join(approvalsDir, entry.Name())
		data, err := os.ReadFile(approvalPath)
		if err != nil {
			continue
		}

		var approval ApprovalRecord
		if err := yaml.Unmarshal(data, &approval); err != nil {
			continue
		}

		approvalsByType[approval.Type] = append(approvalsByType[approval.Type], approval)
	}

	// Display approvals grouped by type
	for _, approvalType := range []string{"policy", "bundle", "drift", "plan"} {
		approvals := approvalsByType[approvalType]
		if len(approvals) == 0 {
			continue
		}

		fmt.Printf("%s Approvals: %d\n", strings.Title(approvalType), len(approvals))
		for _, approval := range approvals {
			fmt.Printf("  • %s\n", approval.ResourceID)
			fmt.Printf("    Approved by: %s\n", approval.ApprovedBy)
			fmt.Printf("    Approved at: %s\n", approval.ApprovedAt.Format("2006-01-02 15:04:05"))
			if approval.Message != "" {
				fmt.Printf("    Message: %s\n", approval.Message)
			}
			fmt.Println()
		}
	}

	totalApprovals := 0
	for _, approvals := range approvalsByType {
		totalApprovals += len(approvals)
	}
	fmt.Printf("Total approvals: %d\n", totalApprovals)

	return nil
}

func runApprovalsPending(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("approvals.list", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "approvals pending")
		return err
	}

	fmt.Println("=== Pending Approvals ===\n")

	hasPending := false

	// Check for unapproved policy changes
	if hasPolicyChanges, err := checkPolicyChanges(); err == nil && hasPolicyChanges {
		fmt.Println("📋 Policy Changes:")
		fmt.Println("  • Policies have changed since last approval")
		fmt.Println("  • Run 'specular policy diff' to see changes")
		fmt.Println("  • Run 'specular policy approve' to approve")
		fmt.Println()
		hasPending = true
	}

	// Check for unapproved bundles
	if pendingBundles, err := checkPendingBundles(); err == nil && len(pendingBundles) > 0 {
		fmt.Printf("📦 Bundles: %d pending\n", len(pendingBundles))
		for _, bundleID := range pendingBundles {
			fmt.Printf("  • %s\n", bundleID)
		}
		fmt.Println("  Run 'specular approve <bundle-id>' to approve")
		fmt.Println()
		hasPending = true
	}

	// Check for unapproved drift
	if hasDrift, err := checkDrift(); err == nil && hasDrift {
		fmt.Println("🔀 Drift Detected:")
		fmt.Println("  • Drift detected but not approved")
		fmt.Println("  • Run 'specular drift check' to see details")
		fmt.Println("  • Run 'specular drift approve' to approve")
		fmt.Println()
		hasPending = true
	}

	if !hasPending {
		fmt.Println("✅ No pending approvals")
		fmt.Println("\nAll governance items are approved and up to date.")
		return nil
	}

	// Exit code 1 indicates pending approvals (for CI/CD integration)
	os.Exit(1)
	return nil
}

// checkPolicyChanges checks if there are unapproved policy changes
func checkPolicyChanges() (bool, error) {
	policiesPath := filepath.Join(".specular", "policies.yaml")
	if _, err := os.Stat(policiesPath); os.IsNotExist(err) {
		return false, nil
	}

	// Check if there are any policy approval records
	approvalsDir := filepath.Join(".specular", "approvals")
	entries, err := os.ReadDir(approvalsDir)
	if err != nil {
		return false, err
	}

	hasPolicyApproval := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "policy-") {
			hasPolicyApproval = true
			break
		}
	}

	// If no approval exists, policies are pending
	if !hasPolicyApproval {
		return true, nil
	}

	// TODO: Check if policy hash has changed since last approval
	// For now, we'll consider it not pending if any approval exists
	return false, nil
}

// checkPendingBundles checks for bundles that haven't been approved
func checkPendingBundles() ([]string, error) {
	bundlesDir := filepath.Join(".specular", "bundles")
	if _, err := os.Stat(bundlesDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Get all bundle files
	entries, err := os.ReadDir(bundlesDir)
	if err != nil {
		return nil, err
	}

	// Get all approved bundle IDs
	approvalsDir := filepath.Join(".specular", "approvals")
	approvedBundles := make(map[string]bool)

	if approvalEntries, err := os.ReadDir(approvalsDir); err == nil {
		for _, entry := range approvalEntries {
			if !strings.HasPrefix(entry.Name(), "bundle-") {
				continue
			}

			approvalPath := filepath.Join(approvalsDir, entry.Name())
			data, err := os.ReadFile(approvalPath)
			if err != nil {
				continue
			}

			var approval ApprovalRecord
			if err := yaml.Unmarshal(data, &approval); err != nil {
				continue
			}

			approvedBundles[approval.ResourceID] = true
		}
	}

	// Find bundles without approvals
	var pending []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tar") {
			continue
		}

		// Extract bundle ID from filename
		bundleID := strings.TrimSuffix(entry.Name(), ".tar")
		if !approvedBundles[bundleID] {
			pending = append(pending, bundleID)
		}
	}

	return pending, nil
}

// checkDrift checks if there is unapproved drift
func checkDrift() (bool, error) {
	// Check for drift baseline file
	driftPath := filepath.Join(".specular", "drift-baseline.json")
	if _, err := os.Stat(driftPath); os.IsNotExist(err) {
		return false, nil
	}

	// Check if there are any drift approval records
	approvalsDir := filepath.Join(".specular", "approvals")
	entries, err := os.ReadDir(approvalsDir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "drift-") {
			// Has drift approval, no pending drift
			return false, nil
		}
	}

	// Drift baseline exists but no approval - pending
	return true, nil
}

func init() {
	rootCmd.AddCommand(approveCmd)
	rootCmd.AddCommand(approvalsCmd)
	approvalsCmd.AddCommand(approvalsListCmd)
	approvalsCmd.AddCommand(approvalsPendingCmd)

	// Flags for approve command
	approveCmd.Flags().String("message", "", "Approval message or comment")
}
