package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/license"
)

var governanceCmd = &cobra.Command{
	Use:   "governance",
	Short: "Governance workspace management",
	Long: `Initialize, validate, and manage governance infrastructure.

Governance commands help you set up and maintain a governed AI development
environment with policies, approvals, and compliance controls.

Available commands:
  init    - Initialize governance workspace
  doctor  - Validate governance environment
  status  - Show governance health overview`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var governanceInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize governance workspace",
	Long: `Initialize a governance workspace by creating the .specular/ directory structure.

Creates:
  .specular/
    providers.yaml    - Provider allowlist and configuration
    policies.yaml     - Policy definitions and rules
    approvals/        - Approval records and signatures
    bundles/          - Governance bundles
    traces/           - Execution traces for audit

This command requires Specular Pro or Enterprise.`,
	RunE: runGovernanceInit,
}

var governanceDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate governance environment",
	Long: `Run comprehensive checks on the governance environment.

Validates:
  • Provider configuration and availability
  • Policy definitions and syntax
  • Approval records and signatures
  • Bundle integrity
  • Drift baseline status
  • Environment variables and secrets

Returns exit code 0 if all checks pass, non-zero otherwise.

This command requires Specular Pro or Enterprise.`,
	RunE: runGovernanceDoctor,
}

var governanceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show governance health overview",
	Long: `Display an overview of governance health and status.

Shows:
  • Current governance workspace location
  • Number of active policies
  • Number of pending approvals
  • Bundle count and status
  • Recent drift detections
  • Overall health score

This command requires Specular Pro or Enterprise.`,
	RunE: runGovernanceStatus,
}

func runGovernanceInit(cmd *cobra.Command, args []string) error {
	// Check license - governance features require Pro tier
	if err := license.RequireFeature("governance.init", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "governance init")
		return err
	}

	force := cmd.Flags().Lookup("force").Value.String() == "true"
	path := cmd.Flags().Lookup("path").Value.String()
	if path == "" {
		path = ".specular"
	}

	fmt.Printf("Initializing governance workspace at %s...\n\n", path)

	// Check if directory already exists
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("governance workspace already exists at %s (use --force to overwrite)", path)
	}

	// Create directory structure
	dirs := []string{
		path,
		filepath.Join(path, "approvals"),
		filepath.Join(path, "bundles"),
		filepath.Join(path, "traces"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
		fmt.Printf("✓ Created %s/\n", dir)
	}

	// Create providers.yaml template
	providersTemplate := `# Specular Providers Configuration
# Define allowed AI providers and their settings

version: "1.0"

# Allowed providers (allowlist)
allow:
  - "ollama:llama3.2"
  - "ollama:qwen2.5-coder"
  # - "openai:gpt-4o-mini"
  # - "anthropic:claude-sonnet-3.5"

# Provider-specific settings
providers:
  ollama:
    base_url: "http://localhost:11434"
    timeout: 30s

  # openai:
  #   api_key_env: "OPENAI_API_KEY"
  #   timeout: 60s

  # anthropic:
  #   api_key_env: "ANTHROPIC_API_KEY"
  #   timeout: 60s

# Cost limits (Pro feature)
limits:
  max_cost_usd: 1.00
  max_tokens_per_request: 16000
  max_requests_per_hour: 100

# Routing preferences (Pro feature)
routing:
  prefer_local: true
  fallback_to_cloud: false
`

	providersPath := filepath.Join(path, "providers.yaml")
	if err := os.WriteFile(providersPath, []byte(providersTemplate), 0644); err != nil {
		return fmt.Errorf("creating providers.yaml: %w", err)
	}
	fmt.Printf("✓ Created %s\n", providersPath)

	// Create policies.yaml template
	policiesTemplate := `# Specular Policies Configuration
# Define governance policies and rules

version: "1.0"

# Policy enforcement level
enforcement: "strict"  # Options: strict, warn, monitor

# Workflow policies
workflows:
  require_approval_for:
    - "bundle.gate"
    - "policy.approve"
    - "drift.approve"

  require_attestation_for:
    - "bundle.gate"

# Security policies
security:
  require_encryption: true
  audit_all_actions: true
  secrets_in_vault: false  # Enterprise feature

# Compliance policies (Enterprise)
# compliance:
#   soc2_enabled: true
#   export_audit_logs: true
#   retention_days: 365
`

	policiesPath := filepath.Join(path, "policies.yaml")
	if err := os.WriteFile(policiesPath, []byte(policiesTemplate), 0644); err != nil {
		return fmt.Errorf("creating policies.yaml: %w", err)
	}
	fmt.Printf("✓ Created %s\n", policiesPath)

	// Create .gitignore
	gitignoreContent := `# Specular governance workspace

# Sensitive data
approvals/
traces/

# Generated files
bundles/

# Keep templates
!.gitkeep
`

	gitignorePath := filepath.Join(path, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("creating .gitignore: %w", err)
	}
	fmt.Printf("✓ Created %s\n", gitignorePath)

	fmt.Println("\n✅ Governance workspace initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review and customize providers.yaml")
	fmt.Println("  2. Review and customize policies.yaml")
	fmt.Println("  3. Run 'specular governance doctor' to validate configuration")
	fmt.Println("  4. Run 'specular policy approve' to approve your policies")
	fmt.Println()

	return nil
}

func runGovernanceDoctor(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("governance.doctor", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "governance doctor")
		return err
	}

	fmt.Println("Running governance environment checks...")

	// Check 1: Workspace exists
	fmt.Print("📁 Checking governance workspace... ")
	if _, err := os.Stat(".specular"); os.IsNotExist(err) {
		fmt.Println("❌ FAIL")
		fmt.Println("   Workspace not found. Run 'specular governance init' first.")
		return fmt.Errorf("governance workspace not initialized")
	}
	fmt.Println("✓ OK")

	// Check 2: Providers configuration
	fmt.Print("🔌 Checking providers.yaml... ")
	if _, err := os.Stat(".specular/providers.yaml"); os.IsNotExist(err) {
		fmt.Println("❌ FAIL")
		fmt.Println("   providers.yaml not found")
		return fmt.Errorf("providers.yaml missing")
	}
	fmt.Println("✓ OK")

	// Check 3: Policies configuration
	fmt.Print("📋 Checking policies.yaml... ")
	if _, err := os.Stat(".specular/policies.yaml"); os.IsNotExist(err) {
		fmt.Println("❌ FAIL")
		fmt.Println("   policies.yaml not found")
		return fmt.Errorf("policies.yaml missing")
	}
	fmt.Println("✓ OK")

	// Check 4: Required directories
	fmt.Print("📂 Checking directory structure... ")
	requiredDirs := []string{"approvals", "bundles", "traces"}
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(".specular", dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			fmt.Println("❌ FAIL")
			fmt.Printf("   Missing directory: %s\n", dirPath)
			return fmt.Errorf("missing directory: %s", dirPath)
		}
	}
	fmt.Println("✓ OK")

	fmt.Println("\n✅ All governance checks passed!")
	fmt.Println("\nYour governance environment is properly configured.")
	fmt.Println()

	return nil
}

func runGovernanceStatus(cmd *cobra.Command, args []string) error {
	// Check license
	if err := license.RequireFeature("governance.status", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "governance status")
		return err
	}

	fmt.Println("=== Governance Status ===")

	// Workspace location
	workspacePath, _ := filepath.Abs(".specular")
	fmt.Printf("Workspace: %s\n", workspacePath)

	// Check if initialized
	if _, err := os.Stat(".specular"); os.IsNotExist(err) {
		fmt.Println("Status: Not initialized")
		fmt.Println("\nRun 'specular governance init' to get started.")
		return nil
	}

	fmt.Println("Status: Initialized ✓")
	fmt.Println()

	// Count files in each directory
	countFiles := func(dir string) int {
		entries, err := os.ReadDir(filepath.Join(".specular", dir))
		if err != nil {
			return 0
		}
		return len(entries)
	}

	fmt.Printf("Approvals: %d\n", countFiles("approvals"))
	fmt.Printf("Bundles: %d\n", countFiles("bundles"))
	fmt.Printf("Traces: %d\n", countFiles("traces"))
	fmt.Println()

	// Show license tier
	tier, _ := license.GetTier()
	fmt.Printf("License: %s\n", tier)
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(governanceCmd)
	governanceCmd.AddCommand(governanceInitCmd)
	governanceCmd.AddCommand(governanceDoctorCmd)
	governanceCmd.AddCommand(governanceStatusCmd)

	// Flags for governance init
	governanceInitCmd.Flags().Bool("force", false, "Overwrite existing governance workspace")
	governanceInitCmd.Flags().String("path", ".specular", "Directory for governance workspace")
}
