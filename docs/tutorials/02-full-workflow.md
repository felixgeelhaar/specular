# Full Workflow: Spec to Production

This tutorial walks through the complete Specular workflow from specification to production-ready code.

## Overview

```
[Spec] → [Plan] → [Build] → [Eval] → [Production]
```

Each step enforces governance and maintains traceability.

## Prerequisites

- Completed [Quick Start](./01-quick-start.md)
- Project with `.specular/spec.yaml`

---

## Step 1: Review and Lock Specification

First, ensure your spec is complete:

```bash
cat .specular/spec.yaml
```

Lock the specification to create immutable hashes:

```bash
specular spec lock
```

This generates `.specular/spec.lock.json` with:
- Per-feature blake3 hashes
- Generated OpenAPI stubs
- Test file paths

**Why lock?** Locked specs enable drift detection - you'll know if implementation diverges from spec.

---

## Step 2: Generate Execution Plan

Convert the spec into an actionable task DAG:

```bash
specular plan
```

Output: `.specular/plan.json`

Review the plan:

```bash
specular plan show
```

You'll see:
- Tasks ordered by dependency
- Priority assignments (P0 first)
- Skill tags (backend, frontend, infra)
- Model hints for AI routing

### Customizing the Plan

Edit priorities or add dependencies:

```bash
specular plan edit
```

Or manually edit `.specular/plan.json`.

---

## Step 3: Execute Build

Run the build with policy enforcement:

```bash
specular build --plan .specular/plan.json
```

### Dry Run First

Always preview before executing:

```bash
specular build --dry-run
```

This shows:
- Commands that will run
- Docker images to use
- Resource limits applied
- Policy checks performed

### What Happens During Build

1. **Policy preflight** - Validates all constraints
2. **Docker sandbox** - Each task runs in isolation
3. **AI routing** - Tasks routed to appropriate models
4. **Artifact collection** - Code, tests, configs saved
5. **Manifest logging** - Full audit trail

### Monitoring Progress

Watch build progress:

```bash
specular build --verbose
```

Or check logs:

```bash
specular logs --follow
```

---

## Step 4: Evaluate Results

After build, run evaluation to check quality:

```bash
specular eval
```

This performs:
- **Drift detection** - Spec vs implementation
- **Quality gates** - Tests, linting, coverage
- **Security scans** - Secrets, dependencies

### Understanding Drift Reports

Output: `.specular/drift.sarif`

View human-readable summary:

```bash
specular eval show
```

Types of drift:
- **Plan drift** - Task hashes don't match spec
- **Code drift** - Implementation diverges from API contract
- **Infra drift** - Config doesn't meet policy

### Fixing Drift

For each finding:

1. **Update implementation** to match spec, OR
2. **Update spec** if requirements changed, then re-lock

```bash
# After fixing
specular spec lock
specular eval
```

---

## Step 5: Iterate

Development is iterative. When requirements change:

```bash
# 1. Update spec
vim .specular/spec.yaml

# 2. Re-lock
specular spec lock

# 3. Re-plan
specular plan

# 4. Re-build (only changed tasks)
specular build --incremental

# 5. Re-eval
specular eval
```

---

## Step 6: Production Readiness

Before shipping, verify:

### Run Full Evaluation

```bash
specular eval --fail-on drift,lint,test,security
```

### Check Coverage

```bash
specular eval --min-coverage 0.80
```

### Generate Reports

```bash
specular eval --report sarif --output results.sarif
```

### Review Audit Trail

```bash
ls .specular/runs/
```

Each run has a manifest with:
- Input/output hashes
- Model selections
- Costs incurred
- Timestamps

---

## Workflow Summary

| Command | Purpose | Output |
|---------|---------|--------|
| `specular spec lock` | Lock spec with hashes | `spec.lock.json` |
| `specular plan` | Generate task DAG | `plan.json` |
| `specular build` | Execute with governance | Code artifacts |
| `specular eval` | Check drift and quality | `drift.sarif` |

---

## Best Practices

1. **Lock early** - Lock spec before any implementation
2. **Dry run always** - Preview builds before executing
3. **Fix drift immediately** - Don't let it accumulate
4. **Review manifests** - Understand what AI did
5. **Iterate small** - Small spec changes, frequent builds

---

## Next Steps

- [Using Templates](./03-using-templates.md) - Start faster with templates
- [CLI Reference](../CLI_REFERENCE.md) - All commands and options
- [Production Guide](../PRODUCTION_GUIDE.md) - Deployment best practices
