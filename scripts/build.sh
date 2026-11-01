#!/usr/bin/env bash
set -euo pipefail

# Scripts directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

VERSION="${1:-${VERSION:-v2.0.1}}"
MAX_SIZE_BYTES=20971520 # 20MB budget

echo "==> Building UI assets..."
npm --prefix ui run build

echo "==> Building Soundcheck binary (${VERSION})..."
mkdir -p bin
CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X main.version=${VERSION} -X github.com/soundcheck/soundcheck/internal/transport/cli.Version=${VERSION}" \
  -o bin/soundcheck ./cmd/soundcheck

# Calculate binary size portably
SIZE_BYTES=$(wc -c < bin/soundcheck | tr -d ' ')
SIZE_MB=$(awk "BEGIN {printf \"%.2f\", ${SIZE_BYTES}/1048576}")

echo "==> Soundcheck binary size: ${SIZE_BYTES} bytes (${SIZE_MB} MB)"

if [ "${SIZE_BYTES}" -gt "${MAX_SIZE_BYTES}" ]; then
  echo "ERROR: Binary size (${SIZE_BYTES} bytes / ${SIZE_MB} MB) exceeds 20MB budget (${MAX_SIZE_BYTES} bytes)!" >&2
  exit 1
fi

echo "==> Build successful: bin/soundcheck (within <= 20MB budget)"
