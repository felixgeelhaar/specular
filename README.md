# ai-dev

**AI-Native Spec and Build Assistant**

A Go-based CLI tool that enables spec-first, policy-enforced software development using AI. Transform natural language product requirements into structured specifications, executable plans, and production-ready code while maintaining traceability and enforcing organizational guardrails.

## Features

- **Interview Mode**: Guided Q&A to generate best-practice specifications
- **SpecLock**: Canonical, hashed specification snapshots for drift detection
- **Plan Generator**: Converts specs into task DAGs with dependencies
- **Drift Detection**: Multi-level drift detection (plan, code, infrastructure)
- **Policy Engine**: YAML-based guardrail enforcement
- **Docker-Only Sandbox**: Secure isolated execution environment
- **Multi-Model Router**: Smart AI model selection per task
- **Eval Gate**: Automated tests, linting, coverage, and security checks

## Installation

```bash
# Clone the repository
git clone https://github.com/felixgeelhaar/ai-dev.git
cd ai-dev

# Build from source
make build

# Or install to GOPATH/bin
make install
```

## Quick Start

```bash
# Use the example spec to get started
cp .aidv/spec.yaml.example .aidv/spec.yaml
cp .aidv/policy.yaml.example .aidv/policy.yaml

# Validate the specification
ai-dev spec validate --in .aidv/spec.yaml

# Generate SpecLock with blake3 hashes
ai-dev spec lock --in .aidv/spec.yaml --out .aidv/spec.lock.json

# Build execution plan from spec
ai-dev plan --in .aidv/spec.yaml --lock .aidv/spec.lock.json --out plan.json

# Execute build with policy enforcement (dry-run)
ai-dev build --plan plan.json --policy .aidv/policy.yaml --dry-run

# Run drift detection
ai-dev eval --plan plan.json --lock .aidv/spec.lock.json --report drift.sarif

# Run the full end-to-end test
./test-e2e.sh
```

### Working Features (v0.2)

✅ **Spec Management**
- Validate YAML specifications
- Generate SpecLock with blake3 hashes
- Load/save specs and locks

✅ **Plan Generation**
- Convert specs to task DAGs
- Automatic dependency inference based on priority (P0 → P1 → P2)
- Skill assignment (go-backend, ui-react, infra, database, testing)
- Model hints (long-context, agentic, codegen)
- Complexity estimation (1-10 scale)

✅ **Build Execution**
- Docker-only sandbox execution
- Policy enforcement (image allowlist, network isolation, resource limits)
- Dependency-aware task execution
- Run manifest generation with SHA-256 hashes
- Dry-run mode for validation
- Real Docker execution with image pulling

✅ **Drift Detection**
- Plan drift (hash mismatches)
- SARIF 2.1.0 report generation
- Error/warning/info severity levels
- CI/CD integration ready

✅ **Policy Engine**
- YAML-based policy configuration
- Docker-only enforcement
- Image allowlist with wildcard patterns (e.g., `golang:*`)
- Network mode validation (default: none)
- Resource limits (CPU, memory)
- Tool configuration validation

✅ **Test Coverage**
- 40.5% - 78.3% across packages
- Race detection enabled
- Table-driven test patterns
- End-to-end integration test

## Project Structure

```
ai-dev/
 ├─ cmd/ai-dev/          # Main CLI entry point
 ├─ internal/
 │   ├─ cmd/             # Cobra command implementations
 │   ├─ interview/       # Q&A engine for spec generation
 │   ├─ spec/            # Specification management
 │   ├─ plan/            # Task DAG generation
 │   ├─ drift/           # Drift detection engine
 │   ├─ policy/          # Policy enforcement
 │   ├─ exec/            # Docker sandbox execution
 │   ├─ router/          # AI model routing
 │   └─ tools/           # External tool integrations
 ├─ docs/                # Documentation (PRD, tech design)
 └─ .aidv/               # Workspace (specs, plans, logs)
```

## Development

```bash
# Run tests
make test

# Run tests with coverage report
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Run all checks (fmt, lint, test)
make check

# Clean build artifacts
make clean
```

## Configuration

### Policy File (.aidv/policy.yaml)

```yaml
execution:
  allow_local: false
  docker:
    required: true
    image_allowlist:
      - ghcr.io/acme/go-builder:1.22
    cpu_limit: "2"
    mem_limit: "2g"
    network: "none"
linters:
  go: { enabled: true, cmd: "golangci-lint run" }
tests:
  require_pass: true
  min_coverage: 0.70
security:
  secrets_scan: true
  dep_scan: true
```

### Provider Configuration (~/.ai-dev/config.yaml)

```yaml
providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    models: { agentic: "claude-sonnet" }
  openai:
    api_key: ${OPENAI_API_KEY}
    models: { code: "gpt-4" }
routing:
  budget_usd: 20
  max_latency_ms: 60000
```

## Core Principles

- **Spec-first**: SpecLock is the single source of truth
- **Traceability**: Stable links across spec → plan → code/tests
- **Governance**: All execution passes policy gates
- **Reproducibility**: Every run emits hashes, costs, and provenance

## Documentation

- [Product Requirements Document](docs/prd.md)
- [Technical Design Document](docs/tech_design.md)
- [Development Guide](CLAUDE.md)

## License

MIT

## Contributing

Contributions are welcome! Please read the development guide in CLAUDE.md before submitting pull requests.
