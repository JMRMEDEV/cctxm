#!/bin/sh
set -e

REPO="jmrmedev/cctxm"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

VERSION=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i location | sed 's/.*tag\///' | tr -d '\r\n')
if [ -z "$VERSION" ]; then
  echo "Failed to fetch latest version"
  exit 1
fi

URL="https://github.com/$REPO/releases/download/$VERSION/cctxm_${VERSION#v}_${OS}_${ARCH}.tar.gz"
echo "Downloading cctxm $VERSION for $OS/$ARCH..."

TMP=$(mktemp -d)
curl -sL "$URL" -o "$TMP/cctxm.tar.gz"
tar -xzf "$TMP/cctxm.tar.gz" -C "$TMP"

echo "Installing to $INSTALL_DIR/cctxm..."
sudo mv "$TMP/cctxm" "$INSTALL_DIR/cctxm"
chmod +x "$INSTALL_DIR/cctxm"
rm -rf "$TMP"

echo "cctxm $VERSION installed successfully."
echo "Run 'cctxm init' in your workspace to get started."
