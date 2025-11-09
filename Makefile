.PHONY: build test test-e2e clean install lint fmt help

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
		go test -v -timeout 30m ./test/e2e/...; \
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

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the specular binary"
	@echo "  install        - Install specular to GOPATH/bin"
	@echo "  test           - Run all tests with race detection"
	@echo "  test-coverage  - Run tests and generate HTML coverage report"
	@echo "  test-e2e       - Run end-to-end tests"
	@echo "  clean          - Remove build artifacts"
	@echo "  lint           - Run golangci-lint"
	@echo "  fmt            - Format all Go code"
	@echo "  tidy           - Tidy Go module dependencies"
	@echo "  check          - Run fmt, lint, and test"
	@echo "  dev            - Build and run specular"
	@echo "  help           - Show this help message"
