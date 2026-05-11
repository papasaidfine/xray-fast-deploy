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
if [ "$(id -u)" != "0" ]; then
  echo "This installer must run as root so it can write to ${DEST}." >&2
  echo "Re-run with sudo, e.g." >&2
  echo "    curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install.sh | sudo bash" >&2
  exit 1
fi

URL="https://github.com/${REPO}/releases/latest/download/xctl-${OS}-${ARCH}"

echo "Installing xctl to ${DEST}"
curl -fsSL -o "${DEST}" "${URL}"
chmod 0755 "${DEST}"

echo "Installed xctl at ${DEST}"
echo "Run: sudo xctl"
