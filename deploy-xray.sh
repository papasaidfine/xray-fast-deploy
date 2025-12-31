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

Client Commands:
  list-clients    List all clients
  add-client      Add new client
  remove-client   Remove a client
  rename-client   Rename a client
  reset-uuid      Reset client UUID
  show-client     Show client configuration
  qr              Show QR code for a client
  qr-all          Show all QR codes
  export          Export all client VLESS links

Server Commands:
  status          Show server status
  change-disguise Change disguised website
  change-port     Change server port
  regen-keys      Regenerate server keys
  restart         Restart Xray service
  test            Test configuration
  logs            Show recent logs
  uninstall       Remove Xray completely

Options:
  --port PORT         Server port (default: 443)
  --disguise DOMAIN   Disguised domain (default: www.apple.com)
  --client NAME       Client name (default: default)
  --name NAME         Client name for operations
  --new-name NAME     New name for rename operation
  --domain DOMAIN     Domain for disguise change

Examples:
  $0 install
  $0 install --port 443 --disguise www.microsoft.com --client myphone
  $0 add-client --name laptop
  $0 rename-client --name laptop --new-name work-laptop
  $0 qr --name laptop
  $0 qr-all
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
    local new_name=""
    local domain=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --port) port="$2"; shift 2 ;;
            --disguise) disguise="$2"; shift 2 ;;
            --client) client_name="$2"; shift 2 ;;
            --name) name="$2"; shift 2 ;;
            --new-name) new_name="$2"; shift 2 ;;
            --domain) domain="$2"; shift 2 ;;
            --help|-h) show_help; exit 0 ;;
            *) shift ;;
        esac
    done

    case "$cmd" in
        # Installation
        install)
            fresh_install "$port" "${disguise}:443" "$disguise" "$client_name"
            ;;
        manage)
            manage_server
            ;;

        # Client commands
        list-clients)
            list_clients
            ;;
        add-client)
            add_client "$name"
            ;;
        remove-client)
            remove_client "$name"
            ;;
        rename-client)
            rename_client "$name" "$new_name"
            ;;
        reset-uuid)
            reset_client_uuid "$name"
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
        qr)
            show_qr "$name"
            ;;
        qr-all)
            show_all_qr
            ;;
        export)
            export_clients
            ;;

        # Server commands
        status)
            show_status
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
        restart)
            restart_service
            ;;
        test)
            test_config
            ;;
        logs)
            show_logs
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
