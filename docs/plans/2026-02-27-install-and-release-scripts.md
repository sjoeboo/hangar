# Install Script & Release Script Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a `curl | bash` binary-download install script and a local release automation script.

**Architecture:** Three files change: `install.sh` becomes a binary downloader (current one renamed to `install-from-source.sh`), and a new `scripts/release.sh` handles tagging + pushing to trigger CI. The tmux config and hooks sections are shared/copied between install scripts.

**Tech Stack:** Bash, GitHub Releases API (JSON, `curl`), `tar`, git tag/push

---

### Task 1: Rename `install.sh` → `install-from-source.sh`

**Files:**
- Rename: `install.sh` → `install-from-source.sh`

**Step 1: Rename the file**

```bash
git mv install.sh install-from-source.sh
```

**Step 2: Update the help text and "To update later" message inside the renamed file**

Open `install-from-source.sh`. Find these two places and update them:

At the top comment block (line ~8), change:
```
#   ./install.sh
```
to:
```
#   ./install-from-source.sh
```

At the bottom "To update later" section (line ~300), change:
```
echo "  cd /path/to/hangar && git pull && ./install.sh"
```
to:
```
echo "  cd /path/to/hangar && git pull && ./install-from-source.sh"
```

**Step 3: Commit**

```bash
git add install-from-source.sh
git commit -m "chore: rename install.sh to install-from-source.sh"
```

---

### Task 2: Create `install.sh` — Binary Downloader

**Files:**
- Create: `install.sh`

This script downloads a pre-built binary from GitHub releases. It reuses the
`configure_tmux` and `install_hooks` functions verbatim from `install-from-source.sh`.

**Step 1: Create `install.sh` with this content**

```bash
#!/usr/bin/env bash
#
# Hangar Installer — downloads a pre-built binary from GitHub releases
#
# Usage:
#   curl -fsSL https://github.com/sjoeboo/hangar/raw/master/install.sh | bash
#
#   Or download and run directly:
#   bash install.sh [options]
#
# Options:
#   --dir <path>          Installation directory (default: ~/.local/bin)
#   --version <version>   Install a specific version (e.g. 1.0.2); default: latest
#   --skip-tmux-config    Skip tmux configuration prompt
#   --skip-hooks          Skip Claude Code hooks installation prompt
#   --non-interactive     Skip all prompts (for CI/automated installs)
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Defaults
INSTALL_DIR="${HOME}/.local/bin"
INSTALL_VERSION=""
SKIP_TMUX_CONFIG=false
SKIP_HOOKS=false
NON_INTERACTIVE=false
REPO="sjoeboo/hangar"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --version)
            INSTALL_VERSION="$2"
            shift 2
            ;;
        --skip-tmux-config)
            SKIP_TMUX_CONFIG=true
            shift
            ;;
        --skip-hooks)
            SKIP_HOOKS=true
            shift
            ;;
        --non-interactive)
            NON_INTERACTIVE=true
            SKIP_TMUX_CONFIG=true
            SKIP_HOOKS=true
            shift
            ;;
        -h|--help)
            echo "Hangar Installer"
            echo ""
            echo "Usage: bash install.sh [options]"
            echo "   or: curl -fsSL https://github.com/sjoeboo/hangar/raw/master/install.sh | bash"
            echo ""
            echo "Options:"
            echo "  --dir <path>          Installation directory (default: ~/.local/bin)"
            echo "  --version <version>   Specific version to install (default: latest)"
            echo "  --skip-tmux-config    Skip tmux configuration prompt"
            echo "  --skip-hooks          Skip Claude Code hooks installation prompt"
            echo "  --non-interactive     Skip all prompts"
            echo "  -h, --help            Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         Hangar Installer               ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# ── Detect platform ───────────────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)
        echo -e "${RED}Error: unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case "$OS" in
    linux|darwin) ;;
    *)
        echo -e "${RED}Error: unsupported OS: $OS${NC}"
        echo "Supported: linux, darwin"
        echo "For other platforms, build from source: https://github.com/${REPO}"
        exit 1
        ;;
esac

echo -e "Platform: ${GREEN}${OS}/${ARCH}${NC}"

# ── Check for tmux ────────────────────────────────────────────────────────────
if ! command -v tmux &>/dev/null; then
    echo -e "${RED}Error: tmux is not installed. Hangar requires tmux.${NC}"
    echo ""
    echo "Install tmux:"
    echo "  brew install tmux          # macOS"
    echo "  sudo apt install tmux      # Debian/Ubuntu"
    echo "  sudo dnf install tmux      # Fedora"
    echo "  sudo pacman -S tmux        # Arch"
    exit 1
fi
echo -e "tmux:     ${GREEN}$(tmux -V 2>/dev/null | head -1)${NC}"
echo ""

# ── Resolve version ───────────────────────────────────────────────────────────
if [[ -z "$INSTALL_VERSION" ]]; then
    echo "Fetching latest release..."
    RELEASE_JSON=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")
    INSTALL_VERSION=$(echo "$RELEASE_JSON" | grep '"tag_name"' | sed 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/')
    if [[ -z "$INSTALL_VERSION" ]]; then
        echo -e "${RED}Error: could not determine latest version from GitHub API.${NC}"
        exit 1
    fi
fi

# Strip leading 'v' if present
INSTALL_VERSION="${INSTALL_VERSION#v}"
echo -e "Version:  ${GREEN}v${INSTALL_VERSION}${NC}"

# ── Build download URL ────────────────────────────────────────────────────────
ASSET_NAME="hangar_${INSTALL_VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${INSTALL_VERSION}/${ASSET_NAME}"

echo -e "Asset:    ${ASSET_NAME}"
echo ""

# ── Download ──────────────────────────────────────────────────────────────────
echo -e "${BLUE}Downloading...${NC}"
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

if ! curl -fsSL --progress-bar "$DOWNLOAD_URL" -o "${TMP_DIR}/${ASSET_NAME}"; then
    echo -e "${RED}Error: download failed.${NC}"
    echo "URL: $DOWNLOAD_URL"
    echo ""
    echo "Check that v${INSTALL_VERSION} exists at:"
    echo "  https://github.com/${REPO}/releases"
    exit 1
fi

# ── Extract ───────────────────────────────────────────────────────────────────
echo -e "${BLUE}Extracting...${NC}"
tar -xzf "${TMP_DIR}/${ASSET_NAME}" -C "$TMP_DIR"

if [[ ! -f "${TMP_DIR}/hangar" ]]; then
    echo -e "${RED}Error: hangar binary not found in archive.${NC}"
    exit 1
fi

# ── Install ───────────────────────────────────────────────────────────────────
mkdir -p "$INSTALL_DIR"
install -m 755 "${TMP_DIR}/hangar" "${INSTALL_DIR}/hangar"
echo -e "${GREEN}✓${NC} Installed to ${INSTALL_DIR}/hangar"

# Verify
INSTALLED_VERSION=$("${INSTALL_DIR}/hangar" version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "unknown")
if [[ "$INSTALLED_VERSION" != "unknown" ]]; then
    echo -e "${GREEN}✓${NC} Verified: hangar v${INSTALLED_VERSION}"
fi

# Check PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo -e "${YELLOW}Note: ${INSTALL_DIR} is not in your PATH.${NC}"
    echo "Add it to your shell config:"
    if [[ -f "$HOME/.zshrc" ]]; then
        echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc && source ~/.zshrc"
    elif [[ -f "$HOME/.bashrc" ]]; then
        echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc && source ~/.bashrc"
    else
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
fi
echo ""

# ── tmux configuration ────────────────────────────────────────────────────────
configure_tmux() {
    local TMUX_CONF="$HOME/.tmux.conf"
    local MARKER="# hangar configuration"
    local VERSION_MARKER="# hangar-tmux-config-version:"
    local CURRENT_VERSION="1"

    if [[ -f "$TMUX_CONF" ]] && grep -q "$MARKER" "$TMUX_CONF" 2>/dev/null; then
        echo -e "${GREEN}tmux already configured for hangar.${NC}"
        return 0
    fi

    echo -e "${BLUE}tmux Configuration${NC}"
    echo "Hangar works best with mouse scroll and clipboard support."
    echo ""
    if [[ -f "$TMUX_CONF" ]]; then
        echo -e "Found existing ${YELLOW}~/.tmux.conf${NC} — settings will be appended."
    else
        echo "No ~/.tmux.conf found — it will be created."
    fi
    echo ""
    echo -e "${BLUE}  •${NC} Mouse scrolling & drag-to-copy"
    echo -e "${BLUE}  •${NC} Auto copy-mode on scroll"
    echo -e "${BLUE}  •${NC} System clipboard integration"
    echo -e "${BLUE}  •${NC} 256-color support + 50k line history"
    echo ""

    if [[ "$SKIP_TMUX_CONFIG" == "true" ]]; then
        echo -e "${YELLOW}Skipping tmux configuration (--skip-tmux-config).${NC}"
        return 0
    fi

    if [[ "$NON_INTERACTIVE" != "true" ]]; then
        read -p "Configure tmux? [Y/n] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Nn]$ ]]; then
            echo "Skipping. Add the config manually later if needed."
            return 0
        fi
    fi

    local OS_LOWER
    OS_LOWER=$(uname -s | tr '[:upper:]' '[:lower:]')
    local IS_WSL=false
    if grep -qi microsoft /proc/version 2>/dev/null || [[ -n "$WSL_DISTRO_NAME" ]]; then
        IS_WSL=true
    fi

    local CLIPBOARD_CMD
    if [[ "$OS_LOWER" == "darwin" ]]; then
        CLIPBOARD_CMD="pbcopy"
    elif [[ "$IS_WSL" == "true" ]]; then
        CLIPBOARD_CMD="clip.exe"
    elif [[ -n "$WAYLAND_DISPLAY" ]] && command -v wl-copy &>/dev/null; then
        CLIPBOARD_CMD="wl-copy"
    elif command -v xclip &>/dev/null; then
        CLIPBOARD_CMD="xclip -in -selection clipboard"
    elif command -v xsel &>/dev/null; then
        CLIPBOARD_CMD="xsel --clipboard --input"
    else
        echo -e "${YELLOW}No clipboard tool found — install xclip for clipboard support.${NC}"
        CLIPBOARD_CMD="xclip -in -selection clipboard"
    fi

    cat >> "$TMUX_CONF" <<TMUXEOF

$MARKER
$VERSION_MARKER $CURRENT_VERSION
# Added by hangar installer - $(date +%Y-%m-%d)

set -g default-terminal "tmux-256color"
set -ag terminal-overrides ",xterm*:Tc:smcup@:rmcup@"
set -ag terminal-overrides ",*256col*:Tc"
set -sg escape-time 0
set -g history-limit 50000
set -g mouse on

bind-key -n WheelUpPane if-shell -F -t = "#{mouse_any_flag}" "send-keys -M" "if -Ft= '#{pane_in_mode}' 'send-keys -M' 'copy-mode -e'"
bind-key -T copy-mode-vi WheelUpPane   send-keys -X scroll-up
bind-key -T copy-mode-vi WheelDownPane send-keys -X scroll-down
bind-key -T copy-mode    WheelUpPane   send-keys -X scroll-up
bind-key -T copy-mode    WheelDownPane send-keys -X scroll-down
bind-key -T copy-mode-vi MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel "${CLIPBOARD_CMD}"
bind-key -T copy-mode    MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel "${CLIPBOARD_CMD}"
# End hangar configuration
TMUXEOF

    echo -e "${GREEN}✓${NC} tmux configured"

    if tmux list-sessions &>/dev/null 2>&1; then
        tmux source-file "$TMUX_CONF" 2>/dev/null || true
        echo -e "${GREEN}✓${NC} tmux config reloaded"
    else
        echo "  Run 'tmux source-file ~/.tmux.conf' to apply"
    fi
}

configure_tmux
echo ""

# ── Claude Code hooks ─────────────────────────────────────────────────────────
install_hooks() {
    if [[ "$SKIP_HOOKS" == "true" ]]; then
        return 0
    fi

    echo -e "${BLUE}Claude Code Hooks${NC}"
    echo "Hangar can install Claude Code lifecycle hooks for real-time"
    echo "status detection (instant running/waiting/idle indicators)."
    echo ""
    echo "This writes to your Claude settings.json (existing settings preserved)."
    echo "Disable later with: hooks_enabled = false in ~/.hangar/config.toml"
    echo ""

    if [[ "$NON_INTERACTIVE" != "true" ]]; then
        read -p "Install Claude Code hooks? [Y/n] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Nn]$ ]]; then
            echo "Skipping. Install later with: hangar hooks install"
            echo ""
            return 0
        fi
    fi

    if "${INSTALL_DIR}/hangar" hooks install 2>/dev/null; then
        echo -e "${GREEN}✓${NC} Claude Code hooks installed"
    else
        echo -e "${YELLOW}Could not install hooks — run 'hangar hooks install' manually.${NC}"
    fi
    echo ""
}

install_hooks

# ── Done ──────────────────────────────────────────────────────────────────────
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Installation complete!             ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "Binary:  ${GREEN}${INSTALL_DIR}/hangar${NC}  (v${INSTALL_VERSION})"
echo ""
echo "Get started:"
echo "  hangar               # Launch the TUI"
echo "  hangar hooks status  # Check hook status"
echo "  hangar --help        # Show all commands"
echo ""
echo "To update:"
echo "  hangar update        # Self-update to latest release"
```

**Step 2: Make it executable**

```bash
chmod +x install.sh
```

**Step 3: Smoke test — help flag (doesn't download anything)**

```bash
bash install.sh --help
```

Expected output includes: `Hangar Installer`, `--version`, `--dir`, `--non-interactive`

**Step 4: Smoke test — non-interactive dry run against a real release**

```bash
bash install.sh --non-interactive --dir /tmp/hangar-test-install
```

Expected: downloads, extracts, installs to `/tmp/hangar-test-install/hangar`, prints "Installation complete!"

Verify:
```bash
/tmp/hangar-test-install/hangar version
rm -rf /tmp/hangar-test-install
```

**Step 5: Commit**

```bash
git add install.sh
git commit -m "feat: add binary-download install script for curl-pipe-bash"
```

---

### Task 3: Create `scripts/release.sh`

**Files:**
- Create: `scripts/release.sh`

**Step 1: Create the `scripts/` directory and the script**

```bash
mkdir -p scripts
```

Create `scripts/release.sh` with this content:

```bash
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

# Must have clean working tree
if ! git diff --quiet || ! git diff --cached --quiet; then
    echo -e "${RED}Error: working tree has uncommitted changes. Commit or stash first.${NC}"
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
    read -p "Choice [1]: " -r CHOICE
    CHOICE="${CHOICE:-1}"

    case "$CHOICE" in
        1) NEW_VERSION="$SUGGEST_PATCH" ;;
        2) NEW_VERSION="$SUGGEST_MINOR" ;;
        3) NEW_VERSION="$SUGGEST_MAJOR" ;;
        4)
            read -p "Enter version (without 'v'): " -r NEW_VERSION
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

if ! grep -qE "^## \[${NEW_VERSION}\]" CHANGELOG.md; then
    echo -e "${RED}Error: CHANGELOG.md has no entry for ${NEW_VERSION}.${NC}"
    echo ""
    echo "Add a section like:"
    echo "  ## [${NEW_VERSION}] - $(date +%Y-%m-%d)"
    echo ""
    echo "Then run this script again."
    exit 1
fi

echo -e "${GREEN}✓${NC} CHANGELOG.md has entry for ${NEW_VERSION}"

# Show the changelog entry
echo ""
echo -e "${BLUE}── Changelog entry ──────────────────────${NC}"
awk "/^## \[${NEW_VERSION}\]/,/^## \[/" CHANGELOG.md | head -30 | grep -v "^## \[${NEW_VERSION}\]" | sed '/^## \[/d'
echo -e "${BLUE}─────────────────────────────────────────${NC}"
echo ""

# ── Pull latest to avoid divergence ──────────────────────────────────────────
echo "Pulling latest from origin/master..."
git pull --ff-only origin master
echo ""

# ── Confirm ───────────────────────────────────────────────────────────────────
echo -e "${BOLD}Ready to release ${NEW_TAG}.${NC}"
echo ""
echo "This will:"
echo "  1. Create git tag ${NEW_TAG}"
echo "  2. Push tag to origin"
echo "  3. GitHub Actions will build and publish the release"
echo ""
read -p "Proceed? [y/N] " -n 1 -r
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
```

**Step 2: Make it executable**

```bash
chmod +x scripts/release.sh
```

**Step 3: Smoke test — safety checks**

From the repo root on master with a clean tree:

```bash
# Test: wrong directory
(cd /tmp && bash /path/to/hangar/scripts/release.sh 2>&1) | grep "run from the root"

# Test: wrong branch (from a worktree on a non-master branch)
bash scripts/release.sh 2>&1 | grep "must be on master"
```

Both should print error messages and exit non-zero.

**Step 4: Dry-run test — CHANGELOG validation**

```bash
# Test: version without CHANGELOG entry
bash scripts/release.sh 99.99.99 2>&1 | grep "has no entry"
```

Expected: error about missing CHANGELOG entry.

**Step 5: Commit**

```bash
git add scripts/release.sh
git commit -m "feat: add release script for tagging and triggering CI"
```

---

### Task 4: Update `.goreleaser.yml` release notes footer

**Files:**
- Modify: `.goreleaser.yml`

The release header already references `install.sh`. The "To update later" section in the
source-build script also needs updating. Update the goreleaser footer to link to both install options.

**Step 1: Update the `install-from-source.sh` "To update later" line**

Open `install-from-source.sh` and change the final message from:
```
echo "To update later:"
echo "  cd /path/to/hangar && git pull && ./install.sh"
```
to:
```
echo "To update later:"
echo "  cd /path/to/hangar && git pull && ./install-from-source.sh"
```

(This was noted in Task 1 but double-check it got done.)

**Step 2: Verify `.goreleaser.yml` install line is correct**

The release header in `.goreleaser.yml` already has:
```
curl -fsSL https://github.com/sjoeboo/hangar/raw/master/install.sh | bash
```
This now correctly points to the binary downloader. No change needed.

**Step 3: Commit if any changes**

```bash
git add install-from-source.sh
git commit -m "chore: fix install-from-source.sh update instructions"
```

(Skip if already done in Task 1.)

---

### Task 5: Final verification

**Step 1: Check all scripts are executable**

```bash
ls -la install.sh install-from-source.sh scripts/release.sh
```

All three should show `-rwxr-xr-x`.

**Step 2: Run `go build` to confirm nothing is broken**

```bash
go build ./...
```

Expected: no output (success).

**Step 3: Run tests**

```bash
go test ./...
```

Expected: all pass (pre-existing failures in `TestNewDialog_*` are acceptable).

**Step 4: Final commit (if anything was missed)**

```bash
git status
# If clean, nothing to do. If any files remain, commit them.
```
