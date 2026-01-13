# Publishing Plugins to the Registry

This guide explains how to publish your plugin to the official Specular plugin registry.

## Overview

The Specular plugin registry is a curated collection of plugins hosted on GitHub. Publishing your plugin makes it discoverable via `specular plugin search` and installable via `specular plugin install registry:your-plugin`.

## Prerequisites

Before publishing:

1. **Stable Plugin** - Plugin should be tested and production-ready
2. **Public Repository** - Source code must be publicly accessible
3. **Complete Manifest** - All required fields must be present
4. **Documentation** - README with installation and usage instructions
5. **License** - Open source license (MIT, Apache-2.0, etc.)

## Registry Format

The registry is a GitHub repository containing an `index.json` file:

```json
{
  "version": "1.0.0",
  "updated": "2024-01-15T00:00:00Z",
  "plugins": {
    "slack-notifier": {
      "name": "slack-notifier",
      "description": "Send notifications to Slack channels",
      "author": "Specular Team",
      "type": "notifier",
      "license": "MIT",
      "repository": "github.com/specular/slack-notifier",
      "homepage": "https://github.com/specular/slack-notifier",
      "keywords": ["slack", "notifications", "webhook"],
      "versions": {
        "1.0.0": {
          "released": "2024-01-01T00:00:00Z",
          "min_specular_version": "1.6.0",
          "checksum": "sha256:abc123...",
          "dependencies": []
        },
        "1.1.0": {
          "released": "2024-01-15T00:00:00Z",
          "min_specular_version": "1.6.0",
          "checksum": "sha256:def456...",
          "dependencies": []
        }
      },
      "latest": "1.1.0",
      "downloads": 1250,
      "verified": true
    }
  }
}
```

## Submission Process

### Step 1: Prepare Your Plugin

Ensure your plugin meets these requirements:

**Manifest (`plugin.yaml`):**
```yaml
name: my-awesome-plugin
version: "1.0.0"
type: notifier
description: "Clear, concise description"
author: "Your Name"
license: "MIT"
entrypoint: "./my-awesome-plugin"
repository: "github.com/yourname/my-awesome-plugin"
homepage: "https://github.com/yourname/my-awesome-plugin"

keywords:
  - relevant
  - searchable
  - terms

min_specular_version: "1.6.0"
```

**Repository structure:**
```
my-awesome-plugin/
├── plugin.yaml          # Required
├── README.md            # Required
├── LICENSE              # Required
├── CHANGELOG.md         # Recommended
├── main.go              # Your code
├── go.mod
└── .github/
    └── workflows/
        └── release.yml  # Automated releases
```

### Step 2: Create a GitHub Release

Tag your repository with a semantic version:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Create a GitHub release with:
- Version tag (e.g., `v1.0.0`)
- Release notes
- Pre-built binaries (recommended)

### Step 3: Submit to Registry

1. **Fork** the [specular-plugins](https://github.com/specular/specular-plugins) repository

2. **Add your plugin** to `index.json`:

```json
{
  "my-awesome-plugin": {
    "name": "my-awesome-plugin",
    "description": "Clear, concise description",
    "author": "Your Name",
    "type": "notifier",
    "license": "MIT",
    "repository": "github.com/yourname/my-awesome-plugin",
    "homepage": "https://github.com/yourname/my-awesome-plugin",
    "keywords": ["relevant", "searchable", "terms"],
    "versions": {
      "1.0.0": {
        "released": "2024-01-15T00:00:00Z",
        "min_specular_version": "1.6.0"
      }
    },
    "latest": "1.0.0"
  }
}
```

3. **Create a pull request** with:
   - Plugin entry in `index.json`
   - Brief description of the plugin
   - Link to documentation

### Step 4: Review Process

The review process checks:

| Criterion | Description |
|-----------|-------------|
| **Functionality** | Plugin works as described |
| **Quality** | Code follows best practices |
| **Security** | No malicious code or vulnerabilities |
| **Documentation** | Clear installation and usage |
| **Compatibility** | Works with stated Specular version |

Review typically takes 2-5 business days.

## Automated Releases

Use GitHub Actions to automate releases:

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build
        run: |
          GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/my-plugin
          GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/my-plugin
          GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/my-plugin
          GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/my-plugin.exe

      - name: Package
        run: |
          cd dist
          for dir in */; do
            tar -czvf "${dir%/}.tar.gz" -C "$dir" .
          done

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: dist/*.tar.gz
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Updating Published Plugins

To release a new version:

1. **Update version** in `plugin.yaml`
2. **Update CHANGELOG.md**
3. **Create new release** on GitHub
4. **Update registry** with new version entry

```bash
# Update plugin
vim plugin.yaml  # Change version to 1.1.0
vim CHANGELOG.md # Add release notes

# Tag and release
git add -A
git commit -m "Release v1.1.0"
git tag v1.1.0
git push origin main v1.1.0

# Submit PR to update registry
```

## Verification Badge

Plugins that pass review receive a "verified" badge:

```json
{
  "my-plugin": {
    "verified": true,
    ...
  }
}
```

Verified plugins are:
- Reviewed by the Specular team
- Tested for security issues
- Confirmed to work as documented

## Best Practices for Publishing

### Naming

- Use lowercase with hyphens: `my-plugin` not `MyPlugin`
- Be descriptive: `slack-notifier` not `notifier`
- Avoid generic names: `security-validator` not `validator`

### Description

- Keep it under 100 characters
- Start with a verb: "Sends", "Validates", "Formats"
- Be specific about functionality

### Keywords

- Include technology names: `slack`, `discord`, `webhook`
- Include use cases: `ci-cd`, `security`, `compliance`
- 5-10 keywords maximum

### Versioning

- Follow semantic versioning strictly
- Document breaking changes in CHANGELOG
- Support multiple minor versions when possible

### Documentation

Include:
- Installation instructions
- Configuration reference
- Usage examples
- Troubleshooting guide

## Removing a Plugin

To remove your plugin from the registry:

1. Open an issue in the specular-plugins repository
2. Provide reason for removal
3. Plugin will be marked as deprecated
4. After 30 days, entry will be removed

Existing installations will continue to work but updates won't be available.

## Questions?

- **Issue tracker:** [github.com/specular/specular-plugins/issues](https://github.com/specular/specular-plugins/issues)
- **Discussions:** [github.com/specular/specular-plugins/discussions](https://github.com/specular/specular-plugins/discussions)
- **Email:** plugins@specular.dev
