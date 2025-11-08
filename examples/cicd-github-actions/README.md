# CI/CD Integration Examples - GitHub Actions

This directory contains examples of integrating Specular bundle workflows into GitHub Actions CI/CD pipelines.

## Overview

These examples demonstrate how to automate bundle creation, approval, verification, and deployment using GitHub Actions. They showcase different deployment strategies and governance levels.

## Prerequisites

- GitHub repository with Actions enabled
- Specular CLI available in the GitHub Actions runner
- SSH or GPG keys configured as GitHub secrets
- Container registry access (GHCR, Docker Hub, or other)

## Examples

### 1. Complete Bundle Workflow

**File**: `bundle-workflow.yml`

**Triggers**:
- Push to `main` branch
- Pull request to `main` branch
- Manual workflow dispatch

**Steps**:
1. Build bundle with appropriate governance level
2. Run automated tests and security scans
3. Request required approvals (for production)
4. Verify bundle integrity
5. Push bundle to registry
6. Deploy to target environment

**Features**:
- Environment-specific governance levels (dev: L1, staging: L2, prod: L3)
- Automated approval for non-production environments
- Manual approval gates for production
- Attestation generation for production releases
- Registry publication with semantic versioning
- Deployment status notifications

## GitHub Secrets Required

Configure these secrets in your repository settings:

### Authentication
- `BUNDLE_SSH_PRIVATE_KEY` - SSH private key for bundle signing
- `BUNDLE_GPG_PRIVATE_KEY` - GPG private key for production bundles
- `BUNDLE_GPG_PASSPHRASE` - Passphrase for GPG key

### Registry Access
- `GHCR_TOKEN` - GitHub Container Registry token (or use `GITHUB_TOKEN`)
- `DOCKERHUB_USERNAME` - Docker Hub username (if using Docker Hub)
- `DOCKERHUB_TOKEN` - Docker Hub access token

### Approval Users
- `PM_EMAIL` - Product Manager email for approvals
- `SECURITY_EMAIL` - Security Engineer email for approvals

## Environment Configuration

### Development Environment
- **Governance Level**: L1 (Basic)
- **Approvals**: None required
- **Attestations**: Optional
- **Deployment**: Automatic on merge

### Staging Environment
- **Governance Level**: L2 (Managed)
- **Approvals**: PM approval required
- **Attestations**: Optional
- **Deployment**: Automatic after approval

### Production Environment
- **Governance Level**: L3 (Defined)
- **Approvals**: PM + Security approval required
- **Attestations**: Mandatory
- **Deployment**: Manual trigger after approvals

## Usage

### 1. Copy workflow to your repository
```bash
mkdir -p .github/workflows
cp bundle-workflow.yml .github/workflows/
```

### 2. Configure secrets
```bash
# Using GitHub CLI
gh secret set BUNDLE_SSH_PRIVATE_KEY < ~/.ssh/id_ed25519
gh secret set PM_EMAIL --body "pm@company.com"
gh secret set SECURITY_EMAIL --body "security@company.com"
```

### 3. Commit and push
```bash
git add .github/workflows/bundle-workflow.yml
git commit -m "feat: add bundle workflow"
git push
```

### 4. Monitor workflow execution
```bash
# View workflow runs
gh run list --workflow=bundle-workflow.yml

# View specific run
gh run view <run-id>
```

## Workflow Patterns

### Pattern 1: Feature Branch Workflow
```yaml
# Build and verify on feature branches
on:
  pull_request:
    branches: [main]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build and verify bundle
        run: |
          specular bundle build --output feature.sbundle.tgz
          specular bundle verify feature.sbundle.tgz
```

### Pattern 2: Release Workflow
```yaml
# Create and publish release bundles
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Create release bundle
        run: |
          specular bundle build \
            --require-approval pm \
            --require-approval security \
            --governance-level L3 \
            --attest \
            --output release-${GITHUB_REF_NAME}.sbundle.tgz
```

### Pattern 3: Multi-Environment Deployment
```yaml
# Deploy to multiple environments
jobs:
  deploy-dev:
    environment: development
    steps:
      - name: Deploy to dev
        run: specular bundle apply dev.sbundle.tgz

  deploy-staging:
    needs: deploy-dev
    environment: staging
    steps:
      - name: Deploy to staging
        run: specular bundle apply staging.sbundle.tgz

  deploy-prod:
    needs: deploy-staging
    environment: production
    steps:
      - name: Deploy to production
        run: specular bundle apply prod.sbundle.tgz
```

## Advanced Features

### 1. Approval Automation

Use GitHub Actions approval gates:
```yaml
jobs:
  approve:
    runs-on: ubuntu-latest
    environment: production # Triggers approval gate
    steps:
      - name: Request approval
        run: echo "Waiting for approval..."
```

### 2. Bundle Diffing

Compare bundles between releases:
```yaml
- name: Compare with previous release
  run: |
    specular bundle pull ghcr.io/org/app:previous
    specular bundle diff previous.sbundle.tgz current.sbundle.tgz
```

### 3. Attestation Verification

Verify attestations in CI:
```yaml
- name: Verify attestations
  run: |
    specular bundle verify release.sbundle.tgz --verify-attestation
```

### 4. Registry Publishing

Publish to multiple registries:
```yaml
- name: Publish to registries
  run: |
    specular bundle push bundle.sbundle.tgz ghcr.io/org/app:${{ github.sha }}
    specular bundle push bundle.sbundle.tgz docker.io/org/app:${{ github.sha }}
```

## Troubleshooting

### Workflow Fails at Build Step
```bash
# Check workflow logs
gh run view <run-id> --log

# Common issues:
# - Missing spec.yaml or spec.lock.json
# - Invalid governance level
# - Missing required files
```

### Approval Step Fails
```bash
# Verify secrets are configured
gh secret list

# Check approval status
specular bundle approval-status bundle.sbundle.tgz
```

### Registry Push Fails
```bash
# Verify registry credentials
echo $GHCR_TOKEN | docker login ghcr.io -u $GITHUB_ACTOR --password-stdin

# Check registry permissions
docker push ghcr.io/org/app:test
```

## Best Practices

### 1. Environment Isolation
- Use separate bundles for each environment
- Configure environment-specific governance levels
- Maintain environment-specific secrets

### 2. Approval Gates
- Use GitHub Environments for approval gates
- Configure required reviewers for production
- Document approval criteria

### 3. Security
- Rotate signing keys regularly
- Use hardware security keys for production
- Enable attestations for all releases
- Scan bundles for vulnerabilities

### 4. Versioning
- Tag bundles with semantic versions
- Include git SHA in bundle metadata
- Maintain bundle changelog

### 5. Monitoring
- Set up workflow status notifications
- Monitor deployment success rates
- Track approval turnaround times
- Alert on failed verifications

## Further Reading

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Bundle User Guide](../../docs/BUNDLE_USER_GUIDE.md)
- [Team Approval Examples](../team-approval/)
- [Registry Publishing Examples](../registry-publishing/)
