# Contributing to Specular

Thank you for your interest in contributing to Specular! This document provides guidelines and instructions for contributing.

## Code of Conduct

This project adheres to the Contributor Covenant [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the [issue tracker](https://github.com/felixgeelhaar/specular/issues) to avoid duplicates. When creating a bug report, include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples** (code samples, command outputs, etc.)
- **Describe the behavior you observed** and what you expected
- **Include details about your environment**:
  - Specular version (`specular version`)
  - Go version (`go version`)
  - Operating system and version
  - Docker version (if relevant)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear and descriptive title**
- **Provide a detailed description** of the proposed functionality
- **Explain why this enhancement would be useful**
- **Include code examples** or mockups if applicable

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following our coding standards
3. **Add tests** for new functionality
4. **Ensure all tests pass** (`make test`)
5. **Update documentation** as needed
6. **Follow commit message conventions** (see below)
7. **Submit a pull request**

## Development Setup

### Prerequisites

- Go 1.22 or later
- Docker (for policy-enforced builds)
- Make

### Getting Started

```bash
# Clone your fork
git clone https://github.com/YOUR-USERNAME/specular.git
cd specular

# Build the project
make build

# Run tests
make test

# Run E2E tests
make test-e2e
```

### Project Structure

```
specular/
├── cmd/                    # CLI commands (Cobra)
├── internal/
│   ├── spec/              # Specification handling
│   ├── plan/              # Plan generation
│   ├── policy/            # Policy enforcement
│   ├── drift/             # Drift detection
│   ├── exec/              # Docker execution
│   └── provider/          # LLM provider abstraction
├── test/                  # Integration and E2E tests
├── docs/                  # Documentation
└── examples/              # Example projects
```

## Coding Standards

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `go vet` and address warnings
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and concise

### Testing

- Write unit tests for new functionality
- Aim for >80% test coverage on new code
- Use table-driven tests where appropriate
- Include integration tests for complex workflows
- Test error cases and edge conditions

Example test structure:

```go
func TestFeatureName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionUnderTest(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**

```
feat(plan): add support for parallel task execution

Implements parallel execution of independent tasks in the plan.
Tasks with no dependencies can now run concurrently.

Closes #123
```

```
fix(policy): correct Docker image validation regex

The previous regex didn't handle tags with multiple dots.
Updated to support semver tags like 1.2.3.

Fixes #456
```

## Documentation

- Update relevant documentation in `docs/`
- Add comments to exported functions and types
- Update README.md if adding new features
- Include examples for new functionality
- Update CHANGELOG.md (if exists)

## Testing Guidelines

### Running Tests

```bash
# Unit tests
go test ./...

# With coverage
make test-coverage

# E2E tests
make test-e2e

# Specific package
go test ./internal/spec/...
```

### Writing Tests

- Test files should be named `*_test.go`
- Place tests in the same package as the code
- Use `testdata/` directories for test fixtures
- Mock external dependencies
- Test both success and failure cases

## Review Process

1. **Automated checks** must pass (CI/CD pipeline)
2. **Code review** by maintainers
3. **Testing** - all tests must pass
4. **Documentation** must be updated
5. **Approval** from at least one maintainer

## Getting Help

- **Documentation**: Check [docs/](docs/)
- **Issues**: Search [existing issues](https://github.com/felixgeelhaar/specular/issues)
- **Discussions**: Start a [discussion](https://github.com/felixgeelhaar/specular/discussions)

## Recognition

Contributors will be acknowledged in:
- GitHub contributors list
- Release notes for significant contributions
- CHANGELOG.md

Thank you for contributing to Specular! 🎉
