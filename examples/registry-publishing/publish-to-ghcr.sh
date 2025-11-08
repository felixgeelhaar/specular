#!/bin/bash
set -euo pipefail

# Publish Bundle to GitHub Container Registry (GHCR)
# This script demonstrates how to publish a Specular bundle to GHCR with proper
# authentication, versioning, and tagging.

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Usage function
usage() {
  cat <<EOF
Usage: $0 <bundle-file> <repo> <version>

Publish a Specular bundle to GitHub Container Registry (GHCR).

Arguments:
  bundle-file   Path to the .sbundle.tgz file
  repo          Repository in format 'owner/name'
  version       Version tag (e.g., v1.0.0)

Environment Variables:
  GITHUB_TOKEN  GitHub Personal Access Token with write:packages scope (required)
  GITHUB_USER   GitHub username (optional, defaults to token owner)

Examples:
  # Publish with automatic tag detection
  GITHUB_TOKEN=ghp_xxx ./publish-to-ghcr.sh release.sbundle.tgz myorg/myapp v1.0.0

  # Publish with custom username
  GITHUB_USER=username GITHUB_TOKEN=ghp_xxx ./publish-to-ghcr.sh \\
    release.sbundle.tgz myorg/myapp v1.0.0

  # Publish from CI/CD
  GITHUB_TOKEN=\${{ secrets.GITHUB_TOKEN }} ./publish-to-ghcr.sh \\
    release.sbundle.tgz \${{ github.repository }} \${{ github.ref_name }}
EOF
  exit 1
}

# Check arguments
if [ $# -lt 3 ]; then
  usage
fi

BUNDLE_FILE="$1"
REPO="$2"
VERSION="$3"

# Validate inputs
if [ ! -f "$BUNDLE_FILE" ]; then
  echo -e "${RED}✗ Bundle file not found: $BUNDLE_FILE${NC}"
  exit 1
fi

if [ -z "${GITHUB_TOKEN:-}" ]; then
  echo -e "${RED}✗ GITHUB_TOKEN environment variable is required${NC}"
  echo "Create a token at: https://github.com/settings/tokens"
  echo "Required scopes: write:packages, read:packages"
  exit 1
fi

echo -e "${BLUE}=== Publishing Bundle to GHCR ===${NC}\n"

# Configuration
REGISTRY="ghcr.io"
IMAGE_NAME="${REGISTRY}/${REPO}"

# Get GitHub username if not provided
if [ -z "${GITHUB_USER:-}" ]; then
  echo -e "${YELLOW}Fetching GitHub username...${NC}"
  GITHUB_USER=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
    https://api.github.com/user | jq -r '.login')

  if [ -z "$GITHUB_USER" ] || [ "$GITHUB_USER" = "null" ]; then
    echo -e "${RED}✗ Failed to get GitHub username. Please set GITHUB_USER.${NC}"
    exit 1
  fi
  echo -e "${GREEN}✓ Using GitHub user: $GITHUB_USER${NC}\n"
fi

# Step 1: Verify bundle
echo -e "${YELLOW}Step 1: Verifying bundle integrity...${NC}"
if specular bundle verify "$BUNDLE_FILE"; then
  echo -e "${GREEN}✓ Bundle verification successful${NC}\n"
else
  echo -e "${RED}✗ Bundle verification failed${NC}"
  exit 1
fi

# Step 2: Authenticate with GHCR
echo -e "${YELLOW}Step 2: Authenticating with GHCR...${NC}"
echo "$GITHUB_TOKEN" | docker login "$REGISTRY" -u "$GITHUB_USER" --password-stdin 2>&1 | grep -v "WARNING"

if [ $? -eq 0 ]; then
  echo -e "${GREEN}✓ Authentication successful${NC}\n"
else
  echo -e "${RED}✗ Authentication failed${NC}"
  exit 1
fi

# Step 3: Prepare tags
echo -e "${YELLOW}Step 3: Preparing image tags...${NC}"

# Primary version tag
TAGS=("${IMAGE_NAME}:${VERSION}")

# Add semantic version tags
if [[ "$VERSION" =~ ^v?([0-9]+)\.([0-9]+)\.([0-9]+)(-[a-zA-Z0-9.]+)?$ ]]; then
  MAJOR="${BASH_REMATCH[1]}"
  MINOR="${BASH_REMATCH[2]}"
  PATCH="${BASH_REMATCH[3]}"
  PRERELEASE="${BASH_REMATCH[4]}"

  # Only add major/minor tags for stable releases (no prerelease)
  if [ -z "$PRERELEASE" ]; then
    TAGS+=("${IMAGE_NAME}:v${MAJOR}")
    TAGS+=("${IMAGE_NAME}:v${MAJOR}.${MINOR}")
    TAGS+=("${IMAGE_NAME}:latest")
  else
    echo -e "  ${YELLOW}Prerelease version detected, skipping major/minor/latest tags${NC}"
  fi
fi

echo "  Tags to publish:"
for TAG in "${TAGS[@]}"; do
  echo "    - $TAG"
done
echo ""

# Step 4: Push bundle to GHCR
echo -e "${YELLOW}Step 4: Publishing bundle to GHCR...${NC}"

for TAG in "${TAGS[@]}"; do
  echo -e "  ${BLUE}Pushing: $TAG${NC}"

  if specular bundle push "$BUNDLE_FILE" "$TAG"; then
    echo -e "  ${GREEN}✓ Successfully pushed: $TAG${NC}"
  else
    echo -e "  ${RED}✗ Failed to push: $TAG${NC}"
    exit 1
  fi
done
echo ""

# Step 5: Verify published bundle
echo -e "${YELLOW}Step 5: Verifying published bundle...${NC}"
TEMP_BUNDLE="$(mktemp -d)/verify.sbundle.tgz"

if specular bundle pull "${IMAGE_NAME}:${VERSION}" --output "$TEMP_BUNDLE"; then
  if specular bundle verify "$TEMP_BUNDLE"; then
    echo -e "${GREEN}✓ Published bundle verified successfully${NC}\n"
    rm -f "$TEMP_BUNDLE"
  else
    echo -e "${RED}✗ Published bundle verification failed${NC}"
    exit 1
  fi
else
  echo -e "${RED}✗ Failed to pull published bundle${NC}"
  exit 1
fi

# Step 6: Generate usage instructions
echo -e "${GREEN}=== Publication Complete ===${NC}"
echo ""
echo "Bundle successfully published to GHCR!"
echo ""
echo "Published tags:"
for TAG in "${TAGS[@]}"; do
  echo "  - $TAG"
done
echo ""
echo "Package URL:"
echo "  https://github.com/orgs/$(echo $REPO | cut -d/ -f1)/packages/container/$(echo $REPO | cut -d/ -f2)"
echo ""
echo "Pull the bundle:"
echo "  specular bundle pull ${IMAGE_NAME}:${VERSION}"
echo ""
echo "Make package public (if desired):"
echo "  gh api -X PATCH /user/packages/container/$(echo $REPO | cut -d/ -f2) \\"
echo "    -f visibility=public"
echo ""

# Optional: Display bundle metadata
read -p "Display bundle metadata? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  echo -e "${YELLOW}Bundle Metadata:${NC}"
  specular bundle metadata "$BUNDLE_FILE"
fi

# Cleanup
docker logout "$REGISTRY" >/dev/null 2>&1

echo -e "${GREEN}✓ Done${NC}"
