# Autonomous Mode

Specular's autonomous mode enables you to describe what you want to build in natural language, and the system will generate a specification, create an execution plan, and implement it automatically using AI and Docker sandboxes.

## Overview

Autonomous mode combines several Specular features into a seamless end-to-end workflow:

1. **Goal → Spec**: AI converts your natural language goal into a structured YAML specification
2. **Spec Locking**: Generates cryptographic hashes for drift detection
3. **Plan Generation**: Creates a task dependency graph (DAG) for execution
4. **Approval Gate**: Optional interactive review before execution (TUI)
5. **Task Execution**: Runs tasks in isolated Docker sandboxes with retry logic
6. **Budget Enforcement**: Pre-flight checks and warnings to prevent cost overruns

## Quick Start

```bash
# Basic usage - describe what you want to build
./specular auto "Build a REST API that manages todo items"

# Skip the approval step (auto-approve)
./specular auto "Build a calculator CLI" --no-approval

# Generate spec and plan without executing
./specular auto "Build a web scraper" --dry-run

# With budget limits
./specular auto "Build a microservices platform" \
  --max-cost 10.0 \
  --max-cost-per-task 2.0
```

## Command Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--no-approval` | false | Skip interactive approval gate, auto-approve plan |
| `--dry-run` | false | Generate spec and plan but don't execute tasks |
| `--max-cost` | 5.0 | Maximum total cost in USD for the entire workflow |
| `--max-cost-per-task` | 1.0 | Maximum cost in USD per individual task |
| `--max-retries` | 3 | Maximum retry attempts per failed task |
| `--timeout` | 30 | Overall workflow timeout in minutes |
| `--verbose` | false | Enable detailed output including AI reasoning |

## How It Works

### 1. Goal → Spec Generation

The system uses AI to convert your natural language goal into a structured specification:

```yaml
product: Todo API
goals:
  - Provide a REST API for managing todo items
  - Support CRUD operations

features:
  - id: todo-endpoints
    title: Todo REST Endpoints
    desc: Implement CRUD endpoints for todo items
    priority: P0
    success:
      - Users can create new todo items
      - Users can list all todo items
      - Users can update existing items
      - Users can delete items
    trace:
      - Use RESTful design principles
      - Return appropriate HTTP status codes
```

**Cost Estimation**: ~2000 tokens (system prompt + goal + response)

### 2. Spec Locking

Generates cryptographic hashes (BLAKE3) for each feature to enable drift detection:

```json
{
  "version": "1.0.0",
  "features": {
    "todo-endpoints": {
      "hash": "abc123...",
      "openapi_path": "",
      "test_paths": []
    }
  }
}
```

### 3. Plan Generation

Creates a task dependency graph with estimated complexity:

```yaml
tasks:
  - id: task-001
    name: implement-todo-endpoints
    feature_id: todo-endpoints
    priority: P0
    complexity: 7
    depends_on: []
    command: docker run --rm golang:1.22 go build ./...
```

**Cost Estimation**: 1000 + (features × 800) tokens

### 4. Approval Gate (Optional)

Interactive TUI showing:
- Product name and goals
- Feature breakdown by priority
- Task count and complexity
- Estimated cost breakdown

Use `--no-approval` to skip this step.

### 5. Task Execution

Each task runs in an isolated Docker sandbox:

```bash
🚀 Executing plan...

🚀 Execution attempt 1/3...
Executing task task-001 (implement-todo-endpoints)...
  ✓ Completed in 245ms

✅ Auto mode completed in 42s
   Total cost: $0.0234
   Tasks executed: 3
```

**Features**:
- Retry logic with exponential backoff
- Progress indicators with task status
- Checkpoint/resume capability
- Docker manifest generation

**Cost Estimation**: Conservative 20% AI usage (most tasks use Docker, not AI)

### 6. Budget Enforcement

Pre-flight checks before each major operation:

```bash
Router initialized: budget=$5.00

⚠️  Budget Warning: 50% of budget used (will be at 55% after spec generation)

📋 Generating execution plan...

⚠️  Budget Warning: 75% of budget used (currently at 78.5%)
```

**Thresholds**:
- 50% usage: Warning
- 75% usage: Warning
- 90% usage: Critical warning
- 100% usage: Operation blocked

## Budget Management

### Setting Budget Limits

```bash
# Set overall budget limit
./specular auto "Build app" --max-cost 10.0

# Set per-task limit to prevent expensive operations
./specular auto "Build app" \
  --max-cost 10.0 \
  --max-cost-per-task 2.0
```

### Cost Estimation

The system uses conservative heuristics to estimate costs:

| Operation | Estimation Method |
|-----------|------------------|
| Spec Generation | ~2000 tokens (system + goal + response) |
| Plan Generation | 1000 + (features × 800) tokens |
| Task Execution | 20% AI usage (most tasks use Docker) |

**Note**: Costs are $0 when using Ollama (local models)

### Budget Exceeded Behavior

When budget is insufficient:

```bash
Error: budget check failed: insufficient budget for spec generation:
       estimated cost $0.0250 exceeds remaining budget $0.02 (limit: $5.00)
```

The operation is blocked to prevent unexpected costs.

## Checkpoint & Resume

Autonomous mode automatically saves checkpoints for long-running workflows:

```bash
.specular/checkpoints/
└── auto-1762811730.json
```

**Checkpoint Contents**:
- Operation ID and timestamps
- Goal and product name
- Task status (pending, in_progress, completed, failed)
- Execution metadata

**Resume Capability**: Coming soon - ability to resume from failed executions

## Generated Artifacts

### Directory Structure

```
.specular/
├── checkpoints/          # Resume checkpoints
│   └── auto-*.json
└── manifests/            # Docker execution records
    └── YYYYMMDD_HHMMSS_task-*.json
```

### Manifest Format

Each executed task generates a manifest:

```json
{
  "timestamp": "2025-11-10T22:55:31.069469+01:00",
  "step_id": "task-001",
  "runner": "docker",
  "image": "golang:1.22",
  "command": ["go", "version"],
  "exit_code": 0,
  "duration": "180.640458ms",
  "input_hashes": {},
  "output_hashes": {}
}
```

## Examples

### Example 1: Simple Calculator CLI

```bash
./specular auto "Create a simple calculator CLI that supports +, -, *, /" --no-approval

# Output:
# ✅ Generated spec: Simple Calculator CLI
#    Features: 2
# ✅ Plan created: 2 tasks
# ✅ Auto mode completed in 31s
#    Tasks executed: 2
```

### Example 2: REST API with Budget Limit

```bash
./specular auto "Build a REST API with authentication and CRUD endpoints" \
  --max-cost 5.0 \
  --verbose

# Output includes budget warnings:
# ⚠️  Budget Warning: 50% of budget used
```

### Example 3: Dry Run Mode

```bash
./specular auto "Build a microservices architecture" --dry-run

# Generates spec and plan without executing:
# 🏁 Dry run complete (no execution)
# ✅ Auto mode completed in 79s
#    Total cost: $0.0000
#    Tasks executed: 0
```

### Example 4: Complex Multi-Service Application

```bash
./specular auto "Build an e-commerce platform with product catalog, orders, and inventory" \
  --no-approval \
  --max-cost 10.0 \
  --max-retries 5 \
  --timeout 60

# Output:
# ✅ Generated spec: E-commerce Platform
#    Features: 5
# ✅ Plan created: 5 tasks
#
# 🚀 Executing plan...
# Executing task task-001 (product-catalog-service)...
#   ✓ Completed in 162ms
# Executing task task-002 (order-processing-service)...
#   ✓ Completed in 150ms
# ...
#
# ✅ Auto mode completed in 1m17s
#    Total cost: $0.0000
#    Tasks executed: 5
```

## Best Practices

### 1. Start Simple

Begin with small, focused goals to verify the system works correctly:

```bash
# Good: Specific and focused
./specular auto "Build a REST API with 2 endpoints: GET /health and POST /echo"

# Avoid: Too vague or complex initially
./specular auto "Build the next Facebook"
```

### 2. Use Dry Run First

Preview the generated spec and plan before executing:

```bash
./specular auto "Build a web scraper" --dry-run
# Review the output
./specular auto "Build a web scraper" --no-approval  # Execute if satisfied
```

### 3. Set Appropriate Budget Limits

Prevent unexpected costs by setting reasonable limits:

```bash
# For exploration and testing
./specular auto "Build app" --max-cost 1.0 --dry-run

# For production use
./specular auto "Build app" --max-cost 10.0 --max-cost-per-task 2.0
```

### 4. Use Ollama for Zero-Cost Testing

Test autonomous mode without API costs:

```bash
# Configure Ollama provider in .specular/providers.yaml
providers:
  - name: ollama
    type: ollama
    config:
      base_url: http://localhost:11434

# Run autonomous mode (no API costs)
./specular auto "Build app" --no-approval
```

### 5. Review Approval Gate

When using interactive mode, carefully review:
- Generated features match your intent
- Task complexity and priorities are appropriate
- Estimated costs are acceptable

### 6. Monitor Execution

Use `--verbose` to see detailed execution logs:

```bash
./specular auto "Build app" --verbose --no-approval
```

## Troubleshooting

### "Budget check failed" Error

**Problem**: Estimated cost exceeds remaining budget

**Solution**: Increase budget limit or simplify the goal

```bash
# Increase budget
./specular auto "Build app" --max-cost 20.0

# Or simplify the goal
./specular auto "Build a simple REST API with 2 endpoints"
```

### "Parse generated spec" Error

**Problem**: AI generated invalid YAML format

**Solution**: Retry with a clearer, more specific goal

```bash
# Too vague
./specular auto "Build something cool"

# Better
./specular auto "Build a REST API that manages todo items with CRUD operations"
```

### Tasks Failing During Execution

**Problem**: Docker tasks fail with non-zero exit codes

**Solution**:
1. Check Docker is running
2. Review task manifests in `.specular/manifests/`
3. Increase retry attempts: `--max-retries 5`

### Slow Execution

**Problem**: Autonomous mode takes too long

**Solution**:
1. Use faster AI model (configure in router.yaml)
2. Reduce goal complexity
3. Use `--dry-run` for initial exploration

## Advanced Usage

### Custom Provider Configuration

Configure AI providers in `.specular/providers.yaml`:

```yaml
providers:
  - name: openai-fast
    type: openai
    config:
      api_key: ${OPENAI_API_KEY}
      model: gpt-4o-mini  # Faster, cheaper model
      temperature: 0.3
      max_tokens: 2000

  - name: ollama-local
    type: ollama
    config:
      base_url: http://localhost:11434
      model: llama3.2
```

### Router Configuration

Configure model routing in `.specular/router.yaml`:

```yaml
router:
  budget:
    limit_usd: 10.0

  hints:
    agentic:
      - provider: ollama-local
        model: llama3.2
        weight: 10
```

### Environment Variables

Configure behavior via environment:

```bash
# Set default budget
export SPECULAR_MAX_COST=5.0

# Set default timeout
export SPECULAR_TIMEOUT=60

# Disable approval gate by default
export SPECULAR_NO_APPROVAL=true

./specular auto "Build app"
```

## Architecture

### Workflow Pipeline

```
User Goal (Natural Language)
        ↓
[Pre-flight Budget Check]
        ↓
[1. Spec Generation] ← AI (Ollama/OpenAI/etc.)
        ↓ (ProductSpec YAML)
[Pre-flight Budget Check]
        ↓
[2. Spec Locking] ← BLAKE3 hashing
        ↓ (SpecLock JSON)
[3. Plan Generation] ← AI + task analysis
        ↓ (Plan with task DAG)
[Pre-flight Budget Check]
        ↓
[4. Approval Gate] ← Optional TUI (bubbletea)
        ↓ (User approval)
[5. Task Execution] ← Docker + retry logic
        ↓ (Execution results)
[6. Checkpoint Save] ← State persistence
        ↓
Complete ✅
```

### Components

| Component | Responsibility |
|-----------|----------------|
| **GoalParser** | Natural language → YAML spec conversion |
| **SpecLocker** | Cryptographic hashing for drift detection |
| **PlanGenerator** | Task DAG creation with complexity estimation |
| **ApprovalGate** | Interactive TUI for plan review |
| **TaskExecutor** | Docker execution with retry logic |
| **CheckpointManager** | State persistence for resume capability |
| **BudgetEnforcer** | Pre-flight checks and threshold warnings |

### Technologies

- **AI Models**: Ollama, OpenAI, Anthropic, Gemini (via router)
- **Containerization**: Docker with isolated sandboxes
- **TUI Framework**: Bubbletea + Lipgloss
- **Hashing**: BLAKE3 for deterministic drift detection
- **State Management**: JSON-based checkpoints

## Limitations

### Current Limitations

1. **Resume Capability**: Not yet implemented (checkpoints saved but resume not available)
2. **AI Generation Quality**: Depends on model capability and prompt clarity
3. **Docker Requirement**: All tasks execute in Docker (no native execution)
4. **Cost Estimation**: Heuristic-based, may not be 100% accurate
5. **Interactive Approval**: Terminal UI only (no web interface)

### Known Issues

1. **Complex Goals**: Very vague goals may produce suboptimal specs
2. **Provider Errors**: OpenAI API errors require API key configuration
3. **Docker Performance**: First run slower due to image pulls
4. **Budget Tracking**: Ollama shows $0 cost (local execution)

## Security Considerations

### Docker Isolation

All tasks run in isolated Docker containers:
- No access to host filesystem (unless explicitly mounted)
- No network access (unless explicitly enabled)
- Resource limits enforced by Docker

### Policy Enforcement

Configure allowed Docker images in `.specular/policy.yaml`:

```yaml
docker:
  allowed_images:
    - golang:1.22
    - python:3.12
    - node:20

  resource_limits:
    memory: 2gb
    cpus: 2
```

### API Keys

Store API keys securely:

```bash
# Use environment variables
export OPENAI_API_KEY="sk-..."

# Or configure in providers.yaml with env var reference
config:
  api_key: ${OPENAI_API_KEY}
```

Never commit API keys to version control.

## Performance

### Typical Execution Times

| Goal Complexity | Features | Tasks | Spec Gen | Plan Gen | Execution | Total |
|-----------------|----------|-------|----------|----------|-----------|-------|
| Simple (CLI) | 1-2 | 1-2 | 5-10s | 5-10s | 1-5s | 15-30s |
| Medium (API) | 3-5 | 3-5 | 10-20s | 10-20s | 5-15s | 30-60s |
| Complex (Services) | 6-10 | 6-10 | 20-40s | 20-40s | 15-30s | 60-120s |

**Note**: First run slower due to Docker image pulls

### Optimization Tips

1. **Use Local Models**: Ollama for zero network latency
2. **Cache Docker Images**: Pre-pull images with `docker pull`
3. **Parallel Execution**: Tasks without dependencies run in parallel
4. **Faster Models**: Use gpt-4o-mini instead of gpt-4o

## Testing

The autonomous mode has comprehensive test coverage:

```bash
# Run all autonomous mode tests
go test ./internal/auto/... -v

# Run budget enforcement tests
go test ./internal/auto/budget_test.go -v

# Run with race detection
go test ./internal/auto/... -race

# Test coverage report
go test ./internal/auto/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Test Coverage**: 64 tests, 100% pass rate

## Related Documentation

- [Plan Generation](./plan-generation.md) - Task DAG creation
- [Docker Execution](./docker-execution.md) - Sandbox execution
- [Router Configuration](./router.md) - AI provider routing
- [Budget Management](./budget.md) - Cost control
- [Checkpoint System](./checkpoints.md) - Resume capability

## Support

For issues or questions:
- GitHub Issues: https://github.com/yourusername/specular/issues
- Documentation: https://docs.specular.dev
- Examples: https://github.com/yourusername/specular/tree/main/examples

## Changelog

### v1.0.0 (Current)
- ✅ Goal → Spec generation with AI
- ✅ Spec locking with BLAKE3 hashing
- ✅ Plan generation with task DAG
- ✅ Approval gate with TUI
- ✅ Docker sandbox execution
- ✅ Budget enforcement with warnings
- ✅ Checkpoint/resume (save only)
- ✅ Cost tracking and reporting

### Future Enhancements
- 🔄 Resume from checkpoints
- 🔄 Web-based approval interface
- 🔄 Real-time streaming output
- 🔄 Multi-stage builds
- 🔄 Distributed execution
- 🔄 Cost optimization suggestions
