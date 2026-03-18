#!/bin/sh
set -e

# HPP Hub CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/hpp-io/hpphub-cli/main/install.sh | bash

REPO="hpp-io/hpphub-cli"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="hpphub"

main() {
  # Detect OS
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *)
      echo "Error: Unsupported OS: $OS"
      exit 1
      ;;
  esac

  # Detect architecture
  ARCH=$(uname -m)
  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
      echo "Error: Unsupported architecture: $ARCH"
      exit 1
      ;;
  esac

  # Build download URL
  SUFFIX="${OS}-${ARCH}"
  if [ "$OS" = "windows" ]; then
    SUFFIX="${SUFFIX}.exe"
    BINARY_NAME="hpphub.exe"
  fi

  # Get latest release tag
  echo "Fetching latest release..."
  LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

  if [ -z "$LATEST" ]; then
    echo "Error: Could not determine latest release"
    exit 1
  fi

  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/hpphub-${SUFFIX}"

  echo "Downloading hpphub ${LATEST} for ${OS}/${ARCH}..."
  curl -fsSL -o "/tmp/${BINARY_NAME}" "$DOWNLOAD_URL"
  chmod +x "/tmp/${BINARY_NAME}"

  # Install
  if [ -w "$INSTALL_DIR" ]; then
    mv "/tmp/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
  else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "/tmp/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
  fi

  echo ""
  echo "  ✓ hpphub ${LATEST} installed to ${INSTALL_DIR}/${BINARY_NAME}"
  echo ""
  echo "  Get started:"
  echo "    hpphub launch openclaw"
  echo ""
}

main
