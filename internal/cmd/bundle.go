package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/bundle"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Governance bundle management",
	Long: `Create, verify, and apply governance bundles.

Governance bundles (.sbundle.tgz) package specifications, policies,
routing configuration, and approvals into a portable, verifiable archive.`,
}

// Bundle build command flags
var (
	buildOutput      string
	buildSpec        string
	buildLock        string
	buildRouting     string
	buildPolicies    []string
	buildInclude     []string
	buildApprovals   []string
	buildAttest      bool
	buildAttestFmt   string
	buildMetadata    []string
	buildGovLevel    string
)

var bundleBuildCmd = &cobra.Command{
	Use:   "build [output]",
	Short: "Build a governance bundle",
	Long: `Create a governance bundle from project files.

Bundles package:
- Product specification (spec.yaml)
- Locked dependencies (spec.lock.json)
- AI provider routing (routing.yaml)
- Governance policies (policies/*.yaml)
- Additional files (--include)

The bundle includes:
- SHA-256 checksums for all files
- Cryptographic integrity digest
- Optional approval signatures
- Optional Sigstore attestation

Examples:
  # Build bundle from current directory
  specular bundle build my-app-v1.0.0.sbundle.tgz

  # Build with specific files
  specular bundle build --spec spec.yaml --lock spec.lock.json bundle.sbundle.tgz

  # Build with policies
  specular bundle build --policy policies/security.yaml --policy policies/compliance.yaml bundle.sbundle.tgz

  # Build with governance level
  specular bundle build --governance-level L3 bundle.sbundle.tgz`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBundleBuild,
}

// Bundle verify command flags
var (
	verifyStrict       bool
	verifyApprovals    bool
	verifyAttestation  bool
	verifyPolicy       string
	verifyTrustedKeys  []string
	verifyOffline      bool
)

var bundleVerifyCmd = &cobra.Command{
	Use:   "verify <bundle>",
	Short: "Verify a governance bundle",
	Long: `Verify the integrity and signatures of a governance bundle.

Verification checks:
- Manifest structure and completeness
- File checksums (SHA-256)
- Required approvals (if --require-approvals)
- Cryptographic attestation (if --verify-attestation)
- Policy compliance (if --policy specified)

Examples:
  # Basic verification
  specular bundle verify my-app-v1.0.0.sbundle.tgz

  # Strict mode (fail on any error)
  specular bundle verify --strict bundle.sbundle.tgz

  # Require approvals
  specular bundle verify --require-approvals bundle.sbundle.tgz

  # Verify attestation
  specular bundle verify --verify-attestation bundle.sbundle.tgz`,
	Args: cobra.ExactArgs(1),
	RunE: runBundleVerify,
}

// Bundle apply command flags
var (
	applyTargetDir string
	applyDryRun    bool
	applyForce     bool
	applyYes       bool
	applyExclude   []string
)

var bundleApplyCmd = &cobra.Command{
	Use:   "apply <bundle>",
	Short: "Apply a governance bundle to a project",
	Long: `Extract and apply a governance bundle to the current project.

This command:
1. Validates the bundle
2. Extracts files to target directory
3. Applies spec, lock, routing, and policies
4. Prompts for confirmation on file overwrites (unless --force or --yes)

Examples:
  # Dry-run to preview changes
  specular bundle apply --dry-run bundle.sbundle.tgz

  # Apply to current directory
  specular bundle apply bundle.sbundle.tgz

  # Apply to specific directory
  specular bundle apply --target-dir /path/to/project bundle.sbundle.tgz

  # Force overwrite all files
  specular bundle apply --force bundle.sbundle.tgz

  # Auto-confirm all prompts
  specular bundle apply --yes bundle.sbundle.tgz`,
	Args: cobra.ExactArgs(1),
	RunE: runBundleApply,
}

func runBundleBuild(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()

	// Determine output path
	output := buildOutput
	if len(args) > 0 {
		output = args[0]
	}
	if output == "" {
		// Generate default output name
		output = "bundle.sbundle.tgz"
	}

	// Use defaults if not specified
	if !cmd.Flags().Changed("spec") {
		buildSpec = defaults.SpecFile()
	}
	if !cmd.Flags().Changed("lock") {
		buildLock = defaults.SpecLockFile()
	}
	if !cmd.Flags().Changed("routing") {
		buildRouting = filepath.Join(defaults.SpecularDir, "routing.yaml")
	}

	// Parse metadata
	metadata := make(map[string]string)
	for _, m := range buildMetadata {
		parts := strings.SplitN(m, "=", 2)
		if len(parts) == 2 {
			metadata[parts[0]] = parts[1]
		}
	}

	// Parse approvals
	var approvals []string
	if len(buildApprovals) > 0 {
		approvals = buildApprovals
	}

	// Build options
	opts := bundle.BundleOptions{
		SpecPath:          buildSpec,
		LockPath:          buildLock,
		RoutingPath:       buildRouting,
		PolicyPaths:       buildPolicies,
		IncludePaths:      buildInclude,
		RequireApprovals:  approvals,
		AttestationFormat: buildAttestFmt,
		Metadata:          metadata,
		GovernanceLevel:   buildGovLevel,
	}

	// Create builder
	fmt.Println("Creating governance bundle...")
	builder, err := bundle.NewBuilder(opts)
	if err != nil {
		return ux.FormatError(err, "creating bundle builder")
	}

	// Build bundle
	if err := builder.Build(output); err != nil {
		return ux.FormatError(err, "building bundle")
	}

	// Get file info
	info, err := os.Stat(output)
	if err != nil {
		return ux.FormatError(err, "reading bundle info")
	}

	fmt.Printf("\n✓ Bundle created successfully: %s (%.2f MB)\n", output, float64(info.Size())/(1024*1024))

	// Get bundle info
	bundleInfo, err := bundle.GetBundleInfo(output)
	if err == nil {
		fmt.Printf("\nBundle Details:\n")
		fmt.Printf("  ID:      %s\n", bundleInfo.ID)
		fmt.Printf("  Version: %s\n", bundleInfo.Version)
		fmt.Printf("  Schema:  %s\n", bundleInfo.Schema)
		fmt.Printf("  Created: %s\n", bundleInfo.Created.Format("2006-01-02 15:04:05"))
		if bundleInfo.GovernanceLevel != "" {
			fmt.Printf("  Governance Level: %s\n", bundleInfo.GovernanceLevel)
		}
		if bundleInfo.IntegrityDigest != "" {
			fmt.Printf("  Digest:  %s\n", bundleInfo.IntegrityDigest)
		}
	}

	return nil
}

func runBundleVerify(cmd *cobra.Command, args []string) error {
	bundlePath := args[0]

	// Check bundle exists
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return ux.FormatError(err, "bundle not found")
	}

	fmt.Printf("Verifying bundle: %s\n\n", bundlePath)

	// Create validator
	opts := bundle.VerifyOptions{
		Strict:             verifyStrict,
		RequireApprovals:   verifyApprovals,
		RequireAttestation: verifyAttestation,
		PolicyPath:         verifyPolicy,
		TrustPublicKeys:    verifyTrustedKeys,
		AllowOffline:       verifyOffline,
	}

	validator := bundle.NewValidator(opts)

	// Verify bundle
	result, err := validator.Verify(bundlePath)
	if err != nil {
		return ux.FormatError(err, "verification failed")
	}

	// Display results
	if result.Valid {
		fmt.Println("✓ Bundle verification PASSED")
	} else {
		fmt.Println("✗ Bundle verification FAILED")
	}

	fmt.Println()

	// Show validation details
	fmt.Printf("Checksum Validation:    %s\n", formatValidationStatus(result.ChecksumValid))
	fmt.Printf("Approval Validation:    %s\n", formatValidationStatus(result.ApprovalsValid))
	fmt.Printf("Attestation Validation: %s\n", formatValidationStatus(result.AttestationValid))
	if result.PolicyCompliant {
		fmt.Printf("Policy Compliance:      %s\n", formatValidationStatus(result.PolicyCompliant))
	}

	// Show errors
	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(result.Errors))
		for i, err := range result.Errors {
			fmt.Printf("  %d. [%s] %s\n", i+1, err.Code, err.Message)
			if err.Field != "" {
				fmt.Printf("     Field: %s\n", err.Field)
			}
		}
	}

	// Show warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(result.Warnings))
		for i, warn := range result.Warnings {
			fmt.Printf("  %d. [%s] %s\n", i+1, warn.Code, warn.Message)
			if warn.Field != "" {
				fmt.Printf("     Field: %s\n", warn.Field)
			}
		}
	}

	if !result.Valid {
		fmt.Println()
		return fmt.Errorf("bundle validation failed with %d errors", len(result.Errors))
	}

	return nil
}

func runBundleApply(cmd *cobra.Command, args []string) error {
	bundlePath := args[0]

	// Check bundle exists
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return ux.FormatError(err, "bundle not found")
	}

	// Determine target directory
	targetDir := applyTargetDir
	if targetDir == "" {
		targetDir = "."
	}

	// Resolve to absolute path
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return ux.FormatError(err, "resolving target directory")
	}

	fmt.Printf("Applying bundle to: %s\n", absTargetDir)
	fmt.Println()

	// Create extractor
	opts := bundle.ApplyOptions{
		TargetDir: absTargetDir,
		DryRun:    applyDryRun,
		Force:     applyForce,
		Yes:       applyYes,
		Exclude:   applyExclude,
	}

	extractor := bundle.NewExtractor(opts)

	// Apply bundle
	if err := extractor.Apply(bundlePath); err != nil {
		return ux.FormatError(err, "applying bundle")
	}

	return nil
}

func formatValidationStatus(valid bool) string {
	if valid {
		return "✓ PASS"
	}
	return "✗ FAIL"
}

func init() {
	// Bundle build flags
	bundleBuildCmd.Flags().StringVarP(&buildOutput, "output", "o", "", "Output bundle path (default: bundle.sbundle.tgz)")
	bundleBuildCmd.Flags().StringVar(&buildSpec, "spec", "", "Path to spec.yaml (default: .specular/spec.yaml)")
	bundleBuildCmd.Flags().StringVar(&buildLock, "lock", "", "Path to spec.lock.json (default: .specular/spec.lock.json)")
	bundleBuildCmd.Flags().StringVar(&buildRouting, "routing", "", "Path to routing.yaml (default: .specular/routing.yaml)")
	bundleBuildCmd.Flags().StringSliceVarP(&buildPolicies, "policy", "p", nil, "Policy files to include (can be specified multiple times)")
	bundleBuildCmd.Flags().StringSliceVarP(&buildInclude, "include", "i", nil, "Additional files/directories to include")
	bundleBuildCmd.Flags().StringSliceVarP(&buildApprovals, "require-approval", "a", nil, "Required approval roles (e.g., pm, lead, security)")
	bundleBuildCmd.Flags().BoolVar(&buildAttest, "attest", false, "Generate Sigstore attestation")
	bundleBuildCmd.Flags().StringVar(&buildAttestFmt, "attest-format", "sigstore", "Attestation format (sigstore, in-toto, slsa)")
	bundleBuildCmd.Flags().StringSliceVarP(&buildMetadata, "metadata", "m", nil, "Bundle metadata (key=value)")
	bundleBuildCmd.Flags().StringVarP(&buildGovLevel, "governance-level", "g", "", "Governance maturity level (L1-L4)")

	// Bundle verify flags
	bundleVerifyCmd.Flags().BoolVar(&verifyStrict, "strict", false, "Fail on any error")
	bundleVerifyCmd.Flags().BoolVar(&verifyApprovals, "require-approvals", false, "Verify all required approvals are present")
	bundleVerifyCmd.Flags().BoolVar(&verifyAttestation, "verify-attestation", false, "Verify cryptographic attestation")
	bundleVerifyCmd.Flags().StringVar(&verifyPolicy, "policy", "", "Verify against policy file")
	bundleVerifyCmd.Flags().StringSliceVar(&verifyTrustedKeys, "trusted-key", nil, "Trusted public keys for signature verification")
	bundleVerifyCmd.Flags().BoolVar(&verifyOffline, "offline", false, "Allow offline verification")

	// Bundle apply flags
	bundleApplyCmd.Flags().StringVarP(&applyTargetDir, "target-dir", "t", "", "Target directory (default: current directory)")
	bundleApplyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Preview changes without applying")
	bundleApplyCmd.Flags().BoolVarP(&applyForce, "force", "f", false, "Overwrite files without prompting")
	bundleApplyCmd.Flags().BoolVarP(&applyYes, "yes", "y", false, "Auto-confirm all prompts")
	bundleApplyCmd.Flags().StringSliceVar(&applyExclude, "exclude", nil, "Exclude patterns (e.g., '*.log')")

	// Register subcommands
	bundleCmd.AddCommand(bundleBuildCmd)
	bundleCmd.AddCommand(bundleVerifyCmd)
	bundleCmd.AddCommand(bundleApplyCmd)

	// Register bundle command with root
	rootCmd.AddCommand(bundleCmd)
}
