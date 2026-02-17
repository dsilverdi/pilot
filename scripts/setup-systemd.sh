#!/bin/bash
#
# Pilot Gateway systemd Service Setup Script
# This script helps set up pilot-gateway as a systemd service
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
SERVICE_NAME="pilot-gateway"
INSTALL_DIR="/opt/pilot"
DATA_DIR="/var/lib/pilot"
SERVICE_USER="pilot"
SERVICE_GROUP="pilot"

# Print colored message
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Create service user
create_user() {
    print_info "Creating service user: $SERVICE_USER"
    if id "$SERVICE_USER" &>/dev/null; then
        print_warn "User $SERVICE_USER already exists"
    else
        useradd --system --no-create-home --shell /bin/false "$SERVICE_USER"
        print_info "User $SERVICE_USER created"
    fi
}

# Create directories
create_directories() {
    print_info "Creating directories..."

    # Install directory
    mkdir -p "$INSTALL_DIR/bin"

    # Data directory (PILOT_HOME)
    mkdir -p "$DATA_DIR/sessions"
    mkdir -p "$DATA_DIR/skills"

    # Set ownership
    chown -R "$SERVICE_USER:$SERVICE_GROUP" "$DATA_DIR"
    chmod 750 "$DATA_DIR"

    print_info "Directories created:"
    print_info "  - Install: $INSTALL_DIR"
    print_info "  - Data:    $DATA_DIR"
}

# Copy binaries
copy_binaries() {
    print_info "Copying binaries..."

    if [ -f "./bin/pilot-gateway" ]; then
        cp ./bin/pilot-gateway "$INSTALL_DIR/bin/"
        chmod +x "$INSTALL_DIR/bin/pilot-gateway"
    elif [ -f "./pilot-gateway" ]; then
        cp ./pilot-gateway "$INSTALL_DIR/bin/"
        chmod +x "$INSTALL_DIR/bin/pilot-gateway"
    else
        print_error "pilot-gateway binary not found. Run 'make build' first."
        exit 1
    fi

    if [ -f "./bin/pilot" ]; then
        cp ./bin/pilot "$INSTALL_DIR/bin/"
        chmod +x "$INSTALL_DIR/bin/pilot"
    elif [ -f "./pilot" ]; then
        cp ./pilot "$INSTALL_DIR/bin/"
        chmod +x "$INSTALL_DIR/bin/pilot"
    fi

    print_info "Binaries copied to $INSTALL_DIR/bin/"
}

# Create environment file
create_env_file() {
    local env_file="$INSTALL_DIR/.env"

    if [ -f "$env_file" ]; then
        print_warn "Environment file already exists: $env_file"
        print_warn "Skipping creation. Edit manually if needed."
        return
    fi

    print_info "Creating environment file..."

    cat > "$env_file" << 'EOF'
# Pilot Gateway Configuration
# Edit this file with your actual values

# Anthropic Authentication (required - set one)
ANTHROPIC_API_KEY=
#ANTHROPIC_OAUTH_TOKEN=

# Data directory (managed by systemd setup)
PILOT_HOME=/var/lib/pilot

# Gateway settings
GATEWAY_ADDR=:8080

# Web search (optional)
#SEARXNG_URL=http://localhost:8081

# Telegram bot (optional)
#TELEGRAM_BOT_TOKEN=
#TELEGRAM_ALLOWED_USERS=
EOF

    chmod 600 "$env_file"
    chown "$SERVICE_USER:$SERVICE_GROUP" "$env_file"

    print_info "Environment file created: $env_file"
    print_warn "Remember to edit $env_file with your API key!"
}

# Create systemd service file
create_service_file() {
    local service_file="/etc/systemd/system/${SERVICE_NAME}.service"

    print_info "Creating systemd service file..."

    cat > "$service_file" << EOF
[Unit]
Description=Pilot Gateway - AI Agent HTTP Service
Documentation=https://github.com/dsilverdi/pilot
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_GROUP
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/bin/pilot-gateway
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Environment
EnvironmentFile=$INSTALL_DIR/.env

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

    print_info "Service file created: $service_file"
}

# Generate API key
generate_api_key() {
    print_info "Generating API key for gateway..."

    # Run as service user with correct PILOT_HOME
    sudo -u "$SERVICE_USER" PILOT_HOME="$DATA_DIR" "$INSTALL_DIR/bin/pilot" api-key generate --name default || {
        print_warn "Failed to generate API key. You can do this later with:"
        print_warn "  sudo -u $SERVICE_USER PILOT_HOME=$DATA_DIR $INSTALL_DIR/bin/pilot api-key generate --name mykey"
    }
}

# Enable and start service
start_service() {
    print_info "Enabling and starting service..."

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"

    sleep 2

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        print_info "Service started successfully!"
    else
        print_error "Service failed to start. Check logs with:"
        print_error "  journalctl -u $SERVICE_NAME -f"
    fi
}

# Print status and next steps
print_status() {
    echo ""
    echo "============================================"
    echo "  Pilot Gateway Setup Complete"
    echo "============================================"
    echo ""
    echo "Service Status:"
    systemctl status "$SERVICE_NAME" --no-pager || true
    echo ""
    echo "Next Steps:"
    echo ""
    echo "1. Edit the environment file with your API key:"
    echo "   sudo nano $INSTALL_DIR/.env"
    echo ""
    echo "2. Restart the service after editing:"
    echo "   sudo systemctl restart $SERVICE_NAME"
    echo ""
    echo "3. Check service logs:"
    echo "   journalctl -u $SERVICE_NAME -f"
    echo ""
    echo "4. Test the gateway:"
    echo "   curl http://localhost:8080/health"
    echo ""
    echo "5. Generate API keys (as service user):"
    echo "   sudo -u $SERVICE_USER PILOT_HOME=$DATA_DIR $INSTALL_DIR/bin/pilot api-key generate --name myapp"
    echo ""
    echo "Data directory: $DATA_DIR"
    echo "Config file:    $INSTALL_DIR/.env"
    echo ""
}

# Uninstall function
uninstall() {
    print_warn "Uninstalling pilot-gateway service..."

    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    systemctl disable "$SERVICE_NAME" 2>/dev/null || true
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    systemctl daemon-reload

    print_info "Service removed. Data preserved in $DATA_DIR"
    print_info "To fully remove, also run:"
    print_info "  sudo rm -rf $INSTALL_DIR $DATA_DIR"
    print_info "  sudo userdel $SERVICE_USER"
}

# Show help
show_help() {
    echo "Pilot Gateway systemd Setup Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  install     Install and configure pilot-gateway as systemd service (default)"
    echo "  uninstall   Remove the systemd service (preserves data)"
    echo "  status      Show service status"
    echo "  help        Show this help"
    echo ""
    echo "Environment variables:"
    echo "  SERVICE_USER   Service user (default: pilot)"
    echo "  INSTALL_DIR    Installation directory (default: /opt/pilot)"
    echo "  DATA_DIR       Data directory (default: /var/lib/pilot)"
    echo ""
}

# Main
main() {
    local command="${1:-install}"

    case "$command" in
        install)
            check_root
            echo ""
            print_info "Installing pilot-gateway as systemd service..."
            echo ""
            create_user
            create_directories
            copy_binaries
            create_env_file
            create_service_file
            start_service
            print_status
            ;;
        uninstall)
            check_root
            uninstall
            ;;
        status)
            systemctl status "$SERVICE_NAME"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
