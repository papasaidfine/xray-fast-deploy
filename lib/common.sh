#!/bin/bash
# Common utilities, constants, and logging functions

# Colors
export RED='\033[0;31m'
export GREEN='\033[0;32m'
export YELLOW='\033[1;33m'
export BLUE='\033[0;34m'
export CYAN='\033[0;36m'
export NC='\033[0m'

# Default settings
export DEFAULT_PORT=443
export DEFAULT_DEST="www.apple.com:443"
export DEFAULT_SNI="www.apple.com"
export CONFIG_FILE="/usr/local/etc/xray/config.json"
export SERVICE_FILE="/etc/systemd/system/xray.service"
export INFO_DIR="$HOME/.xray-reality"

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Check OS compatibility
check_os() {
    if [[ -f /etc/os-release ]]; then
        source /etc/os-release
        if [[ "$ID" != "ubuntu" && "$ID" != "debian" ]]; then
            log_warn "This script is designed for Ubuntu/Debian. Proceeding anyway..."
        fi
    fi
}

# Check if Xray is installed
is_xray_installed() {
    command -v xray &> /dev/null
}

# Check if Xray is running
is_xray_running() {
    systemctl is-active --quiet xray 2>/dev/null
}

# Get server IP
get_server_ip() {
    curl -s -4 ifconfig.me 2>/dev/null || \
    curl -s -4 ipinfo.io/ip 2>/dev/null || \
    curl -s -4 icanhazip.com 2>/dev/null || \
    echo "UNKNOWN"
}

# Load server info from saved file
load_server_info() {
    if [[ -f "$INFO_DIR/server.info" ]]; then
        source "$INFO_DIR/server.info"
    fi
}

# Save server info
save_server_info() {
    local public_key="$1"
    local port="$2"
    local sni="$3"

    mkdir -p "$INFO_DIR"
    cat > "$INFO_DIR/server.info" <<EOF
PUBLIC_KEY="$public_key"
PORT="$port"
SNI="$sni"
SERVER_IP="$(get_server_ip)"
CREATED="$(date '+%Y-%m-%d %H:%M:%S')"
EOF
    chmod 600 "$INFO_DIR/server.info"
}
