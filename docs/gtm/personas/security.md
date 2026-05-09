# Security / GRC track

> If your job is to defend, audit, or attest the AI-touched parts of your
> codebase, this page is the entry point. The wedge for you is **evidence**:
> Specular turns invisible AI behavior into signed, reviewable artifacts.

## Who this is for

- AppSec lead, CISO staff, or GRC analyst owning AI-related compliance.
- Has been asked at least one of these questions in the last 90 days:
  - "Which AI models are touching our production code?"
  - "Can we prove a specific change was reviewed before it shipped?"
  - "How do we evidence AI governance for SOC 2 / ISO 42001 / EU AI Act?"
- Already runs a SAST/SCA stack and is comfortable reading YAML and signed
  digests.

## What Specular gives you

A defensible answer to the three audit questions above, in three artifacts.

### 1. Signed approval bundles — `.specular/approvals/bundle-*.yaml`

Every deployable change can be packaged with `specular bundle create` into a
signed bundle that contains:

- The product spec (`spec.yaml`) and its content hash.
- The plan (`plan.yaml`) and the routing decisions for every step.
- Policy results (`policies.yaml` evaluation per step).
- Approval records (who approved, when, with what message).

Approval is created with `specular approve bundle-<id> --message "..."` and
recorded under `.specular/approvals/`. The format is YAML, intentionally
diff-friendly so reviewers can read it without a UI.

### 2. Drift gates that fail CI on unauthorised change

`specular eval drift --fail-on-change` re-derives the bundle hash from the
current tree and the recorded baseline. If they diverge, CI fails. The only
way back to green is `specular approve drift-<hash> --message "..."` from a
TTY user — the same control plane your existing PR review depends on.

This is what auditors look for: **a deterministic, replayable gate with a
named approver and a timestamp**, not "we trust the engineers to do the right
thing."

### 3. AI behavior telemetry

Specular emits OpenTelemetry metrics over OTLP HTTP that you can pipe into
the observability stack you already operate:

- `specular.ai_trust.routing_decision` — provider, model, hint, reason,
  cost_band per selection. Shows which models are being chosen and why.
- `specular.ai_trust.intervention` — every approval-gate decision, keyed
  by gate type and decision (`approved` / `rejected`).
- `specular.ai_trust.routing_cost_estimate` — USD distribution of decisions.
- `specular.activation.*` — activation funnel and time-to-first-success.

For SIEM correlation, the same events are also emitted as OTel spans with
`component=cli` and `component=provider`.

## Mapping to common frameworks

The artifacts above map cleanly onto the controls auditors actually ask for.

| Framework            | Control area                                   | Specular evidence                               |
|----------------------|------------------------------------------------|-------------------------------------------------|
| SOC 2 (CC8.1)        | Change management                              | Approval bundles + drift gates                  |
| ISO/IEC 42001        | AI management system: change control           | Bundle chain of custody + routing decisions     |
| EU AI Act (Art. 17)  | Quality management for high-risk systems       | Replayable plans + signed approvals             |
| NIST AI RMF (Govern 4) | AI risk monitoring                           | Routing + intervention metrics                  |
| PCI DSS (6.4.5)      | Approval of significant changes                | `specular approve` records under SCM            |

We do not claim Specular makes you compliant on its own. We claim it produces
the **evidence** the framework asks for, in a form you can copy into the
audit packet without manual reconstruction.

## The 30-minute audit walkthrough

```bash
# 1. Read an existing approval bundle.
ls .specular/approvals/
cat .specular/approvals/bundle-<id>.yaml

# 2. Re-derive the drift baseline and confirm the gate.
specular eval drift

# 3. Tamper with the codebase, watch the gate fail.
echo "// drift" >> internal/cmd/init.go
specular eval drift --fail-on-change

# 4. Inspect AI behavior on the dashboard.
#    (open Grafana → search specular.ai_trust.*)
```

After step 4, you have seen the entire control surface end to end. Most
security buyers stop the demo here and start asking who else uses it.

## What Specular does not do

To save us both time, here is what is **out of scope**:

- We do not classify AI output by sensitivity (no model-output DLP).
- We do not run prompts through a content filter.
- We do not block specific models — but we record which model ran, so you
  can write a policy that does.
- We do not host an LLM. Bring your own provider keys.

## Where to go next

- [`../playbooks/pilot-security.md`](../playbooks/pilot-security.md) — pilot
  plan tuned for a Security or GRC champion.
- [`../playbooks/objection-handling.md`](../playbooks/objection-handling.md) —
  pushback we have heard and how we answer it.
- [`../../tutorials/07-approvals.md`](../../tutorials/07-approvals.md) —
  hands-on approval flow.
- [`../../AUTHORIZATION_GUIDE.md`](../../AUTHORIZATION_GUIDE.md) — AuthN/AuthZ
  configuration for organizations with SSO/SAML.
- [`../../../SECURITY.md`](../../../SECURITY.md) — Specular security posture
  and disclosure process.
