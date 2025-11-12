#!/bin/bash
set -e

# TodoList API Automated Deployment Script
# This script is designed to be run by CI/CD pipelines

DEPLOY_DIR="$HOME/todolist-api"
BIN_DIR="$DEPLOY_DIR/bin"
SERVICE_NAME="todolist-api"
LOG_DIR="$DEPLOY_DIR/logs"
BACKUP_DIR="$DEPLOY_DIR/backups"

echo "========================================="
echo "TodoList API Deployment Script"
echo "========================================="
echo "Deployment directory: $DEPLOY_DIR"
echo "Timestamp: $(date)"
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then
   echo "Please do not run this script as root"
   exit 1
fi

# Create necessary directories
echo "[1/8] Creating directories..."
mkdir -p "$BIN_DIR" "$LOG_DIR" "$BACKUP_DIR"

# Backup existing binary if it exists
if [ -f "$BIN_DIR/todolist-api" ]; then
    echo "[2/8] Backing up existing binary..."
    BACKUP_FILE="$BACKUP_DIR/todolist-api-$(date +%Y%m%d-%H%M%S)"
    cp "$BIN_DIR/todolist-api" "$BACKUP_FILE"
    echo "Backup saved to: $BACKUP_FILE"
else
    echo "[2/8] No existing binary to backup"
fi

# Check if new binary exists
if [ ! -f "$BIN_DIR/todolist-api" ]; then
    echo "Error: Binary not found at $BIN_DIR/todolist-api"
    echo "Make sure the CI/CD pipeline copied the binary before running this script"
    exit 1
fi

# Verify binary is executable
echo "[3/8] Verifying binary..."
chmod +x "$BIN_DIR/todolist-api"
if ! "$BIN_DIR/todolist-api" --version 2>/dev/null; then
    echo "Warning: Binary version check failed, but continuing..."
fi

# Stop existing service if running
echo "[4/8] Stopping existing service..."
if sudo systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "Service is running, stopping..."
    sudo systemctl stop "$SERVICE_NAME"
    sleep 2
else
    echo "Service is not running"
fi

# Create or update systemd service file
echo "[5/8] Creating systemd service..."
sudo tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null <<EOF
[Unit]
Description=TodoList API Service
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=$USER
WorkingDirectory=$DEPLOY_DIR
ExecStart=$BIN_DIR/todolist-api
Restart=always
RestartSec=5
StandardOutput=append:$LOG_DIR/todolist-api.log
StandardError=append:$LOG_DIR/todolist-api-error.log

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$LOG_DIR

# Environment file (if exists)
EnvironmentFile=-$DEPLOY_DIR/.env

# Resource limits
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd daemon
echo "[6/8] Reloading systemd daemon..."
sudo systemctl daemon-reload

# Enable service to start on boot
echo "[7/8] Enabling service..."
sudo systemctl enable "$SERVICE_NAME"

# Start the service
echo "[8/8] Starting service..."
sudo systemctl start "$SERVICE_NAME"

# Wait for service to start
echo ""
echo "Waiting for service to start..."
sleep 3

# Check service status
if sudo systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "✅ Service started successfully!"

    # Show service status
    echo ""
    echo "Service Status:"
    sudo systemctl status "$SERVICE_NAME" --no-pager -l

    # Check health endpoint
    echo ""
    echo "Checking health endpoint..."
    if command -v curl &> /dev/null; then
        sleep 2
        if curl -f -s http://localhost:8080/health > /dev/null; then
            echo "✅ Health check passed!"
            echo ""
            echo "Health details:"
            curl -s http://localhost:8080/health/detailed | jq . || curl -s http://localhost:8080/health/detailed
        else
            echo "⚠️  Health check failed - service may still be starting"
        fi
    else
        echo "curl not available, skipping health check"
    fi
else
    echo "❌ Service failed to start!"
    echo ""
    echo "Recent logs:"
    sudo journalctl -u "$SERVICE_NAME" -n 50 --no-pager
    exit 1
fi

echo ""
echo "========================================="
echo "Deployment completed successfully!"
echo "========================================="
echo ""
echo "Useful commands:"
echo "  View logs:         sudo journalctl -u $SERVICE_NAME -f"
echo "  Service status:    sudo systemctl status $SERVICE_NAME"
echo "  Restart service:   sudo systemctl restart $SERVICE_NAME"
echo "  Stop service:      sudo systemctl stop $SERVICE_NAME"
echo "  Health check:      curl http://localhost:8080/health/detailed"
echo ""
