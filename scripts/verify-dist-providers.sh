#!/usr/bin/env bash
# verify-dist-providers.sh - Verify provider CLI wrappers exist in dist/providers/
#
# This script checks that the provider binaries were built and exist
# before goreleaser packages them into archives.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DIST_DIR="$ROOT_DIR/build/providers"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Minimum required providers
REQUIRED_PROVIDERS=(
    "ollama-provider"
)

# Optional providers (warn if missing, don't fail)
OPTIONAL_PROVIDERS=(
    "claude-provider"
    "claude-code-provider"
    "codex-provider"
    "codex-cli-provider"
    "gemini-provider"
    "gemini-cli-provider"
)

log_info "Verifying provider CLI wrappers in $DIST_DIR"

# Check if dist/providers directory exists
if [[ ! -d "$DIST_DIR" ]]; then
    log_error "Directory $DIST_DIR does not exist"
    log_error "Run scripts/package-cli-wrappers.sh first"
    exit 1
fi

# Determine binary extension
EXT=""
if [[ "$(go env GOOS)" == "windows" ]]; then
    EXT=".exe"
fi

# Check required providers
MISSING_REQUIRED=0
for provider in "${REQUIRED_PROVIDERS[@]}"; do
    BINARY="$DIST_DIR/${provider}${EXT}"
    if [[ -f "$BINARY" ]]; then
        log_info "  ✓ Found required: $provider"
    else
        log_error "  ✗ Missing required: $provider"
        ((MISSING_REQUIRED++))
    fi
done

# Check optional providers
MISSING_OPTIONAL=0
for provider in "${OPTIONAL_PROVIDERS[@]}"; do
    BINARY="$DIST_DIR/${provider}${EXT}"
    if [[ -f "$BINARY" ]]; then
        log_info "  ✓ Found optional: $provider"
    else
        log_warn "  ⚠ Missing optional: $provider"
        ((MISSING_OPTIONAL++))
    fi
done

echo ""

# Summary
TOTAL_FOUND=$(find "$DIST_DIR" -type f -name "*-provider*" 2>/dev/null | wc -l | tr -d ' ')
log_info "Found $TOTAL_FOUND provider binaries"

if [[ $MISSING_REQUIRED -gt 0 ]]; then
    log_error "Missing $MISSING_REQUIRED required provider(s)"
    exit 1
fi

if [[ $MISSING_OPTIONAL -gt 0 ]]; then
    log_warn "Missing $MISSING_OPTIONAL optional provider(s)"
fi

log_info "Provider verification passed"
