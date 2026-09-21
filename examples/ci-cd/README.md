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
| [`gitlab-ci.yml`](./gitlab-ci.yml) | GitLab CI (MR gate + optional comment via `GITLAB_API_TOKEN`) |
| [`Jenkinsfile`](./Jenkinsfile) | Jenkins (credentials for provider API keys unchanged) |
| [`circleci-config.yml`](./circleci-config.yml) | CircleCI |
| [`generic-ci.sh`](./generic-ci.sh) | Any shell-based CI (Buildkite, Tekton, etc.) |
| [`github-actions-basic.yml`](./github-actions-basic.yml) | Legacy composite-action sample — prefer the gate workflow above |

## Sensible CI flags

```bash
specular gate --format markdown --policy .specular/policy.yaml --report drift.sarif
```

- **`--format markdown`** — stable `## Specular Change Control` body for MR/PR notes or job summaries
- **`--github-annotations`** — GitHub Checks only (see the dedicated GitHub example)
- **`--strict-spec`** — fail when spec/plan/lock are missing (opt in for greenfield)
- Keep provider tokens in your CI secret store; these examples do not hardcode them

Brownfield repos soft-skip missing drift/policy inputs unless `--strict-spec` is set.
