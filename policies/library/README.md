# Policy library

Open framework → Specular evidence control mappings.

**Canonical seeds** (embedded in the CLI binary):

[`internal/policylibrary/seed/`](../../internal/policylibrary/seed/)

| ID | Framework | Control |
|----|-----------|---------|
| `soc2-cc8.1` | SOC 2 | CC8.1 Change Management |
| `iso42001-clause8` | ISO/IEC 42001 | Clause 8 Operation |
| `eu-ai-act-art17` | EU AI Act | Article 17 QMS |
| `nist-ai-rmf-govern4` | NIST AI RMF | GOVERN 4 |

```bash
specular policy pack list
specular policy pack show soc2-cc8.1
specular policy pack apply soc2-cc8.1 --dry-run
specular policy pack apply soc2-cc8.1
# equivalent legacy surface:
# specular policy library list|show|install …
specular session wait --attest --gate
specular bundle create --policy .specular/policies/soc2-cc8.1.yaml \
  --include .specular/sessions/*.attestation.json evidence.sbundle.tgz
```

## Contributing

Add a YAML fragment under `internal/policylibrary/seed/` with:

- `id`, `framework`, `control`, `title`, `summary`, `version`
- `artifacts[]` (`kind`, `path`, `why`)
- `evidence_commands[]`
- optional `compliance` map

Include unit coverage via the existing `Validate` checks (List loads every seed).
Keep mappings honest: map only to artifacts Specular actually produces.
