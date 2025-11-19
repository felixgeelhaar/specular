# Order Service - Microservice Example

A microservice example demonstrating Specular's spec-first development for building a distributed order processing service.

## Project Type

This is a **microservice** template example showcasing:
- Event-driven architecture
- Message queue integration
- Service-to-service communication
- Saga pattern for distributed transactions

## Getting Started

```bash
cd examples/projects/microservice-orders
specular init --template microservice

# Generate and execute
specular plan
specular build --dry-run
```

## Spec Overview

The specification defines:
- Order creation and management
- Event publishing to message queue
- Integration with inventory service
- Saga orchestration

## Architecture

```
[API Gateway] -> [Order Service] -> [Message Queue]
                       |                  |
                       v                  v
              [PostgreSQL]        [Inventory Service]
```

## Files

- `spec.yaml` - Product specification
- `.specular/` - Generated configuration
