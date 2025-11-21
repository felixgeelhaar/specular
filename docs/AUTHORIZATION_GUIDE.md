# Authorization Guide

Comprehensive guide to Specular's Attribute-Based Access Control (ABAC) authorization system.

## Table of Contents

- [Overview](#overview)
- [Core Concepts](#core-concepts)
- [Getting Started](#getting-started)
- [Policy Builder API](#policy-builder-api)
- [Built-in Roles](#built-in-roles)
- [HTTP Middleware](#http-middleware)
- [Policy Management API](#policy-management-api)
- [Audit Logging](#audit-logging)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [API Reference](#api-reference)

## Overview

Specular's authorization system implements **Attribute-Based Access Control (ABAC)**, a flexible and powerful authorization model that makes decisions based on attributes of:

- **Subject** (who is making the request)
- **Action** (what they want to do)
- **Resource** (what they want to access)
- **Environment** (contextual information like IP, time, etc.)

### Key Features

- **Flexible Policy Language**: Define fine-grained access control policies
- **AWS IAM-style Evaluation**: Default deny, explicit deny wins, require explicit allow
- **Organization-Scoped**: Multi-tenant support with organization isolation
- **Role-Based & Attribute-Based**: Combine traditional RBAC with flexible ABAC
- **Audit Logging**: Track all authorization decisions
- **HTTP Integration**: Middleware for seamless web app integration
- **RESTful API**: Manage policies via HTTP endpoints

### Architecture Decision Record

For architectural details and design rationale, see [ADR-0013: Attribute-Based Access Control (ABAC) Authorization Engine](./adr/0013-abac-authorization-engine.md).

## Core Concepts

### Policies

A **policy** is a rule that grants or denies access. Each policy contains:

```go
type Policy struct {
    ID             string      // Unique identifier
    OrganizationID string      // Organization this policy belongs to
    Name           string      // Human-readable name
    Description    string      // What this policy does
    Effect         Effect      // "allow" or "deny"
    Principals     []Principal // Who this applies to
    Actions        []string    // What actions are allowed/denied
    Resources      []string    // What resources are affected
    Conditions     []Condition // Additional constraints
    Enabled        bool        // Whether this policy is active
}
```

### Principals

A **principal** identifies who the policy applies to. Principals can be:

**Role-based** (traditional RBAC):
```go
Principal{
    Role:  "admin",
    Scope: "organization", // or "team"
}
```

**Attribute-based** (flexible ABAC):
```go
Principal{
    Attribute: "$subject.department",
    Operator:  OperatorEquals,
    Value:     "engineering",
}
```

### Actions

**Actions** define what operations are allowed or denied. Format: `resource:operation`

Examples:
- `plan:approve` - Approve a plan
- `build:execute` - Execute a build
- `policy:create` - Create a policy
- `*:read` - Read any resource
- `*` - All operations (owner only)

### Resources

**Resources** specify what entities the policy applies to. Format: `type:id`

Examples:
- `plan:*` - All plans
- `plan:plan-123` - Specific plan
- `build:build-456` - Specific build
- `*` - All resources (owner only)

### Conditions

**Conditions** add contextual constraints to policies:

```go
Condition{
    Attribute: "$resource.status",
    Operator:  OperatorEquals,
    Value:     "pending",
}
```

**Supported Operators**:
- `equals` / `not_equals`
- `in` / `not_in`
- `greater_than` / `less_than`
- `starts_with` / `ends_with`
- `contains`

**Attribute Sources**:
- `$subject.*` - Subject attributes (user info, roles, etc.)
- `$resource.*` - Resource attributes (status, owner, etc.)
- `$environment.*` - Environmental context (IP, time, etc.)

### Evaluation Logic

Authorization follows AWS IAM-style evaluation:

1. **Default Deny**: Access is denied unless explicitly allowed
2. **Explicit Deny Wins**: If any deny policy matches, access is denied
3. **Require Explicit Allow**: At least one allow policy must match

```
┌─────────────────────────────────────┐
│ 1. Check for Explicit DENY         │
│    └─ If found → DENY (stop)       │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│ 2. Check for Explicit ALLOW         │
│    └─ If found → ALLOW              │
│    └─ If not found → DENY (default) │
└─────────────────────────────────────┘
```

## Getting Started

### Basic Usage

```go
package main

import (
    "context"
    "github.com/felixgeelhaar/specular/internal/authz"
    "github.com/felixgeelhaar/specular/internal/auth"
)

func main() {
    // 1. Create policy store
    policyStore := authz.NewInMemoryPolicyStore()

    // 2. Create attribute resolver
    resourceStore := authz.NewInMemoryResourceStore()
    resolver := authz.NewDefaultAttributeResolver(resourceStore)

    // 3. Create authorization engine
    engine := authz.NewEngine(policyStore, resolver)

    // 4. Add policies
    policy := authz.NewAdminPolicy("org-1", "admin-policy-1")
    policyStore.CreatePolicy(context.Background(), policy)

    // 5. Make authorization decision
    session := &auth.Session{
        UserID:           "user-1",
        OrganizationID:   "org-1",
        OrganizationRole: "admin",
    }

    req := &authz.AuthorizationRequest{
        Subject: session,
        Action:  "plan:approve",
        Resource: authz.Resource{
            Type: "plan",
            ID:   "plan-123",
        },
    }

    decision, err := engine.Evaluate(context.Background(), req)
    if err != nil {
        // Handle error
    }

    if decision.Allowed {
        // Access granted
        fmt.Println("Access granted:", decision.Reason)
    } else {
        // Access denied
        fmt.Println("Access denied:", decision.Reason)
    }
}
```

## Policy Builder API

The **PolicyBuilder** provides a fluent API for constructing policies:

### Basic Example

```go
policy := authz.NewPolicyBuilder("org-1", "Admin Approve Plans").
    WithID("policy-1").
    WithDescription("Admins can approve pending plans").
    AllowRole("admin").
    OnActions("plan:approve").
    OnResourceType("plan").
    WithCondition("$resource.status", authz.OperatorEquals, "pending").
    Build()
```

### Builder Methods

| Method | Description |
|--------|-------------|
| `WithID(id)` | Set policy ID |
| `WithDescription(desc)` | Set description |
| `WithEffect(effect)` | Set effect (allow/deny) |
| `AllowRole(role)` | Add organization-scoped role |
| `AllowTeamRole(role)` | Add team-scoped role |
| `AllowAttribute(attr, op, val)` | Add attribute-based principal |
| `OnActions(actions...)` | Add actions |
| `OnResources(resources...)` | Add specific resources |
| `OnAllResources()` | Apply to all resources (*) |
| `OnResourceType(type)` | Apply to all resources of type |
| `WithCondition(attr, op, val)` | Add condition |
| `Disabled()` | Mark policy as disabled |
| `Build()` | Construct final policy |

### Multiple Roles

```go
policy := authz.NewPolicyBuilder("org-1", "Multi-Role Policy").
    AllowRole("admin").
    AllowRole("member").
    AllowTeamRole("lead").
    OnActions("*:read").
    OnAllResources().
    Build()
```

### Attribute-Based Principals

```go
policy := authz.NewPolicyBuilder("org-1", "Engineering Access").
    AllowAttribute("$subject.department", authz.OperatorEquals, "engineering").
    OnActions("*:read", "*:create").
    OnResourceType("plan").
    Build()
```

## Built-in Roles

Specular provides pre-configured helper functions for common roles:

### Owner Policy

Full access to all resources and operations:

```go
policy := authz.NewOwnerPolicy("org-1", "owner-policy-1")
// Effect: allow
// Principals: owner
// Actions: *
// Resources: *
```

### Admin Policy

Approve, create, update, delete, and read operations:

```go
policy := authz.NewAdminPolicy("org-1", "admin-policy-1")
// Effect: allow
// Principals: admin
// Actions: *:approve, *:create, *:update, *:delete, *:read
// Resources: *
```

### Member Policy

Create, update, and read operations:

```go
policy := authz.NewMemberPolicy("org-1", "member-policy-1")
// Effect: allow
// Principals: member
// Actions: *:create, *:update, *:read
// Resources: *
```

### Viewer Policy

Read-only access:

```go
policy := authz.NewViewerPolicy("org-1", "viewer-policy-1")
// Effect: allow
// Principals: viewer
// Actions: *:read, *:list
// Resources: *
```

### Role Hierarchy

```
Owner (*)
  ├─ Admin (*:approve, *:create, *:update, *:delete, *:read)
  │   ├─ Member (*:create, *:update, *:read)
  │   │   └─ Viewer (*:read, *:list)
```

## Specialized Policy Helpers

### Resource-Specific Policy

Target specific resource types and actions:

```go
policy := authz.NewResourceSpecificPolicy(
    "org-1",
    "plan-approve-policy",
    "Admins Approve Plans",
    "admin",
    "plan:approve",
    "plan",
)
```

### Conditional Policy

Add attribute-based conditions:

```go
condition := authz.Condition{
    Attribute: "$resource.status",
    Operator:  authz.OperatorEquals,
    Value:     "pending",
}

policy := authz.NewConditionalPolicy(
    "org-1",
    "conditional-policy",
    "Approve Pending Plans",
    "admin",
    []string{"plan:approve"},
    "plan",
    condition,
)
```

### Team Policy

Team-scoped access control:

```go
policy := authz.NewTeamPolicy(
    "org-1",
    "team-policy",
    "Team Lead Policy",
    "lead",
    []string{"build:execute", "build:read"},
    []string{"build:*"},
)
```

### Deny Policy

Explicit denial (always wins):

```go
principals := []authz.Principal{
    {Role: "member", Scope: "organization"},
}

policy := authz.NewDenyPolicy(
    "org-1",
    "deny-delete",
    "Members Cannot Delete",
    principals,
    []string{"*:delete"},
    []string{"*"},
)
```

## HTTP Middleware

Protect HTTP endpoints with authorization middleware:

### Basic Setup

```go
package main

import (
    "net/http"
    "github.com/felixgeelhaar/specular/internal/authz"
)

func main() {
    // Setup engine
    policyStore := authz.NewInMemoryPolicyStore()
    resourceStore := authz.NewInMemoryResourceStore()
    resolver := authz.NewDefaultAttributeResolver(resourceStore)
    engine := authz.NewEngine(policyStore, resolver)

    // Create middleware
    mw := authz.NewAuthorizationMiddleware(engine)

    // Wrap handlers
    mux := http.NewServeMux()

    // Require "plan:approve" permission
    mux.Handle("/api/plans/approve", mw.RequirePermission(
        "plan:approve",
        http.HandlerFunc(approvePlanHandler),
    ))

    // Require any of: "plan:create" OR "plan:update"
    mux.Handle("/api/plans", mw.RequireAnyPermission(
        []string{"plan:create", "plan:update"},
        http.HandlerFunc(managePlansHandler),
    ))

    http.ListenAndServe(":8080", mux)
}

func approvePlanHandler(w http.ResponseWriter, r *http.Request) {
    // This only executes if user has "plan:approve" permission
    w.WriteHeader(http.StatusOK)
}
```

### Context Integration

The middleware automatically extracts the authenticated session from the request context:

```go
// In your authentication middleware
ctx := authz.SetSessionInContext(r.Context(), session)
r = r.WithContext(ctx)

// Authorization middleware will use this session
next.ServeHTTP(w, r)
```

### Resource-Aware Authorization

For resource-specific checks, extract resource info from the request:

```go
func customAuthzHandler(w http.ResponseWriter, r *http.Request) {
    session := authz.GetSessionFromContext(r.Context())
    planID := r.URL.Query().Get("plan_id")

    req := &authz.AuthorizationRequest{
        Subject: session,
        Action:  "plan:approve",
        Resource: authz.Resource{
            Type: "plan",
            ID:   planID,
        },
    }

    decision, err := engine.Evaluate(r.Context(), req)
    if !decision.Allowed {
        http.Error(w, decision.Reason, http.StatusForbidden)
        return
    }

    // Proceed with authorized action
}
```

## Policy Management API

Manage policies via RESTful HTTP endpoints:

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/policies` | Create new policy |
| GET | `/api/policies/:id` | Get specific policy |
| PUT | `/api/policies/:id` | Update policy |
| DELETE | `/api/policies/:id` | Delete policy |
| POST | `/api/policies/simulate` | Simulate authorization decision |

### Create Policy

**Request**:
```bash
POST /api/policies
Content-Type: application/json

{
  "name": "Admin Policy",
  "description": "Admins can approve plans",
  "effect": "allow",
  "principals": [
    {"role": "admin", "scope": "organization"}
  ],
  "actions": ["plan:approve"],
  "resources": ["plan:*"],
  "conditions": [],
  "enabled": true
}
```

**Response**: `201 Created`
```json
{
  "id": "policy-org-1-123",
  "organization_id": "org-1",
  "name": "Admin Policy",
  "description": "Admins can approve plans",
  "effect": "allow",
  "principals": [
    {"role": "admin", "scope": "organization"}
  ],
  "actions": ["plan:approve"],
  "resources": ["plan:*"],
  "conditions": [],
  "enabled": true,
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

### Get Policy

**Request**:
```bash
GET /api/policies/policy-123
```

**Response**: `200 OK`
```json
{
  "id": "policy-123",
  "organization_id": "org-1",
  "name": "Admin Policy",
  ...
}
```

### Update Policy

**Request** (partial update):
```bash
PUT /api/policies/policy-123
Content-Type: application/json

{
  "enabled": false
}
```

**Response**: `200 OK`

### Delete Policy

**Request**:
```bash
DELETE /api/policies/policy-123
```

**Response**: `204 No Content`

### Simulate Authorization

Test authorization decisions without affecting actual policies:

**Request**:
```bash
POST /api/policies/simulate
Content-Type: application/json

{
  "subject": {
    "UserID": "user-123",
    "OrganizationID": "org-1",
    "OrganizationRole": "admin"
  },
  "action": "plan:approve",
  "resource": {
    "type": "plan",
    "id": "plan-123"
  },
  "environment": {
    "client_ip": "192.168.1.1"
  }
}
```

**Response**: `200 OK`
```json
{
  "allowed": true,
  "reason": "access granted by policy admin-policy-1",
  "policy_ids": ["admin-policy-1"],
  "timestamp": "2024-01-15T10:00:00Z"
}
```

## Audit Logging

Track all authorization decisions for compliance and debugging:

### Setup

```go
// Create audit logger
auditLogger := authz.NewDefaultAuditLogger(authz.AuditLoggerConfig{
    Writer:             os.Stdout,
    LogAllDecisions:    true,  // Log both allowed and denied
    IncludeEnvironment: true,  // Log environment context
    BufferSize:         100,
})

// Attach to engine
engine = authz.WithAuditLogger(engine, auditLogger)

// Remember to close on shutdown
defer auditLogger.Close()
```

### Audit Entry Format

```json
{
  "timestamp": "2024-01-15T10:00:00Z",
  "allowed": true,
  "reason": "access granted by policy admin-policy-1",
  "user_id": "user-123",
  "email": "admin@example.com",
  "organization_id": "org-1",
  "role": "admin",
  "action": "plan:approve",
  "resource_type": "plan",
  "resource_id": "plan-123",
  "policy_ids": ["admin-policy-1"],
  "environment": {
    "client_ip": "192.168.1.1",
    "user_agent": "Mozilla/5.0..."
  },
  "duration_ms": 5
}
```

### Configuration Options

```go
type AuditLoggerConfig struct {
    Writer             io.Writer // Where to write logs
    LogAllDecisions    bool      // Log both allow and deny
    IncludeEnvironment bool      // Include environment context
    BufferSize         int       // Async buffer size (0 = sync)
}
```

### Log Filtering

Log only denials:
```go
auditLogger := authz.NewDefaultAuditLogger(authz.AuditLoggerConfig{
    Writer:          os.Stdout,
    LogAllDecisions: false, // Only denials
})
```

### In-Memory Logger (Testing)

```go
auditLogger := authz.NewInMemoryAuditLogger()

// Later, retrieve entries
entries := auditLogger.GetEntries()
for _, entry := range entries {
    fmt.Printf("%s: %s\n", entry.UserID, entry.Action)
}
```

### Signed Audit Logging (ECDSA P-256)

For enterprise compliance and tamper-proof audit trails, Specular supports **cryptographically signed audit logging** using ECDSA P-256 signatures.

#### Why Signed Audit Logs?

Signed audit logs provide:

- **Tamper Detection**: Any modification to audit entries is immediately detectable
- **Non-Repudiation**: Cryptographic proof of who created each audit entry
- **Compliance**: Meets SOC2, ISO 27001, HIPAA, and PCI DSS requirements
- **Legal Admissibility**: Audit logs can serve as legal evidence
- **Insider Threat Protection**: Even DBAs cannot forge or modify entries without detection

For architectural details, see [ADR-0014: Signed Audit Logging](./adr/0014-signed-audit-logging.md).

#### Setup Signed Audit Logger

```go
import (
    "github.com/felixgeelhaar/specular/internal/authz"
    "github.com/felixgeelhaar/specular/internal/attestation"
)

// Create base audit logger
baseLogger := authz.NewDefaultAuditLogger(authz.AuditLoggerConfig{
    Writer:             os.Stdout,
    LogAllDecisions:    true,
    IncludeEnvironment: true,
    BufferSize:         1000,
})

// Create signer (ephemeral keys for development)
signer, err := attestation.NewEphemeralSigner("system@specular.dev")
if err != nil {
    log.Fatal(err)
}

// Wrap base logger with signing
signedLogger := authz.NewSignedAuditLogger(baseLogger, signer)

// Use in authorization engine
engine = authz.WithAuditLogger(engine, signedLogger)

// Remember to close on shutdown
defer signedLogger.Close()
```

#### Signed Audit Entry Format

When using signed audit logging, entries include cryptographic signatures:

```json
{
  "timestamp": "2025-11-21T10:00:00Z",
  "allowed": true,
  "reason": "access granted by policy admin-policy-1",
  "user_id": "user-123",
  "organization_id": "org-1",
  "action": "plan:approve",
  "resource_type": "plan",
  "resource_id": "plan-123",
  "policy_ids": ["admin-policy-1"],
  "duration_ms": 5,

  "signature": "MEUCIQDd7Ym...base64-encoded-signature...",
  "public_key": "MFkwEwYHK...base64-encoded-public-key...",
  "signed_by": "system@specular.dev"
}
```

**Signature Fields**:
- `signature`: 64-byte ECDSA P-256 signature (base64-encoded)
- `public_key`: X.509 PKIX public key for verification (base64-encoded)
- `signed_by`: Identity of the signer (email or system identifier)

#### Verifying Audit Entries

Use `AuditVerifier` to verify signed audit entries:

```go
// Create verifier with optional constraints
verifier := authz.NewAuditVerifier(
    authz.WithMaxAge(90 * 24 * time.Hour), // Entries valid for 90 days
    authz.WithAllowedSigners([]string{
        "system@specular.dev",
        "audit-service@example.com",
    }),
)

// Verify single entry
result, err := verifier.Verify(entry)
if err != nil {
    log.Fatalf("verification error: %v", err)
}

if !result.Valid {
    log.Printf("⚠️  TAMPERING DETECTED: %s", result.Reason)
    // Alert security team, trigger incident response
} else {
    log.Printf("✓ Signature verified: %s", result.Reason)
}
```

#### Batch Verification

For compliance reports, verify multiple entries:

```go
// Verify batch of entries
entries := auditLogger.GetEntries()
results, err := verifier.VerifyBatch(entries)
if err != nil {
    log.Fatal(err)
}

// Generate summary
summary := authz.Summarize(results)
log.Printf("Verified %d entries: %d valid, %d invalid, %d unsigned",
    summary.Total, summary.Valid, summary.Invalid, summary.Unsigned)

// Alert on tampering
if summary.Invalid > 0 {
    log.Printf("⚠️  SECURITY ALERT: %d tampered entries detected!", summary.Invalid)
}
```

#### Verifier Options

```go
// No restrictions (verify signature only)
verifier := authz.NewAuditVerifier()

// Age restriction (reject entries older than 90 days)
verifier := authz.NewAuditVerifier(
    authz.WithMaxAge(90 * 24 * time.Hour),
)

// Signer whitelist (only trust specific identities)
verifier := authz.NewAuditVerifier(
    authz.WithAllowedSigners([]string{
        "system@specular.dev",
        "backup-system@specular.dev",
    }),
)

// Combined restrictions
verifier := authz.NewAuditVerifier(
    authz.WithMaxAge(90 * 24 * time.Hour),
    authz.WithAllowedSigners([]string{"system@specular.dev"}),
)
```

#### Production Key Management

For production deployments, use secure key management:

##### HashiCorp Vault Integration

```go
import (
    vault "github.com/hashicorp/vault/api"
)

// Connect to Vault
client, err := vault.NewClient(vault.DefaultConfig())
if err != nil {
    log.Fatal(err)
}

// Read signing key from Vault
secret, err := client.Logical().Read("secret/data/audit-signing-key")
if err != nil {
    log.Fatal(err)
}

// Use key with custom signer implementation
signer := NewVaultSigner(client, "audit-signing-key", "system@specular.dev")
signedLogger := authz.NewSignedAuditLogger(baseLogger, signer)
```

##### AWS KMS Integration

```go
import (
    "github.com/aws/aws-sdk-go/service/kms"
)

// Create KMS client
kmsSvc := kms.New(session.Must(session.NewSession()))

// Use KMS key for signing
signer := NewKMSSigner(kmsSvc, "alias/audit-signing-key", "system@specular.dev")
signedLogger := authz.NewSignedAuditLogger(baseLogger, signer)
```

#### Key Rotation

Implement quarterly key rotation for security:

```go
// Rotate signing keys every 90 days
func rotateSigningKey(keyStore KeyStore) error {
    // Generate new key
    newKey, err := keyStore.GenerateKey()
    if err != nil {
        return err
    }

    // Archive old key (keep public key for verification)
    if err := keyStore.ArchiveCurrentKey(); err != nil {
        return err
    }

    // Activate new key
    return keyStore.SetActiveKey(newKey)
}
```

**Key Retention Policy**:
- Keep **public keys** indefinitely (needed to verify old entries)
- Archive **private keys** after rotation
- Verify old entries remain valid after rotation

#### Compliance Benefits

Signed audit logging helps meet compliance requirements:

| Standard | Requirement | How Signed Audit Logging Helps |
|----------|-------------|-------------------------------|
| **SOC2 Type II** | Tamper-evident audit logs | ECDSA signatures detect any modifications |
| **ISO 27001** | Cryptographic controls for data integrity | P-256 provides industry-standard protection |
| **HIPAA** | Audit log integrity and authenticity | Non-repudiation proves who created entries |
| **PCI DSS** | Tamper detection for audit trails | Invalid signatures trigger security alerts |
| **GDPR** | Demonstrable data integrity | Cryptographic proof of audit log integrity |

#### Performance Characteristics

Signing overhead is minimal:

- **Signing**: ~3ms per entry (async, non-blocking)
- **Verification**: ~5ms per entry (performed during audits, not real-time)
- **Storage**: ~250 bytes additional per entry
- **Memory**: Zero allocations (pre-allocated buffers)

```go
// Benchmark results (Apple M1, Go 1.21)
BenchmarkSign-8       500000    2847 ns/op    0 allocs/op
BenchmarkVerify-8     200000    5123 ns/op    0 allocs/op
```

#### Fail-Safe Design

If signing fails, entries are still logged (unsigned):

```go
func (l *SignedAuditLogger) LogDecision(ctx context.Context, entry *AuditEntry) error {
    // Attempt to sign
    if err := l.signEntry(entry); err != nil {
        // Log error but continue with unsigned entry
        // Ensures audit trail is never lost
        log.Printf("audit: failed to sign entry: %v", err)
    }

    // Always log the entry (signed or unsigned)
    return l.wrapped.LogDecision(ctx, entry)
}
```

This ensures:
- Audit logs are never lost due to signing failures
- Monitoring can alert on signing failure rates
- Unsigned entries can be detected during verification

#### Monitoring Signed Audit Logs

Add metrics for production monitoring:

```go
// Track signing failures
signingFailures := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "audit_signing_failures_total",
        Help: "Total number of audit entry signing failures",
    },
    []string{"organization_id"},
)

// Track verification failures (tampering detected)
verificationFailures := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "audit_verification_failures_total",
        Help: "Total number of tampered audit entries detected",
    },
    []string{"organization_id", "reason"},
)

// Alert on tampering
if !result.Valid {
    verificationFailures.WithLabelValues(
        entry.OrganizationID,
        result.Reason,
    ).Inc()

    // Send security alert
    alertSecurityTeam(entry, result)
}
```

#### CLI Verification Tools

Specular provides CLI tools for audit verification:

```bash
# Verify audit log file
specular audit verify audit.json --signer system@specular.dev

# Verify database audit entries
specular audit verify --database \
  --org org-123 \
  --days 30 \
  --max-age 90d

# Export verified audit report (PDF)
specular audit export \
  --org org-123 \
  --format pdf \
  --signed-only \
  --output compliance-report-2025-q1.pdf

# Audit verification statistics
specular audit stats --org org-123
# Output:
# Total entries:    15,234
# Valid signatures: 15,230 (99.97%)
# Invalid:          0 (0%)
# Unsigned:         4 (0.03%)
```

## HashiCorp Vault Integration

For enterprise deployments, Specular integrates with **HashiCorp Vault** to securely store ECDSA signing keys and other sensitive secrets. This integration provides centralized secret management, key rotation, and compliance with enterprise security standards.

For architectural details, see [ADR-0015: HashiCorp Vault Integration](./adr/0015-hashicorp-vault-integration.md).

### Why HashiCorp Vault?

Vault integration provides:

- **Centralized Secret Management**: Store all signing keys and credentials in a single, secure location
- **Enterprise Features**: Namespaces, replication, auto-unsealing, and high availability
- **Key Rotation**: Automated key rotation with versioning and backward compatibility
- **Audit Trail**: Vault's built-in audit logging tracks all secret access
- **Compliance**: Meets SOC2, ISO 27001, HIPAA, and PCI DSS requirements
- **Cloud-Agnostic**: Works across AWS, Azure, GCP, and on-premises environments
- **Access Control**: Fine-grained ACL policies for secret access

### Architecture Overview

The Vault integration consists of three main components:

```go
// 1. Vault Client - Manages connection and authentication
vault.Client
  ├─ TLS/mTLS support
  ├─ Automatic token renewal
  ├─ Namespace support (Enterprise)
  └─ KV v2 secrets engine

// 2. KV Store - Manages secret storage and versioning
vault.KV
  ├─ Put/Get operations
  ├─ Version management
  ├─ Soft deletion
  └─ Metadata management

// 3. Vault Signer - Implements authz.Signer interface
vault.VaultSigner
  ├─ ECDSA P-256 key storage
  ├─ Key caching (5-minute TTL)
  ├─ Key rotation
  └─ Integration with signed audit logging
```

### Development Setup

#### 1. Start Vault Dev Server

For local development, use Vault's development mode:

```bash
# Start Vault dev server
vault server -dev -dev-root-token-id=root

# In another terminal, configure Vault client
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root'

# Verify connection
vault status
```

#### 2. Configure Specular with Vault

```go
import (
    "context"
    "github.com/felixgeelhaar/specular/internal/vault"
    "github.com/felixgeelhaar/specular/internal/authz"
)

func main() {
    // Create Vault client
    client, err := vault.NewClient(vault.Config{
        Address:   "http://127.0.0.1:8200",
        Token:     "root", // Dev token
        MountPath: "secret", // Default KV v2 mount
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Create Vault-backed signer with auto-generation
    signer, err := client.NewSigner(ctx, vault.SignerConfig{
        KeyPath:      "audit/signing-key",
        Identity:     "system@specular.dev",
        AutoGenerate: true, // Generate key if it doesn't exist
        CacheTTL:     5 * time.Minute,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create base audit logger
    baseLogger := authz.NewDefaultAuditLogger(authz.AuditLoggerConfig{
        Writer:             os.Stdout,
        LogAllDecisions:    true,
        IncludeEnvironment: true,
        BufferSize:         1000,
    })

    // Wrap with signed audit logging
    signedLogger := authz.NewSignedAuditLogger(baseLogger, signer)

    // Use in authorization engine
    engine = authz.WithAuditLogger(engine, signedLogger)
    defer signedLogger.Close()
}
```

### Production Setup

#### 1. Production Vault Configuration

```go
import (
    "crypto/tls"
    "crypto/x509"
    "github.com/felixgeelhaar/specular/internal/vault"
)

func createProductionVaultClient() (*vault.Client, error) {
    // Load CA certificate
    caCert, err := os.ReadFile("/etc/vault/ca.crt")
    if err != nil {
        return nil, err
    }

    // Create Vault client with mTLS
    client, err := vault.NewClient(vault.Config{
        Address:   "https://vault.prod.example.com:8200",
        Token:     "", // Will use VAULT_TOKEN env var
        MountPath: "secret",
        Namespace: "production", // Enterprise feature
        TLSConfig: &vault.TLSConfig{
            CACert:        "/etc/vault/ca.crt",
            ClientCert:    "/etc/vault/client.crt",
            ClientKey:     "/etc/vault/client.key",
            TLSServerName: "vault.prod.example.com",
        },
        TokenTTL: 24 * time.Hour, // Token renewal interval
    })
    if err != nil {
        return nil, err
    }

    // Verify connection
    if err := client.Health(context.Background()); err != nil {
        client.Close()
        return nil, fmt.Errorf("vault health check failed: %w", err)
    }

    return client, nil
}
```

#### 2. Environment Variables

Configure Vault connection via environment variables:

```bash
# Vault server
export VAULT_ADDR='https://vault.prod.example.com:8200'
export VAULT_TOKEN='your-production-token'

# Optional: Namespace (Vault Enterprise)
export VAULT_NAMESPACE='production'

# Optional: TLS configuration
export VAULT_CACERT='/etc/vault/ca.crt'
export VAULT_CLIENT_CERT='/etc/vault/client.crt'
export VAULT_CLIENT_KEY='/etc/vault/client.key'
```

#### 3. Vault Policies

Create appropriate Vault policies for Specular:

```hcl
# Audit signing key policy
path "secret/data/audit/signing-key" {
  capabilities = ["read", "create", "update"]
}

path "secret/metadata/audit/*" {
  capabilities = ["list", "read"]
}

# Read-only access for verification
path "secret/data/audit/*" {
  capabilities = ["read"]
}
```

Apply the policy:

```bash
# Write policy
vault policy write specular-audit specular-audit.hcl

# Create token with policy
vault token create -policy=specular-audit -period=24h
```

### VaultSigner Configuration

#### Basic Configuration

```go
// Create signer with default settings
signer, err := client.NewSigner(ctx, vault.SignerConfig{
    KeyPath:      "audit/signing-key",
    Identity:     "system@specular.dev",
    AutoGenerate: true,
    CacheTTL:     5 * time.Minute, // Default cache TTL
})
```

#### Advanced Configuration

```go
// Disable caching (always fetch from Vault)
signer, err := client.NewSigner(ctx, vault.SignerConfig{
    KeyPath:      "audit/signing-key",
    Identity:     "prod-system@specular.dev",
    AutoGenerate: false, // Fail if key doesn't exist
    CacheTTL:     0,     // No caching
})

// Custom cache TTL for high-volume environments
signer, err := client.NewSigner(ctx, vault.SignerConfig{
    KeyPath:      "audit/signing-key",
    Identity:     "high-volume@specular.dev",
    AutoGenerate: true,
    CacheTTL:     15 * time.Minute, // Longer cache for better performance
})
```

#### Multiple Signers

For multi-tenant environments, create separate signers per organization:

```go
// Organization-specific signers
func createOrgSigner(client *vault.Client, orgID string) (*vault.VaultSigner, error) {
    return client.NewSigner(ctx, vault.SignerConfig{
        KeyPath:      fmt.Sprintf("audit/%s/signing-key", orgID),
        Identity:     fmt.Sprintf("%s@specular.dev", orgID),
        AutoGenerate: true,
        CacheTTL:     5 * time.Minute,
    })
}

// Usage
org1Signer, err := createOrgSigner(client, "org-1")
org2Signer, err := createOrgSigner(client, "org-2")
```

### Key Management

#### Viewing Key Information

```go
// Get key metadata
info, err := signer.GetKeyInfo(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Algorithm: %s\n", info.Algorithm)
fmt.Printf("Identity: %s\n", info.Identity)
fmt.Printf("Created: %s\n", info.CreatedAt)
fmt.Printf("Version: %d\n", info.Version)

// Output:
// Algorithm: ECDSA-P256
// Identity: system@specular.dev
// Created: 2025-11-21T10:00:00Z
// Version: 3
```

#### Manual Key Generation

```go
// Generate a new key (creates version 1)
err := signer.GenerateKey(ctx)
if err != nil {
    log.Fatal(err)
}

// Verify generation
info, _ := signer.GetKeyInfo(ctx)
fmt.Printf("New key generated: version %d\n", info.Version)
```

#### Key Rotation

```go
// Rotate signing key (creates new version)
err := signer.RotateKey(ctx)
if err != nil {
    log.Fatal(err)
}

// Clear cache to force reload
signer.ClearCache()

// Verify rotation
info, _ := signer.GetKeyInfo(ctx)
fmt.Printf("Key rotated to version %d\n", info.Version)
```

**Rotation Schedule**:
- **Development**: Rotate monthly or as needed
- **Production**: Rotate quarterly (every 90 days)
- **After Incident**: Rotate immediately if compromise suspected

#### Automated Key Rotation

```go
// Quarterly rotation scheduler
func startKeyRotationScheduler(signer *vault.VaultSigner) {
    ticker := time.NewTicker(90 * 24 * time.Hour) // Every 90 days
    go func() {
        for range ticker.C {
            if err := signer.RotateKey(context.Background()); err != nil {
                log.Printf("ERROR: key rotation failed: %v", err)
                // Alert security team
                alertSecurityTeam("Key rotation failed", err)
            } else {
                log.Printf("INFO: signing key rotated successfully")
                // Log to audit trail
                auditKeyRotation()
            }
        }
    }()
}
```

### KV v2 Operations

The Vault client provides access to the KV v2 secrets engine for storing other sensitive data:

#### Storing Secrets

```go
kv := client.KV()

// Store database credentials
err := kv.Put(ctx, "db/credentials", map[string]interface{}{
    "username": "admin",
    "password": "secret123",
    "host":     "db.example.com",
    "port":     5432,
})

// Store with custom metadata
metadata := map[string]string{
    "owner":       "engineering",
    "environment": "production",
    "rotation":    "90d",
}
err := kv.PutWithMetadata(ctx, "db/credentials", data, metadata)
```

#### Retrieving Secrets

```go
// Get latest version
secret, err := kv.Get(ctx, "db/credentials")
if err != nil {
    log.Fatal(err)
}

username := secret.Data["username"].(string)
password := secret.Data["password"].(string)

// Get specific version
secretV2, err := kv.GetVersion(ctx, "db/credentials", 2)
oldPassword := secretV2.Data["password"].(string)
```

#### Listing Secrets

```go
// List all secrets under a path
keys, err := kv.List(ctx, "db")
if err != nil {
    log.Fatal(err)
}

for _, key := range keys {
    fmt.Printf("- %s\n", key)
}
// Output:
// - credentials
// - backup-credentials
// - readonly-credentials
```

#### Secret Lifecycle

```go
// Soft delete (can be undeleted)
err := kv.Delete(ctx, "db/old-credentials")

// Delete specific versions
err := kv.DeleteVersions(ctx, "db/credentials", []int{1, 2})

// Permanent destruction (cannot be recovered)
err := kv.Destroy(ctx, "db/credentials", []int{1})
```

### Health Checks

```go
// Vault health check
func checkVaultHealth(client *vault.Client) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Health(ctx); err != nil {
        return fmt.Errorf("vault health check failed: %w", err)
    }

    return nil
}

// Signer health check
func checkSignerHealth(signer *vault.VaultSigner) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Verify key exists and is accessible
    info, err := signer.GetKeyInfo(ctx)
    if err != nil {
        return fmt.Errorf("signer health check failed: %w", err)
    }

    if info.Algorithm != "ECDSA-P256" {
        return fmt.Errorf("unexpected algorithm: %s", info.Algorithm)
    }

    return nil
}

// HTTP health endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
    if err := checkVaultHealth(vaultClient); err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }

    if err := checkSignerHealth(vaultSigner); err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
        "vault":  "connected",
        "signer": "ready",
    })
}
```

### Performance Optimization

#### Cache Hit Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    cacheHits = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "vault_signer_cache_hits_total",
        Help: "Total number of cache hits for signing keys",
    })

    cacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "vault_signer_cache_misses_total",
        Help: "Total number of cache misses (fetched from Vault)",
    })
)

// Track cache performance
func trackCachePerformance(signer *vault.VaultSigner) {
    // Signing with cached key (~3ms)
    start := time.Now()
    sig, pubKey, err := signer.Sign(data)
    duration := time.Since(start)

    if duration < 10*time.Millisecond {
        cacheHits.Inc()
    } else {
        cacheMisses.Inc()
    }
}
```

#### Latency Benchmarks

Based on production testing:

| Operation | Latency (p50) | Latency (p99) | Notes |
|-----------|--------------|---------------|-------|
| **Sign (cached)** | 3ms | 5ms | Using cached key (99% of requests) |
| **Sign (uncached)** | 50ms | 100ms | Fetching from Vault (1% of requests) |
| **Get Secret** | 30ms | 80ms | Network + Vault processing |
| **Put Secret** | 40ms | 100ms | Write + replication |
| **Token Renewal** | 20ms | 60ms | Background operation |
| **Health Check** | 15ms | 40ms | HTTP GET to /v1/sys/health |

**Optimization Recommendations**:
- **Cache TTL**: 5-minute default provides 99% cache hit rate
- **Increase for high volume**: 15-minute TTL for >1000 req/sec
- **Decrease for security**: 1-minute TTL for highly sensitive environments
- **Connection pooling**: Vault client uses keep-alive connections
- **Batch operations**: Group multiple KV operations when possible

### Troubleshooting

#### Connection Issues

**Problem**: `connection refused` when connecting to Vault

**Solutions**:
```bash
# Check Vault server is running
vault status

# Verify VAULT_ADDR
echo $VAULT_ADDR

# Test connectivity
curl $VAULT_ADDR/v1/sys/health

# Check firewall rules (production)
telnet vault.example.com 8200
```

#### Authentication Failures

**Problem**: `permission denied` errors

**Solutions**:
```bash
# Verify token
vault token lookup

# Check token policies
vault token lookup -format=json | jq .data.policies

# Test policy permissions
vault kv get secret/audit/signing-key
vault kv put secret/audit/signing-key test=value

# Renew expired token
vault token renew
```

#### TLS Certificate Issues

**Problem**: `x509: certificate signed by unknown authority`

**Solutions**:
```bash
# Verify CA certificate
openssl x509 -in /etc/vault/ca.crt -text -noout

# Test TLS connection
openssl s_client -connect vault.example.com:8200 -CAfile /etc/vault/ca.crt

# Skip TLS verification (development only!)
export VAULT_SKIP_VERIFY=true
```

#### Key Not Found

**Problem**: `key not found at audit/signing-key`

**Solutions**:
```go
// Enable auto-generation
signer, err := client.NewSigner(ctx, vault.SignerConfig{
    KeyPath:      "audit/signing-key",
    Identity:     "system@specular.dev",
    AutoGenerate: true, // ← Enable this
})

// Or manually generate
signer, err := client.NewSigner(ctx, vault.SignerConfig{
    KeyPath:      "audit/signing-key",
    Identity:     "system@specular.dev",
    AutoGenerate: false,
})
if err != nil {
    // Generate manually
    err = signer.GenerateKey(ctx)
}
```

#### Signing Failures

**Problem**: Audit entries not being signed

**Solutions**:
```go
// Check signer health
info, err := signer.GetKeyInfo(ctx)
if err != nil {
    log.Printf("ERROR: signer unhealthy: %v", err)
}

// Clear cache and retry
signer.ClearCache()

// Verify signature works
testData := []byte("test")
sig, pubKey, err := signer.Sign(testData)
if err != nil {
    log.Printf("ERROR: signing failed: %v", err)
}

// Verify signature
valid, err := signer.VerifySignature(testData, sig, pubKey)
if !valid {
    log.Printf("ERROR: signature verification failed")
}
```

### Security Best Practices

#### 1. Token Management

```go
// ✅ Good: Use environment variables
token := os.Getenv("VAULT_TOKEN")

// ✅ Good: Use short-lived tokens
client, err := vault.NewClient(vault.Config{
    Address:  vaultAddr,
    Token:    token,
    TokenTTL: 1 * time.Hour, // Renew frequently
})

// ❌ Bad: Hard-coded tokens
client, err := vault.NewClient(vault.Config{
    Token: "s.AbCdEf123456", // Never do this!
})
```

#### 2. TLS Configuration

```go
// ✅ Good: Always use TLS in production
client, err := vault.NewClient(vault.Config{
    Address: "https://vault.example.com:8200", // HTTPS
    TLSConfig: &vault.TLSConfig{
        CACert:     "/etc/vault/ca.crt",
        ClientCert: "/etc/vault/client.crt",
        ClientKey:  "/etc/vault/client.key",
    },
})

// ❌ Bad: Insecure skip verify
client, err := vault.NewClient(vault.Config{
    TLSConfig: &vault.TLSConfig{
        InsecureSkipVerify: true, // Never in production!
    },
})
```

#### 3. Key Path Isolation

```go
// ✅ Good: Organization-specific paths
keyPath := fmt.Sprintf("audit/%s/signing-key", organizationID)

// ✅ Good: Environment separation
keyPath := fmt.Sprintf("audit/%s/%s/signing-key", environment, organizationID)

// ❌ Bad: Shared keys across organizations
keyPath := "audit/shared-key" // Security risk!
```

#### 4. Cache TTL Configuration

```go
// ✅ Good: Balance security and performance
// Development: 5 minutes (default)
// Production: 5-15 minutes
// High security: 1 minute
// High volume: 15 minutes

cacheTTL := 5 * time.Minute
if environment == "production" && isHighSecurity {
    cacheTTL = 1 * time.Minute
} else if isHighVolume {
    cacheTTL = 15 * time.Minute
}

// ❌ Bad: Excessive cache duration
cacheTTL := 24 * time.Hour // Too long!
```

#### 5. Error Handling

```go
// ✅ Good: Graceful degradation
sig, pubKey, err := signer.Sign(data)
if err != nil {
    // Log error but continue with unsigned entry
    log.Printf("WARNING: signing failed: %v", err)
    // Entry still logged, just unsigned
    return baseLogger.LogDecision(ctx, entry)
}

// ❌ Bad: Hard failure
sig, pubKey, err := signer.Sign(data)
if err != nil {
    return err // Audit trail lost!
}
```

#### 6. Monitoring and Alerting

```go
// ✅ Good: Monitor Vault operations
prometheus.MustRegister(
    vaultRequestsTotal,
    vaultRequestDuration,
    vaultSigningFailures,
    vaultCacheHits,
    vaultCacheMisses,
)

// Alert on:
// - High signing failure rate (>1%)
// - Low cache hit rate (<95%)
// - Vault connection failures
// - Token renewal failures
```

### Migration Guide

#### Migrating from Ephemeral Keys

**Current Setup** (ephemeral keys):
```go
signer, err := attestation.NewEphemeralSigner("system@specular.dev")
signedLogger := authz.NewSignedAuditLogger(baseLogger, signer)
```

**New Setup** (Vault-backed):
```go
// 1. Create Vault client
vaultClient, err := vault.NewClient(vault.Config{
    Address:   os.Getenv("VAULT_ADDR"),
    Token:     os.Getenv("VAULT_TOKEN"),
    MountPath: "secret",
})
if err != nil {
    log.Fatal(err)
}
defer vaultClient.Close()

// 2. Create Vault signer (auto-generates key)
vaultSigner, err := vaultClient.NewSigner(ctx, vault.SignerConfig{
    KeyPath:      "audit/signing-key",
    Identity:     "system@specular.dev",
    AutoGenerate: true,
    CacheTTL:     5 * time.Minute,
})
if err != nil {
    log.Fatal(err)
}

// 3. Use Vault signer (same interface as ephemeral)
signedLogger := authz.NewSignedAuditLogger(baseLogger, vaultSigner)
```

**Migration Steps**:
1. Deploy Vault server (or configure access to existing Vault)
2. Create Vault policy for audit key access
3. Generate Vault token with policy
4. Update application configuration to use Vault
5. Deploy updated application
6. Verify signatures with new public keys
7. Rotate keys quarterly going forward

## Best Practices

### 1. Principle of Least Privilege

Grant minimum necessary permissions:

```go
// ✅ Good: Specific permissions
policy := authz.NewResourceSpecificPolicy(
    "org-1", "build-executor",
    "Build Executors",
    "executor",
    "build:execute",
    "build",
)

// ❌ Bad: Too broad
policy := authz.NewPolicyBuilder("org-1", "Too Broad").
    AllowRole("executor").
    OnActions("*").
    OnAllResources().
    Build()
```

### 2. Use Deny Policies Sparingly

Explicit denies always win, so use them carefully:

```go
// ✅ Good: Specific denial for security
denyDelete := authz.NewDenyPolicy(
    "org-1", "no-delete-prod",
    "Prevent Production Deletion",
    []authz.Principal{{Role: "member"}},
    []string{"*:delete"},
    []string{"production:*"},
)

// ❌ Bad: Overly broad denial
denyEverything := authz.NewDenyPolicy(
    "org-1", "deny-all",
    "Deny Everything",
    []authz.Principal{{Role: "member"}},
    []string{"*"},
    []string{"*"},
)
```

### 3. Organize Policies by Purpose

```go
// Organizational policies
ownerPolicy := authz.NewOwnerPolicy("org-1", "org-owner")
adminPolicy := authz.NewAdminPolicy("org-1", "org-admin")

// Resource-specific policies
planApproval := authz.NewResourceSpecificPolicy(
    "org-1", "plan-approval", "Plan Approvals",
    "admin", "plan:approve", "plan",
)

// Conditional policies
pendingOnly := authz.NewConditionalPolicy(
    "org-1", "pending-only", "Approve Pending Only",
    "admin", []string{"plan:approve"}, "plan",
    authz.Condition{
        Attribute: "$resource.status",
        Operator:  authz.OperatorEquals,
        Value:     "pending",
    },
)
```

### 4. Test Policies with Simulation

Before deploying policies, test them:

```bash
curl -X POST http://localhost:8080/api/policies/simulate \
  -H "Content-Type: application/json" \
  -d '{
    "subject": {"UserID": "test-user", "OrganizationID": "org-1", "OrganizationRole": "member"},
    "action": "plan:delete",
    "resource": {"type": "plan", "id": "plan-123"}
  }'
```

### 5. Enable Audit Logging in Production

Always log authorization decisions in production:

```go
auditLogger := authz.NewDefaultAuditLogger(authz.AuditLoggerConfig{
    Writer:             productionLogWriter,
    LogAllDecisions:    true,
    IncludeEnvironment: true,
    BufferSize:         1000, // Async for performance
})
engine = authz.WithAuditLogger(engine, auditLogger)
```

### 6. Use Policy Conditions for Dynamic Rules

```go
// Time-based access
businessHours := authz.Condition{
    Attribute: "$environment.hour",
    Operator:  authz.OperatorIn,
    Value:     []int{9, 10, 11, 12, 13, 14, 15, 16, 17},
}

// IP-based access
internalNetwork := authz.Condition{
    Attribute: "$environment.client_ip",
    Operator:  authz.OperatorStartsWith,
    Value:     "192.168.",
}

// Resource state-based
notProduction := authz.Condition{
    Attribute: "$resource.environment",
    Operator:  authz.OperatorNotEquals,
    Value:     "production",
}
```

## Examples

### Example 1: Multi-Tier Organization

```go
// Owner - Full access
ownerPolicy := authz.NewOwnerPolicy("org-1", "owner-policy")
policyStore.CreatePolicy(ctx, ownerPolicy)

// Admin - Management access
adminPolicy := authz.NewAdminPolicy("org-1", "admin-policy")
policyStore.CreatePolicy(ctx, adminPolicy)

// Member - Standard access
memberPolicy := authz.NewMemberPolicy("org-1", "member-policy")
policyStore.CreatePolicy(ctx, memberPolicy)

// Viewer - Read-only
viewerPolicy := authz.NewViewerPolicy("org-1", "viewer-policy")
policyStore.CreatePolicy(ctx, viewerPolicy)
```

### Example 2: Department-Based Access

```go
// Engineering can create and approve plans
engineeringPolicy := authz.NewPolicyBuilder("org-1", "Engineering Policy").
    AllowAttribute("$subject.department", authz.OperatorEquals, "engineering").
    OnActions("plan:create", "plan:approve", "plan:read").
    OnResourceType("plan").
    Build()

// Finance can only view financial reports
financePolicy := authz.NewPolicyBuilder("org-1", "Finance Policy").
    AllowAttribute("$subject.department", authz.OperatorEquals, "finance").
    OnActions("report:read").
    OnResourceType("financial-report").
    Build()
```

### Example 3: Conditional Plan Approval

```go
// Only approve pending plans
pendingApproval := authz.NewConditionalPolicy(
    "org-1",
    "pending-approval",
    "Approve Pending Plans",
    "admin",
    []string{"plan:approve"},
    "plan",
    authz.Condition{
        Attribute: "$resource.status",
        Operator:  authz.OperatorEquals,
        Value:     "pending",
    },
)

// Cannot modify approved plans
denyApprovedMod := authz.NewPolicyBuilder("org-1", "Protect Approved Plans").
    WithEffect(authz.EffectDeny).
    AllowRole("member").
    OnActions("plan:update", "plan:delete").
    OnResourceType("plan").
    WithCondition("$resource.status", authz.OperatorEquals, "approved").
    Build()
```

### Example 4: Team-Based Build Access

```go
// Team leads can execute team builds
teamLeadPolicy := authz.NewTeamPolicy(
    "org-1",
    "team-lead-builds",
    "Team Lead Build Access",
    "lead",
    []string{"build:execute", "build:read", "build:cancel"},
    []string{"build:*"},
)

// Team members can only view builds
teamMemberPolicy := authz.NewTeamPolicy(
    "org-1",
    "team-member-builds",
    "Team Member Build Access",
    "member",
    []string{"build:read"},
    []string{"build:*"},
)
```

## API Reference

### Core Types

#### Policy
```go
type Policy struct {
    ID             string
    OrganizationID string
    Name           string
    Description    string
    Effect         Effect      // "allow" or "deny"
    Principals     []Principal
    Actions        []string
    Resources      []string
    Conditions     []Condition
    Enabled        bool
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

#### Principal
```go
type Principal struct {
    // Role-based
    Role  string // e.g., "admin", "member"
    Scope string // "organization" or "team"

    // Attribute-based
    Attribute string              // e.g., "$subject.department"
    Operator  ConditionOperator   // e.g., OperatorEquals
    Value     interface{}         // e.g., "engineering"
}
```

#### AuthorizationRequest
```go
type AuthorizationRequest struct {
    Subject     *auth.Session
    Action      string
    Resource    Resource
    Environment map[string]interface{}
}
```

#### Decision
```go
type Decision struct {
    Allowed   bool
    Reason    string
    PolicyIDs []string
    Timestamp time.Time
}
```

### Constants

#### Effects
```go
const (
    EffectAllow Effect = "allow"
    EffectDeny  Effect = "deny"
)
```

#### Actions
```go
const (
    ActionApprove = "approve"
    ActionCreate  = "create"
    ActionRead    = "read"
    ActionUpdate  = "update"
    ActionDelete  = "delete"
    ActionList    = "list"
    ActionExecute = "execute"
)
```

#### Roles
```go
const (
    RoleOwner  Role = "owner"
    RoleAdmin  Role = "admin"
    RoleMember Role = "member"
    RoleViewer Role = "viewer"
)
```

#### Operators
```go
const (
    OperatorEquals       ConditionOperator = "equals"
    OperatorNotEquals    ConditionOperator = "not_equals"
    OperatorIn           ConditionOperator = "in"
    OperatorNotIn        ConditionOperator = "not_in"
    OperatorGreaterThan  ConditionOperator = "greater_than"
    OperatorLessThan     ConditionOperator = "less_than"
    OperatorStartsWith   ConditionOperator = "starts_with"
    OperatorEndsWith     ConditionOperator = "ends_with"
    OperatorContains     ConditionOperator = "contains"
)
```

### Format Helpers

```go
// FormatAction formats an action with resource type prefix
func FormatAction(resourceType, action string) string
// Example: FormatAction("plan", ActionApprove) => "plan:approve"

// FormatResource formats a resource identifier
func FormatResource(resourceType, resourceID string) string
// Example: FormatResource("plan", "123") => "plan:123"
// Example: FormatResource("plan", "*") => "plan:*"
```

---

## See Also

- [ADR-0013: ABAC Authorization Engine](./adr/0013-abac-authorization-engine.md)
- [Authorization Tutorial](./tutorials/08-authorization.md)
- [Policy Management Tutorial](./tutorials/05-policy-management.md)
- [API Reference](./API_REFERENCE.md)

---

**Need Help?**
- [GitHub Issues](https://github.com/felixgeelhaar/specular/issues)
- [Documentation](https://docs.specular.dev)

## API Key Authentication

### Overview

API key authentication provides a secure, long-lived credential system for programmatic API access. Keys are stored in HashiCorp Vault with automatic rotation support.

### Quick Start

```go
import "github.com/felixgeelhaar/specular/internal/apikey"

// Initialize manager
manager, err := apikey.NewManager(apikey.Config{
    VaultClient: vaultClient,
    Prefix:      "sk_",              // Default
    TTL:         90 * 24 * time.Hour, // 90 days
})

// Create API key
key, err := manager.CreateKey(ctx, "org-123", "user-456", 
    "Production Key", []string{"read", "write"})
fmt.Println("API Key:", key.Secret) // sk_...

// Use in HTTP server
middleware := apikey.NewMiddleware(manager)
http.Handle("/api/resource", middleware.RequireAPIKey(handler))
```

### HTTP Authentication

API keys use Bearer token authentication:

```bash
curl -H "Authorization: Bearer sk_dGVzdC1zZWNyZXQtZXhhbXBsZQ" \
     -H "X-Organization-ID: org-123" \
     https://api.example.com/resource
```

### Automatic Rotation

```go
// Start rotation scheduler
scheduler, err := apikey.NewScheduler(apikey.SchedulerConfig{
    Manager:       manager,
    CheckInterval: 1 * time.Hour,      // Check hourly
    GracePeriod:   7 * 24 * time.Hour, // 7 day grace period
    RotationTTL:   7 * 24 * time.Hour, // Rotate 7 days before expiry
})

go scheduler.Start(ctx)
```

### Key Lifecycle

```
active → (rotate) → rotated → (grace period) → revoked → deleted
         ↓
      new active key
```

For complete details, see **ADR-0016: API Key Rotation Mechanism**.

