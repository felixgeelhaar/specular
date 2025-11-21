# Authorization: Access Control for Your Application

This tutorial covers implementing attribute-based access control (ABAC) to secure your Specular application.

> **Note**: Authorization features are available in all Specular editions.

## Overview

Authorization controls:
- Who can perform actions (role-based)
- What operations are allowed (action-based)
- Which resources can be accessed (resource-based)
- When and where access is granted (condition-based)

---

## Prerequisites

- Basic understanding of authentication
- Familiarity with HTTP middleware concepts

---

## Step 1: Understanding Authorization Concepts

### The Authorization Model

Specular uses **Attribute-Based Access Control (ABAC)**, which evaluates:

```
Can [Subject] perform [Action] on [Resource]?

Examples:
- Can admin approve plan-123?
- Can engineering@example.com create builds?
- Can viewer list projects?
```

### Built-in Roles

| Role | Permissions | Use Case |
|------|-------------|----------|
| **Owner** | All operations (*) | Organization owners |
| **Admin** | approve, create, update, delete, read | Team leads, managers |
| **Member** | create, update, read | Developers, contributors |
| **Viewer** | read, list | Auditors, stakeholders |

### Evaluation Logic

```
┌─────────────────────────────────┐
│ 1. Any DENY policy matches?    │
│    └─ Yes → DENY (stop)        │
└─────────────────────────────────┘
              ↓
┌─────────────────────────────────┐
│ 2. Any ALLOW policy matches?   │
│    └─ Yes → ALLOW              │
│    └─ No  → DENY (default)     │
└─────────────────────────────────┘
```

---

## Step 2: Create Basic Policies

### Initialize Authorization

Create a simple authorization setup:

```go
package main

import (
    "context"
    "github.com/felixgeelhaar/specular/internal/authz"
)

func main() {
    // 1. Create policy store
    policyStore := authz.NewInMemoryPolicyStore()

    // 2. Create attribute resolver
    resourceStore := authz.NewInMemoryResourceStore()
    resolver := authz.NewDefaultAttributeResolver(resourceStore)

    // 3. Create authorization engine
    engine := authz.NewEngine(policyStore, resolver)

    ctx := context.Background()

    // 4. Add built-in role policies
    ownerPolicy := authz.NewOwnerPolicy("org-1", "owner-policy")
    adminPolicy := authz.NewAdminPolicy("org-1", "admin-policy")
    memberPolicy := authz.NewMemberPolicy("org-1", "member-policy")
    viewerPolicy := authz.NewViewerPolicy("org-1", "viewer-policy")

    policyStore.CreatePolicy(ctx, ownerPolicy)
    policyStore.CreatePolicy(ctx, adminPolicy)
    policyStore.CreatePolicy(ctx, memberPolicy)
    policyStore.CreatePolicy(ctx, viewerPolicy)
}
```

**What this does:**
- Sets up in-memory policy storage (production would use database)
- Creates attribute resolver for dynamic policy conditions
- Initializes authorization engine
- Adds standard role-based policies

---

## Step 3: Test Authorization Decisions

### Check Admin Approval

```go
import "github.com/felixgeelhaar/specular/internal/auth"

// Create session for admin user
session := &auth.Session{
    UserID:           "user-123",
    Email:            "admin@example.com",
    OrganizationID:   "org-1",
    OrganizationRole: "admin",
}

// Check if admin can approve plan
req := &authz.AuthorizationRequest{
    Subject: session,
    Action:  "plan:approve",
    Resource: authz.Resource{
        Type: "plan",
        ID:   "plan-123",
    },
}

decision, err := engine.Evaluate(ctx, req)
if err != nil {
    panic(err)
}

if decision.Allowed {
    fmt.Printf("✅ Access granted: %s\n", decision.Reason)
    // Output: ✅ Access granted: access granted by policy admin-policy
} else {
    fmt.Printf("❌ Access denied: %s\n", decision.Reason)
}
```

### Check Member Delete (Should Deny)

```go
// Create session for member user
memberSession := &auth.Session{
    UserID:           "user-456",
    Email:            "member@example.com",
    OrganizationID:   "org-1",
    OrganizationRole: "member",
}

// Check if member can delete plan
req = &authz.AuthorizationRequest{
    Subject: memberSession,
    Action:  "plan:delete",
    Resource: authz.Resource{
        Type: "plan",
        ID:   "plan-123",
    },
}

decision, err = engine.Evaluate(ctx, req)

// Output: ❌ Access denied: no matching allow policy
```

---

## Step 4: Create Custom Policies

### Resource-Specific Policy

Grant specific permissions for specific resource types:

```go
// Engineering team can approve plans
engineeringPolicy := authz.NewResourceSpecificPolicy(
    "org-1",
    "engineering-approve",
    "Engineering Approves Plans",
    "engineering",
    "plan:approve",
    "plan",
)

policyStore.CreatePolicy(ctx, engineeringPolicy)
```

### Conditional Policy

Add attribute-based conditions:

```go
// Can only approve pending plans
pendingOnly := authz.NewConditionalPolicy(
    "org-1",
    "approve-pending",
    "Approve Pending Plans Only",
    "admin",
    []string{"plan:approve"},
    "plan",
    authz.Condition{
        Attribute: "$resource.status",
        Operator:  authz.OperatorEquals,
        Value:     "pending",
    },
)

policyStore.CreatePolicy(ctx, pendingOnly)

// Register resource attributes so conditions can be evaluated
resourceStore.SetResourceAttributes(ctx, authz.Resource{
    Type: "plan",
    ID:   "plan-123",
}, map[string]interface{}{
    "status": "pending",
    "owner":  "user-123",
})
```

### Deny Policy

Explicit denials (always win):

```go
// Prevent members from deleting production resources
noDeleteProd := authz.NewPolicyBuilder("org-1", "No Delete Production").
    WithID("no-delete-prod").
    WithEffect(authz.EffectDeny).
    AllowRole("member").
    OnActions("*:delete").
    OnResourceType("production").
    WithCondition("$resource.environment", authz.OperatorEquals, "production").
    Build()

policyStore.CreatePolicy(ctx, noDeleteProd)
```

---

## Step 5: Integrate with HTTP Server

### Add Authorization Middleware

```go
import "net/http"

func main() {
    // ... setup engine as before ...

    // Create authorization middleware
    mw := authz.NewAuthorizationMiddleware(engine)

    // Setup HTTP routes
    mux := http.NewServeMux()

    // Require "plan:approve" permission
    mux.Handle("/api/plans/approve",
        mw.RequirePermission("plan:approve",
            http.HandlerFunc(approvePlanHandler),
        ),
    )

    // Require "plan:create" OR "plan:update"
    mux.Handle("/api/plans",
        mw.RequireAnyPermission(
            []string{"plan:create", "plan:update"},
            http.HandlerFunc(managePlansHandler),
        ),
    )

    // Public endpoint (no authorization)
    mux.HandleFunc("/api/health", healthHandler)

    http.ListenAndServe(":8080", mux)
}

func approvePlanHandler(w http.ResponseWriter, r *http.Request) {
    // This only runs if user has "plan:approve" permission
    planID := r.URL.Query().Get("plan_id")

    // ... approve plan logic ...

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Plan approved"))
}
```

### Automatic Session Extraction

The middleware automatically extracts the authenticated session from request context:

```go
// In your authentication middleware
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get session from JWT, cookie, etc.
        session := getSessionFromJWT(r)

        // Add to request context
        ctx := authz.SetSessionInContext(r.Context(), session)
        r = r.WithContext(ctx)

        // Authorization middleware will use this session
        next.ServeHTTP(w, r)
    })
}
```

---

## Step 6: Enable Audit Logging

### Setup Audit Logger

Track all authorization decisions:

```go
import "os"

// Create audit logger
auditLogger := authz.NewDefaultAuditLogger(authz.AuditLoggerConfig{
    Writer:             os.Stdout,  // or file, database, etc.
    LogAllDecisions:    true,        // Log both allow and deny
    IncludeEnvironment: true,        // Include request context
    BufferSize:         100,         // Async buffer (0 = sync)
})

// Attach to engine
engine = authz.WithAuditLogger(engine, auditLogger)

// Graceful shutdown
defer auditLogger.Close()
```

### Audit Log Output

```json
{
  "timestamp": "2024-01-15T10:00:00Z",
  "allowed": true,
  "reason": "access granted by policy admin-policy",
  "user_id": "user-123",
  "email": "admin@example.com",
  "organization_id": "org-1",
  "role": "admin",
  "action": "plan:approve",
  "resource_type": "plan",
  "resource_id": "plan-123",
  "policy_ids": ["admin-policy"],
  "environment": {
    "client_ip": "192.168.1.1",
    "user_agent": "Mozilla/5.0..."
  },
  "duration_ms": 5
}
```

---

## Step 7: Manage Policies via API

### Setup Policy Management Endpoints

```go
// Create policy handlers
policyHandlers := authz.NewPolicyHandlers(policyStore, engine)

// Register routes
policyHandlers.RegisterRoutes(mux)

// Routes added:
// POST   /api/policies          - Create policy
// GET    /api/policies/:id      - Get policy
// PUT    /api/policies/:id      - Update policy
// DELETE /api/policies/:id      - Delete policy
// POST   /api/policies/simulate - Test authorization
```

### Create Policy via API

```bash
curl -X POST http://localhost:8080/api/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Team Lead Build Access",
    "description": "Team leads can execute builds",
    "effect": "allow",
    "principals": [
      {"role": "lead", "scope": "team"}
    ],
    "actions": ["build:execute"],
    "resources": ["build:*"],
    "enabled": true
  }'
```

### Test Policy with Simulation

Before deploying a policy, test it:

```bash
curl -X POST http://localhost:8080/api/policies/simulate \
  -H "Content-Type: application/json" \
  -d '{
    "subject": {
      "UserID": "user-123",
      "OrganizationID": "org-1",
      "OrganizationRole": "member"
    },
    "action": "plan:delete",
    "resource": {
      "type": "plan",
      "id": "plan-123"
    }
  }'

# Response:
# {
#   "allowed": false,
#   "reason": "no matching allow policy",
#   "policy_ids": [],
#   "timestamp": "2024-01-15T10:00:00Z"
# }
```

---

## Step 8: Advanced Use Cases

### Department-Based Access

```go
// Engineering department can create and approve
engineeringPolicy := authz.NewPolicyBuilder("org-1", "Engineering Policy").
    AllowAttribute("$subject.department", authz.OperatorEquals, "engineering").
    OnActions("plan:create", "plan:approve", "plan:read").
    OnResourceType("plan").
    Build()

policyStore.CreatePolicy(ctx, engineeringPolicy)
```

### Time-Based Access

```go
// Only allow during business hours
businessHours := authz.Condition{
    Attribute: "$environment.hour",
    Operator:  authz.OperatorIn,
    Value:     []int{9, 10, 11, 12, 13, 14, 15, 16, 17},
}

timePolicy := authz.NewPolicyBuilder("org-1", "Business Hours Only").
    AllowRole("member").
    OnActions("*:delete").
    OnAllResources().
    WithCondition(businessHours.Attribute, businessHours.Operator, businessHours.Value).
    Build()
```

### Owner-Based Access

```go
// Users can only update their own resources
ownerOnly := authz.NewPolicyBuilder("org-1", "Owner Only Update").
    AllowRole("member").
    OnActions("plan:update").
    OnResourceType("plan").
    WithCondition("$resource.owner", authz.OperatorEquals, "$subject.user_id").
    Build()
```

---

## Best Practices

### 1. Start with Built-in Roles

Use standard roles for most use cases:

```go
✅ Good: Use built-in helpers
ownerPolicy := authz.NewOwnerPolicy("org-1", "owner-policy")
adminPolicy := authz.NewAdminPolicy("org-1", "admin-policy")

❌ Bad: Manually construct everything
policy := &authz.Policy{
    ID:             "owner-policy",
    OrganizationID: "org-1",
    // ... 20 lines of boilerplate ...
}
```

### 2. Use Principle of Least Privilege

Grant minimum necessary permissions:

```go
✅ Good: Specific permissions
authz.NewResourceSpecificPolicy(
    "org-1", "build-exec", "Build Executors",
    "executor", "build:execute", "build",
)

❌ Bad: Overly broad permissions
authz.NewPolicyBuilder("org-1", "Too Broad").
    AllowRole("executor").
    OnActions("*").
    OnAllResources().
    Build()
```

### 3. Test Policies Before Deployment

Use simulation endpoint:

```go
✅ Good: Test first
curl -X POST /api/policies/simulate -d '{"subject": {...}, "action": "plan:delete"}'

❌ Bad: Deploy untested
policyStore.CreatePolicy(ctx, newPolicy) // Hope it works!
```

### 4. Enable Audit Logging

Always log authorization decisions in production:

```go
✅ Good: Full audit trail
auditLogger := authz.NewDefaultAuditLogger(authz.AuditLoggerConfig{
    Writer:          productionLogger,
    LogAllDecisions: true,
    BufferSize:      1000,
})

❌ Bad: No logging
engine := authz.NewEngine(store, resolver) // No audit trail
```

### 5. Use Deny Policies Sparingly

Explicit denies always win:

```go
✅ Good: Specific security denial
authz.NewDenyPolicy(
    "org-1", "no-delete-prod", "Prevent Production Deletion",
    principals, []string{"*:delete"}, []string{"production:*"},
)

❌ Bad: Overly broad denial
authz.NewDenyPolicy(
    "org-1", "deny-everything", "Block Everything",
    principals, []string{"*"}, []string{"*"},
)
```

---

## Troubleshooting

### Access Unexpectedly Denied

**Check audit logs** to see why:

```bash
# Look for denial reason
cat audit.log | jq 'select(.allowed == false)'

# Common reasons:
# - "no matching allow policy" → Need to add allow policy
# - "access denied by policy X" → Explicit deny policy matched
```

**Verify policy is enabled**:

```go
policy, _ := policyStore.GetPolicy(ctx, "policy-id")
if !policy.Enabled {
    fmt.Println("Policy is disabled!")
}
```

### Policy Not Matching

**Check principal attributes**:

```go
// Ensure session has correct organization
session := &auth.Session{
    UserID:           "user-123",
    OrganizationID:   "org-1",  // ← Must match policy's OrganizationID
    OrganizationRole: "admin",  // ← Must match policy's role
}
```

**Verify action format**:

```go
✅ Good: Correct format
Action: "plan:approve"
Action: "build:execute"

❌ Bad: Wrong format
Action: "approve-plan"  // Should be "plan:approve"
Action: "PLAN:APPROVE"  // Should be lowercase
```

### Conditions Not Evaluating

**Ensure resource attributes are registered**:

```go
// BEFORE authorization check
resourceStore.SetResourceAttributes(ctx, authz.Resource{
    Type: "plan",
    ID:   "plan-123",
}, map[string]interface{}{
    "status": "pending",  // ← Needed for $resource.status conditions
    "owner":  "user-123", // ← Needed for $resource.owner conditions
})

// NOW authorization check will work
decision, err := engine.Evaluate(ctx, req)
```

---

## What You Learned

- ✅ Set up ABAC authorization with built-in roles
- ✅ Create custom policies with conditions
- ✅ Protect HTTP endpoints with middleware
- ✅ Enable audit logging for compliance
- ✅ Manage policies via REST API
- ✅ Test policies with simulation
- ✅ Implement advanced use cases

---

## Next Steps

- Read [Authorization Guide](../AUTHORIZATION_GUIDE.md) for complete API reference
- See [ADR-0013](../adr/0013-abac-authorization-engine.md) for architecture details
- Explore [Policy Management](./05-policy-management.md) for governance workflows
- Check [Production Guide](../PRODUCTION_GUIDE.md) for deployment best practices

---

## Additional Resources

- [Authorization Guide](../AUTHORIZATION_GUIDE.md) - Complete reference
- [API Reference](../API_REFERENCE.md) - HTTP endpoints
- [Example Projects](../../examples/projects/) - Full examples
- [GitHub Issues](https://github.com/felixgeelhaar/specular/issues) - Get help
