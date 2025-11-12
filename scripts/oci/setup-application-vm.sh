#!/bin/bash
#
# Phase 3: Application Server Setup Script for Oracle Cloud Infrastructure
#
# This script installs Go, clones the repository, builds the application,
# and sets up the systemd service
#
# Usage:
#   1. Upload this script to your application VM
#   2. Run: chmod +x setup-application-vm.sh
#   3. Run: sudo ./setup-application-vm.sh
#

set -e  # Exit on error
set -u  # Exit on undefined variable

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="todolist-api"
APP_DIR="/opt/${APP_NAME}"
APP_USER="todolist"
GIT_REPO=""
GIT_BRANCH="main"
GO_VERSION="1.21.5"

# Database configuration
DB_HOST=""
DB_PORT="5432"
DB_USER="todolist"
DB_PASSWORD=""
DB_NAME="todolist"

# JWT configuration
JWT_SECRET=""

# Functions
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
    log_error "Please run as root (use sudo)"
    exit 1
fi

# Detect OS
detect_os() {
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
}

# Prompt for configuration
prompt_configuration() {
    log_info "Application Configuration"
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

    echo
}

# Install dependencies
install_dependencies() {
    log_info "Installing system dependencies..."

    if [ "$OS" = "oracle" ]; then
        dnf update -y
        dnf install -y git wget tar curl make
    else
        apt-get update
        apt-get install -y git wget tar curl make
    fi

    log_success "System dependencies installed"
}

# Install Go
install_go() {
    log_info "Installing Go ${GO_VERSION}..."

    # Download Go
    cd /tmp
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz

    # Remove old installation
    rm -rf /usr/local/go

    # Extract and install
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz

    # Add to system PATH
    cat > /etc/profile.d/go.sh << 'EOF'
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$HOME/go/bin
EOF

    # Source for current session
    export PATH=$PATH:/usr/local/go/bin

    # Verify installation
    /usr/local/go/bin/go version
    log_success "Go ${GO_VERSION} installed"
}

# Create application user
create_app_user() {
    log_info "Creating application user..."

    if id "$APP_USER" &>/dev/null; then
        log_warning "User '$APP_USER' already exists"
    else
        useradd -r -s /bin/bash -d ${APP_DIR} -m ${APP_USER}
        log_success "User '$APP_USER' created"
    fi
}

# Clone repository
clone_repository() {
    log_info "Cloning repository..."

    # Create directory
    mkdir -p ${APP_DIR}

    # Clone as root first, then change ownership
    if [ -d "${APP_DIR}/.git" ]; then
        log_warning "Repository already exists, pulling latest..."
        cd ${APP_DIR}
        sudo -u ${APP_USER} git pull origin ${GIT_BRANCH}
    else
        git clone ${GIT_REPO} ${APP_DIR}
    fi

    # Change ownership
    chown -R ${APP_USER}:${APP_USER} ${APP_DIR}

    log_success "Repository cloned"
}

# Build application
build_application() {
    log_info "Building application..."

    cd ${APP_DIR}

    # Build as app user
    sudo -u ${APP_USER} bash << EOF
export PATH=\$PATH:/usr/local/go/bin:\$HOME/go/bin
cd ${APP_DIR}

# Download dependencies
go mod download

# Build server binary
go build -o ${APP_NAME} cmd/server/main.go

# Build migration tool
mkdir -p bin
go build -o bin/migrate cmd/migrate/main.go

# Make executable
chmod +x ${APP_NAME} bin/migrate
EOF

    if [ -f "${APP_DIR}/${APP_NAME}" ]; then
        log_success "Application built successfully"
    else
        log_error "Application build failed"
        exit 1
    fi
}

# Create environment file
create_env_file() {
    log_info "Creating environment configuration..."

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
DB_SSL_MODE=disable

# JWT Configuration
JWT_SECRET_KEY=${JWT_SECRET}
JWT_ACCESS_TOKEN_MINUTES=15
JWT_REFRESH_TOKEN_DAYS=7
JWT_ISSUER=todolist-api

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
LOG_FILE=${APP_DIR}/logs/app.log
LOG_MAX_SIZE=100
LOG_MAX_AGE=30
LOG_MAX_BACKUPS=10

# TLS (using Nginx for SSL)
TLS_ENABLED=false
EOF

    # Secure the file
    chown ${APP_USER}:${APP_USER} ${APP_DIR}/.env
    chmod 600 ${APP_DIR}/.env

    log_success "Environment file created"
}

# Create log directory
create_log_directory() {
    log_info "Creating log directory..."

    mkdir -p ${APP_DIR}/logs
    chown ${APP_USER}:${APP_USER} ${APP_DIR}/logs
    chmod 755 ${APP_DIR}/logs

    log_success "Log directory created"
}

# Run database migrations
run_migrations() {
    log_info "Running database migrations..."

    cd ${APP_DIR}

    # Test database connection first
    if sudo -u ${APP_USER} bash -c "export PATH=\$PATH:/usr/local/go/bin; cd ${APP_DIR}; ./bin/migrate version" &> /dev/null; then
        log_info "Database connection successful"
    else
        log_error "Cannot connect to database. Check credentials and network connectivity."
        exit 1
    fi

    # Run migrations
    if sudo -u ${APP_USER} bash -c "export PATH=\$PATH:/usr/local/go/bin; cd ${APP_DIR}; ./bin/migrate up"; then
        log_success "Migrations completed"
    else
        log_error "Migration failed"
        exit 1
    fi
}

# Install systemd service
install_systemd_service() {
    log_info "Installing systemd service..."

    # Copy service file
    cp ${APP_DIR}/todolist-api.service /etc/systemd/system/

    # Reload systemd
    systemctl daemon-reload

    # Enable service
    systemctl enable ${APP_NAME}

    log_success "Systemd service installed"
}

# Configure firewall
configure_firewall() {
    log_info "Configuring firewall..."

    if command -v firewall-cmd &> /dev/null; then
        # firewalld (Oracle Linux)
        firewall-cmd --permanent --add-port=8080/tcp
        firewall-cmd --permanent --add-port=80/tcp
        firewall-cmd --permanent --add-port=443/tcp
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
}

# Start application
start_application() {
    log_info "Starting application..."

    systemctl start ${APP_NAME}
    sleep 3

    if systemctl is-active --quiet ${APP_NAME}; then
        log_success "Application started successfully"
    else
        log_error "Application failed to start"
        systemctl status ${APP_NAME}
        journalctl -u ${APP_NAME} -n 50
        exit 1
    fi
}

# Test application
test_application() {
    log_info "Testing application..."

    sleep 2

    if curl -f -s http://localhost:8080/health > /dev/null; then
        log_success "Health check passed"
    else
        log_warning "Health check failed. Application might still be starting..."
    fi
}

# Show summary
show_summary() {
    PUBLIC_IP=$(curl -s ifconfig.me || echo "Unable to determine")
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
    -d '{"email":"test@example.com","password":"SecurePass123!"}'

${YELLOW}Next Steps:${NC}
1. Set up Nginx reverse proxy (Phase 4)
2. Configure SSL with Let's Encrypt (Phase 5)
3. Test the API endpoints
4. Set up monitoring and backups

${YELLOW}Important Files:${NC}
  Config: ${APP_DIR}/.env
  Logs:   ${APP_DIR}/logs/app.log
  Binary: ${APP_DIR}/${APP_NAME}

${YELLOW}Security Notes:${NC}
- Environment file contains sensitive data (secured with 600 permissions)
- JWT secret has been auto-generated
- Default user is 'todolist' with restricted permissions

EOF
}

# Main installation
main() {
    log_info "Starting application setup for OCI..."
    echo

    detect_os
    prompt_configuration

    echo
    log_info "Installing dependencies..."
    install_dependencies
    install_go

    log_info "Setting up application..."
    create_app_user
    clone_repository
    build_application
    create_env_file
    create_log_directory

    log_info "Setting up database..."
    run_migrations

    log_info "Installing service..."
    install_systemd_service
    configure_firewall
    start_application
    test_application

    echo
    show_summary
}

# Run main installation
main
