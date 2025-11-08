# Registry Publishing Examples

This directory contains practical examples of publishing Specular bundles to various container registries.

## Overview

Specular bundles can be published to OCI-compliant container registries just like container images. These examples demonstrate how to publish bundles to:

- GitHub Container Registry (GHCR)
- Docker Hub
- Private registries

## Prerequisites

- Specular CLI installed
- Container registry account
- Registry authentication configured
- Bundle file ready to publish

## Supported Registries

### GitHub Container Registry (GHCR)
- **URL**: `ghcr.io`
- **Authentication**: GitHub Personal Access Token (PAT)
- **Pricing**: Free for public repositories
- **Features**: Tight GitHub integration, org-level packages

### Docker Hub
- **URL**: `docker.io` or `registry.hub.docker.com`
- **Authentication**: Username and password/token
- **Pricing**: Free tier available
- **Features**: Popular, well-established registry

### AWS Elastic Container Registry (ECR)
- **URL**: `<aws-account-id>.dkr.ecr.<region>.amazonaws.com`
- **Authentication**: AWS credentials
- **Pricing**: Pay per GB stored
- **Features**: AWS integration, cross-region replication

### Google Container Registry (GCR)
- **URL**: `gcr.io`
- **Authentication**: Google Cloud credentials
- **Pricing**: Pay per GB stored
- **Features**: GCP integration, multi-region support

### Azure Container Registry (ACR)
- **URL**: `<registry-name>.azurecr.io`
- **Authentication**: Azure credentials
- **Pricing**: Multiple tiers
- **Features**: Azure integration, geo-replication

## Examples

### 1. Publish to GHCR

**File**: `publish-to-ghcr.sh`

**Features**:
- Authenticates with GitHub token
- Publishes bundle to GHCR
- Tags with version and latest
- Supports both public and private packages

**Usage**:
```bash
export GITHUB_TOKEN="your-github-token"
./publish-to-ghcr.sh release.sbundle.tgz username/repo v1.0.0
```

### 2. Publish to Docker Hub

**File**: `publish-to-dockerhub.sh`

**Features**:
- Authenticates with Docker Hub credentials
- Publishes bundle to Docker Hub
- Tags with semantic versioning
- Supports both public and private repositories

**Usage**:
```bash
export DOCKERHUB_USERNAME="your-username"
export DOCKERHUB_TOKEN="your-token"
./publish-to-dockerhub.sh release.sbundle.tgz username/repo v1.0.0
```

## Authentication

### GitHub Container Registry

**Using Personal Access Token (PAT)**:
```bash
# Create PAT with write:packages scope
# https://github.com/settings/tokens

# Login
echo "$GITHUB_TOKEN" | docker login ghcr.io -u USERNAME --password-stdin

# Or use Specular's built-in auth
specular bundle push release.sbundle.tgz ghcr.io/user/repo:tag \
  --username USERNAME \
  --password "$GITHUB_TOKEN"
```

**Using GITHUB_TOKEN in Actions**:
```yaml
- name: Login to GHCR
  uses: docker/login-action@v3
  with:
    registry: ghcr.io
    username: ${{ github.actor }}
    password: ${{ secrets.GITHUB_TOKEN }}
```

### Docker Hub

**Using Access Token**:
```bash
# Create access token at https://hub.docker.com/settings/security

# Login
echo "$DOCKERHUB_TOKEN" | docker login -u USERNAME --password-stdin

# Or use Specular
specular bundle push release.sbundle.tgz docker.io/user/repo:tag \
  --username USERNAME \
  --password "$DOCKERHUB_TOKEN"
```

### AWS ECR

**Using AWS CLI**:
```bash
# Get login password
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin \
  123456789012.dkr.ecr.us-east-1.amazonaws.com

# Push bundle
specular bundle push release.sbundle.tgz \
  123456789012.dkr.ecr.us-east-1.amazonaws.com/repo:tag
```

### Google GCR

**Using gcloud**:
```bash
# Configure Docker to use gcloud credentials
gcloud auth configure-docker

# Push bundle
specular bundle push release.sbundle.tgz gcr.io/project-id/repo:tag
```

## Versioning Strategies

### Semantic Versioning
```bash
# Tag with semantic version
specular bundle push bundle.sbundle.tgz registry.io/org/app:v1.2.3

# Tag with major version
specular bundle push bundle.sbundle.tgz registry.io/org/app:v1

# Tag as latest
specular bundle push bundle.sbundle.tgz registry.io/org/app:latest
```

### Git-based Versioning
```bash
# Tag with git commit SHA
SHA=$(git rev-parse --short HEAD)
specular bundle push bundle.sbundle.tgz registry.io/org/app:$SHA

# Tag with git tag
TAG=$(git describe --tags --abbrev=0)
specular bundle push bundle.sbundle.tgz registry.io/org/app:$TAG
```

### Environment-based Tagging
```bash
# Tag for development
specular bundle push bundle.sbundle.tgz registry.io/org/app:dev

# Tag for staging
specular bundle push bundle.sbundle.tgz registry.io/org/app:staging

# Tag for production
specular bundle push bundle.sbundle.tgz registry.io/org/app:prod
```

## Multi-Registry Publishing

Publish to multiple registries for redundancy:

```bash
#!/bin/bash
BUNDLE="release.sbundle.tgz"
VERSION="v1.0.0"

# Publish to GHCR
specular bundle push "$BUNDLE" "ghcr.io/org/app:$VERSION"

# Publish to Docker Hub
specular bundle push "$BUNDLE" "docker.io/org/app:$VERSION"

# Publish to private registry
specular bundle push "$BUNDLE" "registry.company.com/app:$VERSION"
```

## Pulling Bundles

### From GHCR
```bash
specular bundle pull ghcr.io/user/repo:v1.0.0
```

### From Docker Hub
```bash
specular bundle pull docker.io/user/repo:v1.0.0
```

### From Private Registry
```bash
# Login first
docker login registry.company.com

# Pull bundle
specular bundle pull registry.company.com/app:v1.0.0
```

## Best Practices

### 1. Tagging Strategy
- Always tag with semantic version (`v1.2.3`)
- Maintain a `latest` tag for the most recent stable release
- Use environment tags (`dev`, `staging`, `prod`) for deployment tracking
- Include git commit SHA in tags for traceability

### 2. Access Control
- Use least privilege access for registry authentication
- Rotate credentials regularly
- Use short-lived tokens in CI/CD
- Enable registry audit logging

### 3. Registry Organization
- Use consistent naming conventions (e.g., `org/product/component`)
- Separate public and private packages
- Implement lifecycle policies to clean up old bundles
- Document registry structure

### 4. Security
- Scan bundles for vulnerabilities before publishing
- Enable vulnerability scanning in registries
- Sign bundles with attestations
- Use private registries for sensitive bundles

### 5. Performance
- Use registry mirrors for frequently accessed bundles
- Enable caching layers
- Consider registry location relative to deployment targets
- Implement rate limiting

## Troubleshooting

### Authentication Failures
```bash
# Verify credentials
docker login registry.io

# Check token permissions
# - GHCR requires write:packages scope
# - Docker Hub requires appropriate repository access

# Test with docker
docker pull registry.io/org/test:latest
```

### Push Failures
```bash
# Check bundle validity
specular bundle verify bundle.sbundle.tgz

# Verify registry connectivity
curl -I https://registry.io/v2/

# Check rate limits
# GHCR: 5000 requests/hour
# Docker Hub: 100-200 pulls/6 hours (free tier)
```

### Pull Failures
```bash
# Verify tag exists
# For GHCR: https://github.com/orgs/ORG/packages/container/PACKAGE
# For Docker Hub: https://hub.docker.com/r/user/repo/tags

# Check authentication
specular bundle pull registry.io/org/app:tag --username USER --password PASS
```

## Advanced Topics

### 1. Registry Mirroring

Mirror bundles across registries:
```bash
# Pull from source
specular bundle pull source.io/org/app:v1.0.0

# Push to mirror
specular bundle push app-v1.0.0.sbundle.tgz mirror.io/org/app:v1.0.0
```

### 2. Automated Publishing

GitHub Actions example:
```yaml
- name: Publish bundle
  run: |
    specular bundle push bundle.sbundle.tgz \
      ghcr.io/${{ github.repository }}:${{ github.sha }}
  env:
    REGISTRY_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### 3. Lifecycle Management

Clean up old bundles:
```bash
# Delete tags older than 30 days
gh api -X DELETE /user/packages/container/APP/versions/VERSION_ID
```

### 4. Private Registry Setup

Run your own registry:
```bash
# Run registry container
docker run -d -p 5000:5000 --name registry registry:2

# Push to local registry
specular bundle push bundle.sbundle.tgz localhost:5000/app:v1.0.0
```

## Further Reading

- [Bundle User Guide](../../docs/BUNDLE_USER_GUIDE.md)
- [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec)
- [Docker Registry API](https://docs.docker.com/registry/spec/api/)
- [GHCR Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Docker Hub Documentation](https://docs.docker.com/docker-hub/)
