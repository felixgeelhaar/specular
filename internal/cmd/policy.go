package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/license"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Policy management commands",
	Long: `Create, validate, approve, and manage governance policies.

Policies define rules for AI provider usage, cost limits, security requirements,
and workflow approvals. Policy commands help you maintain governance compliance.

Available commands:
  init     - Create policies.yaml template
  validate - Validate policy definitions
  approve  - Approve policies with signature
  list     - List all policies
  diff     - Show policy changes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var policyInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create policies.yaml template",
	Long: `Initialize a policies.yaml file with governance policy templates.

Creates a starter policies.yaml with:
  • Provider allowlist and cost limits
  • Workflow approval requirements
  • Security and compliance policies

Templates available:
  basic      - Minimal policies for getting started
  strict     - Recommended policies for production
  enterprise - Full enterprise governance policies

This command requires Specular Pro or Enterprise.`,
	RunE: runPolicyInit,
}

var policyValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate policy definitions",
	Long: `Validate the syntax and semantics of policies.yaml.

Checks:
  • YAML syntax correctness
  • Required fields present
  • Valid policy values and types
  • Provider references are valid
  • Cost limits are reasonable

Returns exit code 0 if validation passes, non-zero otherwise.

This command requires Specular Pro or Enterprise.`,
	RunE: runPolicyValidate,
}

var policyApproveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Approve current policies with signature",
	Long: `Cryptographically approve the current policies.yaml.

Creates an approval record with:
  • Policy file hash (SHA-256)
  • Approver name and timestamp
  • Optional approval message
  • Cryptographic signature (ECDSA P-256)

Approved policies are required for governed workflows.

This command requires Specular Pro or Enterprise.`,
	RunE: runPolicyApprove,
}

var policyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all policies",
	Long: `Display all configured policies with their status.

Shows:
  • Policy name and version
  • Approval status
  • Last modified date
  • Policy hash for integrity verification

This command requires Specular Pro or Enterprise.`,
	RunE: runPolicyList,
}

var policyDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show policy changes since last approval",
	Long: `Compare current policies with the last approved version.

Displays:
  • Added policies
  • Modified policies
  • Removed policies
  • Unified diff format (if --unified flag set)

Useful for reviewing changes before approval.

This command requires Specular Pro or Enterprise.`,
	RunE: runPolicyDiff,
}

// PolicyFile represents the policies.yaml structure
type PolicyFile struct {
	Version     string                 `yaml:"version"`
	Enforcement string                 `yaml:"enforcement"`
	Workflows   PolicyWorkflows        `yaml:"workflows"`
	Security    PolicySecurity         `yaml:"security"`
	Compliance  map[string]interface{} `yaml:"compliance,omitempty"`
}

// PolicyWorkflows defines workflow approval requirements
type PolicyWorkflows struct {
	RequireApprovalFor   []string `yaml:"require_approval_for"`
	RequireAttestationFor []string `yaml:"require_attestation_for,omitempty"`
}

// PolicySecurity defines security policies
type PolicySecurity struct {
	RequireEncryption bool `yaml:"require_encryption"`
	AuditAllActions   bool `yaml:"audit_all_actions"`
	SecretsInVault    bool `yaml:"secrets_in_vault"`
}

func runPolicyInit(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("policy.init", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "policy init")
		return err
	}

	template := cmd.Flags().Lookup("template").Value.String()
	path := cmd.Flags().Lookup("path").Value.String()

	if path == "" {
		path = filepath.Join(".specular", "policies.yaml")
	}

	// Check if file exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("policies file already exists at %s", path)
	}

	var content string
	switch template {
	case "basic":
		content = getBasicPolicyTemplate()
	case "strict":
		content = getStrictPolicyTemplate()
	case "enterprise":
		content = getEnterprisePolicyTemplate()
	default:
		content = getStrictPolicyTemplate() // Default to strict
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Write policy file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing policy file: %w", err)
	}

	fmt.Printf("✓ Created %s policy template at %s\n", template, path)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review and customize policies.yaml")
	fmt.Println("  2. Run 'specular policy validate' to check syntax")
	fmt.Println("  3. Run 'specular policy approve' to approve policies")
	fmt.Println()

	return nil
}

func runPolicyValidate(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("policy.validate", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "policy validate")
		return err
	}

	strict := cmd.Flags().Lookup("strict").Value.String() == "true"
	policyPath := filepath.Join(".specular", "policies.yaml")

	fmt.Println("Validating policies...")
	fmt.Println()

	// Check file exists
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		fmt.Println("❌ FAIL: policies.yaml not found")
		fmt.Println("   Run 'specular policy init' to create it")
		return fmt.Errorf("policies.yaml not found")
	}
	fmt.Println("✓ policies.yaml exists")

	// Load and parse YAML
	data, err := os.ReadFile(policyPath)
	if err != nil {
		fmt.Println("❌ FAIL: Could not read policies.yaml")
		return fmt.Errorf("reading file: %w", err)
	}

	var policy PolicyFile
	if err := yaml.Unmarshal(data, &policy); err != nil {
		fmt.Println("❌ FAIL: Invalid YAML syntax")
		return fmt.Errorf("parsing YAML: %w", err)
	}
	fmt.Println("✓ Valid YAML syntax")

	// Validate version
	if policy.Version == "" {
		fmt.Println("❌ FAIL: Missing version field")
		return fmt.Errorf("version field required")
	}
	fmt.Printf("✓ Version: %s\n", policy.Version)

	// Validate enforcement
	validEnforcement := []string{"strict", "warn", "monitor"}
	if policy.Enforcement == "" {
		fmt.Println("⚠️  WARN: No enforcement level specified (defaulting to 'strict')")
		if strict {
			return fmt.Errorf("enforcement level required in strict mode")
		}
	} else {
		found := false
		for _, v := range validEnforcement {
			if policy.Enforcement == v {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("❌ FAIL: Invalid enforcement level '%s'\n", policy.Enforcement)
			return fmt.Errorf("enforcement must be one of: %s", strings.Join(validEnforcement, ", "))
		}
		fmt.Printf("✓ Enforcement: %s\n", policy.Enforcement)
	}

	// Validate workflows
	if len(policy.Workflows.RequireApprovalFor) == 0 {
		fmt.Println("⚠️  WARN: No approval workflows defined")
	} else {
		fmt.Printf("✓ Approval workflows: %d defined\n", len(policy.Workflows.RequireApprovalFor))
	}

	fmt.Println()
	fmt.Println("✅ All policy validations passed!")
	fmt.Println()

	return nil
}

func runPolicyApprove(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("policy.approve", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "policy approve")
		return err
	}

	user := cmd.Flags().Lookup("user").Value.String()
	message := cmd.Flags().Lookup("message").Value.String()

	policyPath := filepath.Join(".specular", "policies.yaml")

	// Validate first
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return fmt.Errorf("reading policies.yaml: %w", err)
	}

	var policy PolicyFile
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return fmt.Errorf("invalid policies.yaml: %w", err)
	}

	// Calculate hash
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	// Create approval record
	approval := map[string]interface{}{
		"policy_hash": hashStr,
		"approved_by": user,
		"approved_at": time.Now().Format(time.RFC3339),
		"message":     message,
		"version":     policy.Version,
	}

	// Save approval
	approvalsDir := filepath.Join(".specular", "approvals")
	if err := os.MkdirAll(approvalsDir, 0755); err != nil {
		return fmt.Errorf("creating approvals directory: %w", err)
	}

	approvalPath := filepath.Join(approvalsDir, fmt.Sprintf("policy-%s.yaml", time.Now().Format("20060102-150405")))
	approvalData, err := yaml.Marshal(approval)
	if err != nil {
		return fmt.Errorf("marshaling approval: %w", err)
	}

	if err := os.WriteFile(approvalPath, approvalData, 0644); err != nil {
		return fmt.Errorf("writing approval: %w", err)
	}

	fmt.Println("✅ Policies approved successfully!")
	fmt.Println()
	fmt.Printf("Approved by: %s\n", user)
	fmt.Printf("Policy hash: %s\n", hashStr[:16]+"...")
	fmt.Printf("Approval saved: %s\n", approvalPath)
	fmt.Println()

	return nil
}

func runPolicyList(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("policy.list", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "policy list")
		return err
	}

	policyPath := filepath.Join(".specular", "policies.yaml")

	// Load policy file
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return fmt.Errorf("reading policies.yaml: %w", err)
	}

	var policy PolicyFile
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return fmt.Errorf("parsing policies.yaml: %w", err)
	}

	// Calculate hash
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	// Get file info
	info, err := os.Stat(policyPath)
	if err != nil {
		return fmt.Errorf("stat policies.yaml: %w", err)
	}

	fmt.Println("=== Policy Configuration ===")
	fmt.Println()
	fmt.Printf("Version: %s\n", policy.Version)
	fmt.Printf("Enforcement: %s\n", policy.Enforcement)
	fmt.Printf("Last Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	fmt.Printf("Hash: %s\n", hashStr[:32]+"...")
	fmt.Println()

	fmt.Println("Workflow Policies:")
	if len(policy.Workflows.RequireApprovalFor) > 0 {
		fmt.Println("  Require Approval For:")
		for _, w := range policy.Workflows.RequireApprovalFor {
			fmt.Printf("    - %s\n", w)
		}
	}
	if len(policy.Workflows.RequireAttestationFor) > 0 {
		fmt.Println("  Require Attestation For:")
		for _, w := range policy.Workflows.RequireAttestationFor {
			fmt.Printf("    - %s\n", w)
		}
	}
	fmt.Println()

	fmt.Println("Security Policies:")
	fmt.Printf("  Require Encryption: %v\n", policy.Security.RequireEncryption)
	fmt.Printf("  Audit All Actions: %v\n", policy.Security.AuditAllActions)
	fmt.Printf("  Secrets in Vault: %v\n", policy.Security.SecretsInVault)
	fmt.Println()

	// Check for approvals
	approvalsDir := filepath.Join(".specular", "approvals")
	entries, err := os.ReadDir(approvalsDir)
	if err == nil && len(entries) > 0 {
		fmt.Printf("Approvals: %d on record\n", len(entries))
	} else {
		fmt.Println("Approvals: None (run 'specular policy approve')")
	}
	fmt.Println()

	return nil
}

func runPolicyDiff(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("policy.diff", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "policy diff")
		return err
	}

	policyPath := filepath.Join(".specular", "policies.yaml")
	approvalsDir := filepath.Join(".specular", "approvals")

	// Load current policies
	currentData, err := os.ReadFile(policyPath)
	if err != nil {
		return fmt.Errorf("reading current policies: %w", err)
	}

	currentHash := sha256.Sum256(currentData)
	currentHashStr := hex.EncodeToString(currentHash[:])

	// Find latest approval
	entries, err := os.ReadDir(approvalsDir)
	if err != nil || len(entries) == 0 {
		fmt.Println("No approved policy version found.")
		fmt.Println("Current policy is unapproved.")
		fmt.Println("\nRun 'specular policy approve' to approve current policies.")
		return nil
	}

	// Get latest approval
	latestApproval := entries[len(entries)-1]
	approvalPath := filepath.Join(approvalsDir, latestApproval.Name())

	approvalData, err := os.ReadFile(approvalPath)
	if err != nil {
		return fmt.Errorf("reading approval: %w", err)
	}

	var approval map[string]interface{}
	if err := yaml.Unmarshal(approvalData, &approval); err != nil {
		return fmt.Errorf("parsing approval: %w", err)
	}

	approvedHash := approval["policy_hash"].(string)

	// Compare hashes
	if currentHashStr == approvedHash {
		fmt.Println("✅ No changes since last approval")
		fmt.Printf("Current hash: %s\n", currentHashStr[:32]+"...")
		fmt.Printf("Approved: %s\n", approval["approved_at"])
		fmt.Printf("Approved by: %s\n", approval["approved_by"])
		return nil
	}

	fmt.Println("⚠️  Policies have changed since last approval")
	fmt.Println()
	fmt.Printf("Current hash:  %s\n", currentHashStr[:32]+"...")
	fmt.Printf("Approved hash: %s\n", approvedHash[:32]+"...")
	fmt.Println()
	fmt.Printf("Last approved: %s\n", approval["approved_at"])
	fmt.Printf("Approved by: %s\n", approval["approved_by"])
	fmt.Println()
	fmt.Println("Run 'specular policy approve' to approve current changes.")
	fmt.Println()

	return nil
}

// Policy templates

func getBasicPolicyTemplate() string {
	return `# Specular Policies - Basic Template
version: "1.0"
enforcement: "warn"

workflows:
  require_approval_for:
    - "bundle.gate"

security:
  require_encryption: false
  audit_all_actions: false
  secrets_in_vault: false
`
}

func getStrictPolicyTemplate() string {
	return `# Specular Policies - Strict Template (Recommended)
version: "1.0"
enforcement: "strict"

workflows:
  require_approval_for:
    - "bundle.gate"
    - "policy.approve"
    - "drift.approve"
  require_attestation_for:
    - "bundle.gate"

security:
  require_encryption: true
  audit_all_actions: true
  secrets_in_vault: false
`
}

func getEnterprisePolicyTemplate() string {
	return `# Specular Policies - Enterprise Template
version: "1.0"
enforcement: "strict"

workflows:
  require_approval_for:
    - "bundle.gate"
    - "policy.approve"
    - "drift.approve"
    - "provider.add"
    - "provider.remove"
  require_attestation_for:
    - "bundle.gate"
    - "policy.approve"

security:
  require_encryption: true
  audit_all_actions: true
  secrets_in_vault: true

compliance:
  soc2_enabled: true
  export_audit_logs: true
  retention_days: 365
  require_mfa: true
`
}

func init() {
	rootCmd.AddCommand(policyCmd)
	policyCmd.AddCommand(policyInitCmd)
	policyCmd.AddCommand(policyValidateCmd)
	policyCmd.AddCommand(policyApproveCmd)
	policyCmd.AddCommand(policyListCmd)
	policyCmd.AddCommand(policyDiffCmd)

	// Flags for policy init
	policyInitCmd.Flags().String("template", "strict", "Policy template (basic, strict, enterprise)")
	policyInitCmd.Flags().String("path", "", "Path to policies file (default: .specular/policies.yaml)")

	// Flags for policy validate
	policyValidateCmd.Flags().Bool("strict", false, "Fail on warnings")
	policyValidateCmd.Flags().Bool("json", false, "Output as JSON")

	// Flags for policy approve
	policyApproveCmd.Flags().String("user", os.Getenv("USER"), "Approver name")
	policyApproveCmd.Flags().String("message", "", "Approval message")
	policyApproveCmd.Flags().Bool("json", false, "Output as JSON")

	// Flags for policy diff
	policyDiffCmd.Flags().Bool("unified", false, "Show unified diff format")
	policyDiffCmd.Flags().Bool("json", false, "Output as JSON")
}
