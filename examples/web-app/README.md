# TaskFlow - Web Application Example

This example demonstrates using Specular to build a modern task management web application with:
- React 18 frontend with TypeScript
- Node.js 20 backend with Express
- PostgreSQL 15 database
- Redis 7 for caching
- Real-time WebSocket collaboration
- Complete CI/CD integration

## Project Structure

```
web-app/
├── .specular/
│   ├── spec.yaml              # Product specification
│   ├── policy.yaml            # Security and quality policies
│   └── spec.lock.json         # Generated feature checksums
├── openapi.yaml               # API specification
├── plan.json                  # Generated implementation plan
├── src/
│   ├── frontend/              # React application (to be generated)
│   ├── backend/               # Node.js API server (to be generated)
│   └── database/              # Migrations and seeds (to be generated)
└── README.md                  # This file
```

## Quick Start

### Prerequisites
- Docker installed and running
- Specular CLI installed ([Installation guide](../../docs/installation.md))
- API keys configured (optional for plan generation):
  - `ANTHROPIC_API_KEY` for Claude
  - `OPENAI_API_KEY` for GPT-4
  - `GEMINI_API_KEY` for Gemini

### Step 1: Generate Implementation Plan

Generate a detailed plan from the specification:

```bash
cd examples/web-app
specular plan --spec .specular/spec.yaml --output plan.json
```

This creates:
- `plan.json` - Detailed implementation tasks
- `.specular/spec.lock.json` - Feature checksums for drift detection

### Step 2: Review the Plan

The generated plan includes:
- Task breakdown by feature
- Estimated complexity and priority
- Implementation order
- Test requirements
- Dependencies between tasks

```bash
# View plan summary
cat plan.json | jq '.tasks[] | {id, title, skill, priority}'
```

### Step 3: Execute the Build

Execute the plan with policy enforcement:

```bash
specular build \
  --plan plan.json \
  --policy .specular/policy.yaml \
  --verbose
```

This will:
1. Create Docker containers for isolated execution
2. Generate code following the plan
3. Run tests with >80% coverage requirement
4. Perform security scanning
5. Build production artifacts
6. Generate execution manifest

### Step 4: Detect Drift

After making changes, check for drift from the specification:

```bash
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --lock .specular/spec.lock.json \
  --api-spec openapi.yaml \
  --fail-on-drift
```

This checks:
- Plan drift (spec hash mismatches)
- Code drift (API conformance)
- Infrastructure drift (policy violations)

Results are output in SARIF format for CI/CD integration.

## Features Demonstrated

### 1. User Authentication (feat-001)
**Priority:** P0
**Implementation includes:**
- JWT-based session management
- Email/password authentication
- Password reset flow
- Secure token refresh
- Test coverage >90%

**API Endpoints:**
- `POST /api/auth/register` - User registration
- `POST /api/auth/login` - User login
- `POST /api/auth/logout` - Session termination
- `POST /api/auth/refresh` - Token refresh
- `POST /api/auth/reset-password` - Password reset

### 2. Task Management (feat-002)
**Priority:** P0
**Implementation includes:**
- Full CRUD operations for tasks
- Rich metadata (status, priority, due date, assignee)
- Filtering and pagination
- Soft delete for audit trail
- API response time <100ms

**API Endpoints:**
- `GET /api/tasks` - List tasks with filters
- `GET /api/tasks/{id}` - Get task details
- `POST /api/tasks` - Create task
- `PUT /api/tasks/{id}` - Update task
- `DELETE /api/tasks/{id}` - Soft delete task

### 3. Real-time Collaboration (feat-003)
**Priority:** P1
**Implementation includes:**
- WebSocket-based updates
- Automatic reconnection
- Update delivery <500ms
- Handles 100+ concurrent connections

**WebSocket:**
- `WS /api/ws/tasks` - Real-time task updates

### 4. React Frontend (feat-004)
**Priority:** P0
**Implementation includes:**
- Responsive design (mobile-first)
- React Query for data fetching
- Tailwind CSS for styling
- Lighthouse scores: Performance >85, A11y >90
- Component test coverage >80%

### 5. Database Layer (feat-005)
**Priority:** P0
**Implementation includes:**
- PostgreSQL schema (3NF normalized)
- Reversible migrations
- Optimized indexes
- Automated backups
- Query performance monitoring

## Policy Enforcement

The `policy.yaml` file enforces:

### Security
- ✅ Docker-only execution (no local execution)
- ✅ Secrets scanning (prevent committed secrets)
- ✅ Dependency scanning (npm audit)
- ✅ SAST scanning (static analysis)
- ✅ License compliance (MIT, Apache-2.0, BSD allowed)

### Quality
- ✅ Test coverage >80%
- ✅ TypeScript type checking
- ✅ ESLint linting
- ✅ Prettier formatting
- ✅ Build success required

### Performance
- ✅ Lighthouse thresholds enforced
- ✅ Bundle size limits (JS <500KB, CSS <100KB)
- ✅ API response time <200ms (p95)

### Compliance
- ✅ GDPR compliance enabled
- ✅ Audit logging required
- ✅ Encryption at rest and in transit

## CI/CD Integration

### GitHub Actions

Add to `.github/workflows/specular.yml`:

```yaml
name: Specular CI

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: felixgeelhaar/specular-action@v1
        with:
          command: eval
          spec-file: .specular/spec.yaml
          plan-file: plan.json
          lock-file: .specular/spec.lock.json
          api-spec: openapi.yaml
          fail-on-drift: true
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Upload SARIF results
        if: always()
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: drift.sarif
```

### Docker Image Caching

The GitHub Action automatically caches Docker images, providing:
- **94% speedup** for cached runs (80s → 5s)
- Automatic cache invalidation on spec changes
- Multi-platform support

## Development Workflow

### 1. Modify Specification

Edit `.specular/spec.yaml` to add or change features:

```yaml
features:
  - id: feat-006
    title: Task Comments
    desc: Add comments to tasks for team collaboration
    priority: P1
    # ... rest of feature definition
```

### 2. Regenerate Plan

```bash
specular plan --spec .specular/spec.yaml --output plan.json
```

### 3. Detect Drift

```bash
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --lock .specular/spec.lock.json \
  --report drift.sarif
```

If drift detected, review the SARIF report:
- Plan drift: Spec changed without updating lock
- Code drift: Implementation doesn't match API spec
- Infra drift: Policy violations

### 4. Execute Changes

```bash
specular build --plan plan.json --policy .specular/policy.yaml
```

### 5. Test and Validate

```bash
# Run tests
npm test

# Check coverage
npm run test:coverage

# Run e2e tests
npm run test:e2e

# Validate against OpenAPI
npm run validate:api
```

## Checkpoint and Resume

For long-running operations, use checkpoint/resume:

```bash
# Start build (will save checkpoints)
specular build \
  --plan plan.json \
  --policy .specular/policy.yaml \
  --checkpoint-dir .specular/checkpoints

# If interrupted, resume from last checkpoint
specular build \
  --plan plan.json \
  --policy .specular/policy.yaml \
  --resume \
  --checkpoint-id build-plan.json-<timestamp>
```

Checkpoints are saved every 30 seconds and include:
- Completed tasks
- In-progress tasks
- Task artifacts
- Execution metadata

## Docker Image Pre-warming

Speed up subsequent runs by pre-warming Docker images:

```bash
# Pre-warm all images from plan
specular prewarm --plan plan.json --verbose

# Pre-warm specific images
specular prewarm node:20-alpine postgres:15-alpine redis:7-alpine

# Export for CI/CD cache
specular prewarm --plan plan.json --export .specular/cache

# Import in CI/CD
specular prewarm --import .specular/cache
```

## Troubleshooting

### Build Fails with "Docker not available"

Ensure Docker is installed and running:
```bash
docker version
```

### Tests Fail with Coverage Below Threshold

Check which packages are below 80%:
```bash
npm run test:coverage -- --verbose
```

Add more tests to uncovered files.

### API Drift Detected

Compare OpenAPI spec with actual endpoints:
```bash
# View drift details
cat drift.sarif | jq '.runs[0].results[] | {message, location}'
```

Update implementation or OpenAPI spec to match.

### Policy Violations

Review policy requirements:
```bash
cat .specular/policy.yaml | grep -A 5 "require"
```

Fix violations and re-run build.

## Learn More

- [Specular Documentation](../../docs/getting-started.md)
- [Provider Guide](../../docs/provider-guide.md)
- [CI/CD Integration](../../docs/getting-started.md#cicd-integration-with-github-actions)
- [Policy Reference](../../docs/tech_design.md#policy-system)

## License

This example is provided under the MIT License as part of the Specular project.
