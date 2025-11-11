# Profile System Schema Design

**Feature**: Profile System for Autonomous Mode
**Version**: v1.4.0
**Status**: Design Phase
**Last Updated**: 2025-11-11

---

## Overview

The profile system enables environment-specific configurations for autonomous mode, allowing different behavior in interactive development, CI/CD pipelines, and custom scenarios. Profiles define approval rules, safety limits, routing preferences, and execution policies.

---

## Profile YAML Schema

### File Locations

Profiles are loaded from the following locations (in order of precedence):

1. **Project-level**: `./auto.profiles.yaml` (highest precedence)
2. **User-level**: `~/.specular/auto.profiles.yaml`
3. **Built-in defaults**: Embedded in the binary

### Schema Version

```yaml
schema: "specular.auto.profiles/v1"
```

---

## Profile Structure

### Top-Level Schema

```yaml
schema: "specular.auto.profiles/v1"

profiles:
  # Profile name (default, ci, custom, etc.)
  <profile-name>:
    # Human-readable description
    description: "Profile description"

    # Approval configuration
    approvals:
      # Approval mode: all, critical_only, none
      mode: "critical_only"

      # Interactive mode (true/false)
      interactive: true

      # Auto-approve specific step types
      auto_approve:
        - "spec:update"
        - "plan:gen"

      # Always require approval for these step types
      require_approval:
        - "spec:lock"
        - "build:run"

    # Safety limits
    safety:
      # Maximum number of steps
      max_steps: 12

      # Maximum execution time
      timeout: "25m"

      # Maximum total cost (USD)
      max_cost_usd: 5.0

      # Maximum cost per task (USD)
      max_cost_per_task: 0.50

      # Maximum retries per task
      max_retries: 3

      # Require policy checks
      require_policy: true

      # Allowed step types (whitelist)
      allowed_step_types:
        - "spec:update"
        - "spec:lock"
        - "plan:gen"
        - "build:run"

      # Blocked step types (blacklist)
      blocked_step_types: []

    # Routing configuration
    routing:
      # Preferred agent for execution
      preferred_agent: "cline"

      # Fallback agent if preferred unavailable
      fallback_agent: "openai"

      # Temperature for LLM calls
      temperature: 0.7

      # Model preferences by step type
      model_preferences:
        "spec:update": "claude-sonnet-3.5"
        "plan:gen": "claude-sonnet-3.5"
        "build:run": "claude-opus-3.5"

    # Policy configuration
    policies:
      # Enable policy checks
      enabled: true

      # Policy files to load
      policy_files:
        - "~/.specular/policies/default.rego"
        - "./policies/project.rego"

      # Policy enforcement level: strict, warn, none
      enforcement: "strict"

    # Execution configuration
    execution:
      # Enable trace logging
      trace_logging: true

      # Enable patch generation
      save_patches: false

      # Checkpoint frequency (number of steps)
      checkpoint_frequency: 1

      # Enable JSON output
      json_output: false

      # Enable TUI (if available)
      enable_tui: false

    # Notification hooks (optional)
    hooks:
      on_plan_created:
        - type: "webhook"
          url: "https://example.com/webhook"

      on_complete:
        - type: "slack"
          channel: "#deployments"
          webhook_url: "${SLACK_WEBHOOK_URL}"

      on_error:
        - type: "email"
          recipients:
            - "team@example.com"
```

---

## Built-in Profiles

### Default Profile (Interactive Development)

```yaml
profiles:
  default:
    description: "Default profile for interactive development"

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
      max_steps: 12
      timeout: "25m"
      max_cost_usd: 5.0
      max_cost_per_task: 0.50
      max_retries: 3
      require_policy: true
      allowed_step_types:
        - "spec:update"
        - "spec:lock"
        - "plan:gen"
        - "build:run"
      blocked_step_types: []

    routing:
      preferred_agent: "cline"
      fallback_agent: "openai"
      temperature: 0.7
      model_preferences:
        "spec:update": "claude-sonnet-3.5"
        "plan:gen": "claude-sonnet-3.5"
        "build:run": "claude-opus-3.5"

    policies:
      enabled: true
      policy_files: []
      enforcement: "strict"

    execution:
      trace_logging: true
      save_patches: false
      checkpoint_frequency: 1
      json_output: false
      enable_tui: true

    hooks: {}
```

### CI Profile (Non-Interactive CI/CD)

```yaml
profiles:
  ci:
    description: "Profile for CI/CD pipelines (non-interactive)"

    approvals:
      mode: "none"
      interactive: false
      auto_approve:
        - "spec:update"
        - "spec:lock"
        - "plan:gen"
        - "build:run"
      require_approval: []

    safety:
      max_steps: 8
      timeout: "15m"
      max_cost_usd: 2.0
      max_cost_per_task: 0.25
      max_retries: 2
      require_policy: true
      allowed_step_types:
        - "spec:update"
        - "spec:lock"
        - "plan:gen"
        - "build:run"
      blocked_step_types: []

    routing:
      preferred_agent: "openai"
      fallback_agent: "cline"
      temperature: 0.5
      model_preferences:
        "spec:update": "gpt-4-turbo"
        "plan:gen": "gpt-4-turbo"
        "build:run": "gpt-4"

    policies:
      enabled: true
      policy_files: []
      enforcement: "strict"

    execution:
      trace_logging: true
      save_patches: true
      checkpoint_frequency: 1
      json_output: true
      enable_tui: false

    hooks: {}
```

### Strict Profile (Maximum Safety)

```yaml
profiles:
  strict:
    description: "Maximum safety profile with strict approval gates"

    approvals:
      mode: "all"
      interactive: true
      auto_approve: []
      require_approval:
        - "spec:update"
        - "spec:lock"
        - "plan:gen"
        - "build:run"

    safety:
      max_steps: 5
      timeout: "10m"
      max_cost_usd: 1.0
      max_cost_per_task: 0.20
      max_retries: 1
      require_policy: true
      allowed_step_types:
        - "spec:update"
        - "plan:gen"
      blocked_step_types:
        - "build:run"

    routing:
      preferred_agent: "cline"
      fallback_agent: "openai"
      temperature: 0.3
      model_preferences:
        "spec:update": "claude-opus-3.5"
        "plan:gen": "claude-opus-3.5"

    policies:
      enabled: true
      policy_files: []
      enforcement: "strict"

    execution:
      trace_logging: true
      save_patches: true
      checkpoint_frequency: 1
      json_output: false
      enable_tui: true

    hooks: {}
```

---

## Profile Resolution

### Precedence Rules

1. **CLI flags** override all profile settings
2. **Project-level profiles** (`./auto.profiles.yaml`) override user-level
3. **User-level profiles** (`~/.specular/auto.profiles.yaml`) override built-in defaults
4. **Built-in defaults** are used if no profile found

### Profile Selection

```bash
# Use default profile
specular auto "implement user authentication"

# Use ci profile
specular auto --profile ci "implement user authentication"

# Use custom profile
specular auto --profile production "implement user authentication"

# List available profiles
specular auto --list-profiles
```

### CLI Override Examples

```bash
# Override max_steps from profile
specular auto --profile ci --max-steps 10 "implement feature"

# Override timeout from profile
specular auto --profile default --timeout 30m "implement feature"

# Override approval mode from profile
specular auto --profile ci --require-approval "implement feature"

# Multiple overrides
specular auto --profile ci \
  --max-steps 10 \
  --max-cost 3.0 \
  --timeout 20m \
  "implement feature"
```

---

## Profile Validation

### Required Fields

- `approvals.mode` (enum: all, critical_only, none)
- `approvals.interactive` (boolean)
- `safety.max_steps` (positive integer)
- `safety.timeout` (duration string, e.g., "25m", "1h")
- `safety.max_cost_usd` (positive float)
- `safety.max_cost_per_task` (positive float)

### Validation Rules

1. **Timeout Format**: Must be valid Go duration (e.g., "5m", "1h30m", "25m")
2. **Cost Limits**: `max_cost_per_task` <= `max_cost_usd`
3. **Step Limits**: `max_steps` must be positive integer (1-100)
4. **Retries**: `max_retries` must be non-negative integer (0-10)
5. **Step Types**: Must be one of: `spec:update`, `spec:lock`, `plan:gen`, `build:run`
6. **Approval Mode**: Must be one of: `all`, `critical_only`, `none`
7. **Enforcement Level**: Must be one of: `strict`, `warn`, `none`

### Invalid Profile Handling

- **Invalid YAML**: Return error, abort execution
- **Invalid field values**: Return validation error with specific field
- **Missing required fields**: Use default values from built-in profile
- **Unknown fields**: Warn but continue (forward compatibility)

---

## Profile Merging

When multiple profile sources exist, fields are merged with precedence:

```
CLI Flags > Project Profile > User Profile > Built-in Profile
```

### Merge Behavior

- **Scalar values**: Override (last wins)
- **Arrays**: Replace (not append)
- **Maps**: Merge recursively (field-by-field)

### Example

**Built-in Default**:
```yaml
safety:
  max_steps: 12
  timeout: "25m"
  max_cost_usd: 5.0
```

**User Profile**:
```yaml
safety:
  max_steps: 10
  max_cost_usd: 3.0
```

**Result**:
```yaml
safety:
  max_steps: 10        # From user profile
  timeout: "25m"       # From built-in (not overridden)
  max_cost_usd: 3.0    # From user profile
```

---

## Environment Variable Support

Profile values can reference environment variables:

```yaml
profiles:
  production:
    hooks:
      on_complete:
        - type: "slack"
          webhook_url: "${SLACK_WEBHOOK_URL}"
      on_error:
        - type: "pagerduty"
          api_key: "${PAGERDUTY_API_KEY}"
```

### Variable Expansion Rules

- Format: `${VAR_NAME}` or `$VAR_NAME`
- Undefined variables: Return error or use empty string (configurable)
- Escape: Use `$$` to include literal `$`

---

## Profile API

### Go API

```go
// Load profile by name
profile, err := profiles.Load("ci")

// Load from specific file
profile, err := profiles.LoadFromFile("./custom.profiles.yaml", "production")

// Get current effective profile (after CLI merging)
effectiveProfile := profiles.GetEffective()

// Validate profile
if err := profile.Validate(); err != nil {
    return fmt.Errorf("invalid profile: %w", err)
}
```

---

## Future Enhancements

### Phase 2 (v1.5.0+)
- Profile inheritance (`extends: "default"`)
- Profile templates with parameters
- Remote profile loading (URLs)
- Profile versioning and migration

### Phase 3 (v1.6.0+)
- Encrypted profile fields (secrets)
- Profile signing and verification
- Team-shared profiles (GitHub, S3)
- Dynamic profile selection based on context

---

## References

- **Spec**: `specular_auto_spec_v1.md`
- **GitHub Issue**: [#1 Profile System](https://github.com/felixgeelhaar/specular/issues/1)
- **Implementation**: `internal/profiles/`
- **Examples**: `examples/auto.profiles.yaml`

---

**Status**: Design Complete
**Next**: Implementation of profile data structures
