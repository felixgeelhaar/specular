# Git Hook Plugin

A Specular hook plugin for Git integration that tracks operations and optionally auto-commits changes.

## Features

- **Operation Logging:** Log all Specular operations to a file
- **Auto-Commit:** Automatically commit changes after generation
- **Branch Protection:** Restrict operations to specific branches
- **Git Info:** Include Git context in hook responses

## Installation

```bash
# From local directory
specular plugin install ./git-hook

# From registry (when published)
specular plugin install registry:git-hook
```

## Configuration

Add to your `.specular.yaml`:

```yaml
plugins:
  git-hook:
    auto_commit: false
    commit_prefix: "[specular]"
    log_file: "/var/log/specular-operations.log"
    branch_pattern: "^(main|develop|feature/.*)$"
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `auto_commit` | bool | false | Auto-commit after operations |
| `commit_prefix` | string | [specular] | Prefix for commit messages |
| `log_file` | string | - | Path to operation log file |
| `branch_pattern` | string | - | Regex pattern for allowed branches |

## Events

The plugin responds to these hook events:

| Event | Action |
|-------|--------|
| `generate.start` | Log pre-event, check branch |
| `generate.end` | Log post-event, auto-commit |
| `validate.start` | Log pre-event, check branch |
| `validate.end` | Log post-event |

## Usage

### Pre-Event Hook

```json
{
  "action": "pre",
  "data": {
    "event": "generate.start"
  },
  "config": {
    "branch_pattern": "^main$"
  }
}
```

### Response (Allowed)

```json
{
  "success": true,
  "result": {
    "executed": true,
    "blocked": false
  }
}
```

### Response (Blocked)

```json
{
  "success": false,
  "error": "Operation blocked: branch 'feature/test' does not match pattern '^main$'"
}
```

### Post-Event Hook

```json
{
  "action": "post",
  "data": {
    "event": "generate.end"
  },
  "config": {
    "auto_commit": true
  }
}
```

## Development

### Requirements

- Bash 4.0+
- jq (JSON processor)
- git

### Testing

```bash
# Health check
echo '{"action":"health"}' | ./entrypoint.sh

# Pre-event
echo '{"action":"pre","data":{"event":"test"}}' | ./entrypoint.sh

# Post-event with auto-commit
echo '{"action":"post","data":{"event":"test"},"config":{"auto_commit":true}}' | ./entrypoint.sh
```

## Log Format

Operations are logged in the following format:

```
[2024-01-15T10:30:00Z] Event: pre:generate.start, Branch: main
[2024-01-15T10:30:05Z] Event: post:generate.end, Branch: main
```

## License

MIT License - See LICENSE file for details.
