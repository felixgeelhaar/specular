# Team Approval Workflow Examples

This directory contains practical examples of team-based bundle approval workflows using Specular's governance bundle system.

## Overview

These examples demonstrate how to use bundle approvals to implement governance policies for software releases. They showcase different governance levels and approval patterns.

## Prerequisites

- Specular CLI installed and configured
- SSH or GPG keys set up for signing
- A project with `spec.yaml` and `spec.lock.json` files

## Examples

### 1. Single Approver Workflow (L2 Governance)

**File**: `single-approver.sh`

**Scenario**: A product manager must approve all releases before deployment.

**Governance Level**: L2 (Managed)

**Steps**:
1. Developer builds bundle requiring PM approval
2. PM reviews and approves the bundle
3. Bundle is verified and ready for deployment

**Usage**:
```bash
./single-approver.sh
```

### 2. Multi-Approver Workflow (L3 Governance)

**File**: `multi-approver.sh`

**Scenario**: Both Product Manager and Security Engineer must approve production releases.

**Governance Level**: L3 (Defined)

**Steps**:
1. Developer builds bundle requiring PM and security approvals
2. PM approves product requirements
3. Security engineer approves security posture
4. Bundle is verified with all required approvals
5. Bundle is ready for production deployment

**Usage**:
```bash
./multi-approver.sh
```

## Workflow Patterns

### L1 (Basic) - No Approvals Required
```bash
# Build bundle without approval requirements
specular bundle build --output release.sbundle.tgz
```

### L2 (Managed) - Single Approval
```bash
# Build with single approval requirement
specular bundle build \
  --require-approval pm \
  --governance-level L2 \
  --output release.sbundle.tgz

# Approve as PM
specular bundle approve release.sbundle.tgz \
  --role pm \
  --user pm@company.com
```

### L3 (Defined) - Multiple Approvals
```bash
# Build with multiple approval requirements
specular bundle build \
  --require-approval pm \
  --require-approval security \
  --governance-level L3 \
  --output release.sbundle.tgz

# Each role approves
specular bundle approve release.sbundle.tgz --role pm
specular bundle approve release.sbundle.tgz --role security
```

### L4 (Optimized) - Attestations + Multiple Approvals
```bash
# Build with attestations and multiple approvals
specular bundle build \
  --require-approval pm \
  --require-approval security \
  --governance-level L4 \
  --attest \
  --output release.sbundle.tgz

# Verify includes attestation validation
specular bundle verify release.sbundle.tgz --verify-attestation
```

## Approval Verification

Check approval status at any time:
```bash
specular bundle approval-status release.sbundle.tgz
```

Output shows:
- Required approvals
- Current approval status
- Missing approvals
- Signature verification status

## Best Practices

### 1. Role Definition
- Define clear roles aligned with your organization (pm, security, compliance, etc.)
- Document role responsibilities
- Map roles to actual team members

### 2. Key Management
- Use separate keys for each role
- Store private keys securely (hardware security keys recommended for L4)
- Maintain key rotation policy

### 3. Approval Process
- Establish clear criteria for each approval role
- Document approval checklists
- Maintain approval audit trail

### 4. Security
- Use SSH keys for L2/L3
- Use GPG keys for enhanced security
- Use hardware security keys for L4
- Enable attestations for production releases

### 5. Automation
- Integrate approval checks into CI/CD pipelines
- Automate approval notifications
- Use webhooks for approval status updates

## Troubleshooting

### Approval Rejected
```bash
# Check approval status
specular bundle approval-status release.sbundle.tgz

# Verify signature
specular bundle verify release.sbundle.tgz
```

### Missing Approvals
```bash
# List required approvals
specular bundle approval-status release.sbundle.tgz

# Approve with correct role
specular bundle approve release.sbundle.tgz --role <required-role>
```

### Invalid Signature
```bash
# Verify bundle integrity
specular bundle verify release.sbundle.tgz

# Re-sign if needed
specular bundle approve release.sbundle.tgz --role <role> --force
```

## Further Reading

- [Bundle User Guide](../../docs/BUNDLE_USER_GUIDE.md) - Complete bundle documentation
- [Security Audit](../../docs/SECURITY_AUDIT.md) - Security considerations and best practices
- [CI/CD Examples](../cicd-github-actions/) - GitHub Actions integration
- [Registry Publishing](../registry-publishing/) - Publishing bundles to registries
