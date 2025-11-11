package security

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCredentialStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	if store == nil {
		t.Fatal("Store should not be nil")
	}

	if store.storePath != storePath {
		t.Errorf("Store path mismatch: got %s, want %s", store.storePath, storePath)
	}
}

func TestStoreAndGetCredential(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Store a credential
	err = store.Store("github-token", "ghp_test1234567890", nil, nil)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Retrieve the credential
	value, err := store.Get("github-token")
	if err != nil {
		t.Fatalf("Failed to get credential: %v", err)
	}

	if value != "ghp_test1234567890" {
		t.Errorf("Value mismatch: got %s, want ghp_test1234567890", value)
	}
}

func TestGetNonExistentCredential(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	_, err = store.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent credential")
	}
}

func TestDeleteCredential(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Store a credential
	err = store.Store("test-key", "test-value", nil, nil)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Delete the credential
	err = store.Delete("test-key")
	if err != nil {
		t.Fatalf("Failed to delete credential: %v", err)
	}

	// Verify it's deleted
	_, err = store.Get("test-key")
	if err == nil {
		t.Error("Expected error after deletion")
	}
}

func TestListCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Store multiple credentials
	store.Store("key1", "value1", nil, nil)
	store.Store("key2", "value2", nil, nil)
	store.Store("key3", "value3", nil, nil)

	names := store.List()
	if len(names) != 3 {
		t.Errorf("Expected 3 credentials, got %d", len(names))
	}
}

func TestCredentialExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Store credential with past expiration
	pastTime := time.Now().Add(-1 * time.Hour)
	err = store.Store("expired-key", "expired-value", &pastTime, nil)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Try to get expired credential
	_, err = store.Get("expired-key")
	if err == nil {
		t.Error("Expected error for expired credential")
	}
}

func TestCredentialRotation(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Store credential with rotation policy
	rotationPolicy := &RotationPolicy{
		Enabled:      true,
		IntervalDays: 30,
		LastRotated:  time.Now().Add(-31 * 24 * time.Hour), // 31 days ago
	}

	err = store.Store("rotating-key", "rotating-value", nil, rotationPolicy)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Check if needs rotation
	needsRotation := store.CheckRotation()
	if len(needsRotation) != 1 {
		t.Errorf("Expected 1 credential needing rotation, got %d", len(needsRotation))
	}

	if needsRotation[0] != "rotating-key" {
		t.Errorf("Wrong credential needing rotation: got %s, want rotating-key", needsRotation[0])
	}

	// Mark as rotated
	err = store.MarkRotated("rotating-key")
	if err != nil {
		t.Fatalf("Failed to mark credential as rotated: %v", err)
	}

	// Check again - should not need rotation now
	needsRotation = store.CheckRotation()
	if len(needsRotation) != 0 {
		t.Errorf("Expected 0 credentials needing rotation after marking rotated, got %d", len(needsRotation))
	}
}

func TestGetCredentialInfo(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Store a credential
	err = store.Store("test-key", "test-value", nil, nil)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Get info
	info, err := store.GetInfo("test-key")
	if err != nil {
		t.Fatalf("Failed to get credential info: %v", err)
	}

	if info.Name != "test-key" {
		t.Errorf("Name mismatch: got %s, want test-key", info.Name)
	}

	if info.Value != "" {
		t.Error("Value should be empty in info")
	}
}

func TestCredentialPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	// Create store and add credential
	store1, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	err = store1.Store("persistent-key", "persistent-value", nil, nil)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Create new store instance with same path
	store2, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create second credential store: %v", err)
	}

	// Verify credential persisted
	value, err := store2.Get("persistent-key")
	if err != nil {
		t.Fatalf("Failed to get credential from second store: %v", err)
	}

	if value != "persistent-value" {
		t.Errorf("Value mismatch: got %s, want persistent-value", value)
	}
}

func TestUpdateCredential(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Store initial credential
	err = store.Store("update-key", "initial-value", nil, nil)
	if err != nil {
		t.Fatalf("Failed to store initial credential: %v", err)
	}

	// Get initial info
	info1, _ := store.GetInfo("update-key")
	createdAt := info1.CreatedAt

	// Wait a bit to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Update credential
	err = store.Store("update-key", "updated-value", nil, nil)
	if err != nil {
		t.Fatalf("Failed to update credential: %v", err)
	}

	// Get updated value
	value, err := store.Get("update-key")
	if err != nil {
		t.Fatalf("Failed to get updated credential: %v", err)
	}

	if value != "updated-value" {
		t.Errorf("Value mismatch: got %s, want updated-value", value)
	}

	// Verify timestamps
	info2, _ := store.GetInfo("update-key")
	if !info2.CreatedAt.Equal(createdAt) {
		t.Error("CreatedAt should not change on update")
	}

	if !info2.UpdatedAt.After(createdAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}

func TestEncryptionDecryption(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	testValues := []string{
		"simple",
		"with spaces",
		"special!@#$%^&*()chars",
		"unicode: 你好世界 🚀",
		"very-long-" + string(make([]byte, 1000)),
	}

	for i, testValue := range testValues {
		name := fmt.Sprintf("test-%d", i)

		// Store
		err = store.Store(name, testValue, nil, nil)
		if err != nil {
			t.Fatalf("Failed to store %s: %v", name, err)
		}

		// Retrieve
		value, err := store.Get(name)
		if err != nil {
			t.Fatalf("Failed to get %s: %v", name, err)
		}

		if value != testValue {
			t.Errorf("Value mismatch for %s: got %d bytes, want %d bytes", name, len(value), len(testValue))
		}
	}
}

func TestWrongPassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	// Create store with first passphrase
	store1, err := NewCredentialStore(storePath, "passphrase1")
	if err != nil {
		t.Fatalf("Failed to create first store: %v", err)
	}

	err = store1.Store("test-key", "test-value", nil, nil)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Try to open with different passphrase
	store2, err := NewCredentialStore(storePath, "passphrase2")
	if err != nil {
		t.Fatalf("Failed to create second store: %v", err)
	}

	// Try to get credential - should fail with wrong passphrase
	_, err = store2.Get("test-key")
	if err == nil {
		t.Error("Expected error when using wrong passphrase")
	}
}

func TestCredentialStoreFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "credentials.json")

	store, err := NewCredentialStore(storePath, "test-passphrase")
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	err = store.Store("test-key", "test-value", nil, nil)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Check file permissions
	fileInfo, err := os.Stat(storePath)
	if err != nil {
		t.Fatalf("Failed to stat store file: %v", err)
	}

	// File should be readable/writable only by owner (0600)
	if fileInfo.Mode().Perm() != 0600 {
		t.Errorf("Incorrect file permissions: got %o, want 0600", fileInfo.Mode().Perm())
	}
}
