# Specular CLI MVP Action Plan
**Version:** v0.9-v1.0 Roadmap
**Updated:** 2025-11-06
**Status:** v0.8 Complete → MVP in Progress

---

## 🎯 Overview

This action plan outlines the specific tasks required to complete Specular CLI MVP (v0.9-v1.0). The foundation is solid with 73.7% test coverage and all core packages implemented. Focus now shifts to workflow refinement, error handling, and CI/CD integration.

**Current State (v0.8):**
- ✅ All 10 core packages implemented
- ✅ 4 AI providers with intelligent routing
- ✅ Multi-level drift detection with SARIF reports
- ✅ Docker-only sandbox with policy enforcement
- ✅ Comprehensive test suite with race detection

**MVP Goals (v0.9-v1.0):**
- 🎯 Polish end-to-end workflow (interview → eval)
- 🎯 Production-ready error handling and UX
- 🎯 CI/CD integration with GitHub Actions
- 🎯 Comprehensive documentation and examples
- 🎯 80%+ test coverage target

---

## 📋 Milestone 6: E2E Refinement (v0.9)

**Goal:** Transform the functional CLI into a polished, production-ready tool with excellent developer experience.

### 6.1 Workflow Polish

#### 6.1.1 End-to-End Flow Validation
**Priority:** P0
**Complexity:** 6/10
**Estimated Effort:** 3-4 days

**Tasks:**
- [ ] Create integration test for complete workflow: `interview → spec → plan → build → eval`
- [ ] Verify state transitions and file generation at each stage
- [ ] Test with all 5 interview presets (web-app, api-service, cli-tool, microservice, data-pipeline)
- [ ] Validate proper cleanup on failures (rollback, temp file removal)
- [ ] Test with multiple provider combinations (Ollama, OpenAI, Anthropic, Gemini)

**Acceptance Criteria:**
- Integration test passes with 100% success rate
- All intermediate files (spec.yaml, spec.lock.json, plan.json) generated correctly
- Workflow completes in < 5 minutes for typical project
- Zero file/resource leaks on error paths

**Test File:** `internal/workflow/e2e_test.go`

---

#### 6.1.2 Error Recovery Mechanisms
**Priority:** P0
**Complexity:** 7/10
**Estimated Effort:** 4-5 days

**Tasks:**
- [ ] Implement checkpointing for long-running operations
- [ ] Add resume capability for interrupted workflows
- [ ] Create detailed error messages with actionable suggestions
- [ ] Implement automatic retry with exponential backoff for transient failures
- [ ] Add dry-run validation before expensive operations
- [ ] Create error recovery guide in documentation

**Implementation Details:**

```go
// internal/workflow/checkpoint.go
type Checkpoint struct {
    Stage       string    `json:"stage"`        // interview, spec, plan, build, eval
    Timestamp   time.Time `json:"timestamp"`
    Files       []string  `json:"files"`        // Generated files to preserve
    State       any       `json:"state"`        // Stage-specific state
    Error       string    `json:"error,omitempty"`
}

func SaveCheckpoint(stage string, state any) error
func LoadCheckpoint(stage string) (*Checkpoint, error)
func Resume(fromStage string) error
```

**Error Message Template:**
```
Error: Docker image 'golang:1.22' not found in policy allowlist

Suggestion:
  1. Add 'golang:1.22' to .specular/policy.yaml:
     execution:
       docker:
         image_allowlist:
           - golang:1.22

  2. Or use an allowed image (see: specular policy show)

  3. Run with --dry-run to validate policy before execution

For more help: specular help build
```

**Acceptance Criteria:**
- All errors include actionable suggestions
- Resume works from any stage checkpoint
- Dry-run validates complete workflow without side effects
- Error recovery guide covers top 10 error scenarios

---

#### 6.1.3 Progress Indicators & UX
**Priority:** P1
**Complexity:** 5/10
**Estimated Effort:** 2-3 days

**Tasks:**
- [ ] Add progress bars for long operations (plan generation, build execution)
- [ ] Implement streaming output with timestamps for real-time feedback
- [ ] Show estimated time remaining for model inference
- [ ] Add color-coded output (success=green, warning=yellow, error=red)
- [ ] Create --quiet mode for CI/CD environments
- [ ] Add --verbose mode with detailed metadata (tokens, cost, latency)

**Implementation:**
```go
// internal/ui/progress.go
type ProgressTracker interface {
    Start(total int, message string)
    Increment(delta int)
    SetMessage(message string)
    Finish()
}

// Use github.com/schollz/progressbar/v3 or similar
```

**Example Output:**
```bash
$ specular build --plan plan.json --policy policy.yaml

[1/5] Validating plan...                     ✓ (0.2s)
[2/5] Checking policy compliance...          ✓ (0.1s)
[3/5] Pulling Docker images...
  └─ golang:1.22                              ✓ (12.3s)
  └─ node:22                                  ✓ (8.7s)
[4/5] Executing build tasks...
  └─ task-001: Setup backend                 ⟳ (3.2s elapsed)
      Model: claude-sonnet | Tokens: 1.2k
```

**Acceptance Criteria:**
- Progress visible for all operations > 2 seconds
- Streaming output shows real-time model generation
- Color-coded output works in terminals with color support
- Quiet mode produces minimal output suitable for scripting

**Test File:** `internal/ui/progress_test.go`

---

### 6.2 Documentation & Examples

#### 6.2.1 Getting Started Guide
**Priority:** P0
**Complexity:** 4/10
**Estimated Effort:** 2-3 days

**Tasks:**
- [ ] Create `docs/getting-started.md` with quickstart tutorial
- [ ] Add installation instructions for all platforms (macOS, Linux, Windows)
- [ ] Document provider setup for all 4 providers
- [ ] Create example workflow from scratch project
- [ ] Add troubleshooting section for common issues
- [ ] Include video walkthrough (asciinema recording)

**Structure:**
```markdown
# Getting Started with Specular CLI

## Installation
- Homebrew (macOS/Linux)
- Binary download
- Build from source

## Quick Start (5 minutes)
1. Install specular
2. Configure AI provider
3. Generate your first spec
4. Execute build workflow
5. Review drift report

## Example: Building a REST API
[Complete walkthrough]

## Troubleshooting
[Common issues and solutions]

## Next Steps
- Advanced workflows
- CI/CD integration
- Policy customization
```

**Acceptance Criteria:**
- New user can complete quickstart in < 10 minutes
- All commands work copy-paste style
- Examples cover web-app, api-service, and cli-tool presets
- Troubleshooting covers top 10 user issues

---

#### 6.2.2 Workflow Examples & Templates
**Priority:** P1
**Complexity:** 3/10
**Estimated Effort:** 2 days

**Tasks:**
- [ ] Create `examples/` directory with complete workflow templates
- [ ] Add template for each interview preset
- [ ] Include example specs, plans, and policies
- [ ] Create Makefile targets for running examples
- [ ] Document expected outputs and timing

**Directory Structure:**
```
examples/
├─ web-app/              # Full-stack web application
│  ├─ README.md
│  ├─ prd.md             # Sample PRD
│  ├─ .specular/
│  │  ├─ spec.yaml
│  │  ├─ spec.lock.json
│  │  └─ policy.yaml
│  ├─ plan.json
│  ├─ Makefile
│  └─ expected-output/
├─ api-service/          # RESTful API service
├─ cli-tool/             # Command-line tool
├─ microservice/         # Microservice architecture
└─ data-pipeline/        # Data processing pipeline
```

**Example Makefile:**
```makefile
.PHONY: all clean interview spec plan build eval

all: interview spec plan build eval

interview:
	specular interview --preset web-app --out .specular/spec.yaml

spec:
	specular spec validate --in .specular/spec.yaml
	specular spec lock --in .specular/spec.yaml --out .specular/spec.lock.json

plan:
	specular plan --in .specular/spec.yaml --lock .specular/spec.lock.json --out plan.json

build:
	specular build --plan plan.json --policy .specular/policy.yaml --dry-run

eval:
	specular eval --plan plan.json --lock .specular/spec.lock.json \
	  --spec .specular/spec.yaml --policy .specular/policy.yaml --report drift.sarif

clean:
	rm -rf .specular/spec.lock.json plan.json drift.sarif .specular/runs/
```

**Acceptance Criteria:**
- Each example includes complete workflow from PRD to drift report
- Makefiles work on macOS and Linux
- Documentation explains what each step does
- Examples complete in < 3 minutes each

---

#### 6.2.3 Architecture Decision Records (ADRs)
**Priority:** P2
**Complexity:** 3/10
**Estimated Effort:** 2 days

**Tasks:**
- [ ] Create `docs/adr/` directory for ADRs
- [ ] Document key architectural decisions with rationale
- [ ] Use standard ADR template format

**ADRs to Create:**
1. **ADR-001: Why Docker-Only Sandbox**
   - Context: Need secure execution environment
   - Decision: Docker-only, no local code execution
   - Consequences: Better security, but requires Docker installation

2. **ADR-002: Blake3 for SpecLock Hashing**
   - Context: Need cryptographic feature integrity
   - Decision: Blake3 over SHA-256
   - Consequences: Faster hashing, better collision resistance

3. **ADR-003: YAML Policy Over Executable Code**
   - Context: Need simple, auditable policy definition
   - Decision: Start with YAML, allow JS/TS in v2.0
   - Consequences: Limited expressiveness, but easier to audit

4. **ADR-004: Rule-Based Router First, ML Later**
   - Context: Need intelligent model routing
   - Decision: Implement rule-based in v0.8, learned router in v2.1+
   - Consequences: Good-enough routing now, room for improvement

5. **ADR-005: SARIF 2.1.0 for Drift Reports**
   - Context: Need CI/CD-friendly drift format
   - Decision: Adopt SARIF standard
   - Consequences: Native IDE/CI integration, but complex format

**ADR Template:**
```markdown
# ADR-XXX: Title

## Status
[Proposed | Accepted | Deprecated | Superseded]

## Context
[What is the issue we're seeing that motivates this decision?]

## Decision
[What is the change we're proposing and/or doing?]

## Consequences
[What becomes easier or more difficult to do because of this change?]

## Alternatives Considered
[What other options did we evaluate?]

## References
[Links to relevant discussions, RFCs, or documentation]
```

**Acceptance Criteria:**
- 5 key ADRs documented
- Each ADR includes context, decision, consequences, alternatives
- ADRs linked from main README.md

---

## 📋 Milestone 7: CI Integration (v1.0)

**Goal:** Make Specular CLI production-ready with seamless CI/CD integration and comprehensive deployment documentation.

### 7.1 GitHub Actions Integration

#### 7.1.1 Specular GitHub Action
**Priority:** P0
**Complexity:** 7/10
**Estimated Effort:** 4-5 days

**Tasks:**
- [ ] Create GitHub Action: `specular-action` repository
- [ ] Implement action with inputs for all workflow stages
- [ ] Add caching for Docker images and dependencies
- [ ] Support matrix builds (multiple presets, providers)
- [ ] Generate PR comments with drift reports
- [ ] Publish action to GitHub Marketplace

**Action Interface:**
```yaml
# .github/workflows/specular.yml
name: Specular Workflow

on: [push, pull_request]

jobs:
  specular:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Specular
        uses: specular/specular-action@v1
        with:
          version: '1.0.0'
          provider: anthropic
          provider-api-key: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Run Specular Workflow
        uses: specular/specular-action@v1
        with:
          workflow: full  # interview, spec, plan, build, eval
          preset: api-service
          policy: .specular/policy.yaml
          fail-on-drift: true
          upload-sarif: true  # Upload to GitHub Code Scanning

      - name: Comment PR with Results
        if: github.event_name == 'pull_request'
        uses: specular/specular-action/comment@v1
        with:
          sarif-report: drift.sarif
          show-coverage: true
```

**Action Features:**
- [x] Docker layer caching (restore/save)
- [x] Provider credential management (env vars)
- [x] Artifact upload (specs, plans, reports)
- [x] PR comment generation with drift summary
- [x] SARIF upload to GitHub Code Scanning
- [x] Job summaries with metrics (cost, tokens, time)

**Acceptance Criteria:**
- Action published to GitHub Marketplace
- Works with all 4 providers (Ollama, OpenAI, Anthropic, Gemini)
- Caching reduces run time by 50%+
- PR comments include actionable drift information
- Documentation includes 5+ example workflows

**Repository:** `github.com/specular/specular-action`

---

#### 7.1.2 Docker Image Caching Strategy
**Priority:** P0
**Complexity:** 6/10
**Estimated Effort:** 3-4 days

**Tasks:**
- [ ] Implement Docker layer caching in CI
- [ ] Create pre-built base images for common stacks
- [ ] Add image registry support (GHCR, DockerHub)
- [ ] Optimize Dockerfile for layer reuse
- [ ] Document caching best practices

**Implementation:**

```yaml
# .github/workflows/cache-images.yml
name: Cache Docker Images

on:
  schedule:
    - cron: '0 2 * * 0'  # Weekly rebuild
  workflow_dispatch:

jobs:
  build-cache:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        image:
          - golang:1.22
          - node:22
          - python:3.12
    steps:
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and Push
        uses: docker/build-push-action@v5
        with:
          push: true
          tags: ghcr.io/specular/cache:${{ matrix.image }}
          cache-from: type=registry,ref=ghcr.io/specular/cache:${{ matrix.image }}
          cache-to: type=inline
```

**Pre-built Images:**
```dockerfile
# Base image with common dependencies
FROM golang:1.22 AS go-base
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

FROM node:22 AS node-base
RUN npm install -g pnpm prettier eslint

FROM python:3.12 AS python-base
RUN pip install pytest black pylint
```

**Policy Configuration:**
```yaml
# .specular/policy.yaml - Use cached images
execution:
  docker:
    image_allowlist:
      - ghcr.io/specular/cache:golang-1.22
      - ghcr.io/specular/cache:node-22
      - ghcr.io/specular/cache:python-3.12
```

**Acceptance Criteria:**
- Cached images reduce pull time by 80%+
- Images rebuilt weekly or on-demand
- Documentation shows cache hit rate improvements
- Policy examples use cached images

---

#### 7.1.3 CI/CD Platform Examples
**Priority:** P1
**Complexity:** 4/10
**Estimated Effort:** 2-3 days

**Tasks:**
- [ ] Create example workflows for GitHub Actions
- [ ] Add GitLab CI configuration
- [ ] Include CircleCI config
- [ ] Document Jenkins pipeline integration
- [ ] Add Azure DevOps pipeline example

**Directory Structure:**
```
docs/ci-examples/
├─ github-actions/
│  ├─ basic-workflow.yml          # Simple spec → eval
│  ├─ matrix-providers.yml        # Test with multiple providers
│  ├─ monorepo.yml                # Multiple projects
│  └─ scheduled-drift.yml         # Nightly drift detection
├─ gitlab-ci/
│  └─ .gitlab-ci.yml
├─ circleci/
│  └─ config.yml
├─ jenkins/
│  └─ Jenkinsfile
└─ azure-pipelines/
   └─ azure-pipelines.yml
```

**Example: GitHub Actions Matrix Build**
```yaml
name: Multi-Provider Validation

on: [push]

jobs:
  validate:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        provider: [anthropic, openai, gemini]
        preset: [web-app, api-service, cli-tool]
    steps:
      - uses: actions/checkout@v4
      - uses: specular/specular-action@v1
        with:
          provider: ${{ matrix.provider }}
          preset: ${{ matrix.preset }}
          workflow: spec-only
```

**Acceptance Criteria:**
- Examples work copy-paste on each platform
- Documentation explains platform-specific features
- Matrix builds demonstrate multi-provider testing
- All examples include caching configuration

---

### 7.2 Production Documentation

#### 7.2.1 Deployment Guide
**Priority:** P0
**Complexity:** 4/10
**Estimated Effort:** 2-3 days

**Tasks:**
- [ ] Create `docs/deployment.md` with production best practices
- [ ] Document provider API key management
- [ ] Add secrets management guide (env vars, vaults)
- [ ] Include monitoring and alerting setup
- [ ] Document backup and disaster recovery

**Structure:**
```markdown
# Production Deployment Guide

## Prerequisites
- Docker Engine 24.0+
- AI Provider API keys
- CI/CD platform access

## Provider Setup
### Anthropic Claude
### OpenAI GPT
### Google Gemini
### Ollama (self-hosted)

## Secrets Management
- GitHub Secrets
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault

## Monitoring & Observability
- Metrics to track
- Alert thresholds
- Log aggregation
- Cost tracking

## Backup & Recovery
- SpecLock backups
- Policy versioning
- Disaster recovery plan

## Security Hardening
- Docker security best practices
- Network isolation
- Image scanning
- Dependency updates
```

**Acceptance Criteria:**
- Covers all 4 AI providers
- Includes secrets management for 3+ platforms
- Documents monitoring strategy
- Security checklist with 20+ items

---

#### 7.2.2 Best Practices Documentation
**Priority:** P1
**Complexity:** 3/10
**Estimated Effort:** 2 days

**Tasks:**
- [ ] Create `docs/best-practices.md`
- [ ] Document policy design patterns
- [ ] Add workflow optimization tips
- [ ] Include cost optimization strategies
- [ ] Create troubleshooting playbook

**Topics:**
1. **Policy Design**
   - Allowlist vs denylist strategies
   - Resource limit tuning
   - Test coverage thresholds
   - Security scanning configuration

2. **Workflow Optimization**
   - Interview preset selection
   - Plan complexity management
   - Docker image optimization
   - Parallel task execution

3. **Cost Management**
   - Provider cost comparison
   - Model hint optimization
   - Budget tracking
   - Cheaper model fallback

4. **Performance Tuning**
   - Docker caching strategies
   - Concurrent builds
   - Network optimization
   - Storage management

5. **Troubleshooting**
   - Common error codes
   - Debugging techniques
   - Log analysis
   - Support escalation

**Acceptance Criteria:**
- 5 major topic areas covered
- Each topic has 5+ practical tips
- Includes real-world examples
- Troubleshooting covers top 20 issues

---

## 📋 Milestone 8: Distribution & Release (v1.0.1)

**Timeline:** 1-2 weeks post v1.0 MVP
**Focus:** Multi-platform distribution and installation infrastructure
**Dependencies:** M7 CI Integration complete, v1.0 released

### 8.1 Release Automation

#### 8.1.1 GoReleaser Configuration
**Priority:** P0
**Complexity:** 5/10
**Estimated Effort:** 2-3 days

**Tasks:**
- [ ] Create `.goreleaser.yml` configuration
- [ ] Configure multi-platform builds (linux, darwin, windows)
- [ ] Add architecture support (amd64, arm64)
- [ ] Configure archives and checksums
- [ ] Add changelog generation
- [ ] Setup GPG signing for releases

**Implementation:**

```yaml
# .goreleaser.yml
version: 2

before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - id: specular
    binary: specular
    main: ./cmd/specular
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/felixgeelhaar/specular/internal/version.Version={{.Version}}
      - -X github.com/felixgeelhaar/specular/internal/version.Commit={{.Commit}}
      - -X github.com/felixgeelhaar/specular/internal/version.Date={{.Date}}

archives:
  - id: specular-archive
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - LICENSE
      - README.md
      - docs/**/*

checksum:
  name_template: 'checksums.txt'
  algorithm: sha256

changelog:
  use: github
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
  groups:
    - title: Features
      regexp: "^.*feat[(\\w)]*:+.*$"
      order: 0
    - title: 'Bug Fixes'
      regexp: "^.*fix[(\\w)]*:+.*$"
      order: 1
    - title: Others
      order: 999

release:
  github:
    owner: felixgeelhaar
    name: specular
  draft: false
  prerelease: auto
  mode: append
  header: |
    ## Specular {{ .Tag }} ({{ .Date }})

    Multi-platform AI-powered development workflow automation.
```

**GitHub Actions Integration:**

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write
  packages: write

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

      - uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ secrets.GPG_FINGERPRINT }}
```

**Acceptance Criteria:**
- Automated releases on tag push
- Builds for 6 platforms (linux/darwin/windows × amd64/arm64)
- Checksums and GPG signatures included
- Changelog auto-generated from commits
- Release assets uploaded to GitHub

---

#### 8.1.2 Version Management
**Priority:** P0
**Complexity:** 3/10
**Estimated Effort:** 1 day

**Tasks:**
- [ ] Create `internal/version` package
- [ ] Add version command to CLI
- [ ] Implement semantic versioning
- [ ] Add build metadata (commit, date)
- [ ] Document version tagging process

**Implementation:**

```go
// internal/version/version.go
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version (set by ldflags)
	Version = "dev"
	// Commit is the git commit hash (set by ldflags)
	Commit = "unknown"
	// Date is the build date (set by ldflags)
	Date = "unknown"
)

// Info contains version information
type Info struct {
	Version   string
	Commit    string
	Date      string
	GoVersion string
	Platform  string
}

// GetInfo returns version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns formatted version string
func (i Info) String() string {
	return fmt.Sprintf("Specular %s (%s) built %s with %s for %s",
		i.Version, i.Commit[:8], i.Date, i.GoVersion, i.Platform)
}
```

**CLI Integration:**

```go
// cmd/specular/version.go
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		info := version.GetInfo()

		verbose, _ := cmd.Flags().GetBool("verbose")
		if verbose {
			fmt.Println(info.String())
		} else {
			fmt.Printf("Specular %s\n", info.Version)
		}
	},
}

func init() {
	versionCmd.Flags().BoolP("verbose", "v", false, "Show detailed version info")
	rootCmd.AddCommand(versionCmd)
}
```

**Acceptance Criteria:**
- Version command shows version, commit, date
- Semantic versioning followed (MAJOR.MINOR.PATCH)
- Build metadata injected via ldflags
- Version displayed in help output

---

### 8.2 Package Manager Integration

#### 8.2.1 Homebrew Formula
**Priority:** P0
**Complexity:** 4/10
**Estimated Effort:** 2 days

**Tasks:**
- [ ] Create Homebrew tap repository
- [ ] Generate Homebrew formula
- [ ] Add cask for GUI (if needed)
- [ ] Setup formula auto-update
- [ ] Document installation process

**Repository Structure:**

```
specular-tap/
├─ Formula/
│  └─ specular.rb
├─ Casks/
│  └─ specular-gui.rb (if GUI exists)
├─ README.md
└─ .github/
   └─ workflows/
      └─ update-formula.yml
```

**GoReleaser Homebrew Config:**

```yaml
# .goreleaser.yml (addition)
brews:
  - name: specular
    repository:
      owner: felixgeelhaar
      name: homebrew-tap
      token: "{{ .Env.TAP_GITHUB_TOKEN }}"
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com
    commit_msg_template: "Brew formula update for {{ .ProjectName }} version {{ .Tag }}"
    homepage: "https://github.com/felixgeelhaar/specular"
    description: "AI-powered development workflow automation with policy enforcement"
    license: "MIT"
    skip_upload: false
    dependencies:
      - name: docker
        type: optional
    install: |
      bin.install "specular"

      # Install shell completions
      bash_completion.install "completions/specular.bash" => "specular"
      zsh_completion.install "completions/_specular" => "_specular"
      fish_completion.install "completions/specular.fish"

      # Install man pages
      man1.install "manpages/specular.1.gz"
    test: |
      system "#{bin}/specular", "version"
```

**Usage Documentation:**

```markdown
## Installation via Homebrew

### Add Tap
```bash
brew tap felixgeelhaar/tap
```

### Install Specular
```bash
brew install specular
```

### Upgrade
```bash
brew upgrade specular
```

### Verify Installation
```bash
specular version
```
```

**Acceptance Criteria:**
- Homebrew tap repository created and configured
- Formula auto-generated by GoReleaser
- Works on macOS (both Intel and Apple Silicon)
- Works on Linux (via Homebrew on Linux)
- Shell completions and man pages included

---

#### 8.2.2 Linux Package Managers
**Priority:** P1
**Complexity:** 6/10
**Estimated Effort:** 3-4 days

**Tasks:**
- [ ] Create Debian/Ubuntu packages (.deb)
- [ ] Create RPM packages (.rpm)
- [ ] Setup package signing
- [ ] Create APT repository
- [ ] Create YUM repository
- [ ] Document installation for each distro

**GoReleaser Package Config:**

```yaml
# .goreleaser.yml (addition)
nfpms:
  - id: packages
    package_name: specular
    vendor: Felix Geelhaar
    homepage: https://github.com/felixgeelhaar/specular
    maintainer: Felix Geelhaar <felix@geelhaar.com>
    description: |
      AI-powered development workflow automation
      Specular provides policy-enforced, AI-powered development workflows
      with multi-provider support and drift detection.
    license: MIT
    formats:
      - deb
      - rpm
      - apk

    dependencies:
      - docker-ce
    recommends:
      - git
    suggests:
      - ollama

    bindir: /usr/bin
    contents:
      - src: ./completions/specular.bash
        dst: /usr/share/bash-completion/completions/specular
        file_info:
          mode: 0644
      - src: ./completions/_specular
        dst: /usr/share/zsh/site-functions/_specular
        file_info:
          mode: 0644
      - src: ./completions/specular.fish
        dst: /usr/share/fish/vendor_completions.d/specular.fish
        file_info:
          mode: 0644
      - src: ./manpages/specular.1.gz
        dst: /usr/share/man/man1/specular.1.gz
        file_info:
          mode: 0644
      - src: ./LICENSE
        dst: /usr/share/doc/specular/copyright
        file_info:
          mode: 0644

    scripts:
      postinstall: "scripts/postinstall.sh"
      preremove: "scripts/preremove.sh"

    deb:
      signature:
        key_file: "{{ .Env.GPG_KEY_FILE }}"

    rpm:
      signature:
        key_file: "{{ .Env.GPG_KEY_FILE }}"
```

**Installation Scripts:**

```bash
# scripts/postinstall.sh
#!/bin/bash
echo "Specular installed successfully!"
echo "Run 'specular init' to get started."
echo ""
echo "For shell completion, restart your shell or run:"
echo "  source /usr/share/bash-completion/completions/specular  # bash"
echo "  autoload -Uz compinit && compinit                       # zsh"

# scripts/preremove.sh
#!/bin/bash
echo "Removing Specular..."
```

**Documentation:**

```markdown
## Linux Installation

### Debian/Ubuntu (APT)
```bash
# Add GPG key
curl -fsSL https://github.com/felixgeelhaar/specular/releases/download/KEY.gpg | sudo gpg --dearmor -o /usr/share/keyrings/specular.gpg

# Add repository
echo "deb [signed-by=/usr/share/keyrings/specular.gpg] https://github.com/felixgeelhaar/specular/releases focal main" | sudo tee /etc/apt/sources.list.d/specular.list

# Install
sudo apt update
sudo apt install specular
```

### RHEL/CentOS/Fedora (YUM/DNF)
```bash
# Add repository
sudo tee /etc/yum.repos.d/specular.repo <<EOF
[specular]
name=Specular Repository
baseurl=https://github.com/felixgeelhaar/specular/releases
enabled=1
gpgcheck=1
gpgkey=https://github.com/felixgeelhaar/specular/releases/download/KEY.gpg
EOF

# Install
sudo dnf install specular  # Fedora
sudo yum install specular  # RHEL/CentOS
```

### Alpine (APK)
```bash
# Add repository
echo "https://github.com/felixgeelhaar/specular/releases" | sudo tee -a /etc/apk/repositories

# Install
sudo apk add specular
```

### Direct Download (.deb/.rpm)
```bash
# Download latest release
wget https://github.com/felixgeelhaar/specular/releases/download/v1.0.1/specular_1.0.1_linux_amd64.deb

# Install (Debian/Ubuntu)
sudo dpkg -i specular_1.0.1_linux_amd64.deb
sudo apt-get install -f  # Fix dependencies

# Install (RHEL/CentOS/Fedora)
wget https://github.com/felixgeelhaar/specular/releases/download/v1.0.1/specular_1.0.1_linux_amd64.rpm
sudo rpm -i specular_1.0.1_linux_amd64.rpm
```
```

**Acceptance Criteria:**
- .deb packages for Debian/Ubuntu
- .rpm packages for RHEL/CentOS/Fedora
- .apk packages for Alpine
- GPG-signed packages
- Post-install scripts configure shell completions
- Documentation covers all major distros

---

#### 8.2.3 Windows Package Managers
**Priority:** P1
**Complexity:** 5/10
**Estimated Effort:** 2-3 days

**Tasks:**
- [ ] Create Chocolatey package
- [ ] Create Scoop manifest
- [ ] Setup winget manifest
- [ ] Test on Windows 10/11
- [ ] Document installation process

**Chocolatey Package:**

```xml
<!-- chocolatey/specular.nuspec -->
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
  <metadata>
    <id>specular</id>
    <version>1.0.1</version>
    <title>Specular</title>
    <authors>Felix Geelhaar</authors>
    <owners>Felix Geelhaar</owners>
    <licenseUrl>https://github.com/felixgeelhaar/specular/blob/main/LICENSE</licenseUrl>
    <projectUrl>https://github.com/felixgeelhaar/specular</projectUrl>
    <iconUrl>https://raw.githubusercontent.com/felixgeelhaar/specular/main/docs/logo.png</iconUrl>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <description>
      AI-powered development workflow automation with policy enforcement.
      Supports multiple AI providers (Anthropic, OpenAI, Google, Ollama),
      drift detection, and sandbox execution.
    </description>
    <summary>AI-powered development workflow automation</summary>
    <releaseNotes>https://github.com/felixgeelhaar/specular/releases/tag/v1.0.1</releaseNotes>
    <copyright>2024 Felix Geelhaar</copyright>
    <tags>ai development workflow automation docker cli</tags>
    <dependencies>
      <dependency id="docker-desktop" version="4.0.0" />
    </dependencies>
  </metadata>
  <files>
    <file src="tools\**" target="tools" />
  </files>
</package>
```

```powershell
# chocolatey/tools/chocolateyinstall.ps1
$ErrorActionPreference = 'Stop'

$packageName = 'specular'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$url64 = 'https://github.com/felixgeelhaar/specular/releases/download/v1.0.1/specular_1.0.1_windows_amd64.zip'
$checksum64 = 'CHECKSUM_PLACEHOLDER'

$packageArgs = @{
  packageName   = $packageName
  unzipLocation = $toolsDir
  url64bit      = $url64
  checksum64    = $checksum64
  checksumType64= 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

# Add to PATH
Install-ChocolateyPath -PathToInstall $toolsDir -PathType 'User'
```

**Scoop Manifest:**

```json
{
  "version": "1.0.1",
  "description": "AI-powered development workflow automation",
  "homepage": "https://github.com/felixgeelhaar/specular",
  "license": "MIT",
  "architecture": {
    "64bit": {
      "url": "https://github.com/felixgeelhaar/specular/releases/download/v1.0.1/specular_1.0.1_windows_amd64.zip",
      "hash": "CHECKSUM_PLACEHOLDER",
      "bin": "specular.exe"
    }
  },
  "depends": "docker",
  "checkver": "github",
  "autoupdate": {
    "architecture": {
      "64bit": {
        "url": "https://github.com/felixgeelhaar/specular/releases/download/v$version/specular_$version_windows_amd64.zip"
      }
    }
  }
}
```

**Winget Manifest:**

```yaml
# manifests/f/FelixGeelhaar/Specular/1.0.1/FelixGeelhaar.Specular.yaml
PackageIdentifier: FelixGeelhaar.Specular
PackageVersion: 1.0.1
PackageLocale: en-US
Publisher: Felix Geelhaar
PublisherUrl: https://github.com/felixgeelhaar
PublisherSupportUrl: https://github.com/felixgeelhaar/specular/issues
Author: Felix Geelhaar
PackageName: Specular
PackageUrl: https://github.com/felixgeelhaar/specular
License: MIT
LicenseUrl: https://github.com/felixgeelhaar/specular/blob/main/LICENSE
ShortDescription: AI-powered development workflow automation
Description: |
  Specular provides policy-enforced, AI-powered development workflows
  with multi-provider support and drift detection.
Tags:
  - ai
  - development
  - workflow
  - automation
  - docker
ManifestType: defaultLocale
ManifestVersion: 1.0.0
---
PackageIdentifier: FelixGeelhaar.Specular
PackageVersion: 1.0.1
Installers:
  - Architecture: x64
    InstallerType: zip
    InstallerUrl: https://github.com/felixgeelhaar/specular/releases/download/v1.0.1/specular_1.0.1_windows_amd64.zip
    InstallerSha256: CHECKSUM_PLACEHOLDER
ManifestType: installer
ManifestVersion: 1.0.0
```

**Documentation:**

```markdown
## Windows Installation

### Chocolatey
```powershell
choco install specular
```

### Scoop
```powershell
scoop bucket add felixgeelhaar https://github.com/felixgeelhaar/scoop-bucket
scoop install specular
```

### Winget
```powershell
winget install FelixGeelhaar.Specular
```

### Direct Download
```powershell
# Download from GitHub Releases
Invoke-WebRequest -Uri "https://github.com/felixgeelhaar/specular/releases/download/v1.0.1/specular_1.0.1_windows_amd64.zip" -OutFile "specular.zip"

# Extract
Expand-Archive -Path specular.zip -DestinationPath C:\Program Files\Specular

# Add to PATH
$env:Path += ";C:\Program Files\Specular"
```
```

**Acceptance Criteria:**
- Chocolatey package published
- Scoop manifest in custom bucket
- Winget manifest submitted to microsoft/winget-pkgs
- Works on Windows 10/11 (x64)
- Docker Desktop dependency documented

---

### 8.3 Documentation

#### 8.3.1 Installation Guide
**Priority:** P0
**Complexity:** 3/10
**Estimated Effort:** 1-2 days

**Tasks:**
- [ ] Create comprehensive `docs/installation.md`
- [ ] Add platform-specific instructions
- [ ] Document prerequisites
- [ ] Add troubleshooting section
- [ ] Include verification steps

**Structure:**

```markdown
# Installation Guide

## Prerequisites

### Required
- **Docker**: Docker Engine 24.0+ or Docker Desktop
- **AI Provider Account**: At least one of:
  - Anthropic API key (Claude)
  - OpenAI API key (GPT)
  - Google AI Studio key (Gemini)
  - Ollama (self-hosted, no API key needed)

### Recommended
- **Git**: For version control integration
- **Make**: For build automation

## Installation by Platform

### macOS

#### Homebrew (Recommended)
```bash
brew tap felixgeelhaar/tap
brew install specular
```

#### Direct Download
```bash
# Download latest release
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_darwin_amd64.tar.gz

# Extract
tar -xzf specular_darwin_amd64.tar.gz

# Move to PATH
sudo mv specular /usr/local/bin/

# Verify
specular version
```

### Linux

[Debian/Ubuntu, RHEL/Fedora, Alpine, Direct Download sections as shown above]

### Windows

[Chocolatey, Scoop, Winget, Direct Download sections as shown above]

## Provider Setup

### Anthropic Claude
```bash
export ANTHROPIC_API_KEY="your-key-here"
```

### OpenAI GPT
```bash
export OPENAI_API_KEY="your-key-here"
```

### Google Gemini
```bash
export GOOGLE_API_KEY="your-key-here"
```

### Ollama (Self-Hosted)
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull models
ollama pull codellama
ollama pull llama2
```

## Verification

```bash
# Check version
specular version

# Verify Docker
specular doctor

# Initialize project
specular init --preset api-service
```

## Troubleshooting

### Docker Not Running
**Error:** `Cannot connect to Docker daemon`
**Solution:** Start Docker Desktop or Docker Engine

### API Key Not Set
**Error:** `Provider authentication failed`
**Solution:** Set the appropriate environment variable

### Permission Denied
**Error:** `Permission denied: /usr/local/bin/specular`
**Solution:** Use sudo or install to user directory

## Shell Completion

### Bash
```bash
specular completion bash > /usr/local/etc/bash_completion.d/specular
```

### Zsh
```bash
specular completion zsh > "${fpath[1]}/_specular"
```

### Fish
```bash
specular completion fish > ~/.config/fish/completions/specular.fish
```

## Upgrading

### Homebrew
```bash
brew upgrade specular
```

### APT
```bash
sudo apt update && sudo apt upgrade specular
```

### Chocolatey
```powershell
choco upgrade specular
```

## Uninstallation

[Platform-specific uninstall instructions]
```

**Acceptance Criteria:**
- Covers all 3 platforms (macOS, Linux, Windows)
- Includes all package manager options
- Prerequisites clearly documented
- Troubleshooting covers common issues
- Verification steps included

---

## 📊 Testing & Quality

### Test Coverage Goals

**Current:** 73.7% overall
**Target:** 80%+ overall

#### Priority Coverage Improvements:
1. **router/ (80.4% → 85%)**
   - Add edge case tests for budget exhaustion
   - Test all truncation strategies
   - Add concurrent routing tests

2. **provider/ (81.4% → 85%)**
   - Add error recovery tests
   - Test provider timeout scenarios
   - Add concurrent stream tests

3. **exec/ (87.1% → 90%)**
   - Add Docker daemon failure tests
   - Test resource limit enforcement
   - Add cleanup failure recovery

4. **eval/ (85.0% → 90%)**
   - Add multi-language linter tests
   - Test all security scanners
   - Add concurrent evaluation tests

**Implementation Tasks:**
- [ ] Add table-driven tests for all edge cases
- [ ] Implement integration tests for E2E workflow
- [ ] Add race detection stress tests
- [ ] Create mutation testing suite
- [ ] Document test coverage by component

---

## 🎯 Success Metrics

### MVP Completion Criteria (v1.0)

**Functionality:**
- [x] All core packages implemented ✅
- [ ] E2E workflow polished 🚧
- [ ] Error handling production-ready 🚧
- [ ] CI/CD integration complete 📅
- [ ] Documentation comprehensive 📅

**Quality:**
- [x] 73.7% overall test coverage ✅
- [ ] 80%+ overall coverage target 📅
- [ ] Zero critical bugs 📅
- [ ] < 5 P1 bugs 📅

**Performance:**
- [ ] Interview → eval in < 5 minutes 🚧
- [ ] Provider response time < 30s (p95) 📅
- [ ] Docker image pull < 10s (cached) 📅
- [ ] Drift detection < 2s 📅

**Documentation:**
- [ ] Getting started guide ✅
- [ ] API documentation 🚧
- [ ] CI/CD examples 📅
- [ ] Best practices guide 📅
- [ ] 5+ ADRs documented 📅

**User Experience:**
- [ ] Friendly error messages 🚧
- [ ] Progress indicators 📅
- [ ] Color-coded output 📅
- [ ] Verbose mode with metrics 🚧

---

## 📅 Timeline

### Phase 1: E2E Refinement (2 weeks)
**Week 1:**
- E2E workflow validation
- Error recovery implementation
- Progress indicators

**Week 2:**
- Getting started guide
- Workflow examples
- ADR documentation

### Phase 2: CI Integration (2 weeks)
**Week 3:**
- GitHub Action development
- Docker caching implementation
- CI platform examples

**Week 4:**
- Production deployment guide
- Best practices documentation
- Final testing and polish

**Total Duration:** 4 weeks to MVP (v1.0)

---

## 🚀 Post-MVP Roadmap

### v1.1-v1.3: Pro Alpha (Governance Core)
- Versioned specs with change tracking
- Approval workflows (policy-based gating)
- Cloud sync (S3, GCS, Azure Blob)
- Team collaboration features

### v1.4-v1.6: Pro Beta (Team Awareness)
- Spec Inbox with notifications
- Daily Digest emails
- Analytics dashboard
- Policy pack marketplace

### v1.7-v2.0: Enterprise (Compliance)
- Private SpecHub deployment
- SSO integration (SAML, OIDC)
- Audit log exports (SIEM)
- Compliance reports (SOC2, GDPR)

### v2.1+: Intelligence (Future)
- Learned router (ML-based)
- Semantic spec graph
- Predictive drift detection
- Auto-remediation suggestions

---

## 📝 Notes

**Development Principles:**
- Test-first development (TDD)
- Atomic commits with conventional messages
- Table-driven tests with race detection
- Production-grade code from day one
- Comprehensive documentation

**Review Checkpoints:**
- Weekly progress review against this plan
- Test coverage monitoring (daily)
- Performance benchmarking (per PR)
- Documentation reviews (per milestone)

**Blockers & Dependencies:**
- GitHub Actions marketplace approval (3-5 day SLA)
- Provider API rate limits during testing
- Docker Hub rate limits (use GHCR instead)

---

*This action plan is a living document. Update as priorities shift or new requirements emerge.*
