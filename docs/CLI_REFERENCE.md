# Specular CLI Reference

Complete reference for Specular CLI commands and flags.

## Table of Contents

- [Overview](#overview)
- [Global Flags](#global-flags)
- [Governance Commands](#governance-commands)
  - [governance](#governance)
  - [doctor](#doctor)
- [Policy Management Commands](#policy-management-commands)
  - [policy](#policy)
- [Approval Workflow Commands](#approval-workflow-commands)
  - [approval](#approval)
- [Environment & Configuration Commands](#environment--configuration-commands)
  - [context](#context)
  - [config](#config)
  - [status](#status)
  - [logs](#logs)
- [Specification Commands](#specification-commands)
  - [spec](#spec)
  - [interview](#interview)
- [Planning Commands](#planning-commands)
  - [plan](#plan)
- [Build Commands](#build-commands)
  - [build](#build)
- [Bundle Commands](#bundle-commands)
  - [bundle](#bundle)
- [Drift Detection Commands](#drift-detection-commands)
  - [drift](#drift)
- [Autonomous Mode Commands](#autonomous-mode-commands)
  - [auto](#auto)
- [Checkpoint Commands](#checkpoint-commands)
  - [checkpoint](#checkpoint)
- [Provider Commands](#provider-commands)
  - [provider](#provider)
- [Utility Commands](#utility-commands)
  - [version](#version)

## Overview

Specular is an AI-native development workflow tool that helps you build software through AI-powered specifications, planning, and execution.

```bash
specular [command] [flags]
```

## Global Flags

These flags are available for all commands:

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--format` | `-f` | string | Output format: text, json, or yaml (default: text) |
| `--no-color` | | bool | Disable colored output |
| `--verbose` | `-v` | bool | Enable verbose logging |
| `--help` | `-h` | bool | Display help information |

## Governance Commands

### governance

Manage governance workspace and workflow compliance.

**Usage:**
```bash
specular governance <subcommand> [flags]
```

**Description:**

The governance commands help you establish and maintain governance practices for your AI-powered development workflows. This includes initializing workspace structure, running health checks, and monitoring governance compliance.

**Subcommands:**

#### governance init

Initialize .specular governance workspace structure.

```bash
specular governance init [flags]
```

**Description:**

Creates the governance workspace structure with required directories:
- `.specular/approvals/` - Stores approval records for plans, builds, and drift
- `.specular/bundles/` - Stores build bundles with metadata
- `.specular/traces/` - Stores execution trace logs
- `.specular/policies.yaml` - Policy configuration template
- `.specular/providers.yaml` - Provider configuration template

**Example:**
```bash
$ specular governance init
✓ Created .specular/approvals/
✓ Created .specular/bundles/
✓ Created .specular/traces/
✓ Created .specular/policies.yaml
✓ Created .specular/providers.yaml

Governance workspace initialized successfully.

Next steps:
  1. Review and customize policies.yaml
  2. Configure providers in providers.yaml
  3. Run 'specular governance doctor' to verify setup
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--force` | bool | Overwrite existing files |
| `--template <name>` | string | Use specific governance template |

---

#### governance doctor

Run comprehensive governance health checks.

```bash
specular governance doctor [--format text|json|yaml]
```

**Description:**

Performs comprehensive health checks on your governance configuration:
- **Workspace**: Verifies .specular directory structure
- **Policies**: Validates policies.yaml configuration
- **Providers**: Checks providers.yaml and provider availability
- **Bundles**: Verifies bundle storage and integrity
- **Approvals**: Checks approval workflow configuration
- **Traces**: Validates trace logging setup

**Example:**
```bash
$ specular governance doctor

Governance Health Checks:
  ✓ Workspace: Governance workspace initialized
    • approvals: true
    • bundles: true
    • traces: true
  ✓ Policies: Policy configuration valid
  ✓ Providers: 2 providers configured
  ✓ Bundles: 5 bundles found
  ✓ Approvals: Approval workflow configured
  ✓ Traces: Trace logging enabled

All governance checks passed.
```

**JSON Output:**
```bash
$ specular governance doctor --format json
{
  "workspace": {
    "status": "ok",
    "message": "Governance workspace initialized",
    "details": {
      "approvals": true,
      "bundles": true,
      "traces": true
    }
  },
  "policies": {
    "status": "ok",
    "message": "Policy configuration valid"
  },
  "providers": {
    "status": "ok",
    "message": "2 providers configured"
  }
}
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--format` | string | Output format: text, json, or yaml |
| `--verbose` | bool | Show detailed check information |

---

#### governance status

Display current governance workflow status.

```bash
specular governance status [--format text|json|yaml]
```

**Description:**

Shows the current state of governance workflows including:
- Active approval requests
- Recent bundles created
- Policy compliance status
- Trace log summary

**Example:**
```bash
$ specular governance status

Governance Status:
  Pending Approvals: 2
    • plan-abc123 (waiting for approval)
    • build-def456 (waiting for approval)

  Recent Bundles: 3
    • bundle-xyz789 (2 hours ago)
    • bundle-uvw456 (5 hours ago)

  Policy Compliance: ✓ All policies passing

  Trace Logs: 15 traces in last 24 hours
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--format` | string | Output format: text, json, or yaml |
| `--days <n>` | int | Show status for last N days (default: 7) |

---

### doctor

Unified system health check command.

**Usage:**
```bash
specular doctor [--format text|json|yaml]
```

**Description:**

Runs comprehensive system health checks across all Specular components:
- **Container Runtime**: Docker/Podman detection and health
- **AI Providers**: Provider availability and connectivity
- **Git Repository**: Repository status and configuration
- **Project Structure**: Workspace and file structure validation
- **Governance**: Governance workspace and policy checks
- **Environment**: System environment and dependencies

**Example:**
```bash
$ specular doctor

System Health Checks:

Container Runtime:
  ✓ Docker detected (version 24.0.6)
  ✓ Docker daemon running

AI Providers:
  ✓ Ollama detected (available)
  ✓ Anthropic API key configured
  ✗ OpenAI API key not configured

Git Repository:
  ✓ Git repository detected
  ✓ Branch: main
  ⚠ Uncommitted changes detected

Project:
  ✓ .specular directory exists
  ✓ Spec file exists
  ✓ Policy file exists

Governance:
  ✓ Workspace initialized
  ✓ Policies configured
  ✓ Providers configured

Next Steps:
  1. Configure OpenAI API key (optional)
  2. Commit uncommitted changes
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--format` | string | Output format: text, json, or yaml |
| `--verbose` | bool | Show detailed diagnostic information |

---

## Policy Management Commands

### policy

Manage governance policies and approval workflows.

**Usage:**
```bash
specular policy <subcommand> [flags]
```

**Description:**

Policy commands help you define, validate, and enforce governance policies for your AI-powered development workflows.

**Subcommands:**

#### policy init

Initialize policy configuration with templates.

```bash
specular policy init [--template <name>]
```

**Description:**

Creates a policies.yaml file with pre-configured templates for common governance scenarios:
- **default**: Balanced security and flexibility
- **strict**: Maximum security and compliance
- **permissive**: Minimal restrictions for development
- **ci**: Optimized for CI/CD environments

**Example:**
```bash
$ specular policy init --template strict
✓ Created .specular/policies.yaml (strict template)

Policy configuration initialized.
Review and customize .specular/policies.yaml for your needs.
```

**Available Templates:**
- `default` - Balanced security (recommended for most projects)
- `strict` - Maximum security and compliance
- `permissive` - Minimal restrictions (development only)
- `ci` - CI/CD optimized configuration

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--template <name>` | string | Policy template to use |
| `--force` | bool | Overwrite existing policies.yaml |

---

#### policy validate

Validate policy configuration.

```bash
specular policy validate [--strict] [--format text|json|yaml]
```

**Description:**

Validates the policies.yaml file for:
- YAML syntax correctness
- Required fields presence
- Value range validation
- Policy rule consistency
- Security best practices

**Example:**
```bash
$ specular policy validate
✓ Policy syntax valid
✓ Required fields present
✓ Docker configuration valid
✓ Resource limits valid
✓ Test requirements valid

Policy validation passed.
```

**Strict Mode:**
```bash
$ specular policy validate --strict
✓ Policy syntax valid
✓ Required fields present
✓ Docker configuration valid
⚠ Warning: allow_local is true (not recommended for production)
✗ Error: min_coverage below recommended 80%

Policy validation failed in strict mode.
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--strict` | bool | Enable strict validation mode |
| `--format` | string | Output format: text, json, or yaml |
| `--json` | bool | Output validation results as JSON |

---

#### policy approve

Approve policy changes with audit trail.

```bash
specular policy approve [--reason <text>]
```

**Description:**

Records approval of policy changes in the approval workflow:
- Creates approval record with timestamp
- Captures approver identity (from git config)
- Stores approval reason for audit trail
- Updates policy approval status

**Example:**
```bash
$ specular policy approve --reason "Updated resource limits for production workload"
✓ Policy changes approved
  Approver: user@example.com
  Timestamp: 2025-11-17T10:30:00Z
  Reason: Updated resource limits for production workload

Approval recorded in .specular/approvals/policy-abc123.json
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--reason <text>` | string | Reason for approval (required) |
| `--approver <email>` | string | Approver email (defaults to git config) |

---

#### policy list

List all policies with metadata.

```bash
specular policy list [--format text|json|yaml]
```

**Description:**

Displays all configured policies with:
- Policy category (execution, tests, security)
- Current values and limits
- Last modified timestamp
- Approval status

**Example:**
```bash
$ specular policy list

Execution Policies:
  • allow_local: false
  • docker.required: true
  • docker.resource_limits.cpu: "2"
  • docker.resource_limits.memory: "2Gi"

Test Policies:
  • require_pass: true
  • min_coverage: 0.8

Security Policies:
  • docker.network: "none"
  • image_allowlist: 5 images

Last Updated: 2025-11-17T10:00:00Z
Approval Status: Approved
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--format` | string | Output format: text, json, or yaml |
| `--category <name>` | string | Filter by category (execution, tests, security) |

---

#### policy diff

Compare policy versions.

```bash
specular policy diff [--from <version>] [--to <version>]
```

**Description:**

Compares policy configurations between versions or against current state:
- Shows added, removed, and modified policies
- Highlights value changes
- Indicates approval requirements

**Example:**
```bash
$ specular policy diff --from HEAD~1 --to HEAD

Policy Changes:

Modified:
  execution.docker.resource_limits.cpu: "1" → "2"
  execution.docker.resource_limits.memory: "1Gi" → "2Gi"

Added:
  + security.scan_images: true

Removed:
  - tests.allow_skip: true

Changes require approval.
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--from <version>` | string | Source version (git ref or file path) |
| `--to <version>` | string | Target version (git ref, file path, or current) |
| `--format` | string | Output format: text, json, or yaml |

---

## Approval Workflow Commands

### approval

Manage approval workflow for plans, builds, and drift.

**Usage:**
```bash
specular approval <subcommand> [flags]
```

**Description:**

Approval commands implement governance controls for critical workflow steps, ensuring proper review and authorization before execution.

**Subcommands:**

#### approval approve

Approve plans, builds, or drift with role verification.

```bash
specular approval approve <type> <id> [--reason <text>]
```

**Arguments:**
- `<type>` - What to approve: plan, build, or drift
- `<id>` - ID of the item to approve

**Description:**

Records approval for workflow items:
- Verifies approver has required role/permissions
- Creates approval record with audit trail
- Updates approval status in workflow
- Enables gated workflow progression

**Example:**
```bash
$ specular approval approve plan plan-abc123 --reason "Reviewed and validated implementation plan"
✓ Plan approved
  Type: plan
  ID: plan-abc123
  Approver: user@example.com
  Role: developer
  Timestamp: 2025-11-17T10:30:00Z
  Reason: Reviewed and validated implementation plan

Approval recorded in .specular/approvals/plan-abc123.json
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--reason <text>` | string | Reason for approval (required) |
| `--approver <email>` | string | Approver email (defaults to git config) |
| `--role <name>` | string | Approver role (developer, reviewer, admin) |

---

#### approval list

List all approval records with filtering.

```bash
specular approval list [--type <type>] [--status <status>] [--format text|json|yaml]
```

**Description:**

Displays approval records with filtering options:
- Filter by type (plan, build, drift, policy)
- Filter by status (pending, approved, rejected)
- Sort by timestamp
- Show approver and reason

**Example:**
```bash
$ specular approval list --type plan

Approval Records (plan):
  ✓ plan-abc123 - Approved
    Approver: user@example.com
    Timestamp: 2025-11-17T10:30:00Z
    Reason: Reviewed and validated implementation plan

  ⏳ plan-def456 - Pending
    Created: 2025-11-17T11:00:00Z

  ✗ plan-ghi789 - Rejected
    Approver: reviewer@example.com
    Timestamp: 2025-11-16T15:00:00Z
    Reason: Security concerns in implementation

Total: 3 approvals (1 approved, 1 pending, 1 rejected)
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--type <type>` | string | Filter by type: plan, build, drift, policy |
| `--status <status>` | string | Filter by status: pending, approved, rejected |
| `--format` | string | Output format: text, json, or yaml |
| `--days <n>` | int | Show approvals from last N days |

---

#### approval pending

Show pending approvals requiring action.

```bash
specular approval pending [--format text|json|yaml]
```

**Description:**

Displays all items awaiting approval:
- Shows pending plans, builds, and drift detections
- Highlights items requiring urgent attention
- Provides quick approval commands

**Example:**
```bash
$ specular approval pending

Pending Approvals:

Plans (2):
  • plan-abc123 (created 2 hours ago)
    Description: Add user authentication
    Command: specular approval approve plan plan-abc123

  • plan-def456 (created 30 minutes ago)
    Description: Refactor payment processing
    Command: specular approval approve plan plan-def456

Builds (1):
  • build-xyz789 (created 1 hour ago)
    Description: Build for plan-abc123
    Command: specular approval approve build build-xyz789

Total: 3 pending approvals
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--format` | string | Output format: text, json, or yaml |
| `--type <type>` | string | Filter by type: plan, build, drift |

---

## Environment & Configuration Commands

### context

Display detected environment information including runtime, providers, and project context.

**Usage:**
```bash
specular context
```

**Description:**

The `context` command detects and displays:
- Container runtime (Docker/Podman)
- Available AI providers and API keys
- Git repository information
- Project structure and configuration
- Environment health status

**Output Formats:**

Text format (default):
```bash
$ specular context
```

JSON format:
```bash
$ specular context --format json
```

YAML format:
```bash
$ specular context --format yaml
```

**Example Output:**

```
Environment:
  ✓ Runtime: docker
  ✓ Providers: ollama, anthropic
  ✓ API Keys: 1 configured

Project:
  Directory: specular
  ✓ Git repository (branch: main, clean)
  ✓ Initialized (.specular directory exists)
```

---

### config

Manage Specular global configuration stored at `~/.specular/config.yaml`.

**Usage:**
```bash
specular config <subcommand> [flags]
```

**Subcommands:**

#### config view

Display current configuration.

```bash
specular config view [--format text|json|yaml]
```

**Example:**
```bash
$ specular config view
Configuration file: /Users/you/.specular/config.yaml

providers:
    default: ollama
    preference:
        - ollama
        - anthropic
        - openai
defaults:
    format: text
    no_color: false
    verbose: false
    specular_dir: .specular
budget:
    max_cost_per_day: 20.00
    max_cost_per_request: 1.00
    max_latency_ms: 60000
logging:
    level: info
    enable_file: true
    log_dir: ~/.specular/logs
telemetry:
    enabled: false
    share_usage: false
```

#### config edit

Open configuration file in `$EDITOR`.

```bash
specular config edit
```

**Example:**
```bash
$ specular config edit
# Opens ~/.specular/config.yaml in your default editor
```

#### config get

Get a specific configuration value using dot notation.

```bash
specular config get <key>
```

**Available Keys:**
- `providers.default` - Default provider
- `defaults.format` - Default output format
- `defaults.no_color` - Disable colors
- `defaults.verbose` - Verbose logging
- `defaults.specular_dir` - Specular directory name
- `budget.max_cost_per_day` - Maximum daily cost
- `budget.max_cost_per_request` - Maximum per-request cost
- `budget.max_latency_ms` - Maximum latency in milliseconds
- `logging.level` - Log level (debug, info, warn, error)
- `logging.enable_file` - Enable file logging
- `logging.log_dir` - Log directory path
- `telemetry.enabled` - Enable telemetry
- `telemetry.share_usage` - Share usage data

**Example:**
```bash
$ specular config get providers.default
ollama

$ specular config get budget.max_cost_per_day
20.00
```

#### config set

Set a specific configuration value.

```bash
specular config set <key> <value>
```

**Example:**
```bash
$ specular config set providers.default anthropic
✓ Set providers.default = anthropic

$ specular config set budget.max_cost_per_day 50.0
✓ Set budget.max_cost_per_day = 50.0

$ specular config set defaults.verbose true
✓ Set defaults.verbose = true
```

#### config path

Display the path to the configuration file.

```bash
specular config path
```

**Example:**
```bash
$ specular config path
/Users/you/.specular/config.yaml
```

---

### status

Show environment and project status including health checks and next steps.

**Usage:**
```bash
specular status [--format text|json|yaml]
```

**Description:**

The `status` command provides a comprehensive overview of:
- Environment health (runtime, providers, API keys)
- Project initialization status
- Specification status (exists, locked, features count)
- Plan status (exists, tasks count)
- Build status (last build, success/failure)
- Issues that need attention
- Warnings about project state
- Recommended next steps

**Example Output:**

```
╔══════════════════════════════════════════════════════════════╗
║                      Project Status                          ║
╚══════════════════════════════════════════════════════════════╝

Environment:
  ✓ Runtime: docker
  ✓ AI Providers: 2 available
    • ollama
    • anthropic
  ✓ API Keys: 1 configured

Project:
  Directory: specular
  ✓ Initialized (.specular directory exists)
  ✓ Git repository (branch: main, clean)

Specification:
  ✓ Spec file exists (updated 2 hours ago)
  ✓ Locked (version: 1.0.0, 5 features)

Plan:
  ✓ Plan file exists (updated 1 hour ago)

📋 Next Steps:
   1. Execute plan with 'specular build'

✅ Project is healthy and ready
```

**JSON Output Example:**
```bash
$ specular status --format json
{
  "timestamp": "2025-11-12T10:30:00Z",
  "environment": {
    "runtime": "docker",
    "providers": ["ollama", "anthropic"],
    "api_keys": 1,
    "healthy": true
  },
  "project": {
    "directory": "specular",
    "initialized": true,
    "git_repo": true,
    "git_branch": "main",
    "git_dirty": false
  },
  "spec": {
    "exists": true,
    "locked": true,
    "version": "1.0.0",
    "features": 5,
    "last_updated": "2025-11-12T08:30:00Z"
  },
  "plan": {
    "exists": true,
    "tasks": 12,
    "last_updated": "2025-11-12T09:30:00Z"
  },
  "issues": [],
  "warnings": [],
  "next_steps": ["Execute plan with 'specular build'"],
  "healthy": true
}
```

**Status Indicators:**

- ✓ - Check passed
- ✗ - Check failed
- ⚠ - Warning

**Common Issues and Solutions:**

1. **No container runtime detected**
   ```
   ❌ Issues:
      • No container runtime detected (Docker/Podman required)

   📋 Next Steps:
      1. Install Docker from https://docker.com
   ```

2. **No AI providers detected**
   ```
   ❌ Issues:
      • No AI providers detected

   📋 Next Steps:
      1. Install Ollama or set API keys (OPENAI_API_KEY, ANTHROPIC_API_KEY)
   ```

3. **Project not initialized**
   ```
   ❌ Issues:
      • Project not initialized

   📋 Next Steps:
      1. Run 'specular init' to initialize project
   ```

4. **Git working directory dirty**
   ```
   ⚠️  Warnings:
      • Git working directory has uncommitted changes
   ```

---

### logs

View or tail Specular CLI logs and trace events.

**Usage:**
```bash
specular logs [flags]
specular logs list
```

**Description:**

Logs are stored in `~/.specular/logs/` with each workflow execution getting its own trace file named `trace_<id>.json`.

**Flags:**

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--tail` | | bool | Show only recent log entries |
| `--trace <id>` | | string | Show specific trace log by ID |
| `--follow` | `-f` | bool | Follow log output in real-time |
| `--lines <n>` | `-n` | int | Number of recent lines to show (default: 20) |

**Subcommands:**

#### logs (default)

Show recent log entries from the latest trace file.

```bash
specular logs [--lines 20]
```

**Example:**
```bash
$ specular logs
Showing last 20 lines from trace abc123:

[2025-11-12T10:15:00Z] workflow_start: Starting spec generation
[2025-11-12T10:15:01Z] provider_selected: Using ollama (cost: $0.00)
[2025-11-12T10:15:02Z] spec_generated: Generated 5 features
[2025-11-12T10:15:03Z] workflow_complete: Completed successfully
```

```bash
$ specular logs --lines 50
Showing last 50 lines from trace abc123:
...
```

#### logs --trace

Show a specific trace log by ID.

```bash
specular logs --trace <trace-id>
```

**Example:**
```bash
$ specular logs --trace abc123
Trace log: abc123

[1] {
  "timestamp": "2025-11-12T10:15:00Z",
  "level": "info",
  "message": "Starting spec generation",
  "type": "workflow_start"
}
...
```

#### logs --follow

Follow logs in real-time (like `tail -f`).

```bash
specular logs --follow
```

**Example:**
```bash
$ specular logs --follow
Following trace abc123 (Ctrl+C to stop):

{
  "timestamp": "10:15:00",
  "level": "info",
  "message": "Processing step 1",
  "type": "step_start"
}
...
```

#### logs list

List all available trace logs.

```bash
specular logs list [--format text|json|yaml]
```

**Example:**
```bash
$ specular logs list
Trace logs in /Users/you/.specular/logs:

  2025-11-12 10:15:00  abc123  (1.25 KB)
  2025-11-12 09:30:00  def456  (2.50 KB)
  2025-11-12 08:00:00  ghi789  (3.75 KB)

Total: 3 trace logs
```

**JSON Output:**
```bash
$ specular logs list --format json
[
  {
    "id": "abc123",
    "path": "/Users/you/.specular/logs/trace_abc123.json",
    "size": 1280,
    "created_at": "2025-11-12T10:15:00Z"
  },
  {
    "id": "def456",
    "path": "/Users/you/.specular/logs/trace_def456.json",
    "size": 2560,
    "created_at": "2025-11-12T09:30:00Z"
  }
]
```

**Log Structure:**

Each trace log is a JSON Lines file with structured events:

```json
{"timestamp":"2025-11-12T10:15:00Z","level":"info","type":"workflow_start","message":"Starting workflow"}
{"timestamp":"2025-11-12T10:15:01Z","level":"info","type":"provider_selected","provider":"ollama","cost":0.0}
{"timestamp":"2025-11-12T10:15:02Z","level":"info","type":"step_complete","step":1,"status":"success"}
{"timestamp":"2025-11-12T10:15:03Z","level":"error","type":"error","error":"Connection timeout"}
```

---

## Specification Commands

### spec

Manage project specifications.

**Usage:**
```bash
specular spec <subcommand> [flags]
```

**Subcommands:**

- `spec generate` - Generate specification from description
- `spec lock` - Lock specification to spec.lock.json
- `spec validate` - Validate specification format
- `spec show` - Display current specification

---

### interview

Interactive specification generation (legacy command).

**Usage:**
```bash
specular interview [flags]
```

**Note:** This command is being deprecated in favor of `specular spec new` in v1.4.x.

---

## Planning Commands

### plan

Manage execution plans for AI-powered development workflows.

**Usage:**
```bash
specular plan <subcommand> [flags]
```

**Description:**

Plan commands help you generate, validate, visualize, and review execution plans derived from specifications. Plans decompose high-level features into concrete, executable tasks.

**Subcommands:**

#### plan create

Generate execution plan from locked specification.

```bash
specular plan create [--in <file>] [--out <file>] [--feature <id>]
```

**Description:**

Creates an execution plan by analyzing the specification and decomposing features into tasks:
- Reads locked specification (spec.lock.json)
- Decomposes features into concrete tasks
- Establishes task dependencies and ordering
- Estimates effort and complexity
- Generates executable plan (plan.json)

**Example:**
```bash
$ specular plan create --in .specular/spec.yaml --out plan.json
Generating plan from spec.yaml...

✓ Loaded specification (5 features)
✓ Generated 12 tasks
✓ Established dependencies
✓ Estimated effort: 2-3 days

Plan saved to plan.json

Next steps:
  1. Review plan with 'specular plan review'
  2. Execute plan with 'specular build run --plan plan.json'
```

**Feature Filtering:**
```bash
$ specular plan create --feature feat-001
Generating plan for feature: feat-001...

✓ Generated 3 tasks for feat-001
✓ Plan saved to plan.json
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--in <file>` | string | Input specification file (default: .specular/spec.yaml) |
| `--out <file>` | string | Output plan file (default: plan.json) |
| `--lock <file>` | string | Lock file (default: .specular/spec.lock.json) |
| `--feature <id>` | string | Generate plan for specific feature only |
| `--estimate` | bool | Include effort estimates in plan |

**Backward Compatibility:**

The deprecated form `specular plan` (without subcommand) still works:
```bash
$ specular plan --in spec.yaml --out plan.json
⚠ Warning: 'plan' without subcommand is deprecated. Use 'plan create' instead.
```

---

#### plan visualize

Visualize plan task dependencies and execution flow.

```bash
specular plan visualize [--plan <file>] [--format text|dot|json]
```

**Description:**

Creates visual representation of plan structure:
- Task dependency graph
- Execution flow diagram
- Feature grouping
- Critical path analysis

**Example:**
```bash
$ specular plan visualize --plan plan.json

Plan Visualization:

feat-001: User Authentication
  ├─ task-001: Database schema [P0]
  ├─ task-002: API endpoints [P0] (depends: task-001)
  └─ task-003: UI components [P1] (depends: task-002)

feat-002: Payment Processing
  ├─ task-004: Payment gateway integration [P0]
  └─ task-005: Transaction logging [P1] (depends: task-004)

Critical Path: task-001 → task-002 → task-003 (estimated: 5 days)
```

**DOT Format (for Graphviz):**
```bash
$ specular plan visualize --plan plan.json --format dot > plan.dot
$ dot -Tpng plan.dot -o plan.png
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--plan <file>` | string | Plan file to visualize (default: plan.json) |
| `--format` | string | Output format: text, dot, json |
| `--feature <id>` | string | Visualize specific feature only |

---

#### plan validate

Validate plan structure and consistency.

```bash
specular plan validate [--plan <file>] [--strict]
```

**Description:**

Validates plan for:
- JSON structure correctness
- Required fields presence
- Task ID uniqueness
- Dependency consistency (no cycles)
- Feature coverage completeness
- Spec alignment verification

**Example:**
```bash
$ specular plan validate --plan plan.json
✓ Plan structure valid
✓ All task IDs unique
✓ Dependencies acyclic
✓ All features covered
✓ Plan aligns with spec

Plan validation passed.
```

**Strict Mode:**
```bash
$ specular plan validate --plan plan.json --strict
✓ Plan structure valid
✓ All task IDs unique
✓ Dependencies acyclic
✓ All features covered
⚠ Warning: 2 tasks missing effort estimates
✗ Error: task-005 references non-existent dependency task-999

Plan validation failed in strict mode.
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--plan <file>` | string | Plan file to validate (default: plan.json) |
| `--strict` | bool | Enable strict validation mode |
| `--json` | bool | Output validation results as JSON |

---

#### plan review

Review plan interactively or generate review report.

```bash
specular plan review [--plan <file>]
```

**Description:**

Provides structured review of the execution plan:
- Task breakdown summary
- Feature coverage analysis
- Dependency visualization
- Risk assessment
- Effort estimates

**Example:**
```bash
$ specular plan review --plan plan.json

Plan Review:

Summary:
  Features: 5
  Tasks: 12
  Estimated Effort: 2-3 days
  Critical Path: 5 days

Feature Coverage:
  ✓ feat-001: User Authentication (3 tasks)
  ✓ feat-002: Payment Processing (2 tasks)
  ✓ feat-003: Reporting Dashboard (4 tasks)
  ✓ feat-004: Email Notifications (2 tasks)
  ✓ feat-005: Admin Panel (1 task)

Risks:
  ⚠ task-002 has complex dependencies
  ⚠ feat-003 has tight timeline

Recommendations:
  • Review task-002 dependencies
  • Consider splitting feat-003 into smaller tasks
```

---

#### plan explain

Explain reasoning for specific task in plan.

```bash
specular plan explain <task-id> [--plan <file>]
```

**Description:**

Provides detailed explanation for a specific task:
- Why the task was created
- How it relates to features
- Why dependencies were established
- Estimated complexity reasoning

**Example:**
```bash
$ specular plan explain task-002 --plan plan.json

Task: task-002 (API endpoints)

Feature: feat-001 (User Authentication)

Description:
  Implement REST API endpoints for user authentication including
  login, logout, registration, and password reset functionality.

Dependencies:
  • task-001 (Database schema) - Required for user data storage

Complexity: Medium
  - Standard REST API patterns
  - Integration with existing auth framework
  - Comprehensive test coverage required

Estimated Effort: 1 day
```

---

## Build Commands

### build

Execute and manage plan builds with policy enforcement.

**Usage:**
```bash
specular build <subcommand> [flags]
```

**Description:**

Build commands orchestrate the execution of plans in sandboxed Docker environments with comprehensive policy enforcement, approval workflows, and build verification.

**Subcommands:**

#### build run

Execute the generated plan in sandboxed environment.

```bash
specular build run [--plan <file>] [--policy <file>] [flags]
```

**Description:**

Executes the plan with full governance and policy enforcement:
- Validates plan structure and dependencies
- Enforces execution policies (Docker, resources, network)
- Runs tasks in isolated containers
- Generates build artifacts and traces
- Creates build bundle for verification

**Example:**
```bash
$ specular build run --plan plan.json --policy .specular/policy.yaml
Building plan: plan.json

Policy Checks:
  ✓ Docker required: enabled
  ✓ Resource limits: configured
  ✓ Network isolation: enabled

Executing Tasks:
  ✓ task-001: Database schema (completed in 2m15s)
  ✓ task-002: API endpoints (completed in 5m30s)
  ✓ task-003: UI components (completed in 3m45s)

Build Summary:
  Tasks: 3/3 completed
  Duration: 11m30s
  Exit Code: 0

✓ Build bundle created: .specular/bundles/build-abc123.tar
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--plan <file>` | string | Plan file to execute (default: plan.json) |
| `--policy <file>` | string | Policy file (default: .specular/policy.yaml) |
| `--dry-run` | bool | Show what would be executed without running |
| `--resume` | bool | Resume from previous checkpoint |
| `--checkpoint-dir <dir>` | string | Checkpoint directory |
| `--checkpoint-id <id>` | string | Specific checkpoint to resume from |
| `--feature <id>` | string | Execute specific feature only |
| `--verbose` | bool | Enable verbose logging |
| `--enable-cache` | bool | Enable Docker image caching |
| `--cache-dir <dir>` | string | Cache directory location |
| `--cache-max-age <duration>` | duration | Maximum cache age (default: 168h) |
| `--keep-checkpoint` | bool | Keep checkpoint after successful build |

**Dry Run Example:**
```bash
$ specular build run --plan plan.json --dry-run
Dry run mode: No tasks will be executed

Plan: plan.json
  ✓ task-001: Database schema (golang:1.22)
  ✓ task-002: API endpoints (golang:1.22)
  ✓ task-003: UI components (node:20-alpine)

Would execute 3 tasks
```

**Backward Compatibility:**

The deprecated form `specular build` (without subcommand) still works:
```bash
$ specular build --plan plan.json
⚠ Warning: 'build' without subcommand is deprecated. Use 'build run' instead.
```

---

#### build verify

Run quality gate checks on build bundle.

```bash
specular build verify --bundle <file> [--strict]
```

**Description:**

Performs comprehensive verification of build artifacts:
- Validates bundle structure and integrity
- Verifies task completion status
- Checks test results and coverage
- Validates policy compliance
- Inspects artifacts for quality standards

**Example:**
```bash
$ specular build verify --bundle .specular/bundles/build-abc123.tar

Build Verification:

Bundle Structure:
  ✓ Manifest present
  ✓ Task logs complete
  ✓ Artifacts directory exists

Task Completion:
  ✓ All tasks completed successfully
  ✓ No failed tasks
  ✓ Exit codes valid

Test Results:
  ✓ All tests passed
  ✓ Coverage: 85% (>= 80% required)

Policy Compliance:
  ✓ Docker images from allowlist
  ✓ Resource limits enforced
  ✓ Network isolation verified

✓ Build verification passed
```

**Strict Mode:**
```bash
$ specular build verify --bundle build-abc123.tar --strict
⚠ Warning: Test coverage 85% below recommended 90%
⚠ Warning: 2 tasks missing effort tracking
✗ Error: Container exited with non-zero code

Build verification failed in strict mode.
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--bundle <file>` | string | Build bundle to verify (required) |
| `--strict` | bool | Enable strict verification mode |
| `--format` | string | Output format: text, json, yaml |

---

#### build approve

Approve build for deployment or next stage.

```bash
specular build approve <bundle-id> [--reason <text>]
```

**Description:**

Records approval for build deployment:
- Verifies build verification passed
- Creates approval record with audit trail
- Enables gated deployment workflow
- Updates build approval status

**Example:**
```bash
$ specular build approve build-abc123 --reason "All tests passed, ready for staging"
✓ Build approved
  Build ID: build-abc123
  Approver: user@example.com
  Timestamp: 2025-11-17T10:30:00Z
  Reason: All tests passed, ready for staging

Approval recorded in .specular/approvals/build-abc123.json

Next steps:
  1. Deploy with 'specular deploy --bundle build-abc123'
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--reason <text>` | string | Reason for approval (required) |
| `--approver <email>` | string | Approver email (defaults to git config) |

---

#### build explain

Explain build execution and task outcomes.

```bash
specular build explain [--bundle <file>] [--task <id>]
```

**Description:**

Provides detailed explanation of build execution:
- Overall build summary and statistics
- Task-by-task execution breakdown
- Resource usage and timing
- Policy enforcement details
- Failure analysis (if any)

**Example:**
```bash
$ specular build explain --bundle build-abc123.tar

Build Execution Summary:

Build ID: build-abc123
Started: 2025-11-17T10:00:00Z
Completed: 2025-11-17T10:11:30Z
Duration: 11m30s

Tasks Executed:
  1. task-001: Database schema
     Container: golang:1.22
     Duration: 2m15s
     Exit Code: 0
     Resources: CPU 0.5, Memory 256MB

  2. task-002: API endpoints
     Container: golang:1.22
     Duration: 5m30s
     Exit Code: 0
     Resources: CPU 1.0, Memory 512MB

  3. task-003: UI components
     Container: node:20-alpine
     Duration: 3m45s
     Exit Code: 0
     Resources: CPU 0.8, Memory 384MB

Policy Enforcement:
  ✓ All containers from allowlist
  ✓ Resource limits enforced
  ✓ Network isolation active
  ✓ No policy violations

Build Status: Success
```

**Task-Specific Explanation:**
```bash
$ specular build explain --bundle build-abc123.tar --task task-002

Task: task-002 (API endpoints)

Execution Details:
  Container: golang:1.22
  Command: go build ./cmd/api
  Working Dir: /workspace
  Environment: 5 variables
  Duration: 5m30s
  Exit Code: 0

Resource Usage:
  CPU: 1.0 cores (max allowed: 2.0)
  Memory: 512MB (max allowed: 2GB)
  Disk: 150MB written

Standard Output (last 20 lines):
  go: downloading github.com/gin-gonic/gin v1.9.1
  ...
  Build successful

Policy Checks:
  ✓ Image from allowlist
  ✓ Resource limits respected
  ✓ Network isolated
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--bundle <file>` | string | Build bundle to explain |
| `--task <id>` | string | Explain specific task only |
| `--format` | string | Output format: text, json, yaml |

---

## Bundle Commands

### bundle

Manage build bundles with quality gates and verification.

**Usage:**
```bash
specular bundle <subcommand> [flags]
```

**Description:**

Bundle commands help you create, verify, inspect, and manage build artifacts in a structured, portable format. Bundles package build outputs, logs, and metadata for verification and deployment.

**Subcommands:**

#### bundle create

Create build bundle from execution artifacts.

```bash
specular bundle create [--from <dir>] [--out <file>] [flags]
```

**Description:**

Packages build artifacts into a portable bundle:
- Collects task outputs and logs
- Captures metadata (timestamps, versions, hashes)
- Packages into compressed tarball
- Generates manifest with verification data

**Example:**
```bash
$ specular bundle create --from .specular/runs/abc123 --out build-abc123.tar
Creating bundle from: .specular/runs/abc123

Packaging:
  ✓ Collected 3 task outputs
  ✓ Captured build logs
  ✓ Generated manifest
  ✓ Compressed artifacts

Bundle created: build-abc123.tar (5.2 MB)

Manifest:
  Build ID: abc123
  Tasks: 3
  Created: 2025-11-17T10:11:30Z
  SHA256: a1b2c3d4...
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--from <dir>` | string | Source directory (default: latest run) |
| `--out <file>` | string | Output bundle file |
| `--compression <level>` | string | Compression level: none, fast, best |

**Backward Compatibility:**

The deprecated form `bundle build` still works:
```bash
$ specular bundle build --from runs/abc123
⚠ Warning: 'bundle build' is deprecated. Use 'bundle create' instead.
```

---

#### bundle gate

Run quality gate checks on bundle.

```bash
specular bundle gate --bundle <file> [--strict]
```

**Description:**

Performs comprehensive quality gate verification:
- Validates bundle structure and integrity
- Checks all tasks completed successfully
- Verifies test results and coverage thresholds
- Validates policy compliance
- Runs security and quality checks

**Example:**
```bash
$ specular bundle gate --bundle build-abc123.tar

Quality Gate Checks:

Structure:
  ✓ Bundle format valid
  ✓ Manifest present
  ✓ SHA256 verified

Completion:
  ✓ 3/3 tasks completed
  ✓ All exit codes 0

Tests:
  ✓ All tests passed (45/45)
  ✓ Coverage: 87% (>= 80%)

Security:
  ✓ No vulnerabilities detected
  ✓ All images from allowlist

Quality:
  ✓ No linting errors
  ✓ Code complexity acceptable

✓ All quality gates passed
```

**Strict Mode:**
```bash
$ specular bundle gate --bundle build-abc123.tar --strict
✓ Bundle format valid
✓ Tasks completed
⚠ Warning: Coverage 87% below recommended 90%
✗ Error: 2 medium-severity vulnerabilities found

Quality gate failed in strict mode.
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--bundle <file>` | string | Bundle file to verify (required) |
| `--strict` | bool | Enable strict mode with higher thresholds |
| `--format` | string | Output format: text, json, yaml |

**Backward Compatibility:**

The deprecated form `bundle verify` still works:
```bash
$ specular bundle verify --bundle build-abc123.tar
⚠ Warning: 'bundle verify' is deprecated. Use 'bundle gate' instead.
```

---

#### bundle inspect

Inspect bundle contents and metadata.

```bash
specular bundle inspect <bundle-file> [flags]
```

**Description:**

Provides detailed information about bundle contents:
- Manifest metadata (build ID, timestamps, versions)
- Task list with status and outputs
- File listing with sizes
- Verification checksums
- Policy compliance summary

**Example:**
```bash
$ specular bundle inspect build-abc123.tar

Bundle: build-abc123.tar

Metadata:
  Build ID: abc123
  Created: 2025-11-17T10:11:30Z
  Size: 5.2 MB
  SHA256: a1b2c3d4...

Tasks (3):
  1. task-001: Database schema
     Status: completed
     Exit Code: 0
     Duration: 2m15s
     Outputs: schema.sql (15 KB)

  2. task-002: API endpoints
     Status: completed
     Exit Code: 0
     Duration: 5m30s
     Outputs: api-server (8.5 MB)

  3. task-003: UI components
     Status: completed
     Exit Code: 0
     Duration: 3m45s
     Outputs: dist/ (2.1 MB)

Files (12):
  manifest.json         5.2 KB
  task-001/schema.sql  15.0 KB
  task-002/api-server   8.5 MB
  task-003/dist/        2.1 MB
  ...

Policy Compliance:
  ✓ Docker images verified
  ✓ Resource limits respected
  ✓ Tests passed
```

**JSON Output:**
```bash
$ specular bundle inspect build-abc123.tar --format json
{
  "build_id": "abc123",
  "created": "2025-11-17T10:11:30Z",
  "size": 5452595,
  "sha256": "a1b2c3d4...",
  "tasks": [...]
}
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--format` | string | Output format: text, json, yaml |
| `--tasks` | bool | Show task details only |
| `--files` | bool | Show file listing only |

---

#### bundle list

List all bundles with metadata.

```bash
specular bundle list [--format text|json|yaml]
```

**Description:**

Lists all bundles in the bundles directory with:
- Build ID and timestamp
- Bundle size
- Task count and status
- Approval status
- Age since creation

**Example:**
```bash
$ specular bundle list

Bundles in .specular/bundles:

  build-abc123.tar
    Created: 2 hours ago (2025-11-17T10:11:30Z)
    Size: 5.2 MB
    Tasks: 3/3 completed
    Status: Approved ✓

  build-def456.tar
    Created: 1 day ago (2025-11-16T14:30:00Z)
    Size: 8.7 MB
    Tasks: 5/5 completed
    Status: Pending approval

  build-ghi789.tar
    Created: 3 days ago (2025-11-14T09:15:00Z)
    Size: 4.1 MB
    Tasks: 2/3 completed
    Status: Failed ✗

Total: 3 bundles (1 approved, 1 pending, 1 failed)
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--format` | string | Output format: text, json, yaml |
| `--status <status>` | string | Filter by status: approved, pending, failed |
| `--days <n>` | int | Show bundles from last N days |

---

## Drift Detection Commands

### drift

Detect and manage plan, code, and infrastructure drift.

**Usage:**
```bash
specular drift <subcommand> [flags]
```

**Description:**

Drift commands help you detect and manage divergence between specifications, plans, code, and deployed infrastructure. This ensures implementations stay aligned with approved plans.

**Subcommands:**

#### drift check

Run comprehensive drift detection.

```bash
specular drift check [--spec <file>] [--plan <file>] [--report <file>]
```

**Description:**

Performs multi-layer drift detection:
- **Plan Drift**: Spec vs. plan alignment
- **Code Drift**: Plan vs. implemented code
- **Infrastructure Drift**: Code vs. deployed infrastructure
- **API Drift**: OpenAPI spec vs. actual endpoints

**Example:**
```bash
$ specular drift check --spec .specular/spec.yaml --plan plan.json --report drift.sarif

Drift Detection:

Plan Drift:
  ✓ All features covered in plan
  ✓ Plan aligns with spec v1.0.0

Code Drift:
  ⚠ 2 tasks partially implemented
    • task-002: API endpoints (60% complete)
    • task-005: Email service (not started)
  ✓ 1 task fully implemented
    • task-001: Database schema

Infrastructure Drift:
  ✓ Deployed version matches code
  ✓ Configuration aligned

API Drift:
  ✗ 1 endpoint not in spec
    • POST /api/v1/debug (undocumented)
  ⚠ 1 endpoint signature mismatch
    • GET /api/v1/users response schema differs

Drift Summary:
  Plan Drift: No issues ✓
  Code Drift: 2 warnings
  Infrastructure Drift: No issues ✓
  API Drift: 1 error, 1 warning

SARIF report saved to: drift.sarif
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--spec <file>` | string | Specification file |
| `--plan <file>` | string | Plan file |
| `--lock <file>` | string | Lock file for version verification |
| `--api-spec <file>` | string | OpenAPI specification for API drift |
| `--report <file>` | string | SARIF report output file |
| `--project-root <dir>` | string | Project root directory |

**Backward Compatibility:**

The deprecated form `plan drift` still works:
```bash
$ specular plan drift --spec spec.yaml
⚠ Warning: 'plan drift' is deprecated. Use 'drift check' instead.
```

---

#### drift approve

Approve detected drift with justification.

```bash
specular drift approve [--drift-id <id>] [--reason <text>]
```

**Description:**

Records approval for acceptable drift:
- Documents why drift is acceptable
- Creates approval record with audit trail
- Prevents drift from blocking workflows
- Requires justification for compliance

**Example:**
```bash
$ specular drift approve --drift-id drift-abc123 --reason "Debug endpoint for development only, will remove before production"
✓ Drift approved
  Drift ID: drift-abc123
  Type: API drift
  Item: POST /api/v1/debug
  Approver: user@example.com
  Timestamp: 2025-11-17T10:30:00Z
  Reason: Debug endpoint for development only, will remove before production

Approval recorded in .specular/approvals/drift-abc123.json

Note: Approved drift should be resolved before production deployment.
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--drift-id <id>` | string | Drift detection ID to approve |
| `--reason <text>` | string | Justification for approval (required) |
| `--approver <email>` | string | Approver email (defaults to git config) |
| `--temporary` | bool | Mark as temporary approval (requires resolution) |

---

## Autonomous Mode Commands

### auto

Run autonomous workflow from description to implementation.

**Usage:**
```bash
specular auto "<description>" [flags]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--max-steps <n>` | int | Maximum steps to execute |
| `--scope <scope>` | string | Limit scope (module, file, function) |
| `--interactive` | bool | Enable interactive TUI mode |
| `--resume <checkpoint>` | string | Resume from checkpoint |
| `--output <dir>` | string | Directory to save spec/plan files |

**Example:**
```bash
$ specular auto "Add user authentication with JWT"

$ specular auto "Refactor payment processing" --scope module:payment

$ specular auto "Fix bug in login" --interactive
```

---

## Checkpoint Commands

### checkpoint

Manage workflow checkpoints for resume capability.

**Usage:**
```bash
specular checkpoint <subcommand>
```

**Subcommands:**

- `checkpoint list` - List available checkpoints
- `checkpoint show <id>` - Show checkpoint details

---

## Provider Commands

### provider

Manage AI provider configuration, health, and capabilities.

**Usage:**
```bash
specular provider <subcommand> [flags]
```

**Description:**

Provider commands help you manage AI provider integrations, test connectivity, configure routing preferences, and monitor provider health.

**Subcommands:**

#### provider list

List available AI providers with status and capabilities.

```bash
specular provider list [--format text|json|yaml]
```

**Description:**

Displays all configured and detected AI providers:
- Provider name and type (API, CLI, local)
- Availability status
- Model catalog
- Cost estimates
- Configuration status

**Example:**
```bash
$ specular provider list

Available Providers:

Ollama (local)
  Status: ✓ Available
  Models: 5
    • llama3.2:latest
    • codellama:latest
    • mistral:latest
    • phi3:latest
    • qwen2.5-coder:latest
  Cost: Free

Anthropic (API)
  Status: ✓ Configured
  API Key: ✓ Valid (ANTHROPIC_API_KEY)
  Models: 3
    • claude-3-7-sonnet
    • claude-3-5-sonnet
    • claude-3-opus
  Cost: $3-$15 per 1M tokens

OpenAI (API)
  Status: ✗ Not configured
  API Key: Not set (OPENAI_API_KEY)

Total: 3 providers (2 available, 1 not configured)
```

**JSON Output:**
```bash
$ specular provider list --format json
[
  {
    "name": "ollama",
    "type": "local",
    "status": "available",
    "models": 5,
    "cost": "free"
  },
  {
    "name": "anthropic",
    "type": "api",
    "status": "configured",
    "api_key_set": true,
    "models": 3,
    "cost": "$3-$15/1M tokens"
  }
]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--format` | string | Output format: text, json, yaml |
| `--available` | bool | Show only available providers |

---

#### provider add

Add new AI provider to configuration.

```bash
specular provider add <name> [flags]
```

**Description:**

Dynamically adds a provider to the configuration:
- Supports: ollama, anthropic, openai, gemini, claude-code, codex-cli, copilot-cli
- Auto-detects local providers (ollama)
- Configures API keys for cloud providers
- Updates providers.yaml configuration

**Example:**
```bash
$ specular provider add anthropic --api-key $ANTHROPIC_API_KEY
✓ Added provider: anthropic
  Type: API
  API Key: Configured
  Models: 3

Provider configuration updated in .specular/providers.yaml

Next steps:
  1. Test provider with 'specular provider test anthropic'
  2. Set as default with 'specular provider set-default anthropic'
```

**Add Local Provider:**
```bash
$ specular provider add ollama
✓ Detected Ollama at http://localhost:11434
✓ Added provider: ollama
  Type: Local
  Models: 5

Provider configured successfully.
```

**Supported Providers:**
- `ollama` - Local LLM runtime
- `anthropic` - Anthropic Claude API
- `openai` - OpenAI GPT API
- `gemini` - Google Gemini API
- `claude-code` - Claude Code CLI provider
- `codex-cli` - Codex CLI provider
- `copilot-cli` - GitHub Copilot CLI provider

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--api-key <key>` | string | API key for cloud providers |
| `--endpoint <url>` | string | Custom endpoint URL (for local providers) |
| `--default` | bool | Set as default provider after adding |

---

#### provider remove

Remove AI provider from configuration.

```bash
specular provider remove <name>
```

**Description:**

Removes a provider from the configuration:
- Removes from providers.yaml
- Cleans up provider-specific settings
- Updates routing preferences

**Example:**
```bash
$ specular provider remove openai
⚠ This will remove provider 'openai' from configuration.
Continue? (y/N): y

✓ Removed provider: openai
✓ Updated providers.yaml

Remaining providers: 2 (ollama, anthropic)
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--force` | bool | Skip confirmation prompt |

---

#### provider doctor

Run health checks on provider configuration and connectivity.

```bash
specular provider doctor [--provider <name>] [--format text|json|yaml]
```

**Description:**

Performs comprehensive provider health checks:
- API key validation
- Connectivity tests
- Model availability
- Response latency
- Rate limit status

**Example:**
```bash
$ specular provider doctor

Provider Health Checks:

Ollama:
  ✓ Service running (http://localhost:11434)
  ✓ 5 models available
  ✓ Response time: 45ms
  ✓ No rate limits

Anthropic:
  ✓ API key valid
  ✓ API accessible
  ✓ 3 models available
  ✓ Response time: 320ms
  ✓ Rate limit: 50/min (current: 0)

OpenAI:
  ✗ API key not configured
  ⚠ Set OPENAI_API_KEY environment variable

All configured providers operational.
```

**Check Specific Provider:**
```bash
$ specular provider doctor --provider anthropic

Anthropic Provider Health:
  ✓ API Key: Valid (expires in 45 days)
  ✓ Connectivity: OK (latency: 320ms)
  ✓ Models: 3 available
    • claude-3-7-sonnet ✓
    • claude-3-5-sonnet ✓
    • claude-3-opus ✓
  ✓ Rate Limits: 50/min (current usage: 0)
  ✓ Quota: 90% remaining

Provider is healthy.
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--provider <name>` | string | Check specific provider only |
| `--format` | string | Output format: text, json, yaml |
| `--verbose` | bool | Show detailed diagnostic information |

**Backward Compatibility:**

The deprecated form `provider health` still works:
```bash
$ specular provider health
⚠ Warning: 'provider health' is deprecated. Use 'provider doctor' instead.
```

---

#### provider test

Test specific provider connectivity and response.

```bash
specular provider test <name>
```

**Description:**

Tests provider with a simple request:
- Validates authentication
- Tests model inference
- Measures response time
- Verifies output quality

**Example:**
```bash
$ specular provider test ollama
Testing provider: ollama

✓ Connection successful
✓ Model: llama3.2:latest
✓ Request: "Hello, world!" → Response received
✓ Response time: 1.2s
✓ Output quality: Valid

Provider test passed.
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--model <name>` | string | Specific model to test |
| `--prompt <text>` | string | Custom test prompt |

---

#### provider set-default

Set default AI provider for all operations.

```bash
specular provider set-default <name>
```

**Description:**

Sets the default provider used when no explicit provider is specified:
- Updates global configuration
- Applies to all future operations
- Can be overridden per-command with --provider flag

**Example:**
```bash
$ specular provider set-default anthropic
✓ Set default provider: anthropic

All operations will use Anthropic Claude unless overridden.

To use a different provider for specific commands:
  specular spec generate --provider ollama
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--global` | bool | Set as system-wide default (all projects) |

---

## Utility Commands

### version

Display version information.

**Usage:**
```bash
specular version
```

**Example:**
```bash
$ specular version
Specular CLI v1.4.0
Build: abc123def
Built: 2025-11-12T10:00:00Z
```

---

## Exit Codes

Specular uses standardized exit codes for workflow automation:

| Code | Name | Description |
|------|------|-------------|
| 0 | Success | Operation completed successfully |
| 1 | GeneralError | General error or unknown failure |
| 2 | PolicyViolation | Policy check failed |
| 3 | BuildFailure | Build or execution failed |
| 4 | ValidationError | Input validation failed |
| 5 | ProviderError | AI provider error |
| 6 | ResourceError | Resource limit exceeded (budget, time) |

**Example Usage in Scripts:**
```bash
#!/bin/bash
specular build
exit_code=$?

if [ $exit_code -eq 2 ]; then
    echo "Policy violation detected, cannot proceed"
    exit 1
elif [ $exit_code -eq 6 ]; then
    echo "Budget exceeded, stopping workflow"
    exit 1
fi
```

---

## Environment Variables

Specular recognizes the following environment variables:

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `GEMINI_API_KEY` | Google Gemini API key |
| `EDITOR` | Default text editor (for `config edit`) |
| `NO_COLOR` | Disable colored output |
| `SPECULAR_CONFIG` | Path to config file (default: `~/.specular/config.yaml`) |

---

## Configuration File

The global configuration file is located at `~/.specular/config.yaml`:

```yaml
providers:
  default: ollama
  preference:
    - ollama
    - anthropic
    - openai
    - gemini

defaults:
  format: text
  no_color: false
  verbose: false
  specular_dir: .specular

budget:
  max_cost_per_day: 20.0
  max_cost_per_request: 1.0
  max_latency_ms: 60000

logging:
  level: info
  enable_file: true
  log_dir: ~/.specular/logs

telemetry:
  enabled: false
  share_usage: false
```

---

## Project Structure

Specular projects use the following directory structure:

```
.specular/
├── spec.yaml              # Project specification
├── spec.lock.json         # Locked specification
├── plan.json              # Generated plan
├── config.yaml            # Project-specific config
└── runs/                  # Build execution records
    └── <timestamp>/
        ├── manifest.json
        ├── logs.txt
        └── artifacts/
```

---

## See Also

- [Getting Started Guide](getting-started.md)
- [Provider Guide](provider-guide.md)
- [CLI Alignment Plan](CLI_ALIGNMENT_PLAN.md)
- [Development Guide](DEVELOPER_GUIDE.md)
