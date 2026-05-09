# Security Risk Register

| Severity | Component / File | Description | Mitigation / Status |
|----------|------------------|-------------|----------------------|
| **High** | `internal/apikey/scheduler.go:93` | TODO placeholder for API‑key rotation scheduling logic – rotation may never run, leaving keys stale. | Implement proper rotation schedule; currently flagged as missing functionality.
| **Medium** | `internal/hooks/builtin.go` | Script hooks run arbitrary local commands; powerful admin surface. Hardened env handling but still trusted‑admin. | Maintain strict admin‑only access, audit hook definitions, consider sandboxing.
| **Medium** | Provider CLI integrations (`providers/*/cli.go`) | External binaries (Claude, Gemini, Codex, Ollama) are invoked via `safeutil.SafeCommand`. Trust depends on binary provenance and PATH hygiene. | Enforce binary checksum verification in CI, document required version pins.
| **Low** | Test files (`*_test.go`) | Several `0644`/`0666` file writes used for fixture generation. Not production‑code. | No action needed; already scoped to tests.
| **Low** | `#nosec` / `nolint:gosec` comments | Suppressions exist for constant strings, logging, or known safe patterns. | Periodically review during security audits.

*The register is version‑controlled and should be updated any time a new risk is introduced or an existing one is resolved.*