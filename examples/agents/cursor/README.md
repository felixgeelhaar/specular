# Cursor — Specular stop hook

Reference copy of what `specular session integrate cursor` writes.

Prefer the CLI so `hooks.json` merge is idempotent:

```bash
specular session integrate cursor
specular session integrate cursor --enforce --force  # Level-3 fail-closed
specular session integrate cursor --enforce --require-governed --force  # + governed
```

Alias: `specular session integrate cursor-agent`.

## Files

| File | Role |
|------|------|
| [`specular-session-stop.sh`](./specular-session-stop.sh) | `stop` hook → `session attest` + `gate` |
| [`hooks.json`](./hooks.json) | `hooks.stop` snippet to merge into `.cursor/hooks.json` |

## Manual install

```bash
mkdir -p .cursor/hooks
cp examples/agents/cursor/specular-session-stop.sh .cursor/hooks/
chmod +x .cursor/hooks/specular-session-stop.sh
# Merge hooks.json into .cursor/hooks.json (or use session integrate)
```

Cursor loads project hooks from `.cursor/hooks.json` (trusted workspace).
Confirm under **Settings → Hooks**. Paths are relative to the project root.

## Provenance flow

```
session start (exports SPECULAR_SESSION_* into managed launches)
        │  export SPECULAR_SESSION_ID / SPECULAR_SESSION_HARNESS
        │  into the Cursor agent environment when using a managed session
        ▼
Cursor Agent stop
        │  .cursor/hooks/specular-session-stop.sh
        ▼
session attest  →  .specular/sessions/<id>.attestation.json
        │            provenance.harness from session record
        ▼
specular gate   →  ALLOW/DENY board (advisory default; --enforce fails closed)
```
