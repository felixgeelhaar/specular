# Spec Schema Reference

This document provides a complete reference for the `spec.yaml` product specification format used by Specular.

## Overview

The `spec.yaml` file is the heart of Specular's spec-first development approach. It defines:
- Product goals and technical stack
- Features with priorities and acceptance criteria
- API contracts and endpoints
- Non-functional requirements
- Development milestones

## Schema Structure

```yaml
product:
  name: string          # Required: Product name
  description: string   # Optional: Product description
  goals: [string]       # Required: At least one goal
  tech_stack:           # Optional: Technical stack details
    language: string
    framework: string
    database: string
    runtime: string
    # ... additional custom fields

features:               # Required: At least one feature
  - id: string          # Required: Feature ID (format: feat-NNN)
    title: string       # Required: Feature title
    description: string # Required: Feature description
    priority: string    # Required: P0, P1, or P2
    api: [API]          # Optional: REST API endpoints
    grpc: [gRPC]        # Optional: gRPC services
    events: [Event]     # Optional: Event definitions
    acceptance: [string]# Required: Acceptance criteria (alias: success)
    trace: [string]     # Optional: Code traceability paths
    refs: [string]      # Optional: External references

non_functional:         # Optional: Non-functional requirements
  performance: [string]
  security: [string]
  scalability: [string]
  availability: [string]
  reliability: [string]
  observability: [string]

acceptance: [string]    # Required: Product-level acceptance criteria

milestones:             # Optional: Development milestones
  - id: string
    name: string
    features: [string]  # Feature IDs
    date: string        # Target date (YYYY-MM-DD)
    description: string
```

## Field Reference

### Product Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | The product or project name |
| `description` | string | No | Detailed description of the product |
| `goals` | [string] | Yes | List of high-level product goals (min 1) |
| `tech_stack` | object | No | Technical stack metadata |

**Example:**
```yaml
product:
  name: "Order Service"
  description: "Microservice for order processing"
  goals:
    - "Process customer orders reliably"
    - "Publish events to message queue"
  tech_stack:
    language: "Go"
    framework: "gRPC + HTTP"
    database: "PostgreSQL"
    runtime: "Go 1.22"
```

### Features Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier (format: `feat-NNN`) |
| `title` | string | Yes | Short feature title |
| `description` | string | Yes | Detailed feature description |
| `priority` | string | Yes | Priority level: `P0`, `P1`, or `P2` |
| `api` | [API] | No | REST API endpoint definitions |
| `grpc` | [gRPC] | No | gRPC service definitions |
| `events` | [Event] | No | Event/message definitions |
| `acceptance` | [string] | Yes | Acceptance criteria (min 1) |
| `trace` | [string] | No | Code file paths for traceability |
| `refs` | [string] | No | External reference links |

**Feature ID Format:**
- Must match pattern: `feat-NNN` (e.g., `feat-001`, `feat-123`)
- IDs must be unique within the spec

**Priority Levels:**
| Priority | Meaning |
|----------|---------|
| `P0` | Critical - Must have for MVP |
| `P1` | Important - Should have |
| `P2` | Nice to have - Can defer |

**Example:**
```yaml
features:
  - id: "feat-001"
    title: "User Authentication"
    description: "JWT-based user authentication with social login support"
    priority: "P0"
    api:
      - method: "POST"
        path: "/api/auth/login"
        request_body:
          email: "string"
          password: "string"
        response:
          token: "string"
    acceptance:
      - "Users can login with email/password"
      - "JWT token expires after 1 hour"
    trace:
      - "internal/auth/login.go"
      - "internal/auth/jwt.go"
```

### API Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `method` | string | Yes | HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS) |
| `path` | string | Yes | API path (must start with `/`) |
| `request_body` | object | No | Request body schema |
| `query_params` | object | No | Query parameter schema |
| `response` | object | No | Response body schema |

**Example:**
```yaml
api:
  - method: "GET"
    path: "/api/users"
    query_params:
      page: "number"
      limit: "number"
    response:
      users: "array"
      total: "number"

  - method: "POST"
    path: "/api/users"
    request_body:
      name: "string"
      email: "string"
    response:
      id: "string"
```

### gRPC Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `service` | string | Yes | gRPC service name |
| `method` | string | Yes | RPC method name |
| `request` | object | No | Request message schema |
| `response` | object | No | Response message schema |

**Example:**
```yaml
grpc:
  - service: "InventoryService"
    method: "ReserveItems"
    request:
      items: "array"
    response:
      reservation_id: "string"
```

### Event Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Event name (e.g., `order.created`) |
| `payload` | object | No | Event payload schema |

**Example:**
```yaml
events:
  - name: "order.created"
    payload:
      order_id: "string"
      customer_id: "string"
      total: "number"
```

### Non-Functional Requirements

All fields are optional arrays of requirement strings.

| Field | Description |
|-------|-------------|
| `performance` | Response times, throughput, latency |
| `security` | Authentication, authorization, encryption |
| `scalability` | Load handling, horizontal scaling |
| `availability` | Uptime, failover, redundancy |
| `reliability` | Data consistency, delivery guarantees |
| `observability` | Logging, metrics, tracing |

**Example:**
```yaml
non_functional:
  performance:
    - "API response < 100ms p95"
    - "Support 1000 concurrent users"
  security:
    - "JWT authentication required"
    - "Rate limiting: 100 req/min"
  observability:
    - "Structured JSON logging"
    - "OpenTelemetry tracing"
```

### Milestones Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | No | Milestone identifier |
| `name` | string | Yes | Milestone name (e.g., "MVP", "Beta") |
| `features` | [string] | Yes | List of feature IDs included (min 1) |
| `date` | string | No | Target date (YYYY-MM-DD format) |
| `description` | string | No | Milestone description |

**Example:**
```yaml
milestones:
  - name: "MVP"
    features: ["feat-001", "feat-002"]
    date: "2025-02-15"

  - name: "Beta"
    features: ["feat-003", "feat-004"]
    date: "2025-03-01"
```

## Schema Types

The schema uses the following type conventions for defining request/response bodies:

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text value | `"hello"` |
| `number` | Numeric value | `42`, `3.14` |
| `boolean` | True/false | `true`, `false` |
| `array` | List of items | `[1, 2, 3]` |
| `object` | Nested structure | `{"key": "value"}` |

## Validation Rules

### Required Fields
1. `product.name` must be non-empty
2. `product.goals` must have at least one goal
3. `features` must have at least one feature
4. Each feature must have: `id`, `title`, `description`, `priority`, and at least one `acceptance` criterion
5. `acceptance` (product-level) must have at least one criterion

### Feature ID Validation
- Format: `feat-NNN` where NNN is a number
- IDs must be unique across all features
- Referenced IDs in milestones must exist

### API Validation
- Method must be valid HTTP method
- Path must start with `/`

### Priority Validation
- Must be exactly: `P0`, `P1`, or `P2`

## Complete Example

```yaml
# spec.yaml - Complete Example
product:
  name: "Task Manager API"
  description: "RESTful API for managing tasks and projects"
  goals:
    - "Enable users to create and manage tasks"
    - "Support project-based task organization"
    - "Provide real-time task status updates"
  tech_stack:
    language: "Go"
    framework: "Chi router"
    database: "PostgreSQL"
    runtime: "Go 1.22"

features:
  - id: "feat-001"
    title: "Task CRUD"
    description: "Create, read, update, and delete tasks"
    priority: "P0"
    api:
      - method: "GET"
        path: "/api/tasks"
        query_params:
          status: "string"
          project_id: "string"
        response:
          tasks: "array"
      - method: "POST"
        path: "/api/tasks"
        request_body:
          title: "string"
          description: "string"
          project_id: "string"
        response:
          id: "string"
      - method: "GET"
        path: "/api/tasks/{id}"
      - method: "PUT"
        path: "/api/tasks/{id}"
      - method: "DELETE"
        path: "/api/tasks/{id}"
    acceptance:
      - "Tasks can be created with title and optional description"
      - "Tasks can be filtered by status and project"
      - "Soft delete preserves task history"
    trace:
      - "internal/handlers/tasks.go"
      - "internal/repository/tasks.go"

  - id: "feat-002"
    title: "Task Status Workflow"
    description: "Manage task status transitions"
    priority: "P1"
    api:
      - method: "PATCH"
        path: "/api/tasks/{id}/status"
        request_body:
          status: "string"
    acceptance:
      - "Status transitions: todo -> in_progress -> done"
      - "Invalid transitions return 422 error"
    trace:
      - "internal/workflow/status.go"

non_functional:
  performance:
    - "API response < 100ms p95"
    - "Support 100 req/sec per user"
  security:
    - "JWT authentication required"
    - "Rate limiting: 1000 req/min"
  observability:
    - "Structured JSON logging"
    - "Prometheus metrics at /metrics"

acceptance:
  - "All endpoints require authentication"
  - "API responses follow JSON:API format"
  - "Errors include correlation ID"

milestones:
  - name: "MVP"
    features: ["feat-001"]
    date: "2025-02-01"
  - name: "v1.0"
    features: ["feat-001", "feat-002"]
    date: "2025-03-01"
```

## Spec Lock File

When you run `specular spec lock`, Specular generates a `spec.lock.json` file that contains:
- Canonical (hashed) version of each feature
- Generated artifact paths (OpenAPI specs, test files)
- Version tracking for drift detection

```json
{
  "version": "1.0.0",
  "features": {
    "feat-001": {
      "hash": "blake3:abc123...",
      "openapi_path": ".specular/openapi/feat-001.yaml",
      "test_paths": [".specular/tests/feat-001_test.go"]
    }
  }
}
```

## Related Documentation

- [Getting Started Guide](./getting-started.md) - Initial setup and first spec
- [CLI Reference](./CLI_REFERENCE.md) - Command documentation
- [Plugin Development](./plugin-development/README.md) - Extending Specular
- [Example Projects](../examples/projects/README.md) - Sample specifications
