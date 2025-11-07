<p align="center">
  <img src="docs/assets/logo.svg" alt="Specular Logo" width="400">
</p>

# Specular

**AI-Native Spec and Build Assistant**

A Go-based CLI tool that enables spec-first, policy-enforced software development using AI. Transform natural language product requirements into structured specifications, executable plans, and production-ready code while maintaining traceability and enforcing organizational guardrails.

## Quick Links

📚 **[Getting Started Guide](docs/getting-started.md)** - Complete quickstart tutorial
🎯 **[Examples & Workflows](docs/getting-started.md#common-workflows)** - Real-world use cases
🔧 **[Provider Setup](docs/provider-guide.md)** - Configure AI providers
📖 **[Documentation](#documentation)** - Technical specs and design docs

---

## Features

- **AI Provider Plugin System**: Pluggable architecture for local models (Ollama), cloud APIs (OpenAI, Anthropic, Gemini), and custom providers
- **Intelligent Model Routing**: Smart model selection based on task complexity, budget, and performance constraints
- **Interview Mode**: Guided Q&A to generate best-practice specifications
- **SpecLock**: Canonical, hashed specification snapshots for drift detection
- **Plan Generator**: Converts specs into task DAGs with dependencies
- **Drift Detection**: Multi-level drift detection (plan, code, infrastructure)
- **Policy Engine**: YAML-based guardrail enforcement
- **Docker-Only Sandbox**: Secure isolated execution environment
- **Eval Gate**: Automated tests, linting, coverage, and security checks

## Installation

```bash
# Clone the repository
git clone https://github.com/felixgeelhaar/specular.git
cd specular

# Build from source
make build

# Or install to GOPATH/bin
make install
```

## AI Provider System & Generate Command

The AI provider system enables flexible integration with multiple AI providers (local models, cloud APIs, custom executables) through a pluggable architecture. The `generate` command provides immediate access to AI capabilities with intelligent model routing.

### Quick Setup

```bash
# 1. Initialize provider configuration
./specular provider init

# 2. Edit .specular/providers.yaml to enable desired providers
#    - For local Ollama: set ollama.enabled = true (requires ollama installed)
#    - For OpenAI: set openai.enabled = true and OPENAI_API_KEY env var
#    - For Anthropic: set anthropic.enabled = true and ANTHROPIC_API_KEY env var
#    - For Gemini: set gemini.enabled = true and GEMINI_API_KEY env var

# 3. List configured providers
./specular provider list

# 4. Check provider health
./specular provider health
```

### Generate Command Examples

```bash
# Simple generation with automatic model selection
./specular generate "What is 2 + 2?"

# Fast response with model hint
./specular generate "Count from 1 to 10" --model-hint fast

# Code generation with appropriate model
./specular generate "Write a Go function to reverse a string" --model-hint codegen

# High complexity task with P0 priority (uses most capable model)
./specular generate "Explain microservices architecture" --complexity 8 --priority P0

# With system prompt and temperature control
./specular generate "Tell me a story" \
  --system "You are a creative writer. Keep responses concise." \
  --temperature 0.9 \
  --max-tokens 500

# Verbose mode shows metadata (model, tokens, cost, latency)
./specular generate "What is Go?" --model-hint fast --verbose

# Example verbose output:
# Go is a statically typed, compiled programming language...
#
# ------------------------------------------------------------
# Model:          llama3.2 (ollama)
# Tokens:         42 (in: 12, out: 30)
# Cost:           $0.000000
# Latency:        2.4s
# Selection:      Selected llama3.2: fast model for low complexity task
# Finish Reason:  stop
#
# Budget:         $0.00 spent, $20.00 remaining (0.0% used)
```

### Model Hints

The router automatically selects the best model based on your hints:

- `fast` - Quick responses (llama3.2, claude-haiku, gpt-4o-mini, gemini-flash)
- `codegen` - Code generation (codellama, claude-sonnet, gpt-4o, gemini-flash)
- `agentic` - Complex reasoning (llama3, claude-sonnet-4, gpt-4o, gemini-pro)
- `cheap` - Cost-optimized (local models first, then cloud)
- `long-context` - Large context windows (claude-sonnet, gpt-4-turbo, gemini-pro [1M tokens])

### Provider Management

```bash
# List all configured providers with status
./specular provider list

# Example output:
# NAME         TYPE   ENABLED   SOURCE    VERSION
# ----         ----   -------   ------    -------
# ollama       cli    yes       local     1.0.0
# openai       api    no        builtin   1.0.0
# anthropic    api    no        builtin   1.0.0
# gemini       api    no        builtin   1.0.0
# claude-cli   cli    no        local     1.0.0

# Check health of all enabled providers
./specular provider health

# Check specific provider
./specular provider health ollama

# Example output:
# PROVIDER   STATUS      MESSAGE
# --------   ------      -------
# ollama     ✅ HEALTHY   Executable provider: ./providers/ollama/ollama-provider
```

### Provider Configuration

Providers are configured in `.specular/providers.yaml`:

```yaml
providers:
  # Local Ollama provider (free, requires ollama installed)
  - name: ollama
    type: cli
    enabled: true
    source: local
    config:
      path: ./providers/ollama/ollama-provider
      capabilities:
        streaming: false
        tools: false
        multi_turn: true
        max_context_tokens: 8192
    models:
      fast: llama3.2
      codegen: codellama
      agentic: llama3

  # OpenAI API provider (requires OPENAI_API_KEY)
  - name: openai
    type: api
    enabled: false
    config:
      api_key: ${OPENAI_API_KEY}
      base_url: https://api.openai.com/v1
    models:
      fast: gpt-4o-mini
      codegen: gpt-4o
      long-context: gpt-4-turbo

# Provider selection strategy
strategy:
  preference:
    - ollama      # Try local first (fastest, free)
    - anthropic   # Then cloud APIs
    - openai

  budget:
    max_cost_per_day: 20.0  # USD
    max_cost_per_request: 1.0

  performance:
    max_latency_ms: 60000  # 60 seconds
    prefer_cheap: true     # Prefer cheaper models when quality is similar
```

For detailed provider documentation, see [internal/provider/README.md](internal/provider/README.md).

## Quick Start

### Option 1: Generate spec with interview mode

```bash
# List available presets
specular interview --list

# Run interactive interview (uses cli-tool preset as example)
specular interview --preset cli-tool --out .specular/spec.yaml

# Review the generated spec
cat .specular/spec.yaml
```

### Option 2: Use example spec

```bash
# Use the example spec to get started
cp .specular/spec.yaml.example .specular/spec.yaml
cp .specular/policy.yaml.example .specular/policy.yaml

# Validate the specification
specular spec validate --in .specular/spec.yaml

# Generate SpecLock with blake3 hashes
specular spec lock --in .specular/spec.yaml --out .specular/spec.lock.json

# Build execution plan from spec
specular plan --in .specular/spec.yaml --lock .specular/spec.lock.json --out plan.json

# Execute build with policy enforcement (dry-run)
specular build --plan plan.json --policy .specular/policy.yaml --dry-run

# Run drift detection (plan + code + infrastructure)
specular eval --plan plan.json --lock .specular/spec.lock.json --spec .specular/spec.yaml \
  --policy .specular/policy.yaml --report drift.sarif

# With all drift detection options
specular eval --plan plan.json --lock .specular/spec.lock.json --spec .specular/spec.yaml \
  --policy .specular/policy.yaml --api-spec api/openapi.yaml \
  --ignore "*.test.go" --ignore "vendor/**" \
  --report drift.sarif --fail-on-drift

# Run the full end-to-end test
./test-e2e.sh
```

### Working Features (v0.8)

✅ **AI Provider Plugin System** (Phase 1 Complete)
- Pluggable provider architecture (CLI executables, API clients, gRPC, native Go plugins)
- Provider registry with lifecycle management
- Support for local models (Ollama), cloud APIs (OpenAI, Anthropic, Gemini), and custom providers
- Full streaming support for all API providers via Server-Sent Events (SSE)
- Native Go HTTP clients for all three major cloud providers
- Provider health checking and capability discovery
- YAML-based provider configuration with environment variable expansion
- Automatic provider loading from configuration
- Trust levels (builtin, verified, community) for security

✅ **Intelligent Model Router**
- Multi-model selection based on task complexity, priority, and hints
- Budget tracking and cost management ($0.00 for local models)
- Model selection by hint (codegen, agentic, long-context, fast, cheap)
- Latency-aware routing with configurable constraints
- Provider preference ordering (try local first, then cloud)
- Dynamic model availability based on loaded providers
- Usage statistics and detailed reporting
- **Retry with exponential backoff** - Automatic retry of failed requests with increasing delays
- **Provider fallback** - Cascades to alternative providers when primary fails
- **Context window management** - Validates requests fit model limits, optional auto-truncation with 4 strategies

✅ **Generate Command**
- Direct AI content generation from CLI
- Automatic model selection based on task characteristics
- Support for system prompts and temperature control
- Verbose mode with detailed metadata (model, tokens, cost, latency, selection reason)
- Streaming support for compatible providers
- Error handling with helpful messages

✅ **Interview Mode**
- Guided Q&A to generate specifications
- 5 preset templates (web-app, api-service, cli-tool, microservice, data-pipeline)
- Automatic spec generation from answers
- Question skip logic based on previous answers
- Strict validation mode

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
- Code drift (test coverage, API conformance, file tracking)
- Infrastructure drift (policy compliance, resource validation)
- OpenAPI 3.x contract validation
- Endpoint and method verification
- Path parameter matching
- Docker image allowlist validation (exact, wildcard, prefix matching)
- Execution policy validation (network, resources, tests, security)
- Run manifest validation
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

✅ **Eval Gate**
- Automated quality gate with policy enforcement
- Test execution with race detection and coverage analysis
- Coverage threshold validation (enforce minimum coverage requirements)
- Multi-language linter integration (Go, JavaScript, Python)
- Secrets scanning with gitleaks
- Dependency vulnerability scanning
- Comprehensive reporting with pass/fail status
- Early failure detection (fail fast on quality issues)

✅ **Test Coverage**
- 81.4% - 100% across packages (policy: 100%, drift: 92.4%, plan: 91.6%, interview: 89.1%, spec: 87.8%, exec: 87.1%, eval: 85.0%, provider: 81.4%, router: 80.4%)
- Race detection enabled
- Table-driven test patterns
- End-to-end integration test
- Interview flow testing
- Model selection and budget tests
- Code drift and OpenAPI validation tests
- Infrastructure drift and policy compliance tests
- SARIF report generation and round-trip verification tests
- Eval gate integration tests (RunEvalGate with tests, coverage, Go/JavaScript/Python linters, secrets scan, dependency scan, multi-linter scenarios, failing linter scenarios)
- Eval test runner tests (Go test execution, coverage validation, linter integration, secrets scanning with gitleaks integration)
- Exec package tests (executor orchestration, Docker runner, policy enforcement, manifest generation, image pulling with automatic image pull on first use, image existence checks)
- Policy package tests (YAML loading, validation, default policy, error handling)
- Interview package tests (spec generation, answer helpers, goals/features/milestones builders, completion flow)
- Spec package tests (spec loading/saving, lock generation/loading/saving, round-trip verification, Hash and sortKeys function tests with nested maps and slices, canonicalize with multiple APIs and edge cases)
- Router package tests (config loading/validation/saving, model selection helpers, budget tracking, cheaper model search, provider integration, model availability, SelectModel with providers)
- Provider package tests (configuration loading/validation/saving, registry operations, executable provider lifecycle, OpenAI, Anthropic, and Gemini API providers with mock HTTP servers, streaming support with SSE, role mapping, error handling, health checks, integration tests for all four provider types - CLI/OpenAI/Anthropic/Gemini, multi-provider registry, config round-trip)
- Plan package tests (plan loading/saving, round-trip verification, JSON parsing, file I/O error handling)

## Project Structure

```
specular/
 ├─ cmd/specular/          # Main CLI entry point
 ├─ internal/
 │   ├─ cmd/             # Cobra command implementations (generate, provider, etc.)
 │   ├─ provider/        # AI provider plugin system (registry, executables, config)
 │   ├─ router/          # Intelligent model routing and selection
 │   ├─ interview/       # Q&A engine for spec generation
 │   ├─ spec/            # Specification management
 │   ├─ plan/            # Task DAG generation
 │   ├─ drift/           # Drift detection engine
 │   ├─ policy/          # Policy enforcement
 │   ├─ exec/            # Docker sandbox execution
 │   ├─ eval/            # Quality gate and test execution
 │   └─ tools/           # External tool integrations
 ├─ providers/           # Provider implementations
 │   └─ ollama/          # Ollama provider wrapper
 ├─ docs/                # Documentation (PRD, tech design)
 └─ .specular/               # Workspace (specs, plans, logs, providers.yaml)
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

### Policy File (.specular/policy.yaml)

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
  secrets_scan: true     # Scan for secrets using gitleaks (if available)
  dep_scan: true         # Scan for vulnerabilities using govulncheck (if available)
```

**Security Scans:**
- `secrets_scan`: Uses [gitleaks](https://github.com/gitleaks/gitleaks) to detect hardcoded secrets and credentials. If gitleaks is not installed, the scan is skipped.
- `dep_scan`: Uses [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) to scan dependencies for known vulnerabilities from the Go vulnerability database. Falls back to basic dependency listing if govulncheck is not available.

### Provider Configuration (.specular/providers.yaml)

```yaml
providers:
  # Local Ollama provider (free, requires ollama installed)
  - name: ollama
    type: cli
    enabled: true
    source: local
    version: "1.0.0"
    config:
      path: ./providers/ollama/ollama-provider
      trust_level: community
      capabilities:
        streaming: false
        tools: false
        multi_turn: true
        max_context_tokens: 8192
    models:
      fast: llama3.2
      codegen: codellama
      cheap: llama3.2
      agentic: llama3

  # OpenAI API provider (requires OPENAI_API_KEY)
  - name: openai
    type: api
    enabled: false
    source: builtin
    version: "1.0.0"
    config:
      api_key: ${OPENAI_API_KEY}
      base_url: https://api.openai.com/v1
      capabilities:
        streaming: true
        tools: true
        multi_turn: true
        max_context_tokens: 128000
    models:
      fast: gpt-4o-mini
      codegen: gpt-4o
      long-context: gpt-4-turbo
      cheap: gpt-4o-mini
      agentic: gpt-4o

  # Anthropic Claude API provider (requires ANTHROPIC_API_KEY)
  - name: anthropic
    type: api
    enabled: false
    source: builtin
    version: "1.0.0"
    config:
      api_key: ${ANTHROPIC_API_KEY}
      base_url: https://api.anthropic.com/v1
      capabilities:
        streaming: true
        tools: true
        multi_turn: true
        max_context_tokens: 200000
    models:
      fast: claude-haiku-3.5
      codegen: claude-sonnet-3.5
      agentic: claude-sonnet-4
      long-context: claude-sonnet-3.5

  # Google Gemini API provider (requires GEMINI_API_KEY)
  - name: gemini
    type: api
    enabled: false
    source: builtin
    version: "1.0.0"
    config:
      api_key: ${GEMINI_API_KEY}
      base_url: https://generativelanguage.googleapis.com/v1beta
      capabilities:
        streaming: true
        tools: true
        multi_turn: true
        vision: true
        max_context_tokens: 1000000
    models:
      fast: gemini-2.0-flash-exp
      codegen: gemini-2.0-flash-exp
      agentic: gemini-2.5-pro-exp-03
      long-context: gemini-2.5-pro-exp-03

# Provider selection strategy
strategy:
  # Prefer providers in this order when multiple are available
  preference:
    - ollama      # Local first (fastest, free)
    - claude-cli  # Local Claude CLI (free)
    - anthropic   # Cloud API (high quality)
    - openai      # Cloud API (fallback)

  # Budget constraints
  budget:
    max_cost_per_day: 20.0  # USD
    max_cost_per_request: 1.0  # USD

  # Performance requirements
  performance:
    max_latency_ms: 60000  # 60 seconds
    prefer_cheap: true      # Prefer cheaper models when quality is similar

  # Fallback behavior
  fallback:
    enabled: true
    max_retries: 3
    retry_delay_ms: 1000
    fallback_model: ollama/llama3.2
```

**For detailed documentation, see:**
- [Provider System Guide](internal/provider/README.md) - Provider configuration and custom provider development
- [Provider Selection & Routing Guide](docs/provider-guide.md) - Model selection, retry/fallback, context management, streaming

## Core Principles

- **Spec-first**: SpecLock is the single source of truth
- **Traceability**: Stable links across spec → plan → code/tests
- **Governance**: All execution passes policy gates
- **Reproducibility**: Every run emits hashes, costs, and provenance

## Documentation

- **[Getting Started Guide](docs/getting-started.md)** - Quickstart tutorial and common workflows
- [Product Requirements Document](docs/prd.md) - Product vision and requirements
- [Technical Design Document](docs/tech_design.md) - Architecture and implementation
- [Provider Guide](docs/provider-guide.md) - AI provider configuration and routing
- [Homebrew Tap Setup](docs/homebrew-tap-setup.md) - Distribution via Homebrew
- [Development Guide](CLAUDE.md) - Contributing and development setup

## License

MIT

## Contributing

Contributions are welcome! Please read the development guide in CLAUDE.md before submitting pull requests.
