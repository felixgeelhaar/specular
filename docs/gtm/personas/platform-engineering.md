# Platform Engineering / DevEx track

> If you own the inner-loop and inner-CI experience for your engineers, this
> page is the entry point. It is intentionally short — most of the value
> shows up the first time you wire `specular eval drift` into a pipeline.

## Who this is for

- Director or staff-level engineer running a Platform / DevEx / DevOps team.
- Owns golden-path tooling, CI/CD, the `Makefile` story, and the answer to
  "how do we ship safely at this org?"
- Has 1–5 engineers actively asking for AI tooling guardrails this quarter.

## What you actually get

| You already have                          | Specular adds                                     |
|-------------------------------------------|---------------------------------------------------|
| GitHub Actions / GitLab CI / Jenkins      | A drop-in CI step that gates AI-authored changes. |
| OPA, Conftest, Sentinel, custom policies  | AI-development-aware policies via `policies.yaml`.|
| Snyk / Sonar / Semgrep                    | Coexists; we govern, they scan.                   |
| Cursor / Continue / Cline / pure Claude   | Coexists; we govern at the gate, not the editor.  |
| Internal docs telling people "use approved models" | Enforcement plus a metric that proves it.   |

## The 30-minute walkthrough

The fastest way to evaluate Specular against your platform.

```bash
# 1. Install (one binary, no daemon).
brew install felixgeelhaar/tap/specular   # or: see docs/installation.md

# 2. Initialise inside an existing repo.
specular init --governance L2

# 3. Generate a small change against a real spec.
specular spec add "Add /healthz endpoint that returns build SHA"
specular plan generate
specular build run --dry-run

# 4. Wire the gate into CI (one job).
cat .github/workflows/specular.yml
# - run: specular eval drift --fail-on-change
# - run: specular bundle create
# - run: specular approve --check     # fails until a human approves
```

After step 4 you have:

- A reproducible spec → plan → build chain.
- A drift gate that fails CI on unauthorised AI-driven changes.
- A bundle artifact your release team can attach to deploys.

## Where it slots into the platform

```
Engineer
  │
  ▼
[ IDE / Cursor / Claude Code ]   ← we do nothing here, by design
  │
  ▼
[ Pull Request ]
  │
  ▼
[ CI: lint • test • SAST • specular drift+bundle+approve ]   ← the wedge
  │
  ▼
[ CD: deploy on green + signed bundle ]
```

The shape we keep seeing: **add Specular as a single CI job, then gradually
push policies and approvals upstream into pre-commit hooks** as developer
trust builds. Trying to do it in the opposite direction (start in the IDE,
then expand) tends to stall.

## Metrics you should expect to move

After ~6 weeks of pilot use across 1–2 teams:

| Metric                                  | Direction | Why it should move                           |
|-----------------------------------------|-----------|----------------------------------------------|
| Time-to-first-success (`specular.activation.duration`) | ↓ | Setup is one CLI + one CI job. |
| % AI-authored PRs with explicit model attribution | ↑ to ~100% | Routing decisions are recorded. |
| Mean review time for AI-authored PRs    | ↓         | Reviewers trust the diff because the gate ran.|
| Audit findings tagged "AI provenance"   | ↓         | The bundle is the evidence.                  |
| Cost per AI-authored change             | ↓         | Router prefers cheaper models when adequate. |

If after 8 weeks you cannot show movement on at least three of these, escalate
to us — the wedge is not landing in your environment and we want to know.

## Where to go next

- [`../playbooks/pilot-platform-engineering.md`](../playbooks/pilot-platform-engineering.md)
  — 30/60/90 day pilot plan for a Platform-led rollout.
- [`../playbooks/objection-handling.md`](../playbooks/objection-handling.md) —
  the seven objections we hear and the answers that work.
- [`../../tutorials/04-governance.md`](../../tutorials/04-governance.md) —
  hands-on governance walkthrough.
- [`../../PRODUCTION_GUIDE.md`](../../PRODUCTION_GUIDE.md) — production
  configuration, including telemetry endpoints.
