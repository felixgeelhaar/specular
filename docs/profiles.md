# Autonomous Mode Profiles

Profiles enable environment-specific configurations for autonomous mode, allowing different behavior in interactive development, CI/CD pipelines, and custom scenarios.

## Quick Start

### Using Built-in Profiles

Specular comes with three built-in profiles:

```bash
# Use default profile (interactive development)
specular auto "implement user authentication"

# Use ci profile (non-interactive CI/CD)
specular auto --profile ci "implement user authentication"

# Use strict profile (maximum safety)
specular auto --profile strict "implement user authentication"
```

### List Available Profiles

```bash
specular auto --list-profiles
```

### Override Profile Settings

CLI flags override profile settings:

```bash
# Override max_steps
specular auto --profile ci --max-steps 10 "implement feature"

# Override multiple settings
specular auto --profile default \
  --max-steps 15 \
  --max-cost 10.0 \
  --timeout 30m \
  "implement feature"
```

---

## Built-in Profiles

### Default Profile

**Use case**: Interactive development with balanced safety and flexibility

```yaml
approvals:
  mode: critical_only        # Approve only critical steps
  interactive: true          # Interactive prompts enabled

safety:
  max_steps: 12              # Reasonable step limit
  timeout: 25m               # Ample time for complex tasks
  max_cost_usd: 5.0          # Moderate cost limit
  max_cost_per_task: 0.50

execution:
  trace_logging: true        # Enable debugging
  enable_tui: true           # Interactive UI
  save_patches: false
  json_output: false
```

**When to use**:
- Local development
- Experimenting with features
- Learning autonomous mode

### CI Profile

**Use case**: Non-interactive CI/CD pipelines

```yaml
approvals:
  mode: none                 # No approvals (fully automated)
  interactive: false         # No interactive prompts

safety:
  max_steps: 8               # Stricter limit for CI
  timeout: 15m               # Faster timeout for CI
  max_cost_usd: 2.0          # Lower cost limit
  max_cost_per_task: 0.25

execution:
  trace_logging: true        # Enable for debugging
  enable_tui: false          # No UI in CI
  save_patches: true         # Enable rollback
  json_output: true          # Machine-readable output
```

**When to use**:
- GitHub Actions, GitLab CI, Jenkins
- Automated deployments
- Scheduled tasks
- Integration tests

### Strict Profile

**Use case**: Maximum safety with all approvals required

```yaml
approvals:
  mode: all                  # Approve every step
  interactive: true

safety:
  max_steps: 5               # Very strict limit
  timeout: 10m               # Short timeout
  max_cost_usd: 1.0          # Low cost limit
  blocked_step_types:        # Block dangerous steps
    - "build:run"

execution:
  trace_logging: true
  enable_tui: true
  save_patches: true         # Always enable rollback
```

**When to use**:
- Production deployments
- Critical infrastructure changes
- Learning and experimentation
- Audited environments

---

## Custom Profiles

### Creating Custom Profiles

Create a profile file in one of these locations:

1. **Project-level** (highest precedence): `./auto.profiles.yaml`
2. **User-level**: `~/.specular/auto.profiles.yaml`

Example custom profile:

```yaml
schema: "specular.auto.profiles/v1"

profiles:
  my-profile:
    description: "My custom profile"

    approvals:
      mode: "critical_only"
      interactive: true
      auto_approve:
        - "spec:update"
        - "plan:gen"
      require_approval:
        - "spec:lock"
        - "build:run"

    safety:
      max_steps: 15
      timeout: "30m"
      max_cost_usd: 10.0
      max_cost_per_task: 1.0
      max_retries: 3
      require_policy: true

    routing:
      preferred_agent: "cline"
      fallback_agent: "openai"
      temperature: 0.7

    policies:
      enabled: true
      enforcement: "strict"

    execution:
      trace_logging: true
      save_patches: false
      checkpoint_frequency: 1
      json_output: false
      enable_tui: true
```

### Using Custom Profiles

```bash
specular auto --profile my-profile "implement feature"
```

---

## Profile Resolution

Profiles are resolved with the following precedence (highest to lowest):

1. **CLI flags** (always override)
2. **Project-level profiles** (`./auto.profiles.yaml`)
3. **User-level profiles** (`~/.specular/auto.profiles.yaml`)
4. **Built-in profiles** (embedded in binary)

### Example Resolution

Given:
- Built-in `default` profile: `max_steps: 12`
- User-level override: `max_steps: 15`
- CLI flag: `--max-steps 20`

Result: `max_steps: 20` (CLI flag wins)

---

## Profile Configuration Reference

### Approvals

Controls approval gates and interactive behavior:

```yaml
approvals:
  # Approval mode: "all", "critical_only", "none"
  mode: "critical_only"

  # Enable interactive approval prompts
  interactive: true

  # Step types that don't require approval
  auto_approve:
    - "spec:update"
    - "plan:gen"

  # Step types that always require approval
  require_approval:
    - "spec:lock"
    - "build:run"
```

**Approval Modes**:
- `all`: Approve every step
- `critical_only`: Approve only critical steps (spec:lock, build:run)
- `none`: Auto-approve all steps (non-interactive)

**Step Types**:
- `spec:update`: Update product specification
- `spec:lock`: Lock specification with cryptographic hash
- `plan:gen`: Generate execution plan
- `build:run`: Run build/execution steps

### Safety

Defines execution limits and constraints:

```yaml
safety:
  # Maximum number of workflow steps (1-100)
  max_steps: 12

  # Maximum execution time (e.g., "25m", "1h30m")
  timeout: "25m"

  # Maximum total cost in USD
  max_cost_usd: 5.0

  # Maximum cost per task in USD
  max_cost_per_task: 0.50

  # Maximum retries per task (0-10)
  max_retries: 3

  # Require policy checks before execution
  require_policy: true

  # Whitelist of allowed step types (empty = allow all)
  allowed_step_types:
    - "spec:update"
    - "spec:lock"
    - "plan:gen"
    - "build:run"

  # Blacklist of blocked step types
  blocked_step_types: []
```

### Routing

Configures agent selection and model preferences:

```yaml
routing:
  # Primary agent for execution
  preferred_agent: "cline"

  # Fallback if preferred unavailable
  fallback_agent: "openai"

  # LLM temperature (0.0-1.0)
  # Lower = more consistent, Higher = more creative
  temperature: 0.7

  # Model preferences by step type
  model_preferences:
    "spec:update": "claude-sonnet-3.5"
    "plan:gen": "claude-sonnet-3.5"
    "build:run": "claude-opus-3.5"
```

### Policies

Configures policy checks and enforcement:

```yaml
policies:
  # Enable policy checks
  enabled: true

  # Policy files to load (Rego format)
  policy_files:
    - "~/.specular/policies/default.rego"
    - "./policies/project.rego"

  # Enforcement level: "strict", "warn", "none"
  # - strict: Abort on policy violations
  # - warn: Warn but continue
  # - none: Disable enforcement
  enforcement: "strict"
```

### Execution

Configures execution behavior:

```yaml
execution:
  # Enable comprehensive trace logging
  trace_logging: true

  # Generate patch files for rollback
  save_patches: false

  # Checkpoint creation frequency (steps)
  checkpoint_frequency: 1

  # Enable JSON output format
  json_output: false

  # Enable terminal UI (if available)
  enable_tui: true
```

### Hooks

Defines lifecycle hooks for notifications:

```yaml
hooks:
  # After plan generation
  on_plan_created:
    - type: "webhook"
      url: "${WEBHOOK_URL}"
      method: "POST"

  # Before each step
  on_step_before: []

  # After each step
  on_step_after: []

  # When approval needed
  on_approval_requested: []

  # On successful completion
  on_complete:
    - type: "slack"
      channel: "#deployments"
      webhook_url: "${SLACK_WEBHOOK_URL}"
      message: "✅ Deployment complete"

  # On errors
  on_error:
    - type: "slack"
      channel: "#alerts"
      webhook_url: "${SLACK_WEBHOOK_URL}"
      message: "❌ Deployment failed"
```

**Supported Hook Types**:
- `webhook`: HTTP webhook
- `slack`: Slack notification
- `email`: Email notification (requires SMTP configuration)

---

## Environment Variables

Profile values can reference environment variables:

```yaml
hooks:
  on_complete:
    - type: "slack"
      webhook_url: "${SLACK_WEBHOOK_URL}"
```

Set environment variables before running:

```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/..."
specular auto --profile production "deploy to staging"
```

---

## Common Patterns

### Development Profile

For local development with fast iteration:

```yaml
profiles:
  dev:
    approvals:
      mode: "none"              # No approvals for speed
      interactive: false

    safety:
      max_steps: 20
      timeout: "45m"
      max_cost_usd: 15.0

    routing:
      temperature: 0.8          # Higher creativity

    execution:
      trace_logging: false      # Disable for speed
      enable_tui: true
```

Usage:
```bash
specular auto --profile dev "implement feature"
```

### Production Profile

For production deployments with maximum safety:

```yaml
profiles:
  production:
    approvals:
      mode: "all"               # Approve everything
      interactive: true

    safety:
      max_steps: 5
      timeout: "10m"
      max_cost_usd: 2.0
      blocked_step_types:
        - "build:run"           # Block direct builds

    routing:
      temperature: 0.3          # Lower temperature

    execution:
      save_patches: true        # Enable rollback
      json_output: true
```

Usage:
```bash
specular auto --profile production "deploy to production"
```

### Budget Profile

For cost-conscious development:

```yaml
profiles:
  budget:
    approvals:
      mode: "critical_only"

    safety:
      max_steps: 8
      timeout: "15m"
      max_cost_usd: 1.0         # Strict budget
      max_cost_per_task: 0.15

    routing:
      preferred_agent: "openai"
      model_preferences:
        "spec:update": "gpt-4-turbo"  # More cost-effective
```

Usage:
```bash
specular auto --profile budget "implement feature"
```

---

## Best Practices

### Profile Organization

1. **Use project-level profiles** for team-shared configurations
2. **Use user-level profiles** for personal preferences
3. **Version control project profiles** in `./auto.profiles.yaml`
4. **Document custom profiles** with clear descriptions

### Safety Guidelines

1. **Start with strict profiles** and relax as you gain confidence
2. **Always enable `require_policy`** in production profiles
3. **Use lower `temperature`** for production (0.3-0.5)
4. **Enable `save_patches`** for production deployments
5. **Set appropriate `max_cost_usd`** to prevent runaway costs

### CI/CD Integration

1. **Use `ci` profile** as a starting point
2. **Always set `interactive: false`**
3. **Enable `json_output: true`** for parsing results
4. **Set `save_patches: true`** for rollback capability
5. **Configure hooks** for notifications

Example GitHub Actions:

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run autonomous mode
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
        run: |
          specular auto --profile ci "deploy to staging"
```

### Testing Profiles

Test profiles before using in production:

```bash
# Test with dry-run
specular auto --profile my-profile --dry-run "implement feature"

# Test with verbose output
specular auto --profile my-profile --verbose "implement feature"

# Test cost limits
specular auto --profile my-profile --max-cost 0.10 "simple task"
```

---

## Troubleshooting

### Profile Not Found

**Error**: `profile "my-profile" not found`

**Solutions**:
1. List available profiles: `specular auto --list-profiles`
2. Check profile file location
3. Verify YAML syntax
4. Check profile name matches in file

### Invalid Profile Configuration

**Error**: `invalid profile: max_steps must be between 1 and 100`

**Solutions**:
1. Validate YAML syntax
2. Check required fields
3. Verify value ranges
4. Use `--profile default` temporarily

### CLI Override Not Working

**Issue**: CLI flag doesn't override profile setting

**Solutions**:
1. Verify flag syntax: `--max-steps 10` not `--max_steps 10`
2. Check flag comes after `--profile`
3. Use `--verbose` to see effective configuration

---

## Migration Guide

### From No Profiles to Profiles

Before (v1.3.0):
```bash
specular auto \
  --require-approval \
  --max-steps 12 \
  --max-cost 5.0 \
  "implement feature"
```

After (v1.4.0):
```bash
# Create profile
echo 'schema: "specular.auto.profiles/v1"
profiles:
  my-default:
    approvals:
      mode: "critical_only"
      interactive: true
    safety:
      max_steps: 12
      max_cost_usd: 5.0
    ...' > ./auto.profiles.yaml

# Use profile
specular auto --profile my-default "implement feature"
```

---

## Examples

See [examples/auto.profiles.yaml](../examples/auto.profiles.yaml) for comprehensive examples including:
- Custom default profile
- Production profile with hooks
- Fast development profile
- Testing profile
- Budget-conscious profile

---

## References

- [Profile Schema Design](./profiles-schema.md) - Detailed schema specification
- [GitHub Issue #1](https://github.com/felixgeelhaar/specular/issues/1) - Profile system implementation
- [Examples](../examples/auto.profiles.yaml) - Example profiles

---

**Last Updated**: 2025-11-11
**Version**: v1.4.0
