# Using Templates: Jump-Start Your Project

Templates provide pre-configured specifications for common project types, getting you productive immediately.

## Available Templates

| Template | Use Case | Tech Stack |
|----------|----------|------------|
| `web-app` | Full-stack web applications | React/Next.js, Node.js |
| `api-service` | RESTful or GraphQL APIs | Go, Python, Node.js |
| `cli-tool` | Command-line utilities | Go, Rust |
| `microservice` | Distributed services | Go, gRPC, Message queues |
| `data-pipeline` | ETL and data processing | Python, Airflow |

---

## Step 1: Initialize with Template

```bash
mkdir my-api
cd my-api
specular init --template api-service
```

This creates a pre-populated spec with:
- Common features for that project type
- Sensible default priorities
- Standard API patterns
- Appropriate tech stack

---

## Step 2: Review Generated Spec

```bash
cat .specular/spec.yaml
```

The template provides a starting point. You'll see:
- Placeholder product name
- Common features (auth, CRUD, search)
- Standard non-functional requirements
- Typical acceptance criteria

---

## Step 3: Customize for Your Project

### Option A: Interactive Customization

```bash
specular interview --tui
```

The interview will:
- Pre-fill template values
- Ask only for customizations
- Skip already-defined sections

### Option B: Direct Editing

```bash
vim .specular/spec.yaml
```

Key sections to customize:
- `product.name` - Your actual product name
- `product.description` - What it does
- `features[].title` - Rename to match your domain
- `features[].api` - Adjust endpoints
- `tech_stack` - Change if needed

---

## Step 4: Proceed with Workflow

Once customized:

```bash
# Lock the spec
specular spec lock

# Generate plan
specular plan

# Build
specular build --dry-run
```

---

## Template Deep Dives

### Web App Template

Best for: E-commerce, SaaS dashboards, content platforms

Pre-configured features:
- User authentication (OAuth2/JWT)
- Profile management
- Core CRUD operations
- Search and filtering

Default tech:
- Frontend: React/Next.js
- Backend: Node.js/Express
- Database: PostgreSQL
- Auth: JWT + OAuth2

```bash
specular init --template web-app
```

---

### API Service Template

Best for: Backend services, REST APIs, GraphQL servers

Pre-configured features:
- Resource CRUD endpoints
- Authentication middleware
- Pagination and filtering
- Health checks

Default tech:
- Language: Go or Python
- Framework: Chi, FastAPI, or Express
- Database: PostgreSQL
- Docs: OpenAPI/Swagger

```bash
specular init --template api-service
```

---

### CLI Tool Template

Best for: Developer tools, automation scripts, system utilities

Pre-configured features:
- Command structure (Cobra-style)
- Configuration management
- Output formatting (JSON, table)
- Shell completions

Default tech:
- Language: Go
- Framework: Cobra
- Config: Viper
- Output: Table writer

```bash
specular init --template cli-tool
```

---

### Microservice Template

Best for: Distributed systems, event-driven architectures

Pre-configured features:
- Service endpoints (HTTP + gRPC)
- Event publishing
- Service discovery
- Circuit breakers

Default tech:
- Language: Go
- Transport: gRPC + HTTP
- Messaging: RabbitMQ/Kafka
- Tracing: OpenTelemetry

```bash
specular init --template microservice
```

---

### Data Pipeline Template

Best for: ETL jobs, analytics, data warehousing

Pre-configured features:
- Data extraction
- Transformation rules
- Loading to warehouse
- Monitoring/alerting

Default tech:
- Language: Python
- Orchestration: Airflow
- Storage: Snowflake/BigQuery
- Quality: Great Expectations

```bash
specular init --template data-pipeline
```

---

## Combining with Examples

Each template has a corresponding example project:

```bash
# View example
ls examples/projects/api-bookstore/

# Copy and customize
cp -r examples/projects/api-bookstore my-api
cd my-api
specular init --force
```

---

## Creating Custom Templates

For organization-specific templates:

1. Create a spec file with your standards
2. Save to a shared location
3. Use as starting point:

```bash
# Copy org template
cp /shared/templates/org-api-spec.yaml .specular/spec.yaml

# Initialize around it
specular init --force --skip-spec
```

---

## Best Practices

1. **Start with closest template** - Customize rather than starting blank
2. **Review all features** - Remove what you don't need
3. **Adjust priorities** - P0/P1/P2 based on your MVP
4. **Update tech stack** - Match your team's expertise
5. **Keep acceptance criteria** - They drive quality gates

---

## Next Steps

- [Quick Start](./01-quick-start.md) - Basic setup
- [Full Workflow](./02-full-workflow.md) - Complete spec-to-build process
- [Example Projects](../../examples/projects/) - Reference implementations
