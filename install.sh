#!/usr/bin/env bash
set -e

REPO="tristan-gscn/repaco"
LATEST_RELEASE_URL="https://github.com/${REPO}/releases/download/latest"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64)
    ARCH="amd64"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  darwin|linux)
    ;;
  *)
    echo "Error: unsupported operating system: $OS" >&2
    exit 1
    ;;
esac

BINARY="repaco-${OS}-${ARCH}"
URL="${LATEST_RELEASE_URL}/${BINARY}"

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
mkdir -p "$INSTALL_DIR"
DEST="${INSTALL_DIR}/repaco"

echo "Downloading ${BINARY} from GitHub release..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$DEST"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$DEST" "$URL"
else
  echo "Error: curl or wget is required to download repaco" >&2
  exit 1
fi

chmod +x "$DEST"

echo "Successfully installed repaco to ${DEST}"

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo ""
  echo "Notice: ${INSTALL_DIR} is not in your PATH."
  echo "Add it by running:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi
