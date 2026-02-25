#!/usr/bin/env bash
#
# Hangar Installer (source build)
#
# Usage:
#   git clone git@git.spotify.net:mnicholson/hangar
#   cd hangar
#   ./install.sh
#
# Options:
#   --dir <path>          Installation directory (default: ~/.local/bin)
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
SKIP_TMUX_CONFIG=false
SKIP_HOOKS=false
NON_INTERACTIVE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dir)
            INSTALL_DIR="$2"
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
            echo "Usage: ./install.sh [options]"
            echo ""
            echo "Options:"
            echo "  --dir <path>          Installation directory (default: ~/.local/bin)"
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

# ── Verify we're in the repo root ────────────────────────────────────────────
if [[ ! -f "go.mod" ]] || ! grep -q 'github.com/sjoeboo/hangar' go.mod 2>/dev/null; then
    echo -e "${RED}Error: run this script from the root of the hangar repo.${NC}"
    echo "  git clone git@git.spotify.net:mnicholson/hangar"
    echo "  cd hangar && ./install.sh"
    exit 1
fi

# ── Check for Go ──────────────────────────────────────────────────────────────
if ! command -v go &>/dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH.${NC}"
    echo ""
    echo "Install Go:"
    echo "  brew install go            # macOS Homebrew"
    echo "  mise install go            # mise version manager"
    echo "  asdf install golang latest # asdf version manager"
    echo "  https://go.dev/dl/         # official installer"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo -e "Go:       ${GREEN}${GO_VERSION}${NC}"

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

# ── Build ─────────────────────────────────────────────────────────────────────
echo -e "${BLUE}Building hangar from source...${NC}"

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR="./build"
mkdir -p "$BUILD_DIR"

if go build -ldflags "-X main.Version=${VERSION}" -o "${BUILD_DIR}/hangar" ./cmd/hangar; then
    echo -e "${GREEN}✓${NC} Build successful (version: ${VERSION})"
else
    echo -e "${RED}Error: build failed.${NC}"
    exit 1
fi
echo ""

# ── Install binary ────────────────────────────────────────────────────────────
mkdir -p "$INSTALL_DIR"
cp "${BUILD_DIR}/hangar" "$INSTALL_DIR/hangar"
chmod +x "$INSTALL_DIR/hangar"
echo -e "${GREEN}✓${NC} Installed to ${INSTALL_DIR}/hangar"

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

    # Detect OS + clipboard
    local OS
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    local IS_WSL=false
    if grep -qi microsoft /proc/version 2>/dev/null || [[ -n "$WSL_DISTRO_NAME" ]]; then
        IS_WSL=true
    fi

    local CLIPBOARD_CMD
    if [[ "$OS" == "darwin" ]]; then
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

    cat >> "$TMUX_CONF" <<EOF

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
EOF

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

    if "$INSTALL_DIR/hangar" hooks install 2>/dev/null; then
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
echo -e "Binary:  ${GREEN}${INSTALL_DIR}/hangar${NC}  (${VERSION})"
echo ""
echo "Get started:"
echo "  hangar               # Launch the TUI"
echo "  hangar hooks status  # Check hook status"
echo "  hangar --help        # Show all commands"
echo ""
echo "To update later:"
echo "  cd /path/to/hangar && git pull && ./install.sh"
