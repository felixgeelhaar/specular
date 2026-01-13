#!/usr/bin/env bash
set -euo pipefail

SPECULAR_VERSION="${SPECULAR_VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || printf "0.0.0")}" 
ARCH="amd64"
OUTPUT_DIR="dist"
BINARY="dist/specular-linux-${ARCH}/specular"
mkdir -p "dist/specular-linux-${ARCH}"
go build -o "dist/specular-linux-${ARCH}/specular" ./cmd/specular

if [[ ! -f "$BINARY" ]]; then
  echo "Missing CLI binary at $BINARY. Build release artifacts before running this script."
  exit 1
fi

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

INSTALL_ROOT="$TMPDIR/pkg"
mkdir -p "$INSTALL_ROOT/usr/local/bin"
cp "$BINARY" "$INSTALL_ROOT/usr/local/bin/specular"
chmod 755 "$INSTALL_ROOT/usr/local/bin/specular"

fpm -s dir -t deb \
    -a "$ARCH" \
    -n specular \
    -v "$SPECULAR_VERSION" \
    --maintainer "Specular Releases <noreply@specular.ai>" \
    --url "https://github.com/felixgeelhaar/specular" \
    --description "AI-native spec and build assistant" \
    --license "Apache-2.0" \
    -C "$INSTALL_ROOT" \
    usr/local/bin/specular

mkdir -p "$OUTPUT_DIR"
mv "specular_${SPECULAR_VERSION}_${ARCH}.deb" "$OUTPUT_DIR/"
