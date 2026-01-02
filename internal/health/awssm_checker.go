package health

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/specular/internal/awssm"
)

// AWSSecretsManagerChecker checks if AWS Secrets Manager is accessible and healthy.
type AWSSecretsManagerChecker struct {
	client *awssm.Client
}

// NewAWSSecretsManagerChecker creates a new AWS Secrets Manager health checker.
// If client is nil, the checker will return unhealthy status.
func NewAWSSecretsManagerChecker(client *awssm.Client) *AWSSecretsManagerChecker {
	return &AWSSecretsManagerChecker{
		client: client,
	}
}

// Name returns the name of this health check.
func (c *AWSSecretsManagerChecker) Name() string {
	return "aws-secrets-manager"
}

// Check verifies AWS Secrets Manager is accessible and healthy.
// Returns:
//   - Healthy if AWS Secrets Manager is reachable and responding
//   - Unhealthy if AWS Secrets Manager is unreachable or credentials are invalid
func (c *AWSSecretsManagerChecker) Check(ctx context.Context) *Result {
	if c.client == nil {
		return Unhealthy("AWS Secrets Manager client not configured").
			WithDetail("suggestion", "Configure AWS SM via 'specular config set aws_secrets_manager.enabled true'")
	}

	start := time.Now()
	err := c.client.Health(ctx)
	latency := time.Since(start)

	if err != nil {
		return Unhealthy(fmt.Sprintf("AWS Secrets Manager health check failed: %v", err)).
			WithDetail("region", c.client.Region()).
			WithDetail("error", err.Error()).
			WithLatency(latency)
	}

	result := Healthy("AWS Secrets Manager is healthy").
		WithDetail("region", c.client.Region()).
		WithLatency(latency)

	if c.client.HasSecondary() {
		result = result.WithDetail("secondary_region", c.client.SecondaryRegion())
	}

	if c.client.Endpoint() != "" {
		result = result.WithDetail("endpoint", c.client.Endpoint())
	}

	return result
}

// AWSSecretsManagerSignerChecker checks if the AWS Secrets Manager-backed signer is operational.
type AWSSecretsManagerSignerChecker struct {
	signer *awssm.AWSSecretsManagerSigner
}

// NewAWSSecretsManagerSignerChecker creates a new AWS Secrets Manager signer health checker.
// If signer is nil, the checker will return unhealthy status.
func NewAWSSecretsManagerSignerChecker(signer *awssm.AWSSecretsManagerSigner) *AWSSecretsManagerSignerChecker {
	return &AWSSecretsManagerSignerChecker{
		signer: signer,
	}
}

// Name returns the name of this health check.
func (c *AWSSecretsManagerSignerChecker) Name() string {
	return "aws-secrets-manager-signer"
}

// Check verifies the AWS Secrets Manager-backed signer can retrieve keys and sign data.
// Returns:
//   - Healthy if signing key is accessible and operational
//   - Unhealthy if signing key is missing or signing fails
func (c *AWSSecretsManagerSignerChecker) Check(ctx context.Context) *Result {
	if c.signer == nil {
		return Unhealthy("AWS Secrets Manager signer not configured").
			WithDetail("suggestion", "Configure AWS SM signing key via 'specular awssm init-key'")
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

	// Verify we can get the public key (cached or from AWS SM)
	pubKey, err := c.signer.GetPublicKey(ctx)
	if err != nil {
		return Unhealthy(fmt.Sprintf("Failed to retrieve public key: %v", err)).
			WithDetail("error", err.Error()).
			WithDetail("identity", c.signer.Identity()).
			WithLatency(time.Since(start))
	}

	latency := time.Since(start)

	result := Healthy("AWS Secrets Manager signer is operational").
		WithDetail("identity", c.signer.Identity()).
		WithDetail("algorithm", keyInfo.Algorithm).
		WithDetail("secret_name", keyInfo.SecretName).
		WithDetail("public_key_length", len(pubKey)).
		WithLatency(latency)

	if keyInfo.VersionID != "" {
		result = result.WithDetail("version_id", keyInfo.VersionID)
	}

	if keyInfo.VersionStage != "" {
		result = result.WithDetail("version_stage", keyInfo.VersionStage)
	}

	return result
}
