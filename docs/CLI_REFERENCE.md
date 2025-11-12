# Specular CLI Reference

Complete reference for Specular CLI commands and flags.

## Table of Contents

- [Overview](#overview)
- [Global Flags](#global-flags)
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

Generate execution plan from locked specification.

**Usage:**
```bash
specular plan [flags]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--output <file>` | string | Save plan to specific file |

---

## Build Commands

### build

Execute the generated plan.

**Usage:**
```bash
specular build [flags]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--dry-run` | bool | Show what would be executed |
| `--approve` | bool | Skip approval prompts |

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

Manage AI provider configuration and detection.

**Usage:**
```bash
specular provider <subcommand>
```

**Subcommands:**

- `provider list` - List available providers
- `provider test <name>` - Test provider connectivity
- `provider set-default <name>` - Set default provider

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
