package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/ux"
	"github.com/felixgeelhaar/specular/internal/vault"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage HashiCorp Vault integration",
	Long: `Manage HashiCorp Vault integration for secrets management and audit log signing.

Vault integration provides:
  • Secure storage for ECDSA signing keys
  • Cryptographic signatures for audit logs
  • Key rotation with version history
  • mTLS authentication support

Examples:
  # Check Vault connection status
  specular vault status

  # Initialize a new signing key
  specular vault init-key

  # Show signing key information
  specular vault key-info

  # Rotate the signing key
  specular vault rotate-key
`,
}

var vaultStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check Vault connection status",
	Long:  `Check the connection status and health of the configured Vault server.`,
	RunE:  runVaultStatus,
}

var vaultInitKeyCmd = &cobra.Command{
	Use:   "init-key",
	Short: "Initialize a new signing key in Vault",
	Long: `Initialize a new ECDSA P-256 signing key in Vault for audit log signatures.

This creates a new key at the configured signing_key_path if one doesn't exist.
The key is stored securely in Vault's KV v2 secrets engine.`,
	RunE: runVaultInitKey,
}

var vaultKeyInfoCmd = &cobra.Command{
	Use:   "key-info",
	Short: "Show signing key information",
	Long:  `Display information about the current signing key stored in Vault.`,
	RunE:  runVaultKeyInfo,
}

var vaultRotateKeyCmd = &cobra.Command{
	Use:   "rotate-key",
	Short: "Rotate the signing key",
	Long: `Generate a new signing key and store it as a new version in Vault.

The old key version remains available for verifying old signatures,
but new signatures will use the new key.`,
	RunE: runVaultRotateKey,
}

func init() {
	vaultCmd.AddCommand(vaultStatusCmd)
	vaultCmd.AddCommand(vaultInitKeyCmd)
	vaultCmd.AddCommand(vaultKeyInfoCmd)
	vaultCmd.AddCommand(vaultRotateKeyCmd)

	rootCmd.AddCommand(vaultCmd)
}

// createVaultClient creates a Vault client from configuration
func createVaultClient(cfg *GlobalConfig) (*vault.Client, error) {
	if !cfg.Vault.Enabled {
		return nil, fmt.Errorf("vault integration is not enabled (set vault.enabled=true)")
	}

	if cfg.Vault.Address == "" {
		return nil, fmt.Errorf("vault address is not configured (set vault.address)")
	}

	var tlsCfg *vault.TLSConfig
	if cfg.Vault.TLS.CACert != "" || cfg.Vault.TLS.ClientCert != "" {
		tlsCfg = &vault.TLSConfig{
			CACert:             cfg.Vault.TLS.CACert,
			CAPath:             cfg.Vault.TLS.CAPath,
			ClientCert:         cfg.Vault.TLS.ClientCert,
			ClientKey:          cfg.Vault.TLS.ClientKey,
			TLSServerName:      cfg.Vault.TLS.ServerName,
			InsecureSkipVerify: cfg.Vault.TLS.InsecureSkipVerify,
		}
	}

	client, err := vault.NewClient(vault.Config{
		Address:   cfg.Vault.Address,
		MountPath: cfg.Vault.MountPath,
		Namespace: cfg.Vault.Namespace,
		TLSConfig: tlsCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	return client, nil
}

func runVaultStatus(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	// Check if Vault is enabled
	if !config.Vault.Enabled {
		fmt.Println("Vault integration: disabled")
		fmt.Println("\nTo enable Vault integration:")
		fmt.Println("  specular config set vault.enabled true")
		fmt.Println("  specular config set vault.address https://vault.example.com:8200")
		return nil
	}

	client, err := createVaultClient(config)
	if err != nil {
		return ux.FormatError(err, "creating Vault client")
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check Vault health
	start := time.Now()
	healthErr := client.Health(ctx)
	latency := time.Since(start)

	// Output status
	status := struct {
		Enabled   bool   `json:"enabled" yaml:"enabled"`
		Address   string `json:"address" yaml:"address"`
		MountPath string `json:"mount_path" yaml:"mount_path"`
		Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
		Healthy   bool   `json:"healthy" yaml:"healthy"`
		Latency   string `json:"latency" yaml:"latency"`
		Error     string `json:"error,omitempty" yaml:"error,omitempty"`
	}{
		Enabled:   config.Vault.Enabled,
		Address:   config.Vault.Address,
		MountPath: config.Vault.MountPath,
		Namespace: config.Vault.Namespace,
		Healthy:   healthErr == nil,
		Latency:   latency.String(),
	}

	if healthErr != nil {
		status.Error = healthErr.Error()
	}

	// Use formatter for JSON/YAML output
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, formatErr := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if formatErr != nil {
			return formatErr
		}
		return formatter.Format(status)
	}

	// Text output
	fmt.Println("Vault Status")
	fmt.Println("============")
	fmt.Printf("Enabled:    %t\n", status.Enabled)
	fmt.Printf("Address:    %s\n", status.Address)
	fmt.Printf("Mount Path: %s\n", status.MountPath)
	if status.Namespace != "" {
		fmt.Printf("Namespace:  %s\n", status.Namespace)
	}
	fmt.Printf("Latency:    %s\n", status.Latency)

	if status.Healthy {
		fmt.Println("\n✓ Vault server is healthy")
	} else {
		fmt.Printf("\n✗ Vault server is unhealthy: %s\n", status.Error)
		return fmt.Errorf("vault health check failed")
	}

	return nil
}

func runVaultInitKey(cmd *cobra.Command, args []string) error {
	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	client, err := createVaultClient(config)
	if err != nil {
		return ux.FormatError(err, "creating Vault client")
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if key already exists
	keyPath := config.Vault.SigningKeyPath
	if keyPath == "" {
		keyPath = "specular/audit/signing-key"
	}

	identity := config.Vault.SignerIdentity
	if identity == "" {
		hostname, _ := os.Hostname()
		identity = fmt.Sprintf("specular@%s", hostname)
	}

	// Try to create signer with AutoGenerate enabled
	signer, err := client.NewSigner(ctx, vault.SignerConfig{
		KeyPath:      keyPath,
		Identity:     identity,
		AutoGenerate: true,
	})
	if err != nil {
		return ux.FormatError(err, "initializing signing key")
	}

	// Get key info to confirm
	keyInfo, err := signer.GetKeyInfo(ctx)
	if err != nil {
		return ux.FormatError(err, "reading key info")
	}

	fmt.Println("✓ Signing key initialized successfully")
	fmt.Println()
	fmt.Printf("  Path:      %s\n", keyPath)
	fmt.Printf("  Identity:  %s\n", identity)
	fmt.Printf("  Algorithm: %s\n", keyInfo.Algorithm)
	fmt.Printf("  Version:   %d\n", keyInfo.Version)
	fmt.Printf("  Created:   %s\n", keyInfo.CreatedAt)

	return nil
}

func runVaultKeyInfo(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	client, err := createVaultClient(config)
	if err != nil {
		return ux.FormatError(err, "creating Vault client")
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	keyPath := config.Vault.SigningKeyPath
	if keyPath == "" {
		keyPath = "specular/audit/signing-key"
	}

	identity := config.Vault.SignerIdentity
	if identity == "" {
		hostname, _ := os.Hostname()
		identity = fmt.Sprintf("specular@%s", hostname)
	}

	// Create signer without auto-generate to check if key exists
	signer, err := client.NewSigner(ctx, vault.SignerConfig{
		KeyPath:      keyPath,
		Identity:     identity,
		AutoGenerate: false,
	})
	if err != nil {
		return ux.FormatError(err, "accessing signing key")
	}

	keyInfo, err := signer.GetKeyInfo(ctx)
	if err != nil {
		return ux.FormatError(err, "reading key info")
	}

	// Get public key for display
	pubKey, err := signer.GetPublicKey(ctx)
	if err != nil {
		return ux.FormatError(err, "reading public key")
	}

	info := struct {
		Path         string `json:"path" yaml:"path"`
		Identity     string `json:"identity" yaml:"identity"`
		Algorithm    string `json:"algorithm" yaml:"algorithm"`
		Version      int    `json:"version" yaml:"version"`
		CreatedAt    string `json:"created_at" yaml:"created_at"`
		PublicKeyLen int    `json:"public_key_length" yaml:"public_key_length"`
	}{
		Path:         keyPath,
		Identity:     keyInfo.Identity,
		Algorithm:    keyInfo.Algorithm,
		Version:      keyInfo.Version,
		CreatedAt:    keyInfo.CreatedAt,
		PublicKeyLen: len(pubKey),
	}

	// Use formatter for JSON/YAML output
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, formatErr := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if formatErr != nil {
			return formatErr
		}
		return formatter.Format(info)
	}

	// Text output
	fmt.Println("Signing Key Information")
	fmt.Println("=======================")
	fmt.Printf("Path:             %s\n", info.Path)
	fmt.Printf("Identity:         %s\n", info.Identity)
	fmt.Printf("Algorithm:        %s\n", info.Algorithm)
	fmt.Printf("Version:          %d\n", info.Version)
	fmt.Printf("Created:          %s\n", info.CreatedAt)
	fmt.Printf("Public Key Size:  %d bytes\n", info.PublicKeyLen)

	return nil
}

func runVaultRotateKey(cmd *cobra.Command, args []string) error {
	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	client, err := createVaultClient(config)
	if err != nil {
		return ux.FormatError(err, "creating Vault client")
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keyPath := config.Vault.SigningKeyPath
	if keyPath == "" {
		keyPath = "specular/audit/signing-key"
	}

	identity := config.Vault.SignerIdentity
	if identity == "" {
		hostname, _ := os.Hostname()
		identity = fmt.Sprintf("specular@%s", hostname)
	}

	// Create signer without auto-generate
	signer, err := client.NewSigner(ctx, vault.SignerConfig{
		KeyPath:      keyPath,
		Identity:     identity,
		AutoGenerate: false,
	})
	if err != nil {
		return ux.FormatError(err, "accessing signing key")
	}

	// Get current version before rotation
	oldInfo, err := signer.GetKeyInfo(ctx)
	if err != nil {
		return ux.FormatError(err, "reading current key info")
	}

	fmt.Printf("Rotating signing key (current version: %d)...\n", oldInfo.Version)

	// Rotate the key
	if err := signer.RotateKey(ctx); err != nil {
		return ux.FormatError(err, "rotating key")
	}

	// Get new key info
	newInfo, err := signer.GetKeyInfo(ctx)
	if err != nil {
		return ux.FormatError(err, "reading new key info")
	}

	fmt.Println()
	fmt.Println("✓ Signing key rotated successfully")
	fmt.Println()
	fmt.Printf("  Previous Version: %d\n", oldInfo.Version)
	fmt.Printf("  New Version:      %d\n", newInfo.Version)
	fmt.Printf("  Algorithm:        %s\n", newInfo.Algorithm)
	fmt.Printf("  Created:          %s\n", newInfo.CreatedAt)
	fmt.Println()
	fmt.Println("Note: Old key versions remain available for verifying historical signatures.")

	return nil
}
