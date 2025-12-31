#!/bin/bash
# Server management functions

# Source common if not already loaded
[[ -z "$NC" ]] && source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Show server status
show_status() {
    echo ""
    echo -e "${CYAN}=== Xray REALITY Server Status ===${NC}"

    if is_xray_running; then
        echo -e "Service: ${GREEN}Running${NC}"
    else
        echo -e "Service: ${RED}Stopped${NC}"
    fi

    local server_ip=$(get_server_ip)
    echo -e "Server IP: ${BLUE}$server_ip${NC}"

    if [[ -f "$CONFIG_FILE" ]]; then
        local port=$(jq -r '.inbounds[0].port' "$CONFIG_FILE")
        local dest=$(jq -r '.inbounds[0].streamSettings.realitySettings.dest' "$CONFIG_FILE")
        local client_count=$(jq '.inbounds[0].settings.clients | length' "$CONFIG_FILE")

        echo -e "Port: ${BLUE}$port${NC}"
        echo -e "Disguise: ${BLUE}$dest${NC}"
        echo -e "Clients: ${BLUE}$client_count${NC}"

        # Check port listening
        if ss -tlnp 2>/dev/null | grep -q ":$port"; then
            echo -e "Port Status: ${GREEN}Listening${NC}"
        else
            echo -e "Port Status: ${RED}Not Listening${NC}"
        fi
    fi

    # BBR status
    if sysctl net.ipv4.tcp_congestion_control 2>/dev/null | grep -q bbr; then
        echo -e "BBR: ${GREEN}Enabled${NC}"
    else
        echo -e "BBR: ${YELLOW}Disabled${NC}"
    fi

    echo ""
}

# Change disguised website
change_disguise() {
    local new_dest="$1"
    local new_sni="$2"

    if [[ -z "$new_dest" ]]; then
        echo ""
        echo "Current disguise: $(jq -r '.inbounds[0].streamSettings.realitySettings.dest' "$CONFIG_FILE")"
        echo ""
        echo "Popular options:"
        echo "  1. www.apple.com"
        echo "  2. www.microsoft.com"
        echo "  3. www.google.com"
        echo "  4. www.cloudflare.com"
        echo "  5. www.amazon.com"
        echo "  6. Custom"
        echo ""
        read -p "Select option (1-6): " choice

        case "$choice" in
            1) new_sni="www.apple.com" ;;
            2) new_sni="www.microsoft.com" ;;
            3) new_sni="www.google.com" ;;
            4) new_sni="www.cloudflare.com" ;;
            5) new_sni="www.amazon.com" ;;
            6) read -p "Enter custom domain: " new_sni ;;
            *) log_error "Invalid option"; return 1 ;;
        esac
        new_dest="${new_sni}:443"
    fi

    if [[ -z "$new_sni" ]]; then
        new_sni=$(echo "$new_dest" | cut -d: -f1)
    fi

    # Update config
    local tmp_file=$(mktemp)
    jq ".inbounds[0].streamSettings.realitySettings.dest = \"$new_dest\" |
        .inbounds[0].streamSettings.realitySettings.serverNames = [\"$new_sni\"]" "$CONFIG_FILE" > "$tmp_file"
    mv "$tmp_file" "$CONFIG_FILE"
    chown xray:xray "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"

    # Update server info
    if [[ -f "$INFO_DIR/server.info" ]]; then
        sed -i "s/^SNI=.*/SNI=$new_sni/" "$INFO_DIR/server.info"
    fi

    systemctl restart xray
    log_success "Disguise changed to: $new_sni"
}

# Change server port
change_port() {
    local new_port="$1"

    if [[ -z "$new_port" ]]; then
        echo ""
        echo "Current port: $(jq -r '.inbounds[0].port' "$CONFIG_FILE")"
        read -p "Enter new port: " new_port
    fi

    if ! [[ "$new_port" =~ ^[0-9]+$ ]] || [[ "$new_port" -lt 1 ]] || [[ "$new_port" -gt 65535 ]]; then
        log_error "Invalid port number"
        return 1
    fi

    # Update config
    local tmp_file=$(mktemp)
    jq ".inbounds[0].port = $new_port" "$CONFIG_FILE" > "$tmp_file"
    mv "$tmp_file" "$CONFIG_FILE"
    chown xray:xray "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"

    # Update server info
    if [[ -f "$INFO_DIR/server.info" ]]; then
        sed -i "s/^PORT=.*/PORT=$new_port/" "$INFO_DIR/server.info"
    fi

    systemctl restart xray
    log_success "Port changed to: $new_port"
    log_warn "Remember to update your firewall rules!"
}

# Show recent logs
show_logs() {
    local lines="${1:-50}"
    journalctl -u xray -n "$lines" --no-pager
}

# Follow logs in real-time
follow_logs() {
    journalctl -u xray -f
}

# Restart Xray service
restart_service() {
    systemctl restart xray
    log_success "Service restarted"
}

# Start Xray service
start_service() {
    systemctl start xray
    log_success "Service started"
}

# Stop Xray service
stop_service() {
    systemctl stop xray
    log_success "Service stopped"
}

# Change log level
change_log_level() {
    local level="$1"

    if [[ -z "$level" ]]; then
        local current=$(jq -r '.log.loglevel' "$CONFIG_FILE" 2>/dev/null)
        echo ""
        echo "Current log level: $current"
        echo ""
        echo "Available levels:"
        echo "  1. none     - No logging"
        echo "  2. error    - Errors only"
        echo "  3. warning  - Warnings and errors (default)"
        echo "  4. info     - Connection info + warnings + errors"
        echo "  5. debug    - Verbose debugging"
        echo ""
        read -p "Select level (1-5): " choice

        case "$choice" in
            1) level="none" ;;
            2) level="error" ;;
            3) level="warning" ;;
            4) level="info" ;;
            5) level="debug" ;;
            *) log_error "Invalid option"; return 1 ;;
        esac
    fi

    # Validate level
    if [[ ! "$level" =~ ^(none|error|warning|info|debug)$ ]]; then
        log_error "Invalid log level: $level"
        return 1
    fi

    # Update config
    local tmp_file=$(mktemp)
    jq ".log.loglevel = \"$level\"" "$CONFIG_FILE" > "$tmp_file"
    mv "$tmp_file" "$CONFIG_FILE"
    chown xray:xray "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"

    systemctl restart xray
    log_success "Log level changed to: $level"
}
