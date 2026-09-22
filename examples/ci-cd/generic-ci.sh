#!/usr/bin/env bash
# Generic CI snippet: Specular change-control gate (P1 #8 depth)
#
# Copy into Buildkite, Tekton, Azure Pipelines, Jenkins sh step, or any
# shell-based runner. Prefer `specular gate` over `specular eval drift`.
#
# Exit codes (pass through to the job):
#   0 = ALLOW
#   3 = policy DENY
#   4 = drift DENY
#
# Env (optional):
#   POLICY_FILE        default: .specular/policy.yaml (omitted when missing — brownfield)
#   REPORT_FILE        default: drift.sarif
#   STRICT_SPEC        set to 1 to pass --strict-spec
#   REQUIRE_ATTESTED   set to 1 to pass --require-attested (progressive trust)
#   GATE_MD            default: gate.md

set -u

POLICY_FILE="${POLICY_FILE:-.specular/policy.yaml}"
REPORT_FILE="${REPORT_FILE:-drift.sarif}"
GATE_MD="${GATE_MD:-gate.md}"

ARGS=(gate --format markdown --report "$REPORT_FILE")
if [ -f "$POLICY_FILE" ]; then
  ARGS+=(--policy "$POLICY_FILE")
else
  echo "No $POLICY_FILE — gate will soft-skip policy (brownfield)."
fi
if [ "${STRICT_SPEC:-0}" = "1" ]; then
  ARGS+=(--strict-spec)
fi
if [ "${REQUIRE_ATTESTED:-0}" = "1" ] || [ "${REQUIRE_ATTESTED:-false}" = "true" ]; then
  ARGS+=(--require-attested)
  echo "Progressive trust: --require-attested enabled"
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
