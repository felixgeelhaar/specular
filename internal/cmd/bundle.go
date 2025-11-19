package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/bundle"
	"github.com/felixgeelhaar/specular/internal/license"
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
	buildOutput    string
	buildSpec      string
	buildLock      string
	buildRouting   string
	buildPolicies  []string
	buildInclude   []string
	buildApprovals []string
	buildAttest    bool
	buildAttestFmt string
	buildMetadata  []string
	buildGovLevel  string
)

var bundleCreateCmd = &cobra.Command{
	Use:   "create [output]",
	Short: "Create a governance bundle",
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
  # Create bundle from current directory
  specular bundle create my-app-v1.0.0.sbundle.tgz

  # Create with specific files
  specular bundle create --spec spec.yaml --lock spec.lock.json bundle.sbundle.tgz

  # Create with policies
  specular bundle create --policy policies/security.yaml --policy policies/compliance.yaml bundle.sbundle.tgz

  # Create with governance level
  specular bundle create --governance-level L3 bundle.sbundle.tgz`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBundleCreate,
}

// Bundle gate command flags
var (
	gateStrict      bool
	gateApprovals   bool
	gateAttestation bool
	gatePolicy      string
	gateTrustedKeys []string
	gateOffline     bool
)

var bundleGateCmd = &cobra.Command{
	Use:   "gate <bundle>",
	Short: "Verify and gate a governance bundle",
	Long: `Verify the integrity and governance compliance of a bundle.

Gate checks:
- Manifest structure and completeness
- File checksums (SHA-256)
- Required approvals
- Cryptographic attestation
- Policy compliance
- Provider allowlist
- Drift detection

Exit codes:
  0  - OK (bundle passed all checks)
  20 - Policy violation
  30 - Drift detected
  40 - Missing required approval
  50 - Forbidden provider
  60 - Evaluation failure

Examples:
  # Basic gate check
  specular bundle gate my-app-v1.0.0.sbundle.tgz

  # Strict mode (fail on any error)
  specular bundle gate --strict bundle.sbundle.tgz

  # Require approvals
  specular bundle gate --require-approvals bundle.sbundle.tgz

  # Verify attestation
  specular bundle gate --verify-attestation bundle.sbundle.tgz`,
	Args: cobra.ExactArgs(1),
	RunE: runBundleGate,
}

// Bundle apply command flags
var (
	applyTargetDir string
	applyDryRun    bool
	applyForce     bool
	applyYes       bool
	applyExclude   []string
)

// Bundle push command flags
var (
	pushInsecure  bool
	pushPlatform  string
	pushUserAgent string
)

// Bundle pull command flags
var (
	pullInsecure  bool
	pullUserAgent string
	pullOutput    string
)

// Bundle approve command flags
var (
	approveRole    string
	approveUser    string
	approveComment string
	approveSigType string
	approveKeyPath string
	approveOutput  string
)

// Bundle approval-status command flags
var (
	statusApprovals     []string
	statusRequiredRoles []string
	statusJSON          bool
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

var bundlePushCmd = &cobra.Command{
	Use:   "push <bundle> <registry-ref>",
	Short: "Push a governance bundle to an OCI registry",
	Long: `Upload a governance bundle to an OCI-compatible container registry.

Supported registries:
- GitHub Container Registry (ghcr.io)
- Docker Hub (docker.io)
- Google Container Registry (gcr.io)
- Any OCI-compatible registry

Authentication uses Docker credentials from:
- Docker config file (~/.docker/config.json)
- Credential helpers (docker-credential-*)
- Environment variables (DOCKER_USERNAME, DOCKER_PASSWORD)

Examples:
  # Push to GitHub Container Registry
  specular bundle push my-app-v1.0.0.sbundle.tgz ghcr.io/org/my-app:v1.0.0

  # Push to Docker Hub
  specular bundle push bundle.sbundle.tgz docker.io/username/bundle:latest

  # Push to private registry
  specular bundle push bundle.sbundle.tgz registry.company.com/team/bundle:v1.0.0

  # Push to insecure registry (http)
  specular bundle push --insecure bundle.sbundle.tgz localhost:5000/bundle:test`,
	Args: cobra.ExactArgs(2),
	RunE: runBundlePush,
}

var bundlePullCmd = &cobra.Command{
	Use:   "pull <registry-ref> [output]",
	Short: "Pull a governance bundle from an OCI registry",
	Long: `Download a governance bundle from an OCI-compatible container registry.

The bundle is saved as a .sbundle.tgz file that can be verified and applied.

Authentication uses Docker credentials from:
- Docker config file (~/.docker/config.json)
- Credential helpers (docker-credential-*)
- Environment variables (DOCKER_USERNAME, DOCKER_PASSWORD)

Examples:
  # Pull from GitHub Container Registry
  specular bundle pull ghcr.io/org/my-app:v1.0.0

  # Pull with custom output path
  specular bundle pull ghcr.io/org/my-app:v1.0.0 my-app-v1.0.0.sbundle.tgz

  # Pull from Docker Hub
  specular bundle pull docker.io/username/bundle:latest

  # Pull from insecure registry (http)
  specular bundle pull --insecure localhost:5000/bundle:test`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runBundlePull,
}

var bundleApproveCmd = &cobra.Command{
	Use:   "approve <bundle>",
	Short: "Sign a bundle approval with SSH/GPG key",
	Long: `Create a cryptographic approval signature for a governance bundle.

Approvals represent stakeholder sign-off for governance decisions. Each approval
includes:
- Role (e.g., pm, lead, security, legal)
- User identifier (email or username)
- Timestamp
- Cryptographic signature (SSH or GPG)
- Optional comment

The signature proves that a specific individual in a specific role approved the
bundle at a specific time.

Supported signature types:
- SSH (default) - Uses SSH keys (~/.ssh/id_ed25519, id_rsa, etc.)
- GPG - Uses GPG keys from gpg keyring

Examples:
  # Approve as product manager with default SSH key
  specular bundle approve bundle.sbundle.tgz \
    --role pm \
    --user alice@example.com \
    --comment "Approved for Q1 release"

  # Approve with specific SSH key
  specular bundle approve bundle.sbundle.tgz \
    --role security \
    --user bob@example.com \
    --key-path ~/.ssh/work_id_ed25519

  # Approve with GPG key
  specular bundle approve bundle.sbundle.tgz \
    --role lead \
    --user charlie@example.com \
    --signature-type gpg \
    --key-path F3A29C8B

  # Save approval to specific file
  specular bundle approve bundle.sbundle.tgz \
    --role pm \
    --user alice@example.com \
    --output approvals/pm-alice.json`,
	Args: cobra.ExactArgs(1),
	RunE: runBundleApprove,
}

var bundleApprovalStatusCmd = &cobra.Command{
	Use:   "approval-status <bundle>",
	Short: "Show approval progress for a bundle",
	Long: `Display approval status and verify signatures for a governance bundle.

This command:
1. Computes the bundle digest
2. Loads approval files from the specified paths
3. Verifies each approval signature against the bundle digest
4. Shows which roles have approved and which are missing
5. Displays approval details (who, when, signature status)

Use this command to:
- Check if a bundle has all required approvals before applying
- Verify approval signatures are valid
- Audit who approved a bundle and when
- Track approval progress during governance workflows

Examples:
  # Check approval status with approval files
  specular bundle approval-status bundle.sbundle.tgz \
    --approvals pm-approval.json,lead-approval.json

  # Check status and require specific roles
  specular bundle approval-status bundle.sbundle.tgz \
    --approvals *.json \
    --required-roles pm,lead,security

  # Output status as JSON for scripting
  specular bundle approval-status bundle.sbundle.tgz \
    --approvals approvals/*.json \
    --json

  # Check status from approval directory
  specular bundle approval-status bundle.sbundle.tgz \
    --approvals approvals/pm-*.json,approvals/lead-*.json`,
	Args: cobra.ExactArgs(1),
	RunE: runBundleApprovalStatus,
}

// Bundle diff command flags
var (
	diffJSON  bool
	diffQuiet bool
)

var bundleDiffCmd = &cobra.Command{
	Use:   "diff <bundle-a> <bundle-b>",
	Short: "Compare two governance bundles",
	Long: `Compare two governance bundles and show their differences.

This command loads two bundles and compares:
- Files: Shows files added, removed, or modified (with checksum changes)
- Approvals: Shows approval changes (added or removed)
- Attestations: Indicates if attestation has changed
- Metadata: Shows changes to bundle metadata (version, name, governance level)

Use this command to:
- Review changes between bundle versions
- Verify what changed before applying an update
- Audit differences for compliance purposes
- Track bundle evolution over time

Exit codes:
  0 - Bundles are identical or differences displayed successfully
  1 - Error occurred during comparison
  2 - Differences found (when using --quiet)

Examples:
  # Compare two bundle versions
  specular bundle diff v1.0.0.sbundle.tgz v1.1.0.sbundle.tgz

  # Compare with JSON output for scripting
  specular bundle diff old.sbundle.tgz new.sbundle.tgz --json

  # Quiet mode - only exit code (0=identical, 2=different)
  specular bundle diff bundle-a.sbundle.tgz bundle-b.sbundle.tgz --quiet
  if [ $? -eq 2 ]; then
    echo "Bundles differ"
  fi`,
	Args: cobra.ExactArgs(2),
	RunE: runBundleDiff,
}

// determineOutputPathAndDefaults applies default paths if not explicitly set
func determineOutputPathAndDefaults(cmd *cobra.Command, args []string, defaults *ux.PathDefaults) string {
	// Determine output path
	output := buildOutput
	if len(args) > 0 {
		output = args[0]
	}
	if output == "" {
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

	return output
}

// parseMetadataFlags converts metadata flags (key=value format) to a map
func parseMetadataFlags(metadataFlags []string) map[string]string {
	metadata := make(map[string]string)
	for _, m := range metadataFlags {
		parts := strings.SplitN(m, "=", 2)
		if len(parts) == 2 {
			metadata[parts[0]] = parts[1]
		}
	}
	return metadata
}

// generateBundleAttestation generates and adds attestation to the bundle
func generateBundleAttestation(output, attestFmt string) error {
	fmt.Printf("\nGenerating %s attestation...\n", attestFmt)

	// Determine attestation format
	var format bundle.AttestationFormat
	switch attestFmt {
	case "sigstore":
		format = bundle.AttestationFormatSigstore
	case "in-toto":
		format = bundle.AttestationFormatInToto
	case "slsa":
		format = bundle.AttestationFormatSLSA
	default:
		return fmt.Errorf("unsupported attestation format: %s (supported: sigstore, in-toto, slsa)", attestFmt)
	}

	// Create attestation generator
	attestOpts := bundle.AttestationOptions{
		Format:            format,
		UseKeyless:        false, // For now, require key-based signing
		IncludeRekorEntry: false, // Rekor not yet implemented
	}

	generator := bundle.NewAttestationGenerator(attestOpts)

	// Generate attestation
	ctx := context.Background()
	attestation, attestErr := generator.GenerateAttestation(ctx, output)
	if attestErr != nil {
		fmt.Printf("⚠ Warning: Failed to generate attestation: %v\n", attestErr)
		fmt.Println("Continuing without attestation...")
		return nil // Non-fatal
	}

	// Save attestation to bundle
	if addErr := bundle.AddAttestationToBundle(output, attestation); addErr != nil {
		fmt.Printf("⚠ Warning: Failed to add attestation to bundle: %v\n", addErr)
		fmt.Println("Continuing without attestation...")
		return nil // Non-fatal
	}

	fmt.Printf("✓ Attestation generated and added to bundle\n")
	return nil
}

// displayBundleDetails shows bundle information and metadata
func displayBundleDetails(output, attestFmt string, includeAttest bool) {
	bundleInfo, err := bundle.GetBundleInfo(output)
	if err != nil {
		return // Silently skip if bundle info unavailable
	}

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
	if includeAttest {
		fmt.Printf("  Attestation: %s\n", attestFmt)
	}
}

func runBundleCreate(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	output := determineOutputPathAndDefaults(cmd, args, defaults)
	metadata := parseMetadataFlags(buildMetadata)

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
	if buildErr := builder.Build(output); buildErr != nil {
		return ux.FormatError(buildErr, "building bundle")
	}

	// Get file info
	info, err := os.Stat(output)
	if err != nil {
		return ux.FormatError(err, "reading bundle info")
	}

	fmt.Printf("\n✓ Bundle created successfully: %s (%.2f MB)\n", output, float64(info.Size())/(1024*1024))

	// Generate attestation if requested
	if buildAttest && buildAttestFmt != "" {
		if attestErr := generateBundleAttestation(output, buildAttestFmt); attestErr != nil {
			return attestErr
		}
	}

	// Display bundle details
	displayBundleDetails(output, buildAttestFmt, buildAttest)

	return nil
}

func runBundleGate(cmd *cobra.Command, args []string) error {
	// Check license - bundle gate requires Pro tier
	if err := license.RequireFeature("bundle.gate", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "bundle gate")
		return err
	}

	bundlePath := args[0]

	// Check bundle exists
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return ux.FormatError(err, "bundle not found")
	}

	fmt.Printf("Running governance gate checks on: %s\n\n", bundlePath)

	// Create validator
	opts := bundle.VerifyOptions{
		Strict:             gateStrict,
		RequireApprovals:   gateApprovals,
		RequireAttestation: gateAttestation,
		PolicyPath:         gatePolicy,
		TrustPublicKeys:    gateTrustedKeys,
		AllowOffline:       gateOffline,
	}

	validator := bundle.NewValidator(opts)

	// Verify bundle
	result, err := validator.Verify(bundlePath)
	if err != nil {
		fmt.Printf("✗ Gate check FAILED: %v\n", err)
		os.Exit(60) // Evaluation failure
	}

	// Display results
	if result.Valid {
		fmt.Println("✓ Bundle gate check PASSED")
	} else {
		fmt.Println("✗ Bundle gate check FAILED")
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

	// Determine exit code based on failure type
	if !result.Valid {
		fmt.Println()

		// Check for specific failure types and return appropriate exit codes
		for _, err := range result.Errors {
			switch err.Code {
			case "POLICY_VIOLATION", "POLICY_COMPLIANCE_FAILED":
				fmt.Println("Exit code: 20 (Policy violation)")
				os.Exit(20)
			case "DRIFT_DETECTED":
				fmt.Println("Exit code: 30 (Drift detected)")
				os.Exit(30)
			case "MISSING_APPROVAL", "APPROVAL_FAILED":
				fmt.Println("Exit code: 40 (Missing approval)")
				os.Exit(40)
			case "FORBIDDEN_PROVIDER", "PROVIDER_NOT_ALLOWED":
				fmt.Println("Exit code: 50 (Forbidden provider)")
				os.Exit(50)
			}
		}

		// Default failure exit code
		fmt.Println("Exit code: 60 (Evaluation failure)")
		os.Exit(60)
	}

	// Success
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
	if applyErr := extractor.Apply(bundlePath); applyErr != nil {
		return ux.FormatError(applyErr, "applying bundle")
	}

	return nil
}

func runBundlePush(cmd *cobra.Command, args []string) error {
	bundlePath := args[0]
	registryRef := args[1]

	// Check bundle exists
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return ux.FormatError(err, "bundle not found")
	}

	fmt.Printf("Pushing bundle to: %s\n", registryRef)
	fmt.Println()

	// Create OCI pusher options
	opts := bundle.OCIOptions{
		Reference: registryRef,
		Insecure:  pushInsecure,
		UserAgent: pushUserAgent,
	}

	// Parse platform if specified
	if pushPlatform != "" {
		// Simple platform parsing (e.g., "linux/amd64")
		parts := strings.SplitN(pushPlatform, "/", 2)
		if len(parts) == 2 {
			opts.Platform = &v1.Platform{
				OS:           parts[0],
				Architecture: parts[1],
			}
		}
	}

	pusher := bundle.NewOCIPusher(opts)

	// Push bundle
	if err := pusher.Push(bundlePath); err != nil {
		return ux.FormatError(err, "pushing bundle")
	}

	return nil
}

func runBundlePull(cmd *cobra.Command, args []string) error {
	registryRef := args[0]

	// Determine output path
	output := pullOutput
	if len(args) > 1 {
		output = args[1]
	}
	if output == "" {
		// Generate default output name from reference
		// Extract the last part of the reference for filename
		refParts := strings.Split(registryRef, "/")
		lastPart := refParts[len(refParts)-1]

		// Remove tag/digest from name
		name := strings.Split(lastPart, ":")[0]
		name = strings.Split(name, "@")[0]

		output = fmt.Sprintf("%s.sbundle.tgz", name)
	}

	fmt.Printf("Pulling bundle from: %s\n", registryRef)
	fmt.Printf("Output: %s\n", output)
	fmt.Println()

	// Create OCI puller
	opts := bundle.OCIOptions{
		Reference: registryRef,
		Insecure:  pullInsecure,
		UserAgent: pullUserAgent,
	}

	puller := bundle.NewOCIPuller(opts)

	// Pull bundle
	if err := puller.Pull(output); err != nil {
		return ux.FormatError(err, "pulling bundle")
	}

	return nil
}

func runBundleApprove(cmd *cobra.Command, args []string) error {
	bundlePath := args[0]

	// Check bundle exists
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return ux.FormatError(err, "bundle not found")
	}

	// Validate required flags
	if approveRole == "" {
		return fmt.Errorf("--role is required (e.g., pm, lead, security, legal)")
	}
	if approveUser == "" {
		return fmt.Errorf("--user is required (e.g., your email or username)")
	}

	// Compute bundle digest
	fmt.Println("Computing bundle digest...")
	digest, err := bundle.ComputeBundleDigest(bundlePath)
	if err != nil {
		return ux.FormatError(err, "computing bundle digest")
	}

	fmt.Printf("Bundle digest: %s\n\n", digest)

	// Parse signature type
	sigType := bundle.SignatureType(approveSigType)
	if sigType == "" {
		sigType = bundle.SignatureTypeSSH // Default to SSH
	}

	// Create approval request
	req := bundle.ApprovalRequest{
		BundleDigest:  digest,
		Role:          approveRole,
		User:          approveUser,
		Comment:       approveComment,
		SignatureType: sigType,
		KeyPath:       approveKeyPath,
	}

	// Create signer
	signer := bundle.NewSigner(sigType, approveKeyPath)

	// Sign approval
	fmt.Printf("Creating %s signature...\n", sigType)
	approval, err := signer.SignApproval(req)
	if err != nil {
		return ux.FormatError(err, "signing approval")
	}

	fmt.Println("✓ Approval signed successfully")
	fmt.Println()

	// Display approval details
	fmt.Println("Approval Details:")
	fmt.Printf("  Role:      %s\n", approval.Role)
	fmt.Printf("  User:      %s\n", approval.User)
	fmt.Printf("  Signed At: %s\n", approval.SignedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Signature: %s\n", sigType)
	if approval.PublicKeyFingerprint != "" {
		fmt.Printf("  Key:       %s\n", approval.PublicKeyFingerprint)
	}
	if approval.Comment != "" {
		fmt.Printf("  Comment:   %s\n", approval.Comment)
	}

	// Determine output path
	output := approveOutput
	if output == "" {
		// Generate default output filename
		bundleBase := filepath.Base(bundlePath)
		bundleBase = strings.TrimSuffix(bundleBase, ".sbundle.tgz")
		bundleBase = strings.TrimSuffix(bundleBase, ".tgz")
		output = fmt.Sprintf("%s-%s-%s-approval.json", bundleBase, approveRole, approval.SignedAt.Format("20060102-150405"))
	}

	// Write approval to file
	approvalJSON, err := approval.ToJSON()
	if err != nil {
		return ux.FormatError(err, "marshaling approval")
	}

	if writeErr := os.WriteFile(output, approvalJSON, 0600); writeErr != nil {
		return ux.FormatError(writeErr, "writing approval file")
	}

	fmt.Println()
	fmt.Printf("✓ Approval saved to: %s\n", output)

	return nil
}

// loadApprovalFiles loads and parses approval files from disk
func loadApprovalFiles(approvalPaths []string) ([]*bundle.Approval, error) {
	if len(approvalPaths) == 0 {
		return nil, fmt.Errorf("at least one approval file is required")
	}

	fmt.Println("Loading approval files...")
	var approvals []*bundle.Approval

	for _, approvalPath := range approvalPaths {
		data, err := os.ReadFile(approvalPath)
		if err != nil {
			fmt.Printf("⚠ Warning: Failed to read %s: %v\n", approvalPath, err)
			continue
		}

		var approval bundle.Approval
		if unmarshalErr := json.Unmarshal(data, &approval); unmarshalErr != nil {
			fmt.Printf("⚠ Warning: Failed to parse %s: %v\n", approvalPath, unmarshalErr)
			continue
		}

		approvals = append(approvals, &approval)
	}

	if len(approvals) == 0 {
		return nil, fmt.Errorf("no valid approval files found")
	}

	fmt.Printf("Loaded %d approval(s)\n\n", len(approvals))
	return approvals, nil
}

// verifyApprovalSignatures verifies all approval signatures and returns verified roles and errors
func verifyApprovalSignatures(approvals []*bundle.Approval, digest string) (map[string]*bundle.Approval, []string) {
	fmt.Println("Verifying approval signatures...")
	verifiedRoles := make(map[string]*bundle.Approval)
	var verificationErrors []string

	for _, approval := range approvals {
		verifier := bundle.NewVerifier(bundle.ApprovalVerificationOptions{
			BundleDigest: digest,
		})

		if err := verifier.VerifyApproval(approval); err != nil {
			verificationErrors = append(verificationErrors,
				fmt.Sprintf("Role %s (%s): ✗ INVALID - %v", approval.Role, approval.User, err))
		} else {
			fmt.Printf("✓ Role %s (%s): Valid signature\n", approval.Role, approval.User)
			verifiedRoles[approval.Role] = approval
		}
	}

	if len(verificationErrors) > 0 {
		fmt.Println()
		fmt.Println("Verification Errors:")
		for _, errMsg := range verificationErrors {
			fmt.Printf("  %s\n", errMsg)
		}
	}

	fmt.Println()
	return verifiedRoles, verificationErrors
}

// checkRequiredRoles validates that all required roles have approved
func checkRequiredRoles(requiredRoles []string, verifiedRoles map[string]*bundle.Approval) error {
	if len(requiredRoles) == 0 {
		return nil
	}

	fmt.Println("Checking required roles...")
	var missingRoles []string

	for _, requiredRole := range requiredRoles {
		if _, exists := verifiedRoles[requiredRole]; !exists {
			missingRoles = append(missingRoles, requiredRole)
			fmt.Printf("✗ %s: Missing or invalid approval\n", requiredRole)
		} else {
			fmt.Printf("✓ %s: Approved\n", requiredRole)
		}
	}

	fmt.Println()

	if len(missingRoles) > 0 {
		fmt.Printf("⚠ Bundle is missing %d required approval(s): %s\n",
			len(missingRoles), strings.Join(missingRoles, ", "))
		fmt.Println()
		return fmt.Errorf("bundle requires approvals from: %s", strings.Join(missingRoles, ", "))
	}

	fmt.Println("✓ All required roles have approved")
	return nil
}

// displayApprovalSummaryText displays human-readable approval summary
func displayApprovalSummaryText(verifiedRoles map[string]*bundle.Approval, verificationErrors []string) {
	fmt.Println()
	fmt.Println("Approval Summary:")
	fmt.Printf("  Total approvals: %d\n", len(verifiedRoles))
	fmt.Printf("  Valid signatures: %d\n", len(verifiedRoles))
	fmt.Printf("  Invalid signatures: %d\n", len(verificationErrors))

	if len(verifiedRoles) > 0 {
		fmt.Println()
		fmt.Println("Approved by:")
		for role, approval := range verifiedRoles {
			fmt.Printf("  - %s: %s (signed %s)\n",
				role,
				approval.User,
				approval.SignedAt.Format("2006-01-02 15:04:05"))
			if approval.Comment != "" {
				fmt.Printf("    Comment: %s\n", approval.Comment)
			}
		}
	}
}

// outputApprovalStatusJSON outputs approval status as JSON
func outputApprovalStatusJSON(digest string, approvals []*bundle.Approval, verifiedRoles map[string]*bundle.Approval, verificationErrors []string, requiredRoles []string) error {
	type ApprovalStatus struct {
		BundleDigest     string                      `json:"bundle_digest"`
		TotalApprovals   int                         `json:"total_approvals"`
		ValidApprovals   int                         `json:"valid_approvals"`
		InvalidApprovals int                         `json:"invalid_approvals"`
		VerifiedRoles    map[string]*bundle.Approval `json:"verified_roles"`
		MissingRoles     []string                    `json:"missing_roles,omitempty"`
		Errors           []string                    `json:"errors,omitempty"`
	}

	missingRoles := []string{}
	if len(requiredRoles) > 0 {
		for _, requiredRole := range requiredRoles {
			if _, exists := verifiedRoles[requiredRole]; !exists {
				missingRoles = append(missingRoles, requiredRole)
			}
		}
	}

	status := ApprovalStatus{
		BundleDigest:     digest,
		TotalApprovals:   len(approvals),
		ValidApprovals:   len(verifiedRoles),
		InvalidApprovals: len(verificationErrors),
		VerifiedRoles:    verifiedRoles,
		MissingRoles:     missingRoles,
		Errors:           verificationErrors,
	}

	output, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return ux.FormatError(err, "marshaling status")
	}

	fmt.Println(string(output))
	return nil
}

func runBundleApprovalStatus(cmd *cobra.Command, args []string) error {
	bundlePath := args[0]

	// Check bundle exists
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return ux.FormatError(err, "bundle not found")
	}

	// Compute bundle digest
	fmt.Println("Computing bundle digest...")
	digest, err := bundle.ComputeBundleDigest(bundlePath)
	if err != nil {
		return ux.FormatError(err, "computing bundle digest")
	}
	fmt.Printf("Bundle digest: %s\n\n", digest)

	// Load approval files
	approvals, err := loadApprovalFiles(statusApprovals)
	if err != nil {
		return err
	}

	// Verify signatures
	verifiedRoles, verificationErrors := verifyApprovalSignatures(approvals, digest)

	// Check required roles
	if roleCheckErr := checkRequiredRoles(statusRequiredRoles, verifiedRoles); roleCheckErr != nil {
		return roleCheckErr
	}

	// Display results
	if statusJSON {
		return outputApprovalStatusJSON(digest, approvals, verifiedRoles, verificationErrors, statusRequiredRoles)
	}
	displayApprovalSummaryText(verifiedRoles, verificationErrors)
	return nil
}

// loadBundlesForDiff loads and validates two bundles for comparison
func loadBundlesForDiff(bundlePathA, bundlePathB string, quiet bool) (*bundle.Bundle, *bundle.Bundle, error) {
	// Check both bundles exist
	if _, err := os.Stat(bundlePathA); os.IsNotExist(err) {
		return nil, nil, ux.FormatError(err, fmt.Sprintf("bundle A not found: %s", bundlePathA))
	}
	if _, err := os.Stat(bundlePathB); os.IsNotExist(err) {
		return nil, nil, ux.FormatError(err, fmt.Sprintf("bundle B not found: %s", bundlePathB))
	}

	if !quiet {
		fmt.Printf("Comparing bundles:\n")
		fmt.Printf("  A: %s\n", bundlePathA)
		fmt.Printf("  B: %s\n\n", bundlePathB)
		fmt.Println("Loading bundles...")
	}

	bundleA, err := bundle.LoadBundle(bundlePathA)
	if err != nil {
		return nil, nil, ux.FormatError(err, "loading bundle A")
	}

	bundleB, err := bundle.LoadBundle(bundlePathB)
	if err != nil {
		return nil, nil, ux.FormatError(err, "loading bundle B")
	}

	return bundleA, bundleB, nil
}

// displayFileDiffChanges shows file additions, removals, and modifications
func displayFileDiffChanges(diffResult *bundle.DiffResult) {
	if len(diffResult.FilesAdded) > 0 {
		fmt.Printf("Files Added (%d):\n", len(diffResult.FilesAdded))
		for _, file := range diffResult.FilesAdded {
			fmt.Printf("  + %s (checksum: %s)\n", file.Path, file.Checksum[:16]+"...")
		}
		fmt.Println()
	}

	if len(diffResult.FilesRemoved) > 0 {
		fmt.Printf("Files Removed (%d):\n", len(diffResult.FilesRemoved))
		for _, file := range diffResult.FilesRemoved {
			fmt.Printf("  - %s (checksum: %s)\n", file.Path, file.Checksum[:16]+"...")
		}
		fmt.Println()
	}

	if len(diffResult.FilesModified) > 0 {
		fmt.Printf("Files Modified (%d):\n", len(diffResult.FilesModified))
		for _, file := range diffResult.FilesModified {
			fmt.Printf("  M %s\n", file.Path)
			fmt.Printf("    Old: %s\n", file.OldChecksum[:16]+"...")
			fmt.Printf("    New: %s\n", file.NewChecksum[:16]+"...")
		}
		fmt.Println()
	}
}

// displayApprovalDiffChanges shows approval additions and removals
func displayApprovalDiffChanges(diffResult *bundle.DiffResult) {
	if len(diffResult.ApprovalsAdded) > 0 {
		fmt.Printf("Approvals Added (%d):\n", len(diffResult.ApprovalsAdded))
		for _, approval := range diffResult.ApprovalsAdded {
			fmt.Printf("  + Role: %s, User: %s\n", approval.Role, approval.User)
		}
		fmt.Println()
	}

	if len(diffResult.ApprovalsRemoved) > 0 {
		fmt.Printf("Approvals Removed (%d):\n", len(diffResult.ApprovalsRemoved))
		for _, approval := range diffResult.ApprovalsRemoved {
			fmt.Printf("  - Role: %s, User: %s\n", approval.Role, approval.User)
		}
		fmt.Println()
	}
}

// displayOtherDiffChanges shows attestation and metadata changes
func displayOtherDiffChanges(diffResult *bundle.DiffResult) {
	if diffResult.AttestationChanged {
		fmt.Println("Attestation: CHANGED")
		fmt.Println()
	}

	if diffResult.MetadataChanged {
		fmt.Println("Metadata Changes:")
		for key, change := range diffResult.ManifestMetadataChanges {
			fmt.Printf("  %s: %s\n", key, change)
		}
		fmt.Println()
	}
}

func runBundleDiff(cmd *cobra.Command, args []string) error {
	bundlePathA := args[0]
	bundlePathB := args[1]

	// Load both bundles
	bundleA, bundleB, err := loadBundlesForDiff(bundlePathA, bundlePathB, diffQuiet)
	if err != nil {
		return err
	}

	// Perform diff
	diffResult, err := bundle.DiffBundles(bundleA, bundleB)
	if err != nil {
		return ux.FormatError(err, "comparing bundles")
	}

	// Handle quiet mode
	if diffQuiet {
		if diffResult.HasChanges() {
			os.Exit(2) // Exit code 2 indicates differences found
		}
		return nil // Exit code 0 indicates identical bundles
	}

	// Handle JSON output
	if diffJSON {
		output, marshalErr := json.MarshalIndent(diffResult, "", "  ")
		if marshalErr != nil {
			return ux.FormatError(marshalErr, "marshaling diff result")
		}
		fmt.Println(string(output))
		return nil
	}

	// Human-readable output
	if !diffResult.HasChanges() {
		fmt.Println("✓ No differences found - bundles are identical")
		return nil
	}

	fmt.Println("Differences found:")
	fmt.Println()

	// Show all changes
	displayFileDiffChanges(diffResult)
	displayApprovalDiffChanges(diffResult)
	displayOtherDiffChanges(diffResult)

	// Summary
	fmt.Printf("Summary: %s\n", diffResult.Summary())

	return nil
}

func formatValidationStatus(valid bool) string {
	if valid {
		return "✓ PASS"
	}
	return "✗ FAIL"
}

// Bundle inspect command flags
var (
	inspectJSON bool
)

var bundleInspectCmd = &cobra.Command{
	Use:   "inspect <bundle>",
	Short: "Inspect bundle contents and metadata",
	Long: `Display detailed information about a governance bundle.

Shows:
- Bundle metadata (ID, version, schema, created date)
- Governance level
- Included files with checksums
- Approvals and signatures
- Attestation status
- Policy compliance

Examples:
  # Inspect bundle with human-readable output
  specular bundle inspect my-app-v1.0.0.sbundle.tgz

  # Inspect with JSON output
  specular bundle inspect --json bundle.sbundle.tgz`,
	Args: cobra.ExactArgs(1),
	RunE: runBundleInspect,
}

// Bundle list command flags
var (
	listDir  string
	listJSON bool
)

var bundleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available governance bundles",
	Long: `List governance bundles in a directory.

Shows:
- Bundle ID
- Creation date
- File size
- Governance level
- Approval status

By default, lists bundles in .specular/bundles/ directory.

Examples:
  # List bundles in default directory
  specular bundle list

  # List bundles in specific directory
  specular bundle list --dir /path/to/bundles

  # List with JSON output
  specular bundle list --json`,
	Args: cobra.NoArgs,
	RunE: runBundleList,
}

func runBundleInspect(cmd *cobra.Command, args []string) error {
	// Check license - bundle inspect requires Pro tier
	if err := license.RequireFeature("bundle.inspect", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "bundle inspect")
		return err
	}

	bundlePath := args[0]

	// Check bundle exists
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return ux.FormatError(err, "bundle not found")
	}

	// Load bundle
	fmt.Printf("Inspecting bundle: %s\n\n", bundlePath)

	bundleData, err := bundle.LoadBundle(bundlePath)
	if err != nil {
		return ux.FormatError(err, "loading bundle")
	}

	// JSON output
	if inspectJSON {
		output, marshalErr := json.MarshalIndent(bundleData, "", "  ")
		if marshalErr != nil {
			return ux.FormatError(marshalErr, "marshaling bundle data")
		}
		fmt.Println(string(output))
		return nil
	}

	// Human-readable output
	fmt.Println("=== Bundle Metadata ===")
	fmt.Printf("ID:               %s\n", bundleData.Manifest.ID)
	fmt.Printf("Version:          %s\n", bundleData.Manifest.Version)
	fmt.Printf("Schema:           %s\n", bundleData.Manifest.Schema)
	fmt.Printf("Created:          %s\n", bundleData.Manifest.Created.Format("2006-01-02 15:04:05"))
	if bundleData.Manifest.GovernanceLevel != "" {
		fmt.Printf("Governance Level: %s\n", bundleData.Manifest.GovernanceLevel)
	}
	if bundleData.Manifest.Integrity.Digest != "" {
		fmt.Printf("Integrity Digest: %s\n", bundleData.Manifest.Integrity.Digest)
	}
	fmt.Println()

	// Files
	if len(bundleData.Manifest.Files) > 0 {
		fmt.Printf("=== Files (%d) ===\n", len(bundleData.Manifest.Files))
		for _, file := range bundleData.Manifest.Files {
			fmt.Printf("  %s\n", file.Path)
			fmt.Printf("    Size:     %d bytes\n", file.Size)
			fmt.Printf("    Checksum: %s\n", file.Checksum)
		}
		fmt.Println()
	}

	// Approvals
	if len(bundleData.Approvals) > 0 {
		fmt.Printf("=== Approvals (%d) ===\n", len(bundleData.Approvals))
		for _, approval := range bundleData.Approvals {
			fmt.Printf("  Role: %s\n", approval.Role)
			fmt.Printf("    User:      %s\n", approval.User)
			fmt.Printf("    Signed At: %s\n", approval.SignedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("    Signature: %s\n", approval.SignatureType)
			if approval.Comment != "" {
				fmt.Printf("    Comment:   %s\n", approval.Comment)
			}
			fmt.Println()
		}
	} else {
		fmt.Println("=== Approvals ===")
		fmt.Println("No approvals")
		fmt.Println()
	}

	// Attestation
	if bundleData.Attestation != nil {
		fmt.Println("=== Attestation ===")
		fmt.Printf("Format:    %s\n", bundleData.Attestation.Format)
		fmt.Printf("Timestamp: %s\n", bundleData.Attestation.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	// Metadata
	if len(bundleData.Manifest.Metadata) > 0 {
		fmt.Println("=== Custom Metadata ===")
		for key, value := range bundleData.Manifest.Metadata {
			fmt.Printf("  %s: %s\n", key, value)
		}
		fmt.Println()
	}

	return nil
}

func runBundleList(cmd *cobra.Command, args []string) error {
	// Check license - bundle list requires Pro tier
	if err := license.RequireFeature("bundle.list", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "bundle list")
		return err
	}

	// Determine bundle directory
	bundleDir := listDir
	if bundleDir == "" {
		bundleDir = filepath.Join(".specular", "bundles")
	}

	// Check if directory exists
	if _, err := os.Stat(bundleDir); os.IsNotExist(err) {
		fmt.Printf("No bundles directory found: %s\n", bundleDir)
		fmt.Println("\nRun 'specular governance init' to create the governance workspace.")
		return nil
	}

	// Read directory entries
	entries, err := os.ReadDir(bundleDir)
	if err != nil {
		return ux.FormatError(err, "reading bundles directory")
	}

	// Filter for bundle files (.sbundle.tgz or .tar)
	type BundleInfo struct {
		Path      string    `json:"path"`
		Name      string    `json:"name"`
		Size      int64     `json:"size"`
		Modified  time.Time `json:"modified"`
		BundleID  string    `json:"bundle_id,omitempty"`
		GovLevel  string    `json:"governance_level,omitempty"`
		Approvals int       `json:"approvals"`
	}

	var bundles []BundleInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sbundle.tgz") && !strings.HasSuffix(name, ".tar") {
			continue
		}

		bundlePath := filepath.Join(bundleDir, name)
		info, err := os.Stat(bundlePath)
		if err != nil {
			continue
		}

		bundleInfo := BundleInfo{
			Path:     bundlePath,
			Name:     name,
			Size:     info.Size(),
			Modified: info.ModTime(),
		}

		// Try to load bundle metadata (non-fatal if it fails)
		if bundleData, loadErr := bundle.LoadBundle(bundlePath); loadErr == nil {
			bundleInfo.BundleID = bundleData.Manifest.ID
			bundleInfo.GovLevel = bundleData.Manifest.GovernanceLevel
			bundleInfo.Approvals = len(bundleData.Approvals)
		}

		bundles = append(bundles, bundleInfo)
	}

	if len(bundles) == 0 {
		fmt.Printf("No bundles found in: %s\n", bundleDir)
		return nil
	}

	// JSON output
	if listJSON {
		output, marshalErr := json.MarshalIndent(bundles, "", "  ")
		if marshalErr != nil {
			return ux.FormatError(marshalErr, "marshaling bundle list")
		}
		fmt.Println(string(output))
		return nil
	}

	// Human-readable output
	fmt.Printf("=== Bundles in %s ===\n\n", bundleDir)

	for _, b := range bundles {
		fmt.Printf("📦 %s\n", b.Name)
		if b.BundleID != "" {
			fmt.Printf("   ID:         %s\n", b.BundleID)
		}
		fmt.Printf("   Size:       %.2f MB\n", float64(b.Size)/(1024*1024))
		fmt.Printf("   Modified:   %s\n", b.Modified.Format("2006-01-02 15:04:05"))
		if b.GovLevel != "" {
			fmt.Printf("   Gov Level:  %s\n", b.GovLevel)
		}
		fmt.Printf("   Approvals:  %d\n", b.Approvals)
		fmt.Println()
	}

	fmt.Printf("Total: %d bundle(s)\n", len(bundles))

	return nil
}

func init() {
	// Bundle create flags
	bundleCreateCmd.Flags().StringVarP(&buildOutput, "output", "o", "", "Output bundle path (default: bundle.sbundle.tgz)")
	bundleCreateCmd.Flags().StringVar(&buildSpec, "spec", "", "Path to spec.yaml (default: .specular/spec.yaml)")
	bundleCreateCmd.Flags().StringVar(&buildLock, "lock", "", "Path to spec.lock.json (default: .specular/spec.lock.json)")
	bundleCreateCmd.Flags().StringVar(&buildRouting, "routing", "", "Path to routing.yaml (default: .specular/routing.yaml)")
	bundleCreateCmd.Flags().StringSliceVarP(&buildPolicies, "policy", "p", nil, "Policy files to include (can be specified multiple times)")
	bundleCreateCmd.Flags().StringSliceVarP(&buildInclude, "include", "i", nil, "Additional files/directories to include")
	bundleCreateCmd.Flags().StringSliceVarP(&buildApprovals, "require-approval", "a", nil, "Required approval roles (e.g., pm, lead, security)")
	bundleCreateCmd.Flags().BoolVar(&buildAttest, "attest", false, "Generate Sigstore attestation")
	bundleCreateCmd.Flags().StringVar(&buildAttestFmt, "attest-format", "sigstore", "Attestation format (sigstore, in-toto, slsa)")
	bundleCreateCmd.Flags().StringSliceVarP(&buildMetadata, "metadata", "m", nil, "Bundle metadata (key=value)")
	bundleCreateCmd.Flags().StringVarP(&buildGovLevel, "governance-level", "g", "", "Governance maturity level (L1-L4)")

	// Bundle gate flags
	bundleGateCmd.Flags().BoolVar(&gateStrict, "strict", false, "Fail on any error")
	bundleGateCmd.Flags().BoolVar(&gateApprovals, "require-approvals", false, "Verify all required approvals are present")
	bundleGateCmd.Flags().BoolVar(&gateAttestation, "verify-attestation", false, "Verify cryptographic attestation")
	bundleGateCmd.Flags().StringVar(&gatePolicy, "policy", "", "Verify against policy file")
	bundleGateCmd.Flags().StringSliceVar(&gateTrustedKeys, "trusted-key", nil, "Trusted public keys for signature verification")
	bundleGateCmd.Flags().BoolVar(&gateOffline, "offline", false, "Allow offline verification")

	// Bundle apply flags
	bundleApplyCmd.Flags().StringVarP(&applyTargetDir, "target-dir", "t", "", "Target directory (default: current directory)")
	bundleApplyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Preview changes without applying")
	bundleApplyCmd.Flags().BoolVarP(&applyForce, "force", "f", false, "Overwrite files without prompting")
	bundleApplyCmd.Flags().BoolVarP(&applyYes, "yes", "y", false, "Auto-confirm all prompts")
	bundleApplyCmd.Flags().StringSliceVar(&applyExclude, "exclude", nil, "Exclude patterns (e.g., '*.log')")

	// Bundle push flags
	bundlePushCmd.Flags().BoolVar(&pushInsecure, "insecure", false, "Allow insecure registry connections (http)")
	bundlePushCmd.Flags().StringVar(&pushPlatform, "platform", "", "Target platform (e.g., linux/amd64, linux/arm64)")
	bundlePushCmd.Flags().StringVar(&pushUserAgent, "user-agent", "", "Custom user agent for registry requests")

	// Bundle pull flags
	bundlePullCmd.Flags().BoolVar(&pullInsecure, "insecure", false, "Allow insecure registry connections (http)")
	bundlePullCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "Output bundle path (default: derived from reference)")
	bundlePullCmd.Flags().StringVar(&pullUserAgent, "user-agent", "", "Custom user agent for registry requests")

	// Bundle approve flags
	bundleApproveCmd.Flags().StringVarP(&approveRole, "role", "r", "", "Approval role (e.g., pm, lead, security, legal) - REQUIRED")
	bundleApproveCmd.Flags().StringVarP(&approveUser, "user", "u", "", "Approver identifier (email or username) - REQUIRED")
	bundleApproveCmd.Flags().StringVarP(&approveComment, "comment", "c", "", "Approval comment")
	bundleApproveCmd.Flags().StringVar(&approveSigType, "signature-type", "ssh", "Signature type (ssh, gpg)")
	bundleApproveCmd.Flags().StringVarP(&approveKeyPath, "key-path", "k", "", "Path to private key (default: auto-detect)")
	bundleApproveCmd.Flags().StringVarP(&approveOutput, "output", "o", "", "Output approval file path (default: auto-generated)")
	_ = bundleApproveCmd.MarkFlagRequired("role") //nolint:errcheck // Flag exists, error would be programming error
	_ = bundleApproveCmd.MarkFlagRequired("user") //nolint:errcheck // Flag exists, error would be programming error

	// Bundle approval-status flags
	bundleApprovalStatusCmd.Flags().StringSliceVarP(&statusApprovals, "approvals", "a", nil, "Approval file paths (comma-separated) - REQUIRED")
	bundleApprovalStatusCmd.Flags().StringSliceVarP(&statusRequiredRoles, "required-roles", "r", nil, "Required roles (comma-separated)")
	bundleApprovalStatusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status as JSON")
	_ = bundleApprovalStatusCmd.MarkFlagRequired("approvals") //nolint:errcheck // Flag exists, error would be programming error

	// Bundle diff flags
	bundleDiffCmd.Flags().BoolVar(&diffJSON, "json", false, "Output diff as JSON")
	bundleDiffCmd.Flags().BoolVarP(&diffQuiet, "quiet", "q", false, "Quiet mode - only exit code (0=identical, 2=different)")

	// Bundle inspect flags
	bundleInspectCmd.Flags().BoolVar(&inspectJSON, "json", false, "Output bundle data as JSON")

	// Bundle list flags
	bundleListCmd.Flags().StringVarP(&listDir, "dir", "d", "", "Directory to list bundles from (default: .specular/bundles)")
	bundleListCmd.Flags().BoolVar(&listJSON, "json", false, "Output bundle list as JSON")

	// Register subcommands
	bundleCmd.AddCommand(bundleCreateCmd)
	bundleCmd.AddCommand(bundleGateCmd)
	bundleCmd.AddCommand(bundleInspectCmd)
	bundleCmd.AddCommand(bundleListCmd)
	bundleCmd.AddCommand(bundleApplyCmd)
	bundleCmd.AddCommand(bundlePushCmd)
	bundleCmd.AddCommand(bundlePullCmd)
	bundleCmd.AddCommand(bundleApproveCmd)
	bundleCmd.AddCommand(bundleApprovalStatusCmd)
	bundleCmd.AddCommand(bundleDiffCmd)

	// Register bundle command with root
	rootCmd.AddCommand(bundleCmd)
}
