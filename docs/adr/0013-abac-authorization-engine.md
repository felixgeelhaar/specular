# ADR 0013: ABAC Authorization Engine

**Status**: Accepted
**Date**: 2025-11-20
**Decision Makers**: Engineering Leadership
**Stakeholders**: Engineering, Enterprise Customers, Security Team

## Context

### Current State

After implementing SSO/SAML integration (ADR-0012), Specular now has robust **authentication** (verifying user identity). However, **authorization** (determining what authenticated users can do) remains rudimentary:

**Current Authorization Limitations**:
- Binary access control (authenticated vs not authenticated)
- No resource-level permissions
- No organization/team hierarchy
- Hard-coded permission checks scattered throughout code
- No audit trail for authorization decisions
- Cannot support multi-tenancy requirements (ADR-0011)

**Example of Current Code**:
```go
// Simple binary check - user is either authenticated or not
func (h *Handler) HandleApprove(w http.ResponseWriter, r *http.Request) {
    session := auth.GetSession(r.Context())
    if session == nil {
        http.Error(w, "Unauthorized", 401)
        return
    }

    // No check if user has "approve" permission
    // No check if user belongs to the right organization
    // No check if user can approve THIS specific resource
    approval.Execute()
}
```

### Enterprise Requirements

From ADR-0011 (v2.0 Enterprise Readiness), the authorization system must support:

1. **Multi-Tenancy**:
   - Isolate resources by tenant/organization
   - Prevent cross-tenant data access
   - Tenant-specific policy overrides

2. **Role-Based Access Control (RBAC)**:
   - Standard roles: Owner, Admin, Member, Viewer
   - Custom roles per organization
   - Role inheritance and composition

3. **Resource-Level Permissions**:
   - Fine-grained control: read, write, approve, admin
   - Object-level permissions (e.g., "can approve plan #123")
   - Conditional permissions based on resource state

4. **Organizational Hierarchy**:
   ```
   Organization (tenant)
     └─ Team A
         ├─ User Alice (role: admin)
         └─ User Bob (role: member)
     └─ Team B
         └─ User Carol (role: viewer)
   ```

5. **Audit & Compliance**:
   - Log all authorization decisions (who attempted what)
   - Support SOC2/ISO 27001 audit requirements
   - Answer "who has access to X?" queries

6. **Performance**:
   - <10ms authorization decision latency
   - Cacheable authorization policies
   - Horizontal scalability for 10,000+ concurrent users

### Why ABAC vs RBAC

**Role-Based Access Control (RBAC)** assigns permissions to roles, and users to roles:
```
User → Role → Permissions
Alice → Admin → {approve, write, delete}
Bob → Viewer → {read}
```

**Limitations of RBAC**:
- Role explosion: Need separate roles for every permission combination
- Cannot model conditional permissions (e.g., "approve if value < $10k")
- Difficult to represent resource ownership (e.g., "only approve plans you created")
- Does not scale with multi-tenancy complexity

**Attribute-Based Access Control (ABAC)** evaluates policies based on attributes:
```
Decision = Evaluate(
    Subject Attributes: {user.role, user.team, user.tenant_id},
    Resource Attributes: {resource.type, resource.owner, resource.status},
    Action: "approve",
    Environment: {time, ip_address}
)
```

**ABAC Advantages**:
- ✅ Fine-grained control without role explosion
- ✅ Conditional permissions (e.g., "approve if amount < user.approval_limit")
- ✅ Natural multi-tenancy support (tenant_id attribute)
- ✅ Flexible policy language
- ✅ Future-proof for complex scenarios

**ABAC Challenges**:
- ❌ More complex to implement
- ❌ Policy authoring requires care
- ❌ Testing complexity (many attribute combinations)

**Decision**: Use **ABAC with RBAC fallback** — leverage ABAC for flexibility while providing RBAC-style roles as a user-friendly abstraction.

## Decision

We will implement a **production-grade ABAC authorization engine** with the following design:

### 1. Authorization Model

**Policy Structure** (inspired by AWS IAM and XACML):
```yaml
# Example policy: Team admins can approve plans in their organization
policies:
  - id: team-admin-approve-plans
    version: 1
    effect: allow  # allow | deny
    principals:
      - role: admin
        scope: team
    actions:
      - plan:approve
    resources:
      - plan:*
    conditions:
      - attribute: resource.tenant_id
        operator: equals
        value: $subject.tenant_id
      - attribute: resource.team_id
        operator: equals
        value: $subject.team_id
      - attribute: resource.status
        operator: in
        value: [pending_approval, drift_detected]
```

**Policy Evaluation Algorithm** (AWS IAM-style):
1. Default decision: **DENY** (fail-safe)
2. Evaluate all policies that match (principal, action, resource)
3. If any policy has `effect: deny` → **DENY** (explicit deny wins)
4. If any policy has `effect: allow` and all conditions pass → **ALLOW**
5. Otherwise → **DENY**

**Built-in Roles** (RBAC abstraction over ABAC):
```go
const (
    RoleOwner  = "owner"  // Full control, can manage members
    RoleAdmin  = "admin"  // Can approve, create, update, delete
    RoleMember = "member" // Can create, update (not approve or delete)
    RoleViewer = "viewer" // Read-only access
)
```

Each role maps to a set of ABAC policies stored in the system.

### 2. Architecture

**Component Diagram**:
```
┌─────────────────────────────────────────────────────┐
│                  HTTP Handler                       │
└───────────────────────┬─────────────────────────────┘
                        │
        ┌───────────────▼────────────────┐
        │   Authorization Middleware     │
        │   authz.RequirePermission()    │
        └───────────────┬────────────────┘
                        │
        ┌───────────────▼────────────────┐
        │    Authorization Engine        │
        │    - Policy Store              │
        │    - Attribute Resolver        │
        │    - Policy Evaluator          │
        └───────────────┬────────────────┘
                        │
        ┌───────────────▼────────────────┐
        │     Attribute Sources          │
        │  - Session (user attributes)   │
        │  - Database (resource attrs)   │
        │  - Cache (performance)         │
        └────────────────────────────────┘
```

**Core Components**:

1. **Policy Store**: Manages authorization policies
   ```go
   type PolicyStore interface {
       LoadPolicies(ctx context.Context, tenantID string) ([]*Policy, error)
       CreatePolicy(ctx context.Context, policy *Policy) error
       UpdatePolicy(ctx context.Context, policy *Policy) error
       DeletePolicy(ctx context.Context, policyID string) error
   }
   ```

2. **Attribute Resolver**: Fetches attributes for authorization decisions
   ```go
   type AttributeResolver interface {
       GetSubjectAttributes(ctx context.Context, subject *auth.Session) (Attributes, error)
       GetResourceAttributes(ctx context.Context, resourceType, resourceID string) (Attributes, error)
   }
   ```

3. **Policy Evaluator**: Core decision engine
   ```go
   type Evaluator interface {
       Evaluate(ctx context.Context, req *AuthorizationRequest) (*Decision, error)
   }

   type AuthorizationRequest struct {
       Subject     *auth.Session
       Action      string   // e.g., "plan:approve", "build:create"
       Resource    Resource // {Type: "plan", ID: "plan-123"}
       Environment map[string]interface{} // IP, time, etc.
   }

   type Decision struct {
       Allowed   bool
       Reason    string // Human-readable explanation
       PolicyIDs []string // Policies that contributed to decision
   }
   ```

4. **Authorization Middleware**: HTTP integration
   ```go
   func RequirePermission(action string, resourceType string) func(http.Handler) http.Handler
   ```

### 3. Policy Language

**JSON Policy Format** (programmer-friendly):
```json
{
  "id": "admin-approve-plans",
  "version": 1,
  "effect": "allow",
  "principals": [
    {"role": "admin", "scope": "organization"}
  ],
  "actions": ["plan:approve"],
  "resources": ["plan:*"],
  "conditions": [
    {
      "attribute": "resource.tenant_id",
      "operator": "equals",
      "value": "$subject.tenant_id"
    }
  ]
}
```

**Supported Operators**:
- `equals`, `not_equals`
- `in`, `not_in`
- `greater_than`, `less_than`, `greater_than_or_equals`, `less_than_or_equals`
- `string_like` (glob pattern matching)
- `exists`, `not_exists`

**Attribute References**:
- `$subject.*` - User/session attributes (e.g., `$subject.role`, `$subject.tenant_id`)
- `$resource.*` - Resource attributes (e.g., `$resource.status`, `$resource.owner_id`)
- `$environment.*` - Contextual attributes (e.g., `$environment.time`, `$environment.ip`)

### 4. Performance Optimizations

**Caching Strategy**:
```
┌─────────────┐
│  Request    │
└─────┬───────┘
      │
┌─────▼────────────────────────────┐
│  L1: Policy Cache (in-memory)    │
│  TTL: 5 minutes                   │
│  Key: tenant_id                   │
└─────┬────────────────────────────┘
      │ (miss)
┌─────▼────────────────────────────┐
│  L2: Redis Cache                 │
│  TTL: 15 minutes                  │
└─────┬────────────────────────────┘
      │ (miss)
┌─────▼────────────────────────────┐
│  Database (PostgreSQL)           │
└──────────────────────────────────┘
```

**Attribute Caching**:
- Cache user attributes (role, team, tenant) in session JWT
- Cache resource attributes in Redis (TTL: 1 minute)
- Invalidate cache on resource updates

**Pre-compiled Policies**:
- Compile policy conditions into Go functions at load time
- Avoid runtime parsing overhead

**Batch Evaluation** (future optimization):
```go
// Evaluate multiple actions at once
permissions := authz.EvaluateBatch(ctx, subject, []string{
    "plan:read",
    "plan:approve",
    "plan:delete",
}, resource)
```

### 5. Database Schema

**Organizations & Teams**:
```sql
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(63) UNIQUE NOT NULL,
    settings JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(63) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(organization_id, slug)
);

CREATE TABLE organization_members (
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL, -- From auth.Session.UserID
    role VARCHAR(50) NOT NULL, -- owner, admin, member, viewer
    team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (organization_id, user_id)
);

CREATE INDEX idx_org_members_user ON organization_members(user_id);
CREATE INDEX idx_org_members_team ON organization_members(team_id);
```

**Policies**:
```sql
CREATE TABLE authz_policies (
    id VARCHAR(255) PRIMARY KEY,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    effect VARCHAR(10) NOT NULL CHECK (effect IN ('allow', 'deny')),
    policy_document JSONB NOT NULL, -- Full policy JSON
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_policies_org ON authz_policies(organization_id);
CREATE INDEX idx_policies_enabled ON authz_policies(organization_id, enabled);
```

**Audit Log**:
```sql
CREATE TABLE authz_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    decision BOOLEAN NOT NULL, -- true = allowed, false = denied
    reason TEXT,
    policy_ids TEXT[], -- Array of policy IDs that contributed
    metadata JSONB, -- Subject/resource attributes, environment
    timestamp TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_authz_decisions_org ON authz_decisions(organization_id, timestamp DESC);
CREATE INDEX idx_authz_decisions_user ON authz_decisions(user_id, timestamp DESC);
CREATE INDEX idx_authz_decisions_resource ON authz_decisions(resource_type, resource_id, timestamp DESC);
```

### 6. Integration with Authentication

**Enhanced Session with Authorization Context**:
```go
type Session struct {
    // Existing fields from ADR-0012
    UserID    string
    Email     string
    Provider  string

    // New authorization fields
    OrganizationID string // Tenant/organization user belongs to
    OrganizationRole string // owner, admin, member, viewer
    TeamID         *string // Optional team membership
    TeamRole       *string // Team-specific role override

    // Cached permissions (for performance)
    Permissions []string // e.g., ["plan:read", "plan:create", "build:run"]
}
```

**Token Size Considerations**:
- JWT tokens should remain <4KB (cookie size limit)
- Store only essential attributes in token
- Fetch additional attributes from database when needed

### 7. Example Usage

**HTTP Middleware**:
```go
// Require specific permission
router.Handle("/plans/{id}/approve",
    authz.RequirePermission("plan:approve", "plan")(
        http.HandlerFunc(handleApprovePlan),
    ),
)

// Check permission programmatically
func handleApprovePlan(w http.ResponseWriter, r *http.Request) {
    planID := chi.URLParam(r, "id")

    decision, err := authz.Evaluate(r.Context(), &authz.AuthorizationRequest{
        Subject: auth.GetSession(r.Context()),
        Action: "plan:approve",
        Resource: authz.Resource{
            Type: "plan",
            ID: planID,
        },
    })

    if err != nil || !decision.Allowed {
        http.Error(w, "Forbidden: " + decision.Reason, 403)
        return
    }

    // Proceed with approval
}
```

**Policy Management API**:
```go
// Create custom policy
POST /v2/organizations/{org_id}/policies
{
  "name": "Senior engineers can approve large builds",
  "effect": "allow",
  "principals": [{"attribute": "subject.seniority", "value": "senior"}],
  "actions": ["build:approve"],
  "resources": ["build:*"],
  "conditions": [
    {"attribute": "resource.cost", "operator": "less_than", "value": 10000}
  ]
}
```

## Consequences

### Benefits

1. **Enterprise-Ready Authorization**:
   - Fine-grained permissions support complex organizational structures
   - Multi-tenancy isolation prevents cross-tenant data access
   - Conditional permissions enable sophisticated business rules
   - Audit logging satisfies SOC2/ISO 27001 requirements

2. **Developer Experience**:
   - Simple middleware for common cases: `RequirePermission("plan:approve", "plan")`
   - Declarative policy language (JSON/YAML)
   - Clear error messages explain why access was denied
   - Policy testing framework for validation

3. **Security**:
   - Default-deny ensures fail-safe behavior
   - Explicit deny overrides all allows (prevents privilege escalation)
   - Audit trail for all authorization decisions
   - Attribute-based policies reduce hard-coded logic

4. **Performance**:
   - Multi-tier caching achieves <10ms decision latency
   - Pre-compiled policies eliminate runtime parsing
   - Batch evaluation optimizes bulk operations
   - Horizontal scaling via stateless design

5. **Flexibility**:
   - ABAC supports future requirements without code changes
   - Custom roles via policy composition
   - Dynamic attribute resolution (e.g., fetch resource state)
   - Environment-based decisions (time, IP, etc.)

### Trade-offs

1. **Complexity**:
   - ABAC is more complex than simple RBAC
   - Policy authoring requires understanding attribute model
   - Debugging requires examining attribute values
   - **Mitigation**: Provide RBAC-style role templates, policy testing tools

2. **Performance Overhead**:
   - Attribute resolution requires database queries
   - Policy evaluation adds latency to every request
   - **Mitigation**: Aggressive caching, pre-compiled policies, attribute embedding in JWT

3. **Testing Burden**:
   - Need to test many attribute combinations
   - Policy conflicts must be detected
   - **Mitigation**: Policy simulation tool, automated conflict detection

4. **Migration**:
   - Existing code has ad-hoc permission checks
   - Need to migrate to unified authorization model
   - **Mitigation**: Gradual migration, backward compatibility layer

### Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Policy misconfiguration** | Security vulnerability, access denied | Policy validation on create/update, automated tests, dry-run mode |
| **Performance degradation** | Poor user experience, timeouts | Caching at multiple layers, performance benchmarks, load testing |
| **Attribute resolution failures** | Authorization errors, availability impact | Graceful degradation, cached attributes, circuit breaker |
| **Policy conflicts** | Unpredictable authorization behavior | Conflict detection tool, policy simulation, explicit deny precedence |
| **Audit log size** | Storage costs, query performance | Log retention policies, archival to S3, sampling for high-volume |

## Implementation Plan

### Phase 1: Core Engine (Week 1-2)

**Tasks**:
1. **Data Models & Interfaces**:
   - Define `Policy`, `AuthorizationRequest`, `Decision` types
   - Implement `PolicyStore`, `AttributeResolver`, `Evaluator` interfaces
   - Create database schema (organizations, teams, members, policies, audit)

2. **Policy Evaluator**:
   - Implement policy matching logic (principal, action, resource)
   - Implement condition evaluation (operators, attribute references)
   - Implement decision algorithm (deny precedence, default deny)

3. **Attribute Resolution**:
   - Extract attributes from `auth.Session`
   - Fetch resource attributes from database
   - Environment attributes (time, IP address)

**Acceptance Criteria**:
- Unit tests for policy evaluation (20+ test cases)
- Benchmarks showing <5ms evaluation time (uncached)
- Policy conflict detection works

### Phase 2: HTTP Integration (Week 2-3)

**Tasks**:
1. **Authorization Middleware**:
   - `RequirePermission(action, resourceType)` middleware
   - Extract resource ID from URL/body
   - Return 403 Forbidden with reason on denial

2. **Session Enhancement**:
   - Add `OrganizationID`, `Role`, `TeamID` to `auth.Session`
   - Update `SessionManager` to include authorization attributes
   - Embed basic permissions in JWT for performance

3. **Audit Logging**:
   - Log all authorization decisions to `authz_decisions` table
   - Include subject attributes, resource attributes, decision
   - Provide query API for audit reports

**Acceptance Criteria**:
- Middleware protects all critical endpoints
- Audit log captures 100% of authorization decisions
- Integration tests with real HTTP requests

### Phase 3: Policy Management (Week 3-4)

**Tasks**:
1. **Policy CRUD APIs**:
   - `POST /v2/organizations/{org_id}/policies` - Create policy
   - `GET /v2/organizations/{org_id}/policies` - List policies
   - `PUT /v2/organizations/{org_id}/policies/{id}` - Update policy
   - `DELETE /v2/organizations/{org_id}/policies/{id}` - Delete policy

2. **Built-in Roles**:
   - Create default policies for Owner, Admin, Member, Viewer
   - Auto-assign policies when user joins organization
   - Allow custom role creation via policy composition

3. **Policy Testing Tool**:
   - `authz.Simulate(policy, request)` - Test policy without persisting
   - Web UI for policy simulation
   - Conflict detection: `authz.DetectConflicts(policies)`

**Acceptance Criteria**:
- Policy CRUD APIs fully functional
- Built-in roles cover 90% of use cases
- Policy simulation tool validates policies before deployment

### Phase 4: Performance & Scale (Week 4-5)

**Tasks**:
1. **Caching**:
   - In-memory policy cache (5min TTL)
   - Redis attribute cache (1min TTL)
   - Cache invalidation on policy/resource updates

2. **Optimization**:
   - Pre-compile policies at load time
   - Batch attribute resolution
   - Lazy attribute loading (fetch only if needed)

3. **Load Testing**:
   - Benchmark: 10,000 requests/sec with <10ms p95 latency
   - Test tenant isolation (no cross-tenant leaks)
   - Test cache eviction and repopulation

**Acceptance Criteria**:
- <10ms p95 authorization latency
- Cache hit rate >95%
- Handles 10,000 concurrent users

### Phase 5: Documentation & Migration (Week 5-6)

**Tasks**:
1. **Documentation**:
   - User guide: Creating policies, managing roles
   - Developer guide: Using authorization middleware
   - Policy examples: Common scenarios
   - Troubleshooting guide

2. **Migration from Ad-hoc Checks**:
   - Identify all hard-coded permission checks
   - Replace with `authz.RequirePermission()` or `authz.Evaluate()`
   - Update tests

3. **Enterprise Demo**:
   - Multi-tenant demo with 3 organizations
   - Show cross-tenant isolation
   - Demonstrate policy customization

**Acceptance Criteria**:
- Comprehensive documentation published
- All ad-hoc permission checks migrated
- Demo showcases enterprise features

## Success Metrics

**Technical Metrics**:
- <10ms p95 authorization decision latency
- >95% cache hit rate for policies and attributes
- Zero cross-tenant authorization bypasses
- 100% audit coverage for authorization decisions

**Business Metrics**:
- Support 1,000+ organizations (multi-tenant)
- Enable custom roles for 50+ enterprise customers
- Reduce authorization-related support tickets by 80%
- Pass SOC2 Type II audit (authorization controls)

**Developer Metrics**:
- Authorization check in <5 lines of code
- Policy creation without code changes
- Policy simulation prevents misconfigurations
- Audit queries answer "who has access?" in <1 second

## Related ADRs

- **ADR-0011**: v2.0 Architecture - Enterprise Readiness (authorization requirements)
- **ADR-0012**: SSO/SAML Integration (authentication foundation)
- **ADR-0009**: Observability & Monitoring Strategy (audit logging)
- **ADR-0010**: Governance-First CLI Redesign (policy enforcement)

## References

- [AWS IAM Policy Evaluation Logic](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_evaluation-logic.html)
- [XACML (eXtensible Access Control Markup Language)](http://docs.oasis-open.org/xacml/3.0/xacml-3.0-core-spec-os-en.html)
- [Google Zanzibar](https://research.google/pubs/pub48190/) - Global authorization system
- [NIST ABAC Guide](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-162.pdf)
- [Oso Authorization Library](https://www.osohq.com/docs) - Modern ABAC implementation

---

**Decision Outcome**: **Accepted**

ABAC authorization engine provides the flexibility and granularity required for enterprise multi-tenancy while maintaining performance and developer experience. The policy-based approach future-proofs Specular for complex authorization scenarios without requiring code changes.

The 5-week implementation plan delivers incrementally: core engine → HTTP integration → policy management → performance optimization → documentation. This enables early validation and iteration.

**Next Steps**:
1. Create feature branch: `feature/m9.2.2-abac-authorization`
2. Implement Phase 1: Core Engine (data models, evaluator, attribute resolver)
3. Write comprehensive tests (policy evaluation, condition operators)
4. Create Phase 2: HTTP middleware integration

**Document Owners**: Engineering Leadership
**Review Cycle**: After Phase 2 (HTTP integration complete)
