# Competitive brief: Spotify Xirp

> Specular is built to **win** the agent-session category against Xirp —
> not to sit politely beside it. We compete on parallel harness sessions
> *and* on the governance Xirp does not ship.

## What Xirp is

Xirp is Spotify's macOS desktop app for running Claude Code, Codex, and
Gemini sessions in parallel Git worktrees, with optional Portal catalog
context. It proved demand for vendor-neutral multi-agent orchestration.

## Where Specular competes — and wins

| Dimension | Xirp | Specular |
|-----------|------|----------|
| Parallel sessions | Desktop grid | `session start/list/logs/fork/stop` |
| Harnesses | Claude Code, Codex, Gemini | **Same**, plus governed `specular-auto` |
| Isolation | Git worktrees | Git worktrees (`.specular/worktrees/`) |
| Platforms | **macOS only** | **Linux / macOS / Windows** |
| License | Proprietary + Portal upsell | **Apache 2.0** |
| Governance gate | None (transcripts, no redaction) | **Drift + policy + signed bundles** |
| Harness attribution | Transcript only | **`provenance.harness` + worktree fields** |
| Org context | Portal catalog (paid) | Spec + policy + ADR (in-repo) |

**Competitive thesis**

> Specular is the open, cross-platform control plane for parallel coding
> agents — with the only auditor-ready change-control gate in the category.

Xirp's desktop grid is a UX advantage on Mac. Everywhere else — and
everywhere compliance matters — Specular is the stronger product.

## Threat model (still take Xirp seriously)

1. **Desktop polish.** Grid + PTY is sticky for Mac-native teams. We do
   not ship a GUI yet; we win on CLI density, CI, and evidence.
2. **Portal distribution.** Spotify can attach sessions to Backstage
   commercial motion. Counter with open core + auditor enablement.
3. **Mindshare.** Keep saying "Claude Code / Codex / Gemini in worktrees
   *and* a drift gate" so buyers don't map the whole category to Xirp.

## Product posture

**Ship to compete**

- Native harness launch: `claude-code`, `codex`, `gemini`, `specular-auto`
- Worktree isolation per session
- `session status [--watch]` live board + `session open` worktree helper
- `session wait` scriptable parallel gate + `session restart` harness swap
- `session rm` / `session prune` lifecycle cleanup after fleets finish
- `session harnesses` with PATH availability probe
- `session logs --follow`, `session fork`
- Harness + worktree provenance into attestations
- Drift / policy / bundle outer loop

**Do not clone**

- macOS-only GUI grid
- Portal marketplace / transcript social sharing

## Talking points

**Platform**

> "Same harnesses as Xirp — Claude, Codex, Gemini — in worktrees, on every
> OS, open source. Plus the CI gate Xirp never built. One tool."

**Security**

> "Xirp does not redact transcript uploads. Specular never requires
> uploading conversations. The evidence is a signed bundle in your repo."

**Against Portal lock-in**

> "Portal is a catalog. Specular is change control. If you already bought
> Portal, keep it — and still run Specular sessions so shipping stays
> auditable."

## Proof

```bash
specular session harnesses
specular session start --harness claude-code --name demo "Add /healthz"
specular session start --harness codex --name demo-2 "Add rate limiting"
specular session status --watch
specular session wait demo demo-2
cd "$(specular session open demo)"
specular session restart demo --harness gemini --force
specular session logs demo --follow
specular eval drift --fail-on-change
specular session prune --delete-branch
```

## Sources

- [Introducing Xirp](https://portal.spotify.com/blog/introducing-xirp)
- [Xirp docs](https://backstage.spotify.com/docs/xirp)
- [Xirp FAQ](https://backstage.spotify.com/docs/xirp/faq) — macOS-only, proprietary, no transcript redaction
