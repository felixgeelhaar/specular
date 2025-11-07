# Specular Examples

This directory contains complete examples demonstrating Specular workflows for different types of projects.

## Available Examples

### 1. [Web Application](./web-app/) - Full-Stack React + Node.js
**Use Case:** Modern task management web application
**Technologies:** React 18, Node.js 20, PostgreSQL 15, Redis 7, WebSocket
**Demonstrates:**
- Full-stack application development
- Real-time collaboration features
- Authentication and authorization
- Frontend UI/UX patterns
- Database migrations and schema design
- CI/CD integration with GitHub Actions

**Best For:** Teams building web applications with frontend and backend components

---

### 2. [API Service](./api-service/) - High-Performance Go REST API
**Use Case:** Weather data API service
**Technologies:** Go 1.22, PostgreSQL 15, Redis 7, Prometheus
**Demonstrates:**
- High-throughput API design (10,000+ RPS)
- Low-latency optimization (<50ms p95)
- Caching strategies
- Rate limiting and API key management
- Metrics and monitoring
- Load testing integration

**Best For:** Teams building high-performance backend services and APIs

---

## Getting Started

### Prerequisites

1. **Install Specular CLI**
   ```bash
   # Via Homebrew (recommended)
   brew install felixgeelhaar/tap/specular

   # Or download binary from releases
   # https://github.com/felixgeelhaar/specular/releases
   ```

2. **Install Docker**
   ```bash
   # macOS
   brew install --cask docker

   # Or download from https://www.docker.com/products/docker-desktop
   ```

3. **Configure API Keys** (optional for plan generation)
   ```bash
   export ANTHROPIC_API_KEY="your-key"
   export OPENAI_API_KEY="your-key"
   export GEMINI_API_KEY="your-key"
   ```

### Quick Start with Any Example

```bash
# 1. Navigate to an example
cd examples/web-app  # or api-service

# 2. Generate implementation plan
specular plan --spec .specular/spec.yaml --output plan.json

# 3. Execute the plan
specular build --plan plan.json --policy .specular/policy.yaml

# 4. Detect drift (after making changes)
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --lock .specular/spec.lock.json \
  --api-spec openapi.yaml
```

## Example Structure

Each example follows the same directory structure:

```
example-name/
├── .specular/
│   ├── spec.yaml              # Product specification
│   ├── policy.yaml            # Security and quality policies
│   └── spec.lock.json         # Generated feature checksums (after plan)
├── openapi.yaml               # API specification (if applicable)
├── plan.json                  # Generated implementation plan (after plan)
├── README.md                  # Example-specific documentation
└── src/                       # Generated code (after build)
```

## Common Commands

### Plan Generation
```bash
# Generate plan from specification
specular plan --spec .specular/spec.yaml --output plan.json

# View plan summary
cat plan.json | jq '.tasks[] | {id, title, priority, complexity}'

# Count tasks by priority
cat plan.json | jq '.tasks | group_by(.priority) | map({priority: .[0].priority, count: length})'
```

### Build Execution
```bash
# Execute with policy enforcement
specular build --plan plan.json --policy .specular/policy.yaml

# Dry-run mode (show what would be executed)
specular build --plan plan.json --policy .specular/policy.yaml --dry-run

# Verbose output
specular build --plan plan.json --policy .specular/policy.yaml --verbose

# Resume from checkpoint
specular build --plan plan.json --policy .specular/policy.yaml --resume
```

### Drift Detection
```bash
# Check for plan drift
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --lock .specular/spec.lock.json

# Check for code drift (API conformance)
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --lock .specular/spec.lock.json \
  --api-spec openapi.yaml

# Fail on any drift
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --lock .specular/spec.lock.json \
  --fail-on-drift

# Generate SARIF report
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --lock .specular/spec.lock.json \
  --report drift.sarif
```

### Docker Image Management
```bash
# Pre-warm images for faster subsequent runs
specular prewarm --plan plan.json --verbose

# Pre-warm specific images
specular prewarm golang:1.22 node:20 postgres:15

# Export images for CI/CD cache
specular prewarm --plan plan.json --export .specular/cache

# Import cached images (in CI/CD)
specular prewarm --import .specular/cache

# View cache statistics
specular prewarm --plan plan.json --verbose  # Shows cache stats at end
```

## Understanding the Workflow

### 1. Specification First
Start by defining your product in `.specular/spec.yaml`:
- **Product goals** - What you're building and why
- **Features** - Detailed feature descriptions with success criteria
- **API contracts** - RESTful endpoints with request/response schemas
- **Acceptance criteria** - How to verify the product works
- **Constraints** - Technical limitations and requirements

### 2. Policy Enforcement
Define quality and security policies in `.specular/policy.yaml`:
- **Execution policies** - Docker-only, resource limits
- **Test requirements** - Coverage thresholds, frameworks
- **Security controls** - Secrets scanning, dependency checks
- **Performance targets** - Latency, throughput, bundle size
- **Compliance** - GDPR, audit logging, encryption

### 3. Plan Generation
Specular generates a detailed implementation plan:
- Tasks are broken down by feature
- Each task has complexity, priority, and dependencies
- Tasks are ordered for optimal implementation
- Plan includes test requirements and acceptance criteria

### 4. Sandboxed Execution
All code execution happens in Docker:
- **Security** - Isolated from host system
- **Reproducibility** - Same environment every time
- **Policy enforcement** - Resource limits, network restrictions
- **Artifact collection** - Build outputs, test results, metrics

### 5. Drift Detection
Continuous validation ensures spec compliance:
- **Plan drift** - Detects when spec changes without updating lock
- **Code drift** - Validates API implementation matches OpenAPI spec
- **Infrastructure drift** - Checks for policy violations

### 6. CI/CD Integration
Examples include GitHub Actions workflows:
- Automatic plan generation on spec changes
- Build execution with policy enforcement
- Drift detection on pull requests
- SARIF upload for security findings
- Docker image caching for 94% speedup

## Customizing Examples

### Modify Specifications

Edit `.specular/spec.yaml` to change features:

```yaml
features:
  - id: feat-007
    title: Your New Feature
    desc: Detailed description
    priority: P0
    api:
      - method: GET
        path: /api/your-endpoint
        request: ""
        response: YourResponse
    success:
      - Success criteria 1
      - Success criteria 2
    trace:
      - PRD-007 Your Requirements
```

Then regenerate the plan:
```bash
specular plan --spec .specular/spec.yaml --output plan.json
```

### Adjust Policies

Modify `.specular/policy.yaml` to change enforcement:

```yaml
tests:
  require_pass: true
  min_coverage: 0.90  # Increase coverage requirement

security:
  severity_threshold: "high"  # Only fail on high+ severity

performance:
  bundle_size:
    max_js: "300kb"  # Stricter bundle limit
```

### Update API Specifications

Edit `openapi.yaml` to match your API changes, then validate:

```bash
specular eval \
  --spec .specular/spec.yaml \
  --api-spec openapi.yaml \
  --fail-on-drift
```

## Best Practices

### 1. Start with Examples
- Review existing examples before creating your own
- Copy and modify an example that matches your use case
- Follow the same directory structure

### 2. Specification Quality
- Be specific in feature descriptions
- Define clear success criteria
- Include API contracts for all endpoints
- Document constraints and dependencies

### 3. Policy Design
- Start with defaults, then tighten
- Balance security with developer experience
- Use appropriate coverage thresholds
- Enable caching for faster CI/CD

### 4. Iterative Development
- Generate plan, review, adjust spec, repeat
- Use dry-run mode to preview execution
- Leverage checkpoint/resume for long builds
- Check drift frequently

### 5. CI/CD Integration
- Enable Docker image caching
- Use SARIF format for findings
- Set appropriate fail conditions
- Monitor execution metrics

## Troubleshooting

### "Docker not available" Error
```bash
# Check Docker is running
docker version

# Start Docker Desktop (macOS)
open -a Docker
```

### Low Test Coverage
```bash
# View coverage details
go tool cover -html=coverage.txt  # Go projects
npm run test:coverage -- --verbose  # Node projects
```

### API Drift Detected
```bash
# View drift details
cat drift.sarif | jq '.runs[0].results[]'

# Common causes:
# - OpenAPI spec doesn't match implementation
# - Spec.yaml API definitions changed
# - Missing endpoints in implementation
```

### Policy Violations
```bash
# Review policy requirements
cat .specular/policy.yaml | grep -A 5 "require"

# Common violations:
# - Test coverage below threshold
# - Security vulnerabilities found
# - Linting errors
# - Build failures
```

### Slow CI/CD Runs
```bash
# Enable Docker image caching
specular build --enable-cache --cache-dir .specular/cache

# Pre-warm images before build
specular prewarm --plan plan.json

# In GitHub Actions, cache is automatic (94% speedup)
```

## Next Steps

1. **Try an example** - Start with web-app or api-service
2. **Review generated plan** - Understand how Specular breaks down work
3. **Modify the spec** - Add a small feature and regenerate
4. **Integrate with CI/CD** - Set up GitHub Actions workflow
5. **Create your own** - Use examples as templates

## Learn More

- [Getting Started Guide](../docs/getting-started.md) - Comprehensive documentation
- [Provider Guide](../docs/provider-guide.md) - LLM provider configuration
- [Technical Design](../docs/tech_design.md) - Architecture and design decisions
- [GitHub Action](../action.yml) - CI/CD integration reference

## Contributing

Have an example to share? Contributions welcome!

1. Create a new directory under `examples/`
2. Include complete spec.yaml, policy.yaml, and openapi.yaml
3. Write a comprehensive README.md
4. Submit a pull request

Example contributions we'd love to see:
- Mobile app (React Native, Flutter)
- Microservices architecture
- Data pipeline (ETL/ELT)
- Machine learning service
- GraphQL API
- Serverless application
- Desktop application (Electron, Tauri)

---

**Questions?** Open an issue at [github.com/felixgeelhaar/specular/issues](https://github.com/felixgeelhaar/specular/issues)
