package bundle

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Signer creates approval signatures for bundles.
type Signer struct {
	signatureType SignatureType
	keyPath       string
}

// NewSigner creates a new signer for the specified signature type.
func NewSigner(signatureType SignatureType, keyPath string) *Signer {
	return &Signer{
		signatureType: signatureType,
		keyPath:       keyPath,
	}
}

// SignApproval creates an approval signature for a bundle.
func (s *Signer) SignApproval(req ApprovalRequest) (*Approval, error) {
	if req.BundleDigest == "" {
		return nil, fmt.Errorf("bundle digest is required")
	}

	if req.Role == "" {
		return nil, fmt.Errorf("approval role is required")
	}

	if req.User == "" {
		return nil, fmt.Errorf("user identifier is required")
	}

	// Use provided signature type or fall back to signer's default
	sigType := req.SignatureType
	if sigType == "" {
		sigType = s.signatureType
	}

	// Use provided key path or fall back to signer's default
	keyPath := req.KeyPath
	if keyPath == "" {
		keyPath = s.keyPath
	}

	// If still no key path, try to detect default
	if keyPath == "" {
		var err error
		keyPath, err = detectDefaultKey(sigType)
		if err != nil {
			return nil, fmt.Errorf("failed to detect default key: %w", err)
		}
	}

	approval := &Approval{
		Role:          req.Role,
		User:          req.User,
		SignedAt:      time.Now(),
		SignatureType: sigType,
		Comment:       req.Comment,
	}

	// Sign based on signature type
	switch sigType {
	case SignatureTypeSSH:
		if err := s.signWithSSH(approval, req.BundleDigest, keyPath); err != nil {
			return nil, fmt.Errorf("SSH signing failed: %w", err)
		}
	case SignatureTypeGPG:
		if err := s.signWithGPG(approval, req.BundleDigest, keyPath); err != nil {
			return nil, fmt.Errorf("GPG signing failed: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported signature type: %s", sigType)
	}

	return approval, nil
}

// signWithSSH creates an SSH signature for the approval.
func (s *Signer) signWithSSH(approval *Approval, digest string, keyPath string) error {
	// Read private key
	privateKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Get public key
	publicKey := signer.PublicKey()
	approval.PublicKey = string(ssh.MarshalAuthorizedKey(publicKey))
	approval.PublicKeyFingerprint = ssh.FingerprintSHA256(publicKey)

	// Create message to sign (digest + metadata)
	message := formatSignMessage(approval, digest)

	// Sign the message
	signature, err := signer.Sign(rand.Reader, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	// Encode signature as base64
	approval.Signature = base64.StdEncoding.EncodeToString(signature.Blob)

	return nil
}

// signWithGPG creates a GPG signature using gpg command-line tool.
func (s *Signer) signWithGPG(approval *Approval, digest string, keyPath string) error {
	// For GPG, we'll use the system gpg command
	// keyPath is interpreted as the key ID/fingerprint for GPG

	// Create message to sign
	message := formatSignMessage(approval, digest)

	// Create a temporary file for the message
	tmpFile, createErr := os.CreateTemp("", "specular-sign-*.txt")
	if createErr != nil {
		return fmt.Errorf("failed to create temp file: %w", createErr)
	}
	defer func() {
		if rmErr := os.Remove(tmpFile.Name()); rmErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove temp file: %v\n", rmErr)
		}
	}()
	defer func() {
		if closeErr := tmpFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close temp file: %v\n", closeErr)
		}
	}()

	if _, writeErr := tmpFile.WriteString(message); writeErr != nil {
		return fmt.Errorf("failed to write message: %w", writeErr)
	}
	if closeErr := tmpFile.Close(); closeErr != nil {
		return fmt.Errorf("failed to close temp file before signing: %w", closeErr)
	}

	// Sign with GPG
	var stdout, stderr bytes.Buffer
	// #nosec G204 - tmpFile.Name() is from os.CreateTemp, not user input
	cmd := exec.Command("gpg", "--detach-sign", "--armor", "--output", "-", tmpFile.Name())
	if keyPath != "" {
		cmd.Args = append(cmd.Args[:2], "--local-user", keyPath)
		cmd.Args = append(cmd.Args, cmd.Args[2:]...)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		return fmt.Errorf("gpg signing failed: %w (stderr: %s)", runErr, stderr.String())
	}

	approval.Signature = stdout.String()

	// Get public key
	pubKeyCmd := exec.Command("gpg", "--armor", "--export")
	if keyPath != "" {
		pubKeyCmd.Args = append(pubKeyCmd.Args, keyPath)
	}

	var pubKeyOut bytes.Buffer
	pubKeyCmd.Stdout = &pubKeyOut
	if pubKeyErr := pubKeyCmd.Run(); pubKeyErr != nil {
		return fmt.Errorf("failed to export public key: %w", pubKeyErr)
	}

	approval.PublicKey = pubKeyOut.String()

	// Get key fingerprint
	if keyPath != "" {
		approval.PublicKeyFingerprint = keyPath
	}

	return nil
}

// Verifier verifies approval signatures.
type Verifier struct {
	options ApprovalVerificationOptions
}

// NewVerifier creates a new approval verifier.
func NewVerifier(opts ApprovalVerificationOptions) *Verifier {
	return &Verifier{options: opts}
}

// VerifyApproval verifies an approval signature.
func (v *Verifier) VerifyApproval(approval *Approval) error {
	// Basic validation
	if err := approval.Validate(); err != nil {
		return fmt.Errorf("approval validation failed: %w", err)
	}

	// Perform policy checks
	if err := v.validateApprovalPolicy(approval); err != nil {
		return err
	}

	// Verify cryptographic signature
	if err := v.verifyCryptographicSignature(approval); err != nil {
		return err
	}

	return nil
}

// validateApprovalPolicy validates approval against policy requirements
func (v *Verifier) validateApprovalPolicy(approval *Approval) error {
	// Check expiry
	if err := v.checkExpiry(approval); err != nil {
		return err
	}

	// Check comment requirement
	if err := v.checkCommentRequirement(approval); err != nil {
		return err
	}

	// Check role allowlist
	if err := v.checkRoleAllowed(approval); err != nil {
		return err
	}

	// Check key trust
	if err := v.checkKeyTrusted(approval); err != nil {
		return err
	}

	return nil
}

// checkExpiry validates that the approval hasn't expired
func (v *Verifier) checkExpiry(approval *Approval) error {
	if v.options.MaxAge > 0 && approval.IsExpired(v.options.MaxAge) {
		return fmt.Errorf("approval expired (max age: %s, signed: %s)",
			v.options.MaxAge, approval.SignedAt.Format(time.RFC3339))
	}
	return nil
}

// checkCommentRequirement validates that a comment is present if required
func (v *Verifier) checkCommentRequirement(approval *Approval) error {
	if v.options.RequireComment && approval.Comment == "" {
		return fmt.Errorf("approval comment is required but missing")
	}
	return nil
}

// checkRoleAllowed validates that the approval role is in the allowlist
func (v *Verifier) checkRoleAllowed(approval *Approval) error {
	if len(v.options.AllowedRoles) == 0 {
		return nil
	}

	for _, allowedRole := range v.options.AllowedRoles {
		if approval.Role == allowedRole {
			return nil
		}
	}

	return fmt.Errorf("approval role %q is not in allowed roles: %v",
		approval.Role, v.options.AllowedRoles)
}

// checkKeyTrusted validates that the approval key is in the trust list
func (v *Verifier) checkKeyTrusted(approval *Approval) error {
	if len(v.options.TrustedKeys) == 0 {
		return nil
	}

	for _, trustedKey := range v.options.TrustedKeys {
		if approval.PublicKey == trustedKey || approval.PublicKeyFingerprint == trustedKey {
			return nil
		}
	}

	return fmt.Errorf("approval public key is not in trusted keys list")
}

// verifyCryptographicSignature verifies the signature based on its type
func (v *Verifier) verifyCryptographicSignature(approval *Approval) error {
	switch approval.SignatureType {
	case SignatureTypeSSH:
		if err := v.verifySSHSignature(approval); err != nil {
			return fmt.Errorf("SSH signature verification failed: %w", err)
		}
	case SignatureTypeGPG:
		if err := v.verifyGPGSignature(approval); err != nil {
			return fmt.Errorf("GPG signature verification failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported signature type: %s", approval.SignatureType)
	}

	return nil
}

// verifySSHSignature verifies an SSH signature.
func (v *Verifier) verifySSHSignature(approval *Approval) error {
	// Parse public key
	publicKey, _, _, _, parseErr := ssh.ParseAuthorizedKey([]byte(approval.PublicKey))
	if parseErr != nil {
		return fmt.Errorf("failed to parse public key: %w", parseErr)
	}

	// Decode signature
	signatureBytes, decodeErr := base64.StdEncoding.DecodeString(approval.Signature)
	if decodeErr != nil {
		return fmt.Errorf("failed to decode signature: %w", decodeErr)
	}

	// Reconstruct the message
	message := formatSignMessage(approval, v.options.BundleDigest)

	// Create signature struct
	sig := &ssh.Signature{
		Format: publicKey.Type(),
		Blob:   signatureBytes,
	}

	// Verify signature
	if verifyErr := publicKey.Verify([]byte(message), sig); verifyErr != nil {
		return fmt.Errorf("signature verification failed: %w", verifyErr)
	}

	return nil
}

// verifyGPGSignature verifies a GPG signature using gpg command.
func (v *Verifier) verifyGPGSignature(approval *Approval) error {
	// Import public key to temporary keyring
	keyImportCmd := exec.Command("gpg", "--import", "--no-default-keyring", "--keyring", "trustedkeys.gpg")
	keyImportCmd.Stdin = strings.NewReader(approval.PublicKey)
	if importErr := keyImportCmd.Run(); importErr != nil {
		return fmt.Errorf("failed to import public key: %w", importErr)
	}

	// Create temporary files for message and signature
	msgFile, msgCreateErr := os.CreateTemp("", "specular-verify-msg-*.txt")
	if msgCreateErr != nil {
		return fmt.Errorf("failed to create temp file: %w", msgCreateErr)
	}
	defer func() {
		if rmErr := os.Remove(msgFile.Name()); rmErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove temp file: %v\n", rmErr)
		}
	}()
	defer func() {
		if closeErr := msgFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close temp file: %v\n", closeErr)
		}
	}()

	sigFile, sigCreateErr := os.CreateTemp("", "specular-verify-sig-*.asc")
	if sigCreateErr != nil {
		return fmt.Errorf("failed to create temp file: %w", sigCreateErr)
	}
	defer func() {
		if rmErr := os.Remove(sigFile.Name()); rmErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove temp file: %v\n", rmErr)
		}
	}()
	defer func() {
		if closeErr := sigFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close temp file: %v\n", closeErr)
		}
	}()

	// Write message and signature
	message := formatSignMessage(approval, v.options.BundleDigest)
	if _, msgWriteErr := msgFile.WriteString(message); msgWriteErr != nil {
		return fmt.Errorf("failed to write message: %w", msgWriteErr)
	}
	if msgCloseErr := msgFile.Close(); msgCloseErr != nil {
		return fmt.Errorf("failed to close message file: %w", msgCloseErr)
	}

	if _, sigWriteErr := sigFile.WriteString(approval.Signature); sigWriteErr != nil {
		return fmt.Errorf("failed to write signature: %w", sigWriteErr)
	}
	if sigCloseErr := sigFile.Close(); sigCloseErr != nil {
		return fmt.Errorf("failed to close signature file: %w", sigCloseErr)
	}

	// Verify signature
	var stderr bytes.Buffer
	// #nosec G204 - sigFile.Name() and msgFile.Name() are from os.CreateTemp, not user input
	verifyCmd := exec.Command("gpg", "--verify", sigFile.Name(), msgFile.Name())
	verifyCmd.Stderr = &stderr

	if verifyErr := verifyCmd.Run(); verifyErr != nil {
		return fmt.Errorf("signature verification failed: %w (stderr: %s)", verifyErr, stderr.String())
	}

	return nil
}

// formatSignMessage creates the canonical message format for signing.
// This ensures consistent message formatting across signing and verification.
func formatSignMessage(approval *Approval, bundleDigest string) string {
	var buf strings.Builder

	buf.WriteString("SPECULAR BUNDLE APPROVAL\n")
	buf.WriteString(fmt.Sprintf("Bundle Digest: %s\n", bundleDigest))
	buf.WriteString(fmt.Sprintf("Role: %s\n", approval.Role))
	buf.WriteString(fmt.Sprintf("User: %s\n", approval.User))
	buf.WriteString(fmt.Sprintf("Timestamp: %s\n", approval.SignedAt.Format(time.RFC3339)))

	if approval.Comment != "" {
		buf.WriteString(fmt.Sprintf("Comment: %s\n", approval.Comment))
	}

	return buf.String()
}

// detectDefaultKey attempts to find the default key for the given signature type.
func detectDefaultKey(sigType SignatureType) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch sigType {
	case SignatureTypeSSH:
		// Try common SSH key locations
		sshDir := filepath.Join(homeDir, ".ssh")
		candidates := []string{
			filepath.Join(sshDir, "id_ed25519"),
			filepath.Join(sshDir, "id_rsa"),
			filepath.Join(sshDir, "id_ecdsa"),
		}

		for _, candidate := range candidates {
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate, nil
			}
		}

		return "", fmt.Errorf("no default SSH key found in %s", sshDir)

	case SignatureTypeGPG:
		// For GPG, we'll use the default key (no key ID specified)
		return "", nil

	default:
		return "", fmt.Errorf("unsupported signature type for default key detection: %s", sigType)
	}
}

// ComputeBundleDigest computes a SHA-256 digest of a bundle file.
func ComputeBundleDigest(bundlePath string) (string, error) {
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		return "", fmt.Errorf("failed to read bundle: %w", err)
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash), nil
}

// VerifyAllApprovals verifies a set of approvals against requirements.
func VerifyAllApprovals(approvals []*Approval, opts ApprovalVerificationOptions) error {
	if len(approvals) == 0 && opts.RequireAllRoles && len(opts.AllowedRoles) > 0 {
		return fmt.Errorf("no approvals found but %d roles required", len(opts.AllowedRoles))
	}

	verifier := NewVerifier(opts)

	// Track which roles have been approved
	approvedRoles := make(map[string]bool)

	for _, approval := range approvals {
		if err := verifier.VerifyApproval(approval); err != nil {
			return fmt.Errorf("approval from %s (%s) failed verification: %w",
				approval.User, approval.Role, err)
		}
		approvedRoles[approval.Role] = true
	}

	// If all roles are required, check that all are present
	if opts.RequireAllRoles {
		for _, requiredRole := range opts.AllowedRoles {
			if !approvedRoles[requiredRole] {
				return fmt.Errorf("required role %q is missing approval", requiredRole)
			}
		}
	}

	return nil
}
