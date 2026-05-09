# Objection handling

The seven objections we hear most often, with the answer that has actually
worked, and the proof we lean on. Steal these for your own briefing notes.

---

## 1. "We already have OPA / Conftest / Sentinel for policy."

**The objection.** "Policy-as-code is a solved problem. Why do we need
another layer?"

**The answer.** OPA is unaware of AI development. It does not know which
model authored a change, which router decision was made, what the cost
was, or what spec was claimed to drive it. Specular is the
**AI-development-aware policy and evidence layer** that **uses** OPA-style
policies, it does not replace them.

**Proof.** Show `policies.yaml` integrating with their existing OPA
bundles via the `external` checker. Show the bundle YAML — the OPA result
is a field inside it.

---

## 2. "Won't this slow our pipeline down?"

**The objection.** "We already pay for CI minutes; another step is a tax."

**The answer.** The drift gate is content-hash based. On a clean tree it
runs in well under 5 seconds. On a tree with no AI-relevant changes it is
a no-op. The bundle-create step runs only on merge candidates, not every
commit.

**Proof.** Run `time specular eval drift` in their pilot repo. Numbers
land below the noise floor of any non-trivial CI step. The cost
conversation ends here.

---

## 3. "Our auditors don't ask about AI yet."

**The objection.** "We're not ready for an AI control framework."

**The answer.** Three things: (1) Your auditors **will** ask within the
next assessment cycle — every Big 4 firm has updated their AI questionnaire
in 2025–2026. (2) The bundle artifact is also useful for the
non-AI parts of CC8.1 / change management today. (3) Showing up to the
next audit with the artifacts already in place is dramatically cheaper
than retrofitting.

**Proof.** Forward the relevant 2026 update from your audit firm's
engineering blog. Show the bundle as a generic change-management artifact,
not just an AI one.

---

## 4. "We don't trust open source for the governance layer."

**The objection.** "The governance plane has to be a vendor we can sue."

**The answer.** Three responses, escalating: (1) Specular is Apache 2.0
and the bundle format is a portable YAML schema — there is no lock-in.
(2) The artifacts (bundles, approvals, drift hashes) live inside your
repo, not on someone else's server; even if Specular disappeared, the
evidence does not. (3) For orgs that contractually require a vendor, the
commercial entity behind Specular offers a paid support agreement against
the same binary.

**Proof.** Show `.specular/approvals/*.yaml` — these are your files, in
your repo, owned by you. Show the bundle schema in the docs.

---

## 5. "Our developers will hate another gate."

**The objection.** "Engineering velocity is non-negotiable."

**The answer.** The gate trips when AI-authored changes do not match the
recorded plan. For human-authored changes, it is invisible. For
AI-authored changes that follow the plan, it is also invisible. It only
costs developer time when an AI change goes off-plan, and that is exactly
the moment a human should be looking.

**Proof.** Pull the metric `specular.ai_trust.intervention{decision="approved"}`
divided by total AI-authored PRs after week 4 of a pilot. The number is
typically >0.9 — i.e. >90% of AI changes flow through cleanly.
Developers do not "hate" a gate they almost never see.

---

## 6. "We've already standardised on one model — we don't need a router."

**The objection.** "We use Claude. End of story."

**The answer.** Two parts: (1) "Standardised on one model" almost always
means "standardised in the IDE." Local agents, evaluation tools, build
helpers, and pre-commit hooks routinely call other providers without
anyone noticing. The router is the place where that becomes visible. (2)
Even with one provider, model selection within that provider (Sonnet vs
Opus vs Haiku) is a cost and capability decision your dashboards do not
currently see.

**Proof.** Run `specular.ai_trust.routing_decision` against any active
repo for a week. The distribution is almost never what the buyer expected.

---

## 7. "We'll build this ourselves."

**The objection.** "Our platform team can wire this up in a sprint."

**The answer.** Four pieces are non-trivial to get right: (1) deterministic
content hashing for drift in a tree with generated files; (2) a
policy-evaluation contract that is auditable; (3) a bundle schema that
survives auditor review without bespoke explanation; (4) the OTel
instrumentation that maps cleanly onto AI controls. Most home-grown
versions of this take a quarter to ship and another quarter to harden.
Specular is the result of that work, in a single binary.

**Proof.** Show the bundle YAML schema, the activation/AI-trust metric
list, and the policies.yaml structure. Then ask the buyer how long their
team would take to ship the equivalent. The answer is rarely "a sprint"
once they see the surface.

---

## What to do when the objection is real

Sometimes the buyer is right. The two cases worth recognising:

- **They are below the ICP threshold.** Fewer than 50 engineers, no
  Platform team, no Security org. The wedge will not land. Recommend
  they self-serve the open source release, come back when they cross
  the threshold, and stop selling.

- **They want a feature we do not have.** Common asks:
  signed S3 evidence export, GitHub Enterprise Server SAML approver
  attribution, Splunk-native dashboards. Note them, file an issue, and
  be honest about timing. Buyers respect "Q3" more than they respect
  "soon."

The wedge works because it is narrow. Trying to win every deal is the
fastest way to lose it.
