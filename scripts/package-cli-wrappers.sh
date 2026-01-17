#!/usr/bin/env bash
# package-cli-wrappers.sh - Build and package provider CLI wrappers for distribution
#
# This script builds all provider binaries and copies them to dist/providers/
# for inclusion in release archives.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist/providers"

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

# Provider directories that contain main.go files
PROVIDERS=(
    "claude"
    "claude-code"
    "codex"
    "codex-cli"
    "gemini"
    "gemini-cli"
    "ollama"
)

# Determine OS and architecture
GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"

# Binary extension for Windows
EXT=""
if [[ "$GOOS" == "windows" ]]; then
    EXT=".exe"
fi

log_info "Building provider CLI wrappers for ${GOOS}/${GOARCH}"
log_info "Output directory: $DIST_DIR"

# Create dist directory
mkdir -p "$DIST_DIR"

# Build each provider
BUILT=0
FAILED=0

for provider in "${PROVIDERS[@]}"; do
    PROVIDER_DIR="$ROOT_DIR/providers/$provider"

    if [[ ! -f "$PROVIDER_DIR/main.go" ]]; then
        log_warn "Skipping $provider - no main.go found"
        continue
    fi

    BINARY_NAME="${provider}-provider${EXT}"
    OUTPUT_PATH="$DIST_DIR/$BINARY_NAME"

    log_info "Building $provider -> $BINARY_NAME"

    if CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
        -trimpath \
        -ldflags="-s -w" \
        -o "$OUTPUT_PATH" \
        "$PROVIDER_DIR"; then
        log_info "  ✓ Built $BINARY_NAME"
        BUILT=$((BUILT + 1))
    else
        log_error "  ✗ Failed to build $provider"
        FAILED=$((FAILED + 1))
    fi
done

echo ""
log_info "Build summary: $BUILT succeeded, $FAILED failed"

# List built providers
if [[ $BUILT -gt 0 ]]; then
    echo ""
    log_info "Built providers:"
    ls -la "$DIST_DIR"
fi

# Exit with error if any builds failed
if [[ $FAILED -gt 0 ]]; then
    log_error "Some provider builds failed"
    exit 1
fi

log_info "Provider CLI wrappers packaged successfully"
