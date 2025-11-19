#!/bin/bash
# Specular Pre-Release Preparation Script
# This script prepares a release by:
# 1. Running go mod tidy
# 2. Generating shell completions
# 3. Validating CHANGELOG.md
# 4. Committing generated files
# 5. Creating annotated git tag

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored message
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

# Print usage
usage() {
    cat << EOF
Usage: $(basename "$0") <version> [options]

Prepare a release for Specular CLI.

Arguments:
    version     The version to release (e.g., 1.6.0 or v1.6.0)

Options:
    -h, --help      Show this help message
    -d, --dry-run   Show what would be done without making changes
    -f, --force     Skip confirmation prompts
    -s, --skip-tests Skip running tests before release
    --no-tag        Don't create a git tag
    --no-commit     Don't commit generated files

Examples:
    $(basename "$0") 1.6.0
    $(basename "$0") v1.6.0 --dry-run
    $(basename "$0") 1.6.0 --force --skip-tests
EOF
    exit "${1:-0}"
}

# Parse arguments
VERSION=""
DRY_RUN=false
FORCE=false
SKIP_TESTS=false
NO_TAG=false
NO_COMMIT=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage 0
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        -s|--skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --no-tag)
            NO_TAG=true
            shift
            ;;
        --no-commit)
            NO_COMMIT=true
            shift
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

# Validate version format
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
    error "Invalid version format: $VERSION"
    error "Expected format: v1.2.3 or v1.2.3-beta.1"
    exit 1
fi

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

info "Preparing release $VERSION"
echo ""

# Check we're on a clean git state
if [[ -n "$(git status --porcelain)" ]]; then
    if [[ "$DRY_RUN" == "false" ]]; then
        warn "Working directory is not clean"
        git status --short
        echo ""
        if [[ "$FORCE" == "false" ]]; then
            read -p "Continue anyway? [y/N] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                error "Aborted"
                exit 1
            fi
        fi
    fi
fi

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    error "Tag $VERSION already exists"
    exit 1
fi

# Step 1: Run go mod tidy
info "Step 1/6: Running go mod tidy..."
if [[ "$DRY_RUN" == "true" ]]; then
    echo "  [DRY-RUN] Would run: go mod tidy"
else
    go mod tidy
    success "go mod tidy completed"
fi

# Step 2: Run tests
if [[ "$SKIP_TESTS" == "false" ]]; then
    info "Step 2/6: Running tests..."
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "  [DRY-RUN] Would run: go test ./..."
    else
        if ! go test ./... 2>&1; then
            error "Tests failed. Fix tests before releasing."
            exit 1
        fi
        success "Tests passed"
    fi
else
    warn "Step 2/6: Skipping tests (--skip-tests)"
fi

# Step 3: Build binary
info "Step 3/6: Building binary..."
if [[ "$DRY_RUN" == "true" ]]; then
    echo "  [DRY-RUN] Would run: go build -o specular ./cmd/specular"
else
    go build -o specular ./cmd/specular
    success "Binary built"
fi

# Step 4: Generate shell completions
info "Step 4/6: Generating shell completions..."
mkdir -p completions
if [[ "$DRY_RUN" == "true" ]]; then
    echo "  [DRY-RUN] Would generate: completions/specular.bash"
    echo "  [DRY-RUN] Would generate: completions/_specular (zsh)"
    echo "  [DRY-RUN] Would generate: completions/specular.fish"
else
    ./specular completion bash > completions/specular.bash
    ./specular completion zsh > completions/_specular
    ./specular completion fish > completions/specular.fish
    success "Shell completions generated"
fi

# Step 5: Validate CHANGELOG.md
info "Step 5/6: Validating CHANGELOG.md..."
CHANGELOG_FILE="CHANGELOG.md"
VERSION_WITHOUT_V="${VERSION#v}"

if [[ ! -f "$CHANGELOG_FILE" ]]; then
    error "CHANGELOG.md not found"
    exit 1
fi

# Check if version is in CHANGELOG
if ! grep -q "## \[$VERSION_WITHOUT_V\]" "$CHANGELOG_FILE" && \
   ! grep -q "## \[v$VERSION_WITHOUT_V\]" "$CHANGELOG_FILE" && \
   ! grep -q "## $VERSION_WITHOUT_V" "$CHANGELOG_FILE" && \
   ! grep -q "## v$VERSION_WITHOUT_V" "$CHANGELOG_FILE"; then
    error "Version $VERSION not found in CHANGELOG.md"
    echo ""
    echo "Please add a section like:"
    echo ""
    echo "## [$VERSION_WITHOUT_V] - $(date +%Y-%m-%d)"
    echo ""
    echo "### Added"
    echo "- New features..."
    echo ""
    echo "### Changed"
    echo "- Changes..."
    echo ""
    exit 1
fi
success "CHANGELOG.md contains entry for $VERSION"

# Step 6: Extract release notes from CHANGELOG
info "Step 6/6: Extracting release notes..."
RELEASE_NOTES=$(awk -v version="$VERSION_WITHOUT_V" '
    BEGIN { found=0; output="" }
    /^## \[?v?'"$VERSION_WITHOUT_V"'\]?/ { found=1; next }
    /^## \[?v?[0-9]+\.[0-9]+\.[0-9]+/ && found { exit }
    found { output = output $0 "\n" }
    END { print output }
' "$CHANGELOG_FILE")

if [[ -z "$RELEASE_NOTES" ]]; then
    warn "Could not extract release notes from CHANGELOG"
    RELEASE_NOTES="Release $VERSION"
fi

echo ""
echo "Release notes:"
echo "────────────────────────────────────────"
echo "$RELEASE_NOTES"
echo "────────────────────────────────────────"
echo ""

# Commit generated files
if [[ "$NO_COMMIT" == "false" ]]; then
    info "Committing generated files..."

    # Check if there are changes to commit
    CHANGED_FILES=""
    if [[ -f "go.sum" ]] && git diff --quiet go.sum 2>/dev/null || [[ $? -eq 1 ]]; then
        CHANGED_FILES="$CHANGED_FILES go.sum"
    fi
    if [[ -d "completions" ]]; then
        CHANGED_FILES="$CHANGED_FILES completions/"
    fi

    if [[ -n "$CHANGED_FILES" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "  [DRY-RUN] Would commit: $CHANGED_FILES"
        else
            git add $CHANGED_FILES
            git commit -m "chore(release): prepare $VERSION

- Run go mod tidy
- Generate shell completions"
            success "Committed generated files"
        fi
    else
        info "No generated files to commit"
    fi
else
    warn "Skipping commit (--no-commit)"
fi

# Create annotated git tag
if [[ "$NO_TAG" == "false" ]]; then
    info "Creating annotated git tag $VERSION..."

    if [[ "$DRY_RUN" == "true" ]]; then
        echo "  [DRY-RUN] Would create tag: $VERSION"
    else
        if [[ "$FORCE" == "false" ]]; then
            read -p "Create tag $VERSION? [Y/n] " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Nn]$ ]]; then
                warn "Tag creation skipped"
            else
                git tag -a "$VERSION" -m "Release $VERSION

$RELEASE_NOTES"
                success "Created tag $VERSION"
            fi
        else
            git tag -a "$VERSION" -m "Release $VERSION

$RELEASE_NOTES"
            success "Created tag $VERSION"
        fi
    fi
else
    warn "Skipping tag creation (--no-tag)"
fi

# Summary
echo ""
echo "════════════════════════════════════════"
success "Release preparation complete!"
echo "════════════════════════════════════════"
echo ""

if [[ "$DRY_RUN" == "false" && "$NO_TAG" == "false" ]]; then
    info "Next steps:"
    echo ""
    echo "  1. Push the tag to trigger the release:"
    echo "     git push origin $VERSION"
    echo ""
    echo "  2. Or push all tags:"
    echo "     git push --tags"
    echo ""
    echo "  3. Monitor the release:"
    echo "     https://github.com/felixgeelhaar/specular/actions"
    echo ""
fi
