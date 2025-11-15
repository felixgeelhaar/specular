# Intellectual Property Audit Plan

**Document Version**: 1.0
**Date Created**: 2025-01-15
**Last Updated**: 2025-01-15
**Status**: Active - v1.6.0 Preparation
**Owner**: Engineering Leadership

## Purpose

This document provides a comprehensive audit of Specular's codebase to identify intellectual property (IP) that requires protection. It categorizes code by IP value, assesses competitive risks, and defines migration paths for transitioning to an open-core model with dual repositories.

## Audit Methodology

### IP Value Classification

**🔴 CRITICAL (High-Value IP)** - Must protect in private repository
- Unique competitive differentiators
- Expensive to develop (>2 weeks engineering time)
- Difficult to replicate without access to source
- Core to enterprise value proposition

**🟡 MODERATE (Medium-Value IP)** - Evaluate for protection
- Valuable optimizations or patterns
- Moderate development cost (1-2 weeks)
- Could be independently discovered
- Nice-to-have for enterprise

**🟢 LOW (Low-Value IP)** - Safe for public repository
- Standard patterns and implementations
- Publicly documented approaches
- Easy to replicate independently
- Community value outweighs protection needs

### Risk Assessment Factors

For each component, we evaluate:
1. **Competitive Risk**: Could competitors gain advantage from this code?
2. **Development Cost**: How expensive was this to build?
3. **Uniqueness**: Is this a novel approach or standard pattern?
4. **Enterprise Value**: Is this critical for enterprise sales?
5. **Community Value**: Would open-sourcing benefit adoption?

## Codebase Audit

### Current Repository Structure

```
specular/
├── cmd/specular/              # CLI entry point
├── internal/
│   ├── auto/                  # Autonomous mode
│   ├── autopolicy/            # Autonomous policy enforcement
│   ├── bundle/                # Bundle packaging
│   ├── domain/                # Core domain types
│   ├── exec/                  # Execution engine
│   ├── genplan/               # Plan generation
│   ├── genspec/               # Spec generation
│   ├── interview/             # Interactive prompts
│   ├── llm/                   # LLM provider abstraction
│   ├── patch/                 # Patch generation
│   ├── policy/                # Policy enforcement
│   └── session/               # Session management
├── pkg/                       # Public packages (empty currently)
└── providers/                 # AI provider binaries
```

### Component-by-Component Analysis

---

#### 1. `cmd/specular/` - CLI Entry Point

**IP Value**: 🟢 LOW

**Analysis**:
- Standard CLI application structure
- Uses cobra/viper for command parsing
- No proprietary logic in main.go
- User-facing commands define public API

**Competitive Risk**: **Low** - CLI patterns are standard

**Recommendation**: ✅ **Keep Public**

**Reasoning**:
- CLI interface should be visible to users
- Community can contribute new commands
- No competitive advantage in CLI structure
- Public visibility aids adoption

---

#### 2. `internal/domain/` - Core Domain Types

**IP Value**: 🟢 LOW

**Analysis**:
- Defines Spec, Plan, Build, Policy types
- Data structures for core concepts
- No algorithmic logic, just types
- Shared across all components

**Competitive Risk**: **Low** - Type definitions are not IP

**Recommendation**: ✅ **Keep Public** (move to `pkg/specular/`)

**Reasoning**:
- Public types enable SDK development
- Community integrations need these types
- No proprietary logic in type definitions
- Essential for API compatibility

**Migration**: Move to `pkg/specular/types/` for v1.6.0

---

#### 3. `internal/exec/` - Execution Engine

**IP Value**: 🟢 LOW (mostly), 🟡 MODERATE (optimizations)

**Files Analysis**:

| File | IP Value | Keep Public? | Notes |
|------|----------|--------------|-------|
| `docker.go` | 🟢 LOW | ✅ Yes | Standard Docker SDK usage |
| `sandbox.go` | 🟢 LOW | ✅ Yes | Basic sandboxing patterns |
| `image.go` | 🟡 **MODERATE** | ⚠️ **Evaluate** | **80%+ cache optimization** - consider protecting |
| `cache.go` | 🟡 **MODERATE** | ⚠️ **Evaluate** | **Caching strategies** - moderate value |
| `prewarm.go` | 🟢 LOW | ✅ Yes | Standard prewarming |

**Competitive Risk**: **Low-Medium**

**Recommendation**:
- ✅ **Keep basic execution public** (docker.go, sandbox.go)
- ⚠️ **Evaluate caching optimizations** (image.go, cache.go) - consider keeping basic implementation public, move advanced optimizations to enterprise

**Reasoning**:
- Docker execution is standard pattern
- 80% cache improvement is valuable but could be independently discovered
- Keeping basic version public aids community trust
- Enterprise version could have additional optimizations

**Migration**:
- v1.6.0: Keep all public (document optimization as feature)
- v2.0 M9: Consider extracting advanced caching to private repo

---

#### 4. `internal/genspec/` - Spec Generation

**IP Value**: 🔴 **CRITICAL**

**Analysis**:
- **AI prompting strategies** for spec generation
- **Prompt engineering templates**
- **Multi-turn conversation orchestration**
- **Provider-specific optimizations**

**Competitive Risk**: **CRITICAL** - Core differentiator

**Recommendation**: 🔒 **PROTECT** - Move to private repository

**Reasoning**:
- Unique prompting strategies took months to develop
- Prompt engineering is core competitive advantage
- Easy to copy if exposed
- Expensive to replicate without access

**Migration**:
- v1.6.0: Keep public with basic prompts, add TODO comments for "enterprise prompts"
- v2.0 M9: **Move advanced prompting to `specular-platform/internal/intelligence/spec_generator.go`**

**Public Interface** (pkg/specular/):
```go
// Public interface only
type SpecGenerator interface {
    Generate(ctx context.Context, input SpecInput) (*Spec, error)
}
```

**Private Implementation** (specular-platform/):
```go
// Proprietary prompting strategies
type EnterpriseSpecGenerator struct {
    promptOptimizer  *PromptOptimizer  // 🔒 CRITICAL IP
    conversationFlow *FlowOrchestrator // 🔒 CRITICAL IP
}
```

---

#### 5. `internal/genplan/` - Plan Generation

**IP Value**: 🔴 **CRITICAL**

**Analysis**:
- **AI prompting for plan generation**
- **Dependency ordering algorithms**
- **Step optimization logic**
- **Error recovery strategies**

**Competitive Risk**: **CRITICAL** - Core differentiator

**Recommendation**: 🔒 **PROTECT** - Move to private repository

**Reasoning**:
- Similar to genspec/ - core prompting IP
- Plan optimization algorithms are proprietary
- Dependency resolution is complex and valuable

**Migration**:
- v1.6.0: Keep public with basic implementation
- v2.0 M9: **Move to `specular-platform/internal/intelligence/plan_generator.go`**

---

#### 6. `internal/auto/` - Autonomous Mode

**IP Value**: 🔴 **CRITICAL**

**Analysis**:
- **Multi-step reasoning orchestration**
- **Autonomous decision-making logic**
- **Error recovery and retry strategies**
- **Step sequencing algorithms**

**Competitive Risk**: **CRITICAL** - Flagship feature

**Recommendation**: 🔒 **PROTECT** - Move to private repository

**Reasoning**:
- Autonomous mode is unique competitive feature
- Complex orchestration logic (M7 milestone)
- Expensive to develop (4+ weeks of engineering)
- Difficult to replicate without seeing implementation

**Migration**:
- v1.6.0: Keep public (already released as M7)
- v2.0 M9: **Consider moving advanced autonomous features to private repo**
- Alternative: Keep basic autonomous in public, add "Enterprise Autonomous" with advanced intelligence

**Public vs. Enterprise Split**:
- 🟢 Public: Basic autonomous mode (max 10 steps, basic error handling)
- 🔒 Enterprise: Advanced autonomous (unlimited steps, ML-based optimization, predictive error handling)

---

#### 7. `internal/policy/` - Policy Enforcement

**IP Value**: 🔴 **CRITICAL** (advanced engine), 🟢 LOW (basic parser)

**Files Analysis**:

| File | IP Value | Keep Public? | Notes |
|------|----------|--------------|-------|
| `parser.go` | 🟢 LOW | ✅ Yes | Basic policy parsing |
| `types.go` | 🟢 LOW | ✅ Yes | Policy types (move to pkg/) |
| `validator.go` | 🟡 MODERATE | ⚠️ Evaluate | Basic validation logic |
| `engine.go` | 🔴 **CRITICAL** | 🔒 **Protect** | **Advanced evaluation engine** |
| `compiler.go` | 🔴 **CRITICAL** | 🔒 **Protect** | **Policy DSL compiler** |

**Competitive Risk**: **CRITICAL** for advanced features

**Recommendation**:
- ✅ **Keep basic policy engine public** (parser, types, simple validation)
- 🔒 **Protect advanced engine** (DSL compiler, complex evaluation, dependency analysis)

**Reasoning**:
- Basic policy enforcement shows capability
- Advanced DSL compiler is unique IP
- Policy engine v2 (M12) will be enterprise-only

**Migration**:
- v1.6.0: Keep current implementation public
- v2.0 M12: **Create `specular-platform/internal/policy/engine_v2.go`** with:
  - Advanced DSL compiler
  - Dependency graph analysis
  - Multi-dimensional rule evaluation
  - ML-based policy suggestions

---

#### 8. `internal/autopolicy/` - Autonomous Policy Enforcement

**IP Value**: 🔴 **CRITICAL**

**Analysis**:
- Combines autonomous mode + policy enforcement
- Per-step policy checking logic
- Advanced error handling with policy context

**Competitive Risk**: **HIGH** - Unique combination

**Recommendation**: 🔒 **PROTECT** - Move to private or make enterprise-only

**Reasoning**:
- Novel integration of autonomy and governance
- Complex interaction logic
- Enterprise governance feature

**Migration**:
- v1.6.0: Keep public (already released)
- v2.0 M9: Consider making this enterprise-only feature

---

#### 9. `internal/llm/` - LLM Provider Abstraction

**IP Value**: 🟢 LOW

**Analysis**:
- Standard provider abstraction pattern
- Public API wrappers (OpenAI, Anthropic, etc.)
- Simple adapter interfaces

**Competitive Risk**: **Low** - Standard abstraction

**Recommendation**: ✅ **Keep Public**

**Reasoning**:
- Using public APIs (OpenAI, Anthropic, Gemini)
- Standard adapter pattern
- Community can add more providers
- No proprietary logic

**Enhancement**:
- Move interface to `pkg/specular/provider/` for public SDK

---

#### 10. `internal/patch/` - Patch Generation

**IP Value**: 🟡 MODERATE

**Analysis**:
- Unified diff generation
- Rollback functionality
- Cryptographic attestation integration

**Competitive Risk**: **Medium**

**Recommendation**: ✅ **Keep Public** (for now)

**Reasoning**:
- Standard diff algorithms (GNU diffutils)
- Rollback is valuable but not unique
- Attestation is cryptographic, not algorithmic IP

**Future**: v2.0 could add enterprise patch features (approval workflows, audit trail)

---

#### 11. `internal/interview/` - Interactive Prompts

**IP Value**: 🟢 LOW

**Analysis**:
- CLI interactive prompting (M6 feature)
- Uses survey library
- Standard patterns

**Competitive Risk**: **Low**

**Recommendation**: ✅ **Keep Public**

**Reasoning**:
- Standard CLI UX patterns
- No proprietary logic
- Community value from seeing implementation

---

#### 12. `internal/session/` - Session Management

**IP Value**: 🟢 LOW

**Analysis**:
- Session state tracking
- Checkpoint/resume functionality
- File-based persistence

**Competitive Risk**: **Low-Medium**

**Recommendation**: ✅ **Keep Public**

**Reasoning**:
- Standard session management patterns
- File-based checkpoints are visible to users anyway
- Community benefit from seeing implementation

**Future**: v2.0 Enterprise could add:
- 🔒 Database-backed session storage
- 🔒 Cross-device session sync
- 🔒 Team collaboration on sessions

---

#### 13. `internal/bundle/` - Bundle Packaging

**IP Value**: 🟢 LOW

**Analysis**:
- TAR archive creation
- File bundling
- Standard compression

**Competitive Risk**: **Low**

**Recommendation**: ✅ **Keep Public**

**Reasoning**:
- Standard packaging patterns
- No unique algorithms

---

#### 14. `providers/` - External Provider Binaries

**IP Value**: 🟢 LOW

**Analysis**:
- Wrapper scripts for external AI providers
- CLI provider protocol implementation

**Competitive Risk**: **Low**

**Recommendation**: ✅ **Keep Public**

**Reasoning**:
- Enables community provider development
- Provider protocol should be public
- No proprietary logic in wrappers

---

## Summary Tables

### High-Value IP to Protect

| Component | IP Value | Migration Target | Timeline |
|-----------|----------|------------------|----------|
| `internal/genspec/` | 🔴 CRITICAL | `specular-platform/internal/intelligence/` | v2.0 M9 |
| `internal/genplan/` | 🔴 CRITICAL | `specular-platform/internal/intelligence/` | v2.0 M9 |
| `internal/auto/` (advanced) | 🔴 CRITICAL | `specular-platform/internal/intelligence/` | v2.0 M9 |
| `internal/policy/` (engine v2) | 🔴 CRITICAL | `specular-platform/internal/policy/` | v2.0 M12 |
| `internal/autopolicy/` (enterprise) | 🔴 CRITICAL | Enterprise-only feature | v2.0 M9 |

### Components to Keep Public

| Component | IP Value | Future Public Location | Notes |
|-----------|----------|----------------------|-------|
| `cmd/specular/` | 🟢 LOW | `specular/cmd/specular/` | CLI interface |
| `internal/domain/` | 🟢 LOW | `specular/pkg/specular/types/` | Public types |
| `internal/exec/` (basic) | 🟢 LOW | `specular/internal/exec/` | Basic execution |
| `internal/llm/` | 🟢 LOW | `specular/pkg/specular/provider/` | Provider interface |
| `internal/interview/` | 🟢 LOW | `specular/internal/interview/` | Interactive UX |
| `internal/session/` | 🟢 LOW | `specular/internal/session/` | Basic sessions |
| `internal/patch/` | 🟡 MODERATE | `specular/internal/patch/` | Diff/rollback |
| `internal/bundle/` | 🟢 LOW | `specular/internal/bundle/` | Packaging |
| `providers/` | 🟢 LOW | `specular/providers/` | Provider wrappers |

### Components to Evaluate

| Component | IP Value | Decision Needed | Recommendation |
|-----------|----------|-----------------|----------------|
| `internal/exec/` (cache optimizations) | 🟡 MODERATE | Keep public or move? | Keep basic public, enhance in enterprise |
| `internal/auto/` (basic features) | 🟡 MODERATE | All public or split? | Basic public, advanced enterprise |
| `internal/policy/` (validation) | 🟡 MODERATE | Public or private? | Basic public, v2 engine private |

## Migration Roadmap

### Phase 1: v1.6.0 (Q2 2025) - Preparation

**Goal**: Prepare codebase for extraction without breaking changes

**Actions**:
1. ✅ Create `pkg/specular/` directory structure
2. ✅ Move domain types to `pkg/specular/types/`
3. ✅ Move provider interface to `pkg/specular/provider/`
4. ✅ Add feature flags for enterprise features:
   ```go
   const (
       FeatureBasicSpec      = "basic-spec"       // Public
       FeatureEnterpriseSpec = "enterprise-spec"  // Enterprise-only
       FeatureAdvancedAuto   = "advanced-auto"    // Enterprise-only
       FeaturePolicyV2       = "policy-v2"        // Enterprise-only
   )
   ```
5. ✅ Add TODOs in critical IP files: `// TODO: Move to enterprise repo in v2.0`
6. ✅ Document trade secrets internally
7. ✅ Apply BSL 1.1 license

**Status**: Single public repository with BSL

---

### Phase 2: v2.0 M9 (Q3 2025) - Repository Split

**Goal**: Create private repository and migrate high-value IP

**Actions**:
1. Create `specular-platform/` private repository
2. Set up Go module structure:
   ```go
   // specular-platform/go.mod
   module github.com/felixgeelhaar/specular-platform

   require github.com/felixgeelhaar/specular v1.6.0
   ```
3. Migrate critical IP:
   - Move `internal/genspec/` → `specular-platform/internal/intelligence/spec_generator.go`
   - Move `internal/genplan/` → `specular-platform/internal/intelligence/plan_generator.go`
   - Move advanced `internal/auto/` → `specular-platform/internal/intelligence/autonomous.go`
4. Refactor public repo to use `pkg/specular/` interfaces
5. Implement runtime license checks in enterprise builds
6. Set up private CI/CD pipeline

**Public Repo Changes**:
- Keep basic implementations (good enough for free tier)
- Reference enterprise features via feature flags (disabled in OSS builds)
- Maintain backward compatibility

**Status**: Dual-repository model operational

---

### Phase 3: v2.0 M12 (Q4 2025) - Enterprise Features

**Goal**: Build enterprise-only features in private repo

**Actions**:
1. Build Policy Engine v2 in `specular-platform/internal/policy/`
2. Implement multi-tenancy in `specular-platform/internal/multitenancy/`
3. Build enterprise auth in `specular-platform/internal/enterprise/`
4. Create web dashboard in `specular-platform/web/`
5. Add enterprise integrations in `specular-platform/internal/integrations/`

**Status**: Full open-core model with clear free/enterprise split

---

## Protection Mechanisms

### 1. Legal Protection

**Business Source License (BSL 1.1)** for public repository:
- Prevents commercial competing services
- Converts to Apache 2.0 after 2 years
- Allows internal enterprise use

**Proprietary License** for private repository:
- Full copyright protection
- Enterprise-only distribution
- License validation required

### 2. Technical Protection

**Feature Flags**:
```go
// pkg/specular/features/features.go
package features

func IsEnterpriseEnabled() bool {
    // In OSS builds: return false
    // In Enterprise builds: check license server
    return validateLicense()
}
```

**Build Tags**:
```go
// +build enterprise

package intelligence

// Enterprise-only code
```

**Runtime License Checks**:
```go
// internal/license/check.go (private repo only)
func init() {
    if !validateEnterpriseLicense() {
        panic("Enterprise license required")
    }
}
```

### 3. Operational Protection

**Trade Secret Program**:
- [ ] NDAs for all employees and contractors
- [ ] Access controls on private repository
- [ ] Confidentiality markings in sensitive files:
  ```go
  /*
   * Copyright © 2025 Specular Inc. All Rights Reserved.
   * Trade Secret - Confidential and Proprietary
   * Unauthorized distribution prohibited
   */
  ```
- [ ] Regular audits of code access

## Risk Assessment

### Residual Risks (Cannot Fully Prevent)

**Clean Room Implementation**:
- ❌ Competitors can independently rebuild functionality
- **Mitigation**: Focus on execution speed, stay ahead
- **Acceptance**: Part of doing business

**API Reverse Engineering**:
- ❌ Public API reveals some behavior
- **Mitigation**: Don't expose all enterprise features via public API
- **Acceptance**: API must be public for integrations

**UI/UX Copying**:
- ❌ User interfaces are not copyrightable
- **Mitigation**: Trademark protection, brand differentiation
- **Acceptance**: Focus on experience, not just visuals

### Monitored Risks (Require Enforcement)

**License Violations**:
- ⚠️ Companies using public code to build competing SaaS
- **Mitigation**: Automated scanning, legal enforcement
- **Action Required**: Monitor GitHub forks, web services

**Unauthorized Distribution**:
- ⚠️ Enterprise binaries leaked or redistributed
- **Mitigation**: License validation, binary fingerprinting
- **Action Required**: Legal action if discovered

## Next Steps

### Immediate (Before v1.6.0 Release)

- [ ] Complete this audit review with legal counsel
- [ ] Apply BSL 1.1 license to public repository
- [ ] Create `pkg/specular/` public SDK structure
- [ ] Add feature flags for enterprise features
- [ ] Add confidentiality notices to critical files
- [ ] Document trade secrets program

### Near-Term (v1.6.0 Development)

- [ ] Refactor code to prepare for extraction
- [ ] Write tests for public interfaces
- [ ] Create migration scripts for v2.0 split
- [ ] Set up private repository infrastructure
- [ ] Design enterprise licensing system

### Long-Term (v2.0+)

- [ ] Execute repository split (M9)
- [ ] Implement enterprise features (M10-M12)
- [ ] Establish trade secret protection procedures
- [ ] Regular IP audits (quarterly)
- [ ] Monitor competitive landscape

## Appendix

### Trade Secret Classification

**Classified as Trade Secrets** (maintain confidentiality):
1. AI prompting strategies and templates
2. Autonomous orchestration algorithms
3. Policy engine v2 compiler and evaluator
4. Multi-tenancy isolation mechanisms
5. Performance optimization techniques
6. Enterprise integration implementations

**Not Trade Secrets** (can be public):
1. CLI interface and command structure
2. Basic Docker execution patterns
3. Provider abstraction interfaces
4. Standard data types
5. Interactive prompt implementations

### References

- ADR-0001: IP Protection and Open-Core Strategy
- ROADMAP_v1.6.0.md: M8 Beta Hardening
- ROADMAP_v2.0.md: M9-M12 Enterprise Features
- RELEASE_LESSONS_v1.5.0.md: Process improvements

---

**Document Owner**: Engineering Leadership
**Legal Review**: [Required before v1.6.0]
**Last Audit Date**: 2025-01-15
**Next Audit Date**: Q2 2025 (before v2.0 M9)
