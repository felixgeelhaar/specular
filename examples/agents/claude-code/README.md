# Claude Code — Specular Stop hook

Reference copy of what `specular session integrate claude-code` writes.

Prefer the CLI so settings merge is idempotent:

```bash
specular session integrate claude-code
specular session integrate claude-code --enforce --force  # Level-3 fail-closed
```

## Files

| File | Role |
|------|------|
| [`specular-session-stop.sh`](./specular-session-stop.sh) | Stop hook → `session attest` + `gate` |
| [`settings.hooks.json`](./settings.hooks.json) | `hooks.Stop` snippet to merge into `.claude/settings.json` |

## Manual install

```bash
mkdir -p .claude/hooks
cp examples/agents/claude-code/specular-session-stop.sh .claude/hooks/
chmod +x .claude/hooks/specular-session-stop.sh
# Merge settings.hooks.json into .claude/settings.json (or use session integrate)
```

## Provenance flow

```
session start --harness claude-code
        │  exports SPECULAR_SESSION_ID / SPECULAR_SESSION_HARNESS
        ▼
Claude Code Stop
        │  .claude/hooks/specular-session-stop.sh
        ▼
session attest  →  .specular/sessions/<id>.attestation.json
        │            provenance.harness = claude-code
        ▼
specular gate   →  ALLOW/DENY board (advisory default; --enforce fails closed)
```
