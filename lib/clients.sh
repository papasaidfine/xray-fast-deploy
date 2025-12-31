#!/bin/bash
# Client management functions

# Source common if not already loaded
[[ -z "$NC" ]] && source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Get all clients from config
get_clients() {
    if [[ -f "$CONFIG_FILE" ]]; then
        jq -r '.inbounds[0].settings.clients[] | "\(.email):\(.id)"' "$CONFIG_FILE" 2>/dev/null
    fi
}

# Generate VLESS share link
generate_vless_link() {
    local uuid="$1"
    local server_ip="$2"
    local port="$3"
    local public_key="$4"
    local sni="$5"
    local name="$6"
    local short_id="$7"

    local encoded_name=$(echo -n "$name" | jq -sRr @uri)
    echo "vless://${uuid}@${server_ip}:${port}?encryption=none&flow=xtls-rprx-vision&security=reality&sni=${sni}&fp=chrome&pbk=${public_key}&sid=${short_id}&type=tcp&headerType=none#${encoded_name}"
}

# Show client configuration with QR code
show_client_config() {
    local uuid="$1"
    local name="$2"

    load_server_info

    local server_ip=$(get_server_ip)
    local port=${PORT:-443}
    local sni=${SNI:-www.apple.com}
    local public_key=${PUBLIC_KEY:-""}
    local short_id=""

    # Get short_id from config
    if [[ -f "$CONFIG_FILE" ]]; then
        short_id=$(jq -r '.inbounds[0].streamSettings.realitySettings.shortIds[1] // ""' "$CONFIG_FILE")
    fi

    # Get public key if not in server info
    if [[ -z "$public_key" && -f "$CONFIG_FILE" ]]; then
        local private_key=$(jq -r '.inbounds[0].streamSettings.realitySettings.privateKey' "$CONFIG_FILE")
        public_key=$(get_public_key "$private_key")
    fi

    local vless_link=$(generate_vless_link "$uuid" "$server_ip" "$port" "$public_key" "$sni" "$name" "$short_id")

    echo ""
    echo -e "${CYAN}========== Client Configuration ==========${NC}"
    echo -e "${BLUE}Name:${NC} $name"
    echo -e "${BLUE}Server:${NC} $server_ip"
    echo -e "${BLUE}Port:${NC} $port"
    echo -e "${BLUE}UUID:${NC} $uuid"
    echo -e "${BLUE}Flow:${NC} xtls-rprx-vision"
    echo -e "${BLUE}Security:${NC} reality"
    echo -e "${BLUE}SNI:${NC} $sni"
    echo -e "${BLUE}Fingerprint:${NC} chrome"
    echo -e "${BLUE}Public Key:${NC} $public_key"
    echo -e "${BLUE}Short ID:${NC} $short_id"
    echo ""
    echo -e "${YELLOW}VLESS Link:${NC}"
    echo "$vless_link"
    echo ""
    echo -e "${YELLOW}QR Code:${NC}"
    qrencode -t ANSIUTF8 "$vless_link"
    echo -e "${CYAN}===========================================${NC}"
}

# Add new client
add_client() {
    local name="$1"
    local uuid=$(generate_uuid)

    if [[ -z "$name" ]]; then
        read -p "Enter client name: " name
        [[ -z "$name" ]] && name="client-$(date +%s)"
    fi

    # Check if name already exists
    if jq -e ".inbounds[0].settings.clients[] | select(.email == \"$name\")" "$CONFIG_FILE" >/dev/null 2>&1; then
        log_error "Client '$name' already exists"
        return 1
    fi

    # Add client to config
    local tmp_file=$(mktemp)
    jq ".inbounds[0].settings.clients += [{\"id\": \"$uuid\", \"flow\": \"xtls-rprx-vision\", \"email\": \"$name\"}]" "$CONFIG_FILE" > "$tmp_file"
    mv "$tmp_file" "$CONFIG_FILE"
    chown xray:xray "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"

    # Restart xray
    systemctl restart xray

    log_success "Client '$name' added"
    show_client_config "$uuid" "$name"
}

# Remove client
remove_client() {
    local name="$1"

    if [[ -z "$name" ]]; then
        echo ""
        echo "Current clients:"
        list_clients
        echo ""
        read -p "Enter client name to remove: " name
    fi

    if [[ -z "$name" ]]; then
        log_error "No client name provided"
        return 1
    fi

    # Check if client exists
    if ! jq -e ".inbounds[0].settings.clients[] | select(.email == \"$name\")" "$CONFIG_FILE" >/dev/null 2>&1; then
        log_error "Client '$name' not found"
        return 1
    fi

    # Check if it's the last client
    local client_count=$(jq '.inbounds[0].settings.clients | length' "$CONFIG_FILE")
    if [[ "$client_count" -le 1 ]]; then
        log_error "Cannot remove the last client. Add another client first."
        return 1
    fi

    # Remove client
    local tmp_file=$(mktemp)
    jq "del(.inbounds[0].settings.clients[] | select(.email == \"$name\"))" "$CONFIG_FILE" > "$tmp_file"
    mv "$tmp_file" "$CONFIG_FILE"
    chown xray:xray "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"

    systemctl restart xray
    log_success "Client '$name' removed"
}

# List all clients
list_clients() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        log_error "Config file not found"
        return 1
    fi

    echo ""
    echo -e "${CYAN}=== Registered Clients ===${NC}"
    local idx=1
    while IFS=: read -r email uuid; do
        echo -e "${BLUE}$idx.${NC} $email"
        echo -e "   UUID: $uuid"
        ((idx++))
    done < <(get_clients)
    echo ""
}

# Export all client configurations
export_clients() {
    load_server_info

    local server_ip=$(get_server_ip)
    local port=${PORT:-443}
    local sni=${SNI:-www.apple.com}
    local public_key=${PUBLIC_KEY:-""}
    local short_id=""

    if [[ -f "$CONFIG_FILE" ]]; then
        short_id=$(jq -r '.inbounds[0].streamSettings.realitySettings.shortIds[1] // ""' "$CONFIG_FILE")
    fi

    if [[ -z "$public_key" && -f "$CONFIG_FILE" ]]; then
        local private_key=$(jq -r '.inbounds[0].streamSettings.realitySettings.privateKey' "$CONFIG_FILE")
        public_key=$(get_public_key "$private_key")
    fi

    echo ""
    echo -e "${CYAN}=== All Client Configurations ===${NC}"

    while IFS=: read -r email uuid; do
        local vless_link=$(generate_vless_link "$uuid" "$server_ip" "$port" "$public_key" "$sni" "$email" "$short_id")
        echo ""
        echo -e "${YELLOW}--- $email ---${NC}"
        echo "$vless_link"
    done < <(get_clients)

    echo ""
}

# Get client UUID by name
get_client_uuid() {
    local name="$1"
    jq -r ".inbounds[0].settings.clients[] | select(.email == \"$name\") | .id" "$CONFIG_FILE" 2>/dev/null
}
