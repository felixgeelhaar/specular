# Approvals: Cryptographic Sign-Off Workflows

This tutorial covers creating, verifying, and managing approval workflows.

> **Note**: Approval features require Specular Pro or Enterprise license.

## Overview

Approvals provide:
- Cryptographic signatures (SSH/GPG)
- Multi-role sign-off workflows
- Audit trails with timestamps
- Verification of stakeholder consent

---

## Prerequisites

- Completed [Bundles](./06-bundles.md) tutorial
- SSH key pair (`~/.ssh/id_ed25519` or `~/.ssh/id_rsa`)
- Or GPG key for signing

---

## Step 1: Sign a Bundle Approval

Create a cryptographic approval for a bundle:

```bash
specular bundle approve my-app-v1.0.0.sbundle.tgz \
  --role pm \
  --user alice@company.com \
  --comment "Approved for Q1 release"
```

Output:

```
Computing bundle digest...
Bundle digest: sha256:a1b2c3d4e5f6g7h8...

Creating ssh signature...
✓ Approval signed successfully

Approval Details:
  Role:      pm
  User:      alice@company.com
  Signed At: 2024-01-15 10:30:00
  Signature: ssh
  Key:       SHA256:aBcDeF...
  Comment:   Approved for Q1 release

✓ Approval saved to: my-app-v1.0.0-pm-20240115-103000-approval.json
```

### Common Roles

| Role | Description |
|------|-------------|
| `pm` | Product Manager |
| `lead` | Tech Lead |
| `security` | Security Engineer |
| `legal` | Legal/Compliance |
| `qa` | Quality Assurance |

---

## Step 2: Choose Signature Type

### SSH Signature (Default)

Uses your SSH key automatically:

```bash
specular bundle approve bundle.sbundle.tgz \
  --role lead \
  --user bob@company.com \
  --signature-type ssh
```

### Specific SSH Key

```bash
specular bundle approve bundle.sbundle.tgz \
  --role lead \
  --user bob@company.com \
  --key-path ~/.ssh/work_id_ed25519
```

### GPG Signature

```bash
specular bundle approve bundle.sbundle.tgz \
  --role security \
  --user charlie@company.com \
  --signature-type gpg \
  --key-path F3A29C8B  # GPG key ID
```

---

## Step 3: Custom Output Path

Save approval to specific location:

```bash
specular bundle approve bundle.sbundle.tgz \
  --role pm \
  --user alice@company.com \
  --output approvals/pm-alice.json
```

---

## Step 4: Check Approval Status

View approval progress for a bundle:

```bash
specular bundle approval-status my-app-v1.0.0.sbundle.tgz \
  --approvals approvals/*.json
```

Output:

```
Computing bundle digest...
Bundle digest: sha256:a1b2c3d4e5f6g7h8...

Loading approval files...
Loaded 3 approval(s)

Verifying approval signatures...
✓ Role pm (alice@company.com): Valid signature
✓ Role lead (bob@company.com): Valid signature
✓ Role security (charlie@company.com): Valid signature

Approval Summary:
  Total approvals: 3
  Valid signatures: 3
  Invalid signatures: 0

Approved by:
  - pm: alice@company.com (signed 2024-01-15 09:00:00)
    Comment: Approved for Q1 release
  - lead: bob@company.com (signed 2024-01-15 09:30:00)
  - security: charlie@company.com (signed 2024-01-15 10:00:00)
```

### Check Required Roles

```bash
specular bundle approval-status bundle.sbundle.tgz \
  --approvals approvals/*.json \
  --required-roles pm,lead,security
```

Output when missing roles:

```
Checking required roles...
✓ pm: Approved
✓ lead: Approved
✗ security: Missing or invalid approval

⚠ Bundle is missing 1 required approval(s): security
```

---

## Step 5: Approve Generic Resources

Approve bundles, drift, or policies by ID:

```bash
# Approve a bundle
specular approve bundle-a1b2c3d4 --message "Approved for production"

# Approve drift detection
specular approve drift-x1y2z3a4 --message "Drift accepted for hotfix"

# Approve policy change
specular approve policy-change-123 --message "Security update approved"
```

---

## Step 6: List All Approvals

View all approval records:

```bash
specular approvals list
```

Output:

```
=== Approval Records ===
Policy Approvals: 2
  • policy-change-123
    Approved by: alice@company.com
    Approved at: 2024-01-15 10:30:00
    Message: Security update approved

  • policy-change-456
    Approved by: bob@company.com
    Approved at: 2024-01-10 14:00:00

Bundle Approvals: 3
  • bundle-a1b2c3d4
    Approved by: alice@company.com
    Approved at: 2024-01-15 09:00:00
    Message: Approved for production

  • bundle-a1b2c3d4
    Approved by: bob@company.com
    Approved at: 2024-01-15 09:30:00

  • bundle-a1b2c3d4
    Approved by: charlie@company.com
    Approved at: 2024-01-15 10:00:00

Total approvals: 5
```

---

## Step 7: Check Pending Approvals

Find resources waiting for approval:

```bash
specular approvals pending
```

Output:

```
=== Pending Approvals ===
📋 Policy Changes:
  • Policies have changed since last approval
  • Run 'specular policy diff' to see changes
  • Run 'specular policy approve' to approve

📦 Bundles: 2 pending
  • bundle-x1y2z3a4
  • bundle-p7q8r9s0
  Run 'specular approve <bundle-id>' to approve

🔀 Drift Detected:
  • Drift detected but not approved
  • Run 'specular drift check' to see details
  • Run 'specular drift approve' to approve
```

Exit code 1 if pending approvals exist (for CI/CD).

---

## Multi-Role Approval Workflow

### Complete Workflow Example

```bash
# Step 1: Create bundle
specular bundle create my-app-v1.0.0.sbundle.tgz

# Step 2: PM approves
specular bundle approve my-app-v1.0.0.sbundle.tgz \
  --role pm \
  --user alice@company.com \
  --comment "Product requirements met"

# Step 3: Tech Lead approves
specular bundle approve my-app-v1.0.0.sbundle.tgz \
  --role lead \
  --user bob@company.com \
  --comment "Code quality verified"

# Step 4: Security approves
specular bundle approve my-app-v1.0.0.sbundle.tgz \
  --role security \
  --user charlie@company.com \
  --comment "Security review passed"

# Step 5: Verify all approvals
specular bundle approval-status my-app-v1.0.0.sbundle.tgz \
  --approvals *.json \
  --required-roles pm,lead,security

# Step 6: Gate check passes
specular bundle gate --require-approvals my-app-v1.0.0.sbundle.tgz
```

---

## CI/CD Integration

### GitHub Actions Workflow

```yaml
name: Approval Workflow

on:
  workflow_dispatch:
    inputs:
      approver_email:
        description: 'Approver email'
        required: true
      approver_role:
        description: 'Approver role (pm, lead, security)'
        required: true

jobs:
  approve:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download Bundle
        run: |
          specular bundle pull \
            ghcr.io/${{ github.repository }}:latest \
            bundle.sbundle.tgz

      - name: Sign Approval
        run: |
          specular bundle approve bundle.sbundle.tgz \
            --role ${{ inputs.approver_role }} \
            --user ${{ inputs.approver_email }} \
            --key-path ${{ secrets.SSH_SIGNING_KEY }}

      - name: Upload Approval
        uses: actions/upload-artifact@v4
        with:
          name: approval-${{ inputs.approver_role }}
          path: '*-approval.json'

  gate-check:
    needs: approve
    runs-on: ubuntu-latest
    steps:
      - name: Download Approvals
        uses: actions/download-artifact@v4

      - name: Check All Required Approvals
        run: |
          specular bundle approval-status bundle.sbundle.tgz \
            --approvals approvals/*.json \
            --required-roles pm,lead,security
```

### GitLab CI

```yaml
approve:
  stage: approve
  when: manual
  script:
    - specular bundle approve bundle.sbundle.tgz \
        --role ${APPROVER_ROLE} \
        --user ${APPROVER_EMAIL}
  artifacts:
    paths:
      - '*-approval.json'

gate-check:
  stage: deploy
  needs: [approve]
  script:
    - specular bundle approval-status bundle.sbundle.tgz \
        --approvals *.json \
        --required-roles pm,lead,security
```

---

## Best Practices

### 1. Use Dedicated Signing Keys

Create separate keys for signing approvals:

```bash
ssh-keygen -t ed25519 -f ~/.ssh/approval_key -C "approval@company.com"
```

### 2. Require Multiple Roles

Configure policies to require multiple approvers:

```yaml
# policies.yaml
workflows:
  require_approval_for:
    - "bundle.gate"
  require_roles:
    - "pm"
    - "lead"
    - "security"
```

### 3. Include Meaningful Comments

Document why you're approving:

```bash
specular bundle approve bundle.sbundle.tgz \
  --role security \
  --user security@company.com \
  --comment "OWASP scan passed, no critical vulnerabilities"
```

### 4. Verify Before Deploy

Always check approval status in CI/CD:

```bash
specular bundle approval-status bundle.sbundle.tgz \
  --approvals approvals/*.json \
  --required-roles pm,lead,security

if [ $? -ne 0 ]; then
  echo "Missing required approvals"
  exit 1
fi
```

### 5. Archive Approval Records

Store approvals alongside bundles:

```bash
mkdir -p releases/v1.0.0/approvals
mv *-approval.json releases/v1.0.0/approvals/
mv bundle.sbundle.tgz releases/v1.0.0/
```

---

## Troubleshooting

### "SSH key not found"

Specify key path explicitly:

```bash
specular bundle approve bundle.sbundle.tgz \
  --role pm \
  --user alice@company.com \
  --key-path ~/.ssh/id_ed25519
```

### "Invalid signature"

Check that the bundle hasn't been modified since signing:

```bash
specular bundle gate bundle.sbundle.tgz
```

### "Missing required approvals"

Collect approvals from all required roles:

```bash
specular bundle approval-status bundle.sbundle.tgz \
  --approvals approvals/*.json \
  --required-roles pm,lead,security
```

---

## Command Reference

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `bundle approve` | Sign approval | `--role`, `--user`, `--comment`, `--key-path`, `--signature-type` |
| `bundle approval-status` | Check approvals | `--approvals`, `--required-roles`, `--json` |
| `approve` | Approve resource | `--message` |
| `approvals list` | List all approvals | - |
| `approvals pending` | Show pending | - |

---

## Next Steps

- [CLI Reference](../CLI_REFERENCE.md) - Complete command reference
- [Production Guide](../PRODUCTION_GUIDE.md) - Deployment best practices
- [Governance](./04-governance.md) - Review governance setup
