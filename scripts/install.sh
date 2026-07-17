#!/usr/bin/env bash
set -euo pipefail

REPO="papasaidfine/xray-fast-deploy"

PROXY=""
while [ $# -gt 0 ]; do
  case "$1" in
    --proxy)
      if [ $# -lt 2 ] || [ -z "$2" ]; then
        echo "--proxy requires a URL, e.g. --proxy socks5://127.0.0.1:1080" >&2
        exit 1
      fi
      PROXY="$2"; shift 2 ;;
    --proxy=*) PROXY="${1#*=}"; shift ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

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
CURL_OPTS=(-fsSL)
if [ -n "${PROXY}" ]; then
  echo "Downloading through proxy ${PROXY}"
  CURL_OPTS+=(-x "${PROXY}")
fi
curl "${CURL_OPTS[@]}" -o "${DEST}" "${URL}"
chmod 0755 "${DEST}"

echo "Installed xctl at ${DEST}"
echo "Run: sudo xctl"
