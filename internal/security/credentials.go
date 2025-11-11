package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// Credential represents a securely stored credential
type Credential struct {
	// Name identifies the credential (e.g., "github-token", "aws-key")
	Name string `json:"name"`

	// Value is the encrypted credential value
	Value string `json:"value"`

	// CreatedAt is when the credential was created
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when the credential was last updated
	UpdatedAt time.Time `json:"updatedAt"`

	// ExpiresAt is when the credential expires (optional)
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	// RotationPolicy defines rotation behavior (optional)
	RotationPolicy *RotationPolicy `json:"rotationPolicy,omitempty"`

	// Metadata stores additional credential information
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RotationPolicy defines credential rotation behavior
type RotationPolicy struct {
	// Enabled indicates if auto-rotation is enabled
	Enabled bool `json:"enabled"`

	// IntervalDays is how often to rotate (in days)
	IntervalDays int `json:"intervalDays"`

	// LastRotated is when the credential was last rotated
	LastRotated time.Time `json:"lastRotated"`
}

// CredentialStore manages secure credential storage
type CredentialStore struct {
	mu sync.RWMutex

	// storePath is the file path where credentials are stored
	storePath string

	// masterKey is the encryption key derived from passphrase
	masterKey []byte

	// credentials maps credential names to encrypted credentials
	credentials map[string]*Credential
}

// NewCredentialStore creates a new credential store
func NewCredentialStore(storePath, passphrase string) (*CredentialStore, error) {
	// Derive master key from passphrase using PBKDF2
	salt := []byte("specular-credential-store") // In production, use random salt
	masterKey := pbkdf2.Key([]byte(passphrase), salt, 100000, 32, sha256.New)

	store := &CredentialStore{
		storePath:   storePath,
		masterKey:   masterKey,
		credentials: make(map[string]*Credential),
	}

	// Load existing credentials if store exists
	if _, err := os.Stat(storePath); err == nil {
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("failed to load credentials: %w", err)
		}
	}

	return store, nil
}

// Store stores a credential securely
func (s *CredentialStore) Store(name, value string, expiresAt *time.Time, rotationPolicy *RotationPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Encrypt the value
	encryptedValue, err := s.encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt credential: %w", err)
	}

	now := time.Now()

	// Check if credential already exists
	existing, exists := s.credentials[name]
	var createdAt time.Time
	if exists {
		createdAt = existing.CreatedAt
	} else {
		createdAt = now
	}

	s.credentials[name] = &Credential{
		Name:           name,
		Value:          encryptedValue,
		CreatedAt:      createdAt,
		UpdatedAt:      now,
		ExpiresAt:      expiresAt,
		RotationPolicy: rotationPolicy,
		Metadata:       make(map[string]string),
	}

	// Save to disk
	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	return nil
}

// Get retrieves a credential value
func (s *CredentialStore) Get(name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cred, exists := s.credentials[name]
	if !exists {
		return "", fmt.Errorf("credential %s not found", name)
	}

	// Check if expired
	if cred.ExpiresAt != nil && time.Now().After(*cred.ExpiresAt) {
		return "", fmt.Errorf("credential %s has expired", name)
	}

	// Decrypt the value
	value, err := s.decrypt(cred.Value)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt credential: %w", err)
	}

	return value, nil
}

// Delete removes a credential
func (s *CredentialStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.credentials[name]; !exists {
		return fmt.Errorf("credential %s not found", name)
	}

	delete(s.credentials, name)

	// Save to disk
	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	return nil
}

// List returns all credential names
func (s *CredentialStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.credentials))
	for name := range s.credentials {
		names = append(names, name)
	}
	return names
}

// GetInfo returns credential information without the value
func (s *CredentialStore) GetInfo(name string) (*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cred, exists := s.credentials[name]
	if !exists {
		return nil, fmt.Errorf("credential %s not found", name)
	}

	// Return a copy without the value
	info := &Credential{
		Name:           cred.Name,
		CreatedAt:      cred.CreatedAt,
		UpdatedAt:      cred.UpdatedAt,
		ExpiresAt:      cred.ExpiresAt,
		RotationPolicy: cred.RotationPolicy,
		Metadata:       cred.Metadata,
	}

	return info, nil
}

// CheckRotation checks if credentials need rotation
func (s *CredentialStore) CheckRotation() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	needsRotation := []string{}

	for name, cred := range s.credentials {
		if cred.RotationPolicy == nil || !cred.RotationPolicy.Enabled {
			continue
		}

		// Check if rotation interval has passed
		daysSinceRotation := time.Since(cred.RotationPolicy.LastRotated).Hours() / 24
		if int(daysSinceRotation) >= cred.RotationPolicy.IntervalDays {
			needsRotation = append(needsRotation, name)
		}
	}

	return needsRotation
}

// MarkRotated marks a credential as rotated
func (s *CredentialStore) MarkRotated(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cred, exists := s.credentials[name]
	if !exists {
		return fmt.Errorf("credential %s not found", name)
	}

	if cred.RotationPolicy == nil {
		return fmt.Errorf("credential %s has no rotation policy", name)
	}

	cred.RotationPolicy.LastRotated = time.Now()
	cred.UpdatedAt = time.Now()

	// Save to disk
	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	return nil
}

// encrypt encrypts a value using AES-GCM
func (s *CredentialStore) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a value using AES-GCM
func (s *CredentialStore) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// save saves credentials to disk
func (s *CredentialStore) save() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(s.storePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Marshal credentials to JSON
	data, err := json.MarshalIndent(s.credentials, "", "  ")
	if err != nil {
		return err
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(s.storePath, data, 0600); err != nil {
		return err
	}

	return nil
}

// load loads credentials from disk
func (s *CredentialStore) load() error {
	data, err := os.ReadFile(s.storePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &s.credentials); err != nil {
		return err
	}

	return nil
}
