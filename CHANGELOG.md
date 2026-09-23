# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Approvals list hollow Soft trail**: when no EVID Soft trail is printed
  (including empty / filtered-empty boards), `approvals list` jumps to
  `approvals pending` / `doctor` (pending hollow Soft-trail #176 parity).

- **Approvals pending hollow Soft trail**: when no open exceptions Soft trail
  is printed, `approvals pending` jumps to `doctor` / `list --status open`
  (doctor Soft-trail / OpenExceptions Soft-trail parity on empty boards).

- **Doctor Soft trail Next Steps**: when open soft-ALLOW exceptions are
  present, doctor Next Steps jump to `approvals pending` beside
  `approvals list --status open` (OpenExceptions Soft-trail footer parity).

- **DENY Soft trail boards**: Soft=no DENY rows on `session status` / `list` /
  `wait` and `evidence list` footer to evidence show / explain / session show
  (session) plus Soft trail `approvals pending` / `doctor` (session show DENY
  Soft-trail #173 / Soft-ALLOW Soft-footer parity). Soft=yes Soft-ALLOW footers
  unchanged.

- **Session show DENY Soft trail**: when gate is DENY and SoftAllowIDs are
  empty, `session show` jumps to `approvals pending` / `doctor` (gate DENY
  Soft-trail #172 parity). Soft-ALLOW Soft trail unchanged.

- **Gate DENY Soft trail**: when Approvals are empty (no Soft Overruled /
  OpenExceptions trail), gate text / markdown SoftAllow Hint and
  FormatExplain jump to `approvals pending` / `doctor` (Soft Overruled /
  OpenExceptions Soft-trail parity).

- **Approvals pending `--json`**: emit
  `{summary, policyChanges, bundles, drift, openExceptions}` (doctor
  `open_exceptions` / Soft trail automation parity). Open exceptions alone
  still exit 0; pending policy/bundle/drift still exit 1.

- **Gate advisory OpenExceptions Soft trail**: when open exceptions are
  advisory (not yet soft-ALLOW Overruled), gate text / markdown and
  FormatExplain jump to `approvals show` / `list --status open` /
  `pending` / `doctor` (Soft Overruled Soft-trail parity).

- **Approvals Soft trail**: `approvals list` EVID footers and `approvals show`
  Refs jump to `approvals pending` / `doctor` (gate Soft-trail parity).

- **Soft footer Soft trail**: Soft=yes footers on `session status` / `list` /
  `wait` and `evidence list`, open-exception footers, and `session show` Soft
  jumps include `approvals pending` / `doctor` (gate Soft-trail #167 parity).

- **Gate Soft-ALLOW Soft trail**: text / markdown Soft Overruled boards and
  FormatExplain Soft Refs jump to `approvals pending` / `doctor` beside
  show/list (approve create/close Soft-trail parity).

- **Approve create Soft trail**: after recording an exception, human output
  prints Refs to `approvals show` / `list --status open` / `pending` /
  `doctor` / `gate` (and `evidence show` / `explain` when `--evidence` is set).

- **Approvals close Soft trail**: after `approvals close` / `revoke`, human
  output prints Refs to `approvals pending` / `list --status open` /
  `doctor` / `gate`, plus remaining open exceptions (pending Soft parity).

- **Approvals pending open exceptions**: `approvals pending` lists open
  soft-ALLOW exceptions with `approvals show` / `evidence show` /
  `list --status open` jumps (doctor open_exceptions parity). Open
  exceptions alone do not set exit 1.

- **Soft footer evidence jumps**: Soft=yes rows on `session status` /
  `list` / `wait` jump to `evidence show` / `explain --session` when an
  EvidenceID is known; `evidence list` Soft footers also print `evidence show`
  (session show Soft / approvals show Refs parity).

- **Soft footer `approvals show`**: Soft=yes rows on `session status` /
  `list` / `wait` and `evidence list` jump to `approvals show <id>` when
  SoftAllowIDs are known (session show Soft / FormatExplain Overruled parity).

- **Doctor open exceptions**: `specular doctor` surfaces open soft-ALLOW
  exceptions (`open_exceptions` in JSON) with jumps to `approvals show` /
  `evidence show` / `approvals list --status open`, and a Next Step to review
  them (P1 #6 depth; Soft-ALLOW reverse-nav).

- **Approvals list EVID jumps**: rows with a bound `evidence_id` footer to
  `evidence show <id>` / `explain <id>` (approvals show Refs / evidence list
  Soft-ALLOW reverse-nav parity).

- **Soft-ALLOW evidence bind**: when gate soft-ALLOWs and persists evidence,
  overruled open exceptions are stamped with `evidence_id` so
  `approvals list --evidence <id>` resolves Soft List jumps. `--evidence`
  also joins SoftAllow overrule ResourceIDs from the named evidence record
  (pre-bind Soft trails). `session show` Soft List prefers `--evidence`.

- **Gate Soft-ALLOW list `--evidence`**: text / markdown Soft List prefer
  `approvals list --evidence <id>` when the gate run persisted an evidence
  record (`FormatTextWith` / markdown opts; FormatExplain #157 parity).

- **Evidence list Soft-ALLOW jumps**: Soft=yes rows footer to
  `approvals list --evidence <id>` / `explain <id>`; `--json` rows include
  `softAllowIds` (session status Soft-ALLOW footer parity). FormatExplain
  Soft List/Open also prefer `--evidence <id>` when the record id is known.

- **Provenance show/verify Refs**: human APP output jumps to
  `session show` / `explain --session` / peer `provenance verify|show`
  (FormatExplain session Refs parity).

- **Session status/list/wait Soft-ALLOW jumps**: Soft=yes rows footer to
  `approvals list --evidence <id>` / `session show`; `--json` evidence map
  includes `softAllowIds` (session show SoftAllowIDs parity).

- **Approvals list `--evidence`**: filter the trust board by `evidence_id`
  exact or prefix (EVID column; combinable with `--status`/`--type`/
  `--policy`/`--scope`).

- **Session show soft-ALLOW jumps**: when newest evidence has exception
  overrules, `session show` prints `approvals show <id>` /
  `approvals list --status open` beside Soft (`--json` → `softAllowIds`).

- **Approvals show evidence refs**: when `evidence_id` is bound, Refs jump to
  `evidence show <id>` / `explain <id>`; open exceptions also list
  `approvals list --status open` (FormatExplain → approvals show parity).

- **Gate soft-ALLOW board jumps**: text / markdown Approvals Overruled
  point to `approvals show <id>` / `approvals list --status open`
  (FormatExplain #150 parity on the live gate board).

- **Explain board reverse jumps**: soft-ALLOW `FormatExplain` Approvals/Refs
  point to `approvals show <id>` / `approvals list --status open`; provenance
  sessions in Refs jump to `session show` / `explain --session` (session show
  ↔ evidence explain parity).

- **Approvals list trust board**: human `approvals list` prints
  ID/TYPE/STATUS/POLICY/SCOPE/EVID/APPROVER/EXPIRES with open/closed/expired
  summary; `--json` emits `{summary, records}` (session/evidence list parity).

- **Explain trust filters**: `--verdict` / `--risk` / `--soft-allow` /
  `--attested` / `--governed` / `--protocol` / `--harness` select the newest
  matching Change Evidence Graph record (combinable with one graph selector
  `--file`/`--control`/`--commit`/`--session`; evidence list parity).

- **Evidence list trust board**: human `evidence list` prints GATE/SOFT/RISK/
  ATTEST/GOV/PROTO/COMMIT/SESSION/HARNESS/CREATED (session status vocabulary);
  `--json` emits `{summary, records}` instead of a bare ID array.

- **Session wait trust filters**: `session wait` accepts the same
  `--verdict` / `--soft-allow` / `--risk` / `--protocol` / `--attested` /
  `--governed` / `--harness` flags as status/list; filters the emitted board
  only (wait/attest/gate still cover the full waited set).

- **Session status/list/wait EVID column**: human boards show newest Change
  Evidence Graph id beside GATE (`evidence.<id>.evidenceId` parity) so fleet
  operators can jump to `evidence show <id>` / `explain` without `session show`.

- **Session wait board parity**: human `session wait` and `--json` emit the
  status trust board (`GOV…RISK` + `EXIT`; `--json` → `{summary,sessions,evidence}`)
  after optional `--attest`/`--gate`/`--bundle` so fleet→gate evidence is visible
  without a follow-up `session status`.

- **Session status/list trust filters**: `--verdict` / `--soft-allow` /
  `--risk` / `--protocol` / `--attested` / `--governed` / `--harness`
  narrow the fleet board (and `list`) with evidence-list parity; summary
  counts and `--json` reflect the filtered set (`--watch` re-applies).

- **Session status/list PROTO column**: human boards and
  `session status --json` evidence map show APP protocol schema+bound
  (`protocol`) from the newest matching Change Evidence Graph record
  (`evidence list --protocol` parity; distinct from APP file presence).

- **Session show gate evidence**: `session show` prints COMMIT / GATE / SOFT /
  RISK / `evidenceId` from the newest matching Change Evidence Graph record,
  DENY Next steps when applicable, and jumps to `explain --session` /
  `evidence show`. `--json` adds an `evidence` object beside the session record.

- **Session status/list RISK column**: human boards and
  `session status --json` evidence map show gate risk level
  (`NONE|LOW|MEDIUM|HIGH|CRITICAL`; empty → `NONE`) from the newest
  matching Change Evidence Graph record (`evidence list --risk` parity).

- **Session status/list SOFT column**: human boards and
  `session status --json` evidence map show soft-ALLOW (`softAllow`) when
  the newest matching Change Evidence Graph record has exception overrules
  (`evidence list --soft-allow` parity) beside GATE.

- **Session status/list GATE column**: human boards and
  `session status --json` evidence map show ALLOW/DENY (plus `evidenceId`)
  from the newest Change Evidence Graph record whose
  `gate.provenance.sessions[]` matches the session id.

- **Explain `--session`**: newest Change Evidence Graph record whose
  `gate.provenance.sessions[]` matches the fleet board session id
  (`evidence list --session` parity).

- **Session wait `--bundle` packs Change Evidence Graph**: includes
  `.specular/evidence/*.json` (+ `latest`) beside attest/APP/SARIF so fleet
  packets support `explain <sha>` / `explain --commit` offline.

- **Session status/list COMMIT column**: human boards and
  `session status --json` evidence map show short worktree HEAD beside
  ATTEST/APP so fleet CI can jump to `explain <sha>`.

- **Evidence commit + explain/list by SHA**: gate `Change.Commit` records
  `HEAD`; evidence persists it; `explain abc123` / `--commit` and
  `evidence list --commit` select by SHA prefix (PRODUCT_INTENT §20).

- **Explain / evidence `--control` (PRODUCT_INTENT §20)**: substring filter
  on failed policy checks, exception `--policy`, and soft-ALLOW bind tokens;
  `explain --control SEC-17` + `evidence list --control`. Explain `--policy`
  (fresh policy file) renamed to `--policy-file` (deprecated alias retained).

- **Explain `--file` (PRODUCT_INTENT §20)**: `specular explain --file
  <substr>` selects the newest evidence record whose root / drift finding
  paths contain that substring (same matcher as `evidence list --path`).

- **Gate DENY Next steps**: text / markdown / evidence explain append
  section-specific remediation (`session attest`, APP verify, governed
  start, drift/policy/risk fixes) beyond soft-ALLOW Approvals Hint.

- **Session wait progressive-trust `--require-*`**: `session wait --gate` /
  `--bundle` accept `--require-attested` / `--require-protocol` /
  `--require-governed` (same semantics as `specular gate`); fleet CI example
  + `proof.sh` honor optional `REQUIRE_*` env vars.

- **Session wait product gate (fleet→evidence)**: `session wait --gate` /
  `--bundle` runs product `specular gate` (provenance/drift/policy +
  evidence) instead of legacy drift-only eval; brownfield soft-skips
  missing specs. `--bundle` also packs sibling `.provenance.json` APP docs.

- **Session status/list ATTEST + APP columns**: human boards and
  `session status --json` evidence map show sibling attestation /
  `.provenance.json` presence beside GOV.

### Changed

- **Session wait `--json`**: emits `{summary, sessions, evidence}` (status
  board shape) instead of a bare session array; human wait table adds
  trust columns beside EXIT and prints after `--gate`/`--bundle` so GATE
  reflects newest evidence.

- **Evidence list `--json`**: emits `{summary, records}` trust board instead
  of a bare ID array; human list is a tab board (not one ID per line).

- **Approvals list `--json`**: emits `{summary, records}` trust board instead
  of a bare record array; human list is a tab board (not grouped prose).

- **DENY soft-ALLOW hints include provenance**: gate / evidence Approvals Hint
  and approve exception footer suggest `drift|policy|risk|provenance` (and
  section-specific `--policy` when a single section failed); docs align.

- **Evidence list `--protocol` wording**: help/docs clarify `ok` means
  schema+sibling binding (same as gate Protocol board after #119), not
  schema-only.

### Added

- **CI progressive-trust ladder docs**: `examples/ci-cd/README.md` +
  `generic-ci.sh` point at `examples/policy/progressive-trust.yaml` and
  clarify `--require-protocol` / board `schema+bound` wording.

### Fixed

- **Session store concurrent Save**: unique temp files per `Store.Save` so
  concurrent writers for the same session ID cannot truncate a shared
  `.json.tmp` (fixes `TestStartManyDependencyChain` decode flake); `Load`
  retries briefly on JSON decode errors.

### Added

- **Progressive-trust policy example**: `examples/policy/progressive-trust.yaml`
  combines risk tiers + `provenance.attested|protocol|governed: enforce`;
  doctor advisory Mode points at it (Next Steps + Progressive trust block).

- **Approvals list `--type` / `--policy` / `--scope` (P1 #6 depth)**:
  `specular approvals list` filters by record type and case-insensitive
  policy/scope substrings (combinable with `--status`).

- **Session attest/show APP surface**: `session attest` reports `ProvenancePath`
  + `governed` and prints `provenance verify` next to `auto verify`;
  `session show` lists sibling attestation/APP paths when present.

- **Agents harness `--require-governed` docs**: per-harness READMEs
  (`claude-code` / `cursor` / `codex` / `gemini`) document
  `integrate --enforce --require-governed --force` beside Level-3 enforce.

- **Protocol board schema+bound labels**: gate / evidence / markdown Protocol
  counts annotate `ok` as schema+sibling binding (`docs=N ok=M schema+bound`);
  `provenance verify` prints a Bound line (`sibling` / `projected`).

- **Session integrate `--require-governed` (P1 #4 depth)**: with
  `--enforce`, Stop/SessionEnd hooks also pass `gate --require-governed`
  so fail-closed completion requires a governed (safer native) session.
  Errors if used without `--enforce`.

- **Provenance verify sibling binding**: `specular provenance verify` (and
  gate `protocol: enforce` APP doc counts) bind `.provenance.json` to the
  sibling `.attestation.json` for session / harness / governed / source —
  still schema-level, not cryptographic signatures.

- **Evidence list `--protocol` (P1 #9 depth)**: `specular evidence list
  --protocol[=true|false]` filters records by APP `.provenance.json` docs
  present+schema+bound vs missing/invalid/unbound (`protocolDocs` /
  `protocolOk`).

- **Agents examples `--enforce` docs**: `examples/agents/` README + harness
  guides document Level-3 `session integrate --enforce --force`; reference
  Stop scripts regenerated with advisory mode marker.

- **Evidence list `--governed` (P1 #9 depth)**: `specular evidence list
  --governed[=true|false]` filters Change Evidence Graph records by
  gate provenance.governed (safer native launch), combinable with existing
  soft-allow/attested/session filters.

- **Session integrate `--enforce` (P1 #4 depth)**: opt-in fail-closed
  Stop/SessionEnd hooks (`attest` + `gate --require-attested --require-protocol`)
  so provenance can block agent completion. Default remains advisory; use
  `--force` to rewrite an existing advisory hook.

- **Require governed sessions (P1 #10 depth)**: opt-in policy
  `provenance.governed: enforce` and CLI `--require-governed` / CI
  `REQUIRE_GOVERNED` DENY attested trees without a governed session
  (idle when unattested). Soft-ALLOW via `--policy provenance`. Example:
  `examples/policy/provenance-governed.yaml`.

- **Approvals list `--status` (P1 #6 depth)**: `specular approvals list
  --status open|closed|expired` filters the local trail by lifecycle
  (closed wins over expired).

- **Evidence list soft-ALLOW / attested filters (P1 #9 depth)**:
  `specular evidence list --soft-allow[=true|false]` and `--attested[=true|false]`
  query exception soft-ALLOW overrules and provenance attestation on Change
  Evidence Graph records (combinable with existing filters).

- **Gate `--require-protocol` + CI `REQUIRE_PROTOCOL`**: CLI flag mirrors policy
  `provenance.protocol: enforce` (idle when unattested). GitHub / GitLab /
  Jenkins / generic gate templates accept optional `REQUIRE_PROTOCOL`.

- **CI examples `--require-attested`**: GitHub / GitLab / Jenkins / generic
  gate templates accept optional `REQUIRE_ATTESTED` so progressive trust can
  be toggled in CI without editing `policy.yaml`.

- **Gate `--require-attested`**: CLI flag mirrors policy
  `provenance.attested: enforce` so CI can DENY unattested trees without
  editing `policy.yaml`.

- **Doctor progressive-trust posture**: `specular doctor` reports which
  opt-in governance knobs are active (`risk.tiers`, `provenance.attested`,
  `provenance.protocol`) under Progressive trust / JSON `progressive_trust`.

- **Evidence list session/harness filters (P1 #9 depth)**: `specular evidence
  list --session <id>` (exact) and `--harness <substr>` (case-insensitive)
  filter Change Evidence Graph records by gate provenance; combinable with
  verdict/since/path/risk/limit.

- **Require attested provenance (P1 #10 depth)**: opt-in policy
  `provenance.attested: enforce` DENYs unattested trees (progressive trust).
  Independent of `protocol: enforce` (which stays idle when unattested).
  Soft-ALLOW via `--policy provenance`. Example:
  `examples/policy/provenance-require.yaml`.

- **Exception close / revoke (P1 #6 depth)**: `specular approvals close
  <id>` (alias `revoke`) early-ends an open exception by rewriting the
  YAML in place (`closed_at`/`closed_by` + clamp `expires_at`); soft-ALLOW
  stops on the next gate. Idempotent when already closed or expired.

- **Codex / Gemini native hooks (P1 #4 depth)**: `specular session integrate
  codex|gemini` (aliases `codex-cli` / `gemini-cli`) installs Stop /
  SessionEnd hooks that call `session attest` + `gate`; reference configs
  under `examples/agents/codex/` and `examples/agents/gemini/`.

- **APP protocol enforce (P1 #3/#10 depth)**: opt-in policy `provenance:
  protocol: enforce` DENYs when attested sessions lack valid sibling
  `.provenance.json` docs (or schema validation fails). Soft-ALLOW via
  exception `--policy provenance` (or session id / schema). Without the
  block, APP doc counts stay advisory. Example:
  `examples/policy/provenance-protocol.yaml`.

- **Stronger session provenance (P1 #10 depth)**: gate discovers sibling
  `.provenance.json` APP docs, reports `Protocol` schema with docs/ok counts
  on the board/markdown/explain, and notes `APP docs N/M ok` when attested.

- **AI CHANGE RECORD risk + soft-ALLOW (P1 #2 depth)**: `specular explain` /
  `evidence show` human layout includes Risk (level/enforced/required) and
  Approvals Overruled soft-ALLOW lines; Why reflects exception overrules.

- **Jenkins / generic CI gate depth (P1 #8)**: focused `Jenkinsfile.gate`
  (markdown + SARIF + brownfield policy soft-skip); `generic-ci.sh` and the
  full `Jenkinsfile` gate stage omit `--policy` when the file is missing.

- **GitLab MR gate depth (P1 #7)**: `examples/ci-cd/gitlab-gate.yml` focused
  gate job (markdown + SARIF + brownfield policy soft-skip) with MR note
  **upsert** on `## Specular Change Control`; full `gitlab-ci.yml` comment
  job updated to upsert instead of stacking notes.

### Changed

- **CI / toolchain Go 1.26**: workflows pin `go-version: '1.26'`; `go.mod`
  requires `go 1.26.0` with `toolchain go1.26.8`; `golang.org/x/crypto`
  bumped to v0.57.0 (clears the deferred Go 1.26-compatible OSV path).

### Added

- **Evidence list `--risk` filter (P1 #9 depth)**: `specular evidence list
  --risk NONE|LOW|MEDIUM|HIGH|CRITICAL` filters Change Evidence Graph
  records by gate risk level (combinable with verdict/since/path/limit;
  empty risk treated as NONE).

- **Control pack check (P1 #5 depth)**: `specular policy pack check <id>`
  reports auditor-facing control→evidence presence for embedded pack
  artifacts (glob/file/dir); exit non-zero when required paths are missing;
  `--json` emits the report. Does not certify compliance.

- **Agent Provenance emit + verify (P1 #3 depth)**: `session attest` writes
  `.specular/sessions/<id>.provenance.json` beside the attestation;
  `specular provenance verify [id|path]` schema-checks the open
  `specular.provenance/v1` envelope (not signatures). `show`/`verify`
  prefer the sibling doc when present.

- **Scoped exception soft-ALLOW (P1 #6 depth)**: an open, non-expired
  exception can soft-ALLOW a gate DENY when `--policy` / `--scope` binds to
  that deny (drift finding code/path, failed policy check, or risk
  category/level). Gate board/JSON record `approvals.overrules`; underlying
  Drift/Policy FAIL status is preserved. Unmatched exceptions stay advisory.
  Risk role matching via `--policy`/`--scope` (P1 #1) is unchanged.

- **Risk-adaptive governance (P1 #1 depth)**: opt-in policy `risk:` tiers
  (`low`/`medium`/`high`/`critical`) map advisory path-heuristic levels to
  required approvals; missing roles DENY with Required/Observed/Missing on
  the gate board. Without `risk:`, Risk stays advisory (never flips verdict).
  Open exceptions satisfy a role via matching `--policy` / `--scope`.

- **Agent Provenance Protocol (P1 #3)**: `specular.provenance/v1` envelope
  in `internal/provenance` maps session `attestation.Provenance` into a
  stable document; `specular provenance show [id] [--json]` projects the
  latest (or named) session attestation; gate/evidence JSON gain an
  additive `provenanceProtocol` ref when attested — not a control plane

- **Native agent hooks (P1 #4)**: `specular session integrate <harness>`
  installs Claude Code Stop / Cursor `stop` hooks that call `session attest`
  + `gate` (`--dry-run` / `--force` / `--json`); `session start` exports
  `SPECULAR_SESSION_ID` / `SPECULAR_SESSION_HARNESS`; reference configs under
  `examples/agents/` (`claude-code/`, `cursor/`)

- **Executable control packs CLI**: `specular policy pack list|show|apply`
  over embedded `internal/policylibrary` seeds (id/title/summary list,
  human + `--json` show, apply with `--dry-run` / `--force`) —
  PRODUCT_INTENT §15 / P1 #5; does not claim compliance certification

- **Product intent + `specular gate`**: canonical
  [`docs/PRODUCT_INTENT.md`](docs/PRODUCT_INTENT.md) defines Specular as the
  trust boundary for AI-authored changes; `specular gate` is the thin
  ALLOW/DENY change-control primitive (change + provenance + soft-skip
  drift/policy on brownfield; `--strict-spec` to require specs)

- **Change Evidence Graph (v1) + `specular explain`**: gate persists
  `specular.evidence/v1` records under `.specular/evidence/`;
  `specular explain` / `specular evidence show|list` answer why ALLOW/DENY

- **`specular baseline`**: capture/show/status for an explicit acknowledged
  current state (`.specular/baseline.yaml`) — PRODUCT_INTENT §25 progressive
  trust without claiming the state is good

- **Gate PR/check UX**: `specular gate --format markdown` (stable
  `## Specular Change Control` marker) and `--github-annotations` for Checks
  file annotations; example workflow uses step summary + upsert comment + SARIF

- **Secret-safe attestations**: session/auto attestation goals are redacted
  before signing (`[REDACTED]` + `goalDigest`); secrets never enter
  `session wait --bundle` evidence packs (PRODUCT_INTENT §8/§29)

- **Deterministic gate findings**: sort drift findings and provenance
  session/harness lists so identical inputs yield identical Reason, JSON,
  and evidence IDs (PRODUCT_INTENT §6.7 / P0 #9)

- **Brownfield `specular init`**: Detected / Existing controls / Recommended
  baseline board (PRODUCT_INTENT §24), gate-first next steps, and
  `.github/workflows/examples/specular-gate.yml` PR-check example

- **PCI DSS 6.4.5 policy library seed**: open control→evidence mapping for
  significant-change approval (`specular policy library install pci-dss-6.4.5`)
  — closes the payments/fintech gap already cited in GTM security personas

### Changed

- **`session status --json`**: emits `{summary, sessions}` board (counts +
  records) instead of a bare session array — dashboard-friendly; use
  `session list --json` for the raw array

### Added

- **Session cherry-pick**: `session cherry-pick <into> --from <src>` applies
  a source session HEAD (or `--sha`) into another worktree for dependsOn
  handoff; aborts and reports conflicts

- **Fleet proof (evidence → land)**: extend `session-fleet-bundle.yml` with
  `--stop` + commit/`sync --fetch` land steps; add
  `examples/cicd-github-actions/proof.sh` one-shot local proof; README
  quick proof now covers fleet → bundle → land

- **Session sync --fetch**: `session sync --fetch [--remote]` refreshes
  remotes first and defaults onto `origin/<base>` so fleets rebase onto
  the remote tip instead of a stale local main

- **Fleet abort**: `session stop [ids...] [--all]` and `wait --timeout … --stop`
  kill still-running sessions on timeout (Xirp grid kill-all analogue);
  `restart --no-governed` keeps opt-out across restarts when policy.yaml exists

- **Session merge (local land)**: `session merge [--into] [--ff-only|--no-ff]`
  merges the worktree branch into the primary checkout — land path without
  requiring `gh` (complements `push --pr`)

- **Session push / PR**: `session push [--pr]` publishes the worktree
  branch (`git push -u`) and optionally opens a PR via `gh` with
  harness/goal provenance — closes fleet → evidence → land

- **Governed native sessions**: `session start --governed` (and
  `batch`/`restart`/manifest `governed:`) launches Claude/Codex without
  skip-permissions/full-auto, prepends a Specular governance preamble
  (deny-tools from `.specular/policy.yaml`), and records
  `provenance.governed` on attestations
- **Auto-governed**: native harness starts auto-enable governed when
  `.specular/policy.yaml` (or `policies.yaml`) is present; opt out with
  `--no-governed` / manifest `noGoverned: true`
- **Fleet→evidence CI example**: `examples/cicd-github-actions/session-fleet-bundle.yml`
  plus README quick proof (`policy library` → `session batch --governed` →
  `wait --bundle`); `session status`/`show`/`list` surface the governed flag

- **Session evidence bundle**: `session wait --bundle` packages
  attestations + drift SARIF (+ `--policy` library fragments) into
  `session-evidence.sbundle.tgz` after wait (implies `--gate`; attests
  waited sessions when present) — one-command fleet→auditor packet

- **Open policy library seed (governance moat)**
  - Embedded control mappings: SOC 2 CC8.1, ISO/IEC 42001 Clause 8,
    EU AI Act Art. 17, NIST AI RMF GOVERN 4 (`internal/policylibrary/seed/`)
  - `specular policy library list|show|install` (free — not Pro-gated)
  - Installs to `.specular/policies/<id>.yaml` for `bundle create --policy`
  - Closes the GTM "policy library" claim that previously had no repo files

- **Both-loops session management (response to Xirp)**
  - Specular now owns the **inner loop** as well as the outer gate:
    `specular session start|batch|list|show|status|wait|open|restart|rm|prune|diff|exec|commit|sync|attest|stop|logs|fork|harnesses`
  - **Native harness launch**: Claude Code, Codex, and Gemini run in
    isolated worktrees (not just provenance labels on `specular-auto`)
  - **Live session board**: `session status [--watch]` plus harness PATH
    probe via `session harnesses`; `session open` for worktree `cd`/`$EDITOR`
  - **Scriptable parallel gate**: `session wait [--any] [--timeout]` then drift
  - **Harness swap**: `session restart --harness …` reuses the worktree
  - **Lifecycle cleanup**: `session rm` / `session prune` tear down records,
    logs, exit sidecars, and worktrees after parallel fleets finish
  - **Session Git diff**: `session diff` shows worktree changes vs base or
    another session (Xirp changes-panel analogue)
  - **Fleet manifest launch**: `session batch` / `session start --manifest`
    starts many harness sessions from YAML/JSON (CI-native vs Xirp Mac grid)
  - **Fleet dependsOn**: manifest entries can wait on parent sessions
    (`queued` until parents `completed`; failed parents abort the chain)
  - **Session exec**: `session exec <id> -- <cmd>…` runs commands in the
    worktree with exit-code passthrough (CI substitute for Xirp's PTY)
  - **Session commit**: `session commit` lands worktree changes with a
    provenance-aware message (id/harness/goal)
  - **Session sync**: `session sync` rebases/merges the worktree onto base
    when main moves (conflicts abort + report paths)
  - **Session attest**: `session attest` / `wait --attest` writes signed
    attestations with harness + worktree provenance for native harnesses
  - **Session gate**: `session wait --gate` runs outer-loop drift after wait
    (and optional `--attest`) with fail-on-drift (exit 4) — fleet→evidence
    proof in one command
  - Session fork + log follow for multi-agent operations
  - GTM repositioned to **compete** with Xirp on sessions and win on
    governance (`docs/gtm/competitive/xirp.md`)
  - New `internal/session` registry under `.specular/sessions/`

- **Competitive response to Spotify Xirp (worktree + provenance)**
  - GTM brief at `docs/gtm/competitive/xirp.md` (threat model, talking points)
  - Anti-positioning rows for agentic session managers and software catalogs
  - Objection #10 for Xirp / Portal adoption in `docs/gtm/playbooks/objection-handling.md`
  - `specular worktree` CLI for Git worktree isolation (`.specular/worktrees/<name>`)
  - `specular auto --worktree <name>` runs autonomous mode in an isolated checkout
  - `specular auto --harness <label>` plus attestation `provenance.harness` /
    `worktreePath` / `worktreeBranch` / `worktreeName` for auditor-ready attribution

- **Release Automation (M8.1)**
  - Relicta integration for release orchestration
  - New Makefile targets: release-plan, release-bump, release-notes, release-evaluate
  - Full workflow automation: release-validate, release-approve, release-publish
  - Dry-run support for safe release previews

- **Performance Optimization (M8.2)**
  - Startup time optimization with fast command detection
  - Skip observability initialization for version, help, completion commands
  - Reduced startup time from ~700ms to <10ms for fast commands
  - New benchmark framework in `internal/benchmark/`
  - Makefile targets: bench, bench-startup, bench-binary, perf-report
  - Performance documentation in `docs/PERFORMANCE.md`

- **Sample Projects (M8.4)**
  - Mobile Backend example with Go + Firebase integration
  - Push notifications, real-time sync, authentication, file storage
  - Complete spec.yaml demonstrating api-service template

- **Security Enhancements (M8.6)**
  - Guard‑rail tool to block direct `exec.Command*` usage and unsafe file writes
  - Updated Makefile to run guard‑rail as part of the test suite
  - Added `SECURITY_RISKS.md` with high/medium/low risk register and mitigation status
  - Linked security‑risk register in README quick links

- **GTM Wedge and Role-Based Launch Assets (M9.5)**
  - Focused wedge positioning around policy-enforced AI development in CI/CD
    with auditable drift gates (`docs/gtm/ci-cd-policy-enforcement.md`)
  - Role-based documentation tracks for Platform Engineering and Security
    buyers (`docs/gtm/personas/`)
  - 30/60/90-day pilot playbooks tuned for each persona
    (`docs/gtm/playbooks/pilot-platform-engineering.md`,
    `docs/gtm/playbooks/pilot-security.md`)
  - Objection-handling cheatsheet covering the seven most common buyer
    objections (`docs/gtm/playbooks/objection-handling.md`)
  - Compliance-framework mapping (SOC 2, ISO 42001, EU AI Act, NIST AI RMF,
    PCI DSS) onto Specular evidence artifacts
  - Surfaced the GTM wedge from the README quick-links

- **Activation and AI Trust Telemetry (M9.4)**
  - New OpenTelemetry instruments for the activation funnel: `specular.activation.step`
    (counter keyed by step + status) and `specular.activation.duration`
    (histogram, milestones `init_complete` and `first_success`)
  - `specular init` records funnel events at started, context_detected,
    config_written, providers_configured, completed, and abandoned, enabling
    setup drop-off analysis without code changes
  - Cross-session time-to-first-success tracked via `.specular/.activation.json`
    marker; the next successful non-init command emits the duration metric
  - AI trust signals exposed via `specular.ai_trust.routing_decision` (counter
    with provider, model, hint, reason, cost_band) and
    `specular.ai_trust.routing_cost_estimate` (histogram in USD) — captures
    explainability for every router selection
  - `specular.ai_trust.intervention` records human-in-the-loop approvals and
    rejections from the auto-mode plan gate and the `approve` subcommand
  - `specular.ai_trust.regenerate` recorder ready for upcoming regeneration flows
  - Wired `InitMetricsProvider` into the observability bootstrap so OTLP metrics
    actually export when `SPECULAR_TELEMETRY=on` is set


### Fixed

- Quickstart demo mode now demonstrates workflow and runs doctor
- Fixed broken documentation links (API_REFERENCE.md, best-practices.md)
- Fixed flag shorthand conflict (`-v`) in plugin install command

### Changed

- Updated dependencies to latest versions
- Go upgraded from 1.24 to 1.25
- OpenTelemetry upgraded to v1.39.0
- Various security and performance dependency updates

## [1.6.0] - 2025-11-20

### Added

- **Interactive Plan Review TUI**
  - Full BubbleTea-based terminal UI for reviewing execution plans
  - Two-view system: list view for overview, detail view for task inspection
  - Vim-style navigation (j/k, h/l, enter, esc)
  - Approve/reject workflow with rejection reason prompt
  - Auto-approve empty plans for convenience
  - Styled with lipgloss for professional appearance
  - Comprehensive test suite with 11 tests

- **Platform API Client v2.0**
  - Production-grade HTTP client for Specular Platform integration
  - Configurable retry logic with exponential backoff
  - Smart retry strategy: retries 5xx errors, fails fast on 4xx
  - Context propagation for request cancellation
  - Structured APIError type with request ID tracking
  - Three endpoints: Health, GenerateSpec, GeneratePlan
  - Comprehensive test suite with 14 tests including retry scenarios

- **Plugin System Enhancements**
  - Plugin installation from local directories
  - Plugin installation from GitHub repositories
  - Automatic dependency resolution

- **Build System Improvements**
  - ImageCache support in autonomous mode executor
  - Manifest loading and validation for build approval
  - Plan task counting and build manifest loading
  - Actual version tracking from builds instead of hardcoded values

- **License & Documentation**
  - PRO tier gates for bundle commands
  - Step-by-step tutorial guides for PRO features
  - Tutorial documentation for advanced workflows

- **GitHub Action for CI/CD Integration (M7)**
  - Composite GitHub Action for seamless CI/CD integration
  - Four commands: drift, eval, build, plan
  - Multi-provider AI support (Anthropic, OpenAI, Google)
  - Automatic SARIF upload to GitHub Code Scanning
  - Platform auto-detection (Linux/macOS, AMD64/ARM64)
  - Rich job summaries and PR comment integration
  - Comprehensive documentation (.github/ACTION_README.md)
  - Integration with existing example workflows

### Fixed

- **Security**: Updated golang.org/x/crypto from v0.43.0 to v0.45.0 (fixes 2 moderate CVEs)
- **Concurrency**: Resolved race condition in DefaultLogger
- **Code Quality**: Fixed import grouping and type assertions in TUI code
- **Linting**: Resolved golangci-lint errors (errcheck, govet shadow, goimports)
- **Policy**: Implemented policy hash change detection for approval workflow

### Changed

- **Build Artifacts**: Added specular build artifacts to .gitignore

## [1.2.0] - 2025-11-17

### Major Changes

This release implements **ADR-0010: Governance-First CLI Redesign**, restructuring the CLI around governance, policy, and approval workflows while maintaining full backward compatibility.

### Added

- **Governance Commands** (NEW)
  - `governance init` - Initialize .specular workspace with governance structure
  - `governance doctor` - Comprehensive governance health checks
  - `governance status` - Display governance workflow status
  - Workspace structure: approvals/, bundles/, traces/, policies.yaml, providers.yaml

- **Policy Management Commands** (NEW)
  - `policy init` - Initialize policy configuration with templates
  - `policy validate` - Validate policies with strict mode support
  - `policy approve` - Approve policy changes with audit trail
  - `policy list` - List all policies with metadata
  - `policy diff` - Compare policy versions

- **Approval Workflow Commands** (NEW)
  - `approval approve` - Approve plans, builds, or drifts with role verification
  - `approval list` - List all approval records with filtering
  - `approval pending` - Show pending approvals requiring action

- **Bundle Command Enhancements**
  - `bundle create` - Create bundles (replaces `bundle build`)
  - `bundle gate` - Quality gate checks (replaces `bundle verify`)
  - `bundle inspect` - Inspect bundle contents
  - `bundle list` - List all bundles with metadata
  - Backward compatibility: `bundle build` and `bundle verify` still work

- **Plan Command Enhancements**
  - `plan create` - Generate plans (replaces `plan gen`)
  - `plan visualize` - Visualize task dependencies
  - `plan validate` - Validate plan structure
  - Backward compatibility: `plan gen` still works with deprecation warning

- **Drift Commands** (PROMOTED)
  - `drift check` - Run drift detection (promoted from `plan drift`)
  - `drift approve` - Approve detected drift
  - Backward compatibility: `plan drift` still works

- **Provider Enhancements**
  - `provider add` - Add providers dynamically (ollama, anthropic, openai, claude-code, gemini-cli, codex-cli, copilot-cli)
  - `provider remove` - Remove providers from configuration
  - `provider doctor` - Health checks (renamed from `provider health`)
  - Backward compatibility: `provider health` still works as alias

- **Unified Doctor Command**
  - Top-level `doctor` command with comprehensive system health checks
  - Governance health checks: workspace, policies, providers config, bundles, approvals, traces
  - Container runtime, AI providers, git, and project validation
  - JSON/YAML output support for automation

### Changed

- **CLI Structure**: Reorganized commands around governance workflows
  - Build commands: `build run`, `build verify`, `build approve`, `build explain`
  - Plan commands now use `create` instead of `gen` as primary command
  - Bundle commands use idiomatic names (`create`, `gate` instead of `build`, `verify`)
  - Provider commands enhanced with add/remove capabilities

- **Command Naming**: More idiomatic and consistent across the CLI
  - `gen` → `create` (for plan generation)
  - `build` → `create` (for bundle creation)
  - `verify` → `gate` (for quality gates)
  - `health` → `doctor` (for diagnostics)

- **Deprecation Warnings**: Added helpful deprecation messages for renamed commands
  - Commands show migration path to new structure
  - All deprecated commands remain functional with backward compatibility

### Fixed

- **Backward Compatibility**: Fixed nil pointer dereferences when using deprecated command forms
  - Safely handle missing flags on root commands
  - Proper default values for optional flags
  - Tested with comprehensive E2E test suite (9/10 passing)

- **Build Command**: Fixed nil pointer bugs in `runBuildRun` for flags: resume, checkpoint-dir, checkpoint-id, feature, verbose, enable-cache, cache-dir, cache-max-age, keep-checkpoint

### Documentation

- **ADR-0010**: Complete governance-first CLI redesign specification
- **CLI Reference**: Updated for new command structure (pending)
- **Migration Guide**: Backward compatibility and migration paths documented

### Statistics

- 12 new governance/policy/approval commands
- 7 enhanced provider commands
- 5 refactored build commands
- 5 refactored plan commands
- 4 refactored bundle commands
- Full backward compatibility maintained
- 9/10 E2E tests passing
- All unit tests passing

## [1.1.0] - 2025-11-16

### Added

- **Public SDK**: Domain types (FeatureID, TaskID, Priority) now available in `pkg/specular/types/` for external integrations
  - Enables third-party tools to use Specular's type system
  - Comprehensive test coverage with property-based tests
  - Full API documentation and validation rules

### Changed

- **SDK Architecture**: Migrated domain types from `internal/domain/` to public SDK (`pkg/specular/types/`)
  - Established one-way dependency: internal packages now import from SDK
  - Eliminated 201 lines of duplicate code
  - Net reduction of 198 lines across 40 files
  - Single source of truth for core domain types
- **CI/CD**: Updated GitHub Actions and integration tests to use Go 1.22 for compatibility

### Documentation

- **ARCHITECTURE.md**: Expanded with detailed directory structure and component descriptions
- **OPEN_SOURCE_PRACTICES.md**: Added comprehensive best practices documentation for contributing
- **SDK README**: Complete documentation for public SDK usage

### Fixed

- **Build**: Removed example workflow and applied consistent code formatting
- **.gitignore**: Properly respect .gitignore for docs/adr/ directory
- **GoReleaser**: Fixed SBOM configuration to prevent duplicate upload errors

### Statistics

- 13 commits since v1.0.0
- 40 files modified in SDK migration
- 849 lines of production SDK code
- Zero breaking changes - fully backward compatible

## [1.0.1] - 2025-11-06

### Fixed

- Minor bug fixes and improvements
- Documentation updates

## [1.0.0] - 2025-11-06

### Core Features

- **Specification-driven Development**: YAML-based feature specifications with lock files
- **Docker-based Sandboxed Execution**: Policy enforcement and secure execution environment
- **Multi-layer Drift Detection**: Plan, code, and infrastructure drift detection
- **Multi-LLM Provider Support**: Anthropic, OpenAI, Gemini, Ollama integration
- **Checkpoint/Resume**: Long-running operation support with state management
- **SARIF Reporting**: Standardized drift reporting format
- **Policy Enforcement**: Docker image allowlisting and execution policies
- **Interview Mode**: Guided Q&A for generating best-practice specifications with interactive TUI
- **Plan Generation**: AI-powered task decomposition and planning
- **Comprehensive Testing**: E2E test coverage across all components

### Autonomous Mode

- **Complete autonomous agent implementation** with 14 major features:
  - Profile system with environment-specific configurations (default, ci, production, strict)
  - Structured action plan format with JSON/YAML serialization
  - Standardized exit codes (0-6) for CI/CD integration
  - Per-step policy checks with context-aware enforcement
  - JSON output format for machine-readable results
  - Scope filtering for feature and path-based execution
  - Max steps limit with configurable safety guardrails
  - Interactive TUI with real-time progress visualization
  - Trace logging with comprehensive execution tracking
  - Patch generation with rollback support and safety verification
  - Cryptographic attestations with ECDSA P-256 signatures and SLSA compliance
  - Explain routing command for routing strategy analysis
  - Hooks system with built-in Script, Webhook, and Slack hooks
  - Advanced security with credential management, audit logging, and secret scanning

### Smart Diagnostics & UX

- **Doctor Command**: Comprehensive system health checks
  - Container runtime detection (Docker, Podman)
  - AI provider detection (Ollama, Claude, OpenAI, Gemini, Anthropic)
  - Language/framework detection (7 languages, 6 frameworks)
  - Git repository context
  - CI environment detection (6 CI systems)
  - Project structure validation
  - Dual output formats: colored text and JSON for automation
  - Actionable next steps based on system state

- **Route Command**: Intelligent routing with five subcommands
  - `route show` - Display routing configuration and model catalog
  - `route test` - Test model selection without provider calls
  - `route explain` - Detailed selection reasoning and cost estimates
  - `route optimize` - Historical routing analysis and cost optimization recommendations
  - `route bench` - Model performance benchmarking and comparison
  - Model catalog with 10 models across 3 providers
  - Support for routing hints: `codegen`, `agentic`, `fast`, `cheap`, `long-context`

- **Init Command**: Smart project initialization
  - 5 project templates (web-app, api-service, cli-tool, microservice, data-pipeline)
  - Automatic environment detection and configuration
  - Provider strategy selection (local, cloud, hybrid)
  - Governance level support (L2, L3, L4)
  - Interactive and non-interactive modes

- **Enhanced UX**:
  - CI-safe interactive prompts for missing required flags
  - Smart path defaults for all file operations
  - Enhanced error messages with actionable recovery suggestions
  - Structured error handling with hierarchical error codes
  - Standardized exit codes for CI/CD integration
  - Environment detection that automatically disables prompts in CI/CD

### CLI Provider Protocol

- **Language-agnostic protocol** for custom AI providers
  - JSON-based stdin/stdout communication
  - Three required commands: generate, stream, health
  - Support for multiple CLI-based providers (Claude Code, Codex, Gemini)
  - Comprehensive documentation and router configuration examples

### CI/CD Integration

- **GitHub Actions Integration**:
  - Composite GitHub Action for seamless CI/CD integration
  - SARIF drift report upload for GitHub Security tab
  - Failure annotations on pull requests with drift/policy violations
  - Docker image caching with 80%+ performance improvement
  - Comprehensive example workflows

- **Platform Support**:
  - GitHub Actions workflow examples
  - GitLab CI pipeline configuration
  - CircleCI workflow
  - Jenkins pipeline
  - Multi-stage workflows (validate, plan, build, evaluate, report)
  - Artifact management and caching strategies

### Production Readiness

- **Deployment Patterns**: Single binary, containerized, Kubernetes
- **Security Hardening**: Secret management, Docker security, network isolation, audit logging
- **Performance Tuning**: Docker caching, profile optimization, cost optimization
- **Monitoring & Observability**: Prometheus metrics, OpenTelemetry tracing, structured logging, alerting
- **Disaster Recovery**: Backup strategy, recovery procedures, checkpoint recovery
- **Troubleshooting**: Common issues, debug mode, diagnostic bundles
- **Production checklist** for deployment validation

### Documentation

- **PRODUCTION_GUIDE.md** (1,043 lines): Complete production deployment guide
- **RELEASE_PROCESS.md** (636 lines): Comprehensive release management documentation
- **Best Practices Guide** (1,200+ lines): Workflows, policies, optimization, troubleshooting
- **Checkpoint/Resume Guide** (800+ lines): Complete checkpoint system documentation
- **Progress Indicators Guide** (750+ lines): Display modes, tracking, integration patterns
- **CLI Providers Documentation**: Complete CLI provider protocol specification
- Installation guides for all platforms (Linux, macOS, Windows)
- Architecture Decision Records (ADRs)
- API documentation and examples

### Deliverables

- Multi-platform binaries (Linux, macOS, Windows × AMD64/ARM64)
- Docker images with multi-architecture support
- Homebrew formula for macOS/Linux
- DEB/RPM packages for Linux distributions

### Statistics

- 8,100+ lines of production code
- 138+ tests with comprehensive coverage
- 6,500+ lines of documentation
- Zero security issues (gosec compliance)
- Production-ready quality across all components

[unreleased]: https://github.com/felixgeelhaar/specular/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/felixgeelhaar/specular/compare/v1.0.0...v1.1.0
[1.0.1]: https://github.com/felixgeelhaar/specular/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/felixgeelhaar/specular/releases/tag/v1.0.0
