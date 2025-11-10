# Improvement Roadmap

**Last Updated:** 2025-01-10
**Project:** Specular - AI-Native Development Framework
**Current Quality Grade:** B+ (trending toward A)

## Recent Accomplishments ✅

### Domain-Driven Design Refactoring (Completed 2025-01-10)
- **12 commits** implementing full DDD refactoring
- Created `internal/domain` package with 3 value objects
- Achieved **98% test coverage** on domain package
- Integrated across 6 packages
- Documented in ADR 0006
- Zero regressions, all tests passing

## Priority Roadmap

### 🔥 High Priority (Next 2 Weeks)

#### 0. Autonomous Agent Mode Implementation
**Current:** Manual CLI tool requiring 5+ commands per workflow
**Target:** Single-command autonomous workflows with approval gates
**Effort:** High (7-9 weeks)
**Impact:** Critical

**See:** [AGENT_MODE_ROADMAP.md](AGENT_MODE_ROADMAP.md) and [ADR 0007](adr/0007-autonomous-agent-mode.md)

**Overview:**
Transform Specular into an autonomous agent system (similar to Claude Code) while preserving manual control capabilities. Users will be able to run `specular auto --goal "Build a REST API"` and get working, tested code with minimal intervention.

**Key Features:**
- **Auto Mode:** Full workflow from goal to code with approval gates
- **Error Recovery:** Automatic retry with AI-powered task regeneration
- **Watch Mode:** Continuous monitoring and auto-rebuild on file changes
- **Full Autonomy:** Multi-session workflows with cost optimization

**Implementation Phases:**
1. **Phase 1 (2-3 weeks):** Auto mode foundation with approval gates
2. **Phase 2 (1-2 weeks):** Error recovery and retry logic
3. **Phase 3 (1 week):** Watch mode for continuous monitoring
4. **Phase 4 (2-3 weeks):** Full autonomy with drift-based regeneration

**Success Metrics:**
- 3x faster workflow completion
- 70%+ error recovery rate on first retry
- 90%+ developer adoption of watch mode for TDD
- 30% cost reduction via optimization

**Deliverables:**
- `specular auto` command with natural language goal parsing
- `specular watch` command for continuous validation
- Comprehensive approval gate system
- Budget tracking and cost limits
- Complete test coverage (>80%)
- Production-ready documentation

**Status:** Planning complete, ready for Phase 1 implementation

---

#### 1. Improve Infrastructure Package Testing
**Current:** 36-55% coverage
**Target:** 60-70% coverage
**Effort:** Medium
**Impact:** High

**Packages to improve:**
- `internal/bundle` (36% → 60%)
- `internal/detect` (38.5% → 60%)
- `internal/tui` (40.6% → 60%)
- `internal/exec` (54.8% → 70%)

**Action items:**
- Add unit tests for bundle registry operations
- Test project detection heuristics
- Mock terminal UI components for testing
- Add Docker execution integration tests

**Success criteria:**
- Each package reaches target coverage
- Critical paths have test coverage
- Edge cases handled with tests

---

#### 2. Add Integration Test Suite
**Current:** Minimal end-to-end testing
**Target:** Comprehensive E2E test coverage
**Effort:** High
**Impact:** Critical

**Test scenarios needed:**
```go
// Example E2E test structure
func TestE2E_FullWorkflow(t *testing.T) {
    // 1. Initialize project with spec
    // 2. Generate plan from spec
    // 3. Execute plan with Docker
    // 4. Validate output against spec
    // 5. Run drift detection
}
```

**Coverage areas:**
- Full workflow: spec → plan → execute → validate
- Multi-provider AI routing scenarios
- Checkpoint/resume functionality
- Drift detection workflows
- Bundle approval workflows

**Deliverables:**
- `internal/workflow/e2e_suite_test.go`
- Test fixtures and golden files
- CI integration for E2E tests
- Documentation for running E2E tests

---

#### 3. Performance Benchmarking
**Current:** No systematic benchmarks
**Target:** Baseline metrics for critical paths
**Effort:** Medium
**Impact:** Medium-High

**Areas to benchmark:**
```go
// Priority benchmarks
BenchmarkRouter_SelectModel
BenchmarkPlan_Generate
BenchmarkSpec_Parse
BenchmarkDrift_Detect
BenchmarkExec_DockerRun
```

**Goals:**
- Establish performance baselines
- Identify bottlenecks
- Set performance budgets
- Enable regression detection

**Action items:**
- Add `*_benchmark_test.go` files
- Document performance expectations
- CI integration for benchmark tracking
- Profile memory and CPU usage

---

### 🎯 Medium Priority (1-2 Months)

#### 4. Property-Based Testing for Value Objects
**Benefit:** Catch edge cases automatically
**Effort:** Low
**Impact:** Medium

Use `github.com/leanovate/gopter` or `pgregory.net/rapid`:

```go
import "github.com/leanovate/gopter"

func TestTaskID_Properties(t *testing.T) {
    properties := gopter.NewProperties(nil)

    properties.Property("valid IDs always validate", prop.ForAll(
        func(id string) bool {
            taskID, err := domain.NewTaskID(id)
            if err == nil {
                return taskID.Validate() == nil
            }
            return true
        },
        gen.AlphaString(),
    ))

    properties.TestingRun(t)
}
```

**Deliverables:**
- Property tests for TaskID, FeatureID, Priority
- Fuzz testing for parsers
- Randomized test data generation

---

#### 5. Mutation Testing
**Tool:** `go-mutesting`
**Goal:** Verify test quality
**Effort:** Medium

```bash
# Install mutation testing
go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest

# Run mutation testing on domain package
go-mutesting internal/domain/...
```

**Target:** >80% mutation score on critical packages

---

#### 6. Enhanced Error Handling
**Pattern:** Structured errors with context

```go
// Example improvement
type DomainError struct {
    Code    ErrorCode
    Message string
    Context map[string]interface{}
    Cause   error
}

func (e *DomainError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
    return e.Cause
}
```

**Action items:**
- Audit all error paths
- Add structured logging for errors
- Implement error wrapping consistently
- Document error codes in ADR

---

### 🚀 Long-Term Improvements (3-6 Months)

#### 7. Observability & Monitoring
**Components needed:**
- Structured logging (zerolog or zap)
- Metrics collection (Prometheus)
- Distributed tracing (OpenTelemetry)
- Health check endpoints

```go
// Example metrics
type Metrics struct {
    PlanGenerations      prometheus.Counter
    TaskExecutions       prometheus.Counter
    DriftDetections      prometheus.Counter
    ProviderLatency     prometheus.Histogram
    ProviderErrors      prometheus.Counter
}
```

---

#### 8. Advanced Domain Patterns

##### a. Domain Events
```go
type DomainEvent interface {
    EventID() string
    OccurredAt() time.Time
    AggregateID() string
}

type FeatureCreated struct {
    ID         domain.FeatureID
    Title      string
    OccurredAt time.Time
}
```

##### b. Repository Pattern
```go
type FeatureRepository interface {
    Save(ctx context.Context, feature *spec.Feature) error
    Find(ctx context.Context, id domain.FeatureID) (*spec.Feature, error)
    FindAll(ctx context.Context) ([]*spec.Feature, error)
}
```

##### c. Domain Services
```go
type PlanningService struct {
    specRepo  SpecRepository
    taskGen   TaskGenerator
    validator Validator
}

func (s *PlanningService) CreatePlan(ctx context.Context, specID string) (*plan.Plan, error) {
    // Complex business logic coordinating multiple aggregates
}
```

---

#### 9. Caching Strategy
**Opportunities:**
- Cache AI responses for identical prompts
- Cache Docker image pulls
- Cache spec lock file computations
- Cache provider health checks

```go
type Cache interface {
    Get(ctx context.Context, key string) (interface{}, error)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}

// Implementation with Redis or in-memory
```

---

#### 10. Plugin Architecture
**Goal:** Extensible provider system

```go
type Plugin interface {
    Name() string
    Version() string
    Initialize(config map[string]interface{}) error
    Shutdown() error
}

type ProviderPlugin interface {
    Plugin
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
}
```

---

## Security Enhancements

### S1. Secrets Management
**Priority:** High
**Status:** Needed

- [ ] Integrate with secrets managers (HashiCorp Vault, AWS Secrets Manager)
- [ ] Rotate API keys automatically
- [ ] Audit secret access
- [ ] Encrypt secrets at rest

### S2. Input Validation Hardening
**Priority:** Medium
**Status:** Partially done

- [x] Domain value objects validate input
- [ ] Add fuzzing for parsers
- [ ] Validate file paths prevent traversal
- [ ] Sanitize user-provided strings

### S3. Dependency Security
**Priority:** Ongoing

```bash
# Regular security audits
go list -json -m all | nancy sleuth
govulncheck ./...
```

---

## Performance Optimizations

### P1. Concurrency Improvements
**Opportunities:**
- Parallel task execution where dependencies allow
- Concurrent provider health checks
- Parallel drift detection across features
- Streaming responses for large outputs

```go
// Example: Parallel drift detection
func DetectDriftParallel(features []spec.Feature) []Finding {
    var wg sync.WaitGroup
    findings := make(chan Finding)

    for _, feature := range features {
        wg.Add(1)
        go func(f spec.Feature) {
            defer wg.Done()
            results := detectFeatureDrift(f)
            for _, r := range results {
                findings <- r
            }
        }(feature)
    }

    // Collect results...
}
```

### P2. Memory Optimization
- Stream large file operations
- Pool frequently allocated objects
- Profile memory usage under load
- Implement resource limits

### P3. Database Query Optimization
*If/when persistence layer is added*
- Index strategy
- Query batching
- Connection pooling
- Prepared statements

---

## Developer Experience

### DX1. Enhanced CLI Output
- [ ] Progress bars for long operations
- [ ] Colored output (with NO_COLOR support)
- [ ] Structured output formats (JSON, YAML)
- [ ] Verbose/quiet modes

### DX2. Better Error Messages
```go
// Before
Error: invalid feature ID

// After
Error: Invalid feature ID "MyFeature"
  Expected format: lowercase letters, numbers, and hyphens
  Must start with a letter
  Examples: user-auth, payment-api, data-sync
```

### DX3. Developer Tools
- [ ] Debug mode with detailed logging
- [ ] Dry-run capabilities
- [ ] Explain mode (shows what will happen)
- [ ] Interactive mode for complex operations

---

## Documentation Improvements

### D1. Code Examples
- Add runnable examples to godoc
- Create example projects in `examples/`
- Tutorial-style documentation
- Video walkthroughs

### D2. API Documentation
- OpenAPI specs for any HTTP endpoints
- Generated godoc coverage >80%
- Architecture diagrams (C4 model)
- Sequence diagrams for workflows

### D3. Contributing Guide
- Development setup instructions
- Testing guidelines
- PR checklist
- Code review standards

---

## CI/CD Enhancements

### CI1. GitHub Actions Improvements
```yaml
# .github/workflows/quality.yml
- Coverage reporting with badges
- Benchmark comparisons
- Mutation testing on critical paths
- Security scanning (gosec, govulncheck)
- Linting (golangci-lint)
```

### CI2. Pre-commit Hooks
```bash
# .git/hooks/pre-commit
- Run tests on changed packages
- Format code (gofmt, goimports)
- Check for common mistakes
- Verify no debug code
```

### CI3. Release Automation
- Automated changelog generation
- Semantic versioning
- Release notes from commits
- Binary signing and verification

---

## Quality Gates for v1.0

### Must Have
- [x] Core domain logic >90% coverage
- [x] Type-safe domain models
- [x] Comprehensive ADRs
- [ ] Integration test suite
- [ ] Performance benchmarks
- [ ] Security audit completed
- [ ] Production deployment guide

### Should Have
- [ ] Overall coverage >60%
- [ ] Mutation testing >80% score
- [ ] Load testing results
- [ ] Observability setup
- [ ] Plugin architecture

### Nice to Have
- [ ] Property-based tests
- [ ] Chaos testing
- [ ] A/B testing framework
- [ ] Advanced caching
- [ ] Multi-region support

---

## Metrics to Track

### Code Quality
- Test coverage (target: 60% overall, >90% core)
- Mutation score (target: >80% critical packages)
- Cyclomatic complexity (target: <15 average)
- Code duplication (target: <3%)

### Performance
- Plan generation time (target: <5s for 10 features)
- Task execution overhead (target: <100ms)
- Memory usage (target: <500MB for typical workflow)
- Provider latency (p50, p95, p99)

### Reliability
- Test success rate (target: >99%)
- Build success rate (target: >95%)
- Mean time to recovery (MTTR)
- Error rate in production

---

## Review Schedule

### Weekly
- Test coverage trends
- CI/CD health
- Issue triage

### Monthly
- Performance benchmarks
- Security updates
- Dependency updates
- Roadmap review

### Quarterly
- Architecture review
- Technical debt assessment
- Quality metrics analysis
- User feedback integration

---

## Resources & References

### Books
- *Domain-Driven Design* - Eric Evans
- *Clean Architecture* - Robert C. Martin
- *Release It!* - Michael Nygard
- *The Go Programming Language* - Donovan & Kernighan

### Tools
- **Testing:** testify, gomock, gopter
- **Quality:** golangci-lint, gosec, govulncheck
- **Performance:** pprof, trace, benchstat
- **Monitoring:** Prometheus, Grafana, OpenTelemetry

### Community
- Go Discord community
- DDD community forums
- GitHub Discussions
- Monthly review meetings

---

## Success Metrics

**Q1 2025:**
- **Autonomous Agent Mode:** Phase 1-2 complete (auto mode + error recovery)
- Overall coverage: 45.9% → 55%
- Integration tests: 0 → 20 scenarios
- Performance baselines established
- Infrastructure packages improved
- Developer productivity: 3x faster workflows with auto mode

**Q2 2025:**
- **Autonomous Agent Mode:** Phase 3-4 complete (watch mode + full autonomy)
- Overall coverage: 55% → 60%
- Advanced domain patterns implemented
- Observability setup complete
- Plugin architecture ready
- Agent mode in production use

**Q3-Q4 2025:**
- Production-ready v1.0 release
- Advanced optimizations
- Community contributions
- Enterprise features

---

**Last Reviewed:** 2025-01-10
**Next Review:** 2025-02-10
**Owner:** Specular Core Team
