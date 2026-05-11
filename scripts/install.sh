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

if [ "${XCTL_SYSTEM:-0}" = "1" ] || [ "$(id -u)" = "0" ]; then
  DEST="/usr/local/bin/xctl"
  SUDO=""
  [ "$(id -u)" = "0" ] || SUDO="sudo"
else
  DEST="${HOME}/.local/bin/xctl"
  SUDO=""
fi

URL="https://github.com/${REPO}/releases/latest/download/xctl-${OS}-${ARCH}"

echo "Installing xctl to ${DEST}"
${SUDO} mkdir -p "$(dirname "${DEST}")"
${SUDO} curl -fsSL -o "${DEST}" "${URL}"
${SUDO} chmod 0755 "${DEST}"

echo "Installed: $(${DEST} --version 2>/dev/null || echo "${DEST}")"

case ":${PATH}:" in
  *":$(dirname "${DEST}"):"*) ;;
  *) echo "warning: $(dirname "${DEST}") is not on your PATH" >&2 ;;
esac
