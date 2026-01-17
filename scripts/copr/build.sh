#!/usr/bin/env bash
set -euo pipefail

PROJECT="${COPR_PROJECT:?Specify COPR_PROJECT env var (owner/name)}"
CHROOTS="${COPR_CHROOTS:-fedora-40}"
SPECULAR_VERSION="${SPECULAR_VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || printf "0.0.0")}"
DIST_DIR="dist"
TARBALL="${DIST_DIR}/specular-${SPECULAR_VERSION}.tar.gz"
SOURCE_DIR="packaging/copr"

mkdir -p "$DIST_DIR"
mkdir -p "$SOURCE_DIR"

echo "Building CLI binary for Copr..."
go build -ldflags "-s -w" -o "$DIST_DIR/specular" ./cmd/specular

echo "Packaging source tarball..."
tar czf "$TARBALL" -C "$DIST_DIR" specular
cp "$TARBALL" "${SOURCE_DIR}/specular-${SPECULAR_VERSION}.tar.gz"

pushd "$SOURCE_DIR" >/dev/null
echo "Running copr-cli build for project $PROJECT on $CHROOTS..."
copr-cli build \
  --now \
  --project "$PROJECT" \
  --chroots "$CHROOTS" \
  specular.spec
popd >/dev/null

echo "Copr build requested for specular-${SPECULAR_VERSION} (project $PROJECT)."
