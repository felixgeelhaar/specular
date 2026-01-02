package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/awssm"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var awssmCmd = &cobra.Command{
	Use:   "awssm",
	Short: "Manage AWS Secrets Manager integration",
	Long: `Manage AWS Secrets Manager integration for secrets management and audit log signing.

AWS Secrets Manager integration provides:
  • Secure storage for ECDSA signing keys
  • Cryptographic signatures for audit logs
  • Key rotation with version stages (AWSCURRENT, AWSPREVIOUS)
  • Multi-region support for DR
  • IAM-based authentication

Examples:
  # Check AWS Secrets Manager connection status
  specular awssm status

  # Initialize a new signing key
  specular awssm init-key

  # Show signing key information
  specular awssm key-info

  # Rotate the signing key
  specular awssm rotate-key
`,
}

var awssmStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check AWS Secrets Manager connection status",
	Long:  `Check the connection status and health of AWS Secrets Manager.`,
	RunE:  runAWSSecretsManagerStatus,
}

var awssmInitKeyCmd = &cobra.Command{
	Use:   "init-key",
	Short: "Initialize a new signing key in AWS Secrets Manager",
	Long: `Initialize a new ECDSA P-256 signing key in AWS Secrets Manager for audit log signatures.

This creates a new key at the configured signing_key_name if one doesn't exist.
The key is stored securely in AWS Secrets Manager.`,
	RunE: runAWSSecretsManagerInitKey,
}

var awssmKeyInfoCmd = &cobra.Command{
	Use:   "key-info",
	Short: "Show signing key information",
	Long:  `Display information about the current signing key stored in AWS Secrets Manager.`,
	RunE:  runAWSSecretsManagerKeyInfo,
}

var awssmRotateKeyCmd = &cobra.Command{
	Use:   "rotate-key",
	Short: "Rotate the signing key",
	Long: `Generate a new signing key and store it as a new version in AWS Secrets Manager.

The old key version remains available via AWSPREVIOUS stage for verifying old signatures,
but new signatures will use the new key (AWSCURRENT).`,
	RunE: runAWSSecretsManagerRotateKey,
}

func init() {
	awssmCmd.AddCommand(awssmStatusCmd)
	awssmCmd.AddCommand(awssmInitKeyCmd)
	awssmCmd.AddCommand(awssmKeyInfoCmd)
	awssmCmd.AddCommand(awssmRotateKeyCmd)

	rootCmd.AddCommand(awssmCmd)
}

// createAWSSecretsManagerClient creates an AWS Secrets Manager client from configuration
func createAWSSecretsManagerClient(ctx context.Context, cfg *GlobalConfig) (*awssm.Client, error) {
	if !cfg.AWSSecretsManager.Enabled {
		return nil, fmt.Errorf("AWS Secrets Manager integration is not enabled (set aws_secrets_manager.enabled=true)")
	}

	if cfg.AWSSecretsManager.Region == "" {
		return nil, fmt.Errorf("AWS region is not configured (set aws_secrets_manager.region)")
	}

	client, err := awssm.NewClient(ctx, awssm.Config{
		Region:          cfg.AWSSecretsManager.Region,
		SecondaryRegion: cfg.AWSSecretsManager.SecondaryRegion,
		Profile:         cfg.AWSSecretsManager.Profile,
		AssumeRoleARN:   cfg.AWSSecretsManager.AssumeRoleARN,
		Endpoint:        cfg.AWSSecretsManager.Endpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS Secrets Manager client: %w", err)
	}

	return client, nil
}

func runAWSSecretsManagerStatus(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	// Check if AWS SM is enabled
	if !config.AWSSecretsManager.Enabled {
		fmt.Println("AWS Secrets Manager integration: disabled")
		fmt.Println("\nTo enable AWS Secrets Manager integration:")
		fmt.Println("  specular config set aws_secrets_manager.enabled true")
		fmt.Println("  specular config set aws_secrets_manager.region us-west-2")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := createAWSSecretsManagerClient(ctx, config)
	if err != nil {
		return ux.FormatError(err, "creating AWS Secrets Manager client")
	}
	defer func() { _ = client.Close() }()

	// Check AWS SM health
	start := time.Now()
	healthErr := client.Health(ctx)
	latency := time.Since(start)

	// Output status
	status := struct {
		Enabled         bool   `json:"enabled" yaml:"enabled"`
		Region          string `json:"region" yaml:"region"`
		SecondaryRegion string `json:"secondary_region,omitempty" yaml:"secondary_region,omitempty"`
		Profile         string `json:"profile,omitempty" yaml:"profile,omitempty"`
		Endpoint        string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
		Healthy         bool   `json:"healthy" yaml:"healthy"`
		Latency         string `json:"latency" yaml:"latency"`
		Error           string `json:"error,omitempty" yaml:"error,omitempty"`
	}{
		Enabled:         config.AWSSecretsManager.Enabled,
		Region:          config.AWSSecretsManager.Region,
		SecondaryRegion: config.AWSSecretsManager.SecondaryRegion,
		Profile:         config.AWSSecretsManager.Profile,
		Endpoint:        config.AWSSecretsManager.Endpoint,
		Healthy:         healthErr == nil,
		Latency:         latency.String(),
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
	fmt.Println("AWS Secrets Manager Status")
	fmt.Println("==========================")
	fmt.Printf("Enabled:    %t\n", status.Enabled)
	fmt.Printf("Region:     %s\n", status.Region)
	if status.SecondaryRegion != "" {
		fmt.Printf("Secondary:  %s\n", status.SecondaryRegion)
	}
	if status.Profile != "" {
		fmt.Printf("Profile:    %s\n", status.Profile)
	}
	if status.Endpoint != "" {
		fmt.Printf("Endpoint:   %s\n", status.Endpoint)
	}
	fmt.Printf("Latency:    %s\n", status.Latency)

	if status.Healthy {
		fmt.Println("\n✓ AWS Secrets Manager is healthy")
	} else {
		fmt.Printf("\n✗ AWS Secrets Manager is unhealthy: %s\n", status.Error)
		return fmt.Errorf("AWS Secrets Manager health check failed")
	}

	return nil
}

func runAWSSecretsManagerInitKey(cmd *cobra.Command, args []string) error {
	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := createAWSSecretsManagerClient(ctx, config)
	if err != nil {
		return ux.FormatError(err, "creating AWS Secrets Manager client")
	}
	defer func() { _ = client.Close() }()

	// Get key name and identity
	keyName := config.AWSSecretsManager.SigningKeyName
	if keyName == "" {
		keyName = "specular/audit/signing-key"
	}

	identity := config.AWSSecretsManager.SignerIdentity
	if identity == "" {
		hostname, _ := os.Hostname()
		identity = fmt.Sprintf("specular@%s", hostname)
	}

	// Try to create signer with AutoGenerate enabled
	signer, err := client.NewSigner(ctx, awssm.SignerConfig{
		SecretName:   keyName,
		Identity:     identity,
		AutoGenerate: true,
		Tags:         config.AWSSecretsManager.Tags,
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
	fmt.Printf("  Secret Name:   %s\n", keyName)
	fmt.Printf("  Identity:      %s\n", identity)
	fmt.Printf("  Algorithm:     %s\n", keyInfo.Algorithm)
	if keyInfo.VersionID != "" {
		fmt.Printf("  Version ID:    %s\n", keyInfo.VersionID)
	}
	if keyInfo.VersionStage != "" {
		fmt.Printf("  Version Stage: %s\n", keyInfo.VersionStage)
	}
	fmt.Printf("  Created:       %s\n", keyInfo.CreatedAt)

	return nil
}

func runAWSSecretsManagerKeyInfo(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := createAWSSecretsManagerClient(ctx, config)
	if err != nil {
		return ux.FormatError(err, "creating AWS Secrets Manager client")
	}
	defer func() { _ = client.Close() }()

	keyName := config.AWSSecretsManager.SigningKeyName
	if keyName == "" {
		keyName = "specular/audit/signing-key"
	}

	identity := config.AWSSecretsManager.SignerIdentity
	if identity == "" {
		hostname, _ := os.Hostname()
		identity = fmt.Sprintf("specular@%s", hostname)
	}

	// Create signer without auto-generate to check if key exists
	signer, err := client.NewSigner(ctx, awssm.SignerConfig{
		SecretName:   keyName,
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
		SecretName   string `json:"secret_name" yaml:"secret_name"`
		Identity     string `json:"identity" yaml:"identity"`
		Algorithm    string `json:"algorithm" yaml:"algorithm"`
		VersionID    string `json:"version_id,omitempty" yaml:"version_id,omitempty"`
		VersionStage string `json:"version_stage,omitempty" yaml:"version_stage,omitempty"`
		CreatedAt    string `json:"created_at" yaml:"created_at"`
		PublicKeyLen int    `json:"public_key_length" yaml:"public_key_length"`
	}{
		SecretName:   keyName,
		Identity:     keyInfo.Identity,
		Algorithm:    keyInfo.Algorithm,
		VersionID:    keyInfo.VersionID,
		VersionStage: keyInfo.VersionStage,
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
	fmt.Printf("Secret Name:      %s\n", info.SecretName)
	fmt.Printf("Identity:         %s\n", info.Identity)
	fmt.Printf("Algorithm:        %s\n", info.Algorithm)
	if info.VersionID != "" {
		fmt.Printf("Version ID:       %s\n", info.VersionID)
	}
	if info.VersionStage != "" {
		fmt.Printf("Version Stage:    %s\n", info.VersionStage)
	}
	fmt.Printf("Created:          %s\n", info.CreatedAt)
	fmt.Printf("Public Key Size:  %d bytes\n", info.PublicKeyLen)

	return nil
}

func runAWSSecretsManagerRotateKey(cmd *cobra.Command, args []string) error {
	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := createAWSSecretsManagerClient(ctx, config)
	if err != nil {
		return ux.FormatError(err, "creating AWS Secrets Manager client")
	}
	defer func() { _ = client.Close() }()

	keyName := config.AWSSecretsManager.SigningKeyName
	if keyName == "" {
		keyName = "specular/audit/signing-key"
	}

	identity := config.AWSSecretsManager.SignerIdentity
	if identity == "" {
		hostname, _ := os.Hostname()
		identity = fmt.Sprintf("specular@%s", hostname)
	}

	// Create signer without auto-generate
	signer, err := client.NewSigner(ctx, awssm.SignerConfig{
		SecretName:   keyName,
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

	fmt.Printf("Rotating signing key")
	if oldInfo.VersionID != "" {
		fmt.Printf(" (current version: %s)", oldInfo.VersionID)
	}
	fmt.Println("...")

	// Rotate the key
	if err := signer.RotateKey(ctx, config.AWSSecretsManager.Tags); err != nil {
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
	if oldInfo.VersionID != "" {
		fmt.Printf("  Previous Version: %s\n", oldInfo.VersionID)
	}
	if newInfo.VersionID != "" {
		fmt.Printf("  New Version:      %s\n", newInfo.VersionID)
	}
	fmt.Printf("  Algorithm:        %s\n", newInfo.Algorithm)
	fmt.Printf("  Created:          %s\n", newInfo.CreatedAt)
	fmt.Println()
	fmt.Println("Note: Previous key version is available via AWSPREVIOUS stage for verifying historical signatures.")

	return nil
}
