# Policy Management: Governance Rules and Approval Workflows

This tutorial covers creating, validating, and approving governance policies.

> **Note**: Policy management features require Specular Pro or Enterprise license.

## Overview

Policies define:
- Provider allowlists and cost limits
- Workflow approval requirements
- Security and compliance rules
- Enforcement levels

---

## Prerequisites

- Completed [Governance](./04-governance.md) tutorial
- Governance workspace initialized

---

## Step 1: Initialize Policy Template

Create a policies.yaml file from a template:

```bash
# Default strict template (recommended)
specular policy init

# Basic template for getting started
specular policy init --template basic

# Enterprise template with compliance
specular policy init --template enterprise
```

### Available Templates

| Template | Use Case | Features |
|----------|----------|----------|
| `basic` | Development | Minimal policies, warn mode |
| `strict` | Production (default) | Approvals, attestations, encryption |
| `enterprise` | Compliance | SOC2, audit logs, MFA |

---

## Step 2: Understand Policy Structure

### Basic Template

```yaml
version: "1.0"
enforcement: "warn"

workflows:
  require_approval_for:
    - "bundle.gate"

security:
  require_encryption: false
  audit_all_actions: false
  secrets_in_vault: false
```

### Strict Template (Recommended)

```yaml
version: "1.0"
enforcement: "strict"

workflows:
  require_approval_for:
    - "bundle.gate"
    - "policy.approve"
    - "drift.approve"
  require_attestation_for:
    - "bundle.gate"

security:
  require_encryption: true
  audit_all_actions: true
  secrets_in_vault: false
```

### Enterprise Template

```yaml
version: "1.0"
enforcement: "strict"

workflows:
  require_approval_for:
    - "bundle.gate"
    - "policy.approve"
    - "drift.approve"
    - "provider.add"
    - "provider.remove"
  require_attestation_for:
    - "bundle.gate"
    - "policy.approve"

security:
  require_encryption: true
  audit_all_actions: true
  secrets_in_vault: true

compliance:
  soc2_enabled: true
  export_audit_logs: true
  retention_days: 365
  require_mfa: true
```

---

## Step 3: Validate Policies

Check policy syntax and correctness:

```bash
specular policy validate
```

Output:

```
Validating policies...

✓ policies.yaml exists
✓ Valid YAML syntax
✓ Version: 1.0
✓ Enforcement: strict
✓ Approval workflows: 3 defined

✅ All policy validations passed!
```

### Strict Validation

Fail on warnings:

```bash
specular policy validate --strict
```

---

## Step 4: View Current Policies

List all configured policies:

```bash
specular policy list
```

Output:

```
=== Policy Configuration ===

Version: 1.0
Enforcement: strict
Last Modified: 2024-01-15 10:30:00
Hash: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6...

Workflow Policies:
  Require Approval For:
    - bundle.gate
    - policy.approve
    - drift.approve
  Require Attestation For:
    - bundle.gate

Security Policies:
  Require Encryption: true
  Audit All Actions: true
  Secrets in Vault: false

Approvals: 2 on record
```

---

## Step 5: Approve Policies

Create a cryptographic approval record:

```bash
specular policy approve --user "alice@company.com" --message "Approved for Q1 release"
```

Output:

```
✅ Policies approved successfully!

Approved by: alice@company.com
Policy hash: a1b2c3d4e5f6g7h8...
Approval saved: .specular/approvals/policy-20240115-103000.yaml
```

### Approval Record

The approval record contains:

```yaml
policy_hash: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6..."
approved_by: "alice@company.com"
approved_at: "2024-01-15T10:30:00Z"
message: "Approved for Q1 release"
version: "1.0"
```

---

## Step 6: Check Policy Changes

View changes since last approval:

```bash
specular policy diff
```

### No Changes

```
✅ No changes since last approval
Current hash: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6...
Approved: 2024-01-15T10:30:00Z
Approved by: alice@company.com
```

### Changes Detected

```
⚠️  Policies have changed since last approval

Current hash:  x1y2z3a4b5c6d7e8...
Approved hash: a1b2c3d4e5f6g7h8...

Last approved: 2024-01-15T10:30:00Z
Approved by: alice@company.com

Run 'specular policy approve' to approve current changes.
```

---

## Policy Workflow

### Complete Workflow

```bash
# 1. Initialize (if needed)
specular policy init --template strict

# 2. Customize policies.yaml
vim .specular/policies.yaml

# 3. Validate
specular policy validate --strict

# 4. Review current state
specular policy list

# 5. Approve
specular policy approve --user "your-email@company.com"

# 6. Verify no changes
specular policy diff
```

---

## Customization Examples

### Add Custom Approval Workflow

```yaml
workflows:
  require_approval_for:
    - "bundle.gate"
    - "policy.approve"
    - "drift.approve"
    - "production.deploy"     # Custom workflow
    - "security.override"     # Custom workflow
```

### Increase Security

```yaml
security:
  require_encryption: true
  audit_all_actions: true
  secrets_in_vault: true      # Requires vault setup
```

### Add Compliance (Enterprise)

```yaml
compliance:
  soc2_enabled: true
  export_audit_logs: true
  retention_days: 365
  require_mfa: true
```

---

## CI/CD Integration

### Pre-merge Policy Check

```yaml
# GitHub Actions
- name: Validate Policies
  run: |
    specular policy validate --strict
    specular policy diff
```

### Fail on Unapproved Changes

```yaml
- name: Check Policy Approvals
  run: |
    if ! specular policy diff; then
      echo "Policy changes require approval"
      exit 1
    fi
```

---

## Best Practices

### 1. Version Control Policies

Commit `policies.yaml` to version control:

```bash
git add .specular/policies.yaml
git commit -m "feat: add production governance policies"
```

### 2. Require Multiple Approvers

For production, require approvals from multiple roles:

```yaml
workflows:
  require_approval_for:
    - "bundle.gate"       # Tech lead approval
    - "policy.approve"    # Security approval
    - "drift.approve"     # QA approval
```

### 3. Gradual Enforcement

Start with `warn`, then move to `strict`:

```yaml
# Development
enforcement: "warn"

# Staging
enforcement: "strict"

# Production
enforcement: "strict"
```

### 4. Document Policy Decisions

Include messages in approvals:

```bash
specular policy approve \
  --user "alice@company.com" \
  --message "Approved: Increased cost limits for ML training pipeline"
```

### 5. Regular Policy Reviews

Schedule regular policy audits:

```bash
# Check policy status
specular policy list

# Review changes
specular policy diff

# Re-approve if valid
specular policy approve --user "alice@company.com"
```

---

## Troubleshooting

### "policies.yaml not found"

Initialize policies first:

```bash
specular policy init
```

### "Invalid enforcement level"

Use valid values: `strict`, `warn`, or `monitor`

### "Policies have changed since last approval"

Review and re-approve:

```bash
specular policy diff
specular policy approve --user "your@email.com"
```

---

## Command Reference

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `policy init` | Create template | `--template`, `--path` |
| `policy validate` | Check syntax | `--strict`, `--json` |
| `policy list` | View policies | - |
| `policy approve` | Approve policies | `--user`, `--message` |
| `policy diff` | Show changes | `--unified`, `--json` |

---

## Next Steps

- [Bundles](./06-bundles.md) - Creating governance bundles
- [Approvals](./07-approvals.md) - Approval workflows
- [CLI Reference](../CLI_REFERENCE.md) - Complete command reference
