package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/policy"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage governance policies",
	Long: `Create and apply governance policies for team collaboration and enterprise security.

Policies define constraints for execution, testing, security scanning, and AI model routing.

Subcommands:
  new     Create a new policy file with defaults
  apply   Apply a policy to a target (project, workspace, organization)

Examples:
  specular policy new --output .specular/policy.yaml
  specular policy apply --file .specular/policy.yaml --target project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// policyNewCmd creates a new policy file
var policyNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new policy file with defaults",
	Long: `Create a new governance policy file with sensible defaults.

The policy includes:
- Execution constraints (Docker, local execution)
- Linter and formatter configurations
- Test requirements (coverage, pass requirements)
- Security scanning (secrets, dependencies)
- AI model routing constraints

Examples:
  specular policy new                          # Creates policy.yaml in current directory
  specular policy new --output custom.yaml     # Creates custom.yaml
  specular policy new --strict                 # Creates strict policy with enhanced security`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		outputPath, _ := cmd.Flags().GetString("output")
		strict, _ := cmd.Flags().GetBool("strict")

		// Default output path
		if outputPath == "" {
			outputPath = "policy.yaml"
		}

		// Check if file already exists
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("policy file already exists: %s (use --force to overwrite)", outputPath)
		}

		// Create policy with defaults
		pol := policy.DefaultPolicy()

		// Apply strict mode if requested
		if strict {
			pol.Execution.AllowLocal = false
			pol.Execution.Docker.Required = true
			pol.Tests.RequirePass = true
			pol.Tests.MinCoverage = 0.80
			pol.Security.SecretsScan = true
			pol.Security.DepScan = true
		}

		// Save policy
		if err := policy.SavePolicy(pol, outputPath); err != nil {
			return fmt.Errorf("failed to save policy: %w", err)
		}

		fmt.Printf("✅ Created policy file: %s\n", outputPath)
		fmt.Println()
		fmt.Println("Policy includes:")
		fmt.Printf("  Execution: Docker required = %v, Allow local = %v\n", pol.Execution.Docker.Required, pol.Execution.AllowLocal)
		fmt.Printf("  Tests: Require pass = %v, Min coverage = %.0f%%\n", pol.Tests.RequirePass, pol.Tests.MinCoverage*100)
		fmt.Printf("  Security: Secrets scan = %v, Dependency scan = %v\n", pol.Security.SecretsScan, pol.Security.DepScan)
		fmt.Println()
		fmt.Println("Edit the policy file to customize settings, then use 'specular policy apply' to apply it.")

		return nil
	},
}

// policyApplyCmd applies a policy to a target
var policyApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a policy to a target",
	Long: `Apply a governance policy to a target scope (project, workspace, or organization).

The policy will be validated and copied to the appropriate location based on the target.

Targets:
  project       Apply to current project (.specular/policy.yaml)
  workspace     Apply to workspace (Future - requires workspace support)
  organization  Apply to organization (Future - requires org support)

Examples:
  specular policy apply --file policy.yaml --target project
  specular policy apply --file custom.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		policyFile, _ := cmd.Flags().GetString("file")
		target, _ := cmd.Flags().GetString("target")

		// Validate flags
		if policyFile == "" {
			return fmt.Errorf("--file is required")
		}

		// Load and validate policy
		pol, err := policy.LoadPolicy(policyFile)
		if err != nil {
			return fmt.Errorf("failed to load policy: %w", err)
		}

		// Determine target path based on target type
		var targetPath string
		switch target {
		case "project":
			// Create .specular directory if it doesn't exist
			if err := os.MkdirAll(".specular", 0755); err != nil {
				return fmt.Errorf("failed to create .specular directory: %w", err)
			}
			targetPath = ".specular/policy.yaml"
		case "workspace":
			return fmt.Errorf("workspace target not yet supported (planned for future release)")
		case "organization":
			return fmt.Errorf("organization target not yet supported (planned for future release)")
		default:
			return fmt.Errorf("invalid target '%s'. Valid targets: project, workspace, organization", target)
		}

		// Save policy to target
		if err := policy.SavePolicy(pol, targetPath); err != nil {
			return fmt.Errorf("failed to apply policy: %w", err)
		}

		fmt.Printf("✅ Applied policy to %s: %s\n", target, targetPath)
		fmt.Println()
		fmt.Println("Policy settings:")
		fmt.Printf("  Execution: Docker required = %v\n", pol.Execution.Docker.Required)
		fmt.Printf("  Tests: Min coverage = %.0f%%\n", pol.Tests.MinCoverage*100)
		fmt.Printf("  Security: Secrets scan = %v, Dep scan = %v\n", pol.Security.SecretsScan, pol.Security.DepScan)

		return nil
	},
}

func init() {
	// Add subcommands
	policyCmd.AddCommand(policyNewCmd)
	policyCmd.AddCommand(policyApplyCmd)

	// Flags for policy new
	policyNewCmd.Flags().String("output", "", "Output path for policy file (default: policy.yaml)")
	policyNewCmd.Flags().Bool("strict", false, "Create strict policy with enhanced security")
	policyNewCmd.Flags().Bool("force", false, "Overwrite existing policy file")

	// Flags for policy apply
	policyApplyCmd.Flags().String("file", "", "Policy file to apply (required)")
	policyApplyCmd.Flags().String("target", "project", "Target scope (project, workspace, organization)")

	rootCmd.AddCommand(policyCmd)
}
