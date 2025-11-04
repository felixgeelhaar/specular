.PHONY: build test clean install lint fmt help

# Build the binary
build:
	go build -o ai-dev ./cmd/ai-dev

# Install the binary to GOPATH/bin
install:
	go install ./cmd/ai-dev

# Run tests
test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Clean build artifacts
clean:
	rm -f ai-dev coverage.txt coverage.html
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
	./ai-dev

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the ai-dev binary"
	@echo "  install        - Install ai-dev to GOPATH/bin"
	@echo "  test           - Run all tests with race detection"
	@echo "  test-coverage  - Run tests and generate HTML coverage report"
	@echo "  clean          - Remove build artifacts"
	@echo "  lint           - Run golangci-lint"
	@echo "  fmt            - Format all Go code"
	@echo "  tidy           - Tidy Go module dependencies"
	@echo "  check          - Run fmt, lint, and test"
	@echo "  dev            - Build and run ai-dev"
	@echo "  help           - Show this help message"
