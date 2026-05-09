# Security / GRC pilot playbook

Use this when a Security, AppSec, or GRC leader is the named champion. The
shape is different from a Platform-led pilot: the deliverable is **audit
evidence**, not developer ergonomics, and the scope is narrower.

## Pilot success criteria

State these in writing before day 1.

- One control from your active framework (SOC 2 CC8.1, ISO 42001, EU AI
  Act Art. 17, or equivalent) is mapped to a Specular artifact and
  validated by an internal auditor.
- At least one signed `bundle-*.yaml` is added to the audit evidence
  packet for a real release.
- Drift gate runs in CI for ≥1 production-bound repo within 30 days.
- Time from "auditor asks about an AI-touched change" to "evidence
  delivered" drops by ≥50% compared to the manual baseline.

The pilot is small on purpose. Security pilots fail when they try to do
platform work in disguise. Stay narrow.

## 30-day plan: Evidence

**Week 1 — Map controls to artifacts.**

- Pick **one** framework you are actively assessed against.
- Pick **one** control inside it that touches AI development.
- Write a one-page mapping (Markdown or Confluence) that says:
  - Control text → Specular artifact (bundle / approval record / metric)
  - Where the artifact lives in the repo
  - How to retrieve it on demand
- Get the internal auditor to sign off on the mapping in principle. This
  is the single most important step. Without auditor buy-in, the rest of
  the pilot is theater.

**Week 2 — Stand up evidence flow.**

- Pick a production-bound repo with a Platform Engineering champion you
  trust. (The Security pilot fails alone — partner with PE here.)
- Wire `specular eval drift` + `specular bundle create` into CI. Start
  advisory.
- Capture three real bundles from three real merges into a Markdown file
  with annotations: what changed, what model, what cost.

**Week 3 — Force the audit drill.**

- Pick a recent merge and ask your auditor (or a peer playing the role):
  "Show me the evidence that this AI-touched change was reviewed and
  approved." Time the response.
- Repeat with the same merge but using the Specular bundle as the answer.
  Time again. The delta is your headline number.

**Week 4 — Make the gate blocking.**

- Flip drift from advisory to required.
- Add `specular approve --check` so the gate cannot be bypassed by
  re-running CI.
- Document the approver list (who can `specular approve` what) under
  `.specular/approvals/` and reference it from your access-control
  baseline.

## 60-day plan: Coverage

**Week 5–6 — Telemetry into the SIEM.**

- Configure `SPECULAR_TELEMETRY=on` and point the OTLP endpoint at the
  collector your SIEM ingests from.
- Build three saved searches / dashboards in your SIEM:
  1. Approvals by approver, by gate type, by week.
  2. Routing decisions by provider and model.
  3. Failed drift gates (the security-relevant signal).
- Wire alert: "drift gate failure on `main` not approved within 24h"
  routes to the AppSec rotation.

**Week 7–8 — Tabletop the bypass.**

- Run a tabletop exercise: an engineer attempts to bypass the gate. Walk
  through what they would have to do, what the bundle would look like
  after, and what evidence the auditor would see.
- Document the residual risks. Add them to the risk register; this
  becomes part of the audit narrative ("here is what the control covers,
  here is what it does not, here is how we mitigate the gap").

## 90-day plan: Renewal-ready

**Week 9–12 — Map the rest of the framework.**

- Extend the week-1 mapping document to cover every AI-related control
  in the chosen framework.
- For each control, list the Specular artifact (or "no, requires manual
  evidence — process is X").
- Get auditor sign-off on the full mapping document.
- Hand the document to the GRC team as the canonical AI-control mapping
  for the next assessment cycle.

## Specific framework hooks

Some shortcuts the security team can lean on.

- **SOC 2 CC8.1** — point auditors at `.specular/approvals/` for the
  population of approvals; sample with `git log --grep specular`.
- **ISO/IEC 42001 Clause 8** — point at the bundle YAML schema as the
  AI change-control artifact.
- **EU AI Act Article 17** — the bundle replays the spec → plan → build
  chain and is the documentation control evidence.
- **NIST AI RMF Govern 4.1** — the routing decision metric is the AI-system
  monitoring control.

If your framework is not in this list, write the mapping yourself in
week 1; the auditor will copy your text.

## What to escalate to us

- Any artifact you need that does not exist yet (signed S3 evidence
  exports, SARIF mapping, etc.) — these are exactly the integrations
  we want to ship next.
- Auditor pushback on the evidence quality. We treat that as a P0; if
  the bundles do not survive an auditor, the wedge does not work.
- Cross-framework duplication you find. We want to absorb the mapping
  into the docs so the next pilot does not repeat the work.

Open an issue at <https://github.com/felixgeelhaar/specular/issues> with
the `pilot` and `compliance` labels.
