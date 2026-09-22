# Gemini CLI — Specular SessionEnd hook

Reference copy of what `specular session integrate gemini` writes.

Prefer the CLI so settings merge is idempotent:

```bash
specular session integrate gemini
specular session integrate gemini --enforce --force  # Level-3 fail-closed
```

Alias: `specular session integrate gemini-cli`.

## Files

| File | Role |
|------|------|
| [`specular-session-stop.sh`](./specular-session-stop.sh) | SessionEnd hook → `session attest` + `gate` |
| [`settings.hooks.json`](./settings.hooks.json) | `hooks.SessionEnd` + `hooksConfig.enabled` snippet for `.gemini/settings.json` |

## Manual install

```bash
mkdir -p .gemini/hooks
cp examples/agents/gemini/specular-session-stop.sh .gemini/hooks/
chmod +x .gemini/hooks/specular-session-stop.sh
# Merge settings.hooks.json into .gemini/settings.json (or use session integrate)
```

Gemini requires JSON-only stdout from hooks (logs go to stderr). SessionEnd
is advisory by default; use `session integrate gemini --enforce --force` for fail-closed SessionEnd. Ensure `hooksConfig.enabled` is
true (integrate sets this when merging).

## Provenance flow

```
session start --harness gemini
        │  exports SPECULAR_SESSION_ID / SPECULAR_SESSION_HARNESS
        ▼
Gemini SessionEnd
        │  .gemini/hooks/specular-session-stop.sh
        ▼
session attest  →  .specular/sessions/<id>.attestation.json
        │            provenance.harness = gemini
        ▼
specular gate   →  ALLOW/DENY board (advisory default; --enforce fails closed)
```
