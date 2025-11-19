#!/bin/bash
# Specular Release Verification Script
# This script verifies a release by:
# 1. Downloading release artifacts
# 2. Verifying checksums
# 3. Running smoke tests on binaries
# 4. Testing installation methods

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored message
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[PASS]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[FAIL]${NC} $1" >&2; }

# Counters for test results
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Record test result
pass() {
    success "$1"
    ((TESTS_PASSED++))
}

fail() {
    error "$1"
    ((TESTS_FAILED++))
}

skip() {
    warn "[SKIP] $1"
    ((TESTS_SKIPPED++))
}

# Print usage
usage() {
    cat << EOF
Usage: $(basename "$0") <version> [options]

Verify a Specular release.

Arguments:
    version     The version to verify (e.g., 1.6.0 or v1.6.0)

Options:
    -h, --help          Show this help message
    --skip-download     Use existing artifacts in ./dist
    --skip-homebrew     Skip Homebrew installation test
    --skip-docker       Skip Docker installation test
    --github-token      GitHub token for API access (or set GITHUB_TOKEN)

Examples:
    $(basename "$0") 1.6.0
    $(basename "$0") v1.6.0 --skip-docker
EOF
    exit "${1:-0}"
}

# Parse arguments
VERSION=""
SKIP_DOWNLOAD=false
SKIP_HOMEBREW=false
SKIP_DOCKER=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage 0
            ;;
        --skip-download)
            SKIP_DOWNLOAD=true
            shift
            ;;
        --skip-homebrew)
            SKIP_HOMEBREW=true
            shift
            ;;
        --skip-docker)
            SKIP_DOCKER=true
            shift
            ;;
        --github-token)
            GITHUB_TOKEN="$2"
            shift 2
            ;;
        -*)
            error "Unknown option: $1"
            usage 1
            ;;
        *)
            if [[ -z "$VERSION" ]]; then
                VERSION="$1"
            else
                error "Multiple versions specified: $VERSION and $1"
                usage 1
            fi
            shift
            ;;
    esac
done

# Validate version
if [[ -z "$VERSION" ]]; then
    error "Version is required"
    usage 1
fi

# Normalize version (add v prefix if missing)
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v$VERSION"
fi

info "Verifying release $VERSION"
echo ""

# Create temp directory for verification
WORK_DIR=$(mktemp -d)
trap "rm -rf $WORK_DIR" EXIT

cd "$WORK_DIR"

# GitHub release URL
GITHUB_REPO="felixgeelhaar/specular"
RELEASE_URL="https://github.com/$GITHUB_REPO/releases/tag/$VERSION"
API_URL="https://api.github.com/repos/$GITHUB_REPO/releases/tags/$VERSION"

# ============================================================================
# SECTION 1: Download Release Artifacts
# ============================================================================
info "Section 1: Release Artifacts"
echo "────────────────────────────────────────"

if [[ "$SKIP_DOWNLOAD" == "false" ]]; then
    info "Downloading release artifacts..."

    # Get release info from GitHub API
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        RELEASE_INFO=$(curl -s -H "Authorization: token $GITHUB_TOKEN" "$API_URL")
    else
        RELEASE_INFO=$(curl -s "$API_URL")
    fi

    if echo "$RELEASE_INFO" | grep -q '"message": "Not Found"'; then
        fail "Release $VERSION not found on GitHub"
        exit 1
    fi

    # Download checksums
    CHECKSUMS_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION/checksums.txt"
    if curl -sLO "$CHECKSUMS_URL" 2>/dev/null && [[ -f "checksums.txt" ]]; then
        pass "Downloaded checksums.txt"
    else
        fail "Failed to download checksums.txt"
    fi

    # Detect current platform
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
    esac

    # Download platform-specific archive
    if [[ "$OS" == "darwin" || "$OS" == "linux" ]]; then
        ARCHIVE="specular_${VERSION#v}_${OS}_${ARCH}.tar.gz"
    else
        ARCHIVE="specular_${VERSION#v}_${OS}_${ARCH}.zip"
    fi

    ARCHIVE_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION/$ARCHIVE"
    if curl -sLO "$ARCHIVE_URL" 2>/dev/null && [[ -f "$ARCHIVE" ]]; then
        pass "Downloaded $ARCHIVE"
    else
        fail "Failed to download $ARCHIVE"
    fi
else
    info "Skipping download, using existing artifacts"
fi

echo ""

# ============================================================================
# SECTION 2: Checksum Verification
# ============================================================================
info "Section 2: Checksum Verification"
echo "────────────────────────────────────────"

if [[ -f "checksums.txt" ]]; then
    # Verify checksums using sha256sum or shasum
    if command -v sha256sum &> /dev/null; then
        CHECKSUM_CMD="sha256sum"
    elif command -v shasum &> /dev/null; then
        CHECKSUM_CMD="shasum -a 256"
    else
        skip "No checksum tool available (sha256sum or shasum)"
    fi

    if [[ -n "${CHECKSUM_CMD:-}" ]]; then
        # Filter checksums file for downloaded files
        for file in *.tar.gz *.zip; do
            if [[ -f "$file" ]]; then
                EXPECTED=$(grep "$file" checksums.txt | cut -d' ' -f1)
                if [[ -n "$EXPECTED" ]]; then
                    ACTUAL=$($CHECKSUM_CMD "$file" | cut -d' ' -f1)
                    if [[ "$EXPECTED" == "$ACTUAL" ]]; then
                        pass "Checksum verified: $file"
                    else
                        fail "Checksum mismatch: $file"
                        echo "  Expected: $EXPECTED"
                        echo "  Actual:   $ACTUAL"
                    fi
                else
                    warn "No checksum found for $file"
                fi
            fi
        done
    fi
else
    skip "checksums.txt not available"
fi

echo ""

# ============================================================================
# SECTION 3: Binary Smoke Tests
# ============================================================================
info "Section 3: Binary Smoke Tests"
echo "────────────────────────────────────────"

# Extract binary
BINARY_PATH=""
for archive in *.tar.gz; do
    if [[ -f "$archive" ]]; then
        tar -xzf "$archive"
        if [[ -f "specular" ]]; then
            BINARY_PATH="./specular"
            chmod +x "$BINARY_PATH"
            pass "Extracted binary from $archive"
        fi
    fi
done

for archive in *.zip; do
    if [[ -f "$archive" ]]; then
        unzip -q "$archive"
        if [[ -f "specular.exe" ]]; then
            BINARY_PATH="./specular.exe"
            pass "Extracted binary from $archive"
        fi
    fi
done

if [[ -n "$BINARY_PATH" && -f "$BINARY_PATH" ]]; then
    # Test 1: Version command
    if $BINARY_PATH version 2>&1 | grep -q "${VERSION#v}\|$VERSION"; then
        pass "Version command returns correct version"
    else
        fail "Version command returns incorrect version"
        $BINARY_PATH version
    fi

    # Test 2: Help command
    if $BINARY_PATH --help 2>&1 | grep -q "specular\|spec-first\|policy"; then
        pass "Help command works"
    else
        fail "Help command failed"
    fi

    # Test 3: Provider list
    if $BINARY_PATH provider list 2>&1 | grep -q "anthropic\|openai\|ollama"; then
        pass "Provider list command works"
    else
        fail "Provider list command failed"
    fi

    # Test 4: Config default
    if $BINARY_PATH config --help 2>&1; then
        pass "Config command works"
    else
        fail "Config command failed"
    fi

    # Test 5: Policy commands
    if $BINARY_PATH policy --help 2>&1; then
        pass "Policy command works"
    else
        fail "Policy command failed"
    fi

    # Test 6: Auth commands
    if $BINARY_PATH auth --help 2>&1; then
        pass "Auth command works"
    else
        fail "Auth command failed"
    fi

    # Test 7: Platform commands
    if $BINARY_PATH platform --help 2>&1; then
        pass "Platform command works"
    else
        fail "Platform command failed"
    fi
else
    skip "Binary not available for smoke tests"
fi

echo ""

# ============================================================================
# SECTION 4: Installation Method Tests
# ============================================================================
info "Section 4: Installation Method Tests"
echo "────────────────────────────────────────"

# Test Homebrew installation
if [[ "$SKIP_HOMEBREW" == "false" ]]; then
    if command -v brew &> /dev/null; then
        info "Testing Homebrew installation..."

        # Check if formula exists in tap
        if brew info felixgeelhaar/tap/specular &> /dev/null 2>&1; then
            pass "Homebrew formula available"

            # Note: We don't actually install to avoid conflicts
            # but we can check the formula was updated
            FORMULA_VERSION=$(brew info felixgeelhaar/tap/specular 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "")
            if [[ "$FORMULA_VERSION" == "${VERSION#v}" ]]; then
                pass "Homebrew formula updated to $VERSION"
            else
                warn "Homebrew formula may not be updated yet (found: $FORMULA_VERSION)"
            fi
        else
            warn "Homebrew tap formula not found"
        fi
    else
        skip "Homebrew not installed"
    fi
else
    skip "Homebrew test (--skip-homebrew)"
fi

# Test Docker image
if [[ "$SKIP_DOCKER" == "false" ]]; then
    if command -v docker &> /dev/null; then
        info "Testing Docker image..."

        DOCKER_IMAGE="ghcr.io/felixgeelhaar/specular:$VERSION"

        # Try to pull the image
        if docker pull "$DOCKER_IMAGE" &> /dev/null; then
            pass "Docker image available: $DOCKER_IMAGE"

            # Run version check in container
            if docker run --rm "$DOCKER_IMAGE" version 2>&1 | grep -q "${VERSION#v}\|$VERSION"; then
                pass "Docker image version correct"
            else
                fail "Docker image version incorrect"
            fi

            # Clean up
            docker rmi "$DOCKER_IMAGE" &> /dev/null || true
        else
            warn "Docker image not available yet"
        fi
    else
        skip "Docker not installed"
    fi
else
    skip "Docker test (--skip-docker)"
fi

echo ""

# ============================================================================
# SUMMARY
# ============================================================================
echo "════════════════════════════════════════"
info "Verification Summary"
echo "════════════════════════════════════════"
echo ""
echo "  Passed:  $TESTS_PASSED"
echo "  Failed:  $TESTS_FAILED"
echo "  Skipped: $TESTS_SKIPPED"
echo ""

if [[ $TESTS_FAILED -eq 0 ]]; then
    success "All verification tests passed!"
    exit 0
else
    error "$TESTS_FAILED test(s) failed"
    exit 1
fi
