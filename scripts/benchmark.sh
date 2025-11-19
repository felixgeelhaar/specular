#!/bin/bash
# Specular Performance Benchmark Script
# Measures binary size, startup time, and runs Go benchmarks
# Output format compatible with benchstat for comparison

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[PASS]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[FAIL]${NC} $1" >&2; }

# Print usage
usage() {
    cat << EOF
Usage: $(basename "$0") [options]

Run performance benchmarks for Specular CLI.

Options:
    -h, --help          Show this help message
    -o, --output FILE   Write results to file (default: stdout)
    -j, --json          Output in JSON format
    --skip-build        Skip building binary
    --skip-go-bench     Skip Go benchmarks
    --count N           Run benchmarks N times (default: 3)
    --compare FILE      Compare results with previous benchmark

Examples:
    $(basename "$0")
    $(basename "$0") -o benchmark-results.txt
    $(basename "$0") --compare baseline.txt
EOF
    exit "${1:-0}"
}

# Parse arguments
OUTPUT=""
JSON_OUTPUT=false
SKIP_BUILD=false
SKIP_GO_BENCH=false
BENCH_COUNT=3
COMPARE_FILE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage 0
            ;;
        -o|--output)
            OUTPUT="$2"
            shift 2
            ;;
        -j|--json)
            JSON_OUTPUT=true
            shift
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-go-bench)
            SKIP_GO_BENCH=true
            shift
            ;;
        --count)
            BENCH_COUNT="$2"
            shift 2
            ;;
        --compare)
            COMPARE_FILE="$2"
            shift 2
            ;;
        -*)
            error "Unknown option: $1"
            usage 1
            ;;
        *)
            error "Unexpected argument: $1"
            usage 1
            ;;
    esac
done

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Results variables (instead of associative array for bash 3.x compatibility)
BINARY_SIZE_BYTES=""
BINARY_SIZE_MB=""
HELP_TIME=""
VERSION_TIME=""
PROVIDER_TIME=""

# Build binary if needed
if [[ "$SKIP_BUILD" == "false" ]]; then
    info "Building release binary..."
    go build -ldflags="-s -w" -trimpath -o specular-bench ./cmd/specular
    BINARY="./specular-bench"
else
    if [[ -f "./specular" ]]; then
        BINARY="./specular"
    else
        error "No binary found. Run without --skip-build."
        exit 1
    fi
fi

echo ""
echo "════════════════════════════════════════"
echo "Specular Performance Benchmark"
echo "════════════════════════════════════════"
echo ""
echo "Date: $(date -Iseconds)"
echo "Go version: $(go version | cut -d' ' -f3)"
echo "Platform: $(uname -s)/$(uname -m)"
echo ""

# ============================================================================
# SECTION 1: Binary Size
# ============================================================================
echo "────────────────────────────────────────"
echo "Section 1: Binary Size"
echo "────────────────────────────────────────"

BINARY_SIZE=$(stat -f%z "$BINARY" 2>/dev/null || stat -c%s "$BINARY")
BINARY_SIZE_MB=$(echo "scale=2; $BINARY_SIZE / 1048576" | bc)

echo "Binary: $BINARY"
echo "Size: ${BINARY_SIZE_MB}MB ($BINARY_SIZE bytes)"
echo ""

BINARY_SIZE_BYTES=$BINARY_SIZE

# Check against target (21MB is reasonable for Docker+AWS+Sigstore+TUI)
TARGET_SIZE_MB=22
if (( $(echo "$BINARY_SIZE_MB <= $TARGET_SIZE_MB" | bc -l) )); then
    success "Binary size within target (≤${TARGET_SIZE_MB}MB)"
else
    warn "Binary size exceeds target (${BINARY_SIZE_MB}MB > ${TARGET_SIZE_MB}MB)"
fi
echo ""

# ============================================================================
# SECTION 2: Startup Time
# ============================================================================
echo "────────────────────────────────────────"
echo "Section 2: Startup Time"
echo "────────────────────────────────────────"

# Measure startup time with hyperfine if available, otherwise use time
if command -v hyperfine &> /dev/null; then
    info "Using hyperfine for precise measurements"
    echo ""

    # Help command (minimal startup)
    echo "Command: specular --help"
    HELP_RESULT=$(hyperfine --warmup 3 --runs "$BENCH_COUNT" --export-json /tmp/help_bench.json "$BINARY --help" 2>&1)
    HELP_TIME=$(jq -r '.results[0].mean * 1000' /tmp/help_bench.json)
    echo "Mean: ${HELP_TIME}ms"
    echo ""

    # Version command
    echo "Command: specular version"
    VERSION_RESULT=$(hyperfine --warmup 3 --runs "$BENCH_COUNT" --export-json /tmp/version_bench.json "$BINARY version" 2>&1)
    VERSION_TIME=$(jq -r '.results[0].mean * 1000' /tmp/version_bench.json)
    echo "Mean: ${VERSION_TIME}ms"
    echo ""

    # Provider list command
    echo "Command: specular provider list"
    PROVIDER_RESULT=$(hyperfine --warmup 3 --runs "$BENCH_COUNT" --export-json /tmp/provider_bench.json "$BINARY provider list" 2>&1)
    PROVIDER_TIME=$(jq -r '.results[0].mean * 1000' /tmp/provider_bench.json)
    echo "Mean: ${PROVIDER_TIME}ms"
    echo ""

    rm -f /tmp/help_bench.json /tmp/version_bench.json /tmp/provider_bench.json
else
    warn "hyperfine not installed, using basic measurements"
    warn "Install with: brew install hyperfine"
    echo ""

    # Basic time measurements
    echo "Command: specular --help"
    HELP_TIME_RAW=$( { time $BINARY --help > /dev/null; } 2>&1 | grep real | awk '{print $2}')
    echo "Time: $HELP_TIME_RAW"
    echo ""

    echo "Command: specular version"
    VERSION_TIME_RAW=$( { time $BINARY version > /dev/null; } 2>&1 | grep real | awk '{print $2}')
    echo "Time: $VERSION_TIME_RAW"
    echo ""
fi

# Check against target
TARGET_STARTUP_MS=100
if [[ -n "${HELP_TIME:-}" ]]; then
    if (( $(echo "$HELP_TIME <= $TARGET_STARTUP_MS" | bc -l) )); then
        success "Startup time within target (≤${TARGET_STARTUP_MS}ms)"
    else
        warn "Startup time exceeds target"
    fi
fi
echo ""

# ============================================================================
# SECTION 3: Go Benchmarks
# ============================================================================
if [[ "$SKIP_GO_BENCH" == "false" ]]; then
    echo "────────────────────────────────────────"
    echo "Section 3: Go Benchmarks"
    echo "────────────────────────────────────────"

    # Run benchmarks
    info "Running Go benchmarks (count=$BENCH_COUNT)..."
    echo ""

    # Run benchmarks and save to file
    BENCH_OUTPUT="/tmp/specular_benchmarks.txt"
    go test -bench=. -benchmem -count="$BENCH_COUNT" ./internal/metrics ./internal/spec 2>&1 | tee "$BENCH_OUTPUT"

    echo ""

    # Compare with previous if requested
    if [[ -n "$COMPARE_FILE" && -f "$COMPARE_FILE" ]]; then
        if command -v benchstat &> /dev/null; then
            echo "────────────────────────────────────────"
            echo "Benchmark Comparison"
            echo "────────────────────────────────────────"
            benchstat "$COMPARE_FILE" "$BENCH_OUTPUT"
        else
            warn "benchstat not installed, skipping comparison"
            warn "Install with: go install golang.org/x/perf/cmd/benchstat@latest"
        fi
    fi
else
    BENCH_OUTPUT=""
fi
echo ""

# ============================================================================
# SECTION 4: Memory Profile
# ============================================================================
echo "────────────────────────────────────────"
echo "Section 4: Memory Analysis"
echo "────────────────────────────────────────"

# Get binary sections size
if command -v size &> /dev/null; then
    echo "Binary sections:"
    size "$BINARY"
    echo ""
fi

# Analyze what's in the binary
info "Top packages by size (estimated):"
go build -ldflags="-s -w" -trimpath -o /tmp/specular-size-analysis ./cmd/specular 2>&1
if command -v go-size-analyzer &> /dev/null; then
    go-size-analyzer /tmp/specular-size-analysis
else
    # Use nm to get symbols
    go tool nm -size "$BINARY" 2>/dev/null | sort -rn -k2 | head -20 || echo "Symbol analysis not available"
fi
rm -f /tmp/specular-size-analysis
echo ""

# ============================================================================
# SUMMARY
# ============================================================================
echo "════════════════════════════════════════"
echo "Summary"
echo "════════════════════════════════════════"
echo ""
echo "Binary Size:    ${BINARY_SIZE_MB}MB (target: <${TARGET_SIZE_MB}MB)"
if [[ -n "${HELP_TIME:-}" ]]; then
    echo "Startup Time:   ${HELP_TIME}ms (target: <${TARGET_STARTUP_MS}ms)"
fi
echo ""

# JSON output
if [[ "$JSON_OUTPUT" == "true" ]]; then
    echo "{"
    echo "  \"timestamp\": \"$(date -Iseconds)\","
    echo "  \"go_version\": \"$(go version | cut -d' ' -f3)\","
    echo "  \"platform\": \"$(uname -s)/$(uname -m)\","
    echo "  \"binary_size_bytes\": $BINARY_SIZE,"
    echo "  \"binary_size_mb\": $BINARY_SIZE_MB"
    if [[ -n "${HELP_TIME:-}" ]]; then
        echo "  ,\"help_time_ms\": $HELP_TIME"
        echo "  ,\"version_time_ms\": $VERSION_TIME"
        echo "  ,\"provider_list_time_ms\": $PROVIDER_TIME"
    fi
    echo "}"
fi

# Write to output file if specified
if [[ -n "$OUTPUT" ]]; then
    {
        echo "# Specular Benchmark Results"
        echo "# Date: $(date -Iseconds)"
        echo "# Go: $(go version | cut -d' ' -f3)"
        echo "# Platform: $(uname -s)/$(uname -m)"
        echo ""
        echo "BenchmarkBinarySize 1 $BINARY_SIZE bytes"
        if [[ -n "${HELP_TIME:-}" ]]; then
            printf "BenchmarkStartupHelp 1 %.0f ns/op\n" "$(echo "$HELP_TIME * 1000000" | bc)"
            printf "BenchmarkStartupVersion 1 %.0f ns/op\n" "$(echo "$VERSION_TIME * 1000000" | bc)"
        fi
    } > "$OUTPUT"

    # Append Go benchmarks if run
    if [[ "$SKIP_GO_BENCH" == "false" && -f "$BENCH_OUTPUT" ]]; then
        echo "" >> "$OUTPUT"
        cat "$BENCH_OUTPUT" >> "$OUTPUT"
    fi

    success "Results written to $OUTPUT"
fi

# Cleanup
rm -f specular-bench "$BENCH_OUTPUT" 2>/dev/null || true

info "Benchmark complete"
