# Competitive brief: Spotify Xirp

> Internal brief for champions, design partners, and anyone answering
> "how does Specular relate to Xirp?" Updated for Xirp public beta
> (announced 2026-08-10).

## What Xirp is

Xirp is Spotify's **vendor-neutral agentic development environment**: a
macOS desktop app that orchestrates many concurrent coding-agent sessions
(Claude Code, Codex, Gemini CLI) across projects and Git worktrees. Optional
Spotify Portal connection injects software-catalog context over MCP and
lets teams share session transcripts.

It is a **session manager / meta-harness**, not a governance gate.

| Dimension | Xirp | Specular |
|-----------|------|----------|
| Primary job | Run and switch many agent sessions | Gate, approve, and attest AI-authored change |
| Surface | macOS desktop app | Cross-platform CLI + CI action |
| Isolation | Git worktree per session | Git worktree per session (`specular worktree` / `auto --worktree`) + Docker sandbox |
| Vendor neutrality | Wraps Claude Code / Codex / Gemini CLIs | Routes across API + CLI providers (same agents, plus Ollama) |
| Org context | Portal catalog / Workspaces via MCP | Spec + policy + ADR/spec lock as governed inputs |
| Evidence | Session transcripts (manual upload; **no secret redaction**) | Signed bundles, drift hashes, harness + worktree provenance |
| Open source | No (proprietary; Portal is commercial) | Apache 2.0 core |
| Platforms | macOS only (beta) | Linux / macOS / Windows |
| Where it sits | Inner loop (authoring) | Outer loop (CI/CD change control) |

## Threat model (why this competitor matters)

Xirp does **not** replace Specular's wedge today. It does three things that
can erode Specular's narrative if we stay silent:

1. **Steals the "vendor neutrality" story.** Xirp's launch pitch is the
   same multi-harness freedom Specular already ships via providers. Buyers
   hearing "don't lock to one agent" will map that to Xirp first unless we
   restate it in AI Change Control language.
2. **Normalizes parallel agent sessions.** 50+ concurrent worktrees becomes
   table stakes. Specular without worktree isolation looks like a single-
   threaded CLI next to a grid view — even though our job is the gate.
3. **Portal expands toward org visibility.** Transcript upload + catalog
   context is adjacent to "what did AI change, where?" Without a crisp
   complement story, Platform buyers may assume Portal covers auditability.

## Response thesis

> **Xirp multiplies how many AI sessions your engineers run.
> Specular is the only place those sessions become auditable change.**

Complement, do not clone. We:

- **Coexist in the inner loop** — Xirp (or Cursor, Claude Code alone)
  authors; Specular gates.
- **Own the outer loop** — drift, policy, signed bundles, harness
  provenance in CI.
- **Close the isolation gap** — `specular worktree` / `auto --worktree`
  so Specular-orchestrated sessions get the same Git isolation Xirp
  popularized, with the path and branch recorded in attestation.
- **Close the attribution gap** — `provenance.harness` + worktree fields
  answer "which agent surface authored this?" — something Xirp transcripts
  do not productize as auditor evidence.

## What we will not build

Do not chase Xirp feature-for-feature:

- macOS grid-view desktop UI
- Portal-style catalog marketplace
- Session forking as a UX product
- Transcript social sharing

Those dilute the AI Change Control category claim. If a buyer wants a
desktop session manager, recommend Xirp (or Cursor multi-agent) and sell
Specular as the gate that sits after it.

## Talking points

**For Platform Engineering**

> "Xirp makes parallel agents usable. Specular makes their output shippable
> under your existing CI. Wire `specular eval drift` once; every Xirp
> session that lands a PR still hits the same gate."

**For Security / GRC**

> "Xirp FAQ is explicit: transcripts are not redacted before Portal upload.
> Specular never requires uploading conversation history — the evidence is
> the signed bundle and drift hash in *your* repo."

**For "we're standardising on Portal"**

> "Portal context helps agents start smarter. It does not produce SOC 2 /
> ISO 42001 change-management evidence. Keep Portal for catalog; keep
> Specular for the gate. They share MCP; they do not share the control."

## Proof artefacts to show

1. `specular worktree create demo && specular auto --worktree demo --attest "..."`
   — isolated session with harness + worktree in the attestation JSON.
2. Side-by-side: Xirp FAQ "Does Xirp redact secrets?" → No; Specular
   bundle YAML with no transcript requirement.
3. Anti-positioning row in [`ci-cd-policy-enforcement.md`](../ci-cd-policy-enforcement.md).

## Sources

- [Introducing Xirp (Spotify Portal blog)](https://portal.spotify.com/blog/introducing-xirp)
- [Xirp docs](https://backstage.spotify.com/docs/xirp)
- [Xirp FAQ](https://backstage.spotify.com/docs/xirp/faq) — proprietary, macOS-only, no transcript redaction
