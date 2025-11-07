# v1.2.0 Release Checklist

Quick reference for executing the v1.2.0 release.

## Pre-Release Checks

### Code Quality ✅
- [x] All tests passing (58/58)
- [x] Coverage ≥ 60% (84.6%, 61.4%, 36.5%)
- [x] No race conditions
- [x] Linters passing
- [x] No TODOs in production code

### Documentation ✅
- [x] CHANGELOG.md updated
- [x] Best practices guide complete
- [x] Checkpoint/resume docs complete
- [x] Progress indicators docs complete
- [x] CI/CD examples complete
- [x] Release strategy documented

### Infrastructure 🔲
- [ ] GoReleaser validated (`goreleaser check`)
- [ ] GPG key available (`gpg --list-secret-keys 299BB654DA4CFDE6`)
- [ ] GitHub tokens set:
  - [ ] `GITHUB_TOKEN` (gh auth status)
  - [ ] `TAP_GITHUB_TOKEN` (for Homebrew)
- [ ] Docker logged in (`docker login ghcr.io`)
- [ ] Homebrew tap accessible

## Release Execution

### Step 1: Final Checks

```bash
# Ensure on main with latest
git checkout main
git pull origin main

# Verify clean state
git status  # Should be clean

# Run full test suite
go test -v -race ./...

# Run linters
golangci-lint run ./...

# Test build
go build -o specular ./cmd/specular
./specular version
```

### Step 2: Create Tag

```bash
# Create annotated tag
git tag -a v1.2.0 -m "Release v1.2.0 - CLI Enhancement & Production Readiness

🚀 Enhanced init with 5 templates
📊 Route optimization & benchmarking
✅ Comprehensive testing (84.6% coverage)
🔧 CI/CD integration (4 platforms)
📚 3,000+ lines of documentation

See CHANGELOG.md for details."

# Verify tag
git tag -l v1.2.0

# Push tag (triggers release)
git push origin v1.2.0
```

### Step 3: Monitor Release

```bash
# Watch GitHub Actions
gh run watch

# Or check release status
gh release view v1.2.0
```

### Step 4: Verify Artifacts

```bash
# Check all artifacts present
gh release view v1.2.0 --json assets | jq '.assets[].name'

# Expected:
# - specular_1.2.0_darwin_amd64.tar.gz
# - specular_1.2.0_darwin_arm64.tar.gz
# - specular_1.2.0_linux_amd64.tar.gz
# - specular_1.2.0_linux_arm64.tar.gz
# - specular_1.2.0_windows_amd64.zip
# - checksums.txt
# - checksums.txt.sig
# - *.deb, *.rpm, *.apk packages

# Test Docker image
docker pull ghcr.io/felixgeelhaar/specular:v1.2.0
docker run --rm ghcr.io/felixgeelhaar/specular:v1.2.0 version

# Test Homebrew (if available)
brew tap felixgeelhaar/tap
brew install specular
specular version
```

## Post-Release

### Immediate (Day 0)

- [ ] Announce on GitHub Discussions
- [ ] Tweet/X announcement
- [ ] LinkedIn post
- [ ] Update project website
- [ ] Monitor GitHub Issues

### Week 1

- [ ] Write blog post
- [ ] Gather user feedback
- [ ] Monitor metrics:
  - [ ] Downloads by platform
  - [ ] Docker pulls
  - [ ] GitHub stars
  - [ ] Installation issues

### Week 2-4

- [ ] Analyze usage patterns
- [ ] Plan v1.3.0
- [ ] Update roadmap
- [ ] Celebrate! 🎉

## Rollback Plan

If critical issue found:

```bash
# Mark as pre-release
gh release edit v1.2.0 --prerelease

# Announce issue
# Fix and re-release as v1.2.1
```

## Quick Commands

```bash
# Check test status
go test -v ./...

# Check linters
golangci-lint run ./...

# Create release
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0

# Monitor
gh run watch

# Verify
gh release view v1.2.0
docker pull ghcr.io/felixgeelhaar/specular:v1.2.0
```

## Contact

**Release Manager:** Felix Geelhaar
**Date:** 2025-11-07
**Status:** ✅ Ready for Release
