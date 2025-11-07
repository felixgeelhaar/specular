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
rm -f .specular/spec.yaml .specular/spec.lock.json .specular/policy.yaml plan.json drift.sarif
echo

# Step 1: Copy example files
echo "Step 1: Setting up spec and policy..."
cp .specular/spec.yaml.example .specular/spec.yaml
cp .specular/policy.yaml.example .specular/policy.yaml
echo "✓ Spec and policy copied"
echo

# Step 2: Validate spec
echo "Step 2: Validating spec..."
./ai-dev spec validate --in .specular/spec.yaml
echo

# Step 3: Generate SpecLock
echo "Step 3: Generating SpecLock..."
./ai-dev spec lock --in .specular/spec.yaml --out .specular/spec.lock.json
echo

# Step 4: Generate plan
echo "Step 4: Generating execution plan..."
./ai-dev plan --in .specular/spec.yaml --lock .specular/spec.lock.json --out plan.json
echo

# Step 5: Run build (dry-run mode)
echo "Step 5: Running build (dry-run)..."
./ai-dev build --plan plan.json --policy .specular/policy.yaml --dry-run
echo

# Step 6: Run drift detection
echo "Step 6: Running drift detection..."
./ai-dev eval --plan plan.json --lock .specular/spec.lock.json --report drift.sarif
echo

# Verify generated files exist
echo "Verifying generated files..."
test -f .specular/spec.yaml && echo "✓ spec.yaml exists"
test -f .specular/spec.lock.json && echo "✓ spec.lock.json exists"
test -f .specular/policy.yaml && echo "✓ policy.yaml exists"
test -f plan.json && echo "✓ plan.json exists"
test -f drift.sarif && echo "✓ drift.sarif exists"
echo

echo "=== All tests passed! ==="
echo
echo "Generated files:"
echo "  .specular/spec.yaml       - Product specification"
echo "  .specular/spec.lock.json  - Hashed specification lock"
echo "  .specular/policy.yaml     - Policy enforcement rules"
echo "  plan.json             - Execution plan with task DAG"
echo "  drift.sarif           - Drift detection report"
