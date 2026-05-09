
## Guided onboarding and activation flow

Create a guided `specular init` setup flow that auto-detects providers, writes secure defaults, and gets users to first successful governed workflow without manual YAML editing. Include progressive disclosure for advanced settings.

---

## Security hardening for execution paths

Harden execution security by validating subprocess arguments, tightening sensitive file permissions, and resolving high-priority security scan findings in command execution and config write paths.

---

## Provider reliability and coverage uplift

Increase provider/routing reliability through targeted tests and coverage improvements for provider, router, and config helpers until provider domain meets policy thresholds.

---

## Activation and AI trust telemetry

Define and instrument activation and AI trust metrics, including time-to-first-success, setup drop-off, regenerate rate, and intervention rate, and expose explainability signals for model routing decisions.

---

## GTM wedge and role-based launch assets

Package a focused GTM wedge around policy-enforced AI development in CI/CD with auditable drift gates. Deliver role-based documentation tracks and launch playbooks for platform engineering and security buyers.

---

## Telemetry hardening for activation and AI trust pipeline

Resolve the critical engineering and quality gaps in the M9.4 activation/AI-trust telemetry. Bound routing-decision metric cardinality (drop free-form reason from counter attrs, validate hint against allowlist, move full sentence to a span event/log). Make the .specular/.activation.json marker write atomic via temp+rename and guard the read-modify-write with a lockfile or sentinel so concurrent CLI invocations cannot double-emit first_success or corrupt JSON. Replace metricsOnce with a mutex-guarded re-entrant init that allows recovery from partial init failures, and surface telemetry init status via specular doctor. Add a Version field to the activation marker schema with a documented migration path. Extract activation into its own bounded-context package (internal/activation/) so milestone semantics, marker persistence, and the funnel vocabulary stop leaking across cmd and telemetry. Bound findSpecularDir at the user home boundary and require a sibling marker (config.yaml) to confirm ownership. Add caller-side integration tests for runInit, router SelectModel, auto approval gate, and approve subcommand that drive each call site against a ManualReader and assert the full attribute contract. Add a TestActivationMetricContract golden test that fails on unknown attribute keys/values. Add a concurrency test for recordFirstSuccessIfPending using -race. Add a property-based test for CostBand monotonicity using pgregory.net/rapid.

---

## AI trust signal depth and bundle provenance

Close the AI-product gaps in the just-shipped trust telemetry. Add a specular.ai_trust.safety_event counter with attributes (category, severity, action_taken) covering prompt_injection, secret_leak, forbidden_tool_call, scope_violation, refusal, jailbreak_attempt — wired into the existing hooks, policy checkers, and executor sandbox so off-policy model behavior is visible mid-build, not only post-hoc. Wire RecordRegenerate into real flows (auto retry, eval-gate failure, drift-revert) with a trigger enum (user_reject, eval_failure, agent_self_correct, drift_revert, policy_block) and a previous_model attribute for studying escalation. Split first_success into first_command_success (CLI-ergonomics, retained) and first_wedge_success (fired only after auto/build/eval drift/bundle create succeeds) so time-to-first-wedge-success becomes the headline activation metric. Extend cost-band cutoffs to add 10-100 and >=100 bands so frontier-model spend is resolvable on dashboards. Extend intervention coverage with ide_edit_detected, auto_rollback, and bundle_abandoned gates plus an implicit_reject decision so the intervention rate is not systematically under-counted. Emit a SelectionTrace OTel event capturing the chosen model, score, top-3 considered candidates with reject reasons, hint applied, and routing inputs — wired into the bundle YAML so chain-of-custody answers "which model wrote this artifact" by adding a Provenance struct to ArtifactInfo. Add a specular.evidence.retrieval_duration histogram emitted by bundle show and approve list --since so the auditor JTBD has a measurable outcome.

---

## GTM wedge sharpening and category claim

Sharpen the M9.5 GTM wedge in response to Dunford-grade positioning critique. Rewrite the docs/gtm/README.md wedge sentence around a single primitive (drift gate) and add a "What you'd otherwise do" block naming the buyer's status-quo alternative (hand-rolled GitHub Action + OPA glue + Confluence approved-models page). Stake an explicit category claim — "AI Change Control" — with a two-sentence definition that anchors the buyer mental model to CAB/SOX change-management rather than the crowded "AI governance" swamp; thread the phrase through every persona doc and the README. Make the ICP actionable for outreach by adding named-account triggers (public SOC 2 mentioning AI, AppSec/AI-risk job postings in last 90d, Cursor/Copilot Enterprise adoption signals, ISO 42001 announcements) and a target-list spec covering fintech, digital health, FedRAMP-adjacent public-sector contractors, and EU banks under DORA. Strengthen the multi-threaded sale by adding "Why your counterpart needs you" blocks to each persona doc, with explicit cross-references and the specific ask each persona owes the other. Reframe pilot success criteria as buyer-owned outcomes (review-time delta, evidence-collection hours saved) with the Specular metric as the instrument; demote install-rate metrics to leading indicators. Add docs/gtm/distribution.md with three asymmetric channels (auditor enablement / Big 4 control mapping packet, compliance-influencer co-marketing on ISO 42001 + EU AI Act, and a public bundle gallery indexed by framework). Add objections #8 (vendor governance from the IDE/AI vendor) and #9 (agent-authored / non-PR changes). Add a "What compounds" section to ci-cd-policy-enforcement.md naming a single moat thesis (policy-library network effect, auditor relationships, or approver identity graph). Promote the GTM link to README quick-link position #2 retitled "Why Specular: AI Change Control for regulated orgs."

---

## Activation UX polish and user-facing telemetry transparency

Close the user-facing UX gaps that today block the activation funnel from feeling coherent. Emit step labels matching the telemetry events as runInit fires them ("Detecting project context… ok (3 providers)", "Writing configuration… ok") so the user mental model maps to the Setup-drop-off metric a buyer is asked to trust. After successful executeInit, print a 3-line numbered next-steps block (specular spec add, plan generate, copy CI yaml from docs/gtm/personas/platform-engineering.md) so the highest-friction moment in the journey — between init finishing and first green CI gate — gets explicit scaffolding. Document the .specular/.activation.json marker in docs/getting-started.md under a "What Specular writes locally" subsection and gate the marker behind the same SPECULAR_TELEMETRY env so security buyers do not encounter a hidden timing file. Improve the auto-mode approval bubbletea TUI: handle Esc (cancel without rejection), Enter (default Yes), and ? (help); show a footer hint "y approve · n reject · esc cancel · ? help"; differentiate cancelled from rejected in the caller. Replace color-only priority and decision signaling with text tokens ([P0:CRIT], [P1:HIGH], [P2:LOW], [OK]/[X]) that respect NO_COLOR. Invert docs/gtm/README.md so a buyer hits a "I am Platform / I am Security / I am evaluating" choice in the first 250 words instead of internal framing. Swap the README quick-link emoji from 🎯 to a doc-shaped glyph for Jakob's Law consistency, and add a "Buyer evaluators: skip to Platform Engineering pilot" callout in docs/getting-started.md's first 20 lines. Run a link checker over docs/ and repair broken cross-references between persona docs and tutorials/PRODUCTION_GUIDE/AUTHORIZATION_GUIDE.

---
