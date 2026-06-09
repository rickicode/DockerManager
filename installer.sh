#!/usr/bin/env bash
#
# DockerManager Installer
# https://github.com/rickicode/DockerManager
#
# Usage: curl -fsSL https://raw.githubusercontent.com/rickicode/DockerManager/main/installer.sh | bash
# Or:    wget -qO- https://raw.githubusercontent.com/rickicode/DockerManager/main/installer.sh | bash
#

set -euo pipefail

# --- Config ---
REPO_OWNER="rickicode"
REPO_NAME="DockerManager"
REPO_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}"
API_URL="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="docker-manager"

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# --- Functions ---
info()  { echo -e "${CYAN}[INFO]${NC} $1"; }
ok()    { echo -e "${GREEN}[OK]${NC}   $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
err()   { echo -e "${RED}[ERR]${NC}  $1"; }

cleanup() {
    [ -n "${TMP_DIR:-}" ] && rm -rf "$TMP_DIR"
}

trap cleanup EXIT

# --- Detect OS & Arch ---
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux)  os="linux"   ;;
        Darwin) os="darwin"  ;;
        *)      err "Unsupported OS: $(uname -s)"; exit 1 ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="amd64"   ;;
        aarch64|arm64) arch="arm64"   ;;
        armv7l|armv6l) arch="arm"     ;;
        *)            err "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac

    echo "${os}_${arch}"
}

# --- Check for existing Docker binary ---
check_docker_prereq() {
    if command -v docker &> /dev/null; then
        ok "Docker detected: $(docker --version 2>/dev/null || echo 'installed')"
    else
        warn "Docker not found. DockerManager requires Docker to be installed."
        warn "Install Docker first: https://docs.docker.com/engine/install/"
    fi
}

# --- Install from GitHub release ---
install_from_github() {
    local platform="$1"
    local archive_url download_url

    info "Fetching latest release from ${REPO_URL}..."

    # Get latest release info via GitHub API
    if command -v curl &> /dev/null; then
        release_json=$(curl -fsSL "$API_URL" 2>/dev/null || true)
    elif command -v wget &> /dev/null; then
        release_json=$(wget -qO- "$API_URL" 2>/dev/null || true)
    else
        err "Neither curl nor wget found. Please install one of them."
        exit 1
    fi

    if [ -z "$release_json" ]; then
        warn "Could not fetch latest release from GitHub API."
        warn "Falling back to building from source..."
        install_from_source
        return
    fi

    # Extract version and download URL
    local version
    version=$(echo "$release_json" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\([^"]*\)".*/\1/')

    if [ -z "$version" ]; then
        warn "Could not determine latest version."
        warn "Falling back to building from source..."
        install_from_source
        return
    fi

    info "Latest version: ${version}"

    # Find the asset for this platform
    local asset_name="${BINARY_NAME}_${version}_${platform}.tar.gz"
    download_url=$(echo "$release_json" | grep -o "https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${asset_name}" | head -1)

    if [ -z "$download_url" ]; then
        warn "No pre-built binary for ${platform} in release ${version}."
        warn "Falling back to building from source..."
        install_from_source
        return
    fi

    # Download the archive
    TMP_DIR=$(mktemp -d)
    info "Downloading ${asset_name}..."

    if command -v curl &> /dev/null; then
        curl -fsSL "$download_url" -o "${TMP_DIR}/${asset_name}"
    else
        wget -q "$download_url" -O "${TMP_DIR}/${asset_name}"
    fi

    # Extract
    info "Extracting..."
    tar -xzf "${TMP_DIR}/${asset_name}" -C "$TMP_DIR"

    local binary_path="${TMP_DIR}/${BINARY_NAME}"
    if [ ! -f "$binary_path" ]; then
        err "Binary not found in archive!"
        exit 1
    fi

    # Install
    install_binary "$binary_path"
}

# --- Build from source ---
install_from_source() {
    if ! command -v go &> /dev/null; then
        err "Go is required to build from source."
        err "Install Go first: https://go.dev/dl/"
        exit 1
    fi

    TMP_DIR=$(mktemp -d)
    info "Cloning repository..."

    if ! git clone --depth 1 "${REPO_URL}.git" "$TMP_DIR" 2>/dev/null; then
        # If git is not available or clone fails, try go install
        info "Git clone failed. Trying go install..."
        MODULE_PATH="github.com/${REPO_OWNER}/${REPO_NAME}"
        go install "${MODULE_PATH}@latest" 2>/dev/null && {
            GO_BIN=$(go env GOPATH)/bin
            if [ -f "$GO_BIN/$BINARY_NAME" ]; then
                cp "$GO_BIN/$BINARY_NAME" "${TMP_DIR}/$BINARY_NAME"
            else
                warn "Binary installed via 'go install' but not found at expected path."
                ok "Try running: ${BINARY_NAME} --port 8080"
                return
            fi
        }
    fi

    # Build
    info "Building ${BINARY_NAME}..."
    (cd "$TMP_DIR" && go build -ldflags="-s -w" -o "$BINARY_NAME" .)

    local binary_path="${TMP_DIR}/${BINARY_NAME}"
    if [ ! -f "$binary_path" ]; then
        err "Build failed!"
        exit 1
    fi

    install_binary "$binary_path"
}

# --- Install binary ---
install_binary() {
    local binary_path="$1"

    info "Installing ${BINARY_NAME} to ${INSTALL_DIR}..."

    if [ ! -w "$INSTALL_DIR" ]; then
        info "Requires sudo to install to ${INSTALL_DIR}"
        sudo cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    if command -v ${BINARY_NAME} &> /dev/null; then
        ok "${BINARY_NAME} installed successfully!"
        info "Run: ${BINARY_NAME} --port 8080"
        info "Then open http://localhost:8080 in your browser"
    else
        err "Installation failed. ${INSTALL_DIR} may not be in your PATH."
        info "Try: export PATH=\$PATH:${INSTALL_DIR}"
        exit 1
    fi
}

# --- Main ---
main() {
    echo ""
    echo "  🐳 DockerManager Installer"
    echo "  ${REPO_URL}"
    echo ""

    check_docker_prereq

    local platform
    platform=$(detect_platform)
    info "Detected platform: ${platform}"

    # Try GitHub release first, fall back to source build
    install_from_github "$platform"
}

main "$@"
