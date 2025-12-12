#!/bin/bash
#
# Agent Deck Installer
# https://github.com/asheshgoplani/agent-deck
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash
#
# Options:
#   --name <name>    Custom binary name (default: agent-deck)
#   --dir <path>     Installation directory (default: ~/.local/bin)
#   --version <ver>  Specific version (default: latest)
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Defaults
BINARY_NAME="agent-deck"
INSTALL_DIR="${HOME}/.local/bin"
VERSION="latest"
REPO="asheshgoplani/agent-deck"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --name)
            BINARY_NAME="$2"
            shift 2
            ;;
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            echo "Agent Deck Installer"
            echo ""
            echo "Usage: install.sh [options]"
            echo ""
            echo "Options:"
            echo "  --name <name>    Custom binary name (default: agent-deck)"
            echo "  --dir <path>     Installation directory (default: ~/.local/bin)"
            echo "  --version <ver>  Specific version (default: latest)"
            echo "  -h, --help       Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║        Agent Deck Installer            ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *)
        echo -e "${RED}Error: Unsupported operating system: $OS${NC}"
        echo "Agent Deck only supports macOS and Linux."
        exit 1
        ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "Detected: ${GREEN}${OS}/${ARCH}${NC}"

# Check for tmux and offer to install
if ! command -v tmux &> /dev/null; then
    echo -e "${YELLOW}tmux is not installed.${NC}"
    echo "Agent Deck requires tmux to function."
    echo ""

    # Try to auto-install tmux
    if [[ "$OS" == "darwin" ]]; then
        if command -v brew &> /dev/null; then
            read -p "Install tmux via Homebrew? [Y/n] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                echo -e "Installing tmux..."
                brew install tmux
            fi
        else
            echo "Install tmux with: brew install tmux"
            echo "(Install Homebrew first: https://brew.sh)"
        fi
    else
        # Linux - try apt, dnf, or pacman
        if command -v apt-get &> /dev/null; then
            read -p "Install tmux via apt? [Y/n] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                echo -e "Installing tmux..."
                sudo apt-get update && sudo apt-get install -y tmux
            fi
        elif command -v dnf &> /dev/null; then
            read -p "Install tmux via dnf? [Y/n] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                echo -e "Installing tmux..."
                sudo dnf install -y tmux
            fi
        elif command -v pacman &> /dev/null; then
            read -p "Install tmux via pacman? [Y/n] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                echo -e "Installing tmux..."
                sudo pacman -S --noconfirm tmux
            fi
        else
            echo "Please install tmux manually:"
            echo "  sudo apt install tmux    # Debian/Ubuntu"
            echo "  sudo dnf install tmux    # Fedora"
            echo "  sudo pacman -S tmux      # Arch"
        fi
    fi

    # Check again after attempted install
    if ! command -v tmux &> /dev/null; then
        echo ""
        read -p "tmux not found. Continue anyway? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        echo -e "${GREEN}tmux installed successfully!${NC}"
    fi
fi

# Get version
if [[ "$VERSION" == "latest" ]]; then
    echo -e "Fetching latest version..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "$VERSION" ]]; then
        echo -e "${RED}Error: Could not determine latest version${NC}"
        echo "Please specify a version with --version"
        exit 1
    fi
fi

# Remove 'v' prefix if present for URL
VERSION_NUM="${VERSION#v}"
echo -e "Installing version: ${GREEN}${VERSION}${NC}"

# Download URL
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/agent-deck_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
echo -e "Downloading from: ${BLUE}${DOWNLOAD_URL}${NC}"

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download and extract
echo -e "Downloading..."
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/agent-deck.tar.gz"; then
    echo -e "${RED}Error: Download failed${NC}"
    echo "URL: $DOWNLOAD_URL"
    echo ""
    echo "This could mean:"
    echo "  - The version doesn't exist"
    echo "  - The release hasn't been published yet"
    echo "  - Network issues"
    echo ""
    echo "Try building from source instead:"
    echo "  git clone https://github.com/${REPO}.git"
    echo "  cd agent-deck && make install"
    exit 1
fi

echo -e "Extracting..."
tar -xzf "$TMP_DIR/agent-deck.tar.gz" -C "$TMP_DIR"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Install binary
echo -e "Installing to ${GREEN}${INSTALL_DIR}/${BINARY_NAME}${NC}"
mv "$TMP_DIR/agent-deck" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo -e "${YELLOW}Note: ${INSTALL_DIR} is not in your PATH${NC}"
    echo ""
    echo "Add it to your shell config:"
    echo ""
    if [[ -f "$HOME/.zshrc" ]]; then
        echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc"
        echo "  source ~/.zshrc"
    elif [[ -f "$HOME/.bashrc" ]]; then
        echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
        echo "  source ~/.bashrc"
    else
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
    echo ""
fi

# Verify installation
if "$INSTALL_DIR/$BINARY_NAME" version &> /dev/null; then
    INSTALLED_VERSION=$("$INSTALL_DIR/$BINARY_NAME" version 2>&1 || echo "unknown")
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║     Installation successful!           ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "Version: ${GREEN}${INSTALLED_VERSION}${NC}"
    echo -e "Binary:  ${GREEN}${INSTALL_DIR}/${BINARY_NAME}${NC}"
    echo ""
    echo "Get started:"
    echo "  ${BINARY_NAME}              # Launch the TUI"
    echo "  ${BINARY_NAME} add .        # Add current directory as session"
    echo "  ${BINARY_NAME} --help       # Show help"
else
    echo -e "${RED}Warning: Installation completed but verification failed${NC}"
    echo "The binary was installed but may not work correctly."
fi
