# Specular Example Projects

A collection of sample projects demonstrating Specular's spec-first, AI-driven development workflow.

## Available Examples

| Project | Template | Description | Tech Stack |
|---------|----------|-------------|------------|
| [cli-task-manager](./cli-task-manager) | `cli-tool` | Command-line task manager | Go |
| [web-app-store](./web-app-store) | `web-app` | E-commerce storefront | TypeScript/Next.js |
| [api-bookstore](./api-bookstore) | `api-service` | RESTful bookstore API | Go/Chi |
| [microservice-orders](./microservice-orders) | `microservice` | Order processing service | Go/gRPC |
| [data-pipeline-etl](./data-pipeline-etl) | `data-pipeline` | Analytics ETL pipeline | Python/Airflow |

## Quick Start

1. **Choose an example** that matches your project type

2. **Initialize Specular** in the example directory:
   ```bash
   cd examples/projects/api-bookstore
   specular init --template api-service
   ```

3. **Review the spec** in `spec.yaml` and customize:
   ```bash
   specular interview --tui
   ```

4. **Generate a plan**:
   ```bash
   specular plan
   ```

5. **Execute with governance**:
   ```bash
   specular build --dry-run
   ```

## Using Templates

Start a new project with a template:

```bash
# Create new project directory
mkdir my-new-api
cd my-new-api

# Initialize with template
specular init --template api-service

# Customize through interview
specular interview --tui
```

Available templates:
- `web-app` - Web applications with frontend and backend
- `api-service` - RESTful or GraphQL API services
- `cli-tool` - Command-line tools and utilities
- `microservice` - Distributed microservices
- `data-pipeline` - ETL and data processing pipelines

## Example Structure

Each example includes:

```
example-project/
├── README.md           # Project overview
├── spec.yaml           # Product specification
└── .specular/          # Configuration (after init)
    ├── policy.yaml     # Governance policy
    ├── providers.yaml  # AI provider config
    └── spec.lock.json  # Locked specification
```

## Key Concepts Demonstrated

### Specification-First Development

Each `spec.yaml` defines:
- Product goals and target users
- Features with priorities (P0/P1/P2)
- API contracts and acceptance criteria
- Traceability links to code

### Governance and Policy

Templates include sensible defaults for:
- Docker-only execution
- Security scanning
- Test coverage requirements
- Code quality checks

### AI-Driven Workflow

Specular uses AI to:
- Generate execution plans from specs
- Route tasks to appropriate models
- Detect drift between spec and implementation
- Provide cost-aware model selection

## Contributing

To add a new example:

1. Create directory under `examples/projects/`
2. Add `README.md` with overview
3. Add `spec.yaml` with product specification
4. Update this README's table

## Learn More

- [Specular Documentation](https://github.com/felixgeelhaar/specular)
- [Getting Started Guide](../../docs/getting-started.md)
- [Spec Schema Reference](../../docs/spec-schema.md)
