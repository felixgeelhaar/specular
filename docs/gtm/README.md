# Specular: AI Change Control for regulated engineering orgs

> **30-second buyer scan: pick your role and follow the link.**
>
> - **I run Platform Engineering / DevEx →** [`personas/platform-engineering.md`](./personas/platform-engineering.md) — the gate, the metrics that move, the 30-minute walkthrough.
> - **I run Security / GRC / AppSec →** [`personas/security.md`](./personas/security.md) — bundles as audit evidence, framework mapping, the SIEM hookup.
> - **I am evaluating for a pilot →** [`playbooks/pilot-platform-engineering.md`](./playbooks/pilot-platform-engineering.md) and [`playbooks/pilot-security.md`](./playbooks/pilot-security.md) — buyer-owned outcomes and 30/60/90 plans.
> - **I am preparing a briefing or analyst call →** [`ci-cd-policy-enforcement.md`](./ci-cd-policy-enforcement.md) — long-form positioning, anti-positioning table, moat thesis.
> - **I want to know which objections to expect →** [`playbooks/objection-handling.md`](./playbooks/objection-handling.md) — nine objections, answers, proof artefacts.

If none of the above fit, the framing material below is for **internal
champions and design partners** preparing to introduce Specular into a
larger engineering org.

## The wedge in one sentence

> **Specular is the auditable drift gate for AI-authored code in CI/CD —
> the single chokepoint where every AI-generated change is reviewed,
> approved, and recorded with chain of custody.**

The drift gate is the primitive. Everything else (routing explainability,
intervention metrics, signed bundles) is in service of it.

We are not the broadest AI coding tool. We are the **narrow, defensible
wedge between AI assistants and your existing CI/CD + governance stack**.

## What you'd otherwise do

Without Specular, the buyer's status quo is **a hand-rolled GitHub Action
plus OPA glue plus a Confluence page of approved models**. That alternative
exists in dozens of orgs already; it works on day one and breaks on day
ninety, the moment an auditor or a CISO asks for chain-of-custody evidence.
Specular is the productized version of that pattern, with the audit chain,
the routing explainability, and the cross-session activation telemetry
built in. We are racing the home-grown alternative, not the IDE assistants.

## Category: AI Change Control

We claim the **AI Change Control** category. Not "AI governance" — that
name is already a swamp populated by model-card vendors, prompt-firewalls,
and content filters. AI Change Control is the change-management primitive
applied to AI-authored diffs: the same body of practice that gave us
CAB approvals, SOX change tickets, and CMDB entries, ported to the world
where the diff was authored by Claude or Codex instead of an engineer.

Two-sentence definition for use in briefings, decks, and analyst calls:

> **AI Change Control:** The discipline of recording, approving, and
> auditing every AI-authored change to production systems with a
> deterministic chain of custody. Sits between the AI assistant and the
> deploy gate; complements but does not replace SAST, code review,
> or policy-as-code.

Thread the phrase through every persona doc, every playbook, every
external talk. If an analyst names this category before we do, we lose
the framing battle.

## What compounds (the moat thesis)

Specular's moat is **the policy library**: every pilot org contributes
SOC 2 / ISO 42001 / EU AI Act control mappings as `policies.yaml`
fragments back to the open repository. After ten orgs, the library is
the canonical reference; after fifty, no platform team will start from
scratch. This is a data network effect anchored in regulated content,
not a feature; it compounds with adoption, not with engineering effort.

Adjacent moats we are explicitly *not* counting on as primary: bundle
schema lock-in (low — schema is open), execution-engine performance
(low — inner-loop work happens in the IDE), brand recognition (medium
but lagging). Pick the policy library; everything else is a tailwind.

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

### Named-account triggers

A BDR or solutions engineer should be able to turn this list into a target
account by Monday. The triggers below are the public signals an ICP
account is *ready* — surfacing one or more of these in the last 90 days
is the qualifying event:

- **Public SOC 2 Type 2 report** that mentions AI development controls,
  generative-AI risk, or AI vendor management.
- **Public ISO/IEC 42001 announcement, attestation, or readiness statement**
  in the last 12 months.
- **Job postings for "AI governance," "AI risk," "AppSec — AI,"
  "MLSecOps," or "responsible AI lead"** in the last 90 days
  (LinkedIn search; the AppSec specialisation is the strongest signal).
- **Public adoption of Cursor Enterprise, GitHub Copilot Enterprise,
  Claude Code, Devin, or Spotify Xirp** at organization scale (press
  releases, earnings calls, conference talks, beta waitlists).
- **CISO or CTO mentioning AI change-control, AI auditability, or
  EU AI Act readiness** on a podcast, panel, or blog post.

### Target list spec

For prospecting, the four cuts that consistently match the wedge:

1. **Fintech / regulated finance** (banks, neobanks, payments, insurance) —
   200–5,000 engineers, public SOC 2, increasingly under DORA in the EU.
2. **Digital health** — Epic/Cerner-adjacent EHR vendors, telehealth
   platforms with HIPAA scope, life-sciences SaaS subject to GxP.
3. **Public-sector contractors with FedRAMP / DoD scope** — anyone
   shipping to a Moderate or High authorisation boundary.
4. **EU-domiciled banks and large EU SaaS** under the **DORA** ICT-risk
   regime that took effect January 2025; AI-system change control is an
   open question for every one of them.

Avoid: pure consumer tech without compliance scope, sub-50-engineer
startups regardless of vertical, agency / consultancy shops (they will
build, not buy).

## Deliverables in this directory

- [`ci-cd-policy-enforcement.md`](./ci-cd-policy-enforcement.md) — the wedge
  story in long form: what we replace, what we integrate with, what we
  uniquely do, and the moat thesis.
- [`personas/platform-engineering.md`](./personas/platform-engineering.md) —
  doc track for the Platform Engineering / DevEx buyer.
- [`personas/security.md`](./personas/security.md) — doc track for the
  Security / Governance buyer.
- [`playbooks/pilot-platform-engineering.md`](./playbooks/pilot-platform-engineering.md)
  — 30/60/90-day pilot plan tuned for a Platform Engineering champion.
- [`playbooks/pilot-security.md`](./playbooks/pilot-security.md) — pilot plan
  tuned for a Security or GRC champion, anchored in audit evidence.
- [`playbooks/objection-handling.md`](./playbooks/objection-handling.md) —
  the nine objections we hear most often, with the answer and the proof.
- [`distribution.md`](./distribution.md) — three asymmetric distribution
  channels (auditor enablement, compliance-influencer co-marketing, and
  the public bundle gallery).
- [`competitive/xirp.md`](./competitive/xirp.md) — competitive brief vs
  Spotify Xirp (both-loops thesis: CLI sessions + gate vs desktop grid).

## Maintained metrics for GTM health

Activation and AI-trust telemetry shipped in `internal/telemetry/activation.go`
gives us the metrics this wedge depends on. The headline activation metric
is **time-to-first-wedge-success** — the time from `specular init` start
to the first successful run of `auto`, `build`, `eval drift`, `bundle
create`, or `drift`. CLI-ergonomics metrics (`first_success`) are
secondary; only `first_wedge_success` measures realised wedge value.

| GTM metric                          | Underlying instrument                                                |
|-------------------------------------|----------------------------------------------------------------------|
| **Time-to-first-wedge-success**     | `specular.activation.duration{milestone="first_wedge_success"}`      |
| Time-to-first-success (CLI)         | `specular.activation.duration{milestone="first_success"}`            |
| Setup drop-off                      | `specular.activation.step{status="abandoned"}`                       |
| Intervention rate                   | `specular.ai_trust.intervention`                                     |
| Routing explainability              | `specular.ai_trust.routing_decision` (counter) + span events         |
| Cost-band distribution              | `specular.ai_trust.routing_cost_estimate`                            |
| AI safety events                    | `specular.ai_trust.safety_event`                                     |
| Regenerate trigger mix              | `specular.ai_trust.regenerate{trigger=...}`                          |

If a pilot does not move these, the wedge is not landing — go fix the wedge,
not the dashboards.
