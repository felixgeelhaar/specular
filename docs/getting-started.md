# Getting Started with Specular

Specular is an AI-Native Spec and Build Assistant that enables spec-first, policy-enforced software development. This guide will help you get up and running quickly.

## Prerequisites

- **Docker**: Required for execution sandboxing
  - Docker Engine 24.0+ or Docker Desktop
  - Verify: `docker version`
- **AI Provider**: At least one of the following
  - Anthropic Claude (API key)
  - OpenAI GPT (API key)
  - Google Gemini (API key)
  - Ollama (self-hosted, no API key)

## Installation

### From Source (Development)

```bash
# Clone the repository
git clone https://github.com/felixgeelhaar/specular.git
cd specular

# Build the binary
make build

# Verify installation
./specular version
```

### Via Homebrew (Coming Soon)

```bash
brew tap felixgeelhaar/tap
brew install specular
specular version
```

### Via Download (Coming Soon)

Download pre-built binaries from the [releases page](https://github.com/felixgeelhaar/specular/releases).

## Quick Start Tutorial

This tutorial walks you through creating your first spec-driven project with Specular.

### 1. Set Up AI Provider

Specular needs at least one AI provider configured. Let's use Anthropic Claude:

```bash
# Set your API key
export ANTHROPIC_API_KEY="your-api-key-here"

# Initialize provider configuration
specular provider init

# Verify provider health
specular provider health
```

Example output:
```
✓ anthropic:claude-3-5-sonnet-20241022 (healthy) - Default provider
  Model hint: general, fast, codegen, agentic
  Budget: unlimited
```

### 2. Create Your First Spec

Create a directory for your project:

```bash
mkdir my-app
cd my-app
```

Create a Product Requirements Document (PRD) at `PRD.md`:

```markdown
# My App

## Overview
A simple task management CLI application.

## Requirements

### R1: Task Creation
**Priority:** P0
**Complexity:** 3

Users must be able to create tasks with:
- Title (required)
- Description (optional)
- Due date (optional)

**Acceptance Criteria:**
- Task has unique ID
- Task is persisted to storage
- Command: `myapp add "Task title"`

### R2: List Tasks
**Priority:** P0
**Complexity:** 2

Users must be able to list all tasks.

**Acceptance Criteria:**
- Shows all tasks with ID, title, status
- Command: `myapp list`

### R3: Complete Tasks
**Priority:** P1
**Complexity:** 2

Users must be able to mark tasks as complete.

**Acceptance Criteria:**
- Updates task status
- Command: `myapp complete <task-id>`
```

### 3. Generate Technical Spec

Convert the PRD into a technical specification:

```bash
specular spec generate --in PRD.md --out .specular/spec.yaml
```

This creates a structured YAML specification with:
- Parsed requirements
- Priority and complexity assignments
- Acceptance criteria extraction
- Dependency detection

View the generated spec:
```bash
cat .specular/spec.yaml
```

### 4. Create Policy File

Define organizational guardrails at `.specular/policy.yaml`:

```yaml
version: "1.0"

policies:
  # Code quality
  - id: test-coverage
    description: "Require 80% test coverage"
    rule: "coverage >= 0.80"
    severity: error

  # Security
  - id: no-hardcoded-secrets
    description: "Prevent hardcoded secrets"
    rule: "!contains(code, 'API_KEY') || contains(code, 'os.Getenv')"
    severity: error

  # Dependencies
  - id: approved-licenses
    description: "Only use approved licenses"
    rule: "license in ['MIT', 'Apache-2.0', 'BSD-3-Clause']"
    severity: warning

constraints:
  max_complexity: 8
  max_file_size: 500
  required_tests: true
```

### 5. Generate Implementation Plan

Create an executable plan from the spec:

```bash
specular plan generate \
  --spec .specular/spec.yaml \
  --policy .specular/policy.yaml \
  --out plan.json
```

Example plan output:
```json
{
  "version": "1.0",
  "requirements": [
    {
      "id": "R1",
      "title": "Task Creation",
      "priority": "P0",
      "complexity": 3,
      "tasks": [
        {
          "id": "R1-T1",
          "description": "Define Task data structure",
          "type": "code",
          "estimated_effort": 1
        },
        {
          "id": "R1-T2",
          "description": "Implement storage layer",
          "type": "code",
          "estimated_effort": 2
        }
      ]
    }
  ]
}
```

### 6. Build with AI Assistance

Execute the plan with AI-powered implementation:

```bash
specular build \
  --plan plan.json \
  --policy .specular/policy.yaml \
  --verbose
```

Specular will:
1. ✅ Execute tasks in dependency order
2. 🤖 Generate code using AI providers
3. 🔒 Run code in sandboxed Docker containers
4. 📊 Validate against policy constraints
5. ✅ Run tests and verify acceptance criteria

### 7. Checkpoint and Resume (Long-Running Operations)

For large projects with many tasks, builds and evaluations can be interrupted. Specular supports checkpoint/resume to continue from where you left off:

```bash
# Start a build with automatic checkpointing
specular build \
  --plan plan.json \
  --policy .specular/policy.yaml \
  --checkpoint-dir .specular/checkpoints

# If interrupted, resume from the checkpoint
specular build \
  --plan plan.json \
  --policy .specular/policy.yaml \
  --resume
```

**How it works:**
- Checkpoints are automatically saved to `.specular/checkpoints/`
- Each checkpoint has a unique ID: `build-{plan}-{timestamp}` or `eval-{plan}-{timestamp}`
- When resuming, Specular shows progress:
  ```
  Resuming from checkpoint: build-plan.json-1234567890
    Completed: 15 tasks
    Pending: 5 tasks
    Failed: 0 tasks
  ```
- Completed tasks are skipped on resume
- Failed tasks can be retried
- Checkpoints are automatically cleaned up on successful completion

**Checkpoint Options:**

```bash
# Specify custom checkpoint ID
specular build \
  --plan plan.json \
  --checkpoint-id my-custom-checkpoint \
  --resume

# Keep checkpoint after completion (for inspection)
specular build \
  --plan plan.json \
  --keep-checkpoint

# Use custom checkpoint directory
specular build \
  --plan plan.json \
  --checkpoint-dir /path/to/checkpoints
```

**Checkpoint Structure:**

Checkpoints are stored as JSON files containing:
- Operation metadata (plan, policy files)
- Task states (pending, running, completed, failed, skipped)
- Progress tracking (completion percentage)
- Error information for failed tasks
- Retry attempt counts
- Artifact paths

Example checkpoint content:
```json
{
  "version": "1.0",
  "operation_id": "build-plan.json-1234567890",
  "started_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-15T10:15:00Z",
  "status": "running",
  "tasks": {
    "task1": {
      "id": "task1",
      "status": "completed",
      "started_at": "2025-01-15T10:00:00Z",
      "completed_at": "2025-01-15T10:05:00Z",
      "attempts": 1
    },
    "task2": {
      "id": "task2",
      "status": "running",
      "started_at": "2025-01-15T10:05:00Z",
      "attempts": 1
    }
  },
  "metadata": {
    "plan": "plan.json",
    "policy": ".specular/policy.yaml"
  }
}
```

### 8. Verify Implementation

Check that requirements are met:

```bash
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --results .specular/results.json
```

This validates:
- All acceptance criteria are met
- Policy constraints are satisfied
- Tests pass with required coverage
- No drift from specification

The eval command also supports checkpoint/resume for long-running evaluations:

```bash
# Run evaluation with checkpointing
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --resume

# Evaluation checkpoints track each phase
# - quality-gate: Policy validation
# - plan-drift: Spec hash verification
# - code-drift: Contract test execution
# - infra-drift: Policy violations
# - report-generation: SARIF output
```

## Core Concepts

### Specs vs Plans

- **Spec** (`.specular/spec.yaml`): WHAT to build
  - Generated from PRD
  - Requirements and acceptance criteria
  - Static, version-controlled

- **Plan** (`plan.json`): HOW to build
  - Generated from spec
  - Executable tasks with dependencies
  - Can be regenerated as implementation evolves

### Policy Enforcement

Policies act as guardrails during development:

```yaml
policies:
  - id: architecture-pattern
    description: "Use repository pattern for data access"
    rule: "contains(files, 'repository') && !contains(files, 'direct-db')"
    severity: error
```

### Provider Routing

Specular intelligently routes requests to AI providers based on:

- **Model hints**: `fast`, `general`, `codegen`, `agentic`
- **Complexity**: Simple vs complex tasks
- **Cost optimization**: Budget constraints
- **Availability**: Fallback chains

Example configuration (`.specular/providers.yaml`):
```yaml
providers:
  - name: anthropic
    type: api
    models:
      - id: claude-3-5-sonnet-20241022
        hints: [general, fast, codegen, agentic]
        budget: 100.00

  - name: ollama
    type: executable
    models:
      - id: codellama
        hints: [codegen]
        budget: unlimited
```

### Drift Detection

Specular tracks drift between spec and implementation:

```bash
specular drift detect \
  --spec .specular/spec.yaml \
  --codebase ./src \
  --out drift-report.json
```

Drift types:
- **Feature drift**: Code without spec
- **Spec drift**: Spec without code
- **Behavior drift**: Implementation doesn't match spec

## Common Workflows

### Workflow 1: New Feature Development

```bash
# 1. Update PRD with new requirement
echo "### R4: Task Priority" >> PRD.md

# 2. Regenerate spec
specular spec generate --in PRD.md --out .specular/spec.yaml --incremental

# 3. Generate plan for new requirement only
specular plan generate \
  --spec .specular/spec.yaml \
  --policy .specular/policy.yaml \
  --filter "priority=P0" \
  --out plan-feature.json

# 4. Implement
specular build --plan plan-feature.json --policy .specular/policy.yaml

# 5. Verify
specular eval --spec .specular/spec.yaml --plan plan-feature.json
```

### Workflow 2: Refactoring

```bash
# 1. Update policy with new constraint
cat >> .specular/policy.yaml << EOF
  - id: max-function-length
    description: "Functions must be under 50 lines"
    rule: "function_length <= 50"
    severity: warning
EOF

# 2. Detect violations
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --policy .specular/policy.yaml

# 3. Generate refactoring plan
specular plan refactor \
  --violations violations.json \
  --out refactor-plan.json

# 4. Execute refactoring
specular build --plan refactor-plan.json --policy .specular/policy.yaml
```

### Workflow 3: Resuming Interrupted Builds

When a long-running build is interrupted (network failure, system crash, etc.), use checkpoint/resume:

```bash
# Start a build (automatically checkpointed)
specular build \
  --plan large-plan.json \
  --policy .specular/policy.yaml

# Build gets interrupted after completing 50 of 100 tasks
# Press Ctrl+C or system crashes

# Resume from where you left off
specular build \
  --plan large-plan.json \
  --policy .specular/policy.yaml \
  --resume

# Output shows:
# Resuming from checkpoint: build-large-plan.json-1234567890
#   Completed: 50 tasks
#   Pending: 48 tasks
#   Failed: 2 tasks
#
# ✓ Task1 already completed (skipping)
# ✓ Task2 already completed (skipping)
# ...
# ⟳ Task51 (retry attempt 2)
# ⟲ Task52 (starting)

# After successful completion, inspect checkpoint before cleanup
specular build \
  --plan large-plan.json \
  --keep-checkpoint

# View checkpoint details
cat .specular/checkpoints/build-large-plan.json-*.json

# Manually clean up checkpoint when done
rm .specular/checkpoints/build-large-plan.json-*.json
```

### Workflow 4: Continuous Validation

```bash
# In CI/CD pipeline
specular drift detect --spec .specular/spec.yaml --codebase ./src
specular eval --spec .specular/spec.yaml --plan plan.json
specular policy check --policy .specular/policy.yaml --codebase ./src
```

## CI/CD Integration with GitHub Actions

Specular provides a GitHub Action for seamless integration into your CI/CD pipelines. This enables automated spec generation, drift detection, policy enforcement, and continuous validation on every pull request and merge.

### Setup

The Specular GitHub Action is available directly from your repository once you've added the `action.yml` file. No installation is required - just reference the action in your workflow files.

#### Step 1: Configure API Keys

Set up your AI provider API keys as GitHub Secrets:

1. Navigate to your repository on GitHub
2. Go to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Add your API key(s):
   - `ANTHROPIC_API_KEY` - For Claude models (recommended)
   - `OPENAI_API_KEY` - For GPT models
   - `GEMINI_API_KEY` - For Google Gemini models

**Security Note:** Never commit API keys to your repository. Always use GitHub Secrets.

#### Step 2: Create a Workflow File

Create a workflow file in `.github/workflows/` directory. You can use one of the examples below or create a custom workflow.

### Example 1: Pull Request Validation

This workflow automatically validates pull requests by detecting drift and posting results as comments:

Create `.github/workflows/pr-validation.yml`:

```yaml
name: PR Validation

on:
  pull_request:
    branches: [ main, develop ]
    paths:
      - 'src/**'
      - 'lib/**'
      - '.specular/**'
      - 'PRD.md'

permissions:
  contents: read
  security-events: write
  pull-requests: write

jobs:
  validate:
    name: Validate PR with Specular
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Docker
        uses: docker/setup-buildx-action@v3

      - name: Run Drift Detection
        uses: ./
        with:
          command: 'eval'
          spec-file: '.specular/spec.yaml'
          plan-file: 'plan.json'
          policy-file: '.specular/policy.yaml'
          report-file: 'drift-report.sarif'
          fail-on-drift: 'true'
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Comment PR with Results
        if: always()
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');

            let comment = '## 🔍 Specular Validation Results\n\n';

            if (fs.existsSync('drift-report.sarif')) {
              const sarif = JSON.parse(fs.readFileSync('drift-report.sarif', 'utf8'));
              const results = sarif.runs[0].results || [];

              comment += `**Total Findings:** ${results.length}\n\n`;

              if (results.length > 0) {
                comment += '### Findings\n\n';
                results.slice(0, 10).forEach(r => {
                  const level = r.level || 'note';
                  const icon = level === 'error' ? '❌' : level === 'warning' ? '⚠️' : 'ℹ️';
                  comment += `${icon} **${r.ruleId}**: ${r.message.text}\n`;
                });

                if (results.length > 10) {
                  comment += `\n_...and ${results.length - 10} more findings._\n`;
                }
              } else {
                comment += '✅ No drift detected! Code matches specification.\n';
              }
            } else {
              comment += '⚠️ No SARIF report generated.\n';
            }

            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: comment
            });

      - name: Upload SARIF to Security Tab
        if: always()
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: drift-report.sarif
          category: specular-pr-validation
```

**What this workflow does:**
- ✅ Runs on every pull request to `main` or `develop`
- 🔍 Detects drift between spec and code
- 📊 Posts findings as PR comment
- 🔒 Uploads results to GitHub Security tab
- ❌ Fails PR if drift is detected

### Example 2: Continuous Build Pipeline

This workflow implements a complete spec-first continuous build with multiple stages:

Create `.github/workflows/continuous-build.yml`:

```yaml
name: Continuous Build

on:
  push:
    branches: [ main, develop ]
    paths:
      - 'PRD.md'
      - '.specular/**'
      - 'src/**'
  workflow_dispatch:

permissions:
  contents: read
  security-events: write

jobs:
  spec-generation:
    name: Generate Specification
    runs-on: ubuntu-latest
    if: contains(github.event.head_commit.modified, 'PRD.md')

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Generate Spec from PRD
        uses: ./
        with:
          command: 'spec'
          prd-file: 'PRD.md'
          spec-file: '.specular/spec.yaml'
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
          verbose: 'true'

      - name: Upload Spec Artifact
        uses: actions/upload-artifact@v4
        with:
          name: specification
          path: .specular/spec.yaml
          retention-days: 30

  plan-generation:
    name: Generate Build Plan
    runs-on: ubuntu-latest
    needs: [ spec-generation ]

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Download Spec
        uses: actions/download-artifact@v4
        with:
          name: specification
          path: .specular/

      - name: Generate Build Plan
        uses: ./
        with:
          command: 'plan'
          spec-file: '.specular/spec.yaml'
          policy-file: '.specular/policy.yaml'
          plan-file: 'build-plan.json'
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Upload Plan Artifact
        uses: actions/upload-artifact@v4
        with:
          name: build-plan
          path: build-plan.json
          retention-days: 7

  build:
    name: Execute Build
    runs-on: ubuntu-latest
    needs: [ plan-generation ]

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Docker
        uses: docker/setup-buildx-action@v3

      - name: Download Build Plan
        uses: actions/download-artifact@v4
        with:
          name: build-plan

      - name: Execute Build with Policy Enforcement
        uses: ./
        with:
          command: 'build'
          plan-file: 'build-plan.json'
          policy-file: '.specular/policy.yaml'
          checkpoint-resume: 'true'
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
          verbose: 'true'

      - name: Upload Build Artifacts
        if: success()
        uses: actions/upload-artifact@v4
        with:
          name: build-output
          path: |
            .specular/runs/
            **/*.log
          retention-days: 7

  validate:
    name: Validate Build
    runs-on: ubuntu-latest
    needs: [ build ]

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Download Build Plan
        uses: actions/download-artifact@v4
        with:
          name: build-plan

      - name: Download Spec
        uses: actions/download-artifact@v4
        with:
          name: specification
          path: .specular/

      - name: Run Drift Detection
        uses: ./
        with:
          command: 'eval'
          spec-file: '.specular/spec.yaml'
          plan-file: 'build-plan.json'
          policy-file: '.specular/policy.yaml'
          report-file: 'validation-report.sarif'
          fail-on-drift: 'true'
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Upload SARIF Report
        if: always()
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: validation-report.sarif
          category: specular-build-validation
```

**What this workflow does:**
- 📝 Generates spec from PRD when PRD changes
- 📋 Creates build plan based on spec and policy
- 🏗️ Executes build with policy enforcement
- ✅ Validates final output against specification
- 🔄 Supports checkpoint/resume for long builds
- 📦 Preserves artifacts between jobs

### Action Inputs Reference

| Input | Description | Default | Required |
|-------|-------------|---------|----------|
| `command` | Specular command to run | - | ✅ |
| `prd-file` | Path to PRD (for spec command) | `PRD.md` | ❌ |
| `spec-file` | Path to spec file | `.specular/spec.yaml` | ❌ |
| `plan-file` | Path to plan file | `plan.json` | ❌ |
| `policy-file` | Path to policy file | `.specular/policy.yaml` | ❌ |
| `fail-on-drift` | Fail if drift detected | `true` | ❌ |
| `report-file` | SARIF report output path | `drift.sarif` | ❌ |
| `anthropic-api-key` | Anthropic API key | - | ❌ |
| `openai-api-key` | OpenAI API key | - | ❌ |
| `gemini-api-key` | Google Gemini API key | - | ❌ |
| `project-root` | Project root directory | `.` | ❌ |
| `checkpoint-resume` | Resume from checkpoint | `false` | ❌ |
| `dry-run` | Show what would be executed | `false` | ❌ |
| `verbose` | Enable verbose output | `false` | ❌ |

### Available Commands

#### `spec` - Generate Specification from PRD

```yaml
- uses: ./
  with:
    command: 'spec'
    prd-file: 'PRD.md'
    spec-file: '.specular/spec.yaml'
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

**Use when:** PRD changes and you need to update the technical specification.

#### `plan` - Generate Execution Plan

```yaml
- uses: ./
  with:
    command: 'plan'
    spec-file: '.specular/spec.yaml'
    policy-file: '.specular/policy.yaml'
    plan-file: 'build-plan.json'
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

**Use when:** Spec changes and you need a new implementation plan.

#### `build` - Execute Build Plan

```yaml
- uses: ./
  with:
    command: 'build'
    plan-file: 'build-plan.json'
    policy-file: '.specular/policy.yaml'
    checkpoint-resume: 'true'
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

**Use when:** You want to execute the implementation plan with AI assistance.

#### `eval` - Evaluate and Detect Drift

```yaml
- uses: ./
  with:
    command: 'eval'
    spec-file: '.specular/spec.yaml'
    plan-file: 'build-plan.json'
    policy-file: '.specular/policy.yaml'
    report-file: 'drift-report.sarif'
    fail-on-drift: 'true'
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

**Use when:** You want to validate that code matches the specification.

#### `drift` - Quick Drift Detection

```yaml
- uses: ./
  with:
    command: 'drift'
    spec-file: '.specular/spec.yaml'
    plan-file: 'build-plan.json'
    report-file: 'drift-report.sarif'
    fail-on-drift: 'true'
```

**Use when:** You only need drift detection without full evaluation.

### Understanding SARIF Reports

Specular generates SARIF (Static Analysis Results Interchange Format) reports that integrate with GitHub's Security tab.

**Viewing SARIF Reports:**

1. Navigate to your repository on GitHub
2. Click on the **Security** tab
3. Select **Code scanning alerts**
4. Filter by category (e.g., `specular-pr-validation`)

**SARIF Report Structure:**

```json
{
  "version": "2.1.0",
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "runs": [{
    "tool": {
      "driver": {
        "name": "Specular",
        "version": "1.0.0"
      }
    },
    "results": [
      {
        "ruleId": "SPEC_DRIFT",
        "level": "error",
        "message": {
          "text": "Feature R1 is missing implementation"
        },
        "locations": [{
          "physicalLocation": {
            "artifactLocation": {
              "uri": "src/feature.go"
            }
          }
        }]
      }
    ]
  }]
}
```

**Result Levels:**
- 🔴 `error` - Critical issues that must be fixed
- 🟡 `warning` - Important issues that should be addressed
- 🔵 `note` - Informational findings

### Best Practices for CI/CD

#### 1. Use Checkpoint/Resume for Long Builds

For builds with many tasks, enable checkpoint/resume to handle interruptions:

```yaml
- name: Execute Build
  uses: ./
  with:
    command: 'build'
    plan-file: 'build-plan.json'
    checkpoint-resume: 'true'
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

This allows builds to resume from the last successful checkpoint if GitHub Actions times out or encounters failures.

#### 2. Enable Docker Image Caching (Recommended)

Docker image caching dramatically speeds up CI/CD runs by caching pulled images between runs. This is enabled by default in the GitHub Action:

```yaml
- name: Execute Build
  uses: ./
  with:
    command: 'build'
    plan-file: 'build-plan.json'
    enable-cache: 'true'  # Default: true
    cache-dir: '.specular/cache'  # Default cache location
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

**How it works:**
- First run: Pulls Docker images (golang:1.22, node:20, etc.) and exports them to tar files
- Subsequent runs: Imports cached images (saves 2-5 minutes per run)
- Cache key: Based on plan.json and spec.yaml file hashes
- Automatic cleanup: GitHub Actions removes caches not used for 7 days

**Cache Statistics:**

Without caching:
```
⬇ Pulling image golang:1.22... (45s)
⬇ Pulling image node:20... (30s)
⬇ Pulling image alpine:latest... (5s)
Total: 80 seconds
```

With caching:
```
✓ Using cached image: golang:1.22 (age: 2h)
✓ Using cached image: node:20 (age: 2h)
✓ Using cached image: alpine:latest (age: 2h)
Total: 5 seconds
```

**Disable caching:**

```yaml
- uses: ./
  with:
    command: 'build'
    enable-cache: 'false'  # Disable caching
```

**Pre-warm cache:**

For faster first runs, pre-warm the Docker cache:

```yaml
- name: Pre-warm Docker Cache
  run: specular prewarm --all --verbose

- uses: ./
  with:
    command: 'build'
    enable-cache: 'true'
```

#### 3. Cache Specular Binary

Speed up workflow runs by caching the Specular binary:

```yaml
- name: Cache Specular
  uses: actions/cache@v3
  with:
    path: ~/.specular/bin
    key: specular-${{ runner.os }}-${{ hashFiles('**/go.mod') }}
    restore-keys: |
      specular-${{ runner.os }}-
```

#### 4. Run Drift Detection on Every PR

Catch specification drift early by running `eval` on all pull requests:

```yaml
on:
  pull_request:
    branches: [ main, develop ]

jobs:
  drift-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./
        with:
          command: 'eval'
          fail-on-drift: 'true'
```

#### 5. Use Artifacts for Multi-Job Workflows

Pass data between jobs using GitHub artifacts:

```yaml
# Job 1: Generate spec
- name: Upload Spec
  uses: actions/upload-artifact@v4
  with:
    name: specification
    path: .specular/spec.yaml

# Job 2: Use spec
- name: Download Spec
  uses: actions/download-artifact@v4
  with:
    name: specification
    path: .specular/
```

#### 6. Fail Fast on Policy Violations

Configure strict policy enforcement to maintain code quality:

```yaml
- uses: ./
  with:
    command: 'build'
    policy-file: '.specular/policy.yaml'
    fail-on-drift: 'true'  # Fail immediately on violations
```

#### 7. Separate Workflows for Different Triggers

Create focused workflows for different events:

- **`pr-validation.yml`** - Quick validation on PRs
- **`continuous-build.yml`** - Full build on merge to main
- **`nightly-validation.yml`** - Comprehensive checks on schedule

Example nightly validation:

```yaml
name: Nightly Validation

on:
  schedule:
    - cron: '0 2 * * *'  # Run at 2 AM daily

jobs:
  full-validation:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./
        with:
          command: 'eval'
          spec-file: '.specular/spec.yaml'
          plan-file: 'plan.json'
          policy-file: '.specular/policy.yaml'
          verbose: 'true'
```

#### 8. Protect Sensitive Data

Always use GitHub Secrets for API keys and never commit them to the repository:

```yaml
# ✅ Correct
anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}

# ❌ Wrong - Never do this
anthropic-api-key: 'sk-ant-your-key-here'
```

### Troubleshooting CI/CD Issues

#### Workflow Fails with "Docker not found"

The action requires Docker. Ensure Docker setup:

```yaml
- name: Setup Docker
  uses: docker/setup-buildx-action@v3
```

#### API Rate Limits

If you hit AI provider rate limits, use checkpoint/resume:

```yaml
- uses: ./
  with:
    command: 'build'
    checkpoint-resume: 'true'  # Resume after rate limit resets
```

#### Workflow Times Out

GitHub Actions has a 6-hour timeout. For very large builds:

1. Enable checkpoint/resume
2. Split the plan into smaller chunks
3. Use multiple jobs with artifacts

```yaml
jobs:
  build-phase-1:
    steps:
      - uses: ./
        with:
          plan-file: 'plan-phase-1.json'

  build-phase-2:
    needs: [ build-phase-1 ]
    steps:
      - uses: ./
        with:
          plan-file: 'plan-phase-2.json'
```

#### SARIF Upload Fails

Ensure proper permissions in your workflow:

```yaml
permissions:
  contents: read
  security-events: write  # Required for SARIF upload
```

### Integration with Existing CI/CD

Specular can integrate with existing CI/CD workflows:

#### Example: Integration with Existing Test Suite

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # Run existing tests
      - name: Run Unit Tests
        run: npm test

      # Add Specular validation
      - name: Validate Against Spec
        uses: ./
        with:
          command: 'eval'
          spec-file: '.specular/spec.yaml'
          fail-on-drift: 'true'
```

#### Example: Integration with Deployment Pipeline

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      # Validate before deployment
      - name: Pre-deployment Validation
        uses: ./
        with:
          command: 'eval'
          fail-on-drift: 'true'

      # Deploy only if validation passes
      - name: Deploy to Production
        if: success()
        run: ./deploy.sh
```

## Best Practices

### 1. Version Control Your Specs

```bash
git add .specular/spec.yaml .specular/policy.yaml
git commit -m "feat: add task management spec"
```

### 2. Incremental Development

Start with P0 requirements, then expand:

```bash
# Iteration 1: Critical features only
specular plan generate --filter "priority=P0" --out plan-mvp.json

# Iteration 2: Add P1 features
specular plan generate --filter "priority in [P0,P1]" --out plan-v1.json
```

### 3. Test-First Approach

Configure policy to require tests before implementation:

```yaml
policies:
  - id: test-first
    description: "Tests must exist before implementation"
    rule: "test_exists_before_code"
    severity: error
```

### 4. Regular Drift Checks

Add to your git hooks:

```bash
# .git/hooks/pre-commit
#!/bin/bash
specular drift detect --spec .specular/spec.yaml --codebase ./src --threshold 0.1
```

### 5. Document Decisions

Capture architectural decisions in your PRD:

```markdown
## Architecture Decisions

### AD1: Use SQLite for Storage
**Rationale:** Simplicity for MVP, no external dependencies
**Trade-offs:** May need migration to PostgreSQL for production
```

## Troubleshooting

### Provider Connection Issues

```bash
# Check provider configuration
specular provider list

# Test specific provider
specular provider health anthropic

# Update configuration
specular provider init --force
```

### Docker Execution Failures

```bash
# Verify Docker is running
docker ps

# Check Docker permissions
docker run hello-world

# Increase resource limits (Docker Desktop)
# Settings → Resources → increase memory to 4GB
```

### Policy Violations

```bash
# View detailed policy violations
specular policy check \
  --policy .specular/policy.yaml \
  --codebase ./src \
  --verbose

# Generate fix suggestions
specular policy fix \
  --violations violations.json \
  --out fixes.json
```

### Spec-Code Drift

```bash
# Detailed drift analysis
specular drift detect \
  --spec .specular/spec.yaml \
  --codebase ./src \
  --verbose \
  --out drift-detail.json

# Sync spec to match code
specular spec sync --codebase ./src --spec .specular/spec.yaml

# Or sync code to match spec
specular build --plan plan.json --sync-mode strict
```

### Checkpoint Issues

#### Checkpoint Not Found

```bash
# List available checkpoints
ls -la .specular/checkpoints/

# If checkpoint directory doesn't exist
mkdir -p .specular/checkpoints

# If no checkpoint exists for your plan, you'll see:
# No checkpoint found for: build-plan.json-1234567890
# Starting fresh execution...
```

#### Corrupted Checkpoint

```bash
# If checkpoint is corrupted, delete and start fresh
rm .specular/checkpoints/build-plan.json-*.json

# Start new execution
specular build --plan plan.json --policy .specular/policy.yaml
```

#### Multiple Checkpoints

```bash
# List all checkpoints
ls -la .specular/checkpoints/

# Resume from specific checkpoint by ID
specular build \
  --plan plan.json \
  --checkpoint-id build-plan.json-1234567890 \
  --resume

# Clean up old checkpoints
find .specular/checkpoints -name "build-*.json" -mtime +7 -delete
```

#### Failed Tasks on Resume

```bash
# When resuming, failed tasks are retried automatically
# View checkpoint to see failure details
cat .specular/checkpoints/build-plan.json-*.json | jq '.tasks[] | select(.status=="failed")'

# Example output:
# {
#   "id": "task5",
#   "status": "failed",
#   "error": "API rate limit exceeded",
#   "attempts": 2
# }

# Resume with increased timeout or adjusted policy
specular build \
  --plan plan.json \
  --policy .specular/policy.yaml \
  --resume
```

## Next Steps

- 📖 Read the [PRD Guide](prd.md) for writing effective requirements
- 🔧 Explore [Provider Configuration](provider-guide.md) for multi-provider setup
- 🏗️ Review [Technical Design](tech_design.md) for architecture details
- 🚀 Check [MVP Action Plan](mvp-action-plan.md) for roadmap
- 🎯 See [Examples](../examples) for real-world use cases

## Getting Help

- 📚 Documentation: [https://github.com/felixgeelhaar/specular](https://github.com/felixgeelhaar/specular)
- 🐛 Issues: [https://github.com/felixgeelhaar/specular/issues](https://github.com/felixgeelhaar/specular/issues)
- 💬 Discussions: [https://github.com/felixgeelhaar/specular/discussions](https://github.com/felixgeelhaar/specular/discussions)

## Shell Completion

Enable shell completion for better CLI experience:

**Bash:**
```bash
source <(specular completion bash)
```

**Zsh:**
```bash
specular completion zsh > "${fpath[1]}/_specular"
```

**Fish:**
```bash
specular completion fish | source
```

---

Ready to build spec-first with AI? Start with the Quick Start Tutorial above! 🚀
