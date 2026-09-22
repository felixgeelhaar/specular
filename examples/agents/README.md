# Native agent integrations

PRODUCT_INTENT P1 #4: wire coding-agent harnesses into Specular's existing
session attest + gate surfaces — no new protocol.

## Install via CLI (preferred)

```bash
# Preview files that would be written
specular session integrate claude-code --dry-run
specular session integrate cursor --dry-run
specular session integrate codex --dry-run
specular session integrate gemini --dry-run

# Write hooks + merge settings / hooks.json
specular session integrate claude-code
specular session integrate cursor
specular session integrate codex
specular session integrate gemini

# Then launch a governed session (exports SPECULAR_SESSION_ID for the hook)
specular session start --harness claude-code --governed "Harden JWT validation"
```

On Claude Code / Codex **Stop**, Cursor Agent **stop**, or Gemini **SessionEnd**,
the hook:

1. Resolves `SPECULAR_SESSION_ID` (from `session start`) or the newest
   matching session record under `.specular/sessions/`
2. Runs `specular session attest <id>` (harness + worktree provenance)
3. Runs `specular gate` (advisory — does not block Stop/SessionEnd)

## Example files

| Path | Purpose |
|------|---------|
| [`claude-code/`](./claude-code/) | Claude Code Stop hook + settings snippet |
| [`cursor/`](./cursor/) | Cursor stop hook + `.cursor/hooks.json` snippet |
| [`codex/`](./codex/) | Codex Stop hook + `.codex/hooks.json` snippet |
| [`gemini/`](./gemini/) | Gemini SessionEnd hook + settings snippet |

Aliases: `claude` → `claude-code`, `cursor-agent` → `cursor`,
`codex-cli` → `codex`, `gemini-cli` → `gemini`.
