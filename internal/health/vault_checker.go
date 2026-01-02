package health

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/specular/internal/vault"
)

// VaultChecker checks if HashiCorp Vault is accessible and healthy.
type VaultChecker struct {
	client *vault.Client
}

// NewVaultChecker creates a new Vault health checker.
// If client is nil, the checker will return unhealthy status.
func NewVaultChecker(client *vault.Client) *VaultChecker {
	return &VaultChecker{
		client: client,
	}
}

// Name returns the name of this health check.
func (c *VaultChecker) Name() string {
	return "vault-server"
}

// Check verifies Vault server is accessible and healthy.
// Returns:
//   - Healthy if Vault is reachable and responding
//   - Degraded if Vault is in standby mode
//   - Unhealthy if Vault is unreachable or sealed
func (c *VaultChecker) Check(ctx context.Context) *Result {
	if c.client == nil {
		return Unhealthy("Vault client not configured").
			WithDetail("suggestion", "Configure Vault via 'specular config set vault.enabled true'")
	}

	start := time.Now()
	err := c.client.Health(ctx)
	latency := time.Since(start)

	if err != nil {
		return Unhealthy(fmt.Sprintf("Vault health check failed: %v", err)).
			WithDetail("vault_address", c.client.Address()).
			WithDetail("error", err.Error()).
			WithLatency(latency)
	}

	return Healthy("Vault server is healthy").
		WithDetail("vault_address", c.client.Address()).
		WithDetail("mount_path", c.client.MountPath()).
		WithLatency(latency)
}

// VaultSignerChecker checks if the Vault-backed signer is operational.
type VaultSignerChecker struct {
	signer *vault.VaultSigner
}

// NewVaultSignerChecker creates a new Vault signer health checker.
// If signer is nil, the checker will return unhealthy status.
func NewVaultSignerChecker(signer *vault.VaultSigner) *VaultSignerChecker {
	return &VaultSignerChecker{
		signer: signer,
	}
}

// Name returns the name of this health check.
func (c *VaultSignerChecker) Name() string {
	return "vault-signer"
}

// Check verifies the Vault-backed signer can retrieve keys and sign data.
// Returns:
//   - Healthy if signing key is accessible and operational
//   - Unhealthy if signing key is missing or signing fails
func (c *VaultSignerChecker) Check(ctx context.Context) *Result {
	if c.signer == nil {
		return Unhealthy("Vault signer not configured").
			WithDetail("suggestion", "Configure Vault signing key via 'specular vault init-key'")
	}

	start := time.Now()

	// Try to get key info to verify key accessibility
	keyInfo, err := c.signer.GetKeyInfo(ctx)
	if err != nil {
		return Unhealthy(fmt.Sprintf("Failed to access signing key: %v", err)).
			WithDetail("error", err.Error()).
			WithDetail("identity", c.signer.Identity()).
			WithLatency(time.Since(start))
	}

	// Verify we can get the public key (cached or from Vault)
	pubKey, err := c.signer.GetPublicKey(ctx)
	if err != nil {
		return Unhealthy(fmt.Sprintf("Failed to retrieve public key: %v", err)).
			WithDetail("error", err.Error()).
			WithDetail("identity", c.signer.Identity()).
			WithLatency(time.Since(start))
	}

	latency := time.Since(start)

	return Healthy("Vault signer is operational").
		WithDetail("identity", c.signer.Identity()).
		WithDetail("algorithm", keyInfo.Algorithm).
		WithDetail("key_version", keyInfo.Version).
		WithDetail("public_key_length", len(pubKey)).
		WithLatency(latency)
}
