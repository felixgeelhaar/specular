package attestation

import (
	"crypto/ecdsa"
	"testing"
)

func TestEphemeralSigner(t *testing.T) {
	identity := "test@example.com"
	signer, err := NewEphemeralSigner(identity)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	if signer.Identity() != identity {
		t.Errorf("Identity mismatch: %s != %s", signer.Identity(), identity)
	}
}

func TestSignAndVerify(t *testing.T) {
	// Create signer
	signer, err := NewEphemeralSigner("test@example.com")
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Sign data
	data := []byte("test data to sign")
	signature, publicKey, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	if len(signature) == 0 {
		t.Error("Signature is empty")
	}

	if publicKey == nil {
		t.Error("Public key is nil")
	}

	// Verify it's the correct type
	if _, ok := publicKey.(*ecdsa.PublicKey); !ok {
		t.Error("Public key is not ECDSA")
	}
}

func TestPublicKeyEncoding(t *testing.T) {
	signer, err := NewEphemeralSigner("test@example.com")
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	pubKeyBytes, err := signer.PublicKey()
	if err != nil {
		t.Fatalf("Failed to get public key: %v", err)
	}

	if len(pubKeyBytes) == 0 {
		t.Error("Public key bytes are empty")
	}

	// Test encoding
	encoded := EncodePublicKey(pubKeyBytes)
	if encoded == "" {
		t.Error("Encoded public key is empty")
	}

	// Test decoding
	decoded, err := DecodePublicKey(encoded)
	if err != nil {
		t.Fatalf("Failed to decode public key: %v", err)
	}

	if len(decoded) != len(pubKeyBytes) {
		t.Errorf("Decoded key length mismatch: %d != %d", len(decoded), len(pubKeyBytes))
	}
}

func TestSignatureEncoding(t *testing.T) {
	signer, err := NewEphemeralSigner("test@example.com")
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	data := []byte("test data")
	signature, _, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	// Test encoding
	encoded := EncodeSignature(signature)
	if encoded == "" {
		t.Error("Encoded signature is empty")
	}

	// Test decoding
	decoded, err := DecodeSignature(encoded)
	if err != nil {
		t.Fatalf("Failed to decode signature: %v", err)
	}

	if len(decoded) != len(signature) {
		t.Errorf("Decoded signature length mismatch: %d != %d", len(decoded), len(signature))
	}
}

func TestDifferentSignaturesForDifferentData(t *testing.T) {
	signer, err := NewEphemeralSigner("test@example.com")
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	data1 := []byte("data one")
	data2 := []byte("data two")

	sig1, _, err := signer.Sign(data1)
	if err != nil {
		t.Fatalf("Failed to sign data1: %v", err)
	}

	sig2, _, err := signer.Sign(data2)
	if err != nil {
		t.Fatalf("Failed to sign data2: %v", err)
	}

	// Signatures should be different
	if string(sig1) == string(sig2) {
		t.Error("Signatures for different data should be different")
	}
}

func TestInvalidBase64Decode(t *testing.T) {
	_, err := DecodeSignature("not-valid-base64!@#$")
	if err == nil {
		t.Error("Expected error for invalid base64, got nil")
	}

	_, err = DecodePublicKey("not-valid-base64!@#$")
	if err == nil {
		t.Error("Expected error for invalid base64, got nil")
	}
}
