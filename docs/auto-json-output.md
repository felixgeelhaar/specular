# Autonomous Mode JSON Output Format

## Overview

Specular's autonomous mode can produce machine-readable JSON output for CI/CD integration and programmatic processing. This format provides complete execution details including step results, artifacts, metrics, and audit information.

## Enabling JSON Output

Use the `--json` flag with the `specular auto` command:

```bash
# Output JSON to stdout
specular auto --json "Build a REST API"

# Combine with other flags
specular auto --profile ci --json "Create authentication system"

# Save JSON output to file
specular auto --json "Build dashboard" > output.json
```

## Schema Version

The JSON output follows a versioned schema for backward compatibility:

- **Current Version**: `specular.auto.output/v1`
- **Schema Field**: `schema` (top-level string field)

## Output Structure

### Top-Level Fields

```json
{
  "schema": "specular.auto.output/v1",
  "goal": "User's original objective",
  "status": "completed|failed|partial",
  "steps": [...],
  "artifacts": [...],
  "metrics": {...},
  "audit": {...}
}
```

| Field | Type | Description |
|-------|------|-------------|
| `schema` | string | Output format version identifier |
| `goal` | string | User's original natural language goal |
| `status` | string | Overall execution outcome: `completed`, `failed`, or `partial` |
| `steps` | array | Detailed results for each workflow step |
| `artifacts` | array | Generated files and outputs |
| `metrics` | object | Execution statistics |
| `audit` | object | Provenance and compliance information |

### Step Results

Each step in the `steps` array contains:

```json
{
  "id": "step-1",
  "type": "spec:update",
  "status": "completed",
  "startedAt": "2025-01-15T10:30:00Z",
  "completedAt": "2025-01-15T10:30:05Z",
  "duration": 5000000000,
  "costUSD": 0.50,
  "error": "error message if failed",
  "warnings": ["warning messages"],
  "metadata": {}
}
```

#### Step Types

| Type | Description |
|------|-------------|
| `spec:update` | Specification generation from goal |
| `spec:lock` | Specification locking with hashes |
| `plan:gen` | Execution plan generation |
| `build:run` | Plan execution (building/testing) |

#### Step Status

| Status | Description |
|--------|-------------|
| `pending` | Step not yet started |
| `in_progress` | Step currently executing |
| `completed` | Step finished successfully |
| `failed` | Step encountered an error |
| `skipped` | Step was skipped |

### Artifacts

Generated files and outputs:

```json
{
  "path": "spec.yaml",
  "type": "spec",
  "size": 2048,
  "hash": "sha256:abc123...",
  "createdAt": "2025-01-15T10:30:05Z"
}
```

#### Artifact Types

- `spec` - Product specification
- `lock` - Locked specification with hashes
- `plan` - Execution plan
- `code` - Generated source code
- `test` - Generated test code
- `docs` - Generated documentation
- `config` - Configuration files

### Execution Metrics

Aggregate statistics:

```json
{
  "totalDuration": 19000000000,
  "totalCost": 1.81,
  "stepsExecuted": 4,
  "stepsFailed": 0,
  "stepsSkipped": 0,
  "policyViolations": 0,
  "tokensUsed": 125000,
  "retriesPerformed": 0
}
```

| Field | Type | Description |
|-------|------|-------------|
| `totalDuration` | number | Total execution time in nanoseconds |
| `totalCost` | number | Total cost in USD |
| `stepsExecuted` | number | Count of steps that ran |
| `stepsFailed` | number | Count of failed steps |
| `stepsSkipped` | number | Count of skipped steps |
| `policyViolations` | number | Count of policy check failures |
| `tokensUsed` | number | Total token consumption (optional) |
| `retriesPerformed` | number | Total retry attempts (optional) |

### Audit Trail

Provenance and compliance information:

```json
{
  "checkpointId": "auto-1736936400",
  "profile": "ci",
  "startedAt": "2025-01-15T10:30:00Z",
  "completedAt": "2025-01-15T10:30:19Z",
  "user": "ci-bot",
  "hostname": "ci-runner-01",
  "approvals": [...],
  "policies": [...],
  "version": "v1.4.0"
}
```

#### Approval Events

Tracks user approval interactions:

```json
{
  "stepId": "step-4",
  "timestamp": "2025-01-15T10:30:09Z",
  "approved": true,
  "reason": "build step requires approval",
  "user": "ci-bot"
}
```

#### Policy Events

Records policy check results:

```json
{
  "stepId": "step-1",
  "timestamp": "2025-01-15T10:30:00Z",
  "checkerName": "cost_limit",
  "allowed": true,
  "reason": "explanation if denied",
  "warnings": ["warning messages"],
  "metadata": {
    "estimated_cost": 0.50,
    "total_cost_so_far": 0.0
  }
}
```

## CI/CD Integration Examples

### GitHub Actions

```yaml
name: Autonomous Build

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Run Specular Auto Mode
        id: specular
        run: |
          specular auto --profile ci --json "Build the feature" > output.json

      - name: Parse Results
        run: |
          # Extract status
          STATUS=$(jq -r '.status' output.json)

          # Extract metrics
          COST=$(jq -r '.metrics.totalCost' output.json)
          DURATION=$(jq -r '.metrics.totalDuration' output.json)

          # Check for failures
          if [ "$STATUS" != "completed" ]; then
            echo "Build failed: $STATUS"
            exit 1
          fi

          echo "Build successful! Cost: \$$COST"

      - name: Upload Artifacts
        uses: actions/upload-artifact@v2
        with:
          name: specular-output
          path: output.json
```

### GitLab CI

```yaml
specular_build:
  stage: build
  script:
    - specular auto --profile ci --json "$BUILD_GOAL" > output.json
    - |
      # Validate execution
      if [ "$(jq -r '.status' output.json)" != "completed" ]; then
        echo "Execution failed"
        jq '.steps[] | select(.status=="failed")' output.json
        exit 1
      fi
    - echo "Cost: $(jq -r '.metrics.totalCost' output.json)"
  artifacts:
    reports:
      json: output.json
```

### Jenkins Pipeline

```groovy
pipeline {
  agent any

  stages {
    stage('Build') {
      steps {
        sh '''
          specular auto --profile ci --json "Build API" > output.json

          # Check status
          if [ "$(jq -r '.status' output.json)" != "completed" ]; then
            exit 1
          fi
        '''

        script {
          def output = readJSON file: 'output.json'
          echo "Total cost: $${output.metrics.totalCost}"
          echo "Steps executed: ${output.metrics.stepsExecuted}"
        }
      }
    }
  }

  post {
    always {
      archiveArtifacts artifacts: 'output.json', fingerprint: true
    }
  }
}
```

## Parsing Examples

### Python

```python
import json
import sys

with open('output.json', 'r') as f:
    output = json.load(f)

# Check status
if output['status'] != 'completed':
    print(f"Execution failed: {output['status']}")

    # Print failed steps
    for step in output['steps']:
        if step['status'] == 'failed':
            print(f"Step {step['id']} failed: {step['error']}")

    sys.exit(1)

# Report metrics
metrics = output['metrics']
print(f"✅ Execution completed successfully!")
print(f"   Total cost: ${metrics['totalCost']:.4f}")
print(f"   Duration: {metrics['totalDuration'] / 1e9:.2f}s")
print(f"   Steps executed: {metrics['stepsExecuted']}")

# List artifacts
print(f"\n📦 Generated {len(output['artifacts'])} artifacts:")
for artifact in output['artifacts']:
    print(f"   - {artifact['path']} ({artifact['type']})")
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
)

type AutoOutput struct {
    Schema    string          `json:"schema"`
    Goal      string          `json:"goal"`
    Status    string          `json:"status"`
    Steps     []StepResult    `json:"steps"`
    Artifacts []ArtifactInfo  `json:"artifacts"`
    Metrics   ExecutionMetrics `json:"metrics"`
    Audit     AuditTrail      `json:"audit"`
}

func main() {
    data, err := os.ReadFile("output.json")
    if err != nil {
        panic(err)
    }

    var output AutoOutput
    if err := json.Unmarshal(data, &output); err != nil {
        panic(err)
    }

    // Check status
    if output.Status != "completed" {
        fmt.Printf("Execution failed: %s\n", output.Status)
        os.Exit(1)
    }

    // Report metrics
    fmt.Printf("✅ Execution completed!\n")
    fmt.Printf("   Cost: $%.4f\n", output.Metrics.TotalCost)
    fmt.Printf("   Steps: %d\n", output.Metrics.StepsExecuted)
    fmt.Printf("   Artifacts: %d\n", len(output.Artifacts))
}
```

### Node.js

```javascript
const fs = require('fs');

const output = JSON.parse(fs.readFileSync('output.json', 'utf8'));

// Check status
if (output.status !== 'completed') {
    console.error(`Execution failed: ${output.status}`);

    // Find failed steps
    output.steps
        .filter(step => step.status === 'failed')
        .forEach(step => {
            console.error(`Step ${step.id} failed: ${step.error}`);
        });

    process.exit(1);
}

// Report metrics
console.log('✅ Execution completed!');
console.log(`   Cost: $${output.metrics.totalCost.toFixed(4)}`);
console.log(`   Duration: ${(output.metrics.totalDuration / 1e9).toFixed(2)}s`);
console.log(`   Artifacts: ${output.artifacts.length}`);

// Check for policy violations
if (output.metrics.policyViolations > 0) {
    console.warn(`⚠️  ${output.metrics.policyViolations} policy violations`);
    output.audit.policies
        .filter(p => !p.allowed)
        .forEach(p => {
            console.warn(`   ${p.checkerName}: ${p.reason}`);
        });
}
```

## Cost Tracking and Budgets

### Extracting Cost Information

```bash
# Get total cost
jq '.metrics.totalCost' output.json

# Get cost per step
jq '.steps[] | {id, cost: .costUSD}' output.json

# Check if budget was exceeded
BUDGET=5.00
COST=$(jq -r '.metrics.totalCost' output.json)
if (( $(echo "$COST > $BUDGET" | bc -l) )); then
    echo "Budget exceeded: \$$COST > \$$BUDGET"
fi
```

### Cost Alerts

```bash
# Send alert if cost exceeds threshold
THRESHOLD=10.00
COST=$(jq -r '.metrics.totalCost' output.json)
if (( $(echo "$COST > $THRESHOLD" | bc -l) )); then
    curl -X POST https://alerts.example.com/notify \
        -H "Content-Type: application/json" \
        -d "{\"message\": \"High cost detected: \$$COST\"}"
fi
```

## Error Handling

### Detecting Failures

```bash
#!/bin/bash

STATUS=$(jq -r '.status' output.json)

case "$STATUS" in
    completed)
        echo "✅ Success"
        exit 0
        ;;
    failed)
        echo "❌ Execution failed"
        jq -r '.steps[] | select(.status=="failed") | "Step \(.id): \(.error)"' output.json
        exit 1
        ;;
    partial)
        echo "⚠️  Partial completion"
        COMPLETED=$(jq '.metrics.stepsExecuted' output.json)
        FAILED=$(jq '.metrics.stepsFailed' output.json)
        echo "Completed: $COMPLETED, Failed: $FAILED"
        exit 2
        ;;
esac
```

### Retry Logic

```python
import json
import subprocess
import time

MAX_RETRIES = 3
RETRY_DELAY = 60  # seconds

for attempt in range(MAX_RETRIES):
    # Run specular
    subprocess.run(['specular', 'auto', '--json', 'Build API'],
                   stdout=open('output.json', 'w'))

    # Check result
    with open('output.json', 'r') as f:
        output = json.load(f)

    if output['status'] == 'completed':
        print(f"✅ Success on attempt {attempt + 1}")
        break

    if attempt < MAX_RETRIES - 1:
        print(f"⚠️  Attempt {attempt + 1} failed, retrying in {RETRY_DELAY}s...")
        time.sleep(RETRY_DELAY)
else:
    print("❌ All retry attempts failed")
    sys.exit(1)
```

## Schema Evolution

Future versions of the output schema will maintain backward compatibility:

- **Additive Changes**: New optional fields may be added
- **Breaking Changes**: Will increment schema version (e.g., `v2`)
- **Deprecation**: Old fields marked deprecated before removal

Always check the `schema` field to ensure compatibility:

```bash
EXPECTED="specular.auto.output/v1"
ACTUAL=$(jq -r '.schema' output.json)

if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "Warning: Schema version mismatch"
    echo "Expected: $EXPECTED"
    echo "Actual: $ACTUAL"
fi
```

## Best Practices

1. **Always Check Status**: Never assume success without checking `status` field
2. **Validate Schema**: Check `schema` version for compatibility
3. **Handle Errors**: Parse `error` fields from failed steps for debugging
4. **Track Costs**: Monitor `metrics.totalCost` to avoid budget overruns
5. **Archive Output**: Save JSON output for audit and debugging
6. **Parse Safely**: Use proper JSON parsing libraries, never use `eval()`
7. **Monitor Policies**: Check `metrics.policyViolations` for compliance issues

## See Also

- [Autonomous Mode Guide](./auto-mode.md)
- [Profile System Documentation](./profiles.md)
- [Policy Enforcement](./policy.md)
- [Exit Codes Reference](./exit-codes.md)
