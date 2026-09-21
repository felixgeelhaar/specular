# Competitive brief: Spotify Xirp

> Specular is built to **win** the agent-session category against Xirp —
> not to sit politely beside it. We compete on parallel harness sessions
> *and* on the governance Xirp does not ship.

## What Xirp is

Xirp is Spotify's macOS desktop app for running Claude Code, Codex, and
Gemini sessions in parallel Git worktrees, with optional Portal catalog
context. It proved demand for vendor-neutral multi-agent orchestration.

## Where Specular competes — and wins

| Dimension | Xirp | Specular |
|-----------|------|----------|
| Parallel sessions | Desktop grid | `session start/batch/exec/commit/sync/attest/list/logs/fork/stop` |
| Harnesses | Claude Code, Codex, Gemini | **Same**, plus governed `specular-auto` |
| Isolation | Git worktrees | Git worktrees (`.specular/worktrees/`) |
| Platforms | **macOS only** | **Linux / macOS / Windows** |
| License | Proprietary + Portal upsell | **Apache 2.0** |
| Governance gate | None (transcripts, no redaction) | **Drift + policy + signed bundles** |
| Harness attribution | Transcript only | **`session attest` → `provenance.harness` + worktree fields** |
| Org context | Portal catalog (paid) | Spec + policy + ADR (in-repo) |

**Competitive thesis**

> Specular is the open, cross-platform control plane for parallel coding
> agents — with the only auditor-ready change-control gate in the category.

Xirp's desktop grid is a UX advantage on Mac. Everywhere else — and
everywhere compliance matters — Specular is the stronger product.

## Threat model (still take Xirp seriously)

1. **Desktop polish.** Grid + PTY is sticky for Mac-native teams. We do
   not ship a GUI yet; we win on CLI density, CI, and evidence.
2. **Portal distribution.** Spotify can attach sessions to Backstage
   commercial motion. Counter with open core + auditor enablement.
3. **Mindshare.** Keep saying "Claude Code / Codex / Gemini in worktrees
   *and* a drift gate" so buyers don't map the whole category to Xirp.

## Product posture

**Ship to compete**

- Native harness launch: `claude-code`, `codex`, `gemini`, `specular-auto`
- Worktree isolation per session
- `session status [--watch]` live board + `session open` worktree helper (`status --json` → `{summary,sessions}`)
- `session wait` scriptable parallel gate + `session restart` harness swap
- `session rm` / `session prune` lifecycle cleanup after fleets finish
- `session diff` worktree changes vs base or another session
- `session batch` / `session start --manifest` fleet launch (CI-native vs Mac grid)
- Manifest `dependsOn` for sequential pipelines (implement → review) without a second CI job
- `session cherry-pick --from` applies a peer session tip into another worktree (dependsOn handoff)
- `session exec` worktree command runner (CI substitute for Xirp's per-session PTY)
- `session commit` lands worktree changes with provenance-aware messages
- `session sync [--fetch]` rebases/merges worktrees onto base (fetch remote tip)
- `session push [--pr]` lands the branch (and optional GitHub PR)
- `session merge [--into]` lands the branch into the primary checkout (no gh)
- `session stop [ids…] [--all]` / `wait --timeout --stop` fleet abort
- `session attest` / `wait --attest` signed provenance for native harnesses
- `session wait --gate` fleet→evidence drift proof (fail-on-drift, exit 4)
- `session wait --bundle` one-command attest/gate/evidence packet
- `session start --governed` safer native launch + preamble + attest provenance
- Auto-governed when `.specular/policy.yaml` exists (`--no-governed` to opt out)
- CI example: `examples/cicd-github-actions/session-fleet-bundle.yml` + `proof.sh` (fleet→bundle→land)
- `policy library` open SOC 2 / ISO 42001 / EU AI Act / NIST AI RMF seeds
- `session harnesses` with PATH availability probe
- `session logs --follow`, `session fork`
- Harness + worktree provenance into attestations
- Drift / policy / bundle outer loop

**Do not clone**

- macOS-only GUI grid
- Portal marketplace / transcript social sharing

## Talking points

**Platform**

> "Same harnesses as Xirp — Claude, Codex, Gemini — in worktrees, on every
> OS, open source. Plus the CI gate Xirp never built. One tool."

**Security**

> "Xirp does not redact transcript uploads. Specular never requires
> uploading conversations. The evidence is a signed bundle in your repo."

**Against Portal lock-in**

> "Portal is a catalog. Specular is change control. If you already bought
> Portal, keep it — and still run Specular sessions so shipping stays
> auditable."

## Proof

```bash
specular session harnesses
# One-shot fleet (CI-native vs Xirp's Mac grid):
cat > fleet.yaml <<'EOF'
- name: demo
  harness: claude-code
  goal: Add /healthz
- name: demo-2
  harness: codex
  goal: Add rate limiting
- name: review
  harness: gemini
  goal: Review both changes
  dependsOn: [demo, demo-2]
EOF
specular session batch --governed fleet.yaml
specular session status --watch
specular policy library install soc2-cc8.1
specular session wait --bundle --policy .specular/policies/soc2-cc8.1.yaml demo demo-2 review
# Or abort a stuck fleet:
# specular session wait --timeout 45m --stop
# specular session stop --all
specular session cherry-pick review --from demo
specular session exec demo -- go test ./...
specular session diff demo --stat
specular session commit demo --all
specular session sync demo --fetch
specular session push demo --pr
specular session merge demo
specular auto verify .specular/sessions/demo.attestation.json
specular session diff demo --against demo-2
cd "$(specular session open demo)"
specular session restart demo --harness gemini --force
specular session logs demo --follow
specular session prune --delete-branch
```

## Sources

- [Introducing Xirp](https://portal.spotify.com/blog/introducing-xirp)
- [Xirp docs](https://backstage.spotify.com/docs/xirp)
- [Xirp FAQ](https://backstage.spotify.com/docs/xirp/faq) — macOS-only, proprietary, no transcript redaction
