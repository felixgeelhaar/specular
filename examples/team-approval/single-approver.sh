#!/bin/bash
set -euo pipefail

# Single Approver Workflow - L2 Governance
# This script demonstrates a basic approval workflow where a Product Manager
# must approve all releases before deployment.

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Single Approver Workflow (L2 Governance) ===${NC}\n"

# Configuration
BUNDLE_FILE="release-l2.sbundle.tgz"
PM_USER="${PM_USER:-pm@company.com}"
PM_ROLE="pm"

# Step 1: Build bundle requiring PM approval
echo -e "${YELLOW}Step 1: Building bundle with PM approval requirement...${NC}"
specular bundle build \
  --require-approval "$PM_ROLE" \
  --governance-level L2 \
  --output "$BUNDLE_FILE"

echo -e "${GREEN}✓ Bundle created: $BUNDLE_FILE${NC}\n"

# Step 2: Check initial approval status
echo -e "${YELLOW}Step 2: Checking approval status (before approval)...${NC}"
specular bundle approval-status "$BUNDLE_FILE"
echo ""

# Step 3: PM approves the bundle
echo -e "${YELLOW}Step 3: PM approving the bundle...${NC}"
specular bundle approve "$BUNDLE_FILE" \
  --role "$PM_ROLE" \
  --user "$PM_USER" \
  --comment "Product requirements validated. Ready for deployment."

echo -e "${GREEN}✓ Bundle approved by PM${NC}\n"

# Step 4: Verify bundle with approval
echo -e "${YELLOW}Step 4: Verifying bundle (includes approval verification)...${NC}"
if specular bundle verify "$BUNDLE_FILE"; then
  echo -e "${GREEN}✓ Bundle verification successful${NC}\n"
else
  echo -e "\033[0;31m✗ Bundle verification failed${NC}\n"
  exit 1
fi

# Step 5: Check final approval status
echo -e "${YELLOW}Step 5: Final approval status...${NC}"
specular bundle approval-status "$BUNDLE_FILE"
echo ""

# Step 6: Bundle is ready for deployment
echo -e "${GREEN}=== Workflow Complete ===${NC}"
echo -e "Bundle ${BUNDLE_FILE} is approved and ready for deployment."
echo ""
echo "Next steps:"
echo "  1. Push to registry: specular bundle push $BUNDLE_FILE <registry>/<repo>:tag"
echo "  2. Or apply locally: specular bundle apply $BUNDLE_FILE"
echo ""

# Cleanup (optional)
read -p "Clean up bundle file? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  rm -f "$BUNDLE_FILE"
  echo -e "${GREEN}✓ Cleanup complete${NC}"
fi
