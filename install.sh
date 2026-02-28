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
