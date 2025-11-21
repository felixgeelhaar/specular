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
