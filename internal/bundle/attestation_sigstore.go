package bundle

import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
)

// AttestationGenerator creates cryptographic attestations for bundles.
type AttestationGenerator struct {
	opts AttestationOptions
}

// NewAttestationGenerator creates a new attestation generator.
func NewAttestationGenerator(opts AttestationOptions) *AttestationGenerator {
	// Set defaults
	if opts.RekorURL == "" {
		opts.RekorURL = "https://rekor.sigstore.dev"
	}
	if opts.FulcioURL == "" {
		opts.FulcioURL = "https://fulcio.sigstore.dev"
	}
	if opts.PredicateType == "" {
		opts.PredicateType = "https://in-toto.io/Statement/v1"
	}

	return &AttestationGenerator{
		opts: opts,
	}
}

// GenerateAttestation creates a Sigstore attestation for a bundle.
func (g *AttestationGenerator) GenerateAttestation(ctx context.Context, bundlePath string) (*Attestation, error) {
	// Compute bundle digest
	bundleDigest, err := ComputeBundleDigest(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute bundle digest: %w", err)
	}

	// Create attestation subject
	subject := AttestationSubject{
		Name: bundlePath,
		Digest: map[string]string{
			"sha256": bundleDigest,
		},
	}

	// Create predicate based on format
	var predicate interface{}
	var predicateType string

	switch g.opts.Format {
	case AttestationFormatSLSA, AttestationFormatSigstore:
		// Create SLSA provenance predicate
		predicate = g.createSLSAProvenance(bundlePath)
		predicateType = "https://slsa.dev/provenance/v1"

	case AttestationFormatInToto:
		// Create in-toto statement
		predicate = g.createInTotoStatement(bundlePath)
		predicateType = "https://in-toto.io/Statement/v1"

	default:
		return nil, fmt.Errorf("unsupported attestation format: %s", g.opts.Format)
	}

	// Create in-toto statement envelope
	statement := map[string]interface{}{
		"_type":         "https://in-toto.io/Statement/v1",
		"subject":       []AttestationSubject{subject},
		"predicateType": predicateType,
		"predicate":     predicate,
	}

	// Marshal statement to JSON
	statementJSON, err := json.Marshal(statement)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal statement: %w", err)
	}

	// Sign the attestation
	var attestSig AttestationSignature
	var rekorEntry *RekorEntry

	switch {
	case g.opts.UseKeyless:
		// Use Sigstore keyless signing
		attestSig, rekorEntry, err = g.signKeyless(ctx, statementJSON)
		if err != nil {
			return nil, fmt.Errorf("keyless signing failed: %w", err)
		}
	case g.opts.KeyPath != "":
		// Use key-based signing
		attestSig, err = g.signWithKey(ctx, statementJSON)
		if err != nil {
			return nil, fmt.Errorf("key-based signing failed: %w", err)
		}

		// Optionally upload to Rekor
		if g.opts.IncludeRekorEntry {
			rekorEntry, err = g.uploadToRekor(ctx, statementJSON, attestSig)
			if err != nil {
				return nil, fmt.Errorf("failed to upload to Rekor: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("either keyless signing or key path must be provided")
	}

	// Create attestation
	attestation := &Attestation{
		Format:        g.opts.Format,
		Subject:       subject,
		Predicate:     predicate,
		PredicateType: predicateType,
		Signature:     attestSig,
		RekorEntry:    rekorEntry,
		Timestamp:     time.Now(),
		Metadata:      g.opts.Metadata,
	}

	return attestation, nil
}

// createSLSAProvenance creates a SLSA provenance predicate.
func (g *AttestationGenerator) createSLSAProvenance(bundlePath string) *SLSAProvenance {
	now := time.Now().Format(time.RFC3339)

	return &SLSAProvenance{
		BuildType: "https://specular.dev/bundle/v1",
		Builder: SLSABuilder{
			ID: "https://specular.dev/builder@v1",
			Version: map[string]string{
				"specular": "v1.3.0",
			},
		},
		Invocation: SLSAInvocation{
			ConfigSource: SLSAConfigSource{
				URI: bundlePath,
			},
		},
		Metadata: SLSAMetadata{
			BuildInvocationID: fmt.Sprintf("bundle-%d", time.Now().Unix()),
			BuildStartedOn:    now,
			BuildFinishedOn:   now,
			Completeness: SLSACompleteness{
				Parameters:  true,
				Environment: true,
				Materials:   true,
			},
			Reproducible: true,
		},
		Materials: []SLSAMaterial{
			{
				URI: bundlePath,
			},
		},
	}
}

// createInTotoStatement creates an in-toto statement.
func (g *AttestationGenerator) createInTotoStatement(bundlePath string) map[string]interface{} {
	return map[string]interface{}{
		"builder": map[string]string{
			"id": "https://specular.dev/builder@v1",
		},
		"metadata": map[string]string{
			"buildStartedOn":  time.Now().Format(time.RFC3339),
			"buildFinishedOn": time.Now().Format(time.RFC3339),
		},
		"materials": []map[string]string{
			{
				"uri": bundlePath,
			},
		},
	}
}

// signKeyless performs Sigstore keyless signing.
func (g *AttestationGenerator) signKeyless(ctx context.Context, payload []byte) (AttestationSignature, *RekorEntry, error) {
	// For MVP, return placeholder implementation
	// Full keyless signing requires OIDC flow and Fulcio integration
	// which is complex and requires interactive auth

	return AttestationSignature{}, nil, fmt.Errorf("keyless signing not yet implemented - use key-based signing for now")
}

// signWithKey signs the attestation with a private key.
func (g *AttestationGenerator) signWithKey(ctx context.Context, payload []byte) (AttestationSignature, error) {
	// Load private key
	keyData, err := os.ReadFile(g.opts.KeyPath)
	if err != nil {
		return AttestationSignature{}, fmt.Errorf("failed to read key file: %w", err)
	}

	// Parse private key from PEM
	priv, err := cryptoutils.UnmarshalPEMToPrivateKey(keyData, cryptoutils.SkipPassword)
	if err != nil {
		return AttestationSignature{}, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create signer
	signer, err := signature.LoadSignerVerifier(priv, crypto.SHA256)
	if err != nil {
		return AttestationSignature{}, fmt.Errorf("failed to create signer: %w", err)
	}

	// Hash the payload
	hasher := sha256.New()
	hasher.Write(payload)
	digest := hasher.Sum(nil)

	// Sign the digest
	sig, err := signer.SignMessage(bytes.NewReader(digest))
	if err != nil {
		return AttestationSignature{}, fmt.Errorf("failed to sign: %w", err)
	}

	// Get public key
	pubKey, err := signer.PublicKey()
	if err != nil {
		return AttestationSignature{}, fmt.Errorf("failed to get public key: %w", err)
	}

	pubKeyPEM, err := cryptoutils.MarshalPublicKeyToPEM(pubKey)
	if err != nil {
		return AttestationSignature{}, fmt.Errorf("failed to marshal public key: %w", err)
	}

	return AttestationSignature{
		Signature:          base64.StdEncoding.EncodeToString(sig),
		PublicKey:          string(pubKeyPEM),
		SignatureAlgorithm: "ECDSA-SHA256",
	}, nil
}

// uploadToRekor uploads the signature to Rekor transparency log.
func (g *AttestationGenerator) uploadToRekor(ctx context.Context, payload []byte, sig AttestationSignature) (*RekorEntry, error) {
	// For MVP, return placeholder implementation
	// Full Rekor integration requires complex entry creation and verification

	return nil, fmt.Errorf("Rekor upload not yet implemented")
}

// AttestationVerifier verifies Sigstore attestations.
type AttestationVerifier struct {
	opts AttestationVerificationOptions
}

// NewAttestationVerifier creates a new attestation verifier.
func NewAttestationVerifier(opts AttestationVerificationOptions) *AttestationVerifier {
	if opts.RekorURL == "" {
		opts.RekorURL = "https://rekor.sigstore.dev"
	}

	return &AttestationVerifier{
		opts: opts,
	}
}

// VerifyAttestation verifies a Sigstore attestation.
func (v *AttestationVerifier) VerifyAttestation(ctx context.Context, attestation *Attestation, bundlePath string) error {
	// Validate attestation structure
	if err := attestation.Validate(); err != nil {
		return fmt.Errorf("attestation validation failed: %w", err)
	}

	// Check expiration if MaxAge is set
	if v.opts.MaxAge > 0 && attestation.IsExpired(v.opts.MaxAge) {
		return fmt.Errorf("attestation is expired (max age: %v)", v.opts.MaxAge)
	}

	// Verify subject digest matches bundle
	if bundlePath != "" {
		bundleDigest, err := ComputeBundleDigest(bundlePath)
		if err != nil {
			return fmt.Errorf("failed to compute bundle digest: %w", err)
		}

		subjectDigest, ok := attestation.Subject.Digest["sha256"]
		if !ok {
			return fmt.Errorf("attestation missing sha256 digest")
		}

		if subjectDigest != bundleDigest {
			return fmt.Errorf("attestation digest mismatch: expected %s, got %s",
				bundleDigest, subjectDigest)
		}
	}

	// Verify signature if required
	if v.opts.VerifySignature {
		if err := v.verifySignature(attestation); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Verify Rekor entry if required
	if v.opts.RequireRekorEntry {
		if !attestation.HasRekorEntry() {
			return fmt.Errorf("attestation missing required Rekor entry")
		}

		if err := v.verifyRekorEntry(ctx, attestation); err != nil {
			return fmt.Errorf("Rekor verification failed: %w", err)
		}
	}

	return nil
}

// verifySignature verifies the attestation signature.
func (v *AttestationVerifier) verifySignature(attestation *Attestation) error {
	// For MVP, basic signature verification
	// Full implementation would verify against public key or certificate

	if attestation.Signature.Signature == "" {
		return fmt.Errorf("attestation missing signature")
	}

	// If certificate is present, verify it
	if attestation.Signature.Certificate != "" {
		// Certificate verification would go here
		// This requires parsing the cert chain and verifying trust
		return fmt.Errorf("certificate verification not yet implemented")
	}

	// If public key is present, verify signature
	if attestation.Signature.PublicKey != "" {
		// Public key verification would go here
		// This requires recreating the signed payload and verifying
		return fmt.Errorf("public key verification not yet implemented")
	}

	return fmt.Errorf("no verification method available (need certificate or public key)")
}

// verifyRekorEntry verifies the Rekor transparency log entry.
func (v *AttestationVerifier) verifyRekorEntry(ctx context.Context, attestation *Attestation) error {
	if attestation.RekorEntry == nil {
		return fmt.Errorf("attestation missing Rekor entry")
	}

	// For MVP, just check entry exists
	// Full verification would fetch and verify the entry from Rekor

	if attestation.RekorEntry.UUID == "" {
		return fmt.Errorf("Rekor entry missing UUID")
	}

	// Rekor client integration would go here
	// This requires fetching the entry and verifying inclusion proof

	return fmt.Errorf("Rekor entry verification not yet implemented")
}
