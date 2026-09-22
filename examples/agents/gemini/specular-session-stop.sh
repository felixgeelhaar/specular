#!/usr/bin/env bash
# Specular native SessionEnd hook for Gemini CLI (PRODUCT_INTENT P1 #4).
# Installed by: specular session integrate gemini
# Calls existing session attest + gate surfaces — no new protocol.
# Gemini hooks require JSON-only stdout; logs go to stderr.
set -euo pipefail

# SessionEnd feeds JSON on stdin; drain it.
cat >/dev/null || true

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

if ! command -v specular >/dev/null 2>&1; then
  echo "specular: CLI not on PATH; skip attest/gate" >&2
  printf '%s\n' '{}'
  exit 0
fi

SESSION_ID="${SPECULAR_SESSION_ID:-}"
if [[ -z "$SESSION_ID" && -d .specular/sessions ]] && command -v jq >/dev/null 2>&1; then
  # Prefer the newest session record labeled gemini / gemini-cli (portable mtime).
  newest_mtime=0
  for path in .specular/sessions/*.json; do
    [[ -f "$path" ]] || continue
    case "$path" in *.attestation.json) continue ;; esac
    harness="$(jq -r '.harness // empty' "$path" 2>/dev/null || true)"
    case "$harness" in gemini|gemini-cli) ;; *) continue ;; esac
    mtime="$(stat -c %Y "$path" 2>/dev/null || stat -f %m "$path" 2>/dev/null || echo 0)"
    if [[ "$mtime" -ge "$newest_mtime" ]]; then
      newest_mtime="$mtime"
      SESSION_ID="$(jq -r '.id // empty' "$path" 2>/dev/null || true)"
    fi
  done
fi

if [[ -n "$SESSION_ID" ]]; then
  echo "specular: attesting session ${SESSION_ID} (harness provenance)" >&2
  specular session attest "$SESSION_ID" || echo "specular: attest failed (advisory)" >&2
else
  echo "specular: no SPECULAR_SESSION_ID / gemini session; skip attest" >&2
fi

echo "specular: running gate (advisory — does not block SessionEnd)" >&2
specular gate || echo "specular: gate exited non-zero (advisory)" >&2

# Gemini SessionEnd is best-effort; emit empty JSON object on stdout.
printf '%s\n' '{}'
exit 0
