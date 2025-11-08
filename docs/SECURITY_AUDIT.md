# Security Audit Report - Bundle System
**Date**: 2025-11-08
**Version**: v1.3.0
**Auditor**: Automated Security Review
**Scope**: Cryptography, Signatures, and Validation in `internal/bundle`

## Executive Summary

This security audit reviewed the cryptographic implementations, signing mechanisms, and validation logic in the bundle system. The audit identified **8 security concerns** requiring attention, ranging from incomplete implementations to potential command injection vulnerabilities. Overall, the system demonstrates good security practices in many areas but requires completion of stub implementations and hardening of external command execution.

**Risk Rating**: MEDIUM
**Critical Issues**: 1
**High Priority**: 3
**Medium Priority**: 4

---

## Findings

### ✅ Positive Security Practices

1. **Secure Random Number Generation** ✓
   - **Location**: `internal/bundle/signing.go:118`
   - Uses `crypto/rand` for cryptographic operations
   - Proper entropy source for SSH signature generation

2. **Strong Cryptographic Hash Functions** ✓
   - **Location**: `internal/bundle/signing.go:9`, `attestation_sigstore.go:7`
   - Uses SHA-256 throughout for digest computation
   - No use of deprecated hash functions (MD5, SHA1)

3. **Input Validation** ✓
   - **Location**: `internal/bundle/signing.go:37-47`
   - Validates required fields before signing operations
   - Checks for empty bundle digests, roles, and user identifiers

4. **Comprehensive Error Handling** ✓
   - Errors are properly wrapped with context using `fmt.Errorf`
   - No silent failures or swallowed errors

5. **Time-based Security Controls** ✓
   - **Location**: `internal/bundle/signing.go:206-209`
   - Supports approval expiration checks with configurable MaxAge
   - Prevents use of stale approvals

6. **Trusted Key Verification** ✓
   - **Location**: `internal/bundle/signing.go:232-243`
   - Supports trusted key lists for approval verification
   - Allows enforcement of key trust policies

7. **Role-based Access Control** ✓
   - **Location**: `internal/bundle/signing.go:217-229`
   - Validates approval roles against allowed roles list
   - Supports fine-grained approval policies

8. **Path Traversal Protection** ✓
   - **Location**: `internal/bundle/validator.go:173-176`
   - Validates extracted file paths to prevent directory traversal
   - Uses `filepath.Clean` and prefix checking

---

## Security Concerns

### 🔴 CRITICAL: GPG Command Injection Risk

**Severity**: Critical
**Location**: `internal/bundle/signing.go:150-163`, `signing.go:296-339`
**CWE**: CWE-78 (OS Command Injection)

**Issue**:
The code uses `exec.Command` to invoke GPG with user-provided `keyPath` parameter. While Go's `exec.Command` provides some protection against shell injection, the keyPath is not validated before being passed as an argument.

```go
cmd := exec.Command("gpg", "--detach-sign", "--armor", "--output", "-", tmpFile.Name())
if keyPath != "" {
    cmd.Args = append(cmd.Args[:2], "--local-user", keyPath)
    cmd.Args = append(cmd.Args, cmd.Args[2:]...)
}
```

**Risk**:
An attacker could potentially inject GPG arguments through the keyPath parameter, leading to unintended GPG behavior or information disclosure.

**Recommendation**:
1. Validate `keyPath` against a whitelist of allowed characters (alphanumeric + hyphen + underscore)
2. Consider using a GPG library instead of shell commands (e.g., `github.com/ProtonMail/go-crypto/openpgp`)
3. If shell commands are required, use strict input validation:
   ```go
   if !isValidKeyPath(keyPath) {
       return fmt.Errorf("invalid key path format")
   }
   ```

---

### 🟡 HIGH: Incomplete Signature Verification

**Severity**: High
**Location**: `internal/bundle/attestation_sigstore.go:322-346`
**CWE**: CWE-345 (Insufficient Verification of Data Authenticity)

**Issue**:
The `verifySignature()` function contains placeholder implementations that return "not yet implemented" errors:

```go
if attestation.Signature.Certificate != "" {
    return fmt.Errorf("certificate verification not yet implemented")
}

if attestation.Signature.PublicKey != "" {
    return fmt.Errorf("public key verification not yet implemented")
}
```

**Risk**:
Attestations with signatures cannot be cryptographically verified, undermining the entire attestation security model. This effectively makes attestations optional security theater.

**Recommendation**:
1. **Implement public key verification**:
   - Parse PEM-encoded public key
   - Recreate the signed payload (statement JSON)
   - Verify signature using `sigstore/sigstore` library
2. **Implement certificate verification**:
   - Verify certificate chain against Fulcio root
   - Check certificate validity period
   - Verify OIDC claims in certificate
3. **Disable attestation features** until verification is complete, or document limitations clearly

---

### 🟡 HIGH: Missing Rekor Transparency Log Verification

**Severity**: High
**Location**: `internal/bundle/attestation_sigstore.go:348-365`
**CWE**: CWE-345 (Insufficient Verification of Data Authenticity)

**Issue**:
Rekor entry verification is incomplete:

```go
func (v *AttestationVerifier) verifyRekorEntry(ctx context.Context, attestation *Attestation) error {
    // For MVP, just check entry exists
    // Full verification would fetch and verify the entry from Rekor
    return fmt.Errorf("Rekor entry verification not yet implemented")
}
```

**Risk**:
Cannot verify inclusion in Rekor transparency log, which is a core security property of Sigstore. Attackers could forge Rekor entries without detection.

**Recommendation**:
1. Use `github.com/sigstore/rekor/pkg/client` to fetch entries
2. Verify inclusion proof using Merkle tree verification
3. Validate entry contents match the attestation
4. Check signed entry timestamp (SET) if present

---

### 🟡 HIGH: Temporary File Permissions

**Severity**: High
**Location**: `internal/bundle/signing.go:138-143`, `signing.go:303-327`
**CWE**: CWE-732 (Incorrect Permission Assignment for Critical Resource)

**Issue**:
Temporary files are created without explicit permission control:

```go
tmpFile, err := os.CreateTemp("", "specular-sign-*.txt")
```

On many systems, `os.CreateTemp` creates files with 0600 permissions, but this is not guaranteed and depends on umask.

**Risk**:
Sensitive data (signing messages, signatures) could be readable by other users on multi-user systems.

**Recommendation**:
1. Explicitly set restrictive permissions:
   ```go
   tmpFile, err := os.CreateTemp("", "specular-sign-*.txt")
   if err != nil {
       return err
   }
   if err := tmpFile.Chmod(0600); err != nil {
       return fmt.Errorf("failed to set file permissions: %w", err)
   }
   ```
2. Ensure temporary files are deleted even if errors occur (use defer immediately after creation)

---

### 🟠 MEDIUM: GPG Keyring Management

**Severity**: Medium
**Location**: `internal/bundle/signing.go:296`
**CWE**: CWE-668 (Exposure of Resource to Wrong Sphere)

**Issue**:
Uses a shared keyring file name:

```go
keyImportCmd := exec.Command("gpg", "--import", "--no-default-keyring", "--keyring", "trustedkeys.gpg")
```

**Risk**:
- Multiple concurrent verifications could interfere with each other
- Keyring file may persist with incorrect permissions
- No cleanup of imported keys

**Recommendation**:
1. Use unique temporary keyring per verification:
   ```go
   keyringPath := filepath.Join(tmpDir, "keyring.gpg")
   defer os.RemoveAll(tmpDir)
   ```
2. Use `--keyring` with absolute path to temporary location
3. Ensure proper cleanup in defer statements

---

### 🟠 MEDIUM: Potential DoS in Bundle Digest Calculation

**Severity**: Medium
**Location**: `internal/bundle/signing.go:394-402`
**CWE**: CWE-400 (Uncontrolled Resource Consumption)

**Issue**:
`ComputeBundleDigest` loads the entire file into memory:

```go
func ComputeBundleDigest(bundlePath string) (string, error) {
    data, err := os.ReadFile(bundlePath)
    if err != nil {
        return "", fmt.Errorf("failed to read bundle: %w", err)
    }
    hash := sha256.Sum256(data)
    return fmt.Sprintf("sha256:%x", hash), nil
}
```

**Risk**:
Large bundles (>1GB) could exhaust memory or cause DoS conditions.

**Recommendation**:
Use streaming hash calculation:
```go
func ComputeBundleDigest(bundlePath string) (string, error) {
    file, err := os.Open(bundlePath)
    if err != nil {
        return "", fmt.Errorf("failed to open bundle: %w", err)
    }
    defer file.Close()

    hash := sha256.New()
    if _, err := io.Copy(hash, file); err != nil {
        return "", fmt.Errorf("failed to hash bundle: %w", err)
    }

    return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}
```

---

### 🟠 MEDIUM: SSH Signature Format Assumptions

**Severity**: Medium
**Location**: `internal/bundle/signing.go:280-283`
**CWE**: CWE-20 (Improper Input Validation)

**Issue**:
Assumes signature format matches public key type without validation:

```go
sig := &ssh.Signature{
    Format: publicKey.Type(),
    Blob:   signatureBytes,
}
```

**Risk**:
Signature format mismatches could cause verification failures or allow signature substitution attacks.

**Recommendation**:
1. Validate that decoded signature contains expected format field
2. Verify format matches public key type before attempting verification
3. Add explicit error handling for format mismatches

---

### 🟠 MEDIUM: Missing Canonicalization in Sign Message

**Severity**: Medium
**Location**: `internal/bundle/signing.go:343-357`
**CWE**: CWE-347 (Improper Verification of Cryptographic Signature)

**Issue**:
The `formatSignMessage` function builds the message to sign, but field ordering and formatting could vary:

```go
func formatSignMessage(approval *Approval, bundleDigest string) string {
    var buf strings.Builder
    buf.WriteString("SPECULAR BUNDLE APPROVAL\n")
    buf.WriteString(fmt.Sprintf("Bundle Digest: %s\n", bundleDigest))
    buf.WriteString(fmt.Sprintf("Role: %s\n", approval.Role))
    buf.WriteString(fmt.Sprintf("User: %s\n", approval.User))
    buf.WriteString(fmt.Sprintf("Timestamp: %s\n", approval.SignedAt.Format(time.RFC3339)))
    // ...
}
```

**Risk**:
While the current implementation is deterministic, future modifications could introduce field ordering issues that break signature verification.

**Recommendation**:
1. Document the canonical message format explicitly
2. Add tests that verify message format stability
3. Consider using a structured format (JSON with canonical encoding)
4. Add format version identifier to prevent cross-version attacks

---

## Additional Security Recommendations

### 1. Key Storage and Protection
- **Current**: Keys are read from filesystem without additional protection
- **Recommendation**: Support encrypted private keys with password/passphrase
- **Implementation**: Use SSH agent or GPG agent for key access instead of direct file reads

### 2. Audit Logging
- **Current**: No audit trail of signing operations
- **Recommendation**: Log all signature creation and verification events
- **Implementation**:
  - Log bundle digest, role, user, timestamp for all approvals
  - Log verification successes and failures
  - Consider structured logging for security events

### 3. Rate Limiting
- **Current**: No rate limiting on signature operations
- **Recommendation**: Implement rate limiting to prevent brute force or DoS
- **Implementation**: Add configurable rate limits for:
  - Signature generation per user
  - Verification attempts per bundle
  - Rekor lookups

### 4. Cryptographic Agility
- **Current**: Hard-coded to SHA-256
- **Recommendation**: Support algorithm negotiation for future upgrades
- **Implementation**:
  - Add algorithm identifier to signature metadata
  - Support SHA-384, SHA-512 for future-proofing
  - Document algorithm deprecation policy

### 5. Secure Defaults
- **Current**: Some security features are optional
- **Recommendation**: Enable security features by default
- **Implementation**:
  - Make signature verification mandatory for production
  - Require Rekor entry for high-governance bundles
  - Default to strict validation mode

---

## Testing Recommendations

### Security Test Coverage Needed:

1. **Fuzzing Tests**:
   - Fuzz SSH signature verification with malformed signatures
   - Fuzz GPG keyPath parameter with special characters
   - Fuzz bundle digest computation with crafted files

2. **Negative Tests**:
   - Test rejection of expired approvals
   - Test rejection of untrusted keys
   - Test rejection of modified bundles
   - Test signature format mismatches

3. **Integration Tests**:
   - End-to-end approval workflow with real keys
   - Multi-approver scenarios
   - Key rotation scenarios
   - Attestation with Rekor (when implemented)

4. **Performance Tests**:
   - Large bundle digest calculation
   - Concurrent verification operations
   - Memory usage under load

---

## Compliance Considerations

### SLSA Compliance:
- ✅ Build provenance generation implemented
- ❌ Complete signature verification required for SLSA L2+
- ❌ Rekor transparency required for SLSA L3+

### Sigstore Compliance:
- ⚠️  Keyless signing not yet implemented
- ❌ Fulcio certificate verification missing
- ❌ Rekor inclusion proof verification missing

### Supply Chain Security:
- ✅ Cryptographic signatures supported
- ✅ Role-based approvals implemented
- ⚠️  Verification completeness needs improvement

---

## Priority Remediation Plan

### Immediate (Before Production Release):
1. **Fix**: Implement proper attestation signature verification (HIGH)
2. **Fix**: Add input validation for GPG keyPath parameter (CRITICAL)
3. **Fix**: Use streaming digest calculation (MEDIUM)
4. **Document**: Clearly mark incomplete Rekor features as experimental

### Short-term (Within 1 Month):
1. **Implement**: Complete Rekor verification
2. **Implement**: Temporary file permission hardening
3. **Implement**: GPG keyring isolation
4. **Add**: Security test suite with fuzzing

### Long-term (Within 3 Months):
1. **Implement**: Keyless signing support
2. **Implement**: Certificate verification
3. **Add**: Comprehensive audit logging
4. **Add**: Rate limiting and DoS protection

---

## Conclusion

The bundle system demonstrates solid foundational security practices but requires completion of cryptographic verification implementations before production use. The critical command injection risk should be addressed immediately, and incomplete signature verification features must be completed or disabled.

**Overall Security Posture**: MEDIUM
**Production Readiness**: NOT READY (incomplete verification)
**Recommended Action**: Complete signature and Rekor verification before v1.3.0 release

---

## References

- OWASP Top 10: https://owasp.org/www-project-top-ten/
- CWE Database: https://cwe.mitre.org/
- Sigstore Documentation: https://docs.sigstore.dev/
- SLSA Framework: https://slsa.dev/
- Go Security Best Practices: https://go.dev/doc/security/best-practices

**Report Generated**: 2025-11-08
**Next Review Due**: 2025-12-08 (Monthly)
