# ADR 0008: Structured Error Handling with Error Codes

**Status:** Accepted

**Date:** 2025-01-13

**Decision Makers:** Specular Core Team

## Context

The Specular codebase originally used unstructured error handling with `fmt.Errorf` and `errors.New`, leading to several challenges in error tracking, debugging, and user experience.

### Problems with Unstructured Errors

1. **No Error Categorization**: All errors were plain strings without machine-readable codes
2. **Inconsistent Error Messages**: No standard format across the codebase
3. **Poor Debuggability**: Difficult to track error types and frequency in logs
4. **Limited Context**: No structured way to attach metadata to errors
5. **No User Guidance**: Errors lacked actionable suggestions for resolution
6. **Hard to Test**: Couldn't reliably test specific error types with `errors.Is`

### Example Issues

```go
// Before: Unstructured errors
func (p Priority) Validate() error {
    switch p {
    case PriorityP0, PriorityP1, PriorityP2:
        return nil
    default:
        return fmt.Errorf("invalid priority %q: must be P0, P1, or P2", string(p))
    }
}

// Problems:
// - No error code for categorization
// - No suggestions for the user
// - Can't distinguish from other validation errors
// - Hard to test with errors.Is
```

### Requirements

1. Machine-readable error codes for categorization and tracking
2. Consistent error format across all packages
3. Support for error wrapping and unwrapping (errors.Unwrap)
4. Ability to attach structured context to errors
5. User-friendly suggestions for error resolution
6. Documentation links for complex errors
7. Backward compatibility with existing error handling
8. 100% test coverage for error package
9. Integration with errors.Is and errors.As

## Decision

**We will implement a structured error handling system using the `SpecularError` type in the `internal/errors` package.**

### Error Code System

Error codes follow a hierarchical naming scheme:

```
[CATEGORY]-[NUMBER]

Categories:
- SPEC-001 to SPEC-099: Specification errors
- POLICY-001 to POLICY-099: Policy errors
- PLAN-001 to PLAN-099: Planning errors
- INTERVIEW-001 to INTERVIEW-099: Interview errors
- PROVIDER-001 to PROVIDER-099: Provider errors
- EXEC-001 to EXEC-099: Execution errors
- DRIFT-001 to DRIFT-099: Drift detection errors
- IO-001 to IO-099: File I/O errors
- DOM-001 to DOM-099: Domain validation errors
```

### SpecularError Type

```go
type SpecularError struct {
    Code        ErrorCode                // Machine-readable error code
    Message     string                   // Human-readable error message
    Suggestions []string                 // Actionable suggestions for resolution
    DocsURL     string                   // Documentation link
    Cause       error                    // Wrapped error (optional)
}
```

**Features:**
- Implements `error` interface
- Implements `Unwrap()` for error chains
- Fluent API with `WithSuggestion()`, `WithSuggestions()`, `WithDocs()`
- Helper constructors for common error types

### Domain Error Codes

For domain value object validation:

```go
const (
    ErrCodeDomainInvalid          ErrorCode = "DOM-001" // Generic domain validation error
    ErrCodeDomainPriorityInvalid  ErrorCode = "DOM-002" // Invalid priority value
    ErrCodeDomainIDEmpty          ErrorCode = "DOM-003" // ID cannot be empty
    ErrCodeDomainIDTooLong        ErrorCode = "DOM-004" // ID exceeds maximum length
    ErrCodeDomainIDInvalidFormat  ErrorCode = "DOM-005" // ID format invalid
    ErrCodeDomainIDInvalidStart   ErrorCode = "DOM-006" // ID must start with letter
    ErrCodeDomainIDConsecutive    ErrorCode = "DOM-007" // ID has consecutive hyphens
    ErrCodeDomainIDTrailing       ErrorCode = "DOM-008" // ID ends with hyphen
    ErrCodeDomainValueObjectError ErrorCode = "DOM-099" // Generic value object error
)
```

### Helper Functions

```go
// Creating new errors
func New(code ErrorCode, message string) *SpecularError
func Newf(code ErrorCode, format string, args ...interface{}) *SpecularError

// Wrapping existing errors
func Wrap(code ErrorCode, message string, cause error) *SpecularError
func Wrapf(code ErrorCode, format string, args ...interface{}) *SpecularError

// Common error types
func NewDomainInvalidPriorityError(value string) *SpecularError
func NewDomainIDEmptyError(idType string) *SpecularError
func NewDomainIDTooLongError(idType string, id string, maxLength int) *SpecularError
func NewDomainIDInvalidFormatError(idType string, id string) *SpecularError
func NewDomainIDConsecutiveHyphensError(idType string, id string) *SpecularError
func NewDomainIDTrailingHyphenError(idType string, id string) *SpecularError
```

### Usage Example

```go
// After: Structured errors with helpful information
func (p Priority) Validate() error {
    switch p {
    case PriorityP0, PriorityP1, PriorityP2:
        return nil
    default:
        return errors.NewDomainInvalidPriorityError(string(p))
    }
}

// Error output includes:
// [DOM-002] invalid priority "P99": must be P0, P1, or P2
//
// Suggestions:
//   • Use P0 for critical features
//   • Use P1 for important features
//   • Use P2 for nice-to-have features
```

### Migration Strategy

1. **Phase 1 (Complete)**: Create `internal/errors` package with SpecularError type
2. **Phase 2 (Complete)**: Add domain validation error codes and helpers
3. **Phase 3 (Complete)**: Migrate domain package to use structured errors
4. **Phase 4 (Future)**: Gradually migrate other packages as needed
5. **Phase 5 (Future)**: Add observability integration (error tracking, metrics)

## Consequences

### Benefits

1. **Better Error Tracking**:
   - Machine-readable error codes enable categorization in logs
   - Can track error frequency and types in production
   - Easier to identify and prioritize issues

2. **Improved User Experience**:
   - Actionable suggestions help users resolve errors
   - Documentation links provide detailed guidance
   - Consistent error format across the application

3. **Enhanced Debuggability**:
   - Error codes make it easy to search codebase
   - Structured context provides debugging clues
   - Error wrapping preserves call chain information

4. **Better Testing**:
   - Can test specific error types with `errors.Is`
   - Helper functions ensure consistent error creation
   - 100% test coverage validates error behavior

5. **Maintainability**:
   - Centralized error definitions in one package
   - Helper functions reduce code duplication
   - Clear documentation of error types

6. **Observability Ready**:
   - Error codes enable metric aggregation
   - Structured context supports structured logging
   - Can integrate with error tracking services (Sentry, Rollbar)

### Trade-offs

1. **Additional Abstraction**:
   - More complex than simple `fmt.Errorf`
   - Requires learning error code system
   - **Mitigation**: Helper functions simplify common cases

2. **Migration Cost**:
   - Existing code needs gradual migration
   - Tests may need updates for error assertions
   - **Mitigation**: Backward compatible, migrate incrementally

3. **Error Code Management**:
   - Need to maintain error code registry
   - Risk of code conflicts if not coordinated
   - **Mitigation**: Numbered ranges per category, documented in ADR

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Error code conflicts | Medium | Reserved ranges per category |
| Migration incomplete | Low | Backward compatible, no breaking changes |
| Over-engineering | Low | Only used where value is clear |
| Documentation drift | Medium | Error codes documented in code comments |

## Implementation Details

### Test Coverage

The error package achieves **100% test coverage** with comprehensive tests for:
- Error creation (New, Wrap, helpers)
- Error formatting and messages
- Error unwrapping and chaining
- Suggestions and documentation links
- All helper functions
- Error comparison with `errors.Is`

```bash
$ go test ./internal/errors -v -cover
=== RUN   TestNew
--- PASS: TestNew (0.00s)
...
PASS
coverage: 100.0% of statements
```

### Domain Package Integration

All domain value objects now use structured errors:

```go
// priority.go
func (p Priority) Validate() error {
    switch p {
    case PriorityP0, PriorityP1, PriorityP2:
        return nil
    default:
        return errors.NewDomainInvalidPriorityError(string(p))
    }
}

// feature_id.go
func (f FeatureID) Validate() error {
    s := string(f)
    if s == "" {
        return errors.NewDomainIDEmptyError("feature ID")
    }
    if len(s) > maxFeatureIDLength {
        return errors.NewDomainIDTooLongError("feature ID", s, maxFeatureIDLength)
    }
    // ... more validation with structured errors
}

// task_id.go - similar pattern
```

### Error Testing Example

```go
func TestValidationErrors(t *testing.T) {
    // Test error code matching
    err := errors.NewDomainInvalidPriorityError("P99")

    // Can use errors.Is with error codes
    if !errors.Is(err, errors.ErrCodeDomainPriorityInvalid) {
        t.Error("Expected domain priority invalid error")
    }

    // Error message includes code and suggestions
    errStr := err.Error()
    assert.Contains(t, errStr, "[DOM-002]")
    assert.Contains(t, errStr, "Use P0 for critical features")
}
```

## Alternatives Considered

### Alternative 1: No Structured Errors

**Pros:**
- Simplest approach, no abstraction
- Familiar to all Go developers

**Cons:**
- No error categorization
- Poor user experience
- Difficult to track in production
- Hard to test specific error types

**Decision:** Rejected - benefits don't outweigh costs for production system

### Alternative 2: Third-Party Error Library

**Options Evaluated:**
- `github.com/pkg/errors` - Error wrapping library
- `github.com/hashicorp/go-multierror` - Multiple error handling
- `github.com/cockroachdb/errors` - CockroachDB's error library

**Pros:**
- Battle-tested implementations
- Rich feature sets
- Community support

**Cons:**
- External dependency
- May include unnecessary features
- Less control over error format
- Doesn't match Specular's needs exactly

**Decision:** Rejected - Custom implementation provides exactly what we need without dependencies

### Alternative 3: Error Interface with Types

```go
type ValidationError interface {
    error
    ValidationError() // Marker method
}

type PriorityError struct {
    Value string
}
```

**Pros:**
- Type-safe error handling
- Compile-time checks

**Cons:**
- More boilerplate per error type
- Harder to add suggestions/docs
- No error codes for tracking
- Complex error hierarchy

**Decision:** Rejected - Less flexible than SpecularError approach

## References

### Internal Documentation

- Error package: `internal/errors/errors.go`
- Error tests: `internal/errors/errors_test.go`
- Domain integration: `internal/domain/*.go`

### External Resources

- [Go Error Handling Best Practices](https://go.dev/blog/error-handling-and-go)
- [Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- [Error Handling in Go](https://earthly.dev/blog/golang-errors/)

### Related ADRs

- [ADR 0006: Domain Value Objects](./0006-domain-value-objects.md) - Domain validation errors use structured error system

## Migration Guide

### For New Code

Always use structured errors for new error creation:

```go
// ✅ Good: Structured error with code and suggestions
return errors.NewDomainIDEmptyError("feature ID")

// ❌ Bad: Plain string error
return fmt.Errorf("feature ID cannot be empty")
```

### For Existing Code

Migrate gradually, prioritizing:
1. **High Priority**: Domain validation, user-facing errors
2. **Medium Priority**: Command errors, provider errors
3. **Low Priority**: Internal utility errors

### Testing Structured Errors

```go
// Test error type
err := someFunc()
if !errors.Is(err, errors.ErrCodeDomainPriorityInvalid) {
    t.Error("Expected priority invalid error")
}

// Test error message content
assert.Contains(t, err.Error(), "[DOM-002]")
assert.Contains(t, err.Error(), "Suggestions:")
```

## Success Metrics

1. **Test Coverage**: 100% for error package ✅
2. **Domain Migration**: All domain value objects migrated ✅
3. **Error Consistency**: All domain errors use error codes ✅
4. **User Feedback**: Error messages include actionable suggestions ✅
5. **Observability**: Error codes enable production tracking (future)

## Status

**Accepted** - Implemented in internal/errors package with 100% test coverage.
Domain package fully migrated to use structured errors. Other packages will migrate gradually as needed.

## Appendix A: Complete Error Code Registry

### Specification Errors (SPEC-001 to SPEC-099)
- `SPEC-001`: Spec file not found
- `SPEC-002`: Spec validation failed
- `SPEC-003`: Spec unmarshal error
- `SPEC-004`: Spec marshal error
- `SPEC-005`: SpecLock file not found
- `SPEC-006`: SpecLock invalid
- `SPEC-007`: Spec hash mismatch

### Policy Errors (POLICY-001 to POLICY-099)
- `POLICY-001`: Policy file not found
- `POLICY-002`: Policy validation failed
- `POLICY-003`: Policy violation
- `POLICY-004`: Required tool missing
- `POLICY-005`: Docker image denied
- `POLICY-006`: Network access denied

### Plan Errors (PLAN-001 to PLAN-099)
- `PLAN-001`: Plan file not found
- `PLAN-002`: Plan validation failed
- `PLAN-003`: Drift detected
- `PLAN-004`: Task missing
- `PLAN-005`: Cyclic dependency

### Interview Errors (INTERVIEW-001 to INTERVIEW-099)
- `INTERVIEW-001`: Unknown preset
- `INTERVIEW-002`: Already started
- `INTERVIEW-003`: Not complete
- `INTERVIEW-004`: Validation failed
- `INTERVIEW-005`: Answer required
- `INTERVIEW-006`: Answer invalid

### Provider Errors (PROVIDER-001 to PROVIDER-099)
- `PROVIDER-001`: Provider not found
- `PROVIDER-002`: Configuration error
- `PROVIDER-003`: Authentication failed
- `PROVIDER-004`: API error
- `PROVIDER-005`: Rate limit exceeded
- `PROVIDER-006`: Timeout
- `PROVIDER-007`: Model not found

### Execution Errors (EXEC-001 to EXEC-099)
- `EXEC-001`: Docker not available
- `EXEC-002`: Image pull failed
- `EXEC-003`: Container failed
- `EXEC-004`: Timeout
- `EXEC-005`: Resource limit

### Drift Errors (DRIFT-001 to DRIFT-099)
- `DRIFT-001`: Plan-spec drift
- `DRIFT-002`: Code-contract drift
- `DRIFT-003`: Infrastructure-policy drift

### File I/O Errors (IO-001 to IO-099)
- `IO-001`: File not found
- `IO-002`: File read failed
- `IO-003`: File write failed
- `IO-004`: Directory operation failed
- `IO-005`: File unmarshal error
- `IO-006`: File marshal error

### Domain Validation Errors (DOM-001 to DOM-099)
- `DOM-001`: Generic domain validation error
- `DOM-002`: Invalid priority value
- `DOM-003`: ID cannot be empty
- `DOM-004`: ID too long
- `DOM-005`: Invalid ID format
- `DOM-006`: ID must start with letter
- `DOM-007`: Consecutive hyphens in ID
- `DOM-008`: ID ends with hyphen
- `DOM-099`: Generic value object error
