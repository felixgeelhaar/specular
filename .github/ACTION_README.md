# Specular GitHub Action

Official GitHub Action for integrating [Specular](https://github.com/felixgeelhaar/specular) into your CI/CD pipeline. Automatically detect drift, enforce policies, and maintain spec-code alignment.

## Quick Start

```yaml
name: Drift Detection
on: [pull_request]

permissions:
  contents: read
  security-events: write

jobs:
  drift-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Check for Drift
        uses: felixgeelhaar/specular@v1
        with:
          command: drift
          anthropic-api-key: \${{ secrets.ANTHROPIC_API_KEY }}
```

## Features

- ✅ **Automatic Drift Detection**: Find spec-code mismatches in PRs
- 🔒 **Policy Enforcement**: Enforce organizational standards
- 📊 **SARIF Integration**: Results appear in GitHub Security tab
- 🤖 **Multi-Provider AI**: Support for Anthropic, OpenAI, Google
- 📝 **Job Summaries**: Rich PR comments with findings
- 🎯 **Flexible Configuration**: Customize for your workflow

## Commands

The action supports four main commands:

### 1. `drift` - Detect Specification Drift

Detect when code diverges from the specification:

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    command: drift
    spec-file: .specular/spec.yaml
    plan-file: plan.json
    sarif-output: drift-results.sarif
    upload-sarif: true
```

**Outputs SARIF with:**
- Drift violations by feature
- Code vs spec mismatches
- Missing implementations
- Unexpected implementations

### 2. `eval` - Evaluation & Testing

Run comprehensive evaluation including tests, linting, and security checks:

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    command: eval
    plan-file: plan.json
    policy-file: .specular/policy.yaml
    fail-on: drift,test,security
```

**Checks:**
- Unit and integration tests
- Linting and formatting
- Security vulnerabilities
- Coverage thresholds

### 3. `build` - Execute Build

Execute the build plan with policy enforcement:

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    command: build
    plan-file: plan.json
    policy-file: .specular/policy.yaml
    anthropic-api-key: \${{ secrets.ANTHROPIC_API_KEY }}
```

**Features:**
- Docker-based sandboxing
- Resource limits
- Policy gates
- Run manifests

### 4. `plan` - Generate Execution Plan

Generate an execution plan from a specification:

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    command: plan
    spec-file: .specular/spec.yaml
    plan-file: plan.json
```

**Generates:**
- Task DAG with dependencies
- Complexity estimates
- Model routing hints
- Priority assignments

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `command` | Command to run (drift, eval, build, plan) | Yes | - |
| `version` | Specular version | No | `latest` |
| `spec-file` | Path to spec file | No | `.specular/spec.yaml` |
| `plan-file` | Path to plan file | No | `plan.json` |
| `policy-file` | Path to policy file | No | `.specular/policy.yaml` |
| `fail-on` | Conditions to fail on (comma-separated) | No | `drift,test,security` |
| `sarif-output` | SARIF output file path | No | `specular-results.sarif` |
| `upload-sarif` | Upload SARIF to Code Scanning | No | `true` |
| `anthropic-api-key` | Anthropic API key | No | - |
| `openai-api-key` | OpenAI API key | No | - |
| `google-api-key` | Google API key | No | - |
| `additional-args` | Additional CLI arguments | No | `''` |

## Outputs

| Output | Description |
|--------|-------------|
| `result` | success or failure |
| `drift-count` | Number of drift violations |
| `test-count` | Number of test failures |
| `security-count` | Number of security issues |
| `sarif-file` | Path to SARIF file |

## AI Provider Setup

### Anthropic (Recommended)

1. Get API key from [Anthropic Console](https://console.anthropic.com/)
2. Add to repository secrets as `ANTHROPIC_API_KEY`
3. Use in workflow:

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    anthropic-api-key: \${{ secrets.ANTHROPIC_API_KEY }}
```

### OpenAI

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    openai-api-key: \${{ secrets.OPENAI_API_KEY }}
```

### Google

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    google-api-key: \${{ secrets.GOOGLE_API_KEY }}
```

## Example Workflows

### Drift Detection on Pull Requests

```yaml
name: PR Drift Check
on:
  pull_request:
    branches: [main]
    paths:
      - 'src/**'
      - '.specular/**'

permissions:
  contents: read
  security-events: write
  pull-requests: write

jobs:
  drift:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Detect Drift
        id: drift
        uses: felixgeelhaar/specular@v1
        with:
          command: drift
          anthropic-api-key: \${{ secrets.ANTHROPIC_API_KEY }}

      - name: Comment on PR
        if: failure()
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '⚠️ **Drift Detected:** \${{ steps.drift.outputs.drift-count }} violations found. Check the Security tab for details.'
            })
```

### Continuous Build Pipeline

```yaml
name: Continuous Build
on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3

      - name: Generate Plan
        uses: felixgeelhaar/specular@v1
        with:
          command: plan
          spec-file: .specular/spec.yaml
          plan-file: plan.json

      - name: Execute Build
        uses: felixgeelhaar/specular@v1
        with:
          command: build
          plan-file: plan.json
          anthropic-api-key: \${{ secrets.ANTHROPIC_API_KEY }}

      - name: Run Evaluation
        uses: felixgeelhaar/specular@v1
        with:
          command: eval
          plan-file: plan.json
          fail-on: drift,test,security
```

### Multi-Stage Validation

```yaml
name: Multi-Stage Validation
on: [pull_request]

jobs:
  plan-validation:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Validate Plan
        uses: felixgeelhaar/specular@v1
        with:
          command: plan
          spec-file: .specular/spec.yaml

  drift-detection:
    needs: plan-validation
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Check Drift
        uses: felixgeelhaar/specular@v1
        with:
          command: drift
          anthropic-api-key: \${{ secrets.ANTHROPIC_API_KEY }}

  security-scan:
    needs: drift-detection
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Security Check
        uses: felixgeelhaar/specular@v1
        with:
          command: eval
          fail-on: security
```

## Advanced Configuration

### Custom Policy File

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    command: eval
    policy-file: .specular/custom-policy.yaml
    fail-on: drift,lint,test,security,coverage
```

### Specific Version

```yaml
- uses: felixgeelhaar/specular@v1.6.0
  with:
    version: v1.6.0
    command: drift
```

### Additional CLI Arguments

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    command: drift
    additional-args: '--verbose --explain'
```

## SARIF Integration

Results automatically upload to GitHub Security → Code Scanning:

- View findings inline in PR diff
- Track drift over time
- Set up alerts and notifications
- Export for compliance reporting

### Disable SARIF Upload

```yaml
- uses: felixgeelhaar/specular@v1
  with:
    command: drift
    upload-sarif: false
```

## Permissions

Required GitHub token permissions:

```yaml
permissions:
  contents: read        # Read repository code
  security-events: write # Upload SARIF results
  pull-requests: write  # Comment on PRs (optional)
```

## Troubleshooting

### Action Fails to Install

**Issue:** Download fails or binary not found

**Solution:**
- Check internet connectivity
- Verify release v1.6.0 exists
- Check platform compatibility (linux/macos, amd64/arm64)

### SARIF Upload Fails

**Issue:** Code scanning upload permission denied

**Solution:**
- Add `security-events: write` permission
- Enable Code Scanning in repository settings
- Ensure Advanced Security is enabled (for private repos)

### Drift Command Returns No Results

**Issue:** No drift detected when expected

**Solution:**
- Verify spec file exists and is valid
- Check plan file is up to date
- Ensure code changes match spec expectations
- Use `--explain` flag for AI reasoning

### AI Provider Authentication Fails

**Issue:** API key invalid or quota exceeded

**Solution:**
- Verify secret name matches workflow
- Check API key is valid and active
- Monitor API usage and quotas
- Try alternative provider

## Performance Tips

1. **Cache Dependencies**: Use actions/cache for repeated runs
2. **Parallel Jobs**: Run drift/eval/build in parallel when possible
3. **Conditional Execution**: Use `paths` filter to skip unnecessary runs
4. **Provider Selection**: Choose faster models for quick checks

## Security Best Practices

1. **Store Keys as Secrets**: Never commit API keys
2. **Limit Permissions**: Use minimum required permissions
3. **Review SARIF**: Check security findings before merge
4. **Policy Enforcement**: Enable strict policy mode
5. **Docker Isolation**: All builds run in sandboxed containers

## Support

- **Documentation**: [Specular Docs](https://github.com/felixgeelhaar/specular)
- **Issues**: [GitHub Issues](https://github.com/felixgeelhaar/specular/issues)
- **Examples**: [Example Projects](https://github.com/felixgeelhaar/specular/tree/main/examples)

## License

MIT License - see [LICENSE](https://github.com/felixgeelhaar/specular/blob/main/LICENSE)
