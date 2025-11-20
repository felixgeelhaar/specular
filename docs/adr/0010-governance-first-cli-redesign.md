# ADR 0010: Governance-First CLI Redesign

**Status**: Accepted
**Date**: 2025-11-16
**Decision Makers**: Product & Engineering Leadership
**Stakeholders**: Engineering, Documentation, Future Users

## Context

### Current CLI Structure (v1.0.0 - v1.1.0)

Specular's current CLI is **spec-first oriented**, reflecting its origins as a specification and build assistant:

```
specular interview            # Generate spec via Q&A
specular spec                 # Spec management (lock, validate)
specular plan gen             # Generate plan from spec
specular plan drift           # Drift detection (nested)
specular bundle build         # Create bundle
specular bundle verify        # Verify bundle
specular policy apply/new     # Basic policy operations
specular provider ...         # Provider management
specular auto                 # Autonomous mode
```

**Characteristics**:
- Focused on development workflow (spec → plan → build)
- Governance features nested or secondary
- Commands oriented around artifacts (spec, plan, bundle)
- Policy enforcement present but not prominent

### Strategic Pivot to Governance

Based on:
1. **Market positioning**: AI governance is the differentiated value proposition
2. **README feedback**: Proposed governance-focused messaging resonated
3. **CLI reference spec**: Defines comprehensive governance-first command structure
4. **Zero existing users**: v1.1.0 has no external adoption yet - clean break possible

The product vision has evolved from:
> "AI-Native Spec and Build Assistant"

To:
> "AI-Native Spec and Build Assistant **with Governance**"

Where governance encompasses:
- Policy enforcement and validation
- Drift detection and approval
- Bundle-based deployment gates
- Provider allowlisting and controls
- Cryptographic approvals and attestations
- Audit trails and compliance

### CLI Reference Specification

A comprehensive CLI specification (`specular_cli_reference.md`) defines the target governance-first structure with 10 command categories and standardized exit codes for CI/CD integration.

### Timing Opportunity

**Current state (v1.1.0)**:
- No external users
- No CLI backward compatibility burden
- Open-core model just established (ADR-0001)
- BSL 1.1 license protects IP

This is the **ideal moment** for breaking changes before user adoption begins.

## Decision

We will implement a **governance-first CLI redesign** in v1.2.0 with breaking changes, reorganizing all commands around governance, policy, and compliance workflows.

### New CLI Structure (v1.2.0)

#### 1. Governance Commands (NEW)
```bash
specular governance init      # Initialize governance workspace
specular governance doctor    # Validate governance environment
specular governance status    # Show governance health
```

Creates `.specular/` structure with policies, providers, approvals, bundles, traces.

#### 2. Policy Commands (ENHANCED)
```bash
specular policy init          # Create policies.yaml (NEW)
specular policy validate      # Validate policy definitions (NEW)
specular policy approve       # Approve policies with signature (NEW)
specular policy list          # List all policies (NEW)
specular policy diff          # Show policy changes (NEW)
```

**Before**: `policy apply`, `policy new` (limited)
**After**: Complete policy lifecycle management

#### 3. Approval Commands (NEW)
```bash
specular approve <bundle-id>  # Approve bundle/drift
specular approvals list       # Show all approvals
specular approvals pending    # Show pending approvals
```

#### 4. Bundle Commands (REFACTORED)
```bash
specular bundle create        # Create governed bundle (was: build)
specular bundle gate          # Run governance checks (was: verify)
specular bundle inspect       # Inspect bundle contents (NEW)
specular bundle list          # List bundles with status (NEW)
```

**Exit codes** (for `bundle gate`):
- 0: OK
- 20: Policy violation
- 30: Drift detected
- 40: Missing approval
- 50: Forbidden provider
- 60: Evaluation failure

#### 5. Plan Commands (REFACTORED)
```bash
specular plan create          # Generate plan (was: plan gen)
specular plan visualize       # Show plan graph (NEW)
specular plan validate        # Validate plan (NEW)
```

#### 6. Generation Commands (REFACTORED)
```bash
specular new                  # Interactive interview (was: interview)
specular generate <input>     # Governed generation
specular generate --unsafe    # Ungoverned mode (with warnings)
```

#### 7. Drift Commands (PROMOTED TO TOP-LEVEL)
```bash
specular drift check          # Check all drift types (was: plan drift)
specular drift approve        # Approve drift as intentional (NEW)
```

**Before**: Nested under `plan drift`
**After**: Top-level category (plan, code, policy drift)

#### 8. Provider Commands (ENHANCED)
```bash
specular provider init        # Create providers.yaml (NEW)
specular provider doctor      # Validate providers (ENHANCED)
specular provider list        # List providers (ENHANCED)
specular provider add         # Allow provider (NEW)
specular provider remove      # Disallow provider (NEW)
```

#### 9. System Commands (UNIFIED)
```bash
specular doctor               # Run all checks (NEW - unified)
specular version              # Show versions (ENHANCED)
specular help                 # Help menu (existing)
```

**Unified `doctor`** runs:
- `provider doctor`
- `governance doctor`
- `drift` baseline check
- `policy validate`

#### 10. Global Flags (STANDARDIZED)
```
--json          # Structured JSON output
--verbose       # Detailed logs
--debug         # Deep diagnostics
--quiet         # Suppress nonessential output
--no-color      # Disable color
--path <dir>    # Alternate workspace
--config <file> # Custom config
```

### Complete Command Mapping

| v1.1.0 (Old) | v1.2.0 (New) | Change Type |
|--------------|--------------|-------------|
| `interview` | `new` | BREAKING RENAME |
| `plan gen` | `plan create` | BREAKING RENAME |
| `plan drift` | `drift check` | BREAKING MOVE |
| `bundle build` | `bundle create` | BREAKING RENAME |
| `bundle verify` | `bundle gate` | BREAKING RENAME |
| `policy apply` | `policy validate` | BREAKING RENAME |
| N/A | `governance init/doctor/status` | NEW CATEGORY |
| N/A | `policy init/approve/list/diff` | NEW COMMANDS |
| N/A | `approve / approvals list/pending` | NEW CATEGORY |
| N/A | `bundle inspect/list` | NEW COMMANDS |
| N/A | `plan visualize/validate` | NEW COMMANDS |
| N/A | `drift approve` | NEW COMMAND |
| N/A | `provider init/add/remove` | NEW COMMANDS |
| N/A | `doctor` (unified) | NEW UNIFIED |

## Rationale

### Why Governance-First?

1. **Market Differentiation**: Governance is the unique value proposition vs "build assistants"
2. **Enterprise Appeal**: CIOs care about governance, developers care about productivity
3. **Competitive Moat**: Policy enforcement + drift detection + approvals = defensible
4. **Clear Mental Model**: "Governance for AI-driven development" is easier to explain than "spec-first tool"
5. **CI/CD Integration**: `bundle gate` exit codes enable policy gates in pipelines

### Why v1.2.0 (Not v2.0.0)?

1. **No users yet**: v1.1.0 has zero external adoption
2. **Faster iteration**: No migration burden, faster validation of governance UX
3. **Marketing alignment**: Launch with governance positioning from day one
4. **Semantic versioning**: While breaking, it's additive feature-wise (not removal)

### Why Breaking Changes Now?

**Opportunity Cost Analysis**:
- **Breaking now**: Zero impact (no users)
- **Breaking later**: Migration burden, backward compatibility complexity, split documentation
- **Never breaking**: Stuck with suboptimal UX forever

**Best practice**: GitLab, Kubernetes, many successful projects do major CLI refactors pre-1.0 or early post-1.0.

## Alternatives Considered

### Alternative 1: Phased Migration (v1.2, v1.3, v2.0)

**Approach**:
- v1.2: Add new commands, keep old as deprecated
- v1.3: Warning messages on deprecated commands
- v2.0: Remove deprecated commands

**Pros**:
- Smoother transition (if users existed)
- More conservative

**Cons**:
- 3x longer timeline (3 releases vs 1)
- Confusing documentation (old + new)
- No users to benefit from migration path
- Delays time-to-market for governance positioning

**Rejected**: Overkill for zero-user scenario

### Alternative 2: Maintain Backward Compatibility Forever

**Approach**:
- Add new commands alongside old
- Keep both `plan gen` AND `plan create`
- Maintain dual documentation

**Pros**:
- Never break anyone

**Cons**:
- CLI bloat (2x commands)
- Confusing for new users (which to use?)
- Technical debt forever
- Split brain in docs/tutorials

**Rejected**: Accumulates UX debt

### Alternative 3: Wait Until v2.0.0

**Approach**:
- Keep current CLI through v1.x line
- Do major refactor at v2.0.0 (6-12 months)

**Pros**:
- "Proper" semver major for breaking changes

**Cons**:
- Governance positioning delayed 6-12 months
- Builds adoption on "wrong" UX, must re-educate
- Marketing launches with spec-first messaging, confuses market

**Rejected**: Misses strategic window

### Alternative 4: Governance Commands Only (Additive)

**Approach**:
- Add `governance`, `policy`, `approve` commands
- Keep existing commands unchanged
- Gradual addition, no breaking changes

**Pros**:
- No breaking changes
- Can start immediately

**Cons**:
- Inconsistent UX (`plan gen` vs `bundle create`)
- Doesn't solve drift/bundle naming issues
- Half-governance, half-spec-first (confusing)
- Still need v2.0 to clean up

**Rejected**: Kicks can down road

## Consequences

### Positive Consequences

1. **Clear Governance Positioning**:
   ✅ CLI structure matches "governance for AI-driven development" message
   ✅ First-time users see governance front-and-center
   ✅ Documentation can focus on governance workflows

2. **Improved UX**:
   ✅ Intuitive command names (`bundle create` vs `bundle build`)
   ✅ Logical categorization (all drift under `drift`, not `plan`)
   ✅ Consistent naming (`create` for generation everywhere)

3. **CI/CD Friendliness**:
   ✅ `bundle gate` with standardized exit codes
   ✅ `--json` flags for machine-readable output
   ✅ Clear gate semantics for pipelines

4. **Enterprise Features**:
   ✅ Approval workflows (`approve`, `approvals pending`)
   ✅ Policy lifecycle (`init`, `validate`, `approve`, `diff`)
   ✅ Governance health (`governance doctor`, unified `doctor`)

5. **Zero Migration Pain**:
   ✅ No users to migrate
   ✅ No backward compatibility code
   ✅ Clean slate for documentation

### Negative Consequences

1. **Documentation Rewrite**:
   ❌ All docs must be updated for new commands
   ❌ CLI reference, getting started, tutorials need rewrites
   ❌ Examples in README, ADRs, guides need updates

   **Mitigation**: Comprehensive search/replace, update all at once

2. **E2E Test Updates**:
   ❌ All E2E tests use old command names
   ❌ Test fixtures, scripts need updates
   ❌ CI/CD workflows might reference old commands

   **Mitigation**: Systematic test refactor, grep for old commands

3. **Internal Confusion**:
   ❌ Team must learn new commands
   ❌ Muscle memory for old commands

   **Mitigation**: Cheat sheet, CLI help is comprehensive

4. **Implementation Effort**:
   ❌ 10+ new commands to implement
   ❌ Refactor existing commands
   ❌ Testing across all categories

   **Acceptance**: One-time cost, permanent benefit

### Implementation Complexity

**Estimated Effort**: 40-60 hours
- Governance commands (init, doctor, status): 8 hours
- Policy commands (5 new): 10 hours
- Approval commands (3 new): 6 hours
- Bundle refactor (rename + 2 new): 5 hours
- Plan refactor (rename + 2 new): 5 hours
- Drift promotion (move + 1 new): 4 hours
- Provider enhancements (3 new): 6 hours
- Unified doctor: 4 hours
- Generation refactor (rename): 2 hours
- Testing & docs: 10-15 hours

## Migration Path

### v1.1.0 → v1.2.0 Breaking Changes

Since there are **zero external users**, migration is internal only:

1. **Update E2E tests**: `internal/cmd/*_test.go`, `test/e2e/*`
2. **Update documentation**: README.md, docs/getting-started.md, docs/CLI_REFERENCE.md
3. **Update examples**: `examples/`, scripts
4. **Update ADRs**: Reference new command names
5. **Update CI/CD**: `.github/workflows/*`

**No external migration guide needed** (no users to migrate).

### Command Translation Table

For internal team reference during transition:

```bash
# OLD (v1.1.0)             → NEW (v1.2.0)
specular interview         → specular new
specular plan gen          → specular plan create
specular plan drift        → specular drift check
specular bundle build      → specular bundle create
specular bundle verify     → specular bundle gate
specular policy apply      → specular policy validate
```

## Implementation Plan

### Phase 1: Foundation (Week 1)
- [x] Create this ADR
- [ ] Create governance command skeleton
- [ ] Implement `governance init`
- [ ] Implement `governance doctor`
- [ ] Implement `governance status`

### Phase 2: Policy & Approvals (Week 2)
- [ ] Implement `policy init`
- [ ] Implement `policy validate`
- [ ] Implement `policy approve`
- [ ] Implement `policy list`
- [ ] Implement `policy diff`
- [ ] Implement `approve` command
- [ ] Implement `approvals list`
- [ ] Implement `approvals pending`

### Phase 3: Bundle & Plan Refactor (Week 2-3)
- [ ] Rename `bundle build` → `bundle create`
- [ ] Rename `bundle verify` → `bundle gate`
- [ ] Implement `bundle inspect`
- [ ] Implement `bundle list`
- [ ] Rename `plan gen` → `plan create`
- [ ] Implement `plan visualize`
- [ ] Implement `plan validate`

### Phase 4: Drift & Generation (Week 3)
- [ ] Move `plan drift` → `drift check`
- [ ] Implement `drift approve`
- [ ] Rename `interview` → `new`
- [ ] Update `generate` command structure

### Phase 5: Provider & System (Week 3)
- [ ] Implement `provider init`
- [ ] Enhance `provider doctor`
- [ ] Enhance `provider list`
- [ ] Implement `provider add`
- [ ] Implement `provider remove`
- [ ] Implement unified `doctor` command

### Phase 6: Testing & Documentation (Week 4)
- [ ] Update all E2E tests
- [ ] Update all documentation
- [ ] Update examples
- [ ] Update README
- [ ] Update CHANGELOG.md
- [ ] Update CLI reference
- [ ] Final QA pass

## Success Criteria

1. ✅ All commands from CLI reference spec implemented
2. ✅ All E2E tests passing with new commands
3. ✅ Documentation fully updated
4. ✅ `bundle gate` exit codes work in CI/CD
5. ✅ Governance workflows (init → policy → bundle → gate) functional
6. ✅ Zero references to old command names in docs

## Related Decisions

- **ADR-0001**: IP Protection & Open-Core Strategy (governance positioning)
- **ADR-0003**: Docker-Only Execution (security/governance)
- **ADR-0008**: Structured Error Handling (CLI UX)
- **ADR-0009**: Observability & Monitoring (governance health)

## References

- **CLI Reference Spec**: `/Users/felixgeelhaar/Downloads/specular_cli_reference.md`
- **Proposed README**: `/Users/felixgeelhaar/Downloads/specular_README.md`
- **Current CLI**: `internal/cmd/*.go`
- **GitLab CLI Evolution**: https://about.gitlab.com/blog/2020/08/27/comparing-cli-experiences/
- **Kubernetes CLI Redesign**: https://kubernetes.io/blog/2018/03/kubectl-ga/

## Decision History

- **2025-11-16**: ADR created and accepted
- **2025-11-16**: v1.2.0 implementation begins

---

**Document Owner**: Product & Engineering Leadership
**Implementation Lead**: Engineering Team
**Last Updated**: 2025-11-16
**Next Review**: Post v1.2.0 release
