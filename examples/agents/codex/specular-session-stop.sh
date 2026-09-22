#!/usr/bin/env bash
# Specular native Stop hook for Codex (PRODUCT_INTENT P1 #4).
# Installed by: specular session integrate codex
# Calls existing session attest + gate surfaces — no new protocol.
# Register via .codex/hooks.json (hooks.Stop). Prefer Stop over SessionEnd
# so attest+gate have enough timeout budget.
# Specular hook mode: advisory
set -euo pipefail

# Codex feeds Stop event JSON on stdin; drain.
cat >/dev/null || true

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

if ! command -v specular >/dev/null 2>&1; then
  echo "specular: CLI not on PATH; skip attest/gate" >&2
  exit 0
fi

SESSION_ID="${SPECULAR_SESSION_ID:-}"
if [[ -z "$SESSION_ID" && -d .specular/sessions ]] && command -v jq >/dev/null 2>&1; then
  # Prefer the newest session record labeled codex / codex-cli (portable mtime).
  newest_mtime=0
  for path in .specular/sessions/*.json; do
    [[ -f "$path" ]] || continue
    case "$path" in *.attestation.json) continue ;; esac
    harness="$(jq -r '.harness // empty' "$path" 2>/dev/null || true)"
    case "$harness" in codex|codex-cli) ;; *) continue ;; esac
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
  echo "specular: no SPECULAR_SESSION_ID / matching session; skip attest" >&2
fi

echo "specular: running gate (advisory — does not block Stop)" >&2
specular gate || echo "specular: gate exited non-zero (advisory)" >&2
exit 0
