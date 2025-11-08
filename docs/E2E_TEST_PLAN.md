# End-to-End Test Plan - Bundle System v1.3.0

**Version**: 1.3.0
**Date**: 2025-11-08
**Status**: Test Plan (Tests to be Implemented)
**Purpose**: Define comprehensive end-to-end test scenarios for the bundle governance system

---

## Overview

This document outlines the end-to-end test scenarios required to validate the complete bundle lifecycle, from creation through deployment. These tests should be implemented when E2E test infrastructure is established.

---

## Test Environment Requirements

### Prerequisites
- Go 1.23+ installed
- Git configured
- SSH keys generated (for approval tests)
- GPG keys generated (for L3+ approval tests)
- Docker running (for registry tests)
- GitHub Container Registry access (for registry tests)
- Docker Hub access (for registry tests)

### Test Data
- Sample project with spec.yaml, spec.lock.json, routing.yaml
- Test policies in `.specular/policies/`
- Test SSH keys (Ed25519, ECDSA, RSA)
- Test GPG keys

---

## Test Scenarios

### 1. Basic Bundle Lifecycle (L1 Governance)

**Objective**: Verify basic bundle creation, verification, and application workflow

**Steps**:
1. Create test project with spec.yaml, lock.json, routing.yaml
2. Build bundle with L1 governance (`specular bundle build --governance-level L1`)
3. Verify bundle integrity (`specular bundle verify`)
4. Apply bundle to new directory (`specular bundle apply`)
5. Verify all files extracted correctly
6. Verify checksums match

**Expected Results**:
- Bundle creates successfully in < 5s
- Verification passes in < 2s
- All project files extracted to target directory
- File permissions preserved
- Checksum validation passes

**Success Criteria**:
- ✅ Bundle file exists and is valid .tar.gz
- ✅ Manifest contains all expected files
- ✅ Checksums match for all files
- ✅ Applied files match original files byte-for-byte

---

### 2. Single Approval Workflow (L2 Governance)

**Objective**: Verify PM approval workflow for staging deployments

**Steps**:
1. Build bundle requiring PM approval
   ```bash
   specular bundle build \
     --require-approval pm \
     --governance-level L2 \
     --output staging.sbundle.tgz
   ```
2. Verify bundle without approvals (should fail)
3. Check approval status (should show 0/1 approvals)
4. Approve as PM with SSH key
   ```bash
   specular bundle approve staging.sbundle.tgz \
     --role pm \
     --user pm@test.com \
     --key ~/.ssh/test_ed25519 \
     --comment "Staging approved"
   ```
5. Check approval status (should show 1/1 approvals)
6. Verify bundle with approvals (should pass)
7. Verify signature is valid

**Expected Results**:
- Bundle requires approvals
- Verification fails without approvals
- Approval adds valid signature
- Verification passes with valid approval
- Signature verification succeeds

**Success Criteria**:
- ✅ Missing approval error is actionable
- ✅ Approval status shows required vs. obtained
- ✅ SSH signature is cryptographically valid
- ✅ Bundle verification passes after approval

---

### 3. Multi-Approval Workflow (L3 Governance)

**Objective**: Verify production workflow requiring PM + Security approvals

**Steps**:
1. Build bundle requiring multiple approvals
   ```bash
   specular bundle build \
     --require-approval pm \
     --require-approval security \
     --governance-level L3 \
     --output production.sbundle.tgz
   ```
2. Verify approval status (0/2 approvals)
3. Add PM approval with SSH key
4. Verify approval status (1/2 approvals)
5. Verify bundle (should still fail - partial approvals)
6. Add Security approval with GPG key
7. Verify approval status (2/2 approvals)
8. Verify bundle (should pass)

**Expected Results**:
- Partial approvals don't satisfy requirements
- Both approvals required for verification
- Mixed signature types (SSH + GPG) work correctly
- Approval order doesn't matter

**Success Criteria**:
- ✅ Partial approvals correctly rejected
- ✅ Both SSH and GPG signatures verify
- ✅ Approval roles correctly validated
- ✅ Complete approval set passes verification

---

### 4. Attestation Workflow (L4 Governance)

**Objective**: Verify SLSA attestation generation and verification

**Steps**:
1. Build bundle with attestation
   ```bash
   specular bundle build \
     --require-approval pm \
     --require-approval security \
     --governance-level L4 \
     --attest \
     --output slsa-bundle.sbundle.tgz
   ```
2. Verify attestation is included in bundle
3. Check attestation format (in-toto statement)
4. Add required approvals
5. Verify bundle with attestation checking
   ```bash
   specular bundle verify slsa-bundle.sbundle.tgz \
     --verify-attestation
   ```

**Expected Results**:
- Attestation generated in in-toto format
- Attestation includes build environment metadata
- Attestation verification succeeds (when implemented)
- Bundle contains attestation directory

**Success Criteria**:
- ✅ Attestation file exists in bundle
- ✅ Attestation follows in-toto specification
- ✅ Build environment captured in attestation
- ✅ (Future) Rekor verification passes

---

### 5. Bundle Comparison Workflow

**Objective**: Verify bundle diff functionality for version comparison

**Steps**:
1. Create v1.0.0 bundle
2. Modify spec.yaml (add feature)
3. Create v2.0.0 bundle with changes
4. Run bundle diff
   ```bash
   specular bundle diff v1.0.0.sbundle.tgz v2.0.0.sbundle.tgz
   ```
5. Verify diff output shows changes

**Expected Results**:
- Diff detects file modifications
- Diff shows governance level changes
- Diff highlights approval differences
- Diff output is readable and actionable

**Success Criteria**:
- ✅ Modified files correctly identified
- ✅ Added/removed files detected
- ✅ Governance level changes highlighted
- ✅ Diff format is clear and useful

---

### 6. Registry Publishing Workflow (GHCR)

**Objective**: Verify bundle publishing to GitHub Container Registry

**Steps**:
1. Authenticate with GHCR
   ```bash
   echo $GITHUB_TOKEN | docker login ghcr.io -u $USER --password-stdin
   ```
2. Build production bundle
3. Push to GHCR
   ```bash
   specular bundle push production.sbundle.tgz \
     ghcr.io/org/app:v1.0.0
   ```
4. Verify bundle appears in GHCR
5. Pull bundle from GHCR
   ```bash
   specular bundle pull ghcr.io/org/app:v1.0.0
   ```
6. Verify pulled bundle matches original

**Expected Results**:
- Bundle uploads successfully as OCI artifact
- Bundle appears in GHCR UI
- Pull retrieves identical bundle
- Checksums match after round-trip

**Success Criteria**:
- ✅ OCI artifact format is correct
- ✅ Bundle pullable from registry
- ✅ Round-trip preserves bundle integrity
- ✅ Registry authentication works

---

### 7. Registry Publishing Workflow (Docker Hub)

**Objective**: Verify bundle publishing to Docker Hub

**Steps**:
1. Authenticate with Docker Hub
   ```bash
   echo $DOCKERHUB_TOKEN | docker login -u $USER --password-stdin
   ```
2. Build production bundle
3. Push to Docker Hub
   ```bash
   specular bundle push production.sbundle.tgz \
     docker.io/org/app:v1.0.0
   ```
4. Verify bundle appears in Docker Hub
5. Pull bundle from Docker Hub
6. Verify pulled bundle matches original

**Expected Results**:
- Same as GHCR test
- Docker Hub rate limits not exceeded
- Multi-tag support works (version, latest)

**Success Criteria**:
- ✅ Works with Docker Hub v2 API
- ✅ Rate limiting handled gracefully
- ✅ Semantic versioning tags work
- ✅ Latest tag updates correctly

---

### 8. Error Handling Scenarios

**Objective**: Verify robust error handling and user-friendly messages

#### 8.1 Missing Required Files
**Steps**:
1. Attempt to build bundle without spec.yaml
2. Verify error message is actionable

**Expected Error**:
```
Error: failed to load spec: spec.yaml not found
Suggestion: Ensure spec.yaml exists in the current directory
```

#### 8.2 Corrupted Bundle
**Steps**:
1. Create invalid .tar.gz file
2. Attempt to verify
3. Check error message

**Expected Error**:
```
Error: invalid bundle format: not a valid tar.gz file
Suggestion: Re-download the bundle or re-create it
```

#### 8.3 Checksum Mismatch
**Steps**:
1. Build valid bundle
2. Manually modify a file in the extracted bundle
3. Re-package and verify
4. Check error details

**Expected Error**:
```
Error: checksum mismatch for spec.yaml
Expected: sha256:abc123...
Got: sha256:def456...
Suggestion: Bundle has been tampered with or corrupted
```

#### 8.4 Missing Approvals
**Steps**:
1. Build bundle requiring approvals
2. Attempt to verify without approvals
3. Check error shows which approvals are missing

**Expected Error**:
```
Error: missing required approvals
Required: [pm, security]
Obtained: []
Suggestion: Use 'specular bundle approve' to add approvals
```

#### 8.5 Invalid Signature
**Steps**:
1. Build bundle with approval
2. Modify bundle content after approval
3. Attempt to verify
4. Check signature validation error

**Expected Error**:
```
Error: signature verification failed for pm approval
Signature invalid for current bundle content
Suggestion: Bundle modified after signing - obtain new approval
```

---

### 9. Performance Benchmarks

**Objective**: Validate performance targets are met

#### 9.1 Build Performance
**Test**:
- Bundle with 10 files (total 1MB)
- Bundle with 100 files (total 10MB)
- Bundle with 1000 files (total 100MB)

**Targets**:
- 10 files: < 100ms
- 100 files: < 500ms
- 1000 files: < 5s

#### 9.2 Verify Performance
**Test**:
- Verify bundle with 10 files
- Verify bundle with 100 files
- Verify bundle with 1000 files

**Targets**:
- 10 files: < 50ms
- 100 files: < 200ms
- 1000 files: < 2s

#### 9.3 Registry Operations
**Test**:
- Push 1MB bundle
- Push 10MB bundle
- Push 100MB bundle
- Pull same sizes

**Targets**:
- 1MB: < 3s (push), < 2s (pull)
- 10MB: < 10s (push), < 5s (pull)
- 100MB: < 60s (push), < 30s (pull)

---

## Manual Test Scenarios

These scenarios require manual testing as they involve user interaction:

### 1. Interactive Apply Confirmation
**Steps**:
1. Apply bundle to directory with existing files
2. Verify prompts for overwrite confirmation
3. Test "yes to all" option
4. Test selective file application

### 2. Dry Run Mode
**Steps**:
1. Run bundle apply with --dry-run
2. Verify shows what would change
3. Verify no actual changes made
4. Verify output is clear and useful

### 3. Example Scripts
**Steps**:
1. Run `examples/team-approval/single-approver.sh`
2. Run `examples/team-approval/multi-approver.sh`
3. Verify scripts complete successfully
4. Verify output is colored and clear

---

## Automated Test Suite Structure

When implementing these tests, organize as follows:

```
test/
├── e2e/
│   ├── basic_lifecycle_test.go
│   ├── approval_workflow_test.go
│   ├── attestation_workflow_test.go
│   ├── diff_workflow_test.go
│   ├── registry_ghcr_test.go
│   ├── registry_dockerhub_test.go
│   ├── error_scenarios_test.go
│   └── performance_test.go
├── fixtures/
│   ├── test-project/
│   │   ├── spec.yaml
│   │   ├── spec.lock.json
│   │   └── routing.yaml
│   ├── keys/
│   │   ├── test_ed25519
│   │   ├── test_ed25519.pub
│   │   ├── test_rsa
│   │   └── test_rsa.pub
│   └── bundles/
│       ├── valid-l1.sbundle.tgz
│       ├── valid-l2.sbundle.tgz
│       └── corrupted.sbundle.tgz
└── helpers/
    ├── project.go
    ├── keys.go
    └── registry.go
```

---

## Test Execution

### Run All E2E Tests
```bash
go test -v ./test/e2e/... -timeout 30m
```

### Run Specific Test Suite
```bash
go test -v ./test/e2e/basic_lifecycle_test.go
```

### Run Performance Tests
```bash
go test -v ./test/e2e/performance_test.go -timeout 1h
```

### Run with Coverage
```bash
go test -v -coverprofile=e2e-coverage.txt ./test/e2e/...
```

---

## CI/CD Integration

### GitHub Actions Workflow
```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - name: Run E2E Tests
        run: go test -v ./test/e2e/... -timeout 30m
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./e2e-coverage.txt
```

---

## Test Data Management

### Test Project Template
```yaml
# test/fixtures/test-project/spec.yaml
product: test/e2e-bundle
version: "1.0.0"
description: E2E test project
features:
  - core-feature
  - test-feature
```

### Test Keys Generation
```bash
# Generate test SSH key
ssh-keygen -t ed25519 -f test/fixtures/keys/test_ed25519 -N ""

# Generate test GPG key
gpg --batch --gen-key <<EOF
Key-Type: RSA
Key-Length: 2048
Name-Real: Test User
Name-Email: test@example.com
Expire-Date: 0
%no-protection
%commit
EOF
```

---

## Success Metrics

### Coverage Goals
- **Unit Test Coverage**: 80% (current: 34.3%)
- **Integration Test Coverage**: 70%
- **E2E Test Coverage**: 60% of user workflows

### Quality Gates
- All E2E tests pass on every commit
- No performance regressions (within 10% of targets)
- Zero critical security findings
- All error messages are actionable

---

## Test Maintenance

### Weekly
- Review failed tests
- Update test data
- Check performance trends

### Monthly
- Review test coverage
- Add tests for new features
- Retire obsolete tests

### Quarterly
- Performance baseline updates
- Test infrastructure review
- Security test updates

---

## Future Test Additions

### v1.4.0
- Rekor integration tests
- Complete Sigstore verification tests
- Hardware security key tests
- Policy enforcement tests

### v2.0.0
- Keyless signing tests
- Advanced governance workflows
- Bundle template tests
- Marketplace integration tests

---

## References

- [Bundle User Guide](BUNDLE_USER_GUIDE.md)
- [Security Audit](SECURITY_AUDIT.md)
- [Example Workflows](../examples/)
- [Go Testing Documentation](https://go.dev/doc/tutorial/add-a-test)
- [Testify Framework](https://github.com/stretchr/testify)

---

**Test Plan Version**: 1.0.0
**Last Updated**: 2025-11-08
**Next Review**: 2025-12-08
