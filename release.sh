#!/bin/bash
set -euo pipefail

# NothingLLM Release Script
# Full release flow: verify, test, validate, tag, document.
# Usage: ./release.sh <version>
#   e.g. ./release.sh 0.1.0-rc1

VERSION="${1:?Usage: ./release.sh <version>  (e.g. 0.1.0-rc1)}"
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || { echo "ERROR: not a git repo"; exit 1; })
BUILD_DATE=$(date -Iseconds)
TAG="v${VERSION}"

echo "=== NothingLLM Release: ${TAG} ==="

# 1. Verify clean working tree
echo "[1/7] Checking working tree..."
if [ -n "$(git status --porcelain)" ]; then
    echo "  ERROR: Working tree is not clean."
    echo "  Uncommitted changes:"
    git status --short
    exit 1
fi
echo "  Clean."

# 2. Run tests
echo "[2/7] Running tests..."
./scripts/test.sh
echo "  Tests passed."

# 3. Run docker validation (skip if docker unavailable)
echo "[3/7] Docker validation..."
if command -v docker &>/dev/null && docker info &>/dev/null; then
    ./scripts/test-docker.sh
    echo "  Docker validation passed."
else
    echo "  WARNING: Docker not available, skipping."
fi

# 4. Update VERSION file
echo "[4/7] Writing VERSION..."
echo "${VERSION}" > VERSION
echo "  VERSION: ${VERSION}"

# 5. Create git tag
echo "[5/7] Creating tag..."
if git rev-parse "${TAG}" &>/dev/null; then
    echo "  ERROR: Tag ${TAG} already exists."
    exit 1
fi
git tag -a "${TAG}" -m "Release ${VERSION} (${BUILD_DATE})"
echo "  Tag: ${TAG}"

# 6. Update CHANGELOG
echo "[6/7] Updating CHANGELOG..."
CHANGELOG="CHANGELOG.md"
if [ ! -f "${CHANGELOG}" ]; then
    echo "# Changelog" > "${CHANGELOG}"
    echo "" >> "${CHANGELOG}"
fi

TEMP=$(mktemp)
{
    echo "# Changelog"
    echo ""
    echo "## [${VERSION}] - $(date +%Y-%m-%d)"
    echo ""
    LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    if [ -n "${LAST_TAG}" ] && [ "${LAST_TAG}" != "${TAG}" ]; then
        git log --oneline "${LAST_TAG}..HEAD" 2>/dev/null | sed 's/^/  - /' || echo "  - (no commits)"
    else
        git log --oneline -10 | sed 's/^/  - /'
    fi
    echo ""
    tail -n +2 "${CHANGELOG}"
} > "${TEMP}"
mv "${TEMP}" "${CHANGELOG}"
echo "  CHANGELOG.md updated."

# 7. Print publish instructions
echo "[7/7] Done."
echo ""
echo "======================================"
echo "  Release ${TAG} prepared"
echo "======================================"
echo ""
echo "  Version:    ${VERSION}"
echo "  Commit:     ${GIT_COMMIT}"
echo "  Date:       ${BUILD_DATE}"
echo "  Tag:        ${TAG}"
echo ""
echo "  To publish:"
echo "    git push origin main --tags"
echo ""
echo "  To build release image:"
echo "    docker build \\"
echo "      --build-arg VERSION=${VERSION} \\"
echo "      --build-arg GIT_COMMIT=${GIT_COMMIT} \\"
echo "      --build-arg BUILD_DATE=${BUILD_DATE} \\"
echo "      -t nothingllm:${VERSION} ."
echo ""
echo "  To push to registry:"
echo "    docker tag nothingllm:${VERSION} registry.example.com/nothingllm:${VERSION}"
echo "    docker push registry.example.com/nothingllm:${VERSION}"