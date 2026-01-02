# ADR 0016: AWS Secrets Manager Integration for Cloud-Native Secrets Management

## Status

Accepted

## Context

Following ADR-0015 (HashiCorp Vault Integration), enterprises deploying on AWS require a native secrets management option. While Vault provides excellent cross-cloud capabilities, organizations with AWS-centric infrastructure benefit from:

1. **Native AWS Integration**: IAM-based authentication, VPC endpoints, cross-account access
2. **Managed Service**: No infrastructure to operate, automatic patching and scaling
3. **Cost Optimization**: Pay-per-secret pricing vs. Vault infrastructure costs
4. **Compliance**: AWS manages underlying security, simplifying compliance audits
5. **Multi-Region DR**: Built-in cross-region replication for disaster recovery

**Use Cases**:
- Organizations already invested in AWS ecosystem
- Serverless deployments (Lambda, ECS, EKS)
- Multi-account environments with cross-account secret sharing
- Teams without dedicated infrastructure operations capacity

## Decision

We will implement AWS Secrets Manager as an alternative secrets backend alongside HashiCorp Vault, providing:

1. **AWS SM Client Wrapper** (`internal/awssm/client.go`)
   - AWS credential chain support (env, config, IAM roles, IRSA)
   - Multi-region support for DR
   - Configurable endpoint for LocalStack testing

2. **ECDSA Key Management** (`internal/awssm/signer.go`)
   - AWS SM-backed implementation of `authz.Signer` interface
   - Same ECDSA P-256 + PKCS#8/PKIX format as VaultSigner
   - Key caching for performance
   - Key rotation via version staging (AWSCURRENT, AWSPREVIOUS)

3. **Secrets Operations** (`internal/awssm/secrets.go`)
   - Put/Get secret operations
   - Version stage access (AWSCURRENT, AWSPREVIOUS, AWSPENDING)
   - List and describe capabilities

## Architecture

### AWS SM Client Architecture

```go
// Client wraps AWS Secrets Manager API
type Client struct {
    smClient        *secretsmanager.Client
    region          string
    endpoint        string  // For LocalStack testing

    // DR support
    secondaryClient *secretsmanager.Client
    secondaryRegion string
}

// Configuration
type Config struct {
    Region          string  // Primary AWS region (required)
    SecondaryRegion string  // DR/failover region (optional)
    Profile         string  // AWS profile name
    AssumeRoleARN   string  // Cross-account access
    Endpoint        string  // LocalStack endpoint
    AccessKeyID     string  // Static credentials (not recommended)
    SecretAccessKey string  // Static credentials (not recommended)
}
```

### AWS SM-Backed Signer

```go
// AWSSecretsManagerSigner implements authz.Signer interface
type AWSSecretsManagerSigner struct {
    client     *Client
    secretName string
    identity   string

    // Performance optimization
    cachedKey    *ecdsa.PrivateKey
    cachedPubKey []byte
    cacheExpiry  time.Time
    cacheTTL     time.Duration  // default: 5 minutes
}

// Integration with signed audit logging
signedLogger := authz.NewSignedAuditLogger(
    baseLogger,
    awssmSigner,  // implements authz.Signer
)
```

### Key Storage Format

Keys are stored in AWS Secrets Manager with the same format as Vault for interoperability:

```json
{
    "private_key": "<base64-encoded PKCS#8>",
    "public_key": "<base64-encoded PKIX>",
    "algorithm": "ECDSA-P256",
    "created_at": "2024-01-01T00:00:00Z",
    "identity": "system@specular.dev"
}
```

## Implementation Details

### 1. AWS Credential Chain

The client uses AWS SDK v2's default credential chain:
1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
2. Shared credentials file (~/.aws/credentials)
3. Shared config file (~/.aws/config)
4. IAM role for EC2/ECS/EKS
5. IRSA (IAM Roles for Service Accounts) on EKS

### 2. Multi-Region DR

```go
client, err := awssm.NewClient(ctx, awssm.Config{
    Region:          "us-west-2",
    SecondaryRegion: "us-east-1",  // Automatic failover
})

// Primary and secondary clients available
client.SecretsManager()           // Primary
client.SecondarySecretsManager()  // Secondary
```

### 3. Version Staging

AWS Secrets Manager uses version stages instead of numeric versions:
- `AWSCURRENT`: Active version for new operations
- `AWSPREVIOUS`: Previous version (after rotation)
- `AWSPENDING`: Version being staged during rotation

```go
// Get current version
secret, err := secrets.Get(ctx, "my-key")

// Get previous version for signature verification
secret, err := secrets.GetByStage(ctx, "my-key", "AWSPREVIOUS")
```

### 4. Key Rotation

```go
// Rotate ECDSA signing key
err := signer.RotateKey(ctx, nil)

// Creates new version with AWSCURRENT stage
// Previous version moves to AWSPREVIOUS stage
// Old signatures remain verifiable via AWSPREVIOUS
```

## Usage Examples

### Basic Setup

```go
// Create AWS SM client
client, err := awssm.NewClient(ctx, awssm.Config{
    Region: "us-west-2",
})

// Create AWS SM-backed signer
signer, err := client.NewSigner(ctx, awssm.SignerConfig{
    SecretName:   "specular/audit/signing-key",
    Identity:     "system@specular.dev",
    AutoGenerate: true,
    CacheTTL:     5 * time.Minute,
})

// Use with signed audit logging
signedLogger := authz.NewSignedAuditLogger(
    baseLogger,
    signer,
)
```

### EKS with IRSA

```go
// IRSA automatically provides credentials via environment
client, err := awssm.NewClient(ctx, awssm.Config{
    Region: "us-west-2",
})
// No explicit credentials needed - uses IRSA token
```

### Cross-Account Access

```go
client, err := awssm.NewClient(ctx, awssm.Config{
    Region:        "us-west-2",
    AssumeRoleARN: "arn:aws:iam::123456789012:role/specular-secrets-access",
})
```

### LocalStack Testing

```go
client, err := awssm.NewClient(ctx, awssm.Config{
    Region:          "us-west-2",
    Endpoint:        "http://localhost:4566",
    AccessKeyID:     "test",
    SecretAccessKey: "test",
})
```

## Configuration

### CLI Configuration

```yaml
# ~/.specular/config.yaml
aws_secrets_manager:
  enabled: true
  region: us-west-2
  secondary_region: us-east-1  # For DR
  profile: production          # AWS profile
  signing_key_name: specular/audit/signing-key
  signer_identity: system@specular.dev
  auto_generate_key: true
  tags:
    environment: production
    application: specular
```

### Environment Variables

AWS standard:
- `AWS_REGION`: AWS region
- `AWS_PROFILE`: AWS profile name
- `AWS_ENDPOINT_URL`: Custom endpoint (LocalStack)

Specular-specific:
- `SPECULAR_AWS_SM_ENABLED`: Enable AWS SM integration
- `SPECULAR_AWS_SM_REGION`: Primary region
- `SPECULAR_AWS_SM_SECONDARY_REGION`: DR region
- `SPECULAR_AWS_SM_SIGNING_KEY_NAME`: Secret name for signing key
- `SPECULAR_AWS_SM_SIGNER_IDENTITY`: Signer identity

## Performance Characteristics

| Operation | Latency (p50) | Latency (p99) | Notes |
|-----------|--------------|---------------|-------|
| **Sign (cached)** | ~3ms | ~5ms | Using cached key |
| **Sign (uncached)** | ~80ms | ~200ms | AWS API call |
| **Get Secret** | ~50ms | ~150ms | Network + AWS processing |
| **Put Secret** | ~60ms | ~180ms | Write operation |
| **List Secrets** | ~40ms | ~100ms | Paginated results |

**Optimization**: 5-minute cache TTL provides 99% cache hit rate in normal operation.

## Security Considerations

### IAM Policy

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue",
                "secretsmanager:DescribeSecret"
            ],
            "Resource": "arn:aws:secretsmanager:*:*:secret:specular/*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:ListSecrets"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:CreateSecret",
                "secretsmanager:PutSecretValue"
            ],
            "Resource": "arn:aws:secretsmanager:*:*:secret:specular/audit/*"
        }
    ]
}
```

### VPC Endpoint

For private network access without internet traversal:

```hcl
resource "aws_vpc_endpoint" "secretsmanager" {
  vpc_id              = aws_vpc.main.id
  service_name        = "com.amazonaws.${var.region}.secretsmanager"
  vpc_endpoint_type   = "Interface"
  security_group_ids  = [aws_security_group.secrets.id]
  subnet_ids          = aws_subnet.private[*].id
  private_dns_enabled = true
}
```

### Encryption

- **At Rest**: AWS KMS encryption (customer-managed or AWS-managed keys)
- **In Transit**: TLS 1.2+ to AWS endpoints
- **Key Storage**: ECDSA private keys stored in encrypted secret value

## Monitoring & Observability

### CloudWatch Metrics

AWS provides built-in metrics:
- `CallCount`: Number of API calls
- `ThrottleCount`: Throttled requests
- `ResourceNotFoundExceptions`: Missing secrets

### Health Checks

```go
// AWS SM health check
err := client.Health(ctx)

// Signer health check
info, err := signer.GetKeyInfo(ctx)
```

### CloudTrail Logging

All Secrets Manager API calls are logged in CloudTrail:
- `GetSecretValue`: Secret access
- `PutSecretValue`: Secret updates
- `CreateSecret`: New secret creation
- `DeleteSecret`: Secret deletion

## Comparison with Vault

| Feature | HashiCorp Vault | AWS Secrets Manager |
|---------|-----------------|---------------------|
| **Deployment** | Self-managed | Managed service |
| **Auth** | Token, mTLS, OIDC | IAM, IRSA |
| **Versioning** | Numeric (v1, v2) | Staging (AWSCURRENT, AWSPREVIOUS) |
| **Multi-Region** | Replication clusters | Native cross-region |
| **Cost** | Infrastructure + licensing | Pay per secret |
| **Lock-in** | None | AWS |
| **Enterprise Features** | Vault Enterprise | Included |

## Testing

### Test Coverage

- **Client Tests**: Configuration, health checks, multi-region
- **Secrets Tests**: CRUD operations, versioning, error handling
- **Signer Tests**: Sign/verify, key format compatibility, caching
- **Health Checker Tests**: Nil handling, interface compliance

### LocalStack Testing

```bash
# Start LocalStack
docker run -d -p 4566:4566 localstack/localstack

# Run tests
SPECULAR_AWS_SM_ENDPOINT=http://localhost:4566 go test ./internal/awssm/...
```

## CLI Commands

```bash
# Check AWS SM connection status
specular awssm status

# Initialize a new signing key
specular awssm init-key

# Show signing key information
specular awssm key-info

# Rotate the signing key
specular awssm rotate-key
```

## Rollout Plan

### Phase 1: Core Implementation ✅
- Implement AWS SM client with credential chain
- Add AWSSecretsManagerSigner with authz.Signer interface
- Create health checkers
- CLI commands (status, init-key, key-info, rotate-key)

### Phase 2: Production Hardening
- Multi-region failover testing
- Rate limiting and retry logic
- Enhanced error messages
- Production deployment guides

### Phase 3: Advanced Features
- Automatic rotation via Lambda
- Cross-account secret sharing
- KMS customer-managed keys
- Secrets caching layer integration

## References

- [AWS Secrets Manager Documentation](https://docs.aws.amazon.com/secretsmanager/)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
- [ADR-0014: Signed Audit Logging](0014-signed-audit-logging.md)
- [ADR-0015: HashiCorp Vault Integration](0015-hashicorp-vault-integration.md)

## Decision Makers

- @felixgeelhaar (Architecture, Implementation)

## Date

2026-01-02
