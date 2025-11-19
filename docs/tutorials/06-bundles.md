# Bundles: Portable Governance Packages

This tutorial covers creating, verifying, and distributing governance bundles.

> **Note**: Bundle gate, inspect, and list commands require Specular Pro or Enterprise license.

## Overview

Governance bundles (`.sbundle.tgz`) package:
- Product specifications
- Locked dependencies
- AI provider routing
- Governance policies
- Approval signatures
- Cryptographic attestations

---

## Prerequisites

- Completed [Governance](./04-governance.md) tutorial
- Completed [Policy Management](./05-policy-management.md) tutorial
- Project with spec.yaml and policies

---

## Step 1: Create a Bundle

Package your governance artifacts:

```bash
specular bundle create my-app-v1.0.0.sbundle.tgz
```

Output:

```
Creating governance bundle...

✓ Bundle created successfully: my-app-v1.0.0.sbundle.tgz (0.45 MB)

Bundle Details:
  ID:      bundle-a1b2c3d4
  Version: 1.0
  Schema:  specular.io/bundle/v1
  Created: 2024-01-15 10:30:00
  Digest:  sha256:e5f6g7h8...
```

### Specify Components

```bash
specular bundle create \
  --spec .specular/spec.yaml \
  --lock .specular/spec.lock.json \
  --routing .specular/routing.yaml \
  --policy policies/security.yaml \
  --policy policies/compliance.yaml \
  bundle.sbundle.tgz
```

### Add Custom Files

```bash
specular bundle create \
  --include README.md \
  --include configs/production.yaml \
  bundle.sbundle.tgz
```

### Set Governance Level

```bash
specular bundle create \
  --governance-level L3 \
  bundle.sbundle.tgz
```

Governance levels:
- **L1**: Basic - Minimal governance
- **L2**: Standard - Policies and drift detection
- **L3**: Strict - Approvals and attestations required
- **L4**: Enterprise - Full compliance controls

---

## Step 2: Generate Attestation

Add cryptographic attestation to your bundle:

```bash
specular bundle create \
  --attest \
  --attest-format sigstore \
  bundle.sbundle.tgz
```

Supported formats:
- `sigstore` - Sigstore transparency log
- `in-toto` - in-toto attestation
- `slsa` - SLSA provenance

---

## Step 3: Inspect a Bundle (PRO)

View bundle contents and metadata:

```bash
specular bundle inspect my-app-v1.0.0.sbundle.tgz
```

Output:

```
Inspecting bundle: my-app-v1.0.0.sbundle.tgz

=== Bundle Metadata ===
ID:               bundle-a1b2c3d4
Version:          1.0
Schema:           specular.io/bundle/v1
Created:          2024-01-15 10:30:00
Governance Level: L3
Integrity Digest: sha256:e5f6g7h8...

=== Files (5) ===
  spec.yaml
    Size:     2048 bytes
    Checksum: sha256:...
  spec.lock.json
    Size:     4096 bytes
    Checksum: sha256:...
  routing.yaml
    Size:     1024 bytes
    Checksum: sha256:...
  policies/security.yaml
    Size:     512 bytes
    Checksum: sha256:...
  policies/compliance.yaml
    Size:     768 bytes
    Checksum: sha256:...

=== Approvals (2) ===
  Role: pm
    User:      alice@company.com
    Signed At: 2024-01-15 09:00:00
    Signature: ssh
    Comment:   Approved for Q1 release

  Role: lead
    User:      bob@company.com
    Signed At: 2024-01-15 09:30:00
    Signature: ssh

=== Attestation ===
Format:    sigstore
Timestamp: 2024-01-15 10:30:00
```

### JSON Output

```bash
specular bundle inspect --json bundle.sbundle.tgz
```

---

## Step 4: Gate a Bundle (PRO)

Verify bundle integrity and compliance:

```bash
specular bundle gate my-app-v1.0.0.sbundle.tgz
```

Output:

```
Running governance gate checks on: my-app-v1.0.0.sbundle.tgz

✓ Bundle gate check PASSED

Checksum Validation:    ✓ PASS
Approval Validation:    ✓ PASS
Attestation Validation: ✓ PASS
Policy Compliance:      ✓ PASS
```

### Strict Gate Check

```bash
specular bundle gate --strict bundle.sbundle.tgz
```

### Require Approvals

```bash
specular bundle gate --require-approvals bundle.sbundle.tgz
```

### Verify Attestation

```bash
specular bundle gate --verify-attestation bundle.sbundle.tgz
```

### Gate Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Passed all checks |
| 20 | Policy violation |
| 30 | Drift detected |
| 40 | Missing approval |
| 50 | Forbidden provider |
| 60 | Evaluation failure |

---

## Step 5: List Available Bundles (PRO)

View bundles in your workspace:

```bash
specular bundle list
```

Output:

```
=== Bundles in .specular/bundles ===

📦 my-app-v1.0.0.sbundle.tgz
   ID:         bundle-a1b2c3d4
   Size:       0.45 MB
   Modified:   2024-01-15 10:30:00
   Gov Level:  L3
   Approvals:  2

📦 my-app-v0.9.0.sbundle.tgz
   ID:         bundle-x1y2z3a4
   Size:       0.42 MB
   Modified:   2024-01-10 14:00:00
   Gov Level:  L2
   Approvals:  1

Total: 2 bundle(s)
```

### List from Custom Directory

```bash
specular bundle list --dir /path/to/bundles
```

---

## Step 6: Apply a Bundle

Extract and apply bundle to a project:

```bash
# Preview changes (dry run)
specular bundle apply --dry-run bundle.sbundle.tgz

# Apply to current directory
specular bundle apply bundle.sbundle.tgz

# Apply to specific directory
specular bundle apply --target-dir /path/to/project bundle.sbundle.tgz

# Force overwrite
specular bundle apply --force bundle.sbundle.tgz
```

---

## Step 7: Compare Bundles

View differences between bundle versions:

```bash
specular bundle diff v0.9.0.sbundle.tgz v1.0.0.sbundle.tgz
```

Output:

```
Comparing bundles:
  A: v0.9.0.sbundle.tgz
  B: v1.0.0.sbundle.tgz

Loading bundles...

Differences found:

Files Modified (2):
  M spec.yaml
    Old: sha256:a1b2c3...
    New: sha256:d4e5f6...
  M policies/security.yaml
    Old: sha256:g7h8i9...
    New: sha256:j0k1l2...

Approvals Added (1):
  + Role: security, User: charlie@company.com

Summary: 0 added, 0 removed, 2 modified files; 1 approval added
```

---

## Step 8: Push to Registry

Upload bundle to an OCI registry:

```bash
# Push to GitHub Container Registry
specular bundle push \
  my-app-v1.0.0.sbundle.tgz \
  ghcr.io/org/my-app:v1.0.0

# Push to Docker Hub
specular bundle push \
  bundle.sbundle.tgz \
  docker.io/username/bundle:latest

# Push to private registry
specular bundle push \
  --insecure \
  bundle.sbundle.tgz \
  localhost:5000/bundle:test
```

---

## Step 9: Pull from Registry

Download bundle from registry:

```bash
# Pull from GitHub Container Registry
specular bundle pull ghcr.io/org/my-app:v1.0.0

# Pull with custom output path
specular bundle pull \
  ghcr.io/org/my-app:v1.0.0 \
  my-app-v1.0.0.sbundle.tgz

# Pull from insecure registry
specular bundle pull --insecure localhost:5000/bundle:test
```

---

## CI/CD Integration

### GitHub Actions Workflow

```yaml
name: Governance Bundle

on:
  push:
    tags:
      - 'v*'

jobs:
  build-bundle:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Create Bundle
        run: |
          specular bundle create \
            --attest \
            --governance-level L3 \
            my-app-${{ github.ref_name }}.sbundle.tgz

      - name: Gate Bundle
        run: |
          specular bundle gate \
            --strict \
            --require-approvals \
            my-app-${{ github.ref_name }}.sbundle.tgz

      - name: Push to Registry
        run: |
          specular bundle push \
            my-app-${{ github.ref_name }}.sbundle.tgz \
            ghcr.io/${{ github.repository }}:${{ github.ref_name }}
```

### GitLab CI

```yaml
bundle:
  stage: build
  script:
    - specular bundle create bundle.sbundle.tgz
    - specular bundle gate --strict bundle.sbundle.tgz
  artifacts:
    paths:
      - bundle.sbundle.tgz

deploy:
  stage: deploy
  script:
    - specular bundle apply --yes bundle.sbundle.tgz
  dependencies:
    - bundle
```

---

## Best Practices

### 1. Version Your Bundles

Use semantic versioning in bundle names:

```bash
specular bundle create my-app-v1.2.3.sbundle.tgz
```

### 2. Include Metadata

Add custom metadata for tracking:

```bash
specular bundle create \
  --metadata version=1.2.3 \
  --metadata commit=$(git rev-parse HEAD) \
  --metadata author="$(git config user.email)" \
  bundle.sbundle.tgz
```

### 3. Always Gate Before Apply

Verify bundles before deploying:

```bash
specular bundle gate --strict bundle.sbundle.tgz && \
specular bundle apply bundle.sbundle.tgz
```

### 4. Use Registry for Distribution

Push bundles to OCI registries for team access:

```bash
specular bundle push bundle.sbundle.tgz ghcr.io/org/project:v1.0.0
```

### 5. Keep Bundle History

List bundles regularly and archive old versions:

```bash
specular bundle list --json > bundle-inventory.json
```

---

## Command Reference

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `bundle create` | Create bundle | `--spec`, `--policy`, `--attest`, `--governance-level` |
| `bundle gate` | Verify bundle (PRO) | `--strict`, `--require-approvals`, `--verify-attestation` |
| `bundle inspect` | View contents (PRO) | `--json` |
| `bundle list` | List bundles (PRO) | `--dir`, `--json` |
| `bundle apply` | Apply bundle | `--dry-run`, `--force`, `--target-dir` |
| `bundle diff` | Compare bundles | `--json`, `--quiet` |
| `bundle push` | Push to registry | `--insecure`, `--platform` |
| `bundle pull` | Pull from registry | `--insecure`, `--output` |

---

## Next Steps

- [Approvals](./07-approvals.md) - Signing and verifying approvals
- [CLI Reference](../CLI_REFERENCE.md) - Complete command reference
- [Production Guide](../PRODUCTION_GUIDE.md) - Deployment best practices
