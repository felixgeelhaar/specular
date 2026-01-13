# Markdown Formatter Plugin

A Specular formatter plugin that outputs content as Markdown with customizable formatting options.

## Features

- **Table of Contents:** Auto-generate TOC from headings
- **Metadata Headers:** YAML front matter support
- **Configurable Headings:** Set starting heading level
- **Code Blocks:** Format code with configurable fence style
- **Structured Input:** Handle various input formats

## Installation

```bash
# From local directory
specular plugin install ./markdown-formatter

# From registry (when published)
specular plugin install registry:markdown-formatter
```

## Configuration

Add to your `.specular.yaml`:

```yaml
plugins:
  markdown-formatter:
    include_toc: true
    include_metadata: true
    heading_level: 1
    code_fence_style: backtick
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `include_toc` | bool | false | Generate table of contents |
| `include_metadata` | bool | true | Include YAML front matter |
| `heading_level` | int | 1 | Starting heading level (1-6) |
| `code_fence_style` | string | backtick | Code fence style (backtick/tilde) |

## Usage

### Basic Format Request

```json
{
  "action": "format",
  "data": {
    "title": "My Document",
    "content": "This is the content of my document."
  }
}
```

### Response

```json
{
  "success": true,
  "result": {
    "output": "# My Document\n\nThis is the content of my document.\n",
    "format": "markdown",
    "metadata": {
      "length": 52,
      "lines": 4
    }
  }
}
```

### With Metadata

```json
{
  "action": "format",
  "data": {
    "title": "API Documentation",
    "description": "Complete API reference",
    "metadata": {
      "author": "Team",
      "date": "2024-01-15",
      "version": "1.0.0"
    },
    "content": "..."
  },
  "config": {
    "include_metadata": true
  }
}
```

### Structured Content

```json
{
  "action": "format",
  "data": {
    "title": "Getting Started",
    "content": [
      {
        "heading": "Installation",
        "text": "Run the following command:"
      },
      {
        "code": "npm install my-package",
        "language": "bash"
      },
      {
        "heading": "Configuration",
        "text": "Create a config file..."
      }
    ]
  }
}
```

## Output Examples

### Basic Document

```markdown
# My Document

This is the content.
```

### With TOC and Metadata

```markdown
---
author: "Team"
date: "2024-01-15"
---

# Document Title

## Table of Contents

- [Introduction](#introduction)
- [Getting Started](#getting-started)

## Introduction

Content here...
```

## Development

### Requirements

- Node.js 18+
- No external dependencies

### Testing

```bash
# Health check
echo '{"action":"health"}' | node index.js

# Format content
echo '{"action":"format","data":{"title":"Test","content":"Hello"}}' | node index.js

# With configuration
echo '{"action":"format","data":{"title":"Test","content":"Hello"},"config":{"include_toc":true}}' | node index.js
```

### Running Tests

```bash
npm test
```

## License

MIT License - See LICENSE file for details.
