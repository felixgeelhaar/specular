# ADR 0006: Domain Value Objects for Type Safety

**Status:** Accepted

**Date:** 2025-01-10

**Decision Makers:** Specular Core Team

## Context

The Specular codebase originally used raw `string` types for domain identifiers such as task IDs, feature IDs, and priority levels. This approach led to several issues that compromised type safety and code quality.

### Problems with String-Based Identifiers

1. **Type Confusion**: Task IDs and Feature IDs were both strings, allowing accidental mixing
2. **No Validation**: Invalid values like empty strings or malformed IDs could propagate
3. **Lack of Domain Semantics**: String types don't convey business meaning
4. **Poor IDE Support**: No autocomplete or type-specific methods
5. **Maintenance Burden**: Validation logic scattered across codebase

### Example Issues

```go
// Before: Easy to accidentally swap IDs
func ProcessTask(taskID string, featureID string) {
    // Could accidentally pass featureID where taskID expected!
    DoSomething(featureID, taskID) // Compiler doesn't catch this
}

// Empty or invalid strings accepted everywhere
taskID := ""  // Valid but meaningless
priority := "P99"  // Invalid but type system allows it
```

### Requirements

1. Prevent mixing of different ID types at compile time
2. Validate values at construction time
3. Provide clear domain semantics
4. Maintain JSON serialization compatibility
5. Support pattern matching and validation
6. Achieve high test coverage (>95%)
7. Zero runtime overhead for valid values

## Decision

**We will introduce strongly-typed value objects for domain identifiers in an `internal/domain` package.**

### Value Objects Created

1. **TaskID** - Unique identifier for tasks
2. **FeatureID** - Unique identifier for features
3. **Priority** - Task/feature priority (P0, P1, P2)

Each value object:
- Is an immutable string-based type
- Validates input at construction time
- Provides a `String()` method for conversion
- Has comprehensive unit tests
- Returns descriptive errors for invalid input

### Type Definitions

```go
package domain

type TaskID string      // e.g., "task-001"
type FeatureID string   // e.g., "feat-user-auth"
type Priority string    // P0, P1, P2
```

## Implementation

### Phase 1: Value Object Creation

Created `internal/domain` package with three value objects:

**TaskID** (`task_id.go`):
```go
type TaskID string

var (
    TaskIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
    ErrInvalidTaskID = errors.New("invalid task ID format")
)

// NewTaskID validates and creates a TaskID
func NewTaskID(id string) (TaskID, error) {
    if id == "" {
        return "", fmt.Errorf("%w: empty task ID", ErrInvalidTaskID)
    }
    if !TaskIDRegex.MatchString(id) {
        return "", fmt.Errorf("%w: must match pattern %s",
            ErrInvalidTaskID, TaskIDRegex)
    }
    return TaskID(id), nil
}

func (t TaskID) String() string {
    return string(t)
}

func (t TaskID) Validate() error {
    if t == "" {
        return fmt.Errorf("%w: empty task ID", ErrInvalidTaskID)
    }
    return nil
}
```

**FeatureID** (`feature_id.go`): Similar structure to TaskID

**Priority** (`priority.go`):
```go
type Priority string

const (
    PriorityP0 Priority = "P0"  // Critical
    PriorityP1 Priority = "P1"  // High
    PriorityP2 Priority = "P2"  // Medium
)

func NewPriority(p string) (Priority, error) {
    priority := Priority(p)
    if err := priority.Validate(); err != nil {
        return "", err
    }
    return priority, nil
}
```

### Phase 2: Core Type Integration

Integrated value objects into core domain types:

**spec.Feature**:
- Changed `ID` from `string` to `domain.FeatureID`
- Changed `Priority` from `string` to `domain.Priority`

**plan.Task**:
- Changed `ID` from `string` to `domain.TaskID`
- Changed `FeatureID` from `string` to `domain.FeatureID`
- Changed `Priority` from `string` to `domain.Priority`

### Phase 3: Loader Validation

Added validation to loaders:

**spec.Loader** (`internal/spec/loader.go`):
```go
func (l *Loader) LoadFromFile(path string) (*ProductSpec, error) {
    // ... parse YAML ...

    // Validate FeatureIDs
    for i, f := range result.Features {
        if _, err := domain.NewFeatureID(f.ID.String()); err != nil {
            return nil, fmt.Errorf("invalid feature ID at index %d: %w", i, err)
        }
    }
    return result, nil
}
```

### Phase 4: Test Updates

Updated all test files to use value object constructors:

```go
// Before:
task := plan.Task{
    ID:        "task-001",
    FeatureID: "feat-auth",
    Priority:  "P0",
}

// After:
task := plan.Task{
    ID:        domain.TaskID("task-001"),
    FeatureID: domain.FeatureID("feat-auth"),
    Priority:  domain.PriorityP0,
}
```

### Phase 5: Remaining Package Integration

Integrated domain types into remaining packages:

**drift package** (`internal/drift`):
- Updated `Finding.FeatureID` to use `domain.FeatureID`
- Removed unnecessary `.String()` calls

**router package** (`internal/router`):
- Updated `Usage.TaskID` to use `domain.TaskID`
- Updated `GenerateRequest.TaskID` to use `domain.TaskID`
- Added `.String()` calls for map[string]string metadata

**cmd package** (`internal/cmd`):
- Updated generate command to use `domain.TaskID` constructor

### JSON Serialization

Value objects serialize naturally as strings:

```go
type Task struct {
    ID domain.TaskID `json:"id"`  // Serializes as "task-001"
}

// Go's default JSON encoding handles string-based types automatically
```

### Test Coverage

Achieved **98% test coverage** for domain package:

```bash
$ go test -cover ./internal/domain/
ok  	github.com/felixgeelhaar/specular/internal/domain	0.431s	coverage: 98.0% of statements
```

Test files:
- `feature_id_test.go` - 100+ lines of tests
- `task_id_test.go` - 100+ lines of tests
- `priority_test.go` - 95+ lines of tests

## Consequences

### Positive

- ✅ **Compile-Time Type Safety**: Cannot mix TaskID and FeatureID
- ✅ **Validation at Boundaries**: Invalid values caught at construction
- ✅ **Self-Documenting Code**: Types convey domain meaning
- ✅ **Better IDE Support**: Autocomplete for domain-specific methods
- ✅ **Centralized Validation**: All validation logic in one place
- ✅ **Comprehensive Tests**: 98% test coverage
- ✅ **Zero Runtime Overhead**: String-based types have no performance cost
- ✅ **JSON Compatible**: Transparent serialization/deserialization
- ✅ **Error Traceability**: Clear error messages with validation context

### Negative

- ❌ **Constructor Verbosity**: Must use constructors like `domain.TaskID("task-001")`
- ❌ **String Conversion**: Need `.String()` for map keys and string operations
- ❌ **Migration Effort**: Required updates across entire codebase

### Mitigations

- **Constructor Shorthand**: Use direct type conversion `domain.TaskID("id")` for known-valid values in tests
- **Helper Methods**: Value objects provide `.String()` for easy conversion
- **Incremental Migration**: Completed in phases with full test coverage at each step

### Trade-offs

- Chose compile-time safety over runtime flexibility
- Chose explicit validation over implicit string handling
- Chose type safety over brevity
- Chose domain clarity over simplicity

## Migration Statistics

**10 commits** implementing the refactoring:

1. `feat: add FeatureID and TaskID value objects`
2. `feat: add domain validation to Feature and related types`
3. `feat: add domain validation to Task and Plan types`
4. `refactor(domain): integrate FeatureID and Priority value objects`
5. `refactor(plan): integrate domain.TaskID, domain.Priority into Task struct`
6. `refactor(loaders): Add validation to spec and plan loaders`
7. `fix(tests): update test files for domain type integration`
8. `refactor(drift): integrate domain.FeatureID into drift package`
9. `refactor(router): integrate domain.TaskID into router package`
10. `refactor(cmd): integrate domain.TaskID into generate command`

**Packages Updated**:
- `internal/domain` (created)
- `internal/spec`
- `internal/plan`
- `internal/drift`
- `internal/router`
- `internal/cmd`
- All test files

**Test Results**:
- All 23 test packages passing
- 98% coverage on domain package
- Zero compilation errors
- Zero runtime regressions

## Related Decisions

- ADR 0001: Spec Lock File Format (uses FeatureID for features map)
- ADR 0005: Drift Detection Approach (uses FeatureID for findings)

## References

- [Domain-Driven Design by Eric Evans](https://www.domainlanguage.com/ddd/)
- [Value Objects - Martin Fowler](https://martinfowler.com/bliki/ValueObject.html)
- [Go Type System Best Practices](https://go.dev/doc/effective_go#types)
- [Effective Go - Named Types](https://go.dev/doc/effective_go#named-types)
