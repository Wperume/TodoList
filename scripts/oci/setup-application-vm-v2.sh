#!/bin/bash
#
# Phase 3: Application Server Setup Script for Oracle Cloud Infrastructure (Improved)
#
# This script installs Go, clones the repository, builds the application,
# and sets up the systemd service
#
# Features:
#   - Idempotent: Safe to re-run multiple times
#   - Progress tracking and detailed logging
#   - Checks existing state before each operation
#
# Usage:
#   1. Upload this script to your application VM
#   2. Run: chmod +x setup-application-vm-v2.sh
#   3. Run: sudo ./setup-application-vm-v2.sh
#
# IMPORTANT: For Oracle Cloud free tier VMs (1GB RAM):
#   - Do NOT use --auto-screen (screen installation may fail due to low memory)
#   - If SSH disconnects, simply re-run the script - it will resume from where it left off
#   - Monitor progress: sudo tail -f /var/log/todolist-app-setup.log
#
# Optional: To run in screen automatically (requires >1GB RAM):
#   sudo ./setup-application-vm-v2.sh --auto-screen
#

set -u  # Exit on undefined variable
# Note: We don't use 'set -e' to allow better error handling

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="todolist-api"
APP_DIR="/opt/${APP_NAME}"
APP_USER="todolist"
GIT_REPO=""
GIT_BRANCH="main"
GO_VERSION="1.24.10"

# Database configuration
DB_HOST=""
DB_PORT="5432"
DB_USER="todolist"
DB_PASSWORD=""
DB_NAME="todolist"

# JWT configuration
JWT_SECRET=""

# Script state
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/todolist-app-setup.log"
STATE_FILE="/var/lib/todolist-app-setup.state"

# Auto-screen support
AUTO_SCREEN=false
if [ "${1:-}" = "--auto-screen" ]; then
    AUTO_SCREEN=true
fi

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_step() {
    echo -e "${CYAN}[STEP]${NC} $1" | tee -a "$LOG_FILE"
}

# Mark step as complete
mark_complete() {
    local step=$1
    echo "$step" >> "$STATE_FILE"
    log_success "Step '$step' completed"
}

# Check if step is already complete
is_complete() {
    local step=$1
    if [ -f "$STATE_FILE" ] && grep -q "^${step}$" "$STATE_FILE"; then
        return 0
    fi
    return 1
}

# Setup logging
setup_logging() {
    mkdir -p "$(dirname "$LOG_FILE")"
    mkdir -p "$(dirname "$STATE_FILE")"
    echo "=== TodoList Application Setup - $(date) ===" | tee -a "$LOG_FILE"
}

# Check if running in screen
check_screen() {
    if [ -n "${STY:-}" ]; then
        log_info "Running in screen session: $STY"
        return 0
    fi
    return 1
}

# Auto-start screen if requested and not already in screen
auto_screen() {
    if [ "$AUTO_SCREEN" = true ] && ! check_screen; then
        log_info "Starting screen session for long-running setup..."

        # Install screen if not present
        if ! command -v screen &> /dev/null; then
            log_info "Installing screen..."
            local install_success=false

            if command -v dnf &> /dev/null; then
                if dnf install -y screen 2>/dev/null; then
                    install_success=true
                fi
            elif command -v yum &> /dev/null; then
                if yum install -y screen 2>/dev/null; then
                    install_success=true
                fi
            elif command -v apt-get &> /dev/null; then
                if apt-get update && apt-get install -y screen 2>/dev/null; then
                    install_success=true
                fi
            fi

            # Verify screen was actually installed
            if ! command -v screen &> /dev/null; then
                install_success=false
            fi

            if [ "$install_success" = false ]; then
                log_warn "Failed to install screen - continuing without it"
                log_warn "Screen is optional but helps prevent SSH timeout issues"
                log_warn "Consider running setup manually inside screen session"
                return 0
            fi
        fi

        # Verify screen is available before trying to use it
        if command -v screen &> /dev/null; then
            # Re-exec in screen
            log_info "Re-launching in screen session 'appsetup'..."
            exec screen -S appsetup -L -Logfile "$LOG_FILE.screen" "$0"
        else
            log_warn "Screen not available - continuing without it"
        fi
    fi
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "Please run as root (use sudo)"
        exit 1
    fi
}

# Detect OS
detect_os() {
    log_step "Detecting operating system..."

    if [ -f /etc/oracle-release ]; then
        OS="oracle"
        OS_VERSION=$(cat /etc/oracle-release | grep -oP '\d+' | head -1)
        log_info "Detected: Oracle Linux $OS_VERSION"
    elif [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
        log_info "Detected: $NAME $VERSION_ID"
    else
        log_error "Unable to detect OS"
        exit 1
    fi

    # Detect architecture
    ARCH=$(uname -m)
    log_info "Architecture: $ARCH"

    # Map to Go's architecture naming
    if [ "$ARCH" = "x86_64" ]; then
        GO_ARCH="amd64"
    elif [ "$ARCH" = "aarch64" ]; then
        GO_ARCH="arm64"
    else
        log_error "Unsupported architecture: $ARCH"
        exit 1
    fi
}

# Prompt for configuration
prompt_configuration() {
    if is_complete "configuration_set"; then
        log_info "Configuration already set (skipping prompt)"
        # Try to load from saved config
        if [ -f "${APP_DIR}/.env" ]; then
            DB_HOST=$(grep "^DB_HOST=" ${APP_DIR}/.env | cut -d= -f2)
            DB_PASSWORD=$(grep "^DB_PASSWORD=" ${APP_DIR}/.env | cut -d= -f2)
            JWT_SECRET=$(grep "^JWT_SECRET_KEY=" ${APP_DIR}/.env | cut -d= -f2)
            log_success "Configuration loaded from .env file"
            return
        fi
        log_warning "Could not load saved configuration, prompting again..."
    fi

    log_step "Application Configuration"
    echo

    # Git repository
    if [ -z "$GIT_REPO" ]; then
        read -p "Enter your Git repository URL: " GIT_REPO
    fi

    # Database host
    if [ -z "$DB_HOST" ]; then
        read -p "Enter database host (private IP): " DB_HOST
    fi

    # Database password
    if [ -z "$DB_PASSWORD" ]; then
        read -sp "Enter database password: " DB_PASSWORD
        echo
    fi

    # JWT secret
    if [ -z "$JWT_SECRET" ]; then
        log_info "Generating JWT secret key..."
        JWT_SECRET=$(openssl rand -base64 48)
        log_success "JWT secret generated"
    fi

    mark_complete "configuration_set"
    echo
}

# Install dependencies
install_dependencies() {
    if is_complete "dependencies_installed"; then
        log_info "System dependencies already installed (skipping)"
        return
    fi

    log_step "Installing system dependencies..."

    if [ "$OS" = "oracle" ]; then
        dnf update -y
        dnf install -y git wget tar curl make
    else
        apt-get update
        apt-get install -y git wget tar curl make
    fi

    mark_complete "dependencies_installed"
    log_success "System dependencies installed"
}

# Install Go
install_go() {
    if is_complete "go_installed"; then
        log_info "Go already installed (skipping)"
        export PATH=$PATH:/usr/local/go/bin
        return
    fi

    log_step "Installing Go ${GO_VERSION}..."

    # Check if Go is already installed
    if command -v go &> /dev/null; then
        INSTALLED_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        if [ "$INSTALLED_VERSION" = "$GO_VERSION" ]; then
            log_info "Go ${GO_VERSION} already installed"
            mark_complete "go_installed"
            return
        fi
        log_info "Different Go version found, reinstalling..."
    fi

    # Download Go
    cd /tmp
    local GO_TARBALL="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    if [ ! -f "$GO_TARBALL" ]; then
        log_info "Downloading Go ${GO_VERSION} for ${GO_ARCH}..."
        if ! wget -q "https://go.dev/dl/${GO_TARBALL}"; then
            log_error "Failed to download Go ${GO_VERSION}"
            log_error "URL: https://go.dev/dl/${GO_TARBALL}"
            exit 1
        fi
    fi

    # Verify download
    if [ ! -f "$GO_TARBALL" ]; then
        log_error "Go tarball not found after download: $GO_TARBALL"
        exit 1
    fi

    # Remove old installation
    rm -rf /usr/local/go

    # Extract and install
    log_info "Extracting Go..."
    if ! tar -C /usr/local -xzf "$GO_TARBALL"; then
        log_error "Failed to extract Go tarball"
        exit 1
    fi

    # Add to system PATH
    if [ ! -f /etc/profile.d/go.sh ]; then
        cat > /etc/profile.d/go.sh << 'EOF'
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$HOME/go/bin
EOF
    fi

    # Source for current session
    export PATH=$PATH:/usr/local/go/bin

    # Verify installation
    if /usr/local/go/bin/go version &> /dev/null; then
        mark_complete "go_installed"
        log_success "Go ${GO_VERSION} installed"
    else
        log_error "Go installation failed"
        exit 1
    fi
}

# Create application user
create_app_user() {
    if is_complete "app_user_created"; then
        log_info "Application user already exists (skipping)"
        return
    fi

    log_step "Creating application user..."

    if id "$APP_USER" &>/dev/null; then
        log_info "User '$APP_USER' already exists"
    else
        useradd -r -s /bin/bash -d ${APP_DIR} -m ${APP_USER}
        log_success "User '$APP_USER' created"
    fi

    mark_complete "app_user_created"
}

# Clone repository
clone_repository() {
    if is_complete "repository_cloned"; then
        log_info "Repository already cloned, pulling latest changes..."
        if [ -d "${APP_DIR}/.git" ]; then
            cd ${APP_DIR}
            sudo -u ${APP_USER} git pull origin ${GIT_BRANCH} || true
            chown -R ${APP_USER}:${APP_USER} ${APP_DIR}
        fi
        return
    fi

    log_step "Cloning repository..."

    # Check if directory exists but is not a git repo
    if [ -d "${APP_DIR}" ] && [ ! -d "${APP_DIR}/.git" ]; then
        log_warn "Directory exists but is not a git repository, removing..."
        rm -rf ${APP_DIR}
    fi

    # Clone repository
    if [ -d "${APP_DIR}/.git" ]; then
        log_info "Repository already exists, pulling latest..."
        cd ${APP_DIR}
        git pull origin ${GIT_BRANCH}
        chown -R ${APP_USER}:${APP_USER} ${APP_DIR}
    else
        log_info "Cloning from ${GIT_REPO}..."
        if ! git clone -b ${GIT_BRANCH} ${GIT_REPO} ${APP_DIR}; then
            log_error "Failed to clone repository"
            log_error "Git URL: ${GIT_REPO}"
            log_error "Branch: ${GIT_BRANCH}"
            exit 1
        fi
        chown -R ${APP_USER}:${APP_USER} ${APP_DIR}
    fi

    # Verify clone succeeded
    if [ ! -f "${APP_DIR}/go.mod" ]; then
        log_error "Repository cloned but go.mod not found"
        log_error "This may not be a valid Go project repository"
        exit 1
    fi

    mark_complete "repository_cloned"
    log_success "Repository cloned"
}

# Build application
build_application() {
    if is_complete "application_built"; then
        log_info "Application already built, checking if rebuild needed..."
        # Always rebuild if source files are newer
        if [ -f "${APP_DIR}/${APP_NAME}" ]; then
            log_info "Binary exists, skipping rebuild (delete to force rebuild)"
            return
        fi
    fi

    log_step "Building application..."

    cd ${APP_DIR}

    # Build as app user
    log_info "Downloading Go dependencies..."
    sudo -u ${APP_USER} bash << EOF
export PATH=\$PATH:/usr/local/go/bin:\$HOME/go/bin
cd ${APP_DIR}

# Download dependencies
go mod download
EOF

    if [ $? -ne 0 ]; then
        log_error "Failed to download Go dependencies"
        exit 1
    fi

    log_info "Building server binary..."
    sudo -u ${APP_USER} bash << EOF
export PATH=\$PATH:/usr/local/go/bin:\$HOME/go/bin
cd ${APP_DIR}

# Build server binary
go build -o ${APP_NAME} ./cmd/server
EOF

    if [ $? -ne 0 ]; then
        log_error "Failed to build server binary"
        exit 1
    fi

    log_info "Building migration tool..."
    sudo -u ${APP_USER} bash << EOF
export PATH=\$PATH:/usr/local/go/bin:\$HOME/go/bin
cd ${APP_DIR}

# Build migration tool
mkdir -p bin
go build -o bin/migrate ./cmd/migrate
EOF

    if [ $? -ne 0 ]; then
        log_error "Failed to build migration tool"
        exit 1
    fi

    # Make executable
    chmod +x ${APP_DIR}/${APP_NAME} ${APP_DIR}/bin/migrate

    if [ -f "${APP_DIR}/${APP_NAME}" ]; then
        mark_complete "application_built"
        log_success "Application built successfully"
    else
        log_error "Application build failed"
        exit 1
    fi
}

# Create environment file
create_env_file() {
    if is_complete "env_file_created"; then
        log_info "Environment file already exists (skipping)"
        return
    fi

    log_step "Creating environment configuration..."

    cat > ${APP_DIR}/.env << EOF
# Server Configuration
PORT=8080
GIN_MODE=release

# Database Configuration
DB_HOST=${DB_HOST}
DB_PORT=${DB_PORT}
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}
DB_SSLMODE=disable

# JWT Configuration
JWT_SECRET_KEY=${JWT_SECRET}
JWT_ACCESS_TOKEN_MINUTES=15
JWT_REFRESH_TOKEN_DAYS=7
JWT_ISSUER=todolist-api

# Logging
LOG_LEVEL=info
LOG_JSON_FORMAT=true
LOG_FILE_ENABLED=true
LOG_FILE_PATH=${APP_DIR}/logs/app.log
LOG_MAX_SIZE_MB=100
LOG_MAX_AGE_DAYS=30
LOG_MAX_BACKUPS=10
LOG_COMPRESS=true

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MIN=60

# Security
MAX_REQUEST_BODY_SIZE=1048576
ENABLE_XSS_PROTECTION=true

# CORS
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=*

# TLS (using Nginx for SSL)
TLS_ENABLED=false
EOF

    # Secure the file
    chown ${APP_USER}:${APP_USER} ${APP_DIR}/.env
    chmod 600 ${APP_DIR}/.env

    mark_complete "env_file_created"
    log_success "Environment file created"
}

# Create log directory
create_log_directory() {
    log_step "Creating log directory..."

    mkdir -p ${APP_DIR}/logs
    chown ${APP_USER}:${APP_USER} ${APP_DIR}/logs
    chmod 755 ${APP_DIR}/logs

    log_success "Log directory created"
}

# Run database migrations
run_migrations() {
    if is_complete "migrations_run"; then
        log_info "Database migrations already run (skipping)"
        return
    fi

    log_step "Running database migrations..."

    cd ${APP_DIR}

    # Load environment variables for migration
    export $(grep -v '^#' ${APP_DIR}/.env | xargs)

    # Test database connection first
    log_info "Testing database connection..."
    log_info "Connection details: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"

    # Check if we can reach the database host
    log_info "Checking network connectivity to ${DB_HOST}:${DB_PORT}..."
    if ! command -v nc &> /dev/null; then
        dnf install -y nc 2>/dev/null || yum install -y nc 2>/dev/null || apt-get install -y netcat 2>/dev/null
    fi

    if nc -z -w 5 ${DB_HOST} ${DB_PORT} 2>/dev/null; then
        log_success "Network connectivity OK - port ${DB_PORT} is reachable"
    else
        log_error "Cannot reach ${DB_HOST}:${DB_PORT}"
        log_error "Troubleshooting steps:"
        log_error "  1. Verify database VM is running: sudo systemctl status postgresql-15"
        log_error "  2. Check database VM firewall: sudo firewall-cmd --list-all (or sudo ufw status)"
        log_error "  3. Verify OCI Security List allows port 5432 from this VM's subnet"
        log_error "  4. Check PostgreSQL is listening: sudo netstat -plnt | grep 5432 (on DB VM)"
        log_error "  5. Verify pg_hba.conf allows connections from this IP"
        exit 1
    fi

    # Test actual database connection
    if sudo -u ${APP_USER} bash -c "source ${APP_DIR}/.env && cd ${APP_DIR} && ./bin/migrate version" &> /dev/null; then
        log_success "Database connection successful"
    else
        log_error "Network is reachable but database authentication failed"
        log_error "Check:"
        log_error "  - Database password is correct"
        log_error "  - Database '${DB_NAME}' exists on ${DB_HOST}"
        log_error "  - User '${DB_USER}' has access to database '${DB_NAME}'"
        log_error "  - pg_hba.conf allows password authentication from this IP"
        exit 1
    fi

    # Run migrations
    log_info "Applying migrations..."
    if sudo -u ${APP_USER} bash -c "source ${APP_DIR}/.env && cd ${APP_DIR} && ./bin/migrate up"; then
        mark_complete "migrations_run"
        log_success "Migrations completed"
    else
        log_error "Migration failed"
        exit 1
    fi
}

# Create systemd service
create_systemd_service() {
    log_step "Creating systemd service file..."

    cat > /etc/systemd/system/${APP_NAME}.service << EOF
[Unit]
Description=TodoList REST API
After=network.target

[Service]
Type=simple
User=${APP_USER}
WorkingDirectory=${APP_DIR}
EnvironmentFile=${APP_DIR}/.env
ExecStart=${APP_DIR}/${APP_NAME}
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${APP_NAME}

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${APP_DIR}/logs

[Install]
WantedBy=multi-user.target
EOF

    log_success "Systemd service file created"
}

# Install systemd service
install_systemd_service() {
    if is_complete "systemd_service_installed"; then
        log_info "Systemd service already installed (skipping)"
        return
    fi

    create_systemd_service

    # Reload systemd
    systemctl daemon-reload

    # Enable service
    systemctl enable ${APP_NAME}

    mark_complete "systemd_service_installed"
    log_success "Systemd service installed and enabled"
}

# Configure firewall
configure_firewall() {
    if is_complete "firewall_configured"; then
        log_info "Firewall already configured (skipping)"
        return
    fi

    log_step "Configuring firewall..."

    if command -v firewall-cmd &> /dev/null; then
        # firewalld (Oracle Linux)
        firewall-cmd --permanent --add-port=8080/tcp || true
        firewall-cmd --permanent --add-port=80/tcp || true
        firewall-cmd --permanent --add-port=443/tcp || true
        firewall-cmd --reload
        log_success "Firewall configured (firewalld)"
    elif command -v ufw &> /dev/null; then
        # ufw (Ubuntu)
        ufw allow 8080/tcp
        ufw allow 80/tcp
        ufw allow 443/tcp
        log_success "Firewall configured (ufw)"
    else
        log_warning "No firewall detected. Please configure manually."
    fi

    mark_complete "firewall_configured"
}

# Start application
start_application() {
    log_step "Starting application..."

    # Check if already running
    if systemctl is-active --quiet ${APP_NAME}; then
        log_info "Application is already running, restarting..."
        systemctl restart ${APP_NAME}
    else
        systemctl start ${APP_NAME}
    fi

    sleep 3

    if systemctl is-active --quiet ${APP_NAME}; then
        log_success "Application started successfully"
    else
        log_error "Application failed to start"
        systemctl status ${APP_NAME}
        journalctl -u ${APP_NAME} -n 50 --no-pager
        exit 1
    fi
}

# Test application
test_application() {
    log_step "Testing application..."

    sleep 2

    if curl -f -s http://localhost:8080/health > /dev/null; then
        log_success "Health check passed ✓"
    else
        log_warning "Health check failed. Application might still be starting..."
        log_info "Check logs with: sudo journalctl -u ${APP_NAME} -f"
    fi
}

# Show summary
show_summary() {
    PUBLIC_IP=$(curl -s ifconfig.me 2>/dev/null || echo "Unable to determine")
    PRIVATE_IP=$(ip addr show | grep "inet " | grep -v "127.0.0.1" | awk '{print $2}' | cut -d/ -f1 | head -1)

    cat << EOF

${GREEN}=================================================================
Application Setup Complete!
=================================================================${NC}

Application Information:
  Name: ${APP_NAME}
  Directory: ${APP_DIR}
  User: ${APP_USER}

Network Information:
  Private IP: ${PRIVATE_IP}
  Public IP: ${PUBLIC_IP}
  Application Port: 8080

API Endpoints:
  Health Check: http://localhost:8080/health
  Base URL: http://localhost:8080/api/v1

Service Management:
  Status:  sudo systemctl status ${APP_NAME}
  Start:   sudo systemctl start ${APP_NAME}
  Stop:    sudo systemctl stop ${APP_NAME}
  Restart: sudo systemctl restart ${APP_NAME}
  Logs:    sudo journalctl -u ${APP_NAME} -f

Test Commands:
  # Health check
  curl http://localhost:8080/health

  # Register user
  curl -X POST http://localhost:8080/api/v1/auth/register \\
    -H "Content-Type: application/json" \\
    -d '{"email":"test@example.com","password":"SecurePass123!","firstName":"Test","lastName":"User"}'

${YELLOW}Next Steps:${NC}
1. Set up Nginx reverse proxy (optional)
2. Configure SSL with Let's Encrypt (optional)
3. Test the API endpoints
4. Set up monitoring and backups

${YELLOW}Important Files:${NC}
  Config: ${APP_DIR}/.env
  Logs:   ${APP_DIR}/logs/app.log
  Binary: ${APP_DIR}/${APP_NAME}
  State:  ${STATE_FILE}
  Setup Log: ${LOG_FILE}

${YELLOW}Security Notes:${NC}
- Environment file contains sensitive data (secured with 600 permissions)
- JWT secret has been auto-generated
- Default user is 'todolist' with restricted permissions
- Application runs with systemd security hardening

EOF
}

# Show resume instructions
show_resume_instructions() {
    cat << EOF

${CYAN}=================================================================
SSH Connection Lost or Script Interrupted?
=================================================================${NC}

To check status or resume:

1. Reconnect to your VM:
   ssh opc@your-vm-ip

2. Reattach to screen session (if using screen):
   sudo screen -r appsetup

3. Or just re-run the script (it will skip completed steps):
   sudo ./setup-application-vm-v2.sh

Progress is saved in: ${STATE_FILE}
Logs are saved in: ${LOG_FILE}

To reset and start fresh:
   sudo rm ${STATE_FILE}

EOF
}

# Main installation
main() {
    setup_logging
    log_info "Starting application setup for OCI (Improved Version)..."

    check_root
    auto_screen
    detect_os
    prompt_configuration

    echo
    log_info "Beginning installation steps..."
    echo

    install_dependencies
    install_go
    create_app_user
    clone_repository
    build_application
    create_env_file
    create_log_directory
    run_migrations
    install_systemd_service
    configure_firewall
    start_application
    test_application

    echo
    show_summary

    log_success "All steps completed successfully!"
}

# Trap errors and show resume instructions
trap 'log_error "Script interrupted or error occurred"; show_resume_instructions; exit 1' ERR INT TERM

# Run main installation
main
