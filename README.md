<p align="center">
  <img src="docs/assets/specular_logo.png" alt="Specular Logo" width="400">
</p>

# Specular

**AI-Native Spec and Build Assistant with Governance**

A Go-based CLI tool that enables spec-first, policy-enforced software development using AI. Transform natural language product requirements into structured specifications, executable plans, and production-ready code while maintaining traceability and enforcing organizational guardrails.

## Why Specular?

Most teams are adopting AI for ideation, planning, code generation, and automation. But they lack:

- **Governance** for what AI may do and how decisions are made
- **Policy enforcement** across providers and environments
- **Drift detection** between requirements, plans, and implementation
- **Cost and risk controls** for AI usage
- **Auditable artifacts** with cryptographic integrity
- **Reproducible workflows** from spec to production

**Specular solves this** by providing:

✅ **Spec-First Development**: Transform requirements into formal specifications with AI-assisted interview mode
✅ **Governance & Policy**: Enterprise-grade policy engine with cryptographic approvals and bundle workflows
✅ **Multi-Provider AI**: Intelligent routing across local (Ollama) and cloud (OpenAI, Anthropic, Gemini) models
✅ **Drift Detection**: Continuous validation of spec → plan → code alignment with SARIF reporting
✅ **Docker Sandboxing**: Secure isolated execution with resource limits and image allowlisting
✅ **Autonomous Mode**: Checkpoint/resume for long-running workflows with full state preservation
✅ **Audit & Compliance**: Cryptographic attestations, trace logging, and approval workflows

> **Specular is the control plane and audit layer for AI-driven development.**
> It replaces "wild west prompting" with structured, governed, policy-compliant workflows.

## Quick Links

📚 **[Getting Started](docs/getting-started.md)** – Quickstart plus common workflows
🛠️ **[Installation Guide](docs/installation.md)** – Package, binary, and Docker installs
🔧 **[Provider Guide](docs/provider-guide.md)** – Configure local/cloud AI providers
📘 **[CLI Reference](docs/CLI_REFERENCE.md)** – Command/flag reference
📦 **[Bundle User Guide](docs/BUNDLE_USER_GUIDE.md)** – Governed bundle workflows
🚀 **[Production Guide](docs/PRODUCTION_GUIDE.md)** – Production deployment, security, monitoring

---

## What's New in v1.6.0 🎉

### 🚀 GitHub Action for CI/CD Integration

Seamless integration with GitHub Actions for automated drift detection and policy enforcement:

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    command: drift
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

**Key Features:**
- Four commands: `drift`, `eval`, `build`, `plan`
- Multi-provider AI support (Anthropic, OpenAI, Google)
- Automatic SARIF upload to GitHub Code Scanning
- Platform auto-detection (Linux/macOS, AMD64/ARM64)
- Rich job summaries and PR comment integration
- Comprehensive [Action documentation](.github/ACTION_README.md)

### 🎨 Interactive Plan Review TUI

Full-featured terminal UI for reviewing execution plans before approval:

- **Two-view system**: List view for overview, detail view for task inspection
- **Vim-style navigation**: j/k, h/l, Enter, Esc for efficient interaction
- **Approve/reject workflow**: Rejection reason prompts with full audit trail
- **Auto-approve**: Empty plans auto-approved for convenience
- **Styled interface**: Professional appearance with lipgloss theming
- **11 comprehensive tests**: Ensuring reliability across all scenarios

### 🌐 Platform API Client v2.0

Production-grade HTTP client for Specular Platform integration:

- **Configurable retry logic**: Exponential backoff with smart retry strategy
- **Intelligent routing**: Retries 5xx errors, fails fast on 4xx
- **Context propagation**: Request cancellation and timeout handling
- **Structured errors**: APIError type with request ID tracking
- **Three endpoints**: Health, GenerateSpec, GeneratePlan
- **14 comprehensive tests**: Including retry scenarios and edge cases

### 🔌 Plugin System Enhancements

Extended plugin capabilities for maximum flexibility:

- **Local installation**: Install plugins from directories
- **GitHub installation**: Direct plugin installation from repositories
- **Automatic resolution**: Dependency resolution for plugin chains
- **Five plugin types**: Provider, validator, formatter, hook, notifier

### 📦 Distribution & Availability

**Homebrew Installation:**
```bash
brew tap felixgeelhaar/specular
brew install specular
```

**Shell Completions:** Bash, Zsh, Fish completions included in all releases

**Multi-platform Binaries:** Linux/macOS/Windows on AMD64/ARM64

### 📚 Enhanced Documentation

- **ACTION_README.md**: Complete GitHub Action integration guide (9.7KB)
- **Tutorial guides**: Step-by-step PRO feature walkthroughs
- **Advanced workflows**: Production-grade deployment patterns
- **300KB+ docs**: Comprehensive coverage of all features

---

**Previous Release: v1.2.0** introduced governance-first CLI redesign with policy management, approval workflows, and backward-compatible command structure. [View details in CHANGELOG](CHANGELOG.md#120---2025-11-17).

[View Full Changelog](CHANGELOG.md)

---

## Features

### Core Capabilities

- **AI Provider Plugin System**: Pluggable architecture for local models (Ollama), cloud APIs (OpenAI, Anthropic, Gemini), and custom providers
- **CLI Provider Protocol**: Language-agnostic protocol for creating custom AI providers (covered in the [Provider Guide](docs/provider-guide.md))
- **Intelligent Model Routing**: Smart model selection based on task complexity, budget, and performance constraints
- **Interview Mode**: Guided Q&A with interactive TUI to generate best-practice specifications
- **Enhanced Error System**: Structured errors with error codes, suggestions, and documentation links
- **SpecLock**: Canonical, hashed specification snapshots for drift detection
- **Plan Generator**: Converts specs into task DAGs with dependencies
- **Drift Detection**: Multi-level drift detection (plan, code, infrastructure)
- **Docker-Only Sandbox**: Secure isolated execution environment

### Advanced Features

- **Governance Workflows** (v1.2.0): Enterprise-grade governance with workspace initialization, health checks, and status monitoring
- **Policy Management** (v1.2.0): Full policy lifecycle with init, validate, approve, list, and diff commands
- **Approval Workflows** (v1.2.0): Role-based approvals for plans, builds, and drift with audit trails
- **Cryptographic Attestations** (v1.2.0): ECDSA P-256 signatures for build artifacts and policy changes
- **Autonomous Mode** (v1.0.0): Checkpoint/resume capabilities for long-running sessions with full state preservation
- **Routing Intelligence** (v1.0.0): Provider selection optimization with cost tracking and task-type routing

### Testing & Quality Assurance

- **Eval Gate**: Automated tests, linting, coverage, and security checks
- **Policy Engine**: YAML-based guardrail enforcement
- **Approval Workflows**: SHA256-based governance signatures for artifacts

## Installation

### macOS / Linux (Homebrew)

```bash
# Add the tap
brew tap felixgeelhaar/tap

# Install specular
brew install specular

# Verify installation
specular version
```

### Linux Packages

#### Debian/Ubuntu (.deb)

```bash
# Download the latest .deb package
wget https://github.com/felixgeelhaar/specular/releases/latest/download/specular_linux_amd64.deb

# Install the package
sudo dpkg -i specular_linux_amd64.deb

# Or for ARM64
wget https://github.com/felixgeelhaar/specular/releases/latest/download/specular_linux_arm64.deb
sudo dpkg -i specular_linux_arm64.deb
```

#### RedHat/Fedora/CentOS (.rpm)

```bash
# Download the latest .rpm package
wget https://github.com/felixgeelhaar/specular/releases/latest/download/specular_linux_amd64.rpm

# Install the package
sudo rpm -i specular_linux_amd64.rpm

# Or for ARM64
wget https://github.com/felixgeelhaar/specular/releases/latest/download/specular_linux_arm64.rpm
sudo rpm -i specular_linux_arm64.rpm
```

#### Alpine Linux (.apk)

```bash
# Download the latest .apk package
wget https://github.com/felixgeelhaar/specular/releases/latest/download/specular_linux_amd64.apk

# Install the package
sudo apk add --allow-untrusted specular_linux_amd64.apk

# Or for ARM64
wget https://github.com/felixgeelhaar/specular/releases/latest/download/specular_linux_arm64.apk
sudo apk add --allow-untrusted specular_linux_arm64.apk
```

### Direct Binary Download

Download pre-built binaries from the [releases page](https://github.com/felixgeelhaar/specular/releases):

```bash
# Linux AMD64
wget https://github.com/felixgeelhaar/specular/releases/latest/download/specular_linux_amd64.tar.gz
tar -xzf specular_linux_amd64.tar.gz
sudo mv specular /usr/local/bin/

# macOS AMD64 (Intel)
wget https://github.com/felixgeelhaar/specular/releases/latest/download/specular_darwin_amd64.tar.gz
tar -xzf specular_darwin_amd64.tar.gz
sudo mv specular /usr/local/bin/

# macOS ARM64 (Apple Silicon)
wget https://github.com/felixgeelhaar/specular/releases/latest/download/specular_darwin_arm64.tar.gz
tar -xzf specular_darwin_arm64.tar.gz
sudo mv specular /usr/local/bin/

# Windows AMD64
# Download specular_windows_amd64.zip from releases page
# Extract and add to PATH
```

### Docker

```bash
# Pull the latest image
docker pull ghcr.io/felixgeelhaar/specular:latest

# Run with current directory mounted
docker run -v $(pwd):/workspace ghcr.io/felixgeelhaar/specular:latest version
```

### Build from Source

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
# 1. Add an AI provider (example: Anthropic)
specular provider add anthropic --api-key $ANTHROPIC_API_KEY

# Or add other providers:
# specular provider add openai --api-key $OPENAI_API_KEY
# specular provider add gemini --api-key $GEMINI_API_KEY
# specular provider add ollama  # For local Ollama (requires ollama installed)

# 2. List configured providers
specular provider list

# 3. Check provider health
specular provider doctor

# 4. Set default provider
specular provider set-default anthropic
```

### Generate Command Examples

```bash
# Simple generation with automatic model selection
specular generate "What is 2 + 2?"

# Fast response with model hint
specular generate "Count from 1 to 10" --model-hint fast

# Code generation with appropriate model
specular generate "Write a Go function to reverse a string" --model-hint codegen

# High complexity task with P0 priority (uses most capable model)
specular generate "Explain microservices architecture" --complexity 8 --priority P0

# With system prompt and temperature control
specular generate "Tell me a story" \
  --system "You are a creative writer. Keep responses concise." \
  --temperature 0.9 \
  --max-tokens 500

# Verbose mode shows metadata (model, tokens, cost, latency)
specular generate "What is Go?" --model-hint fast --verbose

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
specular provider list

# Example output:
# NAME         TYPE   ENABLED   SOURCE    VERSION
# ----         ----   -------   ------    -------
# ollama       cli    yes       local     1.0.0
# openai       api    yes       builtin   1.0.0
# anthropic    api    yes       builtin   1.0.0
# gemini       api    no        builtin   1.0.0
# claude-cli   cli    no        local     1.0.0

# Check health of all configured providers
specular provider doctor

# Check specific provider
specular provider doctor ollama

# Example output:
# PROVIDER   STATUS      MESSAGE
# --------   ------      -------
# ollama     ✅ HEALTHY   Executable provider: ./providers/ollama/ollama-provider

# Remove a provider
specular provider remove gemini
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

### Step 1: Initialize Governance Workspace

Specular uses a governance-first approach. Start by initializing your workspace:

```bash
# Initialize governance workspace structure
specular governance init

# This creates:
# .specular/approvals/  - Approval records for plans, builds, drift
# .specular/bundles/    - Build bundles with metadata
# .specular/traces/     - Execution trace logs
# .specular/policies.yaml   - Policy configuration
# .specular/providers.yaml  - Provider configuration
```

### Step 2: Configure AI Providers

Add an AI provider to generate specifications and plans:

```bash
# Check available providers
specular provider list

# Add a provider (example: Anthropic)
specular provider add anthropic --api-key $ANTHROPIC_API_KEY

# Verify provider health
specular provider doctor

# Set as default provider
specular provider set-default anthropic
```

### Step 3: Generate Specification

Create a specification using interview mode:

```bash
# List available presets (web-app, api-service, cli-tool, microservice, data-pipeline)
specular interview --list

# Run interactive TUI interview (recommended)
specular interview --preset cli-tool --out .specular/spec.yaml --tui

# Review the generated specification
cat .specular/spec.yaml

# Validate the specification
specular spec validate --in .specular/spec.yaml

# Generate SpecLock with blake3 hashes for drift detection
specular spec lock --in .specular/spec.yaml --out .specular/spec.lock.json
```

### Step 4: Create Execution Plan

Generate an execution plan from your specification:

```bash
# Create execution plan from spec
specular plan create --in .specular/spec.yaml --lock .specular/spec.lock.json --out plan.json

# Visualize task dependencies
specular plan visualize --in plan.json

# Validate plan structure
specular plan validate --in plan.json
```

### Step 5: Execute with Policy Enforcement

Run the build in a sandboxed Docker environment:

```bash
# Execute build with policy enforcement (dry-run first)
specular build run --plan plan.json --policy .specular/policies.yaml --dry-run

# Run actual build
specular build run --plan plan.json --policy .specular/policies.yaml

# Verify build quality gates
specular build verify --bundle .specular/bundles/latest.tar

# Approve build for deployment
specular build approve --bundle .specular/bundles/latest.tar
```

### Step 6: Detect and Manage Drift

Monitor drift between spec, plan, and implementation:

```bash
# Run drift detection (plan + code + infrastructure)
specular drift check --plan plan.json --lock .specular/spec.lock.json \
  --spec .specular/spec.yaml --policy .specular/policies.yaml \
  --report drift.sarif

# Approve detected drift with justification
specular drift approve --report drift.sarif --justification "Approved architectural change"
```

### Alternative: Quick Example Workflow

If you prefer to start with example files:

```bash
# Use example files
cp .specular/spec.yaml.example .specular/spec.yaml
cp .specular/policy.yaml.example .specular/policies.yaml

# Follow Steps 4-6 above with the example spec
```

### Core Features

✅ **Interactive TUI Mode**
- Beautiful terminal UI powered by bubbletea
- Real-time progress tracking with progress bars
- Visual question navigation
- Answer validation with immediate feedback
- Answer history with up/down arrow navigation
- Strict and non-strict validation modes
- Seamless integration with interview command (`--tui` flag)

✅ **Enhanced Error System**
- Structured errors with hierarchical error codes (CATEGORY-NNN format)
- 8 error categories: SPEC, POLICY, PLAN, INTERVIEW, PROVIDER, EXEC, DRIFT, IO
- Actionable suggestions for every error
- Documentation links for detailed guidance
- Fluent API for error building with method chaining
- Go 1.13+ error wrapping support for `errors.Is()` and `errors.As()`
- Beautiful formatted error messages with bullet points

✅ **CLI Provider Protocol**
- Language-agnostic JSON-based stdin/stdout protocol
- Three required commands: generate, stream, health
- Support for streaming with newline-delimited JSON (NDJSON)
- Protocol specification and examples in the [Provider Guide](docs/provider-guide.md)
- Example router configuration in `.specular/router.example.yaml`
- Reference implementation with ollama provider
- Easy integration with ExecutableProvider adapter

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

## Exit Codes

Specular uses specific exit codes to communicate different types of errors, making it easier to handle errors programmatically in scripts and CI/CD pipelines.

| Code | Name | Description |
|------|------|-------------|
| 0 | Success | Execution completed successfully |
| 1 | General Error | Unexpected runtime error occurred |
| 2 | Usage Error | Invalid CLI usage (bad flags, missing arguments) |
| 3 | Policy Violation | Operation blocked by policy rules |
| 4 | Drift Detected | Specification drift detected, requires intervention |
| 5 | Authentication Error | Authentication or permission failure |
| 6 | Network Error | Network connectivity issue |

### Usage in Scripts

```bash
#!/bin/bash
specular auto "Build REST API"
EXIT_CODE=$?

case $EXIT_CODE in
  0)
    echo "✅ Success"
    ;;
  2)
    echo "❌ Usage error - check your command"
    exit 1
    ;;
  3)
    echo "❌ Policy violation - operation not allowed"
    exit 1
    ;;
  4)
    echo "⚠️  Drift detected - manual intervention required"
    exit 1
    ;;
  5)
    echo "❌ Authentication error - check credentials"
    exit 1
    ;;
  6)
    echo "❌ Network error - check connectivity"
    exit 1
    ;;
  *)
    echo "❌ Unexpected error (code: $EXIT_CODE)"
    exit 1
    ;;
esac
```

### CI/CD Integration

Exit codes enable intelligent error handling in CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Run Specular
  id: specular
  run: specular auto "Deploy application"
  continue-on-error: true

- name: Handle Specular errors
  if: failure()
  run: |
    EXIT_CODE=${{ steps.specular.outputs.exit_code }}
    if [ "$EXIT_CODE" = "3" ]; then
      echo "Policy violation - requires approval"
      # Trigger approval workflow
    elif [ "$EXIT_CODE" = "4" ]; then
      echo "Drift detected - requires manual intervention"
      # Create issue for manual review
    elif [ "$EXIT_CODE" = "6" ]; then
      echo "Network error - retrying..."
      # Retry logic
    fi
```

## Per-Step Policy Enforcement

Specular's autonomous mode includes per-step policy checking to enforce safety guardrails and organizational constraints before each workflow step executes.

### Policy Checking Architecture

Policy checks run automatically before each action step in the autonomous workflow. The policy system supports:

- **Cost Limits** - Per-step and total budget enforcement
- **Timeouts** - Workflow and per-step duration limits
- **Step Type Control** - Whitelist/blacklist for step types (spec:update, plan:gen, build:run)
- **Step Count Limits** - Maximum number of steps allowed
- **Retry Limits** - Maximum retries per failed step

### Profile-Based Policies

Policies are configured per profile, allowing different constraints for different environments:

```yaml
# .claude/profiles/strict.yaml
name: strict
description: Maximum safety profile with strict limits

safety:
  max_cost_usd: 5.0          # Total workflow budget
  max_cost_per_task: 1.0     # Per-step cost limit
  max_steps: 10              # Maximum number of steps
  max_retries: 2             # Maximum retries per step
  timeout: 30m               # Total workflow timeout

  # Step type restrictions
  allowed_step_types:        # Whitelist (empty = all allowed)
    - "spec:update"
    - "spec:lock"
    - "plan:gen"
  blocked_step_types:        # Blacklist (takes precedence)
    - "build:run"            # Block execution steps
```

### Built-in Policy Checkers

**CostLimitChecker**: Enforces budget constraints
- Checks per-step cost estimates against limits
- Tracks total cost across workflow execution
- Warns when approaching 80% of budget

**TimeoutChecker**: Enforces duration constraints
- Validates sufficient time remains for steps
- Checks total workflow duration
- Warns when approaching 80% of timeout

**StepTypeChecker**: Controls allowed operations
- Whitelist-based step type validation
- Blacklist overrides whitelist
- Prevents unauthorized step types from executing

**MaxStepsChecker**: Limits workflow complexity
- Enforces maximum step count
- Prevents runaway workflows
- Warns when approaching limit

**MaxRetriesChecker**: Controls failure retry behavior
- Tracks retries per step ID
- Prevents infinite retry loops
- Enforced per-step across workflow

### Policy Check Flow

```
1. Orchestrator prepares to execute step
2. Policy context created (completed steps, cost, time elapsed)
3. All configured policy checkers run sequentially
4. If ANY checker denies:
   - Execution stops for that step
   - Error returned with exit code 3 (Policy Violation)
   - Audit event logged with denial reason
5. If ALL checkers allow:
   - Warnings collected and logged
   - Step execution proceeds
```

### Policy Violation Handling

When a policy check fails:

```bash
# Policy violation example
$ specular auto "Build complex system"

Step 3: Generate implementation plan
❌ Policy violation: maximum step count exceeded: 11 > 10 limit

# Returns exit code 3
$ echo $?
3
```

Policy violations return exit code 3 and include:
- **Checker name** - Which policy failed (e.g., "max_steps")
- **Reason** - Clear explanation of violation
- **Metadata** - Additional context (current count, limits, etc.)
- **Audit trail** - All policy checks logged for compliance

### Using Policies

**Default profile** (balanced):
```bash
specular auto "Create REST API"
# Uses default profile with moderate limits
```

**Strict profile** (maximum safety):
```bash
specular auto --profile strict "Create REST API"
# Enforces strict cost, step, and timeout limits
```

**CI profile** (automated):
```bash
specular auto --profile ci "Run automated workflow"
# Non-interactive, auto-approve, JSON output
```

**Custom profile**:
```bash
# Create custom profile
cat > .claude/profiles/custom.yaml <<EOF
name: custom
safety:
  max_cost_usd: 10.0
  max_steps: 20
  timeout: 1h
  blocked_step_types:
    - "build:run"  # Only planning, no execution
EOF

specular auto --profile custom "Plan system architecture"
```

**CLI Overrides** (override profile settings):
```bash
# Override max steps limit
specular auto --max-steps 15 "Create REST API"

# Override multiple safety limits
specular auto --max-steps 20 --max-cost 15.0 --timeout 90 "Complex workflow"

# Override with profile
specular auto --profile ci --max-steps 10 "Quick deployment"
```

Available CLI overrides:
- `--max-steps <n>` - Maximum number of workflow steps
- `--max-cost <usd>` - Maximum total cost in USD
- `--max-cost-per-task <usd>` - Maximum cost per individual task
- `--max-retries <n>` - Maximum retries per failed task
- `--timeout <minutes>` - Total workflow timeout in minutes

### Policy Bypass

Policies cannot be bypassed in autonomous mode for security. To execute steps that violate policy:

1. **Adjust profile limits** - Increase cost/step/timeout limits
2. **Remove restrictions** - Clear blocked_step_types list
3. **Manual mode** - Use non-autonomous commands without policy enforcement

### Testing Policy Checks

```go
// Example: Testing custom policy checker
func TestCustomPolicy(t *testing.T) {
    checker := &CustomPolicyChecker{}

    step := &auto.ActionStep{
        ID:   "test-step",
        Type: auto.StepTypeBuildRun,
    }

    result, err := checker.CheckStep(context.Background(), step)
    if err != nil {
        t.Fatal(err)
    }

    if !result.Allowed {
        t.Errorf("Expected step to be allowed: %s", result.Reason)
    }
}
```

## Interactive TUI Mode

Specular's autonomous mode features an interactive terminal UI (TUI) powered by Bubble Tea, providing real-time visualization of workflow execution with progress tracking, step history, and hotkey navigation.

### Enabling TUI Mode

Enable interactive TUI with the `--tui` flag:

```bash
# Run with interactive TUI
specular auto --tui "Create REST API"

# Combine with profiles
specular auto --tui --profile strict "Build microservice"
```

### TUI Features

**Real-Time Progress Tracking**:
- Live progress bar showing completion percentage
- Current step display with status icons
- Execution statistics (completed, pending, failed tasks)
- Real-time cost tracking
- Elapsed time display

**Multiple Views**:
- **Main View** (`default`) - Progress overview with current step and statistics
- **Step List View** (`s` key) - All steps with status icons and types
- **Help View** (`?` key) - Hotkey reference and navigation guide
- **Approval View** (automatic) - Interactive approval prompts

**Interactive Hotkeys**:
- `?` - Toggle help view
- `s` - Toggle step list view
- `v` - Toggle verbose mode
- `y` / `Enter` - Approve plan (during approval)
- `n` / `Esc` - Reject plan (during approval)
- `q` - Quit TUI
- `Ctrl+C` - Force quit

### TUI Output Example

```
🤖 Specular Auto Mode

Goal:  Create REST API with authentication

Profile: default

┌──────────────────────────────────────────────┐
│ ⟳ Progress                                   │
│                                              │
│ [████████████████████░░░░░░░░░░] 12/15 (80%)│
│                                              │
│ Completed: 12                                │
│ Pending:   3                                 │
│ Cost:      $2.5000                           │
│ Elapsed:   2m 15s                            │
└──────────────────────────────────────────────┘

Current Step: Generate authentication middleware

? help • s steps • v verbose • q quit
```

### TUI Architecture

The TUI integrates with the orchestrator through the hooks system:

1. **Hook Registration** - TUI hook registered with orchestrator's hook registry
2. **Event Forwarding** - Orchestrator lifecycle events forwarded to TUI adapter
3. **Real-Time Updates** - Step start/complete/fail events update TUI state
4. **Approval Flow** - Interactive approval requests handled through TUI

Supported orchestrator events:
- `on_workflow_start` - Workflow initialization
- `on_step_before` - Step execution begins
- `on_step_after` - Step execution completes
- `on_step_failed` - Step execution fails
- `on_workflow_complete` - Workflow succeeds
- `on_workflow_failed` - Workflow fails

### TUI Best Practices

1. **Use for Interactive Sessions** - TUI is ideal for development and debugging
2. **Combine with Profiles** - Use `--tui --profile strict` for safe exploration
3. **Step List Navigation** - Press `s` to review all steps and their statuses
4. **Verbose Mode** - Toggle `v` for detailed execution logs
5. **CI/CD Pipelines** - Use `--json` instead of `--tui` for automation

### TUI vs Text Mode

| Feature | TUI Mode | Text Mode |
|---------|----------|-----------|
| **Interface** | Interactive UI with navigation | Sequential log output |
| **Progress** | Real-time progress bar | Periodic status updates |
| **Navigation** | Hotkeys to switch views | N/A |
| **Approval** | Interactive prompt with hotkeys | Terminal input prompt |
| **Best For** | Interactive dev sessions | CI/CD, scripting, logs |

## Trace Logging

Specular's autonomous mode provides detailed trace logging to `~/.specular/logs/` for debugging, auditing, and compliance. Trace logs capture all workflow events with timestamps, context, and structured data in JSON format.

### Enabling Trace Logging

Enable trace logging with the `--trace` flag:

```bash
# Run with trace logging
specular auto --trace "Create REST API"

# Combine with other flags
specular auto --trace --profile strict --tui "Build microservice"

# Output shows log file location
📝 Trace logging enabled: /Users/user/.specular/logs/trace_auto-1234567890.json
```

### Trace Log Format

Trace logs are written as newline-delimited JSON (NDJSON) with one event per line:

```json
{
  "id": "20240115120000.123456",
  "type": "workflow_start",
  "timestamp": "2024-01-15T12:00:00Z",
  "workflow_id": "auto-1234567890",
  "message": "Workflow started",
  "level": "info",
  "data": {
    "goal": "Create REST API",
    "profile": "default"
  }
}
```

### Event Types

The trace logger captures 12 types of events:

**Workflow Events**:
- `workflow_start` - Workflow execution begins
- `workflow_complete` - Workflow finishes successfully

**Step Events**:
- `step_start` - Step execution begins
- `step_complete` - Step finishes successfully (includes duration and cost)
- `step_fail` - Step fails with error details

**Policy Events**:
- `policy_check` - Policy checker evaluates a step (includes allowed/denied reason)

**Approval Events**:
- `approval_request` - Approval requested from user
- `approval_response` - User approves or rejects plan

**Budget Events**:
- `budget_check` - Budget limits checked

**General Events**:
- `error` - Error occurred during execution
- `warning` - Warning issued
- `info` - Informational message

### Event Structure

Each trace event contains:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique event identifier (timestamp-based) |
| `type` | string | Event type (see above) |
| `timestamp` | string | ISO 8601 timestamp when event occurred |
| `workflow_id` | string | Workflow identifier (e.g., `auto-1234567890`) |
| `step_id` | string | Step identifier (for step-related events) |
| `message` | string | Human-readable description |
| `level` | string | Severity level: `info`, `warning`, or `error` |
| `data` | object | Event-specific structured data |
| `duration` | number | Duration in nanoseconds (for timed events) |
| `error` | string | Error message (for error events) |
| `context` | object | Workflow context (goal, profile, progress, cost) |

### Event Context

Step and workflow events include rich context:

```json
{
  "context": {
    "goal": "Create REST API",
    "profile": "default",
    "completed_steps": 3,
    "total_steps": 5,
    "total_cost": 1.25,
    "elapsed_time": 120000000000
  }
}
```

### Log Rotation

Trace logs automatically rotate when they reach 10MB:

- **Max File Size**: 10MB (configurable)
- **Max Files**: 5 rotated files kept (configurable)
- **Rotation Format**: `trace_<workflow_id>_<timestamp>.json`
- **Cleanup**: Oldest files automatically removed

Example rotated files:
```
~/.specular/logs/
├── trace_auto-1234567890.json           # Current log
├── trace_auto-1234567890_20240115_120000.json  # Rotated
└── trace_auto-1234567890_20240115_110000.json  # Rotated
```

### Use Cases

**1. Debugging Failed Workflows**
```bash
# Run with trace logging
specular auto --trace "Complex task"

# Analyze trace log if something fails
cat ~/.specular/logs/trace_auto-*.json | \
  jq 'select(.level == "error" or .level == "warning")'
```

**2. Performance Analysis**
```bash
# Extract step durations
cat ~/.specular/logs/trace_auto-*.json | \
  jq 'select(.type == "step_complete") | {step: .step_id, duration: .duration}'

# Calculate total cost per step
cat ~/.specular/logs/trace_auto-*.json | \
  jq 'select(.data.cost) | {step: .step_id, cost: .data.cost}'
```

**3. Compliance Auditing**
```bash
# Extract all policy checks
cat ~/.specular/logs/trace_auto-*.json | \
  jq 'select(.type == "policy_check") | {step: .step_id, allowed: .data.allowed, reason: .data.reason}'

# Extract approval decisions
cat ~/.specular/logs/trace_auto-*.json | \
  jq 'select(.type == "approval_response") | {approved: .data.approved, timestamp}'
```

**4. Cost Tracking**
```bash
# Sum total workflow cost
cat ~/.specular/logs/trace_auto-*.json | \
  jq 'select(.type == "step_complete") | .data.cost' | \
  awk '{sum+=$1} END {printf "Total Cost: $%.2f\n", sum}'
```

### Best Practices

1. **Enable for Debugging** - Use `--trace` when investigating issues or unexpected behavior
2. **Combine with TUI** - Use `--trace --tui` for visual monitoring + detailed logs
3. **Archive Important Runs** - Copy trace logs for production deployments or critical workflows
4. **Parse with jq** - Use jq for powerful log analysis and filtering
5. **Monitor Disk Usage** - Trace logs can grow large; rotation limits disk usage
6. **CI/CD Integration** - Enable `--trace` in CI/CD pipelines for build failure analysis

### Programmatic Access

Access trace events programmatically in Go:

```go
import "github.com/felixgeelhaar/specular/internal/trace"

// Create logger
config := trace.DefaultConfig()
config.Enabled = true
logger, err := trace.NewLogger(config)

// Log events
logger.LogWorkflowStart("Build API", "default")
logger.LogStepStart("step-1", "Generate Specification")
logger.LogStepComplete("step-1", "Generate Specification", duration, 0.50)

// Access events in memory
events := logger.GetEvents()
for _, event := range events {
    fmt.Printf("%s: %s\n", event.Type, event.Message)
}

// Close logger (flushes to disk)
logger.Close()
```

### Trace vs Other Outputs

| Feature | Trace Logging | JSON Output | TUI Mode |
|---------|---------------|-------------|----------|
| **Format** | NDJSON events | Single JSON doc | Interactive UI |
| **Granularity** | All events with timestamps | Final results only | Real-time status |
| **Use Case** | Debugging, auditing | CI/CD integration | Interactive dev |
| **File Output** | `~/.specular/logs/` | stdout or file | Terminal only |
| **Readability** | Machine (jq required) | Machine + Human | Human-friendly |
| **Performance** | Minimal overhead | No overhead | Minimal overhead |

## JSON Output Format

Specular's autonomous mode supports structured JSON output for CI/CD integration and programmatic result processing. The JSON format captures complete execution state including steps, artifacts, metrics, and audit trail.

### Enabling JSON Output

Enable JSON output with the `--json` flag:

```bash
# Output to stdout
specular auto --json "Create REST API" > result.json

# Use in CI/CD pipeline
specular auto --json --profile ci "Deploy to staging"
```

### JSON Schema

The JSON output follows the `specular.auto.output/v1` schema:

```json
{
  "schema": "specular.auto.output/v1",
  "goal": "Create REST API",
  "status": "completed",
  "steps": [
    {
      "id": "step-1",
      "type": "spec:update",
      "status": "completed",
      "startedAt": "2024-01-15T10:00:00Z",
      "completedAt": "2024-01-15T10:00:05Z",
      "duration": 5000000000,
      "costUSD": 0.50,
      "warnings": []
    }
  ],
  "artifacts": [
    {
      "path": "spec.yaml",
      "type": "spec",
      "size": 2048,
      "hash": "sha256:abc123...",
      "createdAt": "2024-01-15T10:00:05Z"
    }
  ],
  "metrics": {
    "totalDuration": 19000000000,
    "totalCost": 1.81,
    "stepsExecuted": 4,
    "stepsFailed": 0,
    "stepsSkipped": 0,
    "policyViolations": 0
  },
  "audit": {
    "checkpointId": "auto-1705315200",
    "profile": "default",
    "startedAt": "2024-01-15T10:00:00Z",
    "completedAt": "2024-01-15T10:00:19Z",
    "approvals": [],
    "policies": [
      {
        "stepId": "step-1",
        "timestamp": "2024-01-15T10:00:00Z",
        "checkerName": "cost_limit",
        "allowed": true,
        "warnings": ["approaching 50% of budget"]
      }
    ],
    "version": "v1.4.0"
  }
}
```

### Schema Fields

**Top-Level Structure:**
- `schema` (string) - Format version identifier (`specular.auto.output/v1`)
- `goal` (string) - User's original objective
- `status` (string) - Execution outcome: `completed`, `failed`, or `partial`
- `steps` (array) - Results for each executed step
- `artifacts` (array) - Generated files and outputs
- `metrics` (object) - Execution statistics
- `audit` (object) - Provenance and compliance information

**Step Result:**
- `id` (string) - Unique step identifier (`step-1`, `step-2`, etc.)
- `type` (string) - Step category: `spec:update`, `spec:lock`, `plan:gen`, `build:run`
- `status` (string) - Step outcome: `pending`, `in_progress`, `completed`, `failed`, `skipped`
- `startedAt` (timestamp) - When step execution began
- `completedAt` (timestamp) - When step execution finished
- `duration` (nanoseconds) - Time taken to execute the step
- `costUSD` (float) - Estimated cost for this step in USD
- `error` (string, optional) - Error message if step failed
- `warnings` (array, optional) - Non-fatal issues encountered
- `metadata` (object, optional) - Step-specific additional information

**Artifact Info:**
- `path` (string) - File path relative to project root
- `type` (string) - Artifact category: `spec`, `lock`, `plan`, `code`, `test`, etc.
- `size` (int64) - File size in bytes
- `hash` (string) - Content verification hash (SHA256)
- `createdAt` (timestamp) - When artifact was created

**Execution Metrics:**
- `totalDuration` (nanoseconds) - Complete workflow execution time
- `totalCost` (float) - Sum of all step costs in USD
- `stepsExecuted` (int) - Count of steps that ran
- `stepsFailed` (int) - Count of steps that failed
- `stepsSkipped` (int) - Count of steps that were skipped
- `policyViolations` (int) - Count of policy check failures
- `tokensUsed` (int, optional) - Total token consumption
- `retriesPerformed` (int, optional) - Total retry attempts

**Audit Trail:**
- `checkpointId` (string) - Execution checkpoint identifier
- `profile` (string) - Profile used for execution
- `startedAt` (timestamp) - Workflow start time
- `completedAt` (timestamp) - Workflow completion time
- `user` (string, optional) - User who initiated the workflow
- `hostname` (string, optional) - Hostname where execution occurred
- `approvals` (array) - Approval events during execution
- `policies` (array) - Policy check events
- `version` (string, optional) - Specular version used

**Policy Event:**
- `stepId` (string) - Step that was checked
- `timestamp` (timestamp) - When check occurred
- `checkerName` (string) - Which policy checker ran
- `allowed` (boolean) - Whether policy check passed
- `reason` (string, optional) - Policy decision explanation
- `warnings` (array, optional) - Non-blocking policy warnings
- `metadata` (object, optional) - Policy-specific additional information

**Approval Event:**
- `stepId` (string) - Step that required approval
- `timestamp` (timestamp) - When approval occurred
- `approved` (boolean) - Whether step was approved
- `reason` (string, optional) - Why approval was required
- `user` (string, optional) - Who approved

### Status Values

The `status` field indicates the overall execution outcome:

- **`completed`**: All steps executed successfully
- **`failed`**: One or more steps failed
- **`partial`**: Execution stopped due to policy violation or user interruption

### CI/CD Integration Examples

**GitHub Actions:**

```yaml
name: Auto Deploy
on: [push]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Run Specular Auto
        id: auto
        run: |
          specular auto --json --profile ci "Deploy to staging" > result.json
          echo "status=$(jq -r .status result.json)" >> $GITHUB_OUTPUT
          echo "cost=$(jq -r .metrics.totalCost result.json)" >> $GITHUB_OUTPUT

      - name: Check Status
        if: steps.auto.outputs.status != 'completed'
        run: |
          echo "Deployment failed or partial"
          jq '.steps[] | select(.status == "failed")' result.json
          exit 1

      - name: Report Metrics
        run: |
          echo "Total cost: ${{ steps.auto.outputs.cost }}"
          echo "Steps: $(jq .metrics.stepsExecuted result.json)"
          echo "Duration: $(jq .metrics.totalDuration result.json | awk '{print $1/1000000000}')s"
```

**GitLab CI:**

```yaml
deploy:
  script:
    - specular auto --json --profile ci "Deploy to production" > result.json
    - |
      STATUS=$(jq -r .status result.json)
      if [ "$STATUS" != "completed" ]; then
        echo "Deployment $STATUS"
        jq '.steps[] | select(.status == "failed")' result.json
        exit 1
      fi
    - |
      COST=$(jq -r .metrics.totalCost result.json)
      echo "Deployment cost: \$$COST"
  artifacts:
    paths:
      - result.json
    reports:
      junit: result.json
```

**Budget Monitoring:**

```bash
#!/bin/bash
# Monitor and alert on costs

specular auto --json --profile ci "$GOAL" > result.json

COST=$(jq -r .metrics.totalCost result.json)
VIOLATIONS=$(jq -r .metrics.policyViolations result.json)

if (( $(echo "$COST > 5.0" | bc -l) )); then
  echo "⚠️  Cost $COST exceeds budget threshold"
  # Send alert to Slack/PagerDuty
fi

if [ "$VIOLATIONS" -gt 0 ]; then
  echo "🚫 Policy violations detected: $VIOLATIONS"
  jq '.audit.policies[] | select(.allowed == false)' result.json
fi
```

**Progressive Deployment:**

```bash
#!/bin/bash
# Progressive deployment with rollback on failure

specular auto --json --profile prod "Deploy v2.0" > result.json

STATUS=$(jq -r .status result.json)
FAILED_STEPS=$(jq -r .metrics.stepsFailed result.json)

if [ "$STATUS" = "completed" ]; then
  echo "✅ Deployment successful"
  # Promote to 100% traffic
elif [ "$STATUS" = "partial" ] && [ "$FAILED_STEPS" -eq 0 ]; then
  echo "⚠️  Partial deployment (policy stop)"
  # Keep at current traffic level
else
  echo "❌ Deployment failed"
  # Rollback to previous version
  jq '.steps[] | select(.status == "failed")' result.json
  exit 1
fi
```

### Parsing JSON Output

**Python:**

```python
import json
import sys

with open('result.json') as f:
    result = json.load(f)

if result['status'] != 'completed':
    print(f"Execution {result['status']}")
    failed = [s for s in result['steps'] if s['status'] == 'failed']
    for step in failed:
        print(f"  {step['id']}: {step.get('error', 'unknown error')}")
    sys.exit(1)

print(f"Cost: ${result['metrics']['totalCost']:.2f}")
print(f"Duration: {result['metrics']['totalDuration'] / 1e9:.1f}s")
print(f"Steps: {result['metrics']['stepsExecuted']}")
```

**jq Examples:**

```bash
# Get total cost
jq -r '.metrics.totalCost' result.json

# List failed steps
jq '.steps[] | select(.status == "failed") | .id' result.json

# Get policy violations
jq '.audit.policies[] | select(.allowed == false)' result.json

# Calculate success rate
jq '.metrics.stepsExecuted / (.metrics.stepsExecuted + .metrics.stepsFailed)' result.json

# Extract all artifacts
jq -r '.artifacts[].path' result.json

# Get warnings from all steps
jq '.steps[].warnings[]?' result.json

# Find most expensive step
jq '.steps | sort_by(-.costUSD) | .[0]' result.json
```

### Testing JSON Output

```go
func TestJSONOutputIntegration(t *testing.T) {
    // Run with JSON output
    cmd := exec.Command("specular", "auto", "--json", "--dry-run", "Test goal")
    output, err := cmd.Output()
    if err != nil {
        t.Fatalf("command failed: %v", err)
    }

    // Parse JSON
    var result auto.AutoOutput
    if err := json.Unmarshal(output, &result); err != nil {
        t.Fatalf("invalid JSON: %v", err)
    }

    // Verify schema
    if result.Schema != "specular.auto.output/v1" {
        t.Errorf("unexpected schema: %s", result.Schema)
    }

    // Verify required fields
    if result.Goal == "" {
        t.Error("goal is empty")
    }
    if result.Status == "" {
        t.Error("status is empty")
    }

    // Verify metrics
    if result.Metrics.StepsExecuted != len(result.Steps) {
        t.Errorf("metrics mismatch: executed=%d, steps=%d",
            result.Metrics.StepsExecuted, len(result.Steps))
    }
}
```

## Patch Generation and Rollback

Specular can generate patches for every step executed in auto mode, allowing you to safely rollback changes if something goes wrong. This provides an additional safety layer beyond git, enabling step-by-step recovery even during complex workflows.

### What are Patches?

A patch is a JSON file containing:
- **File changes**: Unified diffs showing what changed
- **Metadata**: Step ID, type, timestamp, workflow ID
- **Content snapshots**: Full before/after file contents for reliable rollback
- **Statistics**: Files changed, insertions, deletions

Patches are saved to `~/.specular/patches/` with the naming format: `{workflow-id}_{step-id}.patch.json`

### Enabling Patch Generation

Generate patches with the `--save-patches` flag:

```bash
# Generate patches during execution
specular auto --save-patches "Migrate database schema"

# Patches are saved automatically after each step
# 💾 Patch generation enabled: /Users/you/.specular/patches
```

### Patch File Format

Each patch file contains detailed change information:

```json
{
  "stepId": "step-2",
  "stepType": "exec:task",
  "timestamp": "2025-01-11T14:30:00Z",
  "workflowId": "auto-1762811730",
  "description": "Update database schema",
  "filesChanged": 2,
  "insertions": 15,
  "deletions": 8,
  "files": [
    {
      "path": "migrations/001_users.sql",
      "status": "modified",
      "oldContent": "...",
      "newContent": "...",
      "diff": "--- a/migrations/001_users.sql\n+++ b/migrations/001_users.sql\n...",
      "insertions": 10,
      "deletions": 5
    },
    {
      "path": "migrations/002_posts.sql",
      "status": "added",
      "newContent": "...",
      "diff": "--- /dev/null\n+++ b/migrations/002_posts.sql\n...",
      "insertions": 5,
      "deletions": 0
    }
  ]
}
```

### File Statuses

Patches track four types of file changes:

| Status | Description | Rollback Action |
|--------|-------------|----------------|
| `added` | New file created | Delete the file |
| `modified` | Existing file changed | Restore old content |
| `deleted` | File removed | Recreate the file |
| `renamed` | File moved/renamed | Rename back to original |

### Rollback Commands

The `specular auto rollback` command provides several rollback options:

#### List Available Patches

```bash
# See all patches for a workflow
specular auto rollback auto-1762811730 --list

# Output:
# 📋 Patches for workflow auto-1762811730:
#
#   step-1 (spec:generate)
#     Generated product specification
#     Files: 1, Changes: +50 -0
#     Created: 2025-01-11 14:25:30
#
#   step-2 (exec:task)
#     Update database schema
#     Files: 2, Changes: +15 -8
#     Created: 2025-01-11 14:30:00
```

#### Rollback a Single Step

```bash
# Rollback one specific step
specular auto rollback auto-1762811730 step-2

# Output:
# 🔄 Rolling back step: step-2
# ✅ Successfully rolled back step: step-2
```

#### Rollback to a Specific Step

```bash
# Rollback all steps AFTER step-1 (keeping step-1's changes)
specular auto rollback auto-1762811730 --to step-1

# Output:
# 🔄 Rolling back to step: step-1
#    (This will revert all steps after this one)
#
# 📊 Rollback Summary:
#    Steps reverted: 3
#
# ✅ Rollback completed successfully
```

#### Rollback All Steps

```bash
# Rollback entire workflow (revert all changes)
specular auto rollback auto-1762811730 --all

# Requires confirmation:
# 🔄 Rolling back all steps for workflow: auto-1762811730
#    ⚠️  This will revert all changes made by this workflow
#
# Are you sure? [y/N]: y
```

#### Dry-Run Mode

```bash
# Verify rollback safety without applying changes
specular auto rollback auto-1762811730 step-2 --dry-run

# Output:
# 🔄 Rolling back step: step-2
#
# ⚠️  Warnings:
#    - file migrations/002_posts.sql has been modified since patch was created
#
# ✅ Dry-run complete. Use without --dry-run to apply rollback
```

### Rollback Safety Verification

Before applying a rollback, Specular verifies safety by checking:

1. **File existence**: Are files still present (for modified/deleted files)?
2. **Content drift**: Has the file been modified since the patch was created?
3. **Conflicts**: Will the rollback cause data loss or conflicts?

If warnings are detected, you'll be prompted to confirm:

```bash
specular auto rollback auto-1762811730 step-2

# Output:
# ⚠️  Warnings:
#    - file database.sql has been modified since patch was created
#
# ⚠️  Rollback may not be safe due to conflicts
# Use --dry-run to see details without applying changes
#
# Continue anyway? [y/N]:
```

### Use Cases

**1. Incremental Recovery**

Roll back specific failed steps without losing good changes:

```bash
# Step 3 failed, but steps 1-2 were successful
specular auto rollback auto-1762811730 step-3

# Now you can fix the issue and re-run from step 3
```

**2. Testing Changes**

Try changes in auto mode, then roll back if they don't work:

```bash
# Try changes
specular auto --save-patches "Refactor authentication"

# Check if it works
npm test

# Roll back if tests fail
specular auto rollback auto-1762811730 --all
```

**3. Partial Deployment**

Deploy changes step-by-step, rolling back if issues arise:

```bash
# Deploy with patches enabled
specular auto --save-patches "Deploy new API endpoints"

# If monitoring shows issues after deployment
specular auto rollback auto-1762811730 --to step-5  # Keep first 5 steps
```

**4. Development Iteration**

Experiment with AI-generated changes safely:

```bash
# Generate changes
specular auto --save-patches "Add user profiles feature"

# Review changes
git diff

# Keep or rollback
specular auto rollback auto-1762811730 --all  # or keep with git commit
```

### Programmatic Patch Access

Read and analyze patches programmatically:

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/felixgeelhaar/specular/internal/patch"
)

func main() {
    homeDir, _ := os.UserHomeDir()
    patchDir := filepath.Join(homeDir, ".specular", "patches")

    writer := patch.NewWriter(patchDir)

    // List all patches for a workflow
    patches, err := writer.ListPatches("auto-1762811730")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Analyze patches
    totalFiles := 0
    totalInsertions := 0
    totalDeletions := 0

    for _, p := range patches {
        totalFiles += p.FilesChanged
        totalInsertions += p.Insertions
        totalDeletions += p.Deletions

        fmt.Printf("%s: %d files, +%d -%d\n",
            p.StepID, p.FilesChanged, p.Insertions, p.Deletions)
    }

    fmt.Printf("\nTotal impact: %d files, +%d -%d\n",
        totalFiles, totalInsertions, totalDeletions)
}
```

### Patches vs Git

| Feature | Patches | Git |
|---------|---------|-----|
| Granularity | Per-step recovery | Per-commit recovery |
| Speed | Instant rollback | Requires git operations |
| Scope | Auto mode only | Entire repository |
| History | Temporary (per workflow) | Permanent version control |
| Use Case | Quick recovery during development | Long-term version management |

**Best Practices:**

1. **Use both**: Patches for immediate recovery, Git for permanent history
2. **Enable in CI/CD**: Always use `--save-patches` in automated environments
3. **Clean up old patches**: Patches accumulate in `~/.specular/patches/`, clean periodically
4. **Test rollback**: Use `--dry-run` to verify safety before rolling back
5. **Document decisions**: If you roll back, document why in commit messages
6. **Combine with checkpoints**: Use `--resume` to restart after rollback

### Troubleshooting

**Patches not being generated?**

```bash
# Verify --save-patches flag is set
specular auto --save-patches --verbose "your goal"

# Check patch directory exists
ls ~/.specular/patches/

# Verify disk space
df -h ~/.specular/
```

**Rollback warnings about modified files?**

This means files changed after the patch was created. Options:

1. Use `--dry-run` to see what would change
2. Manually merge changes if needed
3. Use git to save current state before rollback
4. Proceed anyway if you understand the risks

**Can't find workflow ID?**

```bash
# List recent workflows
ls ~/.specular/patches/ | grep "^auto-" | cut -d_ -f1-2 | sort -u

# Or check checkpoint logs
specular checkpoint list
```

## Cryptographic Attestations

Specular can generate cryptographic attestations of workflow executions, providing verifiable proof of what was executed, by whom, and with what results. Attestations enable compliance, auditability, and trust in AI-assisted development workflows.

### What are Attestations?

An attestation is a cryptographically signed document that captures:
- **Workflow metadata**: Goal, status, duration, timestamps
- **Provenance data**: Who ran it, where, with which version, git context
- **Execution proof**: Hashes of the plan and output for tamper detection
- **Digital signature**: ECDSA signature proving authenticity

Attestations follow SLSA (Supply Chain Levels for Software Artifacts) principles for software supply chain security.

### Generating Attestations

Enable attestation generation with the `--attest` flag:

```bash
# Generate attestation for workflow
specular auto --attest --json "Deploy API v2.0"

# Attestation saved to: ~/.specular/attestations/auto-1705315200.attestation.json
```

Attestations are saved to:
- `<output-dir>/<workflow-id>.attestation.json` if `--output` is specified
- `~/.specular/attestations/<workflow-id>.attestation.json` otherwise

### Attestation Format

```json
{
  "version": "1.0",
  "workflowId": "auto-1705315200",
  "goal": "Deploy API v2.0",
  "startTime": "2024-01-15T10:00:00Z",
  "endTime": "2024-01-15T10:19:00Z",
  "duration": "19m0s",
  "status": "completed",
  "provenance": {
    "hostname": "ci-runner-01",
    "platform": "linux",
    "arch": "amd64",
    "gitRepo": "https://github.com/org/project.git",
    "gitCommit": "abc123def456...",
    "gitBranch": "main",
    "gitDirty": false,
    "specularVersion": "v1.5.0",
    "profile": "ci",
    "models": [],
    "totalCost": 2.45,
    "tasksExecuted": 12,
    "tasksFailed": 0
  },
  "planHash": "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
  "outputHash": "sha256:60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
  "signedAt": "2024-01-15T10:19:05Z",
  "signedBy": "ci-bot@example.com",
  "signature": "MEUCIQDXvW...",
  "publicKey": "MFkwEwYHKoZ..."
}
```

### Verifying Attestations

Verify attestation signature and provenance:

```bash
# Basic verification
specular auto verify ~/.specular/attestations/auto-1705315200.attestation.json

# Strict verification with options
specular auto verify attestation.json \
  --max-age 24h \
  --require-clean-git \
  --allowed-identity ci-bot@example.com

# Verify with hash checking
specular auto verify attestation.json \
  --verify-hashes \
  --plan plan.json \
  --output output.json
```

**Verification Output:**

```
🔍 Verifying attestation: attestation.json

📋 Attestation Information:
   Version:     1.0
   Workflow ID: auto-1705315200
   Goal:        Deploy API v2.0
   Status:      completed
   Duration:    19m0s
   Signed by:   ci-bot@example.com
   Signed at:   2024-01-15T10:19:05Z

🖥️  Provenance:
   Hostname: ci-runner-01
   Platform: linux/amd64
   Specular: v1.5.0
   Profile:  ci
   Git:      https://github.com/org/project.git@abc123de
   Cost:     $2.4500
   Tasks:    12 executed, 0 failed

🔐 Verifying signature...
✅ Signature valid

📊 Verifying provenance...
✅ Provenance valid

🎉 Attestation verified successfully!
```

### Verification Options

**`--max-age <duration>`**: Reject attestations older than specified duration
```bash
specular auto verify attestation.json --max-age 24h
```

**`--require-clean-git`**: Require clean git working tree (no uncommitted changes)
```bash
specular auto verify attestation.json --require-clean-git
```

**`--allowed-identity <email>`**: Restrict to specific signer identities
```bash
specular auto verify attestation.json \
  --allowed-identity ci-bot@example.com \
  --allowed-identity alice@example.com
```

**`--verify-hashes`**: Verify plan and output hashes match
```bash
specular auto verify attestation.json \
  --verify-hashes \
  --plan spec-plan.json \
  --output auto-output.json
```

### CI/CD Integration

**GitHub Actions - Generate Attestation:**

```yaml
name: Deploy with Attestation
on: [push]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Run Specular Auto with Attestation
        run: |
          specular auto --attest --json --profile ci "Deploy to production" > output.json

      - name: Upload Attestation
        uses: actions/upload-artifact@v3
        with:
          name: attestation
          path: ~/.specular/attestations/*.attestation.json
          retention-days: 90
```

**GitHub Actions - Verify Attestation:**

```yaml
name: Verify Deployment
on: [workflow_dispatch]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - name: Download Attestation
        uses: actions/download-artifact@v3
        with:
          name: attestation

      - name: Verify Attestation
        run: |
          specular auto verify *.attestation.json \
            --max-age 7d \
            --require-clean-git \
            --allowed-identity ci-bot
```

**GitLab CI - Attestation Pipeline:**

```yaml
stages:
  - deploy
  - verify

deploy:
  stage: deploy
  script:
    - specular auto --attest --json --profile ci "Deploy application"
  artifacts:
    paths:
      - ~/.specular/attestations/*.attestation.json
    expire_in: 3 months

verify:
  stage: verify
  script:
    - specular auto verify ~/.specular/attestations/*.attestation.json --max-age 1h
  dependencies:
    - deploy
```

### Compliance Use Cases

**1. Audit Trail for Regulatory Compliance**

Attestations provide immutable proof of:
- Who executed the workflow (identity)
- What was executed (goal, plan hash)
- When it was executed (timestamps)
- Where it was executed (hostname, platform)
- What the results were (output hash, status)

**2. Supply Chain Security (SLSA)**

Attestations enable SLSA Level 2+ compliance by providing:
- Build provenance tracking
- Cryptographic verification of artifacts
- Tamper-evident execution logs
- Git commit linkage for source tracking

**3. Change Control Gates**

Require attestation verification before production deployment:

```bash
#!/bin/bash
# Pre-deployment verification script

ATTESTATION=$(ls -t ~/.specular/attestations/*.attestation.json | head -1)

if ! specular auto verify "$ATTESTATION" \
     --max-age 1h \
     --require-clean-git \
     --allowed-identity deploy-bot; then
  echo "❌ Attestation verification failed - blocking deployment"
  exit 1
fi

echo "✅ Attestation verified - proceeding with deployment"
# ... deployment steps ...
```

**4. Security Incident Response**

After a security incident, attestations provide forensic evidence:
- Which workflows were affected
- What changes were made
- Who initiated the changes
- Whether any unauthorized modifications occurred

### Attestation Security Model

**Signing:**
- Uses ECDSA (Elliptic Curve Digital Signature Algorithm) with P-256 curve
- Ephemeral key pairs generated per workflow execution
- Signature covers all attestation fields except signature itself
- Base64-encoded signature and public key included in attestation

**Verification:**
- Verifies ECDSA signature using embedded public key
- Checks attestation age (optional)
- Validates signer identity (optional)
- Verifies provenance data integrity
- Optionally verifies plan and output hashes

**Threat Model:**
- **Protects against**: Tampering with attestation data, unauthorized workflow execution claims, post-execution data modification
- **Does not protect against**: Compromised signing keys, malicious code in the workflow itself, infrastructure compromise during execution

**Future Enhancements:**
- Sigstore integration for keyless signing with OIDC
- Rekor transparency log for public attestation storage
- Hardware security module (HSM) support for signing
- Certificate-based identity verification

### Programmatic Attestation Verification

**Go:**

```go
import "github.com/felixgeelhaar/specular/internal/attestation"

// Read attestation
data, _ := os.ReadFile("attestation.json")
att, _ := attestation.FromJSON(data)

// Create verifier with options
verifier := attestation.NewStandardVerifier(
    attestation.WithMaxAge(24 * time.Hour),
    attestation.WithRequireGitClean(true),
    attestation.WithAllowedIdentities([]string{"ci-bot@example.com"}),
)

// Verify signature
if err := verifier.Verify(att); err != nil {
    log.Fatalf("Signature verification failed: %v", err)
}

// Verify provenance
if err := verifier.VerifyProvenance(att); err != nil {
    log.Fatalf("Provenance verification failed: %v", err)
}

// Verify hashes (if plan and output available)
planJSON, _ := os.ReadFile("plan.json")
outputJSON, _ := os.ReadFile("output.json")
if err := verifier.VerifyHashes(att, planJSON, outputJSON); err != nil {
    log.Fatalf("Hash verification failed: %v", err)
}

fmt.Println("✅ Attestation verified successfully!")
```

**Python:**

```python
import json
import subprocess
import sys

# Verify using CLI
result = subprocess.run(
    ["specular", "auto", "verify", "attestation.json",
     "--max-age", "24h",
     "--require-clean-git"],
    capture_output=True,
    text=True
)

if result.returncode != 0:
    print(f"❌ Verification failed: {result.stderr}")
    sys.exit(1)

# Parse attestation for metadata
with open("attestation.json") as f:
    attestation = json.load(f)

print(f"✅ Verified workflow: {attestation['goal']}")
print(f"   Executed by: {attestation['signedBy']}")
print(f"   Status: {attestation['status']}")
print(f"   Cost: ${attestation['provenance']['totalCost']:.4f}")
```

### Best Practices

1. **Always verify attestations** before trusting execution results in production
2. **Set max-age limits** appropriate for your workflow cadence (e.g., 1h for deployments)
3. **Require clean git** for production workflows to ensure reproducibility
4. **Restrict signer identities** to known service accounts or authorized users
5. **Verify hashes** when you have the original plan and output files
6. **Store attestations securely** with appropriate retention policies
7. **Archive attestations** for compliance and audit trail requirements
8. **Monitor attestation generation failures** as they may indicate security issues

## Routing Explanation

The `specular explain` command provides detailed insights into routing decisions made during workflow execution. This helps you understand why specific models and providers were selected, optimize your routing strategy, and debug unexpected behavior.

### What does Explain show?

The explain command analyzes completed workflows and provides:
- **Provider and model selection** for each step
- **Cost breakdown** by provider and model
- **Routing rationale** - why specific selections were made
- **Budget utilization** - how much of your budget was consumed
- **Performance metrics** - latency and duration per step
- **Alternative candidates** - what other options were considered

### Basic Usage

```bash
# Explain routing for a completed workflow
specular explain auto-1762811730

# Output:
# 🔍 Analyzing routing decisions for workflow: auto-1762811730
#
# 🔍 Routing Explanation
# ========================================================================
#
# Workflow ID: auto-1762811730
# Goal:        Deploy API service
# Profile:     production
# Completed:   2025-01-11 14:30:00
#
# 📋 Routing Strategy
# ------------------------------------------------------------------------
#   Budget Limit:      $10.00
#   Prefer Cheap:      true
#   Max Latency:       5000ms
#   Fallback Enabled:  true
#   Provider Order:    anthropic → openai → google
```

### Output Formats

The explain command supports multiple output formats:

#### Text Format (default)

Human-readable format with colored output:

```bash
specular explain auto-1762811730 --format text
```

#### JSON Format

Machine-readable JSON for programmatic analysis:

```bash
specular explain auto-1762811730 --format json

# Output:
# {
#   "workflowId": "auto-1762811730",
#   "goal": "Deploy API service",
#   "profile": "production",
#   "strategy": {
#     "budgetLimit": 10.0,
#     "preferCheap": true,
#     "maxLatency": 5000,
#     "fallbackEnabled": true,
#     "providerPreferences": ["anthropic", "openai", "google"]
#   },
#   "steps": [...]
# }
```

#### Markdown Format

Documentation-friendly Markdown:

```bash
specular explain auto-1762811730 --format markdown > routing-report.md
```

#### Compact Format

Brief summary for quick overview:

```bash
specular explain auto-1762811730 --format compact

# Output:
# Workflow auto-1762811730 (production)
# Goal: Deploy API service
# Cost: $2.4500 | Steps: 8 | Budget: 24.5%
#
# Step Routing:
#   1. spec:generate → anthropic/claude-3-5-sonnet ($0.1200)
#   2. plan:create → anthropic/claude-3-5-sonnet ($0.0800)
#   3. exec:task → anthropic/claude-3-5-sonnet ($0.3400)
#   ...
```

### Step-by-Step Analysis

Each step shows detailed routing information:

```
📍 Step-by-Step Routing Decisions
------------------------------------------------------------------------

1. step-1 (spec:generate)
   Selected: anthropic/claude-3-5-sonnet
   Cost:     $0.1200
   Duration: 2.3s
   Reason:   Selected based on profile preference and cost optimization
   Candidates: openai/gpt-4-turbo, google/gemini-pro
   Signals:
     - complexity: medium
     - prefer_provider: anthropic
     - budget_remaining: $9.88

2. step-2 (plan:create)
   Selected: anthropic/claude-3-5-sonnet
   Cost:     $0.0800
   Duration: 1.8s
   Reason:   Continued with same provider for consistency
   ...
```

### Summary Statistics

The summary provides aggregate metrics:

```
📊 Summary
------------------------------------------------------------------------
  Total Cost:         $2.4500
  Steps Executed:     8
  Avg Latency:        2.1s
  Budget Utilization: 24.5%

  Provider Breakdown:
    anthropic:
      Requests: 6
      Cost:     $2.1000
      Models:   claude-3-5-sonnet, claude-3-haiku

    openai:
      Requests: 2
      Cost:     $0.3500
      Models:   gpt-4-turbo
```

### Use Cases

**1. Cost Optimization**

Understand which steps consume the most budget:

```bash
# Analyze cost distribution
specular explain auto-1762811730 --format json | \
  jq '.steps[] | {step: .stepId, cost: .cost, provider: .selectedProvider}'

# Find expensive steps
specular explain auto-1762811730 --format json | \
  jq '.steps[] | select(.cost > 0.5)'
```

**2. Debugging Routing Behavior**

Understand why a specific model was or wasn't selected:

```bash
# See all routing decisions
specular explain auto-1762811730 | grep "Reason:"

# Check what alternatives were considered
specular explain auto-1762811730 | grep "Candidates:"
```

**3. Profile Tuning**

Analyze routing patterns to improve profile configuration:

```bash
# Generate report for multiple workflows
for workflow in auto-*; do
  specular explain $workflow --format compact >> routing-analysis.txt
done

# Analyze provider distribution
grep "Provider Breakdown" -A 10 routing-analysis.txt
```

**4. Audit and Compliance**

Document model usage for compliance:

```bash
# Generate Markdown report for audit trail
specular explain auto-1762811730 --format markdown \
  --output reports/routing-$(date +%Y%m%d).md
```

### Programmatic Analysis

**Python Example:**

```python
import json
import subprocess

# Get routing explanation as JSON
result = subprocess.run(
    ["specular", "explain", "auto-1762811730", "--format", "json"],
    capture_output=True,
    text=True
)

explanation = json.loads(result.stdout)

# Analyze cost by provider
provider_costs = {}
for step in explanation['steps']:
    provider = step['selectedProvider']
    cost = step['cost']
    provider_costs[provider] = provider_costs.get(provider, 0) + cost

print("Cost by Provider:")
for provider, cost in sorted(provider_costs.items(), key=lambda x: x[1], reverse=True):
    print(f"  {provider}: ${cost:.4f}")

# Find steps that exceeded latency threshold
high_latency_steps = [
    step for step in explanation['steps']
    if float(step['duration'].rstrip('s')) > 3.0
]

if high_latency_steps:
    print(f"\n⚠️  {len(high_latency_steps)} steps exceeded 3s latency")
```

**Go Example:**

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"

    "github.com/felixgeelhaar/specular/internal/explain"
)

func main() {
    // Run explain command
    cmd := exec.Command("specular", "explain", "auto-1762811730", "--format", "json")
    output, err := cmd.Output()
    if err != nil {
        panic(err)
    }

    // Parse explanation
    var explanation explain.RoutingExplanation
    if err := json.Unmarshal(output, &explanation); err != nil {
        panic(err)
    }

    // Analyze budget efficiency
    efficiency := explanation.Summary.TotalCost / float64(explanation.Summary.StepsExecuted)
    fmt.Printf("Average cost per step: $%.4f\n", efficiency)

    // Find most expensive step
    var maxCost float64
    var maxStep string
    for _, step := range explanation.Steps {
        if step.Cost > maxCost {
            maxCost = step.Cost
            maxStep = step.StepID
        }
    }
    fmt.Printf("Most expensive step: %s ($%.4f)\n", maxStep, maxCost)
}
```

### Integration with CI/CD

**GitHub Actions - Routing Analysis:**

```yaml
name: Analyze Routing
on:
  workflow_run:
    workflows: ["Deploy with Specular"]
    types: [completed]

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - name: Get Latest Workflow ID
        id: workflow
        run: |
          WORKFLOW_ID=$(ls -t ~/.specular/checkpoints/ | head -1)
          echo "id=$WORKFLOW_ID" >> $GITHUB_OUTPUT

      - name: Generate Routing Report
        run: |
          specular explain ${{ steps.workflow.outputs.id }} \
            --format markdown \
            --output routing-report.md

      - name: Analyze Costs
        run: |
          COST=$(specular explain ${{ steps.workflow.outputs.id }} --format json | \
                 jq -r '.summary.totalCost')

          if (( $(echo "$COST > 5.0" | bc -l) )); then
            echo "⚠️ High cost detected: \$$COST"
            exit 1
          fi

      - name: Upload Report
        uses: actions/upload-artifact@v3
        with:
          name: routing-report
          path: routing-report.md
```

**Cost Monitoring Script:**

```bash
#!/bin/bash
# monitor-routing-costs.sh

THRESHOLD=5.0

# Get latest completed workflow
WORKFLOW_ID=$(ls -t ~/.specular/checkpoints/ | grep "^auto-" | head -1)

if [ -z "$WORKFLOW_ID" ]; then
  echo "No workflows found"
  exit 0
fi

# Get cost from explanation
COST=$(specular explain "$WORKFLOW_ID" --format json | \
       jq -r '.summary.totalCost')

echo "Workflow: $WORKFLOW_ID"
echo "Total Cost: \$$COST"

# Check threshold
if (( $(echo "$COST > $THRESHOLD" | bc -l) )); then
  echo "⚠️  Cost exceeds threshold of \$$THRESHOLD"

  # Generate detailed report
  specular explain "$WORKFLOW_ID" --format text > cost-alert-$WORKFLOW_ID.txt

  # Send alert (example with Slack)
  curl -X POST "$SLACK_WEBHOOK_URL" \
    -H 'Content-Type: application/json' \
    -d "{
      \"text\": \"🚨 High routing cost detected\",
      \"attachments\": [{
        \"text\": \"Workflow $WORKFLOW_ID cost \$$COST (threshold: \$$THRESHOLD)\"
      }]
    }"

  exit 1
fi

echo "✅ Cost within threshold"
```

### Best Practices

1. **Analyze after every workflow** - Generate explanations to understand routing patterns
2. **Set cost thresholds** - Monitor for unexpected cost increases
3. **Compare routing strategies** - Test different profiles and compare results
4. **Document routing decisions** - Use Markdown format for audit trails
5. **Automate cost monitoring** - Integrate with CI/CD for continuous monitoring
6. **Review provider distribution** - Ensure routing matches your strategy
7. **Optimize based on insights** - Use explanation data to improve profiles

### Troubleshooting

**Command not found?**

```bash
# Verify specular is installed
specular --version

# Check if explain command exists
specular explain --help
```

**Checkpoint not found?**

```bash
# List available checkpoints
ls ~/.specular/checkpoints/

# Verify checkpoint ID format
specular checkpoint list
```

**Empty or incomplete explanation?**

The explain command requires completed workflows with routing metadata. Ensure:
- Workflow completed successfully
- Checkpoint data was saved
- Router was configured with tracking enabled

## Hooks System

The hooks system enables lifecycle notifications and integrations with external services. Hooks execute automatically at workflow events, allowing you to integrate Specular with notification systems, webhooks, logging platforms, and custom automation.

### What are Hooks?

Hooks are event-driven callbacks that trigger at specific points during workflow execution:

- **Workflow Events** - Start, completion, failure
- **Plan Events** - Plan creation, approval, rejection
- **Step Events** - Before/after step execution, step failures
- **Policy Events** - Policy violations
- **Drift Events** - Configuration drift detection

### Built-in Hook Types

Specular provides three built-in hook types:

#### 1. Script Hooks

Execute shell scripts with event data passed as environment variables.

```yaml
hooks:
  on_workflow_complete:
    - type: script
      config:
        script: /path/to/notify.sh
        shell: /bin/bash
        args: ["--verbose"]
```

Environment variables provided to scripts:
- `HOOK_EVENT_TYPE` - Event type (e.g., `on_workflow_complete`)
- `HOOK_WORKFLOW_ID` - Workflow identifier
- `HOOK_<KEY>` - Event-specific data (uppercase)

Example script (`notify.sh`):
```bash
#!/bin/bash
echo "Workflow $HOOK_WORKFLOW_ID completed"
echo "Event: $HOOK_EVENT_TYPE"
echo "Duration: $HOOK_DURATION"
echo "Cost: $HOOK_COST"
```

#### 2. Webhook Hooks

Send HTTP POST requests to external APIs with JSON event payloads.

```yaml
hooks:
  on_step_after:
    - type: webhook
      config:
        url: https://api.example.com/webhooks/specular
        headers:
          Authorization: Bearer ${WEBHOOK_TOKEN}
          X-Source: specular
```

Webhook payload format:
```json
{
  "type": "on_step_after",
  "timestamp": "2025-01-15T10:30:00Z",
  "workflowId": "workflow-123",
  "data": {
    "stepId": "step-001",
    "stepType": "build",
    "cost": 0.05,
    "duration": "2m15s"
  }
}
```

#### 3. Slack Hooks

Send formatted notifications to Slack channels.

```yaml
hooks:
  on_workflow_failed:
    - type: slack
      config:
        webhookUrl: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        channel: "#deployments"
        username: "Specular Bot"
        iconEmoji: ":robot_face:"
```

Slack messages are automatically formatted based on event type:
- **Workflow Start** - 🚀 Workflow started notification
- **Workflow Complete** - ✅ Success with duration and cost
- **Workflow Failed** - ❌ Failure with error details
- **Step Events** - ▶️ Step execution status updates
- **Policy Violations** - 🚫 Policy violation alerts

### Configuring Hooks in Profiles

Add hooks to your profile configuration (`~/.specular/profiles.yaml`):

```yaml
profiles:
  production:
    description: Production deployment profile
    hooks:
      # Notify on workflow start
      on_workflow_start:
        - type: slack
          config:
            webhookUrl: ${SLACK_WEBHOOK_URL}
            channel: "#deployments"

      # Log all steps to webhook
      on_step_after:
        - type: webhook
          config:
            url: https://logs.example.com/api/events
            headers:
              Authorization: Bearer ${LOG_API_TOKEN}

      # Alert on failures
      on_workflow_failed:
        - type: slack
          config:
            webhookUrl: ${SLACK_WEBHOOK_URL}
            channel: "#alerts"
        - type: script
          config:
            script: /path/to/alert-oncall.sh

      # Success notification with metrics
      on_workflow_complete:
        - type: slack
          config:
            webhookUrl: ${SLACK_WEBHOOK_URL}
            channel: "#deployments"
```

### Using Environment Variables

Use environment variables for sensitive configuration:

```bash
# Set webhook URLs and tokens
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
export LOG_API_TOKEN="your-api-token"
export WEBHOOK_TOKEN="your-webhook-token"

# Run with profile that uses these variables
specular auto --profile production "deploy to staging"
```

### Hook Execution Behavior

**Concurrency**: Hooks execute concurrently with a default limit of 10 concurrent executions. This ensures fast notification delivery without overwhelming external services.

**Timeouts**: Each hook has a default timeout of 30 seconds. Hooks that exceed this timeout are terminated.

**Failure Modes**: Configure how hook failures are handled:

```yaml
hooks:
  on_step_after:
    - type: webhook
      config:
        url: https://api.example.com/webhooks
        failureMode: ignore  # Options: ignore, warn, fail
```

- **ignore** - Log failure at debug level, continue workflow
- **warn** - Log warning, continue workflow
- **fail** - Abort workflow on hook failure (default for critical hooks)

### Event Data Reference

Different event types provide different data:

**Workflow Events** (`on_workflow_start`, `on_workflow_complete`, `on_workflow_failed`):
```json
{
  "workflowId": "workflow-123",
  "goal": "Deploy application",
  "duration": "15m30s",
  "cost": 2.45,
  "success": true,
  "error": "error message if failed"
}
```

**Plan Events** (`on_plan_created`, `on_plan_approved`, `on_plan_rejected`):
```json
{
  "workflowId": "workflow-123",
  "planId": "plan-456",
  "steps": 12,
  "estimatedCost": 1.50,
  "estimatedDuration": "10m"
}
```

**Step Events** (`on_step_before`, `on_step_after`, `on_step_failed`):
```json
{
  "workflowId": "workflow-123",
  "stepId": "step-001",
  "stepName": "Build application",
  "stepType": "build",
  "stepIndex": 0,
  "cost": 0.15,
  "duration": "2m15s",
  "total_cost": 2.45,
  "error": "error message if failed"
}
```

**Policy Events** (`on_policy_violation`):
```json
{
  "workflowId": "workflow-123",
  "policy": "max_cost_per_step",
  "reason": "Step cost $5.00 exceeds limit $2.00",
  "stepId": "step-003",
  "severity": "critical"
}
```

### Use Cases

#### 1. CI/CD Integration

Integrate Specular workflows with CI/CD pipelines:

```yaml
# GitHub Actions webhook integration
hooks:
  on_workflow_complete:
    - type: webhook
      config:
        url: https://api.github.com/repos/owner/repo/statuses/${GIT_COMMIT}
        headers:
          Authorization: token ${GITHUB_TOKEN}
          Accept: application/vnd.github.v3+json
```

#### 2. Monitoring and Alerting

Send metrics to monitoring systems:

```yaml
hooks:
  on_step_after:
    - type: webhook
      config:
        url: https://metrics.example.com/api/v1/metrics
        headers:
          X-API-Key: ${METRICS_API_KEY}

  on_workflow_failed:
    - type: slack
      config:
        webhookUrl: ${PAGERDUTY_SLACK_WEBHOOK}
        channel: "#incidents"
```

#### 3. Cost Tracking

Track and report workflow costs:

```bash
#!/bin/bash
# cost-tracker.sh
echo "Recording workflow cost: $HOOK_COST USD"
curl -X POST "https://billing.example.com/api/costs" \
  -H "Authorization: Bearer $BILLING_TOKEN" \
  -d "{\"workflow\": \"$HOOK_WORKFLOW_ID\", \"cost\": $HOOK_COST}"
```

```yaml
hooks:
  on_workflow_complete:
    - type: script
      config:
        script: /path/to/cost-tracker.sh
```

#### 4. Audit Logging

Maintain comprehensive audit logs:

```yaml
hooks:
  on_workflow_start:
    - type: webhook
      config:
        url: https://audit.example.com/api/events

  on_step_before:
    - type: webhook
      config:
        url: https://audit.example.com/api/events

  on_step_after:
    - type: webhook
      config:
        url: https://audit.example.com/api/events

  on_workflow_complete:
    - type: webhook
      config:
        url: https://audit.example.com/api/events
```

### Programmatic Hook Usage

Use hooks programmatically in Go:

```go
package main

import (
    "context"
    "github.com/felixgeelhaar/specular/internal/hooks"
)

func main() {
    // Create registry
    registry := hooks.NewRegistry()

    // Register built-in hook factories
    hooks.RegisterBuiltinHooks(registry)

    // Create hook configuration
    config := &hooks.HookConfig{
        Name:    "slack-notifications",
        Type:    "slack",
        Events:  []hooks.EventType{hooks.EventWorkflowComplete},
        Enabled: true,
        Config: map[string]interface{}{
            "webhookUrl": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
            "channel":    "#deployments",
        },
    }

    // Register hook from configuration
    if err := registry.RegisterFromConfig(config); err != nil {
        panic(err)
    }

    // Trigger hook on event
    event := hooks.NewEvent(
        hooks.EventWorkflowComplete,
        "workflow-123",
        map[string]interface{}{
            "duration": "15m30s",
            "cost":     2.45,
        },
    )

    results := registry.Trigger(context.Background(), event)

    // Check results
    for _, result := range results {
        if !result.Success {
            log.Printf("Hook %s failed: %s", result.HookName, result.Error)
        }
    }
}
```

### Custom Hook Implementation

Implement custom hooks by satisfying the `Hook` interface:

```go
package main

import (
    "context"
    "fmt"
    "github.com/felixgeelhaar/specular/internal/hooks"
)

// CustomHook sends notifications via custom protocol
type CustomHook struct {
    name       string
    eventTypes []hooks.EventType
    enabled    bool
    apiKey     string
}

func (h *CustomHook) Name() string {
    return h.name
}

func (h *CustomHook) EventTypes() []hooks.EventType {
    return h.eventTypes
}

func (h *CustomHook) Enabled() bool {
    return h.enabled
}

func (h *CustomHook) Execute(ctx context.Context, event *hooks.Event) error {
    // Implement custom notification logic
    fmt.Printf("Sending notification for %s\n", event.Type)

    // Extract event data
    workflowID := event.WorkflowID
    eventType := event.Type

    // Send to custom API
    // ... implementation ...

    return nil
}

// Factory function
func NewCustomHook(config *hooks.HookConfig) (hooks.Hook, error) {
    apiKey, ok := config.Config["apiKey"].(string)
    if !ok {
        return nil, fmt.Errorf("apiKey required")
    }

    return &CustomHook{
        name:       config.Name,
        eventTypes: config.Events,
        enabled:    config.Enabled,
        apiKey:     apiKey,
    }, nil
}

func main() {
    // Register custom hook factory
    registry := hooks.NewRegistry()
    registry.RegisterFactory("custom", NewCustomHook)

    // Use in configuration
    config := &hooks.HookConfig{
        Name:   "custom-notifier",
        Type:   "custom",
        Events: []hooks.EventType{hooks.EventWorkflowComplete},
        Config: map[string]interface{}{
            "apiKey": "your-api-key",
        },
    }

    registry.RegisterFromConfig(config)
}
```

### Best Practices

1. **Use Environment Variables for Secrets**
   - Never commit webhook URLs or API tokens to version control
   - Use `${VAR}` syntax in YAML configurations
   - Set environment variables before running workflows

2. **Choose Appropriate Failure Modes**
   - Use `ignore` for non-critical notifications
   - Use `warn` for important but non-blocking hooks
   - Use `fail` only for critical integrations

3. **Minimize Hook Latency**
   - Keep script execution under 5 seconds
   - Use asynchronous webhooks where possible
   - Avoid complex processing in hook handlers

4. **Test Hooks Independently**
   - Test webhook URLs with curl before configuration
   - Verify script execution with sample data
   - Check Slack webhook formatting

5. **Monitor Hook Performance**
   - Review hook execution durations in logs
   - Set appropriate timeouts for external services
   - Use failure modes to prevent workflow delays

6. **Organize Hooks by Environment**
   - Create separate profiles for dev/staging/production
   - Use different notification channels per environment
   - Adjust verbosity based on environment criticality

7. **Document Hook Configurations**
   - Comment webhook URLs with their purpose
   - Document expected environment variables
   - Include examples of event data structure

### Troubleshooting

**Hook not executing?**

Check the hook configuration and event type:

```bash
# Verify hooks are registered for the event
# Look for hook execution in logs
specular auto --verbose "your goal"

# Check profile configuration
cat ~/.specular/profiles.yaml
```

**Webhook timing out?**

Increase the timeout in hook configuration:

```yaml
hooks:
  on_workflow_complete:
    - type: webhook
      config:
        url: https://slow-api.example.com/webhooks
        timeout: 60s  # Increase from default 30s
```

**Script failing silently?**

Add error output and logging:

```bash
#!/bin/bash
set -e  # Exit on error
set -x  # Print commands

echo "Hook executing: $HOOK_EVENT_TYPE" >&2
# ... rest of script ...
```

**Slack messages not formatted?**

Verify webhook URL and test with curl:

```bash
curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  -H 'Content-Type: application/json' \
  -d '{"text": "Test message from Specular"}'
```

**Missing event data?**

Check available data for your event type in the Event Data Reference section. Not all events provide the same data fields.

## Advanced Security

Specular provides enterprise-grade security features including secure credential management, comprehensive audit logging, and automatic secret scanning. These features help maintain security compliance, prevent credential leaks, and provide audit trails for regulatory requirements.

### Credential Management

The credential store provides secure, encrypted storage for sensitive credentials with automatic rotation support.

#### Features

- **AES-GCM Encryption** - Military-grade encryption using AES-256-GCM
- **PBKDF2 Key Derivation** - Secure master key derivation from passphrase
- **Automatic Rotation** - Policy-based credential rotation with tracking
- **Expiration Support** - Set expiration dates for temporary credentials
- **Thread-Safe Operations** - Concurrent-safe credential access
- **Metadata Tracking** - Additional context for each credential

#### Basic Usage

```go
package main

import (
    "time"
    "github.com/felixgeelhaar/specular/internal/security"
)

func main() {
    // Create credential store
    store, err := security.NewCredentialStore(
        "/path/to/credentials.json",
        "your-secure-passphrase",
    )
    if err != nil {
        panic(err)
    }

    // Store a credential
    expiresAt := time.Now().Add(90 * 24 * time.Hour) // 90 days
    rotationPolicy := &security.RotationPolicy{
        Enabled:      true,
        IntervalDays: 30,
        LastRotated:  time.Now(),
    }

    err = store.Store("github-token", "ghp_abc123...", &expiresAt, rotationPolicy)
    if err != nil {
        panic(err)
    }

    // Retrieve a credential
    token, err := store.Get("github-token")
    if err != nil {
        panic(err)
    }

    // Use the credential...
    _ = token

    // Check which credentials need rotation
    needsRotation := store.CheckRotation()
    for _, credName := range needsRotation {
        // Rotate credential...
        store.MarkRotated(credName)
    }

    // List all credentials
    credentials := store.List()
    for _, name := range credentials {
        info, _ := store.GetInfo(name)
        fmt.Printf("Credential: %s (created: %s)\n", name, info.CreatedAt)
    }

    // Delete a credential
    store.Delete("old-credential")
}
```

#### Credential Rotation

Implement automatic credential rotation:

```go
// Define rotation policy
rotationPolicy := &security.RotationPolicy{
    Enabled:      true,
    IntervalDays: 30,  // Rotate every 30 days
    LastRotated:  time.Now(),
}

// Store credential with rotation policy
store.Store("api-key", "your-api-key", nil, rotationPolicy)

// Periodically check for credentials needing rotation
func checkCredentialRotation(store *security.CredentialStore) {
    needsRotation := store.CheckRotation()

    for _, credName := range needsRotation {
        fmt.Printf("Credential %s needs rotation\n", credName)

        // 1. Generate new credential value
        newValue := generateNewCredential(credName)

        // 2. Update external service with new credential
        updateExternalService(credName, newValue)

        // 3. Store new credential value
        info, _ := store.GetInfo(credName)
        store.Store(credName, newValue, info.ExpiresAt, info.RotationPolicy)

        // 4. Mark as rotated
        store.MarkRotated(credName)

        fmt.Printf("Credential %s rotated successfully\n", credName)
    }
}
```

#### Credential Expiration

Set expiration dates for temporary credentials:

```go
// Create temporary credential valid for 7 days
expiresAt := time.Now().Add(7 * 24 * time.Hour)
store.Store("temp-token", "token-value", &expiresAt, nil)

// Attempt to retrieve expired credential
token, err := store.Get("temp-token")
if err != nil {
    // Will fail if credential has expired
    fmt.Println("Credential has expired")
}
```

### Audit Logging

Comprehensive audit logging tracks all security-relevant events with structured JSON logs and daily rotation.

#### Audit Event Types

- **Workflow Events** - Start, completion, failure tracking
- **Credential Events** - Creation, access, updates, deletion, rotation
- **Policy Events** - Violations and enforcement actions
- **Secret Scanning** - Detection and blocking of secrets
- **Access Events** - Granted and denied access attempts

#### Basic Usage

```go
package main

import (
    "time"
    "github.com/felixgeelhaar/specular/internal/security"
)

func main() {
    // Create audit logger
    logger, err := security.NewAuditLogger(
        "/var/log/specular/audit",
        true, // Enable console logging
    )
    if err != nil {
        panic(err)
    }

    // Log workflow events
    logger.LogWorkflowStart(
        "workflow-123",
        "Deploy application",
        "production",
        "user@example.com",
    )

    // Simulate workflow execution...
    time.Sleep(2 * time.Second)

    logger.LogWorkflowComplete(
        "workflow-123",
        "user@example.com",
        2*time.Second,
        0.45, // Cost in USD
    )

    // Log credential access
    logger.LogCredentialAccess(
        "github-token",
        "user@example.com",
        true, // Success
    )

    // Log policy violation
    logger.LogPolicyViolation(
        "max_cost_per_step",
        "step-003",
        "system",
        "Step cost $5.00 exceeds limit $2.00",
    )

    // Log secret detection
    logger.LogSecretDetected(
        "aws_access_key",
        "src/config.ts:42",
        "user@example.com",
        true, // Blocked
    )

    // Log custom audit event
    logger.Log(&security.AuditEvent{
        Type:     security.AuditAccessGranted,
        Severity: security.SeverityInfo,
        Actor:    "user@example.com",
        Resource: "admin-panel",
        Action:   "view",
        Result:   "success",
        Details: map[string]interface{}{
            "ip":        "192.168.1.1",
            "userAgent": "Mozilla/5.0...",
        },
    })
}
```

#### Querying Audit Logs

Query audit logs for compliance reporting:

```go
// Query logs from the last 30 days
startDate := time.Now().Add(-30 * 24 * time.Hour)
endDate := time.Now()

filter := security.AuditFilter{
    StartDate: &startDate,
    EndDate:   &endDate,
    EventTypes: []security.AuditEventType{
        security.AuditCredentialAccessed,
        security.AuditPolicyViolation,
    },
    Severities: []security.AuditSeverity{
        security.SeverityWarning,
        security.SeverityCritical,
    },
    Actors: []string{"user@example.com"},
}

events, err := logger.Query(filter)
if err != nil {
    panic(err)
}

// Process events
for _, event := range events {
    fmt.Printf("%s | %s | %s | %s\n",
        event.Timestamp.Format("2006-01-02 15:04:05"),
        event.Severity,
        event.Type,
        event.Action,
    )
}
```

#### Audit Log Format

Audit logs are stored as newline-delimited JSON:

```json
{
  "id": "1705315200000000000",
  "timestamp": "2025-01-15T10:30:00Z",
  "type": "workflow.start",
  "severity": "info",
  "actor": "user@example.com",
  "resource": "workflow-123",
  "action": "start_workflow",
  "result": "success",
  "details": {
    "goal": "Deploy application",
    "profile": "production"
  }
}
```

#### Daily Log Rotation

Logs are automatically rotated daily:

```
/var/log/specular/audit/
├── audit-2025-01-13.jsonl
├── audit-2025-01-14.jsonl
└── audit-2025-01-15.jsonl  # Current day
```

### Secret Scanning

Automatic detection of hardcoded secrets in code with git integration for pre-commit hooks.

#### Supported Secret Types

- **AWS Credentials** - Access keys and secret keys
- **GitHub Tokens** - Personal access tokens and OAuth tokens
- **Slack Tokens** - Bot, app, and user tokens
- **Private Keys** - RSA, DSA, EC, OpenSSH, PGP keys
- **API Keys** - Generic API key patterns
- **Passwords** - Hardcoded password values
- **JWT Tokens** - JSON Web Tokens
- **Database URLs** - Connection strings with credentials
- **Generic Secrets** - Catch-all for secret patterns

#### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/felixgeelhaar/specular/internal/security"
)

func main() {
    // Create secret scanner
    scanner := security.NewSecretScanner()

    // Add custom exclusions
    scanner.AddExcludePath("test_fixtures")
    scanner.AddExcludePath(".cache")

    // Scan a single file
    matches, err := scanner.ScanFile("src/config.ts")
    if err != nil {
        panic(err)
    }

    // Check results
    if len(matches) > 0 {
        fmt.Println(security.FormatReport(matches))
        os.Exit(1)
    }

    // Scan entire directory
    allMatches, err := scanner.ScanDirectory("./src")
    if err != nil {
        panic(err)
    }

    if len(allMatches) > 0 {
        fmt.Println(security.FormatReport(allMatches))
        os.Exit(1)
    }

    fmt.Println("✅ No secrets detected")
}
```

#### Git Pre-Commit Hook

Integrate secret scanning with git pre-commit hooks:

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Get staged files
git diff --cached --name-only --diff-filter=ACM | while read file; do
    # Scan each file for secrets
    specular scan-secrets "$file"
    if [ $? -ne 0 ]; then
        echo "❌ Secret detected in $file"
        echo "Commit blocked. Remove secrets before committing."
        exit 1
    fi
done

echo "✅ No secrets detected"
exit 0
```

#### Scan Git Diff

Scan only changes in a git diff:

```go
// Get git diff
cmd := exec.Command("git", "diff", "--cached")
output, err := cmd.Output()
if err != nil {
    panic(err)
}

// Scan the diff
scanner := security.NewSecretScanner()
matches, err := scanner.ScanGitDiff(string(output))
if err != nil {
    panic(err)
}

if len(matches) > 0 {
    fmt.Println("🚨 Secrets detected in staged changes:")
    fmt.Println(security.FormatReport(matches))
    os.Exit(1)
}
```

#### CI/CD Integration

Integrate secret scanning into CI/CD pipelines:

```yaml
# GitHub Actions example
name: Security Scan

on: [push, pull_request]

jobs:
  scan-secrets:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Specular
        run: |
          curl -L https://github.com/felixgeelhaar/specular/releases/latest/download/specular-linux-amd64 -o specular
          chmod +x specular

      - name: Scan for secrets
        run: |
          ./specular scan-secrets ./src
          if [ $? -ne 0 ]; then
            echo "❌ Secrets detected!"
            exit 1
          fi
```

#### Custom Secret Patterns

Add custom secret patterns:

```go
scanner := security.NewSecretScanner()

// Add custom pattern
customPattern := security.SecretPattern{
    Type:        security.SecretGenericSecret,
    Pattern:     regexp.MustCompile(`MY_COMPANY_API_KEY:\s*["']([A-Za-z0-9]{40})["']`),
    Description: "My Company API Key",
    Severity:    "critical",
}

scanner.patterns = append(scanner.patterns, customPattern)
```

#### Secret Detection Report

Example secret scanning report:

```
🚨 Found 3 potential secret(s):

## CRITICAL Severity (2)

- **AWS Access Key ID** in `src/config.ts:15`
  Type: aws_access_key
  Match: aws_access***REDACTED***16

- **Private Key** in `src/keys/deploy.pem:1`
  Type: private_key
  Match: -----BEGIN***REDACTED***KEY-----

## HIGH Severity (1)

- **GitHub Personal Access Token** in `scripts/deploy.sh:8`
  Type: github_token
  Match: ghp_abc123***REDACTED***xyz789
```

### Use Cases

#### 1. Secure CI/CD Credentials

Store and manage CI/CD credentials securely:

```go
// Initialize credential store
store, _ := security.NewCredentialStore(
    "/var/lib/specular/credentials.json",
    os.Getenv("SPECULAR_MASTER_KEY"),
)

// Store CI/CD credentials with rotation
rotationPolicy := &security.RotationPolicy{
    Enabled:      true,
    IntervalDays: 90,
    LastRotated:  time.Now(),
}

store.Store("github-deploy-token", os.Getenv("GITHUB_TOKEN"), nil, rotationPolicy)
store.Store("aws-access-key", os.Getenv("AWS_ACCESS_KEY_ID"), nil, rotationPolicy)
store.Store("aws-secret-key", os.Getenv("AWS_SECRET_ACCESS_KEY"), nil, rotationPolicy)

// Use credentials in workflows
githubToken, _ := store.Get("github-deploy-token")
// Deploy using githubToken...
```

#### 2. Compliance Audit Trail

Maintain compliance audit logs:

```go
// Create audit logger for compliance
logger, _ := security.NewAuditLogger(
    "/var/log/specular/compliance-audit",
    false, // Disable console for production
)

// Log all security events
logger.LogCredentialAccess("prod-db-password", "admin@company.com", true)
logger.LogPolicyViolation("data_access", "customer-data", "user@company.com", "Unauthorized access attempt")

// Generate compliance reports
startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
endDate := time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC)

filter := security.AuditFilter{
    StartDate: &startDate,
    EndDate:   &endDate,
    EventTypes: []security.AuditEventType{
        security.AuditCredentialAccessed,
        security.AuditAccessDenied,
        security.AuditPolicyViolation,
    },
}

events, _ := logger.Query(filter)

// Export to CSV for compliance reporting
generateComplianceReport(events)
```

#### 3. Pre-Commit Secret Prevention

Prevent secrets from entering version control:

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "🔍 Scanning for secrets..."

# Get list of staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)

# Scan each file
for file in $STAGED_FILES; do
    if [ -f "$file" ]; then
        # Use Specular secret scanner
        go run ./cmd/scan-secrets/main.go "$file"

        if [ $? -ne 0 ]; then
            echo ""
            echo "❌ Secret detected in $file"
            echo "Please remove the secret before committing."
            echo ""
            echo "Options:"
            echo "  1. Remove the secret and use environment variables"
            echo "  2. Store in secure credential store"
            echo "  3. Use .gitignore if this is a config file"
            echo ""
            exit 1
        fi
    fi
done

echo "✅ No secrets detected"
exit 0
```

#### 4. Automated Credential Rotation

Implement automated credential rotation:

```go
package main

import (
    "fmt"
    "time"
    "github.com/felixgeelhaar/specular/internal/security"
)

func main() {
    store, _ := security.NewCredentialStore("credentials.json", "passphrase")
    logger, _ := security.NewAuditLogger("/var/log/audit", true)

    // Check rotation every hour
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        needsRotation := store.CheckRotation()

        for _, credName := range needsRotation {
            fmt.Printf("Rotating credential: %s\n", credName)

            // Generate new credential
            newValue := generateNewCredential(credName)

            // Update in external systems
            if err := updateExternalSystem(credName, newValue); err != nil {
                logger.Log(&security.AuditEvent{
                    Type:     security.AuditCredentialRotated,
                    Severity: security.SeverityError,
                    Actor:    "rotation-service",
                    Resource: credName,
                    Action:   "rotate_credential",
                    Result:   "failure",
                    Details: map[string]interface{}{
                        "error": err.Error(),
                    },
                })
                continue
            }

            // Store new value
            info, _ := store.GetInfo(credName)
            store.Store(credName, newValue, info.ExpiresAt, info.RotationPolicy)
            store.MarkRotated(credName)

            // Log successful rotation
            logger.Log(&security.AuditEvent{
                Type:     security.AuditCredentialRotated,
                Severity: security.SeverityInfo,
                Actor:    "rotation-service",
                Resource: credName,
                Action:   "rotate_credential",
                Result:   "success",
            })

            fmt.Printf("✅ Credential %s rotated successfully\n", credName)
        }
    }
}

func generateNewCredential(name string) string {
    // Implement credential generation logic
    return "new-credential-value"
}

func updateExternalSystem(name, value string) error {
    // Update credential in external systems (AWS, GitHub, etc.)
    return nil
}
```

### Best Practices

1. **Use Strong Passphrases**
   - Use at least 32 characters for credential store passphrase
   - Store passphrase in environment variable, not in code
   - Rotate passphrase periodically

2. **Enable Audit Logging**
   - Always enable audit logging in production
   - Store audit logs in tamper-proof storage
   - Set up log aggregation for centralized analysis
   - Implement log retention policies

3. **Implement Credential Rotation**
   - Set rotation policies for all credentials
   - Automate rotation where possible
   - Test rotation process regularly
   - Document rotation procedures

4. **Scan for Secrets**
   - Enable secret scanning in pre-commit hooks
   - Integrate into CI/CD pipelines
   - Scan existing codebase for historical secrets
   - Educate team on secret management

5. **Least Privilege Access**
   - Grant minimum necessary permissions
   - Audit credential access regularly
   - Revoke unused credentials
   - Use temporary credentials when possible

6. **Secure Storage**
   - Encrypt credential store at rest
   - Use secure file permissions (0600)
   - Store in secure location
   - Backup encrypted credential store

7. **Monitor and Alert**
   - Set up alerts for policy violations
   - Monitor credential access patterns
   - Alert on detected secrets in code
   - Review audit logs regularly

### Troubleshooting

**Credential store fails to load?**

Check file permissions and passphrase:

```bash
# Verify file permissions
ls -la /path/to/credentials.json

# Should be -rw------- (0600)
chmod 0600 /path/to/credentials.json

# Verify passphrase
echo $SPECULAR_MASTER_KEY
```

**Audit logs not rotating?**

Check log directory permissions:

```bash
# Verify directory permissions
ls -la /var/log/specular/audit

# Should be drwx------ (0700)
chmod 0700 /var/log/specular/audit
```

**Secret scanner false positives?**

Add exclusions for test files and fixtures:

```go
scanner := security.NewSecretScanner()
scanner.AddExcludePath("test_fixtures")
scanner.AddExcludePath("mock_data")
scanner.AddExcludePath("examples")
```

**Credential rotation failing?**

Check rotation policy and external system access:

```go
// Verify rotation policy is enabled
info, _ := store.GetInfo("credential-name")
if info.RotationPolicy == nil || !info.RotationPolicy.Enabled {
    fmt.Println("Rotation not enabled")
}

// Test external system connectivity
if err := testExternalSystemAccess(); err != nil {
    fmt.Printf("Cannot connect to external system: %v\n", err)
}
```

## Scope Filtering

Specular's autonomous mode supports precise scope filtering to execute only specific features or paths within a project. Scope filtering enables targeted changes, faster feedback cycles, and more controlled deployments.

### Why Scope Filtering?

In large projects with many features, running the entire autonomous workflow for every change is inefficient. Scope filtering allows you to:
- **Execute specific features** - Work on one feature without affecting others
- **Target paths or modules** - Focus on specific parts of your codebase
- **Reduce execution time** - Skip irrelevant tasks for faster iterations
- **Control deployment scope** - Deploy incrementally with confidence
- **Test changes in isolation** - Verify feature changes independently

### Pattern Types

Specular supports multiple pattern types for flexible filtering:

**1. Feature ID Pattern** - Match exact feature IDs:
```bash
specular auto --scope "feature:feat-1" "Implement changes"
```

**2. Feature Title Pattern** - Match feature titles with glob wildcards:
```bash
specular auto --scope "feature:User*" "Update user features"
specular auto --scope "feature:*Authentication*" "Work on auth"
```

**3. Path Pattern** - Match file paths or API endpoints with globs:
```bash
specular auto --scope "/api/users/*" "Update user API"
specular auto --scope "src/components/**" "Refactor components"
```

**4. Tag Pattern** - Match by feature tags (future):
```bash
specular auto --scope "@critical" "Fix critical issues"
```

### Basic Usage

**Single Feature:**
```bash
# Execute only feature feat-1
specular auto --scope "feature:feat-1" "Implement user login"
```

**Multiple Patterns (OR logic):**
```bash
# Execute feat-1 OR feat-2
specular auto --scope "feature:feat-1" --scope "feature:feat-2" "Update features"
```

**Path-Based Filtering:**
```bash
# Execute features that touch /api/auth endpoints
specular auto --scope "/api/auth/*" "Add JWT authentication"
```

**Wildcard Patterns:**
```bash
# Execute all features starting with "User"
specular auto --scope "feature:User*" "Update user management"
```

### Dependency Inclusion

By default, Specular automatically includes tasks that matched tasks depend on. This ensures dependencies are executed in the correct order.

**Include dependencies (default):**
```bash
specular auto --scope "feature:feat-2" "Implement feature 2"
# Automatically includes feat-1 if feat-2 depends on it
```

**Exclude dependencies:**
```bash
specular auto --scope "feature:feat-2" --include-dependencies=false "Implement feature 2"
# Only execute feat-2, skip dependencies (may fail if dependencies aren't met)
```

### Scope Filtering Flow

1. **Parse patterns** - Specular parses `--scope` flags into typed patterns
2. **Load specification** - Product spec and execution plan are generated
3. **Match features/tasks** - Each task is checked against scope patterns
4. **Expand dependencies** - If enabled, include tasks that matched tasks depend on
5. **Filter plan** - Create filtered plan with only matched tasks
6. **Execute filtered plan** - Run only the scoped tasks

### Examples

**Scenario 1: Feature-Specific Development**

You're working on user profile features in a large application:

```bash
# Execute only user profile related features
specular auto --scope "feature:*Profile*" "Add profile photo upload"

# Output:
# 📋 Scope filter: title:*Profile* (with dependencies)
#    Matched: 3/20 tasks
# ✅ Filtered plan: 3 tasks
```

**Scenario 2: API Endpoint Changes**

Updating specific API endpoints:

```bash
# Target all authentication endpoints
specular auto --scope "/api/auth/*" "Add 2FA to authentication"

# Multiple API paths
specular auto \
  --scope "/api/auth/*" \
  --scope "/api/users/*" \
  "Update auth and user APIs"
```

**Scenario 3: Incremental Deployment**

Deploy features incrementally in production:

```bash
# Deploy phase 1 features
specular auto --profile prod \
  --scope "feature:feat-1" \
  --scope "feature:feat-2" \
  "Deploy phase 1"

# Later, deploy phase 2 features
specular auto --profile prod \
  --scope "feature:feat-3" \
  --scope "feature:feat-4" \
  "Deploy phase 2"
```

**Scenario 4: Bug Fixes in Specific Modules**

Fix bugs without affecting other features:

```bash
# Fix issues in payment module
specular auto --scope "/api/payments/*" "Fix payment processing bug"

# Target specific feature for hotfix
specular auto --profile strict \
  --scope "feature:checkout" \
  "Hotfix checkout calculation"
```

### CI/CD Integration

**Feature Branch Deployments:**

```yaml
# GitHub Actions - Deploy only changed features
name: Feature Branch Deploy
on:
  push:
    branches:
      - 'feature/**'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Extract feature name
        id: feature
        run: |
          BRANCH=${GITHUB_REF#refs/heads/feature/}
          echo "name=$BRANCH" >> $GITHUB_OUTPUT

      - name: Deploy scoped feature
        run: |
          specular auto --profile ci \
            --scope "feature:*${{ steps.feature.outputs.name }}*" \
            "Deploy feature ${{ steps.feature.outputs.name }}"
```

**Parallel Feature Deployment:**

```yaml
# GitLab CI - Deploy multiple features in parallel
stages:
  - deploy

deploy_feature_1:
  stage: deploy
  script:
    - specular auto --profile ci --scope "feature:feat-1" "Deploy feature 1"
  only:
    changes:
      - "features/feat-1/**"

deploy_feature_2:
  stage: deploy
  script:
    - specular auto --profile ci --scope "feature:feat-2" "Deploy feature 2"
  only:
    changes:
      - "features/feat-2/**"
```

**Selective Rollout:**

```bash
#!/bin/bash
# Progressive deployment with scope filtering

# Phase 1: Deploy to 10% of users
specular auto --profile prod \
  --scope "feature:new-dashboard" \
  "Deploy new dashboard to 10%"

# Monitor metrics...
sleep 3600

# Phase 2: Deploy to 50% of users
specular auto --profile prod \
  --scope "feature:new-dashboard" \
  --scope "feature:analytics" \
  "Deploy to 50%"

# Phase 3: Full rollout
specular auto --profile prod "Deploy all features"
```

### Advanced Patterns

**Complex Filtering:**

```bash
# Combine feature and path patterns
specular auto \
  --scope "feature:User*" \
  --scope "/api/admin/*" \
  "Update user and admin features"

# Multiple features with dependency control
specular auto \
  --scope "feature:feat-1" \
  --scope "feature:feat-2" \
  --scope "feature:feat-3" \
  --include-dependencies=false \
  "Deploy independent features"
```

**Scope Impact Estimation:**

Before execution, Specular shows how many tasks match the scope:

```
📋 Scope filter: feature:User* (with dependencies)
   Matched: 5/20 tasks

✅ Filtered plan: 7 tasks (including 2 dependencies)
```

This helps you understand the impact before execution.

### Programmatic Scope Filtering

**Go API:**

```go
import "github.com/felixgeelhaar/specular/internal/auto"

// Create scope filter
scope, err := auto.NewScope(
    []string{"feature:feat-1", "/api/users/*"},
    true, // include dependencies
)

// Check if feature matches
matches := scope.MatchesFeature(feature)

// Filter plan
filteredPlan := scope.FilterPlan(executionPlan, productSpec)

// Estimate impact
matched, total := scope.EstimateImpact(executionPlan, productSpec)
fmt.Printf("Scope will affect %d/%d tasks\n", matched, total)
```

### Best Practices

1. **Start with feature IDs** for precise control - Use exact feature IDs when you know exactly what to execute
2. **Use wildcards for exploration** - Glob patterns help when feature names follow conventions
3. **Include dependencies by default** - Only disable for truly independent tasks
4. **Test scope filters in dry-run** - Use `--dry-run` to verify what will be executed
5. **Combine with profiles** - Use scopes with profiles for environment-specific execution
6. **Monitor filtered task counts** - Pay attention to how many tasks match your scope
7. **Use multiple scopes for OR logic** - Multiple `--scope` flags are ORed together
8. **Document scope patterns** - Keep a reference of common patterns for your project

### Troubleshooting

**Problem**: Scope matches no tasks
```bash
# Check available features
specular spec list

# Use broader pattern
specular auto --scope "feature:*user*" "..." # instead of exact match
```

**Problem**: Too many tasks matched
```bash
# Use more specific pattern
specular auto --scope "feature:user-login" "..." # instead of "feature:user*"

# Or target specific path
specular auto --scope "/api/auth/login" "..."
```

**Problem**: Dependencies not included
```bash
# Ensure include-dependencies is enabled (default)
specular auto --scope "feature:feat-2" --include-dependencies=true "..."
```

**Problem**: Tasks fail due to missing dependencies
```bash
# Include dependencies explicitly
specular auto --scope "feature:feat-2" "..."  # Dependencies included by default

# Or execute dependencies first
specular auto --scope "feature:feat-1" "..." # Run dependency
specular auto --scope "feature:feat-2" --include-dependencies=false "..." # Then run target
```

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
 ├─ docs/                # Public docs (getting started, installation, CLI reference, guides)
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
- [CLI Reference](docs/CLI_REFERENCE.md) – Complete command/flag reference
- [Provider Guide](docs/provider-guide.md) – Model selection, retry/fallback, context management, streaming
- [Bundle User Guide](docs/BUNDLE_USER_GUIDE.md) – Building, verifying, and distributing governance bundles

## Core Principles

- **Spec-first**: SpecLock is the single source of truth
- **Traceability**: Stable links across spec → plan → code/tests
- **Governance**: All execution passes policy gates
- **Reproducibility**: Every run emits hashes, costs, and provenance

## Documentation

- **[Getting Started](docs/getting-started.md)** – Quickstart tutorial and workflows
- **[Installation Guide](docs/installation.md)** – Package, binary, and Docker installs
- **[CLI Reference](docs/CLI_REFERENCE.md)** – Command/flag details
- **[Provider Guide](docs/provider-guide.md)** – Configure local/cloud AI providers and routing
- **[Bundle User Guide](docs/BUNDLE_USER_GUIDE.md)** – Governed bundle lifecycle
- **[Production Guide](docs/PRODUCTION_GUIDE.md)** – Production deployment, security, monitoring, and operations

## License

**Business Source License 1.1** (BSL 1.1)

Specular is licensed under the Business Source License 1.1, which provides source-available access while protecting our ability to build a sustainable business.

### What You Can Do

✅ **Free for internal use** - Use Specular within your organization for development, regardless of size
✅ **Consulting & integration services** - Use Specular as part of professional services
✅ **Educational & research** - Use Specular for teaching, learning, and academic purposes
✅ **Personal & open-source projects** - Build your own projects with Specular
✅ **Modify & contribute** - Fork, modify, and contribute to the project

### What Requires a Commercial License

❌ **Commercial SaaS** - Offering Specular as a hosted service or managed offering
❌ **Competing services** - Building competing AI-assisted specification/build services
❌ **Cloud provider wrappers** - Packaging Specular as a managed service (AWS, Azure, GCP)

### Automatic Open Source Conversion

After 2 years, the code automatically becomes **Apache License 2.0** (fully open source).

- **All versions** → Apache 2.0 on 2027-11-15 (2 years from BSL adoption)

**Note**: BSL 1.1 was adopted retroactively for all versions on 2025-11-15, which is permissible since the project is in early development with no production users.

### Why BSL?

We chose BSL to:
1. **Enable free use** for developers and companies for internal purposes
2. **Protect our business** from cloud providers commercializing our work without contributing
3. **Ensure long-term openness** via automatic Apache 2.0 conversion
4. **Build sustainably** while keeping the code transparent

**For full license details**, see [LICENSE](LICENSE) file.
**For commercial licensing**, contact felix@felixgeelhaar.de

## Contributing

Contributions are welcome! Please read the development guide in [CLAUDE.md](.github/CLAUDE.md) before submitting pull requests.

For maintainers releasing new versions, see the [Release Process Guide](docs/RELEASE_PROCESS.md).
