package bundle

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
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
	tmpFile, err := os.CreateTemp("", "specular-sign-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	tmpFile.Close()

	// Sign with GPG
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gpg", "--detach-sign", "--armor", "--output", "-", tmpFile.Name())
	if keyPath != "" {
		cmd.Args = append(cmd.Args[:2], "--local-user", keyPath)
		cmd.Args = append(cmd.Args, cmd.Args[2:]...)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpg signing failed: %w (stderr: %s)", err, stderr.String())
	}

	approval.Signature = stdout.String()

	// Get public key
	pubKeyCmd := exec.Command("gpg", "--armor", "--export")
	if keyPath != "" {
		pubKeyCmd.Args = append(pubKeyCmd.Args, keyPath)
	}

	var pubKeyOut bytes.Buffer
	pubKeyCmd.Stdout = &pubKeyOut
	if err := pubKeyCmd.Run(); err != nil {
		return fmt.Errorf("failed to export public key: %w", err)
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

	// Check if expired
	if v.options.MaxAge > 0 && approval.IsExpired(v.options.MaxAge) {
		return fmt.Errorf("approval expired (max age: %s, signed: %s)",
			v.options.MaxAge, approval.SignedAt.Format(time.RFC3339))
	}

	// Check if comment is required
	if v.options.RequireComment && approval.Comment == "" {
		return fmt.Errorf("approval comment is required but missing")
	}

	// Check if role is allowed
	if len(v.options.AllowedRoles) > 0 {
		roleAllowed := false
		for _, allowedRole := range v.options.AllowedRoles {
			if approval.Role == allowedRole {
				roleAllowed = true
				break
			}
		}
		if !roleAllowed {
			return fmt.Errorf("approval role %q is not in allowed roles: %v",
				approval.Role, v.options.AllowedRoles)
		}
	}

	// Check if key is trusted (if trust list provided)
	if len(v.options.TrustedKeys) > 0 {
		keyTrusted := false
		for _, trustedKey := range v.options.TrustedKeys {
			if approval.PublicKey == trustedKey || approval.PublicKeyFingerprint == trustedKey {
				keyTrusted = true
				break
			}
		}
		if !keyTrusted {
			return fmt.Errorf("approval public key is not in trusted keys list")
		}
	}

	// Verify signature based on type
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
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(approval.PublicKey))
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Decode signature
	signatureBytes, err := base64.StdEncoding.DecodeString(approval.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Reconstruct the message
	message := formatSignMessage(approval, v.options.BundleDigest)

	// Create signature struct
	sig := &ssh.Signature{
		Format: publicKey.Type(),
		Blob:   signatureBytes,
	}

	// Verify signature
	if err := publicKey.Verify([]byte(message), sig); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// verifyGPGSignature verifies a GPG signature using gpg command.
func (v *Verifier) verifyGPGSignature(approval *Approval) error {
	// Import public key to temporary keyring
	keyImportCmd := exec.Command("gpg", "--import", "--no-default-keyring", "--keyring", "trustedkeys.gpg")
	keyImportCmd.Stdin = strings.NewReader(approval.PublicKey)
	if err := keyImportCmd.Run(); err != nil {
		return fmt.Errorf("failed to import public key: %w", err)
	}

	// Create temporary files for message and signature
	msgFile, err := os.CreateTemp("", "specular-verify-msg-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(msgFile.Name())
	defer msgFile.Close()

	sigFile, err := os.CreateTemp("", "specular-verify-sig-*.asc")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(sigFile.Name())
	defer sigFile.Close()

	// Write message and signature
	message := formatSignMessage(approval, v.options.BundleDigest)
	if _, err := msgFile.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	msgFile.Close()

	if _, err := sigFile.Write([]byte(approval.Signature)); err != nil {
		return fmt.Errorf("failed to write signature: %w", err)
	}
	sigFile.Close()

	// Verify signature
	var stderr bytes.Buffer
	verifyCmd := exec.Command("gpg", "--verify", sigFile.Name(), msgFile.Name())
	verifyCmd.Stderr = &stderr

	if err := verifyCmd.Run(); err != nil {
		return fmt.Errorf("signature verification failed: %w (stderr: %s)", err, stderr.String())
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
			if _, err := os.Stat(candidate); err == nil {
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

// SignerFromPrivateKey creates a crypto.Signer from various private key types.
// This is useful for testing and advanced signing scenarios.
func SignerFromPrivateKey(privateKey interface{}) (crypto.Signer, error) {
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		return key, nil
	case ed25519.PrivateKey:
		return key, nil
	case crypto.Signer:
		return key, nil
	default:
		return nil, fmt.Errorf("unsupported private key type: %T", privateKey)
	}
}
