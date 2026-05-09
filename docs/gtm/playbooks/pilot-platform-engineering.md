# Platform Engineering pilot playbook

Use this when a Platform / DevEx leader has agreed to evaluate Specular and
you are scoping the pilot. The plan is 90 days. Most pilots that succeed
finish a working drift gate in week 2 and spend the rest of the time
tuning policies and rolling out to additional teams.

## Pilot success criteria

State these in writing **before** day 1. Leaders need a contract.

- One repository has `specular eval drift` running as a required CI check
  by end of week 2.
- ≥80% of AI-authored PRs in the pilot repo have a recorded routing
  decision and bundle by end of week 4.
- Time-to-first-success (`specular.activation.duration{milestone="first_success"}`)
  for new developers in the pilot repo ≤ 30 minutes by end of week 6.
- No P1 incident attributable to the gate by end of week 12.

If three of these four hold, the pilot is a success and the contract is to
expand to 3 additional repos in the next quarter.

## 30-day plan: Foundation

**Week 1 — Install and instrument.**

- Pick the pilot repo. Constraints: ~10–30 active engineers, owned by
  one team, has CI green today, ships at least weekly.
- Install Specular on developer machines (`brew install` or equivalent).
- Run `specular init --governance L2 --yes` in the pilot repo.
- Wire `specular eval drift` and `specular bundle create` into the CI
  pipeline behind a feature flag (advisory, not blocking).
- Stand up the OTLP collector or point Specular at the org's existing one
  (`SPECULAR_TELEMETRY=on`, `SPECULAR_TELEMETRY_ENDPOINT=...`).

**Week 2 — Make the gate blocking.**

- Flip the drift check from advisory to required.
- Add a `specular approve --check` step that fails CI until the bundle
  is approved.
- Document the approve flow in the pilot team's runbook (5 minutes of
  copy-paste).
- Demo the failing gate to the team in a 30-minute session. People need to
  see the red CI to internalise the model.

**Week 3 — Tune policies.**

- Review the first 50 routing decisions in Grafana / your SIEM. Look for:
  - Models you do not want chosen → block via `policies.yaml`.
  - Cost bands above expectation → tighten budget.
  - Hints that always route to the wrong model → adjust router config.
- Edit `policies.yaml`, run `specular policy diff`, get an approval
  recorded against the policy change itself.

**Week 4 — Lock in measurement.**

- Build a dashboard with three panels:
  1. Activation funnel by step + status.
  2. Routing decisions by provider / model / cost_band.
  3. Intervention rate by gate type + decision.
- Schedule a 30-minute weekly review with the pilot team for the rest of
  the pilot. Cancel it the moment it stops surfacing decisions.

## 60-day plan: Hardening

**Week 5–6 — Surface the activation pain.**

- Pull `specular.activation.step{status="abandoned"}` for the pilot repo.
  Where are people dropping off?
- Run a small ergonomics sprint: shorter `init`, better default policies,
  fewer warnings.
- Confirm time-to-first-success ≤ 30 minutes.

**Week 7–8 — Expand within the team.**

- Add pre-commit hooks for fast feedback on the same drift baseline.
- Move the bundle create step earlier so engineers see the artifact during
  PR creation, not only on the merge gate.
- Start tracking the **review-time delta** between AI-authored and
  human-authored PRs. If reviewers are not faster on the AI ones with the
  gate green, the bundle is not communicating enough; iterate on the
  bundle template.

## 90-day plan: Expansion

**Week 9–10 — Pick repo #2.**

- Choose a repo with **different shape** from the pilot — different
  language, different team, different release cadence. Most rollouts
  succeed in the first repo and fail in the second because the gate was
  tuned to a single team's workflow. Catching that here is the point.

**Week 11–12 — Decision gate.**

- Run the success criteria check above. If you have 3/4, get the platform
  leader and the AppSec lead in a room to commit to the next 5 repos.
- Hand off the runbook, dashboards, and policies to the platform team as
  golden-path artifacts.

## Operational guardrails

A few things that have broken pilots historically:

- **Do not start with `--governance L4`.** L2 is the floor for a useful
  pilot; L4 is for high-regulation deploys. Going too strict early causes
  the team to disable the gate.
- **Do not skip the OTel wiring.** Without metrics, the conversation in
  week 8 is opinion vs opinion. With metrics, it is data.
- **Do not let the pilot run silently.** A weekly 30-minute review with
  the lead engineer and the platform owner is the unit of forward motion.
- **Do not roll out to >1 repo before week 9.** Resist this even if
  enthusiasm is high.

## What to escalate to us

- Drift gate flapping (false positives) for the same input on consecutive
  runs — this is a Specular bug, not a config issue.
- Time-to-first-success above 60 minutes after week 6 — likely a setup
  flow gap and we want to fix it upstream.
- Routing reasons that appear non-deterministic — this is part of the
  trust surface and we treat it as a P1.

Open an issue at <https://github.com/felixgeelhaar/specular/issues> with
the `pilot` label, or ping the maintainers directly.
