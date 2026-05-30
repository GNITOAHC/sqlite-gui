#!/bin/bash

set -euo pipefail

REPO="GNITOAHC/sqlite-gui"
DEFAULT_DIR="$HOME/.local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
linux | darwin) ;;
*)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect arch
case "$(uname -m)" in
x86_64) ARCH="amd64" ;;
aarch64 | arm64) ARCH="arm64" ;;
*)
    echo "Unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

# Prompt — read from /dev/tty so this works when piped through curl | bash
if [ -t 0 ] || [ -c /dev/tty ]; then
    printf "Install directory [%s]: " "$DEFAULT_DIR" >/dev/tty
    read -r INSTALL_DIR </dev/tty
    INSTALL_DIR="${INSTALL_DIR:-$DEFAULT_DIR}"
else
    INSTALL_DIR="$DEFAULT_DIR"
fi

# Create directory if needed
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating $INSTALL_DIR ..."
    mkdir -p "$INSTALL_DIR"
fi

ARCHIVE="sqlite-gui-${OS}-${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/latest/download/${ARCHIVE}"
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading sqlite-gui (${OS}/${ARCH}) ..."
if command -v curl >/dev/null 2>&1; then
    curl -LsSf --fail -o "$TMP_DIR/$ARCHIVE" "$URL" || {
        echo "Download failed: $URL" >&2
        exit 1
    }
elif command -v wget >/dev/null 2>&1; then
    wget -qO "$TMP_DIR/$ARCHIVE" "$URL" || {
        echo "Download failed: $URL" >&2
        exit 1
    }
else
    echo "Error: curl or wget is required." >&2
    exit 1
fi

echo "Extracting ..."
tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR"

BINARY=$(find "$TMP_DIR" -maxdepth 1 -name "sqlite-gui*" ! -name "*.tar.gz" | head -1)
if [ -z "$BINARY" ]; then
    echo "Error: could not find sqlite-gui binary in archive." >&2
    exit 1
fi

chmod +x "$BINARY"
mv "$BINARY" "$INSTALL_DIR/sqlite-gui"

echo "Installed: $INSTALL_DIR/sqlite-gui"

# Warn if the install dir is not on PATH
case ":$PATH:" in
*":$INSTALL_DIR:"*) ;;
*) echo "Note: add $INSTALL_DIR to your PATH to use sqlite-gui from anywhere." ;;
esac
