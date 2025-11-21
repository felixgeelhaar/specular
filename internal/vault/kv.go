package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Secret represents a Vault KV v2 secret.
type Secret struct {
	// Data is the secret data (map[string]interface{})
	Data map[string]interface{} `json:"data"`

	// Metadata contains version information
	Metadata *SecretMetadata `json:"metadata,omitempty"`
}

// SecretMetadata contains version metadata for KV v2 secrets.
type SecretMetadata struct {
	CreatedTime  string            `json:"created_time"`
	DeletionTime string            `json:"deletion_time"`
	Destroyed    bool              `json:"destroyed"`
	Version      int               `json:"version"`
	CustomMeta   map[string]string `json:"custom_metadata,omitempty"`
}

// KV provides access to Vault's KV v2 secrets engine.
type KV struct {
	client *Client
}

// NewKV creates a new KV v2 client.
func (c *Client) KV() *KV {
	return &KV{client: c}
}

// Put writes a secret to the KV v2 engine.
//
// Example:
//
//	err := client.KV().Put(ctx, "my-app/db-password", map[string]interface{}{
//	    "username": "admin",
//	    "password": "secret123",
//	})
func (kv *KV) Put(ctx context.Context, path string, data map[string]interface{}) error {
	return kv.PutWithMetadata(ctx, path, data, nil)
}

// PutWithMetadata writes a secret with custom metadata to the KV v2 engine.
func (kv *KV) PutWithMetadata(ctx context.Context, path string, data map[string]interface{}, metadata map[string]string) error {
	// KV v2 API path format: /v1/{mount}/data/{path}
	url := fmt.Sprintf("%s/v1/%s/data/%s", kv.client.address, kv.client.mountPath, path)

	// KV v2 requires data to be wrapped in a "data" field
	payload := map[string]interface{}{
		"data": data,
	}

	// Add custom metadata if provided
	if metadata != nil {
		payload["options"] = map[string]interface{}{
			"custom_metadata": metadata,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal secret data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	kv.client.addHeaders(req)

	resp, err := kv.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to write secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to write secret (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Get reads a secret from the KV v2 engine.
//
// Example:
//
//	secret, err := client.KV().Get(ctx, "my-app/db-password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	password := secret.Data["password"].(string)
func (kv *KV) Get(ctx context.Context, path string) (*Secret, error) {
	return kv.GetVersion(ctx, path, 0) // 0 means latest version
}

// GetVersion reads a specific version of a secret from the KV v2 engine.
func (kv *KV) GetVersion(ctx context.Context, path string, version int) (*Secret, error) {
	// KV v2 API path format: /v1/{mount}/data/{path}
	url := fmt.Sprintf("%s/v1/%s/data/%s", kv.client.address, kv.client.mountPath, path)

	// Add version parameter if specified
	if version > 0 {
		url = fmt.Sprintf("%s?version=%d", url, version)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	kv.client.addHeaders(req)

	resp, err := kv.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("secret not found at path: %s", path)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to read secret (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var response struct {
		Data *Secret `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode secret response: %w", err)
	}

	if response.Data == nil {
		return nil, fmt.Errorf("no data in secret response")
	}

	return response.Data, nil
}

// Delete soft-deletes the latest version of a secret (can be undeleted).
func (kv *KV) Delete(ctx context.Context, path string) error {
	return kv.DeleteVersions(ctx, path, []int{}) // Empty means latest version
}

// DeleteVersions soft-deletes specific versions of a secret.
func (kv *KV) DeleteVersions(ctx context.Context, path string, versions []int) error {
	// KV v2 API path format: /v1/{mount}/delete/{path}
	url := fmt.Sprintf("%s/v1/%s/delete/%s", kv.client.address, kv.client.mountPath, path)

	payload := map[string]interface{}{
		"versions": versions,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	kv.client.addHeaders(req)

	resp, err := kv.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete secret (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Destroy permanently deletes specific versions of a secret (cannot be recovered).
func (kv *KV) Destroy(ctx context.Context, path string, versions []int) error {
	// KV v2 API path format: /v1/{mount}/destroy/{path}
	url := fmt.Sprintf("%s/v1/%s/destroy/%s", kv.client.address, kv.client.mountPath, path)

	payload := map[string]interface{}{
		"versions": versions,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal destroy request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	kv.client.addHeaders(req)

	resp, err := kv.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to destroy secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to destroy secret (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// List lists secret names at a given path.
func (kv *KV) List(ctx context.Context, path string) ([]string, error) {
	// KV v2 API path format: /v1/{mount}/metadata/{path}
	url := fmt.Sprintf("%s/v1/%s/metadata/%s", kv.client.address, kv.client.mountPath, path)

	req, err := http.NewRequestWithContext(ctx, "LIST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	kv.client.addHeaders(req)

	resp, err := kv.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []string{}, nil // Empty list for non-existent paths
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list secrets (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var response struct {
		Data struct {
			Keys []string `json:"keys"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode list response: %w", err)
	}

	return response.Data.Keys, nil
}

// GetMetadata retrieves metadata for a secret without returning the secret data.
func (kv *KV) GetMetadata(ctx context.Context, path string) (*SecretMetadata, error) {
	// KV v2 API path format: /v1/{mount}/metadata/{path}
	url := fmt.Sprintf("%s/v1/%s/metadata/%s", kv.client.address, kv.client.mountPath, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	kv.client.addHeaders(req)

	resp, err := kv.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("secret not found at path: %s", path)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to read metadata (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var response struct {
		Data *SecretMetadata `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode metadata response: %w", err)
	}

	return response.Data, nil
}
