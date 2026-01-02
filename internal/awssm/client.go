package awssm

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Client wraps AWS Secrets Manager API client with opinionated configuration.
//
// This client provides:
// - AWS credential chain support (env, config, IAM roles, IRSA)
// - Multi-region support for DR
// - Configurable endpoint for LocalStack testing
// - Health check method
type Client struct {
	smClient *secretsmanager.Client
	region   string
	endpoint string

	// For DR scenarios
	secondaryClient *secretsmanager.Client
	secondaryRegion string
}

// Config holds AWS Secrets Manager client configuration.
type Config struct {
	// Region is the primary AWS region (required)
	// Example: "us-west-2"
	Region string

	// SecondaryRegion is the DR/failover region (optional)
	SecondaryRegion string

	// Profile is the AWS profile name from ~/.aws/credentials (optional)
	// If not set, uses default credential chain
	Profile string

	// AssumeRoleARN is the ARN of a role to assume (optional)
	// Useful for cross-account access
	AssumeRoleARN string

	// Endpoint is a custom endpoint URL (optional)
	// Useful for LocalStack testing: "http://localhost:4566"
	Endpoint string

	// AccessKeyID and SecretAccessKey for static credentials (optional)
	// Not recommended for production - use IAM roles instead
	AccessKeyID     string
	SecretAccessKey string
}

// NewClient creates a new AWS Secrets Manager client with the provided configuration.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	// Validate required fields
	if cfg.Region == "" {
		return nil, fmt.Errorf("aws region is required")
	}

	// Build primary client
	smClient, err := createSecretsManagerClient(ctx, cfg, cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to create secrets manager client: %w", err)
	}

	client := &Client{
		smClient: smClient,
		region:   cfg.Region,
		endpoint: cfg.Endpoint,
	}

	// Build secondary client for DR if configured
	if cfg.SecondaryRegion != "" {
		secondaryClient, err := createSecretsManagerClient(ctx, cfg, cfg.SecondaryRegion)
		if err != nil {
			return nil, fmt.Errorf("failed to create secondary secrets manager client: %w", err)
		}
		client.secondaryClient = secondaryClient
		client.secondaryRegion = cfg.SecondaryRegion
	}

	return client, nil
}

// createSecretsManagerClient creates a Secrets Manager client for a specific region.
func createSecretsManagerClient(ctx context.Context, cfg Config, region string) (*secretsmanager.Client, error) {
	// Build config options
	var optFns []func(*config.LoadOptions) error

	// Set region
	optFns = append(optFns, config.WithRegion(region))

	// Set profile if specified
	if cfg.Profile != "" {
		optFns = append(optFns, config.WithSharedConfigProfile(cfg.Profile))
	}

	// Set static credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		optFns = append(optFns, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Handle role assumption if specified
	if cfg.AssumeRoleARN != "" {
		stsClient := sts.NewFromConfig(awsCfg)
		creds := stscreds.NewAssumeRoleProvider(stsClient, cfg.AssumeRoleARN)
		awsCfg.Credentials = aws.NewCredentialsCache(creds)
	}

	// Build Secrets Manager client options
	var smOpts []func(*secretsmanager.Options)

	// Set custom endpoint (for LocalStack)
	if cfg.Endpoint != "" {
		smOpts = append(smOpts, func(o *secretsmanager.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	return secretsmanager.NewFromConfig(awsCfg, smOpts...), nil
}

// Health checks AWS Secrets Manager connectivity.
func (c *Client) Health(ctx context.Context) error {
	// Use ListSecrets with MaxResults=1 as a health check
	// This is a lightweight call that validates credentials and connectivity
	input := &secretsmanager.ListSecretsInput{
		MaxResults: aws.Int32(1),
	}

	_, err := c.smClient.ListSecrets(ctx, input)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Region returns the primary AWS region.
func (c *Client) Region() string {
	return c.region
}

// SecondaryRegion returns the secondary/DR AWS region.
func (c *Client) SecondaryRegion() string {
	return c.secondaryRegion
}

// Endpoint returns the custom endpoint (if configured).
func (c *Client) Endpoint() string {
	return c.endpoint
}

// HasSecondary returns true if a secondary region is configured.
func (c *Client) HasSecondary() bool {
	return c.secondaryClient != nil
}

// Close closes the client (no-op for AWS SDK, but follows pattern).
func (c *Client) Close() error {
	// AWS SDK clients don't require explicit cleanup
	return nil
}

// SecretsManager returns the underlying Secrets Manager client.
// This is useful for advanced operations not covered by the wrapper.
func (c *Client) SecretsManager() *secretsmanager.Client {
	return c.smClient
}

// SecondarySecretsManager returns the secondary Secrets Manager client.
// Returns nil if no secondary region is configured.
func (c *Client) SecondarySecretsManager() *secretsmanager.Client {
	return c.secondaryClient
}
