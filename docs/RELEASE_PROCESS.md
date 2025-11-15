# Specular Release Process

This document describes the release process for Specular, including versioning, building, testing, and distribution across multiple package managers.

## Table of Contents

- [Versioning Strategy](#versioning-strategy)
- [Pre-Release Checklist](#pre-release-checklist)
- [Release Process](#release-process)
- [Post-Release Verification](#post-release-verification)
- [Rollback Procedure](#rollback-procedure)
- [Distribution Channels](#distribution-channels)

---

## Versioning Strategy

Specular follows [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH

v1.4.0
│ │ │
│ │ └─ Patch version (backward-compatible bug fixes)
│ └───  Minor version (backward-compatible features)
└───── Major version (breaking changes)
```

### Version Increment Rules

- **MAJOR**: Breaking changes to CLI interface, API, or configuration format
- **MINOR**: New features, enhancements (backward-compatible)
- **PATCH**: Bug fixes, documentation updates (backward-compatible)

### Examples

- `v1.4.0` → `v1.4.1`: Bug fix (patch)
- `v1.4.1` → `v1.5.0`: New feature (minor)
- `v1.5.0` → `v2.0.0`: Breaking change (major)

---

## Pre-Release Checklist

Before creating a new release, ensure all items are complete:

### Code Quality

- [ ] All tests passing (`go test ./...`)
- [ ] Integration tests passing (`go test -tags=integration ./...`)
- [ ] Linters passing (`golangci-lint run`)
- [ ] No critical security issues (`gosec ./...`)
- [ ] Code coverage ≥ 80%

### Documentation

- [ ] CHANGELOG.md updated with release notes
- [ ] README.md reflects new features
- [ ] Docs updated for new commands/flags
- [ ] Breaking changes documented (if any)
- [ ] Migration guide written (for breaking changes)

### Dependencies

- [ ] Dependencies updated (`go get -u ./...`)
- [ ] `go mod tidy` run
- [ ] Vulnerability scan passing (`go list -json -m all | nancy sleuth`)

### Configuration

- [ ] `.goreleaser.yml` validated (`goreleaser check`)
- [ ] Shell completions generated
- [ ] Docker images build successfully
- [ ] Homebrew formula syntax valid

### Testing Across Platforms

- [ ] Linux (amd64, arm64)
- [ ] macOS (amd64, arm64)
- [ ] Windows (amd64)

### Security

- [ ] API keys and secrets not committed
- [ ] GPG_FINGERPRINT set for signing (production)
- [ ] SBOM generation enabled
- [ ] Security scan reports clean

---

## Release Process

### 1. Prepare Release Branch

```bash
# Ensure you're on main branch
git checkout main
git pull origin main

# Create release branch
git checkout -b release/v1.5.0

# Update version in code (if needed)
# internal/version/version.go

# Update CHANGELOG.md
cat >> CHANGELOG.md <<'EOF'
## [1.5.0] - 2025-01-15

### Added
- Interactive prompts for missing required flags
- Enhanced error messages with recovery suggestions
- Docker image caching for faster builds
- Production deployment guide

### Changed
- Improved flag defaults and descriptions

### Fixed
- Cache hit rate calculation
- Policy validation error messages

EOF

# Commit changes
git add CHANGELOG.md
git commit -m "chore: prepare v1.5.0 release"
```

### 2. Create and Push Tag

```bash
# Create annotated tag
git tag -a v1.5.0 -m "$(cat <<'EOF'
Release v1.5.0 - Enhanced UX and Production Readiness

### Highlights
- 🎯 Interactive prompts for better UX
- 📚 Production deployment guide
- 🐳 Docker image caching
- 🔧 Enhanced error messages

See CHANGELOG.md for complete details.
EOF
)"

# Verify tag
git tag -v v1.5.0

# Push tag (triggers CI/CD release)
git push origin v1.5.0
```

### 3. Automated Release (GitHub Actions)

The GitHub Actions workflow automatically:

1. **Runs tests** - Full test suite with coverage
2. **Builds binaries** - Multi-platform builds
3. **Generates completions** - Shell completions for bash/zsh/fish
4. **Creates packages** - .deb, .rpm, .apk for Linux
5. **Builds Docker images** - Multi-arch images (amd64, arm64)
6. **Updates Homebrew** - Updates felixgeelhaar/homebrew-tap
7. **Generates SBOM** - Software Bill of Materials
8. **Signs artifacts** - GPG signatures (if configured)
9. **Creates GitHub Release** - With changelog and assets

**GitHub Actions workflow:**
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ secrets.GPG_FINGERPRINT }}
```

### 4. Manual Release (Local Testing)

For testing the release process locally:

```bash
# Test release with snapshot (no publishing)
goreleaser release --snapshot --clean

# Verify artifacts in dist/
ls -lh dist/

# Test binaries
./dist/specular_linux_amd64/specular version
./dist/specular_darwin_arm64/specular version

# Test packages
dpkg -c dist/specular_1.5.0_linux_amd64.deb
rpm -qpR dist/specular_1.5.0_linux_amd64.rpm

# Test Docker image
docker load < dist/specular_1.5.0_linux_amd64.tar.gz
docker run --rm specular:v1.5.0 version
```

### 5. Publish Release

```bash
# Publish release (requires proper credentials)
export GITHUB_TOKEN="ghp_..."
export TAP_GITHUB_TOKEN="ghp_..."
export GPG_FINGERPRINT="ABC123..."

goreleaser release --clean
```

---

## Post-Release Verification

### 1. Verify GitHub Release

- [ ] Release created at https://github.com/felixgeelhaar/specular/releases/v1.5.0
- [ ] Changelog visible in release notes
- [ ] All artifacts present (binaries, packages, checksums)
- [ ] SBOM file generated
- [ ] GPG signatures present (if configured)

### 2. Verify Package Managers

#### Homebrew (macOS/Linux)

```bash
# Update tap
brew update

# Check version
brew info felixgeelhaar/tap/specular

# Install new version
brew upgrade felixgeelhaar/tap/specular

# Verify installation
specular version
```

#### Debian/Ubuntu (.deb)

```bash
# Download package
wget https://github.com/felixgeelhaar/specular/releases/download/v1.5.0/specular_1.5.0_linux_amd64.deb

# Install
sudo dpkg -i specular_1.5.0_linux_amd64.deb

# Verify
specular version
dpkg -l | grep specular
```

#### RedHat/Fedora (.rpm)

```bash
# Download package
wget https://github.com/felixgeelhaar/specular/releases/download/v1.5.0/specular_1.5.0_linux_amd64.rpm

# Install
sudo rpm -i specular_1.5.0_linux_amd64.rpm

# Verify
specular version
rpm -qa | grep specular
```

#### Alpine Linux (.apk)

```bash
# Download package
wget https://github.com/felixgeelhaar/specular/releases/download/v1.5.0/specular_1.5.0_linux_amd64.apk

# Install
sudo apk add --allow-untrusted specular_1.5.0_linux_amd64.apk

# Verify
specular version
apk info specular
```

#### Docker (ghcr.io)

```bash
# Pull latest
docker pull ghcr.io/felixgeelhaar/specular:latest
docker pull ghcr.io/felixgeelhaar/specular:v1.5.0
docker pull ghcr.io/felixgeelhaar/specular:v1.5
docker pull ghcr.io/felixgeelhaar/specular:v1

# Verify version
docker run --rm ghcr.io/felixgeelhaar/specular:v1.5.0 version

# Verify multi-arch
docker manifest inspect ghcr.io/felixgeelhaar/specular:v1.5.0
```

### 3. Verify Shell Completions

```bash
# Bash
source <(specular completion bash)
specular <TAB><TAB>

# Zsh
autoload -U compinit && compinit
specular <TAB><TAB>

# Fish
specular <TAB><TAB>
```

### 4. Smoke Tests

Run basic smoke tests to ensure the release works:

```bash
# Version check
specular version

# Doctor check
specular doctor --format json

# Provider health
specular provider health

# Simple generation
export ANTHROPIC_API_KEY="..."
specular generate "Hello world" --provider anthropic

# Auto mode dry-run
specular auto "Build a hello world API" --dry-run --profile ci
```

---

## Rollback Procedure

If a release has critical issues, follow this rollback process:

### 1. Delete Git Tag

```bash
# Delete local tag
git tag -d v1.5.0

# Delete remote tag
git push --delete origin v1.5.0
```

### 2. Delete GitHub Release

```bash
# Using GitHub CLI
gh release delete v1.5.0 --yes

# Or delete manually in GitHub UI
```

### 3. Revert Homebrew Formula

```bash
# Clone tap repository
git clone https://github.com/felixgeelhaar/homebrew-tap.git
cd homebrew-tap

# Revert formula commit
git revert <commit-hash>
git push origin main
```

### 4. Notify Users

Create a GitHub issue announcing the rollback:

```markdown
## Release v1.5.0 Rollback Notice

We've rolled back release v1.5.0 due to [critical issue description].

**Affected platforms:**
- Homebrew
- Linux packages (.deb, .rpm, .apk)
- Docker images

**Recommended action:**
If you installed v1.5.0, please downgrade to v1.4.0:

\`\`\`bash
# Homebrew
brew uninstall specular
brew install felixgeelhaar/tap/specular

# Verify version
specular version  # Should show v1.4.0
\`\`\`

We'll release a fixed version (v1.5.1) shortly.
```

### 5. Prepare Hotfix Release

```bash
# Create hotfix branch from previous tag
git checkout -b hotfix/v1.5.1 v1.4.0

# Apply fixes
git cherry-pick <fix-commit>

# Create new release
git tag -a v1.5.1 -m "Hotfix release v1.5.1"
git push origin v1.5.1
```

---

## Distribution Channels

### GitHub Releases

**URL:** https://github.com/felixgeelhaar/specular/releases

**Artifacts:**
- Source code (zip, tar.gz)
- Binary archives (.tar.gz, .zip)
- Linux packages (.deb, .rpm, .apk)
- Checksums (sha256)
- SBOM (SPDX JSON)
- GPG signatures

### Homebrew Tap

**Tap:** `felixgeelhaar/tap`
**Formula:** `specular`

```bash
brew tap felixgeelhaar/tap
brew install specular
```

**Repository:** https://github.com/felixgeelhaar/homebrew-tap

### Docker Registry

**Registry:** GitHub Container Registry (ghcr.io)
**Repository:** `ghcr.io/felixgeelhaar/specular`

**Tags:**
- `latest` - Latest release
- `v1` - Latest v1.x.x
- `v1.5` - Latest v1.5.x
- `v1.5.0` - Specific version
- `<tag>-amd64` - AMD64 architecture
- `<tag>-arm64` - ARM64 architecture

**Visibility:** Public

### Linux Package Repositories

#### Future: APT Repository (Debian/Ubuntu)

```bash
# Add repository
echo "deb [trusted=yes] https://apt.specular.io stable main" | \
  sudo tee /etc/apt/sources.list.d/specular.list

# Update and install
sudo apt update
sudo apt install specular
```

#### Future: YUM Repository (RHEL/CentOS/Fedora)

```bash
# Add repository
sudo tee /etc/yum.repos.d/specular.repo <<EOF
[specular]
name=Specular Repository
baseurl=https://yum.specular.io/rpm
enabled=1
gpgcheck=1
gpgkey=https://yum.specular.io/rpm/RPM-GPG-KEY-specular
EOF

# Install
sudo yum install specular
```

---

## Release Automation

### GitHub Actions Workflow

**Trigger:** Tag push matching `v*`

**Steps:**
1. Checkout code
2. Setup Go environment
3. Run tests
4. Run linters
5. Run security scans
6. Execute GoReleaser
7. Publish to GitHub Releases
8. Update Homebrew tap
9. Push Docker images
10. Generate SBOM
11. Sign artifacts (if configured)

**Required Secrets:**
- `GITHUB_TOKEN` - GitHub releases (auto-provided)
- `TAP_GITHUB_TOKEN` - Homebrew tap updates
- `GPG_FINGERPRINT` - Artifact signing (optional)

### Manual Release Commands

```bash
# Snapshot release (test locally, no push)
goreleaser release --snapshot --clean --skip-publish

# Production release (requires credentials)
goreleaser release --clean

# Release with specific configuration
goreleaser release --clean --config .goreleaser-custom.yml

# Skip specific publishers
goreleaser release --clean --skip-publish --skip-sign
```

---

## Monitoring and Metrics

### Release Metrics to Track

1. **Download counts** - GitHub release assets
2. **Installation methods** - Homebrew vs package managers vs Docker
3. **Platform distribution** - Linux vs macOS vs Windows
4. **Architecture distribution** - amd64 vs arm64
5. **Version adoption rate** - Time to 50% adoption of new version
6. **Rollback rate** - Percentage of releases that need rollback

### Monitoring Tools

- **GitHub Insights** - Download statistics
- **Docker Hub Analytics** - Pull counts, image usage
- **Homebrew Analytics** - Install counts (opt-in)

---

## Best Practices

### Before Each Release

1. **Test thoroughly** - Don't skip the pre-release checklist
2. **Update changelog** - Clear, user-focused release notes
3. **Document breaking changes** - Migration guides for major versions
4. **Coordinate timing** - Release during business hours for quick response
5. **Communicate early** - Pre-release announcements for major versions

### During Release

1. **Monitor CI/CD** - Watch GitHub Actions for failures
2. **Verify artifacts** - Download and test release assets
3. **Test installations** - Verify all package managers work
4. **Check distributions** - Ensure Docker images are multi-arch

### After Release

1. **Monitor feedback** - Watch GitHub issues and discussions
2. **Track metrics** - Download counts, error reports
3. **Update documentation** - Ensure docs match released version
4. **Plan hotfixes** - Be ready to release patches quickly

---

## Emergency Procedures

### Critical Bug in Production

1. **Assess severity** - Is rollback necessary?
2. **Create hotfix branch** - From last stable tag
3. **Fix and test** - Minimal changes, thorough testing
4. **Fast-track release** - Skip minor checklist items if needed
5. **Notify users** - GitHub issue, release notes, social media

### Security Vulnerability

1. **Create private advisory** - GitHub Security Advisories
2. **Develop fix quietly** - Don't publicize vulnerability
3. **Coordinate disclosure** - Set responsible disclosure date
4. **Release patch** - Publish fix and advisory simultaneously
5. **Notify users** - Clear upgrade path and severity

---

## Support and Resources

- **Release Workflow:** `.github/workflows/release.yml`
- **GoReleaser Config:** `.goreleaser.yml`
- **Homebrew Tap:** https://github.com/felixgeelhaar/homebrew-tap
- **Docker Registry:** https://github.com/felixgeelhaar/specular/pkgs/container/specular
- **Release Guidelines:** https://docs.github.com/en/repositories/releasing-projects-on-github

---

**Last Updated:** 2025-01-15
**Version:** 1.0
**Owner:** Specular Core Team
