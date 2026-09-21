# Specular — Product & Engineering Intent

Specular is the trust boundary for AI-authored software changes.

This document defines the long-term product intent of Specular.

It exists to guide product decisions, architecture, APIs, UX, integrations, security decisions, roadmap prioritization, and contributions.

When implementation choices conflict with this document, prefer the choice that strengthens Specular’s core role as an independent, auditable change-control layer between AI software agents and production software delivery.


## 1. Mission

AI coding systems can increasingly plan, modify, test, and operate software with limited human involvement.

The limiting problem is no longer whether an AI system can produce code.

The limiting problem is whether organizations can trust, explain, govern, verify, and audit the changes those systems produce.

Specular exists to answer:

* What was the AI asked to do?
* Which agent, model, tool, and configuration performed the work?
* What did it actually change?
* Does the resulting change still match the approved intent?
* Did the change expand beyond its authorized scope?
* Which policies apply to it?
* Which verification was performed?
* Were required approvals obtained?
* Were exceptions granted?
* Who or what authorized those exceptions?
* Is the artifact being deployed the artifact that was evaluated?
* Can the complete decision be independently reconstructed later?

Specular turns these questions into deterministic, inspectable, enforceable software controls.


## 2. Product Thesis

The software development stack is gaining increasingly autonomous coding agents.

Those agents may include:

* Claude Code
* Codex
* GitHub Copilot
* Cursor
* Gemini
* IDE agents
* CI agents
* internal enterprise agents
* autonomous software-engineering systems
* future systems that do not yet exist

Specular must not depend on any single agent or model winning.

Instead:

Human / System Intent
          │
          ▼
   AI Coding Agent
          │
          ▼
      Code Change
          │
          ▼
┌─────────────────────────┐
│        SPECULAR         │
│                         │
│ provenance              │
│ intent                  │
│ scope                   │
│ drift                   │
│ policy                  │
│ risk                    │
│ verification            │
│ approvals               │
│ evidence                │
└─────────────────────────┘
          │
      ALLOW / DENY
          │
          ▼
        CI/CD
          │
          ▼
      Production

Specular should become the independent trust and evidence layer around AI-authored software.


## 3. Core Product Promise

For every governed software change, Specular should be capable of answering:

Why was this change allowed to ship?

The answer must be reconstructable from evidence rather than inference.

A successful Specular evaluation connects:

intent
  ↓
agent activity
  ↓
change
  ↓
verification
  ↓
policy
  ↓
approval
  ↓
artifact
  ↓
deployment decision

Every important decision should be explainable.

Every important artifact should be attributable.

Every important approval should be verifiable.


## 4. The Primitive: The Gate

The primary product abstraction is:

specular gate

The gate evaluates a proposed software change.

Conceptually:

discover change
      ↓
resolve provenance
      ↓
resolve intent
      ↓
determine affected scope
      ↓
evaluate drift
      ↓
evaluate risk
      ↓
evaluate policies
      ↓
run required verification
      ↓
verify approvals
      ↓
produce evidence
      ↓
ALLOW / DENY

The gate is the product.

Other capabilities exist to make the gate stronger, easier to adopt, more explainable, or more useful.

Users should not need to understand Specular’s internal subsystems to use the gate.


## 5. Example Gate Experience

$ specular gate
SPECULAR CHANGE CONTROL
──────────────────────────────────────────────
Change
  PR             #482
  Branch         feat/auth-hardening
  Files          17
  Diff           +643 −122
Provenance
  Harness        Claude Code
  Model          claude-sonnet
  Session        sp_8f3c
  Intent         AUTH-142
  Confidence     verified
Intent
  ✓ 7/7 requirements addressed
  ✓ expected behavior preserved
Scope
  ✓ changes remain within authorized scope
  ✓ no unexplained subsystems modified
Risk
  HIGH
  Factors:
    authentication
    public API modification
    security-sensitive paths
Policy
  ✓ 14 passed
  ⚠ 1 approval requirement
Verification
  ✓ unit tests
  ✓ integration tests
  ✓ static analysis
  ✓ dependency policy
  ✓ security policy
  ✓ architecture constraints
Approval
  ✓ security
  ✓ service owner
Evidence
  sha256:a893...
──────────────────────────────────────────────
VERDICT: ALLOW

Failures must be equally understandable.

SCOPE DRIFT
Requirement AUTH-142 authorizes changes to:
  internal/auth/**
  api/auth/**
The change also modifies:
  internal/billing/stripe.go
No requirement, task, approval, or plan explains
this modification.
Introduced by:
  Session   sp_8f3c
  Commit    18ac77
Possible resolutions:
  explain the change
  extend authorized intent
  obtain approval
  revert the modification

A developer should rarely need to inspect Specular internals to understand a gate result.


## 6. Product Principles

### 6.1 Agent Independent

Specular must work with any coding agent.

Integrations may provide stronger evidence, but the governance model must not depend on a particular provider.

Agents are producers of changes.

Specular evaluates changes.


### 6.2 Model Independent

Model routing is not the fundamental product.

Models will change continuously.

Specular should understand model identity and model policy where useful, but its value must survive changes in the model ecosystem.


### 6.3 Brownfield First

Existing repositories are the primary adoption target.

A repository must not need to adopt:

* Specular-native specifications
* Specular-native planning
* Specular-native agents
* Specular-native execution

before receiving value.

The ideal first experience is:

cd existing-repository
specular init
specular gate

Specular should discover existing controls whenever possible.

Examples:

* CI configuration
* CODEOWNERS
* repository structure
* test commands
* linters
* dependency management
* security tooling
* branch protection expectations
* agent configuration
* existing policy files

Specular augments existing workflows before replacing anything.


### 6.4 Progressive Trust

Specular should support increasing levels of provenance.

Level 0 — Unknown

The origin of the change cannot be established.

Level 1 — Declared

AI involvement is declared by metadata or the developer.

Level 2 — Integrated

An agent integration supplies provenance.

Level 3 — Observed

Specular observes the agent session and its outputs.

Level 4 — Governed

Specular controls execution capabilities, policy, environment, and evidence generation.

Organizations may adopt progressively stronger levels.

Weak provenance should be represented explicitly rather than pretending certainty.


### 6.5 Evidence Over Claims

Specular should prefer:

verified evidence

over:

asserted state

Whenever possible, conclusions should link to the evidence supporting them.


### 6.6 Explainability Over Scores

Avoid unexplained numbers such as:

AI safety score: 83

Prefer:

Risk: HIGH
Reasons:
  authentication subsystem modified
  new external dependency
  database migration
  incomplete provenance

Scores may exist internally where useful, but decisions exposed to humans must be explainable.


### 6.7 Deterministic Enforcement

Given identical:

* inputs
* policies
* evidence
* configuration

Specular should produce identical policy decisions wherever practical.

LLMs may assist analysis.

They must not silently become the final authority for security-critical enforcement.


### 6.8 Default Deny for Dangerous Capabilities

Sensitive capabilities must be explicit.

Examples:

* network access
* credential access
* filesystem writes
* process execution
* deployment
* external hooks
* privileged containers
* secrets
* destructive operations

Absence of policy should not accidentally grant powerful capabilities.


### 6.9 Advisory Before Blocking

Organizations need confidence before enabling hard gates.

Every important rule should support an adoption progression such as:

disabled
    ↓
observe
    ↓
advisory
    ↓
blocking

Specular should measure whether a policy is ready to become blocking.


## 7. Change Evidence Graph

The conceptual center of Specular should be a Change Evidence Graph.

A change is not merely a Git diff.

It is a graph of intent, activity, verification, authorization, and resulting artifacts.

Example:

Requirement R42
      │
      ├── approved-by ── Alice
      │
      ▼
Agent Session S918
      │
      ├── harness ── Claude Code
      ├── model ──── model-X
      ├── policy ─── regulated-prod
      │
      ▼
Plan P122
      │
      ├── Task T1
      ├── Task T2
      └── Task T3
             │
             ▼
         Commit abc123
             │
             ├── file A
             ├── file B
             └── test C
                    │
                    ▼
             Verification V92
                    │
                    ▼
               Approval A18
                    │
                    ▼
               Artifact Z
                    │
                    ▼
                Deployment

Important entities should have stable identities.

Important evidence should be content-addressable where appropriate.

Important relationships should be queryable.


## 8. Provenance

Provenance answers:

Where did this change come from?

Possible provenance information includes:

* agent harness
* harness version
* model/provider
* model identifier
* session identifier
* initiating user/system
* repository
* base commit
* resulting commits
* prompts or prompt digests
* tool calls
* execution environment
* policy profile
* timestamps
* linked intent
* signatures
* attestations

Secrets must never be included in provenance records.

Sensitive prompts and data should support digest/redacted representations.


## 9. Agent Provenance Protocol

Specular should define an open provenance format.

Conceptually:

{
  "schema": "specular.provenance/v1",
  "agent": {
    "harness": "example-agent",
    "version": "1.4.0"
  },
  "model": {
    "provider": "example",
    "id": "model-x"
  },
  "session": "sp_92d1",
  "intent": [
    "AUTH-142"
  ],
  "repository": "...",
  "base_commit": "...",
  "commits": [
    "abc123"
  ],
  "started_at": "...",
  "finished_at": "...",
  "evidence": "...",
  "signature": "..."
}

The protocol should be:

* open
* versioned
* extensible
* agent-neutral
* independently implementable

Third-party agents should be able to produce native Specular provenance without running inside Specular.

Specular should aspire to make this useful beyond Specular itself.


## 10. Intent

Intent represents what a change is authorized to accomplish.

Intent may originate from:

* Specular specifications
* GitHub Issues
* Jira
* Linear
* pull-request descriptions
* architecture decisions
* requirements systems
* external APIs
* policy-approved manual declarations

Specular-native specs should provide the strongest integration but must not be mandatory.

Intent should be traceable to the resulting change.


## 11. Drift

Drift is a primary Specular capability.

Drift is not one-dimensional.

Specular should eventually distinguish at least:

Intent Drift

Implementation no longer satisfies or corresponds to approved intent.

Scope Drift

Files, services, modules, or systems outside authorized scope were modified.

Behavioral Drift

Observable behavior differs from approved expectations.

Architecture Drift

The change violates architectural boundaries or decisions.

Dependency Drift

Dependencies changed without corresponding authorization or explanation.

Security Drift

The resulting system violates applicable security constraints.

Infrastructure Drift

Infrastructure behavior differs from approved configuration or expectations.

Policy Drift

The implementation is incompatible with organizational policy.

Agent Drift

The agent operates outside its authorized tools, capabilities, models, or environment.

Every reported drift should explain:

what changed
why it is considered drift
which intent/policy establishes the expectation
which evidence establishes the violation
who/what introduced the change
how it may be resolved


## 12. Risk

Governance must be proportional to risk.

Specular should derive an explainable change risk profile from signals such as:

* authentication changes
* authorization changes
* cryptography
* credential handling
* sensitive paths
* public API changes
* database migrations
* infrastructure changes
* dependency changes
* production configuration
* blast radius
* provenance quality
* autonomous execution
* test coverage changes
* policy exceptions
* security findings
* unexplained scope

Example:

Risk: HIGH
Factors:
  + authentication subsystem
  + new external dependency
  + public API modification
  + production configuration
Mitigations:
  ✓ integration tests
  ✓ security analysis
  ✓ security approval

Risk determines governance requirements.

It should not merely produce a dashboard number.


## 13. Risk-Adaptive Governance

Policies should support risk-sensitive requirements.

Example:

risk:
  low:
    approval: none
  medium:
    approval:
      - code-owner
  high:
    approval:
      - security
      - service-owner
  critical:
    autonomous_execution: false
    approvals:
      - security
      - platform
      - service-owner

The objective is strong governance with minimum unnecessary friction.


## 14. Policy

Policy determines whether a change is acceptable.

Policy may govern:

* provenance
* agents
* models
* execution
* networking
* credentials
* repositories
* paths
* dependencies
* testing
* security
* architecture
* approvals
* risk
* deployment
* exceptions
* evidence requirements

Policy evaluation must produce explanations.

Bad:

POLICY FAILED

Good:

POLICY FAILED
SEC-17: Authentication changes require security approval.
Affected paths:
  internal/auth/token.go
  internal/auth/session.go
Required:
  approval: security
Observed:
  approval: service-owner
Resolution:
  obtain security approval


## 15. Executable Control Packs

Compliance mappings should become executable control packs.

Examples:

* SOC 2
* ISO 27001
* ISO 42001
* NIST AI RMF
* EU AI Act-related organizational controls
* internal enterprise standards

A control pack connects:

control objective
      ↓
Specular policies
      ↓
required evidence
      ↓
evaluation
      ↓
exceptions
      ↓
auditor-readable record

Specular must not claim that passing a control pack means an organization is compliant.

Instead, it should accurately state that Specular can implement controls and produce evidence supporting compliance activities.


## 16. Verification

Verification establishes whether the resulting change satisfies required technical checks.

Possible verification includes:

* tests
* static analysis
* security scanners
* dependency checks
* architecture rules
* build verification
* reproducibility checks
* infrastructure validation
* custom organizational checks

Verification results become evidence.

The exact command alone is insufficient.

Where feasible, record:

* command/tool
* version
* configuration
* input artifact
* output digest
* result
* timestamp
* execution environment


## 17. Approvals

Approval is an explicit governance event.

An approval should capture:

* approver identity
* role/context
* object being approved
* exact artifact/change digest
* policy requiring approval
* timestamp
* expiration where applicable
* signature/attestation where appropriate

Approval of one artifact must not silently authorize a different artifact.

Changes after approval should invalidate approval where relevant.


## 18. Exceptions

Real organizations require exceptions.

Specular should support controlled exceptions rather than forcing policy bypasses outside the system.

An exception should contain:

policy
reason
requester
approver
scope
expiration
affected artifact
evidence

Permanent silent suppression should be discouraged.

Exceptions should be searchable and auditable.


## 19. Evidence

Evidence is one of Specular’s primary products.

Machine-readable evidence is necessary.

Human-readable evidence is equally important.

A change record should be understandable without unpacking an archive.

Example:

AI CHANGE RECORD
──────────────────────────────────────
Change       PR #482
Repository   payments-api
Risk         HIGH
Provenance
Harness      Claude Code
Session      sp_92d1
Intent       AUTH-142
Controls
✓ SEC-17
✓ ARCH-04
✓ SOC2 CC8.1 supporting control
⚠ exception EX-192
Verification
✓ 428 tests
✓ SAST
✓ dependency policy
✓ intent alignment
Drift
None detected
Approvals
✓ Security
✓ Service Owner
Artifact
sha256:...
Decision
ALLOWED
Timestamp
2026-09-21T09:41:00Z


## 20. Explain

specular explain should become a major product surface.

Examples:

specular explain
specular explain PR-482
specular explain abc123
specular explain --policy SEC-17
specular explain --file internal/auth/token.go

The core question is:

Why did Specular make this decision?

For an allowed change:

Why was this allowed?

For a denied change:

Why was this denied?

For a line or file:

Why does this change exist?

Explainability should traverse the evidence graph.


## 21. Developer Experience

Specular must integrate into the place developers already work.

The primary developer interfaces are:

1. Pull request
2. CLI
3. IDE/agent integration
4. CI logs

The PR experience should summarize:

Specular Change Control            PASSED
Risk                               Medium
AI provenance                      Verified
Intent                             AUTH-142
Requirements                       8/8
Unexpected scope                   None
Policies                           17/17
Verification                       Passed
Approval                           Required ✓
Evidence →

Failures should create actionable annotations as close as possible to the relevant code.


## 22. Security Experience

Security teams need different questions answered.

Examples:

Which AI-authored changes touched authentication?
Which changes used unapproved models?
Which changes entered production without strong provenance?
Which policy exceptions are active?
Which autonomous sessions received network access?
Which changes modified dependency trust boundaries?

Specular should make these questions easy to answer.


## 23. Auditor Experience

Auditors care about reconstruction.

A future organizational Specular system should answer queries such as:

Show all production changes between
2026-01-01 and 2026-03-31 that:
  involved generative AI
  AND touched authentication
Include:
  intent
  provenance
  verification
  approvals
  exceptions
  deployed artifact

The output should be exportable in human- and machine-readable forms.


## 24. Brownfield Initialization

specular init should optimize for existing repositories.

Example:

$ specular init
Analyzing repository...
Detected
✓ GitHub
✓ GitHub Actions
✓ CODEOWNERS
✓ Go
✓ golangci-lint
✓ Dependabot
✓ Docker
✓ Claude Code
Existing controls
✓ required review
✓ tests
✓ static analysis
✓ dependency updates
Recommended Specular baseline
provenance      advisory
intent drift    advisory
scope drift     advisory
dependencies    blocking
secrets         blocking
approvals       existing CODEOWNERS
Install configuration? [Y/n]

The generated configuration should be minimal.


## 25. Baselines

Existing repositories contain historical behavior that cannot immediately satisfy ideal policy.

Specular should support explicit baselines.

specular baseline

A baseline means:

This is the acknowledged current state.

It does not mean:

This state is good.

New drift from the baseline can then be governed without requiring immediate remediation of all historical debt.

Baselines should be explicit, versioned, and reviewable.


## 26. Sessions

specular session remains strategically useful.

Its purpose is not to make Specular another coding agent.

Its purpose is to provide strong provenance and controlled execution.

Sessions may provide:

* isolated worktrees
* agent attribution
* model attribution
* tool tracking
* execution evidence
* policy enforcement
* capability restrictions
* intent linkage
* deterministic lifecycle records

External agents remain supported.

Running through a Specular session simply provides stronger evidence.


## 27. Autonomous Execution

Autonomous execution is a supporting capability.

It must always remain subordinate to governance.

Autonomy must not weaken:

* provenance
* policy
* execution isolation
* evidence
* approval
* capability controls

Higher autonomy should result in stronger controls, not fewer controls.


## 28. Capability Security Model

Execution should move toward explicit capabilities.

Example conceptual capabilities:

fs.read
fs.write
process.execute
network.outbound
credentials.read
repository.commit
repository.push
hook.invoke
deployment.trigger

Policies grant capabilities.

Example:

capabilities:
  fs.read: true
  fs.write:
    paths:
      - internal/auth/**
      - tests/**
  network.outbound: false
  credentials.read:
    allow:
      - TEST_DATABASE_URL
  deployment.trigger: false

No powerful capability should be accidentally granted because configuration is absent.


## 29. Secrets

Secrets must never become ordinary evidence.

Specular must ensure that:

* manifests do not persist raw secret values
* logs redact credentials
* prompts can be redacted
* environment values are not casually serialized
* evidence exports remain safe to distribute
* plugin/hook systems cannot trivially exfiltrate credentials

Prefer recording:

credential identifier
provider
access event
purpose
digest where meaningful

rather than values.


## 30. Network Access

Network access is a capability.

Sandboxed or autonomous execution should default to:

network = none

Network access must be explicitly granted.

Policies should eventually support destination restrictions.

Example:

network:
  outbound:
    allow:
      - api.github.com
      - proxy.internal.example


## 31. Plugins and Hooks

Plugins and hooks extend the trust boundary and therefore require explicit governance.

They should declare capabilities.

Example:

plugin:
  capabilities:
    network:
      - api.example.com
    filesystem:
      read:
        - reports/**

Untrusted configuration must not be able to create arbitrary network or process execution without policy authorization.


## 32. CI/CD

CI is a primary enforcement point.

The ideal integration should be extremely small.

Conceptually:

- uses: felixgeelhaar/specular@v1

followed by repository discovery and:

specular gate

Users should not have to reconstruct Specular’s internal pipeline in CI configuration.

Advanced configuration remains possible.


## 33. Control Plane

A future Specular service should remain a thin organizational control plane.

Its responsibilities may include:

* organization policy distribution
* RBAC
* identity
* evidence indexing
* cross-repository search
* key management
* approval workflows
* control-pack distribution
* exception management
* SIEM integration
* fleet visibility

The service should not unnecessarily replace:

* Git
* GitHub/GitLab
* CI
* artifact registries
* coding agents
* project-management systems

Specular should integrate with the software delivery ecosystem rather than recreate it.


## 34. CLI Architecture

The intended product hierarchy is:

SPECULAR
│
├── GATE
│   ├── provenance
│   ├── intent
│   ├── scope
│   ├── drift
│   ├── risk
│   ├── policy
│   ├── verification
│   └── approval
│
├── EVIDENCE
│   ├── graph
│   ├── records
│   ├── attestations
│   ├── bundles
│   └── exports
│
├── EXPLAIN
│
├── SESSION
│   ├── agent integrations
│   ├── isolation
│   └── provenance
│
├── POLICY
│   ├── organizational rules
│   ├── capabilities
│   └── control packs
│
└── INTEGRATIONS
    ├── GitHub
    ├── GitLab
    ├── CI
    ├── agents
    ├── project management
    └── security systems

Specification, planning, generation, model routing, and autonomous building are supporting capabilities rather than equal product pillars.


## 35. Features to Demote

### Generic Generation

Generic:

specular generate "..."

is not strategically differentiating.

Agent vendors already provide excellent generation interfaces.

Keep only where it strengthens governance or workflow integration.


### Model Routing

Routing may remain useful for:

* model policy
* cost controls
* approved-provider enforcement
* autonomous execution

It should not define the product.


### Spec Authoring

Specular-native specifications are valuable because they provide strong intent.

They must not become a prerequisite for governance.

External intent systems should remain first-class.


## 36. Features to Protect

The following capabilities directly strengthen the product thesis and should receive disproportionate attention:

* drift detection
* provenance
* evidence
* policy
* approvals
* capability enforcement
* execution isolation
* agent integrations
* GitHub/GitLab integration
* explainability
* brownfield adoption
* deterministic enforcement


## 37. Product Metrics

Do not optimize primarily for:

number of generations
tokens processed
number of agent sessions

Optimize for trust.

Important metrics include:

* gate adoption
* percentage of AI changes with verified provenance
* unexplained-change rate
* gate override rate
* false-positive rate
* false-negative incidents
* policy suppression rate
* exception rate
* exception age
* approval latency
* time to explanation
* time to remediation
* drift recurrence
* advisory → blocking conversion
* evidence completeness

A particularly important question is:

Can an organization safely make Specular blocking without developers bypassing it?


## 38. Quality Bar

A feature is not complete merely because it works.

For core governance functionality it should also be:

* deterministic where practical
* explainable
* testable
* observable
* auditable
* secure by default
* backwards-compatible where required
* usable in automation
* useful interactively
* documented
* represented in evidence where relevant


## 39. Trust Invariants

The following should be treated as architectural invariants.

Invariant 1

A decision must be attributable to the policy and evidence that produced it.

Invariant 2

Changing an evaluated artifact must invalidate evidence tied to the previous artifact where appropriate.

Invariant 3

An approval must identify exactly what was approved.

Invariant 4

Secrets must never become ordinary evidence.

Invariant 5

Dangerous execution capabilities must be explicitly granted.

Invariant 6

Unknown provenance must never be silently represented as verified provenance.

Invariant 7

Policy exceptions must be explicit and auditable.

Invariant 8

A probabilistic model must not silently become the final authority for deterministic security policy.

Invariant 9

Specular must be capable of explaining why a blocking decision occurred.

Invariant 10

Using Specular-native authoring tools must not be required to govern a repository.


## 40. Non-Goals

Specular is not primarily:

* an IDE
* a coding assistant
* a general-purpose LLM client
* a model marketplace
* a generic agent framework
* a CI system
* a Git hosting platform
* a project-management system
* a compliance certification product
* a replacement for human engineering judgment

Specular may integrate with or contain supporting functionality related to these areas.

They must not distract from the trust-boundary mission.


## 41. Strategic Moat

Specular’s defensibility should accumulate in:

Change Evidence

A normalized model connecting intent, AI activity, code, policy, verification, approval, artifacts, and deployment.

Drift Intelligence

High-quality detection and explanation of unauthorized or unexplained change.

Provenance

Agent-neutral provenance with progressively stronger verification.

Policy

Practical software-change governance that can become blocking without destroying developer velocity.

Control Packs

Executable mappings between organizational controls and software-delivery evidence.

Ecosystem

Agent, CI, source-control, project-management, security, and compliance integrations.

The moat should not depend on having better text generation than model providers.


## 42. Open Ecosystem Intent

The provenance schema, evidence primitives, integration SDKs, and core policy concepts should remain open enough to become ecosystem infrastructure.

An agent vendor should be able to integrate Specular provenance.

A security vendor should be able to consume Specular evidence.

A CI vendor should be able to invoke the Specular gate.

An enterprise should be able to write internal policies.

An auditor should be able to verify exported evidence independently where feasible.


## 43. Near-Term Product Priorities

P0 — Establish the Product

1. Unified specular gate
2. Change Evidence Graph
3. High-quality drift explanations
4. Brownfield specular init
5. Baseline support
6. Excellent GitHub PR/check UX
7. Security hardening of the execution boundary
8. Secret-safe manifests and evidence
9. Deterministic policy decisions
10. specular explain

P1 — Establish the Trust Platform

1. Risk-adaptive governance
2. Human-readable evidence records
3. Agent Provenance Protocol
4. Native agent integrations
5. Executable control packs
6. Better approval/exception workflows
7. GitLab support
8. Jenkins/generic CI support
9. Evidence querying
10. Stronger session provenance

P2 — Organizational Scale

1. Specular control plane
2. Organization policy management
3. Cross-repository evidence
4. Enterprise identity/RBAC
5. Key management
6. Fleet visibility
7. Audit search
8. SIEM integrations
9. Central exception management
10. Organization-wide provenance analytics


## 44. Ideal Adoption Journey

Minute 1

brew install specular
cd repository
specular init

Specular discovers the repository.

Minute 5

specular gate

The developer sees useful findings without changing their workflow.

Day 1

Specular runs advisory checks in CI.

Week 1

The organization establishes a baseline and tunes policies.

Week 2

High-confidence security controls become blocking.

Month 1

Agent integrations provide verified provenance.

Later

High-risk repositories use governed Specular sessions and capability-controlled execution.

The product should allow this progression without requiring a platform migration.


## 45. The Five-Command Test

A mature Specular experience should make sense through five commands:

specular init
specular gate
specular explain
specular evidence show
specular session

If the product cannot be explained through these concepts, its surface area is probably becoming too complicated.


## 46. Decision Framework

When considering a new feature, ask:

Does it improve our ability to establish provenance?

If yes, it may belong.

Does it improve our ability to detect meaningful drift?

If yes, it probably belongs.

Does it improve policy enforcement?

If yes, it probably belongs.

Does it improve evidence quality?

If yes, it probably belongs.

Does it make governance easier to adopt?

If yes, it probably belongs.

Does it merely duplicate functionality available from coding agents?

If yes, strongly question it.

Does it require users to migrate their development workflow before receiving value?

If yes, redesign it.

Does it increase autonomous power without increasing control?

If yes, reject it.

Can the resulting decision be explained?

If no, redesign it.


## 47. Long-Term Vision

Software organizations should eventually be able to state:

AI systems are allowed to modify our software.
We do not need to blindly trust those systems.
Every governed change has attributable intent.
Every AI-authored change has provenance.
Every material deviation is detectable.
Every sensitive capability is controlled.
Every required verification is recorded.
Every required approval is attributable.
Every exception is explicit.
Every deployed artifact can be traced to the
decision that allowed it to ship.
And that decision can be independently reconstructed.

Specular is the infrastructure that makes this possible.


## 48. One-Sentence Definition

Specular is an open change-control and evidence layer that verifies AI-authored software changes against human intent, organizational policy, and required verification before they ship.


## 49. Short Definition

The trust boundary for AI-authored code.


## 50. Product Test

Before shipping any major Specular feature, ask one final question:

Does this make organizations more capable of safely trusting AI-authored software changes?

If the answer is no, the feature is probably outside Specular’s core intent.
