#!/bin/bash
# Xray REALITY VPN Deployment & Management Script
# Supports: Ubuntu 20.04+, Debian 11+

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/lib"

# Source all library modules
source "$LIB_DIR/common.sh"
source "$LIB_DIR/config.sh"
source "$LIB_DIR/clients.sh"
source "$LIB_DIR/server.sh"
source "$LIB_DIR/install.sh"
source "$LIB_DIR/menu.sh"

# Show help
show_help() {
    cat <<EOF
Xray REALITY VPN Deployment & Management Script

Usage: $0 [command] [options]

Commands:
  install         Fresh installation (interactive if no options)
  manage          Open management menu
  status          Show server status
  add-client      Add new client
  remove-client   Remove a client
  list-clients    List all clients
  show-client     Show client configuration
  export          Export all client configs
  change-disguise Change disguised website
  change-port     Change server port
  regen-keys      Regenerate server keys
  logs            Show recent logs
  restart         Restart Xray service
  test            Test configuration
  uninstall       Remove Xray completely

Install options:
  --port PORT         Server port (default: 443)
  --disguise DOMAIN   Disguised domain (default: www.apple.com)
  --client NAME       First client name (default: default)

Examples:
  $0 install
  $0 install --port 443 --disguise www.microsoft.com --client myphone
  $0 add-client --name laptop
  $0 change-disguise --domain www.google.com
  $0 manage

EOF
}

# Parse command line arguments
parse_args() {
    local cmd="$1"
    shift

    local port="$DEFAULT_PORT"
    local disguise="$DEFAULT_SNI"
    local client_name="default"
    local name=""
    local domain=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --port) port="$2"; shift 2 ;;
            --disguise) disguise="$2"; shift 2 ;;
            --client) client_name="$2"; shift 2 ;;
            --name) name="$2"; shift 2 ;;
            --domain) domain="$2"; shift 2 ;;
            --help|-h) show_help; exit 0 ;;
            *) shift ;;
        esac
    done

    case "$cmd" in
        install)
            fresh_install "$port" "${disguise}:443" "$disguise" "$client_name"
            ;;
        manage)
            manage_server
            ;;
        status)
            show_status
            ;;
        add-client)
            add_client "$name"
            ;;
        remove-client)
            remove_client "$name"
            ;;
        list-clients)
            list_clients
            ;;
        show-client)
            if [[ -z "$name" ]]; then
                list_clients
                read -p "Enter client name: " name
            fi
            local uuid=$(get_client_uuid "$name")
            if [[ -n "$uuid" ]]; then
                show_client_config "$uuid" "$name"
            else
                log_error "Client not found"
            fi
            ;;
        export)
            export_clients
            ;;
        change-disguise)
            if [[ -n "$domain" ]]; then
                change_disguise "${domain}:443" "$domain"
            else
                change_disguise
            fi
            ;;
        change-port)
            change_port "$port"
            ;;
        regen-keys)
            regenerate_keys
            ;;
        logs)
            show_logs
            ;;
        restart)
            restart_service
            ;;
        test)
            test_config
            ;;
        uninstall)
            uninstall_xray
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
}

# Main entry point
main() {
    check_root

    # No arguments - auto detect mode
    if [[ $# -eq 0 ]]; then
        if is_xray_installed && [[ -f "$CONFIG_FILE" ]]; then
            log_info "Existing installation detected"
            manage_server
        else
            log_info "No installation detected, starting fresh install..."
            echo ""
            echo "Installation options (press Enter for defaults):"
            read -p "Port [$DEFAULT_PORT]: " port
            read -p "Disguised domain [$DEFAULT_SNI]: " disguise
            read -p "First client name [default]: " client_name

            port="${port:-$DEFAULT_PORT}"
            disguise="${disguise:-$DEFAULT_SNI}"
            client_name="${client_name:-default}"

            fresh_install "$port" "${disguise}:443" "$disguise" "$client_name"
        fi
    else
        parse_args "$@"
    fi
}

main "$@"
