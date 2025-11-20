# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records (ADRs) documenting significant technical decisions made in the Specular project.

## What are ADRs?

Architecture Decision Records capture important architectural decisions along with their context and consequences. They help team members and contributors understand:
- **Why** decisions were made
- **What** alternatives were considered
- **What** trade-offs were accepted
- **How** the decision impacts the system

## ADR Index

### [ADR 0001: Spec Lock File Format](./0001-spec-lock-format.md)
**Status:** Accepted | **Date:** 2025-01-07

**Decision:** Use JSON format for spec.lock.json

**Context:** Need to detect when product specifications change after code generation for drift detection.

**Key Points:**
- Chose JSON over YAML, binary, or custom formats
- Deterministic serialization for reliable hashing
- Git-friendly diffs for version control
- Fast parsing with Go stdlib

**Impact:** Enables reliable plan drift detection with minimal overhead

---

### [ADR 0002: Checkpoint and Resume Mechanism](./0002-checkpoint-mechanism.md)
**Status:** Accepted | **Date:** 2025-01-07

**Decision:** Implement auto-saving JSON-based checkpoint system with configurable resume support

**Context:** Code generation can take 5-30 minutes. Failures or interruptions force users to restart from scratch.

**Key Points:**
- Auto-save every 30 seconds during execution
- Resume from last successful task
- Atomic writes for consistency
- Optional cleanup on success

**Impact:**
- Resilient to interruptions (network, Docker, process kills)
- Save time on retries (skip completed tasks)
- Better debugging (inspect state at failure)
- <5% performance overhead

---

### [ADR 0003: Docker-Only Execution for Code Generation](./0003-docker-only-execution.md)
**Status:** Accepted | **Date:** 2025-01-07

**Decision:** All code generation and execution MUST happen inside Docker containers with strict resource limits

**Context:** Generated code from LLMs is untrusted and could execute arbitrary commands (delete files, exfiltrate data, install backdoors).

**Key Points:**
- Zero trust security model
- Defense in depth: sandboxing + resource limits + network restrictions
- Image allowlisting enforced
- Local execution requires explicit opt-in with warnings

**Impact:**
- ✅ Strong security (malicious code cannot access host)
- ✅ Reproducibility (same environment every time)
- ❌ Performance overhead (~100-500ms per container)
- Mitigated with Docker image caching (94% speedup)

---

### [ADR 0004: Provider Abstraction for Multi-LLM Support](./0004-provider-abstraction.md)
**Status:** Accepted | **Date:** 2025-01-07

**Decision:** Implement provider abstraction layer with routing system for intelligent model selection

**Context:** LLM landscape is rapidly evolving with multiple providers offering different capabilities, pricing, and specializations.

**Key Points:**
- Unified Provider interface for all LLMs
- Model hint system (fast, balanced, quality, codegen)
- Automatic fallback on provider failures
- Cost optimization through routing

**Model Hints:**
- **Fast**: Claude 3 Haiku, GPT-3.5 Turbo (<2s, $0.50-$1/1M tokens)
- **Balanced**: GPT-4 Turbo, Claude 3 Sonnet (3-5s, $3-$15/1M tokens)
- **Quality**: Claude 3 Opus, GPT-4 (10-30s, $15-$75/1M tokens)
- **Codegen**: GPT-4, Claude 3 Sonnet (optimized for code)

**Impact:**
- ✅ Flexibility: Users choose provider/model
- ✅ Resilience: Automatic fallback
- ✅ Cost optimization: Route to appropriate model
- ❌ Increased testing complexity

---

### [ADR 0005: Drift Detection and SARIF Output Format](./0005-drift-detection-approach.md)
**Status:** Accepted | **Date:** 2025-01-07

**Decision:** Implement comprehensive drift detection with output in SARIF v2.1.0 format

**Context:** Over time, specifications change, code evolves, and infrastructure policies update. Need continuous validation to prevent implementations from drifting from intent.

**Drift Types:**
1. **Plan Drift**: Spec changes without updating spec.lock
2. **Code Drift**: Implementation doesn't conform to OpenAPI contracts
3. **Infrastructure Drift**: Code violates security/quality policies

**Key Points:**
- SARIF is industry standard (OASIS spec)
- Native GitHub/GitLab CI/CD integration
- Rich metadata (locations, severity, remediation)
- Tool ecosystem (parsers, validators, viewers)

**Finding Codes:**
- Plan: `MISSING_FEATURE_LOCK`, `FEATURE_HASH_MISMATCH`
- Code: `MISSING_API_PATH`, `MISSING_API_METHOD`
- Infra: `DISALLOWED_IMAGE`, `POLICY_VIOLATION`

**Impact:**
- ✅ CI/CD native (GitHub Code Scanning, GitLab Security)
- ✅ Actionable findings with clear context
- ✅ Trend tracking over time
- ❌ Verbose files (~10KB-1MB, mitigated with compression)

---

## ADR Template

New ADRs should follow this structure:

```markdown
# ADR XXXX: [Title]

**Status:** [Proposed | Accepted | Deprecated | Superseded]

**Date:** YYYY-MM-DD

**Decision Makers:** [Team/Individual]

## Context
What is the issue/problem/opportunity?

## Decision
What did we decide to do?

## Alternatives Considered
What other options were evaluated?

## Consequences
What are the results (positive and negative)?

## Related Decisions
Links to related ADRs

## References
External resources, documentation, RFCs
```

## ADR Lifecycle

### Status Definitions
- **Proposed**: Under discussion, not yet accepted
- **Accepted**: Decision made and being implemented
- **Deprecated**: No longer applicable (superseded or invalidated)
- **Superseded**: Replaced by newer ADR

### When to Create an ADR

Create an ADR when making decisions that:
- ✅ Affect system architecture or design
- ✅ Have long-term consequences
- ✅ Are difficult or expensive to change
- ✅ Involve significant trade-offs
- ✅ Impact multiple teams or components
- ✅ Establish patterns or standards

### When NOT to Create an ADR

Don't create ADRs for:
- ❌ Implementation details (can change easily)
- ❌ Temporary workarounds
- ❌ Team processes (use other docs)
- ❌ Trivial decisions with no long-term impact

## Contributing

### Proposing a New ADR

1. **Create a new file**: `docs/adr/XXXX-short-title.md`
   - Use next available number (XXXX)
   - Use lowercase with hyphens for filename

2. **Use the template**: Follow the structure above

3. **Discuss**: Share with team for feedback

4. **Update status**:
   - Start as "Proposed"
   - Change to "Accepted" after team consensus

5. **Update this index**: Add entry to this README

### Updating Existing ADRs

ADRs are **immutable** after acceptance. To change a decision:

1. **Create new ADR**: Don't modify the original
2. **Mark old as superseded**: Update status and link to new ADR
3. **Explain in new ADR**: Why the decision changed

Example:
```markdown
# ADR 0001: Original Decision
**Status:** Superseded by [ADR 0010](./0010-new-decision.md)
```

## Future ADRs

Potential topics for future ADRs:

### Architecture & Design
- [ ] Spec versioning and migration strategy
- [ ] OpenAPI validation approach
- [ ] Plugin system for extensibility
- [ ] Distributed execution coordination

### Security & Compliance
- [ ] Secrets management (Vault, AWS Secrets Manager)
- [ ] Prompt injection protection strategies
- [ ] SBOM generation and tracking
- [ ] SOC 2 compliance implementation

### Performance & Scalability
- [ ] Parallel task execution strategy
- [ ] Database scaling approach
- [ ] Caching strategy for LLM responses
- [ ] Cost optimization algorithms

### Operations & Deployment
- [ ] Multi-region deployment strategy
- [ ] Backup and disaster recovery
- [ ] Monitoring and observability
- [ ] SLA definitions and tracking

### Developer Experience
- [ ] CLI framework selection
- [ ] Configuration management
- [ ] Error handling and messaging
- [ ] Testing strategy

## Resources

### ADR Tools
- [adr-tools](https://github.com/npryce/adr-tools) - Command-line tools for working with ADRs
- [log4brains](https://github.com/thomvaill/log4brains) - Architecture knowledge base with ADRs

### Further Reading
- [Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions) - Michael Nygard
- [ADR GitHub Organization](https://adr.github.io/) - Templates and resources
- [When Should I Write an ADR?](https://engineering.atspotify.com/2020/04/when-should-i-write-an-architecture-decision-record/)

## Questions?

For questions about existing ADRs or proposing new ones:
- Open an issue: [github.com/felixgeelhaar/specular/issues](https://github.com/felixgeelhaar/specular/issues)
- Discuss in PR if related to specific code change
- Ask in team channels for internal decisions
