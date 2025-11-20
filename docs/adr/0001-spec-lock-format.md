# ADR 0001: Spec Lock File Format

**Status:** Accepted

**Date:** 2025-01-07

**Decision Makers:** Specular Core Team

## Context

Specular needs a mechanism to detect when a product specification (`spec.yaml`) has changed after code has been generated. This enables drift detection - identifying when the implementation may no longer match the intended product definition.

### Requirements
1. Detect any changes to feature definitions in `spec.yaml`
2. Support both content and structural changes
3. Minimize file size and complexity
4. Enable version control friendly diffs
5. Support programmatic reading and writing
6. Fast comparison operations

### Options Considered

#### Option 1: YAML Format
**Pros:**
- Consistent with spec.yaml format
- Human-readable
- Easy to edit manually
- Supports comments

**Cons:**
- YAML parsing overhead
- Multiple valid representations of same data
- Whitespace sensitivity makes diffs unreliable
- Comments and formatting affect hashes

#### Option 2: JSON Format
**Pros:**
- Canonical representation (consistent formatting)
- Fast parsing in Go (stdlib `encoding/json`)
- Deterministic serialization
- Compact representation
- Clear diffs in version control
- Industry standard for lock files

**Cons:**
- Less human-readable than YAML
- No comments support
- Manual editing more error-prone

#### Option 3: Binary Format (Protocol Buffers, MessagePack)
**Pros:**
- Smallest file size
- Fastest parsing
- Schema evolution support (protobuf)

**Cons:**
- Not human-readable at all
- Requires special tools to inspect
- Version control diffs meaningless
- Additional dependencies

#### Option 4: Custom Text Format
**Pros:**
- Full control over format
- Optimize for specific use case

**Cons:**
- Maintenance burden
- No ecosystem tooling
- Error-prone parsing
- Reinventing the wheel

## Decision

**We will use JSON format for the spec lock file (`.specular/spec.lock.json`).**

### Rationale

1. **Deterministic Hashing**: JSON's canonical representation ensures that the same data always produces the same hash, which is critical for drift detection.

2. **Ecosystem Maturity**: JSON is universally supported with robust tooling (jq, JSON validators, IDE support) and Go's stdlib provides excellent performance.

3. **Git-Friendly Diffs**: JSON's structure creates meaningful, line-based diffs in version control, making it easy to see what changed.

4. **Performance**: Go's `encoding/json` package is fast enough for our use case (files are typically < 100KB) and provides streaming capabilities if needed.

5. **Lock File Conventions**: Industry standard for lock files (package-lock.json, yarn.lock, Cargo.lock) demonstrates JSON's suitability for this purpose.

6. **Balance**: JSON strikes the right balance between human-readability (for debugging) and machine-efficiency (for processing).

## Implementation

### File Structure
```json
{
  "version": "1.0",
  "generated_at": "2025-01-07T10:30:00Z",
  "spec_path": ".specular/spec.yaml",
  "spec_hash": "sha256:abc123...",
  "features": {
    "feat-001": {
      "id": "feat-001",
      "title": "User Authentication",
      "hash": "sha256:def456...",
      "priority": "P0",
      "api_endpoints": 5,
      "locked_at": "2025-01-07T10:30:00Z"
    }
  },
  "metadata": {
    "total_features": 5,
    "total_endpoints": 23,
    "specular_version": "1.0.0"
  }
}
```

### Hash Algorithm
- **SHA-256** for cryptographic security and collision resistance
- Hash includes: feature ID, title, description, API contracts, success criteria, trace IDs
- Excludes: timestamps, metadata, non-functional data

### Serialization
```go
// Ensure deterministic JSON output
encoder := json.NewEncoder(file)
encoder.SetIndent("", "  ")  // 2-space indentation for readability
encoder.SetEscapeHTML(false)  // Don't escape HTML characters
```

## Consequences

### Positive
- ✅ Fast, reliable drift detection
- ✅ Clear version control history
- ✅ Easy debugging with standard JSON tools
- ✅ No external dependencies
- ✅ Future-proof with version field

### Negative
- ❌ Not as human-friendly as YAML
- ❌ No support for comments
- ❌ Manual edits require valid JSON syntax

### Mitigations
- Provide `specular lock` command for regeneration (no manual editing needed)
- Include helpful metadata in lock file for debugging
- Use descriptive field names
- Pretty-print with consistent formatting

### Trade-offs
- Chose reliability over editability
- Chose performance over maximum compression
- Chose ecosystem compatibility over custom optimization

## Related Decisions
- ADR 0005: Drift Detection Approach
- Future ADR: Spec versioning and migration

## References
- [JSON Specification (RFC 8259)](https://tools.ietf.org/html/rfc8259)
- [Go encoding/json Package](https://pkg.go.dev/encoding/json)
- [npm package-lock.json Format](https://docs.npmjs.com/cli/v10/configuring-npm/package-lock-json)
