# Specular GTM Wedge

This directory holds the focused go-to-market wedge for Specular and the
role-based assets that pair with it. It is intended for design partners,
internal champions, and anyone preparing to introduce Specular inside a
larger engineering org.

## The wedge in one sentence

> **Specular is the policy-enforced control plane for AI development in CI/CD,
> with auditable drift gates that make every AI-generated change reviewable,
> approvable, and reversible.**

We are not the broadest AI coding tool. We are the **narrow, defensible wedge
between AI assistants and your existing CI/CD + governance stack**.

## Why this wedge

Three forces are colliding inside enterprise engineering orgs in 2026:

1. **AI assistants now author non-trivial production code** — Claude, Codex,
   Gemini, and local models routinely generate, edit, and approve PRs.
2. **CI/CD and policy are the only durable choke points** — every change
   eventually flows through a pipeline, and every regulated org already has
   policy gates there.
3. **Drift, not bugs, is the new audit problem** — leadership needs to answer
   "what changed, who approved it, and what was the AI's reasoning" for every
   release.

Most AI coding tools optimize for the IDE. Specular optimizes for the gate.

## Buyer ICP

The wedge is sharpest for orgs that meet **all** of:

- Already use a code-review-and-CI pipeline (GitHub Actions, GitLab, Jenkins).
- Have a Platform Engineering or DevEx team that owns developer guardrails.
- Have a Security or Governance function (CISO org, AppSec, GRC) that has
  started asking AI-related compliance questions (SOC 2, EU AI Act, ISO 42001).
- 200+ engineers, or fewer engineers but in a regulated vertical (fintech,
  healthcare, public sector, defense).

Below ~50 engineers and outside regulated industries, the pain is real but
the buyer is the same person who builds; we sell open source first.

## Deliverables in this directory

- [`ci-cd-policy-enforcement.md`](./ci-cd-policy-enforcement.md) — the wedge
  story in long form: what we replace, what we integrate with, what we
  uniquely do.
- [`personas/platform-engineering.md`](./personas/platform-engineering.md) —
  doc track for the Platform Engineering / DevEx buyer.
- [`personas/security.md`](./personas/security.md) — doc track for the
  Security / Governance buyer.
- [`playbooks/pilot-platform-engineering.md`](./playbooks/pilot-platform-engineering.md)
  — 30/60/90-day pilot plan tuned for a Platform Engineering champion.
- [`playbooks/pilot-security.md`](./playbooks/pilot-security.md) — pilot plan
  tuned for a Security or GRC champion, anchored in audit evidence.
- [`playbooks/objection-handling.md`](./playbooks/objection-handling.md) —
  the seven objections we hear most often, with the answer and the proof.

## Maintained metrics for GTM health

Activation and AI-trust telemetry shipped in `internal/telemetry/activation.go`
gives us the metrics this wedge depends on:

| GTM metric                   | Underlying instrument                          |
|------------------------------|------------------------------------------------|
| Time-to-first-success        | `specular.activation.duration{milestone="first_success"}` |
| Setup drop-off               | `specular.activation.step{status="abandoned"}` |
| Intervention rate            | `specular.ai_trust.intervention`               |
| Routing explainability       | `specular.ai_trust.routing_decision`           |
| Cost-band distribution       | `specular.ai_trust.routing_cost_estimate`      |

If a pilot does not move these, the wedge is not landing — go fix the wedge,
not the dashboards.
