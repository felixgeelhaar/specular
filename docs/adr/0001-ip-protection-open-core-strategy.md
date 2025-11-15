# ADR 0001: IP Protection and Open-Core Strategy

**Status**: Accepted
**Date**: 2025-01-15
**Decision Makers**: Product & Engineering Leadership
**Stakeholders**: Engineering, Legal, Business Development

## Context

Specular is an AI-native spec and build assistant with policy enforcement. As of v1.5.0, the entire codebase is public under an MIT license to facilitate community adoption and transparency. However, as we plan for v2.0 (M9-M12 milestones) with enterprise features including multi-tenancy, SSO/SAML, advanced policy engines, and SaaS offerings, we face strategic questions about intellectual property (IP) protection:

### Business Model Evolution

1. **Current State (v1.5.0)**:
   - Fully open-source CLI tool (MIT license)
   - Free for all users
   - Community-driven development
   - Single public repository

2. **Planned State (v2.0)**:
   - Free CLI for individual developers
   - Enterprise platform with advanced features
   - SaaS offering for organizations
   - Need to protect competitive advantages

### IP Protection Concerns

**High-Value Intellectual Property**:
- AI prompting strategies and orchestration algorithms
- Advanced policy engine logic and DSL compiler
- Multi-tenancy architecture and tenant isolation mechanisms
- Autonomous mode intelligence (step sequencing, error recovery)
- Enterprise integrations (SSO/SAML, ServiceNow, Jira)
- Performance optimization techniques (80%+ cache improvements)

**Competitive Risks**:
- Direct competitors copying innovations
- Cloud providers (AWS, Azure, GCP) building managed Specular services
- Enterprise software companies wrapping our technology
- Loss of competitive moat for enterprise features

**Community Value**:
- Open source builds trust and credibility
- Community contributions enhance the ecosystem
- Public development accelerates bug fixes
- GitHub stars and visibility drive adoption

### Strategic Tension

We need to balance:
- ✅ **Open Source Benefits**: Community trust, contributions, visibility, adoption
- 🔒 **IP Protection**: Competitive advantage, revenue generation, business sustainability

## Decision

We will adopt an **Open-Core Model** with a **Business Source License (BSL)** for the public repository, transitioning to a dual-repository strategy for v2.0.

### Licensing Strategy

#### Public Repository (specular/) - Business Source License 1.1

**License**: Business Source License 1.1 (BSL)

**Parameters**:
- **Licensor**: Specular Inc.
- **Licensed Work**: Specular CLI and Core Engine
- **Additional Use Grant**: All uses permitted EXCEPT:
  - Providing a commercial AI-assisted specification and build service to third parties
  - Offering Specular as a managed service or SaaS product
  - Competing directly with Specular's commercial offerings
- **Change Date**: 2 years from each release (e.g., v1.6.0 released 2025-06 → converts 2027-06)
- **Change License**: Apache License 2.0

**What BSL Allows**:
- ✅ Internal use by companies (unlimited scale)
- ✅ Consulting and integration services
- ✅ Educational and research use
- ✅ Modifications and derivatives (for allowed uses)
- ✅ Becoming Apache 2.0 after change date

**What BSL Prevents** (during restriction period):
- ❌ AWS/Azure/GCP offering "Managed Specular"
- ❌ Competitors launching competing SaaS products
- ❌ Cloud providers monetizing our work without contribution

**Rationale**: BSL provides maximum protection while maintaining source availability and community benefits. After 2 years, code becomes fully open source (Apache 2.0), ensuring long-term openness while protecting short-term competitive advantage.

#### Private Repository (specular-platform/) - Proprietary License

**License**: Proprietary Commercial License

**Protection**: Full copyright protection for:
- Enterprise-only features
- Multi-tenant architecture
- SaaS platform code
- Advanced AI orchestration
- Proprietary integrations

### Repository Structure

#### Phase 1: Current (v1.5.0 - v1.6.0)

**Single Public Repository with BSL**:
```
specular/                          # PUBLIC (BSL 1.1 → Apache 2.0)
├── cmd/specular/                  # CLI
├── internal/
│   ├── domain/                    # Core business logic
│   ├── exec/                      # Execution engine
│   ├── policy/                    # Basic policy engine
│   ├── auto/                      # Autonomous mode
│   └── providers/                 # AI provider integrations
├── pkg/specular/                  # Public SDK
└── docs/                          # Documentation
```

**Actions**:
1. Replace MIT license with BSL 1.1
2. Add feature flags for future enterprise features (stubbed)
3. Document trade secrets internally
4. Prepare codebase for extraction

#### Phase 2: v2.0 M9+ (Production Hardening)

**Dual Repository Model**:

```
# PUBLIC REPOSITORY
specular/                          # BSL 1.1 → Apache 2.0
├── cmd/specular/                  # Free CLI
├── internal/
│   ├── domain/                    # Core types
│   ├── exec/                      # Basic execution
│   ├── policy/                    # Basic policy engine
│   └── providers/                 # AI providers
├── pkg/specular/                  # Public SDK (used by both repos)
└── docs/                          # Public documentation

# PRIVATE REPOSITORY
specular-platform/                 # Proprietary
├── cmd/
│   ├── specular-server/          # API server
│   ├── specular-worker/          # Background workers
│   └── specular-webhook/         # Webhook service
├── internal/
│   ├── intelligence/             # Advanced AI orchestration
│   ├── policy/                   # Enterprise policy engine v2
│   ├── multitenancy/             # Multi-tenant architecture
│   ├── enterprise/               # SSO, SAML, RBAC, audit
│   ├── integrations/             # Enterprise integrations
│   └── analytics/                # Usage analytics
├── web/                          # Dashboard UI
└── deployments/                  # K8s, Terraform
```

**Go Module Relationship**:
```go
// specular-platform/go.mod (PRIVATE)
module github.com/felixgeelhaar/specular-platform

require (
    github.com/felixgeelhaar/specular v1.6.0  // Imports public SDK
    // Enterprise dependencies
)
```

**Private repo imports public SDK** - ensures one-way dependency, no reverse coupling.

### Feature Split: Free vs. Enterprise

#### Free Tier (Public specular/)

**Target Users**: Individual developers, small teams, open-source projects

**Features**:
- ✅ Full CLI functionality
- ✅ Spec and plan generation
- ✅ Local build execution
- ✅ Basic policy enforcement
- ✅ Docker sandbox execution
- ✅ AI provider integrations (OpenAI, Anthropic, etc.)
- ✅ Checkpoint/resume functionality
- ✅ Patch generation
- ✅ Community plugins
- ✅ File-based configuration

#### Enterprise Tier (Private specular-platform/)

**Target Users**: Enterprises, SaaS customers, organizations requiring compliance

**Features**:
- 🔒 Multi-tenancy with tenant isolation
- 🔒 SSO/SAML authentication
- 🔒 RBAC/ABAC authorization
- 🔒 Web dashboard and UI
- 🔒 RESTful and GraphQL APIs
- 🔒 Webhooks and event streaming
- 🔒 Advanced observability (OpenTelemetry, distributed tracing)
- 🔒 Compliance features (SOC2, ISO 27001, GDPR)
- 🔒 High availability and disaster recovery
- 🔒 Advanced policy engine v2
- 🔒 Enterprise integrations (Slack Enterprise, ServiceNow, Jira, Azure AD)
- 🔒 Priority support and SLAs
- 🔒 Usage analytics and reporting

### Additional IP Protection Mechanisms

#### 1. Trade Secret Protection

**Classification**: High-value algorithms as trade secrets:
- AI prompting strategies and orchestration
- Policy engine optimization algorithms
- Multi-tenancy implementation details

**Requirements**:
- Keep in private repository
- Use NDAs with employees and contractors
- Mark sensitive files with confidentiality headers
- Implement access controls

#### 2. Copyright Notices

Add headers to sensitive files in private repo:
```go
/*
 * Copyright © 2025 Specular Inc. All Rights Reserved.
 *
 * This file contains proprietary and confidential information.
 * Unauthorized copying or distribution is strictly prohibited.
 *
 * Trade Secret - Do Not Distribute
 */
```

#### 3. Runtime License Validation (Enterprise Builds)

Implement license checking in enterprise binaries:
- Validate enterprise subscription status
- Check feature entitlements
- Phone home to license server
- Graceful degradation for expired licenses

#### 4. Patent Strategy (Future Consideration)

**Not immediately pursued**, but consider for:
- Novel AI orchestration methods
- Unique policy enforcement algorithms
- Innovative drift detection techniques

**Cost**: $10,000-$50,000 per patent
**Time**: 2-4 years
**Decision**: Defer until Series A funding or significant revenue

## Consequences

### Positive Consequences

1. **IP Protection**:
   - ✅ Prevents cloud providers from commoditizing Specular
   - ✅ Protects competitive moat for enterprise features
   - ✅ Enables sustainable business model
   - ✅ Trade secrets remain confidential

2. **Community Trust**:
   - ✅ Code remains source-available (BSL)
   - ✅ Automatic conversion to Apache 2.0 (long-term openness)
   - ✅ Community can still contribute to free tier
   - ✅ Transparency builds credibility

3. **Business Flexibility**:
   - ✅ Clear value differentiation (free vs. paid)
   - ✅ Multiple revenue streams (SaaS, enterprise licenses, support)
   - ✅ Freemium funnel (free CLI → enterprise platform)
   - ✅ Proven model (GitLab, Sentry, CockroachDB)

4. **Development Efficiency**:
   - ✅ Faster iteration on enterprise features (private repo)
   - ✅ Public API surface defined (pkg/specular/)
   - ✅ Clear separation of concerns
   - ✅ Independent release cycles

### Negative Consequences

1. **Community Perception**:
   - ❌ Some may view BSL as "not truly open source"
   - ❌ Confusion about licensing terms
   - ❌ Potential contributor friction (which repo to contribute to?)

   **Mitigation**: Clear communication, FAQ, contributor guide

2. **Operational Complexity**:
   - ❌ Managing two repositories
   - ❌ Synchronizing releases
   - ❌ Maintaining shared SDK (pkg/specular/)

   **Mitigation**: Automated CI/CD, shared types in public SDK

3. **Legal Compliance**:
   - ❌ Must enforce license terms
   - ❌ Monitor for license violations
   - ❌ Legal costs for enforcement

   **Mitigation**: Automated license scanning, clear terms

4. **Competitive Risks Remain**:
   - ❌ Clean room implementations (independent rebuilds)
   - ❌ API reverse engineering
   - ❌ UI/UX copying (not copyrightable)

   **Acceptance**: Focus on execution speed, relationships, ecosystem

### Migration Path

#### v1.6.0 (M8: Beta Hardening) - Q2 2025

**Actions**:
1. ✅ Replace LICENSE with BSL 1.1
2. ✅ Update README.md to clarify licensing
3. ✅ Create IP audit plan
4. ✅ Add feature flags for enterprise features (stubbed in public builds)
5. ✅ Organize code for extraction (pkg/specular/)
6. ✅ Document trade secrets internally

**Status**: Preparation phase, single public repo with BSL

#### v2.0 M9 (Production Hardening) - Q3 2025

**Actions**:
1. Create private `specular-platform/` repository
2. Move high-value IP to private repo:
   - Multi-tenancy architecture
   - SSO/SAML implementations
   - Advanced policy engine v2
   - Enterprise observability
3. Refactor public repo to use `pkg/specular/` SDK
4. Implement license validation in enterprise builds
5. Set up private CI/CD pipeline

**Status**: Dual-repository model activated

#### v2.0 M11-M12 (Enterprise Integrations & Launch) - Q4 2025

**Actions**:
1. Build SaaS platform in private repo
2. Enterprise integrations (ServiceNow, etc.) in private repo
3. Launch enterprise offering publicly
4. Web dashboard in private repo
5. Plugin marketplace (public repo, community-driven)

**Status**: Full open-core model operational

#### v3.0+ (2027+)

**Actions**:
1. v1.6.0 code converts to Apache 2.0 (2 years after release)
2. Evaluate patent strategy based on revenue
3. Potential open-sourcing of v2.0 basic features (as v2.x reaches change date)

**Status**: Long-term openness maintained via BSL change dates

## Alternatives Considered

### Alternative 1: Keep Fully Open Source (MIT/Apache 2.0)

**Pros**:
- Maximum community trust
- Easiest contributor onboarding
- Best for ecosystem growth

**Cons**:
- ❌ No IP protection
- ❌ Cloud providers can compete directly
- ❌ Difficult to justify enterprise pricing
- ❌ Vulnerable to commoditization

**Rejected**: Too risky for business sustainability

### Alternative 2: Fully Proprietary (Closed Source)

**Pros**:
- Maximum IP protection
- Full control over code
- Easier to enforce licenses

**Cons**:
- ❌ No community trust or contributions
- ❌ Difficult to gain initial adoption
- ❌ No SEO or visibility benefits
- ❌ Appears "closed" and "untrusted"

**Rejected**: Sacrifices too much community value

### Alternative 3: AGPL License (Copyleft)

**Pros**:
- Forces SaaS providers to open-source modifications
- Protects against cloud providers wrapping product
- Still technically open source

**Cons**:
- ❌ Many enterprises avoid AGPL (license incompatibility)
- ❌ May discourage adoption
- ❌ Still allows competing SaaS (if they open-source)
- ❌ Complex compliance for enterprise users

**Rejected**: Too restrictive for enterprise adoption

### Alternative 4: Dual Licensing (GPL + Commercial)

**Pros**:
- Proven model (MySQL, Qt)
- Forces commercial users to buy licenses or open-source

**Cons**:
- ❌ GPL is restrictive for integrations
- ❌ Complex for users to understand
- ❌ May discourage free tier adoption

**Rejected**: BSL provides better balance

### Alternative 5: Thin Public CLI, All Logic in Private SaaS

**Pros**:
- Maximum IP protection for business logic
- Only public API is visible

**Cons**:
- ❌ Requires internet connection (breaks air-gapped use)
- ❌ No community contributions to core logic
- ❌ Trust issues (all logic in black box)
- ❌ Not truly open source

**Rejected**: Sacrifices too much utility and trust

## Implementation Plan

### Immediate Actions (v1.6.0 Development)

- [x] Document this ADR
- [ ] Replace LICENSE file with BSL 1.1
- [ ] Create IP_AUDIT_PLAN.md
- [ ] Update README.md with licensing section
- [ ] Add FAQ about BSL to docs
- [ ] Create contributor guide explaining repo strategy
- [ ] Add copyright notices to internal files
- [ ] Create feature flags for enterprise features

### Future Actions (v2.0 M9+)

- [ ] Create private `specular-platform/` repository
- [ ] Set up GitHub organization (felixgeelhaar → specular-inc)
- [ ] Configure private Go module proxy
- [ ] Implement license validation in enterprise builds
- [ ] Migrate high-value IP to private repo
- [ ] Establish trade secret protection program (NDAs, access controls)
- [ ] Legal review of license enforcement strategy

## References

- **Business Source License**: https://mariadb.com/bsl11/
- **GitLab Open Core Model**: https://about.gitlab.com/company/stewardship/
- **Sentry Licensing**: https://blog.sentry.io/2019/11/06/relicensing-sentry/
- **CockroachDB BSL**: https://www.cockroachlabs.com/blog/oss-relicensing-cockroachdb/
- **Open Core Definition**: https://en.wikipedia.org/wiki/Open-core_model

## Decision History

- **2025-01-15**: ADR created and accepted
- **2025-01-15**: BSL 1.1 license approved for v1.6.0
- **Future**: Review after v1.6.0 beta (Q2 2025)

---

**Document Owner**: Product & Engineering Leadership
**Legal Review**: [Pending - consult IP attorney before v1.6.0 release]
**Last Updated**: 2025-01-15
**Next Review**: Q2 2025 (post v1.6.0 beta)
