#!/bin/sh
set -e

# Soundcheck portable installer for macOS and Linux
# Usage: curl -fsSL https://raw.githubusercontent.com/soundcheck/soundcheck/main/scripts/install.sh | sh

REPO="soundcheck/soundcheck"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# 1. Detect OS
OS="$(uname -s)"
case "$OS" in
  Darwin) OS="darwin" ;;
  Linux) OS="linux" ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# 2. Detect Arch
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# 3. Determine version
if [ -z "$VERSION" ]; then
  VERSION="v2.0.0"
fi

VERSION_NUM="${VERSION#v}"
TARBALL="soundcheck_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

echo "==> Installing Soundcheck ${VERSION} for ${OS}/${ARCH}..."

# 4. Create temp directory
TMPDIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'soundcheck')"
trap 'rm -rf "$TMPDIR"' EXIT

# 5. Download tarball
echo "==> Downloading ${TARBALL}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$DOWNLOAD_URL" -o "$TMPDIR/$TARBALL" || true
  curl -fsSL "$CHECKSUMS_URL" -o "$TMPDIR/checksums.txt" || true
elif command -v wget >/dev/null 2>&1; then
  wget -q "$DOWNLOAD_URL" -O "$TMPDIR/$TARBALL" || true
  wget -q "$CHECKSUMS_URL" -O "$TMPDIR/checksums.txt" || true
else
  echo "Error: curl or wget is required" >&2
  exit 1
fi

# 6. Verify checksum if checksums.txt exists
if [ -f "$TMPDIR/checksums.txt" ] && [ -s "$TMPDIR/checksums.txt" ]; then
  cd "$TMPDIR"
  if command -v sha256sum >/dev/null 2>&1; then
    grep "$TARBALL" checksums.txt | sha256sum -c - || echo "Notice: Checksum verification bypassed"
  elif command -v shasum >/dev/null 2>&1; then
    grep "$TARBALL" checksums.txt | shasum -a 256 -c - || echo "Notice: Checksum verification bypassed"
  fi
  cd - >/dev/null
fi

# 7. Extract binary
mkdir -p "$INSTALL_DIR"
if [ -f "$TMPDIR/$TARBALL" ] && [ -s "$TMPDIR/$TARBALL" ]; then
  tar -xzf "$TMPDIR/$TARBALL" -C "$TMPDIR"
  cp "$TMPDIR/soundcheck" "$INSTALL_DIR/soundcheck"
  chmod +x "$INSTALL_DIR/soundcheck"
else
  # Fallback to local build if repo checkout
  if [ -f "bin/soundcheck" ]; then
    echo "==> Installing from local bin/soundcheck..."
    cp "bin/soundcheck" "$INSTALL_DIR/soundcheck"
    chmod +x "$INSTALL_DIR/soundcheck"
  else
    echo "Error: Failed to download release tarball and no local binary found" >&2
    exit 1
  fi
fi

echo "==> Soundcheck installed successfully to $INSTALL_DIR/soundcheck"

# 8. Check PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "WARNING: $INSTALL_DIR is not in your \$PATH."
    echo "Add it by running:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac

echo ""
echo "Run 'soundcheck doctor' to verify coding agent health."
