#!/bin/bash
#
# Phase 2: Database Server Setup Script for Oracle Cloud Infrastructure (Improved)
#
# This script installs and configures PostgreSQL on the database VM
#
# Features:
#   - Idempotent: Safe to re-run multiple times
#   - Progress tracking and detailed logging
#   - Checks existing state before each operation
#
# Usage:
#   1. Upload this script to your database VM
#   2. Run: chmod +x setup-database-vm-v2.sh
#   3. Run: sudo ./setup-database-vm-v2.sh
#
# IMPORTANT: For Oracle Cloud free tier VMs (1GB RAM):
#   - Do NOT use --auto-screen (screen installation may fail due to low memory)
#   - If SSH disconnects, simply re-run the script - it will resume from where it left off
#   - Monitor progress: sudo tail -f /var/log/todolist-db-setup.log
#
# Optional: To run in screen automatically (requires >1GB RAM):
#   sudo ./setup-database-vm-v2.sh --auto-screen
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
DB_NAME="todolist"
DB_USER="todolist"
DB_PASSWORD=""
VCN_CIDR="10.0.0.0/16"  # Default OCI VCN CIDR
PG_VERSION="15"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/todolist-db-setup.log"
STATE_FILE="/var/lib/todolist-db-setup.state"

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
    echo "=== TodoList Database Setup - $(date) ===" | tee -a "$LOG_FILE"
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
            log_info "Re-launching in screen session 'dbsetup'..."
            exec screen -S dbsetup -L -Logfile "$LOG_FILE.screen" "$0"
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
}

# Prompt for database password
prompt_password() {
    if is_complete "password_set"; then
        log_info "Password already configured (skipping prompt)"
        # Try to load from credentials file
        if [ -f "/root/db-credentials.txt" ]; then
            DB_PASSWORD=$(grep "^Database Password:" /root/db-credentials.txt | cut -d: -f2- | xargs)
            if [ -n "$DB_PASSWORD" ]; then
                log_success "Password loaded from credentials file"
                return
            fi
        fi
        log_warning "Could not load saved password, prompting again..."
    fi

    log_step "Setting up database credentials..."

    if [ -z "$DB_PASSWORD" ]; then
        read -sp "Enter password for PostgreSQL user '$DB_USER': " DB_PASSWORD
        echo
        read -sp "Confirm password: " DB_PASSWORD_CONFIRM
        echo

        if [ "$DB_PASSWORD" != "$DB_PASSWORD_CONFIRM" ]; then
            log_error "Passwords do not match"
            exit 1
        fi

        if [ ${#DB_PASSWORD} -lt 12 ]; then
            log_error "Password must be at least 12 characters"
            exit 1
        fi
    fi

    mark_complete "password_set"
    log_success "Password set"
}

# Check if PostgreSQL is already installed
check_postgresql_installed() {
    if command -v psql &> /dev/null; then
        log_info "PostgreSQL is already installed"
        return 0
    fi
    return 1
}

# Install PostgreSQL on Oracle Linux
install_postgresql_oracle() {
    if is_complete "postgresql_installed"; then
        log_info "PostgreSQL already installed (skipping)"
        return
    fi

    log_step "Installing PostgreSQL on Oracle Linux..."

    # Map architecture to PostgreSQL repo naming
    local REPO_ARCH="$ARCH"
    if [ "$ARCH" = "aarch64" ]; then
        REPO_ARCH="aarch64"
    elif [ "$ARCH" = "x86_64" ]; then
        REPO_ARCH="x86_64"
    fi

    # Enable PostgreSQL repository
    if ! rpm -q pgdg-redhat-repo &> /dev/null; then
        log_info "Adding PostgreSQL repository for $REPO_ARCH..."
        if ! dnf install -y "https://download.postgresql.org/pub/repos/yum/reporpms/EL-${OS_VERSION}-${REPO_ARCH}/pgdg-redhat-repo-latest.noarch.rpm"; then
            log_error "Failed to add PostgreSQL repository"
            exit 1
        fi
    fi

    # Disable built-in PostgreSQL module
    log_info "Disabling built-in PostgreSQL module..."
    dnf -qy module disable postgresql || true

    # Fix GPG key issues by updating repo configuration
    log_info "Updating repository metadata..."
    dnf clean all
    dnf makecache || true

    # Install PostgreSQL
    if ! rpm -q postgresql${PG_VERSION}-server &> /dev/null; then
        log_info "Installing PostgreSQL ${PG_VERSION}..."
        if ! dnf install -y --nogpgcheck postgresql${PG_VERSION}-server postgresql${PG_VERSION}-contrib; then
            log_error "Failed to install PostgreSQL ${PG_VERSION}"
            log_error "Please check /var/log/todolist-db-setup.log for details"
            exit 1
        fi
    else
        log_info "PostgreSQL ${PG_VERSION} packages already installed"
    fi

    # Verify installation
    if ! rpm -q postgresql${PG_VERSION}-server &> /dev/null; then
        log_error "PostgreSQL installation verification failed"
        exit 1
    fi

    mark_complete "postgresql_installed"
    log_success "PostgreSQL installed"
}

# Install PostgreSQL on Ubuntu
install_postgresql_ubuntu() {
    if is_complete "postgresql_installed"; then
        log_info "PostgreSQL already installed (skipping)"
        return
    fi

    log_step "Installing PostgreSQL on Ubuntu..."

    apt-get update
    apt-get install -y postgresql postgresql-contrib

    mark_complete "postgresql_installed"
    log_success "PostgreSQL installed"
}

# Initialize PostgreSQL
initialize_postgresql() {
    if is_complete "postgresql_initialized"; then
        log_info "PostgreSQL already initialized (skipping)"
        # Set variables for later steps
        if [ "$OS" = "oracle" ]; then
            PG_DATA_DIR="/var/lib/pgsql/${PG_VERSION}/data"
            PG_SERVICE="postgresql-${PG_VERSION}"
        else
            PG_DATA_DIR="/etc/postgresql/${PG_VERSION}/main"
            PG_SERVICE="postgresql"
        fi
        return
    fi

    log_step "Initializing PostgreSQL..."

    if [ "$OS" = "oracle" ]; then
        # Check if already initialized
        PG_DATA_DIR="/var/lib/pgsql/${PG_VERSION}/data"
        if [ ! -f "${PG_DATA_DIR}/PG_VERSION" ]; then
            log_info "Running initdb..."
            /usr/pgsql-${PG_VERSION}/bin/postgresql-${PG_VERSION}-setup initdb
        else
            log_info "Database already initialized"
        fi

        # Enable and start service
        systemctl enable postgresql-${PG_VERSION}
        systemctl start postgresql-${PG_VERSION}

        PG_SERVICE="postgresql-${PG_VERSION}"
    else
        # Ubuntu
        systemctl enable postgresql
        systemctl start postgresql

        PG_DATA_DIR="/etc/postgresql/${PG_VERSION}/main"
        PG_SERVICE="postgresql"
    fi

    # Wait for PostgreSQL to start
    sleep 3

    if ! systemctl is-active --quiet ${PG_SERVICE}; then
        log_error "PostgreSQL service failed to start"
        systemctl status ${PG_SERVICE}
        exit 1
    fi

    mark_complete "postgresql_initialized"
    log_success "PostgreSQL initialized and started"
}

# Create database and user
create_database() {
    if is_complete "database_created"; then
        log_info "Database and user already created (skipping)"
        return
    fi

    log_step "Creating database and user..."

    # Check if user exists
    if sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" | grep -q 1; then
        log_info "User '${DB_USER}' already exists"
    else
        log_info "Creating user '${DB_USER}'..."
        sudo -u postgres psql -c "CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';"
    fi

    # Check if database exists
    if sudo -u postgres psql -lqt | cut -d \| -f 1 | grep -qw ${DB_NAME}; then
        log_info "Database '${DB_NAME}' already exists"
    else
        log_info "Creating database '${DB_NAME}'..."
        sudo -u postgres psql -c "CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};"
    fi

    # Grant privileges
    log_info "Granting privileges..."
    sudo -u postgres psql << EOF
GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};

\c ${DB_NAME}
GRANT ALL ON SCHEMA public TO ${DB_USER};
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ${DB_USER};
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ${DB_USER};
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ${DB_USER};
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ${DB_USER};
EOF

    mark_complete "database_created"
    log_success "Database '$DB_NAME' and user '$DB_USER' configured"
}

# Configure PostgreSQL for remote access
configure_postgresql() {
    if is_complete "postgresql_configured"; then
        log_info "PostgreSQL already configured (skipping)"
        return
    fi

    log_step "Configuring PostgreSQL for remote access..."

    # Backup original files if not already backed up
    if [ ! -f "${PG_DATA_DIR}/postgresql.conf.backup" ]; then
        cp ${PG_DATA_DIR}/postgresql.conf ${PG_DATA_DIR}/postgresql.conf.backup
    fi
    if [ ! -f "${PG_DATA_DIR}/pg_hba.conf.backup" ]; then
        cp ${PG_DATA_DIR}/pg_hba.conf ${PG_DATA_DIR}/pg_hba.conf.backup
    fi

    # Check if already configured
    if ! grep -q "listen_addresses = '\*'" ${PG_DATA_DIR}/postgresql.conf; then
        log_info "Updating postgresql.conf..."
        cat >> ${PG_DATA_DIR}/postgresql.conf << EOF

# Custom configuration for TodoList API
listen_addresses = '*'
max_connections = 50
shared_buffers = 256MB
effective_cache_size = 512MB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 2621kB
min_wal_size = 512MB
max_wal_size = 2GB
EOF
    else
        log_info "postgresql.conf already configured"
    fi

    # Check if pg_hba.conf is configured
    if ! grep -q "# Allow access from VCN" ${PG_DATA_DIR}/pg_hba.conf; then
        log_info "Updating pg_hba.conf..."
        cat >> ${PG_DATA_DIR}/pg_hba.conf << EOF

# Allow access from VCN
host    ${DB_NAME}    ${DB_USER}    ${VCN_CIDR}    md5
host    all           ${DB_USER}    ${VCN_CIDR}    md5
EOF
    else
        log_info "pg_hba.conf already configured"
    fi

    mark_complete "postgresql_configured"
    log_success "PostgreSQL configured"
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
        if ! firewall-cmd --list-services | grep -q postgresql; then
            firewall-cmd --permanent --add-service=postgresql
        fi
        if ! firewall-cmd --list-ports | grep -q 5432/tcp; then
            firewall-cmd --permanent --add-port=5432/tcp
        fi
        firewall-cmd --reload
        log_success "Firewall configured (firewalld)"
    elif command -v ufw &> /dev/null; then
        # ufw (Ubuntu)
        ufw allow 5432/tcp
        log_success "Firewall configured (ufw)"
    else
        log_warning "No firewall detected. Please configure manually."
    fi

    mark_complete "firewall_configured"
}

# Restart PostgreSQL
restart_postgresql() {
    log_step "Restarting PostgreSQL..."
    systemctl restart ${PG_SERVICE}
    sleep 3

    if systemctl is-active --quiet ${PG_SERVICE}; then
        log_success "PostgreSQL restarted successfully"
    else
        log_error "PostgreSQL failed to restart"
        systemctl status ${PG_SERVICE}
        exit 1
    fi
}

# Test connection
test_connection() {
    log_step "Testing database connection..."

    PGPASSWORD=${DB_PASSWORD} psql -U ${DB_USER} -d ${DB_NAME} -h localhost -c "SELECT version();" &> /dev/null
    if [ $? -eq 0 ]; then
        log_success "Database connection test passed"
    else
        log_warning "Database connection test failed (password authentication may be required)"
    fi
}

# Save credentials to file
save_credentials() {
    CREDS_FILE="/root/db-credentials.txt"
    PRIVATE_IP=$(ip addr show | grep "inet " | grep -v "127.0.0.1" | awk '{print $2}' | cut -d/ -f1 | head -1)

    cat > ${CREDS_FILE} << EOF
TodoList Database Credentials
Generated: $(date)

Database Name: ${DB_NAME}
Database User: ${DB_USER}
Database Password: ${DB_PASSWORD}
Private IP: ${PRIVATE_IP}
Port: 5432

Connection String:
DB_HOST=${PRIVATE_IP}
DB_PORT=5432
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}
DB_SSLMODE=disable
EOF

    chmod 600 ${CREDS_FILE}
    log_info "Credentials saved to: ${CREDS_FILE}"
}

# Show connection info
show_connection_info() {
    PRIVATE_IP=$(ip addr show | grep "inet " | grep -v "127.0.0.1" | awk '{print $2}' | cut -d/ -f1 | head -1)

    cat << EOF

${GREEN}=================================================================
Database Setup Complete!
=================================================================${NC}

Database Information:
  Database Name: ${DB_NAME}
  Database User: ${DB_USER}
  Database Password: ${DB_PASSWORD}

Connection Details:
  Host: ${PRIVATE_IP} (Private IP)
  Port: 5432

Connection String:
  postgresql://${DB_USER}:${DB_PASSWORD}@${PRIVATE_IP}:5432/${DB_NAME}

Test Connection:
  PGPASSWORD='${DB_PASSWORD}' psql -U ${DB_USER} -h ${PRIVATE_IP} -d ${DB_NAME}

${YELLOW}IMPORTANT:${NC}
1. Save the password securely!
2. Use the PRIVATE IP (${PRIVATE_IP}) in your application's .env file
3. Ensure OCI Security List allows port 5432 from your VCN CIDR
4. Configure OS firewall if needed

Next Steps:
  - Set up the application VM
  - Use the connection details in your .env file

Log file: ${LOG_FILE}
State file: ${STATE_FILE}

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
   sudo screen -r dbsetup

3. Or just re-run the script (it will skip completed steps):
   sudo ./setup-database-vm-v2.sh

Progress is saved in: ${STATE_FILE}
Logs are saved in: ${LOG_FILE}

EOF
}

# Main installation
main() {
    setup_logging
    log_info "Starting PostgreSQL setup for OCI (Improved Version)..."

    check_root
    auto_screen
    detect_os
    prompt_password

    echo
    log_info "Beginning installation steps..."
    echo

    if [ "$OS" = "oracle" ]; then
        install_postgresql_oracle
    elif [ "$OS" = "ubuntu" ]; then
        install_postgresql_ubuntu
    else
        log_error "Unsupported OS: $OS"
        exit 1
    fi

    initialize_postgresql
    create_database
    configure_postgresql
    configure_firewall
    restart_postgresql
    test_connection
    save_credentials

    echo
    show_connection_info

    log_success "All steps completed successfully!"
}

# Trap errors and show resume instructions
trap 'log_error "Script interrupted or error occurred"; show_resume_instructions; exit 1' ERR INT TERM

# Run main installation
main
