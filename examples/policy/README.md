# Risk-adaptive policy example

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
