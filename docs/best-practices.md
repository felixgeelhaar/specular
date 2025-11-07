# Specular Best Practices and Patterns Guide

This guide provides recommended patterns, workflows, and practices for effective use of Specular in production environments.

## Table of Contents

1. [Specification Management](#specification-management)
2. [Policy Configuration](#policy-configuration)
3. [Routing Optimization](#routing-optimization)
4. [Drift Detection Workflows](#drift-detection-workflows)
5. [Team Collaboration](#team-collaboration)
6. [CI/CD Integration](#cicd-integration)
7. [Security & Compliance](#security--compliance)
8. [Common Pitfalls](#common-pitfalls)
9. [Performance Optimization](#performance-optimization)
10. [Troubleshooting](#troubleshooting)

---

## Specification Management

### Creating High-Quality Specs

**Use the Interactive Interview**

The `specular interview --tui` command provides the best user experience for creating specifications:

```bash
# Start interactive interview with TUI
specular interview --tui --out .specular/spec.yaml

# Use presets for common project types
specular interview --tui --preset web-app --out .specular/spec.yaml
```

**Benefits:**
- ✅ Guided questions ensure completeness
- ✅ Real-time validation catches errors immediately
- ✅ Progress tracking shows how much is left
- ✅ Structured output ensures schema compliance

**Smart Context Detection**

Let Specular detect your environment automatically:

```bash
# Automatic detection (recommended)
specular init --template web-app

# Skip detection if you prefer manual configuration
specular init --template web-app --no-detect

# Accept all defaults non-interactively
specular init --template web-app --yes
```

**Spec Organization Best Practices**

1. **Feature Granularity**: Keep features focused and independent
   - ✅ Good: "User Authentication", "Product Search", "Checkout Flow"
   - ❌ Bad: "Entire Application", "Backend", "Frontend"

2. **Priority Assignment**: Use priorities strategically
   - P0: Critical path features (must have for MVP)
   - P1: Important features (should have soon)
   - P2: Nice-to-have features (can defer)

3. **Success Criteria**: Make criteria measurable and testable
   ```yaml
   success_criteria:
     - "API returns 200 OK for valid requests"
     - "Response time < 200ms for 95th percentile"
     - "Test coverage ≥ 80% for authentication module"
   ```

### Version Control for Specs

**Always Commit spec.lock.json**

```bash
# After creating/updating spec
git add .specular/spec.yaml .specular/spec.lock.json
git commit -m "feat: add user authentication specification"
```

**Why?**
- Lock file contains immutable hashes for drift detection
- Enables reproducible builds across environments
- Provides audit trail for specification changes

**Spec Update Workflow**

```bash
# 1. Update spec.yaml
vim .specular/spec.yaml

# 2. Regenerate lock file
specular spec lock --in .specular/spec.yaml --out .specular/spec.lock.json

# 3. Review changes
git diff .specular/spec.lock.json

# 4. Commit both files
git add .specular/spec.yaml .specular/spec.lock.json
git commit -m "feat: add payment processing feature"
```

---

## Policy Configuration

### Multi-Environment Policy Strategy

**Use Different Policies per Environment**

```plaintext
.specular/
├── policy.yaml              # Default (development)
├── policy.staging.yaml      # Staging environment
└── policy.production.yaml   # Production (strictest)
```

**Development Policy** (`.specular/policy.yaml`):
```yaml
execution:
  allow_local: true  # Allow local execution for speed
  docker:
    required: false
    network: "bridge"  # Allow network access

tests:
  require_pass: true
  min_coverage: 0.50  # Lower for rapid iteration

security:
  secrets_scan: true
  dep_scan: false  # Skip to save time
```

**Production Policy** (`.specular/policy.production.yaml`):
```yaml
execution:
  allow_local: false  # Docker only
  docker:
    required: true
    image_allowlist:
      - "ghcr.io/myorg/builder:*"
    network: "none"  # No network access
    cpu_limit: "2"
    mem_limit: "2g"

tests:
  require_pass: true
  min_coverage: 0.80  # Strict coverage

security:
  secrets_scan: true
  dep_scan: true  # Full security scanning

routing:
  allow_models:
    - provider: anthropic
      names: ["claude-3.5-sonnet"]  # Specific approved models
  deny_tools: ["shell_local", "file_delete"]
```

### Docker Image Management

**Create Organization-Specific Builder Images**

```dockerfile
# Dockerfile.builder
FROM golang:1.22-alpine

# Install required tools
RUN apk add --no-cache \
    git \
    make \
    golangci-lint

# Set up non-root user
RUN addgroup -g 1000 builder && \
    adduser -D -u 1000 -G builder builder

USER builder
WORKDIR /workspace

# Pre-download common dependencies
COPY go.mod go.sum ./
RUN go mod download
```

**Build and Push**:
```bash
docker build -t ghcr.io/myorg/go-builder:1.22 -f Dockerfile.builder .
docker push ghcr.io/myorg/go-builder:1.22
```

**Use in Policy**:
```yaml
execution:
  docker:
    image_allowlist:
      - "ghcr.io/myorg/go-builder:1.22"
      - "ghcr.io/myorg/node-builder:22"
```

### Governance Levels

**Choose Appropriate Governance**

```bash
# L2: Basic (development teams)
specular init --governance L2

# L3: Standard (most organizations)
specular init --governance L3

# L4: Strict (regulated industries)
specular init --governance L4
```

**Governance Level Comparison:**

| Feature | L2 | L3 | L4 |
|---------|----|----|-----|
| Docker Required | ❌ | ✅ | ✅ |
| Image Allowlist | ❌ | ✅ | ✅ |
| Network Isolation | ❌ | ✅ | ✅ |
| Secrets Scanning | ✅ | ✅ | ✅ |
| Dep Scanning | ❌ | ✅ | ✅ |
| Min Coverage | 50% | 70% | 80% |
| SARIF Output | ✅ | ✅ | ✅ |
| Run Manifests | ❌ | ✅ | ✅ |

---

## Routing Optimization

### Understanding Provider Strategies

**Local-First for Development**

```bash
# Uses Ollama if available, fast iteration
specular init --local --providers ollama
```

**Cloud for Production**

```bash
# Uses cloud providers for quality and reliability
specular init --cloud --providers anthropic,openai
```

**Hybrid for Best of Both**

```bash
# Ollama for cheap tasks, Claude for complex ones
specular init --providers ollama,anthropic
```

### Router Configuration

**Configure router.yaml for Optimal Performance**

```yaml
# .specular/router.yaml
budget:
  max_cost_usd: 50.0
  warn_threshold_usd: 40.0

latency:
  max_ms: 60000
  prefer_fast: true

cost:
  prefer_cheap: true  # Use cheaper models when possible

quality:
  min_confidence: 0.7
  upgrade_on_failure: true  # Retry with better model if failed

fallback:
  enabled: true
  chain:
    - anthropic:claude-3.5-sonnet
    - openai:gpt-4
    - google:gemini-2.5-pro
```

### Route Optimization Commands

**Analyze Routing Performance**

```bash
# Get routing optimization recommendations
specular route optimize --period 30d --format text

# View as JSON for automation
specular route optimize --period 30d --format json > optimization.json
```

**Benchmark Models**

```bash
# Compare model performance
specular route bench --models anthropic:claude-3.5-sonnet,openai:gpt-4

# Quick benchmark (fewer iterations)
specular route bench --quick
```

**Test Routing Decisions**

```bash
# Test what model would be selected
specular route test --task "Generate unit tests for auth module" \
  --complexity 7 --priority P0

# Explain routing decision
specular route explain --task "Refactor legacy code" --verbose
```

---

## Drift Detection Workflows

### Pre-Commit Drift Prevention

**Git Pre-Commit Hook**

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash
set -e

echo "Running Specular drift detection..."

# Check if spec.lock.json exists
if [ -f ".specular/spec.lock.json" ]; then
    specular eval --spec .specular/spec.lock.json --policy .specular/policy.yaml

    # Check exit code
    if [ $? -ne 0 ]; then
        echo "❌ Drift detected! Please fix before committing."
        echo "Run 'specular eval' for details."
        exit 1
    fi

    echo "✅ No drift detected"
fi

exit 0
```

Make it executable:
```bash
chmod +x .git/hooks/pre-commit
```

### Pull Request Drift Checks

**Require Drift Checks in PR Workflow**

```yaml
# .github/workflows/pr-drift-check.yml
name: PR Drift Detection

on:
  pull_request:
    branches: [main, develop]

jobs:
  drift-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Drift Detection
        uses: ./.github/actions/specular
        with:
          command: eval
          fail-on-drift: true

      - name: Comment PR
        if: always()
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            if (fs.existsSync('.specular/drift.sarif')) {
              // Post drift results as comment
              // (implementation in github-actions-basic.yml)
            }
```

### Continuous Drift Monitoring

**Scheduled Drift Checks**

```yaml
# Run nightly drift detection
on:
  schedule:
    - cron: '0 2 * * *'  # 2 AM daily

jobs:
  nightly-drift:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Full Drift Scan
        uses: ./.github/actions/specular
        with:
          command: eval
          policy-file: .specular/policy.production.yaml

      - name: Alert on Drift
        if: failure()
        uses: actions/slack-notify@v1
        with:
          channel: '#alerts'
          message: '🚨 Drift detected in production spec'
```

### SARIF Integration

**View Drift in GitHub Security Tab**

The GitHub Action automatically uploads SARIF reports:

```yaml
- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: .specular/drift.sarif
    category: specular-drift
```

Navigate to: `Repository → Security → Code scanning alerts`

---

## Team Collaboration

### Specification Review Process

**1. Spec Author Creates Feature**

```bash
# Create feature spec
specular interview --tui

# Generate plan to verify feasibility
specular plan --spec .specular/spec.lock.json --out plan.json

# Review plan estimates
cat plan.json | jq '.tasks[] | {id, title, estimate_hours}'
```

**2. Submit for Review**

```bash
git checkout -b feature/add-payment-processing
git add .specular/spec.yaml .specular/spec.lock.json plan.json
git commit -m "feat: add payment processing specification"
git push origin feature/add-payment-processing

# Create PR
gh pr create --title "Add Payment Processing Spec" \
  --body "$(cat <<EOF
## Specification Changes

- Added payment processing feature (P0)
- Estimated: 40 hours
- Dependencies: user authentication, product catalog

## Plan Preview

\`\`\`json
$(cat plan.json | jq '.summary')
\`\`\`
EOF
)"
```

**3. Reviewers Validate**

```bash
# Checkout PR
gh pr checkout 123

# Validate spec
specular spec validate --in .specular/spec.yaml

# Review plan
specular plan --spec .specular/spec.lock.json --out plan-review.json
cat plan-review.json | jq '.tasks[] | {title, dependencies, estimate_hours}'

# Check for drift
specular eval --spec .specular/spec.lock.json
```

### Shared Router Configuration

**Team Router Settings**

Create shared `.specular/router.yaml` in repository:

```yaml
# Team defaults
budget:
  max_cost_usd: 100.0  # Per developer per month

latency:
  max_ms: 60000

cost:
  prefer_cheap: true

# Fallback chain (in preference order)
fallback:
  enabled: true
  chain:
    - anthropic:claude-3.5-sonnet  # Primary
    - openai:gpt-4                 # Fallback
    - google:gemini-2.5-pro        # Last resort
```

**Individual Overrides**

Developers can override locally (not committed):

```yaml
# ~/.specular/config.yaml
providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}

routing:
  budget_usd: 50  # Personal limit
  prefer_local: true  # Use Ollama when possible
```

### Documentation Standards

**Maintain Specification Changelog**

Create `.specular/CHANGELOG.md`:

```markdown
# Specification Changelog

## [Unreleased]

### Added
- Payment processing feature with Stripe integration
- Email notification system

### Changed
- User authentication now supports OAuth2
- Increased test coverage requirement to 80%

### Deprecated
- Legacy session-based auth (will remove in v2.0)

## [1.0.0] - 2024-01-15

### Added
- Initial product specification
- User authentication
- Product catalog
- Shopping cart
```

---

## CI/CD Integration

### Platform-Specific Best Practices

**GitHub Actions**

```yaml
# Use matrix for multi-environment testing
strategy:
  matrix:
    policy: [development, staging, production]

steps:
  - name: Test with ${{ matrix.policy }} policy
    uses: ./.github/actions/specular
    with:
      command: eval
      policy-file: .specular/policy.${{ matrix.policy }}.yaml
```

**GitLab CI**

```yaml
# Use artifacts between stages
validate_spec:
  artifacts:
    paths:
      - .specular/spec.lock.json
    expire_in: 1 hour

generate_plan:
  dependencies:
    - validate_spec  # Download artifacts
```

**CircleCI**

```yaml
# Use workspaces for data sharing
- persist_to_workspace:
    root: .
    paths:
      - .specular/spec.lock.json
      - plan.json

# Later job
- attach_workspace:
    at: .
```

**Jenkins**

```groovy
// Use stash/unstash
stash includes: "${LOCK_FILE}", name: 'spec-lock'

// Later stage
unstash 'spec-lock'
```

### Caching Strategies

**Cache Specular Binary**

```yaml
# GitHub Actions
- name: Cache Specular
  uses: actions/cache@v4
  with:
    path: /usr/local/bin/specular
    key: specular-${{ env.SPECULAR_VERSION }}-${{ runner.os }}-${{ runner.arch }}
```

**Cache Docker Images**

```yaml
# Use Docker layer caching
- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Build with cache
  uses: docker/build-push-action@v5
  with:
    cache-from: type=gha
    cache-to: type=gha,mode=max
```

---

## Security & Compliance

### API Key Management

**Never Commit API Keys**

```bash
# Use environment variables
export ANTHROPIC_API_KEY="your-key-here"
export OPENAI_API_KEY="your-key-here"

# Or use direnv (.envrc)
export ANTHROPIC_API_KEY=$(cat ~/.secrets/anthropic-key)
```

**CI/CD Secret Management**

```yaml
# GitHub Actions - use secrets
- name: Run with API keys
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  uses: ./.github/actions/specular
  with:
    command: plan
```

### Audit Trail

**Enable Run Manifests**

```yaml
# policy.yaml
execution:
  emit_manifests: true
  manifest_dir: ".specular/runs"
```

**Review Manifests**

```bash
# List all runs
ls -lh .specular/runs/

# View specific run
cat .specular/runs/2024-01-15T10-30-45.json | jq '{
  timestamp,
  command,
  exit_code,
  duration_ms,
  cost_usd
}'
```

### Secrets Scanning

**Configure Secrets Detection**

```yaml
# policy.yaml
security:
  secrets_scan: true
  secrets_patterns:
    - "AKIAIOSFODNN7EXAMPLE"  # AWS
    - "sk-[a-zA-Z0-9]{48}"    # OpenAI
    - "xoxb-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{24}"  # Slack
```

**Pre-Commit Scanning**

```bash
# Add to .git/hooks/pre-commit
specular eval --scan-secrets-only
```

---

## Common Pitfalls

### Pitfall #1: Skipping spec.lock.json

**❌ Problem:**
```bash
# Only committing spec.yaml
git add .specular/spec.yaml
git commit -m "Update spec"
```

**✅ Solution:**
```bash
# Always regenerate and commit lock file
specular spec lock --in .specular/spec.yaml --out .specular/spec.lock.json
git add .specular/spec.yaml .specular/spec.lock.json
git commit -m "feat: update user authentication spec"
```

### Pitfall #2: Overly Broad Features

**❌ Problem:**
```yaml
features:
  - id: backend
    title: "Entire Backend System"
    description: "Build the whole backend"
```

**✅ Solution:**
```yaml
features:
  - id: auth
    title: "User Authentication"
    description: "JWT-based authentication with refresh tokens"

  - id: user-mgmt
    title: "User Management"
    description: "CRUD operations for user profiles"

  - id: api-gateway
    title: "API Gateway"
    description: "Request routing and rate limiting"
```

### Pitfall #3: Ignoring Drift Warnings

**❌ Problem:**
```bash
# Drift detected but ignored
specular eval  # Shows drift
# ... continues anyway
git commit -m "Ship it!"
```

**✅ Solution:**
```bash
# Fix drift before committing
specular eval

# If drift found, investigate
cat .specular/drift.sarif | jq '.runs[0].results'

# Fix issues, then verify
specular eval  # Should show no drift
git commit -m "fix: resolve spec drift in authentication"
```

### Pitfall #4: Weak Success Criteria

**❌ Problem:**
```yaml
success_criteria:
  - "Feature works"
  - "Users are happy"
  - "No bugs"
```

**✅ Solution:**
```yaml
success_criteria:
  - "API endpoint responds with 200 OK for valid login"
  - "Invalid credentials return 401 Unauthorized within 100ms"
  - "JWT tokens expire after 15 minutes of inactivity"
  - "Test coverage ≥ 85% for authentication module"
  - "Zero SQL injection vulnerabilities detected by semgrep"
```

### Pitfall #5: Not Using Templates

**❌ Problem:**
```bash
# Manually creating everything from scratch
mkdir .specular
touch .specular/spec.yaml
touch .specular/policy.yaml
touch .specular/router.yaml
# ... lots of manual configuration
```

**✅ Solution:**
```bash
# Use init with templates
specular init --template web-app --governance L3

# Or use interview for guided setup
specular interview --tui --preset web-app
```

---

## Performance Optimization

### Reduce AI Costs

**1. Use Prefer-Cheap Routing**

```yaml
# router.yaml
cost:
  prefer_cheap: true
  max_cost_per_task: 0.50
```

**2. Leverage Local Models**

```bash
# Install Ollama
ollama pull codellama

# Use local-first strategy
specular init --local --providers ollama,anthropic
```

**3. Set Budget Limits**

```yaml
# router.yaml
budget:
  max_cost_usd: 20.0
  warn_threshold_usd: 15.0
  alert_on_exceed: true
```

**4. Monitor Costs**

```bash
# View cost breakdown
specular route optimize --period 30d --format json | \
  jq '.total_cost, .potential_savings'
```

### Reduce Latency

**1. Prefer Fast Models**

```yaml
# router.yaml
latency:
  max_ms: 30000  # 30 seconds max
  prefer_fast: true
```

**2. Use Timeouts**

```yaml
# router.yaml
timeouts:
  planning: 60000   # 60 seconds
  generation: 120000  # 2 minutes
  evaluation: 30000  # 30 seconds
```

**3. Enable Caching** (Future Feature)

```yaml
# router.yaml
cache:
  enabled: true
  ttl_minutes: 60
  max_size_mb: 500
```

### Optimize CI/CD Pipeline

**Parallel Job Execution**

```yaml
# GitHub Actions
jobs:
  validate:
    # ...

  plan:
    needs: validate
    # ...

  # These can run in parallel
  drift-check:
    needs: validate
    # ...

  security-scan:
    needs: validate
    # ...
```

**Skip Redundant Checks**

```yaml
# Only run on relevant file changes
on:
  pull_request:
    paths:
      - '.specular/**'
      - 'src/**'
      - 'tests/**'
```

---

## Troubleshooting

### Common Issues and Solutions

#### Issue: "spec.lock.json not found"

**Cause:** Lock file not generated or committed

**Solution:**
```bash
specular spec lock --in .specular/spec.yaml --out .specular/spec.lock.json
git add .specular/spec.lock.json
git commit -m "chore: add spec lock file"
```

#### Issue: "Policy violation: image not allowed"

**Cause:** Docker image not in allowlist

**Solution:**
```yaml
# Add to policy.yaml
execution:
  docker:
    image_allowlist:
      - "your-image:tag"
```

#### Issue: "Drift detected but no details shown"

**Cause:** SARIF file not generated or malformed

**Solution:**
```bash
# Regenerate SARIF
specular eval --spec .specular/spec.lock.json --policy .specular/policy.yaml

# View SARIF content
cat .specular/drift.sarif | jq '.runs[0].results'
```

#### Issue: "Router returning errors"

**Cause:** No providers configured or API keys missing

**Solution:**
```bash
# Run diagnostics
specular doctor

# Check provider health
specular route show --verbose

# Test specific provider
specular route test --task "test" --providers anthropic
```

#### Issue: "High AI costs"

**Cause:** Using expensive models for simple tasks

**Solution:**
```yaml
# Enable cost optimization
routing:
  cost:
    prefer_cheap: true
    max_cost_per_task: 0.50

# Review routing decisions
specular route optimize --period 30d
```

### Debugging Tips

**Enable Verbose Output**

```bash
# Most commands support --verbose
specular plan --verbose
specular build --verbose
specular route test --verbose
```

**Check System Health**

```bash
# Run full diagnostics
specular doctor --format json | jq '.'

# Check specific components
specular doctor --check docker
specular doctor --check providers
specular doctor --check config
```

**View Run Manifests**

```bash
# Find latest run
ls -lt .specular/runs/ | head -1

# View details
cat .specular/runs/latest.json | jq '{
  command,
  exit_code,
  duration_ms,
  error_message
}'
```

---

## Appendix: Quick Reference

### Command Cheat Sheet

```bash
# Initialization
specular init --template <type> --governance <L2|L3|L4>
specular interview --tui

# Specification
specular spec generate --in PRD.md
specular spec lock --in .specular/spec.yaml
specular spec validate --in .specular/spec.yaml

# Planning
specular plan --spec .specular/spec.lock.json --out plan.json

# Execution
specular build --plan plan.json --policy .specular/policy.yaml

# Evaluation
specular eval --spec .specular/spec.lock.json --policy .specular/policy.yaml

# Routing
specular route show
specular route test --task "description"
specular route explain --task "description"
specular route optimize --period 30d
specular route bench --models model1,model2

# Diagnostics
specular doctor
specular version
```

### File Structure Reference

```plaintext
project/
├── .specular/
│   ├── spec.yaml              # Human-readable spec
│   ├── spec.lock.json         # Immutable hashed spec
│   ├── policy.yaml            # Governance rules
│   ├── router.yaml            # Routing configuration
│   ├── drift.sarif            # Drift detection results
│   ├── runs/                  # Execution manifests
│   │   └── 2024-01-15-*.json
│   ├── checkpoints/           # Resume points
│   └── cache/                 # Cached data
├── plan.json                  # Generated execution plan
└── .github/
    └── workflows/
        └── specular.yml       # CI/CD integration
```

### Exit Codes

| Code | Meaning | Action |
|------|---------|--------|
| 0 | Success | Continue |
| 1 | General error | Check error message |
| 2 | Usage error | Review command syntax |
| 3 | Policy violation | Update policy or code |
| 4 | Drift detected | Fix drift before continuing |
| 5 | Authentication error | Check API keys |
| 6 | Network error | Check connectivity |

---

## Further Reading

- [Installation Guide](installation.md)
- [Architecture Decision Records](../adrs/)
- [Example Projects](../examples/)
- [Contributing Guide](../CONTRIBUTING.md)
- [API Documentation](api-reference.md)

---

**Last Updated:** 2024-01-15
**Version:** 1.2.0
