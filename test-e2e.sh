#!/bin/bash
#
# End-to-end integration test for ai-dev CLI
# Tests the complete workflow: spec → lock → plan → eval
#

set -e

echo "=== ai-dev End-to-End Integration Test ==="
echo

# Clean up previous test artifacts
echo "Cleaning up previous test artifacts..."
rm -f .aidv/spec.yaml .aidv/spec.lock.json .aidv/policy.yaml plan.json drift.sarif
echo

# Step 1: Copy example files
echo "Step 1: Setting up spec and policy..."
cp .aidv/spec.yaml.example .aidv/spec.yaml
cp .aidv/policy.yaml.example .aidv/policy.yaml
echo "✓ Spec and policy copied"
echo

# Step 2: Validate spec
echo "Step 2: Validating spec..."
./ai-dev spec validate --in .aidv/spec.yaml
echo

# Step 3: Generate SpecLock
echo "Step 3: Generating SpecLock..."
./ai-dev spec lock --in .aidv/spec.yaml --out .aidv/spec.lock.json
echo

# Step 4: Generate plan
echo "Step 4: Generating execution plan..."
./ai-dev plan --in .aidv/spec.yaml --lock .aidv/spec.lock.json --out plan.json
echo

# Step 5: Run build (dry-run mode)
echo "Step 5: Running build (dry-run)..."
./ai-dev build --plan plan.json --policy .aidv/policy.yaml --dry-run
echo

# Step 6: Run drift detection
echo "Step 6: Running drift detection..."
./ai-dev eval --plan plan.json --lock .aidv/spec.lock.json --report drift.sarif
echo

# Verify generated files exist
echo "Verifying generated files..."
test -f .aidv/spec.yaml && echo "✓ spec.yaml exists"
test -f .aidv/spec.lock.json && echo "✓ spec.lock.json exists"
test -f .aidv/policy.yaml && echo "✓ policy.yaml exists"
test -f plan.json && echo "✓ plan.json exists"
test -f drift.sarif && echo "✓ drift.sarif exists"
echo

echo "=== All tests passed! ==="
echo
echo "Generated files:"
echo "  .aidv/spec.yaml       - Product specification"
echo "  .aidv/spec.lock.json  - Hashed specification lock"
echo "  .aidv/policy.yaml     - Policy enforcement rules"
echo "  plan.json             - Execution plan with task DAG"
echo "  drift.sarif           - Drift detection report"
