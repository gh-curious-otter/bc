#!/bin/bash
# bc installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/rpuneet/bc/main/scripts/install.sh | bash

set -e

# Configuration
REPO="rpuneet/bc"
INSTALL_DIR="${BC_INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="bc"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Print with arrow prefix
info() {
    echo -e "${CYAN}→${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$OS" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    info "Detecting OS... ${OS} ${ARCH}"
}

# Get latest release version
get_latest_version() {
    info "Fetching latest version..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$VERSION" ]; then
        error "Failed to fetch latest version. Check your network connection."
    fi

    info "Latest version: ${VERSION}"
}

# Download and install binary
download_and_install() {
    BINARY_SUFFIX=""
    if [ "$OS" = "windows" ]; then
        BINARY_SUFFIX=".exe"
    fi

    # GoReleaser produces archives: bc_VERSION_OS_ARCH.tar.gz (or .zip for Windows)
    if [ "$OS" = "windows" ]; then
        ARCHIVE_EXT="zip"
    else
        ARCHIVE_EXT="tar.gz"
    fi
    ARCHIVE_NAME="bc_${VERSION#v}_${OS}_${ARCH}.${ARCHIVE_EXT}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
    TMP_DIR=$(mktemp -d)

    info "Downloading bc ${VERSION}..."

    if ! curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE_NAME}"; then
        rm -rf "$TMP_DIR"
        error "Failed to download bc. Check if release exists for ${OS}/${ARCH}."
    fi

    # Extract binary from archive
    if [ "$ARCHIVE_EXT" = "zip" ]; then
        unzip -q "${TMP_DIR}/${ARCHIVE_NAME}" -d "$TMP_DIR"
    else
        tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "$TMP_DIR"
    fi

    TMP_FILE="${TMP_DIR}/bc${BINARY_SUFFIX}"
    chmod +x "$TMP_FILE"

    info "Installing to ${INSTALL_DIR}..."

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        warn "Need elevated permissions to install to ${INSTALL_DIR}"
        sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    rm -rf "$TMP_DIR"
}

# Verify installation
verify_installation() {
    info "Verifying installation..."

    if ! command -v bc &> /dev/null; then
        # bc might not be in PATH yet
        if [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
            success "bc installed successfully!"
            echo ""
            echo "Add ${INSTALL_DIR} to your PATH if not already:"
            echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        else
            error "Installation failed. Binary not found."
        fi
    else
        success "bc installed successfully!"
    fi
}

# Check dependencies
check_dependencies() {
    if ! command -v tmux &> /dev/null; then
        warn "tmux is not installed (required for bc agents)"
        echo ""
        case "$OS" in
            darwin)
                echo "  Install with: brew install tmux"
                ;;
            linux)
                echo "  Install with: apt install tmux  # or your package manager"
                ;;
        esac
        echo ""
    fi
}

# Print next steps
print_next_steps() {
    echo ""
    echo "Run 'bc' to get started."
    echo ""
    echo "Quick start:"
    echo "  bc init       # Initialize a new workspace"
    echo "  bc up         # Start the root agent"
    echo "  bc home       # Open the TUI dashboard"
    echo ""
    echo "Documentation: https://github.com/${REPO}#readme"
}

# Main
main() {
    echo ""
    echo "bc Installer"
    echo "============"
    echo ""

    detect_platform
    get_latest_version
    download_and_install
    verify_installation
    check_dependencies
    print_next_steps
}

main "$@"
