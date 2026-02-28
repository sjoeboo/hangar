#!/usr/bin/env bash
#
# Hangar Release Script
#
# Creates and pushes a git tag to trigger the GoReleaser CI workflow.
#
# Usage:
#   ./scripts/release.sh
#   ./scripts/release.sh 1.2.3   # specify version directly
#
# Prerequisites:
#   - Clean working tree (no uncommitted changes)
#   - On the master branch
#   - CHANGELOG.md has an entry for the new version: ## [X.Y.Z]
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

# ── Safety checks ─────────────────────────────────────────────────────────────

# Must be in repo root
if [[ ! -f "go.mod" ]] || ! grep -q 'github.com/sjoeboo/hangar' go.mod 2>/dev/null; then
    echo -e "${RED}Error: run from the root of the hangar repo.${NC}"
    exit 1
fi

# Must be on master
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" != "master" ]]; then
    echo -e "${RED}Error: must be on master branch (currently on: $CURRENT_BRANCH).${NC}"
    exit 1
fi

# Must have clean working tree (checks modified, staged, and untracked files)
if [[ -n "$(git status --porcelain)" ]]; then
    echo -e "${RED}Error: working tree is not clean. Commit, stash, or remove untracked files first.${NC}"
    git status --short
    exit 1
fi

# ── Determine current version ─────────────────────────────────────────────────
CURRENT_TAG=$(git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 || true)
if [[ -z "$CURRENT_TAG" ]]; then
    CURRENT_VERSION="0.0.0"
else
    CURRENT_VERSION="${CURRENT_TAG#v}"
fi

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         Hangar Release                 ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "Current version: ${YELLOW}v${CURRENT_VERSION}${NC}"
echo ""

# ── Parse current version ─────────────────────────────────────────────────────
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"
MAJOR=${MAJOR:-0}; MINOR=${MINOR:-0}; PATCH=${PATCH:-0}

SUGGEST_PATCH="${MAJOR}.${MINOR}.$((PATCH + 1))"
SUGGEST_MINOR="${MAJOR}.$((MINOR + 1)).0"
SUGGEST_MAJOR="$((MAJOR + 1)).0.0"

# ── Determine new version ─────────────────────────────────────────────────────
if [[ -n "$1" ]]; then
    NEW_VERSION="${1#v}"
else
    echo -e "Select new version:"
    echo -e "  ${BOLD}1)${NC} patch  → v${SUGGEST_PATCH}  (bug fixes)"
    echo -e "  ${BOLD}2)${NC} minor  → v${SUGGEST_MINOR}  (new features, backwards compatible)"
    echo -e "  ${BOLD}3)${NC} major  → v${SUGGEST_MAJOR}  (breaking changes)"
    echo -e "  ${BOLD}4)${NC} custom"
    echo ""
    read -p "Choice [1]: " -r CHOICE < /dev/tty
    CHOICE="${CHOICE:-1}"

    case "$CHOICE" in
        1) NEW_VERSION="$SUGGEST_PATCH" ;;
        2) NEW_VERSION="$SUGGEST_MINOR" ;;
        3) NEW_VERSION="$SUGGEST_MAJOR" ;;
        4)
            read -p "Enter version (without 'v'): " -r NEW_VERSION < /dev/tty
            ;;
        *)
            echo -e "${RED}Invalid choice.${NC}"
            exit 1
            ;;
    esac
fi

# Validate format
if ! [[ "$NEW_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: invalid version format '${NEW_VERSION}'. Expected X.Y.Z${NC}"
    exit 1
fi

NEW_TAG="v${NEW_VERSION}"
echo ""
echo -e "Releasing: ${GREEN}${NEW_TAG}${NC}"
echo ""

# ── Validate CHANGELOG ────────────────────────────────────────────────────────
if [[ ! -f "CHANGELOG.md" ]]; then
    echo -e "${RED}Error: CHANGELOG.md not found.${NC}"
    exit 1
fi

if ! grep -qF "## [${NEW_VERSION}]" CHANGELOG.md; then
    echo -e "${RED}Error: CHANGELOG.md has no entry for ${NEW_VERSION}.${NC}"
    echo ""
    echo "Add a section like:"
    echo "  ## [${NEW_VERSION}] - $(date +%Y-%m-%d)"
    echo ""
    echo "Then run this script again."
    exit 1
fi

echo -e "${GREEN}✓${NC} CHANGELOG.md has entry for ${NEW_VERSION}"

# ── Pull latest to avoid divergence ──────────────────────────────────────────
echo "Pulling latest from origin/master..."
if ! git pull --ff-only origin master; then
    echo -e "${RED}Error: could not fast-forward to origin/master.${NC}"
    echo "Resolve divergence manually (e.g. git rebase origin/master) then try again."
    exit 1
fi
echo ""

# Show the changelog entry
echo ""
echo -e "${BLUE}── Changelog entry ──────────────────────${NC}"
ESCAPED_VERSION="${NEW_VERSION//./\\.}"
awk "/^## \[${ESCAPED_VERSION}\]/,/^## \[/" CHANGELOG.md | grep -v "^## \[${ESCAPED_VERSION}\]" | sed '/^## \[/d'
echo -e "${BLUE}─────────────────────────────────────────${NC}"
echo ""

# ── Confirm ───────────────────────────────────────────────────────────────────
echo -e "${BOLD}Ready to release ${NEW_TAG}.${NC}"
echo ""
echo "This will:"
echo "  1. Create git tag ${NEW_TAG}"
echo "  2. Push tag to origin"
echo "  3. GitHub Actions will build and publish the release"
echo ""
read -p "Proceed? [y/N] " -n 1 -r < /dev/tty
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

# ── Tag and push ──────────────────────────────────────────────────────────────
echo ""
echo -e "${BLUE}Creating tag ${NEW_TAG}...${NC}"
git tag -a "$NEW_TAG" -m "Release ${NEW_TAG}"

echo -e "${BLUE}Pushing tag to origin...${NC}"
git push origin "$NEW_TAG"

echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Release triggered!                 ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "Tag:     ${GREEN}${NEW_TAG}${NC}"
echo -e "CI:      ${GREEN}https://github.com/sjoeboo/hangar/actions${NC}"
echo -e "Release: ${GREEN}https://github.com/sjoeboo/hangar/releases/tag/${NEW_TAG}${NC}"
echo ""
echo "The release will be live in ~2 minutes once CI completes."
