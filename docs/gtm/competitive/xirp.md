# Competitive brief: Spotify Xirp

> Internal brief for champions and design partners. Specular manages **both**
> loops — parallel agent authoring and the change-control gate. Updated after
> the decision to own inner-loop session orchestration (not only complement
> Xirp).

## What Xirp is

Xirp is Spotify's **vendor-neutral agentic development environment**: a
macOS desktop app that orchestrates many concurrent coding-agent sessions
(Claude Code, Codex, Gemini CLI) across projects and Git worktrees. Optional
Spotify Portal connection injects software-catalog context over MCP and
lets teams share session transcripts.

Xirp proved the category of **parallel agent session management**. Specular
implements that category in the CLI — and keeps the governance gate that
Xirp does not have.

| Dimension | Xirp | Specular |
|-----------|------|----------|
| Primary job | Desktop session grid for coding agents | **Both loops**: parallel sessions + auditable change control |
| Surface | macOS desktop app | Cross-platform CLI + CI action |
| Inner loop | Persistent terminals, grid, worktrees | `session start/list/show/stop` + worktree isolation |
| Outer loop | Transcripts (manual upload; no redaction) | Drift gate, policy, signed bundles, harness provenance |
| Vendor neutrality | Wraps Claude Code / Codex / Gemini CLIs | Routes API + CLI providers; harness label in attestation |
| Org context | Portal catalog / Workspaces via MCP | Spec + policy + ADR/spec lock as governed inputs |
| Evidence | Session transcripts | Signed bundles + drift hashes + harness/worktree fields |
| Open source | No (proprietary; Portal commercial) | Apache 2.0 core |
| Platforms | macOS only (beta) | Linux / macOS / Windows |

## Threat model

Xirp still matters even when Specular owns both loops:

1. **Desktop UX density.** Grid view and persistent PTYs are a polished
   authoring experience. Specular's first slice is CLI session orchestration
   — not a macOS GUI. Buyers who want a visual grid may still install Xirp.
2. **Portal catalog distribution.** Spotify can bundle sessions with Portal
   commercial motion. Specular wins on evidence and open core, not on
   Backstage marketplace presence.
3. **"Vendor neutrality" mindshare.** Continue naming multi-harness
   explicitly so "don't lock to one agent" maps to Specular sessions +
   providers, not only to Xirp.

## Response thesis

> **Specular manages both loops: parallel agent sessions in the inner loop,
> and the only auditor-ready change-control gate in the outer loop.**

We do **not** defer authoring to Xirp. We:

- **Own the inner loop (CLI)** — `specular session start` launches goals in
  isolated worktrees; `list` / `show` / `stop` track parallel runs with
  harness provenance.
- **Own the outer loop** — drift, policy, signed bundles; every session's
  harness + worktree identity is in the attestation.
- **Differentiate on evidence** — no transcript upload required; no macOS
  lock-in; Apache 2.0.
- **Coexist when useful** — if a team already runs Xirp for PTY grid UX,
  Specular still gates what ships. That is coexistence, not dependency.

## What we build vs defer

**Build (both-loops roadmap)**

- Session registry: start / list / show / stop
- Worktree isolation per session
- Harness + worktree provenance in attestations
- Multi-session status (`working` / `completed` / `failed` / `stopped`)
- Later: attach logs, mid-session harness switch, richer status
  (`waiting` on approval)

**Defer (do not clone the desktop)**

- macOS grid-view GUI
- Portal-style catalog marketplace
- Transcript social sharing
- Session forking as a social UX product

Deferring the desktop is not deferring the inner loop. The CLI session
manager **is** the inner loop.

## Talking points

**For Platform Engineering**

> "One CLI owns parallel agents and the gate. `session start` for isolated
> work; `eval drift` for ship. You do not need a second vendor for
> authoring just to get governance."

**For Security / GRC**

> "Xirp FAQ: transcripts are not redacted before Portal upload. Specular
> sessions never require uploading conversation history — evidence is the
> signed bundle and drift hash in *your* repo, with harness attribution."

**For "we're standardising on Portal / Xirp"**

> "Keep Xirp if you want the desktop grid. Specular still runs the sessions
> you want governed — or gates PRs from any harness. Portal context and
> Specular policy are complementary inputs; only Specular produces change-
> control evidence."

## Proof artefacts to show

1. `specular session start --name demo "Add /healthz"` then
   `specular session list` — parallel inner-loop control surface.
2. `specular auto --worktree demo --attest "..."` — harness + worktree in
   attestation JSON.
3. Side-by-side: Xirp FAQ "Does Xirp redact secrets?" → No; Specular
   bundle YAML with no transcript requirement.

## Sources

- [Introducing Xirp (Spotify Portal blog)](https://portal.spotify.com/blog/introducing-xirp)
- [Xirp docs](https://backstage.spotify.com/docs/xirp)
- [Xirp FAQ](https://backstage.spotify.com/docs/xirp/faq) — proprietary, macOS-only, no transcript redaction
