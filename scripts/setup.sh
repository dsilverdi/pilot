#!/bin/bash

# Pilot Setup Script
# Installs pilot to a location in your PATH

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 Pilot Setup"
echo "=============="
echo

# Determine install directory
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Check if running with sudo for /usr/local/bin
if [[ "$INSTALL_DIR" == "/usr/local/bin" ]] && [[ $EUID -ne 0 ]]; then
    echo -e "${YELLOW}Note: Installing to $INSTALL_DIR requires sudo${NC}"
    SUDO="sudo"
else
    SUDO=""
fi

# Alternative: Install to ~/bin if user prefers no sudo
if [[ "$1" == "--user" ]]; then
    INSTALL_DIR="$HOME/bin"
    SUDO=""
    mkdir -p "$INSTALL_DIR"

    # Check if ~/bin is in PATH
    if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
        echo -e "${YELLOW}Warning: $HOME/bin is not in your PATH${NC}"
        echo "Add this to your shell config (~/.bashrc or ~/.zshrc):"
        echo "  export PATH=\"\$HOME/bin:\$PATH\""
        echo
    fi
fi

echo "Install directory: $INSTALL_DIR"
echo

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    echo "Please install Go from https://go.dev/dl/"
    exit 1
fi

echo "Go version: $(go version)"
echo

# Get the script directory (where the project is)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Building pilot..."
cd "$PROJECT_DIR"

# Build with version info
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-s -w -X main.version=${VERSION}"

go build -ldflags "$LDFLAGS" -o pilot ./cmd/pilot

if [[ ! -f "pilot" ]]; then
    echo -e "${RED}Error: Build failed${NC}"
    exit 1
fi

echo -e "${GREEN}Build successful${NC}"
echo

# Install
echo "Installing to $INSTALL_DIR..."
$SUDO mv pilot "$INSTALL_DIR/pilot"
$SUDO chmod +x "$INSTALL_DIR/pilot"

echo -e "${GREEN}✓ Pilot installed successfully!${NC}"
echo

# Verify installation
if command -v pilot &> /dev/null; then
    echo "Installed version:"
    pilot -v
    echo
    echo "Run 'pilot' to start interactive mode"
    echo "Run 'pilot -p \"your prompt\"' for direct execution"
else
    echo -e "${YELLOW}Note: You may need to restart your terminal or run:${NC}"
    echo "  source ~/.bashrc  # or ~/.zshrc"
fi
