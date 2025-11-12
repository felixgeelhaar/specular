# Integration Test Strategy

This document outlines the strategy for integration and end-to-end testing to cover functionality that cannot be tested with unit tests.

## Current State

### Unit Test Coverage (Completed)

Successfully improved test coverage through unit tests focusing on:
- **Utility functions**: Getters, setters, serialization methods
- **Data structures**: Config, Result, Approval, Attestation, Manifest
- **Error handling**: BundleError, RegistryError with proper interfaces
- **Pure business logic**: Functions without external dependencies

**Results**: +6% coverage improvement, 63 new test functions, 23 functions at 100% coverage

### Coverage Gaps Requiring Integration Tests

The following packages have significant coverage gaps that require integration testing:

#### internal/detect (38.5% coverage)
**Gap Analysis**: 10 functions at 0% coverage, all using `exec.Command()`

**Functions requiring integration tests:**
- `DetectAll()` - Main orchestration function
- `detectDocker()`, `detectPodman()` - Container runtime detection
- `detectOllama()`, `detectClaude()`, `detectOpenAI()`, `detectGemini()`, `detectAnthropic()` - AI provider detection
- `detectProviderWithCLI()` - Generic CLI provider detection
- `detectGit()` - Git repository information

**Testing Challenges:**
- Requires Docker/Podman installed and running
- Requires AI CLI tools (ollama, claude, etc.)
- Requires Git repository context
- Environment-dependent behavior

#### internal/builder (coverage unknown)
**Expected Gaps**: Bundle building workflow with file I/O

**Functions likely requiring integration tests:**
- Bundle tarball creation
- File copying and archiving
- Manifest generation with checksums
- Integration with internal/bundle package

#### internal/extractor (coverage unknown)
**Expected Gaps**: Bundle extraction workflow with file I/O

**Functions likely requiring integration tests:**
- OCI artifact pulling
- Tarball extraction
- File system operations
- Integrity verification

## Integration Test Strategy

### Phase 1: Detection Integration Tests

**Objective**: Test detection functions with actual tools installed

#### Test Setup Requirements

```go
// tests/integration/detect_test.go
// +build integration

package integration_test

import (
    "os"
    "os/exec"
    "testing"

    "github.com/felixgeelhaar/specular/internal/detect"
)

// TestDetectDocker tests Docker detection with real Docker installation
func TestDetectDocker(t *testing.T) {
    // Check if Docker is available in test environment
    if _, err := exec.LookPath("docker"); err != nil {
        t.Skip("Docker not available in test environment")
    }

    runtime := detect.detectDocker()

    // Verify detection worked
    if !runtime.Available {
        t.Error("Docker should be detected as available")
    }

    // Version should be populated
    if runtime.Version == "" {
        t.Error("Docker version should be detected")
    }
}
```

#### Test Environment Setup

**Option 1: Local Development**
- Developers run integration tests locally with tools installed
- Use build tags to separate unit tests from integration tests
- `go test -tags=integration ./tests/integration/...`

**Option 2: CI/CD with Tool Installation**
```yaml
# .github/workflows/integration-tests.yml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install test dependencies
        run: |
          # Install Docker (usually pre-installed on GitHub runners)
          docker --version

          # Install Ollama
          curl -fsSL https://ollama.ai/install.sh | sh

      - name: Run integration tests
        run: go test -v -tags=integration ./tests/integration/...
```

**Option 3: Docker-in-Docker for Controlled Environment**
```dockerfile
# tests/integration/Dockerfile
FROM golang:1.21

# Install Docker CLI
RUN apt-get update && apt-get install -y \
    docker.io \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Ollama
RUN curl -fsSL https://ollama.ai/install.sh | sh

WORKDIR /app
COPY . .

CMD ["go", "test", "-v", "-tags=integration", "./tests/integration/..."]
```

#### Test Organization

```
tests/
├── integration/
│   ├── detect/
│   │   ├── docker_test.go
│   │   ├── podman_test.go
│   │   ├── ollama_test.go
│   │   ├── git_test.go
│   │   └── all_test.go
│   ├── builder/
│   │   ├── build_test.go
│   │   └── manifest_test.go
│   ├── extractor/
│   │   ├── extract_test.go
│   │   └── verify_test.go
│   └── e2e/
│       ├── auto_mode_test.go
│       ├── bundle_lifecycle_test.go
│       └── checkpoint_test.go
└── fixtures/
    ├── test-bundle-1/
    ├── test-bundle-2/
    └── test-repos/
```

### Phase 2: Builder Integration Tests

**Objective**: Test bundle building with file system operations

#### Test Approach

```go
// tests/integration/builder/build_test.go
// +build integration

func TestBuildBundle(t *testing.T) {
    // Create temp directory for test
    tmpDir := t.TempDir()

    // Set up test bundle structure
    setupTestBundle(t, tmpDir)

    // Build bundle
    bundlePath := filepath.Join(tmpDir, "test.bundle")
    err := builder.Build(&builder.Config{
        SpecFile:   filepath.Join(tmpDir, "spec.yaml"),
        OutputPath: bundlePath,
    })

    if err != nil {
        t.Fatalf("Build failed: %v", err)
    }

    // Verify bundle was created
    if _, err := os.Stat(bundlePath); err != nil {
        t.Error("Bundle file not created")
    }

    // Verify bundle contents
    verifyBundleContents(t, bundlePath)
}
```

#### Test Fixtures

```yaml
# tests/fixtures/test-bundle-1/spec.yaml
schema: specular.dev/spec/v1
metadata:
  name: test-bundle
  version: 1.0.0
  description: Test bundle for integration tests

actions:
  - name: test-action
    type: exec
    config:
      command: echo
      args: ["Hello, World!"]
```

### Phase 3: Extractor Integration Tests

**Objective**: Test bundle extraction and verification

#### Test Approach

```go
// tests/integration/extractor/extract_test.go
// +build integration

func TestExtractBundle(t *testing.T) {
    // Use pre-built test bundle
    bundlePath := "../../fixtures/test-bundle-1.tar.gz"

    // Create temp extraction directory
    extractDir := t.TempDir()

    // Extract bundle
    err := extractor.Extract(&extractor.Config{
        BundlePath: bundlePath,
        OutputDir:  extractDir,
    })

    if err != nil {
        t.Fatalf("Extract failed: %v", err)
    }

    // Verify extracted files
    expectedFiles := []string{
        "manifest.json",
        "spec.yaml",
        "spec.lock.json",
    }

    for _, file := range expectedFiles {
        path := filepath.Join(extractDir, file)
        if _, err := os.Stat(path); err != nil {
            t.Errorf("Expected file not found: %s", file)
        }
    }
}
```

### Phase 4: End-to-End Tests

**Objective**: Test complete workflows from start to finish

#### E2E Test Scenarios

##### Scenario 1: Auto Mode Workflow
```go
// tests/integration/e2e/auto_mode_test.go
// +build integration

func TestAutoModeComplete(t *testing.T) {
    // Set up test repository
    repoDir := setupTestRepository(t)

    // Run auto mode
    result := runSpecular(t, "auto", "--scope", "detect", "--max-steps", "5")

    // Verify spec was generated
    specPath := filepath.Join(repoDir, "spec.yaml")
    if _, err := os.Stat(specPath); err != nil {
        t.Error("Spec file not generated")
    }

    // Verify plan was generated
    planPath := filepath.Join(repoDir, "plan.json")
    if _, err := os.Stat(planPath); err != nil {
        t.Error("Plan file not generated")
    }

    // Verify exit code indicates success
    if result.ExitCode != 0 {
        t.Errorf("Auto mode failed with exit code %d", result.ExitCode)
    }
}
```

##### Scenario 2: Bundle Lifecycle
```go
// tests/integration/e2e/bundle_lifecycle_test.go
// +build integration

func TestBundleLifecycle(t *testing.T) {
    tmpDir := t.TempDir()

    // Step 1: Build bundle
    t.Run("build", func(t *testing.T) {
        result := runSpecular(t, "bundle", "build",
            "--spec", "fixtures/test-bundle-1/spec.yaml",
            "--output", filepath.Join(tmpDir, "test.bundle"))

        if result.ExitCode != 0 {
            t.Fatalf("Bundle build failed: %s", result.Stderr)
        }
    })

    // Step 2: Verify bundle
    t.Run("verify", func(t *testing.T) {
        result := runSpecular(t, "bundle", "verify",
            filepath.Join(tmpDir, "test.bundle"))

        if result.ExitCode != 0 {
            t.Fatalf("Bundle verify failed: %s", result.Stderr)
        }
    })

    // Step 3: Apply bundle
    t.Run("apply", func(t *testing.T) {
        result := runSpecular(t, "bundle", "apply",
            filepath.Join(tmpDir, "test.bundle"),
            "--dry-run")

        if result.ExitCode != 0 {
            t.Fatalf("Bundle apply failed: %s", result.Stderr)
        }
    })
}
```

##### Scenario 3: Checkpoint Resume
```go
// tests/integration/e2e/checkpoint_test.go
// +build integration

func TestCheckpointResume(t *testing.T) {
    tmpDir := t.TempDir()

    // Step 1: Run auto mode with checkpoint
    t.Run("create checkpoint", func(t *testing.T) {
        result := runSpecular(t, "auto",
            "--scope", "detect",
            "--max-steps", "2", // Stop early
            "--checkpoint-dir", tmpDir)

        // Verify checkpoint was created
        checkpoints, _ := filepath.Glob(filepath.Join(tmpDir, "checkpoint-*.json"))
        if len(checkpoints) == 0 {
            t.Fatal("No checkpoint created")
        }
    })

    // Step 2: Resume from checkpoint
    t.Run("resume from checkpoint", func(t *testing.T) {
        result := runSpecular(t, "auto", "resume",
            "--checkpoint-dir", tmpDir)

        if result.ExitCode != 0 {
            t.Fatalf("Resume failed: %s", result.Stderr)
        }
    })
}
```

## Test Execution Strategy

### Build Tags for Test Separation

```go
// Unit tests - no build tag required
// Run with: go test ./...

// Integration tests - require 'integration' build tag
// +build integration
// Run with: go test -tags=integration ./tests/integration/...

// E2E tests - require 'e2e' build tag
// +build e2e
// Run with: go test -tags=e2e ./tests/integration/e2e/...
```

### CI/CD Pipeline Configuration

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run unit tests
        run: go test -v -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v3

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - name: Install dependencies
        run: |
          docker --version
          git --version
      - name: Run integration tests
        run: go test -v -tags=integration ./tests/integration/...

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - name: Build specular
        run: go build -o bin/specular ./cmd/specular
      - name: Run E2E tests
        run: go test -v -tags=e2e -timeout=30m ./tests/integration/e2e/...
```

## Mock vs. Real Dependencies

### When to Use Mocks

✅ **Use mocks for:**
- External API calls (OpenAI, Anthropic, Gemini APIs)
- Network operations (OCI registry pulls)
- Time-dependent operations (for deterministic tests)
- Rate-limited services

### When to Use Real Dependencies

✅ **Use real dependencies for:**
- File system operations (with temp directories)
- Local tool detection (Docker, Git, etc.)
- Command execution (with controlled test environments)
- Database operations (with test databases)

## Test Data Management

### Fixture Organization

```
tests/fixtures/
├── bundles/           # Pre-built test bundles
│   ├── valid-bundle-1.tar.gz
│   ├── valid-bundle-2.tar.gz
│   └── invalid-bundle.tar.gz
├── specs/             # Test spec files
│   ├── minimal-spec.yaml
│   ├── complete-spec.yaml
│   └── invalid-spec.yaml
├── policies/          # Test policy files
│   ├── permissive-policy.rego
│   └── strict-policy.rego
└── repos/             # Test Git repositories
    ├── go-project/
    ├── node-project/
    └── python-project/
```

### Fixture Generation Scripts

```bash
#!/bin/bash
# scripts/generate-test-fixtures.sh

# Generate test bundles
echo "Generating test bundles..."
go run ./tests/fixtures/generate-bundles.go

# Create test Git repositories
echo "Creating test repositories..."
./tests/fixtures/create-test-repos.sh

# Build sample projects
echo "Building sample projects..."
./tests/fixtures/build-samples.sh
```

## Performance Testing

### Load Testing

```go
// tests/performance/load_test.go
// +build performance

func BenchmarkDetectAll(b *testing.B) {
    for i := 0; i < b.N; i++ {
        detect.DetectAll()
    }
}

func BenchmarkBuildBundle(b *testing.B) {
    tmpDir := b.TempDir()

    for i := 0; i < b.N; i++ {
        builder.Build(&builder.Config{
            SpecFile:   "fixtures/specs/minimal-spec.yaml",
            OutputPath: filepath.Join(tmpDir, fmt.Sprintf("bundle-%d.tar.gz", i)),
        })
    }
}
```

## Coverage Goals

### Target Coverage by Package Type

| Package Type | Target | Strategy |
|--------------|--------|----------|
| Pure business logic | 90%+ | Unit tests |
| With file I/O | 70%+ | Integration tests |
| With external tools | 50%+ | Integration tests with tool checks |
| E2E workflows | N/A | Scenario coverage (not line coverage) |

### Measuring Integration Test Coverage

```bash
# Run integration tests with coverage
go test -tags=integration -coverprofile=integration-coverage.out ./tests/integration/...

# Merge with unit test coverage
go tool cover -html=integration-coverage.out -o integration-coverage.html

# Compare coverage improvements
go tool cover -func=integration-coverage.out | grep "total:"
```

## Next Steps

### Immediate Priorities

1. **Set up integration test structure** (tests/integration/ directory)
2. **Create test fixtures** (sample bundles, specs, repositories)
3. **Implement detection integration tests** (Docker, Git detection)
4. **Add CI/CD integration test pipeline**

### Medium-Term Goals

1. **Builder integration tests** (bundle creation workflows)
2. **Extractor integration tests** (bundle extraction workflows)
3. **E2E test scenarios** (auto mode, bundle lifecycle)

### Long-Term Goals

1. **Performance benchmarks** (detect, build, extract operations)
2. **Chaos testing** (network failures, disk full, interrupted operations)
3. **Security testing** (malicious bundles, privilege escalation attempts)
4. **Contract testing** (provider interface compatibility)

---

Last Updated: 2025-01-12
