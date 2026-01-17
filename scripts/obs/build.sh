#!/usr/bin/env bash
set -euo pipefail

OBS_PROJECT="${OBS_PROJECT:?Specify OBS_PROJECT env var (namespace/project)}"
OBS_API_URL="${OBS_API_URL:-https://api.opensuse.org}"
OBS_WORKDIR="${OBS_WORKDIR:-/tmp/specular-obs}"
SPECULAR_VERSION="${SPECULAR_VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || printf "0.0.0")}"
DIST_DIR="dist"
TARBALL="${DIST_DIR}/specular-${SPECULAR_VERSION}.tar.gz"

mkdir -p "$DIST_DIR"

echo "Building CLI binary for OBS..."
go build -ldflags "-s -w" -o "$DIST_DIR/specular" ./cmd/specular

echo "Creating source tarball..."
tar czf "$TARBALL" -C "$DIST_DIR" specular

echo "Checking out OBS project $OBS_PROJECT..."
rm -rf "$OBS_WORKDIR"
osc -A "$OBS_API_URL" checkout "$OBS_PROJECT" "$OBS_WORKDIR"

cp "$TARBALL" "$OBS_WORKDIR/specular-${SPECULAR_VERSION}.tar.gz"
cp packaging/obs/specular.spec "$OBS_WORKDIR/specular.spec"

pushd "$OBS_WORKDIR" >/dev/null
osc add "specular-${SPECULAR_VERSION}.tar.gz" specular.spec || true
osc commit -m "Release ${SPECULAR_VERSION}"
osc build
popd >/dev/null

echo "OBS build submitted for ${SPECULAR_VERSION} in $OBS_PROJECT."
