#!/bin/bash
#
# Update Application Script for OCI
# This script pulls the latest code, regenerates Swagger docs, rebuilds, and restarts the application
#
# Usage:
#   sudo ./update-application.sh              # Update to latest main branch
#   sudo ./update-application.sh develop      # Update to specific branch
#

set -e  # Exit on any error

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_DIR="/opt/todolist-api"
APP_NAME="todolist-api"
APP_USER="todolist"
SERVICE_NAME="todolist-api"
GO_PATH="/usr/local/go/bin/go"
SWAG_PATH="/usr/local/go/bin/swag"
BRANCH="${1:-main}"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

# Verify application directory exists
if [ ! -d "$APP_DIR" ]; then
    log_error "Application directory not found: $APP_DIR"
    exit 1
fi

log_info "======================================"
log_info "TodoList API Update Script"
log_info "======================================"
log_info "Branch: $BRANCH"
log_info "Application Directory: $APP_DIR"
log_info ""

# Step 1: Stop the service
log_info "Step 1/8: Stopping $SERVICE_NAME service..."
if systemctl is-active --quiet $SERVICE_NAME; then
    systemctl stop $SERVICE_NAME
    log_success "Service stopped"
else
    log_warning "Service was not running"
fi

# Step 2: Backup current binary
log_info "Step 2/8: Backing up current binary..."
if [ -f "$APP_DIR/$APP_NAME" ]; then
    BACKUP_NAME="${APP_NAME}.backup.$(date +%Y%m%d_%H%M%S)"
    sudo -u $APP_USER cp "$APP_DIR/$APP_NAME" "$APP_DIR/$BACKUP_NAME"
    log_success "Backup created: $BACKUP_NAME"
else
    log_warning "No existing binary to backup"
fi

# Step 3: Pull latest code
log_info "Step 3/8: Pulling latest code from Git..."
cd $APP_DIR

# Fetch latest changes
sudo -u $APP_USER git fetch origin --tags

# Show current and target commits
CURRENT_COMMIT=$(sudo -u $APP_USER git rev-parse HEAD)
TARGET_COMMIT=$(sudo -u $APP_USER git rev-parse origin/$BRANCH)

log_info "Current commit: $CURRENT_COMMIT"
log_info "Target commit:  $TARGET_COMMIT"

if [ "$CURRENT_COMMIT" = "$TARGET_COMMIT" ]; then
    log_warning "Already up to date! No changes to deploy."
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Update cancelled"
        systemctl start $SERVICE_NAME
        exit 0
    fi
fi

# Clean working directory to avoid -dirty version tags
log_info "Cleaning Git working directory..."
sudo -u $APP_USER git reset --hard HEAD
sudo -u $APP_USER git clean -fd

# Pull changes
sudo -u $APP_USER git pull origin $BRANCH
log_success "Code updated to latest $BRANCH"

# Step 4: Check if swag is installed
log_info "Step 4/8: Checking Swagger generator (swag)..."

# Check multiple possible swag locations
SWAG_LOCATIONS=(
    "/usr/local/go/bin/swag"
    "/home/$APP_USER/go/bin/swag"
    "$HOME/go/bin/swag"
)

SWAG_FOUND=false
for location in "${SWAG_LOCATIONS[@]}"; do
    if [ -f "$location" ]; then
        SWAG_PATH="$location"
        SWAG_FOUND=true
        log_success "swag found at $SWAG_PATH"
        break
    fi
done

if [ "$SWAG_FOUND" = false ]; then
    log_warning "swag not found, installing..."

    # Try installing with a specific compatible version for Go 1.24
    # swag v1.16.x is known to work well with Go 1.24
    log_info "Installing swag v1.16.4 (compatible with Go 1.24)..."

    sudo -u $APP_USER bash -c "export PATH=$PATH:/usr/local/go/bin && $GO_PATH install github.com/swaggo/swag/cmd/swag@v1.16.4" 2>&1 | tee /tmp/swag-install.log

    if [ ${PIPESTATUS[0]} -ne 0 ]; then
        log_error "Failed to install swag v1.16.4"
        log_info "Trying latest version..."
        sudo -u $APP_USER bash -c "export PATH=$PATH:/usr/local/go/bin && $GO_PATH install github.com/swaggo/swag/cmd/swag@latest" 2>&1 | tee /tmp/swag-install.log
    fi

    # Check if installation succeeded
    for location in "${SWAG_LOCATIONS[@]}"; do
        if [ -f "$location" ]; then
            SWAG_PATH="$location"
            SWAG_FOUND=true
            log_success "swag installed at $SWAG_PATH"
            break
        fi
    done

    if [ "$SWAG_FOUND" = false ]; then
        log_error "Failed to install swag. Check /tmp/swag-install.log for details"
        log_warning "Attempting to continue without regenerating Swagger docs..."
        log_warning "Swagger UI may not reflect latest changes!"
        SWAG_PATH=""
    fi
fi

# Step 5: Regenerate Swagger documentation
log_info "Step 5/8: Regenerating Swagger documentation..."
if [ -n "$SWAG_PATH" ] && [ -f "$SWAG_PATH" ]; then
    sudo -u $APP_USER bash -c "cd $APP_DIR && $SWAG_PATH init -g cmd/server/main.go -o docs"
    if [ $? -eq 0 ]; then
        log_success "Swagger documentation regenerated"
    else
        log_warning "Failed to regenerate Swagger docs, using existing docs"
    fi
else
    log_warning "Skipping Swagger regeneration (swag not available)"
    log_info "Using pre-generated docs from Git repository"
fi

# Step 6: Rebuild application
log_info "Step 6/8: Building application..."

# Get version information
VERSION=$(sudo -u $APP_USER bash -c "cd $APP_DIR && git describe --tags --always --dirty 2>/dev/null || echo 'deployed'")
COMMIT=$(sudo -u $APP_USER bash -c "cd $APP_DIR && git rev-parse --short HEAD 2>/dev/null || echo 'unknown'")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

log_info "Version: $VERSION"
log_info "Commit: $COMMIT"
log_info "Build Date: $BUILD_DATE"

# Build with version information
LDFLAGS="-X 'todolist-api/internal/version.Version=$VERSION' -X 'todolist-api/internal/version.Commit=$COMMIT' -X 'todolist-api/internal/version.BuildDate=$BUILD_DATE'"
sudo -u $APP_USER bash -c "cd $APP_DIR && export PATH=/usr/local/go/bin:\$PATH && $GO_PATH build -ldflags \"$LDFLAGS\" -o $APP_NAME cmd/server/main.go"

if [ ! -f "$APP_DIR/$APP_NAME" ]; then
    log_error "Build failed! Binary not created"

    # Restore backup if available
    if [ -f "$APP_DIR/$BACKUP_NAME" ]; then
        log_info "Restoring backup..."
        sudo -u $APP_USER cp "$APP_DIR/$BACKUP_NAME" "$APP_DIR/$APP_NAME"
        log_warning "Backup restored, starting old version"
    fi

    systemctl start $SERVICE_NAME
    exit 1
fi

log_success "Application built successfully"

# Step 7: Set SELinux context (if SELinux is enabled)
log_info "Step 7/8: Configuring SELinux..."
if command -v getenforce &> /dev/null; then
    SELINUX_STATUS=$(getenforce)
    if [ "$SELINUX_STATUS" != "Disabled" ]; then
        chcon -t bin_t $APP_DIR/$APP_NAME 2>/dev/null || true
        log_success "SELinux context set"
    else
        log_info "SELinux is disabled, skipping"
    fi
else
    log_info "SELinux not found, skipping"
fi

# Step 8: Start the service
log_info "Step 8/8: Starting $SERVICE_NAME service..."
systemctl start $SERVICE_NAME

# Wait a moment for service to start
sleep 2

# Verify service is running
if systemctl is-active --quiet $SERVICE_NAME; then
    log_success "Service started successfully"
else
    log_error "Service failed to start!"
    log_info "Checking service status..."
    systemctl status $SERVICE_NAME

    # Restore backup if available
    if [ -f "$APP_DIR/$BACKUP_NAME" ]; then
        log_info "Attempting to restore backup..."
        sudo -u $APP_USER cp "$APP_DIR/$BACKUP_NAME" "$APP_DIR/$APP_NAME"
        chcon -t bin_t $APP_DIR/$APP_NAME 2>/dev/null || true
        systemctl start $SERVICE_NAME
        sleep 2

        if systemctl is-active --quiet $SERVICE_NAME; then
            log_warning "Backup restored and service started"
        else
            log_error "Failed to start service even with backup"
        fi
    fi

    exit 1
fi

# Show service status
log_info ""
log_info "======================================"
log_info "Deployment Summary"
log_info "======================================"
systemctl status $SERVICE_NAME --no-pager -l | head -20

log_info ""
log_success "Update completed successfully!"
log_info ""
log_info "New commit: $(git rev-parse HEAD)"
log_info ""
log_info "You can now access:"
log_info "  - API: http://localhost:8080/api/v1"
log_info "  - Health: http://localhost:8080/health"
log_info "  - Swagger: http://localhost:8080/swagger/index.html"
log_info ""
log_info "View logs: sudo journalctl -u $SERVICE_NAME -f"
log_info ""

# Cleanup old backups (keep last 5)
log_info "Cleaning up old backups..."
cd $APP_DIR
ls -t ${APP_NAME}.backup.* 2>/dev/null | tail -n +6 | xargs -r rm -f
BACKUP_COUNT=$(ls ${APP_NAME}.backup.* 2>/dev/null | wc -l)
log_info "Kept $BACKUP_COUNT most recent backup(s)"
