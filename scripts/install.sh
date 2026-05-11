#!/usr/bin/env bash
set -euo pipefail

REPO="papasaidfine/xray-fast-deploy"

case "$(uname -s)" in
  Linux) OS="linux" ;;
  *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

DEST="/usr/local/bin/xctl"
SUDO=""
if [ "$(id -u)" != "0" ]; then
  if ! command -v sudo >/dev/null 2>&1; then
    echo "xctl needs to be installed to ${DEST} (root-owned), but sudo is not available." >&2
    echo "Re-run this script as root." >&2
    exit 1
  fi
  SUDO="sudo"
fi

URL="https://github.com/${REPO}/releases/latest/download/xctl-${OS}-${ARCH}"

echo "Installing xctl to ${DEST}"
${SUDO} curl -fsSL -o "${DEST}" "${URL}"
${SUDO} chmod 0755 "${DEST}"

echo "Installed xctl at ${DEST}"
echo "Run: sudo xctl"
