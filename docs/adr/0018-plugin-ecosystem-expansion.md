# ADR-0018: Plugin Ecosystem Expansion

## Status

Accepted

## Date

2024-01-15

## Context

The Specular CLI's plugin system was introduced in v1.5.0 with basic functionality for extending CLI capabilities through external executables. While functional, the initial implementation lacked several features necessary for a mature plugin ecosystem:

1. **No Version Management:** Plugins had no semantic versioning support, making updates and compatibility checks impossible
2. **Limited Installation Sources:** Only local directory installation was supported
3. **No Registry:** No centralized discovery mechanism for finding plugins
4. **No Dependency Management:** Plugins couldn't declare or resolve dependencies on other plugins
5. **No Update Mechanism:** No way to update installed plugins to newer versions
6. **No Installation Tracking:** No lockfile to track installed plugins and their versions

Users requested:
- Ability to install plugins from GitHub repositories
- Version-specific installation (e.g., `plugin@v1.2.0`)
- Plugin search and discovery
- Automatic dependency resolution
- Plugin updates

## Decision

We will implement a comprehensive plugin ecosystem expansion with the following components:

### 1. Semantic Versioning

Implement full semver 2.0.0 support including:
- Version parsing (`1.2.3`, `1.2.3-beta`, `1.2.3+build`)
- Version comparison for sorting and conflict detection
- Constraint operators (`>=`, `<=`, `>`, `<`, `=`, `^`, `~`)

**Implementation:** `internal/plugin/version.go`

### 2. Plugin Source Parsing

Support multiple installation sources:
- Local directories: `./my-plugin`, `/absolute/path`
- GitHub repositories: `github.com/user/repo@v1.0.0`
- GitHub with branch/tag: `github.com/user/repo@main`
- GitHub with subpath: `github.com/user/repo/plugins/foo@v1.0.0`
- Registry: `registry:plugin-name@1.0.0`

**Implementation:** `internal/plugin/source.go`

### 3. GitHub-Hosted Registry

Use a GitHub repository as the plugin registry:
- Central `index.json` with all plugin metadata
- JSON format for easy parsing and human readability
- Cached locally with configurable TTL
- Default: `https://raw.githubusercontent.com/specular/specular-plugins/main/index.json`
- Custom registries supported via `--registry` flag

**Rationale for GitHub-based registry:**
- No infrastructure to maintain
- Transparent review process via PRs
- Version control for index changes
- Community can contribute
- Cheap/free hosting

**Alternative considered:** Dedicated registry service
- Rejected due to operational overhead and hosting costs

**Implementation:** `internal/plugin/registry.go`

### 4. Dependency Resolution

Implement dependency resolution with:
- Topological sorting for correct installation order
- Circular dependency detection
- Version conflict detection
- Optional dependency support
- Dependency tree visualization

**Algorithm:**
1. Parse manifest dependencies
2. Build dependency graph
3. Detect circular dependencies
4. Check for version conflicts
5. Topologically sort for installation order
6. Install dependencies before dependent plugins

**Implementation:** `internal/plugin/resolver.go`

### 5. Installation Tracking (Lockfile)

Track installed plugins in `~/.specular/plugins.lock.json`:
```json
{
  "version": "1",
  "plugins": {
    "slack-notifier": {
      "name": "slack-notifier",
      "version": "1.2.0",
      "source": "github.com/specular/slack-notifier@v1.2.0",
      "checksum": "sha256:abc123...",
      "installed_at": "2024-01-15T10:00:00Z",
      "dependencies": []
    }
  },
  "updated": "2024-01-15T10:00:00Z"
}
```

**Implementation:** `internal/plugin/lockfile.go`

### 6. Enhanced Manifest Fields

Add to `plugin.yaml`:
```yaml
# Source repository for updates
repository: "github.com/user/repo"

# Search keywords for registry
keywords:
  - slack
  - notifications

# Plugin dependencies
dependencies:
  - name: common-utils
    version: ">=1.0.0"
    optional: false
```

**Implementation:** Modified `internal/plugin/types.go`

### 7. CLI Commands

New commands:
- `specular plugin update [name]` - Update plugins
- `specular plugin search <query>` - Search registry
- `specular plugin registry-info <name>` - Show registry entry

Enhanced commands:
- `specular plugin install <source>` - Support GitHub URLs, registry references

**Implementation:** Modified `internal/cmd/plugin.go`

## Consequences

### Positive

1. **Discoverability:** Users can search for plugins via registry
2. **Reliability:** Version constraints ensure compatibility
3. **Maintainability:** Update mechanism keeps plugins current
4. **Ecosystem Growth:** Lower barrier for plugin authors
5. **Reproducibility:** Lockfile enables consistent environments
6. **Safety:** Dependency resolution prevents conflicts

### Negative

1. **Complexity:** More code to maintain
2. **Network Dependency:** Registry requires internet access
3. **Storage:** Cache and lockfile use disk space
4. **Migration:** Existing plugins need manifest updates

### Risks

1. **Registry Availability:** GitHub outages affect plugin installation
   - *Mitigation:* Local cache with configurable TTL
2. **Version Conflicts:** Complex dependency graphs may have conflicts
   - *Mitigation:* Clear error messages with resolution hints
3. **Breaking Changes:** Manifest schema changes could break existing plugins
   - *Mitigation:* Versioned schema with backward compatibility

## Implementation Plan

### Phase 1: Foundation (Completed)
- `version.go` - Semver parsing and constraints
- `lockfile.go` - Installation tracking
- `types.go` - Add Dependencies, Repository, Keywords fields

### Phase 2: Enhanced Installation (Completed)
- `source.go` - Source parsing for all formats
- `manager.go` - InstallWithOptions, Update, UpdateAll methods
- `plugin update` CLI command

### Phase 3: Registry (Completed)
- `registry.go` - Registry client with caching
- `plugin search` CLI command
- `plugin registry-info` CLI command

### Phase 4: Dependencies (Completed)
- `resolver.go` - Dependency resolution
- Circular detection
- Conflict detection
- Topological sorting

### Phase 5: Testing (Completed)
- Integration tests
- E2E tests
- Unit tests for all new code

### Phase 6: Documentation (Completed)
- Plugin development guide
- Manifest reference
- Protocol reference
- Language guides
- Publishing guide

### Phase 7: Examples (Pending)
- Example plugins for each type
- Example plugins for each language

## Testing Strategy

1. **Unit Tests:** All version, source, registry, resolver functions
2. **Integration Tests:** Plugin workflow (create, build, install, health)
3. **E2E Tests:** CLI command behavior
4. **Mock HTTP:** Registry tests with mock server

## Security Considerations

1. **Registry Content:** Only serve from known GitHub repo
2. **Checksum Verification:** Future enhancement for integrity checks
3. **Source Validation:** Reject malformed URLs
4. **No Code Execution:** Registry index is data-only

## Migration Path

1. Existing plugins continue to work without changes
2. New fields are optional with sensible defaults
3. Lockfile created on first install/update
4. Registry usage is opt-in (direct install still works)

## References

- [Semantic Versioning 2.0.0](https://semver.org/)
- [npm Registry](https://docs.npmjs.com/about-the-public-npm-registry)
- [Go Modules](https://go.dev/ref/mod)
- [Cargo Registry](https://doc.rust-lang.org/cargo/reference/registries.html)

## Changelog

- 2024-01-15: Initial draft
- 2024-01-15: Accepted after implementation of Phases 1-6
