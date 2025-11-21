package authz

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// mockSigner implements the Signer interface for testing.
type mockSigner struct {
	privateKey *ecdsa.PrivateKey
	identity   string
}

// newMockSigner creates a test signer.
func newMockSigner(identity string) (*mockSigner, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	return &mockSigner{
		privateKey: privateKey,
		identity:   identity,
	}, nil
}

// Sign generates an ECDSA signature.
func (m *mockSigner) Sign(data []byte) ([]byte, []byte, error) {
	hash := sha256.Sum256(data)

	r, s, err := ecdsa.Sign(rand.Reader, m.privateKey, hash[:])
	if err != nil {
		return nil, nil, err
	}

	signature := append(r.Bytes(), s.Bytes()...)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&m.privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	return signature, publicKeyBytes, nil
}

// Identity returns the signer's identity.
func (m *mockSigner) Identity() string {
	return m.identity
}

// TestSignedAuditLogger tests the signed audit logger.
func TestSignedAuditLogger(t *testing.T) {
	// Create a mock signer
	signer, err := newMockSigner("test@example.com")
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	// Create an in-memory audit logger
	inMemoryLogger := NewInMemoryAuditLogger()

	// Wrap with signed logger
	signedLogger := NewSignedAuditLogger(inMemoryLogger, signer)

	// Create a test entry
	entry := &AuditEntry{
		Timestamp:      time.Now(),
		Allowed:        true,
		Reason:         "test decision",
		UserID:         "user-1",
		Email:          "user@example.com",
		OrganizationID: "org-1",
		Role:           "admin",
		Action:         "plan:approve",
		ResourceType:   "plan",
		ResourceID:     "plan-123",
		PolicyIDs:      []string{"policy-1"},
		RequestID:      "req-1",
		Duration:       100 * time.Millisecond,
	}

	// Log the entry
	if err := signedLogger.LogDecision(context.Background(), entry); err != nil {
		t.Fatalf("failed to log entry: %v", err)
	}

	// Verify the entry was signed
	entries := inMemoryLogger.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	signed := entries[0]

	if signed.Signature == "" {
		t.Error("expected signature to be set")
	}
	if signed.PublicKey == "" {
		t.Error("expected public key to be set")
	}
	if signed.SignedBy != "test@example.com" {
		t.Errorf("expected signed_by 'test@example.com', got %s", signed.SignedBy)
	}
}

// TestAuditVerifier_ValidSignature tests verifying a valid signature.
func TestAuditVerifier_ValidSignature(t *testing.T) {
	// Create a mock signer
	signer, err := newMockSigner("admin@example.com")
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	// Create a signed entry
	entry := &AuditEntry{
		Timestamp:      time.Now(),
		Allowed:        false,
		Reason:         "policy denied",
		UserID:         "user-2",
		OrganizationID: "org-1",
		Role:           "member",
		Action:         "plan:delete",
		ResourceType:   "plan",
		ResourceID:     "plan-456",
	}

	// Sign the entry
	inMemoryLogger := NewInMemoryAuditLogger()
	signedLogger := NewSignedAuditLogger(inMemoryLogger, signer)

	if err := signedLogger.LogDecision(context.Background(), entry); err != nil {
		t.Fatalf("failed to log entry: %v", err)
	}

	entries := inMemoryLogger.GetEntries()
	signedEntry := entries[0]

	// Verify the signature
	verifier := NewAuditVerifier()
	result, err := verifier.Verify(signedEntry)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid signature, got: %s", result.Reason)
	}
}

// TestAuditVerifier_TamperedEntry tests detecting a tampered entry.
func TestAuditVerifier_TamperedEntry(t *testing.T) {
	// Create a mock signer
	signer, err := newMockSigner("admin@example.com")
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	// Create and sign an entry
	entry := &AuditEntry{
		Timestamp:      time.Now(),
		Allowed:        true,
		Reason:         "policy allowed",
		UserID:         "user-3",
		OrganizationID: "org-1",
		Role:           "admin",
		Action:         "plan:approve",
		ResourceType:   "plan",
		ResourceID:     "plan-789",
	}

	inMemoryLogger := NewInMemoryAuditLogger()
	signedLogger := NewSignedAuditLogger(inMemoryLogger, signer)

	if err := signedLogger.LogDecision(context.Background(), entry); err != nil {
		t.Fatalf("failed to log entry: %v", err)
	}

	entries := inMemoryLogger.GetEntries()
	signedEntry := entries[0]

	// Tamper with the entry (change the decision)
	signedEntry.Allowed = false

	// Verify - should fail
	verifier := NewAuditVerifier()
	result, err := verifier.Verify(signedEntry)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if result.Valid {
		t.Error("expected invalid signature for tampered entry")
	}
	if result.Reason != "signature verification failed" {
		t.Errorf("expected 'signature verification failed', got: %s", result.Reason)
	}
}

// TestAuditVerifier_UnsignedEntry tests verifying an unsigned entry.
func TestAuditVerifier_UnsignedEntry(t *testing.T) {
	entry := &AuditEntry{
		Timestamp:      time.Now(),
		Allowed:        true,
		Reason:         "test decision",
		UserID:         "user-4",
		OrganizationID: "org-1",
		Role:           "member",
		Action:         "plan:read",
		ResourceType:   "plan",
		ResourceID:     "plan-100",
	}

	verifier := NewAuditVerifier()
	result, err := verifier.Verify(entry)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if result.Valid {
		t.Error("expected invalid result for unsigned entry")
	}
	if result.Reason != "entry is not signed" {
		t.Errorf("expected 'entry is not signed', got: %s", result.Reason)
	}
}

// TestAuditVerifier_WithMaxAge tests age-based verification.
func TestAuditVerifier_WithMaxAge(t *testing.T) {
	// Create a mock signer
	signer, err := newMockSigner("admin@example.com")
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	// Create an old entry
	entry := &AuditEntry{
		Timestamp:      time.Now().Add(-2 * time.Hour), // 2 hours ago
		Allowed:        true,
		Reason:         "test decision",
		UserID:         "user-5",
		OrganizationID: "org-1",
		Role:           "admin",
		Action:         "plan:approve",
		ResourceType:   "plan",
		ResourceID:     "plan-200",
	}

	// Sign the entry
	inMemoryLogger := NewInMemoryAuditLogger()
	signedLogger := NewSignedAuditLogger(inMemoryLogger, signer)

	if err := signedLogger.LogDecision(context.Background(), entry); err != nil {
		t.Fatalf("failed to log entry: %v", err)
	}

	entries := inMemoryLogger.GetEntries()
	signedEntry := entries[0]

	// Verify with 1 hour max age - should fail
	verifier := NewAuditVerifier(WithMaxAge(1 * time.Hour))
	result, err := verifier.Verify(signedEntry)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if result.Valid {
		t.Error("expected invalid result for old entry")
	}

	// Verify with 3 hour max age - should pass
	verifier = NewAuditVerifier(WithMaxAge(3 * time.Hour))
	result, err = verifier.Verify(signedEntry)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid result for entry within max age, got: %s", result.Reason)
	}
}

// TestAuditVerifier_WithAllowedSigners tests signer restriction.
func TestAuditVerifier_WithAllowedSigners(t *testing.T) {
	// Create a mock signer
	signer, err := newMockSigner("unauthorized@example.com")
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	// Create and sign an entry
	entry := &AuditEntry{
		Timestamp:      time.Now(),
		Allowed:        true,
		Reason:         "test decision",
		UserID:         "user-6",
		OrganizationID: "org-1",
		Role:           "admin",
		Action:         "plan:approve",
		ResourceType:   "plan",
		ResourceID:     "plan-300",
	}

	inMemoryLogger := NewInMemoryAuditLogger()
	signedLogger := NewSignedAuditLogger(inMemoryLogger, signer)

	if err := signedLogger.LogDecision(context.Background(), entry); err != nil {
		t.Fatalf("failed to log entry: %v", err)
	}

	entries := inMemoryLogger.GetEntries()
	signedEntry := entries[0]

	// Verify with allowed signers - should fail
	verifier := NewAuditVerifier(WithAllowedSigners([]string{"admin@example.com", "system@example.com"}))
	result, err := verifier.Verify(signedEntry)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if result.Valid {
		t.Error("expected invalid result for unauthorized signer")
	}
	if result.Reason != "signer not allowed: unauthorized@example.com" {
		t.Errorf("unexpected reason: %s", result.Reason)
	}

	// Verify without signer restriction - should pass
	verifier = NewAuditVerifier()
	result, err = verifier.Verify(signedEntry)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid result without signer restriction, got: %s", result.Reason)
	}
}

// TestAuditVerifier_VerifyBatch tests batch verification.
func TestAuditVerifier_VerifyBatch(t *testing.T) {
	// Create a mock signer
	signer, err := newMockSigner("admin@example.com")
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	inMemoryLogger := NewInMemoryAuditLogger()
	signedLogger := NewSignedAuditLogger(inMemoryLogger, signer)

	// Create multiple entries
	for i := 0; i < 5; i++ {
		entry := &AuditEntry{
			Timestamp:      time.Now(),
			Allowed:        i%2 == 0,
			Reason:         "test decision",
			UserID:         "user-" + string(rune('1'+i)),
			OrganizationID: "org-1",
			Role:           "admin",
			Action:         "plan:approve",
			ResourceType:   "plan",
			ResourceID:     "plan-" + string(rune('1'+i)),
		}

		if err := signedLogger.LogDecision(context.Background(), entry); err != nil {
			t.Fatalf("failed to log entry %d: %v", i, err)
		}
	}

	entries := inMemoryLogger.GetEntries()

	// Verify all entries
	verifier := NewAuditVerifier()
	results, err := verifier.VerifyBatch(entries)
	if err != nil {
		t.Fatalf("batch verification failed: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	// All should be valid
	for i, result := range results {
		if !result.Valid {
			t.Errorf("entry %d should be valid, got: %s", i, result.Reason)
		}
	}

	// Summarize results
	summary := Summarize(results)
	if summary.Total != 5 {
		t.Errorf("expected total 5, got %d", summary.Total)
	}
	if summary.Valid != 5 {
		t.Errorf("expected 5 valid, got %d", summary.Valid)
	}
	if summary.Invalid != 0 {
		t.Errorf("expected 0 invalid, got %d", summary.Invalid)
	}
	if summary.Unsigned != 0 {
		t.Errorf("expected 0 unsigned, got %d", summary.Unsigned)
	}
}

// TestSignedAuditLogger_Integration tests end-to-end signed audit flow.
func TestSignedAuditLogger_Integration(t *testing.T) {
	// Create auth context with owner role (has full access)
	session := &auth.Session{
		UserID:           "user-integration",
		Email:            "integration@example.com",
		OrganizationID:   "org-1",
		OrganizationRole: "owner",
	}

	// Setup policy store
	policyStore := NewInMemoryPolicyStore()
	if err := policyStore.LoadBuiltInPolicies("org-1"); err != nil {
		t.Fatalf("failed to load policies: %v", err)
	}

	// Create signer
	signer, err := newMockSigner("system@example.com")
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	// Create signed audit logger
	inMemoryLogger := NewInMemoryAuditLogger()
	signedLogger := NewSignedAuditLogger(inMemoryLogger, signer)

	// Create resource store and attribute resolver
	resourceStore := NewInMemoryResourceStore()
	attrResolver := NewDefaultAttributeResolver(resourceStore)

	// Create authorization engine with signed audit logging
	engine := NewEngine(policyStore, attrResolver)
	WithAuditLogger(engine, signedLogger)
	engine.auditLogger = signedLogger

	// Make an authorization decision
	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-integration-1",
		},
	}

	decision, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("authorization failed: %v", err)
	}

	if !decision.Allowed {
		t.Error("expected authorization to be allowed")
	}

	// Get audit entries
	entries := inMemoryLogger.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	auditEntry := entries[0]

	// Verify the audit entry is signed
	if auditEntry.Signature == "" {
		t.Error("expected audit entry to be signed")
	}

	// Verify the signature
	verifier := NewAuditVerifier()
	result, err := verifier.Verify(auditEntry)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid signature, got: %s", result.Reason)
	}
}

// TestEncodeDecodeBase64 tests base64 encoding and decoding.
func TestEncodeDecodeBase64(t *testing.T) {
	original := []byte("test data for base64 encoding")

	encoded := encodeBase64(original)
	if encoded == "" {
		t.Error("expected non-empty encoded string")
	}

	decoded, err := decodeBase64(encoded)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if string(decoded) != string(original) {
		t.Errorf("expected %s, got %s", string(original), string(decoded))
	}
}

// TestSignerAdapter tests the signer adapter.
func TestSignerAdapter(t *testing.T) {
	identity := "adapter@example.com"
	testData := []byte("test data to sign")

	signerFunc := func(data []byte) ([]byte, []byte, error) {
		// Simple mock signature
		return []byte("signature"), []byte("publicKey"), nil
	}

	adapter := NewSignerAdapter(identity, signerFunc)

	if adapter.Identity() != identity {
		t.Errorf("expected identity %s, got %s", identity, adapter.Identity())
	}

	sig, pubKey, err := adapter.Sign(testData)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	if string(sig) != "signature" {
		t.Errorf("expected signature 'signature', got %s", string(sig))
	}

	if string(pubKey) != "publicKey" {
		t.Errorf("expected public key 'publicKey', got %s", string(pubKey))
	}
}
