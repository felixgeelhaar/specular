# Specular GitHub Action

This GitHub Action integrates Specular AI Governance into your CI/CD pipeline for spec validation, planning, building, and drift detection.

## Features

- 🔒 **Spec Validation**: Lock and validate product specifications
- 📋 **Plan Generation**: Generate executable plans from specs
- 🏗️ **Policy-Enforced Build**: Build with organizational guardrails
- 🔍 **Drift Detection**: Detect spec, plan, and code drift with SARIF reporting
- 📊 **GitHub Security Integration**: Upload drift findings to GitHub Security tab
- 🚀 **Multi-Platform Support**: Works on Linux, macOS, and Windows runners

## Inputs

### Required

| Name | Description | Default |
|------|-------------|---------|
| `command` | Specular command to run (`spec`, `plan`, `build`, `eval`, `doctor`) | **Required** |

### Optional

| Name | Description | Default |
|------|-------------|---------|
| `version` | Specular version to install | `latest` |
| `spec-file` | Path to spec.yaml file | `.specular/spec.yaml` |
| `lock-file` | Path to spec.lock.json file | `.specular/spec.lock.json` |
| `plan-file` | Path to plan.json file | `plan.json` |
| `policy-file` | Path to policy.yaml file | `.specular/policy.yaml` |
| `router-file` | Path to router.yaml file | `.specular/router.yaml` |
| `fail-on-drift` | Fail the build if drift is detected | `true` |
| `anthropic-api-key` | Anthropic API key for Claude models | - |
| `openai-api-key` | OpenAI API key for GPT models | - |
| `gemini-api-key` | Google Gemini API key | - |
| `enable-cache` | Enable Docker image caching | `true` |
| `cache-dir` | Directory for Docker image cache | `.specular/cache` |
| `cache-max-age` | Maximum cache age (e.g., 168h for 7 days) | `168h` |
| `additional-args` | Additional arguments to pass to specular | - |

## Outputs

| Name | Description |
|------|-------------|
| `result` | Command execution result (`success`/`failure`) |
| `exit-code` | Exit code from specular command |
| `drift-detected` | Whether drift was detected (`true`/`false`) |

## Usage Examples

### Basic Drift Detection

```yaml
name: Specular Drift Detection

on:
  pull_request:
  push:
    branches: [ main ]

jobs:
  drift-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Detect Drift
        uses: ./.github/actions/specular
        with:
          command: eval
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
          fail-on-drift: 'true'
```

### Complete CI Pipeline with Docker Caching

```yaml
name: Specular CI Pipeline

on: [pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Lock Spec
        uses: ./.github/actions/specular
        with:
          command: spec

      - name: Generate Plan
        uses: ./.github/actions/specular
        with:
          command: plan

      - name: Evaluate
        uses: ./.github/actions/specular
        with:
          command: eval
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
          openai-api-key: ${{ secrets.OPENAI_API_KEY }}

      - name: Build with Docker Cache
        uses: ./.github/actions/specular
        with:
          command: build
          enable-cache: 'true'  # Cache Docker images (enabled by default)
          cache-max-age: '168h'  # Keep cache for 7 days
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

### Custom Version and Arguments

```yaml
- name: Run with Custom Version
  uses: ./.github/actions/specular
  with:
    version: 'v1.4.0'
    command: eval
    additional-args: '--verbose --json'
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

### System Health Check

```yaml
- name: Run Diagnostics
  uses: ./.github/actions/specular
  with:
    command: doctor
    additional-args: '--format json'
```

## Setting Up API Keys

Store your API keys as GitHub Secrets:

1. Go to your repository Settings
2. Navigate to Secrets and variables → Actions
3. Click "New repository secret"
4. Add secrets for your AI providers:
   - `ANTHROPIC_API_KEY` for Claude models
   - `OPENAI_API_KEY` for GPT models
   - `GEMINI_API_KEY` for Gemini models

## SARIF Drift Reporting

When running the `eval` command, drift findings are automatically uploaded to GitHub's Security tab:

```yaml
- name: Detect Drift
  uses: ./.github/actions/specular
  with:
    command: eval
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

The action will:
1. Run drift detection
2. Generate `.specular/drift.sarif` report
3. Upload findings to GitHub Security
4. Annotate PR with drift warnings (if any)
5. Fail the build if `fail-on-drift: true` and drift is detected

## Exit Codes

The action uses standardized exit codes:

- `0` - Success
- `1` - General error
- `2` - Validation error
- `3` - Policy violation
- `4` - Drift detected
- `5` - Build failure
- `6` - Test failure

## Docker Image Caching

The action automatically caches Docker images between runs to significantly improve build performance. Caching is enabled by default and uses GitHub Actions cache.

### How It Works

1. **First Run**: Downloads and caches Docker images specified in your policy
2. **Subsequent Runs**: Restores images from cache (if valid)
3. **Cache Key**: Based on OS and configuration files (`*.yaml`, `*.json`)
4. **Automatic Cleanup**: GitHub automatically removes old caches (7 days default retention)

### Performance Impact

- **Without Cache**: 30-60 seconds to pull Docker images
- **With Cache**: < 5 seconds to restore from cache
- **Cache Size**: Typically 100-500MB per image

### Cache Configuration

```yaml
- name: Run Build with Custom Cache
  uses: ./.github/actions/specular
  with:
    command: build
    enable-cache: 'true'
    cache-dir: '.specular/cache'
    cache-max-age: '336h'  # 14 days
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

### Disable Caching

To disable caching (not recommended for CI):

```yaml
- name: Run Build Without Cache
  uses: ./.github/actions/specular
  with:
    command: build
    enable-cache: 'false'
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

### Cache Management

GitHub Actions automatically manages cache lifecycle:
- Maximum 10GB total cache per repository
- Least recently used caches are evicted first
- Caches older than 7 days are automatically deleted

### Manual Cache Control

To force cache rebuild:
1. Go to repository Settings → Actions → Caches
2. Delete the `specular-docker-*` cache
3. Next workflow run will rebuild the cache

## Troubleshooting

### Action Fails to Install

Ensure you're using a supported runner:
- `ubuntu-latest` (recommended)
- `ubuntu-22.04`
- `macos-latest`
- `windows-latest`

### API Key Issues

Verify:
1. Secrets are correctly named (case-sensitive)
2. Secrets are set at repository level
3. Workflows have access to secrets

### Drift Not Detected

Check:
1. `.specular/spec.lock.json` exists
2. Policy file is valid YAML
3. Spec and code are in sync

### Slow Build Times

Enable Docker caching:
1. Verify `enable-cache: 'true'` in action inputs
2. Check GitHub Actions cache is available (not disabled)
3. Ensure cache size is under repository limit (10GB)
4. Review cache hit rate in workflow logs

## Advanced Configuration

### Custom Policy File

```yaml
- name: Run with Custom Policy
  uses: ./.github/actions/specular
  with:
    command: build
    policy-file: .specular/custom-policy.yaml
```

### Multiple Providers

```yaml
- name: Multi-Provider Evaluation
  uses: ./.github/actions/specular
  with:
    command: eval
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
    openai-api-key: ${{ secrets.OPENAI_API_KEY }}
    gemini-api-key: ${{ secrets.GEMINI_API_KEY }}
```

### Skip Drift Failure

```yaml
- name: Detect Drift (Warning Only)
  uses: ./.github/actions/specular
  with:
    command: eval
    fail-on-drift: 'false'
    anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

## Support

For issues and feature requests, please visit:
- [GitHub Issues](https://github.com/felixgeelhaar/specular/issues)
- [Documentation](https://github.com/felixgeelhaar/specular/docs)

## License

MIT License - See LICENSE file for details
