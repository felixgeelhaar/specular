#!/bin/bash
set -euo pipefail

# Publish Bundle to Docker Hub
# This script demonstrates how to publish a Specular bundle to Docker Hub with
# proper authentication, versioning, and tagging.

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

Publish a Specular bundle to Docker Hub.

Arguments:
  bundle-file   Path to the .sbundle.tgz file
  repo          Repository in format 'username/name'
  version       Version tag (e.g., v1.0.0)

Environment Variables:
  DOCKERHUB_USERNAME  Docker Hub username (required)
  DOCKERHUB_TOKEN     Docker Hub access token (required)

Examples:
  # Publish with environment variables
  DOCKERHUB_USERNAME=myuser DOCKERHUB_TOKEN=dckr_xxx \\
    ./publish-to-dockerhub.sh release.sbundle.tgz myuser/myapp v1.0.0

  # Publish from CI/CD
  DOCKERHUB_USERNAME=\${{ secrets.DOCKERHUB_USERNAME }} \\
  DOCKERHUB_TOKEN=\${{ secrets.DOCKERHUB_TOKEN }} \\
    ./publish-to-dockerhub.sh release.sbundle.tgz myuser/myapp \$VERSION

Create an access token at: https://hub.docker.com/settings/security
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

if [ -z "${DOCKERHUB_USERNAME:-}" ]; then
  echo -e "${RED}✗ DOCKERHUB_USERNAME environment variable is required${NC}"
  exit 1
fi

if [ -z "${DOCKERHUB_TOKEN:-}" ]; then
  echo -e "${RED}✗ DOCKERHUB_TOKEN environment variable is required${NC}"
  echo "Create a token at: https://hub.docker.com/settings/security"
  exit 1
fi

echo -e "${BLUE}=== Publishing Bundle to Docker Hub ===${NC}\n"

# Configuration
REGISTRY="docker.io"
IMAGE_NAME="${REGISTRY}/${REPO}"

# Step 1: Verify bundle
echo -e "${YELLOW}Step 1: Verifying bundle integrity...${NC}"
if specular bundle verify "$BUNDLE_FILE"; then
  echo -e "${GREEN}✓ Bundle verification successful${NC}\n"
else
  echo -e "${RED}✗ Bundle verification failed${NC}"
  exit 1
fi

# Step 2: Authenticate with Docker Hub
echo -e "${YELLOW}Step 2: Authenticating with Docker Hub...${NC}"
echo "$DOCKERHUB_TOKEN" | docker login "$REGISTRY" -u "$DOCKERHUB_USERNAME" --password-stdin 2>&1 | grep -v "WARNING"

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

# Step 4: Push bundle to Docker Hub
echo -e "${YELLOW}Step 4: Publishing bundle to Docker Hub...${NC}"

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
echo "Bundle successfully published to Docker Hub!"
echo ""
echo "Published tags:"
for TAG in "${TAGS[@]}"; do
  echo "  - $TAG"
done
echo ""
echo "Repository URL:"
echo "  https://hub.docker.com/r/${REPO}"
echo ""
echo "Pull the bundle:"
echo "  specular bundle pull ${IMAGE_NAME}:${VERSION}"
echo ""
echo "View on Docker Hub:"
echo "  https://hub.docker.com/r/${REPO}/tags"
echo ""

# Optional: Display rate limit information
read -p "Check Docker Hub rate limits? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  echo -e "${YELLOW}Docker Hub Rate Limits:${NC}"
  TOKEN=$(curl -s "https://auth.docker.io/token?service=registry.docker.io&scope=repository:ratelimitpreview/test:pull" | jq -r .token)
  curl -s --head -H "Authorization: Bearer $TOKEN" https://registry-1.docker.io/v2/ratelimitpreview/test/manifests/latest | grep -i ratelimit
fi

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
