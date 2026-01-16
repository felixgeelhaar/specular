# UX & Workflow Inventory

This report captures the verification status, security scan results, and the outstanding UX/process gaps that the `specular` CLI still exposes.

## 1. Verification & Health

- `cd cli && go test ./... -coverprofile=coverage.out` ran successfully; the generated `coverage.out` feeds `coverctl check`. The `core` domain reports **89.5%**, `execution` 44.7%, `security` 68.8%, `observability` 79.3%, `cli` 23.9%, and `sdk` 69.5% (all above threshold).  
- The `providers` domain is **73.4%**, which still misses the configured minimum of **75%** (`cli/.coverctl.yaml:15-39`). Improving this domain means adding regression tests for `internal/provider`, `internal/router`, and their helpers; the registry currently exercises only 70.6% of `internal/provider` code.

## 2. Security Scan Overview

- `verdict scan --include gosec --json` (current assessment: 174 findings) still reports:
  * **7 × G101 (hardcoded credentials)** rooted in `cli/internal/auth/errors.go:10-37` even though the constants already carry `//nolint:gosec`. We should explicitly baseline those fingerprints or move the strings out of Go code if we want to silence gosec.  
  * **32 × G204 (unvalidated subprocess args)** covering most of the provider executables (`providers/claude/main.go:144,218`, `providers/gemini/main.go:165,229`, `providers/codex/main.go:212`, `providers/ollama/main.go:178,281`), the plugin manager (`internal/plugin/manager.go:269,466,670`), detector helpers (`internal/detect/detect.go:113-219`), and the Docker/exec helpers (`internal/exec/docker.go:21`, `internal/exec/cache.go:277,319,378`). The fix path is to validate or sanitize those arguments before passing them to `exec.Command*` and to consider `exec.CommandContext` wrappers that trim user input.  
  * **23 × G306 (open files written with 0644)** triggered by CLI config/manifest writers (`internal/cmd/build.go:612`, `internal/cmd/approve.go:147`, `internal/cmd/config_validate.go:485`, `internal/cache/cache.go:299`, `internal/plugin/registry.go:219`, `internal/plugin/lockfile.go:116`, etc.). Whenever we write secrets, approvals, or cache metadata we should drop permissions to `0600` unless there is a compelling reason for broader access.  

## 3. UX & Workflow Gaps

### Manifest pipeline & tests

- `internal/exec/manifest.go` now returns a single `(string, error)` from `SaveManifest`, and `internal/exec/manifest_test.go` already matches that signature, so the old failure is resolved. Running `go test ./...` again proves the manifest helpers still pass (no regressions).  

### `.specular/providers.yaml` editing

- `provider list` and `provider doctor` (see `cli/internal/cmd/provider.go:50-190`) still expect humans to create/patch `.specular/providers.yaml` manually. The UX gap identified in `reports/specular-review.md:15-38` remains: unauthenticated users have to open the YAML file, find the right descriptor, and toggle `enabled: true`.  
- Given the descriptor-driven catalog already lives in `internal/provider`, we should surface it in the CLI (e.g., `provider add` could prompt with descriptor metadata or a `--enable-recommendations` flag). A minimal short-term improvement is to auto-populate a stub when no config exists and print the detected recommendations instead of just noding “Edit the file”.  

### Roady/spec/doc sync

- `roady status` still reports the **Release automation + telemetry** plan as approved but with five pending tasks and 18 drift issues (`roady drift detect` output summarized below). The unmatched tasks (`Implement 1.–10.`, `Document Gaps & Workarounds`, etc.) show the spec changed after the plan was generated.  
- Drift hints: run `roady plan prune` to drop the orphaned tasks or regenerate the plan; rerun `roady plan generate --ai` once the spec (or descriptors) stabilizes so the plan reflects the current set of features.

## 4. Next Steps

1. **Cover the providers domain**: add targeted tests for `internal/provider`, `internal/router`, and the descriptor catalog so `coverctl` stops failing on the `providers` domain.  
2. **Tiered security follow-up**: either baseline the G101 fingerprints in `.verdict/baseline.json` or refactor the error constants; sanitize all command arguments before invoking `exec.Command*`; adjust file writes flagged by G306 to use `0o600` (the emitter is many CLI helpers).  
3. **Provider UX demo**: build a lightweight helper (e.g., `specular provider init --recommendations`) that auto-writes `.specular/providers.yaml` from descriptor metadata and prints an annotated list of recommended channels, reducing the forced manual edit.  
4. **Roady drift cleanup**: rerun `roady plan prune` + `roady plan generate` to bring the execution plan back in sync with the spec, and document the new baseline so future runs do not trigger spec/plan mismatch noise.
