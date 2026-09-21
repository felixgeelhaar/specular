# Native agent integrations

PRODUCT_INTENT P1 #4 starter: wire one coding-agent harness into Specular's
existing session attest + gate surfaces — no new protocol.

## Install via CLI (preferred)

```bash
# Preview files that would be written
specular session integrate claude-code --dry-run

# Write .claude/hooks/specular-session-stop.sh + merge hooks.Stop into settings
specular session integrate claude-code

# Then launch a governed session (exports SPECULAR_SESSION_ID for the Stop hook)
specular session start --harness claude-code --governed "Harden JWT validation"
```

On Claude Code **Stop**, the hook:

1. Resolves `SPECULAR_SESSION_ID` (from `session start`) or the newest
   `claude-code` session record under `.specular/sessions/`
2. Runs `specular session attest <id>` (harness + worktree provenance)
3. Runs `specular gate` (advisory — does not block Stop)

## Example files

| Path | Purpose |
|------|---------|
| [`claude-code/`](./claude-code/) | Reference copies of the Claude Code Stop hook + settings snippet |

Codex / Gemini / Aider / Cursor hooks can follow the same pattern later:
call `specular session attest` + `specular gate` with harness labels already
recorded by `session start --harness …`.
