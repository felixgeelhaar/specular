# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**specular** is a Go-based CLI tool that enables spec-first, policy-enforced software development using AI. It transforms natural language product requirements into structured specifications, executable plans, and production-ready code while maintaining traceability and enforcing organizational guardrails.

### Core Principles

- **Spec-first**: SpecLock (`.specular/spec.lock.json`) is the single source of truth
- **Traceability**: Stable links across spec → plan → code/tests with hash-based verification
- **Governance**: All execution passes policy gates (Docker-only, linters, tests, coverage, security)
- **Reproducibility**: Every run emits hashes, costs, and provenance for audit trails

## Architecture

### Directory Structure

```
specular/
 ├─ cmd/                 # Cobra commands (interview, spec, plan, build, eval)
 ├─ internal/
 │   ├─ interview/       # Q&A engine (slot-filling, templates, validators)
 │   ├─ spec/            # schema, canonicalizer, SpecLock, OpenAPI/test stubs
 │   ├─ plan/            # DAG generator (tickets, deps, estimates)
 │   ├─ drift/           # plan/spec/code drift detectors + SARIF
 │   ├─ policy/          # YAML + (optional) OPA/CUE adapters
 │   ├─ exec/            # Docker runner (limits, logs, sandboxing)
 │   ├─ router/          # model routing (rule-based → learned)
 │   └─ tools/           # linters, formatters, tests, codegen adapters
 ├─ .specular/               # workspace: policy.yaml, spec.lock.json, runs/, logs/
 └─ docs/                # PRD and technical design documentation
```

### Core Workflow

The system follows a linear pipeline with strict validation at each stage:

```
[PRD.md or Interview] → spec.generate → .specular/spec.yaml + .specular/spec.lock.json
                              ↓
                       plan.build → plan.json (DAG w/ ExpectedHash)
                              ↓
                       build.run → Docker sandbox (policy-enforced)
                              ↓
                        eval.run → tests, lint, security, drift
                              ↓
                  reports → drift.sarif + scorecard.json + run manifests
```

## Key Components

### interview/ - Guided Specification Generation
Implements finite-state slot-filling to guide product managers from zero to best-practice spec. Uses critical slots first with adaptive follow-ups. Outputs schema-conformant `spec.yaml` and `interview.log.jsonl` for audit.

### spec/ - Specification Management
- Schema: JSON Schema (draft 2020-12) for `ProductSpec`
- Canonicalization: Normalizes ordering/whitespace for stable hashing
- SpecLock: `.specular/spec.lock.json` with per-feature blake3 hash, generated OpenAPI, and acceptance test stubs
- Validation: Uses `gojsonschema` with optional CUE semantics

### plan/ - Task DAG Generation
Converts `spec.lock.json` into `plan.json` containing tasks, dependencies, estimates, and `ExpectedHash` per feature. Generates topological order with priority (P0/P1/P2), skill tags, and model hints.

### drift/ - Multi-Level Drift Detection
- **Plan drift**: Compare DAG `ExpectedHash` vs SpecLock hash
- **Code drift**: Contract tests (OpenAPI/golden), AST/interface checks, route/status/type conformance
- **Infra drift**: Policy validation against compose/K8s manifests
- Reports: SARIF format with human summary and spec `trace` backrefs

### policy/ - Guardrail Enforcement
YAML-based policy for common guardrails with optional OPA/CUE for complex org rules. Performs preflight checks before any step; hard-fails on violations. Covers execution (Docker-only, image allowlist, net limits), linters/formatters, tests/coverage, security scans, and allowed models/tools.

### exec/ - Secure Sandboxed Execution
Docker-only runner enforcing network controls, CPU/mem limits, read-only FS, `cap-drop=ALL`, `pids-limit`. Uses ephemeral workdir mounts and logs all commands, images, env, exit codes, stdout/stderr. Emits `.specular/runs/<ts>.json` manifests with input/output hashes.

### router/ - Multi-Model AI Routing
- Phase 1: Rule-based routing (task kind, token budget, latency, cost)
- Phase 2: Confidence-aware retries/upgrades
- Phase 3: Learned router using run telemetry

### tools/ - External Tool Integration
Adapters for `golangci-lint`, `eslint`/`prettier`, unit/integration test runners, `semgrep`/dep scanners, codegen writers, OpenAPI validators.

## Data Models

### ProductSpec
Core specification structure containing:
- Product name and goals
- Features with ID, title, description, priority (P0/P1/P2), API definitions, success criteria, and trace references
- Non-functional requirements
- Acceptance criteria
- Milestones

### SpecLock
Canonical, hashed snapshot of specifications:
- Version identifier
- Features map with per-feature blake3 hash, OpenAPI path, and test paths
- Immutable reference for drift detection

### Plan
Task DAG structure:
- Tasks with ID, feature ID, expected hash, dependencies, skill type, priority, and model hints
- Topological ordering for execution
- Hash-based linkage to SpecLock

### Policy
Enforcement configuration:
- Execution constraints (Docker-only, image allowlist, resource limits, network controls)
- Linter/formatter configurations
- Test requirements (pass/fail, coverage thresholds)
- Security scanning (secrets, dependencies)
- Model routing rules (allowed models, denied tools)

## CLI Commands

```bash
# Interactive interview mode to generate spec from Q&A
specular interview --out .specular/spec.yaml [--preset saas-api|mobile-app|internal-tool] [--strict] [--tui]

# Generate spec from PRD markdown
specular spec generate --in PRD.md --out .specular/spec.yaml

# Build execution plan from spec
specular plan --in .specular/spec.yaml --out plan.json

# Execute build with policy enforcement
specular build --plan plan.json [--policy .specular/policy.yaml] [--dry-run]

# Run evaluation and drift detection
specular eval --plan plan.json --report drift.sarif
```

### Common Flags
- `--fail-on drift,lint,test,security` - Set failure conditions
- `--no-apply` - Explain proposed actions without modifying repo
- `--runner docker` - Default runner (local disabled unless policy allows)

## Critical Algorithms

### Spec Canonicalization & Hashing
Features are marshaled to JSON with stable ordering, then hashed using blake3 to produce `SpecLock.Features[id].Hash`. This ensures deterministic hashing regardless of input ordering.

### Plan Drift Detection
```go
func DetectPlanDrift(lock SpecLock, plan Plan) []Finding {
  // For each task in plan:
  //   1. Verify FeatureID exists in SpecLock
  //   2. Compare task.ExpectedHash with lock.Features[id].Hash
  //   3. Report UNKNOWN_FEATURE or HASH_MISMATCH findings
}
```

### Code Drift Detection
Generates contract tests from OpenAPI and `acceptance[]` criteria. Validates routes, status codes, and request/response schemas. For Go backends, uses reflection to compare handlers against OpenAPI shapes.

### Policy Enforcement Gate
```go
func EnforcePolicy(step Step, pol Policy) error {
  // Validates:
  //   1. Docker-only requirement if !pol.Execution.AllowLocal
  //   2. Image allowlist for Docker steps
  //   3. Network profile compliance
  //   4. Tool denylist
  //   5. Coverage thresholds
}
```

### Docker Sandbox Execution
Command pattern:
```bash
docker run --rm \
  --network <profile> \
  --cpus <cpu> --memory <mem> \
  --read-only --pids-limit 256 --cap-drop ALL \
  -v <workdir>:/workspace -w /workspace \
  <image> <cmd...>
```
Captures logs, exit code, and emits run manifest for audit.

## Configuration Examples

### .specular/policy.yaml
```yaml
execution:
  allow_local: false
  docker:
    required: true
    image_allowlist:
      - ghcr.io/acme/go-builder:1.22
      - node:22
    cpu_limit: "2"
    mem_limit: "2g"
    network: "none"
linters:
  go: { enabled: true, cmd: "golangci-lint run" }
  ts: { enabled: true, cmd: "pnpm eslint ." }
formatters:
  go: { enabled: true, cmd: "gofmt -w ." }
  ts: { enabled: true, cmd: "pnpm prettier -w ." }
tests:
  require_pass: true
  min_coverage: 0.70
security:
  secrets_scan: true
  dep_scan: true
routing:
  allow_models:
    - provider: anthropic
      names: ["claude-3.5-sonnet", "claude-4.x-sonnet"]
    - provider: openai
      names: ["gpt-4.1", "gpt-5-codex"]
  deny_tools: ["shell_local"]
```

### ~/.specular/config.yaml
```yaml
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    models: { long_context: "gpt-4.1", code: "gpt-5-codex" }
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    models: { agentic: "claude-sonnet" }
  google:
    api_key: ${GOOGLE_API_KEY}
    models: { long_context: "gemini-2.5-pro", fast: "gemini-2.5-flash" }
routing:
  budget_usd: 20
  max_latency_ms: 60000
  upgrade_policy: "confidence<0.7 || tests_fail"
```

## Security & Compliance

- **Data minimization**: Redact secrets before LLM calls; support offline mode
- **Sandboxing**: Docker with least privilege; no default network access
- **Supply chain**: Image allowlists; pinned tool versions
- **Auditability**: Per-run manifests with hashes, model selection, and costs
- **PII/compliance**: Configurable rules (GDPR logging, retention policies)

## Development Roadmap

| Milestone | Deliverables |
|-----------|--------------|
| M1: Foundations | Cobra CLI, spec schema, canonicalizer, SpecLock, hashing |
| M2: Interview MVP | Slot engine, validators, presets, spec.yaml output |
| M3: Policy + Sandbox | YAML policy, Docker runner, preflight gates |
| M4: Drift + Eval | Plan/code drift, OpenAPI contracts, SARIF |
| M5: Routing Alpha | Provider adapters, rule-based router |
| M6: E2E Alpha | spec→plan→build→eval pipeline |
| M7: CI Integration | GitHub Action, failure annotations |
| M8: Beta Hardening | Performance, docs, samples, extension hooks |

## Key Success Metrics

- Avg. spec completeness score: ≥ 0.85
- Drift incidents caught automatically: ≥ 95%
- Policy violations prevented pre-run: 100%
- Avg. setup time for new product spec: < 15 minutes
- User satisfaction (PMs / Tech Leads): ≥ 8 / 10

## Development Guidelines

When implementing features:

1. **Maintain Traceability**: Ensure all artifacts (specs, plans, code) maintain hash-based links
2. **Enforce Policies**: All execution paths must pass through policy gates
3. **Stable Hashing**: Use canonicalization before hashing to ensure reproducibility
4. **Docker-First**: Default to sandboxed execution; local execution requires explicit policy opt-in
5. **Schema Validation**: Validate all specs against JSON Schema before processing
6. **Drift Prevention**: Generate contract tests and acceptance criteria from specs
7. **Audit Everything**: Emit manifests for all runs with hashes, costs, and provenance
8. **Fail Fast**: Hard-fail on policy violations; no silent degradation
