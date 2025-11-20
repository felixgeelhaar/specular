# ADR 0011: v2.0 Architecture - Enterprise Readiness

**Status**: Accepted
**Date**: 2025-11-20
**Decision Makers**: Product & Engineering Leadership
**Stakeholders**: Engineering, Enterprise Customers, Community

## Context

### Current State (v1.6.0)

Specular v1.6.0 represents a mature beta product with:

- **Core Features**: Spec-first development, policy enforcement, drift detection, autonomous mode
- **CI/CD Integration**: GitHub Action with SARIF support
- **Plugin System**: Extensible architecture with 5 plugin types
- **Governance**: Policy management, approval workflows, cryptographic attestations
- **Distribution**: Multi-platform binaries, Homebrew, GitHub releases

**Production Limitations**:
- Single-tenant architecture
- Limited high availability support
- No zero-downtime deployment guarantees
- Basic observability (logging only)
- No multi-region support
- Missing enterprise authentication (SSO/SAML)
- No compliance certifications (SOC2, ISO 27001)
- Limited horizontal scalability

### Strategic Requirements for v2.0

Based on early adopter feedback and enterprise requirements:

1. **Enterprise Adoption Blockers**:
   - Lack of SOC2/ISO 27001 certification
   - No SSO/SAML integration
   - Insufficient audit logging for compliance
   - No multi-tenancy for SaaS deployment
   - Missing disaster recovery capabilities

2. **Scale Requirements**:
   - Support 10,000+ concurrent users per instance
   - Handle 1,000+ organizations (multi-tenant)
   - Process 100,000+ builds per day
   - Maintain <100ms p95 response time
   - Achieve 99.9% uptime SLA

3. **Operational Excellence**:
   - Zero-downtime deployments
   - Comprehensive observability (metrics, traces, logs)
   - Automated disaster recovery
   - Chaos engineering validation
   - Multi-region deployment support

### Version Bump Rationale

v2.0 represents a **major version** due to:

1. **Breaking API Changes**:
   - Multi-tenancy requires organization/tenant context in all requests
   - Database schema changes (tenant isolation)
   - Configuration format evolution (YAML standardization)
   - CLI command adjustments for tenant awareness

2. **Architectural Evolution**:
   - Stateless microservices architecture
   - Event-driven patterns (CQRS, Event Sourcing)
   - Distributed state management
   - API Gateway for centralized routing

3. **Migration Complexity**:
   - Data migration from single-tenant to multi-tenant schema
   - Configuration migration (v1 → v2 format)
   - Deployment model changes (single instance → clustered)

## Decision

We will implement **v2.0 as a production-hardened, enterprise-ready platform** with the following architectural pillars:

### 1. Multi-Tenancy Architecture

**Pattern**: Database-per-tenant with optional schema-per-tenant

```
┌─────────────────────────────────────────┐
│          API Gateway Layer              │
│  (Authentication, Routing, Rate Limit)  │
└─────────────────────────────────────────┘
                    │
        ┌───────────┴───────────┐
        │                       │
┌───────▼──────┐      ┌────────▼─────┐
│  Tenant A    │      │   Tenant B   │
│  Database    │      │   Database   │
│  + Redis     │      │   + Redis    │
└──────────────┘      └──────────────┘
```

**Key Design Decisions**:

- **Tenant Isolation**: Physical database separation for data security
- **Tenant Context**: JWT tokens carry `tenant_id` for request routing
- **Resource Quotas**: Per-tenant CPU, memory, storage, API rate limits
- **Custom Policies**: Tenant-specific policy configurations and overrides
- **Billing Integration**: Usage tracking per tenant for metering

**Schema Changes**:
```sql
-- All tables gain tenant_id for schema-per-tenant fallback
ALTER TABLE specs ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE plans ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE bundles ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE approvals ADD COLUMN tenant_id UUID NOT NULL;

-- New tables for multi-tenancy
CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(63) UNIQUE NOT NULL,
    plan VARCHAR(50) NOT NULL, -- free, pro, enterprise
    settings JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE tenant_users (
    tenant_id UUID REFERENCES tenants(id),
    user_id UUID REFERENCES users(id),
    role VARCHAR(50) NOT NULL, -- owner, admin, member, viewer
    PRIMARY KEY (tenant_id, user_id)
);
```

### 2. High Availability & Zero-Downtime Deployments

**Pattern**: Rolling updates with health checks and connection draining

```
┌──────────────────────────────────────────┐
│         Load Balancer (L7)               │
│   (Health Checks, Sticky Sessions)       │
└──────────────────────────────────────────┘
         │              │              │
    ┌────▼───┐     ┌───▼────┐    ┌───▼────┐
    │ Pod 1  │     │ Pod 2  │    │ Pod 3  │
    │ v2.0.1 │     │ v2.0.0 │    │ v2.0.0 │
    └────────┘     └────────┘    └────────┘
     (New)         (Draining)     (Active)
```

**Implementation**:

1. **Graceful Shutdown**:
   ```go
   // Capture SIGTERM and drain connections
   func (s *Server) Shutdown(ctx context.Context) error {
       s.inShutdown.Store(true)

       // Stop accepting new requests
       s.httpServer.SetKeepAlivesEnabled(false)

       // Wait for active requests to complete (max 30s)
       ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
       defer cancel()

       return s.httpServer.Shutdown(ctx)
   }
   ```

2. **Health Check Endpoints**:
   - `GET /health/live` - Liveness probe (process alive)
   - `GET /health/ready` - Readiness probe (accepting traffic)
   - `GET /health/startup` - Startup probe (initialization complete)

   ```go
   type HealthStatus struct {
       Status     string            `json:"status"` // ok, degraded, error
       Version    string            `json:"version"`
       Checks     map[string]Check  `json:"checks"`
       Timestamp  time.Time         `json:"timestamp"`
   }

   type Check struct {
       Status  string `json:"status"`
       Message string `json:"message,omitempty"`
   }
   ```

3. **Connection Draining**:
   - Mark instance as "draining" when SIGTERM received
   - Readiness probe returns 503 to remove from load balancer
   - Complete in-flight requests before shutdown
   - Timeout after 30 seconds (force close remaining connections)

### 3. Enterprise Security

**Authentication**:
- SSO/SAML 2.0 integration (Okta, Auth0, Azure AD)
- OAuth 2.0 / OpenID Connect support
- API key management with rotation
- JWT with refresh tokens (15min access, 7day refresh)

**Authorization**:
- Attribute-Based Access Control (ABAC)
- Resource-level permissions (read, write, approve, admin)
- Organization → Team → User hierarchy
- Policy inheritance with overrides

**Audit Trail**:
```go
type AuditEvent struct {
    ID          string          `json:"id"`
    TenantID    string          `json:"tenant_id"`
    UserID      string          `json:"user_id"`
    Action      string          `json:"action"` // policy.approve, build.run, etc
    Resource    string          `json:"resource"`
    Metadata    json.RawMessage `json:"metadata"`
    IPAddress   string          `json:"ip_address"`
    UserAgent   string          `json:"user_agent"`
    Result      string          `json:"result"` // success, failure
    Timestamp   time.Time       `json:"timestamp"`
    Signature   string          `json:"signature"` // ECDSA signature
}
```

**Secrets Management**:
- HashiCorp Vault integration
- AWS Secrets Manager support
- Azure Key Vault support
- Encrypted configuration at rest
- Automatic secret rotation

### 4. Observability & Monitoring

**Three Pillars**:

1. **Metrics** (Prometheus):
   ```go
   // Custom business metrics
   specular_builds_total{tenant, status}
   specular_drift_violations_total{tenant, severity}
   specular_policy_checks_total{tenant, result}

   // Performance metrics
   http_request_duration_seconds{method, path, status}
   database_query_duration_seconds{operation}
   cache_hit_ratio{cache_type}
   ```

2. **Distributed Tracing** (OpenTelemetry + Jaeger):
   ```go
   // Trace context propagation
   ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

   // Span creation
   ctx, span := tracer.Start(ctx, "policy.evaluate")
   defer span.End()

   // Add attributes
   span.SetAttributes(
       attribute.String("policy.id", policyID),
       attribute.String("tenant.id", tenantID),
   )
   ```

3. **Structured Logging** (JSON):
   ```go
   log.Info().
       Str("tenant_id", tenantID).
       Str("request_id", requestID).
       Dur("duration_ms", elapsed).
       Msg("build completed successfully")
   ```

**Alerting**:
- SLO-based alerts (error rate, latency, availability)
- PagerDuty / Opsgenie integration
- Anomaly detection for business metrics
- Capacity planning alerts (CPU, memory, disk)

### 5. API v2 Design

**RESTful API Evolution**:

```
v1.x:
POST /plan/generate
POST /build/run
GET  /drift/check

v2.0:
POST /v2/tenants/{tenant_id}/plans
POST /v2/tenants/{tenant_id}/builds
GET  /v2/tenants/{tenant_id}/drift
```

**GraphQL API** (new in v2.0):
```graphql
type Query {
  tenant(id: ID!): Tenant
  plans(tenantId: ID!, filters: PlanFilters): PlanConnection
  driftReport(tenantId: ID!, planId: ID!): DriftReport
}

type Mutation {
  createPlan(input: CreatePlanInput!): Plan
  executeBuild(input: ExecuteBuildInput!): Build
  approveDrift(input: ApproveDriftInput!): Approval
}
```

**API Gateway Features**:
- Request routing by tenant
- Rate limiting per tenant/user
- API key validation
- Request/response transformation
- Circuit breaking for downstream services

### 6. Data Management & Scale

**Sharding Strategy**:
- Horizontal partitioning by tenant_id
- Shard key: `tenant_id` (ensures tenant data co-located)
- Automatic rebalancing when adding new shards
- Cross-shard queries via distributed query engine

**Caching Architecture**:
```
┌─────────────┐
│  Application │
└──────┬──────┘
       │
┌──────▼───────────────────┐
│  Redis Cluster (L1)      │
│  (Hot data, 1min TTL)    │
└──────┬───────────────────┘
       │ (miss)
┌──────▼───────────────────┐
│  PostgreSQL (L2)         │
│  (Warm data, persistent) │
└──────────────────────────┘
```

**Archival Policy**:
- Automatic archival after 90 days (configurable per tenant)
- Cold storage in S3/Azure Blob
- Compliance-driven retention (GDPR: right to erasure)
- Point-in-time recovery up to 30 days

### 7. Migration Strategy

**v1.x → v2.0 Migration Path**:

1. **Configuration Migration**:
   ```bash
   # Automated migration tool
   specular migrate config \
     --from ~/.specular/config.v1.yaml \
     --to ~/.specular/config.v2.yaml
   ```

2. **Data Migration**:
   ```bash
   # Migrate single-tenant data to multi-tenant schema
   specular migrate database \
     --from postgres://localhost/specular_v1 \
     --to postgres://localhost/specular_v2 \
     --tenant-id default \
     --dry-run
   ```

3. **Dual API Support** (6 months):
   - v2.0 accepts both `/v1/*` and `/v2/*` endpoints
   - Deprecation warnings for v1 API usage
   - Automatic translation layer (v1 requests → v2 internal)

4. **Deprecation Timeline**:
   - **v1.9.0** (Q1 2025): Deprecation warnings for v1 API
   - **v2.0.0** (Q4 2025): Dual support (v1 + v2)
   - **v2.1.0** (Q1 2026): v1 API removed

**Extended Support**:
- v1.6.0 → v1.9.x receives security patches until June 2026
- Critical bug fixes backported for 6 months post-v2.0
- Community support via GitHub Discussions

## Consequences

### Benefits

1. **Enterprise Adoption**:
   - SOC2/ISO 27001 certification path unlocked
   - Multi-tenancy enables SaaS deployment model
   - SSO integration removes authentication barrier
   - Audit logging satisfies compliance requirements

2. **Operational Excellence**:
   - 99.9% uptime SLA achievable
   - Zero-downtime deployments reduce risk
   - Comprehensive observability enables proactive issue detection
   - Disaster recovery automation reduces MTTR

3. **Scale & Performance**:
   - Horizontal scaling supports 10,000+ concurrent users
   - Multi-tenancy enables efficient resource utilization
   - Caching reduces database load by 80%+
   - Sharding eliminates single database bottleneck

4. **Developer Experience**:
   - GraphQL API improves frontend integration
   - Improved error messages with trace context
   - Better local development with health checks
   - Migration tools reduce upgrade friction

### Trade-offs

1. **Complexity**:
   - Multi-tenancy adds tenant context to all requests
   - Distributed tracing requires instrumentation overhead
   - More complex deployment topology (API Gateway, Redis Cluster)

2. **Migration Burden**:
   - Breaking changes require code updates for integrations
   - Data migration downtime (estimated 2-4 hours for large instances)
   - Learning curve for new CLI commands and API

3. **Operational Cost**:
   - Additional infrastructure: Redis Cluster, Prometheus, Jaeger
   - Compliance certification costs (~$50k for SOC2 Type II)
   - SRE time for monitoring setup and runbook creation

4. **Development Velocity**:
   - Short-term slowdown during migration implementation
   - More rigorous testing required (multi-tenant scenarios)
   - Backward compatibility maintenance burden (6 months)

### Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **SOC2 audit delays** | Blocks enterprise sales | Start audit prep in M9, engage auditors early |
| **Multi-tenancy performance** | Poor user experience at scale | Extensive load testing, dedicated instances for large tenants |
| **Breaking changes adoption** | Slow v2.0 uptake, customer churn | Excellent migration tools, dual API support, extended v1.x support |
| **Data migration failures** | Data loss, downtime | Comprehensive testing, rollback procedures, backup validation |
| **Increased operational complexity** | Higher MTTR, more incidents | Runbooks, automated remediation, chaos engineering |

## Implementation Plan

### Phase 1: Foundation (M9 - 8 weeks)

**M9.1: High Availability & Reliability**
- Graceful shutdown mechanism
- Health check endpoints (liveness, readiness, startup)
- Connection draining
- Rolling update validation
- Disaster recovery automation

**M9.2: Enterprise Security**
- SSO/SAML integration (Okta, Auth0)
- ABAC authorization engine
- Audit logging with ECDSA signatures
- HashiCorp Vault integration

**M9.3: Observability & Monitoring**
- OpenTelemetry instrumentation
- Prometheus metrics export
- Jaeger distributed tracing
- Grafana dashboard templates
- PagerDuty alerting integration

### Phase 2: Scale (M10 - 6 weeks)

**M10.1: Multi-Tenancy Architecture**
- Tenant isolation (database-per-tenant)
- Organization management
- Resource quotas per tenant
- Tenant-specific policy overrides

**M10.2: Horizontal Scaling**
- Stateless architecture (externalize sessions to Redis)
- Load balancing with health-based routing
- Kubernetes HPA integration
- Connection pooling optimization

**M10.3: Data Management at Scale**
- Sharding implementation
- Archival policies (S3/Azure Blob)
- Zero-downtime schema migrations
- Data lifecycle management

### Phase 3: Integration (M11 - 7 weeks)

**M11.1: Enterprise Tools Integration**
- Slack, Microsoft Teams, Jira
- Jenkins, GitLab CI, CircleCI
- AWS/Azure/GCP marketplace listings
- Enterprise container registry (multi-registry support)

**M11.2: API Ecosystem**
- RESTful API v2 with tenant context
- GraphQL API
- Official SDKs (Go, Python, TypeScript, Java)
- Developer portal with API explorer

**M11.3: Data Export & Reporting**
- Bulk export APIs (CSV, JSON, XLSX)
- Custom report builder
- Tableau, Power BI, Looker integration

### Phase 4: Governance & Launch (M12 - 8 weeks)

**M12.1: Compliance & Certifications**
- SOC2 Type II certification
- ISO 27001 readiness
- GDPR compliance features
- HIPAA/PCI DSS guides

**M12.2: Governance Framework**
- Policy Engine v2 (advanced DSL)
- Multi-stage approval workflows
- Change management system
- Risk scoring and dashboards

**M12.3: Community & Ecosystem**
- Plugin marketplace
- Certification program (Developer, Administrator)
- Annual user conference
- Comprehensive video tutorials

**M12.4: v2.0 Release Preparation**
- Migration tool development
- Breaking changes documentation
- Marketing materials and launch event
- Staged rollout plan

## Success Metrics

**Technical Goals**:
- 99.9% uptime SLA (measured monthly)
- <100ms p95 response time, <200ms p99
- Support 10,000+ concurrent users per instance
- Zero critical vulnerabilities, <48hr resolution for high severity
- 90%+ test coverage maintained

**Business Goals**:
- 1,000+ active installations within 6 months of v2.0 GA
- 50+ production deployments in Fortune 500 companies
- 10+ enterprise customers on paid plans
- 500+ GitHub stars, 50+ contributors, 100+ community plugins

**Adoption Goals**:
- 80% of v1.x users migrated to v2.0 within 3 months
- NPS score 50+ from enterprise customers
- Featured in major tech publications (TechCrunch, The New Stack)
- Considered "production-ready" by analyst firms (Gartner, Forrester)

## Related ADRs

- **ADR-0001**: Multi-Model Routing (foundation for tenant-aware routing)
- **ADR-0003**: Docker-Only Execution (security isolation model)
- **ADR-0004**: Provider Abstraction (extensible for multi-tenancy)
- **ADR-0005**: Drift Detection Approach (enhanced with tenant context)
- **ADR-0007**: Autonomous Agent Mode (stateless for horizontal scaling)
- **ADR-0009**: Observability & Monitoring Strategy (expanded in v2.0)
- **ADR-0010**: Governance-First CLI Redesign (tenant-aware commands)

## References

- [ROADMAP_v2.0.md](../ROADMAP_v2.0.md) - Detailed milestone breakdown
- [Multi-Tenancy Patterns](https://docs.microsoft.com/en-us/azure/architecture/patterns/multitenancy)
- [The Twelve-Factor App](https://12factor.net/)
- [OpenTelemetry Best Practices](https://opentelemetry.io/docs/concepts/observability-primer/)
- [OWASP Multi-Tenant Security](https://owasp.org/www-project-multi-tenant-security/)

---

**Decision Outcome**: **Accepted**

v2.0 architecture represents Specular's evolution from beta product to enterprise-grade platform. The breaking changes are justified by the significant value delivered: multi-tenancy for SaaS deployment, 99.9% uptime SLA, compliance certifications, and horizontal scalability to 10,000+ users.

The 6-month dual API support period and comprehensive migration tooling minimize upgrade friction. The staged rollout across M9-M12 (8 months) allows for iterative delivery and validation.

This architecture positions Specular as the trusted choice for enterprise AI governance and enables sustainable business growth through SaaS offerings.

**Next Steps**:
1. Create GitHub Project for M9-M12 milestone tracking
2. Set up `develop/v2.0` branch for integration
3. Begin M9.1.1 implementation (graceful shutdown + health checks)
4. Schedule SOC2 audit kick-off meeting

**Document Owners**: Product & Engineering Leadership
**Review Cycle**: After each milestone completion (M9, M10, M11, M12)
