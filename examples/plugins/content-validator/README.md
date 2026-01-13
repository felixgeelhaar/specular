# Content Validator Plugin

A Specular validator plugin that validates content against configurable rules.

## Features

- **Length Validation:** Enforce minimum and maximum content length
- **Required Keywords:** Ensure specific keywords are present
- **Forbidden Patterns:** Block content matching regex patterns
- **Configurable Severity:** Filter results by severity level

## Installation

```bash
# From local directory
specular plugin install ./content-validator

# From registry (when published)
specular plugin install registry:content-validator
```

## Configuration

Add to your `.specular.yaml`:

```yaml
plugins:
  content-validator:
    max_length: 5000
    min_length: 100
    required_keywords: "summary,conclusion"
    forbidden_patterns: "TODO,FIXME,password"
    severity_threshold: warning
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `max_length` | int | 10000 | Maximum content length |
| `min_length` | int | 0 | Minimum content length |
| `required_keywords` | string | - | Comma-separated required keywords |
| `forbidden_patterns` | string | - | Comma-separated regex patterns |
| `severity_threshold` | string | warning | Minimum severity to report |

## Usage

The plugin responds to the `validate` action:

```json
{
  "action": "validate",
  "data": {
    "content": "Your content to validate..."
  },
  "config": {
    "max_length": 5000
  }
}
```

### Response

```json
{
  "success": true,
  "result": {
    "valid": true,
    "messages": []
  }
}
```

### Validation Issues

```json
{
  "success": true,
  "result": {
    "valid": false,
    "messages": [
      {
        "severity": "error",
        "message": "Content too long: 6000 characters, maximum is 5000",
        "rule": "max_length"
      },
      {
        "severity": "warning",
        "message": "Required keyword missing: 'summary'",
        "rule": "required_keyword"
      }
    ]
  }
}
```

## Development

### Requirements

- Python 3.8+
- No external dependencies (stdlib only)

### Testing

```bash
# Health check
echo '{"action":"health"}' | python3 main.py

# Validate content
echo '{"action":"validate","data":{"content":"Test content"},"config":{"min_length":100}}' | python3 main.py
```

### Running Tests

```bash
python3 -m pytest tests/
```

## License

MIT License - See LICENSE file for details.
