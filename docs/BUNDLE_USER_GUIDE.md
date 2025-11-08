# Governance Bundle User Guide

**Version**: 1.3.0
**Last Updated**: 2025-11-08

## Table of Contents

1. [Introduction](#introduction)
2. [Core Concepts](#core-concepts)
3. [Getting Started](#getting-started)
4. [Command Reference](#command-reference)
5. [Workflow Examples](#workflow-examples)
6. [Best Practices](#best-practices)
7. [Troubleshooting](#troubleshooting)
8. [Advanced Topics](#advanced-topics)

---

## Introduction

Governance bundles provide a secure, verifiable way to package, distribute, and apply project specifications, policies, and configurations. They combine cryptographic integrity verification, role-based approvals, and transparency logging to ensure trustworthy software delivery.

### Why Use Governance Bundles?

**For Organizations**:
- Ensure specifications are reviewed and approved before deployment
- Maintain audit trail of who approved what changes
- Distribute configurations securely across teams
- Prevent unauthorized modifications

**For Teams**:
- Streamline approval workflows
- Version control for governance artifacts
- Cryptographic verification of authenticity
- Integration with CI/CD pipelines

**For Individuals**:
- Package project configurations reproducibly
- Share specifications with integrity guarantees
- Track provenance of project artifacts

---

## Core Concepts

### What is a Bundle?

A governance bundle (`.sbundle.tgz`) is a compressed archive containing:
- **Specification** (`spec.yaml`): Product requirements and features
- **Lock file** (`spec.lock.json`): Versioned dependencies
- **Routing** (`routing.yaml`): Request routing configuration
- **Policies** (`policies/*.yaml`): Security and compliance policies
- **Manifest** (`manifest.yaml`): Bundle metadata and checksums
- **Approvals** (optional): Cryptographic signatures from approvers
- **Attestations** (optional): Sigstore transparency log entries

### Bundle Lifecycle

```
┌─────────────┐      ┌──────────┐      ┌──────────┐      ┌─────────┐
│   Create    │──────│  Approve │──────│  Publish │──────│  Apply  │
│   Bundle    │      │  Bundle  │      │  Bundle  │      │ Bundle  │
└─────────────┘      └──────────┘      └──────────┘      └─────────┘
      │                    │                  │                │
  Build from          Sign with          Push to          Extract &
   project           SSH/GPG            registry          validate
    files              keys                                locally
```

### Security Model

1. **Integrity**: SHA-256 checksums for all files
2. **Authenticity**: Cryptographic signatures (SSH/GPG)
3. **Transparency**: Optional Sigstore attestations
4. **Authorization**: Role-based approval requirements
5. **Non-repudiation**: Immutable audit trail

### Governance Levels

Specular supports four governance maturity levels:

- **L1 (Basic)**: Checksums only, no approvals required
- **L2 (Managed)**: Single role approval (e.g., PM or Lead)
- **L3 (Defined)**: Multi-role approvals (e.g., PM + Security)
- **L4 (Optimized)**: Full attestations with transparency logging

---

## Getting Started

### Prerequisites

```bash
# Install Specular
brew install specular

# Or download from releases
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular
chmod +x specular
sudo mv specular /usr/local/bin/
```

### Quick Start

1. **Initialize a Project**:
```bash
specular init --template basic
cd my-project
```

2. **Create Your First Bundle**:
```bash
specular bundle build \
  --spec .specular/spec.yaml \
  --lock .specular/spec.lock.json \
  --output my-bundle.sbundle.tgz
```

3. **Verify the Bundle**:
```bash
specular bundle verify my-bundle.sbundle.tgz
```

4. **Apply the Bundle**:
```bash
specular bundle apply my-bundle.sbundle.tgz --target ./output
```

### Your First Approval Workflow

1. **Create a Bundle Requiring Approvals**:
```bash
specular bundle build \
  --require-approval pm \
  --require-approval lead \
  --governance-level L3 \
  --output governed-bundle.sbundle.tgz
```

2. **Approve as PM**:
```bash
specular bundle approve governed-bundle.sbundle.tgz \
  --role pm \
  --user alice@example.com \
  --comment "Reviewed product requirements"
```

3. **Approve as Lead**:
```bash
specular bundle approve governed-bundle.sbundle.tgz \
  --role lead \
  --user bob@example.com \
  --comment "Approved technical implementation"
```

4. **Check Approval Status**:
```bash
specular bundle approval-status governed-bundle.sbundle.tgz
```

5. **Verify All Approvals**:
```bash
specular bundle verify governed-bundle.sbundle.tgz
```

---

## Command Reference

### `bundle build` - Create a Bundle

Create a governance bundle from project files.

**Syntax**:
```bash
specular bundle build [flags]
```

**Flags**:
- `--spec <path>`: Path to spec.yaml (default: .specular/spec.yaml)
- `--lock <path>`: Path to spec.lock.json (default: .specular/spec.lock.json)
- `--routing <path>`: Path to routing.yaml (default: .specular/routing.yaml)
- `-p, --policy <path>`: Policy files (can be specified multiple times)
- `-i, --include <path>`: Additional files/directories to include
- `-o, --output <path>`: Output bundle path (default: bundle.sbundle.tgz)
- `-g, --governance-level <level>`: Governance level (L1-L4)
- `-a, --require-approval <role>`: Required approval roles (repeatable)
- `-m, --metadata <key=value>`: Bundle metadata (repeatable)
- `--attest`: Generate Sigstore attestation
- `--attest-format <format>`: Attestation format (sigstore, in-toto, slsa)

**Examples**:

Basic bundle:
```bash
specular bundle build --output basic.sbundle.tgz
```

With policies:
```bash
specular bundle build \
  --policy policies/security.yaml \
  --policy policies/compliance.yaml \
  --output governed.sbundle.tgz
```

With approvals required:
```bash
specular bundle build \
  --require-approval pm \
  --require-approval security \
  --governance-level L3 \
  --output approved.sbundle.tgz
```

With attestation:
```bash
specular bundle build \
  --attest \
  --attest-format slsa \
  --output attested.sbundle.tgz
```

---

### `bundle verify` - Verify Bundle Integrity

Verify a bundle's integrity, signatures, and approvals.

**Syntax**:
```bash
specular bundle verify <bundle-path> [flags]
```

**Flags**:
- `--strict`: Fail on any validation warning
- `--require-approvals`: Verify all required approvals are present
- `--verify-attestation`: Verify Sigstore attestation

**Examples**:

Basic verification:
```bash
specular bundle verify my-bundle.sbundle.tgz
```

Strict verification with approvals:
```bash
specular bundle verify \
  --strict \
  --require-approvals \
  governed-bundle.sbundle.tgz
```

Verify attestation:
```bash
specular bundle verify \
  --verify-attestation \
  attested-bundle.sbundle.tgz
```

**Exit Codes**:
- `0`: Bundle is valid
- `1`: Validation failed
- `2`: Warnings present (strict mode only)

---

### `bundle apply` - Extract and Apply Bundle

Extract a bundle and optionally apply it to a target directory.

**Syntax**:
```bash
specular bundle apply <bundle-path> [flags]
```

**Flags**:
- `--target <path>`: Target directory for extraction (default: current directory)
- `--dry-run`: Show what would be applied without making changes
- `--verify`: Verify bundle before applying
- `--force`: Overwrite existing files

**Examples**:

Dry run:
```bash
specular bundle apply my-bundle.sbundle.tgz --dry-run
```

Apply to specific directory:
```bash
specular bundle apply my-bundle.sbundle.tgz --target ./deployment
```

Verify then apply:
```bash
specular bundle apply my-bundle.sbundle.tgz --verify --target ./prod
```

---

### `bundle approve` - Sign Bundle for Approval

Create a cryptographic approval signature for a bundle.

**Syntax**:
```bash
specular bundle approve <bundle-path> [flags]
```

**Flags**:
- `--role <role>`: Approver role (pm, lead, security, etc.)
- `--user <email>`: Approver identifier (email)
- `--comment <text>`: Approval comment or justification
- `--key-path <path>`: Path to private key
- `--signature-type <type>`: Signature type (ssh, gpg)

**Examples**:

SSH signature (default):
```bash
specular bundle approve my-bundle.sbundle.tgz \
  --role pm \
  --user alice@example.com \
  --comment "Approved for Q4 2025 release"
```

GPG signature:
```bash
specular bundle approve my-bundle.sbundle.tgz \
  --role security \
  --user bob@example.com \
  --signature-type gpg \
  --key-path 299BB654DA4CFDE6
```

With custom key:
```bash
specular bundle approve my-bundle.sbundle.tgz \
  --role lead \
  --user carol@example.com \
  --key-path ~/.ssh/id_ed25519_work
```

---

### `bundle approval-status` - Check Approval Progress

Display approval status and missing approvals for a bundle.

**Syntax**:
```bash
specular bundle approval-status <bundle-path> [flags]
```

**Flags**:
- `--format <format>`: Output format (text, json, yaml)

**Examples**:

Text output:
```bash
specular bundle approval-status my-bundle.sbundle.tgz
```

JSON output:
```bash
specular bundle approval-status my-bundle.sbundle.tgz --format json
```

---

### `bundle diff` - Compare Bundles

Compare two bundles and show their differences.

**Syntax**:
```bash
specular bundle diff <bundle-a> <bundle-b> [flags]
```

**Flags**:
- `--json`: Output differences in JSON format
- `--quiet`: Exit with code 2 if differences found, no output

**Examples**:

Human-readable diff:
```bash
specular bundle diff old.sbundle.tgz new.sbundle.tgz
```

JSON diff:
```bash
specular bundle diff old.sbundle.tgz new.sbundle.tgz --json
```

CI/CD check:
```bash
specular bundle diff old.sbundle.tgz new.sbundle.tgz --quiet
if [ $? -eq 2 ]; then
  echo "Bundles differ - review changes"
  exit 1
fi
```

**Exit Codes**:
- `0`: Bundles are identical
- `1`: Error during comparison
- `2`: Bundles differ (quiet mode)

---

### `bundle push` - Publish to Registry

Push a bundle to an OCI-compatible registry.

**Syntax**:
```bash
specular bundle push <bundle-path> <registry-ref> [flags]
```

**Flags**:
- `--platform <os/arch>`: Target platform (default: linux/amd64)
- `--annotations <key=value>`: OCI annotations (repeatable)

**Examples**:

GitHub Container Registry:
```bash
specular bundle push my-bundle.sbundle.tgz \
  ghcr.io/myorg/my-bundle:v1.0.0
```

Docker Hub:
```bash
specular bundle push my-bundle.sbundle.tgz \
  docker.io/myorg/my-bundle:latest
```

With annotations:
```bash
specular bundle push my-bundle.sbundle.tgz \
  ghcr.io/myorg/my-bundle:v1.0.0 \
  --annotations org.opencontainers.image.description="Production bundle" \
  --annotations org.opencontainers.image.version="1.0.0"
```

---

### `bundle pull` - Download from Registry

Pull a bundle from an OCI-compatible registry.

**Syntax**:
```bash
specular bundle pull <registry-ref> [flags]
```

**Flags**:
- `-o, --output <path>`: Output path (default: inferred from ref)
- `--platform <os/arch>`: Pull specific platform

**Examples**:

Pull latest:
```bash
specular bundle pull ghcr.io/myorg/my-bundle:latest
```

Pull specific version:
```bash
specular bundle pull ghcr.io/myorg/my-bundle:v1.0.0 \
  --output production.sbundle.tgz
```

---

## Workflow Examples

### Example 1: Single Approver Workflow (L2)

**Scenario**: PM approves all releases

1. **Build bundle requiring PM approval**:
```bash
specular bundle build \
  --require-approval pm \
  --governance-level L2 \
  --output release-v1.0.sbundle.tgz
```

2. **PM reviews and approves**:
```bash
specular bundle approve release-v1.0.sbundle.tgz \
  --role pm \
  --user pm@company.com \
  --comment "Approved for production release"
```

3. **Verify and publish**:
```bash
specular bundle verify release-v1.0.sbundle.tgz
specular bundle push release-v1.0.sbundle.tgz \
  ghcr.io/company/product:v1.0
```

---

### Example 2: Multi-Approver Workflow (L3)

**Scenario**: PM and Security must both approve

1. **Build bundle requiring multiple approvals**:
```bash
specular bundle build \
  --require-approval pm \
  --require-approval security \
  --governance-level L3 \
  --output secure-release.sbundle.tgz
```

2. **PM approves first**:
```bash
specular bundle approve secure-release.sbundle.tgz \
  --role pm \
  --user pm@company.com \
  --comment "Product requirements validated"
```

3. **Check status (1 of 2 approvals)**:
```bash
specular bundle approval-status secure-release.sbundle.tgz
# Output shows PM approved, Security pending
```

4. **Security approves**:
```bash
specular bundle approve secure-release.sbundle.tgz \
  --role security \
  --user security@company.com \
  --comment "Security review passed"
```

5. **Verify all approvals present**:
```bash
specular bundle verify --require-approvals secure-release.sbundle.tgz
```

---

### Example 3: CI/CD Integration

**GitHub Actions Workflow**:

```yaml
name: Build and Publish Bundle

on:
  push:
    tags:
      - 'v*'

jobs:
  build-bundle:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Specular
        run: |
          curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular
          chmod +x specular
          sudo mv specular /usr/local/bin/

      - name: Build Bundle
        run: |
          specular bundle build \
            --require-approval pm \
            --governance-level L2 \
            --output bundle.sbundle.tgz

      - name: Upload Bundle Artifact
        uses: actions/upload-artifact@v3
        with:
          name: bundle
          path: bundle.sbundle.tgz

  approve-bundle:
    needs: build-bundle
    runs-on: ubuntu-latest
    environment: production  # Requires manual approval in GitHub
    steps:
      - uses: actions/download-artifact@v3
        with:
          name: bundle

      - name: PM Approval
        run: |
          specular bundle approve bundle.sbundle.tgz \
            --role pm \
            --user ${{ secrets.PM_EMAIL }} \
            --comment "Automated approval for ${GITHUB_REF_NAME}"
        env:
          SSH_PRIVATE_KEY: ${{ secrets.PM_SSH_KEY }}

      - name: Verify Bundle
        run: specular bundle verify --require-approvals bundle.sbundle.tgz

  publish-bundle:
    needs: approve-bundle
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v3
        with:
          name: bundle

      - name: Login to GHCR
        run: echo "${{ secrets.GITHUB_TOKEN }}" | docker login ghcr.io -u ${{ github.actor }} --password-stdin

      - name: Push Bundle
        run: |
          specular bundle push bundle.sbundle.tgz \
            ghcr.io/${{ github.repository }}:${GITHUB_REF_NAME}
```

---

### Example 4: Development to Production Pipeline

**Development**:
```bash
# Build dev bundle (no approvals)
specular bundle build \
  --governance-level L1 \
  --output dev.sbundle.tgz

# Apply locally for testing
specular bundle apply dev.sbundle.tgz --target ./test-env
```

**Staging**:
```bash
# Build staging bundle (PM approval)
specular bundle build \
  --require-approval pm \
  --governance-level L2 \
  --output staging.sbundle.tgz

# PM approves
specular bundle approve staging.sbundle.tgz \
  --role pm \
  --user pm@company.com

# Push to staging registry
specular bundle push staging.sbundle.tgz \
  registry.company.com/app:staging
```

**Production**:
```bash
# Build production bundle (PM + Security)
specular bundle build \
  --require-approval pm \
  --require-approval security \
  --governance-level L3 \
  --attest \
  --output production.sbundle.tgz

# PM approves
specular bundle approve production.sbundle.tgz \
  --role pm \
  --user pm@company.com \
  --comment "Prod release Q4 2025"

# Security approves
specular bundle approve production.sbundle.tgz \
  --role security \
  --user security@company.com \
  --comment "Security scan passed"

# Verify everything
specular bundle verify \
  --require-approvals \
  --verify-attestation \
  production.sbundle.tgz

# Push to production registry
specular bundle push production.sbundle.tgz \
  registry.company.com/app:v1.0.0
```

---

## Best Practices

### Bundle Creation

1. **Use Semantic Versioning**: Tag bundles with semver (v1.0.0, v1.1.0-beta)
2. **Include All Dependencies**: Add lock files to ensure reproducibility
3. **Set Appropriate Governance Level**: Match governance to risk level
4. **Add Meaningful Metadata**: Use `--metadata` for context

### Approval Workflows

1. **Define Clear Roles**: Document who can approve what
2. **Require Comments**: Mandate `--comment` for audit trail
3. **Rotate Keys Regularly**: Update SSH/GPG keys quarterly
4. **Separate Environments**: Different approval requirements per environment

### Registry Management

1. **Use Tags Wisely**: `:latest` for development, semantic versions for production
2. **Enable Immutability**: Configure registry to prevent tag overwrites
3. **Implement Retention Policies**: Auto-delete old development bundles
4. **Backup Critical Bundles**: Store production bundles in multiple locations

### Security

1. **Always Verify**: Run `bundle verify` before `bundle apply`
2. **Use Strict Mode**: Enable `--strict` in production pipelines
3. **Enable Attestations**: Use `--attest` for L4 governance
4. **Audit Approvals**: Regularly review approval logs
5. **Protect Private Keys**: Never commit keys to version control

### Performance

1. **Minimize Bundle Size**: Exclude unnecessary files with `.specularignore`
2. **Use Compression**: Bundles are automatically gzip-compressed
3. **Cache Bundles**: Reuse verified bundles when possible
4. **Parallel Operations**: Bundle operations are optimized for concurrency

---

## Troubleshooting

### Common Issues

#### "Missing required approval for role X"

**Problem**: Bundle requires approval that hasn't been provided

**Solution**:
```bash
# Check which approvals are missing
specular bundle approval-status my-bundle.sbundle.tgz

# Add missing approval
specular bundle approve my-bundle.sbundle.tgz \
  --role <missing-role> \
  --user <your-email>
```

#### "Checksum mismatch for file: spec.yaml"

**Problem**: File was modified after bundle creation

**Solution**:
```bash
# If expected, rebuild the bundle
specular bundle build --output new-bundle.sbundle.tgz

# If unexpected, investigate tampering
specular bundle diff original.sbundle.tgz current.sbundle.tgz
```

#### "Failed to verify signature"

**Problem**: SSH/GPG signature verification failed

**Solution**:
```bash
# Check public key matches
ssh-keygen -l -f ~/.ssh/id_ed25519.pub

# Verify key is in approval
specular bundle approval-status my-bundle.sbundle.tgz --format json | jq '.approvals[].public_key_fingerprint'

# Re-approve with correct key
specular bundle approve my-bundle.sbundle.tgz \
  --role <role> \
  --user <email> \
  --key-path ~/.ssh/id_ed25519
```

#### "Bundle not found in registry"

**Problem**: Bundle doesn't exist or incorrect reference

**Solution**:
```bash
# Check if bundle was pushed
docker manifest inspect ghcr.io/myorg/bundle:v1.0.0

# Verify authentication
docker login ghcr.io

# Try alternate tag
specular bundle pull ghcr.io/myorg/bundle:latest
```

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
# Verbose output
specular bundle build --verbose

# Debug logging
specular --log-level debug bundle verify my-bundle.sbundle.tgz

# Trace ID for distributed debugging
specular --trace abc123 bundle apply my-bundle.sbundle.tgz
```

---

## Advanced Topics

### Custom Governance Levels

Define custom governance requirements beyond L1-L4:

```yaml
# .specular/governance.yaml
levels:
  custom-high:
    required_approvals:
      - pm
      - lead
      - security
      - compliance
    require_attestation: true
    require_rekor_entry: true
    max_approval_age: 24h
```

### Approval Policies

Create organization-wide approval policies:

```yaml
# policies/approval-policy.yaml
approval_policy:
  roles:
    pm:
      description: "Product Manager"
      trusted_keys:
        - "SHA256:abc123..."
    security:
      description: "Security Team"
      required_for:
        - production
        - staging
      max_age: 168h  # 1 week
```

### Webhook Integration

Set up webhooks for approval events:

```bash
# Configure webhook endpoint
specular config set approval.webhook.url https://api.company.com/approvals

# Enable webhook notifications
specular bundle approve my-bundle.sbundle.tgz \
  --role pm \
  --user pm@company.com \
  --notify-webhook
```

### Bundle Signing with Hardware Keys

Use YubiKey or other hardware security modules:

```bash
# SSH with hardware key
ssh-keygen -K

# Approve with hardware key
specular bundle approve my-bundle.sbundle.tgz \
  --role security \
  --user sec@company.com \
  --key-path ~/.ssh/id_ed25519_sk
```

---

## Additional Resources

- **Specification Reference**: `docs/SPEC_FORMAT.md`
- **Security Audit**: `docs/SECURITY_AUDIT.md`
- **API Documentation**: `docs/API.md`
- **Contributing Guide**: `CONTRIBUTING.md`

---

**Need Help?**
- GitHub Issues: https://github.com/felixgeelhaar/specular/issues
- Documentation: https://docs.specular.dev
- Community: https://discord.gg/specular

**Last Updated**: 2025-11-08
**Version**: 1.3.0
