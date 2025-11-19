# Governance: Setting Up Governed AI Development

This tutorial covers setting up and managing a governance workspace for AI-driven development.

> **Note**: Governance features require Specular Pro or Enterprise license.

## Overview

Governance in Specular provides:
- Provider allowlists and cost limits
- Policy enforcement and approval workflows
- Audit trails and compliance controls
- Cryptographic attestations

---

## Prerequisites

- Specular Pro or Enterprise license
- Completed [Full Workflow](./02-full-workflow.md) tutorial
- Basic understanding of policy concepts

---

## Step 1: Initialize Governance Workspace

Create the governance infrastructure:

```bash
specular governance init
```

This creates:

```
.specular/
├── providers.yaml    # Provider allowlist and configuration
├── policies.yaml     # Policy definitions and rules
├── approvals/        # Approval records and signatures
├── bundles/          # Governance bundles
└── traces/           # Execution traces for audit
```

### Customize the Location

```bash
# Initialize in custom directory
specular governance init --path /path/to/governance

# Force overwrite existing
specular governance init --force
```

---

## Step 2: Configure Providers

Edit `.specular/providers.yaml` to define allowed AI providers:

```yaml
version: "1.0"

# Allowed providers (allowlist)
allow:
  - "ollama:llama3.2"
  - "ollama:qwen2.5-coder"
  - "openai:gpt-4o-mini"
  - "anthropic:claude-sonnet-3.5"

# Provider-specific settings
providers:
  ollama:
    base_url: "http://localhost:11434"
    timeout: 30s

  openai:
    api_key_env: "OPENAI_API_KEY"
    timeout: 60s

# Cost limits
limits:
  max_cost_usd: 10.00
  max_tokens_per_request: 16000
  max_requests_per_hour: 100

# Routing preferences
routing:
  prefer_local: true
  fallback_to_cloud: false
```

### Key Configuration Options

| Option | Description |
|--------|-------------|
| `allow` | Explicit allowlist of provider:model pairs |
| `providers.*` | Provider-specific settings (URLs, keys, timeouts) |
| `limits.max_cost_usd` | Maximum cost per run in USD |
| `limits.max_tokens_per_request` | Token limit per API call |
| `routing.prefer_local` | Prioritize local models |

---

## Step 3: Define Policies

Edit `.specular/policies.yaml` to set governance rules:

```yaml
version: "1.0"

# Enforcement level
enforcement: "strict"  # Options: strict, warn, monitor

# Workflow policies
workflows:
  require_approval_for:
    - "bundle.gate"
    - "policy.approve"
    - "drift.approve"

  require_attestation_for:
    - "bundle.gate"

# Security policies
security:
  require_encryption: true
  audit_all_actions: true
  secrets_in_vault: false
```

### Enforcement Levels

| Level | Behavior |
|-------|----------|
| `strict` | Block operations that violate policies |
| `warn` | Allow but warn about violations |
| `monitor` | Log violations for audit only |

---

## Step 4: Validate Environment

Run the doctor command to check your setup:

```bash
specular governance doctor
```

Output:

```
Running governance environment checks...
📁 Checking governance workspace... ✓ OK
🔌 Checking providers.yaml... ✓ OK
📋 Checking policies.yaml... ✓ OK
📂 Checking directory structure... ✓ OK

✅ All governance checks passed!

Your governance environment is properly configured.
```

### Common Issues

| Issue | Solution |
|-------|----------|
| Workspace not found | Run `specular governance init` |
| providers.yaml missing | Re-run init or create manually |
| Directory structure incomplete | Re-run init with `--force` |

---

## Step 5: Check Governance Status

View the current health overview:

```bash
specular governance status
```

Output:

```
=== Governance Status ===
Workspace: /path/to/project/.specular
Status: Initialized ✓

Approvals: 0
Bundles: 0
Traces: 0

License: pro
```

---

## Best Practices

### 1. Start with Monitor Mode

Begin with `enforcement: "monitor"` to observe policy effects before enforcing:

```yaml
enforcement: "monitor"
```

### 2. Use Local Providers First

Prioritize local models for development to reduce costs:

```yaml
routing:
  prefer_local: true
  fallback_to_cloud: false
```

### 3. Set Reasonable Cost Limits

Start with low limits and increase as needed:

```yaml
limits:
  max_cost_usd: 1.00
  max_requests_per_hour: 50
```

### 4. Require Approvals for Production

Enforce approvals for production-bound artifacts:

```yaml
workflows:
  require_approval_for:
    - "bundle.gate"
    - "drift.approve"
```

### 5. Enable Audit Logging

Always enable auditing for compliance:

```yaml
security:
  audit_all_actions: true
```

---

## Integration with CI/CD

### GitHub Actions

```yaml
jobs:
  governance-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Specular
        run: |
          curl -fsSL https://get.specular.dev | bash

      - name: Validate Governance
        run: |
          specular governance doctor
```

### GitLab CI

```yaml
governance:
  stage: validate
  script:
    - specular governance doctor
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
```

---

## Governance Workflow Summary

| Step | Command | Purpose |
|------|---------|---------|
| 1 | `governance init` | Create workspace structure |
| 2 | Edit `providers.yaml` | Define allowed providers |
| 3 | Edit `policies.yaml` | Set governance rules |
| 4 | `governance doctor` | Validate configuration |
| 5 | `governance status` | Monitor health |

---

## Next Steps

- [Policy Management](./05-policy-management.md) - Detailed policy workflows
- [Bundles](./06-bundles.md) - Creating governance bundles
- [Approvals](./07-approvals.md) - Approval workflows
- [CLI Reference](../CLI_REFERENCE.md) - Complete command reference
