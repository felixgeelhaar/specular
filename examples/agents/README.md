# Native agent integrations

PRODUCT_INTENT P1 #4 starter: wire coding-agent harnesses into Specular's
existing session attest + gate surfaces — no new protocol.

## Install via CLI (preferred)

```bash
# Preview files that would be written
specular session integrate claude-code --dry-run
specular session integrate cursor --dry-run

# Write hooks + merge settings / hooks.json
specular session integrate claude-code
specular session integrate cursor

# Then launch a governed session (exports SPECULAR_SESSION_ID for the Stop hook)
specular session start --harness claude-code --governed "Harden JWT validation"
```

On Claude Code **Stop** / Cursor Agent **stop**, the hook:

1. Resolves `SPECULAR_SESSION_ID` (from `session start`) or the newest
   matching session record under `.specular/sessions/`
2. Runs `specular session attest <id>` (harness + worktree provenance)
3. Runs `specular gate` (advisory — does not block Stop)

## Example files

| Path | Purpose |
|------|---------|
| [`claude-code/`](./claude-code/) | Reference copies of the Claude Code Stop hook + settings snippet |
| [`cursor/`](./cursor/) | Reference copies of the Cursor stop hook + `.cursor/hooks.json` snippet |

Codex / Gemini / Aider hooks can follow the same pattern later:
call `specular session attest` + `specular gate` with harness labels already
recorded by `session start --harness …`.
