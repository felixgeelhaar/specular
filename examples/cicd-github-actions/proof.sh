#!/usr/bin/env bash
# Local one-shot proof: fleet → evidence → land (CI-native vs Mac session grids).
# Usage: from repo root, with specular on PATH:
#   ./examples/cicd-github-actions/proof.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

FLEET="${FLEET_PATH:-examples/cicd-github-actions/fleet.yaml}"
POLICY_ID="${POLICY_ID:-soc2-cc8.1}"
BUNDLE_OUT="${BUNDLE_OUT:-session-evidence.sbundle.tgz}"
TIMEOUT="${WAIT_TIMEOUT:-15m}"

echo "==> policy library install ${POLICY_ID}"
specular policy library install "$POLICY_ID"

echo "==> session batch --governed ${FLEET}"
specular session batch --governed "$FLEET"
specular session status

echo "==> session wait --timeout ${TIMEOUT} --stop --bundle"
specular session wait --timeout "$TIMEOUT" --stop --bundle \
  --policy ".specular/policies/${POLICY_ID}.yaml" \
  --bundle-out "$BUNDLE_OUT"

echo "==> land: diff / commit / sync --fetch (implement)"
if specular session show implement >/dev/null 2>&1; then
  specular session diff implement --stat || true
  if [[ -n "$(specular session diff implement --name-only 2>/dev/null || true)" ]]; then
    specular session commit implement --all -m "proof: land implement session"
  else
    echo "    (no worktree changes to commit)"
  fi
  if ! specular session sync implement --fetch; then
    echo "    --fetch failed (no remote?); syncing onto local base"
    specular session sync implement
  fi
fi

echo "==> artifacts"
ls -la "$BUNDLE_OUT" drift.sarif .specular/sessions/*.attestation.json 2>/dev/null || true
echo "Proof complete. Optional next: specular session push implement --pr"
echo "                          or: specular session merge implement"
