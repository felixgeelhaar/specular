#!/bin/bash
# git-hook - A Specular hook plugin for Git integration
# Type: hook
#
# This plugin provides Git integration for Specular operations:
# - Logs operations to file
# - Optionally auto-commits changes
# - Validates branch patterns

set -euo pipefail

VERSION="1.0.0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Read entire input
INPUT=$(cat)

# Parse fields from JSON
ACTION=$(echo "$INPUT" | jq -r '.action // "unknown"')

# Logging functions (to stderr, never stdout)
log_debug() {
  [[ "${DEBUG:-false}" == "true" ]] && echo "[DEBUG] $*" >&2
}

log_info() {
  echo "[INFO] $*" >&2
}

log_error() {
  echo "[ERROR] $*" >&2
}

# JSON response helpers
respond_success() {
  local result="$1"
  echo "{\"success\": true, \"result\": $result}"
}

respond_error() {
  local message="$1"
  local escaped
  escaped=$(echo "$message" | jq -Rs '.')
  echo "{\"success\": false, \"error\": $escaped}"
}

# Get config value with default
get_config() {
  local key="$1"
  local default="${2:-}"
  echo "$INPUT" | jq -r ".config.$key // \"$default\""
}

# Get data value
get_data() {
  local key="$1"
  local default="${2:-}"
  echo "$INPUT" | jq -r ".data.$key // \"$default\""
}

# Check if we're in a git repository
is_git_repo() {
  git rev-parse --git-dir >/dev/null 2>&1
}

# Get current branch name
get_branch() {
  git rev-parse --abbrev-ref HEAD 2>/dev/null || echo ""
}

# Check if branch matches pattern
branch_matches() {
  local pattern="$1"
  local branch
  branch=$(get_branch)

  if [[ -z "$pattern" ]]; then
    return 0  # No pattern = all branches allowed
  fi

  if echo "$branch" | grep -qE "$pattern"; then
    return 0
  fi

  return 1
}

# Log operation to file
log_operation() {
  local event="$1"
  local log_file
  log_file=$(get_config "log_file" "")

  if [[ -n "$log_file" ]]; then
    local timestamp branch
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    branch=$(get_branch)

    echo "[$timestamp] Event: $event, Branch: $branch" >> "$log_file"
  fi
}

# Auto-commit changes
auto_commit() {
  local message="$1"
  local auto_commit prefix

  auto_commit=$(get_config "auto_commit" "false")

  if [[ "$auto_commit" != "true" ]]; then
    return 0
  fi

  prefix=$(get_config "commit_prefix" "[specular]")

  # Check if there are changes to commit
  if git diff --quiet && git diff --cached --quiet; then
    log_debug "No changes to commit"
    return 0
  fi

  log_info "Auto-committing changes..."
  git add -A
  git commit -m "$prefix $message" 2>&1 || {
    log_error "Failed to commit"
    return 1
  }

  return 0
}

# Handle health check
handle_health() {
  respond_success "{\"status\": \"healthy\", \"version\": \"$VERSION\", \"name\": \"git-hook\"}"
}

# Handle pre-event hook
handle_pre() {
  local event
  event=$(get_data "event" "unknown")

  log_info "Pre-event: $event"
  log_operation "pre:$event"

  # Check branch pattern
  local branch_pattern
  branch_pattern=$(get_config "branch_pattern" "")

  if [[ -n "$branch_pattern" ]] && is_git_repo; then
    if ! branch_matches "$branch_pattern"; then
      local branch
      branch=$(get_branch)
      respond_error "Operation blocked: branch '$branch' does not match pattern '$branch_pattern'"
      exit 0
    fi
  fi

  respond_success '{"executed": true, "blocked": false}'
}

# Handle post-event hook
handle_post() {
  local event
  event=$(get_data "event" "unknown")

  log_info "Post-event: $event"
  log_operation "post:$event"

  # Auto-commit if configured
  if is_git_repo; then
    auto_commit "After $event" || true
  fi

  respond_success '{"executed": true}'
}

# Handle execute hook (generic)
handle_execute() {
  local event
  event=$(get_data "event" "unknown")

  log_info "Execute: $event"
  log_operation "$event"

  # Gather git info if in repo
  local git_info='{}'
  if is_git_repo; then
    local branch commit
    branch=$(get_branch)
    commit=$(git rev-parse --short HEAD 2>/dev/null || echo "")

    git_info=$(jq -n \
      --arg branch "$branch" \
      --arg commit "$commit" \
      '{branch: $branch, commit: $commit}')
  fi

  respond_success "{\"executed\": true, \"git\": $git_info}"
}

# Main handler
case "$ACTION" in
  "health")
    handle_health
    ;;
  "pre")
    handle_pre
    ;;
  "post")
    handle_post
    ;;
  "execute")
    handle_execute
    ;;
  *)
    respond_error "unknown action: $ACTION"
    exit 1
    ;;
esac
