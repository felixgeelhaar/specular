.PHONY: build test test-e2e clean install lint fmt help \
       bench bench-startup bench-binary perf-report build-optimized \
       release-plan release-bump release-bump-version release-notes release-evaluate \
       release-validate release-approve release-publish release-publish-dry-run release \
       release-status release-cancel release-reset release-blast-radius \
       release-snapshot release-local verify-release

# Build the binary
build:
	go build -o specular ./cmd/specular

# Install the binary to GOPATH/bin
install:
	go install ./cmd/specular

# Run tests
test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Run end-to-end tests
test-e2e: build
	@echo "Running E2E tests..."
	@if [ -d "test/e2e" ]; then \
		go test -v -tags=e2e -timeout 30m ./test/e2e/...; \
	else \
		echo "E2E tests not yet implemented (test/e2e directory not found)"; \
		echo "See docs/E2E_TEST_PLAN.md for implementation plan"; \
		exit 0; \
	fi

# Clean build artifacts
clean:
	rm -f specular coverage.txt coverage.html
	go clean

# Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Tidy dependencies
tidy:
	go mod tidy

# Run all checks (fmt, lint, test)
check: fmt lint test

# Development build and run
dev: build
	./specular

# =============================================================================
# Performance & Benchmarking
# =============================================================================

# Run all benchmarks
bench:
	go test -bench=. -benchmem ./internal/benchmark/...

# Measure startup time (10 iterations)
bench-startup: build-optimized
	@echo "Measuring startup time..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		/usr/bin/time -p ./specular-opt version 2>&1 | grep real; \
	done
	@rm -f specular-opt

# Analyze binary size
bench-binary:
	@echo "Building variants..."
	@go build -o specular-normal ./cmd/specular
	@go build -ldflags="-s -w" -o specular-stripped ./cmd/specular
	@go build -ldflags="-s -w" -trimpath -o specular-trimpath ./cmd/specular
	@echo ""
	@echo "Binary Size Comparison:"
	@echo "======================="
	@ls -lh specular-normal specular-stripped specular-trimpath | awk '{print $$9 ": " $$5}'
	@rm -f specular-normal specular-stripped specular-trimpath

# Build optimized binary
build-optimized:
	go build -ldflags="-s -w" -trimpath -o specular-opt ./cmd/specular

# Generate performance report
perf-report: build-optimized
	@echo "Performance Report"
	@echo "=================="
	@echo ""
	@echo "Binary Info:"
	@ls -lh specular-opt | awk '{print "  Size: " $$5}'
	@echo "  Platform: $$(go env GOOS)/$$(go env GOARCH)"
	@echo "  Go Version: $$(go version | awk '{print $$3}')"
	@echo ""
	@echo "Dependencies: $$(go list -m all 2>/dev/null | wc -l | tr -d ' ') modules"
	@echo "Transitive:   $$(go list -f '{{len .Deps}}' ./cmd/specular) packages"
	@echo ""
	@echo "Startup Times (5 runs):"
	@for cmd in "version" "--help" "doctor --help"; do \
		sum=0; \
		for i in 1 2 3 4 5; do \
			t=$$(/usr/bin/time -p ./specular-opt $$cmd 2>&1 | grep real | awk '{print $$2}'); \
			sum=$$(echo "$$sum + $$t" | bc); \
		done; \
		avg=$$(echo "scale=3; $$sum / 5" | bc); \
		echo "  specular $$cmd: $${avg}s avg"; \
	done
	@rm -f specular-opt

# =============================================================================
# Release Targets (powered by relicta)
# =============================================================================

# Version for release (override with: make verify-release VERSION=v1.6.0)
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

# Analyze commits and plan release (relicta plan)
release-plan:
	@echo "Analyzing commits since last release..."
	relicta plan --analyze

# Bump version based on conventional commits
release-bump:
	@echo "Calculating next version..."
	relicta bump --level auto

# Bump to specific version
release-bump-version:
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "v0.0.0" ]; then \
		echo "Error: VERSION required. Usage: make release-bump-version VERSION=1.6.0"; \
		exit 1; \
	fi
	relicta bump --version $(VERSION)

# Generate changelog and release notes
release-notes:
	@echo "Generating release notes..."
	relicta notes

# Evaluate release risk (CGP - Change Governance Protocol)
release-evaluate:
	@echo "Evaluating release risk..."
	relicta evaluate

# Validate release before publishing
release-validate:
	@echo "Validating release..."
	relicta validate

# Approve release for publishing
release-approve:
	@echo "Approving release..."
	relicta approve

# Publish release (creates tag, runs plugins)
release-publish:
	@echo "Publishing release..."
	relicta publish

# Publish release dry-run (no actual changes)
release-publish-dry-run:
	@echo "Publishing release (dry-run)..."
	relicta publish --dry-run

# Full release workflow: plan -> bump -> notes -> evaluate -> approve -> publish
release: check release-plan
	@echo "Starting release workflow..."
	@echo "Step 1: Bump version"
	relicta bump --level auto
	@echo "Step 2: Generate notes"
	relicta notes
	@echo "Step 3: Evaluate risk"
	relicta evaluate
	@echo "Step 4: Approve release"
	relicta approve
	@echo "Step 5: Publish release"
	relicta publish
	@echo "Release complete!"

# Check current release status
release-status:
	@relicta status

# Cancel in-progress release
release-cancel:
	@echo "Canceling release..."
	relicta cancel --reason "Canceled via make release-cancel"

# Reset failed release state
release-reset:
	@echo "Resetting release state..."
	relicta reset --force

# Analyze blast radius in monorepo
release-blast-radius:
	@echo "Analyzing blast radius..."
	relicta blast-radius --transitive

# Build release snapshot with goreleaser (no tag, no publish)
release-snapshot:
	@echo "Building release snapshot..."
	goreleaser release --snapshot --clean --skip=publish

# Local release with goreleaser (skip Docker and signing)
release-local:
	@echo "Running local release (skip docker, sign, publish)..."
	goreleaser release --snapshot --clean --skip=publish,docker,sign

# Verify a published release
verify-release:
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "v0.0.0" ]; then \
		echo "Error: VERSION required. Usage: make verify-release VERSION=v1.6.0"; \
		exit 1; \
	fi
	@echo "Verifying release $(VERSION)..."
	./scripts/verify-release.sh $(VERSION)

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Test:"
	@echo "  build              - Build the specular binary"
	@echo "  install            - Install specular to GOPATH/bin"
	@echo "  test               - Run all tests with race detection"
	@echo "  test-coverage      - Run tests and generate HTML coverage report"
	@echo "  test-e2e           - Run end-to-end tests"
	@echo "  clean              - Remove build artifacts"
	@echo "  lint               - Run golangci-lint"
	@echo "  fmt                - Format all Go code"
	@echo "  tidy               - Tidy Go module dependencies"
	@echo "  check              - Run fmt, lint, and test"
	@echo "  dev                - Build and run specular"
	@echo ""
	@echo "Performance & Benchmarking:"
	@echo "  bench              - Run Go benchmarks"
	@echo "  bench-startup      - Measure startup time"
	@echo "  bench-binary       - Compare binary sizes"
	@echo "  build-optimized    - Build optimized binary"
	@echo "  perf-report        - Generate performance report"
	@echo ""
	@echo "Release (powered by relicta):"
	@echo "  release-plan       - Analyze commits and suggest version bump"
	@echo "  release-bump       - Auto-bump version based on commits"
	@echo "  release-bump-version VERSION=x.y.z - Bump to specific version"
	@echo "  release-notes      - Generate changelog and release notes"
	@echo "  release-evaluate   - Evaluate release risk (CGP)"
	@echo "  release-validate   - Validate release before publishing"
	@echo "  release-approve    - Approve release for publishing"
	@echo "  release-publish    - Publish release (creates tag, runs plugins)"
	@echo "  release-publish-dry-run - Dry-run publish (no changes)"
	@echo "  release            - Full release workflow (plan->publish)"
	@echo "  release-status     - Check current release status"
	@echo "  release-cancel     - Cancel in-progress release"
	@echo "  release-reset      - Reset failed release state"
	@echo "  release-blast-radius - Analyze monorepo impact"
	@echo "  release-snapshot   - Build snapshot with goreleaser"
	@echo "  release-local      - Local release (skip docker/sign)"
	@echo "  verify-release VERSION=vx.y.z - Verify published release"
	@echo ""
	@echo "  help               - Show this help message"
