#!/bin/bash
set -euo pipefail

# Multi-Approver Workflow - L3 Governance
# This script demonstrates a production-grade approval workflow where both
# Product Manager and Security Engineer must approve releases.

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Multi-Approver Workflow (L3 Governance) ===${NC}\n"

# Configuration
BUNDLE_FILE="release-l3.sbundle.tgz"
PM_USER="${PM_USER:-pm@company.com}"
PM_ROLE="pm"
SECURITY_USER="${SECURITY_USER:-security@company.com}"
SECURITY_ROLE="security"

# Step 1: Build bundle requiring multiple approvals
echo -e "${YELLOW}Step 1: Building bundle with multiple approval requirements...${NC}"
specular bundle build \
  --require-approval "$PM_ROLE" \
  --require-approval "$SECURITY_ROLE" \
  --governance-level L3 \
  --output "$BUNDLE_FILE"

echo -e "${GREEN}✓ Bundle created: $BUNDLE_FILE${NC}\n"

# Step 2: Check initial approval status
echo -e "${YELLOW}Step 2: Checking approval status (no approvals yet)...${NC}"
specular bundle approval-status "$BUNDLE_FILE"
echo ""

# Step 3: PM approves the bundle
echo -e "${YELLOW}Step 3: PM reviewing and approving...${NC}"
echo "  - Validating product requirements"
echo "  - Checking feature completeness"
echo "  - Verifying user acceptance criteria"

specular bundle approve "$BUNDLE_FILE" \
  --role "$PM_ROLE" \
  --user "$PM_USER" \
  --comment "Product requirements validated. All user stories complete."

echo -e "${GREEN}✓ PM approval granted${NC}\n"

# Step 4: Check approval status after first approval
echo -e "${YELLOW}Step 4: Approval status (1/2 approvals)...${NC}"
specular bundle approval-status "$BUNDLE_FILE"
echo ""

# Step 5: Security engineer approves
echo -e "${YELLOW}Step 5: Security Engineer reviewing and approving...${NC}"
echo "  - Security scan completed"
echo "  - Vulnerability assessment passed"
echo "  - Compliance requirements met"
echo "  - Code review completed"

specular bundle approve "$BUNDLE_FILE" \
  --role "$SECURITY_ROLE" \
  --user "$SECURITY_USER" \
  --comment "Security review passed. No critical vulnerabilities found."

echo -e "${GREEN}✓ Security approval granted${NC}\n"

# Step 6: Verify bundle with all approvals
echo -e "${YELLOW}Step 6: Verifying bundle (all approvals present)...${NC}"
if specular bundle verify "$BUNDLE_FILE"; then
  echo -e "${GREEN}✓ Bundle verification successful${NC}\n"
else
  echo -e "${RED}✗ Bundle verification failed${NC}\n"
  exit 1
fi

# Step 7: Check final approval status
echo -e "${YELLOW}Step 7: Final approval status (2/2 approvals)...${NC}"
specular bundle approval-status "$BUNDLE_FILE"
echo ""

# Step 8: Bundle is ready for production deployment
echo -e "${GREEN}=== Workflow Complete ===${NC}"
echo -e "Bundle ${BUNDLE_FILE} has all required approvals and is ready for production deployment."
echo ""
echo "Approval Summary:"
echo "  ✓ PM approval: $PM_USER"
echo "  ✓ Security approval: $SECURITY_USER"
echo "  ✓ All signatures verified"
echo "  ✓ Bundle integrity confirmed"
echo ""
echo "Next steps:"
echo "  1. Push to registry: specular bundle push $BUNDLE_FILE <registry>/<repo>:tag"
echo "  2. Apply to production: specular bundle apply $BUNDLE_FILE"
echo "  3. Create deployment record: specular bundle metadata $BUNDLE_FILE"
echo ""

# Optional: Display bundle diff
read -p "Show bundle diff from previous version? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]] && [ -f "previous-release.sbundle.tgz" ]; then
  echo -e "${YELLOW}Comparing with previous release...${NC}"
  specular bundle diff previous-release.sbundle.tgz "$BUNDLE_FILE"
fi

# Cleanup (optional)
read -p "Clean up bundle file? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  rm -f "$BUNDLE_FILE"
  echo -e "${GREEN}✓ Cleanup complete${NC}"
fi
