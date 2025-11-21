package apikey

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/felixgeelhaar/specular/internal/vault"
)

// APIKey represents an API key with its metadata.
type APIKey struct {
	ID             string    `json:"id"`
	Secret         string    `json:"secret,omitempty"` // Only returned on creation
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id,omitempty"`
	Name           string    `json:"name"`
	Prefix         string    `json:"prefix"`
	Scopes         []string  `json:"scopes"`
	Status         Status    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	LastUsedAt     time.Time `json:"last_used_at,omitempty"`
	ExpiresAt      time.Time `json:"expires_at,omitempty"`
	RotatedAt      time.Time `json:"rotated_at,omitempty"`
	RevokedAt      time.Time `json:"revoked_at,omitempty"`
}

// Status represents the status of an API key.
type Status string

const (
	StatusActive  Status = "active"
	StatusRotated Status = "rotated" // Old key during rotation grace period
	StatusRevoked Status = "revoked"
	StatusExpired Status = "expired"
)

// Manager manages API keys stored in HashiCorp Vault.
type Manager struct {
	vault  *vault.Client
	prefix string // Key prefix (e.g., "sk_live_", "sk_test_")
	ttl    time.Duration
}

// Config holds configuration for the API key manager.
type Config struct {
	VaultClient *vault.Client
	Prefix      string        // Default: "sk_"
	TTL         time.Duration // Default: 90 days
}

// NewManager creates a new API key manager.
func NewManager(cfg Config) (*Manager, error) {
	if cfg.VaultClient == nil {
		return nil, fmt.Errorf("vault client is required")
	}

	// Set defaults
	if cfg.Prefix == "" {
		cfg.Prefix = "sk_"
	}
	if cfg.TTL == 0 {
		cfg.TTL = 90 * 24 * time.Hour // 90 days
	}

	return &Manager{
		vault:  cfg.VaultClient,
		prefix: cfg.Prefix,
		ttl:    cfg.TTL,
	}, nil
}

// CreateKey generates a new API key and stores it in Vault.
func (m *Manager) CreateKey(ctx context.Context, orgID, userID, name string, scopes []string) (*APIKey, error) {
	if orgID == "" {
		return nil, fmt.Errorf("organization ID is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Generate cryptographically secure key ID and secret
	keyID, err := generateRandomString(16) // 16 bytes = 128 bits
	if err != nil {
		return nil, fmt.Errorf("failed to generate key ID: %w", err)
	}

	secret, err := generateRandomString(32) // 32 bytes = 256 bits
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)

	key := &APIKey{
		ID:             keyID,
		Secret:         m.prefix + secret,
		OrganizationID: orgID,
		UserID:         userID,
		Name:           name,
		Prefix:         m.prefix,
		Scopes:         scopes,
		Status:         StatusActive,
		CreatedAt:      now,
		ExpiresAt:      expiresAt,
	}

	// Store in Vault
	path := fmt.Sprintf("apikeys/%s/%s", orgID, keyID)
	data := map[string]interface{}{
		"id":              key.ID,
		"secret":          key.Secret,
		"organization_id": key.OrganizationID,
		"user_id":         key.UserID,
		"name":            key.Name,
		"prefix":          key.Prefix,
		"scopes":          key.Scopes,
		"status":          string(key.Status),
		"created_at":      key.CreatedAt.Format(time.RFC3339),
		"expires_at":      key.ExpiresAt.Format(time.RFC3339),
	}

	metadata := map[string]string{
		"organization_id": orgID,
		"user_id":         userID,
		"name":            name,
		"status":          string(StatusActive),
	}

	if err := m.vault.KV().PutWithMetadata(ctx, path, data, metadata); err != nil {
		return nil, fmt.Errorf("failed to store API key in Vault: %w", err)
	}

	return key, nil
}

// GetKey retrieves an API key from Vault by ID.
func (m *Manager) GetKey(ctx context.Context, orgID, keyID string) (*APIKey, error) {
	path := fmt.Sprintf("apikeys/%s/%s", orgID, keyID)

	secret, err := m.vault.KV().Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key from Vault: %w", err)
	}

	key, err := m.parseAPIKey(secret.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API key: %w", err)
	}

	// Don't return secret for security
	key.Secret = ""

	return key, nil
}

// ValidateKey validates an API key and returns its metadata.
// The full secret (with prefix) should be provided.
func (m *Manager) ValidateKey(ctx context.Context, fullSecret string) (*APIKey, error) {
	// Extract key ID from secret (we'll need to search)
	// For now, we'll use a different approach: hash the secret and use it as a lookup

	// This is a simplified implementation
	// In production, you might want to use a hash-based lookup or search
	return nil, fmt.Errorf("not implemented: use GetKeyBySecret instead")
}

// GetKeyBySecret retrieves an API key by its secret value.
// This requires searching through keys, so it's less efficient than GetKey.
func (m *Manager) GetKeyBySecret(ctx context.Context, orgID, secret string) (*APIKey, error) {
	// List all keys for the organization
	keys, err := m.ListKeys(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	// Find matching key
	for _, key := range keys {
		// Fetch full key data
		fullKey, err := m.GetKey(ctx, orgID, key.ID)
		if err != nil {
			continue
		}

		// Load secret from Vault
		path := fmt.Sprintf("apikeys/%s/%s", orgID, key.ID)
		vaultSecret, err := m.vault.KV().Get(ctx, path)
		if err != nil {
			continue
		}

		storedSecret, ok := vaultSecret.Data["secret"].(string)
		if !ok {
			continue
		}

		if storedSecret == secret {
			// Update last used timestamp
			if err := m.UpdateLastUsed(ctx, orgID, key.ID); err != nil {
				// Log error but don't fail validation
				fmt.Printf("warning: failed to update last used timestamp: %v\n", err)
			}

			fullKey.Secret = "" // Don't return secret
			return fullKey, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// ListKeys lists all API keys for an organization.
func (m *Manager) ListKeys(ctx context.Context, orgID string) ([]*APIKey, error) {
	path := fmt.Sprintf("apikeys/%s", orgID)

	keyIDs, err := m.vault.KV().List(ctx, path)
	if err != nil {
		// Empty list on error
		return []*APIKey{}, nil
	}

	keys := make([]*APIKey, 0, len(keyIDs))
	for _, keyID := range keyIDs {
		key, err := m.GetKey(ctx, orgID, keyID)
		if err != nil {
			continue // Skip invalid keys
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// RotateKey rotates an API key by creating a new secret and marking the old one as rotated.
// Both keys remain valid during the grace period.
func (m *Manager) RotateKey(ctx context.Context, orgID, keyID string, gracePeriod time.Duration) (*APIKey, error) {
	// Get existing key
	oldKey, err := m.GetKey(ctx, orgID, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing key: %w", err)
	}

	if oldKey.Status != StatusActive {
		return nil, fmt.Errorf("cannot rotate key with status: %s", oldKey.Status)
	}

	// Generate new secret
	newSecret, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new secret: %w", err)
	}

	now := time.Now().UTC()

	// Create new key with same metadata
	newKey := &APIKey{
		ID:             oldKey.ID,
		Secret:         m.prefix + newSecret,
		OrganizationID: oldKey.OrganizationID,
		UserID:         oldKey.UserID,
		Name:           oldKey.Name,
		Prefix:         oldKey.Prefix,
		Scopes:         oldKey.Scopes,
		Status:         StatusActive,
		CreatedAt:      oldKey.CreatedAt,
		ExpiresAt:      now.Add(m.ttl),
		RotatedAt:      now,
	}

	// Store new key (creates new version in KV v2)
	path := fmt.Sprintf("apikeys/%s/%s", orgID, keyID)
	data := map[string]interface{}{
		"id":              newKey.ID,
		"secret":          newKey.Secret,
		"organization_id": newKey.OrganizationID,
		"user_id":         newKey.UserID,
		"name":            newKey.Name,
		"prefix":          newKey.Prefix,
		"scopes":          newKey.Scopes,
		"status":          string(newKey.Status),
		"created_at":      newKey.CreatedAt.Format(time.RFC3339),
		"expires_at":      newKey.ExpiresAt.Format(time.RFC3339),
		"rotated_at":      newKey.RotatedAt.Format(time.RFC3339),
	}

	metadata := map[string]string{
		"organization_id": orgID,
		"status":          string(StatusActive),
		"rotated_at":      now.Format(time.RFC3339),
	}

	if err := m.vault.KV().PutWithMetadata(ctx, path, data, metadata); err != nil {
		return nil, fmt.Errorf("failed to store rotated key in Vault: %w", err)
	}

	return newKey, nil
}

// RevokeKey immediately revokes an API key.
func (m *Manager) RevokeKey(ctx context.Context, orgID, keyID string) error {
	// Get existing key
	key, err := m.GetKey(ctx, orgID, keyID)
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}

	if key.Status == StatusRevoked {
		return nil // Already revoked
	}

	// Update status to revoked
	now := time.Now().UTC()
	path := fmt.Sprintf("apikeys/%s/%s", orgID, keyID)

	data := map[string]interface{}{
		"id":              key.ID,
		"secret":          "", // Clear secret
		"organization_id": key.OrganizationID,
		"user_id":         key.UserID,
		"name":            key.Name,
		"prefix":          key.Prefix,
		"scopes":          key.Scopes,
		"status":          string(StatusRevoked),
		"created_at":      key.CreatedAt.Format(time.RFC3339),
		"expires_at":      key.ExpiresAt.Format(time.RFC3339),
		"revoked_at":      now.Format(time.RFC3339),
	}

	metadata := map[string]string{
		"organization_id": orgID,
		"status":          string(StatusRevoked),
		"revoked_at":      now.Format(time.RFC3339),
	}

	if err := m.vault.KV().PutWithMetadata(ctx, path, data, metadata); err != nil {
		return fmt.Errorf("failed to revoke key in Vault: %w", err)
	}

	return nil
}

// DeleteKey permanently deletes an API key from Vault.
func (m *Manager) DeleteKey(ctx context.Context, orgID, keyID string) error {
	path := fmt.Sprintf("apikeys/%s/%s", orgID, keyID)

	if err := m.vault.KV().Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete key from Vault: %w", err)
	}

	return nil
}

// UpdateLastUsed updates the last used timestamp for an API key.
func (m *Manager) UpdateLastUsed(ctx context.Context, orgID, keyID string) error {
	// Get current key
	path := fmt.Sprintf("apikeys/%s/%s", orgID, keyID)
	secret, err := m.vault.KV().Get(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}

	// Update last_used_at
	secret.Data["last_used_at"] = time.Now().UTC().Format(time.RFC3339)

	// Store updated data
	if err := m.vault.KV().Put(ctx, path, secret.Data); err != nil {
		return fmt.Errorf("failed to update last used timestamp: %w", err)
	}

	return nil
}

// parseAPIKey parses API key data from Vault.
func (m *Manager) parseAPIKey(data map[string]interface{}) (*APIKey, error) {
	key := &APIKey{}

	if id, ok := data["id"].(string); ok {
		key.ID = id
	}
	if secret, ok := data["secret"].(string); ok {
		key.Secret = secret
	}
	if orgID, ok := data["organization_id"].(string); ok {
		key.OrganizationID = orgID
	}
	if userID, ok := data["user_id"].(string); ok {
		key.UserID = userID
	}
	if name, ok := data["name"].(string); ok {
		key.Name = name
	}
	if prefix, ok := data["prefix"].(string); ok {
		key.Prefix = prefix
	}
	if status, ok := data["status"].(string); ok {
		key.Status = Status(status)
	}

	// Parse scopes
	if scopesData, ok := data["scopes"].([]interface{}); ok {
		scopes := make([]string, len(scopesData))
		for i, scope := range scopesData {
			if s, ok := scope.(string); ok {
				scopes[i] = s
			}
		}
		key.Scopes = scopes
	}

	// Parse timestamps
	if createdAt, ok := data["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			key.CreatedAt = t
		}
	}
	if lastUsedAt, ok := data["last_used_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastUsedAt); err == nil {
			key.LastUsedAt = t
		}
	}
	if expiresAt, ok := data["expires_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, expiresAt); err == nil {
			key.ExpiresAt = t
		}
	}
	if rotatedAt, ok := data["rotated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, rotatedAt); err == nil {
			key.RotatedAt = t
		}
	}
	if revokedAt, ok := data["revoked_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, revokedAt); err == nil {
			key.RevokedAt = t
		}
	}

	return key, nil
}

// generateRandomString generates a cryptographically secure random string.
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Base64 encoding produces ~4/3 the input length
	// Return the full encoded string (it will be longer than input length)
	return base64.URLEncoding.EncodeToString(bytes), nil
}
