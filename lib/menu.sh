#!/bin/bash
# Interactive menu

# Source common if not already loaded
[[ -z "$NC" ]] && source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Display menu options
show_menu() {
    local count=$(get_client_count)
    echo ""
    echo -e "${CYAN}=== Xray REALITY Management ===${NC}"
    echo ""
    echo -e "${YELLOW}[ Clients: $count ]${NC}"
    echo "  1. List clients"
    echo "  2. Add new client"
    echo "  3. Remove client"
    echo "  4. Rename client"
    echo "  5. Reset client UUID"
    echo "  6. Show client config"
    echo "  7. Show client QR code"
    echo "  8. Show all QR codes"
    echo "  9. Export all VLESS links"
    echo ""
    echo -e "${YELLOW}[ Server ]${NC}"
    echo " 10. Show server status"
    echo " 11. Change disguised website"
    echo " 12. Change port"
    echo " 13. Regenerate server keys"
    echo " 14. Restart service"
    echo " 15. Test configuration"
    echo " 16. View logs (last 50 lines)"
    echo " 17. Follow logs (live)"
    echo ""
    echo -e "${YELLOW}[ System ]${NC}"
    echo " 18. Uninstall"
    echo "  0. Exit"
    echo ""
}

# Interactive management loop
manage_server() {
    while true; do
        show_menu
        read -p "Select option: " choice

        case "$choice" in
            # Client management
            1)
                list_clients
                ;;
            2)
                add_client
                ;;
            3)
                remove_client
                ;;
            4)
                rename_client
                ;;
            5)
                reset_client_uuid
                ;;
            6)
                list_clients
                read -p "Enter client name: " name
                local uuid=$(get_client_uuid "$name")
                if [[ -n "$uuid" ]]; then
                    show_client_config "$uuid" "$name"
                else
                    log_error "Client not found"
                fi
                ;;
            7)
                show_qr
                ;;
            8)
                show_all_qr
                ;;
            9)
                export_clients
                ;;
            # Server management
            10)
                show_status
                ;;
            11)
                change_disguise
                ;;
            12)
                change_port
                ;;
            13)
                regenerate_keys
                ;;
            14)
                restart_service
                ;;
            15)
                test_config
                ;;
            16)
                show_logs 50
                ;;
            17)
                follow_logs
                ;;
            # System
            18)
                uninstall_xray
                exit 0
                ;;
            0)
                exit 0
                ;;
            *)
                log_error "Invalid option"
                ;;
        esac

        echo ""
        read -p "Press Enter to continue..."
    done
}
