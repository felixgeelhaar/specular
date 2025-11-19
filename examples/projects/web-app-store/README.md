# E-commerce Store - Web Application Example

A complete web application example demonstrating Specular's spec-first development approach for building an e-commerce store.

## Project Type

This is a **web-app** template example showcasing:
- User authentication
- Product catalog
- Shopping cart
- Checkout flow

## Getting Started

```bash
# Initialize specular
cd examples/projects/web-app-store
specular init --template web-app

# Generate plan from spec
specular plan

# Review and execute
specular build --dry-run
```

## Spec Overview

The specification defines:
- 4 core features (P0)
- RESTful API for products and orders
- OAuth2 authentication
- PostgreSQL database

## Files

- `spec.yaml` - Product specification
- `.specular/` - Generated configuration

## Next Steps

1. Run `specular interview --tui` to customize the spec
2. Generate an execution plan with `specular plan`
3. Build with governance using `specular build`
