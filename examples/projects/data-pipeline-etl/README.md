# Analytics ETL Pipeline - Data Pipeline Example

A data pipeline example demonstrating Specular's spec-first development for building ETL workflows.

## Project Type

This is a **data-pipeline** template example showcasing:
- Extract, Transform, Load (ETL) patterns
- Scheduled batch processing
- Data validation and quality checks
- Multiple data source integration

## Getting Started

```bash
cd examples/projects/data-pipeline-etl
specular init --template data-pipeline

# Generate and execute
specular plan
specular build --dry-run
```

## Spec Overview

The specification defines:
- Data extraction from multiple sources
- Transformation rules and validation
- Loading to data warehouse
- Monitoring and alerting

## Pipeline Architecture

```
[Source DB] ─┐
             ├─> [Extract] -> [Transform] -> [Load] -> [Data Warehouse]
[API Data] ──┘        |            |           |
                      v            v           v
                 [Staging]   [Validation]  [Metrics]
```

## Files

- `spec.yaml` - Product specification
- `.specular/` - Generated configuration
