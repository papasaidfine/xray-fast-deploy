#!/bin/bash
# Interactive menu

# Source common if not already loaded
[[ -z "$NC" ]] && source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Display menu options
show_menu() {
    echo ""
    echo -e "${CYAN}=== Xray REALITY Management ===${NC}"
    echo ""
    echo "  1. Show server status"
    echo "  2. List clients"
    echo "  3. Add new client"
    echo "  4. Remove client"
    echo "  5. Show client config"
    echo "  6. Export all client configs"
    echo "  7. Change disguised website"
    echo "  8. Change port"
    echo "  9. Regenerate server keys"
    echo " 10. View logs (last 50 lines)"
    echo " 11. Follow logs (live)"
    echo " 12. Restart service"
    echo " 13. Test configuration"
    echo " 14. Uninstall"
    echo "  0. Exit"
    echo ""
}

# Interactive management loop
manage_server() {
    while true; do
        show_menu
        read -p "Select option: " choice

        case "$choice" in
            1)
                show_status
                ;;
            2)
                list_clients
                ;;
            3)
                add_client
                ;;
            4)
                remove_client
                ;;
            5)
                list_clients
                read -p "Enter client name: " name
                local uuid=$(get_client_uuid "$name")
                if [[ -n "$uuid" ]]; then
                    show_client_config "$uuid" "$name"
                else
                    log_error "Client not found"
                fi
                ;;
            6)
                export_clients
                ;;
            7)
                change_disguise
                ;;
            8)
                change_port
                ;;
            9)
                regenerate_keys
                ;;
            10)
                show_logs 50
                ;;
            11)
                follow_logs
                ;;
            12)
                restart_service
                ;;
            13)
                test_config
                ;;
            14)
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
