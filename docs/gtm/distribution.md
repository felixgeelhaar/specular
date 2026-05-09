# Distribution

> Direct outbound + OSS pull is the default GTM motion every B2B dev-tool
> ships with. It is necessary and insufficient. This doc names three
> **asymmetric** distribution channels — moves where one unit of effort
> produces durable, compounding reach — and tracks the bets we are
> placing on each.

## The three asymmetric channels

### 1. Auditor enablement

**The thesis.** When a Big 4 audit firm accepts the Specular bundle as
**recognised audit evidence**, every regulated client of that firm
inherits the path of least resistance: ship the bundle, pass the audit.
The audit firm becomes a recommender we never compensate.

**What we ship.**

- A **Big-4-formatted control mapping packet** for SOC 2 (CC8.1 + the
  AI-specific 2026 update), ISO/IEC 42001 Clause 8, EU AI Act Article 17,
  and NIST AI RMF Govern 4. Each entry: control text → Specular artifact
  → sample evidence file.
- A **reference audit walkthrough** (markdown + 5-minute screencast)
  that an auditor can follow on a real `.specular/approvals/` directory
  to verify the chain of custody in under 30 minutes.
- A **bundle attestation API** (Q3) that auditors can call to verify
  signature, hash, and approver attribution programmatically — so they
  do not need to learn YAML.

**Investment.** One staff-engineer week per quarter to keep the mapping
current with framework updates. One technical-marketing week per quarter
to refresh the screencast.

**Lead measure.** Number of auditor seats with the packet bookmarked.
Goal: 5 by end of Q3 2026, 25 by end of FY26.

**Lag measure.** Number of net-new pilot accounts where "our auditor
recommended Specular" is the source of inbound. Goal: 10% of FY26 inbound.

### 2. Compliance-influencer co-marketing

**The thesis.** ISO 42001 and the EU AI Act are 2025–2026 attention
attractors. The independent consultancies and influencers who write the
"how to comply" content are starved for **specific, demoable artifacts**.
Specular's bundle YAML is one. Co-authored pieces (we ghost-write or
bring expert engineering, they bring distribution and credibility)
compound over evergreen search traffic in the regulated buyer's
research path.

**What we ship.**

- One **co-authored article per quarter** with an ISO 42001 or EU AI
  Act consultancy. Topic format: "How [Framework Article N] is
  evidenced in practice." Each piece references real `.specular/`
  artifacts and is hosted on both properties.
- One **joint webinar per quarter** with the same partner: 30 minutes
  framework explainer, 30 minutes Specular walkthrough, 30 minutes
  Q&A.
- A **public reference list** of consultancies that have validated the
  Specular bundle against their framework methodology, hosted at
  `docs/gtm/references.md` (Q3).

**Investment.** Two engineering days + one writer day per piece. One
SE day per webinar. Quarterly cadence is the floor; bi-monthly if we
find a second tier-1 partner.

**Lead measure.** Co-authored pieces published per quarter (target: 1).
Inbound referrals from the partner's website per piece (target: 5+
per piece).

**Lag measure.** Pipeline sourced from compliance-influencer content
in the last 12 months. Goal: 20% of FY26 sourced pipeline.

### 3. Public bundle gallery

**The thesis.** SEO around "ISO 42001 evidence example," "SOC 2 AI
change-control sample," "EU AI Act Article 17 documentation" is
underserved. A public gallery of anonymised real `bundle-*.yaml` files,
indexed by framework and by pattern, becomes the canonical search
result for those queries — and the buyer arrives at our doorstep
already convinced the artifact is real.

**What we ship.**

- A `gallery/` directory in `docs/gtm/` (or a separate marketing
  property if we want analytics fidelity), hosting 20+ anonymised
  bundles drawn from real (or carefully synthesised) projects.
- A **search-optimised landing page per framework** linking the gallery
  entries that satisfy the specific control. Target query: "soc 2
  cc8.1 ai change control example."
- An **OG-image generator** that creates a clean, readable preview for
  every bundle so links shared on Slack / LinkedIn / Twitter render as
  inline auditor-friendly evidence cards.
- A **submission flow** so external orgs can contribute their own
  (anonymised) bundles — feeds the policy-library moat described in
  the wedge doc.

**Investment.** One front-end week to stand up the gallery. One
content-engineering week per quarter to add 5 new bundles. The
submission flow is a Q4 initiative.

**Lead measure.** Organic search traffic to gallery pages. Goal: 5K
unique visitors / month by end of FY26 against framework keywords.

**Lag measure.** Pipeline sourced from gallery as the first-touch
attribution. Goal: 15% of FY26 inbound.

## What we are explicitly not betting on

- **Conference keynotes.** Effort/payoff ratio is bad for a wedge in
  this stage. Sponsor when convenient; do not anchor distribution on it.
- **Paid search at the category level.** Until we own "AI Change
  Control" as a category claim, paid search at the broader "AI
  governance" terms is buying clicks for someone else's keyword.
- **Vendor co-marketing with AI-IDE companies.** Their incentive is
  to own the governance surface themselves; partnering creates a
  conflict of interest the moment a deal becomes contested. Use them
  as ICP signals, not as channel partners.

## Cadence

- **Quarterly:** refresh auditor packet (Channel 1), publish one
  co-authored piece + one webinar (Channel 2), add 5 gallery entries
  (Channel 3).
- **Monthly:** review lead/lag measures above. If two consecutive
  months miss the lead measure for a channel, retire or rework the
  bet — do not double-down on a stalled motion.
- **Annually:** rebuild this doc against the prior year's actuals.
  The bets that worked stay; the bets that did not are replaced, not
  rationalised.

The wedge wins because it is narrow and the distribution wins because
it compounds. Skipping either is the fastest way to lose a closing
two-year window.
