# Policy examples

## Progressive trust (combined ladder)

Copy or merge [`progressive-trust.yaml`](progressive-trust.yaml) into
`.specular/policy.yaml` for the full gate-first ladder: risk-adaptive
approvals plus `provenance.attested|protocol|governed: enforce`.

```bash
cp examples/policy/progressive-trust.yaml .specular/policy.yaml
# or merge the risk: / provenance: blocks into an existing policy

specular doctor          # Progressive trust → Mode: progressive
specular gate            # DENY when knobs fail
specular session integrate claude-code --enforce --require-governed --force
```

Individual knobs below can be enabled one at a time before adopting the
combined file.

## Risk-adaptive

Copy or merge [`risk-adaptive.yaml`](risk-adaptive.yaml) into
`.specular/policy.yaml` to enforce approvals by change-risk level.

```bash
# After a HIGH/CRITICAL change is DENY'd for missing security approval:
specular approve exception-auth-review \
  --reason "security reviewed auth change" \
  --policy security
specular gate
```

See `docs/CLI_REFERENCE.md` (gate) and `docs/PRODUCT_INTENT.md` §12–§13.

## APP protocol enforce

Copy or merge [`provenance-protocol.yaml`](provenance-protocol.yaml) into
`.specular/policy.yaml` so attested sessions must carry valid sibling
`.provenance.json` documents (`session attest` emits these).

```bash
# Soft-ALLOW a temporary missing APP doc:
specular approve exception-app-protocol \
  --reason "migrate attest hooks" \
  --policy provenance
specular gate
```

## Require attested provenance

Copy or merge [`provenance-require.yaml`](provenance-require.yaml) so the
gate DENYs unattested trees (progressive trust beyond Level 0–1). Protocol
enforce alone stays idle when unattested; combine both knobs when ready.

```bash
specular approve exception-brownfield \
  --reason "ramp attestation hooks" \
  --policy provenance
specular gate
```

## Require governed sessions

Copy or merge [`provenance-governed.yaml`](provenance-governed.yaml) so
attested sessions must have launched with `--governed` (safer native launch).
Idle when unattested — pair with `attested: enforce` when ready.

```bash
specular session start --harness claude-code --governed --name auth "…"
specular gate --require-governed
```

See `docs/CLI_REFERENCE.md` (gate / provenance).
