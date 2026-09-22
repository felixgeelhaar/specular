# Codex — Specular Stop hook

Reference copy of what `specular session integrate codex` writes.

Prefer the CLI so `hooks.json` merge is idempotent:

```bash
specular session integrate codex
```

Alias: `specular session integrate codex-cli`.

## Files

| File | Role |
|------|------|
| [`specular-session-stop.sh`](./specular-session-stop.sh) | Stop hook → `session attest` + `gate` |
| [`hooks.json`](./hooks.json) | `hooks.Stop` snippet to merge into `.codex/hooks.json` |

## Manual install

```bash
mkdir -p .codex/hooks
cp examples/agents/codex/specular-session-stop.sh .codex/hooks/
chmod +x .codex/hooks/specular-session-stop.sh
# Merge hooks.json into .codex/hooks.json (or use session integrate)
```

Project `.codex/` hooks load when the Codex project layer is trusted.
Review/trust the Stop hook via Codex `/hooks`. Prefer **Stop** (turn end)
over SessionEnd so attest+gate have enough timeout budget.

## Provenance flow

```
session start --harness codex
        │  exports SPECULAR_SESSION_ID / SPECULAR_SESSION_HARNESS
        ▼
Codex Stop
        │  .codex/hooks/specular-session-stop.sh
        ▼
session attest  →  .specular/sessions/<id>.attestation.json
        │            provenance.harness = codex
        ▼
specular gate   →  ALLOW/DENY board (advisory default; --enforce fails closed)
```
