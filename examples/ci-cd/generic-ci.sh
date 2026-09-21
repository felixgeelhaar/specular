#!/usr/bin/env bash
# Generic CI snippet: Specular change-control gate
#
# Copy into Buildkite, Tekton, Azure Pipelines, or any shell-based runner.
# Prefer `specular gate` over `specular eval drift` as the CI entrypoint.
#
# Exit codes (pass through to the job):
#   0 = ALLOW
#   3 = policy DENY
#   4 = drift DENY
#
# Env (optional):
#   POLICY_FILE   default: .specular/policy.yaml
#   REPORT_FILE   default: drift.sarif
#   STRICT_SPEC   set to 1 to pass --strict-spec
#   GATE_MD       default: gate.md

set -u

POLICY_FILE="${POLICY_FILE:-.specular/policy.yaml}"
REPORT_FILE="${REPORT_FILE:-drift.sarif}"
GATE_MD="${GATE_MD:-gate.md}"

ARGS=(gate --format markdown --policy "$POLICY_FILE" --report "$REPORT_FILE")
if [ "${STRICT_SPEC:-0}" = "1" ]; then
  ARGS+=(--strict-spec)
fi

# Non-interactive. Do not pass --github-annotations outside GitHub Actions
# (see .github/workflows/examples/specular-gate.yml for that pattern).
set +e
specular "${ARGS[@]}" >"$GATE_MD"
code=$?
set -e

cat "$GATE_MD"

case "$code" in
  0) echo "Gate ALLOW" ;;
  3) echo "Gate DENY (policy)" ;;
  4) echo "Gate DENY (drift)" ;;
  *) echo "Gate failed (exit $code)" ;;
esac

exit "$code"
