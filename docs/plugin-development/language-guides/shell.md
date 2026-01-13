# Shell Plugin Development

This guide covers shell script (Bash) best practices for Specular plugin development.

## Why Shell?

- **Simplicity:** Quick scripts for basic tasks
- **No Dependencies:** Works on any Unix system
- **System Integration:** Direct access to system tools
- **Prototyping:** Fast to develop and iterate

## Quick Start

```bash
specular plugin create my-plugin --type hook --lang shell
cd my-plugin
echo '{"action":"health"}' | ./entrypoint.sh
```

## Project Structure

```
my-plugin/
├── plugin.yaml       # Plugin manifest
├── entrypoint.sh     # Main entry point
├── lib/              # Helper scripts
│   ├── handlers.sh
│   └── utils.sh
└── tests/
    └── test.sh
```

## Code Templates

### Basic Plugin Structure

```bash
#!/bin/bash
# My Plugin - A Specular hook plugin
# Type: hook

set -euo pipefail

VERSION="1.0.0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Read entire input
INPUT=$(cat)

# Parse action from JSON
ACTION=$(echo "$INPUT" | jq -r '.action // "unknown"')

# Handle actions
case "$ACTION" in
  "health")
    echo "{\"success\": true, \"result\": {\"status\": \"healthy\", \"version\": \"$VERSION\"}}"
    ;;

  "execute"|"pre"|"post")
    # Extract data
    EVENT=$(echo "$INPUT" | jq -r '.data.event // ""')
    CONFIG=$(echo "$INPUT" | jq -c '.config // {}')

    # Your logic here
    result=$(handle_hook "$EVENT" "$CONFIG")

    echo "{\"success\": true, \"result\": $result}"
    ;;

  *)
    echo "{\"success\": false, \"error\": \"unknown action: $ACTION\"}"
    exit 1
    ;;
esac
```

### Modular Structure

```bash
#!/bin/bash
# entrypoint.sh - Main entry point

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load libraries
source "$SCRIPT_DIR/lib/utils.sh"
source "$SCRIPT_DIR/lib/handlers.sh"

VERSION="1.0.0"

main() {
  local input action

  input=$(cat)
  action=$(json_get "$input" '.action' 'unknown')

  case "$action" in
    "health")
      handle_health
      ;;
    "execute"|"pre"|"post")
      handle_hook "$input"
      ;;
    *)
      respond_error "unknown action: $action"
      exit 1
      ;;
  esac
}

main "$@"
```

### lib/utils.sh

```bash
#!/bin/bash
# lib/utils.sh - Utility functions

# Extract value from JSON
# Usage: json_get "$json" '.path.to.value' 'default'
json_get() {
  local json="$1"
  local path="$2"
  local default="${3:-}"

  local result
  result=$(echo "$json" | jq -r "$path // empty" 2>/dev/null)

  if [[ -z "$result" ]]; then
    echo "$default"
  else
    echo "$result"
  fi
}

# Create JSON success response
# Usage: respond_success '{"key": "value"}'
respond_success() {
  local result="$1"
  echo "{\"success\": true, \"result\": $result}"
}

# Create JSON error response
# Usage: respond_error "error message"
respond_error() {
  local message="$1"
  local escaped
  escaped=$(echo "$message" | jq -Rs '.')
  echo "{\"success\": false, \"error\": $escaped}"
}

# Log to stderr (never stdout)
# Usage: log_info "message"
log_debug() {
  [[ "${DEBUG:-false}" == "true" ]] && echo "[DEBUG] $*" >&2
}

log_info() {
  echo "[INFO] $*" >&2
}

log_error() {
  echo "[ERROR] $*" >&2
}

# Check required tools
check_dependencies() {
  local deps=("$@")
  local missing=()

  for dep in "${deps[@]}"; do
    if ! command -v "$dep" &>/dev/null; then
      missing+=("$dep")
    fi
  done

  if [[ ${#missing[@]} -gt 0 ]]; then
    respond_error "missing dependencies: ${missing[*]}"
    exit 1
  fi
}
```

### lib/handlers.sh

```bash
#!/bin/bash
# lib/handlers.sh - Action handlers

handle_health() {
  respond_success "{\"status\": \"healthy\", \"version\": \"$VERSION\"}"
}

handle_hook() {
  local input="$1"

  local event config
  event=$(json_get "$input" '.data.event' '')
  config=$(echo "$input" | jq -c '.config // {}')

  log_info "Processing event: $event"

  # Your hook logic here
  case "$event" in
    "generate.start")
      handle_generate_start "$input" "$config"
      ;;
    "generate.end")
      handle_generate_end "$input" "$config"
      ;;
    *)
      log_debug "Unhandled event: $event"
      respond_success '{"executed": true, "handled": false}'
      ;;
  esac
}

handle_generate_start() {
  local input="$1"
  local config="$2"

  # Pre-processing logic
  local model
  model=$(json_get "$input" '.data.payload.model' 'unknown')
  log_info "Generation starting with model: $model"

  respond_success '{"executed": true, "modified": false}'
}

handle_generate_end() {
  local input="$1"
  local config="$2"

  # Post-processing logic
  local tokens
  tokens=$(json_get "$input" '.data.payload.usage.total_tokens' '0')
  log_info "Generation completed: $tokens tokens"

  respond_success "{\"executed\": true, \"tokens\": $tokens}"
}
```

### Configuration Handling

```bash
#!/bin/bash
# Configuration handling

# Parse and validate configuration
parse_config() {
  local config="$1"

  # Required fields
  local api_key
  api_key=$(json_get "$config" '.api_key' '')

  if [[ -z "$api_key" ]]; then
    respond_error "config: api_key is required"
    exit 1
  fi

  # Optional with defaults
  local timeout endpoint retries
  timeout=$(json_get "$config" '.timeout' '30')
  endpoint=$(json_get "$config" '.endpoint' 'https://api.example.com')
  retries=$(json_get "$config" '.retries' '3')

  # Validate
  if ! [[ "$timeout" =~ ^[0-9]+$ ]]; then
    respond_error "config: timeout must be a number"
    exit 1
  fi

  # Export for use
  export CONFIG_API_KEY="$api_key"
  export CONFIG_TIMEOUT="$timeout"
  export CONFIG_ENDPOINT="$endpoint"
  export CONFIG_RETRIES="$retries"
}
```

### HTTP Requests with curl

```bash
#!/bin/bash
# HTTP client functions

# Make HTTP request with retry
# Usage: http_request "POST" "/path" '{"data": "value"}'
http_request() {
  local method="$1"
  local path="$2"
  local body="${3:-}"

  local url="${CONFIG_ENDPOINT}${path}"
  local attempt=0
  local max_attempts="${CONFIG_RETRIES:-3}"
  local timeout="${CONFIG_TIMEOUT:-30}"

  while [[ $attempt -lt $max_attempts ]]; do
    local response status_code

    if [[ -n "$body" ]]; then
      response=$(curl -s -w "\n%{http_code}" \
        --max-time "$timeout" \
        -X "$method" \
        -H "Authorization: Bearer $CONFIG_API_KEY" \
        -H "Content-Type: application/json" \
        -d "$body" \
        "$url" 2>/dev/null)
    else
      response=$(curl -s -w "\n%{http_code}" \
        --max-time "$timeout" \
        -X "$method" \
        -H "Authorization: Bearer $CONFIG_API_KEY" \
        "$url" 2>/dev/null)
    fi

    status_code=$(echo "$response" | tail -n1)
    response=$(echo "$response" | sed '$d')

    if [[ "$status_code" =~ ^2 ]]; then
      echo "$response"
      return 0
    fi

    log_debug "Request failed (attempt $((attempt + 1))): HTTP $status_code"
    ((attempt++))

    if [[ $attempt -lt $max_attempts ]]; then
      local delay=$((2 ** attempt))
      sleep "$delay"
    fi
  done

  log_error "Request failed after $max_attempts attempts"
  return 1
}

# Send notification
send_notification() {
  local message="$1"
  local channel="${2:-#general}"

  local payload
  payload=$(jq -n \
    --arg text "$message" \
    --arg channel "$channel" \
    '{text: $text, channel: $channel}')

  http_request "POST" "/webhook" "$payload"
}
```

### Error Handling

```bash
#!/bin/bash
# Error handling patterns

set -euo pipefail

# Trap for cleanup
cleanup() {
  local exit_code=$?
  # Cleanup logic here
  rm -f "$TEMP_FILE" 2>/dev/null || true
  exit $exit_code
}
trap cleanup EXIT

# Error handler
error_handler() {
  local line_no=$1
  log_error "Error on line $line_no"
  respond_error "internal error on line $line_no"
  exit 1
}
trap 'error_handler ${LINENO}' ERR

# Safe execution with error capture
safe_exec() {
  local cmd="$1"
  local error_msg="${2:-command failed}"

  local output exit_code

  output=$($cmd 2>&1) && exit_code=0 || exit_code=$?

  if [[ $exit_code -ne 0 ]]; then
    log_error "$error_msg: $output"
    respond_error "$error_msg"
    exit 1
  fi

  echo "$output"
}

# Validate required environment
require_env() {
  local var_name="$1"
  if [[ -z "${!var_name:-}" ]]; then
    respond_error "environment variable $var_name is required"
    exit 1
  fi
}
```

## Testing

### Basic Test Script

```bash
#!/bin/bash
# tests/test.sh - Plugin tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN="$SCRIPT_DIR/../entrypoint.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

tests_passed=0
tests_failed=0

# Test helper
run_test() {
  local name="$1"
  local input="$2"
  local expected="$3"

  local output
  output=$(echo "$input" | "$PLUGIN")

  if echo "$output" | jq -e "$expected" >/dev/null 2>&1; then
    echo -e "${GREEN}PASS${NC}: $name"
    ((tests_passed++))
  else
    echo -e "${RED}FAIL${NC}: $name"
    echo "  Input: $input"
    echo "  Output: $output"
    echo "  Expected: $expected"
    ((tests_failed++))
  fi
}

# Health check test
run_test "health check" \
  '{"action":"health"}' \
  '.success == true and .result.status == "healthy"'

# Execute hook test
run_test "execute hook" \
  '{"action":"execute","data":{"event":"test"}}' \
  '.success == true and .result.executed == true'

# Unknown action test
run_test "unknown action" \
  '{"action":"invalid"}' \
  '.success == false and (.error | contains("unknown action"))'

# Configuration test
run_test "with config" \
  '{"action":"execute","data":{"event":"test"},"config":{"debug":true}}' \
  '.success == true'

# Summary
echo ""
echo "Tests passed: $tests_passed"
echo "Tests failed: $tests_failed"

if [[ $tests_failed -gt 0 ]]; then
  exit 1
fi
```

### Advanced Test Framework

```bash
#!/bin/bash
# tests/framework.sh - Test framework

declare -a TEST_FUNCTIONS=()

# Register test
test() {
  TEST_FUNCTIONS+=("$1")
}

# Assertions
assert_equals() {
  local expected="$1"
  local actual="$2"
  local message="${3:-}"

  if [[ "$expected" == "$actual" ]]; then
    return 0
  else
    echo "  Assertion failed: $message"
    echo "  Expected: $expected"
    echo "  Actual: $actual"
    return 1
  fi
}

assert_json() {
  local json="$1"
  local query="$2"
  local message="${3:-}"

  if echo "$json" | jq -e "$query" >/dev/null 2>&1; then
    return 0
  else
    echo "  Assertion failed: $message"
    echo "  JSON: $json"
    echo "  Query: $query"
    return 1
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local message="${3:-}"

  if [[ "$haystack" == *"$needle"* ]]; then
    return 0
  else
    echo "  Assertion failed: $message"
    echo "  Expected to contain: $needle"
    echo "  Actual: $haystack"
    return 1
  fi
}

# Run all tests
run_tests() {
  local passed=0
  local failed=0

  for test_fn in "${TEST_FUNCTIONS[@]}"; do
    echo -n "Testing $test_fn... "
    if $test_fn; then
      echo "PASS"
      ((passed++))
    else
      echo "FAIL"
      ((failed++))
    fi
  done

  echo ""
  echo "Results: $passed passed, $failed failed"
  [[ $failed -eq 0 ]]
}

# Usage in test file:
# source tests/framework.sh
#
# test test_health_check() {
#   local output=$(echo '{"action":"health"}' | ./entrypoint.sh)
#   assert_json "$output" '.success == true' "should succeed"
# }
#
# run_tests
```

## Best Practices

### 1. Always Use Strict Mode

```bash
#!/bin/bash
set -euo pipefail

# -e: Exit on error
# -u: Error on undefined variables
# -o pipefail: Fail on pipe errors
```

### 2. Quote Variables

```bash
# Good
local value="$input"
echo "$value"

# Bad - can break on spaces/special chars
local value=$input
echo $value
```

### 3. Use Local Variables

```bash
# Good
my_function() {
  local result
  result=$(some_command)
  echo "$result"
}

# Bad - pollutes global namespace
my_function() {
  result=$(some_command)
  echo "$result"
}
```

### 4. Check Command Existence

```bash
if ! command -v jq &>/dev/null; then
  echo "jq is required but not installed"
  exit 1
fi
```

### 5. Handle JSON Safely

```bash
# Escape strings for JSON
escape_json() {
  local text="$1"
  echo "$text" | jq -Rs '.'
}

# Build JSON safely
build_json() {
  local message="$1"
  local status="$2"

  jq -n \
    --arg msg "$message" \
    --argjson status "$status" \
    '{message: $msg, success: $status}'
}
```

### 6. Use Temporary Files Safely

```bash
# Create temp file
TEMP_FILE=$(mktemp)
trap 'rm -f "$TEMP_FILE"' EXIT

# Use temp directory
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT
```

## Required Dependencies

Shell plugins require these tools:

| Tool | Purpose | Install |
|------|---------|---------|
| `jq` | JSON parsing | `brew install jq` / `apt install jq` |
| `curl` | HTTP requests | Usually pre-installed |
| `bash` | Shell (v4+) | Usually pre-installed |

Check in your plugin:

```bash
check_dependencies jq curl
```

## Limitations

Shell plugins have some limitations compared to compiled languages:

1. **Performance:** Slower for complex processing
2. **JSON Handling:** Requires external tools (jq)
3. **Error Handling:** Less robust than try/catch
4. **Portability:** May vary between systems
5. **Complexity:** Harder to maintain large scripts

## When to Use Shell

**Good use cases:**
- Simple hooks that run system commands
- Wrappers around existing CLI tools
- Quick prototypes
- CI/CD integrations

**Consider other languages for:**
- Complex data processing
- Network-heavy operations
- Large plugins with many features
- Production-critical plugins
