# ADR 0016: API Key Rotation Mechanism

## Status

Accepted

## Context

Following ADR-0015 (HashiCorp Vault Integration), we need secure API authentication with automated key rotation for platform API access. Current limitations include:

1. **No API Authentication**: Platform lacks secure API key management
2. **Manual Key Management**: No automated rotation or lifecycle management
3. **Compliance Gaps**: Enterprise compliance requires key rotation policies
4. **Security Risk**: Long-lived credentials without rotation increase breach risk
5. **Scalability**: Need multi-organization API key management

**Compliance Requirements**:
- **SOC2 Type II**: Automated credential rotation and access controls
- **ISO 27001**: Cryptographic key lifecycle management
- **NIST 800-53**: Regular credential rotation (SI-4, IA-5)
- **PCI DSS**: Quarterly key rotation for API credentials

## Decision

We will implement an enterprise-grade API key rotation system with HashiCorp Vault storage, providing:

1. **API Key Manager** (`internal/apikey/manager.go`)
   - Cryptographically secure key generation (256-bit)
   - Vault-backed persistent storage with KV v2
   - Key lifecycle management (active, rotated, revoked, expired)
   - Scope-based authorization

2. **HTTP Middleware** (`internal/apikey/middleware.go`)
   - Bearer token authentication
   - Scope validation (all scopes / any scope)
   - Context-based key propagation
   - Standardized error responses

3. **Automatic Rotation Scheduler** (`internal/apikey/scheduler.go`)
   - Configurable rotation intervals
   - Grace period support for smooth transitions
   - Rotation status monitoring
   - Expired key cleanup

## Architecture

### API Key Structure

```go
type APIKey struct {
    ID             string    // Unique key identifier
    Secret         string    // sk_<base64-encoded-256-bit>
    OrganizationID string    // Multi-tenant support
    UserID         string    // Optional user association
    Name           string    // Human-readable name
    Prefix         string    // "sk_" (similar to Stripe)
    Scopes         []string  // Permission scopes
    Status         Status    // active|rotated|revoked|expired
    CreatedAt      time.Time
    LastUsedAt     time.Time
    ExpiresAt      time.Time
    RotatedAt      time.Time
    RevokedAt      time.Time
}
```

### Vault Storage Schema

API keys are stored in Vault KV v2 at `apikeys/{orgID}/{keyID}`:

```json
{
  "data": {
    "id": "key-abc123",
    "secret": "sk_dGVzdC1zZWNyZXQtZXhhbXBsZQ",
    "organization_id": "org-123",
    "user_id": "user-456",
    "name": "Production API Key",
    "prefix": "sk_",
    "scopes": ["read", "write", "admin"],
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "expires_at": "2024-04-01T00:00:00Z"
  },
  "metadata": {
    "organization_id": "org-123",
    "user_id": "user-456",
    "name": "Production API Key",
    "status": "active"
  }
}
```

### Key Rotation Flow

```
┌─────────────┐
│ Active Key  │ ← Used for authentication
└──────┬──────┘
       │ 7 days before expiry
       ▼
┌─────────────┐
│   Rotate    │ ← Scheduler triggers rotation
└──────┬──────┘
       │
       ├──► New Key (active)     ← New secret generated
       │
       └──► Old Key (rotated)    ← Grace period (7 days)
              │
              │ After grace period
              ▼
         ┌─────────┐
         │ Revoke  │
         └─────────┘
```

### HTTP Authentication Flow

```
Client Request
   │
   ├─► Extract Bearer token from Authorization header
   │
   ├─► Validate organization ID (X-Organization-ID header)
   │
   ├─► Retrieve key from Vault by secret
   │
   ├─► Validate key status (must be active)
   │
   ├─► Check expiration
   │
   ├─► Validate scopes (if required)
   │
   └─► Store key in context → Continue to handler
```

## Implementation Details

### Manager Operations

```go
// Create new API key
key, err := manager.CreateKey(ctx, orgID, userID, "Production Key", []string{"read", "write"})

// Validate and retrieve by secret
key, err := manager.GetKeyBySecret(ctx, orgID, secret)

// Rotate with grace period
newKey, err := manager.RotateKey(ctx, orgID, keyID, 7*24*time.Hour)

// Revoke immediately
err := manager.RevokeKey(ctx, orgID, keyID)

// Permanent deletion
err := manager.DeleteKey(ctx, orgID, keyID)
```

### Middleware Usage

```go
// Basic authentication
middleware := apikey.NewMiddleware(manager)
http.Handle("/api/resource", middleware.RequireAPIKey(handler))

// Require specific scopes (all must be present)
http.Handle("/api/admin",
    middleware.RequireAPIKey(
        middleware.RequireScopes([]string{"admin", "write"}, handler)))

// Require any of the scopes
http.Handle("/api/resource",
    middleware.RequireAPIKey(
        middleware.RequireAnyScope([]string{"admin", "write"}, handler)))

// Access key in handler
func handler(w http.ResponseWriter, r *http.Request) {
    key := apikey.GetAPIKeyFromContext(r.Context())
    // Use key.OrganizationID, key.UserID, etc.
}
```

### Scheduler Configuration

```go
scheduler, err := apikey.NewScheduler(apikey.SchedulerConfig{
    Manager:       manager,
    CheckInterval: 1 * time.Hour,      // Check every hour
    GracePeriod:   7 * 24 * time.Hour, // 7 days grace period
    RotationTTL:   7 * 24 * time.Hour, // Rotate 7 days before expiry
})

// Start background scheduler
go scheduler.Start(ctx)

// Manual rotation
count, err := scheduler.RotateAllKeys(ctx, orgID)

// Force rotation (ignores expiry)
count, err := scheduler.ForceRotateAllKeys(ctx, orgID)

// Get rotation status
status, err := scheduler.GetRotationStatus(ctx, orgID)
// status.NeedingRotation, status.DaysUntilRotation, etc.

// Cleanup old keys
cleaned, err := scheduler.CleanupExpiredKeys(ctx, orgID, 30*24*time.Hour)
```

## Security Considerations

### Cryptographic Security

- **Key Generation**: 256-bit (32 bytes) cryptographically secure random keys using `crypto/rand`
- **Key Format**: Base64 URL-safe encoding with "sk_" prefix (similar to Stripe)
- **Storage**: Keys encrypted at rest in Vault with TLS in transit
- **Rotation**: Automatic rotation reduces long-term credential exposure

### Access Control

- **Organization Isolation**: Keys scoped to organization IDs
- **Scope-Based Authorization**: Fine-grained permission control
- **Bearer Token**: Industry-standard HTTP authentication
- **Secret Clearing**: Secrets cleared on revocation

### Rotation Security

- **Grace Period**: Smooth transitions without service disruption
- **Rotation Window**: Configurable advance rotation (default: 7 days before expiry)
- **Status Tracking**: Clear lifecycle states prevent key reuse
- **Last Used Tracking**: Monitor key usage patterns

## Testing

Comprehensive test coverage includes:

- **Manager Tests** (`manager_test.go`):
  - Key generation with cryptographic validation
  - CRUD operations with Vault mock
  - Rotation logic and grace periods
  - Scope validation

- **Middleware Tests** (`middleware_test.go`):
  - Bearer token extraction and validation
  - Scope authorization (all/any)
  - Context propagation
  - Error response formats

- **Scheduler Tests** (`scheduler_test.go`):
  - Rotation timing calculations
  - Cleanup logic
  - Scheduler lifecycle
  - Concurrent operations

**Test Results**: 25+ passing tests covering core functionality

## Migration Path

### Phase 1: Initial Deployment
1. Deploy API key system to staging
2. Create test keys for each organization
3. Validate authentication flow
4. Test rotation scheduler

### Phase 2: Production Rollout
1. Enable API key authentication for new endpoints
2. Generate initial production keys
3. Start rotation scheduler with conservative settings
4. Monitor key usage and rotation

### Phase 3: Full Migration
1. Migrate existing authentication to API keys
2. Enable required scopes for sensitive endpoints
3. Implement rate limiting per key
4. Establish rotation policies (90-day TTL)

## Configuration

### Default Values

```go
const (
    DefaultKeyPrefix      = "sk_"
    DefaultKeyTTL         = 90 * 24 * time.Hour  // 90 days
    DefaultCheckInterval  = 1 * time.Hour         // Hourly checks
    DefaultGracePeriod    = 7 * 24 * time.Hour   // 7 days
    DefaultRotationTTL    = 7 * 24 * time.Hour   // Rotate 7 days before expiry
    DefaultCleanupAge     = 30 * 24 * time.Hour  // Delete after 30 days
)
```

### Environment Variables

```bash
# Vault Configuration (from ADR-0015)
VAULT_ADDR=https://vault.example.com:8200
VAULT_TOKEN=s.xxxxxxxxxxxxx
VAULT_MOUNT_PATH=secret

# API Key Configuration
API_KEY_TTL=2160h              # 90 days
API_KEY_CHECK_INTERVAL=1h      # Rotation check frequency
API_KEY_GRACE_PERIOD=168h      # 7 days
API_KEY_ROTATION_TTL=168h      # Rotate 7 days before expiry
```

## Open Questions

1. **Rate Limiting**: Should rate limits be per-key or per-organization?
   - **Decision**: Implement both with configurable limits

2. **Key Recovery**: Should revoked keys support recovery within grace period?
   - **Decision**: No recovery to maintain security. Create new key instead.

3. **Multi-Environment**: Should dev/staging keys have different TTLs?
   - **Decision**: Yes, configure per environment (dev: 30d, staging: 60d, prod: 90d)

4. **Webhook Notifications**: Should rotation events trigger webhooks?
   - **Future**: Consider for Phase 4 with hooks system integration

## References

- **ADR-0015**: HashiCorp Vault Integration (foundational)
- **NIST 800-63B**: Digital Identity Guidelines (authentication)
- **OWASP API Security Top 10**: API authentication best practices
- **Stripe API**: Key format inspiration (sk_ prefix)
- **RFC 6750**: OAuth 2.0 Bearer Token Usage

## Related Work

- M9.2.4: HashiCorp Vault Integration (prerequisite)
- M9.2.6: OAuth 2.0 Integration (future)
- M9.2.7: SAML 2.0 Enterprise SSO (future)
- M9.3: Enterprise Encryption (future enhancement)

## Success Criteria

1. ✅ Cryptographically secure 256-bit key generation
2. ✅ Vault-backed persistent storage with versioning
3. ✅ HTTP Bearer token authentication middleware
4. ✅ Automatic rotation with configurable policies
5. ✅ Grace period support for smooth transitions
6. ✅ Scope-based authorization
7. ✅ Comprehensive test coverage (25+ tests)
8. ✅ Production-ready error handling

## Date

2025-11-21

## Authors

- Felix Geelhaar (Implementation Lead)
- Claude Code (Design Assistant)
