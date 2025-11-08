package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	v1 "github.com/google/go-containerregistry/pkg/v1"

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
	approveRole      string
	approveUser      string
	approveComment   string
	approveSigType   string
	approveKeyPath   string
	approveOutput    string
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

	// Generate attestation if requested
	if buildAttest && buildAttestFmt != "" {
		fmt.Printf("\nGenerating %s attestation...\n", buildAttestFmt)

		// Determine attestation format
		var format bundle.AttestationFormat
		switch buildAttestFmt {
		case "sigstore":
			format = bundle.AttestationFormatSigstore
		case "in-toto":
			format = bundle.AttestationFormatInToto
		case "slsa":
			format = bundle.AttestationFormatSLSA
		default:
			return fmt.Errorf("unsupported attestation format: %s (supported: sigstore, in-toto, slsa)", buildAttestFmt)
		}

		// Create attestation generator
		attestOpts := bundle.AttestationOptions{
			Format:             format,
			UseKeyless:         false, // For now, require key-based signing
			IncludeRekorEntry:  false, // Rekor not yet implemented
		}

		generator := bundle.NewAttestationGenerator(attestOpts)

		// Generate attestation
		ctx := context.Background()
		attestation, err := generator.GenerateAttestation(ctx, output)
		if err != nil {
			fmt.Printf("⚠ Warning: Failed to generate attestation: %v\n", err)
			fmt.Println("Continuing without attestation...")
		} else {
			// Save attestation to bundle
			if err := bundle.AddAttestationToBundle(output, attestation); err != nil {
				fmt.Printf("⚠ Warning: Failed to add attestation to bundle: %v\n", err)
				fmt.Println("Continuing without attestation...")
			} else {
				fmt.Printf("✓ Attestation generated and added to bundle\n")
			}
		}
	}

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
		if buildAttest {
			fmt.Printf("  Attestation: %s\n", buildAttestFmt)
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

	if err := os.WriteFile(output, approvalJSON, 0644); err != nil {
		return ux.FormatError(err, "writing approval file")
	}

	fmt.Println()
	fmt.Printf("✓ Approval saved to: %s\n", output)

	return nil
}

func runBundleApprovalStatus(cmd *cobra.Command, args []string) error {
	bundlePath := args[0]

	// Check bundle exists
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return ux.FormatError(err, "bundle not found")
	}

	// Check approval files provided
	if len(statusApprovals) == 0 {
		return fmt.Errorf("at least one approval file is required (use --approvals)")
	}

	// Compute bundle digest
	fmt.Println("Computing bundle digest...")
	digest, err := bundle.ComputeBundleDigest(bundlePath)
	if err != nil {
		return ux.FormatError(err, "computing bundle digest")
	}

	fmt.Printf("Bundle digest: %s\n\n", digest)

	// Load approval files
	fmt.Println("Loading approval files...")
	var approvals []*bundle.Approval
	for _, approvalPath := range statusApprovals {
		data, err := os.ReadFile(approvalPath)
		if err != nil {
			fmt.Printf("⚠ Warning: Failed to read %s: %v\n", approvalPath, err)
			continue
		}

		var approval bundle.Approval
		if err := json.Unmarshal(data, &approval); err != nil {
			fmt.Printf("⚠ Warning: Failed to parse %s: %v\n", approvalPath, err)
			continue
		}

		approvals = append(approvals, &approval)
	}

	if len(approvals) == 0 {
		return fmt.Errorf("no valid approval files found")
	}

	fmt.Printf("Loaded %d approval(s)\n\n", len(approvals))

	// Verify each approval
	fmt.Println("Verifying approval signatures...")
	verifiedRoles := make(map[string]*bundle.Approval)
	var verificationErrors []string

	for _, approval := range approvals {
		// Create verifier
		verifier := bundle.NewVerifier(bundle.ApprovalVerificationOptions{
			BundleDigest: digest,
		})

		// Verify signature
		if err := verifier.VerifyApproval(approval); err != nil {
			verificationErrors = append(verificationErrors,
				fmt.Sprintf("Role %s (%s): ✗ INVALID - %v", approval.Role, approval.User, err))
		} else {
			fmt.Printf("✓ Role %s (%s): Valid signature\n", approval.Role, approval.User)
			verifiedRoles[approval.Role] = approval
		}
	}

	// Show verification errors if any
	if len(verificationErrors) > 0 {
		fmt.Println()
		fmt.Println("Verification Errors:")
		for _, errMsg := range verificationErrors {
			fmt.Printf("  %s\n", errMsg)
		}
	}

	fmt.Println()

	// Check required roles if specified
	if len(statusRequiredRoles) > 0 {
		fmt.Println("Checking required roles...")
		var missingRoles []string
		for _, requiredRole := range statusRequiredRoles {
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
	}

	// Display approval summary
	if !statusJSON {
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
	} else {
		// Output as JSON
		type ApprovalStatus struct {
			BundleDigest     string                        `json:"bundle_digest"`
			TotalApprovals   int                           `json:"total_approvals"`
			ValidApprovals   int                           `json:"valid_approvals"`
			InvalidApprovals int                           `json:"invalid_approvals"`
			VerifiedRoles    map[string]*bundle.Approval   `json:"verified_roles"`
			MissingRoles     []string                      `json:"missing_roles,omitempty"`
			Errors           []string                      `json:"errors,omitempty"`
		}

		missingRoles := []string{}
		if len(statusRequiredRoles) > 0 {
			for _, requiredRole := range statusRequiredRoles {
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
	bundleApproveCmd.MarkFlagRequired("role")
	bundleApproveCmd.MarkFlagRequired("user")

	// Bundle approval-status flags
	bundleApprovalStatusCmd.Flags().StringSliceVarP(&statusApprovals, "approvals", "a", nil, "Approval file paths (comma-separated) - REQUIRED")
	bundleApprovalStatusCmd.Flags().StringSliceVarP(&statusRequiredRoles, "required-roles", "r", nil, "Required roles (comma-separated)")
	bundleApprovalStatusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status as JSON")
	bundleApprovalStatusCmd.MarkFlagRequired("approvals")

	// Register subcommands
	bundleCmd.AddCommand(bundleBuildCmd)
	bundleCmd.AddCommand(bundleVerifyCmd)
	bundleCmd.AddCommand(bundleApplyCmd)
	bundleCmd.AddCommand(bundlePushCmd)
	bundleCmd.AddCommand(bundlePullCmd)
	bundleCmd.AddCommand(bundleApproveCmd)
	bundleCmd.AddCommand(bundleApprovalStatusCmd)

	// Register bundle command with root
	rootCmd.AddCommand(bundleCmd)
}
