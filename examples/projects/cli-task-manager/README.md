# Task Manager CLI

A sample CLI application demonstrating best practices for building command-line tools with Go and Cobra. This project showcases patterns for configuration management, storage, and shell completions.

## Features

- **Task Management**: Add, list, complete, and delete tasks
- **Configuration**: YAML configuration with environment variable overrides
- **Storage**: JSON file storage with atomic writes
- **Shell Completions**: Bash, Zsh, Fish, and PowerShell

## Building with Specular

This project includes a Specular spec file demonstrating how to define your CLI application:

```bash
# Generate implementation from spec
specular build run --spec spec.yaml

# Run in autonomous mode
specular auto --spec spec.yaml --profile development
```

## Installation

```bash
# Build from source
go build -o task-cli ./cmd/task-cli

# Or with Go install
go install ./cmd/task-cli
```

## Usage

```bash
# Add a task
task-cli add "Complete project documentation"
task-cli add "Review pull requests" --priority high

# List tasks
task-cli list
task-cli list --all       # Include completed
task-cli list --priority high

# Complete a task
task-cli complete 1

# Delete a task
task-cli delete 2

# Configuration
task-cli config show
task-cli config set storage.path ~/.tasks.json
task-cli config set default_priority medium

# Shell completions
task-cli completion bash > /etc/bash_completion.d/task-cli
task-cli completion zsh > "${fpath[1]}/_task-cli"
```

## Configuration

Configuration is loaded from (in order of precedence):
1. Environment variables (prefix: `TASK_`)
2. Config file (`~/.task-cli/config.yaml`)
3. Default values

```yaml
# ~/.task-cli/config.yaml
storage:
  path: ~/.task-cli/tasks.json

defaults:
  priority: medium

display:
  colors: true
  date_format: "2006-01-02"
```

## Project Structure

```
cli-task-manager/
├── cmd/
│   └── task-cli/
│       └── main.go           # Entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── storage/
│   │   └── storage.go        # JSON file storage
│   └── task/
│       └── task.go           # Task domain model
├── spec.yaml                  # Specular specification
├── go.mod
└── README.md
```

## Patterns Demonstrated

### 1. Cobra Command Structure
- Root command with global flags
- Subcommands with specific flags
- Persistent pre-run for initialization

### 2. Viper Configuration
- YAML configuration file
- Environment variable bindings
- Default values

### 3. Storage Layer
- Interface-based design
- Atomic file writes
- Error handling

### 4. Domain Model
- Task struct with validation
- Status enumeration
- Time handling

## Specular Spec Example

See `spec.yaml` for a complete specification that can be used with Specular to:
- Generate boilerplate code
- Validate against policies
- Track implementation progress

## License

MIT License - see parent project for details.
