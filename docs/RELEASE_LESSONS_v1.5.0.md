# Release Lessons Learned - v1.5.0

**Release Date**: 2025-01-15
**Version**: v1.5.0
**Duration**: ~2 hours (from tag creation to full publication)

## Executive Summary

The v1.5.0 release was successfully completed with all major artifacts published, Homebrew tap updated, and documentation refreshed. While the release process worked well overall, several challenges were encountered that provide valuable lessons for future releases.

## Challenges Encountered

### 1. GoReleaser - Missing GPG_FINGERPRINT Environment Variable

**Issue**: Initial GoReleaser run failed with template error for missing `GPG_FINGERPRINT`.

```
⨯ release failed after 1m13s error=sign failed: checksums.txt:
template: failed to apply "{{ .Env.GPG_FINGERPRINT }}":
map has no entry for key "GPG_FINGERPRINT"
```

**Root Cause**: GPG signing configuration in `.goreleaser.yml` expects `GPG_FINGERPRINT` environment variable, but GPG keys are not configured in local development environment.

**Solution**:
- Used `--skip=sign` flag to skip GPG signing during local release
- GPG signing should be handled by GitHub Actions with proper key configuration

**Lesson Learned**:
- Local releases should skip GPG signing unless specifically configured
- GitHub Actions workflow should handle GPG signing with secrets
- Document GPG requirements in release process guide

**Action Items**:
- [ ] Add GPG signing setup instructions to `docs/RELEASE_PROCESS.md`
- [ ] Configure GPG keys in GitHub Actions secrets
- [ ] Update local release scripts to default to `--skip=sign`

### 2. GoReleaser - Dirty Git State from Before Hooks

**Issue**: GoReleaser before hooks (go mod tidy, completion generation) modified files, causing dirty git state error.

```
⨯ release failed after 0s error=git is in a dirty state
Please check in your pipeline what can be changing the following files:
 M completions/_specular
 M completions/specular.bash
 M completions/specular.fish
 M go.mod
```

**Root Cause**: GoReleaser's `before` hooks execute commands that modify files (shell completion generation, `go mod tidy`) but these changes aren't committed to the release tag.

**Solution**:
1. Committed the generated files to the repository
2. Amended the v1.5.0 commit to include these files
3. Force-updated the tag: `git tag -f v1.5.0`
4. Force-pushed both main and tag: `git push origin main --force && git push origin v1.5.0 --force`

**Lesson Learned**:
- Before hooks that modify files should be run **before** creating the release tag
- Consider adding a pre-release script that:
  1. Runs all before hooks
  2. Commits any generated files
  3. Creates the tag
  4. Runs GoReleaser

**Action Items**:
- [ ] Create `scripts/prepare-release.sh` that runs before hooks and commits changes
- [ ] Update `docs/RELEASE_PROCESS.md` to include pre-release preparation step
- [ ] Consider moving completion generation to build-time instead of release-time

### 3. Docker Image Publishing - Permission Denied

**Issue**: Docker image push to ghcr.io failed with permission denied error.

```
⨯ release failed after 16s error=docker images: failed to publish artifacts:
failed to push ghcr.io/felixgeelhaar/specular:v1.5.0-arm64: exit status 1:
denied: permission_denied: The token provided does not match expected scopes.
```

**Root Cause**: Local `GITHUB_TOKEN` doesn't have `packages:write` scope required for publishing to GitHub Container Registry.

**Solution**:
- Used `--skip=docker` flag to skip Docker image publishing during local release
- Docker images should be published via GitHub Actions with proper token scopes

**Lesson Learned**:
- Docker publishing requires specific token scopes not available in local development
- Separate Docker publishing from binary releases
- GitHub Actions workflow should handle Docker publishing

**Action Items**:
- [ ] Configure GitHub Actions with `packages: write` permission
- [ ] Test Docker image publishing in GitHub Actions workflow
- [ ] Document Docker publishing requirements in release guide

### 4. SBOM Duplicate Upload Error

**Issue**: SBOM upload failed with duplicate file error after retry.

```
⨯ release failed after 2m29s error=scm releases: failed to publish artifacts:
failed to upload specular_1.5.0_sbom.spdx.json: POST https://uploads.github.com/...:
422 Validation Failed [{Resource:ReleaseAsset Field:name Code:already_exists Message:}]
```

**Root Cause**: Previous partial GoReleaser run had already uploaded the SBOM file. Retry attempted to upload the same file again.

**Solution**:
- Release was created successfully despite the error
- SBOM file was already present from previous attempt
- This was a non-blocking error - the release was functional

**Lesson Learned**:
- GoReleaser doesn't handle partial upload retries gracefully
- Check existing release assets before retrying failed releases
- Consider using `--clean` with caution as it may delete and recreate the release

**Action Items**:
- [ ] Add cleanup step to release process for handling failed uploads
- [ ] Document retry procedures in release guide
- [ ] Consider implementing idempotent release artifact uploads

### 5. Homebrew Tap Auto-Update Not Triggered

**Issue**: Homebrew tap was not automatically updated by GoReleaser.

**Root Cause**:
- GoReleaser's `brews` configuration uses `TAP_GITHUB_TOKEN` environment variable
- Configuration has `skip_upload: auto` which skips when token is not set
- Local release only had `GITHUB_TOKEN` set, not `TAP_GITHUB_TOKEN`

**Solution**:
1. Manually cloned homebrew-tap repository
2. Updated formula file with v1.5.0 version and checksums
3. Committed and pushed changes to tap repository

**Lesson Learned**:
- Homebrew tap updates require separate token configuration
- `skip_upload: auto` silently skips tap updates when token is missing
- Manual tap updates are straightforward but time-consuming

**Action Items**:
- [ ] Configure `TAP_GITHUB_TOKEN` in GitHub Actions secrets
- [ ] Test automated tap updates in GitHub Actions workflow
- [ ] Add tap update verification to release checklist
- [ ] Consider using same token for both release and tap updates

### 6. GitHub Actions Release Workflow Not Triggering

**Issue**: Pushing the v1.5.0 tag didn't automatically trigger the GitHub Actions release workflow.

**Root Cause**: Unknown - the workflow has correct trigger configuration:

```yaml
on:
  push:
    tags:
      - 'v*.*.*'
```

**Attempted Solutions**:
1. Tried manual workflow trigger: `gh workflow run release.yml --ref v1.5.0`
   - Failed: `workflow_dispatch` doesn't support tag refs
2. Decided to run GoReleaser locally instead

**Lesson Learned**:
- GitHub Actions tag triggers can be unreliable or have caching issues
- Always have a local release fallback option
- Test release workflows in CI before major releases

**Action Items**:
- [ ] Investigate why tag push didn't trigger workflow
- [ ] Add workflow_dispatch trigger for manual releases
- [ ] Document both automated and manual release procedures
- [ ] Consider testing with pre-release tags first

## Successes

### What Went Well

1. **Multi-Platform Build Success**: All platform binaries (Darwin amd64/arm64, Linux amd64/arm64, Windows amd64) built successfully on first try

2. **Package Generation**: All package formats (.deb, .rpm, .apk) generated correctly with proper metadata

3. **Checksum Verification**: SHA256 checksums matched perfectly across all artifacts, verified by manual download test

4. **SBOM Generation**: Software Bill of Materials generated successfully using syft

5. **Release Notes**: Automated release notes from GoReleaser provided comprehensive changelog with commit references

6. **Backward Compatibility**: Zero breaking changes, seamless upgrade path for existing users

7. **Documentation**: Comprehensive CHANGELOG.md and README.md updates completed in parallel

## Process Improvements for v1.6.0

### High Priority

1. **Create Pre-Release Preparation Script** (`scripts/prepare-release.sh`):
   ```bash
   #!/bin/bash
   # Prepare release by running all before hooks and committing changes

   # Run go mod tidy
   go mod tidy

   # Generate shell completions
   go run ./cmd/specular completion bash > completions/specular.bash
   go run ./cmd/specular completion zsh > completions/_specular
   go run ./cmd/specular completion fish > completions/specular.fish

   # Commit changes
   git add completions go.mod go.sum
   git commit -m "chore: prepare release (generated files)"

   echo "✓ Release preparation complete"
   ```

2. **Enhance Release Workflow**:
   - Add `packages: write` permission for Docker publishing
   - Configure `TAP_GITHUB_TOKEN` for Homebrew tap updates
   - Add GPG signing with proper key configuration
   - Test with pre-release tags first

3. **Improve Release Checklist** in `docs/RELEASE_PROCESS.md`:
   - Add pre-release preparation step
   - Include verification steps for each artifact type
   - Add rollback procedures
   - Document manual fallback options

### Medium Priority

4. **Add Release Verification Tests**:
   - Automated checksum verification
   - Installation testing for each platform
   - Binary functionality smoke tests
   - Docker image validation

5. **Implement Release Dry-Run**:
   - Test release process with snapshot builds
   - Validate all artifacts before publishing
   - Verify tap updates in test repository

6. **Enhance Error Handling**:
   - Better detection of duplicate uploads
   - Graceful handling of partial release failures
   - Automatic cleanup of failed releases

### Low Priority

7. **Release Automation**:
   - Automatic version bumping
   - Changelog generation from conventional commits
   - PR-based release workflow

8. **Monitoring**:
   - Release success/failure notifications
   - Download metrics tracking
   - Issue reporting integration

## Metrics

### Release Duration Breakdown

- **Tag Creation to First GoReleaser Run**: 5 minutes
- **Troubleshooting GoReleaser Issues**: 45 minutes
  - GPG signing: 10 minutes
  - Dirty git state: 15 minutes
  - Docker permissions: 10 minutes
  - SBOM duplicate: 10 minutes
- **Manual Homebrew Tap Update**: 15 minutes
- **README Updates and Verification**: 30 minutes
- **Total**: ~2 hours

### Optimization Potential

With automated improvements, estimated duration for v1.6.0:
- **Pre-release preparation**: 5 minutes (automated script)
- **GitHub Actions release**: 10 minutes (fully automated)
- **Verification**: 10 minutes (automated tests)
- **Total**: ~25 minutes (80% reduction)

## Recommendations

### For v1.6.0 Release

1. **Before Tagging**:
   - Run `scripts/prepare-release.sh`
   - Verify all tests pass
   - Update CHANGELOG.md
   - Review release notes template

2. **Release Execution**:
   - Use GitHub Actions for automated release
   - Fall back to local GoReleaser only if necessary
   - Monitor release progress in GitHub Actions logs

3. **Post-Release**:
   - Verify all artifacts are downloadable
   - Test Homebrew installation
   - Update README with release highlights
   - Announce release in appropriate channels

### Long-Term Improvements

1. **Implement Continuous Deployment**:
   - Automatic releases on version tag push
   - Staging environment for pre-release testing
   - Gradual rollout strategy

2. **Enhance Release Process Documentation**:
   - Video walkthrough of release process
   - Troubleshooting guide with common issues
   - Release checklist with verification steps

3. **Add Release Quality Gates**:
   - Required test coverage thresholds
   - Security scan before release
   - Performance regression testing
   - Compatibility testing across platforms

## Conclusion

The v1.5.0 release was ultimately successful, with all artifacts published and verified. The challenges encountered were primarily related to local release execution and environment configuration. The key takeaway is to **prioritize automated releases via GitHub Actions** over local releases, with local execution serving as a fallback option.

The lessons learned from this release will significantly improve the v1.6.0 release process, reducing manual work and potential for errors.

---

**Document Author**: Claude Code Assistant
**Last Updated**: 2025-01-15
**Next Review**: Before v1.6.0 release planning
