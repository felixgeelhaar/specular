# Policy-enforced AI development in CI/CD

This is the long-form version of the Specular wedge. Read this if you are
preparing a launch deck, briefing an analyst, or writing a pitch.

## The shift

For most of the last decade, the unit of governance inside engineering orgs
was the **pull request**. Reviewers caught bad changes, CI ran tests, and
release engineers cut builds. The model worked because humans wrote the code.

AI assistants break that model in three ways:

1. **Authorship is no longer human.** A PR may contain code authored by
   Claude, Codex, Gemini, a local Ollama model, or any combination — chosen
   by a router the reviewer never sees.
2. **Volume scales faster than review.** A team of ten engineers can now
   generate the diff volume of a team of fifty. PR review is the new
   bottleneck.
3. **Reasoning is invisible.** Why a model picked a particular approach,
   what alternatives it considered, and what cost was paid for the answer
   are nowhere in the diff.

Boards, regulators, and CISOs are asking a question the existing toolchain
cannot answer: **"For every AI-touched change shipping to production, can
you show me what was generated, by which model, against which policy, and
who approved it?"**

## What Specular replaces, and what it does not

### Replaces

- **Bespoke policy scripts** that engineering teams hand-roll inside CI to
  block unreviewed AI output (custom GitHub Actions, ad-hoc Lua, OPA glue).
- **Implicit trust** in vendor IDE plugins to enforce org guardrails.
- **Spreadsheet-based AI governance** (e.g. tracking "approved models" in
  Confluence with no enforcement).

### Does not replace

- **Code review.** Specular adds a gate; humans still review.
- **Your AI assistant.** Specular does not generate code from scratch. It
  routes, governs, and audits whatever model your engineers already use.
- **Your CI provider.** Specular runs inside GitHub Actions, GitLab CI,
  Jenkins, Buildkite, or any runner that can shell out.
- **Your SAST/SCA stack.** Specular complements Snyk, Sonar, Semgrep — it
  does not duplicate them.

## What Specular uniquely does

Three capabilities are core to the wedge.

### 1. Auditable drift gates

`specular eval drift` produces a deterministic hash of the spec, plan, and
configuration that drove a generated change. The hash is signed, timestamped,
and stored under `.specular/approvals/`. Any subsequent change that cannot
re-derive the same hash from the same inputs **fails the gate** until a
named human approves the new state.

This turns "the AI did something" into "the AI did exactly this, against
this approved baseline, and here is the evidence" — the kind of evidence
SOC 2 and ISO 42001 auditors recognize.

### 2. Routing explainability

Every model selection emits a `specular.ai_trust.routing_decision` metric
with `provider`, `model`, `hint`, `reason`, and `cost_band` attributes. The
human-readable reason ("matched hint: codegen, budget-optimized") is part
of the audit record. When a regulator asks why a particular change went to
a particular model, the answer is on the dashboard, not in tribal memory.

### 3. Approval bundles with chain of custody

`specular bundle create` packages the spec, plan, generated artifacts,
policy results, and routing decisions into a single signed bundle. Approval
is a `specular approve bundle-<id>` against a TTY user, with the approver,
timestamp, message, and content hash captured under YAML in
`.specular/approvals/`. This is what ships in the audit packet.

## How a Specular-governed change actually flows

```
┌────────────┐   spec      ┌────────────┐    plan     ┌────────────┐
│  Engineer  │────────────▶│  specular  │────────────▶│   Router   │
│ (or agent) │             │   spec gen │             │ (model+cost│
└────────────┘             └────────────┘             │  decision) │
                                                      └─────┬──────┘
                                                            │
                                                            ▼
                                                      ┌────────────┐
                                                      │   Build    │
                                                      │  (assists  │
                                                      │  by model) │
                                                      └─────┬──────┘
                                                            │
                                                            ▼
        ┌─────────────────────────────────────────────────────┐
        │  CI/CD: specular eval drift  →  policy check        │
        │  →  bundle create  →  approval gate                 │
        └─────────────────────────────────────────────────────┘
                                                            │
                                                            ▼
                                                       ┌────────┐
                                                       │ Deploy │
                                                       └────────┘
```

The key property: the governed surface is **the gate, not the IDE**. We do
not care whether the engineer used Cursor, Zed, vim, or pure Claude — by
the time a change reaches the pipeline, drift, routing, and approval are
enforced regardless of authoring path.

## Anti-positioning

There are several adjacent categories Specular is **not** in. Saying so
plainly is part of the wedge.

| Adjacent category            | Why we are not it                                  |
|------------------------------|----------------------------------------------------|
| AI IDE assistant             | We do not author code; we govern it.               |
| Generic policy-as-code (OPA) | We are AI-development-aware; OPA is not.           |
| LLM observability platform   | We instrument the dev workflow, not the model API. |
| Code-review SaaS             | Humans still review; we add the gate.              |
| MLOps platform               | Specular governs the dev → production path of code, not model training. |

## What compounds (the moat thesis)

The wedge needs a moat that strengthens with adoption, not a feature
that depreciates. Three candidates were considered; one is being
invested in, the other two are tailwinds.

**Primary moat: the policy library.** Every pilot org contributes
SOC 2 / ISO 42001 / EU AI Act control mappings as `policies.yaml`
fragments back to the open repository. The first contributor pays the
research cost; every subsequent adopter inherits a battle-tested
checklist. After ~10 orgs, the library is a credible reference; after
~50, no Platform team will start from scratch. **This is a data network
effect anchored in regulated content** — adoption increases the
quality of the asset, which increases the rate of new adoption. The
tax to fork the library is the same research cost the first
contributor paid; the tax to switch *away* once you've adopted it is
proportional to the number of controls you've mapped against it.

**Secondary tailwinds (real but not load-bearing):**

- **Auditor relationships.** Big 4 acceptance of the bundle YAML schema
  as audit evidence reduces friction at every audit cycle. Worth
  investing in (see `distribution.md`) but not a moat — auditors will
  accept any well-formed evidence.
- **Approver identity graph.** Orgs that accumulate years of
  `.specular/approvals/` records develop migration friction; this
  compounds slowly and is a retention signal, not an acquisition moat.

**What we are explicitly not betting on:** bundle schema lock-in
(low — schema is open and intentionally portable), execution-engine
performance (low — inner-loop authoring still happens in the IDE),
brand recognition (lagging the wedge by 18+ months in 2026).

## Proof points to lead with

When briefing a buyer, the three artifacts that move the conversation:

1. **A live run of `specular eval drift`** showing a baseline hash, a
   tampered change failing the gate, and a re-approval restoring the gate.
2. **A Grafana panel** of `specular.ai_trust.routing_decision` grouped by
   model, hint, and cost_band — the buyer sees AI behavior they did not
   know was knowable.
3. **A real `.specular/approvals/bundle-*.yaml` file** showing the chain
   of custody. This is the artifact a CISO will copy into their evidence
   packet on the spot.

Lead with these three. Talking points come second.
