#!/bin/bash
# Installation and dependency management

# Source common if not already loaded
[[ -z "$NC" ]] && source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Install dependencies
install_dependencies() {
    log_info "Installing dependencies..."
    apt update -qq
    apt install -y -qq curl wget jq qrencode unzip openssl >/dev/null 2>&1
    log_success "Dependencies installed"
}

# Install Xray
install_xray() {
    log_info "Installing Xray..."
    bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install >/dev/null 2>&1

    # Create xray user if not exists
    id xray &>/dev/null || useradd -r -M -s /usr/sbin/nologin xray

    # Create service file
    cat > "$SERVICE_FILE" <<'EOF'
[Unit]
Description=Xray Service
Documentation=https://github.com/xtls
After=network.target nss-lookup.target

[Service]
User=xray
Group=xray
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
NoNewPrivileges=true
ExecStart=/usr/local/bin/xray run -config /usr/local/etc/xray/config.json
Restart=on-failure
RestartPreventExitStatus=23
LimitNPROC=10000
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
EOF

    chmod +x /usr/local/bin/xray
    systemctl daemon-reload
    log_success "Xray installed"
}

# Configure OS firewall to allow Xray port
configure_firewall() {
    local port="$1"
    log_info "Configuring firewall for port $port..."

    # Check if iptables has a REJECT/DROP rule that would block traffic
    if iptables -L INPUT -n 2>/dev/null | grep -qE "(REJECT|DROP).*0\.0\.0\.0/0.*0\.0\.0\.0/0"; then
        # Find the line number of the REJECT/DROP rule
        local reject_line=$(iptables -L INPUT -n --line-numbers 2>/dev/null | grep -E "(REJECT|DROP).*0\.0\.0\.0/0.*0\.0\.0\.0/0" | head -1 | awk '{print $1}')

        if [[ -n "$reject_line" ]]; then
            # Check if port rule already exists
            if ! iptables -L INPUT -n 2>/dev/null | grep -q "dpt:$port"; then
                # Insert rule before REJECT/DROP
                iptables -I INPUT "$reject_line" -p tcp --dport "$port" -m state --state NEW -j ACCEPT
                log_success "Added iptables rule for port $port"
            else
                log_info "Port $port already allowed in iptables"
            fi
        fi
    else
        # No REJECT rule, check if we need to add ACCEPT anyway
        if ! iptables -L INPUT -n 2>/dev/null | grep -q "dpt:$port"; then
            iptables -A INPUT -p tcp --dport "$port" -j ACCEPT 2>/dev/null || true
        fi
    fi

    # Handle firewalld if active
    if systemctl is-active --quiet firewalld 2>/dev/null; then
        firewall-cmd --add-port="$port/tcp" --permanent >/dev/null 2>&1
        firewall-cmd --reload >/dev/null 2>&1
        log_success "Added firewalld rule for port $port"
    fi

    # Handle ufw if active
    if systemctl is-active --quiet ufw 2>/dev/null || ufw status 2>/dev/null | grep -q "Status: active"; then
        ufw allow "$port/tcp" >/dev/null 2>&1
        log_success "Added ufw rule for port $port"
    fi

    # Try to persist iptables rules
    if command -v netfilter-persistent &>/dev/null; then
        netfilter-persistent save >/dev/null 2>&1 || true
    elif [[ -f /etc/iptables/rules.v4 ]]; then
        iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
    fi
}

# Enable BBR congestion control
enable_bbr() {
    log_info "Enabling BBR congestion control..."

    # Load tcp_bbr module
    modprobe tcp_bbr 2>/dev/null || true

    # Add to sysctl if not present
    if ! grep -q "net.core.default_qdisc=fq" /etc/sysctl.conf 2>/dev/null; then
        echo "net.core.default_qdisc=fq" >> /etc/sysctl.conf
    fi
    if ! grep -q "net.ipv4.tcp_congestion_control=bbr" /etc/sysctl.conf 2>/dev/null; then
        echo "net.ipv4.tcp_congestion_control=bbr" >> /etc/sysctl.conf
    fi

    sysctl -p >/dev/null 2>&1

    # Verify
    if sysctl net.ipv4.tcp_congestion_control 2>/dev/null | grep -q bbr; then
        log_success "BBR enabled"
    else
        log_warn "BBR may not be fully enabled (kernel support required)"
    fi
}

# Uninstall Xray
uninstall_xray() {
    echo ""
    log_warn "This will completely remove Xray and all configurations!"
    read -p "Are you sure? (yes/no): " confirm

    if [[ "$confirm" != "yes" ]]; then
        log_info "Cancelled"
        return 0
    fi

    systemctl stop xray 2>/dev/null || true
    systemctl disable xray 2>/dev/null || true

    bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ remove >/dev/null 2>&1 || true

    rm -rf /usr/local/etc/xray
    rm -f "$SERVICE_FILE"
    rm -rf "$INFO_DIR"

    userdel xray 2>/dev/null || true

    systemctl daemon-reload

    log_success "Xray uninstalled"
}

# Full fresh installation
fresh_install() {
    local port="${1:-$DEFAULT_PORT}"
    local dest="${2:-$DEFAULT_DEST}"
    local sni="${3:-$DEFAULT_SNI}"
    local client_name="${4:-default}"

    echo ""
    echo -e "${CYAN}=== Xray REALITY Fresh Installation ===${NC}"
    echo ""

    check_os
    install_dependencies
    install_xray
    enable_bbr
    configure_firewall "$port"

    # Generate credentials
    generate_keys
    local uuid=$(generate_uuid)
    local short_id=$(generate_short_id)

    # Create config
    create_config "$uuid" "$PRIVATE_KEY" "$dest" "$sni" "$port" "$short_id"

    # Save server info
    save_server_info "$PUBLIC_KEY" "$port" "$sni"

    # Test config
    if ! test_config; then
        log_error "Configuration test failed"
        exit 1
    fi

    # Start service
    systemctl enable xray
    systemctl start xray

    sleep 2

    if is_xray_running; then
        log_success "Xray started successfully"
    else
        log_error "Failed to start Xray"
        journalctl -u xray -n 20 --no-pager
        exit 1
    fi

    # Show client config
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}    Installation Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"

    show_client_config "$uuid" "$client_name"

    echo ""
    log_info "Don't forget to open port $port in your firewall!"
    echo -e "  ${YELLOW}GCP:${NC} gcloud compute firewall-rules create allow-xray --allow tcp:$port --source-ranges 0.0.0.0/0"
    echo -e "  ${YELLOW}UFW:${NC} ufw allow $port/tcp"
    echo ""
}
