# Bookstore API - API Service Example

A RESTful API service example demonstrating Specular's spec-first development for building a bookstore backend.

## Project Type

This is an **api-service** template example showcasing:
- RESTful API design
- CRUD operations
- Authentication middleware
- Database integration

## Getting Started

```bash
cd examples/projects/api-bookstore
specular init --template api-service

# Generate and execute
specular plan
specular build --dry-run
```

## Spec Overview

The specification defines:
- Book management (CRUD)
- Author profiles
- Categories and search
- JWT authentication

## Files

- `spec.yaml` - Product specification
- `.specular/` - Generated configuration

## API Endpoints

- `GET /api/books` - List books
- `POST /api/books` - Create book
- `GET /api/books/{id}` - Get book details
- `PUT /api/books/{id}` - Update book
- `DELETE /api/books/{id}` - Delete book
