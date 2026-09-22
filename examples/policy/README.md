# Policy examples

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

See `docs/CLI_REFERENCE.md` (gate / provenance).
