# Specular CI/CD examples

Vendor-agnostic samples that wire **`specular gate`** as the change-control
entrypoint (PRODUCT_INTENT gate-first), matching the GitHub PR check pattern in
[`.github/workflows/examples/specular-gate.yml`](../../.github/workflows/examples/specular-gate.yml).

Prefer `specular gate` in CI over calling `specular eval drift` directly. Gate
runs change discovery → provenance → drift (when specs exist) → policy →
ALLOW/DENY, and writes evidence under `.specular/evidence/`.

| Exit code | Meaning |
|-----------|---------|
| `0` | ALLOW |
| `3` | policy DENY |
| `4` | drift DENY |

## Files

| File | Platform |
|------|----------|
| [`gitlab-gate.yml`](./gitlab-gate.yml) | **GitLab MR gate (recommended)** — markdown + upsert note + SARIF |
| [`gitlab-ci.yml`](./gitlab-ci.yml) | GitLab full pipeline (validate/plan/build + gate) |
| [`Jenkinsfile.gate`](./Jenkinsfile.gate) | **Jenkins gate-only (recommended)** — markdown + SARIF + brownfield soft-skip |
| [`Jenkinsfile`](./Jenkinsfile) | Jenkins full pipeline (credentials for provider API keys unchanged) |
| [`circleci-config.yml`](./circleci-config.yml) | CircleCI |
| [`generic-ci.sh`](./generic-ci.sh) | Any shell-based CI (Buildkite, Tekton, etc.) — brownfield-safe |
| [`github-actions-basic.yml`](./github-actions-basic.yml) | Legacy composite-action sample — prefer the gate workflow above |

### GitLab MR gate (P1 #7)

Copy [`gitlab-gate.yml`](./gitlab-gate.yml) to `.gitlab-ci.yml` (or `include:` it).
Optional CI/CD variable `GITLAB_API_TOKEN` (api scope) upserts an MR note on
the stable `## Specular Change Control` marker — same UX as the GitHub
example (update in place instead of stacking notes). Without the token,
gate still runs and fails the pipeline on DENY.

Brownfield: missing `.specular/policy.yaml` soft-skips policy inside gate.

### Jenkins / generic CI (P1 #8)

Use [`Jenkinsfile.gate`](./Jenkinsfile.gate) for a focused PR/branch gate, or
[`generic-ci.sh`](./generic-ci.sh) inside any shell runner. Both omit
`--policy` when the file is missing (brownfield) and archive `gate.md` /
`drift.sarif` for reviewers.

## Sensible CI flags

```bash
# Policy-first (recommended): merge the combined ladder once
#   examples/policy/progressive-trust.yaml → .specular/policy.yaml
specular gate --format markdown --policy .specular/policy.yaml --report drift.sarif

# Or toggle progressive-trust knobs via CLI / CI vars (without editing policy):
specular gate --format markdown --require-attested --report drift.sarif
specular gate --format markdown --require-protocol --report drift.sarif
specular gate --format markdown --require-governed --report drift.sarif
```

- **`--format markdown`** — stable `## Specular Change Control` body for MR/PR notes or job summaries
- **`--github-annotations`** — GitHub Checks only (see the dedicated GitHub example)
- **`--strict-spec`** — fail when spec/plan/lock are missing (opt in for greenfield)
- **`--require-attested`** — DENY when no session attestations (opt in; same as
  `provenance.attested: enforce`). Wired via `REQUIRE_ATTESTED` in GitHub vars,
  GitLab CI variables, Jenkins params, and `generic-ci.sh`.
- **`--require-protocol`** — DENY when APP docs are missing/invalid/unbound
  (opt in; same as `provenance.protocol: enforce`; idle when unattested). Wired
  via `REQUIRE_PROTOCOL` in the same templates. Gate board prints
  `docs=N ok=M schema+bound`.
- **`--require-governed`** — DENY when attested sessions are not governed
  (opt in; same as `provenance.governed: enforce`; idle when unattested). Wired
  via `REQUIRE_GOVERNED` in the same templates.
- **Policy ladder** — [`examples/policy/progressive-trust.yaml`](../policy/progressive-trust.yaml)
  enables risk tiers + all three provenance knobs in one file (`specular doctor`
  points advisory Mode at it).
- Keep provider tokens in your CI secret store; these examples do not hardcode them

Brownfield repos soft-skip missing drift/policy inputs unless `--strict-spec` is set.
