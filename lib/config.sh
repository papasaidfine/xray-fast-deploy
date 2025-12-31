#!/bin/bash
# Configuration management and key generation

# Source common if not already loaded
[[ -z "$NC" ]] && source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Generate X25519 key pair (sets PRIVATE_KEY and PUBLIC_KEY globals)
generate_keys() {
    local output=$(xray x25519)
    PRIVATE_KEY=$(echo "$output" | grep "Private key:" | awk '{print $3}')
    PUBLIC_KEY=$(echo "$output" | grep "Public key:" | awk '{print $3}')
}

# Generate UUID
generate_uuid() {
    cat /proc/sys/kernel/random/uuid
}

# Generate short ID (random hex)
generate_short_id() {
    openssl rand -hex 8
}

# Get public key from private key
get_public_key() {
    local private_key="$1"
    xray x25519 -i "$private_key" 2>/dev/null | grep "Public key:" | awk '{print $3}'
}

# Create initial Xray configuration
create_config() {
    local uuid="$1"
    local private_key="$2"
    local dest="$3"
    local sni="$4"
    local port="$5"
    local short_id="$6"

    mkdir -p /usr/local/etc/xray
    mkdir -p "$INFO_DIR"

    cat > "$CONFIG_FILE" <<EOF
{
  "log": {
    "loglevel": "warning"
  },
  "inbounds": [
    {
      "port": $port,
      "protocol": "vless",
      "settings": {
        "clients": [
          {
            "id": "$uuid",
            "flow": "xtls-rprx-vision",
            "email": "default@xray"
          }
        ],
        "decryption": "none"
      },
      "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "realitySettings": {
          "show": false,
          "dest": "$dest",
          "serverNames": [
            "$sni"
          ],
          "privateKey": "$private_key",
          "shortIds": [
            "",
            "$short_id"
          ]
        }
      },
      "sniffing": {
        "enabled": true,
        "destOverride": [
          "http",
          "tls"
        ]
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "freedom",
      "tag": "direct"
    },
    {
      "protocol": "blackhole",
      "tag": "block"
    }
  ]
}
EOF

    chown -R xray:xray /usr/local/etc/xray
    chmod 600 "$CONFIG_FILE"
}

# Test configuration file
test_config() {
    log_info "Testing configuration..."
    if /usr/local/bin/xray run -test -config "$CONFIG_FILE"; then
        log_success "Configuration is valid"
        return 0
    else
        log_error "Configuration has errors"
        return 1
    fi
}

# Regenerate server keys (dangerous operation)
regenerate_keys() {
    echo ""
    log_warn "This will regenerate server keys. ALL existing client configs will stop working!"
    read -p "Are you sure? (yes/no): " confirm

    if [[ "$confirm" != "yes" ]]; then
        log_info "Cancelled"
        return 0
    fi

    generate_keys
    local short_id=$(generate_short_id)

    # Update config
    local tmp_file=$(mktemp)
    jq ".inbounds[0].streamSettings.realitySettings.privateKey = \"$PRIVATE_KEY\" |
        .inbounds[0].streamSettings.realitySettings.shortIds = [\"\", \"$short_id\"]" "$CONFIG_FILE" > "$tmp_file"
    mv "$tmp_file" "$CONFIG_FILE"
    chown xray:xray "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"

    # Update server info
    save_server_info "$PUBLIC_KEY" \
        "$(jq -r '.inbounds[0].port' "$CONFIG_FILE")" \
        "$(jq -r '.inbounds[0].streamSettings.realitySettings.serverNames[0]' "$CONFIG_FILE")"

    systemctl restart xray

    log_success "Keys regenerated"
    echo -e "${BLUE}New Public Key:${NC} $PUBLIC_KEY"
    echo ""
    log_warn "Update all clients with the new public key!"
}
