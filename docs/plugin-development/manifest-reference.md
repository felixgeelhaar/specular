# Plugin Manifest Reference

The plugin manifest (`plugin.yaml`) defines your plugin's metadata, configuration, and capabilities.

## Complete Schema

```yaml
# Required fields
name: my-plugin                    # Unique identifier
version: "1.0.0"                   # Semantic version
type: hook                         # Plugin type
description: "Short description"   # Plugin description
author: "Author Name"              # Author name or organization
entrypoint: "./my-plugin"          # Path to executable

# Optional fields
license: "MIT"                     # SPDX license identifier
homepage: "https://example.com"    # Plugin homepage URL
repository: "github.com/user/repo" # Source repository URL
min_specular_version: "1.6.0"      # Minimum required Specular version

# Optional: Search keywords for registry
keywords:
  - notification
  - webhook
  - automation

# Optional: Plugin capabilities
capabilities:
  - hook
  - async

# Optional: Plugin dependencies
dependencies:
  - name: other-plugin
    version: ">=1.0.0"
    optional: false

# Optional: Configuration schema
config:
  - name: api_key
    type: string
    description: "API key for authentication"
    required: true
    secret: true

  - name: timeout
    type: int
    description: "Request timeout in seconds"
    required: false
    default: 30
```

## Field Reference

### Required Fields

#### `name`

**Type:** `string`
**Required:** Yes

Unique identifier for the plugin. Must be lowercase, alphanumeric, with hyphens allowed.

```yaml
name: my-awesome-plugin
```

**Naming conventions:**
- Use lowercase letters, numbers, and hyphens
- Start with a letter
- Be descriptive but concise
- Avoid generic names like "plugin" or "tool"

#### `version`

**Type:** `string`
**Required:** Yes

Semantic version following [semver](https://semver.org/) specification.

```yaml
version: "1.2.3"
version: "0.1.0-beta"
version: "2.0.0+build.123"
```

**Version format:**
- `MAJOR.MINOR.PATCH` required
- Prerelease: `1.0.0-alpha`, `1.0.0-beta.2`
- Build metadata: `1.0.0+20231201`

#### `type`

**Type:** `string` (enum)
**Required:** Yes

The plugin type determines how Specular interacts with your plugin.

| Type | Description | Actions |
|------|-------------|---------|
| `provider` | AI provider integration | `generate`, `health` |
| `validator` | Policy validator | `validate`, `health` |
| `formatter` | Output formatter | `format`, `health` |
| `hook` | Event hook | `pre`, `post`, `execute`, `health` |
| `notifier` | Notification sender | `notify`, `health` |

```yaml
type: hook
```

#### `description`

**Type:** `string`
**Required:** Yes

Short description of what the plugin does. Displayed in `plugin list` and `plugin info`.

```yaml
description: "Sends notifications to Slack channels"
```

**Best practices:**
- Keep under 100 characters
- Start with a verb (Sends, Validates, Formats)
- Be specific about functionality

#### `author`

**Type:** `string`
**Required:** Yes

Author name or organization.

```yaml
author: "Jane Doe"
author: "Acme Corporation"
```

#### `entrypoint`

**Type:** `string`
**Required:** Yes

Path to the executable relative to the plugin directory.

```yaml
# Binary
entrypoint: "./my-plugin"

# Script with interpreter
entrypoint: "python3 main.py"
entrypoint: "node index.js"
entrypoint: "bash entrypoint.sh"
```

### Optional Fields

#### `license`

**Type:** `string`
**Required:** No

SPDX license identifier.

```yaml
license: "MIT"
license: "Apache-2.0"
license: "GPL-3.0"
```

See [SPDX License List](https://spdx.org/licenses/) for valid identifiers.

#### `homepage`

**Type:** `string`
**Required:** No

URL to the plugin's homepage or documentation.

```yaml
homepage: "https://github.com/user/plugin#readme"
```

#### `repository`

**Type:** `string`
**Required:** No

Source repository URL. Used for updates and registry indexing.

```yaml
repository: "github.com/user/plugin"
repository: "gitlab.com/user/plugin"
```

#### `min_specular_version`

**Type:** `string`
**Required:** No

Minimum Specular version required to run this plugin.

```yaml
min_specular_version: "1.6.0"
```

#### `keywords`

**Type:** `[]string`
**Required:** No

Tags for registry search indexing.

```yaml
keywords:
  - slack
  - notifications
  - webhook
  - ci-cd
```

**Best practices:**
- Include relevant technology names
- Add use case keywords
- Limit to 5-10 keywords

#### `capabilities`

**Type:** `[]string`
**Required:** No

Declares specific capabilities the plugin provides.

```yaml
capabilities:
  - hook
  - async
  - streaming
```

**Common capabilities:**
- `async` - Supports asynchronous operations
- `streaming` - Supports streaming responses
- `batch` - Supports batch processing

#### `dependencies`

**Type:** `[]PluginDependency`
**Required:** No

Other plugins this plugin depends on.

```yaml
dependencies:
  - name: common-utils
    version: ">=1.0.0"
    optional: false

  - name: optional-feature
    version: "^2.0.0"
    optional: true
```

**Version constraint operators:**

| Operator | Meaning | Example |
|----------|---------|---------|
| `=` | Exact version | `=1.2.3` |
| `>` | Greater than | `>1.0.0` |
| `>=` | Greater or equal | `>=1.0.0` |
| `<` | Less than | `<2.0.0` |
| `<=` | Less or equal | `<=2.0.0` |
| `^` | Compatible (same major) | `^1.2.3` → `>=1.2.3 <2.0.0` |
| `~` | Approximately (same minor) | `~1.2.3` → `>=1.2.3 <1.3.0` |

#### `config`

**Type:** `[]ConfigField`
**Required:** No

Declares configuration fields the plugin accepts.

```yaml
config:
  - name: webhook_url
    type: string
    description: "Slack webhook URL"
    required: true
    secret: true

  - name: channel
    type: string
    description: "Default channel"
    required: false
    default: "#general"

  - name: timeout
    type: int
    description: "Request timeout (seconds)"
    required: false
    default: 30

  - name: enabled
    type: bool
    description: "Enable notifications"
    required: false
    default: true
```

### ConfigField Schema

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Configuration key (required) |
| `type` | `string` | Value type: `string`, `int`, `bool`, `float`, `[]string` (required) |
| `description` | `string` | Field description |
| `required` | `bool` | Whether field must be set |
| `default` | `any` | Default value if not specified |
| `secret` | `bool` | Treat as sensitive (masked in logs) |

## Examples

### Provider Plugin

```yaml
name: custom-llm
version: "1.0.0"
type: provider
description: "Custom LLM provider integration"
author: "ML Team"
license: "Apache-2.0"
entrypoint: "./custom-llm"

min_specular_version: "1.6.0"
repository: "github.com/company/custom-llm"

keywords:
  - llm
  - ai
  - provider

capabilities:
  - streaming
  - async

config:
  - name: api_endpoint
    type: string
    description: "LLM API endpoint"
    required: true

  - name: api_key
    type: string
    description: "API authentication key"
    required: true
    secret: true

  - name: model
    type: string
    description: "Model identifier"
    required: false
    default: "gpt-4"

  - name: temperature
    type: float
    description: "Sampling temperature"
    required: false
    default: 0.7
```

### Validator Plugin

```yaml
name: security-validator
version: "2.0.0"
type: validator
description: "Validates content against security policies"
author: "Security Team"
license: "MIT"
entrypoint: "python3 main.py"

min_specular_version: "1.6.0"

keywords:
  - security
  - validation
  - policy
  - compliance

config:
  - name: policy_file
    type: string
    description: "Path to policy definition"
    required: false
    default: ".security-policy.yaml"

  - name: severity_threshold
    type: string
    description: "Minimum severity to report"
    required: false
    default: "warning"

  - name: fail_on_error
    type: bool
    description: "Exit with error on validation failure"
    required: false
    default: true
```

### Hook Plugin with Dependencies

```yaml
name: git-integration
version: "1.0.0"
type: hook
description: "Git integration hooks for CI/CD"
author: "DevOps Team"
license: "MIT"
entrypoint: "./git-integration"

dependencies:
  - name: notification-base
    version: ">=1.0.0"
    optional: false

  - name: metrics-collector
    version: "^2.0.0"
    optional: true

config:
  - name: repo_path
    type: string
    description: "Git repository path"
    required: false
    default: "."

  - name: branch_pattern
    type: string
    description: "Branch pattern to watch"
    required: false
    default: "main|develop"
```

## Validation

Specular validates manifests when loading plugins. Common validation errors:

| Error | Cause | Fix |
|-------|-------|-----|
| `missing required field: name` | Name not specified | Add `name` field |
| `invalid version format` | Non-semver version | Use `MAJOR.MINOR.PATCH` |
| `unknown plugin type` | Invalid type value | Use valid type enum |
| `entrypoint not found` | Missing executable | Build plugin or fix path |
| `invalid config type` | Unknown type in config | Use valid type: string, int, bool, float |

## Best Practices

1. **Version properly** - Follow semver, bump major on breaking changes
2. **Describe clearly** - Help users understand what your plugin does
3. **Document config** - Provide clear descriptions and sensible defaults
4. **Declare dependencies** - List required plugins with version constraints
5. **Use keywords** - Help users find your plugin in the registry
6. **Set minimum version** - Ensure compatibility with required Specular features
